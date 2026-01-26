package convert

import (
	"encoding/json"
	"strings"

	"github.com/lich0821/ccNexus/internal/transformer"
)

// ClaudeReqToOpenAI converts Claude request to OpenAI Chat request
func ClaudeReqToOpenAI(claudeReq []byte, model string) ([]byte, error) {
	var req transformer.ClaudeRequest
	if err := json.Unmarshal(claudeReq, &req); err != nil {
		return nil, err
	}

	var messages []transformer.OpenAIMessage

	// Convert system prompt
	if req.System != nil {
		systemText := extractSystemText(req.System)
		if systemText != "" {
			messages = append(messages, transformer.OpenAIMessage{
				Role:    "system",
				Content: systemText,
			})
		}
	}

	// Convert messages
	for _, msg := range req.Messages {
		switch content := msg.Content.(type) {
		case string:
			messages = append(messages, transformer.OpenAIMessage{Role: msg.Role, Content: content})
		case []interface{}:
			// Check for tool_result blocks
			var textParts []string
			var toolCalls []transformer.OpenAIToolCall
			var toolResults []transformer.OpenAIMessage
			hasThinking := false

			for _, block := range content {
				m, ok := block.(map[string]interface{})
				if !ok {
					continue
				}
				switch m["type"] {
				case "text":
					if text, ok := m["text"].(string); ok {
						textParts = append(textParts, text)
					}
				case "thinking":
					// Skip thinking blocks - they are Claude's internal reasoning
					// and should not be forwarded to other APIs
					hasThinking = true
					continue
				case "tool_use":
					args, _ := json.Marshal(m["input"])
					id, ok := m["id"].(string)
					if !ok || id == "" {
						continue
					}
					name, ok := m["name"].(string)
					if !ok || name == "" {
						continue
					}
					toolCalls = append(toolCalls, transformer.OpenAIToolCall{
						ID:   id,
						Type: "function",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{Name: name, Arguments: string(args)},
					})
				case "tool_result":
					callID, ok := m["tool_use_id"].(string)
					if !ok || callID == "" {
						continue
					}
					toolResults = append(toolResults, transformer.OpenAIMessage{
						Role:       "tool",
						Content:    extractToolResultContent(m["content"]),
						ToolCallID: callID,
					})
				}
			}

			// Add main message if has text or tool_calls
			if len(textParts) > 0 || len(toolCalls) > 0 {
				openaiMsg := transformer.OpenAIMessage{Role: msg.Role}
				if len(textParts) > 0 {
					openaiMsg.Content = strings.Join(textParts, "")
				}
				if len(toolCalls) > 0 {
					openaiMsg.ToolCalls = toolCalls
				}
				messages = append(messages, openaiMsg)
			} else if hasThinking && msg.Role == "assistant" {
				messages = append(messages, transformer.OpenAIMessage{
					Role:    "assistant",
					Content: "(thinking...)",
				})
			}

			// Add tool result messages
			messages = append(messages, toolResults...)
		}
	}

	openaiReq := transformer.OpenAIRequest{
		Model:    model,
		Messages: messages,
		Stream:   req.Stream,
	}

	if req.MaxTokens > 0 {
		openaiReq.MaxCompletionTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		openaiReq.Temperature = &req.Temperature
	}

	// Convert tools
	if len(req.Tools) > 0 {
		for _, tool := range req.Tools {
			openaiReq.Tools = append(openaiReq.Tools, transformer.OpenAITool{
				Type: "function",
				Function: struct {
					Name        string                 `json:"name"`
					Description string                 `json:"description,omitempty"`
					Parameters  map[string]interface{} `json:"parameters"`
				}{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			})
		}
		// Convert tool_choice
		if req.ToolChoice != nil {
			switch tc := req.ToolChoice.(type) {
			case map[string]interface{}:
				if choiceType, _ := tc["type"].(string); choiceType == "tool" {
					if name, ok := tc["name"].(string); ok {
						openaiReq.ToolChoice = map[string]interface{}{"type": "function", "function": map[string]string{"name": name}}
					}
				} else if choiceType == "any" {
					openaiReq.ToolChoice = "required"
				} else if choiceType == "auto" {
					openaiReq.ToolChoice = "auto"
				}
			case string:
				openaiReq.ToolChoice = tc
			}
		} else {
			openaiReq.ToolChoice = "auto"
		}
	}

	// Enable usage tracking for streaming
	if req.Stream {
		openaiReq.StreamOptions = &transformer.StreamOptions{IncludeUsage: true}
	}

	return json.Marshal(openaiReq)
}

// OpenAIReqToClaude converts OpenAI Chat request to Claude request
func OpenAIReqToClaude(openaiReq []byte, model string) ([]byte, error) {
	var req transformer.OpenAIRequest
	if err := json.Unmarshal(openaiReq, &req); err != nil {
		return nil, err
	}

	claudeReq := map[string]interface{}{
		"model":      model,
		"max_tokens": 8192,
		"stream":     req.Stream,
	}

	if req.MaxTokens > 0 {
		claudeReq["max_tokens"] = req.MaxTokens
	} else if req.MaxCompletionTokens > 0 {
		claudeReq["max_tokens"] = req.MaxCompletionTokens
	}
	if req.Temperature != nil {
		claudeReq["temperature"] = *req.Temperature
	}

	// Convert messages
	var systemPrompt string
	var messages []map[string]interface{}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if content, ok := msg.Content.(string); ok {
				systemPrompt += content + "\n"
			}
			continue
		}

		claudeMsg := map[string]interface{}{"role": msg.Role}

		// Handle content
		switch content := msg.Content.(type) {
		case string:
			claudeMsg["content"] = content
		case []interface{}:
			claudeMsg["content"] = convertOpenAIContentToClaude(content)
		}

		// Handle tool_calls
		if len(msg.ToolCalls) > 0 {
			var blocks []map[string]interface{}
			if text, ok := claudeMsg["content"].(string); ok && text != "" {
				blocks = append(blocks, map[string]interface{}{"type": "text", "text": text})
			}
			for _, tc := range msg.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				blocks = append(blocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": args,
				})
			}
			claudeMsg["content"] = blocks
		}

		// Handle tool message
		if msg.Role == "tool" {
			claudeMsg["role"] = "user"
			claudeMsg["content"] = []map[string]interface{}{
				{"type": "tool_result", "tool_use_id": msg.ToolCallID, "content": msg.Content},
			}
		}

		messages = append(messages, claudeMsg)
	}

	if systemPrompt != "" {
		claudeReq["system"] = strings.TrimSpace(systemPrompt)
	}
	claudeReq["messages"] = messages

	// Convert tools
	if len(req.Tools) > 0 {
		var tools []map[string]interface{}
		for _, tool := range req.Tools {
			if tool.Type == "function" {
				tools = append(tools, map[string]interface{}{
					"name":         tool.Function.Name,
					"description":  tool.Function.Description,
					"input_schema": tool.Function.Parameters,
				})
			}
		}
		if len(tools) > 0 {
			claudeReq["tools"] = tools
		}
	}

	return json.Marshal(claudeReq)
}

// ClaudeRespToOpenAI converts Claude response to OpenAI Chat response
func ClaudeRespToOpenAI(claudeResp []byte, model string) ([]byte, error) {
	var resp transformer.ClaudeResponse
	if err := json.Unmarshal(claudeResp, &resp); err != nil {
		return nil, err
	}

	var textContent string
	var toolCalls []map[string]interface{}

	for _, block := range resp.Content {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		switch blockMap["type"] {
		case "text":
			textContent += blockMap["text"].(string)
		case "thinking":
			// Skip thinking blocks in response
			continue
		case "tool_use":
			args, _ := json.Marshal(blockMap["input"])
			toolCalls = append(toolCalls, map[string]interface{}{
				"id":   blockMap["id"],
				"type": "function",
				"function": map[string]interface{}{
					"name":      blockMap["name"],
					"arguments": string(args),
				},
			})
		}
	}

	message := map[string]interface{}{"role": "assistant", "content": textContent}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}

	finishReason := "stop"
	if resp.StopReason == "tool_use" {
		finishReason = "tool_calls"
	}

	openaiResp := map[string]interface{}{
		"id":      resp.ID,
		"object":  "chat.completion",
		"model":   model,
		"choices": []map[string]interface{}{{"index": 0, "message": message, "finish_reason": finishReason}},
		"usage": map[string]interface{}{
			"prompt_tokens":     resp.Usage.InputTokens,
			"completion_tokens": resp.Usage.OutputTokens,
			"total_tokens":      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	return json.Marshal(openaiResp)
}

// OpenAIRespToClaude converts OpenAI Chat response to Claude response
func OpenAIRespToClaude(openaiResp []byte) ([]byte, error) {
	var resp transformer.OpenAIResponse
	if err := json.Unmarshal(openaiResp, &resp); err != nil {
		return nil, err
	}

	content := make([]map[string]interface{}, 0) // Initialize as empty array, not nil
	stopReason := "end_turn"

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.Message.Content != "" {
			content = append(content, splitThinkTaggedText(choice.Message.Content)...)
		}
		for _, tc := range choice.Message.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			content = append(content, map[string]interface{}{
				"type":  "tool_use",
				"id":    tc.ID,
				"name":  tc.Function.Name,
				"input": args,
			})
			stopReason = "tool_use"
		}
	}

	claudeResp := map[string]interface{}{
		"id":          resp.ID,
		"type":        "message",
		"role":        "assistant",
		"content":     content,
		"model":       resp.Model,
		"stop_reason": stopReason,
		"usage": map[string]interface{}{
			"input_tokens":  resp.Usage.PromptTokens,
			"output_tokens": resp.Usage.CompletionTokens,
		},
	}

	return json.Marshal(claudeResp)
}

// ClaudeStreamToOpenAI converts Claude SSE event to OpenAI Chat stream chunk
func ClaudeStreamToOpenAI(event []byte, ctx *transformer.StreamContext, model string) ([]byte, error) {
	eventType, jsonData := parseSSE(event)
	if jsonData == "" {
		return nil, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, nil
	}

	switch eventType {
	case "message_start":
		if msg, ok := data["message"].(map[string]interface{}); ok {
			ctx.MessageID, _ = msg["id"].(string)
		}
		return nil, nil

	case "content_block_start":
		if block, ok := data["content_block"].(map[string]interface{}); ok {
			if block["type"] == "tool_use" {
				ctx.ToolBlockStarted = true
				ctx.CurrentToolID, _ = block["id"].(string)
				ctx.CurrentToolName, _ = block["name"].(string)
			}
		}
		return nil, nil

	case "content_block_delta":
		delta, ok := data["delta"].(map[string]interface{})
		if !ok {
			return nil, nil
		}
		switch delta["type"] {
		case "text_delta":
			text, _ := delta["text"].(string)
			return buildOpenAIChunk(ctx.MessageID, model, text, nil, "")
		case "input_json_delta":
			ctx.ToolArguments += delta["partial_json"].(string)
		}
		return nil, nil

	case "content_block_stop":
		if ctx.ToolBlockStarted {
			chunk, _ := buildOpenAIChunk(ctx.MessageID, model, "", []map[string]interface{}{
				{"index": ctx.ContentIndex, "id": ctx.CurrentToolID, "type": "function",
					"function": map[string]interface{}{"name": ctx.CurrentToolName, "arguments": ctx.ToolArguments}},
			}, "")
			ctx.ToolBlockStarted = false
			ctx.ToolArguments = ""
			ctx.ContentIndex++
			return chunk, nil
		}
		return nil, nil

	case "message_delta":
		if delta, ok := data["delta"].(map[string]interface{}); ok {
			stopReason, _ := delta["stop_reason"].(string)
			finish := "stop"
			if stopReason == "tool_use" {
				finish = "tool_calls"
			}
			return buildOpenAIChunk(ctx.MessageID, model, "", nil, finish)
		}
		return nil, nil

	case "message_stop":
		return []byte("data: [DONE]\n\n"), nil
	}

	return nil, nil
}

// OpenAIStreamToClaude converts OpenAI Chat stream chunk to Claude SSE event
func OpenAIStreamToClaude(event []byte, ctx *transformer.StreamContext) ([]byte, error) {
	_, jsonData := parseSSE(event)
	if jsonData == "" || jsonData == "[DONE]" {
		if jsonData == "[DONE]" {
			var result []byte
			emitText, emitThinking := makeThinkEmitters(ctx, &result)
			flushThinkTaggedStream(ctx, emitText, emitThinking)
			// Close any open content blocks before message_stop
			if ctx.ThinkingBlockStarted {
				result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ThinkingIndex})...)
				ctx.ThinkingBlockStarted = false
			}
			if ctx.ContentBlockStarted {
				result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ContentIndex})...)
				ctx.ContentBlockStarted = false
			}
			if ctx.ToolBlockStarted {
				result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ToolIndex})...)
				ctx.ToolBlockStarted = false
			}
			// Send message_delta with stop_reason if not sent
			if !ctx.FinishReasonSent {
				result = append(result, buildClaudeEvent("message_delta", map[string]interface{}{
					"delta": map[string]interface{}{"stop_reason": "end_turn", "stop_sequence": nil},
					"usage": map[string]interface{}{"output_tokens": 0},
				})...)
			}
			result = append(result, buildClaudeEvent("message_stop", map[string]interface{}{})...)
			return result, nil
		}
		return nil, nil
	}

	var chunk transformer.OpenAIStreamChunk
	if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
		return nil, nil
	}

	var result []byte

	// message_start
	if !ctx.MessageStartSent {
		ctx.MessageStartSent = true
		ctx.MessageID = chunk.ID
		result = append(result, buildClaudeEvent("message_start", map[string]interface{}{
			"message": map[string]interface{}{
				"id": chunk.ID, "type": "message", "role": "assistant",
				"content": []interface{}{}, "model": ctx.ModelName,
				"stop_reason": nil, "stop_sequence": nil,
				"usage": map[string]interface{}{"input_tokens": 0, "output_tokens": 0},
			},
		})...)
	}

	if len(chunk.Choices) == 0 {
		if chunk.Usage != nil {
			usageObj := map[string]interface{}{
				"input_tokens":  chunk.Usage.PromptTokens,
				"output_tokens": chunk.Usage.CompletionTokens,
			}
			msgDelta := map[string]interface{}{
				"delta": map[string]interface{}{},
				"usage": usageObj,
			}
			result = append(result, buildClaudeEvent("message_delta", msgDelta)...)
		}
		return result, nil
	}

	choice := chunk.Choices[0]
	delta := choice.Delta
	if chunk.Usage != nil && delta.Role == "" && delta.Content == "" && delta.ReasoningContent == "" && len(delta.ToolCalls) == 0 && choice.FinishReason == nil {
		usageObj := map[string]interface{}{
			"input_tokens":  chunk.Usage.PromptTokens,
			"output_tokens": chunk.Usage.CompletionTokens,
		}
		msgDelta := map[string]interface{}{
			"delta": map[string]interface{}{},
			"usage": usageObj,
		}
		result = append(result, buildClaudeEvent("message_delta", msgDelta)...)
		return result, nil
	}

	// Reasoning/Thinking content (before text content)
	if delta.ReasoningContent != "" {
		if !ctx.ThinkingBlockStarted {
			ctx.ThinkingBlockStarted = true
			ctx.ThinkingIndex = ctx.ContentIndex
			ctx.ContentIndex++
			result = append(result, buildClaudeEvent("content_block_start", map[string]interface{}{
				"index": ctx.ThinkingIndex, "content_block": map[string]interface{}{"type": "thinking", "thinking": ""},
			})...)
		}
		result = append(result, buildClaudeEvent("content_block_delta", map[string]interface{}{
			"index": ctx.ThinkingIndex, "delta": map[string]interface{}{"type": "thinking_delta", "thinking": delta.ReasoningContent},
		})...)
	}

	// Text content
	if delta.Content != "" {
		content := ctx.ThinkingBuffer + delta.Content
		ctx.ThinkingBuffer = ""

		emitText, emitThinking := makeThinkEmitters(ctx, &result)
		emitTextWithClose := func(text string) {
			if text == "" {
				return
			}
			if ctx.ThinkingBlockStarted && !ctx.ContentBlockStarted && !ctx.InThinkingTag {
				result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ThinkingIndex})...)
				ctx.ThinkingBlockStarted = false
			}
			emitText(text)
		}
		emitThinkingWithClose := func(text string) {
			if text == "" {
				return
			}
			emitThinking(text)
			if ctx.ThinkingBlockStarted {
				result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ThinkingIndex})...)
				ctx.ThinkingBlockStarted = false
			}
		}

		consumeThinkTaggedStream(content, ctx, emitTextWithClose, emitThinkingWithClose)
	}

	// Tool calls
	for _, tc := range delta.ToolCalls {
		// New tool call (has ID)
		if tc.ID != "" {
			// Close thinking block if open
			if ctx.ThinkingBlockStarted {
				result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ThinkingIndex})...)
				ctx.ThinkingBlockStarted = false
			}
			// Close text block if open
			if ctx.ContentBlockStarted {
				result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ContentIndex})...)
				ctx.ContentBlockStarted = false
				ctx.ContentIndex++
			}
			// Close previous tool block if open
			if ctx.ToolBlockStarted {
				result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ToolIndex})...)
				ctx.ContentIndex++
			}
			ctx.ToolBlockStarted = true
			ctx.ToolIndex = ctx.ContentIndex
			ctx.CurrentToolID = tc.ID
			ctx.CurrentToolName = tc.Function.Name
			ctx.ToolArguments = ""
			result = append(result, buildClaudeEvent("content_block_start", map[string]interface{}{
				"index": ctx.ToolIndex, "content_block": map[string]interface{}{"type": "tool_use", "id": tc.ID, "name": tc.Function.Name, "input": map[string]interface{}{}},
			})...)
		}
		// Accumulate arguments
		if tc.Function.Arguments != "" {
			ctx.ToolArguments += tc.Function.Arguments
			result = append(result, buildClaudeEvent("content_block_delta", map[string]interface{}{
				"index": ctx.ToolIndex, "delta": map[string]interface{}{"type": "input_json_delta", "partial_json": tc.Function.Arguments},
			})...)
		}
	}

	// Finish
	if choice.FinishReason != nil {
		if ctx.ThinkingBlockStarted {
			result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ThinkingIndex})...)
			ctx.ThinkingBlockStarted = false
		}
		if ctx.ContentBlockStarted {
			result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ContentIndex})...)
			ctx.ContentBlockStarted = false
		}
		if ctx.ToolBlockStarted {
			result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ToolIndex})...)
			ctx.ToolBlockStarted = false
		}
		stopReason := "end_turn"
		if *choice.FinishReason == "tool_calls" {
			stopReason = "tool_use"
		}
		result = append(result, buildClaudeEvent("message_delta", map[string]interface{}{
			"delta": map[string]interface{}{"stop_reason": stopReason, "stop_sequence": nil},
			"usage": map[string]interface{}{"output_tokens": 0},
		})...)
		ctx.FinishReasonSent = true
	}

	return result, nil
}

// Helper functions

func convertClaudeContentToOpenAI(content []interface{}) (interface{}, []transformer.OpenAIToolCall) {
	var textParts []string
	var toolCalls []transformer.OpenAIToolCall

	for _, block := range content {
		m, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		switch m["type"] {
		case "text":
			textParts = append(textParts, m["text"].(string))
		case "thinking":
			// Skip thinking blocks
			continue
		case "tool_use":
			args, _ := json.Marshal(m["input"])
			toolCalls = append(toolCalls, transformer.OpenAIToolCall{
				ID:   m["id"].(string),
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{Name: m["name"].(string), Arguments: string(args)},
			})
		}
	}

	if len(textParts) > 0 {
		return strings.Join(textParts, ""), toolCalls
	}
	return "", toolCalls
}

func convertOpenAIContentToClaude(content []interface{}) []map[string]interface{} {
	var result []map[string]interface{}
	for _, item := range content {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		switch m["type"] {
		case "text":
			result = append(result, map[string]interface{}{"type": "text", "text": m["text"]})
		case "image_url":
			if urlObj, ok := m["image_url"].(map[string]interface{}); ok {
				if url, ok := urlObj["url"].(string); ok && strings.HasPrefix(url, "data:") {
					parts := strings.SplitN(url, ",", 2)
					if len(parts) == 2 {
						mediaType := strings.TrimPrefix(strings.Split(parts[0], ";")[0], "data:")
						result = append(result, map[string]interface{}{
							"type":   "image",
							"source": map[string]interface{}{"type": "base64", "media_type": mediaType, "data": parts[1]},
						})
					}
				}
			}
		}
	}
	return result
}

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
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}
