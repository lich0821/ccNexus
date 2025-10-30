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
	"github.com/lich0821/ccNexus/internal/transformer"
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

// normalizeAPIUrl ensures the API URL has the correct format
// Removes http:// or https:// prefix if present, as we'll add https:// when making requests
func normalizeAPIUrl(apiUrl string) string {
	// Remove http:// or https:// prefix
	apiUrl = strings.TrimPrefix(apiUrl, "https://")
	apiUrl = strings.TrimPrefix(apiUrl, "http://")
	// Remove trailing slash
	apiUrl = strings.TrimSuffix(apiUrl, "/")
	return apiUrl
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
	stats := NewStats()

	// Set stats path and load existing stats
	statsPath, err := GetStatsPath()
	if err == nil {
		stats.SetStatsPath(statsPath)
		if err := stats.Load(); err != nil {
			// Log error but continue with empty stats
			// Note: We can't use logger here as it may not be initialized yet
		}
	}

	return &Proxy{
		config:       cfg,
		stats:        stats,
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

	return newEndpoint
}

// shouldRetry determines if a response should trigger a retry
func shouldRetry(statusCode int) bool {
	// Retry on any non-200 status code
	return statusCode != http.StatusOK
}

// handleProxy handles the main proxy logic
func (p *Proxy) handleProxy(w http.ResponseWriter, r *http.Request) {
	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body: %v", err)
		logger.DebugLog("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	logger.DebugLog("=== Proxy Request ===")
	logger.DebugLog("Method: %s, Path: %s", r.Method, r.URL.Path)
	logger.DebugLog("Request Body: %s", string(bodyBytes))

	endpoints := p.getEnabledEndpoints()
	if len(endpoints) == 0 {
		logger.Error("No enabled endpoints available")
		logger.DebugLog("No enabled endpoints available")
		http.Error(w, "No enabled endpoints configured", http.StatusServiceUnavailable)
		return
	}

	// Determine max retries based on number of endpoints
	// If only 1 endpoint, allow 1 retry (total 2 attempts)
	// If multiple endpoints, try each one once
	var maxRetries int
	if len(endpoints) == 1 {
		maxRetries = 2 // Try the same endpoint twice
	} else {
		maxRetries = len(endpoints)
	}

	// Try each endpoint
	for retry := 0; retry < maxRetries; retry++ {
		endpoint := p.getCurrentEndpoint()

		// Check if endpoint is empty (shouldn't happen, but safe check)
		if endpoint.Name == "" {
			logger.Error("Got empty endpoint, no enabled endpoints available")
			logger.DebugLog("Got empty endpoint, no enabled endpoints available")
			http.Error(w, "No enabled endpoints available", http.StatusServiceUnavailable)
			return
		}

		// Record request
		p.stats.RecordRequest(endpoint.Name)

		// Get transformer for this endpoint
		transformerName := endpoint.Transformer
		if transformerName == "" {
			transformerName = "claude"
		}

		var trans transformer.Transformer
		var err error

		// For OpenAI and Gemini transformers, create instance with model name
		if transformerName == "openai" {
			if endpoint.Model == "" {
				logger.Error("[%s] OpenAI transformer requires model field", endpoint.Name)
				logger.DebugLog("[%s] OpenAI transformer requires model field", endpoint.Name)
				p.stats.RecordError(endpoint.Name)
				// Only rotate if there are multiple endpoints
				if len(endpoints) > 1 {
					p.rotateEndpoint()
				}
				continue
			}
			trans = transformer.NewOpenAITransformer(endpoint.Model)
		} else if transformerName == "gemini" {
			if endpoint.Model == "" {
				logger.Error("[%s] Gemini transformer requires model field", endpoint.Name)
				logger.DebugLog("[%s] Gemini transformer requires model field", endpoint.Name)
				p.stats.RecordError(endpoint.Name)
				// Only rotate if there are multiple endpoints
				if len(endpoints) > 1 {
					p.rotateEndpoint()
				}
				continue
			}
			trans = transformer.NewGeminiTransformer(endpoint.Model)
		} else if transformerName == "claude" {
			// For Claude transformer, create instance with optional model
			if endpoint.Model != "" {
				trans = transformer.NewClaudeTransformerWithModel(endpoint.Model)
				logger.Debug("[%s] Using Claude transformer with model override: %s", endpoint.Name, endpoint.Model)
			} else {
				trans = transformer.NewClaudeTransformer()
				logger.Debug("[%s] Using Claude transformer with model passthrough", endpoint.Name)
			}
		} else {
			// Get registered transformer for other types
			trans, err = transformer.Get(transformerName)
			if err != nil {
				logger.Error("[%s] Failed to get transformer '%s': %v", endpoint.Name, transformerName, err)
				logger.DebugLog("[%s] Failed to get transformer '%s': %v", endpoint.Name, transformerName, err)
				p.stats.RecordError(endpoint.Name)
				// Only rotate if there are multiple endpoints
				if len(endpoints) > 1 {
					p.rotateEndpoint()
				}
				continue
			}
		}

		// Transform request from Claude format to target API format
		transformedBody, err := trans.TransformRequest(bodyBytes)
		if err != nil {
			logger.Error("[%s] Failed to transform request: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			// Only rotate if there are multiple endpoints
			if len(endpoints) > 1 {
				p.rotateEndpoint()
			}
			continue
		}

		logger.Debug("[%s] Using transformer: %s", endpoint.Name, transformerName)
		logger.DebugLog("[%s] Transformer: %s", endpoint.Name, transformerName)
		logger.DebugLog("[%s] Transformed Request: %s", endpoint.Name, string(transformedBody))

		// Parse the transformed request to check if thinking is enabled
		var thinkingEnabled bool
		if transformerName == "openai" {
			var openaiReq map[string]interface{}
			if err := json.Unmarshal(transformedBody, &openaiReq); err == nil {
				if enableThinking, ok := openaiReq["enable_thinking"].(bool); ok {
					thinkingEnabled = enableThinking
				}
			}
		}

		// Create new request
		targetPath := r.URL.Path
		if transformerName == "openai" && targetPath == "/v1/messages" {
			targetPath = "/v1/chat/completions"
		} else if transformerName == "gemini" && targetPath == "/v1/messages" {
			var geminiReq struct {
				Stream bool `json:"stream"`
			}
			json.Unmarshal(transformedBody, &geminiReq)

			if geminiReq.Stream {
				targetPath = fmt.Sprintf("/v1beta/models/%s:streamGenerateContent", endpoint.Model)
			} else {
				targetPath = fmt.Sprintf("/v1beta/models/%s:generateContent", endpoint.Model)
			}
		}

		// Normalize API URL (remove http/https prefix if present)
		normalizedAPIUrl := normalizeAPIUrl(endpoint.APIUrl)

		targetURL := fmt.Sprintf("https://%s%s", normalizedAPIUrl, targetPath)
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(transformedBody))
		if err != nil {
			logger.Error("[%s] Failed to create request: %v", endpoint.Name, err)
			logger.DebugLog("[%s] Failed to create request: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			// Only rotate if there are multiple endpoints
			if len(endpoints) > 1 {
				p.rotateEndpoint()
			}
			continue
		}

		// Copy headers
		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		// Set authentication header based on transformer type
		if transformerName == "openai" {
			proxyReq.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
		} else if transformerName == "gemini" {
			q := proxyReq.URL.Query()
			q.Set("key", endpoint.APIKey)
			proxyReq.URL.RawQuery = q.Encode()
		} else {
			// Set both x-api-key and Authorization headers for compatibility
			// Some services use x-api-key (e.g., Anthropic Claude), others use Bearer token
			proxyReq.Header.Set("x-api-key", endpoint.APIKey)
			proxyReq.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
		}

		proxyReq.Header.Set("Host", normalizedAPIUrl)
		proxyReq.Header.Set("Content-Type", "application/json")

		// Send request
		client := &http.Client{
			Timeout: 120 * time.Second,
		}

		resp, err := client.Do(proxyReq)
		if err != nil {
			logger.Error("[%s] Request failed: %v", endpoint.Name, err)
			logger.DebugLog("[%s] Request Error: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			// Only rotate if there are multiple endpoints
			if len(endpoints) > 1 {
				p.rotateEndpoint()
			}
			continue
		}

		logger.DebugLog("[%s] Response Status: %d", endpoint.Name, resp.StatusCode)

		// Parse request to check if streaming was requested
		var claudeReq struct {
			Stream bool `json:"stream"`
		}
		json.Unmarshal(bodyBytes, &claudeReq)

		// Check if this is a streaming response
		contentType := resp.Header.Get("Content-Type")
		isStreaming := contentType == "text/event-stream" ||
			(claudeReq.Stream && strings.Contains(contentType, "text/event-stream"))

		// Handle streaming responses differently
		if resp.StatusCode == http.StatusOK && isStreaming {
			// Copy response headers
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)

			// Get flusher
			flusher, ok := w.(http.Flusher)
			if !ok {
				logger.Error("[%s] ResponseWriter does not support flushing", endpoint.Name)
				logger.DebugLog("[%s] ResponseWriter does not support flushing", endpoint.Name)
				resp.Body.Close()
				return
			}

			// Create a stream context for this specific stream
			var streamCtx *transformer.StreamContext
			if transformerName == "openai" || transformerName == "gemini" {
				streamCtx = transformer.NewStreamContext()
				if transformerName == "openai" {
					streamCtx.EnableThinking = thinkingEnabled
				}
			}

			// Stream and transform SSE events in real-time
			scanner := bufio.NewScanner(resp.Body)
			var inputTokens, outputTokens int
			var buffer bytes.Buffer
			eventCount := 0
			streamDone := false

			for scanner.Scan() && !streamDone {
				line := scanner.Text()
				buffer.WriteString(line + "\n")

				// Check for [DONE] marker to stop reading immediately
				if strings.Contains(line, "data: [DONE]") {
					streamDone = true
				}

				// When we hit an empty line, we have a complete event
				if line == "" {
					eventCount++
					// Transform the buffered event
					eventData := buffer.Bytes()

					logger.DebugLog("[%s] SSE Event #%d (Original): %s", endpoint.Name, eventCount, string(eventData))

					var transformedEvent []byte
					var err error

					// Transform based on transformer type
					if transformerName == "openai" {
						transformedEvent, err = trans.(*transformer.OpenAITransformer).TransformResponseWithContext(eventData, true, streamCtx)
					} else if transformerName == "gemini" {
						transformedEvent, err = trans.(*transformer.GeminiTransformer).TransformResponseWithContext(eventData, true, streamCtx)
					} else {
						transformedEvent, err = trans.TransformResponse(eventData, true)
					}

					if err != nil {
						logger.Error("[%s] Failed to transform SSE event #%d: %v", endpoint.Name, eventCount, err)
						logger.Error("[%s] Original event data:\n%s", endpoint.Name, string(eventData))
						logger.DebugLog("[%s] SSE Transform Error #%d: %v", endpoint.Name, eventCount, err)
						buffer.Reset()
						continue
					}

					logger.DebugLog("[%s] SSE Event #%d (Transformed): %s", endpoint.Name, eventCount, string(transformedEvent))

					// Write transformed event
					_, writeErr := w.Write(transformedEvent)
					if writeErr != nil {
						logger.Error("[%s] Failed to write event #%d to client: %v", endpoint.Name, eventCount, writeErr)
						logger.DebugLog("[%s] Write Error #%d: %v", endpoint.Name, eventCount, writeErr)
						streamDone = true
						break
					}
					flusher.Flush()

					// Parse token usage
					scanner2 := bufio.NewScanner(bytes.NewReader(transformedEvent))
					for scanner2.Scan() {
						eventLine := scanner2.Text()
						if strings.HasPrefix(eventLine, "data: ") {
							jsonData := strings.TrimPrefix(eventLine, "data: ")
							var event map[string]interface{}
							if err := json.Unmarshal([]byte(jsonData), &event); err == nil {
								eventType, _ := event["type"].(string)
								if eventType == "message_start" {
									if message, ok := event["message"].(map[string]interface{}); ok {
										if usage, ok := message["usage"].(map[string]interface{}); ok {
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
										}
									}
								}
								if eventType == "message_delta" {
									if usage, ok := event["usage"].(map[string]interface{}); ok {
										if output, ok := usage["output_tokens"].(float64); ok {
											outputTokens = int(output)
										}
									}
									if delta, ok := event["delta"].(map[string]interface{}); ok {
										if stopReason, ok := delta["stop_reason"].(string); ok && stopReason != "" {
											if stopReason != "tool_use" {
												// Don't set streamDone here, wait for message_stop event
											}
										}
									}
								}
								if eventType == "message_stop" {
									streamDone = true
								}
							}
						}
					}

					buffer.Reset()

					if streamDone {
						break
					}
				}
			}

			resp.Body.Close()

			if inputTokens > 0 || outputTokens > 0 {
				p.stats.RecordTokens(endpoint.Name, inputTokens, outputTokens)
			}

			return
		}

		// For non-streaming responses, read the full body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.Error("[%s] Failed to read response: %v", endpoint.Name, err)
			logger.DebugLog("[%s] Failed to read response: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			// Only rotate if there are multiple endpoints
			if len(endpoints) > 1 {
				p.rotateEndpoint()
			}
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
			var errorMsg string
			if len(finalBody) > 0 && len(finalBody) < 1000 {
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

			logger.DebugLog("[%s] Error Response Body: %s", endpoint.Name, string(finalBody))

			if errorMsg != "" {
				logger.Warn("[%s] HTTP %d %s: %s", endpoint.Name, resp.StatusCode, http.StatusText(resp.StatusCode), errorMsg)
			} else {
				logger.Warn("[%s] HTTP %d %s", endpoint.Name, resp.StatusCode, http.StatusText(resp.StatusCode))
			}

			p.stats.RecordError(endpoint.Name)
			// Only rotate if there are multiple endpoints
			if len(endpoints) > 1 {
				p.rotateEndpoint()
			}

			if retry < maxRetries-1 {
				continue
			}
		}

		// Success - handle non-streaming response
		if resp.StatusCode == http.StatusOK && len(finalBody) > 0 {
			logger.DebugLog("[%s] Response Body (Original): %s", endpoint.Name, string(finalBody))

			// Transform response
			transformedResp, err := trans.TransformResponse(finalBody, false)
			if err != nil {
				logger.Error("[%s] Failed to transform response: %v", endpoint.Name, err)
				logger.DebugLog("[%s] Transform Error: %v", endpoint.Name, err)
				p.stats.RecordError(endpoint.Name)
				// Only rotate if there are multiple endpoints
				if len(endpoints) > 1 {
					p.rotateEndpoint()
				}
				continue
			}

			logger.DebugLog("[%s] Response Body (Transformed): %s", endpoint.Name, string(transformedResp))

			// Copy response headers
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}

			w.WriteHeader(resp.StatusCode)
			w.Write(transformedResp)

			// Extract token usage
			var apiResp APIResponse
			if err := json.Unmarshal(transformedResp, &apiResp); err == nil {
				if apiResp.Usage.InputTokens > 0 || apiResp.Usage.OutputTokens > 0 {
					p.stats.RecordTokens(endpoint.Name, apiResp.Usage.InputTokens, apiResp.Usage.OutputTokens)
				}
			}

			return
		}

		// Copy response headers for error responses
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)

		return
	}

	// All endpoints failed
	logger.Error("All %d endpoints failed after retries", maxRetries)
	logger.DebugLog("All %d endpoints failed after retries", maxRetries)
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
	p.currentIndex = 0

	return nil
}
