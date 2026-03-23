package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
)

// ModelInfo represents a single model information
type ModelInfo struct {
	ID         string `json:"id"`
	Object     string `json:"object"`
	Created    int64  `json:"created"`
	OwnedBy    string `json:"owned_by"`
	EndpointID string `json:"endpoint_id"` // Source endpoint identifier
}

// ModelsCache represents cached models data with TTL
type ModelsCache struct {
	data      []ModelInfo
	updatedAt time.Time
	ttl       time.Duration
	mu        sync.RWMutex
}

// NewModelsCache creates a new models cache
func NewModelsCache(ttlMinutes int) *ModelsCache {
	if ttlMinutes <= 0 {
		ttlMinutes = 30 // Default 30 minutes
	}
	return &ModelsCache{
		data:      []ModelInfo{},
		updatedAt: time.Time{},
		ttl:       time.Duration(ttlMinutes) * time.Minute,
	}
}

// Get returns cached data if valid
func (c *ModelsCache) Get() ([]ModelInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Since(c.updatedAt) > c.ttl {
		return nil, false
	}
	return c.data, true
}

// Set updates cached data
func (c *ModelsCache) Set(data []ModelInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = data
	c.updatedAt = time.Now()
}

// Clear clears the cache
func (c *ModelsCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = []ModelInfo{}
	c.updatedAt = time.Time{}
}

// fetchModelsFromEndpoint fetches models from a specific endpoint
func (p *Proxy) fetchModelsFromEndpoint(ep config.Endpoint) ([]ModelInfo, error) {
	var modelsURL string
	var req *http.Request
	var err error

	switch strings.ToLower(ep.Transformer) {
	case "openai", "openai2":
		// OpenAI compatible endpoints
		baseURL := strings.TrimSuffix(ep.APIUrl, "/")
		if strings.Contains(baseURL, "/v1") {
			modelsURL = baseURL + "/models"
		} else {
			modelsURL = baseURL + "/v1/models"
		}
		req, err = http.NewRequest("GET", modelsURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		// Add authorization header
		if ep.AuthMode == config.AuthModeAPIKey && ep.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+ep.APIKey)
		}

	case "gemini":
		// Google Gemini endpoints
		baseURL := strings.TrimSuffix(ep.APIUrl, "/")
		if strings.Contains(baseURL, "/v1") {
			modelsURL = baseURL + "/models"
		} else {
			modelsURL = baseURL + "/v1beta/models"
		}
		// Add API key as query parameter
		if ep.AuthMode == config.AuthModeAPIKey && ep.APIKey != "" {
			modelsURL = modelsURL + "?key=" + ep.APIKey
		}
		req, err = http.NewRequest("GET", modelsURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

	default:
		// For transformers without /v1/models support (claude, codex)
		return nil, fmt.Errorf("transformer %s does not support /v1/models", ep.Transformer)
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "ccNexus/1.0")

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		Data []struct {
			ID         string `json:"id"`
			Object     string `json:"object"`
			Created    int64  `json:"created"`
			OwnedBy    string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to ModelInfo with endpoint_id
	models := make([]ModelInfo, len(result.Data))
	for i, m := range result.Data {
		models[i] = ModelInfo{
			ID:         m.ID,
			Object:     m.Object,
			Created:    m.Created,
			OwnedBy:    m.OwnedBy,
			EndpointID: ep.Name,
		}
	}

	return models, nil
}

// getDefaultModels returns default models for endpoints that don't support /v1/models
func (p *Proxy) getDefaultModels(ep config.Endpoint) []ModelInfo {
	var modelID string
	var ownedBy string

	switch strings.ToLower(ep.Transformer) {
	case "claude":
		// Claude endpoints
		if ep.Model != "" {
			modelID = ep.Model
		} else {
			modelID = "claude-sonnet-4-20250514" // Default Claude model
		}
		ownedBy = "anthropic"

	case "openai2":
		// Codex endpoints
		if ep.Model != "" {
			modelID = ep.Model
		} else if ep.AuthMode == config.AuthModeCodexTokenPool {
			modelID = "gpt-5-codex" // Default Codex model
		} else {
			modelID = "gpt-4o" // Default OpenAI model
		}
		ownedBy = "openai"

	default:
		// Fallback for any other transformer
		if ep.Model != "" {
			modelID = ep.Model
		} else {
			modelID = "unknown-model"
		}
		ownedBy = strings.ToLower(ep.Transformer)
	}

	return []ModelInfo{
		{
			ID:         modelID,
			Object:     "model",
			Created:    time.Now().Unix(),
			OwnedBy:    ownedBy,
			EndpointID: ep.Name,
		},
	}
}

// handleModels handles GET /v1/models requests
func (p *Proxy) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for refresh parameter
	refresh := r.URL.Query().Get("refresh") == "true"
	refreshEnabled := p.config.ModelsCacheRefreshEnabled

	if refresh && !refreshEnabled {
		http.Error(w, "Refresh is disabled in configuration", http.StatusForbidden)
		return
	}

	// Try to get from cache if not refreshing
	if !refresh {
		if cached, ok := p.modelsCache.Get(); ok {
			p.writeModelsResponse(w, cached)
			return
		}
	}

	// Fetch from endpoints
	endpoints := p.config.GetEndpoints()
	allModels := []ModelInfo{}
	allFailed := true

	for _, ep := range endpoints {
		if !ep.Enabled {
			continue
		}

		var models []ModelInfo
		var err error

		// Try to fetch from endpoint's /v1/models API
		models, err = p.fetchModelsFromEndpoint(ep)
		if err != nil {
		// If fetch fails, use default models for this endpoint
		logger.Debug("Failed to fetch models from %s: %v", ep.Name, err)
			models = p.getDefaultModels(ep)
		} else {
			allFailed = false
		}

		allModels = append(allModels, models...)
	}

	// If all endpoints failed, still return the aggregated default models
	if allFailed {
		logger.Debug("All endpoints failed to fetch models, returning default models")
	}

	// Cache the result
	p.modelsCache.Set(allModels)

	// Write response
	p.writeModelsResponse(w, allModels)
}

// writeModelsResponse writes the models list response
func (p *Proxy) writeModelsResponse(w http.ResponseWriter, models []ModelInfo) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Object string       `json:"object"`
		Data   []ModelInfo  `json:"data"`
	}{
		Object: "list",
		Data:   models,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Debug("Failed to encode models response: %v", err)
	}
}

// refreshModelsCache refreshes the models cache in background
func (p *Proxy) refreshModelsCache() {
	logger.Debug("Refreshing models cache in background")

	endpoints := p.config.GetEndpoints()
	allModels := []ModelInfo{}

	for _, ep := range endpoints {
		if !ep.Enabled {
			continue
		}

		var models []ModelInfo
		var err error

		models, err = p.fetchModelsFromEndpoint(ep)
		if err != nil {
			logger.Debug("Background refresh: failed to fetch models from %s: %v", ep.Name, err)
			models = p.getDefaultModels(ep)
		}

		allModels = append(allModels, models...)
	}

	p.modelsCache.Set(allModels)
	logger.Debug("Models cache refreshed, total models: %d", len(allModels))
}
