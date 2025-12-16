package main

import (
    "context"
    _ "embed"
    "encoding/json"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/lich0821/ccNexus/internal/config"
    "github.com/lich0821/ccNexus/internal/logger"
    "github.com/lich0821/ccNexus/internal/proxy"
    "github.com/lich0821/ccNexus/internal/storage"
    "github.com/lich0821/ccNexus/internal/tray"
    "github.com/lich0821/ccNexus/service"
    "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed wails.json
var wailsJSON []byte

//go:embed CHANGELOG_CN.json
var changelogZH []byte

//go:embed CHANGELOG.json
var changelogEN []byte

// WailsInfo represents the info section from wails.json
type WailsInfo struct {
    Info struct {
        ProductVersion string `json:"productVersion"`
    } `json:"info"`
}

// App struct
type App struct {
    ctx      context.Context
    config   *config.Config
    proxy    *proxy.Proxy
    storage  *storage.SQLiteStorage
    ctxMutex sync.RWMutex
    trayIcon []byte

    // Services
    stats    *service.StatsService
    endpoint *service.EndpointService
    settings *service.SettingsService
    webdav   *service.WebDAVService
    archive  *service.ArchiveService
    update   *service.UpdateService
    terminal *service.TerminalService
}

// NewApp creates a new App application struct
func NewApp(trayIcon []byte) *App {
    return &App{trayIcon: trayIcon}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
    a.ctxMutex.Lock()
    a.ctx = ctx
    a.ctxMutex.Unlock()

    logger.Info("Application starting...")

    if os.Getenv("DEBUG") != "" {
        if err := logger.GetLogger().EnableDebugFile("debug.log"); err != nil {
            logger.Warn("Failed to enable debug file: %v", err)
        }
    }

    homeDir, err := os.UserHomeDir()
    if err != nil {
        logger.Error("Failed to get home directory: %v", err)
        homeDir = "."
    }
    configDir := filepath.Join(homeDir, ".ccNexus")
    if err := os.MkdirAll(configDir, 0755); err != nil {
        logger.Error("Failed to create config directory: %v", err)
    }

    configPath := filepath.Join(configDir, "config.json")
    statsPath := filepath.Join(configDir, "stats.json")
    dbPath := filepath.Join(configDir, "ccnexus.db")

    if err := storage.MigrateFromJSON(configPath, statsPath, dbPath); err != nil {
        logger.Error("Migration failed: %v", err)
    }

    sqliteStorage, err := storage.NewSQLiteStorage(dbPath)
    if err != nil {
        logger.Error("Failed to initialize storage: %v", err)
        cfg, err := config.Load(configPath)
        if err != nil {
            cfg = config.DefaultConfig()
        }
        a.config = cfg
        logger.Error("Cannot start without storage")
        return
    }
    a.storage = sqliteStorage

    configAdapter := storage.NewConfigStorageAdapter(sqliteStorage)
    cfg, err := config.LoadFromStorage(configAdapter)
    if err != nil {
        logger.Warn("Failed to load config from storage: %v, using default", err)
        cfg = config.DefaultConfig()
        if err := cfg.SaveToStorage(configAdapter); err != nil {
            logger.Warn("Failed to save default config: %v", err)
        }
    }
    a.config = cfg

    if cfg.GetLogLevel() >= 0 {
        logger.GetLogger().SetMinLevel(logger.LogLevel(cfg.GetLogLevel()))
    }

    deviceID, err := sqliteStorage.GetOrCreateDeviceID()
    if err != nil {
        logger.Warn("Failed to get device ID: %v, using default", err)
        deviceID = "default"
    } else {
        logger.Info("Device ID: %s", deviceID)
    }

    statsAdapter := storage.NewStatsStorageAdapter(sqliteStorage)
    a.proxy = proxy.New(cfg, statsAdapter, deviceID)

    a.proxy.SetOnEndpointSuccess(func(endpointName string) {
        runtime.EventsEmit(ctx, "endpoint:success", endpointName)
    })

    // Initialize services
    version := a.GetVersion()
    a.stats = service.NewStatsService(a.proxy, a.config)
    a.endpoint = service.NewEndpointService(a.config, a.proxy, a.storage)
    a.settings = service.NewSettingsService(a.config, a.storage)
    a.webdav = service.NewWebDAVService(a.config, a.storage, version)
    a.archive = service.NewArchiveService(a.storage)
    a.update = service.NewUpdateService(a.config, a.storage, version)
    a.terminal = service.NewTerminalService(a.config, a.storage)

    a.initTray()

    go func() {
        if err := a.proxy.Start(); err != nil {
            logger.Error("Proxy server error: %v", err)
        }
    }()

    time.Sleep(300 * time.Millisecond)
    runtime.WindowShow(ctx)

    logger.Info("Application started successfully")
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
    if a.proxy != nil {
        a.proxy.Stop()
    }
    if a.storage != nil {
        if err := a.storage.Close(); err != nil {
            logger.Warn("Failed to close storage: %v", err)
        }
    }
    logger.Info("Application stopped")
    logger.GetLogger().Close()
}

// initTray initializes the system tray
func (a *App) initTray() {
    lang := a.config.GetLanguage()
    if lang == "" {
        lang = a.settings.GetSystemLanguage()
    }
    tray.Setup(a.trayIcon, a.ShowWindow, a.HideWindow, a.Quit, lang)
}

// ShowWindow shows the application window
func (a *App) ShowWindow() {
    a.ctxMutex.RLock()
    ctx := a.ctx
    a.ctxMutex.RUnlock()

    if ctx != nil {
        for i := 0; i < 3; i++ {
            runtime.WindowShow(ctx)
            time.Sleep(50 * time.Millisecond)
            runtime.WindowSetAlwaysOnTop(ctx, true)
            runtime.WindowSetAlwaysOnTop(ctx, false)
            break
        }
    }
}

// HideWindow hides the application window
func (a *App) HideWindow() {
    a.ctxMutex.RLock()
    ctx := a.ctx
    a.ctxMutex.RUnlock()

    if ctx != nil {
        runtime.WindowHide(ctx)
    }
}

// beforeClose is called when the window is about to close
func (a *App) beforeClose(ctx context.Context) bool {
    width, height := runtime.WindowGetSize(ctx)
    a.settings.SaveWindowSize(width, height)

    behavior := a.config.GetCloseWindowBehavior()

    if behavior == "quit" {
        return false
    } else if behavior == "minimize" {
        a.HideWindow()
        return true
    }

    runtime.EventsEmit(ctx, "show-close-dialog")
    return true
}

// Quit quits the application
func (a *App) Quit() {
    logger.Info("Quitting application...")

    a.ctxMutex.RLock()
    ctx := a.ctx
    a.ctxMutex.RUnlock()

    if ctx != nil {
        width, height := runtime.WindowGetSize(ctx)
        a.settings.SaveWindowSize(width, height)
    }

    if a.proxy != nil {
        if err := a.proxy.GetStats().FlushSave(); err != nil {
            logger.Warn("Failed to save stats: %v", err)
        }
        a.proxy.Stop()
    }
    logger.GetLogger().Close()

    os.Exit(0)
}

// GetVersion returns the application version from wails.json
func (a *App) GetVersion() string {
    var info WailsInfo
    if err := json.Unmarshal(wailsJSON, &info); err != nil {
        return "unknown"
    }
    return info.Info.ProductVersion
}

// GetChangelog returns the changelog content based on language
func (a *App) GetChangelog(lang string) string {
    if lang == "zh-CN" {
        return string(changelogZH)
    }
    return string(changelogEN)
}

// OpenURL opens a URL in the default browser
func (a *App) OpenURL(url string) {
    runtime.BrowserOpenURL(a.ctx, url)
}

// SelectDirectory opens a directory selection dialog
func (a *App) SelectDirectory() string {
    dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
        Title: "Select Project Directory",
    })
    if err != nil {
        logger.Error("Failed to open directory dialog: %v", err)
        return ""
    }
    return dir
}

// ========== Stats Bindings ==========

func (a *App) GetStats() string                          { return a.stats.GetStats() }
func (a *App) GetStatsDaily() string                     { return a.stats.GetStatsDaily() }
func (a *App) GetStatsYesterday() string                 { return a.stats.GetStatsYesterday() }
func (a *App) GetStatsWeekly() string                    { return a.stats.GetStatsWeekly() }
func (a *App) GetStatsMonthly() string                   { return a.stats.GetStatsMonthly() }
func (a *App) GetStatsTrend() string                     { return a.stats.GetStatsTrend() }
func (a *App) GetStatsTrendByPeriod(period string) string { return a.stats.GetStatsTrendByPeriod(period) }

// ========== Endpoint Bindings ==========

func (a *App) AddEndpoint(name, apiUrl, apiKey, transformer, model, remark string) error {
    return a.endpoint.AddEndpoint(name, apiUrl, apiKey, transformer, model, remark)
}
func (a *App) RemoveEndpoint(index int) error { return a.endpoint.RemoveEndpoint(index) }
func (a *App) UpdateEndpoint(index int, name, apiUrl, apiKey, transformer, model, remark string) error {
    return a.endpoint.UpdateEndpoint(index, name, apiUrl, apiKey, transformer, model, remark)
}
func (a *App) ToggleEndpoint(index int, enabled bool) error { return a.endpoint.ToggleEndpoint(index, enabled) }
func (a *App) ReorderEndpoints(names []string) error        { return a.endpoint.ReorderEndpoints(names) }
func (a *App) GetCurrentEndpoint() string                   { return a.endpoint.GetCurrentEndpoint() }
func (a *App) SwitchToEndpoint(endpointName string) error   { return a.endpoint.SwitchToEndpoint(endpointName) }
func (a *App) TestEndpoint(index int) string                { return a.endpoint.TestEndpoint(index) }
func (a *App) TestEndpointLight(index int) string           { return a.endpoint.TestEndpointLight(index) }
func (a *App) TestAllEndpointsZeroCost() string             { return a.endpoint.TestAllEndpointsZeroCost() }
func (a *App) FetchModels(apiUrl, apiKey, transformer string) string {
    return a.endpoint.FetchModels(apiUrl, apiKey, transformer)
}

// ========== Settings Bindings ==========

func (a *App) GetConfig() string { return a.settings.GetConfig() }
func (a *App) UpdateConfig(configJSON string) error {
    return a.settings.UpdateConfig(configJSON, a.proxy)
}
func (a *App) UpdatePort(port int) error                       { return a.settings.UpdatePort(port) }
func (a *App) GetSystemLanguage() string                       { return a.settings.GetSystemLanguage() }
func (a *App) GetLanguage() string                             { return a.settings.GetLanguage() }
func (a *App) SetLanguage(language string) error               { return a.settings.SetLanguage(language) }
func (a *App) GetTheme() string                                { return a.settings.GetTheme() }
func (a *App) SetTheme(theme string) error                     { return a.settings.SetTheme(theme) }
func (a *App) GetThemeAuto() bool                              { return a.settings.GetThemeAuto() }
func (a *App) SetThemeAuto(auto bool) error                    { return a.settings.SetThemeAuto(auto) }
func (a *App) GetAutoLightTheme() string                       { return a.settings.GetAutoLightTheme() }
func (a *App) SetAutoLightTheme(theme string) error            { return a.settings.SetAutoLightTheme(theme) }
func (a *App) GetAutoDarkTheme() string                        { return a.settings.GetAutoDarkTheme() }
func (a *App) SetAutoDarkTheme(theme string) error             { return a.settings.SetAutoDarkTheme(theme) }
func (a *App) GetLogs() string                                 { return a.settings.GetLogs() }
func (a *App) GetLogsByLevel(level int) string                 { return a.settings.GetLogsByLevel(level) }
func (a *App) ClearLogs()                                      { a.settings.ClearLogs() }
func (a *App) SetLogLevel(level int)                           { a.settings.SetLogLevel(level) }
func (a *App) GetLogLevel() int                                { return a.settings.GetLogLevel() }
func (a *App) SetCloseWindowBehavior(behavior string) error    { return a.settings.SetCloseWindowBehavior(behavior) }

// ========== WebDAV Bindings ==========

func (a *App) UpdateWebDAVConfig(url, username, password string) error {
    return a.webdav.UpdateWebDAVConfig(url, username, password)
}
func (a *App) TestWebDAVConnection(url, username, password string) string {
    return a.webdav.TestWebDAVConnection(url, username, password)
}
func (a *App) BackupToWebDAV(filename string) error { return a.webdav.BackupToWebDAV(filename) }
func (a *App) RestoreFromWebDAV(filename, choice string) error {
    return a.webdav.RestoreFromWebDAV(filename, choice, func(cfg *config.Config) error {
        return a.proxy.UpdateConfig(cfg)
    })
}
func (a *App) ListWebDAVBackups() string                       { return a.webdav.ListWebDAVBackups() }
func (a *App) DeleteWebDAVBackups(filenames []string) error    { return a.webdav.DeleteWebDAVBackups(filenames) }
func (a *App) DetectWebDAVConflict(filename string) string     { return a.webdav.DetectWebDAVConflict(filename) }

// ========== Archive Bindings ==========

func (a *App) ListArchives() string                            { return a.archive.ListArchives() }
func (a *App) GetArchiveData(month string) string              { return a.archive.GetArchiveData(month) }
func (a *App) GetArchiveTrend(month string) string             { return a.archive.GetArchiveTrend(month) }
func (a *App) GenerateMockArchives(monthsCount int) string     { return a.archive.GenerateMockArchives(monthsCount) }

// ========== Update Bindings ==========

func (a *App) CheckForUpdates() string                                { return a.update.CheckForUpdates() }
func (a *App) GetUpdateSettings() string                              { return a.update.GetUpdateSettings() }
func (a *App) SetUpdateSettings(autoCheck bool, checkInterval int) error {
    return a.update.SetUpdateSettings(autoCheck, checkInterval)
}
func (a *App) SkipVersion(version string) error                       { return a.update.SkipVersion(version) }
func (a *App) DownloadUpdate(url, filename string) error              { return a.update.DownloadUpdate(url, filename) }
func (a *App) GetDownloadProgress() string                            { return a.update.GetDownloadProgress() }
func (a *App) CancelDownload()                                        { a.update.CancelDownload() }
func (a *App) InstallUpdate(filePath string) string                   { return a.update.InstallUpdate(filePath) }
func (a *App) ApplyUpdate(newExePath string) string                   { return a.update.ApplyUpdate(newExePath) }
func (a *App) SendUpdateNotification(title, message string) error     { return a.update.SendUpdateNotification(title, message) }

// ========== Terminal Bindings ==========

func (a *App) DetectTerminals() string                                      { return a.terminal.DetectTerminals() }
func (a *App) GetTerminalConfig() string                                    { return a.terminal.GetTerminalConfig() }
func (a *App) SaveTerminalConfig(selectedTerminal string, projectDirs []string) error {
    return a.terminal.SaveTerminalConfig(selectedTerminal, projectDirs)
}
func (a *App) AddProjectDir(dir string) error                               { return a.terminal.AddProjectDir(dir) }
func (a *App) RemoveProjectDir(dir string) error                            { return a.terminal.RemoveProjectDir(dir) }
func (a *App) LaunchTerminal(dir string) error                              { return a.terminal.LaunchTerminal(dir) }
func (a *App) GetSessions(projectDir string) string                         { return a.terminal.GetSessions(projectDir) }
func (a *App) DeleteSession(projectDir, sessionID string) error             { return a.terminal.DeleteSession(projectDir, sessionID) }
func (a *App) RenameSession(projectDir, sessionID, alias string) error      { return a.terminal.RenameSession(projectDir, sessionID, alias) }
func (a *App) GetSessionData(projectDir, sessionID string) string           { return a.terminal.GetSessionData(projectDir, sessionID) }
func (a *App) LaunchSessionTerminal(dir, sessionID string) error            { return a.terminal.LaunchSessionTerminal(dir, sessionID) }
