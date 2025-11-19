package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lich0821/ccNexus/internal/logger"
)

// DailyStats represents statistics for a single day
type DailyStats struct {
	Date         string `json:"date"` // Format: "2006-01-02"
	Requests     int    `json:"requests"`
	Errors       int    `json:"errors"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}

// EndpointStats represents statistics for a single endpoint
type EndpointStats struct {
	Requests     int                    `json:"requests"`     // Computed from DailyHistory
	Errors       int                    `json:"errors"`       // Computed from DailyHistory
	InputTokens  int                    `json:"inputTokens"`  // Computed from DailyHistory
	OutputTokens int                    `json:"outputTokens"` // Computed from DailyHistory
	LastUsed     time.Time              `json:"lastUsed"`
	DailyHistory map[string]*DailyStats `json:"dailyHistory"` // Key: date string (source of truth)
}

// Stats represents overall proxy statistics
type Stats struct {
	TotalRequests int                       `json:"totalRequests"` // Computed from EndpointStats
	EndpointStats map[string]*EndpointStats `json:"endpointStats"`
	mu            sync.RWMutex
	statsPath     string // Path to stats file

	// Save optimization
	savePending   bool
	saveTimer     *time.Timer
	saveMu        sync.Mutex
	saveDebounce  time.Duration
	lastSaveError error
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{
		EndpointStats: make(map[string]*EndpointStats),
		saveDebounce:  2 * time.Second, // Debounce save operations by 2 seconds
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

	if _, exists := s.EndpointStats[endpointName]; !exists {
		s.EndpointStats[endpointName] = &EndpointStats{
			DailyHistory: make(map[string]*DailyStats),
		}
	}

	stats := s.EndpointStats[endpointName]
	stats.LastUsed = time.Now()

	// Record to daily stats (source of truth)
	date := time.Now().Format("2006-01-02")
	s.recordDailyRequest(endpointName, date)

	// Schedule save with debounce
	s.scheduleSave()
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

	// Record to daily stats (source of truth)
	date := time.Now().Format("2006-01-02")
	s.recordDailyError(endpointName, date)

	// Schedule save with debounce
	s.scheduleSave()
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

	// Record to daily stats (source of truth)
	date := time.Now().Format("2006-01-02")
	s.recordDailyTokens(endpointName, date, inputTokens, outputTokens)

	// Schedule save with debounce
	s.scheduleSave()
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

// scheduleSave schedules a save operation with debounce to avoid frequent writes
func (s *Stats) scheduleSave() {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	// If a save is already pending, reset the timer
	if s.savePending {
		if s.saveTimer != nil {
			s.saveTimer.Stop()
		}
	}

	s.savePending = true
	s.saveTimer = time.AfterFunc(s.saveDebounce, func() {
		s.saveMu.Lock()
		s.savePending = false
		s.saveMu.Unlock()

		if err := s.Save(); err != nil {
			s.saveMu.Lock()
			s.lastSaveError = err
			s.saveMu.Unlock()
			logger.Error("Failed to save stats: %v", err)
		}
	})
}

// GetStats returns a copy of current statistics (thread-safe)
// Computes aggregated values from DailyHistory
func (s *Stats) GetStats() (int, map[string]*EndpointStats) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deep copy and compute aggregated values
	statsCopy := make(map[string]*EndpointStats)
	totalRequests := 0

	for name, stats := range s.EndpointStats {
		// Deep copy daily history
		dailyCopy := make(map[string]*DailyStats)
		var requests, errors, inputTokens, outputTokens int

		for date, daily := range stats.DailyHistory {
			dailyCopy[date] = &DailyStats{
				Date:         daily.Date,
				Requests:     daily.Requests,
				Errors:       daily.Errors,
				InputTokens:  daily.InputTokens,
				OutputTokens: daily.OutputTokens,
			}

			// Compute aggregated values from daily history
			requests += daily.Requests
			errors += daily.Errors
			inputTokens += daily.InputTokens
			outputTokens += daily.OutputTokens
		}

		totalRequests += requests

		statsCopy[name] = &EndpointStats{
			Requests:     requests,
			Errors:       errors,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			LastUsed:     stats.LastUsed,
			DailyHistory: dailyCopy,
		}
	}

	return totalRequests, statsCopy
}

// Reset resets all statistics
func (s *Stats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalRequests = 0
	s.EndpointStats = make(map[string]*EndpointStats)

	// Force immediate save
	go func() {
		if err := s.Save(); err != nil {
			logger.Error("Failed to save stats after reset: %v", err)
		}
	}()
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

	// Compute aggregated values before saving
	s.computeAggregatedValues()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.statsPath, data, 0644)
}

// computeAggregatedValues computes aggregated values from daily history
// Must be called with read lock held
func (s *Stats) computeAggregatedValues() {
	totalRequests := 0

	for _, stats := range s.EndpointStats {
		var requests, errors, inputTokens, outputTokens int

		for _, daily := range stats.DailyHistory {
			requests += daily.Requests
			errors += daily.Errors
			inputTokens += daily.InputTokens
			outputTokens += daily.OutputTokens
		}

		stats.Requests = requests
		stats.Errors = errors
		stats.InputTokens = inputTokens
		stats.OutputTokens = outputTokens

		totalRequests += requests
	}

	s.TotalRequests = totalRequests
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

	s.EndpointStats = loaded.EndpointStats
	if s.EndpointStats == nil {
		s.EndpointStats = make(map[string]*EndpointStats)
	}

	// Recompute aggregated values from daily history to ensure consistency
	s.computeAggregatedValues()

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
// Only returns endpoints that have data in the specified period
func (s *Stats) GetPeriodStats(startDate, endDate string) map[string]*DailyStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*DailyStats)

	for endpointName, stats := range s.EndpointStats {
		aggregated := &DailyStats{
			Date: startDate + " to " + endDate,
		}

		hasData := false
		for date, daily := range stats.DailyHistory {
			if date >= startDate && date <= endDate {
				aggregated.Requests += daily.Requests
				aggregated.Errors += daily.Errors
				aggregated.InputTokens += daily.InputTokens
				aggregated.OutputTokens += daily.OutputTokens
				hasData = true
			}
		}

		// Only include endpoints that have data in this period
		if hasData {
			result[endpointName] = aggregated
		}
	}

	return result
}

// GetDailyStats returns statistics for a specific date
// Only returns endpoints that have data on the specified date
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

// FlushSave forces an immediate save, canceling any pending debounced save
func (s *Stats) FlushSave() error {
	s.saveMu.Lock()
	if s.saveTimer != nil {
		s.saveTimer.Stop()
		s.saveTimer = nil
	}
	s.savePending = false
	s.saveMu.Unlock()

	return s.Save()
}

// GetLastSaveError returns the last save error if any
func (s *Stats) GetLastSaveError() error {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	return s.lastSaveError
}
