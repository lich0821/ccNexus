package openai2

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/transformer"
)

// OpenAI2Transformer transforms between Claude and OpenAI Responses API formats
type OpenAI2Transformer struct {
	model string
}

// NewOpenAI2Transformer creates a new OpenAI2 transformer
func NewOpenAI2Transformer(model string) *OpenAI2Transformer {
	return &OpenAI2Transformer{model: model}
}

// Name returns the transformer name
func (t *OpenAI2Transformer) Name() string {
	return "openai2"
}

// TransformRequest converts Claude format request to OpenAI Responses API format
func (t *OpenAI2Transformer) TransformRequest(claudeReq []byte) ([]byte, error) {
	var req transformer.ClaudeRequest
	if err := json.Unmarshal(claudeReq, &req); err != nil {
		return nil, fmt.Errorf("failed to parse Claude request: %w", err)
	}

	openai2Req := transformer.OpenAI2Request{
		Model:  t.model,
		Stream: req.Stream,
	}

	// Note: temperature is not sent as some endpoints don't support it

	// Convert system prompt to instructions
	if req.System != nil {
		switch sys := req.System.(type) {
		case string:
			openai2Req.Instructions = sys
		case []interface{}:
			var parts []string
			for _, block := range sys {
				if blockMap, ok := block.(map[string]interface{}); ok {
					if text, ok := blockMap["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
			openai2Req.Instructions = strings.Join(parts, "\n\n")
		}
	}

	// Convert messages to input items
	inputItems := make([]transformer.OpenAI2InputItem, 0, len(req.Messages))
	for _, msg := range req.Messages {
		item := transformer.OpenAI2InputItem{
			Type: "message",
			Role: msg.Role,
		}

		switch content := msg.Content.(type) {
		case string:
			contentType := "input_text"
			if msg.Role == "assistant" {
				contentType = "output_text"
			}
			item.Content = []transformer.OpenAI2ContentPart{{Type: contentType, Text: content}}

		case []interface{}:
			item.Content = t.convertContentBlocks(content, msg.Role)
		}

		inputItems = append(inputItems, item)
	}
	openai2Req.Input = inputItems

	// Convert tools
	if len(req.Tools) > 0 {
		tools := make([]transformer.OpenAI2Tool, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, transformer.OpenAI2Tool{
				Type:        "function",
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			})
		}
		openai2Req.Tools = tools
	}

	return json.Marshal(openai2Req)
}

// convertContentBlocks converts Claude content blocks to OpenAI2 content parts
func (t *OpenAI2Transformer) convertContentBlocks(blocks []interface{}, role string) []transformer.OpenAI2ContentPart {
	var parts []transformer.OpenAI2ContentPart

	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}

		blockType, _ := blockMap["type"].(string)
		switch blockType {
		case "text":
			text, _ := blockMap["text"].(string)
			contentType := "input_text"
			if role == "assistant" {
				contentType = "output_text"
			}
			parts = append(parts, transformer.OpenAI2ContentPart{Type: contentType, Text: text})

		case "tool_use":
			id, _ := blockMap["id"].(string)
			name, _ := blockMap["name"].(string)
			input, _ := blockMap["input"].(map[string]interface{})
			args, _ := json.Marshal(input)
			parts = append(parts, transformer.OpenAI2ContentPart{
				Type:      "tool_use",
				ID:        id,
				Name:      name,
				Arguments: string(args),
			})

		case "tool_result":
			toolUseID, _ := blockMap["tool_use_id"].(string)
			output := extractToolResultContent(blockMap["content"])
			parts = append(parts, transformer.OpenAI2ContentPart{
				Type:      "tool_result",
				ToolUseID: toolUseID,
				Output:    output,
			})
		}
	}

	return parts
}

// extractToolResultContent extracts content from tool_result block
func extractToolResultContent(content interface{}) string {
	if content == nil {
		return ""
	}
	if str, ok := content.(string); ok {
		return str
	}
	if arr, ok := content.([]interface{}); ok {
		var parts []string
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	if bytes, err := json.Marshal(content); err == nil {
		return string(bytes)
	}
	return fmt.Sprintf("%v", content)
}

// TransformResponse converts OpenAI Responses API response to Claude format
func (t *OpenAI2Transformer) TransformResponse(targetResp []byte, isStreaming bool) ([]byte, error) {
	if isStreaming {
		return nil, fmt.Errorf("use TransformResponseWithContext for streaming responses")
	}
	return t.transformNonStreamingResponse(targetResp)
}

// TransformResponseWithContext converts OpenAI Responses API response to Claude format
func (t *OpenAI2Transformer) TransformResponseWithContext(targetResp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if isStreaming {
		if ctx == nil {
			return nil, fmt.Errorf("StreamContext is required for streaming responses")
		}
		return t.transformStreamingResponse(targetResp, ctx)
	}
	return t.transformNonStreamingResponse(targetResp)
}

// transformNonStreamingResponse converts non-streaming response
func (t *OpenAI2Transformer) transformNonStreamingResponse(respBytes []byte) ([]byte, error) {
	var resp transformer.OpenAI2Response
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI2 response: %w", err)
	}

	content := make([]map[string]interface{}, 0)
	stopReason := "end_turn"

	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if part.Type == "output_text" && part.Text != "" {
					content = append(content, map[string]interface{}{
						"type": "text",
						"text": part.Text,
					})
				}
			}

		case "function_call":
			var input map[string]interface{}
			if err := json.Unmarshal([]byte(item.Arguments), &input); err != nil {
				input = map[string]interface{}{"raw": item.Arguments}
			}
			content = append(content, map[string]interface{}{
				"type":  "tool_use",
				"id":    item.CallID,
				"name":  item.Name,
				"input": input,
			})
			stopReason = "tool_use"
		}
	}

	if len(content) == 0 {
		content = append(content, map[string]interface{}{"type": "text", "text": ""})
	}

	claudeResp := map[string]interface{}{
		"id":            resp.ID,
		"type":          "message",
		"role":          "assistant",
		"content":       content,
		"model":         t.model,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": map[string]interface{}{
			"input_tokens":  resp.Usage.InputTokens,
			"output_tokens": resp.Usage.OutputTokens,
		},
	}

	return json.Marshal(claudeResp)
}

// transformStreamingResponse handles streaming response transformation
func (t *OpenAI2Transformer) transformStreamingResponse(data []byte, ctx *transformer.StreamContext) ([]byte, error) {
	// Find the data: line in the event
	var jsonData string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			jsonData = strings.TrimPrefix(line, "data: ")
			break
		}
	}

	if jsonData == "" {
		return nil, nil
	}

	if jsonData == "[DONE]" {
		return t.buildMessageStop(ctx)
	}

	var event transformer.OpenAI2StreamEvent
	if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
		logger.Debug("[OpenAI2] Failed to parse stream event: %v", err)
		return nil, nil
	}

	return t.handleStreamEvent(&event, ctx)
}

// handleStreamEvent processes a single stream event
func (t *OpenAI2Transformer) handleStreamEvent(event *transformer.OpenAI2StreamEvent, ctx *transformer.StreamContext) ([]byte, error) {
	var events []map[string]interface{}

	switch event.Type {
	case "response.created":
		if event.Response != nil {
			ctx.MessageID = event.Response.ID
		}
		if !ctx.MessageStartSent {
			ctx.MessageStartSent = true
			events = append(events, t.buildMessageStart(ctx))
		}

	case "response.output_item.added":
		// Track output item type
		if event.Item != nil {
			if event.Item.Type == "message" {
				// This is the actual message output, reset content block state
				ctx.ContentBlockStarted = false
			} else if event.Item.Type == "reasoning" {
				// Start thinking block
				if !ctx.ThinkingBlockStarted {
					ctx.ThinkingBlockStarted = true
					ctx.ThinkingIndex = ctx.ContentIndex
					events = append(events, map[string]interface{}{
						"type":  "content_block_start",
						"index": ctx.ContentIndex,
						"content_block": map[string]interface{}{
							"type":     "thinking",
							"thinking": "",
						},
					})
					ctx.ContentIndex++
				}
			}
		}

	case "response.content_part.added":
		// Only start text content block for message items (not reasoning)
		if event.Item == nil || event.Item.Type != "reasoning" {
			if !ctx.ContentBlockStarted {
				ctx.ContentBlockStarted = true
				events = append(events, map[string]interface{}{
					"type":  "content_block_start",
					"index": ctx.ContentIndex,
					"content_block": map[string]interface{}{
						"type": "text",
						"text": "",
					},
				})
			}
		}

	case "response.reasoning_summary_text.delta":
		// Convert reasoning delta to thinking delta
		if event.Delta != "" && ctx.ThinkingBlockStarted {
			events = append(events, map[string]interface{}{
				"type":  "content_block_delta",
				"index": ctx.ThinkingIndex,
				"delta": map[string]interface{}{
					"type":     "thinking_delta",
					"thinking": event.Delta,
				},
			})
		}

	case "response.reasoning_summary_text.done", "response.reasoning_summary_part.done":
		// Close thinking block when reasoning is done
		if ctx.ThinkingBlockStarted {
			events = append(events, map[string]interface{}{
				"type":  "content_block_stop",
				"index": ctx.ThinkingIndex,
			})
			ctx.ThinkingBlockStarted = false
		}

	case "response.output_text.delta":
		// Only process text delta for actual message content
		if event.Delta != "" {
			if !ctx.ContentBlockStarted {
				ctx.ContentBlockStarted = true
				events = append(events, map[string]interface{}{
					"type":  "content_block_start",
					"index": ctx.ContentIndex,
					"content_block": map[string]interface{}{
						"type": "text",
						"text": "",
					},
				})
			}
			events = append(events, map[string]interface{}{
				"type":  "content_block_delta",
				"index": ctx.ContentIndex,
				"delta": map[string]interface{}{
					"type": "text_delta",
					"text": event.Delta,
				},
			})
		}

	case "response.output_text.done":
		if ctx.ContentBlockStarted {
			events = append(events, map[string]interface{}{
				"type":  "content_block_stop",
				"index": ctx.ContentIndex,
			})
			ctx.ContentIndex++
			ctx.ContentBlockStarted = false
		}

	case "response.function_call_arguments.delta":
		if !ctx.ToolBlockStarted {
			ctx.ToolBlockStarted = true
			events = append(events, map[string]interface{}{
				"type":  "content_block_start",
				"index": ctx.ContentIndex,
				"content_block": map[string]interface{}{
					"type":  "tool_use",
					"id":    event.Item.CallID,
					"name":  event.Item.Name,
					"input": map[string]interface{}{},
				},
			})
		}
		if event.Delta != "" {
			events = append(events, map[string]interface{}{
				"type":  "content_block_delta",
				"index": ctx.ContentIndex,
				"delta": map[string]interface{}{
					"type":         "input_json_delta",
					"partial_json": event.Delta,
				},
			})
		}

	case "response.function_call_arguments.done":
		if ctx.ToolBlockStarted {
			events = append(events, map[string]interface{}{
				"type":  "content_block_stop",
				"index": ctx.ContentIndex,
			})
			ctx.ContentIndex++
			ctx.ToolBlockStarted = false
		}

	case "response.completed":
		if event.Response != nil && event.Response.Usage.InputTokens > 0 {
			ctx.InputTokens = event.Response.Usage.InputTokens
			ctx.OutputTokens = event.Response.Usage.OutputTokens
		}
		// Build final stop events
		return t.buildMessageStop(ctx)

	// Ignore events that don't need conversion
	case "response.output_item.done",
		"response.content_part.done",
		"response.in_progress",
		"response.done":
		// These events are ignored for Claude format conversion
	}

	return t.marshalEvents(events)
}

// buildMessageStart creates the message_start event
func (t *OpenAI2Transformer) buildMessageStart(ctx *transformer.StreamContext) map[string]interface{} {
	return map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            ctx.MessageID,
			"type":          "message",
			"role":          "assistant",
			"content":       []interface{}{},
			"model":         t.model,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]interface{}{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	}
}

// buildMessageStop creates the message_stop event with final message_delta
func (t *OpenAI2Transformer) buildMessageStop(ctx *transformer.StreamContext) ([]byte, error) {
	if ctx.FinishReasonSent {
		return nil, nil
	}
	ctx.FinishReasonSent = true

	stopReason := "end_turn"
	if ctx.ToolBlockStarted || ctx.ContentIndex > 0 && ctx.ToolBlockStarted {
		stopReason = "tool_use"
	}

	events := []map[string]interface{}{
		{
			"type": "message_delta",
			"delta": map[string]interface{}{
				"stop_reason":   stopReason,
				"stop_sequence": nil,
			},
			"usage": map[string]interface{}{
				"output_tokens": ctx.OutputTokens,
			},
		},
		{"type": "message_stop"},
	}

	return t.marshalEvents(events)
}

// marshalEvents converts events to SSE format
func (t *OpenAI2Transformer) marshalEvents(events []map[string]interface{}) ([]byte, error) {
	if len(events) == 0 {
		return nil, nil
	}

	var result strings.Builder
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		result.WriteString("event: ")
		result.WriteString(event["type"].(string))
		result.WriteString("\ndata: ")
		result.Write(data)
		result.WriteString("\n\n")
	}

	return []byte(result.String()), nil
}
