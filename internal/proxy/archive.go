package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lich0821/ccNexus/internal/logger"
)

// MonthlyArchive represents archived statistics for a single month
type MonthlyArchive struct {
	Month      string                    `json:"month"`      // Format: "YYYY-MM"
	ArchivedAt time.Time                 `json:"archivedAt"` // When this archive was created
	Summary    ArchiveSummary            `json:"summary"`    // Month summary statistics
	Endpoints  map[string]*EndpointStats `json:"endpoints"`  // Endpoint statistics with daily history
}

// ArchiveSummary represents summary statistics for a month
type ArchiveSummary struct {
	TotalRequests     int `json:"totalRequests"`
	TotalErrors       int `json:"totalErrors"`
	TotalInputTokens  int `json:"totalInputTokens"`
	TotalOutputTokens int `json:"totalOutputTokens"`
}

// ArchiveManager manages monthly data archives
type ArchiveManager struct {
	archivePath string
	mu          sync.RWMutex
}

// NewArchiveManager creates a new archive manager
func NewArchiveManager() (*ArchiveManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	archivePath := filepath.Join(homeDir, ".ccNexus", "archives")
	if err := os.MkdirAll(archivePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	return &ArchiveManager{
		archivePath: archivePath,
	}, nil
}

// CheckAndArchive checks if there are complete months to archive and archives them
// Also archives current month data in T+1 mode (without removing data)
func (am *ArchiveManager) CheckAndArchive(stats *Stats) error {
	now := time.Now()
	currentMonth := now.Format("2006-01")

	// Archive complete months (with lock and data removal)
	am.mu.Lock()
	monthsToArchive := am.getMonthsToArchive(stats, currentMonth)

	if len(monthsToArchive) > 0 {
		logger.Info("Found %d month(s) to archive: %v", len(monthsToArchive), monthsToArchive)

		// Archive each month
		for _, month := range monthsToArchive {
			if err := am.archiveMonth(stats, month); err != nil {
				am.mu.Unlock()
				logger.Error("Failed to archive month %s: %v", month, err)
				return fmt.Errorf("failed to archive month %s: %w", month, err)
			}
			logger.Info("Successfully archived month: %s", month)
		}

		// Save stats after removing archived data
		if err := stats.FlushSave(); err != nil {
			am.mu.Unlock()
			logger.Error("Failed to save stats after archiving: %v", err)
			return fmt.Errorf("failed to save stats after archiving: %w", err)
		}
	} else {
		logger.Debug("No complete months to archive")
	}
	am.mu.Unlock()

	// Archive current month data (T+1 mode, without removing data)
	// This is done separately without holding the lock for too long
	if err := am.ArchiveCurrentMonth(stats); err != nil {
		logger.Error("Failed to archive current month: %v", err)
		// Don't return error, just log it to avoid blocking complete month archiving
	}

	return nil
}

// getMonthsToArchive returns a list of months that should be archived
func (am *ArchiveManager) getMonthsToArchive(stats *Stats, currentMonth string) []string {
	stats.mu.RLock()
	defer stats.mu.RUnlock()

	monthsMap := make(map[string]bool)

	// Collect all months from all endpoints
	for _, epStats := range stats.EndpointStats {
		for date := range epStats.DailyHistory {
			// Extract month from date (YYYY-MM-DD -> YYYY-MM)
			if len(date) >= 7 {
				month := date[:7]
				if month < currentMonth {
					monthsMap[month] = true
				}
			}
		}
	}

	// Convert map to sorted slice
	months := make([]string, 0, len(monthsMap))
	for month := range monthsMap {
		months = append(months, month)
	}
	sort.Strings(months)

	return months
}

// archiveMonth archives data for a specific month
func (am *ArchiveManager) archiveMonth(stats *Stats, month string) error {
	// Extract month data
	monthData := am.extractMonthData(stats, month)
	if len(monthData) == 0 {
		return fmt.Errorf("no data found for month %s", month)
	}

	// Calculate summary
	summary := am.calculateSummary(monthData)

	// Create archive
	archive := &MonthlyArchive{
		Month:      month,
		ArchivedAt: time.Now(),
		Summary:    summary,
		Endpoints:  monthData,
	}

	// Save archive file
	archiveFile := filepath.Join(am.archivePath, fmt.Sprintf("%s.json", month))
	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal archive: %w", err)
	}

	if err := os.WriteFile(archiveFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write archive file: %w", err)
	}

	// Remove month data from stats
	if err := am.removeMonthData(stats, month); err != nil {
		return fmt.Errorf("failed to remove month data: %w", err)
	}

	return nil
}

// extractMonthData extracts all data for a specific month from stats
func (am *ArchiveManager) extractMonthData(stats *Stats, month string) map[string]*EndpointStats {
	result := make(map[string]*EndpointStats)

	for epName, epStats := range stats.EndpointStats {
		monthHistory := make(map[string]*DailyStats)
		var requests, errors, inputTokens, outputTokens int

		// Extract daily history for this month
		for date, daily := range epStats.DailyHistory {
			if strings.HasPrefix(date, month) {
				monthHistory[date] = &DailyStats{
					Date:         daily.Date,
					Requests:     daily.Requests,
					Errors:       daily.Errors,
					InputTokens:  daily.InputTokens,
					OutputTokens: daily.OutputTokens,
				}

				// Accumulate totals
				requests += daily.Requests
				errors += daily.Errors
				inputTokens += daily.InputTokens
				outputTokens += daily.OutputTokens
			}
		}

		// Only include endpoints that have data for this month
		if len(monthHistory) > 0 {
			result[epName] = &EndpointStats{
				Requests:     requests,
				Errors:       errors,
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				LastUsed:     epStats.LastUsed,
				DailyHistory: monthHistory,
			}
		}
	}

	return result
}

// removeMonthData removes all data for a specific month from stats
func (am *ArchiveManager) removeMonthData(stats *Stats, month string) error {
	for _, epStats := range stats.EndpointStats {
		for date := range epStats.DailyHistory {
			if strings.HasPrefix(date, month) {
				delete(epStats.DailyHistory, date)
			}
		}
	}

	// Recompute aggregated values after removal
	stats.computeAggregatedValues()

	return nil
}

// calculateSummary calculates summary statistics for month data
func (am *ArchiveManager) calculateSummary(monthData map[string]*EndpointStats) ArchiveSummary {
	var summary ArchiveSummary

	for _, epStats := range monthData {
		summary.TotalRequests += epStats.Requests
		summary.TotalErrors += epStats.Errors
		summary.TotalInputTokens += epStats.InputTokens
		summary.TotalOutputTokens += epStats.OutputTokens
	}

	return summary
}

// LoadArchive loads an archived month's data
func (am *ArchiveManager) LoadArchive(month string) (*MonthlyArchive, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	archiveFile := filepath.Join(am.archivePath, fmt.Sprintf("%s.json", month))
	data, err := os.ReadFile(archiveFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("archive not found for month %s", month)
		}
		return nil, fmt.Errorf("failed to read archive file: %w", err)
	}

	var archive MonthlyArchive
	if err := json.Unmarshal(data, &archive); err != nil {
		return nil, fmt.Errorf("failed to unmarshal archive: %w", err)
	}

	return &archive, nil
}

// ListArchives returns a list of all available archive months
func (am *ArchiveManager) ListArchives() ([]string, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	entries, err := os.ReadDir(am.archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive directory: %w", err)
	}

	var months []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			// Extract month from filename (YYYY-MM.json -> YYYY-MM)
			month := strings.TrimSuffix(name, ".json")
			months = append(months, month)
		}
	}

	// Sort in descending order (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(months)))

	return months, nil
}

// GetArchiveSummary returns summary statistics for an archived month
func (am *ArchiveManager) GetArchiveSummary(month string) (*ArchiveSummary, error) {
	archive, err := am.LoadArchive(month)
	if err != nil {
		return nil, err
	}

	return &archive.Summary, nil
}

// GetArchivePath returns the archive directory path
func (am *ArchiveManager) GetArchivePath() string {
	return am.archivePath
}

// ArchiveCurrentMonth archives current month's data up to yesterday (T+1 mode)
// Unlike archiveMonth, this method does NOT remove data from stats
func (am *ArchiveManager) ArchiveCurrentMonth(stats *Stats) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	currentMonth := now.Format("2006-01")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	// Extract current month data up to yesterday (T+1)
	monthData := am.extractMonthDataUpToDate(stats, currentMonth, yesterday)

	// If no data available, skip archiving
	if len(monthData) == 0 {
		logger.Debug("No current month data to archive for %s", currentMonth)
		return nil
	}

	// Calculate summary
	summary := am.calculateSummary(monthData)

	// Create archive
	archive := &MonthlyArchive{
		Month:      currentMonth,
		ArchivedAt: time.Now(),
		Summary:    summary,
		Endpoints:  monthData,
	}

	// Save archive file
	archiveFile := filepath.Join(am.archivePath, fmt.Sprintf("%s.json", currentMonth))
	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal current month archive: %w", err)
	}

	if err := os.WriteFile(archiveFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write current month archive file: %w", err)
	}

	logger.Info("Successfully archived current month data: %s (up to %s)", currentMonth, yesterday)
	return nil
}

// extractMonthDataUpToDate extracts month data up to a specific date (inclusive)
// Used for T+1 archiving of current month
func (am *ArchiveManager) extractMonthDataUpToDate(stats *Stats, month string, endDate string) map[string]*EndpointStats {
	result := make(map[string]*EndpointStats)

	for epName, epStats := range stats.EndpointStats {
		monthHistory := make(map[string]*DailyStats)
		var requests, errors, inputTokens, outputTokens int

		// Extract daily history for this month up to endDate
		for date, daily := range epStats.DailyHistory {
			if strings.HasPrefix(date, month) && date <= endDate {
				monthHistory[date] = &DailyStats{
					Date:         daily.Date,
					Requests:     daily.Requests,
					Errors:       daily.Errors,
					InputTokens:  daily.InputTokens,
					OutputTokens: daily.OutputTokens,
				}

				// Accumulate totals
				requests += daily.Requests
				errors += daily.Errors
				inputTokens += daily.InputTokens
				outputTokens += daily.OutputTokens
			}
		}

		// Only include endpoints that have data for this month
		if len(monthHistory) > 0 {
			result[epName] = &EndpointStats{
				Requests:     requests,
				Errors:       errors,
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				LastUsed:     epStats.LastUsed,
				DailyHistory: monthHistory,
			}
		}
	}

	return result
}
