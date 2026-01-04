package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// safeConfigKeys 定义可以安全跨设备和跨平台备份/恢复的 app_config 配置项。
// 这些配置是平台无关的，不包含设备特定或路径相关的值。
// 不在此列表中的配置项（如 device_id、terminal_*、backup_local_dir、proxy_url 等）
// 是设备/平台特定的，不应在不同设备间同步。
var safeConfigKeys = []string{
	// 应用设置
	"port", "logLevel", "language",
	// 主题设置
	"theme", "themeAuto", "autoLightTheme", "autoDarkTheme",
	// 窗口关闭行为
	"closeWindowBehavior",
	// WebDAV 设置（URL 和凭证是通用的）
	"webdav_url", "webdav_username", "webdav_password", "webdav_configPath", "webdav_statsPath",
	// 备份提供商类型（不包括本地路径）
	"backup_provider",
	// S3 设置（云配置是通用的）
	"backup_s3_endpoint", "backup_s3_region", "backup_s3_bucket", "backup_s3_prefix",
	"backup_s3_accessKey", "backup_s3_secretKey", "backup_s3_sessionToken",
	"backup_s3_useSSL", "backup_s3_forcePathStyle",
	// 更新设置
	"update_autoCheck", "update_checkInterval",
}

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

	// 设置 busy_timeout，当数据库被锁定时等待最多 5 秒
	// 这可以避免并发写入时的 SQLITE_BUSY 错误
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy_timeout: %w", err)
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

func (s *SQLiteStorage) GetEndpoints() ([]Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, name, api_url, api_key, enabled, transformer, model, remark, sort_order, created_at, updated_at FROM endpoints ORDER BY sort_order ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []Endpoint
	for rows.Next() {
		var ep Endpoint
		if err := rows.Scan(&ep.ID, &ep.Name, &ep.APIUrl, &ep.APIKey, &ep.Enabled, &ep.Transformer, &ep.Model, &ep.Remark, &ep.SortOrder, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints, rows.Err()
}

func (s *SQLiteStorage) SaveEndpoint(ep *Endpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`INSERT INTO endpoints (name, api_url, api_key, enabled, transformer, model, remark, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ep.Name, ep.APIUrl, ep.APIKey, ep.Enabled, ep.Transformer, ep.Model, ep.Remark, ep.SortOrder)
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

	_, err := s.db.Exec(`UPDATE endpoints SET api_url=?, api_key=?, enabled=?, transformer=?, model=?, remark=?, sort_order=?, updated_at=CURRENT_TIMESTAMP WHERE name=?`,
		ep.APIUrl, ep.APIKey, ep.Enabled, ep.Transformer, ep.Model, ep.Remark, ep.SortOrder, ep.Name)
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

// DeleteMonthlyStats deletes all daily stats for a specific month
func (s *SQLiteStorage) DeleteMonthlyStats(month string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM daily_stats WHERE strftime('%Y-%m', date) = ?`, month)
	return err
}

// CreateBackupCopy 创建数据库备份副本，只保留安全的 app_config 配置项。
// 设备特定的配置（device_id、终端设置、本地路径等）会被排除。
func (s *SQLiteStorage) CreateBackupCopy(backupPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 使用 VACUUM INTO 创建数据库副本
	_, err := s.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// 打开备份数据库并清理设备特定的 app_config 数据
	backupDB, err := sql.Open("sqlite", backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup: %w", err)
	}
	defer backupDB.Close()

	// 删除所有不在安全列表中的 app_config 条目
	// 这会移除 device_id、terminal_*、backup_local_dir、proxy_url、windowWidth/Height 等
	placeholders := make([]string, len(safeConfigKeys))
	args := make([]interface{}, len(safeConfigKeys))
	for i, key := range safeConfigKeys {
		placeholders[i] = "?"
		args[i] = key
	}
	query := fmt.Sprintf("DELETE FROM app_config WHERE key NOT IN (%s)", strings.Join(placeholders, ","))
	_, err = backupDB.Exec(query, args...)
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
	query := fmt.Sprintf(`SELECT id, name, api_url, api_key, enabled, transformer, model, remark, COALESCE(sort_order, 0) as sort_order, created_at, updated_at FROM %s.endpoints`, dbName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []Endpoint
	for rows.Next() {
		var ep Endpoint
		if err := rows.Scan(&ep.ID, &ep.Name, &ep.APIUrl, &ep.APIKey, &ep.Enabled, &ep.Transformer, &ep.Model, &ep.Remark, &ep.SortOrder, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
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
	if local.Remark != remote.Remark {
		conflicts = append(conflicts, "remark")
	}

	return conflicts
}

// MergeStrategy 定义合并时如何处理冲突
type MergeStrategy string

const (
	MergeStrategyKeepLocal      MergeStrategy = "keep_local"      // 冲突时保留本地，添加新数据
	MergeStrategyOverwriteLocal MergeStrategy = "overwrite_local" // 冲突时用备份覆盖本地
)

// MergeFromBackup 从备份数据库合并数据
func (s *SQLiteStorage) MergeFromBackup(backupDBPath string, strategy MergeStrategy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 挂载备份数据库
	_, err := s.db.Exec(fmt.Sprintf("ATTACH DATABASE '%s' AS backup", backupDBPath))
	if err != nil {
		return fmt.Errorf("failed to attach backup database: %w", err)
	}
	defer s.db.Exec("DETACH DATABASE backup")

	// 开启事务
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. 根据策略合并端点配置
	if err := s.mergeEndpoints(tx, strategy); err != nil {
		return fmt.Errorf("failed to merge endpoints: %w", err)
	}

	// 2. 根据策略合并每日统计数据
	if err := s.mergeDailyStats(tx, strategy); err != nil {
		return fmt.Errorf("failed to merge daily stats: %w", err)
	}

	// 3. 合并安全的 app_config 配置项（仅平台无关的设置）
	if err := s.mergeAppConfig(tx, strategy); err != nil {
		return fmt.Errorf("failed to merge app config: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// mergeEndpoints 根据策略合并端点配置
func (s *SQLiteStorage) mergeEndpoints(tx *sql.Tx, strategy MergeStrategy) error {
	switch strategy {
	case MergeStrategyKeepLocal:
		// 只插入新端点（忽略冲突）
		_, err := tx.Exec(`
			INSERT OR IGNORE INTO endpoints
			(name, api_url, api_key, enabled, transformer, model, remark, sort_order)
			SELECT name, api_url, api_key, enabled, transformer, model, remark, COALESCE(sort_order, 0)
			FROM backup.endpoints
		`)
		return err
	case MergeStrategyOverwriteLocal:
		// 替换已存在的端点
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO endpoints
			(name, api_url, api_key, enabled, transformer, model, remark, sort_order)
			SELECT name, api_url, api_key, enabled, transformer, model, remark, COALESCE(sort_order, 0)
			FROM backup.endpoints
		`)
		return err
	default:
		return fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}

// mergeDailyStats 根据策略合并每日统计数据
// 注意：备份数据的 device_id 会被替换为本地的 device_id，以避免跨设备恢复时产生重复记录
func (s *SQLiteStorage) mergeDailyStats(tx *sql.Tx, strategy MergeStrategy) error {
	// 获取本地 device_id，如果不存在则使用 'default'
	var localDeviceID string
	err := tx.QueryRow(`SELECT COALESCE((SELECT value FROM app_config WHERE key = 'device_id'), 'default')`).Scan(&localDeviceID)
	if err != nil {
		localDeviceID = "default"
	}

	switch strategy {
	case MergeStrategyKeepLocal:
		// 保留本地数据，只插入本地不存在的记录
		// 使用本地 device_id 替代备份的 device_id 以避免重复
		_, err := tx.Exec(`
			INSERT OR IGNORE INTO daily_stats
			(endpoint_name, date, requests, errors, input_tokens, output_tokens, device_id)
			SELECT endpoint_name, date, requests, errors, input_tokens, output_tokens, ?
			FROM backup.daily_stats
		`, localDeviceID)
		return err
	case MergeStrategyOverwriteLocal:
		// 用备份数据覆盖本地数据
		// 步骤1：删除主数据库中的冲突记录（只匹配 endpoint_name 和 date）
		_, err := tx.Exec(`
			DELETE FROM daily_stats
			WHERE EXISTS (
				SELECT 1 FROM backup.daily_stats b
				WHERE b.endpoint_name = daily_stats.endpoint_name
				AND b.date = daily_stats.date
			)
		`)
		if err != nil {
			return err
		}

		// 步骤2：使用本地 device_id 插入备份数据
		_, err = tx.Exec(`
			INSERT INTO daily_stats
			(endpoint_name, date, requests, errors, input_tokens, output_tokens, device_id)
			SELECT endpoint_name, date, requests, errors, input_tokens, output_tokens, ?
			FROM backup.daily_stats
		`, localDeviceID)
		return err
	default:
		return fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}

// mergeAppConfig 根据策略合并安全的 app_config 配置项
// 只有 safeConfigKeys 中的配置会被合并；设备特定的配置会保留本地值
func (s *SQLiteStorage) mergeAppConfig(tx *sql.Tx, strategy MergeStrategy) error {
	// 构建安全配置项的占位符
	placeholders := make([]string, len(safeConfigKeys))
	args := make([]interface{}, len(safeConfigKeys))
	for i, key := range safeConfigKeys {
		placeholders[i] = "?"
		args[i] = key
	}
	keysFilter := strings.Join(placeholders, ",")

	switch strategy {
	case MergeStrategyKeepLocal:
		// 保留本地值，只插入备份中新增的配置项
		query := fmt.Sprintf(`
			INSERT OR IGNORE INTO app_config (key, value)
			SELECT key, value FROM backup.app_config
			WHERE key IN (%s)
		`, keysFilter)
		_, err := tx.Exec(query, args...)
		return err
	case MergeStrategyOverwriteLocal:
		// 用备份值覆盖本地值（仅限安全配置项）
		query := fmt.Sprintf(`
			INSERT OR REPLACE INTO app_config (key, value)
			SELECT key, value FROM backup.app_config
			WHERE key IN (%s)
		`, keysFilter)
		_, err := tx.Exec(query, args...)
		return err
	default:
		return fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}
