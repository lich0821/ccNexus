package logger

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (l LogLevel) Icon() string {
	switch l {
	case DEBUG:
		return "🔍"
	case INFO:
		return "ℹ️"
	case WARN:
		return "⚠️"
	case ERROR:
		return "❌"
	default:
		return "📝"
	}
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	Icon      string    `json:"icon"`
	LevelStr  string    `json:"levelStr"`
}

// Logger manages application logs
type Logger struct {
	mu           sync.RWMutex
	entries      []LogEntry
	maxSize      int
	minLevel     LogLevel // Minimum level to record
	consoleLevel LogLevel // Minimum level to print to console
	debugFile    *os.File // Debug log file (only in debug mode)
	debugMu      sync.Mutex
}

var (
	instance *Logger
	once     sync.Once
)

// GetLogger returns the singleton logger instance
func GetLogger() *Logger {
	once.Do(func() {
		instance = &Logger{
			entries:      make([]LogEntry, 0),
			maxSize:      1000,  // Keep last 1000 logs
			minLevel:     DEBUG, // Default to DEBUG level to capture all logs
			consoleLevel: INFO,  // Default console level to INFO (skip DEBUG in console)
		}
	})
	return instance
}

// SetMinLevel sets the minimum log level to record
func (l *Logger) SetMinLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

// SetConsoleLevel sets the minimum log level to print to console
func (l *Logger) SetConsoleLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.consoleLevel = level
}

// GetMinLevel returns the current minimum log level
func (l *Logger) GetMinLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.minLevel
}

// Log adds a new log entry
func (l *Logger) Log(level LogLevel, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Skip if below minimum level
	if level < l.minLevel {
		return
	}

	message := fmt.Sprintf(format, args...)
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Icon:      level.Icon(),
		LevelStr:  level.String(),
	}

	// Add to memory
	l.entries = append(l.entries, entry)

	// Trim if exceeds max size
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[len(l.entries)-l.maxSize:]
	}

	// Print to console only if level >= consoleLevel
	if level >= l.consoleLevel {
		fmt.Printf("%s [%s] %s\n", entry.Icon, entry.LevelStr, entry.Message)
	}
}

// GetLogs returns all log entries
func (l *Logger) GetLogs() []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy
	result := make([]LogEntry, len(l.entries))
	copy(result, l.entries)
	return result
}

// GetLogsByLevel returns logs filtered by minimum level
func (l *Logger) GetLogsByLevel(minLevel LogLevel) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]LogEntry, 0)
	for _, entry := range l.entries {
		if entry.Level >= minLevel {
			result = append(result, entry)
		}
	}
	return result
}

// Clear removes all log entries
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = make([]LogEntry, 0)
}

// Convenience methods
func Debug(format string, args ...interface{}) {
	GetLogger().Log(DEBUG, format, args...)
}

func Info(format string, args ...interface{}) {
	GetLogger().Log(INFO, format, args...)
}

func Warn(format string, args ...interface{}) {
	GetLogger().Log(WARN, format, args...)
}

func Error(format string, args ...interface{}) {
	GetLogger().Log(ERROR, format, args...)
}

// EnableDebugFile enables debug file logging (only in debug mode)
func (l *Logger) EnableDebugFile(filepath string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.debugFile != nil {
		l.debugFile.Close()
	}

	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	l.debugFile = f
	return nil
}

// DebugLog writes to debug.log file (bypasses log level)
func (l *Logger) DebugLog(format string, args ...interface{}) {
	l.debugMu.Lock()
	defer l.debugMu.Unlock()

	if l.debugFile == nil {
		return
	}

	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	fmt.Fprintf(l.debugFile, "[%s] %s\n", timestamp, message)
}

// Close closes the debug log file
func (l *Logger) Close() {
	l.debugMu.Lock()
	defer l.debugMu.Unlock()

	if l.debugFile != nil {
		l.debugFile.Close()
		l.debugFile = nil
	}
}

// DebugLog writes to debug.log file (convenience function)
func DebugLog(format string, args ...interface{}) {
	GetLogger().DebugLog(format, args...)
}
