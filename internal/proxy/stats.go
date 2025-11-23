package proxy

import (
	"os"
	"path/filepath"
	"reflect"
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

// StatsStorage defines the interface for stats persistence
type StatsStorage interface {
	RecordDailyStat(stat interface{}) error
	GetTotalStats() (int, map[string]interface{}, error)
	GetDailyStats(endpointName, startDate, endDate string) ([]interface{}, error)
}

// StatRecord represents a stat record for storage
type StatRecord struct {
	EndpointName string
	Date         string
	Requests     int
	Errors       int
	InputTokens  int
	OutputTokens int
	DeviceID     string
}

// StatsData represents aggregated stats data
type StatsData struct {
	Requests     int
	Errors       int
	InputTokens  int64
	OutputTokens int64
}

// DailyRecord represents daily stats
type DailyRecord struct {
	Date         string
	Requests     int
	Errors       int
	InputTokens  int
	OutputTokens int
}

// Stats represents overall proxy statistics
type Stats struct {
	storage       StatsStorage
	deviceID      string
	mu            sync.RWMutex
	statsPath     string // Path to stats file (for backward compatibility)

	// Save optimization
	savePending   bool
	saveTimer     *time.Timer
	saveMu        sync.Mutex
	saveDebounce  time.Duration
	lastSaveError error
}

// NewStats creates a new Stats instance
func NewStats(storage StatsStorage, deviceID string) *Stats {
	return &Stats{
		storage:      storage,
		deviceID:     deviceID,
		saveDebounce: 2 * time.Second, // Debounce save operations by 2 seconds
	}
}

// SetStatsPath sets the path for stats persistence (for backward compatibility)
func (s *Stats) SetStatsPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statsPath = path
}

// RecordRequest records a request for an endpoint
func (s *Stats) RecordRequest(endpointName string) {
	date := time.Now().Format("2006-01-02")

	stat := &StatRecord{
		EndpointName: endpointName,
		Date:         date,
		Requests:     1,
		Errors:       0,
		InputTokens:  0,
		OutputTokens: 0,
		DeviceID:     s.deviceID,
	}

	if err := s.storage.RecordDailyStat(stat); err != nil {
		logger.Error("Failed to record request: %v", err)
	}
}

// RecordError records an error for an endpoint
func (s *Stats) RecordError(endpointName string) {
	date := time.Now().Format("2006-01-02")

	stat := &StatRecord{
		EndpointName: endpointName,
		Date:         date,
		Requests:     0,
		Errors:       1,
		InputTokens:  0,
		OutputTokens: 0,
		DeviceID:     s.deviceID,
	}

	if err := s.storage.RecordDailyStat(stat); err != nil {
		logger.Error("Failed to record error: %v", err)
	}
}

// RecordTokens records token usage for an endpoint
func (s *Stats) RecordTokens(endpointName string, inputTokens, outputTokens int) {
	date := time.Now().Format("2006-01-02")

	stat := &StatRecord{
		EndpointName: endpointName,
		Date:         date,
		Requests:     0,
		Errors:       0,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		DeviceID:     s.deviceID,
	}

	if err := s.storage.RecordDailyStat(stat); err != nil {
		logger.Error("Failed to record tokens: %v", err)
	}
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
func (s *Stats) GetStats() (int, map[string]*EndpointStats) {
	totalRequests, statsData, err := s.storage.GetTotalStats()
	if err != nil {
		logger.Error("Failed to get stats: %v", err)
		return 0, make(map[string]*EndpointStats)
	}

	// Convert to EndpointStats format
	result := make(map[string]*EndpointStats)
	for name, data := range statsData {
		// Type assert to get the actual data using reflection
		v := reflect.ValueOf(data)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		result[name] = &EndpointStats{
			Requests:     int(v.FieldByName("Requests").Int()),
			Errors:       int(v.FieldByName("Errors").Int()),
			InputTokens:  int(v.FieldByName("InputTokens").Int()),
			OutputTokens: int(v.FieldByName("OutputTokens").Int()),
			LastUsed:     time.Now(),
			DailyHistory: make(map[string]*DailyStats),
		}
	}

	return totalRequests, result
}

// Reset resets all statistics
func (s *Stats) Reset() {
	// Note: With SQLite storage, we don't reset the database
	// This would require deleting all records, which we don't want to do
	logger.Warn("Reset is not supported with SQLite storage")
}

// Save saves statistics to file (for backward compatibility, does nothing with SQLite)
func (s *Stats) Save() error {
	// With SQLite, stats are saved immediately on record
	return nil
}

// Load loads statistics from file (for backward compatibility, does nothing with SQLite)
func (s *Stats) Load() error {
	// With SQLite, stats are loaded on demand from storage
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
	// Get all endpoints from storage
	totalRequests, statsData, err := s.storage.GetTotalStats()
	if err != nil {
		logger.Error("Failed to get stats: %v", err)
		return make(map[string]*DailyStats)
	}

	_ = totalRequests // unused
	result := make(map[string]*DailyStats)

	// For each endpoint, get daily stats in the period
	for endpointName := range statsData {
		dailyRecords, err := s.storage.GetDailyStats(endpointName, startDate, endDate)
		if err != nil {
			logger.Error("Failed to get daily stats for %s: %v", endpointName, err)
			continue
		}

		if len(dailyRecords) == 0 {
			continue
		}

		// Aggregate the period
		aggregated := &DailyStats{
			Date: startDate + " to " + endDate,
		}

		for _, dailyInterface := range dailyRecords {
			// Use reflection to extract fields
			v := reflect.ValueOf(dailyInterface)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}

			aggregated.Requests += int(v.FieldByName("Requests").Int())
			aggregated.Errors += int(v.FieldByName("Errors").Int())
			aggregated.InputTokens += int(v.FieldByName("InputTokens").Int())
			aggregated.OutputTokens += int(v.FieldByName("OutputTokens").Int())
		}

		result[endpointName] = aggregated
	}

	return result
}

// GetDailyStats returns statistics for a specific date
func (s *Stats) GetDailyStats(date string) map[string]*DailyStats {
	// Get all endpoints from storage
	totalRequests, statsData, err := s.storage.GetTotalStats()
	if err != nil {
		logger.Error("Failed to get stats: %v", err)
		return make(map[string]*DailyStats)
	}

	_ = totalRequests // unused
	result := make(map[string]*DailyStats)

	// For each endpoint, get stats for the specific date
	for endpointName := range statsData {
		dailyRecords, err := s.storage.GetDailyStats(endpointName, date, date)
		if err != nil {
			logger.Error("Failed to get daily stats for %s: %v", endpointName, err)
			continue
		}

		if len(dailyRecords) > 0 {
			// Use reflection to extract fields
			v := reflect.ValueOf(dailyRecords[0])
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}

			result[endpointName] = &DailyStats{
				Date:         v.FieldByName("Date").String(),
				Requests:     int(v.FieldByName("Requests").Int()),
				Errors:       int(v.FieldByName("Errors").Int()),
				InputTokens:  int(v.FieldByName("InputTokens").Int()),
				OutputTokens: int(v.FieldByName("OutputTokens").Int()),
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
