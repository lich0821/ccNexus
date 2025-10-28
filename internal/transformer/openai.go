package transformer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/logger"
)

// OpenAITransformer transforms between Claude and OpenAI API formats
// This transformer is now stateless - all state is passed via StreamContext
type OpenAITransformer struct {
	model string // Target OpenAI model name
}

// NewOpenAITransformer creates a new OpenAI transformer
func NewOpenAITransformer(model string) *OpenAITransformer {
	return &OpenAITransformer{
		model: model,
	}
}

// extractToolResultContent extracts content from tool_result block
func extractToolResultContent(content interface{}) string {
	if content == nil {
		return "No content provided"
	}

	// Handle string content
	if str, ok := content.(string); ok {
		return str
	}

	// Handle array of content blocks
	if contentArray, ok := content.([]interface{}); ok {
		var result string
		for _, item := range contentArray {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemType, _ := itemMap["type"].(string); itemType == "text" {
					if text, ok := itemMap["text"].(string); ok {
						result += text + "\n"
					}
				} else if text, ok := itemMap["text"].(string); ok {
					result += text + "\n"
				} else {
					if jsonBytes, err := json.Marshal(itemMap); err == nil {
						result += string(jsonBytes) + "\n"
					} else {
						result += fmt.Sprintf("%v\n", itemMap)
					}
				}
			} else if str, ok := item.(string); ok {
				result += str + "\n"
			} else {
				result += fmt.Sprintf("%v\n", item)
			}
		}
		return strings.TrimSpace(result)
	}

	// Handle dict content
	if contentMap, ok := content.(map[string]interface{}); ok {
		if contentType, _ := contentMap["type"].(string); contentType == "text" {
			if text, ok := contentMap["text"].(string); ok {
				return text
			}
		}
		if jsonBytes, err := json.Marshal(contentMap); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", contentMap)
	}

	// Fallback for any other type
	return fmt.Sprintf("%v", content)
}

// TransformRequest converts Claude format request to OpenAI format
func (t *OpenAITransformer) TransformRequest(claudeReq []byte) ([]byte, error) {
	var req ClaudeRequest
	if err := json.Unmarshal(claudeReq, &req); err != nil {
		return nil, fmt.Errorf("failed to parse Claude request: %w", err)
	}

	// Convert messages
	openaiMessages := make([]OpenAIMessage, 0, len(req.Messages))

	// Add system message if present
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
			openaiMessages = append(openaiMessages, OpenAIMessage{
				Role:    "system",
				Content: strings.TrimSpace(systemContent),
			})
		}
	}

	for _, msg := range req.Messages {
		openaiMsg := OpenAIMessage{
			Role: msg.Role,
		}

		// Handle content - can be string or array
		switch content := msg.Content.(type) {
		case string:
			openaiMsg.Content = content
		case []interface{}:
			// Check if this is a user message with tool_result
			if msg.Role == "user" {
				hasToolResult := false
				for _, block := range content {
					if blockMap, ok := block.(map[string]interface{}); ok {
						if blockType, _ := blockMap["type"].(string); blockType == "tool_result" {
							hasToolResult = true
							break
						}
					}
				}

				if hasToolResult {
					var textContent string

					for _, block := range content {
						if blockMap, ok := block.(map[string]interface{}); ok {
							blockType, _ := blockMap["type"].(string)

							switch blockType {
							case "text":
								if text, ok := blockMap["text"].(string); ok {
									textContent += text + "\n"
								}
							case "tool_result":
								toolUseID, _ := blockMap["tool_use_id"].(string)

								var resultContent string
								if contentVal, ok := blockMap["content"]; ok {
									resultContent = extractToolResultContent(contentVal)
								}

								textContent += fmt.Sprintf("Tool result for %s:\n%s\n", toolUseID, resultContent)
							}
						}
					}

					openaiMsg.Content = strings.TrimSpace(textContent)
					openaiMessages = append(openaiMessages, openaiMsg)
					continue
				}
			}

			// Regular handling for other message types
			var textParts []string
			for _, block := range content {
				if blockMap, ok := block.(map[string]interface{}); ok {
					blockType, _ := blockMap["type"].(string)

					switch blockType {
					case "text":
						if text, ok := blockMap["text"].(string); ok {
							textParts = append(textParts, text)
						}
					case "tool_use":
						logger.Debug("Found tool_use block in request message (role: %s)", msg.Role)
					case "image":
						logger.Debug("Image block found but not supported in OpenAI transformer")
					}
				}
			}

			openaiMsg.Content = strings.Join(textParts, "\n")
		default:
			openaiMsg.Content = fmt.Sprintf("%v", content)
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	// Create OpenAI request
	openaiReq := OpenAIRequest{
		Model:       t.model,
		Messages:    openaiMessages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      req.Stream,
	}

	// Convert tools to OpenAI format
	if len(req.Tools) > 0 {
		openaiTools := make([]OpenAITool, 0, len(req.Tools))
		for _, tool := range req.Tools {
			openaiTool := OpenAITool{
				Type: "function",
			}
			openaiTool.Function.Name = tool.Name
			openaiTool.Function.Description = tool.Description
			openaiTool.Function.Parameters = tool.InputSchema

			openaiTools = append(openaiTools, openaiTool)
		}
		openaiReq.Tools = openaiTools
	}

	// Convert tool_choice to OpenAI format
	if req.ToolChoice != nil {
		switch tc := req.ToolChoice.(type) {
		case map[string]interface{}:
			choiceType, _ := tc["type"].(string)
			switch choiceType {
			case "auto":
				openaiReq.ToolChoice = "auto"
			case "any":
				openaiReq.ToolChoice = "any"
			case "tool":
				if name, ok := tc["name"].(string); ok {
					openaiReq.ToolChoice = map[string]interface{}{
						"type": "function",
						"function": map[string]string{
							"name": name,
						},
					}
				}
			}
		case string:
			openaiReq.ToolChoice = tc
		}
	}

	// Handle thinking parameter
	if req.Thinking != nil {
		switch thinking := req.Thinking.(type) {
		case map[string]interface{}:
			if thinkingType, ok := thinking["type"].(string); ok && thinkingType == "enabled" {
				openaiReq.EnableThinking = true
			}
		case bool:
			openaiReq.EnableThinking = thinking
		}
	}

	return json.Marshal(openaiReq)
}

// TransformResponse converts OpenAI format response to Claude format
func (t *OpenAITransformer) TransformResponse(targetResp []byte, isStreaming bool) ([]byte, error) {
	if isStreaming {
		return nil, fmt.Errorf("use TransformResponseWithContext for streaming responses")
	}
	return t.transformNonStreamingResponse(targetResp)
}

// TransformResponseWithContext converts OpenAI format response to Claude format
func (t *OpenAITransformer) TransformResponseWithContext(targetResp []byte, isStreaming bool, ctx *StreamContext) ([]byte, error) {
	if isStreaming {
		if ctx == nil {
			return nil, fmt.Errorf("StreamContext is required for streaming responses")
		}
		return t.transformStreamingResponse(targetResp, ctx)
	}
	return t.transformNonStreamingResponse(targetResp)
}

// transformNonStreamingResponse converts OpenAI non-streaming response to Claude format
func (t *OpenAITransformer) transformNonStreamingResponse(openaiResp []byte) ([]byte, error) {
	var resp OpenAIResponse
	if err := json.Unmarshal(openaiResp, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	// Convert to Claude format
	content := make([]map[string]interface{}, 0)

	// Extract content from first choice
	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]

		// Add text content if present
		if choice.Message.Content != "" {
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": choice.Message.Content,
			})
		}

		// Add tool calls if present
		if len(choice.Message.ToolCalls) > 0 {
			for _, toolCall := range choice.Message.ToolCalls {
				// Parse arguments from JSON string to map
				var input map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
					logger.Warn("Failed to parse tool arguments: %v", err)
					input = map[string]interface{}{"raw": toolCall.Function.Arguments}
				}

				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    toolCall.ID,
					"name":  toolCall.Function.Name,
					"input": input,
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

		// Map finish_reason to stop_reason
		stopReason := "end_turn"
		switch choice.FinishReason {
		case "stop":
			stopReason = "end_turn"
		case "length":
			stopReason = "max_tokens"
		case "tool_calls":
			stopReason = "tool_use"
		case "content_filter":
			stopReason = "end_turn"
		}

		// Build response
		claudeResp := map[string]interface{}{
			"id":            resp.ID,
			"type":          "message",
			"role":          "assistant",
			"content":       content,
			"model":         resp.Model,
			"stop_reason":   stopReason,
			"stop_sequence": nil,
			"usage": map[string]interface{}{
				"input_tokens":  resp.Usage.PromptTokens,
				"output_tokens": resp.Usage.CompletionTokens,
			},
		}

		return json.Marshal(claudeResp)
	}

	// Fallback if no choices
	return nil, fmt.Errorf("no choices in OpenAI response")
}

// transformStreamingResponse converts OpenAI streaming response to Claude format
func (t *OpenAITransformer) transformStreamingResponse(openaiStream []byte, ctx *StreamContext) ([]byte, error) {
	var result bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(openaiStream))

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			result.WriteString(line + "\n")
			continue
		}

		// Skip event: lines
		if strings.HasPrefix(line, "event:") {
			continue
		}

		// Parse SSE data line
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for [DONE] marker
			if data == "[DONE]" {
				// Before sending message_stop, ensure we've sent content_block_stop and message_delta
				if ctx.ContentBlockStarted && !ctx.FinishReasonSent {
					// Send content_block_stop
					blockStopEvent := ClaudeStreamEvent{
						Type:  "content_block_stop",
						Index: ctx.ContentIndex,
					}
					blockStopJSON, _ := json.Marshal(blockStopEvent)
					result.WriteString("event: content_block_stop\n")
					result.WriteString("data: " + string(blockStopJSON) + "\n")
					result.WriteString("\n")

					// Send message_delta with usage
					messageDeltaEvent := ClaudeStreamEvent{
						Type: "message_delta",
						Delta: struct {
							Type string `json:"type"`
							Text string `json:"text"`
						}{
							Type: "text",
							Text: "",
						},
						Usage: struct {
							OutputTokens int `json:"output_tokens"`
						}{
							OutputTokens: ctx.TotalOutputTokens,
						},
					}
					messageDeltaJSON, _ := json.Marshal(messageDeltaEvent)
					result.WriteString("event: message_delta\n")
					result.WriteString("data: " + string(messageDeltaJSON) + "\n")
					result.WriteString("\n")

					// Add empty event separator
					result.WriteString("\n")
				}

				// Send message_stop event (only type field, no other fields)
				stopEvent := map[string]interface{}{
					"type": "message_stop",
				}
				stopJSON, _ := json.Marshal(stopEvent)
				result.WriteString("event: message_stop\n")
				result.WriteString("data: " + string(stopJSON) + "\n")
				result.WriteString("\n")

				// Reset context after stream ends
				ctx.MessageStartSent = false
				ctx.ContentBlockStarted = false
				ctx.MessageID = ""
				ctx.ModelName = ""
				ctx.TotalOutputTokens = 0
				ctx.ContentIndex = 0
				ctx.FinishReasonSent = false

				continue
			}

			// Parse OpenAI chunk
			var chunk OpenAIStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// If parse fails, log the error and pass through original line
				logger.Debug("Failed to parse OpenAI chunk: %v, data: %s", err, data)
				result.WriteString(line + "\n")
				continue
			}

			// Store message ID and model on first chunk
			if ctx.MessageID == "" && chunk.ID != "" {
				ctx.MessageID = chunk.ID
			}
			if ctx.ModelName == "" && chunk.Model != "" {
				ctx.ModelName = chunk.Model
			}

			// Send message_start immediately on first chunk
			if !ctx.MessageStartSent && ctx.MessageID != "" {
				// Send message_start event
				startEvent := ClaudeStreamEvent{
					Type: "message_start",
					Message: struct {
						ID      string `json:"id"`
						Type    string `json:"type"`
						Role    string `json:"role"`
						Content []struct {
							Type string `json:"type"`
							Text string `json:"text"`
						} `json:"content"`
						Model      string `json:"model"`
						StopReason string `json:"stop_reason"`
						Usage      struct {
							InputTokens  int `json:"input_tokens"`
							OutputTokens int `json:"output_tokens"`
						} `json:"usage"`
					}{
						ID:   ctx.MessageID,
						Type: "message",
						Role: "assistant",
						Content: []struct {
							Type string `json:"type"`
							Text string `json:"text"`
						}{},
						Model: ctx.ModelName,
					},
				}
				startJSON, _ := json.Marshal(startEvent)
				result.WriteString("event: message_start\n")
				result.WriteString("data: " + string(startJSON) + "\n")
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
				ctx.ContentBlockStarted = false
				ctx.ContentIndex = 0
			}

			// Convert to Claude events
			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]

				// Check if this chunk has any content
				hasContent := choice.Delta.Content != ""
				hasReasoning := choice.Delta.ReasoningContent != ""

				// Handle content delta
				if hasReasoning && ctx.EnableThinking {
					if ctx.MessageStartSent && !ctx.ThinkingBlockStarted {
						// Send thinking content_block_start
						thinkingStartEvent := map[string]interface{}{
							"type":  "content_block_start",
							"index": ctx.ThinkingIndex,
							"content_block": map[string]interface{}{
								"type":     "thinking",
								"thinking": "",
							},
						}
						thinkingStartJSON, _ := json.Marshal(thinkingStartEvent)
						result.WriteString("event: content_block_start\n")
						result.WriteString("data: " + string(thinkingStartJSON) + "\n")
						result.WriteString("\n")
						ctx.ThinkingBlockStarted = true
					}

					// Send thinking content delta
					if ctx.ThinkingBlockStarted {
						thinkingDeltaEvent := map[string]interface{}{
							"type":  "content_block_delta",
							"index": ctx.ThinkingIndex,
							"delta": map[string]interface{}{
								"type":     "thinking_delta",
								"thinking": choice.Delta.ReasoningContent,
							},
						}
						thinkingDeltaJSON, _ := json.Marshal(thinkingDeltaEvent)
						result.WriteString("event: content_block_delta\n")
						result.WriteString("data: " + string(thinkingDeltaJSON) + "\n")
						result.WriteString("\n")
					}
				}

				// Handle regular content
				if hasContent {
					// If we're transitioning from thinking to text, close thinking block first
					if ctx.ThinkingBlockStarted {
						thinkingStopEvent := map[string]interface{}{
							"type":  "content_block_stop",
							"index": ctx.ThinkingIndex,
						}
						thinkingStopJSON, _ := json.Marshal(thinkingStopEvent)
						result.WriteString("event: content_block_stop\n")
						result.WriteString("data: " + string(thinkingStopJSON) + "\n")
						result.WriteString("\n")
						ctx.ThinkingBlockStarted = false
					}

					// Start text content block if not started yet
					if !ctx.ContentBlockStarted {
						blockStartEvent := map[string]interface{}{
							"type":  "content_block_start",
							"index": ctx.ContentIndex,
							"content_block": map[string]interface{}{
								"type": "text",
								"text": "",
							},
						}
						blockStartJSON, _ := json.Marshal(blockStartEvent)
						result.WriteString("event: content_block_start\n")
						result.WriteString("data: " + string(blockStartJSON) + "\n")
						result.WriteString("\n")
						ctx.ContentBlockStarted = true
					}

					// Send text delta
					deltaEvent := map[string]interface{}{
						"type":  "content_block_delta",
						"index": ctx.ContentIndex,
						"delta": map[string]interface{}{
							"type": "text_delta",
							"text": choice.Delta.Content,
						},
					}
					deltaJSON, _ := json.Marshal(deltaEvent)
					result.WriteString("event: content_block_delta\n")
					result.WriteString("data: " + string(deltaJSON) + "\n")
					result.WriteString("\n")

					ctx.TotalOutputTokens++ // Rough estimation
				}

				// Handle tool calls in streaming
				if len(choice.Delta.ToolCalls) > 0 {
					for _, toolCall := range choice.Delta.ToolCalls {
						// Determine the tool call index (default to 0 if not provided)
						toolCallIndex := 0
						if toolCall.Index != nil {
							toolCallIndex = *toolCall.Index
						}

						// Check if this is a new tool call
						// A new tool call is identified by:
						isNewToolCall := ctx.CurrentToolCall == nil ||
							toolCallIndex != ctx.ToolIndex ||
							(toolCall.ID != "" && ctx.CurrentToolCall.ID != "" && toolCall.ID != ctx.CurrentToolCall.ID)

						if isNewToolCall {
							// Close previous tool block if any
							if ctx.ToolBlockStarted {
								toolStopEvent := map[string]interface{}{
									"type":  "content_block_stop",
									"index": ctx.LastToolIndex,
								}
								toolStopJSON, _ := json.Marshal(toolStopEvent)
								result.WriteString("event: content_block_stop\n")
								result.WriteString("data: " + string(toolStopJSON) + "\n")
								result.WriteString("\n")
							}

							// Close thinking block first if it's still open
							if ctx.ThinkingBlockStarted {
								thinkingStopEvent := map[string]interface{}{
									"type":  "content_block_stop",
									"index": ctx.ThinkingIndex,
								}
								thinkingStopJSON, _ := json.Marshal(thinkingStopEvent)
								result.WriteString("event: content_block_stop\n")
								result.WriteString("data: " + string(thinkingStopJSON) + "\n")
								result.WriteString("\n")
								ctx.ThinkingBlockStarted = false
							}

							// Close text block if it's still open
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

							// Update OpenAI tool index tracker
							ctx.ToolIndex = toolCallIndex

							// Increment LastToolIndex for new tool block
							ctx.LastToolIndex++

							// Initialize new tool call
							ctx.CurrentToolCall = &OpenAIToolCall{
								ID:   toolCall.ID,
								Type: "function",
							}
							if toolCall.Function.Name != "" {
								ctx.CurrentToolCall.Function.Name = toolCall.Function.Name
							}
							ctx.ToolCallBuffer = ""

							// Mark that we need to send content_block_start
							ctx.ToolBlockStarted = false
							ctx.ToolBlockPending = true
						} else {
							// Update tool call ID if it's provided in a subsequent chunk
							if toolCall.ID != "" && ctx.CurrentToolCall.ID == "" {
								ctx.CurrentToolCall.ID = toolCall.ID
							}
							// Update function name if it's provided in a subsequent chunk
							if toolCall.Function.Name != "" && ctx.CurrentToolCall.Function.Name == "" {
								ctx.CurrentToolCall.Function.Name = toolCall.Function.Name
							}
						}

						// Accumulate tool arguments
						if toolCall.Function.Arguments != "" {
							ctx.ToolCallBuffer += toolCall.Function.Arguments

							// Calculate current Anthropic tool index for deltas
							anthropicToolIndex := ctx.LastToolIndex

							// If tool block is pending, send content_block_start first
							if ctx.ToolBlockPending && !ctx.ToolBlockStarted {
								// Send content_block_start for tool_use
								toolStartEvent := map[string]interface{}{
									"type":  "content_block_start",
									"index": anthropicToolIndex,
									"content_block": map[string]interface{}{
										"type":  "tool_use",
										"id":    ctx.CurrentToolCall.ID,
										"name":  ctx.CurrentToolCall.Function.Name,
										"input": map[string]interface{}{},
									},
								}
								toolStartJSON, _ := json.Marshal(toolStartEvent)
								result.WriteString("event: content_block_start\n")
								result.WriteString("data: " + string(toolStartJSON) + "\n")
								result.WriteString("\n")

								// Send an empty input_json_delta immediately after content_block_start
								emptyDeltaEvent := map[string]interface{}{
									"type":  "content_block_delta",
									"index": anthropicToolIndex,
									"delta": map[string]interface{}{
										"type":         "input_json_delta",
										"partial_json": "",
									},
								}
								emptyDeltaJSON, _ := json.Marshal(emptyDeltaEvent)
								result.WriteString("event: content_block_delta\n")
								result.WriteString("data: " + string(emptyDeltaJSON) + "\n")
								result.WriteString("\n")

								ctx.ToolBlockStarted = true
								ctx.ToolBlockPending = false
							}

							// Always send input_json_delta for the actual arguments
							if ctx.ToolBlockStarted {
								inputDeltaEvent := map[string]interface{}{
									"type":  "content_block_delta",
									"index": anthropicToolIndex,
									"delta": map[string]interface{}{
										"type":         "input_json_delta",
										"partial_json": toolCall.Function.Arguments,
									},
								}
								inputDeltaJSON, _ := json.Marshal(inputDeltaEvent)
								result.WriteString("event: content_block_delta\n")
								result.WriteString("data: " + string(inputDeltaJSON) + "\n")
								result.WriteString("\n")
							}
						}
					}
				}

				// Handle finish
				if choice.FinishReason != nil && *choice.FinishReason != "" {
					finishReason := *choice.FinishReason
					ctx.FinishReasonSent = true

					// Close any open blocks
					if ctx.ToolBlockStarted {
						// Close all tool blocks
						for i := 1; i <= ctx.LastToolIndex; i++ {
							toolStopEvent := map[string]interface{}{
								"type":  "content_block_stop",
								"index": i,
							}
							toolStopJSON, _ := json.Marshal(toolStopEvent)
							result.WriteString("event: content_block_stop\n")
							result.WriteString("data: " + string(toolStopJSON) + "\n")
							result.WriteString("\n")
						}
					} else if ctx.ContentBlockStarted {
						blockStopEvent := map[string]interface{}{
							"type":  "content_block_stop",
							"index": ctx.ContentIndex,
						}
						blockStopJSON, _ := json.Marshal(blockStopEvent)
						result.WriteString("event: content_block_stop\n")
						result.WriteString("data: " + string(blockStopJSON) + "\n")
						result.WriteString("\n")
					}

					// Map OpenAI finish_reason to Claude stop_reason
					claudeStopReason := "end_turn"
					switch finishReason {
					case "stop":
						claudeStopReason = "end_turn"
					case "length":
						claudeStopReason = "max_tokens"
					case "tool_calls", "function_call":
						claudeStopReason = "tool_use"
					case "content_filter":
						claudeStopReason = "end_turn"
					}

					// Send message_delta with usage and stop_reason
					messageDeltaEvent := map[string]interface{}{
						"type": "message_delta",
						"delta": map[string]interface{}{
							"stop_reason": claudeStopReason,
						},
						"usage": map[string]interface{}{
							"output_tokens": ctx.TotalOutputTokens,
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

					// Reset context after stream ends
					ctx.MessageStartSent = false
					ctx.ContentBlockStarted = false
					ctx.ToolBlockStarted = false
					ctx.MessageID = ""
					ctx.ModelName = ""
					ctx.TotalOutputTokens = 0
					ctx.ContentIndex = 0
					ctx.FinishReasonSent = false
				}
			}
		} else {
			// Pass through other lines
			result.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning stream: %w", err)
	}

	return result.Bytes(), nil
}

// Name returns the transformer name
func (t *OpenAITransformer) Name() string {
	return "openai"
}
