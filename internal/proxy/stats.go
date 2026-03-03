package proxy

import (
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
	GetPeriodStatsAggregated(startDate, endDate string) (map[string]interface{}, error)
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

	// Save optimization
	savePending   bool
	saveTimer     *time.Timer
	saveMu        sync.Mutex
	saveDebounce  time.Duration
	lastSaveError error

	// Event callback for stats updates (4-period data)
	onStatsUpdated func(endpointName string, endpointPeriods, totalPeriods map[string]interface{})
}

// NewStats creates a new Stats instance
func NewStats(storage StatsStorage, deviceID string) *Stats {
	return &Stats{
		storage:      storage,
		deviceID:     deviceID,
		saveDebounce: 2 * time.Second, // Debounce save operations by 2 seconds
	}
}

// SetOnStatsUpdated sets the callback for stats update events
func (s *Stats) SetOnStatsUpdated(callback func(endpointName string, endpointPeriods, totalPeriods map[string]interface{})) {
	s.onStatsUpdated = callback
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
	// Don't emit event here - wait for RecordTokens which is always called after
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
	} else {
		// Emit stats update event for errors (not followed by RecordTokens)
		s.emitStatsUpdate(endpointName)
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
	} else {
		// Emit stats update event after successful record
		s.emitStatsUpdate(endpointName)
	}
}

// emitStatsUpdate queries current stats for the endpoint and emits an update event
func (s *Stats) emitStatsUpdate(endpointName string) {
	if s.onStatsUpdated == nil {
		return
	}

	today, yesterday, weekStart, monthStart := getPeriodDates()

	// 1. Query 4-period stats for the single endpoint
	dailyStats, _ := s.storage.GetDailyStats(endpointName, today, today)
	yesterdayStats, _ := s.storage.GetDailyStats(endpointName, yesterday, yesterday)
	weeklyStats, _ := s.storage.GetDailyStats(endpointName, weekStart, today)
	monthlyStats, _ := s.storage.GetDailyStats(endpointName, monthStart, today)

	// Aggregate single endpoint data for all 4 periods
	endpointPeriods := map[string]interface{}{
		"daily":     aggregateDailyStats(dailyStats),
		"yesterday": aggregateDailyStats(yesterdayStats),
		"weekly":    aggregateDailyStats(weeklyStats),
		"monthly":   aggregateDailyStats(monthlyStats),
	}

	// 2. Query aggregated stats for all endpoints across 4 periods
	dailyTotals, _ := s.storage.GetPeriodStatsAggregated(today, today)
	yesterdayTotals, _ := s.storage.GetPeriodStatsAggregated(yesterday, yesterday)
	weeklyTotals, _ := s.storage.GetPeriodStatsAggregated(weekStart, today)
	monthlyTotals, _ := s.storage.GetPeriodStatsAggregated(monthStart, today)

	// Aggregate totals for all endpoints
	totalPeriods := map[string]interface{}{
		"daily":     sumAllEndpoints(dailyTotals),
		"yesterday": sumAllEndpoints(yesterdayTotals),
		"weekly":    sumAllEndpoints(weeklyTotals),
		"monthly":   sumAllEndpoints(monthlyTotals),
	}

	// 3. Push event with both single endpoint 4-period and total 4-period data
	s.onStatsUpdated(endpointName, endpointPeriods, totalPeriods)
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
		// Use type assertion instead of reflection for better performance and type safety
		stats := extractStatsData(data)
		if stats != nil {
			result[name] = &EndpointStats{
				Requests:     stats.Requests,
				Errors:       stats.Errors,
				InputTokens:  int(stats.InputTokens),
				OutputTokens: int(stats.OutputTokens),
				LastUsed:     time.Now(),
				DailyHistory: make(map[string]*DailyStats),
			}
		}
	}

	return totalRequests, result
}

// extractStatsData safely extracts stats data using type assertion instead of reflection
func extractStatsData(data interface{}) *StatsData {
	// Try direct type assertion first
	if stats, ok := data.(*StatsData); ok {
		return stats
	}

	// Try interface with matching methods
	type StatsLike interface {
		GetRequests() int
		GetErrors() int
		GetInputTokens() int64
		GetOutputTokens() int64
	}

	if statsLike, ok := data.(StatsLike); ok {
		return &StatsData{
			Requests:     statsLike.GetRequests(),
			Errors:       statsLike.GetErrors(),
			InputTokens:  statsLike.GetInputTokens(),
			OutputTokens: statsLike.GetOutputTokens(),
		}
	}

	// Try struct with matching fields (compatibility layer)
	type StatsStruct struct {
		Requests     int
		Errors       int
		InputTokens  int64
		OutputTokens int64
	}

	// Use type switch for known types
	switch v := data.(type) {
	case StatsStruct:
		return &StatsData{
			Requests:     v.Requests,
			Errors:       v.Errors,
			InputTokens:  v.InputTokens,
			OutputTokens: v.OutputTokens,
		}
	case *StatsStruct:
		if v != nil {
			return &StatsData{
				Requests:     v.Requests,
				Errors:       v.Errors,
				InputTokens:  v.InputTokens,
				OutputTokens: v.OutputTokens,
			}
		}
	}

	// Last resort: use reflection with error handling
	return extractStatsDataUsingReflection(data)
}

// extractStatsDataUsingReflection is a fallback that uses reflection safely
func extractStatsDataUsingReflection(data interface{}) *StatsData {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	// Safely extract fields
	getIntField := func(name string) int {
		field := v.FieldByName(name)
		if !field.IsValid() {
			return 0
		}
		if field.Kind() == reflect.Int || field.Kind() == reflect.Int64 {
			return int(field.Int())
		}
		return 0
	}

	getInt64Field := func(name string) int64 {
		field := v.FieldByName(name)
		if !field.IsValid() {
			return 0
		}
		if field.Kind() == reflect.Int || field.Kind() == reflect.Int64 {
			return field.Int()
		}
		return 0
	}

	return &StatsData{
		Requests:     getIntField("Requests"),
		Errors:       getIntField("Errors"),
		InputTokens:  getInt64Field("InputTokens"),
		OutputTokens: getInt64Field("OutputTokens"),
	}
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


// GetPeriodStats returns aggregated statistics for a time period
func (s *Stats) GetPeriodStats(startDate, endDate string) map[string]*DailyStats {
	// Use single aggregated query instead of N+1 queries
	endpointStats, err := s.storage.GetPeriodStatsAggregated(startDate, endDate)
	if err != nil {
		logger.Error("Failed to get period stats: %v", err)
		return make(map[string]*DailyStats)
	}

	result := make(map[string]*DailyStats)
	for endpointName, statsInterface := range endpointStats {
		// Use type assertion instead of reflection
		stats := extractStatsData(statsInterface)
		if stats != nil {
			result[endpointName] = &DailyStats{
				Date:         startDate + " to " + endDate,
				Requests:     stats.Requests,
				Errors:       stats.Errors,
				InputTokens:  int(stats.InputTokens),
				OutputTokens: int(stats.OutputTokens),
			}
		}
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
			// Use type assertion for the first daily record
			record := extractDailyRecord(dailyRecords[0])
			if record != nil {
				result[endpointName] = record
			}
		}
	}

	return result
}

// extractDailyRecord safely extracts a daily record using type assertion instead of reflection
func extractDailyRecord(record interface{}) *DailyStats {
	// Try direct type assertion
	if daily, ok := record.(*DailyStats); ok {
		return daily
	}

	// Try struct with matching fields
	type DailyRecordLike struct {
		Date         string
		Requests     int
		Errors       int
		InputTokens  int
		OutputTokens int
	}

	switch v := record.(type) {
	case DailyRecordLike:
		return &DailyStats{
			Date:         v.Date,
			Requests:     v.Requests,
			Errors:       v.Errors,
			InputTokens:  v.InputTokens,
			OutputTokens: v.OutputTokens,
		}
	case *DailyRecordLike:
		if v != nil {
			return &DailyStats{
				Date:         v.Date,
				Requests:     v.Requests,
				Errors:       v.Errors,
				InputTokens:  v.InputTokens,
				OutputTokens: v.OutputTokens,
			}
		}
	}

	// Fallback: use reflection safely
	return extractDailyRecordUsingReflection(record)
}

// extractDailyRecordUsingReflection is a fallback that uses reflection safely
func extractDailyRecordUsingReflection(record interface{}) *DailyStats {
	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	getStringField := func(name string) string {
		field := v.FieldByName(name)
		if !field.IsValid() || field.Kind() != reflect.String {
			return ""
		}
		return field.String()
	}

	getIntField := func(name string) int {
		field := v.FieldByName(name)
		if !field.IsValid() {
			return 0
		}
		if field.Kind() == reflect.Int || field.Kind() == reflect.Int64 {
			return int(field.Int())
		}
		return 0
	}

	return &DailyStats{
		Date:         getStringField("Date"),
		Requests:     getIntField("Requests"),
		Errors:       getIntField("Errors"),
		InputTokens:  getIntField("InputTokens"),
		OutputTokens: getIntField("OutputTokens"),
	}
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

// getPeriodDates calculates start dates for today, yesterday, this week, and this month
func getPeriodDates() (today, yesterday, weekStart, monthStart string) {
	now := time.Now()
	today = now.Format("2006-01-02")
	yesterday = now.AddDate(0, 0, -1).Format("2006-01-02")

	// Week starts on Monday (weekday 1)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Adjust Sunday to 7
	}
	weekStart = now.AddDate(0, 0, -(weekday-1)).Format("2006-01-02")

	// First day of current month
	monthStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	return
}

// aggregateDailyStats aggregates daily statistics for a single endpoint across multiple days
func aggregateDailyStats(stats []interface{}) map[string]interface{} {
	var totalRequests, totalErrors int
	var totalInputTokens, totalOutputTokens int64

	for _, stat := range stats {
		if dailyStat := extractDailyRecord(stat); dailyStat != nil {
			totalRequests += dailyStat.Requests
			totalErrors += dailyStat.Errors
			totalInputTokens += int64(dailyStat.InputTokens)
			totalOutputTokens += int64(dailyStat.OutputTokens)
		}
	}

	return map[string]interface{}{
		"requests":     totalRequests,
		"errors":       totalErrors,
		"inputTokens":  totalInputTokens,
		"outputTokens": totalOutputTokens,
	}
}

// sumAllEndpoints aggregates statistics across all endpoints
func sumAllEndpoints(endpointStats map[string]interface{}) map[string]interface{} {
	var totalRequests, totalErrors int
	var totalInputTokens, totalOutputTokens int64

	for _, data := range endpointStats {
		if stats := extractStatsData(data); stats != nil {
			totalRequests += stats.Requests
			totalErrors += stats.Errors
			totalInputTokens += stats.InputTokens
			totalOutputTokens += stats.OutputTokens
		}
	}

	return map[string]interface{}{
		"requests":     totalRequests,
		"errors":       totalErrors,
		"inputTokens":  totalInputTokens,
		"outputTokens": totalOutputTokens,
	}
}
