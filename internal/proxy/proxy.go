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
	"github.com/lich0821/ccNexus/internal/tokencount"
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
// If URL has http:// or https:// prefix, keep it; otherwise default to https://
func normalizeAPIUrl(apiUrl string) string {
	apiUrl = strings.TrimSuffix(apiUrl, "/")
	if !strings.HasPrefix(apiUrl, "http://") && !strings.HasPrefix(apiUrl, "https://") {
		return "https://" + apiUrl
	}
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
	config           *config.Config
	stats            *Stats
	currentIndex     int
	mu               sync.RWMutex
	server           *http.Server
	activeRequests   map[string]bool // tracks active requests by endpoint name
	activeRequestsMu sync.RWMutex    // protects activeRequests map
}

// New creates a new Proxy instance
func New(cfg *config.Config, statsStorage StatsStorage, deviceID string) *Proxy {
	stats := NewStats(statsStorage, deviceID)

	// Set stats path for backward compatibility
	statsPath, err := GetStatsPath()
	if err == nil {
		stats.SetStatsPath(statsPath)
	}

	return &Proxy{
		config:         cfg,
		stats:          stats,
		currentIndex:   0,
		activeRequests: make(map[string]bool),
	}
}

// Start starts the proxy server
func (p *Proxy) Start() error {
	port := p.config.GetPort()

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleProxy)
	mux.HandleFunc("/v1/messages/count_tokens", p.handleCountTokens)
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

// markRequestActive marks an endpoint as having active requests
func (p *Proxy) markRequestActive(endpointName string) {
	p.activeRequestsMu.Lock()
	defer p.activeRequestsMu.Unlock()
	p.activeRequests[endpointName] = true
}

// markRequestInactive marks an endpoint as having no active requests
func (p *Proxy) markRequestInactive(endpointName string) {
	p.activeRequestsMu.Lock()
	defer p.activeRequestsMu.Unlock()
	delete(p.activeRequests, endpointName)
}

// hasActiveRequests checks if an endpoint has active requests
func (p *Proxy) hasActiveRequests(endpointName string) bool {
	p.activeRequestsMu.RLock()
	defer p.activeRequestsMu.RUnlock()
	return p.activeRequests[endpointName]
}

// isCurrentEndpoint checks if the given endpoint is still the current one
func (p *Proxy) isCurrentEndpoint(endpointName string) bool {
	current := p.getCurrentEndpoint()
	return current.Name == endpointName
}

// rotateEndpoint switches to the next endpoint (thread-safe)
// waitForActive: if true, waits briefly for active requests to complete before switching
func (p *Proxy) rotateEndpoint() config.Endpoint {
	p.mu.Lock()
	defer p.mu.Unlock()

	endpoints := p.getEnabledEndpoints()
	if len(endpoints) == 0 {
		// Return empty endpoint if no enabled endpoints
		return config.Endpoint{}
	}

	oldIndex := p.currentIndex
	oldEndpoint := endpoints[oldIndex%len(endpoints)]

	// Check if there are active requests on the current endpoint
	// Wait a short time for them to complete (max 500ms)
	if p.hasActiveRequests(oldEndpoint.Name) {
		logger.Debug("[SWITCH] Waiting for active requests on %s to complete...", oldEndpoint.Name)
		p.mu.Unlock() // Release lock while waiting

		for i := 0; i < 10; i++ { // Check 10 times, 50ms each = 500ms max
			time.Sleep(50 * time.Millisecond)
			if !p.hasActiveRequests(oldEndpoint.Name) {
				break
			}
		}

		p.mu.Lock() // Re-acquire lock
		if p.hasActiveRequests(oldEndpoint.Name) {
			logger.Warn("[SWITCH] Active requests still present on %s after waiting, forcing switch", oldEndpoint.Name)
		}
	}

	p.currentIndex = (p.currentIndex + 1) % len(endpoints)

	newEndpoint := endpoints[p.currentIndex]
	logger.Debug("[SWITCH] %s (#%d) → %s (#%d)",
		oldEndpoint.Name, oldIndex+1, newEndpoint.Name, p.currentIndex+1)

	return newEndpoint
}

// GetCurrentEndpointName returns the current endpoint name (thread-safe)
func (p *Proxy) GetCurrentEndpointName() string {
	endpoint := p.getCurrentEndpoint()
	return endpoint.Name
}

// SetCurrentEndpoint manually switches to a specific endpoint by name
// Returns error if endpoint not found or not enabled
// Thread-safe and won't affect ongoing requests
func (p *Proxy) SetCurrentEndpoint(targetName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	endpoints := p.getEnabledEndpoints()
	if len(endpoints) == 0 {
		return fmt.Errorf("no enabled endpoints")
	}

	// Find the endpoint by name
	for i, ep := range endpoints {
		if ep.Name == targetName {
			oldEndpoint := endpoints[p.currentIndex%len(endpoints)]
			p.currentIndex = i
			logger.Info("[MANUAL SWITCH] %s → %s", oldEndpoint.Name, ep.Name)
			return nil
		}
	}

	return fmt.Errorf("endpoint '%s' not found or not enabled", targetName)
}

// shouldRetry determines if a response should trigger a retry
func shouldRetry(statusCode int) bool {
	// Retry on any non-200 status code
	return statusCode != http.StatusOK
}

// cleanIncompleteToolCalls removes incomplete tool_use/tool_result pairs from messages
// This ensures compatibility when switching between different API endpoints
func cleanIncompleteToolCalls(bodyBytes []byte) ([]byte, error) {
	var req map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		// If we can't parse it, return original
		return bodyBytes, nil
	}

	messages, ok := req["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		return bodyBytes, nil
	}

	// Track which tool_use IDs have matching tool_result
	toolUseIDs := make(map[string]bool)
	toolResultIDs := make(map[string]bool)

	// First pass: collect all tool_use and tool_result IDs
	for _, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}

		role, _ := msgMap["role"].(string)
		content := msgMap["content"]

		// Handle content as array
		if contentArray, ok := content.([]interface{}); ok {
			for _, block := range contentArray {
				if blockMap, ok := block.(map[string]interface{}); ok {
					blockType, _ := blockMap["type"].(string)

					if blockType == "tool_use" && role == "assistant" {
						if id, ok := blockMap["id"].(string); ok {
							toolUseIDs[id] = true
						}
					} else if blockType == "tool_result" && role == "user" {
						if toolUseID, ok := blockMap["tool_use_id"].(string); ok {
							toolResultIDs[toolUseID] = true
						}
					}
				}
			}
		}
	}

	// Find incomplete tool_use IDs (those without matching tool_result)
	incompleteToolUseIDs := make(map[string]bool)
	for id := range toolUseIDs {
		if !toolResultIDs[id] {
			incompleteToolUseIDs[id] = true
		}
	}

	// Find orphaned tool_result IDs (those without matching tool_use)
	orphanedToolResultIDs := make(map[string]bool)
	for id := range toolResultIDs {
		if !toolUseIDs[id] {
			orphanedToolResultIDs[id] = true
		}
	}

	// If no incomplete pairs, return original
	if len(incompleteToolUseIDs) == 0 && len(orphanedToolResultIDs) == 0 {
		return bodyBytes, nil
	}

	if len(incompleteToolUseIDs) > 0 {
		logger.Debug("Found %d incomplete tool_use blocks, cleaning up", len(incompleteToolUseIDs))
	}
	if len(orphanedToolResultIDs) > 0 {
		logger.Debug("Found %d orphaned tool_result blocks, cleaning up", len(orphanedToolResultIDs))
	}

	// Second pass: clean up messages
	cleanedMessages := make([]interface{}, 0, len(messages))
	for _, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			cleanedMessages = append(cleanedMessages, msg)
			continue
		}

		role, _ := msgMap["role"].(string)
		content := msgMap["content"]

		// Handle array content for both assistant and user messages
		contentArray, ok := content.([]interface{})
		if !ok {
			cleanedMessages = append(cleanedMessages, msg)
			continue
		}

		// Filter out incomplete tool_use blocks (from assistant) and orphaned tool_result blocks (from user)
		cleanedContent := make([]interface{}, 0)
		hasContent := false

		for _, block := range contentArray {
			blockMap, ok := block.(map[string]interface{})
			if !ok {
				cleanedContent = append(cleanedContent, block)
				hasContent = true
				continue
			}

			blockType, _ := blockMap["type"].(string)

			// Skip incomplete tool_use blocks from assistant messages
			if blockType == "tool_use" && role == "assistant" {
				if id, ok := blockMap["id"].(string); ok {
					if incompleteToolUseIDs[id] {
						logger.Debug("Removing incomplete tool_use block: %s", id)
						continue
					}
				}
			}

			// Skip orphaned tool_result blocks from user messages
			if blockType == "tool_result" && role == "user" {
				if toolUseID, ok := blockMap["tool_use_id"].(string); ok {
					if orphanedToolResultIDs[toolUseID] {
						logger.Debug("Removing orphaned tool_result block: %s", toolUseID)
						continue
					}
				}
			}

			cleanedContent = append(cleanedContent, block)
			hasContent = true
		}

		// Only add message if it has content
		if hasContent {
			msgMap["content"] = cleanedContent
			cleanedMessages = append(cleanedMessages, msgMap)
		} else {
			if role == "assistant" {
				logger.Debug("Removing assistant message with only incomplete tool_use blocks")
			} else if role == "user" {
				logger.Debug("Removing user message with only orphaned tool_result blocks")
			}
		}
	}

	req["messages"] = cleanedMessages
	return json.Marshal(req)
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
		http.Error(w, "No enabled endpoints configured", http.StatusServiceUnavailable)
		return
	}

	// Determine max retries: always try each endpoint twice before moving to next
	// Total attempts = number of endpoints * 2 (each endpoint gets 2 chances)
	maxRetries := len(endpoints) * 2
	endpointAttempts := 0 // Track attempts for current endpoint

	// Try each endpoint
	for retry := 0; retry < maxRetries; retry++ {
		endpoint := p.getCurrentEndpoint()

		// Check if endpoint is empty (shouldn't happen, but safe check)
		if endpoint.Name == "" {
			logger.Error("Got empty endpoint, no enabled endpoints available")
			http.Error(w, "No enabled endpoints available", http.StatusServiceUnavailable)
			return
		}

		// Increment attempt counter for current endpoint
		endpointAttempts++

		// Mark this endpoint as having active requests
		p.markRequestActive(endpoint.Name)

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
				p.stats.RecordError(endpoint.Name)
				p.markRequestInactive(endpoint.Name)
				// Retry logic: if first attempt, retry same endpoint; if second attempt, rotate
				if endpointAttempts >= 2 {
					p.rotateEndpoint()
					endpointAttempts = 0 // Reset counter for next endpoint
				}
				continue
			}
			trans = transformer.NewOpenAITransformer(endpoint.Model)
		} else if transformerName == "gemini" {
			if endpoint.Model == "" {
				logger.Error("[%s] Gemini transformer requires model field", endpoint.Name)
				p.stats.RecordError(endpoint.Name)
				p.markRequestInactive(endpoint.Name)
				// Retry logic: if first attempt, retry same endpoint; if second attempt, rotate
				if endpointAttempts >= 2 {
					p.rotateEndpoint()
					endpointAttempts = 0 // Reset counter for next endpoint
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
				p.stats.RecordError(endpoint.Name)
				p.markRequestInactive(endpoint.Name)
				// Retry logic: if first attempt, retry same endpoint; if second attempt, rotate
				if endpointAttempts >= 2 {
					p.rotateEndpoint()
					endpointAttempts = 0 // Reset counter for next endpoint
				}
				continue
			}
		}

		// Transform request from Claude format to target API format
		transformedBody, err := trans.TransformRequest(bodyBytes)
		if err != nil {
			logger.Error("[%s] Failed to transform request: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.markRequestInactive(endpoint.Name)
			// Retry logic: if first attempt, retry same endpoint; if second attempt, rotate
			if endpointAttempts >= 2 {
				p.rotateEndpoint()
				endpointAttempts = 0 // Reset counter for next endpoint
			}
			continue
		}

		logger.Debug("[%s] Using transformer: %s", endpoint.Name, transformerName)
		logger.DebugLog("[%s] Transformer: %s", endpoint.Name, transformerName)
		logger.DebugLog("[%s] Transformed Request: %s", endpoint.Name, string(transformedBody))

		// Clean incomplete tool_use/tool_result pairs after transformation
		// This ensures compatibility when switching between different API endpoints
		cleanedBody, err := cleanIncompleteToolCalls(transformedBody)
		if err != nil {
			logger.Warn("[%s] Failed to clean tool calls: %v, using original transformed request", endpoint.Name, err)
			cleanedBody = transformedBody
		}
		transformedBody = cleanedBody

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

		// Normalize API URL (add https:// if no protocol specified)
		normalizedAPIUrl := normalizeAPIUrl(endpoint.APIUrl)

		targetURL := fmt.Sprintf("%s%s", normalizedAPIUrl, targetPath)
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(transformedBody))
		if err != nil {
			logger.Error("[%s] Failed to create request: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.markRequestInactive(endpoint.Name)
			// Retry logic: if first attempt, retry same endpoint; if second attempt, rotate
			if endpointAttempts >= 2 {
				p.rotateEndpoint()
				endpointAttempts = 0 // Reset counter for next endpoint
			}
			continue
		}

		// Copy headers (except Host and authentication headers)
		for key, values := range r.Header {
			if key == "Host" {
				continue
			}
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		// Set authentication header based on transformer type
		switch transformerName {
		case "openai":
			proxyReq.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
		case "gemini":
			q := proxyReq.URL.Query()
			q.Set("key", endpoint.APIKey)
			proxyReq.URL.RawQuery = q.Encode()
		default:
			// Set both x-api-key and Authorization headers for compatibility
			// Some services use x-api-key (e.g., Anthropic Claude), others use Bearer token
			proxyReq.Header.Set("x-api-key", endpoint.APIKey)
			proxyReq.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
		}

		// Set Host to target API (required for proper routing)
		hostOnly := strings.TrimPrefix(strings.TrimPrefix(normalizedAPIUrl, "https://"), "http://")
		proxyReq.Header.Set("Host", hostOnly)

		// Send request
		client := &http.Client{
			Timeout: 300 * time.Second, // 5 minutes timeout for slow endpoints
		}

		resp, err := client.Do(proxyReq)
		if err != nil {
			logger.Error("[%s] Request failed: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.markRequestInactive(endpoint.Name)
			// Retry logic: if first attempt, retry same endpoint; if second attempt, rotate
			if endpointAttempts >= 2 {
				p.rotateEndpoint()
				endpointAttempts = 0 // Reset counter for next endpoint
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
			// Increase buffer size to handle large SSE events (default 64KB is too small)
			// Set to 1MB to handle large tool responses and content
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 1024*1024)
			var inputTokens, outputTokens int
			var buffer bytes.Buffer
			var outputText strings.Builder
			eventCount := 0
			streamDone := false

			for scanner.Scan() && !streamDone {
				line := scanner.Text()

				// Check if endpoint has been switched - if so, abort streaming
				if !p.isCurrentEndpoint(endpoint.Name) {
					logger.Warn("[%s] Endpoint switched during streaming, terminating stream gracefully", endpoint.Name)
					streamDone = true
					break
				}

				// Check for [DONE] marker to stop reading immediately
				if strings.Contains(line, "data: [DONE]") {
					streamDone = true
					buffer.WriteString(line + "\n")
					// Process the [DONE] event immediately
					eventData := buffer.Bytes()
					logger.DebugLog("[%s] SSE Event #%d (Original): %s", endpoint.Name, eventCount+1, string(eventData))

					var transformedEvent []byte
					var err error
					switch transformerName {
					case "openai":
						transformedEvent, err = trans.(*transformer.OpenAITransformer).TransformResponseWithContext(eventData, true, streamCtx)
					case "gemini":
						transformedEvent, err = trans.(*transformer.GeminiTransformer).TransformResponseWithContext(eventData, true, streamCtx)
					default:
						transformedEvent, err = trans.TransformResponse(eventData, true)
					}

					if err == nil {
						logger.DebugLog("[%s] SSE Event #%d (Transformed): %s", endpoint.Name, eventCount+1, string(transformedEvent))
						_, writeErr := w.Write(transformedEvent)
						if writeErr != nil {
							logger.Error("[%s] Failed to write [DONE] event: %v", endpoint.Name, writeErr)
						} else {
							flusher.Flush()
						}
					}
					break
				}

				buffer.WriteString(line + "\n")

				// When we hit an empty line, we have a complete event
				if line == "" {
					eventCount++
					// Transform the buffered event
					eventData := buffer.Bytes()

					logger.DebugLog("[%s] SSE Event #%d (Original): %s", endpoint.Name, eventCount, string(eventData))

					var transformedEvent []byte
					var err error

					// Transform based on transformer type
					switch transformerName {
					case "openai":
						transformedEvent, err = trans.(*transformer.OpenAITransformer).TransformResponseWithContext(eventData, true, streamCtx)
					case "gemini":
						transformedEvent, err = trans.(*transformer.GeminiTransformer).TransformResponseWithContext(eventData, true, streamCtx)
					default:
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

					// Check again before writing to make sure endpoint hasn't been switched
					if !p.isCurrentEndpoint(endpoint.Name) {
						logger.Warn("[%s] Endpoint switched before writing event #%d, aborting stream", endpoint.Name, eventCount)
						streamDone = true
						break
					}

					// Write transformed event
					_, writeErr := w.Write(transformedEvent)
					if writeErr != nil {
						logger.Error("[%s] Failed to write event #%d to client: %v", endpoint.Name, eventCount, writeErr)
						logger.DebugLog("[%s] Write Error #%d: %v", endpoint.Name, eventCount, writeErr)
						streamDone = true
						break
					}
					flusher.Flush()

					// Parse token usage and collect output text
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
											if cacheRead, ok := usage["cache_read_input_tokens"].(float64); ok && cacheRead > 0 {
												inputTokens += int(cacheRead)
											}
											if cacheCreate, ok := usage["cache_creation_input_tokens"].(float64); ok && cacheCreate > 0 {
												inputTokens += int(cacheCreate)
											}
										}
									}
								}
								if eventType == "content_block_delta" {
									if delta, ok := event["delta"].(map[string]interface{}); ok {
										if text, ok := delta["text"].(string); ok {
											outputText.WriteString(text)
										}
									}
								}
								if eventType == "message_delta" {
									if usage, ok := event["usage"].(map[string]interface{}); ok {
										if input, ok := usage["input_tokens"].(float64); ok {
											inputTokens = int(input)
										}
										if cacheRead, ok := usage["cache_read_input_tokens"].(float64); ok && cacheRead > 0 {
											inputTokens += int(cacheRead)
										}
										if cacheCreate, ok := usage["cache_creation_input_tokens"].(float64); ok && cacheCreate > 0 {
											inputTokens += int(cacheCreate)
										}
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

			// Check for scanner errors or unexpected stream termination
			if err := scanner.Err(); err != nil {
				logger.Error("[%s] Stream scanner error: %v", endpoint.Name, err)
			}

			// Process any remaining data in buffer (for events without trailing newline)
			if buffer.Len() > 0 && !streamDone {
				eventCount++
				eventData := buffer.Bytes()
				logger.DebugLog("[%s] SSE Event #%d (Final, no trailing newline): %s", endpoint.Name, eventCount, string(eventData))

				var transformedEvent []byte
				var err error

				// Transform based on transformer type
				switch transformerName {
				case "openai":
					transformedEvent, err = trans.(*transformer.OpenAITransformer).TransformResponseWithContext(eventData, true, streamCtx)
				case "gemini":
					transformedEvent, err = trans.(*transformer.GeminiTransformer).TransformResponseWithContext(eventData, true, streamCtx)
				default:
					transformedEvent, err = trans.TransformResponse(eventData, true)
				}

				if err == nil {
					logger.DebugLog("[%s] SSE Event #%d (Transformed): %s", endpoint.Name, eventCount, string(transformedEvent))
					_, writeErr := w.Write(transformedEvent)
					if writeErr != nil {
						logger.Error("[%s] Failed to write final event #%d to client: %v", endpoint.Name, eventCount, writeErr)
					} else {
						flusher.Flush()

						// Parse token usage and check for message_stop
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
												if cacheRead, ok := usage["cache_read_input_tokens"].(float64); ok && cacheRead > 0 {
													inputTokens += int(cacheRead)
												}
												if cacheCreate, ok := usage["cache_creation_input_tokens"].(float64); ok && cacheCreate > 0 {
													inputTokens += int(cacheCreate)
												}
											}
										}
									}
									if eventType == "content_block_delta" {
										if delta, ok := event["delta"].(map[string]interface{}); ok {
											if text, ok := delta["text"].(string); ok {
												outputText.WriteString(text)
											}
										}
									}
									if eventType == "message_delta" {
										if usage, ok := event["usage"].(map[string]interface{}); ok {
											if input, ok := usage["input_tokens"].(float64); ok {
												inputTokens = int(input)
											}
											if cacheRead, ok := usage["cache_read_input_tokens"].(float64); ok && cacheRead > 0 {
												inputTokens += int(cacheRead)
											}
											if cacheCreate, ok := usage["cache_creation_input_tokens"].(float64); ok && cacheCreate > 0 {
												inputTokens += int(cacheCreate)
											}
											if output, ok := usage["output_tokens"].(float64); ok {
												outputTokens = int(output)
											}
										}
									}
									if eventType == "message_stop" {
										streamDone = true
									}
								}
							}
						}
					}
				} else {
					logger.Error("[%s] Failed to transform final SSE event #%d: %v", endpoint.Name, eventCount, err)
				}

				buffer.Reset()
			}

			// If stream didn't end properly (no message_stop event sent), send one now
			if !streamDone {
				logger.Warn("[%s] Stream ended unexpectedly without proper termination, sending synthetic message_stop", endpoint.Name)

				// Close any open blocks (thinking, tool, or content)
				if streamCtx != nil {
					if streamCtx.ThinkingBlockStarted {
						blockStopEvent := map[string]interface{}{
							"type":  "content_block_stop",
							"index": streamCtx.ThinkingIndex,
						}
						blockStopJSON, _ := json.Marshal(blockStopEvent)
						w.Write([]byte("event: content_block_stop\n"))
						w.Write([]byte("data: " + string(blockStopJSON) + "\n\n"))
						flusher.Flush()
					}

					if streamCtx.ToolBlockStarted {
						blockStopEvent := map[string]interface{}{
							"type":  "content_block_stop",
							"index": streamCtx.LastToolIndex,
						}
						blockStopJSON, _ := json.Marshal(blockStopEvent)
						w.Write([]byte("event: content_block_stop\n"))
						w.Write([]byte("data: " + string(blockStopJSON) + "\n\n"))
						flusher.Flush()
					}

					if streamCtx.ContentBlockStarted {
						blockStopEvent := map[string]interface{}{
							"type":  "content_block_stop",
							"index": streamCtx.ContentIndex,
						}
						blockStopJSON, _ := json.Marshal(blockStopEvent)
						w.Write([]byte("event: content_block_stop\n"))
						w.Write([]byte("data: " + string(blockStopJSON) + "\n\n"))
						flusher.Flush()
					}
				}

				// Send message_delta with stop_reason
				var outputTokensForDelta int
				if streamCtx != nil {
					outputTokensForDelta = streamCtx.OutputTokens
				} else {
					outputTokensForDelta = outputTokens
				}

				messageDeltaEvent := map[string]interface{}{
					"type": "message_delta",
					"delta": map[string]interface{}{
						"stop_reason": "end_turn",
					},
					"usage": map[string]interface{}{
						"output_tokens": outputTokensForDelta,
					},
				}
				messageDeltaJSON, _ := json.Marshal(messageDeltaEvent)
				w.Write([]byte("event: message_delta\n"))
				w.Write([]byte("data: " + string(messageDeltaJSON) + "\n\n"))
				flusher.Flush()

				// Send message_stop event
				stopEvent := map[string]interface{}{
					"type": "message_stop",
				}
				stopJSON, _ := json.Marshal(stopEvent)
				w.Write([]byte("event: message_stop\n"))
				w.Write([]byte("data: " + string(stopJSON) + "\n\n"))
				flusher.Flush()
			}

			// Fallback: estimate tokens when usage is 0
			if inputTokens == 0 || outputTokens == 0 {
				if inputTokens == 0 {
					var req tokencount.CountTokensRequest
					if json.Unmarshal(bodyBytes, &req) == nil {
						inputTokens = tokencount.EstimateInputTokens(&req)
						logger.Debug("[%s] Estimated streaming input tokens: %d", endpoint.Name, inputTokens)
					}
				}

				if outputTokens == 0 && outputText.Len() > 0 {
					outputTokens = tokencount.EstimateOutputTokens(outputText.String())
					logger.Debug("[%s] Estimated streaming output tokens: %d", endpoint.Name, outputTokens)
				}
			}

			if inputTokens > 0 || outputTokens > 0 {
				p.stats.RecordTokens(endpoint.Name, inputTokens, outputTokens)
			}

			// Clean up before returning
			p.markRequestInactive(endpoint.Name)
			return
		}

		// For non-streaming responses, read the full body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.Error("[%s] Failed to read response: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.markRequestInactive(endpoint.Name)
			// Retry logic: if first attempt, retry same endpoint; if second attempt, rotate
			if endpointAttempts >= 2 {
				p.rotateEndpoint()
				endpointAttempts = 0 // Reset counter for next endpoint
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
				logger.Error("[%s] HTTP %d: %s", endpoint.Name, resp.StatusCode, errorMsg)
			} else {
				logger.Error("[%s] HTTP %d %s", endpoint.Name, resp.StatusCode, http.StatusText(resp.StatusCode))
			}

			p.stats.RecordError(endpoint.Name)
			p.markRequestInactive(endpoint.Name)
			// Retry logic: if first attempt, retry same endpoint; if second attempt, rotate
			if endpointAttempts >= 2 {
				p.rotateEndpoint()
				endpointAttempts = 0 // Reset counter for next endpoint
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
				p.stats.RecordError(endpoint.Name)
				p.markRequestInactive(endpoint.Name)
				// Retry logic: if first attempt, retry same endpoint; if second attempt, rotate
				if endpointAttempts >= 2 {
					p.rotateEndpoint()
					endpointAttempts = 0 // Reset counter for next endpoint
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
				inputTokens := apiResp.Usage.InputTokens
				outputTokens := apiResp.Usage.OutputTokens

				// Fallback: estimate tokens when usage is 0
				if inputTokens == 0 || outputTokens == 0 {
					if inputTokens == 0 {
						var req tokencount.CountTokensRequest
						if json.Unmarshal(bodyBytes, &req) == nil {
							inputTokens = tokencount.EstimateInputTokens(&req)
							logger.Debug("[%s] Estimated input tokens: %d", endpoint.Name, inputTokens)
						}
					}

					if outputTokens == 0 {
						var resp map[string]interface{}
						if json.Unmarshal(transformedResp, &resp) == nil {
							if content, ok := resp["content"].([]interface{}); ok {
								var totalText strings.Builder
								for _, item := range content {
									if block, ok := item.(map[string]interface{}); ok {
										if blockType, _ := block["type"].(string); blockType == "text" {
											if text, ok := block["text"].(string); ok {
												totalText.WriteString(text)
											}
										}
									}
								}
								if totalText.Len() > 0 {
									outputTokens = tokencount.EstimateOutputTokens(totalText.String())
									logger.Debug("[%s] Estimated output tokens: %d", endpoint.Name, outputTokens)
								}
							}
						}
					}
				}

				if inputTokens > 0 || outputTokens > 0 {
					p.stats.RecordTokens(endpoint.Name, inputTokens, outputTokens)
				}
			}

			// Clean up before returning
			p.markRequestInactive(endpoint.Name)
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

		// Clean up before returning
		p.markRequestInactive(endpoint.Name)
		return
	}

	// All endpoints failed
	logger.Error("All endpoints failed after %d retries", maxRetries)
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

// handleCountTokens handles token counting with fallback
func (p *Proxy) handleCountTokens(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req tokencount.CountTokensRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	endpoint := p.getCurrentEndpoint()
	if endpoint.Name == "" {
		// No endpoint available, use local estimation
		tokens := tokencount.EstimateInputTokens(&req)
		resp := tokencount.CountTokensResponse{InputTokens: tokens}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Try to proxy to backend API
	normalizedAPIUrl := normalizeAPIUrl(endpoint.APIUrl)
	targetURL := fmt.Sprintf("%s/v1/messages/count_tokens", normalizedAPIUrl)

	proxyReq, err := http.NewRequest("POST", targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		// Fallback to local estimation
		tokens := tokencount.EstimateInputTokens(&req)
		resp := tokencount.CountTokensResponse{InputTokens: tokens}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	proxyReq.Header.Set("x-api-key", endpoint.APIKey)
	proxyReq.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
	proxyReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second} // Token counting should be fast
	resp, err := client.Do(proxyReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Fallback to local estimation
		tokens := tokencount.EstimateInputTokens(&req)
		response := tokencount.CountTokensResponse{InputTokens: tokens}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		logger.Debug("[%s] count_tokens failed, using estimation: %d", endpoint.Name, tokens)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		// Fallback to local estimation
		tokens := tokencount.EstimateInputTokens(&req)
		response := tokencount.CountTokensResponse{InputTokens: tokens}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var apiResp tokencount.CountTokensResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil || apiResp.InputTokens == 0 {
		// Fallback to local estimation if parse failed or tokens is 0
		tokens := tokencount.EstimateInputTokens(&req)
		response := tokencount.CountTokensResponse{InputTokens: tokens}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		logger.Debug("[%s] count_tokens returned 0, using estimation: %d", endpoint.Name, tokens)
		return
	}

	// Return API response
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
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
