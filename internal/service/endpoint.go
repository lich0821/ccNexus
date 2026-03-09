package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/proxy"
	"github.com/lich0821/ccNexus/internal/storage"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// createHTTPClient creates an HTTP client with optional proxy support
func (e *EndpointService) createHTTPClient(timeout time.Duration, targetURL string) *http.Client {
	// Always create client with proper transport configuration
	// Enhanced for large SSE streaming and HTTP/2 support
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:           100,
			MaxIdleConnsPerHost:    10,
			IdleConnTimeout:        90 * time.Second,
			TLSHandshakeTimeout:    10 * time.Second,
			ExpectContinueTimeout:  1 * time.Second,
			ResponseHeaderTimeout:  30 * time.Second,
			WriteBufferSize:        128 * 1024, // 128KB write buffer
			ReadBufferSize:         128 * 1024, // 128KB read buffer
			MaxResponseHeaderBytes: 64 * 1024,  // 64KB max response headers
		},
	}

	proxyURL := e.resolveProxyURLForTarget(targetURL)
	// Override with proxy transport if configured
	if strings.TrimSpace(proxyURL) != "" {
		logger.Debug("Using proxy for request: %s", proxyURL)
		if transport, err := proxy.CreateProxyTransport(proxyURL); err == nil {
			client.Transport = transport
		} else {
			logger.Warn("Failed to create proxy transport: %v, using direct connection", err)
		}
	} else {
		logger.Debug("No proxy configured, using direct connection for request")
	}

	return client
}

func (e *EndpointService) resolveProxyURLForTarget(targetURL string) string {
	targetURL = strings.TrimSpace(targetURL)
	if targetURL != "" {
		if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
			targetURL = "https://" + targetURL
		}
		parsed, err := url.Parse(targetURL)
		if err == nil && parsed != nil {
			host := strings.ToLower(strings.TrimSpace(parsed.Host))
			cleanPath := path.Clean(strings.TrimSpace(parsed.Path))
			if host == "chatgpt.com" && strings.Contains(cleanPath, "/backend-api/codex") {
				if codexProxy := e.config.GetCodexProxy(); codexProxy != nil && strings.TrimSpace(codexProxy.URL) != "" {
					return codexProxy.URL
				}
			}
		}
	}
	if proxyCfg := e.config.GetProxy(); proxyCfg != nil && strings.TrimSpace(proxyCfg.URL) != "" {
		return proxyCfg.URL
	}
	return ""
}

// Test endpoint constants
const (
	testMessage   = "你是什么模型?"
	testMaxTokens = 16

	codexTestClientVersion = "0.101.0"
	codexTestUserAgent     = "codex_cli_rs/0.101.0 (Mac OS 26.0.1; arm64) Apple_Terminal/464"
)

// EndpointService handles endpoint management operations
type EndpointService struct {
	config  *config.Config
	proxy   *proxy.Proxy
	storage *storage.SQLiteStorage
}

// NewEndpointService creates a new EndpointService
func NewEndpointService(cfg *config.Config, p *proxy.Proxy, s *storage.SQLiteStorage) *EndpointService {
	return &EndpointService{
		config:  cfg,
		proxy:   p,
		storage: s,
	}
}

// normalizeAPIUrl ensures the API URL has the correct format
func normalizeAPIUrl(apiUrl string) string {
	return strings.TrimSuffix(apiUrl, "/")
}

func (e *EndpointService) resolveEndpointAuth(endpoint config.Endpoint) (string, *storage.EndpointCredential, error) {
	authMode := config.NormalizeAuthMode(endpoint.AuthMode)
	if config.IsTokenPoolAuthMode(authMode) {
		if e.storage == nil {
			return "", nil, fmt.Errorf("token pool mode requires storage")
		}
		cred, err := e.storage.GetUsableEndpointCredential(endpoint.Name, time.Now().UTC())
		if err != nil {
			return "", nil, fmt.Errorf("failed to select token from token pool: %w", err)
		}
		if cred == nil || strings.TrimSpace(cred.AccessToken) == "" {
			return "", nil, fmt.Errorf("no usable token in token pool")
		}
		return strings.TrimSpace(cred.AccessToken), cred, nil
	}

	apiKey := strings.TrimSpace(endpoint.APIKey)
	if apiKey == "" {
		return "", nil, fmt.Errorf("api key is required")
	}
	return apiKey, nil, nil
}

func (e *EndpointService) resolveEndpointAPIKey(endpoint config.Endpoint) (string, error) {
	apiKey, _, err := e.resolveEndpointAuth(endpoint)
	return apiKey, err
}

// AddEndpoint adds a new endpoint
func (e *EndpointService) AddEndpoint(name, apiUrl, apiKey, authMode, transformer, model, remark string) error {
	endpoints := e.config.GetEndpoints()
	for _, ep := range endpoints {
		if ep.Name == name {
			return fmt.Errorf("endpoint name '%s' already exists", name)
		}
	}

	if transformer == "" {
		transformer = "claude"
	}
	authMode = config.NormalizeAuthMode(authMode)
	if config.IsTokenPoolAuthMode(authMode) {
		apiKey = ""
	}

	apiUrl = normalizeAPIUrl(apiUrl)

	newEndpoint := config.Endpoint{
		Name:        name,
		APIUrl:      apiUrl,
		APIKey:      apiKey,
		AuthMode:    authMode,
		Enabled:     true,
		Transformer: transformer,
		Model:       model,
		Remark:      remark,
	}
	config.ApplyEndpointAuthModeRules(&newEndpoint)
	endpoints = append(endpoints, newEndpoint)

	e.config.UpdateEndpoints(endpoints)

	if err := e.config.Validate(); err != nil {
		return err
	}

	if err := e.proxy.UpdateConfig(e.config); err != nil {
		return err
	}

	if e.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(e.storage)
		if err := e.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	if newEndpoint.Model != "" {
		logger.Info("Endpoint added: %s (%s) [%s/%s]", newEndpoint.Name, newEndpoint.APIUrl, newEndpoint.Transformer, newEndpoint.Model)
	} else {
		logger.Info("Endpoint added: %s (%s) [%s]", newEndpoint.Name, newEndpoint.APIUrl, newEndpoint.Transformer)
	}

	return nil
}

// RemoveEndpoint removes an endpoint by index
func (e *EndpointService) RemoveEndpoint(index int) error {
	endpoints := e.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		return fmt.Errorf("invalid endpoint index: %d", index)
	}

	removedName := endpoints[index].Name
	endpoints = append(endpoints[:index], endpoints[index+1:]...)
	e.config.UpdateEndpoints(endpoints)

	if len(endpoints) > 0 {
		if err := e.config.Validate(); err != nil {
			return err
		}
	}

	if err := e.proxy.UpdateConfig(e.config); err != nil {
		return err
	}

	if e.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(e.storage)
		if err := e.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	logger.Info("Endpoint removed: %s", removedName)
	return nil
}

// UpdateEndpoint updates an endpoint by index
func (e *EndpointService) UpdateEndpoint(index int, name, apiUrl, apiKey, authMode, transformer, model, remark string) error {
	endpoints := e.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		return fmt.Errorf("invalid endpoint index: %d", index)
	}

	oldName := endpoints[index].Name

	if oldName != name {
		for i, ep := range endpoints {
			if i != index && ep.Name == name {
				return fmt.Errorf("endpoint name '%s' already exists", name)
			}
		}
	}

	enabled := endpoints[index].Enabled

	if transformer == "" {
		transformer = "claude"
	}
	authMode = config.NormalizeAuthMode(authMode)
	if config.IsTokenPoolAuthMode(authMode) {
		apiKey = ""
	}

	apiUrl = normalizeAPIUrl(apiUrl)

	updatedEndpoint := config.Endpoint{
		Name:        name,
		APIUrl:      apiUrl,
		APIKey:      apiKey,
		AuthMode:    authMode,
		Enabled:     enabled,
		Transformer: transformer,
		Model:       model,
		Remark:      remark,
	}
	config.ApplyEndpointAuthModeRules(&updatedEndpoint)
	endpoints[index] = updatedEndpoint

	e.config.UpdateEndpoints(endpoints)

	if err := e.config.Validate(); err != nil {
		return err
	}

	if err := e.proxy.UpdateConfig(e.config); err != nil {
		return err
	}

	if e.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(e.storage)
		if err := e.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	if oldName != name {
		if updatedEndpoint.Model != "" {
			logger.Info("Endpoint updated: %s → %s (%s) [%s/%s]", oldName, updatedEndpoint.Name, updatedEndpoint.APIUrl, updatedEndpoint.Transformer, updatedEndpoint.Model)
		} else {
			logger.Info("Endpoint updated: %s → %s (%s) [%s]", oldName, updatedEndpoint.Name, updatedEndpoint.APIUrl, updatedEndpoint.Transformer)
		}
	} else {
		if updatedEndpoint.Model != "" {
			logger.Info("Endpoint updated: %s (%s) [%s/%s]", updatedEndpoint.Name, updatedEndpoint.APIUrl, updatedEndpoint.Transformer, updatedEndpoint.Model)
		} else {
			logger.Info("Endpoint updated: %s (%s) [%s]", updatedEndpoint.Name, updatedEndpoint.APIUrl, updatedEndpoint.Transformer)
		}
	}

	return nil
}

// ToggleEndpoint toggles the enabled state of an endpoint
func (e *EndpointService) ToggleEndpoint(index int, enabled bool) error {
	endpoints := e.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		return fmt.Errorf("invalid endpoint index: %d", index)
	}

	endpointName := endpoints[index].Name
	endpoints[index].Enabled = enabled
	e.config.UpdateEndpoints(endpoints)

	if err := e.proxy.UpdateConfig(e.config); err != nil {
		return err
	}

	if e.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(e.storage)
		if err := e.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	if enabled {
		logger.Info("Endpoint enabled: %s", endpointName)
	} else {
		logger.Info("Endpoint disabled: %s", endpointName)
	}

	return nil
}

// ReorderEndpoints reorders endpoints based on the provided name array
func (e *EndpointService) ReorderEndpoints(names []string) error {
	endpoints := e.config.GetEndpoints()

	if len(names) != len(endpoints) {
		return fmt.Errorf("names array length (%d) doesn't match endpoints count (%d)", len(names), len(endpoints))
	}

	seen := make(map[string]bool)
	for _, name := range names {
		if seen[name] {
			return fmt.Errorf("duplicate endpoint name in reorder request: %s", name)
		}
		seen[name] = true
	}

	endpointMap := make(map[string]config.Endpoint)
	for _, ep := range endpoints {
		endpointMap[ep.Name] = ep
	}

	newEndpoints := make([]config.Endpoint, 0, len(names))
	for _, name := range names {
		ep, exists := endpointMap[name]
		if !exists {
			return fmt.Errorf("endpoint not found: %s", name)
		}
		newEndpoints = append(newEndpoints, ep)
	}

	e.config.UpdateEndpoints(newEndpoints)

	if err := e.config.Validate(); err != nil {
		return err
	}

	if err := e.proxy.UpdateConfig(e.config); err != nil {
		return err
	}

	if e.storage != nil {
		configAdapter := storage.NewConfigStorageAdapter(e.storage)
		if err := e.config.SaveToStorage(configAdapter); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	logger.Info("Endpoints reordered: %v", names)
	return nil
}

// GetCurrentEndpoint returns the current active endpoint name
func (e *EndpointService) GetCurrentEndpoint() string {
	if e.proxy == nil {
		return ""
	}
	return e.proxy.GetCurrentEndpointName()
}

// SwitchToEndpoint manually switches to a specific endpoint by name
func (e *EndpointService) SwitchToEndpoint(endpointName string) error {
	if e.proxy == nil {
		return fmt.Errorf("proxy not initialized")
	}
	return e.proxy.SetCurrentEndpoint(endpointName)
}

// TestEndpoint tests an endpoint by sending a simple request
func (e *EndpointService) TestEndpoint(index int) string {
	endpoints := e.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Invalid endpoint index: %d", index),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	endpoint := endpoints[index]
	logger.Info("Testing endpoint: %s (%s)", endpoint.Name, endpoint.APIUrl)

	apiKey, credential, authErr := e.resolveEndpointAuth(endpoint)
	if authErr != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Authentication unavailable: %v", authErr),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	var requestBody []byte
	var err error
	var apiPath string

	transformer := endpoint.Transformer
	if transformer == "" {
		transformer = "claude"
	}

	switch transformer {
	case "claude":
		apiPath = "/v1/messages"
		model := endpoint.Model
		if model == "" {
			model = "claude-sonnet-4-5-20250929"
		}
		requestBody, err = json.Marshal(map[string]interface{}{
			"model":      model,
			"max_tokens": testMaxTokens,
			"messages": []map[string]string{
				{"role": "user", "content": testMessage},
			},
		})

	case "openai":
		apiPath = "/v1/chat/completions"
		model := endpoint.Model
		if model == "" {
			model = "gpt-4-turbo"
		}
		requestBody, err = json.Marshal(map[string]interface{}{
			"model":      model,
			"max_tokens": testMaxTokens,
			"messages": []map[string]interface{}{
				{"role": "user", "content": testMessage},
			},
		})

	case "openai2":
		apiPath = "/v1/responses"
		model := endpoint.Model
		if model == "" {
			model = "gpt-5-codex"
		}
		requestBody, err = json.Marshal(map[string]interface{}{
			"model": model,
			"input": []map[string]interface{}{
				{
					"type": "message",
					"role": "user",
					"content": []map[string]interface{}{
						{"type": "input_text", "text": testMessage},
					},
				},
			},
		})

	case "gemini":
		model := endpoint.Model
		if model == "" {
			model = "gemini-pro"
		}
		apiPath = "/v1beta/models/" + model + ":generateContent"
		requestBody, err = json.Marshal(map[string]interface{}{
			"contents": []map[string]interface{}{
				{"parts": []map[string]string{{"text": testMessage}}},
			},
			"generationConfig": map[string]int{"maxOutputTokens": testMaxTokens},
		})

	default:
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Unsupported transformer: %s", transformer),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to build request: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	normalizedAPIUrl := normalizeAPIUrl(endpoint.APIUrl)
	if !strings.HasPrefix(normalizedAPIUrl, "http://") && !strings.HasPrefix(normalizedAPIUrl, "https://") {
		normalizedAPIUrl = "https://" + normalizedAPIUrl
	}
	if transformer == "openai2" && isCodexBackendAPIURL(normalizedAPIUrl) {
		requestBody = ensureCodexResponsesProbePayload(requestBody)
	}
	url := fmt.Sprintf("%s%s", normalizedAPIUrl, normalizeEndpointPathForBaseURL(normalizedAPIUrl, apiPath))

	req, err := http.NewRequest("POST", url, bytes.NewReader(requestBody))
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to create request: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	req.Header.Set("Content-Type", "application/json")
	switch transformer {
	case "claude":
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case "openai", "openai2":
		req.Header.Set("Authorization", "Bearer "+apiKey)
	case "gemini":
		q := req.URL.Query()
		q.Add("key", apiKey)
		req.URL.RawQuery = q.Encode()
	}
	applyCodexCredentialHeadersForTest(req, credential, requestBody)

	client := e.createHTTPClient(30*time.Second, req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Request failed: %v", err),
		}
		data, _ := json.Marshal(result)
		logger.Error("Test failed for %s: %v", endpoint.Name, err)
		return string(data)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to read response: %v", err),
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	if resp.StatusCode != http.StatusOK {
		result := map[string]interface{}{
			"success":    false,
			"statusCode": resp.StatusCode,
			"message":    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}
		data, _ := json.Marshal(result)
		logger.Error("Test failed for %s: HTTP %d", endpoint.Name, resp.StatusCode)
		return string(data)
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		result := map[string]interface{}{
			"success": true,
			"message": string(respBody),
		}
		data, _ := json.Marshal(result)
		logger.Info("Test successful for %s", endpoint.Name)
		return string(data)
	}

	var message string
	switch transformer {
	case "claude":
		if content, ok := responseData["content"].([]interface{}); ok && len(content) > 0 {
			if textBlock, ok := content[0].(map[string]interface{}); ok {
				if text, ok := textBlock["text"].(string); ok {
					message = text
				}
			}
		}
	case "openai":
		if choices, ok := responseData["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if msg, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := msg["content"].(string); ok {
						message = content
					}
				}
			}
		}
	case "gemini":
		if candidates, ok := responseData["candidates"].([]interface{}); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]interface{}); ok {
				if content, ok := candidate["content"].(map[string]interface{}); ok {
					if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
						if part, ok := parts[0].(map[string]interface{}); ok {
							if text, ok := part["text"].(string); ok {
								message = text
							}
						}
					}
				}
			}
		}
	}

	if message == "" {
		message = string(respBody)
	}

	result := map[string]interface{}{
		"success": true,
		"message": message,
	}
	data, _ := json.Marshal(result)
	logger.Info("Test successful for %s", endpoint.Name)
	return string(data)
}

// Remaining methods (TestEndpointLight, TestAllEndpointsZeroCost, FetchModels, etc.)
// will be added in the next part due to size constraints

// TestEndpointLight tests endpoint availability with minimal token consumption
func (e *EndpointService) TestEndpointLight(index int) string {
	endpoints := e.config.GetEndpoints()

	if index < 0 || index >= len(endpoints) {
		return e.testResult(false, "invalid_index", "models", fmt.Sprintf("Invalid endpoint index: %d", index))
	}

	endpoint := endpoints[index]
	logger.Info("Testing endpoint (light): %s (%s)", endpoint.Name, endpoint.APIUrl)

	apiKey, credential, err := e.resolveEndpointAuth(endpoint)
	if err != nil {
		return e.testResult(false, "invalid_key", "auth", fmt.Sprintf("Authentication unavailable: %v", err))
	}

	transformer := endpoint.Transformer
	if transformer == "" {
		transformer = "claude"
	}

	normalizedURL := normalizeAPIUrl(endpoint.APIUrl)
	if !strings.HasPrefix(normalizedURL, "http://") && !strings.HasPrefix(normalizedURL, "https://") {
		normalizedURL = "https://" + normalizedURL
	}
	isCodexOpenAI2 := isCodexOpenAI2Endpoint(transformer, normalizedURL)

	// Codex endpoints are validated by a minimal ping-style inference request only.
	if isCodexOpenAI2 {
		statusCode, minErr := e.testMinimalRequest(normalizedURL, apiKey, transformer, endpoint.Model, credential)
		if minErr == nil {
			return e.testResult(true, "ok", "minimal", "Minimal ping request successful")
		}
		if statusCode == 401 {
			return e.testResult(false, "invalid_key", "minimal", "Authentication failed: HTTP 401")
		}
		if statusCode == 403 {
			return e.testResult(false, "unknown", "minimal", "Upstream denied test request (HTTP 403)")
		}
		if statusCode == 405 {
			return e.testResult(false, "unknown", "minimal", "Method not allowed (may work in real client)")
		}
		return e.testResult(false, "error", "minimal", fmt.Sprintf("Test failed: %v", minErr))
	}

	authMode := config.NormalizeAuthMode(endpoint.AuthMode)
	if config.IsTokenPoolAuthMode(authMode) {
		// Token pool credentials are best validated by an actual minimal inference request.
		statusCode, minErr := e.testMinimalRequest(normalizedURL, apiKey, transformer, endpoint.Model, credential)
		if minErr == nil {
			return e.testResult(true, "ok", "minimal", "Minimal request successful")
		}
		if statusCode == 401 {
			return e.testResult(false, "invalid_key", "minimal", "Authentication failed: HTTP 401")
		}
		if statusCode == 403 {
			return e.testResult(false, "unknown", "minimal", "Upstream denied test request (HTTP 403)")
		}
		if statusCode == 405 {
			return e.testResult(false, "unknown", "minimal", "Method not allowed (may work in real client)")
		}
		return e.testResult(false, "error", "minimal", fmt.Sprintf("Test failed: %v", minErr))
	}

	// Step 1: Try models API
	statusCode, err := e.testModelsAPI(normalizedURL, apiKey, transformer)
	if err == nil {
		return e.testResult(true, "ok", "models", "Models API accessible")
	}
	if statusCode == 401 || statusCode == 403 {
		return e.testResult(false, "invalid_key", "models", fmt.Sprintf("Authentication failed: HTTP %d", statusCode))
	}

	// Step 2: Try token count (Claude) or billing API (OpenAI)
	if transformer == "claude" {
		statusCode, err = e.testTokenCountAPI(normalizedURL, apiKey)
		if err == nil {
			return e.testResult(true, "ok", "token_count", "Token count API accessible")
		}
		if statusCode == 401 || statusCode == 403 {
			return e.testResult(false, "invalid_key", "token_count", fmt.Sprintf("Authentication failed: HTTP %d", statusCode))
		}
	} else if transformer == "openai" || transformer == "openai2" {
		statusCode, err = e.testBillingAPI(normalizedURL, apiKey)
		if err == nil {
			return e.testResult(true, "ok", "billing", "Billing API accessible")
		}
		if statusCode == 401 || statusCode == 403 {
			return e.testResult(false, "invalid_key", "billing", fmt.Sprintf("Authentication failed: HTTP %d", statusCode))
		}
	}

	// Step 3: Minimal request (fallback)
	statusCode, err = e.testMinimalRequest(normalizedURL, apiKey, transformer, endpoint.Model, nil)
	if err == nil {
		return e.testResult(true, "ok", "minimal", "Minimal request successful")
	}
	if statusCode == 401 || statusCode == 403 {
		return e.testResult(false, "invalid_key", "minimal", fmt.Sprintf("Authentication failed: HTTP %d", statusCode))
	}
	if statusCode == 405 {
		return e.testResult(false, "unknown", "minimal", "Method not allowed (may work in real client)")
	}

	return e.testResult(false, "error", "minimal", fmt.Sprintf("Test failed: %v", err))
}

func (e *EndpointService) testResult(success bool, status, method, message string) string {
	result := map[string]interface{}{
		"success": success,
		"status":  status,
		"method":  method,
		"message": message,
	}
	data, _ := json.Marshal(result)
	return string(data)
}

// TestAllEndpointsZeroCost tests all endpoints using zero-cost methods only
// Uses concurrent testing with limited parallelism for optimal performance
func (e *EndpointService) TestAllEndpointsZeroCost() string {
	endpoints := e.config.GetEndpoints()
	results := make(map[string]string)
	mu := &sync.Mutex{}

	// Use semaphore to limit concurrent requests (max 15 concurrent)
	const maxConcurrent = 15
	semaphore := make(chan struct{}, maxConcurrent)
	wg := &sync.WaitGroup{}

	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(ep config.Endpoint) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Test the endpoint
			status := e.testSingleEndpointZeroCost(ep)

			// Store result
			mu.Lock()
			results[ep.Name] = status
			mu.Unlock()
		}(endpoint)
	}

	wg.Wait()
	data, _ := json.Marshal(results)
	return string(data)
}

// testSingleEndpointZeroCost tests a single endpoint using zero-cost methods
func (e *EndpointService) testSingleEndpointZeroCost(endpoint config.Endpoint) string {
	transformer := endpoint.Transformer
	if transformer == "" {
		transformer = "claude"
	}

	normalizedURL := normalizeAPIUrl(endpoint.APIUrl)
	if !strings.HasPrefix(normalizedURL, "http://") && !strings.HasPrefix(normalizedURL, "https://") {
		normalizedURL = "https://" + normalizedURL
	}

	status := "unknown"

	apiKey, credential, err := e.resolveEndpointAuth(endpoint)
	if err != nil {
		return "invalid_key"
	}
	if isCodexOpenAI2Endpoint(transformer, normalizedURL) {
		statusCode, minErr := e.testMinimalRequest(normalizedURL, apiKey, transformer, endpoint.Model, credential)
		if minErr == nil {
			return "ok"
		}
		if statusCode == 401 || statusCode == 403 {
			return "invalid_key"
		}
		return "unknown"
	}

	statusCode, err := e.testModelsAPI(normalizedURL, apiKey, transformer)
	if err == nil {
		status = "ok"
	} else if statusCode == 401 || statusCode == 403 {
		status = "invalid_key"
	} else {
		if transformer == "claude" {
			statusCode, err = e.testTokenCountAPI(normalizedURL, apiKey)
			if err == nil {
				status = "ok"
			} else if statusCode == 401 || statusCode == 403 {
				status = "invalid_key"
			}
		} else if transformer == "openai" || transformer == "openai2" {
			statusCode, err = e.testBillingAPI(normalizedURL, apiKey)
			if err == nil {
				status = "ok"
			} else if statusCode == 401 || statusCode == 403 {
				status = "invalid_key"
			}
		}
	}

	return status
}

func (e *EndpointService) testModelsAPI(apiUrl, apiKey, transformer string) (int, error) {
	var url string
	if transformer == "gemini" {
		url = fmt.Sprintf("%s/v1beta/models?key=%s", apiUrl, apiKey)
	} else {
		url = fmt.Sprintf("%s/v1/models", apiUrl)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	if transformer != "gemini" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := e.createHTTPClient(8*time.Second, req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("failed to read response")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return resp.StatusCode, fmt.Errorf("failed to parse response")
	}

	if data, ok := result["data"].([]interface{}); ok {
		if len(data) == 0 {
			return resp.StatusCode, fmt.Errorf("no models found")
		}
		return resp.StatusCode, nil
	}

	if models, ok := result["models"].([]interface{}); ok {
		if len(models) == 0 {
			return resp.StatusCode, fmt.Errorf("no models found")
		}
		return resp.StatusCode, nil
	}

	return resp.StatusCode, fmt.Errorf("unexpected response format")
}

func (e *EndpointService) testTokenCountAPI(apiUrl, apiKey string) (int, error) {
	url := fmt.Sprintf("%s/v1/messages/count_tokens", apiUrl)

	body, _ := json.Marshal(map[string]interface{}{
		"model": "claude-sonnet-4-5-20250929",
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
	})

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "token-counting-2024-11-01")

	client := e.createHTTPClient(8*time.Second, req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("failed to read response")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return resp.StatusCode, fmt.Errorf("failed to parse response")
	}

	if _, ok := result["input_tokens"]; !ok {
		return resp.StatusCode, fmt.Errorf("invalid response: no input_tokens")
	}

	return resp.StatusCode, nil
}

func (e *EndpointService) testBillingAPI(apiUrl, apiKey string) (int, error) {
	url := fmt.Sprintf("%s/v1/dashboard/billing/credit_grants", apiUrl)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := e.createHTTPClient(8*time.Second, req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("failed to read response")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return resp.StatusCode, fmt.Errorf("failed to parse response")
	}

	return resp.StatusCode, nil
}

func (e *EndpointService) testMinimalRequest(apiUrl, apiKey, transformer, model string, credential *storage.EndpointCredential) (int, error) {
	var reqURL string
	var body []byte
	var apiPath string

	switch transformer {
	case "claude":
		apiPath = "/v1/messages"
		if model == "" {
			model = "claude-sonnet-4-5-20250929"
		}
		body, _ = json.Marshal(map[string]interface{}{
			"model":      model,
			"max_tokens": 1,
			"messages":   []map[string]string{{"role": "user", "content": "Hi"}},
		})
	case "openai":
		apiPath = "/v1/chat/completions"
		if model == "" {
			model = "gpt-4-turbo"
		}
		body, _ = json.Marshal(map[string]interface{}{
			"model":      model,
			"max_tokens": 1,
			"messages":   []map[string]interface{}{{"role": "user", "content": "Hi"}},
		})
	case "openai2":
		apiPath = "/v1/responses"
		if model == "" && isCodexBackendAPIURL(apiUrl) {
			model = "gpt-5-codex"
		}
		if model == "" {
			model = "gpt-4-turbo"
		}
		// For Codex backend, build probe payload through the same OpenAI->Responses
		// conversion path used by runtime proxy requests, to avoid test/runtime drift.
		if isCodexBackendAPIURL(apiUrl) {
			probeOpenAIReq, _ := json.Marshal(map[string]interface{}{
				"model":      model,
				"stream":     false,
				"messages":   []map[string]interface{}{{"role": "user", "content": "ping"}},
				"max_tokens": 1,
			})
			converted, convErr := convert.OpenAIReqToOpenAI2(probeOpenAIReq, model)
			if convErr != nil {
				return 0, fmt.Errorf("failed to build codex probe payload: %w", convErr)
			}
			body = converted
		} else {
			body, _ = json.Marshal(map[string]interface{}{
				"model":             model,
				"max_output_tokens": 16,
				"input": []map[string]interface{}{
					{"type": "message", "role": "user", "content": []map[string]interface{}{{"type": "input_text", "text": "ping"}}},
				},
			})
		}
	case "gemini":
		if model == "" {
			model = "gemini-2.0-flash"
		}
		reqURL = fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", apiUrl, model, apiKey)
		body, _ = json.Marshal(map[string]interface{}{
			"contents":         []map[string]interface{}{{"parts": []map[string]string{{"text": "Hi"}}}},
			"generationConfig": map[string]int{"maxOutputTokens": 1},
		})
	default:
		return 0, fmt.Errorf("unsupported transformer: %s", transformer)
	}
	if reqURL == "" {
		reqURL = fmt.Sprintf("%s%s", apiUrl, normalizeEndpointPathForBaseURL(apiUrl, apiPath))
	}
	if transformer == "openai2" && isCodexBackendAPIURL(apiUrl) {
		body = ensureCodexResponsesProbePayload(body)
	}

	attempts := 1
	isCodexRequest := transformer == "openai2" && isCodexBackendAPIURL(apiUrl)
	if isCodexRequest {
		attempts = 5
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequest("POST", reqURL, bytes.NewReader(body))
		if err != nil {
			return 0, err
		}

		req.Header.Set("Content-Type", "application/json")
		if transformer == "claude" {
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		} else if transformer != "gemini" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
		applyCodexCredentialHeadersForTest(req, credential, body)

		timeout := 30 * time.Second
		if isCodexRequest {
			// Align with production proxy request transport stack to avoid protocol mismatch.
			timeout = 45 * time.Second
		}
		client := e.createHTTPClient(timeout, reqURL)
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if isCodexRequest && attempt < attempts && isTransientTestRequestError(err) {
				logger.Warn("Codex minimal test request transient failure (%d/%d): %v", attempt, attempts, err)
				time.Sleep(250 * time.Millisecond)
				continue
			}
			return 0, err
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			detail := strings.TrimSpace(string(respBody))
			if len(detail) > 400 {
				detail = detail[:400] + "..."
			}
			if detail == "" {
				return resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
			}
			return resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, detail)
		}
		resp.Body.Close()
		return http.StatusOK, nil
	}

	if lastErr != nil {
		return 0, lastErr
	}
	return 0, fmt.Errorf("minimal request failed")
}

func resolveNormalizedAPIURL(raw string) string {
	urlValue := normalizeAPIUrl(raw)
	if !strings.HasPrefix(urlValue, "http://") && !strings.HasPrefix(urlValue, "https://") {
		urlValue = "https://" + urlValue
	}
	return urlValue
}

func canonicalAPIURL(raw string) string {
	normalized := resolveNormalizedAPIURL(raw)
	parsed, err := url.Parse(normalized)
	if err != nil || parsed == nil {
		return strings.TrimSuffix(strings.ToLower(normalized), "/")
	}
	parsed.Host = strings.ToLower(strings.TrimSpace(parsed.Host))
	parsed.Scheme = strings.ToLower(strings.TrimSpace(parsed.Scheme))
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	return parsed.String()
}

func isCodexBackendAPIURL(raw string) bool {
	normalized := resolveNormalizedAPIURL(raw)
	parsed, err := url.Parse(normalized)
	if err != nil || parsed == nil {
		return false
	}
	cleanPath := path.Clean(strings.TrimSpace(parsed.Path))
	return strings.HasSuffix(cleanPath, "/backend-api/codex")
}

func isCodexOpenAI2Endpoint(transformer, rawURL string) bool {
	return strings.TrimSpace(transformer) == "openai2" && isCodexBackendAPIURL(rawURL)
}

func isTransientTestRequestError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "eof") ||
		strings.Contains(message, "timeout awaiting response headers") ||
		strings.Contains(message, "i/o timeout") ||
		strings.Contains(message, "malformed http response") ||
		strings.Contains(message, "http/1.x transport connection broken") ||
		strings.Contains(message, "connection reset by peer") ||
		strings.Contains(message, "broken pipe")
}

func ensureCodexResponsesProbePayload(payload []byte) []byte {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" || strings.HasPrefix(trimmed, "[") {
		return payload
	}

	var body map[string]interface{}
	if err := json.Unmarshal(payload, &body); err != nil {
		return payload
	}
	body["store"] = false
	body["stream"] = true
	if _, ok := body["instructions"]; !ok {
		body["instructions"] = ""
	}
	updated, err := json.Marshal(body)
	if err != nil {
		return payload
	}
	return updated
}

func normalizeEndpointPathForBaseURL(baseURL, apiPath string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed == nil {
		return apiPath
	}

	cleanPath := path.Clean(strings.TrimSpace(parsed.Path))
	if !strings.HasSuffix(cleanPath, "/backend-api/codex") {
		return apiPath
	}

	switch strings.TrimSpace(apiPath) {
	case "/v1/responses":
		return "/responses"
	case "/v1/responses/compact":
		return "/responses/compact"
	default:
		return apiPath
	}
}

func isCodexProviderType(providerType string) bool {
	p := strings.ToLower(strings.TrimSpace(providerType))
	return p == "" || p == "codex"
}

func isResponsesRequestPath(requestPath string) bool {
	trimmed := strings.TrimSpace(requestPath)
	return strings.HasSuffix(trimmed, "/responses") || strings.HasSuffix(trimmed, "/responses/compact")
}

func ensureHeader(headers http.Header, key, value string) {
	if headers == nil || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
		return
	}
	if strings.TrimSpace(headers.Get(key)) == "" {
		headers.Set(key, value)
	}
}

func applyCodexCredentialHeadersForTest(req *http.Request, credential *storage.EndpointCredential, payload []byte) {
	if req == nil || credential == nil {
		return
	}
	if !isCodexProviderType(credential.ProviderType) {
		return
	}
	if !isResponsesRequestPath(req.URL.Path) {
		return
	}

	ensureHeader(req.Header, "Version", codexTestClientVersion)
	ensureHeader(req.Header, "Session_id", uuid.NewString())
	ensureHeader(req.Header, "User-Agent", codexTestUserAgent)
	if isStreamingPayload(payload) {
		req.Header.Set("Accept", "text/event-stream")
	} else if len(payload) > 0 {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Originator", "codex_cli_rs")
	if accountID := strings.TrimSpace(credential.AccountID); accountID != "" {
		req.Header.Set("Chatgpt-Account-Id", accountID)
	}
}

func isStreamingPayload(payload []byte) bool {
	if len(payload) == 0 {
		return false
	}
	var req map[string]interface{}
	if err := json.Unmarshal(payload, &req); err != nil {
		return false
	}
	stream, ok := req["stream"].(bool)
	return ok && stream
}

func (e *EndpointService) resolveTokenPoolKeyForAPI(apiURL, transformer string) (string, error) {
	apiKey, _, err := e.resolveTokenPoolAuthForAPI(apiURL, transformer)
	return apiKey, err
}

func (e *EndpointService) resolveTokenPoolAuthForAPI(apiURL, transformer string) (string, *storage.EndpointCredential, error) {
	if e.storage == nil {
		return "", nil, fmt.Errorf("storage unavailable")
	}

	canonicalTarget := canonicalAPIURL(apiURL)
	targetTransformer := strings.TrimSpace(transformer)
	endpoints := e.config.GetEndpoints()

	foundCandidate := false
	var lastErr error
	for _, ep := range endpoints {
		if !config.IsTokenPoolAuthMode(ep.AuthMode) {
			continue
		}
		if canonicalAPIURL(ep.APIUrl) != canonicalTarget {
			continue
		}
		if targetTransformer != "" && strings.TrimSpace(ep.Transformer) != "" && strings.TrimSpace(ep.Transformer) != targetTransformer {
			continue
		}

		foundCandidate = true
		apiKey, cred, err := e.resolveEndpointAuth(ep)
		if err == nil && strings.TrimSpace(apiKey) != "" {
			return strings.TrimSpace(apiKey), cred, nil
		}
		lastErr = err
	}

	if !foundCandidate {
		return "", nil, fmt.Errorf("no token pool endpoint matched API URL")
	}
	if lastErr != nil {
		return "", nil, lastErr
	}
	return "", nil, fmt.Errorf("no usable token in token pool")
}

// FetchModels fetches available models from the API provider
func (e *EndpointService) FetchModels(apiUrl, apiKey, transformer string) string {
	logger.Info("Fetching models for transformer: %s", transformer)

	if transformer == "" {
		transformer = "claude"
	}

	normalizedAPIUrl := normalizeAPIUrl(apiUrl)
	if !strings.HasPrefix(normalizedAPIUrl, "http://") && !strings.HasPrefix(normalizedAPIUrl, "https://") {
		normalizedAPIUrl = "https://" + normalizedAPIUrl
	}
	var resolvedCredential *storage.EndpointCredential
	resolvedAPIKey := strings.TrimSpace(apiKey)
	if resolvedAPIKey == "" {
		var resolveErr error
		resolvedAPIKey, resolvedCredential, resolveErr = e.resolveTokenPoolAuthForAPI(normalizedAPIUrl, transformer)
		if resolveErr != nil {
			result := map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Unable to resolve token pool credential: %v", resolveErr),
				"models":  []string{},
			}
			data, _ := json.Marshal(result)
			return string(data)
		}
	}

	var models []string
	var err error

	switch transformer {
	case "claude":
		models, err = e.fetchOpenAIModels(normalizedAPIUrl, resolvedAPIKey, transformer, resolvedCredential)
	case "openai", "openai2":
		if transformer == "openai2" && isCodexBackendAPIURL(normalizedAPIUrl) {
			models, err = e.fetchCodexModels(normalizedAPIUrl, resolvedAPIKey, resolvedCredential)
			break
		}
		models, err = e.fetchOpenAIModels(normalizedAPIUrl, resolvedAPIKey, transformer, resolvedCredential)
	case "gemini":
		models, err = e.fetchGeminiModels(normalizedAPIUrl, resolvedAPIKey)
	default:
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Unsupported transformer: %s", transformer),
			"models":  []string{},
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"message": err.Error(),
			"models":  []string{},
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Found %d models", len(models)),
		"models":  models,
	}
	data, _ := json.Marshal(result)
	logger.Info("Fetched %d models for %s", len(models), transformer)
	return string(data)
}

func (e *EndpointService) fetchOpenAIModels(apiUrl, apiKey, transformer string, credential *storage.EndpointCredential) ([]string, error) {
	modelsPath := "/v1/models"
	if isCodexBackendAPIURL(apiUrl) {
		modelsPath = "/models"
	}
	url := fmt.Sprintf("%s%s", strings.TrimSuffix(apiUrl, "/"), modelsPath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Error("Failed to create request for %s: %v", url, err)
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	applyCodexCredentialHeadersForTest(req, credential, nil)
	logger.Debug("Fetching models from: %s (transformer=%s)", url, transformer)

	client := e.createHTTPClient(30*time.Second, req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Request failed for %s: %v", url, err)
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body for error details
		body, _ := io.ReadAll(resp.Body)
		errMsg := string(body)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "..."
		}
		logger.Error("Models API failed for %s: HTTP %d - %s", url, resp.StatusCode, errMsg)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, errMsg)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("Failed to parse models response from %s: %v", url, err)
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(result.Data) == 0 {
		logger.Warn("No models found in response from %s", url)
	}

	seen := make(map[string]bool)
	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		id := strings.TrimSpace(m.ID)
		if id != "" && !seen[id] {
			seen[id] = true
			models = append(models, id)
		}
	}

	logger.Debug("Successfully fetched %d models from %s", len(models), url)
	return models, nil
}

func (e *EndpointService) fetchCodexModels(apiURL, apiKey string, credential *storage.EndpointCredential) ([]string, error) {
	// Keep signature for compatibility with existing callers.
	_ = apiURL
	_ = apiKey
	_ = credential

	// Codex model listing is served from local registry.
	models := codexRegistryModels()
	if len(models) == 0 {
		return nil, fmt.Errorf("codex registry model list is empty")
	}
	logger.Debug("Using local Codex registry model list (%d models)", len(models))
	return models, nil
}

func codexRegistryModels() []string {
	return []string{
		"gpt-5",
		"gpt-5-codex",
		"gpt-5-codex-mini",
		"gpt-5.1",
		"gpt-5.1-codex",
		"gpt-5.1-codex-mini",
		"gpt-5.1-codex-max",
		"gpt-5.2",
		"gpt-5.2-codex",
		"gpt-5.3-codex",
		"gpt-5.3-codex-spark",
		"gpt-5.4",
	}
}

func (e *EndpointService) fetchGeminiModels(apiUrl, apiKey string) ([]string, error) {
	url := fmt.Sprintf("%s/v1beta/models?key=%s", apiUrl, apiKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Error("Failed to create request for %s: %v", apiUrl, err)
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	logger.Debug("Fetching Gemini models from: %s", apiUrl)

	client := e.createHTTPClient(30*time.Second, req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Request failed for %s: %v", apiUrl, err)
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := string(body)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "..."
		}
		logger.Error("Gemini Models API failed for %s: HTTP %d - %s", apiUrl, resp.StatusCode, errMsg)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, errMsg)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("Failed to parse Gemini models response from %s: %v", apiUrl, err)
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	models := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		name := m.Name
		if strings.HasPrefix(name, "models/") {
			name = strings.TrimPrefix(name, "models/")
		}
		models = append(models, name)
	}

	if len(models) == 0 {
		logger.Warn("No Gemini models found in response from %s", apiUrl)
	} else {
		logger.Debug("Successfully fetched %d Gemini models from %s", len(models), apiUrl)
	}

	return models, nil
}
