package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lich0821/ccNexus/internal/storage"
)

func marshalBackupListResult(success bool, message string, backups []BackupListItem) string {
	if backups == nil {
		backups = []BackupListItem{}
	}
	result := BackupListResult{
		Success: success,
		Message: message,
		Backups: backups,
	}
	data, _ := json.Marshal(result)
	return string(data)
}

func marshalConflictResult(success bool, message string, conflicts []storage.MergeConflict) string {
	result := map[string]interface{}{
		"success": success,
	}
	if message != "" {
		result["message"] = message
	}
	if conflicts != nil {
		result["conflicts"] = conflicts
	}
	data, _ := json.Marshal(result)
	return string(data)
}

func ensureDBFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	filename = filepath.Base(filename)
	if filename == "" {
		return filename
	}

	// Keep legacy .json for backward compatibility
	if strings.HasSuffix(filename, ".json") {
		return filename
	}
	if strings.HasSuffix(filename, ".db") {
		return filename
	}
	return filename + ".db"
}

func tempDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get_home_dir_failed")
	}
	dir := filepath.Join(homeDir, ".ccNexus", "temp")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create_temp_dir_failed")
	}
	return dir, nil
}

func sortBackupsByModTimeDesc(backups []BackupListItem) {
	sort.SliceStable(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})
}

func nowMeta(version string) []byte {
	meta := map[string]interface{}{
		"backupTime": time.Now(),
		"version":    version,
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	return data
}
