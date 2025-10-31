package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Endpoint represents a single API endpoint configuration
type Endpoint struct {
	Name        string `json:"name"`
	APIUrl      string `json:"apiUrl"`
	APIKey      string `json:"apiKey"`
	Enabled     bool   `json:"enabled"`
	Transformer string `json:"transformer,omitempty"` // Transformer type: claude, openai, gemini, deepseek
	Model       string `json:"model,omitempty"`       // Target model name for non-Claude APIs
}

// Config represents the application configuration
type Config struct {
	Port      int        `json:"port"`
	Endpoints []Endpoint `json:"endpoints"`
	LogLevel  int        `json:"logLevel"` // 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR
	Language  string     `json:"language"`  // UI language: en, zh-CN
	mu        sync.RWMutex
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:     3000,
		LogLevel: 1,  // Default to INFO level
		Language: "", // Empty means auto-detect
		Endpoints: []Endpoint{
			{
				Name:        "Claude Official",
				APIUrl:      "api.anthropic.com",
				APIKey:      "your-api-key-here",
				Enabled:     true,
				Transformer: "claude",
			},
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if len(c.Endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}

	for i, ep := range c.Endpoints {
		if ep.APIUrl == "" {
			return fmt.Errorf("endpoint %d: apiUrl is required", i+1)
		}
		if ep.APIKey == "" {
			return fmt.Errorf("endpoint %d: apiKey is required", i+1)
		}

		// Default to claude transformer if not specified
		if ep.Transformer == "" {
			c.Endpoints[i].Transformer = "claude"
		}

		// Non-Claude transformers require model field
		if ep.Transformer != "claude" && ep.Model == "" {
			return fmt.Errorf("endpoint %d (%s): model is required for transformer '%s'", i+1, ep.Name, ep.Transformer)
		}
	}

	return nil
}

// GetEndpoints returns a copy of endpoints (thread-safe)
func (c *Config) GetEndpoints() []Endpoint {
	c.mu.RLock()
	defer c.mu.RUnlock()

	endpoints := make([]Endpoint, len(c.Endpoints))
	copy(endpoints, c.Endpoints)
	return endpoints
}

// GetPort returns the configured port (thread-safe)
func (c *Config) GetPort() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Port
}

// GetLogLevel returns the configured log level (thread-safe)
func (c *Config) GetLogLevel() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LogLevel
}

// UpdateEndpoints updates the endpoints (thread-safe)
func (c *Config) UpdateEndpoints(endpoints []Endpoint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Endpoints = endpoints
}

// UpdatePort updates the port (thread-safe)
func (c *Config) UpdatePort(port int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Port = port
}

// UpdateLogLevel updates the log level (thread-safe)
func (c *Config) UpdateLogLevel(level int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LogLevel = level
}

// GetLanguage returns the configured language (thread-safe)
func (c *Config) GetLanguage() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Language
}

// UpdateLanguage updates the language (thread-safe)
func (c *Config) UpdateLanguage(language string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Language = language
}

// GetConfigPath returns the default config file path
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".ccNexus")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// Save saves configuration to file
func (c *Config) Save(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	return nil
}
