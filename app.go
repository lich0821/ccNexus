package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/proxy"
	"github.com/lich0821/ccNexus/internal/storage"
	"github.com/lich0821/ccNexus/internal/tray"
	"github.com/lich0821/ccNexus/internal/webdav"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed wails.json
var wailsJSON []byte

// WailsInfo represents the info section from wails.json
type WailsInfo struct {
	Info struct {
		ProductVersion string `json:"productVersion"`
	} `json:"info"`
}

// Test endpoint constants
const (
	testMessage   = "你是什么模型?"
	testMaxTokens = 16
)

// normalizeAPIUrl ensures the API URL has the correct format
// Preserves http:// or https:// prefix if present, otherwise keeps as-is
func normalizeAPIUrl(apiUrl string) string {
	// Just remove trailing slash, keep protocol as-is
	apiUrl = strings.TrimSuffix(apiUrl, "/")
	return apiUrl
}

// App struct
type App struct {
	ctx        context.Context
	config     *config.Config
	proxy      *proxy.Proxy
	storage    *storage.SQLiteStorage
	configPath string
	ctxMutex   sync.RWMutex
	trayIcon   []byte
}

// NewApp creates a new App application struct
func NewApp(trayIcon []byte) *App {
	return &App{
		trayIcon: trayIcon,
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctxMutex.Lock()
	a.ctx = ctx
	a.ctxMutex.Unlock()

	logger.Info("Application starting...")

	// Enable debug file logging when DEBUG environment variable is set
	if os.Getenv("DEBUG") != "" {
		if err := logger.GetLogger().EnableDebugFile("debug.log"); err != nil {
			logger.Warn("Failed to enable debug file: %v", err)
		} else {
			logger.Info("Debug file logging enabled: debug.log")
		}
	}

	// Get config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Failed to get home directory: %v", err)
		homeDir = "."
	}
	configDir := filepath.Join(homeDir, ".ccNexus")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Error("Failed to create config directory: %v", err)
	}

	// Setup paths
	configPath := filepath.Join(configDir, "config.json")
	statsPath := filepath.Join(configDir, "stats.json")
	dbPath := filepath.Join(configDir, "ccnexus.db")

	a.configPath = configPath
	logger.Debug("Config path: %s", configPath)
	logger.Debug("Database path: %s", dbPath)

	// Run migration from JSON to SQLite if needed
	if err := storage.MigrateFromJSON(configPath, statsPath, dbPath); err != nil {
		logger.Error("Migration failed: %v", err)
	}

	// Initialize SQLite storage
	sqliteStorage, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		logger.Error("Failed to initialize storage: %v", err)
		// Fallback to JSON mode
		cfg, err := config.Load(configPath)
		if err != nil {
			logger.Warn("Failed to load config: %v, using default", err)
			cfg = config.DefaultConfig()
		}
		a.config = cfg
		// Create proxy without storage (will fail, but we log the error)
		logger.Error("Cannot start without storage")
		return
	}
	a.storage = sqliteStorage

	// Load configuration from SQLite
	configAdapter := storage.NewConfigStorageAdapter(sqliteStorage)
	cfg, err := config.LoadFromStorage(configAdapter)
	if err != nil {
		logger.Warn("Failed to load config from storage: %v, using default", err)
		cfg = config.DefaultConfig()
		// Save default config to storage
		if err := cfg.SaveToStorage(configAdapter); err != nil {
			logger.Warn("Failed to save default config: %v", err)
		}
	}
	a.config = cfg

	// Restore log level from config if it was previously set
	if cfg.GetLogLevel() >= 0 {
		logger.GetLogger().SetMinLevel(logger.LogLevel(cfg.GetLogLevel()))
		logger.Debug("Log level restored from config: %d", cfg.GetLogLevel())
	}

	// Get or create device ID
	deviceID, err := sqliteStorage.GetOrCreateDeviceID()
	if err != nil {
		logger.Warn("Failed to get device ID: %v, using default", err)
		deviceID = "default"
	} else {
		logger.Info("Device ID: %s", deviceID)
	}

	// Create proxy with storage
	statsAdapter := storage.NewStatsStorageAdapter(sqliteStorage)
	a.proxy = proxy.New(cfg, statsAdapter, deviceID)

	// Initialize system tray first
	a.initTray()

	// Start proxy in background
	go func() {
		if err := a.proxy.Start(); err != nil {
			logger.Error("Proxy server error: %v", err)
		}
	}()

	// Wait for tray to initialize, then show window
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
		lang = a.GetSystemLanguage()
	}
	tray.Setup(a.trayIcon, a.ShowWindow, a.HideWindow, a.Quit, lang)
}

// ShowWindow shows the application window
func (a *App) ShowWindow() {
	a.ctxMutex.RLock()
	ctx := a.ctx
	a.ctxMutex.RUnlock()

	if ctx != nil {
		runtime.WindowShow(ctx)
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
	// Save current window size before showing close dialog
	a.saveWindowSize(ctx)

	// Check if user has already set a preference
	behavior := a.config.GetCloseWindowBehavior()

	if behavior == "quit" {
		// User previously chose to quit, so quit directly
		return false // Allow window to close (will trigger shutdown)
	} else if behavior == "minimize" {
		// User previously chose to minimize, so hide window
		a.HideWindow()
		return true // Prevent window close
	}

	// No preference set, show dialog to ask user
	runtime.EventsEmit(ctx, "show-close-dialog")

	// Return true to prevent window close (dialog will handle the action)
	return true
}

// saveWindowSize saves the current window size to config
func (a *App) saveWindowSize(ctx context.Context) {
	// Get current window size
	width, height := runtime.WindowGetSize(ctx)

	// Only save if size is valid
	if width > 0 && height > 0 {
		a.config.UpdateWindowSize(width, height)
		// Save to SQLite storage
		if a.storage != nil {
			configAdapter := storage.NewConfigStorageAdapter(a.storage)
			if err := a.config.SaveToStorage(configAdapter); err != nil {
				logger.Warn("Failed to save window size: %v", err)
			} else {
				logger.Debug("Window size saved: %dx%d", width, height)
			}
		}
	}
}

// SetCloseWindowBehavior sets the user's preference for close window behavior
func (a *App) SetCloseWindowBehavior(behavior string) error {
	if behavior != "quit" && behavior != "minimize" {
		return fmt.Errorf("invalid behavior: %s (must be 'quit' or 'minimize')", behavior)
	}

	a.config.UpdateCloseWindowBehavior(behavior)

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := a.config.SaveToStorage(configAdapter); err != nil {
			logger.Warn("Failed to save close window behavior: %v", err)
			return err
		}
	}

	logger.Info("Close window behavior set to: %s", behavior)
	return nil
}

// Quit quits the application
func (a *App) Quit() {
	logger.Info("Quitting application...")

	// Save window size before quitting
	a.ctxMutex.RLock()
	ctx := a.ctx
	a.ctxMutex.RUnlock()

	if ctx != nil {
		a.saveWindowSize(ctx)
	}

	// Flush any pending saves and cleanup
	if a.proxy != nil {
		if err := a.proxy.GetStats().FlushSave(); err != nil {
			logger.Warn("Failed to save stats: %v", err)
		}
		a.proxy.Stop()
	}
	logger.GetLogger().Close()

	os.Exit(0)
}

// GetConfig returns the current configuration
func (a *App) GetConfig() string {
	data, _ := json.Marshal(a.config)
	return string(data)
}

// GetVersion returns the application version from wails.json
func (a *App) GetVersion() string {
	var info WailsInfo
	if err := json.Unmarshal(wailsJSON, &info); err != nil {
		logger.Warn("Failed to parse wails.json for version: %v", err)
		return "unknown"
	}
	return info.Info.ProductVersion
}

// UpdateConfig updates the configuration
func (a *App) UpdateConfig(configJSON string) error {
	var newConfig config.Config
	if err := json.Unmarshal([]byte(configJSON), &newConfig); err != nil {
		return fmt.Errorf("invalid config format: %w", err)
	}

	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Update proxy
	if err := a.proxy.UpdateConfig(&newConfig); err != nil {
		return err
	}

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := newConfig.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	a.config = &newConfig
	return nil
}

// GetStats returns current proxy statistics
func (a *App) GetStats() string {
	totalRequests, endpointStats := a.proxy.GetStats().GetStats()

	stats := map[string]interface{}{
		"totalRequests": totalRequests,
		"endpoints":     endpointStats,
	}

	data, _ := json.Marshal(stats)
	return string(data)
}

// GetStatsDaily returns statistics for today
func (a *App) GetStatsDaily() string {
	now := time.Now()
	today := now.Format("2006-01-02")

	dailyStats := a.proxy.GetStats().GetDailyStats(today)

	// Calculate totals
	var totalRequests, totalErrors, totalInputTokens, totalOutputTokens int
	for _, stats := range dailyStats {
		totalRequests += stats.Requests
		totalErrors += stats.Errors
		totalInputTokens += stats.InputTokens
		totalOutputTokens += stats.OutputTokens
	}

	// Count active and total endpoints
	endpoints := a.config.GetEndpoints()
	totalEndpoints := len(endpoints)
	activeEndpoints := 0
	for _, ep := range endpoints {
		if ep.Enabled {
			activeEndpoints++
		}
	}

	result := map[string]interface{}{
		"period":            "daily",
		"date":              today,
		"totalRequests":     totalRequests,
		"totalErrors":       totalErrors,
		"totalSuccess":      totalRequests - totalErrors,
		"totalInputTokens":  totalInputTokens,
		"totalOutputTokens": totalOutputTokens,
		"activeEndpoints":   activeEndpoints,
		"totalEndpoints":    totalEndpoints,
		"endpoints":         dailyStats,
	}

	data, _ := json.Marshal(result)
	return string(data)
}

// GetStatsYesterday returns statistics for yesterday
func (a *App) GetStatsYesterday() string {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	yesterdayStats := a.proxy.GetStats().GetDailyStats(yesterday)

	// Calculate totals
	var totalRequests, totalErrors, totalInputTokens, totalOutputTokens int
	for _, stats := range yesterdayStats {
		totalRequests += stats.Requests
		totalErrors += stats.Errors
		totalInputTokens += stats.InputTokens
		totalOutputTokens += stats.OutputTokens
	}

	// Count active and total endpoints
	endpoints := a.config.GetEndpoints()
	totalEndpoints := len(endpoints)
	activeEndpoints := 0
	for _, ep := range endpoints {
		if ep.Enabled {
			activeEndpoints++
		}
	}

	result := map[string]interface{}{
		"period":            "yesterday",
		"date":              yesterday,
		"totalRequests":     totalRequests,
		"totalErrors":       totalErrors,
		"totalSuccess":      totalRequests - totalErrors,
		"totalInputTokens":  totalInputTokens,
		"totalOutputTokens": totalOutputTokens,
		"activeEndpoints":   activeEndpoints,
		"totalEndpoints":    totalEndpoints,
		"endpoints":         yesterdayStats,
	}

	data, _ := json.Marshal(result)
	return string(data)
}

// GetStatsWeekly returns statistics for this week (Monday to now)
func (a *App) GetStatsWeekly() string {
	now := time.Now()
	// Calculate the start of this week (Monday)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	daysFromMonday := weekday - 1
	startOfWeek := now.AddDate(0, 0, -daysFromMonday)
	startDate := startOfWeek.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	weeklyStats := a.proxy.GetStats().GetPeriodStats(startDate, endDate)

	// Calculate totals
	var totalRequests, totalErrors, totalInputTokens, totalOutputTokens int
	for _, stats := range weeklyStats {
		totalRequests += stats.Requests
		totalErrors += stats.Errors
		totalInputTokens += stats.InputTokens
		totalOutputTokens += stats.OutputTokens
	}

	// Count active and total endpoints
	endpoints := a.config.GetEndpoints()
	totalEndpoints := len(endpoints)
	activeEndpoints := 0
	for _, ep := range endpoints {
		if ep.Enabled {
			activeEndpoints++
		}
	}

	result := map[string]interface{}{
		"period":            "weekly",
		"startDate":         startDate,
		"endDate":           endDate,
		"totalRequests":     totalRequests,
		"totalErrors":       totalErrors,
		"totalSuccess":      totalRequests - totalErrors,
		"totalInputTokens":  totalInputTokens,
		"totalOutputTokens": totalOutputTokens,
		"activeEndpoints":   activeEndpoints,
		"totalEndpoints":    totalEndpoints,
		"endpoints":         weeklyStats,
	}

	data, _ := json.Marshal(result)
	return string(data)
}

// GetStatsMonthly returns statistics for this month (1st to now)
func (a *App) GetStatsMonthly() string {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startDate := startOfMonth.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	monthlyStats := a.proxy.GetStats().GetPeriodStats(startDate, endDate)

	// Calculate totals
	var totalRequests, totalErrors, totalInputTokens, totalOutputTokens int
	for _, stats := range monthlyStats {
		totalRequests += stats.Requests
		totalErrors += stats.Errors
		totalInputTokens += stats.InputTokens
		totalOutputTokens += stats.OutputTokens
	}

	// Count active and total endpoints
	endpoints := a.config.GetEndpoints()
	totalEndpoints := len(endpoints)
	activeEndpoints := 0
	for _, ep := range endpoints {
		if ep.Enabled {
			activeEndpoints++
		}
	}

	result := map[string]interface{}{
		"period":            "monthly",
		"startDate":         startDate,
		"endDate":           endDate,
		"totalRequests":     totalRequests,
		"totalErrors":       totalErrors,
		"totalSuccess":      totalRequests - totalErrors,
		"totalInputTokens":  totalInputTokens,
		"totalOutputTokens": totalOutputTokens,
		"activeEndpoints":   activeEndpoints,
		"totalEndpoints":    totalEndpoints,
		"endpoints":         monthlyStats,
	}

	data, _ := json.Marshal(result)
	return string(data)
}

// GetStatsTrend returns trend comparison data
func (a *App) GetStatsTrend() string {
	now := time.Now()

	// Today vs Yesterday
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	todayStats := a.proxy.GetStats().GetDailyStats(today)
	yesterdayStats := a.proxy.GetStats().GetDailyStats(yesterday)

	// Calculate totals for today
	var todayRequests, todayErrors, todayInputTokens, todayOutputTokens int
	for _, stats := range todayStats {
		todayRequests += stats.Requests
		todayErrors += stats.Errors
		todayInputTokens += stats.InputTokens
		todayOutputTokens += stats.OutputTokens
	}

	// Calculate totals for yesterday
	var yesterdayRequests, yesterdayErrors, yesterdayInputTokens, yesterdayOutputTokens int
	for _, stats := range yesterdayStats {
		yesterdayRequests += stats.Requests
		yesterdayErrors += stats.Errors
		yesterdayInputTokens += stats.InputTokens
		yesterdayOutputTokens += stats.OutputTokens
	}

	// Calculate percentage changes
	requestsTrend := calculateTrend(todayRequests, yesterdayRequests)
	errorsTrend := calculateTrend(todayErrors, yesterdayErrors)
	tokensTrend := calculateTrend(todayInputTokens+todayOutputTokens, yesterdayInputTokens+yesterdayOutputTokens)

	result := map[string]interface{}{
		"daily": map[string]interface{}{
			"current":       todayRequests,
			"previous":      yesterdayRequests,
			"trend":         requestsTrend,
			"currentErrors": todayErrors,
			"previousErrors": yesterdayErrors,
			"errorsTrend":   errorsTrend,
			"currentTokens": todayInputTokens + todayOutputTokens,
			"previousTokens": yesterdayInputTokens + yesterdayOutputTokens,
			"tokensTrend":   tokensTrend,
		},
	}

	data, _ := json.Marshal(result)
	return string(data)
}

// calculateTrend calculates percentage change between current and previous values
func calculateTrend(current, previous int) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		// When previous is 0 but current > 0, return a large positive number
		// to indicate significant increase (capped at 999.9% for display purposes)
		return 999.9
	}
	return ((float64(current) - float64(previous)) / float64(previous)) * 100.0
}

// AddEndpoint adds a new endpoint
func (a *App) AddEndpoint(name, apiUrl, apiKey, transformer, model, remark string) error {
	// Check for duplicate endpoint name
	endpoints := a.config.GetEndpoints()
	for _, ep := range endpoints {
		if ep.Name == name {
			return fmt.Errorf("endpoint name '%s' already exists", name)
		}
	}

	// Default to claude if transformer not specified
	if transformer == "" {
		transformer = "claude"
	}

	// Normalize API URL (remove trailing slash only)
	apiUrl = normalizeAPIUrl(apiUrl)

	endpoints = append(endpoints, config.Endpoint{
		Name:        name,
		APIUrl:      apiUrl,
		APIKey:      apiKey,
		Enabled:     true,
		Transformer: transformer,
		Model:       model,
		Remark:      remark,
	})

	a.config.UpdateEndpoints(endpoints)

	if err := a.config.Validate(); err != nil {
		return err
	}

	if err := a.proxy.UpdateConfig(a.config); err != nil {
		return err
	}

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := a.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	if model != "" {
		logger.Info("Endpoint added: %s (%s) [%s/%s]", name, apiUrl, transformer, model)
	} else {
		logger.Info("Endpoint added: %s (%s) [%s]", name, apiUrl, transformer)
	}

	return nil
}

// RemoveEndpoint removes an endpoint by index
func (a *App) RemoveEndpoint(index int) error {
	endpoints := a.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		return fmt.Errorf("invalid endpoint index: %d", index)
	}

	// Save endpoint name before removal for logging
	removedName := endpoints[index].Name

	// Remove the endpoint
	endpoints = append(endpoints[:index], endpoints[index+1:]...)
	a.config.UpdateEndpoints(endpoints)

	// Skip validation if no endpoints left (allow empty state)
	if len(endpoints) > 0 {
		if err := a.config.Validate(); err != nil {
			return err
		}
	}

	if err := a.proxy.UpdateConfig(a.config); err != nil {
		return err
	}

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := a.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	logger.Info("Endpoint removed: %s", removedName)

	return nil
}

// UpdateEndpoint updates an endpoint by index
func (a *App) UpdateEndpoint(index int, name, apiUrl, apiKey, transformer, model, remark string) error {
	endpoints := a.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		return fmt.Errorf("invalid endpoint index: %d", index)
	}

	// Save old name for logging
	oldName := endpoints[index].Name

	// Check for duplicate endpoint name (only if name is changing)
	if oldName != name {
		for i, ep := range endpoints {
			if i != index && ep.Name == name {
				return fmt.Errorf("endpoint name '%s' already exists", name)
			}
		}
	}

	// Preserve the Enabled status
	enabled := endpoints[index].Enabled

	// Default to claude if transformer not specified
	if transformer == "" {
		transformer = "claude"
	}

	// Normalize API URL (remove trailing slash only)
	apiUrl = normalizeAPIUrl(apiUrl)

	endpoints[index] = config.Endpoint{
		Name:        name,
		APIUrl:      apiUrl,
		APIKey:      apiKey,
		Enabled:     enabled,
		Transformer: transformer,
		Model:       model,
		Remark:      remark,
	}

	a.config.UpdateEndpoints(endpoints)

	if err := a.config.Validate(); err != nil {
		return err
	}

	if err := a.proxy.UpdateConfig(a.config); err != nil {
		return err
	}

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := a.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	if oldName != name {
		if model != "" {
			logger.Info("Endpoint updated: %s → %s (%s) [%s/%s]", oldName, name, apiUrl, transformer, model)
		} else {
			logger.Info("Endpoint updated: %s → %s (%s) [%s]", oldName, name, apiUrl, transformer)
		}
	} else {
		if model != "" {
			logger.Info("Endpoint updated: %s (%s) [%s/%s]", name, apiUrl, transformer, model)
		} else {
			logger.Info("Endpoint updated: %s (%s) [%s]", name, apiUrl, transformer)
		}
	}

	return nil
}

// UpdatePort updates the proxy port
func (a *App) UpdatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}

	a.config.UpdatePort(port)

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := a.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	// Note: Changing port requires restart
	return nil
}

// ToggleEndpoint toggles the enabled state of an endpoint
func (a *App) ToggleEndpoint(index int, enabled bool) error {
	endpoints := a.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		return fmt.Errorf("invalid endpoint index: %d", index)
	}

	endpointName := endpoints[index].Name
	endpoints[index].Enabled = enabled
	a.config.UpdateEndpoints(endpoints)

	if err := a.proxy.UpdateConfig(a.config); err != nil {
		return err
	}

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := a.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	if enabled {
		logger.Info("Endpoint enabled: %s", endpointName)
	} else {
		logger.Info("Endpoint disabled: %s", endpointName)
	}

	return nil
}

// OpenURL opens a URL in the default browser
func (a *App) OpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

// GetLogs returns all log entries
func (a *App) GetLogs() string {
	logs := logger.GetLogger().GetLogs()
	data, _ := json.Marshal(logs)
	return string(data)
}

// GetLogsByLevel returns logs filtered by level
func (a *App) GetLogsByLevel(level int) string {
	logs := logger.GetLogger().GetLogsByLevel(logger.LogLevel(level))
	data, _ := json.Marshal(logs)
	return string(data)
}

// ClearLogs clears all log entries
func (a *App) ClearLogs() {
	logger.GetLogger().Clear()
}

// SetLogLevel sets the minimum log level to record
func (a *App) SetLogLevel(level int) {
	logger.GetLogger().SetMinLevel(logger.LogLevel(level))

	// Save to config
	a.config.UpdateLogLevel(level)

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := a.config.SaveToStorage(configAdapter); err != nil {
			logger.Warn("Failed to save log level: %v", err)
		} else {
			logger.Debug("Log level saved: %d", level)
		}
	}
}

// GetLogLevel returns the current minimum log level
func (a *App) GetLogLevel() int {
	return a.config.GetLogLevel()
}

// GetSystemLanguage detects the system language
func (a *App) GetSystemLanguage() string {
	// Try to get system language from environment variables
	locale := os.Getenv("LANG")
	if locale == "" {
		locale = os.Getenv("LC_ALL")
	}
	if locale == "" {
		locale = os.Getenv("LANGUAGE")
	}
	if locale == "" {
		return "en"
	}

	// Parse locale (e.g., "zh_CN.UTF-8" -> "zh-CN")
	// Simple check for Chinese
	if strings.Contains(strings.ToLower(locale), "zh") {
		return "zh-CN"
	}
	return "en"
}

// GetLanguage returns the current language setting
func (a *App) GetLanguage() string {
	lang := a.config.GetLanguage()
	if lang == "" {
		// Auto-detect if not set
		return a.GetSystemLanguage()
	}
	return lang
}

// SetLanguage sets the UI language
func (a *App) SetLanguage(language string) error {
	a.config.UpdateLanguage(language)

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := a.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save language: %w", err)
		}
	}

	// Update tray menu language
	tray.UpdateLanguage(language)

	logger.Info("Language changed to: %s", language)
	return nil
}

// TestEndpoint tests an endpoint by sending a simple request
func (a *App) TestEndpoint(index int) string {
	endpoints := a.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Invalid endpoint index: %d", index),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	endpoint := endpoints[index]
	logger.Info("Testing endpoint: %s (%s)", endpoint.Name, endpoint.APIUrl)

	// Build test request based on transformer type
	var requestBody []byte
	var err error
	var apiPath string

	transformer := endpoint.Transformer
	if transformer == "" {
		transformer = "claude"
	}

	switch transformer {
	case "claude":
		// Claude API format
		apiPath = "/v1/messages"
		model := endpoint.Model
		if model == "" {
			model = "claude-sonnet-4-5-20250929"
		}
		requestBody, err = json.Marshal(map[string]interface{}{
			"model":      model,
			"max_tokens": testMaxTokens,
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": testMessage,
				},
			},
		})

	case "openai":
		// OpenAI API format
		apiPath = "/v1/chat/completions"
		model := endpoint.Model
		if model == "" {
			model = "gpt-4-turbo"
		}
		requestBody, err = json.Marshal(map[string]interface{}{
			"model":      model,
			"max_tokens": testMaxTokens,
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": testMessage,
				},
			},
		})

	case "gemini":
		// Gemini API format
		model := endpoint.Model
		if model == "" {
			model = "gemini-pro"
		}
		apiPath = "/v1beta/models/" + model + ":generateContent"
		requestBody, err = json.Marshal(map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]string{
						{"text": testMessage},
					},
				},
			},
			"generationConfig": map[string]int{
				"maxOutputTokens": testMaxTokens,
			},
		})

	default:
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Unsupported transformer: %s", transformer),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to build request: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	// Build full URL
	normalizedAPIUrl := normalizeAPIUrl(endpoint.APIUrl)
	// Add https:// if no protocol specified
	if !strings.HasPrefix(normalizedAPIUrl, "http://") && !strings.HasPrefix(normalizedAPIUrl, "https://") {
		normalizedAPIUrl = "https://" + normalizedAPIUrl
	}
	url := fmt.Sprintf("%s%s", normalizedAPIUrl, apiPath)

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewReader(requestBody))
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to create request: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	// Set headers based on transformer
	req.Header.Set("Content-Type", "application/json")
	switch transformer {
	case "claude":
		req.Header.Set("x-api-key", endpoint.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case "openai":
		req.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
	case "gemini":
		// Gemini uses API key in query parameter
		q := req.URL.Query()
		q.Add("key", endpoint.APIKey)
		req.URL.RawQuery = q.Encode()
	}

	// Send request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Request failed: %v", err),
		}
		data, _ := json.Marshal(result)
		logger.Error("Test failed for %s: %v", endpoint.Name, err)
		return string(data)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to read response: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}
		data, _ := json.Marshal(result)
		logger.Error("Test failed for %s: HTTP %d", endpoint.Name, resp.StatusCode)
		return string(data)
	}

	// Parse response to extract content
	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		// If we can't parse JSON, just return the raw response
		result := map[string]interface{}{
			"success": true,
			"message": string(respBody),
		}
		data, _ := json.Marshal(result)
		logger.Info("Test successful for %s", endpoint.Name)
		return string(data)
	}

	// Extract message based on transformer type
	var message string
	switch transformer {
	case "claude":
		if content, ok := responseData["content"].([]interface{}); ok && len(content) > 0 {
			if textBlock, ok := content[0].(map[string]interface{}); ok {
				if text, ok := textBlock["text"].(string); ok {
					message = text
				}
			}
		}
	case "openai":
		if choices, ok := responseData["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if msg, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := msg["content"].(string); ok {
						message = content
					}
				}
			}
		}
	case "gemini":
		if candidates, ok := responseData["candidates"].([]interface{}); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]interface{}); ok {
				if content, ok := candidate["content"].(map[string]interface{}); ok {
					if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
						if part, ok := parts[0].(map[string]interface{}); ok {
							if text, ok := part["text"].(string); ok {
								message = text
							}
						}
					}
				}
			}
		}
	}

	// If we couldn't extract a message, return the full response
	if message == "" {
		message = string(respBody)
	}

	result := map[string]interface{}{
		"success": true,
		"message": message,
	}
	data, _ := json.Marshal(result)
	logger.Info("Test successful for %s", endpoint.Name)
	return string(data)
}

// GetCurrentEndpoint returns the current active endpoint name
func (a *App) GetCurrentEndpoint() string {
	if a.proxy == nil {
		return ""
	}
	return a.proxy.GetCurrentEndpointName()
}

// SwitchToEndpoint manually switches to a specific endpoint by name
func (a *App) SwitchToEndpoint(endpointName string) error {
	if a.proxy == nil {
		return fmt.Errorf("proxy not initialized")
	}

	return a.proxy.SetCurrentEndpoint(endpointName)
}

// ReorderEndpoints reorders endpoints based on the provided name array
func (a *App) ReorderEndpoints(names []string) error {
	endpoints := a.config.GetEndpoints()

	// Verify length matches
	if len(names) != len(endpoints) {
		return fmt.Errorf("names array length (%d) doesn't match endpoints count (%d)", len(names), len(endpoints))
	}

	// Check for duplicates in names array
	seen := make(map[string]bool)
	for _, name := range names {
		if seen[name] {
			return fmt.Errorf("duplicate endpoint name in reorder request: %s", name)
		}
		seen[name] = true
	}

	// Create a map for quick lookup of endpoints by name
	endpointMap := make(map[string]config.Endpoint)
	for _, ep := range endpoints {
		endpointMap[ep.Name] = ep
	}

	// Build new order and verify all names exist
	newEndpoints := make([]config.Endpoint, 0, len(names))
	for _, name := range names {
		ep, exists := endpointMap[name]
		if !exists {
			return fmt.Errorf("endpoint not found: %s", name)
		}
		newEndpoints = append(newEndpoints, ep)
	}

	// Update config
	a.config.UpdateEndpoints(newEndpoints)

	if err := a.config.Validate(); err != nil {
		return err
	}

	if err := a.proxy.UpdateConfig(a.config); err != nil {
		return err
	}

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := a.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	logger.Info("Endpoints reordered: %v", names)

	return nil
}

// UpdateWebDAVConfig updates the WebDAV configuration
func (a *App) UpdateWebDAVConfig(url, username, password string) error {
	webdavConfig := &config.WebDAVConfig{
		URL:        url,
		Username:   username,
		Password:   password,
		ConfigPath: "/ccNexus/config",
		StatsPath:  "/ccNexus/stats",
	}

	a.config.UpdateWebDAV(webdavConfig)

	// Save to SQLite storage
	if a.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(a.storage)
		if err := a.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save WebDAV config: %w", err)
		}
	}

	logger.Info("WebDAV configuration updated: %s", url)
	return nil
}

// TestWebDAVConnection tests the WebDAV connection with provided credentials
func (a *App) TestWebDAVConnection(url, username, password string) string {
	webdavCfg := &config.WebDAVConfig{
		URL:      url,
		Username: username,
		Password: password,
	}

	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("创建WebDAV客户端失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	testResult := client.TestConnection()
	data, _ := json.Marshal(testResult)
	return string(data)
}

// BackupToWebDAV backs up configuration and stats to WebDAV
func (a *App) BackupToWebDAV(filename string) error {
	logger.Info("Starting backup process for file: %s", filename)

	webdavCfg := a.config.GetWebDAV()
	if webdavCfg == nil {
		logger.Error("WebDAV configuration is not set")
		return fmt.Errorf("WebDAV未配置")
	}
	logger.Debug("WebDAV config loaded: URL=%s, Username=%s", webdavCfg.URL, webdavCfg.Username)

	if a.storage == nil {
		logger.Error("Storage is not initialized")
		return fmt.Errorf("存储未初始化")
	}

	// Create WebDAV client
	logger.Debug("Creating WebDAV client...")
	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		logger.Error("Failed to create WebDAV client: %v", err)
		return fmt.Errorf("创建WebDAV客户端失败: %w", err)
	}
	logger.Debug("WebDAV client created successfully")

	// Create sync manager
	manager := webdav.NewManager(client)

	// Create temporary backup of database (without app_config)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Failed to get home directory: %v", err)
		return fmt.Errorf("获取用户目录失败: %w", err)
	}
	tempDir := filepath.Join(homeDir, ".ccNexus", "temp")
	logger.Debug("Creating temp directory: %s", tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		logger.Error("Failed to create temp directory: %v", err)
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	tempBackupPath := filepath.Join(tempDir, "backup_temp.db")

	// Remove existing temp file if it exists
	if _, err := os.Stat(tempBackupPath); err == nil {
		logger.Debug("Removing existing temp file: %s", tempBackupPath)
		os.Remove(tempBackupPath)
	}

	defer func() {
		logger.Debug("Cleaning up temp file: %s", tempBackupPath)
		os.Remove(tempBackupPath)
		logger.Debug("Cleaning up temp directory: %s", tempDir)
		os.RemoveAll(tempDir)
	}()

	// Create backup copy without app_config
	logger.Info("Creating database backup copy (excluding app_config)...")
	if err := a.storage.CreateBackupCopy(tempBackupPath); err != nil {
		logger.Error("Failed to create database backup: %v", err)
		return fmt.Errorf("创建数据库备份失败: %w", err)
	}

	// Check file size
	if fileInfo, err := os.Stat(tempBackupPath); err == nil {
		logger.Debug("Backup file created: %s (size: %d bytes)", tempBackupPath, fileInfo.Size())
	}

	// Backup to WebDAV
	version := a.GetVersion()
	logger.Info("Uploading backup to WebDAV (version: %s)...", version)
	if err := manager.BackupDatabase(tempBackupPath, version, filename); err != nil {
		logger.Error("Failed to upload backup to WebDAV: %v", err)
		return fmt.Errorf("备份失败: %w", err)
	}

	logger.Info("Backup created successfully: %s", filename)
	return nil
}

// RestoreFromWebDAV restores configuration and stats from WebDAV
func (a *App) RestoreFromWebDAV(filename, choice string) error {
	webdavCfg := a.config.GetWebDAV()
	if webdavCfg == nil {
		return fmt.Errorf("WebDAV未配置")
	}

	if a.storage == nil {
		return fmt.Errorf("存储未初始化")
	}

	// If user chose to keep local config, do nothing
	if choice == "local" {
		logger.Info("User chose to keep local configuration")
		return nil
	}

	// Create WebDAV client
	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		return fmt.Errorf("创建WebDAV客户端失败: %w", err)
	}

	// Create sync manager
	manager := webdav.NewManager(client)

	// Create temporary directory for downloaded backup
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户目录失败: %w", err)
	}
	tempDir := filepath.Join(homeDir, ".ccNexus", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	tempRestorePath := filepath.Join(tempDir, "restore_temp.db")
	defer os.Remove(tempRestorePath)  // Clean up temp file
	defer os.RemoveAll(tempDir)       // Clean up temp directory

	// Download and restore database from WebDAV
	if err := manager.RestoreDatabase(filename, tempRestorePath); err != nil {
		return fmt.Errorf("恢复失败: %w", err)
	}

	// Determine merge strategy based on user choice
	var strategy storage.MergeStrategy
	if choice == "remote" {
		strategy = storage.MergeStrategyOverwriteLocal
	} else {
		strategy = storage.MergeStrategyKeepLocal
	}

	// Merge the restored database into current database
	if err := a.storage.MergeFromBackup(tempRestorePath, strategy); err != nil {
		return fmt.Errorf("合并数据失败: %w", err)
	}

	// Reload configuration from storage
	configAdapter := storage.NewConfigStorageAdapter(a.storage)
	newConfig, err := config.LoadFromStorage(configAdapter)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// Update in-memory config
	a.config = newConfig

	// Update proxy config
	if err := a.proxy.UpdateConfig(newConfig); err != nil {
		return fmt.Errorf("更新代理配置失败: %w", err)
	}

	logger.Info("Configuration and statistics restored from: %s", filename)
	return nil
}

// ListWebDAVBackups lists all backups on WebDAV server
func (a *App) ListWebDAVBackups() string {
	logger.Info("Listing WebDAV backups...")

	webdavCfg := a.config.GetWebDAV()
	if webdavCfg == nil {
		logger.Error("WebDAV configuration is not set")
		result := map[string]interface{}{
			"success": false,
			"message": "WebDAV未配置",
			"backups": []interface{}{},
		}
		data, _ := json.Marshal(result)
		return string(data)
	}
	logger.Debug("WebDAV config: URL=%s, Username=%s", webdavCfg.URL, webdavCfg.Username)

	// Create WebDAV client
	logger.Debug("Creating WebDAV client...")
	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		logger.Error("Failed to create WebDAV client: %v", err)
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("创建WebDAV客户端失败: %v", err),
			"backups": []interface{}{},
		}
		data, _ := json.Marshal(result)
		return string(data)
	}
	logger.Debug("WebDAV client created successfully")

	// Create sync manager
	manager := webdav.NewManager(client)

	// List backups
	logger.Info("Fetching backup list from WebDAV...")
	backups, err := manager.ListConfigBackups()
	if err != nil {
		logger.Error("Failed to list backups: %v", err)
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("获取备份列表失败: %v", err),
			"backups": []interface{}{},
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	logger.Info("Found %d backup(s)", len(backups))
	for i, backup := range backups {
		logger.Debug("Backup %d: %s (size: %d bytes, modified: %s)", i+1, backup.Filename, backup.Size, backup.ModTime)
	}

	result := map[string]interface{}{
		"success": true,
		"message": "获取备份列表成功",
		"backups": backups,
	}
	data, _ := json.Marshal(result)
	return string(data)
}

// DeleteWebDAVBackups deletes backups from WebDAV server
func (a *App) DeleteWebDAVBackups(filenames []string) error {
	webdavCfg := a.config.GetWebDAV()
	if webdavCfg == nil {
		return fmt.Errorf("WebDAV未配置")
	}

	// Create WebDAV client
	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		return fmt.Errorf("创建WebDAV客户端失败: %w", err)
	}

	// Create sync manager
	manager := webdav.NewManager(client)

	// Delete backups
	if err := manager.DeleteConfigBackups(filenames); err != nil {
		return fmt.Errorf("删除备份失败: %w", err)
	}

	logger.Info("Backups deleted: %v", filenames)
	return nil
}

// DetectWebDAVConflict detects conflicts between local and remote config
func (a *App) DetectWebDAVConflict(filename string) string {
	webdavCfg := a.config.GetWebDAV()
	if webdavCfg == nil {
		result := map[string]interface{}{
			"success": false,
			"message": "WebDAV未配置",
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	if a.storage == nil {
		result := map[string]interface{}{
			"success": false,
			"message": "存储未初始化",
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	// Create WebDAV client
	client, err := webdav.NewClient(webdavCfg)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("创建WebDAV客户端失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	// Create sync manager
	manager := webdav.NewManager(client)

	// Create temporary directory for downloaded backup
	homeDir, err := os.UserHomeDir()
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("获取用户目录失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}
	tempDir := filepath.Join(homeDir, ".ccNexus", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("创建临时目录失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}
	tempRestorePath := filepath.Join(tempDir, "conflict_check_temp.db")
	defer os.Remove(tempRestorePath)  // Clean up temp file
	defer os.RemoveAll(tempDir)       // Clean up temp directory

	// Download database from WebDAV
	if err := manager.RestoreDatabase(filename, tempRestorePath); err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("下载备份失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	// Detect endpoint conflicts
	conflicts, err := a.storage.DetectEndpointConflicts(tempRestorePath)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("检测冲突失败: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	result := map[string]interface{}{
		"success":   true,
		"conflicts": conflicts,
	}
	data, _ := json.Marshal(result)
	return string(data)
}

// ListArchives returns a list of all available archive months
func (a *App) ListArchives() string {
	if a.storage == nil {
		result := map[string]interface{}{
			"success":  false,
			"message":  "Storage not initialized",
			"archives": []string{},
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	months, err := a.storage.GetArchiveMonths()
	if err != nil {
		logger.Error("Failed to get archive months: %v", err)
		result := map[string]interface{}{
			"success":  false,
			"message":  fmt.Sprintf("Failed to load archives: %v", err),
			"archives": []string{},
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	result := map[string]interface{}{
		"success":  true,
		"archives": months,
	}
	data, _ := json.Marshal(result)
	return string(data)
}

// GetArchiveData returns archived data for a specific month
func (a *App) GetArchiveData(month string) string {
	if a.storage == nil {
		result := map[string]interface{}{
			"success": false,
			"message": "Storage not initialized",
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	// Get monthly archive data from storage
	archiveData, err := a.storage.GetMonthlyArchiveData(month)
	if err != nil {
		logger.Error("Failed to get archive data for %s: %v", month, err)
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to load archive: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	// Build archive structure compatible with frontend
	// Format: { endpoints: { endpointName: { dailyHistory: { date: stats } } }, summary: {...} }
	endpoints := make(map[string]map[string]interface{})
	var totalRequests, totalErrors, totalInputTokens, totalOutputTokens int

	for _, record := range archiveData {
		// Initialize endpoint if not exists
		if endpoints[record.EndpointName] == nil {
			endpoints[record.EndpointName] = map[string]interface{}{
				"dailyHistory": make(map[string]interface{}),
			}
		}

		// Add daily record
		dailyHistory := endpoints[record.EndpointName]["dailyHistory"].(map[string]interface{})
		dailyHistory[record.Date] = map[string]interface{}{
			"date":         record.Date,
			"requests":     record.Requests,
			"errors":       record.Errors,
			"inputTokens":  record.InputTokens,
			"outputTokens": record.OutputTokens,
		}

		// Accumulate totals
		totalRequests += record.Requests
		totalErrors += record.Errors
		totalInputTokens += record.InputTokens
		totalOutputTokens += record.OutputTokens
	}

	// Build summary
	summary := map[string]interface{}{
		"totalRequests":     totalRequests,
		"totalErrors":       totalErrors,
		"totalInputTokens":  totalInputTokens,
		"totalOutputTokens": totalOutputTokens,
	}

	// Build final result
	archive := map[string]interface{}{
		"endpoints": endpoints,
		"summary":   summary,
	}

	result := map[string]interface{}{
		"success": true,
		"archive": archive,
	}

	data, _ := json.Marshal(result)
	return string(data)
}

// GenerateMockArchives generates mock archive data for testing
// Note: This function is kept for backward compatibility but is no longer needed
// Archive data is now stored in SQLite and can be queried via GetArchiveData()
func (a *App) GenerateMockArchives(monthsCount int) string {
	result := map[string]interface{}{
		"success": false,
		"message": "Mock archives are no longer supported. Use real data from SQLite.",
	}
	data, _ := json.Marshal(result)
	return string(data)
}

