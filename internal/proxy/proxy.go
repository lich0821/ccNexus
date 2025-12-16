package proxy

import (
	"context"
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
	activeRequests   map[string]bool              // tracks active requests by endpoint name
	activeRequestsMu sync.RWMutex                 // protects activeRequests map
	endpointCtx      map[string]context.Context   // context per endpoint for cancellation
	endpointCancel   map[string]context.CancelFunc // cancel functions per endpoint
	ctxMu            sync.RWMutex                 // protects context maps
	onEndpointSuccess func(endpointName string)   // callback when endpoint request succeeds
}

// New creates a new Proxy instance
func New(cfg *config.Config, statsStorage StatsStorage, deviceID string) *Proxy {
	stats := NewStats(statsStorage, deviceID)

	return &Proxy{
		config:         cfg,
		stats:          stats,
		currentIndex:   0,
		activeRequests: make(map[string]bool),
		endpointCtx:    make(map[string]context.Context),
		endpointCancel: make(map[string]context.CancelFunc),
	}
}

// SetOnEndpointSuccess sets the callback for successful endpoint requests
func (p *Proxy) SetOnEndpointSuccess(callback func(endpointName string)) {
	p.onEndpointSuccess = callback
}

// Start starts the proxy server
func (p *Proxy) Start() error {
	return p.StartWithMux(nil)
}

// StartWithMux starts the proxy server with an optional custom mux
func (p *Proxy) StartWithMux(customMux *http.ServeMux) error {
	port := p.config.GetPort()

	var mux *http.ServeMux
	if customMux != nil {
		mux = customMux
	} else {
		mux = http.NewServeMux()
	}

	// Register proxy routes
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

// getEndpointContext returns a context for the given endpoint, creating one if needed
func (p *Proxy) getEndpointContext(endpointName string) context.Context {
	p.ctxMu.Lock()
	defer p.ctxMu.Unlock()

	if ctx, ok := p.endpointCtx[endpointName]; ok {
		return ctx
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.endpointCtx[endpointName] = ctx
	p.endpointCancel[endpointName] = cancel
	return ctx
}

// cancelEndpointRequests cancels all requests for the given endpoint
func (p *Proxy) cancelEndpointRequests(endpointName string) {
	p.ctxMu.Lock()
	defer p.ctxMu.Unlock()

	if cancel, ok := p.endpointCancel[endpointName]; ok {
		cancel()
		delete(p.endpointCtx, endpointName)
		delete(p.endpointCancel, endpointName)
	}
}

// rotateEndpoint switches to the next endpoint (thread-safe)
// waitForActive: if true, waits briefly for active requests to complete before switching
func (p *Proxy) rotateEndpoint() config.Endpoint {
	p.mu.Lock()
	defer p.mu.Unlock()

	endpoints := p.getEnabledEndpoints()
	if len(endpoints) == 0 {
		return config.Endpoint{}
	}

	oldIndex := p.currentIndex % len(endpoints)
	oldEndpoint := endpoints[oldIndex]

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

		// Re-fetch endpoints after re-acquiring lock (may have changed)
		endpoints = p.getEnabledEndpoints()
		if len(endpoints) == 0 {
			return config.Endpoint{}
		}
	}

	// Use oldIndex to calculate next, avoiding skip if currentIndex was modified during wait
	p.currentIndex = (oldIndex + 1) % len(endpoints)

	newEndpoint := endpoints[p.currentIndex]
	logger.Debug("[SWITCH] %s → %s (#%d)", oldEndpoint.Name, newEndpoint.Name, p.currentIndex+1)

	return newEndpoint
}

// GetCurrentEndpointName returns the current endpoint name (thread-safe)
func (p *Proxy) GetCurrentEndpointName() string {
	endpoint := p.getCurrentEndpoint()
	return endpoint.Name
}

// SetCurrentEndpoint manually switches to a specific endpoint by name
// Returns error if endpoint not found or not enabled
// Thread-safe and cancels ongoing requests on the old endpoint
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
			if oldEndpoint.Name != targetName {
				// Cancel all requests on the old endpoint
				p.cancelEndpointRequests(oldEndpoint.Name)
			}
			p.currentIndex = i
			logger.Info("[MANUAL SWITCH] %s → %s", oldEndpoint.Name, ep.Name)
			return nil
		}
	}

	return fmt.Errorf("endpoint '%s' not found or not enabled", targetName)
}

// ClientFormat represents the API format used by the client
type ClientFormat string

const (
	ClientFormatClaude          ClientFormat = "claude"           // Claude Code: /v1/messages
	ClientFormatOpenAIChat      ClientFormat = "openai_chat"      // Codex (chat): /v1/chat/completions
	ClientFormatOpenAIResponses ClientFormat = "openai_responses" // Codex (responses): /v1/responses
)

// detectClientFormat identifies the client format based on request path
func detectClientFormat(path string) ClientFormat {
	switch {
	case strings.HasPrefix(path, "/v1/chat/completions") || strings.HasPrefix(path, "/chat/completions"):
		return ClientFormatOpenAIChat
	case strings.HasPrefix(path, "/v1/responses") || strings.HasPrefix(path, "/responses"):
		return ClientFormatOpenAIResponses
	default:
		return ClientFormatClaude
	}
}

// handleProxy handles the main proxy logic
func (p *Proxy) handleProxy(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Detect client format
	clientFormat := detectClientFormat(r.URL.Path)

	logger.DebugLog("=== Proxy Request ===")
	logger.DebugLog("Method: %s, Path: %s, ClientFormat: %s", r.Method, r.URL.Path, clientFormat)
	logger.DebugLog("Request Body: %s", string(bodyBytes))

	var streamReq struct {
		Model    string      `json:"model"`
		Thinking interface{} `json:"thinking"`
		Stream   bool        `json:"stream"`
	}
	json.Unmarshal(bodyBytes, &streamReq)

	endpoints := p.getEnabledEndpoints()
	if len(endpoints) == 0 {
		logger.Error("No enabled endpoints available")
		http.Error(w, "No enabled endpoints configured", http.StatusServiceUnavailable)
		return
	}

	maxRetries := len(endpoints) * 2
	endpointAttempts := 0
	lastEndpointName := ""

	for retry := 0; retry < maxRetries; retry++ {
		endpoint := p.getCurrentEndpoint()
		if endpoint.Name == "" {
			http.Error(w, "No enabled endpoints available", http.StatusServiceUnavailable)
			return
		}

		// Reset attempts counter if endpoint changed (e.g., manual switch)
		if lastEndpointName != "" && lastEndpointName != endpoint.Name {
			endpointAttempts = 0
		}
		lastEndpointName = endpoint.Name

		endpointAttempts++
		p.markRequestActive(endpoint.Name)
		p.stats.RecordRequest(endpoint.Name)

		trans, err := prepareTransformerForClient(clientFormat, endpoint)
		if err != nil {
			logger.Error("[%s] %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.markRequestInactive(endpoint.Name)
			if endpointAttempts >= 2 {
				p.rotateEndpoint()
				endpointAttempts = 0
			}
			continue
		}

		transformerName := trans.Name()

		transformedBody, err := trans.TransformRequest(bodyBytes)
		if err != nil {
			logger.Error("[%s] Failed to transform request: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.markRequestInactive(endpoint.Name)
			if endpointAttempts >= 2 {
				p.rotateEndpoint()
				endpointAttempts = 0
			}
			continue
		}

		logger.DebugLog("[%s] Transformer: %s", endpoint.Name, transformerName)
		logger.DebugLog("[%s] Transformed Request: %s", endpoint.Name, string(transformedBody))

		cleanedBody, err := cleanIncompleteToolCalls(transformedBody)
		if err != nil {
			logger.Warn("[%s] Failed to clean tool calls: %v", endpoint.Name, err)
			cleanedBody = transformedBody
		}
		transformedBody = cleanedBody

		var thinkingEnabled bool
		if strings.Contains(transformerName, "openai") {
			var openaiReq map[string]interface{}
			if err := json.Unmarshal(transformedBody, &openaiReq); err == nil {
				if enable, ok := openaiReq["enable_thinking"].(bool); ok {
					thinkingEnabled = enable
				}
			}
		}

		proxyReq, err := buildProxyRequest(r, endpoint, transformedBody, transformerName)
		if err != nil {
			logger.Error("[%s] Failed to create request: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.markRequestInactive(endpoint.Name)
			if endpointAttempts >= 2 {
				p.rotateEndpoint()
				endpointAttempts = 0
			}
			continue
		}

		ctx := p.getEndpointContext(endpoint.Name)
		resp, err := sendRequest(ctx, proxyReq)
		if err != nil {
			logger.Error("[%s] Request failed: %v", endpoint.Name, err)
			p.stats.RecordError(endpoint.Name)
			p.markRequestInactive(endpoint.Name)
			if endpointAttempts >= 2 {
				p.rotateEndpoint()
				endpointAttempts = 0
			}
			continue
		}

		contentType := resp.Header.Get("Content-Type")
		isStreaming := contentType == "text/event-stream" || (streamReq.Stream && strings.Contains(contentType, "text/event-stream"))

		if resp.StatusCode == http.StatusOK && isStreaming {
			inputTokens, outputTokens, outputText := p.handleStreamingResponse(w, resp, endpoint, trans, transformerName, thinkingEnabled, streamReq.Model, bodyBytes)

			// Fallback: estimate tokens when usage is 0
			if inputTokens == 0 || outputTokens == 0 {
				inputTokens, outputTokens = p.estimateTokens(bodyBytes, outputText, inputTokens, outputTokens, endpoint.Name)
			}

			p.stats.RecordTokens(endpoint.Name, inputTokens, outputTokens)
			p.markRequestInactive(endpoint.Name)
			if p.onEndpointSuccess != nil {
				p.onEndpointSuccess(endpoint.Name)
			}
			logger.Debug("[%s] Request completed successfully (streaming)", endpoint.Name)
			return
		}

		if resp.StatusCode == http.StatusOK {
			inputTokens, outputTokens, err := p.handleNonStreamingResponse(w, resp, endpoint, trans)
			if err == nil {
				p.stats.RecordTokens(endpoint.Name, inputTokens, outputTokens)
				p.markRequestInactive(endpoint.Name)
				if p.onEndpointSuccess != nil {
					p.onEndpointSuccess(endpoint.Name)
				}
				logger.Debug("[%s] Request completed successfully", endpoint.Name)
				return
			}
		}

		if shouldRetry(resp.StatusCode) {
			var errBody []byte
			if resp.Header.Get("Content-Encoding") == "gzip" {
				errBody, _ = decompressGzip(resp.Body)
			} else {
				errBody, _ = io.ReadAll(resp.Body)
			}
			resp.Body.Close()
			errMsg := string(errBody)
			if len(errMsg) > 200 {
				errMsg = errMsg[:200] + "..."
			}
			logger.Warn("[%s] Request failed %d: %s", endpoint.Name, resp.StatusCode, errMsg)
			logger.DebugLog("[%s] Request failed %d: %s", endpoint.Name, resp.StatusCode, errMsg)
			p.stats.RecordError(endpoint.Name)
			p.markRequestInactive(endpoint.Name)
			if endpointAttempts >= 2 {
				p.rotateEndpoint()
				endpointAttempts = 0
			}
			continue
		}

		var respBody []byte
		if resp.Header.Get("Content-Encoding") == "gzip" {
			respBody, _ = decompressGzip(resp.Body)
		} else {
			respBody, _ = io.ReadAll(resp.Body)
		}
		resp.Body.Close()
		p.markRequestInactive(endpoint.Name)
		// Log non-200 responses for debugging
		if resp.StatusCode != http.StatusOK {
			errMsg := string(respBody)
			if len(errMsg) > 500 {
				errMsg = errMsg[:500] + "..."
			}
			logger.Warn("[%s] Response %d: %s", endpoint.Name, resp.StatusCode, errMsg)
			logger.DebugLog("[%s] Response %d: %s", endpoint.Name, resp.StatusCode, errMsg)
		}
		// Remove Content-Encoding header since we've decompressed
		for key, values := range resp.Header {
			if key == "Content-Encoding" || key == "Content-Length" {
				continue
			}
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}

	http.Error(w, "All endpoints failed", http.StatusServiceUnavailable)
}
