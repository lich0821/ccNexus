package gemini

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/transformer"
)

// GeminiTransformer transforms between Claude and Gemini API formats
type GeminiTransformer struct {
	model string // Target Gemini model name
}

// NewGeminiTransformer creates a new Gemini transformer
func NewGeminiTransformer(model string) *GeminiTransformer {
	return &GeminiTransformer{
		model: model,
	}
}

// cleanGeminiSchema removes unsupported fields from JSON schema for Gemini
func cleanGeminiSchema(schema map[string]interface{}) map[string]interface{} {
	cleaned := make(map[string]interface{})

	for key, value := range schema {
		// Remove unsupported fields
		if key == "additionalProperties" || key == "default" {
			continue
		}

		// Check for unsupported format in string types
		if key == "format" {
			if schemaType, ok := schema["type"].(string); ok && schemaType == "string" {
				if format, ok := value.(string); ok {
					if format != "enum" && format != "date-time" {
						continue
					}
				}
			}
		}

		// Recursively clean nested objects
		if valueMap, ok := value.(map[string]interface{}); ok {
			cleaned[key] = cleanGeminiSchema(valueMap)
		} else if valueArray, ok := value.([]interface{}); ok {
			// Clean array items
			cleanedArray := make([]interface{}, len(valueArray))
			for i, item := range valueArray {
				if itemMap, ok := item.(map[string]interface{}); ok {
					cleanedArray[i] = cleanGeminiSchema(itemMap)
				} else {
					cleanedArray[i] = item
				}
			}
			cleaned[key] = cleanedArray
		} else {
			cleaned[key] = value
		}
	}

	return cleaned
}

// TransformRequest converts Claude format request to Gemini format
func (t *GeminiTransformer) TransformRequest(claudeReq []byte) ([]byte, error) {
	var req transformer.ClaudeRequest
	if err := json.Unmarshal(claudeReq, &req); err != nil {
		return nil, fmt.Errorf("failed to parse Claude request: %w", err)
	}

	// Convert messages to Gemini contents
	geminiContents := make([]transformer.GeminiContent, 0, len(req.Messages))

	for _, msg := range req.Messages {
		geminiContent := transformer.GeminiContent{
			Role:  msg.Role,
			Parts: make([]transformer.GeminiPart, 0),
		}

		// Map Claude roles to Gemini roles
		if msg.Role == "assistant" {
			geminiContent.Role = "model"
		}

		// Handle content - can be string or array
		switch content := msg.Content.(type) {
		case string:
			geminiContent.Parts = append(geminiContent.Parts, transformer.GeminiPart{
				Text: content,
			})
		case []interface{}:
			// Extract text, tool_use, and tool_result from content blocks
			for _, block := range content {
				if blockMap, ok := block.(map[string]interface{}); ok {
					blockType, _ := blockMap["type"].(string)

					switch blockType {
					case "text":
						if text, ok := blockMap["text"].(string); ok {
							geminiContent.Parts = append(geminiContent.Parts, transformer.GeminiPart{
								Text: text,
							})
						}
					case "tool_use":
						name, _ := blockMap["name"].(string)
						input, _ := blockMap["input"].(map[string]interface{})

						geminiContent.Parts = append(geminiContent.Parts, transformer.GeminiPart{
							FunctionCall: &transformer.GeminiFunctionCall{
								Name: name,
								Args: input,
							},
						})
					case "tool_result":
						toolUseID, _ := blockMap["tool_use_id"].(string)

						// Extract tool result content
						var resultContent map[string]interface{}
						if content, ok := blockMap["content"].(string); ok {
							resultContent = map[string]interface{}{
								"result": content,
							}
						} else if contentArray, ok := blockMap["content"].([]interface{}); ok {
							// Handle array of content blocks
							var textParts []string
							for _, c := range contentArray {
								if cMap, ok := c.(map[string]interface{}); ok {
									if cType, _ := cMap["type"].(string); cType == "text" {
										if text, ok := cMap["text"].(string); ok {
											textParts = append(textParts, text)
										}
									}
								}
							}
							resultContent = map[string]interface{}{
								"result": strings.Join(textParts, "\n"),
							}
						} else if contentMap, ok := blockMap["content"].(map[string]interface{}); ok {
							resultContent = contentMap
						}

						// Use tool_use_id as the function name for response
						geminiContent.Parts = append(geminiContent.Parts, transformer.GeminiPart{
							FunctionResponse: &transformer.GeminiFunctionResponse{
								Name:     toolUseID,
								Response: resultContent,
							},
						})
					}
				}
			}
		default:
			geminiContent.Parts = append(geminiContent.Parts, transformer.GeminiPart{
				Text: fmt.Sprintf("%v", content),
			})
		}

		geminiContents = append(geminiContents, geminiContent)
	}

	// Create Gemini request
	geminiReq := transformer.GeminiRequest{
		Contents: geminiContents,
	}

	// Add system instruction if present
	if req.System != nil {
		var systemContent string
		switch sys := req.System.(type) {
		case string:
			systemContent = sys
		case []interface{}:
			// Extract text from system message blocks
			var textParts []string
			for _, block := range sys {
				if blockMap, ok := block.(map[string]interface{}); ok {
					if blockType, ok := blockMap["type"].(string); ok && blockType == "text" {
						if text, ok := blockMap["text"].(string); ok {
							textParts = append(textParts, text)
						}
					}
				}
			}
			systemContent = strings.Join(textParts, "\n\n")
		default:
			systemContent = fmt.Sprintf("%v", sys)
		}

		if systemContent != "" {
			geminiReq.SystemInstruction = &transformer.GeminiContent{
				Parts: []transformer.GeminiPart{
					{Text: strings.TrimSpace(systemContent)},
				},
			}
		}
	}

	// Convert tools to Gemini format
	if len(req.Tools) > 0 {
		geminiTools := make([]transformer.GeminiTool, 0)
		functionDeclarations := make([]transformer.GeminiFunctionDeclaration, 0, len(req.Tools))

		for _, tool := range req.Tools {
			cleanedSchema := cleanGeminiSchema(tool.InputSchema)

			functionDeclarations = append(functionDeclarations, transformer.GeminiFunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  cleanedSchema,
			})
		}

		geminiTools = append(geminiTools, transformer.GeminiTool{
			FunctionDeclarations: functionDeclarations,
		})

		geminiReq.Tools = geminiTools
	}

	// Add generation config
	geminiReq.GenerationConfig = &transformer.GeminiGenerationConfig{}

	if req.Temperature != 0 {
		temp := req.Temperature
		geminiReq.GenerationConfig.Temperature = &temp
	}

	if req.MaxTokens > 0 {
		geminiReq.GenerationConfig.MaxOutputTokens = &req.MaxTokens
	}

	return json.Marshal(geminiReq)
}

// TransformResponse converts Gemini format response to Claude format
func (t *GeminiTransformer) TransformResponse(targetResp []byte, isStreaming bool) ([]byte, error) {
	if isStreaming {
		return nil, fmt.Errorf("use TransformResponseWithContext for streaming responses")
	}
	return t.transformNonStreamingResponse(targetResp)
}

// TransformResponseWithContext converts Gemini format response to Claude format
func (t *GeminiTransformer) TransformResponseWithContext(targetResp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if isStreaming {
		if ctx == nil {
			return nil, fmt.Errorf("StreamContext is required for streaming responses")
		}
		return t.transformStreamingResponse(targetResp, ctx)
	}
	return t.transformNonStreamingResponse(targetResp)
}

// transformNonStreamingResponse converts Gemini non-streaming response to Claude format
func (t *GeminiTransformer) transformNonStreamingResponse(geminiResp []byte) ([]byte, error) {
	var resp transformer.GeminiResponse
	if err := json.Unmarshal(geminiResp, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	// Convert to Claude format
	content := make([]map[string]interface{}, 0)

	// Extract content from first candidate
	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]

		// Process parts
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				content = append(content, map[string]interface{}{
					"type": "text",
					"text": part.Text,
				})
			}

			if part.FunctionCall != nil {
				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    fmt.Sprintf("toolu_%s", part.FunctionCall.Name),
					"name":  part.FunctionCall.Name,
					"input": part.FunctionCall.Args,
				})
			}
		}

		// Make sure content is never empty
		if len(content) == 0 {
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": "",
			})
		}

		// Map Gemini finish_reason to Claude stop_reason
		stopReason := "end_turn"
		switch candidate.FinishReason {
		case "STOP":
			stopReason = "end_turn"
		case "MAX_TOKENS":
			stopReason = "max_tokens"
		case "SAFETY", "RECITATION":
			stopReason = "end_turn"
		case "OTHER":
			stopReason = "end_turn"
		}

		// Build response
		claudeResp := map[string]interface{}{
			"id":            fmt.Sprintf("msg_%d", candidate.Index),
			"type":          "message",
			"role":          "assistant",
			"content":       content,
			"model":         t.model,
			"stop_reason":   stopReason,
			"stop_sequence": nil,
			"usage": map[string]interface{}{
				"input_tokens":  resp.UsageMetadata.PromptTokenCount,
				"output_tokens": resp.UsageMetadata.CandidatesTokenCount,
			},
		}

		return json.Marshal(claudeResp)
	}

	// Fallback if no candidates
	return nil, fmt.Errorf("no candidates in Gemini response")
}

// transformStreamingResponse converts Gemini streaming response to Claude format
func (t *GeminiTransformer) transformStreamingResponse(geminiStream []byte, ctx *transformer.StreamContext) ([]byte, error) {
	var result bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(geminiStream))

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			result.WriteString(line + "\n")
			continue
		}

		// Parse JSON line
		if after, ok := strings.CutPrefix(line, "data: "); ok {
			line = after
		}

		// Check for [DONE] marker
		if line == "[DONE]" {
			// Send final events
			if ctx.ContentBlockStarted && !ctx.FinishReasonSent {
				// Send content_block_stop
				blockStopEvent := map[string]interface{}{
					"type":  "content_block_stop",
					"index": ctx.ContentIndex,
				}
				blockStopJSON, _ := json.Marshal(blockStopEvent)
				result.WriteString("event: content_block_stop\n")
				result.WriteString("data: " + string(blockStopJSON) + "\n")
				result.WriteString("\n")

				// Send message_delta with usage
				messageDeltaEvent := map[string]interface{}{
					"type": "message_delta",
					"delta": map[string]interface{}{
						"stop_reason": "end_turn",
					},
					"usage": map[string]interface{}{
						"output_tokens": ctx.OutputTokens,
					},
				}
				messageDeltaJSON, _ := json.Marshal(messageDeltaEvent)
				result.WriteString("event: message_delta\n")
				result.WriteString("data: " + string(messageDeltaJSON) + "\n")
				result.WriteString("\n")
			}

			// Send message_stop event
			stopEvent := map[string]interface{}{
				"type": "message_stop",
			}
			stopJSON, _ := json.Marshal(stopEvent)
			result.WriteString("event: message_stop\n")
			result.WriteString("data: " + string(stopJSON) + "\n")
			result.WriteString("\n")

			continue
		}

		// Parse Gemini chunk
		var chunk transformer.GeminiStreamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			logger.Debug("[Gemini Transformer] Failed to parse chunk: %v, data: %s", err, line)
			continue
		}

		// Send message_start on first chunk
		if !ctx.MessageStartSent {
			ctx.MessageID = fmt.Sprintf("msg_%d", 0)
			ctx.ModelName = t.model

			// Send message_start event
			startEvent := map[string]interface{}{
				"type": "message_start",
				"message": map[string]interface{}{
					"id":      ctx.MessageID,
					"type":    "message",
					"role":    "assistant",
					"content": []interface{}{},
					"model":   ctx.ModelName,
					"usage": map[string]interface{}{
						"input_tokens":  0,
						"output_tokens": 0,
					},
				},
			}
			startJSON, _ := json.Marshal(startEvent)
			result.WriteString("event: message_start\n")
			result.WriteString("data: " + string(startJSON) + "\n")
			result.WriteString("\n")

			// Send initial content_block_start for text
			blockStartEvent := map[string]interface{}{
				"type":  "content_block_start",
				"index": 0,
				"content_block": map[string]interface{}{
					"type": "text",
					"text": "",
				},
			}
			blockStartJSON, _ := json.Marshal(blockStartEvent)
			result.WriteString("event: content_block_start\n")
			result.WriteString("data: " + string(blockStartJSON) + "\n")
			result.WriteString("\n")

			// Send ping event
			pingEvent := map[string]interface{}{
				"type": "ping",
			}
			pingJSON, _ := json.Marshal(pingEvent)
			result.WriteString("event: ping\n")
			result.WriteString("data: " + string(pingJSON) + "\n")
			result.WriteString("\n")

			ctx.MessageStartSent = true
			ctx.ContentBlockStarted = true
			ctx.ContentIndex = 0
		}

		// Process candidates
		if len(chunk.Candidates) > 0 {
			candidate := chunk.Candidates[0]

			// Process parts
			for _, part := range candidate.Content.Parts {
				// Handle text content
				if part.Text != "" {
					if ctx.ContentBlockStarted {
						deltaEvent := map[string]interface{}{
							"type":  "content_block_delta",
							"index": ctx.ContentIndex,
							"delta": map[string]interface{}{
								"type": "text_delta",
								"text": part.Text,
							},
						}
						deltaJSON, _ := json.Marshal(deltaEvent)
						result.WriteString("event: content_block_delta\n")
						result.WriteString("data: " + string(deltaJSON) + "\n")
						result.WriteString("\n")
					}
				}

				// Handle function calls
				if part.FunctionCall != nil {
					// Close text block if open
					if ctx.ContentBlockStarted && !ctx.ToolBlockStarted {
						blockStopEvent := map[string]interface{}{
							"type":  "content_block_stop",
							"index": ctx.ContentIndex,
						}
						blockStopJSON, _ := json.Marshal(blockStopEvent)
						result.WriteString("event: content_block_stop\n")
						result.WriteString("data: " + string(blockStopJSON) + "\n")
						result.WriteString("\n")
						ctx.ContentBlockStarted = false
					}

					// Start new tool block
					ctx.LastToolIndex++
					toolIndex := ctx.LastToolIndex

					toolID := fmt.Sprintf("toolu_%s", part.FunctionCall.Name)

					// Send content_block_start for tool_use
					toolStartEvent := map[string]interface{}{
						"type":  "content_block_start",
						"index": toolIndex,
						"content_block": map[string]interface{}{
							"type":  "tool_use",
							"id":    toolID,
							"name":  part.FunctionCall.Name,
							"input": map[string]interface{}{},
						},
					}
					toolStartJSON, _ := json.Marshal(toolStartEvent)
					result.WriteString("event: content_block_start\n")
					result.WriteString("data: " + string(toolStartJSON) + "\n")
					result.WriteString("\n")

					// Send input_json_delta with full args
					argsJSON, _ := json.Marshal(part.FunctionCall.Args)
					inputDeltaEvent := map[string]interface{}{
						"type":  "content_block_delta",
						"index": toolIndex,
						"delta": map[string]interface{}{
							"type":         "input_json_delta",
							"partial_json": string(argsJSON),
						},
					}
					inputDeltaJSON, _ := json.Marshal(inputDeltaEvent)
					result.WriteString("event: content_block_delta\n")
					result.WriteString("data: " + string(inputDeltaJSON) + "\n")
					result.WriteString("\n")

					// Close tool block
					toolStopEvent := map[string]interface{}{
						"type":  "content_block_stop",
						"index": toolIndex,
					}
					toolStopJSON, _ := json.Marshal(toolStopEvent)
					result.WriteString("event: content_block_stop\n")
					result.WriteString("data: " + string(toolStopJSON) + "\n")
					result.WriteString("\n")

					ctx.ToolBlockStarted = true
				}
			}

			// Handle finish reason
			if candidate.FinishReason != "" && !ctx.FinishReasonSent {
				ctx.FinishReasonSent = true

				// Close any open blocks
				if ctx.ContentBlockStarted {
					blockStopEvent := map[string]interface{}{
						"type":  "content_block_stop",
						"index": ctx.ContentIndex,
					}
					blockStopJSON, _ := json.Marshal(blockStopEvent)
					result.WriteString("event: content_block_stop\n")
					result.WriteString("data: " + string(blockStopJSON) + "\n")
					result.WriteString("\n")
				}

				// Map Gemini finish_reason to Claude stop_reason
				claudeStopReason := "end_turn"
				switch candidate.FinishReason {
				case "STOP":
					claudeStopReason = "end_turn"
				case "MAX_TOKENS":
					claudeStopReason = "max_tokens"
				case "SAFETY", "RECITATION", "OTHER":
					claudeStopReason = "end_turn"
				}

				// Send message_delta with usage and stop_reason
				messageDeltaEvent := map[string]interface{}{
					"type": "message_delta",
					"delta": map[string]interface{}{
						"stop_reason": claudeStopReason,
					},
					"usage": map[string]interface{}{
						"output_tokens": ctx.OutputTokens,
					},
				}
				messageDeltaJSON, _ := json.Marshal(messageDeltaEvent)
				result.WriteString("event: message_delta\n")
				result.WriteString("data: " + string(messageDeltaJSON) + "\n")
				result.WriteString("\n")

				// Send message_stop
				stopEvent := map[string]interface{}{
					"type": "message_stop",
				}
				stopJSON, _ := json.Marshal(stopEvent)
				result.WriteString("event: message_stop\n")
				result.WriteString("data: " + string(stopJSON) + "\n")
				result.WriteString("\n")
			}
		}

		// Update usage metadata
		if chunk.UsageMetadata != nil {
			ctx.InputTokens = chunk.UsageMetadata.PromptTokenCount
			ctx.OutputTokens = chunk.UsageMetadata.CandidatesTokenCount
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning stream: %w", err)
	}

	return result.Bytes(), nil
}

// Name returns the transformer name
func (t *GeminiTransformer) Name() string {
	return "gemini"
}
