package openai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/transformer"
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
	var req transformer.ClaudeRequest
	if err := json.Unmarshal(claudeReq, &req); err != nil {
		return nil, fmt.Errorf("failed to parse Claude request: %w", err)
	}

	// Convert messages
	openaiMessages := make([]transformer.OpenAIMessage, 0, len(req.Messages))

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
			openaiMessages = append(openaiMessages, transformer.OpenAIMessage{
				Role:    "system",
				Content: strings.TrimSpace(systemContent),
			})
		}
	}

	for _, msg := range req.Messages {
		openaiMsg := transformer.OpenAIMessage{
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
						// Tool use blocks are handled elsewhere, skip silently
					case "image":
						logger.Debug("[OpenAI Transformer] Image block found but not supported")
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
	openaiReq := transformer.OpenAIRequest{
		Model:               t.model,
		Messages:            openaiMessages,
		MaxCompletionTokens: req.MaxTokens,
		Temperature:         req.Temperature,
		Stream:              req.Stream,
	}

	// Enable usage tracking for streaming requests
	if req.Stream {
		openaiReq.StreamOptions = &transformer.StreamOptions{
			IncludeUsage: true,
		}
	}

	// Convert tools to OpenAI format
	if len(req.Tools) > 0 {
		openaiTools := make([]transformer.OpenAITool, 0, len(req.Tools))
		for _, tool := range req.Tools {
			openaiTool := transformer.OpenAITool{
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
				openaiReq.ToolChoice = "required"
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
	} else if len(openaiReq.Tools) > 0 {
		// If tools are present but no tool_choice specified, default to "auto"
		// This encourages OpenAI models to use tools when appropriate
		openaiReq.ToolChoice = "auto"
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
func (t *OpenAITransformer) TransformResponseWithContext(targetResp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if isStreaming {
		if ctx == nil {
			return nil, fmt.Errorf("StreamContext is required for streaming responses")
		}
		return t.transformStreamingResponseV3(targetResp, ctx)
	}
	return t.transformNonStreamingResponse(targetResp)
}

// transformNonStreamingResponse converts OpenAI non-streaming response to Claude format
func (t *OpenAITransformer) transformNonStreamingResponse(openaiResp []byte) ([]byte, error) {
	// Check for error response first
	if isErr, errMsg := isOpenAIError(string(openaiResp)); isErr {
		return nil, fmt.Errorf("OpenAI API error: %s", errMsg)
	}

	var resp transformer.OpenAIResponse
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
					logger.Warn("[OpenAI Transformer] Failed to parse tool arguments: %v", err)
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

// isValidJSON checks if a string is valid JSON
func isValidJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

// isOpenAIError checks if the data is an OpenAI error response
func isOpenAIError(data string) (bool, string) {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(data), &errResp); err == nil && errResp.Error.Message != "" {
		return true, errResp.Error.Message
	}
	return false, ""
}

// Name returns the transformer name
func (t *OpenAITransformer) Name() string {
	return "openai"
}
