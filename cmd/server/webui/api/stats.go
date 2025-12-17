package api

import (
	"net/http"
	"time"

	"github.com/lich0821/ccNexus/internal/logger"
)

// handleStatsSummary returns overall statistics
func (h *Handler) handleStatsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	totalRequests, endpointStats := h.proxy.GetStats().GetStats()

	// Calculate totals
	totalErrors := 0
	var totalInputTokens int64 = 0
	var totalOutputTokens int64 = 0

	for _, stats := range endpointStats {
		totalErrors += stats.Errors
		totalInputTokens += int64(stats.InputTokens)
		totalOutputTokens += int64(stats.OutputTokens)
	}

	WriteSuccess(w, map[string]interface{}{
		"TotalRequests":     totalRequests,
		"TotalErrors":       totalErrors,
		"TotalInputTokens":  totalInputTokens,
		"TotalOutputTokens": totalOutputTokens,
		"Endpoints":         endpointStats,
	})
}

// handleStatsDaily returns today's statistics
func (h *Handler) handleStatsDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	today := time.Now().Format("2006-01-02")
	stats, err := h.getStatsForPeriod(today, today)
	if err != nil {
		logger.Error("Failed to get daily stats: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get daily stats")
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"period": "daily",
		"date":   today,
		"stats":  stats,
	})
}

// handleStatsWeekly returns this week's statistics
func (h *Handler) handleStatsWeekly(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	now := time.Now()
	// Get start of week (Monday)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday
	}
	startOfWeek := now.AddDate(0, 0, -(weekday - 1))
	startDate := startOfWeek.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	stats, err := h.getStatsForPeriod(startDate, endDate)
	if err != nil {
		logger.Error("Failed to get weekly stats: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get weekly stats")
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"period":    "weekly",
		"startDate": startDate,
		"endDate":   endDate,
		"stats":     stats,
	})
}

// handleStatsMonthly returns this month's statistics
func (h *Handler) handleStatsMonthly(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startDate := startOfMonth.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	stats, err := h.getStatsForPeriod(startDate, endDate)
	if err != nil {
		logger.Error("Failed to get monthly stats: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get monthly stats")
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"period":    "monthly",
		"startDate": startDate,
		"endDate":   endDate,
		"stats":     stats,
	})
}

// handleStatsTrends returns trend comparison data
func (h *Handler) handleStatsTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	// Get today's stats
	todayStats, err := h.getStatsForPeriod(today, today)
	if err != nil {
		logger.Error("Failed to get today's stats: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get trend stats")
		return
	}

	// Get yesterday's stats
	yesterdayStats, err := h.getStatsForPeriod(yesterday, yesterday)
	if err != nil {
		logger.Error("Failed to get yesterday's stats: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get trend stats")
		return
	}

	// Calculate changes
	trends := map[string]interface{}{
		"todayVsYesterday": map[string]interface{}{
			"requests": map[string]interface{}{
				"today":     todayStats["totalRequests"],
				"yesterday": yesterdayStats["totalRequests"],
				"change":    calculatePercentChange(yesterdayStats["totalRequests"].(int), todayStats["totalRequests"].(int)),
			},
			"errors": map[string]interface{}{
				"today":     todayStats["totalErrors"],
				"yesterday": yesterdayStats["totalErrors"],
				"change":    calculatePercentChange(yesterdayStats["totalErrors"].(int), todayStats["totalErrors"].(int)),
			},
			"inputTokens": map[string]interface{}{
				"today":     todayStats["totalInputTokens"],
				"yesterday": yesterdayStats["totalInputTokens"],
				"change":    calculatePercentChange(int(yesterdayStats["totalInputTokens"].(int64)), int(todayStats["totalInputTokens"].(int64))),
			},
			"outputTokens": map[string]interface{}{
				"today":     todayStats["totalOutputTokens"],
				"yesterday": yesterdayStats["totalOutputTokens"],
				"change":    calculatePercentChange(int(yesterdayStats["totalOutputTokens"].(int64)), int(todayStats["totalOutputTokens"].(int64))),
			},
		},
	}

	WriteSuccess(w, trends)
}

// getStatsForPeriod retrieves statistics for a date range
func (h *Handler) getStatsForPeriod(startDate, endDate string) (map[string]interface{}, error) {
	allStats, err := h.storage.GetAllStats()
	if err != nil {
		return nil, err
	}

	totalRequests := 0
	totalErrors := 0
	var totalInputTokens int64 = 0
	var totalOutputTokens int64 = 0
	endpointStats := make(map[string]interface{})

	for endpointName, stats := range allStats {
		epRequests := 0
		epErrors := 0
		var epInputTokens int64 = 0
		var epOutputTokens int64 = 0

		for _, stat := range stats {
			if stat.Date >= startDate && stat.Date <= endDate {
				epRequests += stat.Requests
				epErrors += stat.Errors
				epInputTokens += int64(stat.InputTokens)
				epOutputTokens += int64(stat.OutputTokens)
			}
		}

		if epRequests > 0 {
			endpointStats[endpointName] = map[string]interface{}{
				"requests":     epRequests,
				"errors":       epErrors,
				"inputTokens":  epInputTokens,
				"outputTokens": epOutputTokens,
			}

			totalRequests += epRequests
			totalErrors += epErrors
			totalInputTokens += epInputTokens
			totalOutputTokens += epOutputTokens
		}
	}

	return map[string]interface{}{
		"totalRequests":     totalRequests,
		"totalErrors":       totalErrors,
		"totalSuccess":      totalRequests - totalErrors,
		"totalInputTokens":  totalInputTokens,
		"totalOutputTokens": totalOutputTokens,
		"endpoints":         endpointStats,
	}, nil
}

// calculatePercentChange calculates the percentage change between two values
func calculatePercentChange(old, new int) float64 {
	if old == 0 {
		if new == 0 {
			return 0
		}
		return 100.0
	}
	return float64(new-old) / float64(old) * 100.0
}
