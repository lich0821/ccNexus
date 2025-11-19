package proxy

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// GenerateMockArchives generates mock archive data for testing
func GenerateMockArchives(archivePath string, monthsCount int) error {
	if err := os.MkdirAll(archivePath, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	now := time.Now()

	// Generate archives for the past N months
	for i := monthsCount; i > 0; i-- {
		monthDate := now.AddDate(0, -i, 0)
		month := monthDate.Format("2006-01")

		archive := generateMockArchive(month, monthDate)

		// Save archive file
		archiveFile := filepath.Join(archivePath, fmt.Sprintf("%s.json", month))
		data, err := json.MarshalIndent(archive, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal archive: %w", err)
		}

		if err := os.WriteFile(archiveFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write archive file: %w", err)
		}
	}

	return nil
}

// generateMockArchive generates mock data for a single month
func generateMockArchive(month string, monthDate time.Time) *MonthlyArchive {
	// Get the number of days in this month
	year, m, _ := monthDate.Date()
	firstDay := time.Date(year, m, 1, 0, 0, 0, 0, monthDate.Location())
	lastDay := firstDay.AddDate(0, 1, -1)
	daysInMonth := lastDay.Day()

	// Create mock endpoints
	endpoints := map[string]*EndpointStats{
		"Claude Official": generateMockEndpointData(month, daysInMonth, 1.0),
		"OpenAI Proxy":    generateMockEndpointData(month, daysInMonth, 0.8),
		"Gemini API":      generateMockEndpointData(month, daysInMonth, 0.6),
	}

	// Calculate summary
	summary := ArchiveSummary{}
	for _, ep := range endpoints {
		summary.TotalRequests += ep.Requests
		summary.TotalErrors += ep.Errors
		summary.TotalInputTokens += ep.InputTokens
		summary.TotalOutputTokens += ep.OutputTokens
	}

	return &MonthlyArchive{
		Month:      month,
		ArchivedAt: time.Now(),
		Summary:    summary,
		Endpoints:  endpoints,
	}
}

// generateMockEndpointData generates mock daily data for an endpoint
func generateMockEndpointData(month string, daysInMonth int, activityFactor float64) *EndpointStats {
	dailyHistory := make(map[string]*DailyStats)

	var totalRequests, totalErrors, totalInputTokens, totalOutputTokens int

	for day := 1; day <= daysInMonth; day++ {
		date := fmt.Sprintf("%s-%02d", month, day)

		// Generate random but realistic data
		baseRequests := int(float64(rand.Intn(100)+50) * activityFactor)
		requests := baseRequests + rand.Intn(20) - 10 // Add some variance
		if requests < 0 {
			requests = 0
		}

		// Error rate around 2-5%
		errorRate := 0.02 + rand.Float64()*0.03
		errors := int(float64(requests) * errorRate)

		// Token usage: 500-2000 input, 300-1500 output per request
		inputTokensPerReq := 500 + rand.Intn(1500)
		outputTokensPerReq := 300 + rand.Intn(1200)
		inputTokens := requests * inputTokensPerReq
		outputTokens := requests * outputTokensPerReq

		dailyHistory[date] = &DailyStats{
			Date:         date,
			Requests:     requests,
			Errors:       errors,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		}

		totalRequests += requests
		totalErrors += errors
		totalInputTokens += inputTokens
		totalOutputTokens += outputTokens
	}

	return &EndpointStats{
		Requests:     totalRequests,
		Errors:       totalErrors,
		InputTokens:  totalInputTokens,
		OutputTokens: totalOutputTokens,
		LastUsed:     time.Now(),
		DailyHistory: dailyHistory,
	}
}

// GenerateMockArchivesForUser generates mock archives in the user's archive directory
func GenerateMockArchivesForUser(monthsCount int) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	archivePath := filepath.Join(homeDir, ".ccNexus", "archives")
	return GenerateMockArchives(archivePath, monthsCount)
}
