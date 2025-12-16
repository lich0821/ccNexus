package service

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/lich0821/ccNexus/internal/config"
    "github.com/lich0821/ccNexus/internal/logger"
    "github.com/lich0821/ccNexus/internal/storage"
    "github.com/lich0821/ccNexus/internal/updater"
)

// UpdateService handles auto-update operations
type UpdateService struct {
    config  *config.Config
    storage *storage.SQLiteStorage
    updater *updater.Updater
    version string
}

// NewUpdateService creates a new UpdateService
func NewUpdateService(cfg *config.Config, s *storage.SQLiteStorage, version string) *UpdateService {
    return &UpdateService{
        config:  cfg,
        storage: s,
        version: version,
        updater: updater.New(version),
    }
}

// CheckForUpdates checks if a new version is available
func (u *UpdateService) CheckForUpdates() string {
    logger.Info("CheckForUpdates called, current version: %s", u.version)

    info, err := u.updater.CheckForUpdates()
    if err != nil {
        logger.Error("CheckForUpdates failed: %v", err)
        result := map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    logger.Info("Update check result: hasUpdate=%v, latest=%s", info.HasUpdate, info.LatestVersion)

    updateCfg := u.config.GetUpdate()
    updateCfg.LastCheckTime = time.Now().Format(time.RFC3339)
    u.config.UpdateUpdate(updateCfg)

    if u.storage != nil {
        configAdapter := storage.NewConfigStorageAdapter(u.storage)
        u.config.SaveToStorage(configAdapter)
    }

    result := map[string]interface{}{
        "success": true,
        "info":    info,
    }
    data, _ := json.Marshal(result)
    return string(data)
}

// GetUpdateSettings returns update settings
func (u *UpdateService) GetUpdateSettings() string {
    updateCfg := u.config.GetUpdate()
    data, _ := json.Marshal(updateCfg)
    return string(data)
}

// SetUpdateSettings updates update settings
func (u *UpdateService) SetUpdateSettings(autoCheck bool, checkInterval int) error {
    updateCfg := u.config.GetUpdate()
    updateCfg.AutoCheck = autoCheck
    updateCfg.CheckInterval = checkInterval

    u.config.UpdateUpdate(updateCfg)

    if u.storage != nil {
        configAdapter := storage.NewConfigStorageAdapter(u.storage)
        if err := u.config.SaveToStorage(configAdapter); err != nil {
            return fmt.Errorf("failed to save update settings: %w", err)
        }
    }

    logger.Info("Update settings changed: autoCheck=%v, interval=%d hours", autoCheck, checkInterval)
    return nil
}

// SkipVersion skips a specific version
func (u *UpdateService) SkipVersion(version string) error {
    updateCfg := u.config.GetUpdate()
    updateCfg.SkippedVersion = version

    u.config.UpdateUpdate(updateCfg)

    if u.storage != nil {
        configAdapter := storage.NewConfigStorageAdapter(u.storage)
        if err := u.config.SaveToStorage(configAdapter); err != nil {
            return fmt.Errorf("failed to save skipped version: %w", err)
        }
    }

    logger.Info("Version skipped: %s", version)
    return nil
}

// DownloadUpdate downloads the update file
func (u *UpdateService) DownloadUpdate(url, filename string) error {
    return u.updater.DownloadUpdate(url, filename)
}

// GetDownloadProgress returns download progress
func (u *UpdateService) GetDownloadProgress() string {
    progress := u.updater.GetDownloadProgress()
    data, _ := json.Marshal(progress)
    return string(data)
}

// CancelDownload cancels the current download
func (u *UpdateService) CancelDownload() {
    u.updater.CancelDownload()
}

// InstallUpdate installs the downloaded update
func (u *UpdateService) InstallUpdate(filePath string) string {
    result, err := u.updater.InstallUpdate(filePath)
    if err != nil {
        errorResult := map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        }
        data, _ := json.Marshal(errorResult)
        return string(data)
    }
    data, _ := json.Marshal(result)
    return string(data)
}

// ApplyUpdate applies the update and exits the program
func (u *UpdateService) ApplyUpdate(newExePath string) string {
    err := updater.ApplyUpdate(newExePath)
    if err != nil {
        return fmt.Sprintf(`{"success":false,"error":"%s"}`, err.Error())
    }
    return `{"success":true,"message":"update_applying"}`
}

// SendUpdateNotification sends a system notification for updates
func (u *UpdateService) SendUpdateNotification(title, message string) error {
    err := updater.SendNotification(title, message)
    if err != nil {
        logger.Error("Failed to send notification: %v", err)
    }
    return err
}
