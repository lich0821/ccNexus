package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func (b *BackupService) newS3ClientFromConfig(cfg *config.S3BackupConfig) (*minio.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("backup_s3_not_configured")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("backup_s3_not_configured")
	}
	secure := cfg.UseSSL

	endpoint = strings.TrimRight(endpoint, "/")
	if strings.Contains(endpoint, "://") {
		u, err := url.Parse(endpoint)
		if err != nil || u.Host == "" {
			return nil, fmt.Errorf("backup_s3_endpoint_invalid")
		}
		if u.Path != "" && u.Path != "/" {
			return nil, fmt.Errorf("backup_s3_endpoint_path_not_supported")
		}
		if u.RawQuery != "" || u.Fragment != "" {
			return nil, fmt.Errorf("backup_s3_endpoint_invalid")
		}
		switch u.Scheme {
		case "http":
			secure = false
		case "https":
			secure = true
		default:
			return nil, fmt.Errorf("backup_s3_endpoint_invalid")
		}
		endpoint = u.Host
	}

	endpoint = strings.TrimRight(endpoint, "/")
	if endpoint == "" || strings.ContainsAny(endpoint, "?#") {
		return nil, fmt.Errorf("backup_s3_endpoint_invalid")
	}
	if strings.Contains(endpoint, "/") {
		return nil, fmt.Errorf("backup_s3_endpoint_path_not_supported")
	}

	bucketLookup := minio.BucketLookupAuto
	if cfg.ForcePathStyle {
		bucketLookup = minio.BucketLookupPath
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, cfg.SessionToken),
		Secure:       secure,
		BucketLookup: bucketLookup,
		Region:       cfg.Region,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (b *BackupService) s3ObjectKey(prefix, filename string) string {
	prefix = strings.TrimSpace(prefix)
	filename = ensureDBFilename(filename)
	if prefix == "" {
		return filename
	}
	prefix = strings.TrimPrefix(prefix, "/")
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return filename
	}
	return path.Join(prefix, filename)
}

func (b *BackupService) getS3Config() (*config.S3BackupConfig, error) {
	backup := b.config.GetBackup()
	if backup == nil || backup.S3 == nil {
		return nil, fmt.Errorf("backup_s3_not_configured")
	}
	cfg := backup.S3
	if strings.TrimSpace(cfg.Endpoint) == "" || strings.TrimSpace(cfg.Bucket) == "" {
		return nil, fmt.Errorf("backup_s3_not_configured")
	}
	return cfg, nil
}

func (b *BackupService) listS3Backups() string {
	cfg, err := b.getS3Config()
	if err != nil {
		return marshalBackupListResult(false, "S3 未配置", nil)
	}

	client, err := b.newS3ClientFromConfig(cfg)
	if err != nil {
		return marshalBackupListResult(false, fmt.Sprintf("创建 S3 客户端失败: %v", err), nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix != "" {
		prefix = strings.TrimPrefix(prefix, "/")
		prefix = strings.TrimSuffix(prefix, "/") + "/"
	}

	var backups []BackupListItem
	for obj := range client.ListObjects(ctx, cfg.Bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			return marshalBackupListResult(false, fmt.Sprintf("获取备份列表失败: %v", obj.Err), nil)
		}
		name := obj.Key
		// Skip metadata files
		if strings.HasSuffix(name, ".meta.json") {
			continue
		}
		// Include .db files only
		if !strings.HasSuffix(name, ".db") {
			continue
		}
		backups = append(backups, BackupListItem{
			Filename: strings.TrimPrefix(name, prefix),
			Size:     obj.Size,
			ModTime:  obj.LastModified,
		})
	}

	sortBackupsByModTimeDesc(backups)
	return marshalBackupListResult(true, "获取备份列表成功", backups)
}

func (b *BackupService) backupToS3(filename string) error {
	if b.storage == nil {
		return fmt.Errorf("storage_not_initialized")
	}

	cfg, err := b.getS3Config()
	if err != nil {
		return err
	}
	client, err := b.newS3ClientFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("s3_client_failed")
	}

	filename = ensureDBFilename(filename)
	if filename == "" {
		return fmt.Errorf("filename_required")
	}

	// Create temp backup file in unique temp directory
	tmpDir, cleanup, err := tempDirUnique("s3_backup")
	if err != nil {
		return err
	}
	defer cleanup()

	tmpPath := filepath.Join(tmpDir, "backup.db")
	if err := b.storage.CreateBackupCopy(tmpPath); err != nil {
		logger.Error("Failed to create backup copy: %v", err)
		return fmt.Errorf("create_db_backup_failed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	objectKey := b.s3ObjectKey(cfg.Prefix, filename)
	_, err = client.FPutObject(ctx, cfg.Bucket, objectKey, tmpPath, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		logger.Error("Failed to upload backup to S3: %v", err)
		return fmt.Errorf("backup_upload_failed")
	}

	metaPath := filepath.Join(tmpDir, "backup.meta.json")
	_ = os.WriteFile(metaPath, nowMeta(b.version), 0644)
	if _, err := client.FPutObject(ctx, cfg.Bucket, objectKey+".meta.json", metaPath, minio.PutObjectOptions{ContentType: "application/json"}); err != nil {
		logger.Warn("Failed to upload S3 metadata: %v", err)
	}

	return nil
}

func (b *BackupService) downloadS3BackupToTemp(filename string) (string, func(), error) {
	cfg, err := b.getS3Config()
	if err != nil {
		return "", nil, err
	}
	client, err := b.newS3ClientFromConfig(cfg)
	if err != nil {
		return "", nil, fmt.Errorf("s3_client_failed")
	}

	filename = ensureDBFilename(filename)
	if filename == "" {
		return "", nil, fmt.Errorf("filename_required")
	}

	tmpDir, cleanup, err := tempDirUnique("s3_restore")
	if err != nil {
		return "", nil, err
	}
	tmpPath := filepath.Join(tmpDir, "restore.db")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	objectKey := b.s3ObjectKey(cfg.Prefix, filename)
	if err := client.FGetObject(ctx, cfg.Bucket, objectKey, tmpPath, minio.GetObjectOptions{}); err != nil {
		cleanup()
		logger.Error("Failed to download backup from S3: %v", err)
		return "", nil, fmt.Errorf("restore_download_failed")
	}

	return tmpPath, cleanup, nil
}

func (b *BackupService) detectS3Conflict(filename string) string {
	if b.storage == nil {
		return marshalConflictResult(false, "存储未初始化", nil)
	}

	tmpPath, cleanup, err := b.downloadS3BackupToTemp(filename)
	if err != nil {
		return marshalConflictResult(false, err.Error(), nil)
	}
	defer cleanup()

	conflicts, err := b.storage.DetectEndpointConflicts(tmpPath)
	if err != nil {
		return marshalConflictResult(false, fmt.Sprintf("检测冲突失败: %v", err), nil)
	}
	return marshalConflictResult(true, "", conflicts)
}

func (b *BackupService) restoreFromS3(filename, choice string, reloadConfig func(*config.Config) error) error {
	if b.storage == nil {
		return fmt.Errorf("storage_not_initialized")
	}

	tmpPath, cleanup, err := b.downloadS3BackupToTemp(filename)
	if err != nil {
		return err
	}
	defer cleanup()

	strategy := storage.MergeStrategyKeepLocal
	if choice == "remote" {
		strategy = storage.MergeStrategyOverwriteLocal
	}

	if err := b.storage.MergeFromBackup(tmpPath, strategy); err != nil {
		logger.Error("Failed to merge from backup: %v", err)
		return fmt.Errorf("merge_data_failed")
	}

	configAdapter := storage.NewConfigStorageAdapter(b.storage)
	newConfig, err := config.LoadFromStorage(configAdapter)
	if err != nil {
		logger.Error("Failed to load config from storage: %v", err)
		return fmt.Errorf("load_config_failed")
	}

	*b.config = *newConfig
	if err := reloadConfig(newConfig); err != nil {
		logger.Error("Failed to reload config: %v", err)
		return fmt.Errorf("update_proxy_config_failed")
	}
	return nil
}

func (b *BackupService) deleteS3Backups(filenames []string) error {
	cfg, err := b.getS3Config()
	if err != nil {
		return err
	}
	client, err := b.newS3ClientFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("s3_client_failed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		for _, name := range filenames {
			name = ensureDBFilename(name)
			if name == "" {
				continue
			}
			key := b.s3ObjectKey(cfg.Prefix, name)
			objectsCh <- minio.ObjectInfo{Key: key}
			objectsCh <- minio.ObjectInfo{Key: key + ".meta.json"}
		}
	}()

	for res := range client.RemoveObjects(ctx, cfg.Bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		if res.Err != nil {
			return fmt.Errorf("delete_backup_failed")
		}
	}
	return nil
}

func (b *BackupService) testS3Connection(endpoint, region, bucket, prefix, accessKey, secretKey, sessionToken string, useSSL, forcePathStyle bool) BackupTestResult {
	endpoint = strings.TrimSpace(endpoint)
	bucket = strings.TrimSpace(bucket)
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)
	prefix = strings.TrimSpace(prefix)

	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		return BackupTestResult{Success: false, Message: "参数不完整"}
	}

	tmpCfg := &config.S3BackupConfig{
		Endpoint:       endpoint,
		Region:         strings.TrimSpace(region),
		Bucket:         bucket,
		Prefix:         prefix,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		SessionToken:   strings.TrimSpace(sessionToken),
		UseSSL:         useSSL,
		ForcePathStyle: forcePathStyle,
	}
	client, err := b.newS3ClientFromConfig(tmpCfg)
	if err != nil {
		return BackupTestResult{Success: false, Message: fmt.Sprintf("创建客户端失败: %v", err)}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// HeadBucket is cheap and doesn't modify state.
	_, err = client.BucketExists(ctx, bucket)
	if err != nil {
		return BackupTestResult{Success: false, Message: fmt.Sprintf("连接失败: %v", err)}
	}
	return BackupTestResult{Success: true, Message: "连接成功"}
}
