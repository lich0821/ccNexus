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

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/tokencount"
	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/cc"
	"github.com/lich0821/ccNexus/internal/transformer/cx/chat"
	"github.com/lich0821/ccNexus/internal/transformer/cx/responses"
)

// handleStreamingResponse processes streaming SSE responses
func (p *Proxy) handleStreamingResponse(w http.ResponseWriter, resp *http.Response, endpoint config.Endpoint, trans transformer.Transformer, transformerName string, thinkingEnabled bool, modelName string, bodyBytes []byte) (int, int, string) {
	// Copy response headers except Content-Length and Content-Encoding
	for key, values := range resp.Header {
		if key == "Content-Length" || key == "Content-Encoding" {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.Error("[%s] ResponseWriter does not support flushing", endpoint.Name)
		resp.Body.Close()
		return 0, 0, ""
	}

	// Handle gzip-encoded response body
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			logger.Error("[%s] Failed to create gzip reader: %v", endpoint.Name, err)
			resp.Body.Close()
			return 0, 0, ""
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	// Create stream context for all transformers except pure passthrough
	var streamCtx *transformer.StreamContext
	switch transformerName {
	case "cx_chat_openai", "cx_resp_openai2":
		// Pure passthrough - no context needed
	default:
		// cc_claude needs context for input_tokens fallback
		streamCtx = transformer.NewStreamContext()
		streamCtx.ModelName = modelName
		// Pre-estimate input tokens for fallback
		if bodyBytes != nil {
			streamCtx.InputTokens = p.estimateInputTokens(bodyBytes)
		}
	}

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var inputTokens, outputTokens int
	var buffer bytes.Buffer
	var outputText strings.Builder
	eventCount := 0
	streamDone := false

	for scanner.Scan() && !streamDone {
		line := scanner.Text()

		if !p.isCurrentEndpoint(endpoint.Name) {
			logger.Warn("[%s] Endpoint switched during streaming, terminating stream gracefully", endpoint.Name)
			streamDone = true
			break
		}

		if strings.Contains(line, "data: [DONE]") {
			// Fallback: update output_tokens if not provided
			if outputTokens == 0 && outputText.String() != "" {
				outputTokens = tokencount.EstimateOutputTokens(outputText.String())
			}

			if streamCtx != nil {
				streamCtx.OutputTokens = outputTokens
				streamCtx.InputTokens = inputTokens
			}

			// Create message_delta to output
			deltaEvent := map[string]interface{}{
				"type": "message_delta",
				"delta": map[string]interface{}{
					"stop_reason":   "end_turn",
					"stop_sequence": nil,
				},
				"usage": map[string]interface{}{
					"output_tokens": outputTokens,
				},
			}
			deltaData, _ := json.Marshal(deltaEvent)
			deltaSSE := fmt.Sprintf("data: %s\n\n", deltaData)

			// For cc_claude, send directly to avoid transformer modifying it
			if transformerName == "cc_claude" {
				w.Write([]byte(deltaSSE))
				flusher.Flush()
			} else {
				// Transform event for other transformers
				transformedDelta, _ := p.transformStreamEvent([]byte(deltaSSE), trans, transformerName, streamCtx)
				if len(transformedDelta) > 0 {
					w.Write(transformedDelta)
					flusher.Flush()
				}
			}

			streamDone = true
			buffer.WriteString(line + "\n")
			eventData := buffer.Bytes()
			logger.DebugLog("[%s] SSE Event #%d (Original): %s", endpoint.Name, eventCount+1, string(eventData))

			transformedEvent, err := p.transformStreamEvent(eventData, trans, transformerName, streamCtx)
			if err == nil && len(transformedEvent) > 0 {
				logger.DebugLog("[%s] SSE Event #%d (Transformed): %s", endpoint.Name, eventCount+1, string(transformedEvent))
				w.Write(transformedEvent)
				flusher.Flush()
			}
			break
		}

		buffer.WriteString(line + "\n")

		if line == "" {
			eventCount++
			eventData := buffer.Bytes()
			logger.DebugLog("[%s] SSE Event #%d (Original): %s", endpoint.Name, eventCount, string(eventData))

			transformedEvent, err := p.transformStreamEvent(eventData, trans, transformerName, streamCtx)
			if err != nil {
				logger.Error("[%s] Failed to transform SSE event: %v", endpoint.Name, err)
			} else if len(transformedEvent) > 0 {
				logger.DebugLog("[%s] SSE Event #%d (Transformed): %s", endpoint.Name, eventCount, string(transformedEvent))

				p.extractTokensFromEvent(transformedEvent, &inputTokens, &outputTokens)
				p.extractTextFromEvent(transformedEvent, &outputText)

				// Check if this is message_stop event - send custom message_delta before it
				isMessageStop := strings.Contains(string(transformedEvent), `"type":"message_stop"`)
				if isMessageStop {
					// Estimate output tokens if not provided
					if outputTokens == 0 && outputText.String() != "" {
						outputTokens = tokencount.EstimateOutputTokens(outputText.String())
					}

					if streamCtx != nil {
						streamCtx.OutputTokens = outputTokens
						streamCtx.InputTokens = inputTokens
					}

					// Create message_delta event with output_tokens
					deltaEvent := map[string]interface{}{
						"type": "message_delta",
						"delta": map[string]interface{}{
							"stop_reason":   "end_turn",
							"stop_sequence": nil,
						},
						"usage": map[string]interface{}{
							"output_tokens": outputTokens,
						},
					}
					deltaData, _ := json.Marshal(deltaEvent)
					deltaSSE := fmt.Sprintf("data: %s\n\n", deltaData)

					// For cc_claude, send directly
					if transformerName == "cc_claude" {
						w.Write([]byte(deltaSSE))
						flusher.Flush()
					} else {
						// Transform event for other transformers
						transformedDelta, _ := p.transformStreamEvent([]byte(deltaSSE), trans, transformerName, streamCtx)
						if len(transformedDelta) > 0 {
							w.Write(transformedDelta)
							flusher.Flush()
						}
					}
				}

				if _, writeErr := w.Write(transformedEvent); writeErr != nil {
					// Client disconnected (broken pipe) is normal for cancelled requests
					if strings.Contains(writeErr.Error(), "broken pipe") || strings.Contains(writeErr.Error(), "connection reset") {
						logger.Debug("[%s] Client disconnected: %v", endpoint.Name, writeErr)
					} else {
						logger.Error("[%s] Failed to write transformed event: %v", endpoint.Name, writeErr)
					}
					streamDone = true
					break
				}
				flusher.Flush()
			}
			buffer.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("[%s] Scanner error: %v", endpoint.Name, err)
	}

	resp.Body.Close()
	return inputTokens, outputTokens, outputText.String()
}

// transformStreamEvent transforms a single SSE event
func (p *Proxy) transformStreamEvent(eventData []byte, trans transformer.Transformer, transformerName string, streamCtx *transformer.StreamContext) ([]byte, error) {
	switch transformerName {
	// Claude Code transformers
	case "cc_claude":
		return trans.(*cc.ClaudeTransformer).TransformResponseWithContext(eventData, true, streamCtx)
	case "cc_openai":
		return trans.(*cc.OpenAITransformer).TransformResponseWithContext(eventData, true, streamCtx)
	case "cc_openai2":
		return trans.(*cc.OpenAI2Transformer).TransformResponseWithContext(eventData, true, streamCtx)
	case "cc_gemini":
		return trans.(*cc.GeminiTransformer).TransformResponseWithContext(eventData, true, streamCtx)
	// Codex Chat transformers
	case "cx_chat_claude":
		return trans.(*chat.ClaudeTransformer).TransformResponseWithContext(eventData, true, streamCtx)
	case "cx_chat_openai":
		return eventData, nil // passthrough
	case "cx_chat_openai2":
		return trans.(*chat.OpenAI2Transformer).TransformResponseWithContext(eventData, true, streamCtx)
	case "cx_chat_gemini":
		return trans.(*chat.GeminiTransformer).TransformResponseWithContext(eventData, true, streamCtx)
	// Codex Responses transformers
	case "cx_resp_claude":
		return trans.(*responses.ClaudeTransformer).TransformResponseWithContext(eventData, true, streamCtx)
	case "cx_resp_openai":
		return trans.(*responses.OpenAITransformer).TransformResponseWithContext(eventData, true, streamCtx)
	case "cx_resp_openai2":
		return eventData, nil // passthrough
	case "cx_resp_gemini":
		return trans.(*responses.GeminiTransformer).TransformResponseWithContext(eventData, true, streamCtx)
	default:
		return trans.TransformResponse(eventData, true)
	}
}

// extractTokensFromEvent extracts token counts from SSE event
func (p *Proxy) extractTokensFromEvent(eventData []byte, inputTokens, outputTokens *int) {
	scanner := bufio.NewScanner(bytes.NewReader(eventData))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		jsonData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		if eventType == "message_start" {
			if message, ok := event["message"].(map[string]interface{}); ok {
				if usage, ok := message["usage"].(map[string]interface{}); ok {
					if input, ok := usage["input_tokens"].(float64); ok {
						*inputTokens = int(input)
					}
				}
			}
		} else if eventType == "message_delta" {
			if usage, ok := event["usage"].(map[string]interface{}); ok {
				if output, ok := usage["output_tokens"].(float64); ok {
					*outputTokens = int(output)
				}
			}
		}
	}
}

// extractTextFromEvent extracts text content from transformed event
func (p *Proxy) extractTextFromEvent(transformedEvent []byte, outputText *strings.Builder) {
	scanner := bufio.NewScanner(bytes.NewReader(transformedEvent))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		jsonData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)

		// Handle content_block_delta (new format)
		if eventType == "content_block_delta" {
			if delta, ok := event["delta"].(map[string]interface{}); ok {
				if deltaType, _ := delta["type"].(string); deltaType == "text_delta" {
					if text, ok := delta["text"].(string); ok {
						outputText.WriteString(text)
					}
				}
			}
		}

		// Handle content_block (old format)
		if delta, ok := event["delta"].(map[string]interface{}); ok {
			if text, ok := delta["text"].(string); ok {
				outputText.WriteString(text)
			}
		}
	}
}

// decompressGzip decompresses gzip-encoded response body
func decompressGzip(body io.ReadCloser) ([]byte, error) {
	gzipReader, err := gzip.NewReader(body)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	return io.ReadAll(gzipReader)
}
