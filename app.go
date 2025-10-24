package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yourusername/claude-proxy/internal/config"
	"github.com/yourusername/claude-proxy/internal/proxy"
)

// App struct
type App struct {
	ctx    context.Context
	config *config.Config
	proxy  *proxy.Proxy
	configPath string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Get config path
	configPath, err := config.GetConfigPath()
	if err != nil {
		log.Printf("⚠️  Failed to get config path: %v", err)
		configPath = "config.json"
	}
	a.configPath = configPath

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("⚠️  Failed to load config: %v, using default", err)
		cfg = config.DefaultConfig()
	}
	a.config = cfg

	// Save default config if it doesn't exist
	if err := cfg.Save(configPath); err != nil {
		log.Printf("⚠️  Failed to save config: %v", err)
	}

	// Create proxy
	a.proxy = proxy.New(cfg)

	// Start proxy in background
	go func() {
		if err := a.proxy.Start(); err != nil {
			log.Printf("❌ Proxy server error: %v", err)
		}
	}()

	log.Println("✅ Application started")
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.proxy != nil {
		a.proxy.Stop()
	}
	log.Println("👋 Application stopped")
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
func (a *App) AddEndpoint(name, apiUrl, apiKey string) error {
	endpoints := a.config.GetEndpoints()
	endpoints = append(endpoints, config.Endpoint{
		Name:   name,
		APIUrl: apiUrl,
		APIKey: apiKey,
	})

	a.config.UpdateEndpoints(endpoints)

	if err := a.config.Validate(); err != nil {
		return err
	}

	if err := a.proxy.UpdateConfig(a.config); err != nil {
		return err
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

	return a.config.Save(a.configPath)
}

// UpdateEndpoint updates an endpoint by index
func (a *App) UpdateEndpoint(index int, name, apiUrl, apiKey string) error {
	endpoints := a.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		return fmt.Errorf("invalid endpoint index: %d", index)
	}

	endpoints[index] = config.Endpoint{
		Name:   name,
		APIUrl: apiUrl,
		APIKey: apiKey,
	}

	a.config.UpdateEndpoints(endpoints)

	if err := a.config.Validate(); err != nil {
		return err
	}

	if err := a.proxy.UpdateConfig(a.config); err != nil {
		return err
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
