package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/storage"
)

type BackupService struct {
	config  *config.Config
	storage *storage.SQLiteStorage
	version string

	webdav *WebDAVService
}

func NewBackupService(cfg *config.Config, s *storage.SQLiteStorage, version string, webdav *WebDAVService) *BackupService {
	return &BackupService{config: cfg, storage: s, version: version, webdav: webdav}
}

func (b *BackupService) UpdateBackupProvider(provider string) error {
	if !isValidBackupProvider(provider) {
		return fmt.Errorf("backup_provider_invalid")
	}

	backup := cloneBackupConfig(b.config.GetBackup())
	if backup == nil {
		backup = &config.BackupConfig{}
	}
	backup.Provider = provider
	b.config.UpdateBackup(backup)

	return b.saveConfig()
}

func (b *BackupService) UpdateLocalBackupDir(dir string) error {
	dir = normalizeUserInput(dir)
	if dir == "" {
		return fmt.Errorf("backup_local_dir_not_set")
	}

	backup := cloneBackupConfig(b.config.GetBackup())
	if backup == nil {
		backup = &config.BackupConfig{}
	}
	backup.Provider = string(BackupProviderLocal)
	backup.Local = &config.LocalBackupConfig{Dir: dir}
	b.config.UpdateBackup(backup)

	return b.saveConfig()
}

func (b *BackupService) UpdateS3BackupConfig(endpoint, region, bucket, prefix, accessKey, secretKey, sessionToken string, useSSL, forcePathStyle bool) error {
	endpoint = normalizeUserInput(endpoint)
	region = normalizeUserInput(region)
	bucket = normalizeUserInput(bucket)
	prefix = normalizeUserInput(prefix)
	accessKey = normalizeUserInput(accessKey)
	secretKey = normalizeUserInput(secretKey)
	sessionToken = normalizeUserInput(sessionToken)

	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		return fmt.Errorf("backup_s3_not_configured")
	}

	backup := cloneBackupConfig(b.config.GetBackup())
	if backup == nil {
		backup = &config.BackupConfig{}
	}
	backup.Provider = string(BackupProviderS3)
	backup.S3 = &config.S3BackupConfig{
		Endpoint:       endpoint,
		Region:         region,
		Bucket:         bucket,
		Prefix:         prefix,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		SessionToken:   sessionToken,
		UseSSL:         useSSL,
		ForcePathStyle: forcePathStyle,
	}
	b.config.UpdateBackup(backup)

	return b.saveConfig()
}

func (b *BackupService) ListBackups(provider string) string {
	provider = normalizeUserInput(provider)
	if provider == "" {
		provider = string(BackupProviderWebDAV)
	}

	switch BackupProvider(provider) {
	case BackupProviderWebDAV:
		if b.webdav == nil {
			return marshalBackupListResult(false, "WebDAV服务未初始化", nil)
		}
		return b.webdav.ListWebDAVBackups()
	case BackupProviderLocal:
		return b.listLocalBackups()
	case BackupProviderS3:
		return b.listS3Backups()
	default:
		return marshalBackupListResult(false, "未知备份类型", nil)
	}
}

func (b *BackupService) DeleteBackups(provider string, filenames []string) error {
	provider = normalizeUserInput(provider)
	if provider == "" {
		provider = string(BackupProviderWebDAV)
	}

	switch BackupProvider(provider) {
	case BackupProviderWebDAV:
		if b.webdav == nil {
			return fmt.Errorf("webdav_not_configured")
		}
		return b.webdav.DeleteWebDAVBackups(filenames)
	case BackupProviderLocal:
		return b.deleteLocalBackups(filenames)
	case BackupProviderS3:
		return b.deleteS3Backups(filenames)
	default:
		return fmt.Errorf("backup_provider_invalid")
	}
}

func (b *BackupService) BackupToProvider(provider, filename string) error {
	provider = normalizeUserInput(provider)
	if provider == "" {
		provider = string(BackupProviderWebDAV)
	}

	switch BackupProvider(provider) {
	case BackupProviderWebDAV:
		if b.webdav == nil {
			return fmt.Errorf("webdav_not_configured")
		}
		return b.webdav.BackupToWebDAV(filename)
	case BackupProviderLocal:
		return b.backupToLocal(filename)
	case BackupProviderS3:
		return b.backupToS3(filename)
	default:
		return fmt.Errorf("backup_provider_invalid")
	}
}

func (b *BackupService) DetectBackupConflict(provider, filename string) string {
	provider = normalizeUserInput(provider)
	if provider == "" {
		provider = string(BackupProviderWebDAV)
	}

	switch BackupProvider(provider) {
	case BackupProviderWebDAV:
		if b.webdav == nil {
			return marshalConflictResult(false, "WebDAV服务未初始化", nil)
		}
		return b.webdav.DetectWebDAVConflict(filename)
	case BackupProviderLocal:
		return b.detectLocalConflict(filename)
	case BackupProviderS3:
		return b.detectS3Conflict(filename)
	default:
		return marshalConflictResult(false, "未知备份类型", nil)
	}
}

func (b *BackupService) RestoreFromProvider(provider, filename, choice string, reloadConfig func(*config.Config) error) error {
	provider = normalizeUserInput(provider)
	if provider == "" {
		provider = string(BackupProviderWebDAV)
	}

	switch BackupProvider(provider) {
	case BackupProviderWebDAV:
		if b.webdav == nil {
			return fmt.Errorf("webdav_not_configured")
		}
		return b.webdav.RestoreFromWebDAV(filename, choice, reloadConfig)
	case BackupProviderLocal:
		return b.restoreFromLocal(filename, choice, reloadConfig)
	case BackupProviderS3:
		return b.restoreFromS3(filename, choice, reloadConfig)
	default:
		return fmt.Errorf("backup_provider_invalid")
	}
}

func (b *BackupService) TestS3Connection(endpoint, region, bucket, prefix, accessKey, secretKey, sessionToken string, useSSL, forcePathStyle bool) string {
	result := b.testS3Connection(endpoint, region, bucket, prefix, accessKey, secretKey, sessionToken, useSSL, forcePathStyle)
	data, _ := json.Marshal(result)
	return string(data)
}

func (b *BackupService) saveConfig() error {
	if b.storage == nil {
		return fmt.Errorf("storage_not_initialized")
	}

	configAdapter := storage.NewConfigStorageAdapter(b.storage)
	if err := b.config.SaveToStorage(configAdapter); err != nil {
		return fmt.Errorf("save_config_failed")
	}
	return nil
}

func isValidBackupProvider(provider string) bool {
	switch BackupProvider(provider) {
	case BackupProviderWebDAV, BackupProviderLocal, BackupProviderS3:
		return true
	default:
		return false
	}
}

func cloneBackupConfig(src *config.BackupConfig) *config.BackupConfig {
	if src == nil {
		return nil
	}

	dst := &config.BackupConfig{Provider: src.Provider}
	if src.Local != nil {
		dst.Local = &config.LocalBackupConfig{Dir: src.Local.Dir}
	}
	if src.S3 != nil {
		dst.S3 = &config.S3BackupConfig{
			Endpoint:       src.S3.Endpoint,
			Region:         src.S3.Region,
			Bucket:         src.S3.Bucket,
			Prefix:         src.S3.Prefix,
			AccessKey:      src.S3.AccessKey,
			SecretKey:      src.S3.SecretKey,
			SessionToken:   src.S3.SessionToken,
			UseSSL:         src.S3.UseSSL,
			ForcePathStyle: src.S3.ForcePathStyle,
		}
	}
	return dst
}

func normalizeUserInput(v string) string {
	return strings.TrimSpace(v)
}
