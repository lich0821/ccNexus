package storage

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	s := &SQLiteStorage{db: db}
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

	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStorage) GetEndpoints() ([]Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, name, api_url, api_key, enabled, transformer, model, remark, created_at, updated_at FROM endpoints`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []Endpoint
	for rows.Next() {
		var ep Endpoint
		if err := rows.Scan(&ep.ID, &ep.Name, &ep.APIUrl, &ep.APIKey, &ep.Enabled, &ep.Transformer, &ep.Model, &ep.Remark, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints, rows.Err()
}

func (s *SQLiteStorage) SaveEndpoint(ep *Endpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`INSERT INTO endpoints (name, api_url, api_key, enabled, transformer, model, remark) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ep.Name, ep.APIUrl, ep.APIKey, ep.Enabled, ep.Transformer, ep.Model, ep.Remark)
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

	_, err := s.db.Exec(`UPDATE endpoints SET api_url=?, api_key=?, enabled=?, transformer=?, model=?, remark=?, updated_at=CURRENT_TIMESTAMP WHERE name=?`,
		ep.APIUrl, ep.APIKey, ep.Enabled, ep.Transformer, ep.Model, ep.Remark, ep.Name)
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

func GenerateDeviceID() string {
	return fmt.Sprintf("default-%s", time.Now().Format("20060102"))
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
