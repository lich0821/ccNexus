package config

import (
	"encoding/json"
	"fmt"
	"strconv"
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
	Remark      string `json:"remark,omitempty"`      // Optional remark for the endpoint
}

// WebDAVConfig represents WebDAV synchronization configuration
type WebDAVConfig struct {
	URL        string `json:"url"`        // WebDAV server URL
	Username   string `json:"username"`   // Username
	Password   string `json:"password"`   // Password
	ConfigPath string `json:"configPath"` // Config backup path (default /ccNexus/config)
	StatsPath  string `json:"statsPath"`  // Stats backup path (default /ccNexus/stats)
}

// LocalBackupConfig represents local backup configuration
type LocalBackupConfig struct {
	Dir string `json:"dir"` // Local directory to store backups
}

// S3BackupConfig represents S3-compatible backup configuration
type S3BackupConfig struct {
	Endpoint       string `json:"endpoint"`
	Region         string `json:"region,omitempty"`
	Bucket         string `json:"bucket"`
	Prefix         string `json:"prefix,omitempty"`
	AccessKey      string `json:"accessKey"`
	SecretKey      string `json:"secretKey"`
	SessionToken   string `json:"sessionToken,omitempty"`
	UseSSL         bool   `json:"useSSL"`
	ForcePathStyle bool   `json:"forcePathStyle"`
}

// BackupConfig represents backup/sync configuration across providers
type BackupConfig struct {
	Provider string             `json:"provider"` // webdav | local | s3
	Local    *LocalBackupConfig `json:"local,omitempty"`
	S3       *S3BackupConfig    `json:"s3,omitempty"`
}

// UpdateConfig represents update configuration
type UpdateConfig struct {
	AutoCheck      bool   `json:"autoCheck"`      // Auto check for updates
	CheckInterval  int    `json:"checkInterval"`  // Check interval in hours
	LastCheckTime  string `json:"lastCheckTime"`  // Last check time (RFC3339)
	SkippedVersion string `json:"skippedVersion"` // Skipped version
}

// TerminalConfig represents terminal launcher configuration
type TerminalConfig struct {
	SelectedTerminal string   `json:"selectedTerminal"` // Selected terminal ID
	ProjectDirs      []string `json:"projectDirs"`      // Project directories
}

// ProxyConfig represents HTTP proxy configuration
type ProxyConfig struct {
	URL string `json:"url"` // Proxy URL, e.g., http://127.0.0.1:7890 or socks5://127.0.0.1:1080
}

// Config represents the application configuration
type Config struct {
	Port                int             `json:"port"`
	Endpoints           []Endpoint      `json:"endpoints"`
	LogLevel            int             `json:"logLevel"`                      // 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR
	Language            string          `json:"language"`                      // UI language: en, zh-CN
	Theme               string          `json:"theme"`                         // UI theme: light, dark
	ThemeAuto           bool            `json:"themeAuto"`                     // Auto switch theme based on time
	AutoLightTheme      string          `json:"autoLightTheme,omitempty"`      // Theme to use in daytime when auto mode is on
	AutoDarkTheme       string          `json:"autoDarkTheme,omitempty"`       // Theme to use in nighttime when auto mode is on
	WindowWidth         int             `json:"windowWidth"`                   // Window width in pixels
	WindowHeight        int             `json:"windowHeight"`                  // Window height in pixels
	CloseWindowBehavior string          `json:"closeWindowBehavior,omitempty"` // "quit", "minimize", "ask"
	WebDAV              *WebDAVConfig   `json:"webdav,omitempty"`              // WebDAV synchronization config
	Backup              *BackupConfig   `json:"backup,omitempty"`              // Backup/sync configuration
	Update              *UpdateConfig   `json:"update,omitempty"`              // Update configuration
	Terminal            *TerminalConfig `json:"terminal,omitempty"`            // Terminal launcher config
	Proxy               *ProxyConfig    `json:"proxy,omitempty"`               // HTTP proxy config
	mu                  sync.RWMutex
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:         3000,
		LogLevel:     1,       // Default to INFO level
		Language:     "zh-CN", // Default to Chinese
		WindowWidth:  1024,    // Default window width
		WindowHeight: 768,     // Default window height
		Endpoints: []Endpoint{
			{
				Name:        "Claude Official",
				APIUrl:      "api.anthropic.com",
				APIKey:      "your-api-key-here",
				Enabled:     true,
				Transformer: "claude",
			},
		},
		Update: &UpdateConfig{
			AutoCheck:     true,
			CheckInterval: 24,
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

// GetWindowSize returns the configured window size (thread-safe)
func (c *Config) GetWindowSize() (width, height int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.WindowWidth, c.WindowHeight
}

// UpdateWindowSize updates the window size (thread-safe)
func (c *Config) UpdateWindowSize(width, height int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.WindowWidth = width
	c.WindowHeight = height
}

// GetCloseWindowBehavior returns the close window behavior (thread-safe)
// Returns: "quit", "minimize", "ask"
func (c *Config) GetCloseWindowBehavior() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CloseWindowBehavior
}

// UpdateCloseWindowBehavior updates the close window behavior (thread-safe)
// Accepts: "quit", "minimize", "ask"
func (c *Config) UpdateCloseWindowBehavior(behavior string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CloseWindowBehavior = behavior
}

// GetTheme returns the configured theme (thread-safe)
// Returns: "light", "dark"
func (c *Config) GetTheme() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Theme
}

// UpdateTheme updates the theme (thread-safe)
// Accepts: "light", "dark"
func (c *Config) UpdateTheme(theme string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Theme = theme
}

// GetThemeAuto returns whether auto theme switching is enabled (thread-safe)
func (c *Config) GetThemeAuto() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ThemeAuto
}

// UpdateThemeAuto updates the auto theme setting (thread-safe)
func (c *Config) UpdateThemeAuto(auto bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ThemeAuto = auto
}

// GetAutoLightTheme returns the theme to use in daytime when auto mode is on (thread-safe)
func (c *Config) GetAutoLightTheme() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AutoLightTheme
}

// UpdateAutoLightTheme updates the auto light theme (thread-safe)
func (c *Config) UpdateAutoLightTheme(theme string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AutoLightTheme = theme
}

// GetAutoDarkTheme returns the theme to use in nighttime when auto mode is on (thread-safe)
func (c *Config) GetAutoDarkTheme() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AutoDarkTheme
}

// UpdateAutoDarkTheme updates the auto dark theme (thread-safe)
func (c *Config) UpdateAutoDarkTheme(theme string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AutoDarkTheme = theme
}

// GetWebDAV returns the WebDAV configuration (thread-safe)
func (c *Config) GetWebDAV() *WebDAVConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.WebDAV
}

// UpdateWebDAV updates the WebDAV configuration (thread-safe)
func (c *Config) UpdateWebDAV(webdav *WebDAVConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.WebDAV = webdav
}

// GetBackup returns the backup configuration (thread-safe)
func (c *Config) GetBackup() *BackupConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Backup
}

// UpdateBackup updates the backup configuration (thread-safe)
func (c *Config) UpdateBackup(backup *BackupConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Backup = backup
}

// GetUpdate returns the Update configuration (thread-safe)
func (c *Config) GetUpdate() *UpdateConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Update == nil {
		return &UpdateConfig{
			AutoCheck:     true,
			CheckInterval: 24,
		}
	}
	return c.Update
}

// UpdateUpdate updates the Update configuration (thread-safe)
func (c *Config) UpdateUpdate(update *UpdateConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Update = update
}

// GetTerminal returns the Terminal configuration (thread-safe)
func (c *Config) GetTerminal() *TerminalConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Terminal == nil {
		return &TerminalConfig{
			SelectedTerminal: "cmd",
			ProjectDirs:      []string{},
		}
	}
	return c.Terminal
}

// UpdateTerminal updates the Terminal configuration (thread-safe)
func (c *Config) UpdateTerminal(terminal *TerminalConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Terminal = terminal
}

// GetProxy returns the Proxy configuration (thread-safe)
func (c *Config) GetProxy() *ProxyConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Proxy
}

// UpdateProxy updates the Proxy configuration (thread-safe)
func (c *Config) UpdateProxy(proxy *ProxyConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Proxy = proxy
}

// StorageAdapter defines the interface needed for loading/saving config
type StorageAdapter interface {
	GetEndpoints() ([]StorageEndpoint, error)
	SaveEndpoint(ep *StorageEndpoint) error
	UpdateEndpoint(ep *StorageEndpoint) error
	DeleteEndpoint(name string) error
	GetConfig(key string) (string, error)
	SetConfig(key, value string) error
}

// StorageEndpoint represents an endpoint in storage
type StorageEndpoint struct {
	Name        string
	APIUrl      string
	APIKey      string
	Enabled     bool
	Transformer string
	Model       string
	Remark      string
	SortOrder   int
}

// LoadFromStorage loads configuration from SQLite storage
func LoadFromStorage(storage StorageAdapter) (*Config, error) {
	config := &Config{
		Endpoints: []Endpoint{},
	}

	// Load endpoints
	endpoints, err := storage.GetEndpoints()
	if err != nil {
		return nil, fmt.Errorf("failed to load endpoints: %w", err)
	}

	for _, ep := range endpoints {
		endpoint := Endpoint{
			Name:        ep.Name,
			APIUrl:      ep.APIUrl,
			APIKey:      ep.APIKey,
			Enabled:     ep.Enabled,
			Transformer: ep.Transformer,
			Model:       ep.Model,
			Remark:      ep.Remark,
		}
		if endpoint.Transformer == "" {
			endpoint.Transformer = "claude"
		}
		config.Endpoints = append(config.Endpoints, endpoint)
	}

	// Load app config
	if portStr, err := storage.GetConfig("port"); err == nil && portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Port = port
		}
	}
	if config.Port == 0 {
		config.Port = 3000
	}

	if logLevelStr, err := storage.GetConfig("logLevel"); err == nil && logLevelStr != "" {
		if logLevel, err := strconv.Atoi(logLevelStr); err == nil {
			config.LogLevel = logLevel
		}
	}

	if lang, err := storage.GetConfig("language"); err == nil {
		config.Language = lang
	}

	if widthStr, err := storage.GetConfig("windowWidth"); err == nil && widthStr != "" {
		if width, err := strconv.Atoi(widthStr); err == nil {
			config.WindowWidth = width
		}
	}
	if config.WindowWidth == 0 {
		config.WindowWidth = 1024
	}

	if heightStr, err := storage.GetConfig("windowHeight"); err == nil && heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil {
			config.WindowHeight = height
		}
	}
	if config.WindowHeight == 0 {
		config.WindowHeight = 768
	}

	// Load close window behavior
	if behaviorStr, err := storage.GetConfig("closeWindowBehavior"); err == nil && behaviorStr != "" {
		config.CloseWindowBehavior = behaviorStr
	}
	// Default to "ask" if not set
	if config.CloseWindowBehavior == "" {
		config.CloseWindowBehavior = "ask"
	}

	// Load theme
	if theme, err := storage.GetConfig("theme"); err == nil && theme != "" {
		config.Theme = theme
	}
	// Default to "light" if not set
	if config.Theme == "" {
		config.Theme = "light"
	}

	// Load themeAuto
	if themeAuto, err := storage.GetConfig("themeAuto"); err == nil && themeAuto != "" {
		config.ThemeAuto = themeAuto == "true"
	}

	// Load autoLightTheme
	if autoLightTheme, err := storage.GetConfig("autoLightTheme"); err == nil && autoLightTheme != "" {
		config.AutoLightTheme = autoLightTheme
	}
	// Default to "light" if not set
	if config.AutoLightTheme == "" {
		config.AutoLightTheme = "light"
	}

	// Load autoDarkTheme
	if autoDarkTheme, err := storage.GetConfig("autoDarkTheme"); err == nil && autoDarkTheme != "" {
		config.AutoDarkTheme = autoDarkTheme
	}
	// Default to "dark" if not set
	if config.AutoDarkTheme == "" {
		config.AutoDarkTheme = "dark"
	}

	// Load WebDAV config if exists
	if url, err := storage.GetConfig("webdav_url"); err == nil && url != "" {
		username, _ := storage.GetConfig("webdav_username")
		password, _ := storage.GetConfig("webdav_password")
		configPath, _ := storage.GetConfig("webdav_configPath")
		statsPath, _ := storage.GetConfig("webdav_statsPath")

		config.WebDAV = &WebDAVConfig{
			URL:        url,
			Username:   username,
			Password:   password,
			ConfigPath: configPath,
			StatsPath:  statsPath,
		}
	}

	// Load Backup config
	provider, _ := storage.GetConfig("backup_provider")
	if provider != "" {
		config.Backup = &BackupConfig{Provider: provider}
	}
	if provider == "local" {
		backupDir, _ := storage.GetConfig("backup_local_dir")
		config.Backup.Local = &LocalBackupConfig{Dir: backupDir}
	}
	if provider == "s3" {
		s3Endpoint, _ := storage.GetConfig("backup_s3_endpoint")
		s3Region, _ := storage.GetConfig("backup_s3_region")
		s3Bucket, _ := storage.GetConfig("backup_s3_bucket")
		s3Prefix, _ := storage.GetConfig("backup_s3_prefix")
		s3AccessKey, _ := storage.GetConfig("backup_s3_accessKey")
		s3SecretKey, _ := storage.GetConfig("backup_s3_secretKey")
		s3SessionToken, _ := storage.GetConfig("backup_s3_sessionToken")
		s3UseSSLStr, _ := storage.GetConfig("backup_s3_useSSL")
		s3ForcePathStyleStr, _ := storage.GetConfig("backup_s3_forcePathStyle")

		config.Backup.S3 = &S3BackupConfig{
			Endpoint:       s3Endpoint,
			Region:         s3Region,
			Bucket:         s3Bucket,
			Prefix:         s3Prefix,
			AccessKey:      s3AccessKey,
			SecretKey:      s3SecretKey,
			SessionToken:   s3SessionToken,
			UseSSL:         s3UseSSLStr == "true",
			ForcePathStyle: s3ForcePathStyleStr == "true",
		}
	}

	// Load Update config
	config.Update = &UpdateConfig{
		AutoCheck:     true,
		CheckInterval: 24,
	}
	if autoCheckStr, err := storage.GetConfig("update_autoCheck"); err == nil && autoCheckStr != "" {
		config.Update.AutoCheck = autoCheckStr == "true"
	}
	if intervalStr, err := storage.GetConfig("update_checkInterval"); err == nil && intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil {
			config.Update.CheckInterval = interval
		}
	}
	if lastCheck, err := storage.GetConfig("update_lastCheckTime"); err == nil {
		config.Update.LastCheckTime = lastCheck
	}
	if skipped, err := storage.GetConfig("update_skippedVersion"); err == nil {
		config.Update.SkippedVersion = skipped
	}

	// Load Terminal config
	config.Terminal = &TerminalConfig{
		SelectedTerminal: "cmd",
		ProjectDirs:      []string{},
	}
	if selectedTerminal, err := storage.GetConfig("terminal_selected"); err == nil && selectedTerminal != "" {
		config.Terminal.SelectedTerminal = selectedTerminal
	}
	if projectDirsStr, err := storage.GetConfig("terminal_projectDirs"); err == nil && projectDirsStr != "" {
		var dirs []string
		if err := json.Unmarshal([]byte(projectDirsStr), &dirs); err == nil {
			config.Terminal.ProjectDirs = dirs
		}
	}

	// Load Proxy config
	if proxyURL, err := storage.GetConfig("proxy_url"); err == nil && proxyURL != "" {
		config.Proxy = &ProxyConfig{URL: proxyURL}
	}

	return config, nil
}

// SaveToStorage saves configuration to SQLite storage
func (c *Config) SaveToStorage(storage StorageAdapter) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get existing endpoints from storage
	existingEndpoints, err := storage.GetEndpoints()
	if err != nil {
		return fmt.Errorf("failed to get existing endpoints: %w", err)
	}

	existingNames := make(map[string]bool)
	for _, ep := range existingEndpoints {
		existingNames[ep.Name] = true
	}

	// Save/update endpoints
	for i, ep := range c.Endpoints {
		endpoint := &StorageEndpoint{
			Name:        ep.Name,
			APIUrl:      ep.APIUrl,
			APIKey:      ep.APIKey,
			Enabled:     ep.Enabled,
			Transformer: ep.Transformer,
			Model:       ep.Model,
			Remark:      ep.Remark,
			SortOrder:   i, // Use array index as sort order
		}

		if existingNames[ep.Name] {
			if err := storage.UpdateEndpoint(endpoint); err != nil {
				return fmt.Errorf("failed to update endpoint %s: %w", ep.Name, err)
			}
		} else {
			if err := storage.SaveEndpoint(endpoint); err != nil {
				return fmt.Errorf("failed to save endpoint %s: %w", ep.Name, err)
			}
		}
		delete(existingNames, ep.Name)
	}

	// Delete endpoints that no longer exist
	for name := range existingNames {
		if err := storage.DeleteEndpoint(name); err != nil {
			return fmt.Errorf("failed to delete endpoint %s: %w", name, err)
		}
	}

	// Save app config
	storage.SetConfig("port", strconv.Itoa(c.Port))
	storage.SetConfig("logLevel", strconv.Itoa(c.LogLevel))
	storage.SetConfig("language", c.Language)
	storage.SetConfig("theme", c.Theme)
	storage.SetConfig("themeAuto", strconv.FormatBool(c.ThemeAuto))
	storage.SetConfig("autoLightTheme", c.AutoLightTheme)
	storage.SetConfig("autoDarkTheme", c.AutoDarkTheme)
	storage.SetConfig("windowWidth", strconv.Itoa(c.WindowWidth))
	storage.SetConfig("windowHeight", strconv.Itoa(c.WindowHeight))
	storage.SetConfig("closeWindowBehavior", c.CloseWindowBehavior)

	// Save WebDAV config
	if c.WebDAV != nil {
		storage.SetConfig("webdav_url", c.WebDAV.URL)
		storage.SetConfig("webdav_username", c.WebDAV.Username)
		storage.SetConfig("webdav_password", c.WebDAV.Password)
		storage.SetConfig("webdav_configPath", c.WebDAV.ConfigPath)
		storage.SetConfig("webdav_statsPath", c.WebDAV.StatsPath)
	}

	// Save Backup config
	if c.Backup != nil {
		storage.SetConfig("backup_provider", c.Backup.Provider)
		if c.Backup.Local != nil {
			storage.SetConfig("backup_local_dir", c.Backup.Local.Dir)
		}
		if c.Backup.S3 != nil {
			storage.SetConfig("backup_s3_endpoint", c.Backup.S3.Endpoint)
			storage.SetConfig("backup_s3_region", c.Backup.S3.Region)
			storage.SetConfig("backup_s3_bucket", c.Backup.S3.Bucket)
			storage.SetConfig("backup_s3_prefix", c.Backup.S3.Prefix)
			storage.SetConfig("backup_s3_accessKey", c.Backup.S3.AccessKey)
			storage.SetConfig("backup_s3_secretKey", c.Backup.S3.SecretKey)
			storage.SetConfig("backup_s3_sessionToken", c.Backup.S3.SessionToken)
			storage.SetConfig("backup_s3_useSSL", strconv.FormatBool(c.Backup.S3.UseSSL))
			storage.SetConfig("backup_s3_forcePathStyle", strconv.FormatBool(c.Backup.S3.ForcePathStyle))
		}
	}

	// Save Update config
	if c.Update != nil {
		storage.SetConfig("update_autoCheck", strconv.FormatBool(c.Update.AutoCheck))
		storage.SetConfig("update_checkInterval", strconv.Itoa(c.Update.CheckInterval))
		storage.SetConfig("update_lastCheckTime", c.Update.LastCheckTime)
		storage.SetConfig("update_skippedVersion", c.Update.SkippedVersion)
	}

	// Save Terminal config
	if c.Terminal != nil {
		storage.SetConfig("terminal_selected", c.Terminal.SelectedTerminal)
		if dirsJSON, err := json.Marshal(c.Terminal.ProjectDirs); err == nil {
			storage.SetConfig("terminal_projectDirs", string(dirsJSON))
		}
	}

	// Save Proxy config
	if c.Proxy != nil {
		storage.SetConfig("proxy_url", c.Proxy.URL)
	} else {
		storage.SetConfig("proxy_url", "")
	}

	return nil
}
