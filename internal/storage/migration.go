package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lich0821/ccNexus/internal/logger"
)

func MigrateFromJSON(configPath, statsPath, dbPath string) error {
	if _, err := os.Stat(dbPath); err == nil {
		logger.Info("Database already exists, skipping migration")
		return nil
	}

	configExists := fileExists(configPath)
	statsExists := fileExists(statsPath)

	if !configExists && !statsExists {
		logger.Info("No legacy files found, starting fresh")
		return nil
	}

	logger.Info("Starting migration from JSON to SQLite...")

	storage, err := NewSQLiteStorage(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer storage.Close()

	if configExists {
		if err := migrateConfig(configPath, storage); err != nil {
			return fmt.Errorf("failed to migrate config: %w", err)
		}
	}

	if statsExists {
		if err := migrateStats(statsPath, storage); err != nil {
			return fmt.Errorf("failed to migrate stats: %w", err)
		}
	}

	if err := backupLegacyFiles(configPath, statsPath); err != nil {
		logger.Warn("Failed to backup legacy files: %v", err)
	}

	logger.Info("Migration completed successfully")
	return nil
}

func migrateConfig(configPath string, storage *SQLiteStorage) error {
	logger.Info("Migrating config from %s", configPath)

	config, err := LoadLegacyConfig(configPath)
	if err != nil {
		return err
	}

	for _, ep := range config.Endpoints {
		endpoint := &Endpoint{
			Name:        ep.Name,
			APIUrl:      ep.APIUrl,
			APIKey:      ep.APIKey,
			Enabled:     ep.Enabled,
			Transformer: ep.Transformer,
			Model:       ep.Model,
			Remark:      ep.Remark,
		}
		if endpoint.Transformer == "" {
			endpoint.Transformer = "claude"
		}
		if err := storage.SaveEndpoint(endpoint); err != nil {
			logger.Warn("Failed to migrate endpoint %s: %v", ep.Name, err)
		}
	}

	storage.SetConfig("port", fmt.Sprintf("%d", config.Port))
	storage.SetConfig("logLevel", fmt.Sprintf("%d", config.LogLevel))
	storage.SetConfig("language", config.Language)
	storage.SetConfig("windowWidth", fmt.Sprintf("%d", config.WindowWidth))
	storage.SetConfig("windowHeight", fmt.Sprintf("%d", config.WindowHeight))

	if config.WebDAV != nil {
		storage.SetConfig("webdav_url", config.WebDAV.URL)
		storage.SetConfig("webdav_username", config.WebDAV.Username)
		storage.SetConfig("webdav_password", config.WebDAV.Password)
		storage.SetConfig("webdav_configPath", config.WebDAV.ConfigPath)
		storage.SetConfig("webdav_statsPath", config.WebDAV.StatsPath)
	}

	logger.Info("Config migration completed: %d endpoints", len(config.Endpoints))
	return nil
}

func migrateStats(statsPath string, storage *SQLiteStorage) error {
	logger.Info("Migrating stats from %s", statsPath)

	stats, err := LoadLegacyStats(statsPath)
	if err != nil {
		return err
	}

	deviceID := "migrated"
	count := 0

	for endpointName, epStats := range stats.EndpointStats {
		for date, daily := range epStats.DailyHistory {
			stat := &DailyStat{
				EndpointName: endpointName,
				Date:         date,
				Requests:     daily.Requests,
				Errors:       daily.Errors,
				InputTokens:  daily.InputTokens,
				OutputTokens: daily.OutputTokens,
				DeviceID:     deviceID,
			}
			if err := storage.RecordDailyStat(stat); err != nil {
				logger.Warn("Failed to migrate stat for %s on %s: %v", endpointName, date, err)
			} else {
				count++
			}
		}
	}

	logger.Info("Stats migration completed: %d daily records", count)
	return nil
}

func backupLegacyFiles(configPath, statsPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	backupDir := filepath.Join(homeDir, ".ccNexus", "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102-150405")

	if fileExists(configPath) {
		backupPath := filepath.Join(backupDir, fmt.Sprintf("config.json.%s.bak", timestamp))
		if err := copyFile(configPath, backupPath); err != nil {
			return err
		}
		logger.Info("Backed up config to %s", backupPath)
		// Don't remove the original file - keep it for compatibility
	}

	if fileExists(statsPath) {
		backupPath := filepath.Join(backupDir, fmt.Sprintf("stats.json.%s.bak", timestamp))
		if err := copyFile(statsPath, backupPath); err != nil {
			return err
		}
		logger.Info("Backed up stats to %s", backupPath)
		// Don't remove the original file - keep it for compatibility
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
