package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/storage"
	"github.com/lich0821/ccNexus/internal/transformer"
)

type proxyRequestContext struct {
	httpRequest                 *http.Request
	bodyBytes                   []byte
	clientFormat                ClientFormat
	streamRequested             bool
	requestModel                string
	requestStart                time.Time
	requestBytes                int
	endpoints                   []config.Endpoint
	specifiedEndpoint           *config.Endpoint
	modelOverride               string
	useSpecificEndpoint         bool
	refreshedCredentialAttempts map[int64]bool
}

type endpointAttempt struct {
	endpoint           config.Endpoint
	authMode           string
	apiKey             string
	credentialID       int64
	selectedCredential *storage.EndpointCredential
	transformer        transformer.Transformer
	transformerName    string
	transformedBody    []byte
	modelName          string
	thinkingEnabled    bool
	proxyRequest       *http.Request
	response           *http.Response
}

type attemptResult int

const (
	attemptResultDone attemptResult = iota
	attemptResultRetrySameEndpoint
	attemptResultRetryNextEndpoint
)

func (p *Proxy) handleProxyRequest(w http.ResponseWriter, r *http.Request) {
	reqCtx, err := p.newProxyRequestContext(w, r)
	if err != nil {
		return
	}

	maxRetries := p.computeMaxRetries(reqCtx.endpoints)
	endpointAttempts := 0
	lastEndpointName := ""

	for retry := 0; retry < maxRetries; retry++ {
		endpoint := p.nextEndpointForRequest(reqCtx)
		if endpoint.Name == "" {
			http.Error(w, "No enabled endpoints available", http.StatusServiceUnavailable)
			return
		}

		if lastEndpointName != "" && lastEndpointName != endpoint.Name {
			endpointAttempts = 0
		}
		lastEndpointName = endpoint.Name
		endpointAttempts++

		attempt := &endpointAttempt{endpoint: endpoint}
		result := p.runEndpointAttempt(w, reqCtx, attempt)
		if result == attemptResultDone {
			return
		}

		if result == attemptResultRetrySameEndpoint {
			endpointAttempts = 0
			continue
		}

		if endpointAttempts >= 2 && !reqCtx.useSpecificEndpoint {
			p.rotateEndpoint()
			endpointAttempts = 0
		}
	}

	http.Error(w, "All endpoints failed", http.StatusServiceUnavailable)
}

func (p *Proxy) newProxyRequestContext(w http.ResponseWriter, r *http.Request) (*proxyRequestContext, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return nil, err
	}
	defer r.Body.Close()

	clientFormat := detectClientFormat(r.URL.Path)
	logger.DebugLog("=== Proxy Request ===")
	logger.DebugLog("Method: %s, Path: %s, ClientFormat: %s", r.Method, r.URL.Path, clientFormat)
	logger.DebugLog("Request Body: %s", string(bodyBytes))

	var streamReq struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	_ = json.Unmarshal(bodyBytes, &streamReq)

	endpoints := p.getEnabledEndpoints()
	if len(endpoints) == 0 {
		logger.Error("No enabled endpoints available")
		http.Error(w, "No enabled endpoints configured", http.StatusServiceUnavailable)
		return nil, errNoEnabledEndpoints
	}

	specifiedEndpoint, modelOverride, resolveErr := p.resolver.ResolveEndpoint(r, bodyBytes)
	if resolveErr != nil {
		logger.Warn("端点解析失败: %v", resolveErr)
		writeInvalidRequestError(w, resolveErr.Error())
		return nil, resolveErr
	}

	useSpecificEndpoint := specifiedEndpoint != nil
	if useSpecificEndpoint {
		logger.Debug("[Resolver] 使用指定端点: %s", specifiedEndpoint.Name)
	}

	return &proxyRequestContext{
		httpRequest:                 r,
		bodyBytes:                   bodyBytes,
		clientFormat:                clientFormat,
		streamRequested:             streamReq.Stream,
		requestModel:                strings.TrimSpace(streamReq.Model),
		requestStart:                time.Now(),
		requestBytes:                len(bodyBytes),
		endpoints:                   endpoints,
		specifiedEndpoint:           specifiedEndpoint,
		modelOverride:               modelOverride,
		useSpecificEndpoint:         useSpecificEndpoint,
		refreshedCredentialAttempts: make(map[int64]bool),
	}, nil
}

func (p *Proxy) nextEndpointForRequest(reqCtx *proxyRequestContext) config.Endpoint {
	if reqCtx.useSpecificEndpoint && reqCtx.specifiedEndpoint != nil {
		return *reqCtx.specifiedEndpoint
	}
	return p.getCurrentEndpoint()
}

func (p *Proxy) runEndpointAttempt(w http.ResponseWriter, reqCtx *proxyRequestContext, attempt *endpointAttempt) attemptResult {
	p.markRequestActive(attempt.endpoint.Name)

	if result := p.prepareEndpointAttempt(reqCtx, attempt); result != attemptResultDone {
		p.markRequestInactive(attempt.endpoint.Name)
		return result
	}

	p.logUpstreamRequest(reqCtx, attempt)
	resp, err := sendRequest(p.getEndpointContext(attempt.endpoint.Name), attempt.proxyRequest, p.httpClient, p.config)
	if err != nil {
		return p.handleSendError(err, attempt)
	}
	attempt.response = resp

	return p.handleAttemptResponse(w, reqCtx, attempt)
}

func (p *Proxy) prepareEndpointAttempt(reqCtx *proxyRequestContext, attempt *endpointAttempt) attemptResult {
	if result := p.resolveAttemptAuth(reqCtx, attempt); result != attemptResultDone {
		return result
	}

	trans, err := prepareTransformerForClient(reqCtx.clientFormat, attempt.endpoint)
	if err != nil {
		logger.Error("[%s] %v", attempt.endpoint.Name, err)
		p.stats.RecordError(attempt.endpoint.Name)
		return attemptResultRetryNextEndpoint
	}
	attempt.transformer = trans
	attempt.transformerName = trans.Name()

	transformedBody, err := trans.TransformRequest(reqCtx.bodyBytes)
	if err != nil {
		logger.Error("[%s] Failed to transform request: %v", attempt.endpoint.Name, err)
		p.stats.RecordError(attempt.endpoint.Name)
		return attemptResultRetryNextEndpoint
	}

	logger.DebugLog("[%s] Transformer: %s", attempt.endpoint.Name, attempt.transformerName)
	logger.DebugLog("[%s] Transformed Request: %s", attempt.endpoint.Name, string(transformedBody))

	if reqCtx.modelOverride != "" {
		transformedBody = overrideModelInPayload(transformedBody, reqCtx.modelOverride)
		logger.DebugLog("[%s] 应用模型覆盖后的请求: %s", attempt.endpoint.Name, string(transformedBody))
	}

	cleanedBody, err := cleanIncompleteToolCalls(transformedBody)
	if err != nil {
		logger.Warn("[%s] Failed to clean tool calls: %v", attempt.endpoint.Name, err)
		cleanedBody = transformedBody
	}
	if config.NormalizeAuthMode(attempt.endpoint.AuthMode) == config.AuthModeCodexTokenPool {
		cleanedBody = overrideModelInPayload(cleanedBody, attempt.endpoint.Model)
	}
	attempt.transformedBody = cleanedBody
	attempt.modelName = resolveAttemptModelName(reqCtx, attempt)
	attempt.thinkingEnabled = detectThinkingEnabled(attempt.transformerName, attempt.transformedBody)

	proxyReq, err := buildProxyRequest(reqCtx.httpRequest, attempt.endpoint, attempt.apiKey, attempt.transformedBody, attempt.transformerName, attempt.selectedCredential)
	if err != nil {
		logger.Error("[%s] Failed to create request: %v", attempt.endpoint.Name, err)
		p.stats.RecordError(attempt.endpoint.Name)
		return attemptResultRetryNextEndpoint
	}
	attempt.proxyRequest = proxyReq

	return attemptResultDone
}

func (p *Proxy) resolveAttemptAuth(reqCtx *proxyRequestContext, attempt *endpointAttempt) attemptResult {
	attempt.authMode = config.NormalizeAuthMode(attempt.endpoint.AuthMode)
	attempt.apiKey = strings.TrimSpace(attempt.endpoint.APIKey)

	if config.IsTokenPoolAuthMode(attempt.authMode) {
		credential, err := p.selectCredential(attempt.endpoint.Name)
		if err != nil {
			logger.Warn("[%s] Failed to select token pool credential: %v", attempt.endpoint.Name, err)
			p.stats.RecordError(attempt.endpoint.Name)
			return attemptResultRetryNextEndpoint
		}
		if credential == nil || strings.TrimSpace(credential.AccessToken) == "" {
			logger.Warn("[%s] No usable token in token pool", attempt.endpoint.Name)
			p.stats.RecordError(attempt.endpoint.Name)
			return attemptResultRetryNextEndpoint
		}

		attempt.selectedCredential = credential
		if shouldTryCredentialRefresh(credential, time.Now().UTC()) {
			refreshed, refreshErr := p.refreshCredential(attempt.endpoint, credential)
			if refreshErr != nil {
				logger.Warn("[%s] Preflight credential refresh failed (id=%d): %v", attempt.endpoint.Name, credential.ID, refreshErr)
			} else {
				attempt.selectedCredential = refreshed
				reqCtx.refreshedCredentialAttempts[refreshed.ID] = true
			}
		}

		attempt.apiKey = strings.TrimSpace(credential.AccessToken)
		if attempt.selectedCredential != nil {
			attempt.apiKey = strings.TrimSpace(attempt.selectedCredential.AccessToken)
			attempt.credentialID = attempt.selectedCredential.ID
		}
		return attemptResultDone
	}

	if attempt.apiKey == "" {
		logger.Warn("[%s] API key mode but apiKey is empty", attempt.endpoint.Name)
		p.stats.RecordError(attempt.endpoint.Name)
		return attemptResultRetryNextEndpoint
	}

	return attemptResultDone
}

func (p *Proxy) logUpstreamRequest(reqCtx *proxyRequestContext, attempt *endpointAttempt) {
	proxyLabel := strings.TrimSpace(resolveProxyURLForRequest(p.config, attempt.proxyRequest.URL))
	action := "Requesting"
	if reqCtx.streamRequested {
		action = "Streaming"
	}
	if proxyLabel == "" {
		logger.Debug("[%s] %s %s %d", attempt.endpoint.Name, action, attempt.modelName, reqCtx.requestBytes)
		return
	}
	logger.Debug("[%s] %s %s %d %s", attempt.endpoint.Name, action, attempt.modelName, reqCtx.requestBytes, proxyLabel)
}

func (p *Proxy) handleSendError(err error, attempt *endpointAttempt) attemptResult {
	logger.Error("[%s] Request failed: %v", attempt.endpoint.Name, err)
	p.markRequestInactive(attempt.endpoint.Name)
	if isTransientNetworkError(err) {
		logger.Warn("[%s] Transient network error, retrying same endpoint: %v", attempt.endpoint.Name, err)
		time.Sleep(300 * time.Millisecond)
		return attemptResultRetrySameEndpoint
	}
	p.markCredentialFailure(attempt.credentialID, 0, err.Error())
	p.recordCredentialUsage(attempt.credentialID, attempt.endpoint.Name, 0, 1, 0, 0)
	p.stats.RecordError(attempt.endpoint.Name)
	return attemptResultRetryNextEndpoint
}

func (p *Proxy) handleAttemptResponse(w http.ResponseWriter, reqCtx *proxyRequestContext, attempt *endpointAttempt) attemptResult {
	resp := attempt.response
	if resp.StatusCode == http.StatusOK {
		p.captureCodexRateLimitsFromHeaders(attempt.endpoint, attempt.credentialID, resp.Header)
	}

	if resp.StatusCode == http.StatusOK && !reqCtx.streamRequested && shouldAggregateCodexStreaming(attempt.endpoint, attempt.transformerName) {
		return p.handleAggregatedStreamingSuccess(w, reqCtx, attempt)
	}

	isStreaming := shouldHandleAsStreamingResponse(resp.Header.Get("Content-Type"), reqCtx.streamRequested, attempt.endpoint, attempt.transformerName)
	if resp.StatusCode == http.StatusOK && isStreaming {
		inputTokens, outputTokens, outputText := p.handleStreamingResponse(w, resp, attempt.endpoint, attempt.transformer, attempt.transformerName, attempt.thinkingEnabled, attempt.modelName, reqCtx.bodyBytes, attempt.credentialID)
		p.finishSuccessfulAttempt(reqCtx, attempt, inputTokens, outputTokens, outputText)
		return attemptResultDone
	}

	if resp.StatusCode == http.StatusOK {
		inputTokens, outputTokens, err := p.handleNonStreamingResponse(w, resp, attempt.endpoint, attempt.transformer)
		if err == nil {
			p.finishSuccessfulAttempt(reqCtx, attempt, inputTokens, outputTokens, "")
			return attemptResultDone
		}
	}

	if shouldRetry(resp.StatusCode) {
		return p.handleRetryableStatus(resp, attempt)
	}

	return p.handleFinalStatus(w, reqCtx, attempt)
}

func (p *Proxy) handleAggregatedStreamingSuccess(w http.ResponseWriter, reqCtx *proxyRequestContext, attempt *endpointAttempt) attemptResult {
	inputTokens, outputTokens, outputText, err := p.handleStreamingAsNonStreaming(w, attempt.response, attempt.endpoint, attempt.transformer, attempt.credentialID)
	if err == nil {
		p.finishSuccessfulAttempt(reqCtx, attempt, inputTokens, outputTokens, outputText)
		return attemptResultDone
	}

	logger.Warn("[%s] Failed to aggregate streaming response as non-stream: %v", attempt.endpoint.Name, err)
	p.markCredentialFailure(attempt.credentialID, 0, err.Error())
	p.recordCredentialUsage(attempt.credentialID, attempt.endpoint.Name, 0, 1, 0, 0)
	p.stats.RecordError(attempt.endpoint.Name)
	p.markRequestInactive(attempt.endpoint.Name)
	return attemptResultRetryNextEndpoint
}

func (p *Proxy) finishSuccessfulAttempt(reqCtx *proxyRequestContext, attempt *endpointAttempt, inputTokens, outputTokens int, outputText string) {
	if inputTokens == 0 || outputTokens == 0 {
		inputTokens, outputTokens = p.estimateTokens(reqCtx.bodyBytes, outputText, inputTokens, outputTokens, attempt.endpoint.Name)
	}
	p.stats.RecordRequest(attempt.endpoint.Name)
	p.stats.RecordTokens(attempt.endpoint.Name, inputTokens, outputTokens)
	p.recordCredentialUsage(attempt.credentialID, attempt.endpoint.Name, 1, 0, inputTokens, outputTokens)
	p.markCredentialSuccess(attempt.credentialID)
	p.markRequestInactive(attempt.endpoint.Name)
	if p.onEndpointSuccess != nil {
		p.onEndpointSuccess(attempt.endpoint.Name)
	}
	totalElapsed := time.Since(reqCtx.requestStart).Round(time.Millisecond)
	logger.Debug("[%s] Requested tokens=%d/%d latency=%s cred_id=%d", attempt.endpoint.Name, inputTokens, outputTokens, totalElapsed, attempt.credentialID)
}

func (p *Proxy) handleRetryableStatus(resp *http.Response, attempt *endpointAttempt) attemptResult {
	errBody := readResponseBody(resp)
	errMsg := truncateString(string(errBody), 200)
	logger.Warn("[%s] Request failed %d: %s", attempt.endpoint.Name, resp.StatusCode, errMsg)
	logger.DebugLog("[%s] Request failed %d: %s", attempt.endpoint.Name, resp.StatusCode, errMsg)
	p.markCredentialFailure(attempt.credentialID, resp.StatusCode, errMsg)
	p.recordCredentialUsage(attempt.credentialID, attempt.endpoint.Name, 0, 1, 0, 0)
	p.stats.RecordError(attempt.endpoint.Name)
	p.markRequestInactive(attempt.endpoint.Name)
	return attemptResultRetryNextEndpoint
}

func (p *Proxy) handleFinalStatus(w http.ResponseWriter, reqCtx *proxyRequestContext, attempt *endpointAttempt) attemptResult {
	resp := attempt.response
	respBody := readResponseBody(resp)
	skipCredentialPenalty := false

	if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && attempt.credentialID > 0 {
		errMsg := truncateString(string(respBody), 500)
		if !shouldTreatCredentialAuthFailure(resp.StatusCode, errMsg) {
			skipCredentialPenalty = true
			logger.Warn("[%s] Upstream %d looks like route/gateway denial, skipping credential invalidation", attempt.endpoint.Name, resp.StatusCode)
		}
		if !skipCredentialPenalty {
			if p.tryRefreshAfterAuthFailure(reqCtx, attempt, resp.StatusCode) {
				p.markRequestInactive(attempt.endpoint.Name)
				return attemptResultRetrySameEndpoint
			}
			p.markCredentialFailure(attempt.credentialID, resp.StatusCode, errMsg)
			p.recordCredentialUsage(attempt.credentialID, attempt.endpoint.Name, 0, 1, 0, 0)
			p.stats.RecordError(attempt.endpoint.Name)
			p.markRequestInactive(attempt.endpoint.Name)
			logger.Warn("[%s] Credential auth failed (%d), retrying with next token", attempt.endpoint.Name, resp.StatusCode)
			return attemptResultRetrySameEndpoint
		}
		p.stats.RecordError(attempt.endpoint.Name)
	}

	p.markRequestInactive(attempt.endpoint.Name)
	if resp.StatusCode != http.StatusOK {
		errMsg := truncateString(string(respBody), 500)
		if resp.StatusCode == http.StatusBadRequest &&
			strings.Contains(errMsg, "api.responses.write") &&
			strings.Contains(attempt.transformerName, "openai2") {
			logger.Warn("[%s] Upstream rejected /v1/responses scope (api.responses.write). Try transformer=openai (chat/completions) for this token.", attempt.endpoint.Name)
		}
		if skipCredentialPenalty {
			p.markCredentialFailure(attempt.credentialID, 0, errMsg)
		} else {
			p.markCredentialFailure(attempt.credentialID, resp.StatusCode, errMsg)
		}
		p.recordCredentialUsage(attempt.credentialID, attempt.endpoint.Name, 0, 1, 0, 0)
		logger.Warn("[%s] Response %d: %s", attempt.endpoint.Name, resp.StatusCode, errMsg)
		logger.DebugLog("[%s] Response %d: %s", attempt.endpoint.Name, resp.StatusCode, errMsg)
	}

	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
	return attemptResultDone
}

func (p *Proxy) tryRefreshAfterAuthFailure(reqCtx *proxyRequestContext, attempt *endpointAttempt, statusCode int) bool {
	if attempt.selectedCredential == nil ||
		!isCodexProviderType(attempt.selectedCredential.ProviderType) ||
		strings.TrimSpace(attempt.selectedCredential.RefreshToken) == "" ||
		reqCtx.refreshedCredentialAttempts[attempt.credentialID] {
		return false
	}

	reqCtx.refreshedCredentialAttempts[attempt.credentialID] = true
	refreshed, refreshErr := p.refreshCredential(attempt.endpoint, attempt.selectedCredential)
	if refreshErr == nil {
		logger.Info("[%s] Credential refreshed after %d, retrying with updated token (id=%d)", attempt.endpoint.Name, statusCode, attempt.credentialID)
		if refreshed != nil && refreshed.ID > 0 {
			reqCtx.refreshedCredentialAttempts[refreshed.ID] = true
		}
		return true
	}
	logger.Warn("[%s] Credential refresh failed after %d (id=%d): %v", attempt.endpoint.Name, statusCode, attempt.credentialID, refreshErr)
	return false
}

func resolveAttemptModelName(reqCtx *proxyRequestContext, attempt *endpointAttempt) string {
	if reqCtx.modelOverride != "" {
		logger.Debug("[%s] 使用模型覆盖值: %s", attempt.endpoint.Name, reqCtx.modelOverride)
		return reqCtx.modelOverride
	}
	if reqCtx.requestModel == "" || (attempt.authMode == config.AuthModeCodexTokenPool && strings.TrimSpace(attempt.endpoint.Model) != "") {
		return attempt.endpoint.Model
	}
	return reqCtx.requestModel
}

func detectThinkingEnabled(transformerName string, transformedBody []byte) bool {
	if !strings.Contains(transformerName, "openai") {
		return false
	}
	var openaiReq map[string]interface{}
	if err := json.Unmarshal(transformedBody, &openaiReq); err != nil {
		return false
	}
	enable, _ := openaiReq["enable_thinking"].(bool)
	return enable
}

func writeInvalidRequestError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	errorResp := map[string]interface{}{
		"error": map[string]interface{}{
			"type":    "invalid_request_error",
			"message": message,
		},
	}
	jsonBytes, err := json.Marshal(errorResp)
	if err == nil {
		_, _ = w.Write(jsonBytes)
	}
}

func readResponseBody(resp *http.Response) []byte {
	defer resp.Body.Close()
	if resp.Header.Get("Content-Encoding") == "gzip" {
		body, _ := decompressGzip(resp.Body)
		return body
	}
	body, _ := io.ReadAll(resp.Body)
	return body
}

func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	for key, values := range resp.Header {
		if key == "Content-Encoding" || key == "Content-Length" {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
}

func truncateString(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

var errNoEnabledEndpoints = io.EOF
