package service

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/lich0821/ccNexus/internal/logger"
    "github.com/lich0821/ccNexus/internal/storage"
)

// ArchiveService handles archive data operations
type ArchiveService struct {
    storage *storage.SQLiteStorage
}

// NewArchiveService creates a new ArchiveService
func NewArchiveService(s *storage.SQLiteStorage) *ArchiveService {
    return &ArchiveService{storage: s}
}

// ListArchives returns a list of all available archive months
func (a *ArchiveService) ListArchives() string {
    if a.storage == nil {
        result := map[string]interface{}{
            "success":  false,
            "message":  "Storage not initialized",
            "archives": []string{},
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    months, err := a.storage.GetArchiveMonths()
    if err != nil {
        logger.Error("Failed to get archive months: %v", err)
        result := map[string]interface{}{
            "success":  false,
            "message":  fmt.Sprintf("Failed to load archives: %v", err),
            "archives": []string{},
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    result := map[string]interface{}{
        "success":  true,
        "archives": months,
    }
    data, _ := json.Marshal(result)
    return string(data)
}

// GetArchiveData returns archived data for a specific month
func (a *ArchiveService) GetArchiveData(month string) string {
    if a.storage == nil {
        result := map[string]interface{}{
            "success": false,
            "message": "Storage not initialized",
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    archiveData, err := a.storage.GetMonthlyArchiveData(month)
    if err != nil {
        logger.Error("Failed to get archive data for %s: %v", month, err)
        result := map[string]interface{}{
            "success": false,
            "message": fmt.Sprintf("Failed to load archive: %v", err),
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    endpoints := make(map[string]map[string]interface{})
    var totalRequests, totalErrors, totalInputTokens, totalOutputTokens int

    for _, record := range archiveData {
        if endpoints[record.EndpointName] == nil {
            endpoints[record.EndpointName] = map[string]interface{}{
                "dailyHistory": make(map[string]interface{}),
            }
        }

        dailyHistory := endpoints[record.EndpointName]["dailyHistory"].(map[string]interface{})
        dailyHistory[record.Date] = map[string]interface{}{
            "date":         record.Date,
            "requests":     record.Requests,
            "errors":       record.Errors,
            "inputTokens":  record.InputTokens,
            "outputTokens": record.OutputTokens,
        }

        totalRequests += record.Requests
        totalErrors += record.Errors
        totalInputTokens += record.InputTokens
        totalOutputTokens += record.OutputTokens
    }

    summary := map[string]interface{}{
        "totalRequests":     totalRequests,
        "totalErrors":       totalErrors,
        "totalInputTokens":  totalInputTokens,
        "totalOutputTokens": totalOutputTokens,
    }

    archive := map[string]interface{}{
        "endpoints": endpoints,
        "summary":   summary,
    }

    result := map[string]interface{}{
        "success": true,
        "archive": archive,
    }

    data, _ := json.Marshal(result)
    return string(data)
}

// GetArchiveTrend returns trend comparison between selected month and previous month
func (a *ArchiveService) GetArchiveTrend(month string) string {
    if a.storage == nil {
        result := map[string]interface{}{
            "success": false,
            "message": "Storage not initialized",
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    t, err := time.Parse("2006-01", month)
    if err != nil {
        result := map[string]interface{}{
            "success": false,
            "message": fmt.Sprintf("Invalid month format: %v", err),
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    previousMonth := t.AddDate(0, -1, 0).Format("2006-01")

    currentData, err := a.storage.GetMonthlyArchiveData(month)
    if err != nil {
        logger.Error("Failed to get current month data: %v", err)
        result := map[string]interface{}{
            "success": false,
            "message": fmt.Sprintf("Failed to load current month: %v", err),
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    previousData, err := a.storage.GetMonthlyArchiveData(previousMonth)
    if err != nil {
        logger.Debug("Previous month %s has no data, returning flat trend", previousMonth)
        result := map[string]interface{}{
            "success":     true,
            "trend":       0.0,
            "errorsTrend": 0.0,
            "tokensTrend": 0.0,
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    var currentRequests, currentErrors, currentTokens int
    for _, record := range currentData {
        currentRequests += record.Requests
        currentErrors += record.Errors
        currentTokens += record.InputTokens + record.OutputTokens
    }

    var previousRequests, previousErrors, previousTokens int
    for _, record := range previousData {
        previousRequests += record.Requests
        previousErrors += record.Errors
        previousTokens += record.InputTokens + record.OutputTokens
    }

    requestsTrend := calculateTrend(currentRequests, previousRequests)
    errorsTrend := calculateTrend(currentErrors, previousErrors)
    tokensTrend := calculateTrend(currentTokens, previousTokens)

    result := map[string]interface{}{
        "success":     true,
        "trend":       requestsTrend,
        "errorsTrend": errorsTrend,
        "tokensTrend": tokensTrend,
    }

    data, _ := json.Marshal(result)
    return string(data)
}

// GenerateMockArchives is deprecated - returns error message
func (a *ArchiveService) GenerateMockArchives(monthsCount int) string {
    result := map[string]interface{}{
        "success": false,
        "message": "Mock archives are no longer supported. Use real data from SQLite.",
    }
    data, _ := json.Marshal(result)
    return string(data)
}

// DeleteArchive deletes all data for a specific month
func (a *ArchiveService) DeleteArchive(month string) string {
    if a.storage == nil {
        result := map[string]interface{}{
            "success": false,
            "message": "Storage not initialized",
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    err := a.storage.DeleteMonthlyStats(month)
    if err != nil {
        logger.Error("Failed to delete archive for %s: %v", month, err)
        result := map[string]interface{}{
            "success": false,
            "message": fmt.Sprintf("Failed to delete archive: %v", err),
        }
        data, _ := json.Marshal(result)
        return string(data)
    }

    logger.Info("Archive deleted for month: %s", month)
    result := map[string]interface{}{
        "success": true,
        "message": fmt.Sprintf("Archive for %s deleted successfully", month),
    }
    data, _ := json.Marshal(result)
    return string(data)
}
