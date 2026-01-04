package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// CodexSessionMeta represents the first line of Codex session file
type CodexSessionMeta struct {
	Type    string `json:"type"`
	Payload struct {
		Cwd string `json:"cwd"`
	} `json:"payload"`
}

// getCodexSessionsDir returns ~/.codex/sessions path
func getCodexSessionsDir() string {
	var home string
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	} else {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".codex", "sessions")
}

// getCodexAliasFilePath returns the path to the Codex alias file
func getCodexAliasFilePath() string {
	return filepath.Join(getCodexSessionsDir(), "aliases.json")
}

// loadCodexAliases loads Codex session aliases from file
func loadCodexAliases() map[string]string {
	data, err := os.ReadFile(getCodexAliasFilePath())
	if err != nil {
		return make(map[string]string)
	}
	var aliases map[string]string
	if err := json.Unmarshal(data, &aliases); err != nil {
		return make(map[string]string)
	}
	return aliases
}

// saveCodexAliases saves Codex session aliases to file
func saveCodexAliases(aliases map[string]string) error {
	aliasFile := getCodexAliasFilePath()
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

// GetCodexSessionsForProject returns Codex sessions for a project directory
func GetCodexSessionsForProject(projectDir string) ([]SessionInfo, error) {
	sessionsDir := getCodexSessionsDir()

	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return []SessionInfo{}, nil
	}

	aliases := loadCodexAliases()
	var sessions []SessionInfo

	// Recursively scan sessions directory
	filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Match rollout-*.jsonl files
		if !strings.HasPrefix(info.Name(), "rollout-") || !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}

		// Parse session metadata and summary
		meta, summary, err := parseCodexSession(path)
		if err != nil {
			return nil
		}

		// Check if cwd matches project directory
		if !pathsEqual(meta.Payload.Cwd, projectDir) {
			return nil
		}

		// Extract sessionId from filename
		sessionId := extractCodexSessionId(info.Name())

		sessions = append(sessions, SessionInfo{
			SessionID: sessionId,
			ModTime:   info.ModTime().Unix(),
			Size:      info.Size(),
			Summary:   summary,
			Alias:     aliases[sessionId],
		})

		return nil
	})

	// Sort by modification time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModTime > sessions[j].ModTime
	})

	return sessions, nil
}

// parseCodexSession parses Codex session file for metadata and first user message
func parseCodexSession(filePath string) (*CodexSessionMeta, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var meta *CodexSessionMeta
	var summary string
	lineCount := 0

	for scanner.Scan() && lineCount < 30 {
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

		// First line: session_meta
		if msgType == "session_meta" && meta == nil {
			var m CodexSessionMeta
			if err := json.Unmarshal([]byte(line), &m); err == nil {
				meta = &m
			}
			continue
		}

		// Find first user message in event_msg
		if msgType == "event_msg" && summary == "" {
			payload, ok := data["payload"].(map[string]interface{})
			if !ok {
				continue
			}
			payloadType, _ := payload["type"].(string)
			if payloadType == "user_message" {
				if msg, ok := payload["message"].(string); ok && msg != "" {
					runes := []rune(msg)
					if len(runes) > 100 {
						summary = string(runes[:100]) + "..."
					} else {
						summary = msg
					}
				}
			}
		}

		// Stop if we have both
		if meta != nil && summary != "" {
			break
		}
	}

	if meta == nil {
		return nil, "", os.ErrNotExist
	}

	return meta, summary, nil
}

// extractCodexSessionId extracts session ID from filename
// rollout-2025-01-03T12-34-56-uuid.jsonl -> uuid
func extractCodexSessionId(filename string) string {
	name := strings.TrimPrefix(filename, "rollout-")
	name = strings.TrimSuffix(name, ".jsonl")
	// Skip timestamp part (19 chars + 1 hyphen = 20)
	if len(name) > 20 {
		return name[20:]
	}
	return name
}

// pathsEqual compares two paths for equality (case-insensitive on Windows)
func pathsEqual(path1, path2 string) bool {
	clean1 := filepath.Clean(path1)
	clean2 := filepath.Clean(path2)

	if runtime.GOOS == "windows" {
		return strings.EqualFold(clean1, clean2)
	}
	return clean1 == clean2
}

// GetCodexSessionData returns all messages for a specific Codex session
func GetCodexSessionData(sessionID string) ([]MessageData, error) {
	sessionFile := findCodexSessionFile(sessionID)
	if sessionFile == "" {
		return nil, os.ErrNotExist
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

		// Parse event_msg for user messages
		if msgType == "event_msg" {
			payload, ok := data["payload"].(map[string]interface{})
			if !ok {
				continue
			}
			payloadType, _ := payload["type"].(string)
			if payloadType == "user_message" {
				if msg, ok := payload["message"].(string); ok && msg != "" {
					messages = append(messages, MessageData{Type: "user", Content: msg})
				}
			} else if payloadType == "agent_message" {
				if msg, ok := payload["message"].(string); ok && msg != "" {
					messages = append(messages, MessageData{Type: "assistant", Content: msg})
				}
			}
		}
	}

	return messages, scanner.Err()
}

// DeleteCodexSession deletes a Codex session file
func DeleteCodexSession(sessionID string) error {
	sessionFile := findCodexSessionFile(sessionID)
	if sessionFile == "" {
		return nil
	}
	// Also remove alias
	aliases := loadCodexAliases()
	delete(aliases, sessionID)
	saveCodexAliases(aliases)

	return os.Remove(sessionFile)
}

// RenameCodexSession sets an alias for a Codex session
func RenameCodexSession(sessionID, alias string) error {
	aliases := loadCodexAliases()
	if alias == "" {
		delete(aliases, sessionID)
	} else {
		aliases[sessionID] = alias
	}
	return saveCodexAliases(aliases)
}

// findCodexSessionFile finds session file by ID
func findCodexSessionFile(sessionID string) string {
	sessionsDir := getCodexSessionsDir()
	var sessionFile string

	filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.Contains(info.Name(), sessionID) {
			sessionFile = path
			return filepath.SkipAll
		}
		return nil
	})

	return sessionFile
}
