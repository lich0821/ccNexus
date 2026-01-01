package webdav

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/lich0821/ccNexus/internal/logger"
)

// Manager WebDAV 同步管理器
type Manager struct {
	client *Client
}

// NewManager 创建同步管理器
func NewManager(client *Client) *Manager {
	return &Manager{
		client: client,
	}
}

// ensureDBExtension 确保文件名有 .db 后缀
func ensureDBExtension(filename string) string {
	if strings.HasSuffix(filename, ".json") {
		return strings.TrimSuffix(filename, ".json") + ".db"
	}
	if strings.HasSuffix(filename, ".db") {
		return filename
	}
	return filename + ".db"
}

// DatabaseBackupData represents metadata for database backups
type DatabaseBackupData struct {
	BackupTime time.Time `json:"backupTime"` // 备份时间
	Version    string    `json:"version"`    // ccNexus 版本
}

// BackupDatabase backs up the database file to WebDAV
func (m *Manager) BackupDatabase(dbPath string, version string, filename string) error {
	logger.Info("[WebDAV] Starting database backup: %s", filename)

	// Read database file
	logger.Info("[WebDAV] Reading database file: %s", dbPath)
	dbData, err := os.ReadFile(dbPath)
	if err != nil {
		logger.Error("[WebDAV] Failed to read database file: %v", err)
		return err
	}
	logger.Info("[WebDAV] Database file read successfully: %d bytes", len(dbData))

	// Create metadata
	metadata := &DatabaseBackupData{
		BackupTime: time.Now(),
		Version:    version,
	}

	// Serialize metadata
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		logger.Error("[WebDAV] Failed to serialize metadata: %v", err)
		return err
	}

	// Upload database file (use .db extension)
	dbFilename := ensureDBExtension(filename)
	logger.Info("[WebDAV] Uploading database file: %s (%d bytes)", dbFilename, len(dbData))

	if err := m.client.UploadBackup(dbFilename, dbData, true); err != nil {
		logger.Error("[WebDAV] Failed to upload database: %v", err)
		return err
	}
	logger.Info("[WebDAV] Database uploaded successfully")

	// Upload metadata file (use .meta.json extension)
	metaFilename := dbFilename + ".meta.json"
	logger.Info("[WebDAV] Uploading metadata file: %s", metaFilename)
	if err := m.client.UploadBackup(metaFilename, metadataJSON, true); err != nil {
		// Non-fatal: metadata upload failed, but database is uploaded
		logger.Warn("[WebDAV] Failed to upload metadata: %v", err)
	} else {
		logger.Info("[WebDAV] Metadata uploaded successfully")
	}

	logger.Info("[WebDAV] Backup completed successfully")
	return nil
}

// RestoreDatabase downloads and restores the database file from WebDAV
func (m *Manager) RestoreDatabase(filename string, targetPath string) error {
	// Ensure filename has .db extension
	dbFilename := ensureDBExtension(filename)

	// Download database file
	dbData, err := m.client.DownloadBackup(dbFilename, true)
	if err != nil {
		return err
	}

	// Write to target path
	if err := os.WriteFile(targetPath, dbData, 0644); err != nil {
		return err
	}

	return nil
}

// ListConfigBackups 列出配置备份
func (m *Manager) ListConfigBackups() ([]BackupFile, error) {
	// Get all backups (both .json and .db files)
	allBackups, err := m.client.ListBackups(true)
	if err != nil {
		return nil, err
	}

	// Filter to only include .db files (exclude .meta.json files)
	var dbBackups []BackupFile
	for _, backup := range allBackups {
		// Skip metadata files
		if strings.HasSuffix(backup.Filename, ".meta.json") {
			continue
		}
		// Include .db files only
		if strings.HasSuffix(backup.Filename, ".db") {
			dbBackups = append(dbBackups, backup)
		}
	}

	return dbBackups, nil
}

// DeleteConfigBackups 删除配置备份
func (m *Manager) DeleteConfigBackups(filenames []string) error {
	// For each filename, delete both .db and .meta.json files
	var allFilenames []string
	for _, filename := range filenames {
		// Add the main file
		allFilenames = append(allFilenames, filename)

		// Add metadata file if it's a .db file
		if strings.HasSuffix(filename, ".db") {
			allFilenames = append(allFilenames, filename+".meta.json")
		}
	}

	return m.client.DeleteBackups(allFilenames, true)
}
