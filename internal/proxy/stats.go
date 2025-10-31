package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EndpointStats represents statistics for a single endpoint
type EndpointStats struct {
	Requests     int       `json:"requests"`
	Errors       int       `json:"errors"`
	InputTokens  int       `json:"inputTokens"`
	OutputTokens int       `json:"outputTokens"`
	LastUsed     time.Time `json:"lastUsed"`
}

// Stats represents overall proxy statistics
type Stats struct {
	TotalRequests  int                       `json:"totalRequests"`
	EndpointStats  map[string]*EndpointStats `json:"endpointStats"`
	mu             sync.RWMutex
	statsPath      string // Path to stats file
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{
		EndpointStats: make(map[string]*EndpointStats),
	}
}

// SetStatsPath sets the path for stats persistence
func (s *Stats) SetStatsPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statsPath = path
}

// RecordRequest records a request for an endpoint
func (s *Stats) RecordRequest(endpointName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalRequests++

	if _, exists := s.EndpointStats[endpointName]; !exists {
		s.EndpointStats[endpointName] = &EndpointStats{}
	}

	stats := s.EndpointStats[endpointName]
	stats.Requests++
	stats.LastUsed = time.Now()

	// Auto-save after recording
	go s.saveAsync()
}

// RecordError records an error for an endpoint
func (s *Stats) RecordError(endpointName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.EndpointStats[endpointName]; !exists {
		s.EndpointStats[endpointName] = &EndpointStats{}
	}

	s.EndpointStats[endpointName].Errors++

	// Auto-save after recording
	go s.saveAsync()
}

// RecordTokens records token usage for an endpoint
func (s *Stats) RecordTokens(endpointName string, inputTokens, outputTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.EndpointStats[endpointName]; !exists {
		s.EndpointStats[endpointName] = &EndpointStats{}
	}

	stats := s.EndpointStats[endpointName]
	stats.InputTokens += inputTokens
	stats.OutputTokens += outputTokens

	// Auto-save after recording
	go s.saveAsync()
}

// GetStats returns a copy of current statistics (thread-safe)
func (s *Stats) GetStats() (int, map[string]*EndpointStats) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deep copy
	statsCopy := make(map[string]*EndpointStats)
	for name, stats := range s.EndpointStats {
		statsCopy[name] = &EndpointStats{
			Requests:     stats.Requests,
			Errors:       stats.Errors,
			InputTokens:  stats.InputTokens,
			OutputTokens: stats.OutputTokens,
			LastUsed:     stats.LastUsed,
		}
	}

	return s.TotalRequests, statsCopy
}

// Reset resets all statistics
func (s *Stats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalRequests = 0
	s.EndpointStats = make(map[string]*EndpointStats)

	// Save empty stats
	go s.saveAsync()
}

// Save saves statistics to file
func (s *Stats) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.statsPath == "" {
		return nil
	}

	dir := filepath.Dir(s.statsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.statsPath, data, 0644)
}

// saveAsync saves statistics asynchronously (non-blocking)
func (s *Stats) saveAsync() {
	_ = s.Save()
}

// Load loads statistics from file
func (s *Stats) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.statsPath == "" {
		return nil
	}

	data, err := os.ReadFile(s.statsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var loaded Stats
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	s.TotalRequests = loaded.TotalRequests
	s.EndpointStats = loaded.EndpointStats
	if s.EndpointStats == nil {
		s.EndpointStats = make(map[string]*EndpointStats)
	}

	return nil
}

// GetStatsPath returns the stats file path
func GetStatsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".ccNexus")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "stats.json"), nil
}
