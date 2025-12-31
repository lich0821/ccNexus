package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/storage"
)

func (b *BackupService) getLocalDir() (string, error) {
	backup := b.config.GetBackup()
	if backup == nil || backup.Local == nil || backup.Local.Dir == "" {
		return "", fmt.Errorf("backup_local_not_configured")
	}
	return backup.Local.Dir, nil
}

func (b *BackupService) listLocalBackups() string {
	dir, err := b.getLocalDir()
	if err != nil {
		return marshalBackupListResult(false, "本地备份未配置", nil)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return marshalBackupListResult(false, fmt.Sprintf("读取备份目录失败: %v", err), nil)
	}

	var backups []BackupListItem
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "" {
			continue
		}
		if filepath.Ext(name) != ".db" && filepath.Ext(name) != ".json" {
			continue
		}
		if strings.HasSuffix(name, ".meta.json") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupListItem{
			Filename: name,
			Size:     info.Size(),
			ModTime:  info.ModTime(),
		})
	}

	sortBackupsByModTimeDesc(backups)
	return marshalBackupListResult(true, "获取备份列表成功", backups)
}

func (b *BackupService) backupToLocal(filename string) error {
	if b.storage == nil {
		return fmt.Errorf("storage_not_initialized")
	}

	dir, err := b.getLocalDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create_backup_dir_failed")
	}

	filename = ensureDBFilename(filename)
	if filename == "" {
		return fmt.Errorf("filename_required")
	}

	finalPath := filepath.Join(dir, filename)
	tmpPath := finalPath + ".tmp"
	_ = os.Remove(tmpPath)

	if err := b.storage.CreateBackupCopy(tmpPath); err != nil {
		return fmt.Errorf("create_db_backup_failed")
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("backup_write_failed")
	}

	_ = os.WriteFile(finalPath+".meta.json", nowMeta(b.version), 0644)
	return nil
}

func (b *BackupService) detectLocalConflict(filename string) string {
	if b.storage == nil {
		return marshalConflictResult(false, "存储未初始化", nil)
	}

	dir, err := b.getLocalDir()
	if err != nil {
		return marshalConflictResult(false, "本地备份未配置", nil)
	}

	filename = ensureDBFilename(filename)
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); err != nil {
		return marshalConflictResult(false, "备份文件不存在", nil)
	}

	conflicts, err := b.storage.DetectEndpointConflicts(path)
	if err != nil {
		return marshalConflictResult(false, fmt.Sprintf("检测冲突失败: %v", err), nil)
	}

	return marshalConflictResult(true, "", conflicts)
}

func (b *BackupService) restoreFromLocal(filename, choice string, reloadConfig func(*config.Config) error) error {
	if b.storage == nil {
		return fmt.Errorf("storage_not_initialized")
	}

	dir, err := b.getLocalDir()
	if err != nil {
		return err
	}

	filename = ensureDBFilename(filename)
	backupPath := filepath.Join(dir, filename)
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup_file_not_found")
	}

	strategy := storage.MergeStrategyKeepLocal
	if choice == "remote" {
		strategy = storage.MergeStrategyOverwriteLocal
	}

	if err := b.storage.MergeFromBackup(backupPath, strategy); err != nil {
		return fmt.Errorf("merge_data_failed")
	}

	configAdapter := storage.NewConfigStorageAdapter(b.storage)
	newConfig, err := config.LoadFromStorage(configAdapter)
	if err != nil {
		return fmt.Errorf("load_config_failed")
	}
	*b.config = *newConfig

	if err := reloadConfig(newConfig); err != nil {
		return fmt.Errorf("update_proxy_config_failed")
	}

	return nil
}

func (b *BackupService) deleteLocalBackups(filenames []string) error {
	dir, err := b.getLocalDir()
	if err != nil {
		return err
	}

	for _, name := range filenames {
		name = ensureDBFilename(name)
		if name == "" {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
		_ = os.Remove(filepath.Join(dir, name+".meta.json"))
	}
	return nil
}
