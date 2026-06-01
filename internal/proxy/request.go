package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/proxy"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/storage"
	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/cc"
	"github.com/lich0821/ccNexus/internal/transformer/cx/chat"
	"github.com/lich0821/ccNexus/internal/transformer/cx/responses"
)

const (
	codexClientVersion = "0.101.0"
	codexUserAgent     = "codex_cli_rs/0.101.0 (Mac OS 26.0.1; arm64) Apple_Terminal/464"
)

// prepareTransformerForClient creates transformer based on client format and endpoint.
// effectiveModel is the final model that should reach upstream after applying
// endpoint-level overrides or falling back to the original request model.
func prepareTransformerForClient(clientFormat ClientFormat, endpoint config.Endpoint, effectiveModel string) (transformer.Transformer, error) {
	endpointTransformer := endpoint.Transformer
	if endpointTransformer == "" {
		endpointTransformer = "claude"
	}

	switch clientFormat {
	case ClientFormatClaude:
		return prepareCCTransformer(endpoint, endpointTransformer, effectiveModel)
	case ClientFormatOpenAIChat:
		return prepareCxChatTransformer(endpoint, endpointTransformer, effectiveModel)
	case ClientFormatOpenAIResponses:
		return prepareCxRespTransformer(endpoint, endpointTransformer, effectiveModel)
	}

	return nil, fmt.Errorf("unsupported client format: %s", clientFormat)
}

// prepareCCTransformer creates transformer for Claude Code client
func prepareCCTransformer(endpoint config.Endpoint, endpointTransformer string, effectiveModel string) (transformer.Transformer, error) {
	switch endpointTransformer {
	case "claude":
		if effectiveModel != "" {
			logger.Debug("[%s] Using cc_claude with model override: %s", endpoint.Name, effectiveModel)
			return cc.NewClaudeTransformerWithModel(effectiveModel), nil
		}
		return cc.NewClaudeTransformer(), nil
	case "openai":
		return cc.NewOpenAITransformer(effectiveModel), nil
	case "openai2":
		return cc.NewOpenAI2Transformer(effectiveModel), nil
	case "gemini":
		return cc.NewGeminiTransformer(effectiveModel), nil
	default:
		return nil, fmt.Errorf("unsupported endpoint transformer: %s", endpointTransformer)
	}
}

// prepareCxChatTransformer creates transformer for Codex Chat API client
func prepareCxChatTransformer(endpoint config.Endpoint, endpointTransformer string, effectiveModel string) (transformer.Transformer, error) {
	switch endpointTransformer {
	case "claude":
		return chat.NewClaudeTransformer(effectiveModel), nil
	case "openai":
		return chat.NewOpenAITransformer(effectiveModel), nil
	case "openai2":
		return chat.NewOpenAI2Transformer(effectiveModel), nil
	case "gemini":
		return chat.NewGeminiTransformer(effectiveModel), nil
	default:
		return nil, fmt.Errorf("unsupported endpoint transformer for Codex Chat: %s", endpointTransformer)
	}
}

// prepareCxRespTransformer creates transformer for Codex Responses API client
func prepareCxRespTransformer(endpoint config.Endpoint, endpointTransformer string, effectiveModel string) (transformer.Transformer, error) {
	switch endpointTransformer {
	case "claude":
		return responses.NewClaudeTransformer(effectiveModel), nil
	case "openai":
		return responses.NewOpenAITransformer(effectiveModel), nil
	case "openai2":
		return responses.NewOpenAI2Transformer(effectiveModel), nil
	case "gemini":
		return responses.NewGeminiTransformer(effectiveModel), nil
	default:
		return nil, fmt.Errorf("unsupported endpoint transformer for Codex Responses: %s", endpointTransformer)
	}
}

// getTargetPath determines the target API path based on transformer name
func getTargetPath(originalPath string, endpoint config.Endpoint, transformedBody []byte, transformerName string, modelName string) string {
	switch transformerName {
	case "cc_claude", "cx_chat_claude", "cx_resp_claude":
		return "/v1/messages"
	case "cc_openai", "cx_chat_openai", "cx_resp_openai":
		return "/v1/chat/completions"
	case "cc_openai2", "cx_resp_openai2", "cx_chat_openai2":
		return "/v1/responses"
	case "cc_gemini", "cx_chat_gemini", "cx_resp_gemini":
		var geminiReq struct {
			Stream bool `json:"stream"`
		}
		json.Unmarshal(transformedBody, &geminiReq)
		model := strings.TrimSpace(modelName)
		if model == "" {
			model = strings.TrimSpace(endpoint.Model)
		}
		if geminiReq.Stream {
			return fmt.Sprintf("/v1beta/models/%s:streamGenerateContent", model)
		}
		return fmt.Sprintf("/v1beta/models/%s:generateContent", model)
	}
	return originalPath
}

// buildProxyRequest creates an HTTP request for the target API
func buildProxyRequest(r *http.Request, endpoint config.Endpoint, apiKey string, transformedBody []byte, transformerName string, modelName string, credential *storage.EndpointCredential) (*http.Request, error) {
	targetPath := getTargetPath(r.URL.Path, endpoint, transformedBody, transformerName, modelName)
	if targetPath == "" {
		targetPath = r.URL.Path
	}

	normalizedAPIUrl := normalizeAPIUrl(endpoint.APIUrl)
	targetPath = normalizeTargetPathForBaseURL(normalizedAPIUrl, targetPath)
	requestBody := transformedBody
	if isCodexBackendBaseURL(normalizedAPIUrl) && isResponsesPath(targetPath) {
		requestBody = ensureCodexResponsesPayload(requestBody)
	}
	targetURL := fmt.Sprintf("%s%s", normalizedAPIUrl, targetPath)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}

	// Copy headers (except Host and Accept-Encoding)
	for key, values := range r.Header {
		if key == "Host" || key == "Accept-Encoding" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Force gzip or no compression to avoid unsupported encodings (e.g., brotli)
	proxyReq.Header.Set("Accept-Encoding", "gzip, identity")

	// Set authentication based on transformer type
	switch transformerName {
	case "cc_openai", "cc_openai2", "cx_chat_openai", "cx_chat_openai2", "cx_resp_openai", "cx_resp_openai2":
		proxyReq.Header.Set("Authorization", "Bearer "+apiKey)
	case "cc_gemini", "cx_chat_gemini", "cx_resp_gemini":
		q := proxyReq.URL.Query()
		q.Set("key", apiKey)
		q.Set("alt", "sse")
		proxyReq.URL.RawQuery = q.Encode()
	default:
		// Claude endpoints
		proxyReq.Header.Set("x-api-key", apiKey)
		proxyReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	// Set Host header
	if parsedBase, err := url.Parse(normalizedAPIUrl); err == nil && strings.TrimSpace(parsedBase.Host) != "" {
		proxyReq.Header.Set("Host", parsedBase.Host)
	}
	applyCodexCredentialHeaders(proxyReq, credential, requestBody)

	return proxyReq, nil
}

func applyCodexCredentialHeaders(req *http.Request, credential *storage.EndpointCredential, payload []byte) {
	if req == nil || credential == nil {
		return
	}
	if !isCodexProviderType(credential.ProviderType) {
		return
	}
	if !isResponsesPath(req.URL.Path) {
		return
	}

	// Match Codex client headers for oauth credentials.
	ensureHeader(req.Header, "Version", codexClientVersion)
	ensureHeader(req.Header, "Session_id", uuid.NewString())
	ensureHeader(req.Header, "User-Agent", codexUserAgent)

	if isStreamingRequest(payload) {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Originator", "codex_cli_rs")
	if accountID := strings.TrimSpace(credential.AccountID); accountID != "" {
		req.Header.Set("Chatgpt-Account-Id", accountID)
	}
}

func ensureHeader(headers http.Header, key, value string) {
	if headers == nil || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
		return
	}
	if strings.TrimSpace(headers.Get(key)) == "" {
		headers.Set(key, value)
	}
}

func isResponsesPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasSuffix(trimmed, "/responses") || strings.HasSuffix(trimmed, "/responses/compact")
}

func isStreamingRequest(payload []byte) bool {
	var streamReq struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal(payload, &streamReq); err != nil {
		return false
	}
	return streamReq.Stream
}

func isCodexProviderType(providerType string) bool {
	p := strings.ToLower(strings.TrimSpace(providerType))
	return p == "" || p == "codex"
}

// normalizeTargetPathForBaseURL adjusts OpenAI Responses paths for Codex backend base URLs.
// This is endpoint URL compatibility handling and is independent from auth mode.
func normalizeTargetPathForBaseURL(baseURL, targetPath string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed == nil {
		return targetPath
	}

	cleanPath := path.Clean(strings.TrimSpace(parsed.Path))
	isCodexBackend := strings.HasSuffix(cleanPath, "/backend-api/codex")
	if !isCodexBackend {
		return targetPath
	}

	switch strings.TrimSpace(targetPath) {
	case "/v1/responses":
		return "/responses"
	case "/v1/responses/compact":
		return "/responses/compact"
	default:
		return targetPath
	}
}

func isCodexBackendBaseURL(baseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed == nil {
		return false
	}
	cleanPath := path.Clean(strings.TrimSpace(parsed.Path))
	return strings.HasSuffix(cleanPath, "/backend-api/codex")
}

func ensureCodexResponsesPayload(payload []byte) []byte {
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

func overrideModelInPayload(payload []byte, model string) []byte {
	if strings.TrimSpace(model) == "" {
		return payload
	}
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" || strings.HasPrefix(trimmed, "[") {
		return payload
	}

	var body map[string]interface{}
	if err := json.Unmarshal(payload, &body); err != nil {
		return payload
	}
	body["model"] = model
	updated, err := json.Marshal(body)
	if err != nil {
		return payload
	}
	return updated
}

// sendRequest sends the HTTP request and returns the response
func sendRequest(ctx context.Context, proxyReq *http.Request, httpClient *http.Client, cfg *config.Config) (*http.Response, error) {
	proxyReq = proxyReq.WithContext(ctx)

	proxyURL := resolveProxyURLForRequest(cfg, proxyReq.URL)
	// Apply proxy if configured
	if strings.TrimSpace(proxyURL) != "" {
		// Clone the client and replace transport for this request
		clientWithProxy := &http.Client{
			Timeout: httpClient.Timeout,
		}

		transport, err := CreateProxyTransport(proxyURL)
		if err != nil {
			logger.Warn("Failed to create proxy transport: %v, using direct connection", err)
			clientWithProxy.Transport = httpClient.Transport
		} else {
			clientWithProxy.Transport = transport
		}

		return clientWithProxy.Do(proxyReq)
	}

	return httpClient.Do(proxyReq)
}

func resolveProxyURLForRequest(cfg *config.Config, targetURL *url.URL) string {
	if cfg == nil {
		return ""
	}
	if isCodexRequestURL(targetURL) {
		if codexProxy := cfg.GetCodexProxy(); codexProxy != nil && strings.TrimSpace(codexProxy.URL) != "" {
			return codexProxy.URL
		}
	}
	if proxyCfg := cfg.GetProxy(); proxyCfg != nil && strings.TrimSpace(proxyCfg.URL) != "" {
		return proxyCfg.URL
	}
	return ""
}

func isCodexRequestURL(targetURL *url.URL) bool {
	if targetURL == nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(targetURL.Host))
	if host != "chatgpt.com" {
		return false
	}
	cleanPath := path.Clean(strings.TrimSpace(targetURL.Path))
	return strings.Contains(cleanPath, "/backend-api/codex")
}

// CreateProxyTransport creates an http.Transport with proxy support
func CreateProxyTransport(proxyURL string) (*http.Transport, error) {
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	transport := &http.Transport{
		MaxIdleConns:           100,
		MaxIdleConnsPerHost:    10,
		IdleConnTimeout:        90 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
		ResponseHeaderTimeout:  90 * time.Second,
		WriteBufferSize:        128 * 1024, // 128KB write buffer for large SSE streams
		ReadBufferSize:         128 * 1024, // 128KB read buffer for large SSE streams
		MaxResponseHeaderBytes: 64 * 1024,  // 64KB max response headers
	}

	switch parsed.Scheme {
	case "socks5", "socks5h":
		auth := &proxy.Auth{}
		if parsed.User != nil {
			auth.User = parsed.User.Username()
			auth.Password, _ = parsed.User.Password()
		} else {
			auth = nil
		}
		dialer, err := proxy.SOCKS5("tcp", parsed.Host, auth, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
		}
		transport.Dial = dialer.Dial
	case "http", "https":
		transport.Proxy = http.ProxyURL(parsed)
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", parsed.Scheme)
	}

	return transport, nil
}
