package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

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
	if inputTokens == 0 && outputTokens == 0 {
		// Defensive fallback: if upstream SSE is misrouted to non-stream path,
		// extract usage directly from raw event payload.
		p.extractTokensFromEvent(bodyBytes, &inputTokens, &outputTokens)
	}

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

	if usage, ok := resp["usage"].(map[string]interface{}); ok {
		return extractInputOutputTokens(usage)
	}

	return 0, 0
}

// extractInputOutputTokens normalizes usage fields across API formats.
// Supports:
// - Claude/OpenAI Responses: input_tokens/output_tokens
// - OpenAI Chat: prompt_tokens/completion_tokens
func extractInputOutputTokens(usage map[string]interface{}) (int, int) {
	var inputTokens, outputTokens int

	if input, ok := usage["input_tokens"]; ok {
		inputTokens = parseTokenNumber(input)
	} else if input, ok := usage["prompt_tokens"]; ok {
		inputTokens = parseTokenNumber(input)
	}

	if output, ok := usage["output_tokens"]; ok {
		outputTokens = parseTokenNumber(output)
	} else if output, ok := usage["completion_tokens"]; ok {
		outputTokens = parseTokenNumber(output)
	}

	return inputTokens, outputTokens
}

func parseTokenNumber(value interface{}) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
		if f, err := v.Float64(); err == nil {
			return int(f)
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0
		}
		if i, err := strconv.Atoi(trimmed); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return int(f)
		}
	}
	return 0
}

func extractResponseOutputText(responseBody []byte) string {
	var payload map[string]interface{}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return ""
	}

	var builder strings.Builder

	appendString := func(value interface{}) {
		if text, ok := value.(string); ok && text != "" {
			builder.WriteString(text)
		}
	}

	// OpenAI Chat style: choices[].message.content
	if choices, ok := payload["choices"].([]interface{}); ok {
		for _, choiceVal := range choices {
			choice, ok := choiceVal.(map[string]interface{})
			if !ok {
				continue
			}
			message, ok := choice["message"].(map[string]interface{})
			if !ok {
				continue
			}
			appendString(message["content"])
		}
	}

	// Claude style: content[].text
	if content, ok := payload["content"].([]interface{}); ok {
		for _, blockVal := range content {
			block, ok := blockVal.(map[string]interface{})
			if !ok {
				continue
			}
			appendString(block["text"])
		}
	}

	// OpenAI Responses style: output[].content[].text
	if output, ok := payload["output"].([]interface{}); ok {
		for _, outVal := range output {
			item, ok := outVal.(map[string]interface{})
			if !ok {
				continue
			}
			content, ok := item["content"].([]interface{})
			if !ok {
				continue
			}
			for _, partVal := range content {
				part, ok := partVal.(map[string]interface{})
				if !ok {
					continue
				}
				appendString(part["text"])
			}
		}
	}

	return builder.String()
}
