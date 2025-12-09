package webdav

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
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

// DatabaseBackupData represents metadata for database backups
type DatabaseBackupData struct {
	BackupTime time.Time `json:"backupTime"` // 备份时间
	Version    string    `json:"version"`    // ccNexus 版本
}

// BackupDatabase backs up the database file to WebDAV
func (m *Manager) BackupDatabase(dbPath string, version string, filename string) error {
	fmt.Printf("[WebDAV] Starting database backup: %s\n", filename)

	// Read database file
	fmt.Printf("[WebDAV] Reading database file: %s\n", dbPath)
	dbData, err := os.ReadFile(dbPath)
	if err != nil {
		return fmt.Errorf("Failed to read database file: %v", err)
	}
	fmt.Printf("[WebDAV] Database file read successfully: %d bytes\n", len(dbData))

	// Create metadata
	metadata := &DatabaseBackupData{
		BackupTime: time.Now(),
		Version:    version,
	}

	// Serialize metadata
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize metadata: %v", err)
	}

	// Upload database file (use .db extension)
	dbFilename := filename
	if len(dbFilename) > 5 && dbFilename[len(dbFilename)-5:] == ".json" {
		// Replace .json with .db
		dbFilename = dbFilename[:len(dbFilename)-5] + ".db"
	} else if len(dbFilename) < 3 || dbFilename[len(dbFilename)-3:] != ".db" {
		// Add .db extension if not present
		dbFilename = dbFilename + ".db"
	}
	fmt.Printf("[WebDAV] Uploading database file: %s (%d bytes)\n", dbFilename, len(dbData))

	if err := m.client.UploadBackup(dbFilename, dbData, true); err != nil {
		fmt.Printf("[WebDAV] Failed to upload database: %v\n", err)
		return fmt.Errorf("Failed to upload database: %v", err)
	}
	fmt.Printf("[WebDAV] Database uploaded successfully\n")

	// Upload metadata file (use .meta.json extension)
	metaFilename := dbFilename + ".meta.json"
	fmt.Printf("[WebDAV] Uploading metadata file: %s\n", metaFilename)
	if err := m.client.UploadBackup(metaFilename, metadataJSON, true); err != nil {
		// Non-fatal: metadata upload failed, but database is uploaded
		// Log warning but don't fail the backup
		fmt.Printf("[WebDAV] Warning: Failed to upload metadata: %v\n", err)
	} else {
		fmt.Printf("[WebDAV] Metadata uploaded successfully\n")
	}

	fmt.Printf("[WebDAV] Backup completed successfully\n")
	return nil
}

// RestoreDatabase downloads and restores the database file from WebDAV
func (m *Manager) RestoreDatabase(filename string, targetPath string) error {
	// Ensure filename has .db extension
	dbFilename := filename
	if len(dbFilename) > 5 && dbFilename[len(dbFilename)-5:] == ".json" {
		// Replace .json with .db
		dbFilename = dbFilename[:len(dbFilename)-5] + ".db"
	} else if len(dbFilename) < 3 || dbFilename[len(dbFilename)-3:] != ".db" {
		// Add .db extension if not present
		dbFilename = dbFilename + ".db"
	}

	// Download database file
	dbData, err := m.client.DownloadBackup(dbFilename, true)
	if err != nil {
		return fmt.Errorf("Failed to download database: %v", err)
	}

	// Write to target path
	if err := os.WriteFile(targetPath, dbData, 0644); err != nil {
		return fmt.Errorf("Failed to write database file: %v", err)
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
		if len(backup.Filename) > 10 && backup.Filename[len(backup.Filename)-10:] == ".meta.json" {
			continue
		}
		// Include .db files
		if len(backup.Filename) > 3 && backup.Filename[len(backup.Filename)-3:] == ".db" {
			dbBackups = append(dbBackups, backup)
		}
		// Also include legacy .json files for backward compatibility
		if len(backup.Filename) > 5 && backup.Filename[len(backup.Filename)-5:] == ".json" {
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
		if len(filename) > 3 && filename[len(filename)-3:] == ".db" {
			metaFilename := filename + ".meta.json"
			allFilenames = append(allFilenames, metaFilename)
		}
	}

	return m.client.DeleteBackups(allFilenames, true)
}
