package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/storage"
	"github.com/lich0821/ccNexus/internal/tray"
)

// SettingsService handles settings operations
type SettingsService struct {
	config  *config.Config
	storage *storage.SQLiteStorage
}

// NewSettingsService creates a new SettingsService
func NewSettingsService(cfg *config.Config, s *storage.SQLiteStorage) *SettingsService {
	return &SettingsService{config: cfg, storage: s}
}

// GetConfig returns the current configuration as JSON
func (s *SettingsService) GetConfig() string {
	data, _ := json.Marshal(s.config)
	return string(data)
}

// UpdateConfig updates the configuration
func (s *SettingsService) UpdateConfig(configJSON string, proxy interface{ UpdateConfig(*config.Config) error }) error {
	var newConfig config.Config
	if err := json.Unmarshal([]byte(configJSON), &newConfig); err != nil {
		return fmt.Errorf("invalid config format: %w", err)
	}

	if strings.TrimSpace(newConfig.ListenAddr) == "" {
		newConfig.ListenAddr = "127.0.0.1"
	}

	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if err := proxy.UpdateConfig(&newConfig); err != nil {
		return err
	}

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := newConfig.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	*s.config = newConfig
	return nil
}

// UpdatePort updates the proxy port
func (s *SettingsService) UpdatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}

	s.config.UpdatePort(port)

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	return nil
}

// UpdateNetwork updates port and listen address together
func (s *SettingsService) UpdateNetwork(port int, listenAddr string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}

	if strings.TrimSpace(listenAddr) == "" {
		return fmt.Errorf("invalid listen address")
	}

	s.config.UpdatePort(port)
	s.config.UpdateListenAddr(listenAddr)

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	return nil
}

// GetSystemLanguage detects the system language
func (s *SettingsService) GetSystemLanguage() string {
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

	if strings.Contains(strings.ToLower(locale), "zh") {
		return "zh-CN"
	}
	return "en"
}

// GetLanguage returns the current language setting
func (s *SettingsService) GetLanguage() string {
	lang := s.config.GetLanguage()
	if lang == "" {
		return s.GetSystemLanguage()
	}
	return lang
}

// SetLanguage sets the UI language
func (s *SettingsService) SetLanguage(language string) error {
	s.config.UpdateLanguage(language)

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save language: %w", err)
		}
	}

	tray.UpdateLanguage(language)
	logger.Info("Language changed to: %s", language)
	return nil
}

// GetTheme returns the current theme setting
func (s *SettingsService) GetTheme() string {
	theme := s.config.GetTheme()
	if theme == "" {
		return "light"
	}
	return theme
}

// SetTheme sets the UI theme
func (s *SettingsService) SetTheme(theme string) error {
	s.config.UpdateTheme(theme)

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save theme: %w", err)
		}
	}

	logger.Info("Theme changed to: %s", theme)
	return nil
}

// GetThemeAuto returns whether auto theme switching is enabled
func (s *SettingsService) GetThemeAuto() bool {
	return s.config.GetThemeAuto()
}

// SetThemeAuto enables or disables auto theme switching
func (s *SettingsService) SetThemeAuto(auto bool) error {
	s.config.UpdateThemeAuto(auto)

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save theme auto setting: %w", err)
		}
	}

	logger.Info("Theme auto mode changed to: %v", auto)
	return nil
}

// GetAutoLightTheme returns the theme to use in daytime when auto mode is on
func (s *SettingsService) GetAutoLightTheme() string {
	theme := s.config.GetAutoLightTheme()
	if theme == "" {
		return "light"
	}
	return theme
}

// SetAutoLightTheme sets the theme to use in daytime when auto mode is on
func (s *SettingsService) SetAutoLightTheme(theme string) error {
	s.config.UpdateAutoLightTheme(theme)

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save auto light theme: %w", err)
		}
	}

	logger.Info("Auto light theme changed to: %s", theme)
	return nil
}

// GetAutoDarkTheme returns the theme to use in nighttime when auto mode is on
func (s *SettingsService) GetAutoDarkTheme() string {
	theme := s.config.GetAutoDarkTheme()
	if theme == "" {
		return "dark"
	}
	return theme
}

// SetAutoDarkTheme sets the theme to use in nighttime when auto mode is on
func (s *SettingsService) SetAutoDarkTheme(theme string) error {
	s.config.UpdateAutoDarkTheme(theme)

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save auto dark theme: %w", err)
		}
	}

	logger.Info("Auto dark theme changed to: %s", theme)
	return nil
}

// GetLogs returns all log entries
func (s *SettingsService) GetLogs() string {
	logs := logger.GetLogger().GetLogs()
	data, _ := json.Marshal(logs)
	return string(data)
}

// GetLogsByLevel returns logs filtered by level
func (s *SettingsService) GetLogsByLevel(level int) string {
	logs := logger.GetLogger().GetLogsByLevel(logger.LogLevel(level))
	data, _ := json.Marshal(logs)
	return string(data)
}

// ClearLogs clears all log entries
func (s *SettingsService) ClearLogs() {
	logger.GetLogger().Clear()
}

// SetLogLevel sets the minimum log level to record
func (s *SettingsService) SetLogLevel(level int) {
	logger.GetLogger().SetMinLevel(logger.LogLevel(level))
	s.config.UpdateLogLevel(level)

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			logger.Warn("Failed to save log level: %v", err)
		} else {
			logger.Debug("Log level saved: %d", level)
		}
	}
}

// GetLogLevel returns the current minimum log level
func (s *SettingsService) GetLogLevel() int {
	return s.config.GetLogLevel()
}

// SetCloseWindowBehavior sets the user's preference for close window behavior
func (s *SettingsService) SetCloseWindowBehavior(behavior string) error {
	if behavior != "quit" && behavior != "minimize" && behavior != "ask" {
		return fmt.Errorf("invalid behavior: %s (must be 'quit', 'minimize', or 'ask')", behavior)
	}

	s.config.UpdateCloseWindowBehavior(behavior)

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			logger.Warn("Failed to save close window behavior: %v", err)
			return err
		}
	}

	logger.Info("Close window behavior set to: %s", behavior)
	return nil
}

// SaveWindowSize saves the window size to config
func (s *SettingsService) SaveWindowSize(width, height int) {
	if width > 0 && height > 0 {
		s.config.UpdateWindowSize(width, height)
		if s.storage != nil {
			configAdapter := storage.NewConfigStorageAdapter(s.storage)
			if err := s.config.SaveToStorage(configAdapter); err != nil {
				logger.Warn("Failed to save window size: %v", err)
			} else {
				logger.Debug("Window size saved: %dx%d", width, height)
			}
		}
	}
}

// GetProxyURL returns the current proxy URL
func (s *SettingsService) GetProxyURL() string {
	if proxy := s.config.GetProxy(); proxy != nil {
		return proxy.URL
	}
	return ""
}

// SetProxyURL sets the proxy URL
func (s *SettingsService) SetProxyURL(proxyURL string) error {
	var proxyCfg *config.ProxyConfig
	if proxyURL != "" {
		proxyCfg = &config.ProxyConfig{URL: proxyURL}
	}
	s.config.UpdateProxy(proxyCfg)

	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save proxy config: %w", err)
		}
	}

	logger.Info("Proxy URL changed to: %s", proxyURL)
	return nil
}

// SettingsData represents the settings data for batch save
type SettingsData struct {
	CloseWindowBehavior       string `json:"closeWindowBehavior"`
	ProxyURL                  string `json:"proxyUrl"`
	Theme                     string `json:"theme"`
	ThemeAuto                 bool   `json:"themeAuto"`
	AutoLightTheme            string `json:"autoLightTheme"`
	AutoDarkTheme             string `json:"autoDarkTheme"`
	ClaudeNotificationEnabled bool   `json:"claudeNotificationEnabled"`
	ClaudeNotificationType    string `json:"claudeNotificationType"`
}

// SaveSettings saves all settings in a single operation to avoid database lock issues
func (s *SettingsService) SaveSettings(settingsJSON string) error {
	var settings SettingsData
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return fmt.Errorf("invalid settings format: %w", err)
	}

	// Validate close window behavior
	if settings.CloseWindowBehavior != "" &&
		settings.CloseWindowBehavior != "quit" &&
		settings.CloseWindowBehavior != "minimize" &&
		settings.CloseWindowBehavior != "ask" {
		return fmt.Errorf("invalid close window behavior: %s", settings.CloseWindowBehavior)
	}

	// Update all settings in memory
	if settings.CloseWindowBehavior != "" {
		s.config.UpdateCloseWindowBehavior(settings.CloseWindowBehavior)
	}

	if settings.Theme != "" {
		s.config.UpdateTheme(settings.Theme)
	}

	s.config.UpdateThemeAuto(settings.ThemeAuto)

	// Update auto theme settings
	if settings.AutoLightTheme != "" {
		s.config.UpdateAutoLightTheme(settings.AutoLightTheme)
	}
	if settings.AutoDarkTheme != "" {
		s.config.UpdateAutoDarkTheme(settings.AutoDarkTheme)
	}

	// Update proxy config
	var proxyCfg *config.ProxyConfig
	if settings.ProxyURL != "" {
		proxyCfg = &config.ProxyConfig{URL: settings.ProxyURL}
	}
	s.config.UpdateProxy(proxyCfg)

	// Update Claude notification config
	// Validate notification type
	if settings.ClaudeNotificationType != "" &&
		settings.ClaudeNotificationType != "toast" &&
		settings.ClaudeNotificationType != "dialog" &&
		settings.ClaudeNotificationType != "disabled" {
		return fmt.Errorf("invalid notification type: %s", settings.ClaudeNotificationType)
	}
	s.config.UpdateClaudeNotification(settings.ClaudeNotificationEnabled, settings.ClaudeNotificationType)

	// Save to storage only once
	if s.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(s.storage)
		if err := s.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save settings: %w", err)
		}
	}

	// Apply Claude notification hook to ~/.claude/settings.json
	claudeService := NewClaudeConfigService(s.config)
	if err := claudeService.UpdateNotificationHook(); err != nil {
		logger.Warn("Failed to update Claude notification hook: %v", err)
		// Don't fail the whole save operation, just log the warning
	}

	logger.Info("Settings saved: closeWindowBehavior=%s, theme=%s, themeAuto=%v, proxyUrl=%s, claudeNotification=%v",
		settings.CloseWindowBehavior, settings.Theme, settings.ThemeAuto, settings.ProxyURL, settings.ClaudeNotificationEnabled)
	return nil
}
