package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	credentialStatusActive      = "active"
	credentialStatusExpiring    = "expiring"
	credentialStatusExpired     = "expired"
	credentialStatusInvalid     = "invalid"
	credentialStatusCooldown    = "cooldown"
	credentialStatusDisabled    = "disabled"
	credentialStatusNeedRefresh = "need_refresh"
)

const (
	expiringThreshold    = 24 * time.Hour
	needRefreshThreshold = 30 * time.Minute
	defaultCooldown      = 5 * time.Minute
)

func toNullString(s string) sql.NullString {
	if strings.TrimSpace(s) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t.UTC(), Valid: true}
}

func fromNullTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time.UTC()
	return &v
}

func deriveCredentialStatus(cred *EndpointCredential, now time.Time) string {
	if !cred.Enabled {
		return credentialStatusDisabled
	}
	if cred.ExpiresAt != nil && !now.Before(cred.ExpiresAt.UTC()) {
		return credentialStatusExpired
	}
	if cred.CooldownUntil != nil && now.Before(cred.CooldownUntil.UTC()) {
		return credentialStatusCooldown
	}
	if cred.Status == credentialStatusInvalid {
		return credentialStatusInvalid
	}
	if cred.ExpiresAt != nil {
		remaining := cred.ExpiresAt.UTC().Sub(now)
		if remaining <= needRefreshThreshold && cred.RefreshToken != "" {
			return credentialStatusNeedRefresh
		}
		if remaining <= expiringThreshold {
			return credentialStatusExpiring
		}
	}
	return credentialStatusActive
}

func classifyCredentialForStats(stats *TokenPoolStats, cred *EndpointCredential, now time.Time) {
	stats.Total++
	switch deriveCredentialStatus(cred, now) {
	case credentialStatusDisabled:
		stats.Disabled++
	case credentialStatusExpired:
		stats.Expired++
	case credentialStatusInvalid:
		stats.Invalid++
	case credentialStatusCooldown:
		stats.Cooldown++
	case credentialStatusNeedRefresh:
		stats.NeedRefresh++
	case credentialStatusExpiring:
		stats.Expiring++
	default:
		stats.Active++
	}
}

func scanCredential(scanner interface {
	Scan(dest ...interface{}) error
}) (*EndpointCredential, error) {
	var cred EndpointCredential
	var accountID, email, accessToken, refreshToken, idToken, status, lastError, remark sql.NullString
	var lastRefresh, expiresAt, cooldownUntil, lastCheckedAt, lastUsedAt sql.NullTime

	if err := scanner.Scan(
		&cred.ID,
		&cred.EndpointName,
		&cred.ProviderType,
		&accountID,
		&email,
		&accessToken,
		&refreshToken,
		&idToken,
		&lastRefresh,
		&expiresAt,
		&status,
		&cred.Enabled,
		&cred.FailureCount,
		&cooldownUntil,
		&lastCheckedAt,
		&lastUsedAt,
		&lastError,
		&remark,
		&cred.CreatedAt,
		&cred.UpdatedAt,
	); err != nil {
		return nil, err
	}

	cred.AccountID = accountID.String
	cred.Email = email.String
	cred.AccessToken = accessToken.String
	cred.RefreshToken = refreshToken.String
	cred.IDToken = idToken.String
	cred.LastRefresh = fromNullTime(lastRefresh)
	cred.ExpiresAt = fromNullTime(expiresAt)
	cred.Status = status.String
	cred.CooldownUntil = fromNullTime(cooldownUntil)
	cred.LastCheckedAt = fromNullTime(lastCheckedAt)
	cred.LastUsedAt = fromNullTime(lastUsedAt)
	cred.LastError = lastError.String
	cred.Remark = remark.String

	return &cred, nil
}

func (s *SQLiteStorage) GetEndpointCredentials(endpointName string) ([]EndpointCredential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT
			id, endpoint_name, provider_type, account_id, email, access_token, refresh_token, id_token,
			last_refresh, expires_at, status, enabled, failure_count, cooldown_until, last_checked_at,
			last_used_at, last_error, remark, created_at, updated_at
		FROM endpoint_credentials
		WHERE endpoint_name=?
		ORDER BY enabled DESC, failure_count ASC, COALESCE(last_used_at, created_at) ASC, updated_at DESC
	`, endpointName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	credentials := make([]EndpointCredential, 0)
	now := time.Now().UTC()
	for rows.Next() {
		cred, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		cred.Status = deriveCredentialStatus(cred, now)
		credentials = append(credentials, *cred)
	}

	return credentials, rows.Err()
}

func (s *SQLiteStorage) GetCredentialByID(id int64) (*EndpointCredential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRow(`
		SELECT
			id, endpoint_name, provider_type, account_id, email, access_token, refresh_token, id_token,
			last_refresh, expires_at, status, enabled, failure_count, cooldown_until, last_checked_at,
			last_used_at, last_error, remark, created_at, updated_at
		FROM endpoint_credentials
		WHERE id=?
	`, id)

	cred, err := scanCredential(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	cred.Status = deriveCredentialStatus(cred, time.Now().UTC())
	return cred, nil
}

func (s *SQLiteStorage) SaveEndpointCredential(cred *EndpointCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cred.ProviderType == "" {
		cred.ProviderType = "codex"
	}
	if cred.Status == "" {
		cred.Status = credentialStatusActive
	}

	result, err := s.db.Exec(`
		INSERT INTO endpoint_credentials (
			endpoint_name, provider_type, account_id, email, access_token, refresh_token, id_token,
			last_refresh, expires_at, status, enabled, failure_count, cooldown_until, last_checked_at,
			last_used_at, last_error, remark
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		cred.EndpointName,
		cred.ProviderType,
		toNullString(cred.AccountID),
		toNullString(cred.Email),
		cred.AccessToken,
		toNullString(cred.RefreshToken),
		toNullString(cred.IDToken),
		toNullTime(cred.LastRefresh),
		toNullTime(cred.ExpiresAt),
		cred.Status,
		cred.Enabled,
		cred.FailureCount,
		toNullTime(cred.CooldownUntil),
		toNullTime(cred.LastCheckedAt),
		toNullTime(cred.LastUsedAt),
		toNullString(cred.LastError),
		toNullString(cred.Remark),
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	cred.ID = id
	return nil
}

func (s *SQLiteStorage) UpdateEndpointCredential(cred *EndpointCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cred.ID <= 0 {
		return fmt.Errorf("credential id is required")
	}
	if cred.ProviderType == "" {
		cred.ProviderType = "codex"
	}
	if cred.Status == "" {
		cred.Status = credentialStatusActive
	}

	result, err := s.db.Exec(`
		UPDATE endpoint_credentials SET
			provider_type=?,
			account_id=?,
			email=?,
			access_token=?,
			refresh_token=?,
			id_token=?,
			last_refresh=?,
			expires_at=?,
			status=?,
			enabled=?,
			failure_count=?,
			cooldown_until=?,
			last_checked_at=?,
			last_used_at=?,
			last_error=?,
			remark=?,
			updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND endpoint_name=?
	`,
		cred.ProviderType,
		toNullString(cred.AccountID),
		toNullString(cred.Email),
		cred.AccessToken,
		toNullString(cred.RefreshToken),
		toNullString(cred.IDToken),
		toNullTime(cred.LastRefresh),
		toNullTime(cred.ExpiresAt),
		cred.Status,
		cred.Enabled,
		cred.FailureCount,
		toNullTime(cred.CooldownUntil),
		toNullTime(cred.LastCheckedAt),
		toNullTime(cred.LastUsedAt),
		toNullString(cred.LastError),
		toNullString(cred.Remark),
		cred.ID,
		cred.EndpointName,
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("credential not found")
	}

	return nil
}

func (s *SQLiteStorage) DeleteEndpointCredential(endpointName string, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`DELETE FROM credential_rate_limits WHERE credential_id=?`, id); err != nil {
		return err
	}
	result, err := s.db.Exec(`DELETE FROM endpoint_credentials WHERE endpoint_name=? AND id=?`, endpointName, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("credential not found")
	}

	return nil
}

func (s *SQLiteStorage) SetEndpointCredentialEnabled(endpointName string, id int64, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		UPDATE endpoint_credentials
		SET enabled=?, updated_at=CURRENT_TIMESTAMP
		WHERE endpoint_name=? AND id=?
	`, enabled, endpointName, id)
	return err
}

func (s *SQLiteStorage) GetTokenPoolStats(endpointName string) (TokenPoolStats, error) {
	credentials, err := s.GetEndpointCredentials(endpointName)
	if err != nil {
		return TokenPoolStats{}, err
	}

	stats := TokenPoolStats{}
	now := time.Now().UTC()
	for i := range credentials {
		classifyCredentialForStats(&stats, &credentials[i], now)
	}
	return stats, nil
}

func (s *SQLiteStorage) GetAllTokenPoolStats() (map[string]TokenPoolStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT
			id, endpoint_name, provider_type, account_id, email, access_token, refresh_token, id_token,
			last_refresh, expires_at, status, enabled, failure_count, cooldown_until, last_checked_at,
			last_used_at, last_error, remark, created_at, updated_at
		FROM endpoint_credentials
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now().UTC()
	stats := make(map[string]TokenPoolStats)
	for rows.Next() {
		cred, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		entry := stats[cred.EndpointName]
		classifyCredentialForStats(&entry, cred, now)
		stats[cred.EndpointName] = entry
	}

	return stats, rows.Err()
}

func (s *SQLiteStorage) GetUsableEndpointCredential(endpointName string, now time.Time) (*EndpointCredential, error) {
	credentials, err := s.GetEndpointCredentials(endpointName)
	if err != nil {
		return nil, err
	}

	for i := range credentials {
		status := deriveCredentialStatus(&credentials[i], now)
		if status == credentialStatusActive || status == credentialStatusExpiring || status == credentialStatusNeedRefresh {
			credentials[i].Status = status
			return &credentials[i], nil
		}
	}

	return nil, nil
}

func (s *SQLiteStorage) MarkCredentialSuccess(id int64, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		UPDATE endpoint_credentials
		SET
			status=?,
			failure_count=0,
			cooldown_until=NULL,
			last_error='',
			last_checked_at=?,
			last_used_at=?,
			updated_at=CURRENT_TIMESTAMP
		WHERE id=?
	`, credentialStatusActive, now.UTC(), now.UTC(), id)
	return err
}

func (s *SQLiteStorage) MarkCredentialFailure(id int64, statusCode int, errMsg string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}

	switch statusCode {
	case 401, 403:
		_, err := s.db.Exec(`
			UPDATE endpoint_credentials
			SET
				status=?,
				failure_count=failure_count+1,
				last_error=?,
				last_checked_at=?,
				updated_at=CURRENT_TIMESTAMP
			WHERE id=?
		`, credentialStatusInvalid, errMsg, now.UTC(), id)
		return err
	case 429:
		cooldownUntil := now.UTC().Add(defaultCooldown)
		_, err := s.db.Exec(`
			UPDATE endpoint_credentials
			SET
				status=?,
				failure_count=failure_count+1,
				cooldown_until=?,
				last_error=?,
				last_checked_at=?,
				updated_at=CURRENT_TIMESTAMP
			WHERE id=?
		`, credentialStatusCooldown, cooldownUntil, errMsg, now.UTC(), id)
		return err
	default:
		_, err := s.db.Exec(`
			UPDATE endpoint_credentials
			SET
				status=?,
				failure_count=failure_count+1,
				last_error=?,
				last_checked_at=?,
				updated_at=CURRENT_TIMESTAMP
			WHERE id=?
		`, credentialStatusActive, errMsg, now.UTC(), id)
		return err
	}
}
