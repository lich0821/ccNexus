package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (s *SQLiteStorage) UpsertCredentialRateLimits(credentialID int64, data *CodexRateLimitsData, status, errMsg string, updatedAt time.Time) error {
	if credentialID <= 0 {
		return fmt.Errorf("credential id is required")
	}

	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}

	var payload sql.NullString
	if data != nil {
		raw, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal rate limits failed: %w", err)
		}
		payload = toNullString(string(raw))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if payload.Valid {
		_, err := s.db.Exec(`
			INSERT INTO credential_rate_limits (credential_id, snapshot_json, last_status, last_error, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(credential_id) DO UPDATE SET
				snapshot_json=excluded.snapshot_json,
				last_status=excluded.last_status,
				last_error=excluded.last_error,
				updated_at=excluded.updated_at
		`, credentialID, payload, status, toNullString(errMsg), updatedAt.UTC())
		return err
	}

	result, err := s.db.Exec(`
		UPDATE credential_rate_limits
		SET last_status=?, last_error=?, updated_at=?
		WHERE credential_id=?
	`, status, toNullString(errMsg), updatedAt.UTC(), credentialID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		_, err = s.db.Exec(`
			INSERT INTO credential_rate_limits (credential_id, snapshot_json, last_status, last_error, updated_at)
			VALUES (?, NULL, ?, ?, ?)
		`, credentialID, status, toNullString(errMsg), updatedAt.UTC())
		return err
	}
	return nil
}

func (s *SQLiteStorage) GetCredentialRateLimits(credentialID int64) (*CredentialRateLimits, error) {
	if credentialID <= 0 {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRow(`
		SELECT credential_id, snapshot_json, last_status, last_error, updated_at
		FROM credential_rate_limits
		WHERE credential_id=?
	`, credentialID)

	var id int64
	var snapshot sql.NullString
	var status, lastError sql.NullString
	var updatedAt sql.NullTime
	if err := row.Scan(&id, &snapshot, &status, &lastError, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	result := &CredentialRateLimits{
		CredentialID: id,
		Status:       status.String,
		Error:        lastError.String,
		UpdatedAt:    fromNullTime(updatedAt),
	}

	if snapshot.Valid && snapshot.String != "" {
		var data CodexRateLimitsData
		if err := json.Unmarshal([]byte(snapshot.String), &data); err == nil {
			result.Data = &data
		}
	}

	return result, nil
}

func (s *SQLiteStorage) GetCredentialRateLimitsByEndpoint(endpointName string) (map[int64]*CredentialRateLimits, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT r.credential_id, r.snapshot_json, r.last_status, r.last_error, r.updated_at
		FROM credential_rate_limits r
		JOIN endpoint_credentials c ON c.id = r.credential_id
		WHERE c.endpoint_name=?
	`, endpointName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]*CredentialRateLimits)
	for rows.Next() {
		var id int64
		var snapshot sql.NullString
		var status, lastError sql.NullString
		var updatedAt sql.NullTime
		if err := rows.Scan(&id, &snapshot, &status, &lastError, &updatedAt); err != nil {
			return nil, err
		}

		entry := &CredentialRateLimits{
			CredentialID: id,
			Status:       status.String,
			Error:        lastError.String,
			UpdatedAt:    fromNullTime(updatedAt),
		}
		if snapshot.Valid && snapshot.String != "" {
			var data CodexRateLimitsData
			if err := json.Unmarshal([]byte(snapshot.String), &data); err == nil {
				entry.Data = &data
			}
		}
		result[id] = entry
	}

	return result, rows.Err()
}
