package proxy

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
)

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string
	Data  string
}

// parseSSEResponse parses Server-Sent Events and extracts token usage
func parseSSEResponse(data []byte) (int, int) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var inputTokens, outputTokens int

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event:") {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")

			// Parse the JSON data
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
				continue
			}

			eventType, _ := event["type"].(string)

			// Check if this is a message_start event with usage info
			if eventType == "message_start" {
				if message, ok := event["message"].(map[string]interface{}); ok {
					if usage, ok := message["usage"].(map[string]interface{}); ok {
						// Try to get input_tokens
						if input, ok := usage["input_tokens"].(float64); ok {
							inputTokens = int(input)
						}

						// Also check cache_read_input_tokens
						if cacheRead, ok := usage["cache_read_input_tokens"].(float64); ok && cacheRead > 0 {
							inputTokens += int(cacheRead)
						}

						// Also check cache_creation_input_tokens
						if cacheCreate, ok := usage["cache_creation_input_tokens"].(float64); ok && cacheCreate > 0 {
							inputTokens += int(cacheCreate)
						}

						// Get output_tokens from message_start (usually 0, will be updated in message_delta)
						if output, ok := usage["output_tokens"].(float64); ok {
							outputTokens = int(output)
						}
					}
				}
			}

			// Check for message_delta events (which contain the actual output token count)
			if eventType == "message_delta" {
				// Check event.usage first (common format)
				if usage, ok := event["usage"].(map[string]interface{}); ok {
					if output, ok := usage["output_tokens"].(float64); ok {
						outputTokens = int(output)
					}
				}
				// Also check delta.usage (alternative structure)
				if delta, ok := event["delta"].(map[string]interface{}); ok {
					if usage, ok := delta["usage"].(map[string]interface{}); ok {
						if output, ok := usage["output_tokens"].(float64); ok {
							outputTokens = int(output)
						}
					}
				}
			}
		}
	}

	return inputTokens, outputTokens
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Usage represents token usage information from API response
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// APIResponse represents the structure of API responses to extract usage
type APIResponse struct {
	Usage Usage `json:"usage"`
}

// Proxy represents the proxy server
type Proxy struct {
	config       *config.Config
	stats        *Stats
	currentIndex int
	mu           sync.RWMutex
	server       *http.Server
}

// New creates a new Proxy instance
func New(cfg *config.Config) *Proxy {
	return &Proxy{
		config:       cfg,
		stats:        NewStats(),
		currentIndex: 0,
	}
}

// Start starts the proxy server
func (p *Proxy) Start() error {
	port := p.config.GetPort()

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleProxy)
	mux.HandleFunc("/health", p.handleHealth)
	mux.HandleFunc("/stats", p.handleStats)

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	logger.Info("ccNexus starting on port %d", port)
	logger.Info("Configured %d endpoints", len(p.config.GetEndpoints()))
	logger.Debug("Server listening on http://localhost:%d", port)

	return p.server.ListenAndServe()
}

// Stop stops the proxy server
func (p *Proxy) Stop() error {
	if p.server != nil {
		return p.server.Close()
	}
	return nil
}

// getEnabledEndpoints returns only the enabled endpoints
func (p *Proxy) getEnabledEndpoints() []config.Endpoint {
	allEndpoints := p.config.GetEndpoints()
	enabled := make([]config.Endpoint, 0)
	for _, ep := range allEndpoints {
		if ep.Enabled {
			enabled = append(enabled, ep)
		}
	}
	return enabled
}

// getCurrentEndpoint returns the current endpoint (thread-safe)
func (p *Proxy) getCurrentEndpoint() config.Endpoint {
	p.mu.RLock()
	defer p.mu.RUnlock()

	endpoints := p.getEnabledEndpoints()
	if len(endpoints) == 0 {
		// Return empty endpoint if no enabled endpoints
		return config.Endpoint{}
	}
	// Make sure currentIndex is within bounds
	index := p.currentIndex % len(endpoints)
	return endpoints[index]
}

// rotateEndpoint switches to the next endpoint (thread-safe)
func (p *Proxy) rotateEndpoint() config.Endpoint {
	p.mu.Lock()
	defer p.mu.Unlock()

	endpoints := p.getEnabledEndpoints()
	if len(endpoints) == 0 {
		// Return empty endpoint if no enabled endpoints
		return config.Endpoint{}
	}

	oldIndex := p.currentIndex
	oldEndpoint := endpoints[oldIndex]
	p.currentIndex = (p.currentIndex + 1) % len(endpoints)

	newEndpoint := endpoints[p.currentIndex]
	logger.Info("[SWITCH] %s (#%d) → %s (#%d)",
		oldEndpoint.Name, oldIndex+1, newEndpoint.Name, p.currentIndex+1)
	logger.Debug("Rotating from endpoint %s to %s due to error", oldEndpoint.Name, newEndpoint.Name)

	return newEndpoint
}

// shouldRetry determines if a response should trigger a retry
func shouldRetry(statusCode int) bool {
	// Retry on any non-200 status code
	return statusCode != http.StatusOK
}

// handleProxy handles the main proxy logic
func (p *Proxy) handleProxy(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Received %s request to %s", r.Method, r.URL.Path)

	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	endpoints := p.getEnabledEndpoints()
	if len(endpoints) == 0 {
		logger.Error("No enabled endpoints available")
		http.Error(w, "No enabled endpoints configured", http.StatusServiceUnavailable)
		return
	}

	logger.Debug("Using %d enabled endpoints for request", len(endpoints))
	maxRetries := len(endpoints)

	// Try each endpoint
	for retry := 0; retry < maxRetries; retry++ {
		endpoint := p.getCurrentEndpoint()

		// Check if endpoint is empty (shouldn't happen, but safe check)
		if endpoint.Name == "" {
			logger.Error("Got empty endpoint, no enabled endpoints available")
			http.Error(w, "No enabled endpoints available", http.StatusServiceUnavailable)
			return
		}

		// Record request
		p.stats.RecordRequest(endpoint.Name)

		// Create new request
		targetURL := fmt.Sprintf("https://%s%s", endpoint.APIUrl, r.URL.Path)
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		logger.Debug("[%s] Forwarding request to %s", endpoint.Name, targetURL)

		proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(bodyBytes))
		if err != nil {
			logger.Error("[%s] Failed to create request: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.rotateEndpoint()
			continue
		}

		// Copy headers
		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		// Override with endpoint's API key
		proxyReq.Header.Set("x-api-key", endpoint.APIKey)
		proxyReq.Header.Set("Host", endpoint.APIUrl)

		// Send request
		client := &http.Client{
			Timeout: 120 * time.Second,
		}

		resp, err := client.Do(proxyReq)
		if err != nil {
			logger.Error("[%s] Request failed: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.rotateEndpoint()
			continue
		}

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.Error("[%s] Failed to read response: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.rotateEndpoint()
			continue
		}

		// Handle gzip compressed response
		var finalBody []byte = respBody
		if len(respBody) > 1 && respBody[0] == 0x1f && respBody[1] == 0x8b {
			// This is gzip compressed
			gzReader, err := gzip.NewReader(bytes.NewReader(respBody))
			if err == nil {
				decompressed, err := io.ReadAll(gzReader)
				gzReader.Close()
				if err == nil {
					finalBody = decompressed
				}
			}
		}

		// Check if we should retry
		if shouldRetry(resp.StatusCode) {
			// Try to extract error message from response body
			var errorMsg string
			if len(finalBody) > 0 && len(finalBody) < 1000 {
				// Try to parse as JSON error
				var errResp map[string]interface{}
				if err := json.Unmarshal(finalBody, &errResp); err == nil {
					if errData, ok := errResp["error"]; ok {
						if errMap, ok := errData.(map[string]interface{}); ok {
							if msg, ok := errMap["message"].(string); ok {
								errorMsg = msg
							}
						}
					}
				}
			}

			if errorMsg != "" {
				logger.Warn("[%s] HTTP %d %s: %s", endpoint.Name, resp.StatusCode, http.StatusText(resp.StatusCode), errorMsg)
			} else {
				logger.Warn("[%s] HTTP %d %s", endpoint.Name, resp.StatusCode, http.StatusText(resp.StatusCode))
			}

			p.stats.RecordError(endpoint.Name)
			p.rotateEndpoint()

			// If this is not the last retry, continue to next endpoint
			if retry < maxRetries-1 {
				continue
			}
		}

		// Success - extract token usage and return response
		if resp.StatusCode == http.StatusOK && len(finalBody) > 0 {
			logger.Debug("[%s] Request successful, status: %d", endpoint.Name, resp.StatusCode)

			// Check if this is a streaming response
			isStreaming := resp.Header.Get("Content-Type") == "text/event-stream"

			if isStreaming {
				// Parse Server-Sent Events
				inputTokens, outputTokens := parseSSEResponse(finalBody)

				if inputTokens > 0 || outputTokens > 0 {
					p.stats.RecordTokens(endpoint.Name, inputTokens, outputTokens)
					logger.Debug("[%s] Tokens used - Input: %d, Output: %d", endpoint.Name, inputTokens, outputTokens)
				}
			} else {
				// Standard JSON response
				var apiResp APIResponse
				if err := json.Unmarshal(finalBody, &apiResp); err == nil {
					if apiResp.Usage.InputTokens > 0 || apiResp.Usage.OutputTokens > 0 {
						p.stats.RecordTokens(endpoint.Name, apiResp.Usage.InputTokens, apiResp.Usage.OutputTokens)
						logger.Debug("[%s] Tokens used - Input: %d, Output: %d", endpoint.Name, apiResp.Usage.InputTokens, apiResp.Usage.OutputTokens)
					}
				}
			}
		}

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)

		// Keep endpoint for cache efficiency (only rotate on errors)
		return
	}

	// All endpoints failed
	logger.Error("All %d endpoints failed after retries", maxRetries)
	http.Error(w, "All endpoints unavailable", http.StatusServiceUnavailable)
}

// handleHealth handles health check requests
func (p *Proxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	totalRequests, endpointStats := p.stats.GetStats()
	endpoints := p.config.GetEndpoints()

	response := map[string]interface{}{
		"status":         "ok",
		"totalEndpoints": len(endpoints),
		"currentIndex":   p.currentIndex,
		"stats": map[string]interface{}{
			"totalRequests": totalRequests,
			"endpoints":     endpointStats,
		},
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%v", response)
}

// handleStats handles statistics requests
func (p *Proxy) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	totalRequests, endpointStats := p.stats.GetStats()

	response := map[string]interface{}{
		"totalRequests": totalRequests,
		"endpoints":     endpointStats,
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%v", response)
}

// GetStats returns current statistics
func (p *Proxy) GetStats() *Stats {
	return p.stats
}

// UpdateConfig updates the proxy configuration
func (p *Proxy) UpdateConfig(cfg *config.Config) error {
	// Only validate if there are endpoints
	if len(cfg.GetEndpoints()) > 0 {
		if err := cfg.Validate(); err != nil {
			return err
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = cfg
	p.currentIndex = 0 // Reset to first endpoint

	logger.Debug("Proxy configuration reloaded: %d total endpoints, %d enabled", len(cfg.GetEndpoints()), len(p.getEnabledEndpoints()))
	return nil
}
