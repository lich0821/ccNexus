package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/storage"
	"github.com/lich0821/ccNexus/internal/webdav"
)

// WebDAVService handles WebDAV backup/restore operations
type WebDAVService struct {
	config  *config.Config
	storage *storage.SQLiteStorage
	version string
}

// NewWebDAVService creates a new WebDAVService
func NewWebDAVService(cfg *config.Config, s *storage.SQLiteStorage, version string) *WebDAVService {
	return &WebDAVService{config: cfg, storage: s, version: version}
}

// UpdateWebDAVConfig updates the WebDAV configuration
func (w *WebDAVService) UpdateWebDAVConfig(url, username, password string) error {
	webdavConfig := &config.WebDAVConfig{
		URL:        url,
		Username:   username,
		Password:   password,
		ConfigPath: "/ccNexus/config",
		StatsPath:  "/ccNexus/stats",
	}

	w.config.UpdateWebDAV(webdavConfig)

	if w.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(w.storage)
		if err := w.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save WebDAV config: %w", err)
		}
	}

	logger.Info("WebDAV configuration updated: %s", url)
	return nil
}

// TestWebDAVConnection tests the WebDAV connection with provided credentials
func (w *WebDAVService) TestWebDAVConnection(url, username, password string) string {
	webdavCfg := &config.WebDAVConfig{
		URL:      url,
		Username: username,
		Password: password,
	}

	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("创建WebDAV客户端失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	testResult := client.TestConnection()
	data, _ := json.Marshal(testResult)
	return string(data)
}

// BackupToWebDAV backs up configuration and stats to WebDAV
func (w *WebDAVService) BackupToWebDAV(filename string) error {
	logger.Info("Starting backup process for file: %s", filename)

	webdavCfg := w.config.GetWebDAV()
	if webdavCfg == nil {
		logger.Error("WebDAV configuration is not set")
		return fmt.Errorf("webdav_not_configured")
	}

	if w.storage == nil {
		logger.Error("Storage is not initialized")
		return fmt.Errorf("storage_not_initialized")
	}

	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		logger.Error("Failed to create WebDAV client: %v", err)
		return fmt.Errorf("webdav_client_failed")
	}

	manager := webdav.NewManager(client)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Failed to get home directory: %v", err)
		return fmt.Errorf("get_home_dir_failed")
	}
	tempDir := filepath.Join(homeDir, ".ccNexus", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		logger.Error("Failed to create temp directory: %v", err)
		return fmt.Errorf("create_temp_dir_failed")
	}
	tempBackupPath := filepath.Join(tempDir, "backup_temp.db")

	if _, err := os.Stat(tempBackupPath); err == nil {
		os.Remove(tempBackupPath)
	}

	defer func() {
		os.Remove(tempBackupPath)
		os.RemoveAll(tempDir)
	}()

	logger.Info("Creating database backup copy (excluding app_config)...")
	if err := w.storage.CreateBackupCopy(tempBackupPath); err != nil {
		logger.Error("Failed to create database backup: %v", err)
		return fmt.Errorf("create_db_backup_failed")
	}

	logger.Info("Uploading backup to WebDAV (version: %s)...", w.version)
	if err := manager.BackupDatabase(tempBackupPath, w.version, filename); err != nil {
		logger.Error("Failed to upload backup to WebDAV: %v", err)
		return fmt.Errorf("backup_upload_failed")
	}

	logger.Info("Backup created successfully: %s", filename)
	return nil
}

// RestoreFromWebDAV restores configuration and stats from WebDAV
func (w *WebDAVService) RestoreFromWebDAV(filename, choice string, reloadConfig func(*config.Config) error) error {
	webdavCfg := w.config.GetWebDAV()
	if webdavCfg == nil {
		return fmt.Errorf("webdav_not_configured")
	}

	if w.storage == nil {
		return fmt.Errorf("storage_not_initialized")
	}

	if choice == "keep_local" {
		logger.Info("User chose to merge configuration (keep local on conflicts)")
	}

	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		return fmt.Errorf("webdav_client_failed")
	}

	manager := webdav.NewManager(client)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get_home_dir_failed")
	}
	tempDir := filepath.Join(homeDir, ".ccNexus", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("create_temp_dir_failed")
	}
	tempRestorePath := filepath.Join(tempDir, "restore_temp.db")
	defer os.Remove(tempRestorePath)
	defer os.RemoveAll(tempDir)

	if err := manager.RestoreDatabase(filename, tempRestorePath); err != nil {
		return fmt.Errorf("restore_download_failed")
	}

	var strategy storage.MergeStrategy
	if choice == "remote" {
		strategy = storage.MergeStrategyOverwriteLocal
	} else {
		strategy = storage.MergeStrategyKeepLocal
	}

	if err := w.storage.MergeFromBackup(tempRestorePath, strategy); err != nil {
		logger.Error("合并备份数据失败: %v", err)
		return fmt.Errorf("merge_data_failed")
	}

	configAdapter := storage.NewConfigStorageAdapter(w.storage)
	newConfig, err := config.LoadFromStorage(configAdapter)
	if err != nil {
		return fmt.Errorf("load_config_failed")
	}

	*w.config = *newConfig

	if err := reloadConfig(newConfig); err != nil {
		return fmt.Errorf("update_proxy_config_failed")
	}

	logger.Info("Configuration and statistics restored from: %s", filename)
	return nil
}

// ListWebDAVBackups lists all backups on WebDAV server
func (w *WebDAVService) ListWebDAVBackups() string {
	logger.Info("Listing WebDAV backups...")

	webdavCfg := w.config.GetWebDAV()
	if webdavCfg == nil {
		result := map[string]interface{}{
			"success": false,
			"message": "WebDAV未配置",
			"backups": []interface{}{},
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("创建WebDAV客户端失败: %v", err),
			"backups": []interface{}{},
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	manager := webdav.NewManager(client)

	backups, err := manager.ListConfigBackups()
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("获取备份列表失败: %v", err),
			"backups": []interface{}{},
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	logger.Info("Found %d backup(s)", len(backups))

	result := map[string]interface{}{
		"success": true,
		"message": "获取备份列表成功",
		"backups": backups,
	}
	data, _ := json.Marshal(result)
	return string(data)
}

// DeleteWebDAVBackups deletes backups from WebDAV server
func (w *WebDAVService) DeleteWebDAVBackups(filenames []string) error {
	webdavCfg := w.config.GetWebDAV()
	if webdavCfg == nil {
		return fmt.Errorf("webdav_not_configured")
	}

	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		return fmt.Errorf("webdav_client_failed")
	}

	manager := webdav.NewManager(client)

	if err := manager.DeleteConfigBackups(filenames); err != nil {
		return fmt.Errorf("delete_backup_failed")
	}

	logger.Info("Backups deleted: %v", filenames)
	return nil
}

// DetectWebDAVConflict detects conflicts between local and remote config
func (w *WebDAVService) DetectWebDAVConflict(filename string) string {
	webdavCfg := w.config.GetWebDAV()
	if webdavCfg == nil {
		result := map[string]interface{}{
			"success": false,
			"message": "WebDAV未配置",
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	if w.storage == nil {
		result := map[string]interface{}{
			"success": false,
			"message": "存储未初始化",
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("创建WebDAV客户端失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	manager := webdav.NewManager(client)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("获取用户目录失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}
	tempDir := filepath.Join(homeDir, ".ccNexus", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("创建临时目录失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}
	tempRestorePath := filepath.Join(tempDir, "conflict_check_temp.db")
	defer os.Remove(tempRestorePath)
	defer os.RemoveAll(tempDir)

	if err := manager.RestoreDatabase(filename, tempRestorePath); err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("下载备份失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	conflicts, err := w.storage.DetectEndpointConflicts(tempRestorePath)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("检测冲突失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	result := map[string]interface{}{
		"success":   true,
		"conflicts": conflicts,
	}
	data, _ := json.Marshal(result)
	return string(data)
}
