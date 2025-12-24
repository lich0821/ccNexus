package storage

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db     *sql.DB
	dbPath string
	mu     sync.RWMutex
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	s := &SQLiteStorage{
		db:     db,
		dbPath: dbPath,
	}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS endpoints (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		api_url TEXT NOT NULL,
		api_key TEXT NOT NULL,
		enabled BOOLEAN DEFAULT TRUE,
		transformer TEXT DEFAULT 'claude',
		model TEXT,
		remark TEXT,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS daily_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		endpoint_name TEXT NOT NULL,
		date TEXT NOT NULL,
		requests INTEGER DEFAULT 0,
		errors INTEGER DEFAULT 0,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		device_id TEXT DEFAULT 'default',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(endpoint_name, date, device_id)
	);

	CREATE TABLE IF NOT EXISTS app_config (
		key TEXT PRIMARY KEY,
		value TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats(date);
	CREATE INDEX IF NOT EXISTS idx_daily_stats_endpoint ON daily_stats(endpoint_name);
	CREATE INDEX IF NOT EXISTS idx_daily_stats_device ON daily_stats(device_id);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Migration: Add sort_order column if it doesn't exist
	if err := s.migrateSortOrder(); err != nil {
		return err
	}

	// Migration: Add model_redirects column if it doesn't exist
	if err := s.migrateModelRedirects(); err != nil {
		return err
	}

	return nil
}

// migrateSortOrder adds the sort_order column to existing databases
func (s *SQLiteStorage) migrateSortOrder() error {
	// Check if sort_order column exists
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('endpoints') WHERE name='sort_order'`).Scan(&count)
	if err != nil {
		return err
	}

	// If column doesn't exist, add it and set default values
	if count == 0 {
		// Add the column
		if _, err := s.db.Exec(`ALTER TABLE endpoints ADD COLUMN sort_order INTEGER DEFAULT 0`); err != nil {
			return err
		}

		// Set sort_order for existing endpoints based on their current ID order
		if _, err := s.db.Exec(`UPDATE endpoints SET sort_order = id WHERE sort_order = 0`); err != nil {
			return err
		}
	}

	return nil
}

// migrateModelRedirects adds the model_redirects column to existing databases
func (s *SQLiteStorage) migrateModelRedirects() error {
	// Check if model_redirects column exists
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('endpoints') WHERE name='model_redirects'`).Scan(&count)
	if err != nil {
		return err
	}

	// If column doesn't exist, add it with default value
	if count == 0 {
		if _, err := s.db.Exec(`ALTER TABLE endpoints ADD COLUMN model_redirects TEXT DEFAULT ''`); err != nil {
			return err
		}
	}

	// Update existing NULL values to empty string (for both new and existing columns)
	if _, err := s.db.Exec(`UPDATE endpoints SET model_redirects = '' WHERE model_redirects IS NULL`); err != nil {
		return err
	}

	return nil
}

func (s *SQLiteStorage) GetEndpoints() ([]Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, name, api_url, api_key, enabled, transformer, model, model_redirects, remark, sort_order, created_at, updated_at FROM endpoints ORDER BY sort_order ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []Endpoint
	for rows.Next() {
		var ep Endpoint
		var modelRedirects sql.NullString
		if err := rows.Scan(&ep.ID, &ep.Name, &ep.APIUrl, &ep.APIKey, &ep.Enabled, &ep.Transformer, &ep.Model, &modelRedirects, &ep.Remark, &ep.SortOrder, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, err
		}
		// Handle NULL values
		if modelRedirects.Valid {
			ep.ModelRedirects = modelRedirects.String
		} else {
			ep.ModelRedirects = ""
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints, rows.Err()
}

func (s *SQLiteStorage) SaveEndpoint(ep *Endpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`INSERT INTO endpoints (name, api_url, api_key, enabled, transformer, model, model_redirects, remark, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ep.Name, ep.APIUrl, ep.APIKey, ep.Enabled, ep.Transformer, ep.Model, ep.ModelRedirects, ep.Remark, ep.SortOrder)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	ep.ID = id
	return nil
}

func (s *SQLiteStorage) UpdateEndpoint(ep *Endpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`UPDATE endpoints SET api_url=?, api_key=?, enabled=?, transformer=?, model=?, model_redirects=?, remark=?, sort_order=?, updated_at=CURRENT_TIMESTAMP WHERE name=?`,
		ep.APIUrl, ep.APIKey, ep.Enabled, ep.Transformer, ep.Model, ep.ModelRedirects, ep.Remark, ep.SortOrder, ep.Name)
	return err
}

func (s *SQLiteStorage) DeleteEndpoint(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM endpoints WHERE name=?`, name)
	return err
}

func (s *SQLiteStorage) RecordDailyStat(stat *DailyStat) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO daily_stats (endpoint_name, date, requests, errors, input_tokens, output_tokens, device_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(endpoint_name, date, device_id) DO UPDATE SET
			requests = requests + excluded.requests,
			errors = errors + excluded.errors,
			input_tokens = input_tokens + excluded.input_tokens,
			output_tokens = output_tokens + excluded.output_tokens
	`, stat.EndpointName, stat.Date, stat.Requests, stat.Errors, stat.InputTokens, stat.OutputTokens, stat.DeviceID)

	return err
}

func (s *SQLiteStorage) GetDailyStats(endpointName, startDate, endDate string) ([]DailyStat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, endpoint_name, date, SUM(requests), SUM(errors), SUM(input_tokens), SUM(output_tokens), device_id, created_at
		FROM daily_stats WHERE endpoint_name=? AND date>=? AND date<=? GROUP BY date ORDER BY date DESC`

	rows, err := s.db.Query(query, endpointName, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStat
	for rows.Next() {
		var stat DailyStat
		if err := rows.Scan(&stat.ID, &stat.EndpointName, &stat.Date, &stat.Requests, &stat.Errors, &stat.InputTokens, &stat.OutputTokens, &stat.DeviceID, &stat.CreatedAt); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

func (s *SQLiteStorage) GetAllStats() (map[string][]DailyStat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, endpoint_name, date, SUM(requests), SUM(errors), SUM(input_tokens), SUM(output_tokens), device_id, created_at
		FROM daily_stats GROUP BY endpoint_name, date ORDER BY date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]DailyStat)
	for rows.Next() {
		var stat DailyStat
		if err := rows.Scan(&stat.ID, &stat.EndpointName, &stat.Date, &stat.Requests, &stat.Errors, &stat.InputTokens, &stat.OutputTokens, &stat.DeviceID, &stat.CreatedAt); err != nil {
			return nil, err
		}
		result[stat.EndpointName] = append(result[stat.EndpointName], stat)
	}

	return result, rows.Err()
}

func (s *SQLiteStorage) GetConfig(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var value string
	err := s.db.QueryRow(`SELECT value FROM app_config WHERE key=?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *SQLiteStorage) SetConfig(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`INSERT INTO app_config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP`, key, value)
	return err
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

func (s *SQLiteStorage) GetTotalStats() (int, map[string]*EndpointStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT endpoint_name, SUM(requests), SUM(errors), SUM(input_tokens), SUM(output_tokens)
		FROM daily_stats GROUP BY endpoint_name`

	rows, err := s.db.Query(query)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	result := make(map[string]*EndpointStats)
	totalRequests := 0

	for rows.Next() {
		var endpointName string
		var requests, errors int
		var inputTokens, outputTokens int64

		if err := rows.Scan(&endpointName, &requests, &errors, &inputTokens, &outputTokens); err != nil {
			return 0, nil, err
		}

		result[endpointName] = &EndpointStats{
			Requests:     requests,
			Errors:       errors,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		}
		totalRequests += requests
	}

	return totalRequests, result, rows.Err()
}

func (s *SQLiteStorage) GetEndpointTotalStats(endpointName string) (*EndpointStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT SUM(requests), SUM(errors), SUM(input_tokens), SUM(output_tokens)
		FROM daily_stats WHERE endpoint_name=?`

	var requests, errors int
	var inputTokens, outputTokens int64

	err := s.db.QueryRow(query, endpointName).Scan(&requests, &errors, &inputTokens, &outputTokens)
	if err == sql.ErrNoRows {
		return &EndpointStats{}, nil
	}
	if err != nil {
		return nil, err
	}

	return &EndpointStats{
		Requests:     requests,
		Errors:       errors,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}, nil
}

// GetOrCreateDeviceID returns the device ID, creating one if it doesn't exist
func (s *SQLiteStorage) GetOrCreateDeviceID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try to get existing device ID
	var deviceID string
	err := s.db.QueryRow(`SELECT value FROM app_config WHERE key = 'device_id'`).Scan(&deviceID)

	if err == nil && deviceID != "" {
		return deviceID, nil
	}

	// Generate new device ID
	deviceID = generateDeviceID()

	// Save to database
	_, err = s.db.Exec(`INSERT OR REPLACE INTO app_config (key, value) VALUES ('device_id', ?)`, deviceID)
	if err != nil {
		return "", err
	}

	return deviceID, nil
}

func generateDeviceID() string {
	// Use timestamp + random string for uniqueness
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("device-%x", timestamp)[:16]
}

func GenerateDeviceID() string {
	return generateDeviceID()
}

// GetDBPath returns the database file path
func (s *SQLiteStorage) GetDBPath() string {
	return s.dbPath
}

// GetArchiveMonths returns a list of all months that have data
func (s *SQLiteStorage) GetArchiveMonths() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT DISTINCT strftime('%Y-%m', date) as month
		FROM daily_stats
		WHERE date IS NOT NULL AND date != ''
		ORDER BY month DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var months []string
	for rows.Next() {
		var month string
		if err := rows.Scan(&month); err != nil {
			return nil, err
		}
		months = append(months, month)
	}

	return months, rows.Err()
}

// MonthlyArchiveData represents archive data for a specific month
type MonthlyArchiveData struct {
	Month        string
	EndpointName string
	Date         string
	Requests     int
	Errors       int
	InputTokens  int
	OutputTokens int
}

// GetMonthlyArchiveData returns all daily stats for a specific month
func (s *SQLiteStorage) GetMonthlyArchiveData(month string) ([]MonthlyArchiveData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT endpoint_name, date, SUM(requests), SUM(errors), SUM(input_tokens), SUM(output_tokens)
		FROM daily_stats
		WHERE strftime('%Y-%m', date) = ?
		GROUP BY endpoint_name, date
		ORDER BY date DESC, endpoint_name`

	rows, err := s.db.Query(query, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MonthlyArchiveData
	for rows.Next() {
		var data MonthlyArchiveData
		data.Month = month
		if err := rows.Scan(&data.EndpointName, &data.Date, &data.Requests, &data.Errors, &data.InputTokens, &data.OutputTokens); err != nil {
			return nil, err
		}
		results = append(results, data)
	}

	return results, rows.Err()
}

// CreateBackupCopy creates a backup copy of the database without app_config data
func (s *SQLiteStorage) CreateBackupCopy(backupPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use VACUUM INTO to create a copy
	_, err := s.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Open the backup and remove app_config data
	backupDB, err := sql.Open("sqlite", backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup: %w", err)
	}
	defer backupDB.Close()

	// Delete app_config data (device-specific settings)
	_, err = backupDB.Exec("DELETE FROM app_config")
	if err != nil {
		return fmt.Errorf("failed to clean app_config: %w", err)
	}

	return nil
}

// MergeConflict represents an endpoint merge conflict
type MergeConflict struct {
	EndpointName   string   `json:"endpointName"`
	ConflictFields []string `json:"conflictFields"`
	LocalEndpoint  Endpoint `json:"localEndpoint"`
	RemoteEndpoint Endpoint `json:"remoteEndpoint"`
}

// DetectEndpointConflicts detects conflicts between local and remote endpoints
func (s *SQLiteStorage) DetectEndpointConflicts(remoteDBPath string) ([]MergeConflict, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Attach remote database
	_, err := s.db.Exec(fmt.Sprintf("ATTACH DATABASE '%s' AS remote", remoteDBPath))
	if err != nil {
		return nil, fmt.Errorf("failed to attach remote database: %w", err)
	}
	defer s.db.Exec("DETACH DATABASE remote")

	// Get local endpoints
	localEndpoints, err := s.getEndpointsFromDB(s.db, "main")
	if err != nil {
		return nil, err
	}

	// Get remote endpoints
	remoteEndpoints, err := s.getEndpointsFromDB(s.db, "remote")
	if err != nil {
		return nil, err
	}

	// Build local endpoint map
	localMap := make(map[string]Endpoint)
	for _, ep := range localEndpoints {
		localMap[ep.Name] = ep
	}

	// Detect conflicts
	var conflicts []MergeConflict
	for _, remote := range remoteEndpoints {
		if local, exists := localMap[remote.Name]; exists {
			// Check for differences
			conflictFields := compareEndpoints(local, remote)
			if len(conflictFields) > 0 {
				conflicts = append(conflicts, MergeConflict{
					EndpointName:   remote.Name,
					ConflictFields: conflictFields,
					LocalEndpoint:  local,
					RemoteEndpoint: remote,
				})
			}
		}
	}

	return conflicts, nil
}

// getEndpointsFromDB gets endpoints from a specific database (main or attached)
func (s *SQLiteStorage) getEndpointsFromDB(db *sql.DB, dbName string) ([]Endpoint, error) {
	query := fmt.Sprintf(`SELECT id, name, api_url, api_key, enabled, transformer, model, COALESCE(model_redirects, '') as model_redirects, remark, COALESCE(sort_order, 0) as sort_order, created_at, updated_at FROM %s.endpoints`, dbName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []Endpoint
	for rows.Next() {
		var ep Endpoint
		if err := rows.Scan(&ep.ID, &ep.Name, &ep.APIUrl, &ep.APIKey, &ep.Enabled, &ep.Transformer, &ep.Model, &ep.ModelRedirects, &ep.Remark, &ep.SortOrder, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints, rows.Err()
}

// compareEndpoints compares two endpoints and returns conflicting fields
func compareEndpoints(local, remote Endpoint) []string {
	var conflicts []string

	if local.APIUrl != remote.APIUrl {
		conflicts = append(conflicts, "apiUrl")
	}
	if local.APIKey != remote.APIKey {
		conflicts = append(conflicts, "apiKey")
	}
	if local.Enabled != remote.Enabled {
		conflicts = append(conflicts, "enabled")
	}
	if local.Transformer != remote.Transformer {
		conflicts = append(conflicts, "transformer")
	}
	if local.Model != remote.Model {
		conflicts = append(conflicts, "model")
	}
	if local.ModelRedirects != remote.ModelRedirects {
		conflicts = append(conflicts, "modelRedirects")
	}
	if local.Remark != remote.Remark {
		conflicts = append(conflicts, "remark")
	}

	return conflicts
}

// MergeStrategy defines how to handle conflicts during merge
type MergeStrategy string

const (
	MergeStrategyKeepLocal      MergeStrategy = "keep_local"      // Keep local on conflict, add new
	MergeStrategyOverwriteLocal MergeStrategy = "overwrite_local" // Overwrite local on conflict
)

// MergeFromBackup merges data from a backup database
func (s *SQLiteStorage) MergeFromBackup(backupDBPath string, strategy MergeStrategy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Attach backup database
	_, err := s.db.Exec(fmt.Sprintf("ATTACH DATABASE '%s' AS backup", backupDBPath))
	if err != nil {
		return fmt.Errorf("failed to attach backup database: %w", err)
	}
	defer s.db.Exec("DETACH DATABASE backup")

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Merge endpoints based on strategy
	if err := s.mergeEndpoints(tx, strategy); err != nil {
		return fmt.Errorf("failed to merge endpoints: %w", err)
	}

	// 2. Merge daily_stats based on strategy
	if err := s.mergeDailyStats(tx, strategy); err != nil {
		return fmt.Errorf("failed to merge daily stats: %w", err)
	}

	// 3. Do NOT merge app_config (keep local device-specific settings)

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// mergeEndpoints merges endpoints based on strategy
func (s *SQLiteStorage) mergeEndpoints(tx *sql.Tx, strategy MergeStrategy) error {
	switch strategy {
	case MergeStrategyKeepLocal:
		// Insert only new endpoints (ignore conflicts)
		_, err := tx.Exec(`
			INSERT OR IGNORE INTO endpoints
			(name, api_url, api_key, enabled, transformer, model, model_redirects, remark, sort_order)
			SELECT name, api_url, api_key, enabled, transformer, model, COALESCE(model_redirects, ''), remark, COALESCE(sort_order, 0)
			FROM backup.endpoints
		`)
		return err
	case MergeStrategyOverwriteLocal:
		// Replace existing endpoints
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO endpoints
			(name, api_url, api_key, enabled, transformer, model, model_redirects, remark, sort_order)
			SELECT name, api_url, api_key, enabled, transformer, model, COALESCE(model_redirects, ''), remark, COALESCE(sort_order, 0)
			FROM backup.endpoints
		`)
		return err
	default:
		return fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}

// mergeDailyStats merges daily stats based on strategy
func (s *SQLiteStorage) mergeDailyStats(tx *sql.Tx, strategy MergeStrategy) error {
	switch strategy {
	case MergeStrategyKeepLocal:
		// Keep local data, only insert records that don't exist locally
		_, err := tx.Exec(`
			INSERT OR IGNORE INTO daily_stats
			(endpoint_name, date, requests, errors, input_tokens, output_tokens, device_id)
			SELECT endpoint_name, date, requests, errors, input_tokens, output_tokens, device_id
			FROM backup.daily_stats
		`)
		return err
	case MergeStrategyOverwriteLocal:
		// Overwrite local data with backup data
		// Step 1: Delete conflicting rows from main database
		_, err := tx.Exec(`
			DELETE FROM daily_stats
			WHERE EXISTS (
				SELECT 1 FROM backup.daily_stats b
				WHERE b.endpoint_name = daily_stats.endpoint_name
				AND b.date = daily_stats.date
				AND b.device_id = daily_stats.device_id
			)
		`)
		if err != nil {
			return err
		}

		// Step 2: Insert backup data directly (no accumulation)
		_, err = tx.Exec(`
			INSERT INTO daily_stats
			(endpoint_name, date, requests, errors, input_tokens, output_tokens, device_id)
			SELECT endpoint_name, date, requests, errors, input_tokens, output_tokens, device_id
			FROM backup.daily_stats
		`)
		return err
	default:
		return fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}
