package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/proxy"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	config     *config.Config
	proxy      *proxy.Proxy
	configPath string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	logger.Debug("Application starting...")

	// Get config path
	configPath, err := config.GetConfigPath()
	if err != nil {
		logger.Warn("Failed to get config path: %v, using default", err)
		configPath = "config.json"
	}
	a.configPath = configPath
	logger.Debug("Config path: %s", configPath)

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Warn("Failed to load config: %v, using default", err)
		cfg = config.DefaultConfig()
	}
	a.config = cfg

	// Restore log level from config
	logger.GetLogger().SetMinLevel(logger.LogLevel(cfg.GetLogLevel()))
	logger.Debug("Log level restored: %d", cfg.GetLogLevel())

	// Save default config if it doesn't exist
	if err := cfg.Save(configPath); err != nil {
		logger.Warn("Failed to save config: %v", err)
	}

	// Create proxy
	a.proxy = proxy.New(cfg)

	// Start proxy in background
	go func() {
		if err := a.proxy.Start(); err != nil {
			logger.Error("Proxy server error: %v", err)
		}
	}()

	logger.Info("Application started successfully")
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.proxy != nil {
		a.proxy.Stop()
	}
	logger.Info("Application stopped")
}

// GetConfig returns the current configuration
func (a *App) GetConfig() string {
	data, _ := json.Marshal(a.config)
	return string(data)
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

	// Save to file
	if err := newConfig.Save(a.configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
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

// AddEndpoint adds a new endpoint
func (a *App) AddEndpoint(name, apiUrl, apiKey, transformer, model string) error {
	// Default to claude if transformer not specified
	if transformer == "" {
		transformer = "claude"
	}

	endpoints := a.config.GetEndpoints()
	endpoints = append(endpoints, config.Endpoint{
		Name:        name,
		APIUrl:      apiUrl,
		APIKey:      apiKey,
		Enabled:     true,
		Transformer: transformer,
		Model:       model,
	})

	a.config.UpdateEndpoints(endpoints)

	if err := a.config.Validate(); err != nil {
		return err
	}

	if err := a.proxy.UpdateConfig(a.config); err != nil {
		return err
	}

	if model != "" {
		logger.Info("Endpoint added: %s (%s) [%s/%s]", name, apiUrl, transformer, model)
	} else {
		logger.Info("Endpoint added: %s (%s) [%s]", name, apiUrl, transformer)
	}

	return a.config.Save(a.configPath)
}

// RemoveEndpoint removes an endpoint by index
func (a *App) RemoveEndpoint(index int) error {
	endpoints := a.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		return fmt.Errorf("invalid endpoint index: %d", index)
	}

	// Show confirmation dialog
	selection, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "Confirm Delete",
		Message:       fmt.Sprintf("Are you sure you want to delete endpoint '%s'?", endpoints[index].Name),
		Buttons:       []string{"Delete", "Cancel"},
		DefaultButton: "Cancel",
	})

	if err != nil {
		return fmt.Errorf("dialog error: %w", err)
	}

	// If user clicked Cancel, return without error
	if selection != "Delete" {
		return nil
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

	logger.Info("Endpoint removed: %s", removedName)

	return a.config.Save(a.configPath)
}

// UpdateEndpoint updates an endpoint by index
func (a *App) UpdateEndpoint(index int, name, apiUrl, apiKey, transformer, model string) error {
	endpoints := a.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		return fmt.Errorf("invalid endpoint index: %d", index)
	}

	// Save old name for logging
	oldName := endpoints[index].Name

	// Preserve the Enabled status
	enabled := endpoints[index].Enabled

	// Default to claude if transformer not specified
	if transformer == "" {
		transformer = "claude"
	}

	endpoints[index] = config.Endpoint{
		Name:        name,
		APIUrl:      apiUrl,
		APIKey:      apiKey,
		Enabled:     enabled,
		Transformer: transformer,
		Model:       model,
	}

	a.config.UpdateEndpoints(endpoints)

	if err := a.config.Validate(); err != nil {
		return err
	}

	if err := a.proxy.UpdateConfig(a.config); err != nil {
		return err
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

	return a.config.Save(a.configPath)
}

// UpdatePort updates the proxy port
func (a *App) UpdatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}

	a.config.UpdatePort(port)

	if err := a.config.Save(a.configPath); err != nil {
		return err
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

	if enabled {
		logger.Info("Endpoint enabled: %s", endpointName)
	} else {
		logger.Info("Endpoint disabled: %s", endpointName)
	}

	return a.config.Save(a.configPath)
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
	if err := a.config.Save(a.configPath); err != nil {
		logger.Warn("Failed to save log level to config: %v", err)
	} else {
		logger.Debug("Log level saved to config: %d", level)
	}
}

// GetLogLevel returns the current minimum log level
func (a *App) GetLogLevel() int {
	return int(logger.GetLogger().GetMinLevel())
}
