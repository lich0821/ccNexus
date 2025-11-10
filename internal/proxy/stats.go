package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DailyStats represents statistics for a single day
type DailyStats struct {
	Date         string `json:"date"`         // Format: "2006-01-02"
	Requests     int    `json:"requests"`
	Errors       int    `json:"errors"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}

// EndpointStats represents statistics for a single endpoint
type EndpointStats struct {
	Requests     int                   `json:"requests"`
	Errors       int                   `json:"errors"`
	InputTokens  int                   `json:"inputTokens"`
	OutputTokens int                   `json:"outputTokens"`
	LastUsed     time.Time             `json:"lastUsed"`
	DailyHistory map[string]*DailyStats `json:"dailyHistory,omitempty"` // Key: date string
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
		s.EndpointStats[endpointName] = &EndpointStats{
			DailyHistory: make(map[string]*DailyStats),
		}
	}

	stats := s.EndpointStats[endpointName]
	stats.Requests++
	stats.LastUsed = time.Now()

	// Record to daily stats
	date := time.Now().Format("2006-01-02")
	s.recordDailyRequest(endpointName, date)

	// Auto-save after recording
	go s.saveAsync()
}

// recordDailyRequest records a request to daily history (internal method, no lock)
func (s *Stats) recordDailyRequest(endpointName, date string) {
	stats := s.EndpointStats[endpointName]
	if stats.DailyHistory == nil {
		stats.DailyHistory = make(map[string]*DailyStats)
	}

	if _, exists := stats.DailyHistory[date]; !exists {
		stats.DailyHistory[date] = &DailyStats{Date: date}
	}

	stats.DailyHistory[date].Requests++
}

// RecordError records an error for an endpoint
func (s *Stats) RecordError(endpointName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.EndpointStats[endpointName]; !exists {
		s.EndpointStats[endpointName] = &EndpointStats{
			DailyHistory: make(map[string]*DailyStats),
		}
	}

	s.EndpointStats[endpointName].Errors++

	// Record to daily stats
	date := time.Now().Format("2006-01-02")
	s.recordDailyError(endpointName, date)

	// Auto-save after recording
	go s.saveAsync()
}

// recordDailyError records an error to daily history (internal method, no lock)
func (s *Stats) recordDailyError(endpointName, date string) {
	stats := s.EndpointStats[endpointName]
	if stats.DailyHistory == nil {
		stats.DailyHistory = make(map[string]*DailyStats)
	}

	if _, exists := stats.DailyHistory[date]; !exists {
		stats.DailyHistory[date] = &DailyStats{Date: date}
	}

	stats.DailyHistory[date].Errors++
}

// RecordTokens records token usage for an endpoint
func (s *Stats) RecordTokens(endpointName string, inputTokens, outputTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.EndpointStats[endpointName]; !exists {
		s.EndpointStats[endpointName] = &EndpointStats{
			DailyHistory: make(map[string]*DailyStats),
		}
	}

	stats := s.EndpointStats[endpointName]
	stats.InputTokens += inputTokens
	stats.OutputTokens += outputTokens

	// Record to daily stats
	date := time.Now().Format("2006-01-02")
	s.recordDailyTokens(endpointName, date, inputTokens, outputTokens)

	// Auto-save after recording
	go s.saveAsync()
}

// recordDailyTokens records token usage to daily history (internal method, no lock)
func (s *Stats) recordDailyTokens(endpointName, date string, inputTokens, outputTokens int) {
	stats := s.EndpointStats[endpointName]
	if stats.DailyHistory == nil {
		stats.DailyHistory = make(map[string]*DailyStats)
	}

	if _, exists := stats.DailyHistory[date]; !exists {
		stats.DailyHistory[date] = &DailyStats{Date: date}
	}

	stats.DailyHistory[date].InputTokens += inputTokens
	stats.DailyHistory[date].OutputTokens += outputTokens
}

// GetStats returns a copy of current statistics (thread-safe)
func (s *Stats) GetStats() (int, map[string]*EndpointStats) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deep copy
	statsCopy := make(map[string]*EndpointStats)
	for name, stats := range s.EndpointStats {
		// Deep copy daily history
		dailyCopy := make(map[string]*DailyStats)
		for date, daily := range stats.DailyHistory {
			dailyCopy[date] = &DailyStats{
				Date:         daily.Date,
				Requests:     daily.Requests,
				Errors:       daily.Errors,
				InputTokens:  daily.InputTokens,
				OutputTokens: daily.OutputTokens,
			}
		}

		statsCopy[name] = &EndpointStats{
			Requests:     stats.Requests,
			Errors:       stats.Errors,
			InputTokens:  stats.InputTokens,
			OutputTokens: stats.OutputTokens,
			LastUsed:     stats.LastUsed,
			DailyHistory: dailyCopy,
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

// GetPeriodStats returns aggregated statistics for a time period
func (s *Stats) GetPeriodStats(startDate, endDate string) map[string]*DailyStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*DailyStats)

	for endpointName, stats := range s.EndpointStats {
		aggregated := &DailyStats{
			Date: startDate + " to " + endDate,
		}

		for date, daily := range stats.DailyHistory {
			if date >= startDate && date <= endDate {
				aggregated.Requests += daily.Requests
				aggregated.Errors += daily.Errors
				aggregated.InputTokens += daily.InputTokens
				aggregated.OutputTokens += daily.OutputTokens
			}
		}

		result[endpointName] = aggregated
	}

	return result
}

// GetDailyStats returns statistics for a specific date
func (s *Stats) GetDailyStats(date string) map[string]*DailyStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*DailyStats)

	for endpointName, stats := range s.EndpointStats {
		if daily, exists := stats.DailyHistory[date]; exists {
			result[endpointName] = &DailyStats{
				Date:         daily.Date,
				Requests:     daily.Requests,
				Errors:       daily.Errors,
				InputTokens:  daily.InputTokens,
				OutputTokens: daily.OutputTokens,
			}
		}
	}

	return result
}

// CleanupOldData removes data older than specified days
func (s *Stats) CleanupOldData(daysToKeep int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep).Format("2006-01-02")

	for _, stats := range s.EndpointStats {
		if stats.DailyHistory == nil {
			continue
		}

		for date := range stats.DailyHistory {
			if date < cutoffDate {
				delete(stats.DailyHistory, date)
			}
		}
	}

	// Auto-save after cleanup
	go s.saveAsync()
}
