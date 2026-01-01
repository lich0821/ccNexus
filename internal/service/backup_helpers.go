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

// tempDirUnique 创建一个唯一的临时子目录，返回目录路径和清理函数
// 调用者应在操作完成后调用 cleanup 函数清理整个子目录
func tempDirUnique(prefix string) (dir string, cleanup func(), err error) {
	baseDir, err := tempDir()
	if err != nil {
		return "", nil, err
	}
	// 使用时间戳创建唯一子目录
	subDir := filepath.Join(baseDir, fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano()))
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return "", nil, fmt.Errorf("create_temp_dir_failed")
	}
	cleanup = func() {
		_ = os.RemoveAll(subDir)
	}
	return subDir, cleanup, nil
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
