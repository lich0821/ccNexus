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
	"github.com/lich0821/ccNexus/internal/transformer/claude"
	"github.com/lich0821/ccNexus/internal/transformer/gemini"
	"github.com/lich0821/ccNexus/internal/transformer/openai"
)

// prepareTransformer creates and returns the appropriate transformer for the endpoint
func prepareTransformer(endpoint config.Endpoint) (transformer.Transformer, error) {
	transformerName := endpoint.Transformer
	if transformerName == "" {
		transformerName = "claude"
	}

	switch transformerName {
	case "openai":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("OpenAI transformer requires model field")
		}
		return openai.NewOpenAITransformer(endpoint.Model), nil
	case "gemini":
		if endpoint.Model == "" {
			return nil, fmt.Errorf("Gemini transformer requires model field")
		}
		return gemini.NewGeminiTransformer(endpoint.Model), nil
	case "claude":
		if endpoint.Model != "" {
			logger.Debug("[%s] Using Claude transformer with model override: %s", endpoint.Name, endpoint.Model)
			return claude.NewClaudeTransformerWithModel(endpoint.Model), nil
		}
		logger.Debug("[%s] Using Claude transformer with model passthrough", endpoint.Name)
		return claude.NewClaudeTransformer(), nil
	default:
		return transformer.Get(transformerName)
	}
}

// buildProxyRequest creates an HTTP request for the target API
func buildProxyRequest(r *http.Request, endpoint config.Endpoint, transformedBody []byte, transformerName string) (*http.Request, error) {
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
	case "openai":
		proxyReq.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
	case "gemini":
		q := proxyReq.URL.Query()
		q.Set("key", endpoint.APIKey)
		q.Set("alt", "sse")
		proxyReq.URL.RawQuery = q.Encode()
	default:
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
