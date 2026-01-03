package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
)

// ClaudeSettings represents ~/.claude/settings.json structure
// We only care about the hooks field, other fields are preserved as-is
type ClaudeSettings map[string]interface{}

// ClaudeHookConfig represents the hooks.Stop configuration
type ClaudeHookConfig struct {
	Hooks []ClaudeHookItem `json:"hooks"`
}

// ClaudeHookItem represents a single hook item
type ClaudeHookItem struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// ClaudeConfigService handles Claude settings.json operations
type ClaudeConfigService struct {
	config *config.Config
}

// NewClaudeConfigService creates a new service
func NewClaudeConfigService(cfg *config.Config) *ClaudeConfigService {
	return &ClaudeConfigService{config: cfg}
}

// getClaudeSettingsPath returns ~/.claude/settings.json path
func (s *ClaudeConfigService) getClaudeSettingsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".claude", "settings.json"), nil
}

// readClaudeSettings reads and parses settings.json
func (s *ClaudeConfigService) readClaudeSettings() (ClaudeSettings, error) {
	path, err := s.getClaudeSettingsPath()
	if err != nil {
		return nil, err
	}

	// File doesn't exist, return empty settings
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return make(ClaudeSettings), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		// If JSON is invalid, return empty settings instead of error
		logger.Warn("Failed to parse Claude settings.json, using empty config: %v", err)
		return make(ClaudeSettings), nil
	}

	return settings, nil
}

// writeClaudeSettings writes settings to file
func (s *ClaudeConfigService) writeClaudeSettings(settings ClaudeSettings) error {
	path, err := s.getClaudeSettingsPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// getNotificationText returns localized notification text
func (s *ClaudeConfigService) getNotificationText() (title, message string) {
	lang := s.config.GetLanguage()
	if lang == "en" {
		return "ccNexus", "Task completed, ready for interaction!"
	}
	return "ccNexus", "任务完成，已就绪！"
}

// generateNotificationScript generates PowerShell notification script for Windows
func (s *ClaudeConfigService) generateNotificationScript(notifType string) string {
	title, message := s.getNotificationText()

	if runtime.GOOS != "windows" {
		// macOS support
		if runtime.GOOS == "darwin" {
			switch notifType {
			case "toast":
				return fmt.Sprintf(`osascript -e 'display notification "%s" with title "%s" sound name "Glass"'`, message, title)
			case "dialog":
				return fmt.Sprintf(`osascript -e 'display dialog "%s" with title "%s" buttons {"OK"} default button 1 with icon note'`, message, title)
			default:
				return ""
			}
		}
		// Linux support
		if runtime.GOOS == "linux" {
			switch notifType {
			case "toast":
				return fmt.Sprintf(`notify-send "%s" "%s"`, title, message)
			case "dialog":
				return fmt.Sprintf(`zenity --info --title="%s" --text="%s" 2>/dev/null || notify-send "%s" "%s"`, title, message, title, message)
			default:
				return ""
			}
		}
		logger.Warn("Notifications not supported on this platform: %s", runtime.GOOS)
		return ""
	}

	// Windows PowerShell commands
	switch notifType {
	case "toast":
		// Windows BalloonTip - 系统托盘气泡通知，从右下角滑出，5秒后自动消失
		return fmt.Sprintf(`powershell -Command "[void][reflection.assembly]::loadwithpartialname('System.Windows.Forms');[void][reflection.assembly]::loadwithpartialname('System.Drawing');$n=new-object system.windows.forms.notifyicon;$n.icon=[System.Drawing.SystemIcons]::Information;$n.visible=$true;$n.showballoontip(5000,'%s','%s','Info');Start-Sleep -Seconds 6;$n.Dispose()"`, title, message)
	case "dialog":
		// MessageBox 对话框 - 需要手动点击确认才消失
		return fmt.Sprintf(`powershell -Command "Add-Type -AssemblyName PresentationFramework;[System.Windows.MessageBox]::Show('%s','%s','OK','Information')"`, message, title)
	default:
		return ""
	}
}

// UpdateNotificationHook updates the hooks.Stop in settings.json
func (s *ClaudeConfigService) UpdateNotificationHook() error {
	enabled, notifType := s.config.GetClaudeNotification()

	settings, err := s.readClaudeSettings()
	if err != nil {
		logger.Warn("Failed to read Claude settings: %v, creating new config", err)
		settings = make(ClaudeSettings)
	}

	// Get or create hooks map
	var hooks map[string]interface{}
	if h, ok := settings["hooks"].(map[string]interface{}); ok {
		hooks = h
	} else {
		hooks = make(map[string]interface{})
	}

	// Update hooks.Stop based on config
	if enabled && notifType != "disabled" {
		script := s.generateNotificationScript(notifType)
		if script != "" {
			// Create Stop hook configuration
			stopHook := []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": script,
						},
					},
				},
			}
			hooks["Stop"] = stopHook
			logger.Info("Claude notification hook enabled (type: %s)", notifType)
		}
	} else {
		// Remove Stop hook if disabled
		delete(hooks, "Stop")
		logger.Info("Claude notification hook disabled")
	}

	// Update settings
	if len(hooks) > 0 {
		settings["hooks"] = hooks
	} else {
		delete(settings, "hooks")
	}

	if err := s.writeClaudeSettings(settings); err != nil {
		return fmt.Errorf("failed to update Claude settings: %w", err)
	}

	return nil
}

// GetCurrentHook returns the current hooks.Stop value
func (s *ClaudeConfigService) GetCurrentHook() string {
	settings, err := s.readClaudeSettings()
	if err != nil {
		return ""
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return ""
	}

	stopHook, ok := hooks["Stop"]
	if !ok {
		return ""
	}

	// Try to extract the command from the hook structure
	if stopArr, ok := stopHook.([]interface{}); ok && len(stopArr) > 0 {
		if first, ok := stopArr[0].(map[string]interface{}); ok {
			if innerHooks, ok := first["hooks"].([]interface{}); ok && len(innerHooks) > 0 {
				if innerFirst, ok := innerHooks[0].(map[string]interface{}); ok {
					if cmd, ok := innerFirst["command"].(string); ok {
						return cmd
					}
				}
			}
		}
	}

	return ""
}

// IsNotificationEnabled checks if notification is currently enabled in Claude settings
func (s *ClaudeConfigService) IsNotificationEnabled() bool {
	return s.GetCurrentHook() != ""
}
