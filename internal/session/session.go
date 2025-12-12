package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// SessionInfo represents a session's information
type SessionInfo struct {
	SessionID string `json:"sessionId"`
	Summary   string `json:"summary"`
	ModTime   int64  `json:"modTime"`
	Size      int64  `json:"size"`
	Alias     string `json:"alias,omitempty"`
}

// getClaudeProjectsDir returns the Claude projects directory
func getClaudeProjectsDir() string {
	var home string
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	} else {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".claude", "projects")
}

// getAliasFilePath returns the path to the alias file
func getAliasFilePath() string {
	var home string
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	} else {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".claude", "cc-tool", "aliases.json")
}

// encodeProjectPath encodes a project path to Claude Code's directory name format
// Claude Code encodes paths by replacing special characters with hyphens
// Example: E:\OthProjects\ccNexus -> E--OthProjects-ccNexus
func encodeProjectPath(projectPath string) string {
	// Normalize to forward slashes first
	normalized := filepath.ToSlash(projectPath)
	// Remove trailing slash
	normalized = strings.TrimSuffix(normalized, "/")

	// Replace special characters with hyphens
	// : and / are replaced with -
	encoded := strings.ReplaceAll(normalized, ":", "-")
	encoded = strings.ReplaceAll(encoded, "/", "-")

	return encoded
}

// GetSessionsForProject returns all sessions for a project directory
func GetSessionsForProject(projectDir string) ([]SessionInfo, error) {
	projectEncoded := encodeProjectPath(projectDir)
	sessionsDir := filepath.Join(getClaudeProjectsDir(), projectEncoded)

	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return []SessionInfo{}, nil
	}

	aliases := loadAliases()
	var sessions []SessionInfo

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		// Skip agent sessions
		if strings.HasPrefix(entry.Name(), "agent-") {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), ".jsonl")
		filePath := filepath.Join(sessionsDir, entry.Name())

		info, err := entry.Info()
		if err != nil {
			continue
		}

		summary, isValid := parseSessionSummary(filePath)

		// Skip non-session files (files without user or assistant messages)
		if !isValid {
			continue
		}

		alias := aliases[sessionID]

		sessions = append(sessions, SessionInfo{
			SessionID: sessionID,
			Summary:   summary,
			ModTime:   info.ModTime().Unix(),
			Size:      info.Size(),
			Alias:     alias,
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModTime > sessions[j].ModTime
	})

	return sessions, nil
}

// extractContent extracts text from message content (handles both string and array formats)
func extractContent(content interface{}) string {
	// Try string format first (user messages)
	if str, ok := content.(string); ok {
		return str
	}

	// Try array format (assistant messages)
	if arr, ok := content.([]interface{}); ok {
		for _, item := range arr {
			if obj, ok := item.(map[string]interface{}); ok {
				if itemType, ok := obj["type"].(string); ok && itemType == "text" {
					if text, ok := obj["text"].(string); ok {
						return text
					}
				}
			}
		}
	}

	return ""
}

// parseSessionSummary extracts the first user message as summary and checks if file is valid
func parseSessionSummary(filePath string) (string, bool) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineCount := 0
	var firstSystemMsg string
	hasValidMessage := false

	for scanner.Scan() && lineCount < 20 {
		lineCount++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}

		msgType, _ := data["type"].(string)

		// Look for user message first
		if msgType == "user" {
			hasValidMessage = true
			if message, ok := data["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					runes := []rune(content)
					if len(runes) > 100 {
						return string(runes[:100]) + "...", true
					}
					return content, true
				}
			}
		}

		// Save first assistant message as fallback
		if msgType == "assistant" {
			hasValidMessage = true
			if firstSystemMsg == "" {
				if message, ok := data["message"].(map[string]interface{}); ok {
					content := extractContent(message["content"])
					if content != "" {
						runes := []rune(content)
						if len(runes) > 100 {
							firstSystemMsg = string(runes[:100]) + "..."
						} else {
							firstSystemMsg = content
						}
					}
				}
			}
		}
	}

	return firstSystemMsg, hasValidMessage
}

// loadAliases loads session aliases from file
func loadAliases() map[string]string {
	aliasFile := getAliasFilePath()
	data, err := os.ReadFile(aliasFile)
	if err != nil {
		return make(map[string]string)
	}

	var aliases map[string]string
	if err := json.Unmarshal(data, &aliases); err != nil {
		return make(map[string]string)
	}

	return aliases
}

// saveAliases saves session aliases to file
func saveAliases(aliases map[string]string) error {
	aliasFile := getAliasFilePath()

	// Ensure directory exists
	dir := filepath.Dir(aliasFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(aliases, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(aliasFile, data, 0644)
}

// RenameSession sets an alias for a session
func RenameSession(projectDir, sessionID, alias string) error {
	aliases := loadAliases()
	if alias == "" {
		delete(aliases, sessionID)
	} else {
		aliases[sessionID] = alias
	}
	return saveAliases(aliases)
}

// DeleteSession deletes a session file
func DeleteSession(projectDir, sessionID string) error {
	projectEncoded := encodeProjectPath(projectDir)
	sessionFile := filepath.Join(getClaudeProjectsDir(), projectEncoded, sessionID+".jsonl")

	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		return nil
	}

	// Also remove alias if exists
	aliases := loadAliases()
	delete(aliases, sessionID)
	saveAliases(aliases)

	return os.Remove(sessionFile)
}

// FormatTime formats a Unix timestamp to a readable string
func FormatTime(timestamp int64) string {
	t := time.Unix(timestamp, 0)
	now := time.Now()

	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	if t.Year() == now.Year() {
		return t.Format("01-02 15:04")
	}
	return t.Format("2006-01-02 15:04")
}

// FormatSize formats file size to a readable string
func FormatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
	)

	switch {
	case size >= MB:
		return strings.TrimSuffix(strings.TrimSuffix(
			strings.Replace(string(rune(size/MB))+"."+ string(rune((size%MB)*10/MB)), ".", "", 1),
			"0"), ".") + " MB"
	case size >= KB:
		return strings.TrimSuffix(strings.TrimSuffix(
			string(rune(size/KB))+"."+string(rune((size%KB)*10/KB)),
			"0"), ".") + " KB"
	default:
		return string(rune(size)) + " B"
	}
}

// MessageData represents a single message in a session
type MessageData struct {
	Type    string `json:"type"`    // "user" or "assistant"
	Content string `json:"content"` // message content
}

// GetSessionData returns all messages for a specific session
func GetSessionData(projectDir, sessionID string) ([]MessageData, error) {
	projectEncoded := encodeProjectPath(projectDir)
	sessionFile := filepath.Join(getClaudeProjectsDir(), projectEncoded, sessionID+".jsonl")

	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		return nil, err
	}

	file, err := os.Open(sessionFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []MessageData
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}

		msgType, _ := data["type"].(string)
		if msgType != "user" && msgType != "assistant" {
			continue
		}

		message, ok := data["message"].(map[string]interface{})
		if !ok {
			continue
		}

		content := extractContent(message["content"])
		if content != "" {
			messages = append(messages, MessageData{
				Type:    msgType,
				Content: content,
			})
		}
	}

	return messages, scanner.Err()
}
