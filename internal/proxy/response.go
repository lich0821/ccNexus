package proxy

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/transformer"
)

// handleNonStreamingResponse processes non-streaming responses
func (p *Proxy) handleNonStreamingResponse(w http.ResponseWriter, resp *http.Response, endpoint config.Endpoint, trans transformer.Transformer) (int, int, error) {
	var bodyBytes []byte
	var err error

	if resp.Header.Get("Content-Encoding") == "gzip" {
		bodyBytes, err = decompressGzip(resp.Body)
		if err != nil {
			logger.Error("[%s] Failed to decompress gzip response: %v", endpoint.Name, err)
			return 0, 0, err
		}
	} else {
		bodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("[%s] Failed to read response body: %v", endpoint.Name, err)
			return 0, 0, err
		}
	}
	resp.Body.Close()

	logger.DebugLog("[%s] Response Body: %s", endpoint.Name, string(bodyBytes))

	// Transform response back to Claude format
	transformedResp, err := trans.TransformResponse(bodyBytes, false)
	if err != nil {
		logger.Error("[%s] Failed to transform response: %v", endpoint.Name, err)
		return 0, 0, err
	}

	logger.DebugLog("[%s] Transformed Response: %s", endpoint.Name, string(transformedResp))

	// Extract token usage
	inputTokens, outputTokens := extractTokenUsage(transformedResp)

	// Copy response headers
	for key, values := range resp.Header {
		if key == "Content-Length" || key == "Content-Encoding" {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(transformedResp)

	return inputTokens, outputTokens, nil
}

// extractTokenUsage extracts token counts from response
func extractTokenUsage(responseBody []byte) (int, int) {
	var resp map[string]interface{}
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return 0, 0
	}

	var inputTokens, outputTokens int

	if usage, ok := resp["usage"].(map[string]interface{}); ok {
		if input, ok := usage["input_tokens"].(float64); ok {
			inputTokens = int(input)
		}
		if output, ok := usage["output_tokens"].(float64); ok {
			outputTokens = int(output)
		}
	}

	return inputTokens, outputTokens
}
