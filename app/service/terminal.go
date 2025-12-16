package service

import (
    "encoding/json"
    "fmt"

    "github.com/lich0821/ccNexus/internal/config"
    "github.com/lich0821/ccNexus/internal/logger"
    "github.com/lich0821/ccNexus/internal/session"
    "github.com/lich0821/ccNexus/internal/storage"
    "github.com/lich0821/ccNexus/internal/terminal"
)

// TerminalService handles terminal and session operations
type TerminalService struct {
    config  *config.Config
    storage *storage.SQLiteStorage
}

// NewTerminalService creates a new TerminalService
func NewTerminalService(cfg *config.Config, s *storage.SQLiteStorage) *TerminalService {
    return &TerminalService{config: cfg, storage: s}
}

// DetectTerminals detects available terminals on the system
func (t *TerminalService) DetectTerminals() string {
    terminals := terminal.DetectTerminals()
    data, _ := json.Marshal(terminals)
    return string(data)
}

// GetTerminalConfig returns the terminal configuration
func (t *TerminalService) GetTerminalConfig() string {
    terminalCfg := t.config.GetTerminal()
    data, _ := json.Marshal(terminalCfg)
    return string(data)
}

// SaveTerminalConfig saves the terminal configuration
func (t *TerminalService) SaveTerminalConfig(selectedTerminal string, projectDirs []string) error {
    terminalCfg := &config.TerminalConfig{
        SelectedTerminal: selectedTerminal,
        ProjectDirs:      projectDirs,
    }
    t.config.UpdateTerminal(terminalCfg)

    if t.storage != nil {
        configAdapter := storage.NewConfigStorageAdapter(t.storage)
        if err := t.config.SaveToStorage(configAdapter); err != nil {
            return fmt.Errorf("failed to save terminal config: %w", err)
        }
    }

    logger.Info("Terminal config saved: terminal=%s, dirs=%d", selectedTerminal, len(projectDirs))
    return nil
}

// AddProjectDir adds a project directory
func (t *TerminalService) AddProjectDir(dir string) error {
    terminalCfg := t.config.GetTerminal()
    for _, d := range terminalCfg.ProjectDirs {
        if d == dir {
            return fmt.Errorf("directory_already_exists")
        }
    }
    terminalCfg.ProjectDirs = append(terminalCfg.ProjectDirs, dir)
    t.config.UpdateTerminal(terminalCfg)

    if t.storage != nil {
        configAdapter := storage.NewConfigStorageAdapter(t.storage)
        if err := t.config.SaveToStorage(configAdapter); err != nil {
            return fmt.Errorf("failed to save terminal config: %w", err)
        }
    }

    logger.Info("Project directory added: %s", dir)
    return nil
}

// RemoveProjectDir removes a project directory
func (t *TerminalService) RemoveProjectDir(dir string) error {
    terminalCfg := t.config.GetTerminal()
    newDirs := make([]string, 0, len(terminalCfg.ProjectDirs))
    for _, d := range terminalCfg.ProjectDirs {
        if d != dir {
            newDirs = append(newDirs, d)
        }
    }
    terminalCfg.ProjectDirs = newDirs
    t.config.UpdateTerminal(terminalCfg)

    if t.storage != nil {
        configAdapter := storage.NewConfigStorageAdapter(t.storage)
        if err := t.config.SaveToStorage(configAdapter); err != nil {
            return fmt.Errorf("failed to save terminal config: %w", err)
        }
    }

    logger.Info("Project directory removed: %s", dir)
    return nil
}

// LaunchTerminal launches a terminal in the specified directory
func (t *TerminalService) LaunchTerminal(dir string) error {
    terminalCfg := t.config.GetTerminal()
    terminalID := terminalCfg.SelectedTerminal
    if terminalID == "" {
        terminalID = "cmd"
    }

    logger.Info("Launching terminal: %s in %s", terminalID, dir)
    return terminal.LaunchTerminal(terminalID, dir)
}

// GetSessions returns all sessions for a project directory
func (t *TerminalService) GetSessions(projectDir string) string {
    sessions, err := session.GetSessionsForProject(projectDir)
    if err != nil {
        logger.Error("Failed to get sessions for %s: %v", projectDir, err)
        result := map[string]interface{}{
            "success":  false,
            "message":  err.Error(),
            "sessions": []interface{}{},
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    result := map[string]interface{}{
        "success":  true,
        "sessions": sessions,
    }
    data, _ := json.Marshal(result)
    return string(data)
}

// DeleteSession deletes a session
func (t *TerminalService) DeleteSession(projectDir, sessionID string) error {
    if err := session.DeleteSession(projectDir, sessionID); err != nil {
        logger.Error("Failed to delete session %s: %v", sessionID, err)
        return err
    }
    logger.Info("Session deleted: %s", sessionID)
    return nil
}

// RenameSession sets an alias for a session
func (t *TerminalService) RenameSession(projectDir, sessionID, alias string) error {
    if err := session.RenameSession(projectDir, sessionID, alias); err != nil {
        logger.Error("Failed to rename session %s: %v", sessionID, err)
        return err
    }
    logger.Info("Session renamed: %s -> %s", sessionID, alias)
    return nil
}

// GetSessionData returns all messages for a specific session
func (t *TerminalService) GetSessionData(projectDir, sessionID string) string {
    messages, err := session.GetSessionData(projectDir, sessionID)
    if err != nil {
        logger.Error("Failed to get session data for %s: %v", sessionID, err)
        result := map[string]interface{}{
            "success": false,
            "message": err.Error(),
            "data":    []interface{}{},
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    result := map[string]interface{}{
        "success": true,
        "data":    messages,
    }
    data, _ := json.Marshal(result)
    return string(data)
}

// LaunchSessionTerminal launches a terminal with optional session resume
func (t *TerminalService) LaunchSessionTerminal(dir, sessionID string) error {
    terminalCfg := t.config.GetTerminal()
    terminalID := terminalCfg.SelectedTerminal
    if terminalID == "" {
        terminalID = "cmd"
    }

    if sessionID != "" {
        logger.Info("Launching terminal with session: %s in %s", sessionID, dir)
    } else {
        logger.Info("Launching new terminal in %s", dir)
    }
    return terminal.LaunchTerminalWithSession(terminalID, dir, sessionID)
}
