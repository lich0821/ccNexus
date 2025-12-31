package service

import "time"

type BackupProvider string

const (
	BackupProviderWebDAV BackupProvider = "webdav"
	BackupProviderLocal  BackupProvider = "local"
	BackupProviderS3     BackupProvider = "s3"
)

type BackupListItem struct {
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"modTime"`
}

type BackupListResult struct {
	Success bool             `json:"success"`
	Message string           `json:"message"`
	Backups []BackupListItem `json:"backups"`
}

type BackupTestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
