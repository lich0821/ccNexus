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
	"strings"
	"sync"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/proxy"
	"github.com/lich0821/ccNexus/internal/tray"
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
// Removes http:// or https:// prefix if present
func normalizeAPIUrl(apiUrl string) string {
	// Remove http:// or https:// prefix
	apiUrl = strings.TrimPrefix(apiUrl, "https://")
	apiUrl = strings.TrimPrefix(apiUrl, "http://")
	// Remove trailing slash
	apiUrl = strings.TrimSuffix(apiUrl, "/")
	return apiUrl
}

// App struct
type App struct {
	ctx        context.Context
	config     *config.Config
	proxy      *proxy.Proxy
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
		// Save default config only if it doesn't exist
		if err := cfg.Save(configPath); err != nil {
			logger.Warn("Failed to save config: %v", err)
		}
	}
	a.config = cfg

	// Restore log level from config if it was previously set
	if cfg.GetLogLevel() >= 0 {
		logger.GetLogger().SetMinLevel(logger.LogLevel(cfg.GetLogLevel()))
		logger.Debug("Log level restored from config: %d", cfg.GetLogLevel())
	}

	// Create proxy
	a.proxy = proxy.New(cfg)

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
		// Save stats before stopping
		if err := a.proxy.GetStats().Save(); err != nil {
			logger.Warn("Failed to save stats on shutdown: %v", err)
		}
		a.proxy.Stop()
	}
	logger.Info("Application stopped")
	logger.GetLogger().Close()
}

// initTray initializes the system tray
func (a *App) initTray() {
	tray.Setup(a.trayIcon, a.ShowWindow, a.HideWindow, a.Quit)
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
	runtime.WindowHide(ctx)
	return true // Return true to prevent window close (just hide it)
}

// Quit quits the application
func (a *App) Quit() {
	logger.Info("Quitting application...")

	// Save stats and cleanup
	if a.proxy != nil {
		if err := a.proxy.GetStats().Save(); err != nil {
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
func (a *App) AddEndpoint(name, apiUrl, apiKey, transformer, model, remark string) error {
	// Default to claude if transformer not specified
	if transformer == "" {
		transformer = "claude"
	}

	// Normalize API URL (remove http/https prefix if present)
	apiUrl = normalizeAPIUrl(apiUrl)

	endpoints := a.config.GetEndpoints()
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
func (a *App) UpdateEndpoint(index int, name, apiUrl, apiKey, transformer, model, remark string) error {
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

	// Normalize API URL (remove http/https prefix if present)
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
	if err := a.config.Save(a.configPath); err != nil {
		return fmt.Errorf("failed to save language: %w", err)
	}
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
	url := fmt.Sprintf("https://%s%s", endpoint.APIUrl, apiPath)

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
