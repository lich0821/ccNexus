package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/yourusername/claude-proxy/internal/config"
)

// Proxy represents the proxy server
type Proxy struct {
	config           *config.Config
	stats            *Stats
	currentIndex     int
	mu               sync.RWMutex
	server           *http.Server
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

	log.Printf("🚀 ccNexus starting on port %d", port)
	log.Printf("🔑 Configured %d endpoints", len(p.config.GetEndpoints()))

	return p.server.ListenAndServe()
}

// Stop stops the proxy server
func (p *Proxy) Stop() error {
	if p.server != nil {
		return p.server.Close()
	}
	return nil
}

// getCurrentEndpoint returns the current endpoint (thread-safe)
func (p *Proxy) getCurrentEndpoint() config.Endpoint {
	p.mu.RLock()
	defer p.mu.RUnlock()

	endpoints := p.config.GetEndpoints()
	return endpoints[p.currentIndex]
}

// rotateEndpoint switches to the next endpoint (thread-safe)
func (p *Proxy) rotateEndpoint() config.Endpoint {
	p.mu.Lock()
	defer p.mu.Unlock()

	endpoints := p.config.GetEndpoints()
	oldIndex := p.currentIndex
	p.currentIndex = (p.currentIndex + 1) % len(endpoints)

	newEndpoint := endpoints[p.currentIndex]
	log.Printf("🔄 Rotating endpoint: #%d -> #%d (%s)", oldIndex+1, p.currentIndex+1, newEndpoint.Name)

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
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	endpoints := p.config.GetEndpoints()
	maxRetries := len(endpoints)

	// Try each endpoint
	for retry := 0; retry < maxRetries; retry++ {
		endpoint := p.getCurrentEndpoint()

		// Record request
		p.stats.RecordRequest(endpoint.Name)

		// Create new request
		targetURL := fmt.Sprintf("https://%s%s", endpoint.APIUrl, r.URL.Path)
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(bodyBytes))
		if err != nil {
			log.Printf("❌ Failed to create request: %v", err)
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
		log.Printf("📤 [%s #%d] %s %s", endpoint.Name, p.currentIndex+1, r.Method, r.URL.Path)

		client := &http.Client{
			Timeout: 120 * time.Second,
		}

		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Printf("❌ Request failed: %v", err)
			p.stats.RecordError(endpoint.Name)
			p.rotateEndpoint()
			continue
		}

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("❌ Failed to read response: %v", err)
			p.stats.RecordError(endpoint.Name)
			p.rotateEndpoint()
			continue
		}

		// Check if we should retry
		if shouldRetry(resp.StatusCode) {
			log.Printf("⚠️  Non-200 response (%d), rotating endpoint", resp.StatusCode)
			p.stats.RecordError(endpoint.Name)
			p.rotateEndpoint()

			// If this is not the last retry, continue to next endpoint
			if retry < maxRetries-1 {
				log.Printf("🔁 Retrying request (%d/%d)", retry+1, maxRetries)
				continue
			}
		}

		// Success or last retry - return response
		log.Printf("📥 [%s #%d] %d %d bytes", endpoint.Name, p.currentIndex+1, resp.StatusCode, len(respBody))

		// Copy response headers
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
	log.Printf("❌ All endpoints failed after %d retries", maxRetries)
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

	log.Printf("✅ Configuration updated: %d endpoints", len(cfg.GetEndpoints()))
	return nil
}
