package storage

import "reflect"

// StatsStorageAdapter adapts SQLiteStorage to be used by proxy.Stats
// It implements the proxy.StatsStorage interface
type StatsStorageAdapter struct {
	storage *SQLiteStorage
}

// NewStatsStorageAdapter creates a new adapter
func NewStatsStorageAdapter(storage *SQLiteStorage) *StatsStorageAdapter {
	return &StatsStorageAdapter{storage: storage}
}

// RecordDailyStat records a daily stat
func (a *StatsStorageAdapter) RecordDailyStat(stat interface{}) error {
	// Use reflection to extract fields from the stat record
	v := reflect.ValueOf(stat)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	dailyStat := &DailyStat{
		EndpointName: v.FieldByName("EndpointName").String(),
		Date:         v.FieldByName("Date").String(),
		Requests:     int(v.FieldByName("Requests").Int()),
		Errors:       int(v.FieldByName("Errors").Int()),
		InputTokens:  int(v.FieldByName("InputTokens").Int()),
		OutputTokens: int(v.FieldByName("OutputTokens").Int()),
		DeviceID:     v.FieldByName("DeviceID").String(),
	}
	return a.storage.RecordDailyStat(dailyStat)
}

// GetTotalStats gets total stats for all endpoints
func (a *StatsStorageAdapter) GetTotalStats() (int, map[string]interface{}, error) {
	totalRequests, endpointStats, err := a.storage.GetTotalStats()
	if err != nil {
		return 0, nil, err
	}

	result := make(map[string]interface{})
	for name, stats := range endpointStats {
		result[name] = &StatsDataCompat{
			Requests:     stats.Requests,
			Errors:       stats.Errors,
			InputTokens:  stats.InputTokens,
			OutputTokens: stats.OutputTokens,
		}
	}

	return totalRequests, result, nil
}

// StatsDataCompat is a compatible stats data structure
type StatsDataCompat struct {
	Requests     int
	Errors       int
	InputTokens  int64
	OutputTokens int64
}

// GetDailyStats gets daily stats for an endpoint
func (a *StatsStorageAdapter) GetDailyStats(endpointName, startDate, endDate string) ([]interface{}, error) {
	dailyStats, err := a.storage.GetDailyStats(endpointName, startDate, endDate)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(dailyStats))
	for i, stat := range dailyStats {
		result[i] = &DailyRecordCompat{
			Date:         stat.Date,
			Requests:     stat.Requests,
			Errors:       stat.Errors,
			InputTokens:  stat.InputTokens,
			OutputTokens: stat.OutputTokens,
		}
	}

	return result, nil
}

// DailyRecordCompat is a compatible daily record structure
type DailyRecordCompat struct {
	Date         string
	Requests     int
	Errors       int
	InputTokens  int
	OutputTokens int
}
