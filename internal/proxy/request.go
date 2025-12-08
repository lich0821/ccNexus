package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/cc"
	"github.com/lich0821/ccNexus/internal/transformer/cx/chat"
	"github.com/lich0821/ccNexus/internal/transformer/cx/responses"
)

// prepareTransformerForClient creates transformer based on client format and endpoint
func prepareTransformerForClient(clientFormat ClientFormat, endpoint config.Endpoint) (transformer.Transformer, error) {
	endpointTransformer := endpoint.Transformer
	if endpointTransformer == "" {
		endpointTransformer = "claude"
	}

	switch clientFormat {
	case ClientFormatClaude:
		return prepareCCTransformer(endpoint, endpointTransformer)
	case ClientFormatOpenAIChat:
		return prepareCxChatTransformer(endpoint, endpointTransformer)
	case ClientFormatOpenAIResponses:
		return prepareCxRespTransformer(endpoint, endpointTransformer)
	}

	return nil, fmt.Errorf("unsupported client format: %s", clientFormat)
}

// prepareCCTransformer creates transformer for Claude Code client
func prepareCCTransformer(endpoint config.Endpoint, endpointTransformer string) (transformer.Transformer, error) {
	switch endpointTransformer {
	case "claude":
		if endpoint.Model != "" {
			logger.Debug("[%s] Using cc_claude with model override: %s", endpoint.Name, endpoint.Model)
			return cc.NewClaudeTransformerWithModel(endpoint.Model), nil
		}
		return cc.NewClaudeTransformer(), nil
	case "openai":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("OpenAI transformer requires model field")
		}
		return cc.NewOpenAITransformer(endpoint.Model), nil
	case "openai2":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("OpenAI2 transformer requires model field")
		}
		return cc.NewOpenAI2Transformer(endpoint.Model), nil
	case "gemini":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("Gemini transformer requires model field")
		}
		return cc.NewGeminiTransformer(endpoint.Model), nil
	default:
		return nil, fmt.Errorf("unsupported endpoint transformer: %s", endpointTransformer)
	}
}

// prepareCxChatTransformer creates transformer for Codex Chat API client
func prepareCxChatTransformer(endpoint config.Endpoint, endpointTransformer string) (transformer.Transformer, error) {
	switch endpointTransformer {
	case "claude":
		model := endpoint.Model
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
		return chat.NewClaudeTransformer(model), nil
	case "openai":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("OpenAI transformer requires model field")
		}
		return chat.NewOpenAITransformer(endpoint.Model), nil
	case "openai2":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("OpenAI2 transformer requires model field")
		}
		return chat.NewOpenAI2Transformer(endpoint.Model), nil
	case "gemini":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("Gemini transformer requires model field")
		}
		return chat.NewGeminiTransformer(endpoint.Model), nil
	default:
		return nil, fmt.Errorf("unsupported endpoint transformer for Codex Chat: %s", endpointTransformer)
	}
}

// prepareCxRespTransformer creates transformer for Codex Responses API client
func prepareCxRespTransformer(endpoint config.Endpoint, endpointTransformer string) (transformer.Transformer, error) {
	switch endpointTransformer {
	case "claude":
		model := endpoint.Model
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
		return responses.NewClaudeTransformer(model), nil
	case "openai":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("OpenAI transformer requires model field")
		}
		return responses.NewOpenAITransformer(endpoint.Model), nil
	case "openai2":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("OpenAI2 transformer requires model field")
		}
		return responses.NewOpenAI2Transformer(endpoint.Model), nil
	case "gemini":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("Gemini transformer requires model field")
		}
		return responses.NewGeminiTransformer(endpoint.Model), nil
	default:
		return nil, fmt.Errorf("unsupported endpoint transformer for Codex Responses: %s", endpointTransformer)
	}
}

// getTargetPath determines the target API path based on transformer name
func getTargetPath(originalPath string, endpoint config.Endpoint, transformedBody []byte, transformerName string) string {
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
		if geminiReq.Stream {
			return fmt.Sprintf("/v1beta/models/%s:streamGenerateContent", endpoint.Model)
		}
		return fmt.Sprintf("/v1beta/models/%s:generateContent", endpoint.Model)
	}
	return originalPath
}

// buildProxyRequest creates an HTTP request for the target API
func buildProxyRequest(r *http.Request, endpoint config.Endpoint, transformedBody []byte, transformerName string) (*http.Request, error) {
	targetPath := getTargetPath(r.URL.Path, endpoint, transformedBody, transformerName)
	if targetPath == "" {
		targetPath = r.URL.Path
	}

	normalizedAPIUrl := normalizeAPIUrl(endpoint.APIUrl)
	targetURL := fmt.Sprintf("%s%s", normalizedAPIUrl, targetPath)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(transformedBody))
	if err != nil {
		return nil, err
	}

	// Copy headers (except Host)
	for key, values := range r.Header {
		if key == "Host" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Set authentication based on transformer type
	switch transformerName {
	case "cc_openai", "cc_openai2", "cx_chat_openai", "cx_chat_openai2", "cx_resp_openai", "cx_resp_openai2":
		proxyReq.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
	case "cc_gemini", "cx_chat_gemini", "cx_resp_gemini":
		q := proxyReq.URL.Query()
		q.Set("key", endpoint.APIKey)
		q.Set("alt", "sse")
		proxyReq.URL.RawQuery = q.Encode()
	default:
		// Claude endpoints
		proxyReq.Header.Set("x-api-key", endpoint.APIKey)
		proxyReq.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
	}

	// Set Host header
	hostOnly := strings.TrimPrefix(strings.TrimPrefix(normalizedAPIUrl, "https://"), "http://")
	proxyReq.Header.Set("Host", hostOnly)

	return proxyReq, nil
}

// sendRequest sends the HTTP request and returns the response
func sendRequest(proxyReq *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: 300 * time.Second,
	}
	return client.Do(proxyReq)
}
