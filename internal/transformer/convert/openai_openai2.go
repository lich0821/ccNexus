package convert

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/transformer"
)

// OpenAIReqToOpenAI2 converts OpenAI Chat request to OpenAI Responses request
func OpenAIReqToOpenAI2(openaiReq []byte, model string) ([]byte, error) {
	var req transformer.OpenAIRequest
	if err := json.Unmarshal(openaiReq, &req); err != nil {
		return nil, err
	}

	openai2Req := map[string]interface{}{
		"model":  model,
		"stream": req.Stream,
	}

	var input []map[string]interface{}
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if content, ok := msg.Content.(string); ok {
				openai2Req["instructions"] = content
			}
			continue
		}

		item := map[string]interface{}{"type": "message", "role": msg.Role}
		var contentParts []map[string]interface{}

		switch content := msg.Content.(type) {
		case string:
			textType := "input_text"
			if msg.Role == "assistant" {
				textType = "output_text"
			}
			contentParts = append(contentParts, map[string]interface{}{"type": textType, "text": content})
		}
		item["content"] = contentParts
		input = append(input, item)
	}
	openai2Req["input"] = input

	if req.MaxCompletionTokens > 0 {
		openai2Req["max_output_tokens"] = req.MaxCompletionTokens
	}

	if len(req.Tools) > 0 {
		var tools []map[string]interface{}
		for _, tool := range req.Tools {
			if tool.Type == "function" {
				tools = append(tools, map[string]interface{}{
					"type":        "function",
					"name":        tool.Function.Name,
					"description": tool.Function.Description,
					"parameters":  tool.Function.Parameters,
				})
			}
		}
		openai2Req["tools"] = tools
	}

	return json.Marshal(openai2Req)
}

// OpenAI2ReqToOpenAI converts OpenAI Responses request to OpenAI Chat request
func OpenAI2ReqToOpenAI(openai2Req []byte, model string) ([]byte, error) {
	var req transformer.OpenAI2Request
	if err := json.Unmarshal(openai2Req, &req); err != nil {
		return nil, err
	}

	var messages []transformer.OpenAIMessage

	if req.Instructions != "" {
		messages = append(messages, transformer.OpenAIMessage{Role: "system", Content: req.Instructions})
	}

	switch v := req.Input.(type) {
	case string:
		messages = append(messages, transformer.OpenAIMessage{Role: "user", Content: v})
	case []interface{}:
		var pendingToolCalls []transformer.OpenAIToolCall

		for _, item := range v {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			itemType, _ := itemMap["type"].(string)
			switch itemType {
			case "message":
				// Flush pending tool calls
				if len(pendingToolCalls) > 0 {
					messages = append(messages, transformer.OpenAIMessage{Role: "assistant", ToolCalls: pendingToolCalls})
					pendingToolCalls = nil
				}
				role, _ := itemMap["role"].(string)
				text := extractOpenAI2Text(itemMap["content"])
				messages = append(messages, transformer.OpenAIMessage{Role: role, Content: text})

			case "function_call":
				callID, _ := itemMap["call_id"].(string)
				name, _ := itemMap["name"].(string)
				args, _ := itemMap["arguments"].(string)
				pendingToolCalls = append(pendingToolCalls, transformer.OpenAIToolCall{
					ID:   callID,
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{Name: name, Arguments: args},
				})

			case "function_call_output":
				// Flush pending tool calls first
				if len(pendingToolCalls) > 0 {
					messages = append(messages, transformer.OpenAIMessage{Role: "assistant", ToolCalls: pendingToolCalls})
					pendingToolCalls = nil
				}
				callID, _ := itemMap["call_id"].(string)
				output, _ := itemMap["output"].(string)
				messages = append(messages, transformer.OpenAIMessage{Role: "tool", Content: output, ToolCallID: callID})
			}
		}

		// Flush remaining
		if len(pendingToolCalls) > 0 {
			messages = append(messages, transformer.OpenAIMessage{Role: "assistant", ToolCalls: pendingToolCalls})
		}
	}

	openaiReq := transformer.OpenAIRequest{
		Model:    model,
		Messages: messages,
		Stream:   req.Stream,
	}

	if req.MaxOutputTokens > 0 {
		openaiReq.MaxCompletionTokens = req.MaxOutputTokens
	}

	if len(req.Tools) > 0 {
		for _, tool := range req.Tools {
			var params map[string]interface{}
			switch tool.Type {
			case "function":
				params = tool.Parameters
			case "custom":
				// Custom tools (like apply_patch) use format instead of parameters
				// Convert to a function that accepts a single string input
				params = map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"input": map[string]interface{}{"type": "string", "description": "The input for this tool"},
					},
					"required": []string{"input"},
				}
			default:
				continue
			}
			openaiReq.Tools = append(openaiReq.Tools, transformer.OpenAITool{
				Type: "function",
				Function: struct {
					Name        string                 `json:"name"`
					Description string                 `json:"description,omitempty"`
					Parameters  map[string]interface{} `json:"parameters"`
				}{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  params,
				},
			})
		}
	}

	return json.Marshal(openaiReq)
}

// OpenAIRespToOpenAI2 converts OpenAI Chat response to OpenAI Responses response
func OpenAIRespToOpenAI2(openaiResp []byte) ([]byte, error) {
	var resp transformer.OpenAIResponse
	if err := json.Unmarshal(openaiResp, &resp); err != nil {
		return nil, err
	}

	var output []map[string]interface{}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.Message.Content != "" {
			output = append(output, map[string]interface{}{
				"type": "message",
				"role": "assistant",
				"content": []map[string]interface{}{
					{"type": "output_text", "text": choice.Message.Content},
				},
			})
		}
		for _, tc := range choice.Message.ToolCalls {
			output = append(output, map[string]interface{}{
				"type":      "function_call",
				"call_id":   tc.ID,
				"name":      tc.Function.Name,
				"arguments": tc.Function.Arguments,
			})
		}
	}

	openai2Resp := map[string]interface{}{
		"id":     resp.ID,
		"object": "response",
		"status": "completed",
		"output": output,
		"usage": map[string]interface{}{
			"input_tokens":  resp.Usage.PromptTokens,
			"output_tokens": resp.Usage.CompletionTokens,
			"total_tokens":  resp.Usage.TotalTokens,
		},
	}

	return json.Marshal(openai2Resp)
}

// OpenAI2RespToOpenAI converts OpenAI Responses response to OpenAI Chat response
func OpenAI2RespToOpenAI(openai2Resp []byte, model string) ([]byte, error) {
	var resp transformer.OpenAI2Response
	if err := json.Unmarshal(openai2Resp, &resp); err != nil {
		return nil, err
	}

	var textContent string
	var toolCalls []map[string]interface{}

	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if part.Type == "output_text" {
					textContent += part.Text
				}
			}
		case "function_call":
			toolCalls = append(toolCalls, map[string]interface{}{
				"id":   item.CallID,
				"type": "function",
				"function": map[string]interface{}{
					"name":      item.Name,
					"arguments": item.Arguments,
				},
			})
		}
	}

	message := map[string]interface{}{"role": "assistant", "content": textContent}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
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

// OpenAIStreamToOpenAI2 converts OpenAI Chat stream chunk to OpenAI Responses stream event
func OpenAIStreamToOpenAI2(event []byte, ctx *transformer.StreamContext) ([]byte, error) {
	_, jsonData := parseSSE(event)
	if jsonData == "" || jsonData == "[DONE]" {
		if jsonData == "[DONE]" && !ctx.FinishReasonSent {
			// Handle [DONE] if finish_reason wasn't received
			var result strings.Builder
			writeEvent := func(evt map[string]interface{}) {
				d, _ := json.Marshal(evt)
				result.WriteString(fmt.Sprintf("data: %s\n\n", d))
			}
			if ctx.ContentBlockStarted {
				writeEvent(map[string]interface{}{"type": "response.output_text.done", "output_index": 0, "content_index": 0})
				writeEvent(map[string]interface{}{"type": "response.content_part.done", "output_index": 0, "content_index": 0, "part": map[string]interface{}{"type": "output_text"}})
				writeEvent(map[string]interface{}{"type": "response.output_item.done", "output_index": 0, "item": map[string]interface{}{"type": "message", "role": "assistant", "status": "completed"}})
			}
			writeEvent(map[string]interface{}{
				"type": "response.completed",
				"response": map[string]interface{}{
					"id": ctx.MessageID, "object": "response", "status": "completed",
					"usage": map[string]interface{}{"input_tokens": ctx.InputTokens, "output_tokens": ctx.OutputTokens, "total_tokens": ctx.InputTokens + ctx.OutputTokens},
				},
			})
			result.WriteString("data: [DONE]\n\n")
			return []byte(result.String()), nil
		}
		return nil, nil
	}

	// Check for error response
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(jsonData), &errResp); err == nil && errResp.Error.Message != "" {
		return nil, fmt.Errorf("upstream error: %s", errResp.Error.Message)
	}

	var chunk transformer.OpenAIStreamChunk
	if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
		return nil, nil
	}

	var result strings.Builder
	writeEvent := func(evt map[string]interface{}) {
		d, _ := json.Marshal(evt)
		result.WriteString(fmt.Sprintf("data: %s\n\n", d))
	}

	if !ctx.MessageStartSent {
		ctx.MessageStartSent = true
		ctx.MessageID = chunk.ID
		writeEvent(map[string]interface{}{
			"type":     "response.created",
			"response": map[string]interface{}{"id": chunk.ID, "object": "response", "status": "in_progress"},
		})
	}

	if len(chunk.Choices) > 0 {
		delta := chunk.Choices[0].Delta
		finishReason := chunk.Choices[0].FinishReason

		// Handle text content
		if delta.Content != "" {
			if !ctx.ContentBlockStarted {
				ctx.ContentBlockStarted = true
				writeEvent(map[string]interface{}{
					"type": "response.output_item.added", "output_index": 0,
					"item": map[string]interface{}{"type": "message", "role": "assistant", "status": "in_progress", "content": []interface{}{}},
				})
				writeEvent(map[string]interface{}{
					"type": "response.content_part.added", "output_index": 0, "content_index": 0,
					"part": map[string]interface{}{"type": "output_text", "text": ""},
				})
			}
			writeEvent(map[string]interface{}{"type": "response.output_text.delta", "output_index": 0, "content_index": 0, "delta": delta.Content})
		}

		// Handle tool calls
		for _, tc := range delta.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			// New tool call (has ID)
			if tc.ID != "" {
				ctx.ToolCallCounter++
				ctx.CurrentToolID = tc.ID
				ctx.CurrentToolName = tc.Function.Name
				ctx.ToolArguments = ""
				writeEvent(map[string]interface{}{
					"type": "response.output_item.added", "output_index": idx + 1,
					"item": map[string]interface{}{"type": "function_call", "call_id": tc.ID, "name": tc.Function.Name, "arguments": "", "status": "in_progress"},
				})
			}
			// Accumulate arguments
			if tc.Function.Arguments != "" {
				ctx.ToolArguments += tc.Function.Arguments
				writeEvent(map[string]interface{}{
					"type": "response.function_call_arguments.delta", "output_index": idx + 1, "delta": tc.Function.Arguments,
				})
			}
		}

		// Handle finish
		if finishReason != nil && *finishReason != "" {
			if ctx.ContentBlockStarted {
				writeEvent(map[string]interface{}{"type": "response.output_text.done", "output_index": 0, "content_index": 0})
				writeEvent(map[string]interface{}{"type": "response.content_part.done", "output_index": 0, "content_index": 0, "part": map[string]interface{}{"type": "output_text"}})
				writeEvent(map[string]interface{}{"type": "response.output_item.done", "output_index": 0, "item": map[string]interface{}{"type": "message", "role": "assistant", "status": "completed"}})
				ctx.ContentBlockStarted = false
			}
			if *finishReason == "tool_calls" && ctx.CurrentToolID != "" {
				writeEvent(map[string]interface{}{"type": "response.function_call_arguments.done", "output_index": 1, "arguments": ctx.ToolArguments})
				writeEvent(map[string]interface{}{
					"type": "response.output_item.done", "output_index": 1,
					"item": map[string]interface{}{"type": "function_call", "call_id": ctx.CurrentToolID, "name": ctx.CurrentToolName, "arguments": ctx.ToolArguments, "status": "completed"},
				})
			}
			writeEvent(map[string]interface{}{
				"type": "response.completed",
				"response": map[string]interface{}{
					"id": ctx.MessageID, "object": "response", "status": "completed",
					"usage": map[string]interface{}{"input_tokens": ctx.InputTokens, "output_tokens": ctx.OutputTokens, "total_tokens": ctx.InputTokens + ctx.OutputTokens},
				},
			})
			result.WriteString("data: [DONE]\n\n")
			ctx.FinishReasonSent = true
		}
	}

	if result.Len() > 0 {
		return []byte(result.String()), nil
	}
	return nil, nil
}

// OpenAI2StreamToOpenAI converts OpenAI Responses stream event to OpenAI Chat stream chunk
func OpenAI2StreamToOpenAI(event []byte, ctx *transformer.StreamContext, model string) ([]byte, error) {
	_, jsonData := parseSSE(event)
	if jsonData == "" || jsonData == "[DONE]" {
		if jsonData == "[DONE]" {
			return []byte("data: [DONE]\n\n"), nil
		}
		return nil, nil
	}

	var evt transformer.OpenAI2StreamEvent
	if err := json.Unmarshal([]byte(jsonData), &evt); err != nil {
		return nil, nil
	}

	switch evt.Type {
	case "response.created":
		if evt.Response != nil {
			ctx.MessageID = evt.Response.ID
		}
		return nil, nil

	case "response.output_text.delta":
		return buildOpenAIChunk(ctx.MessageID, model, evt.Delta, nil, "")

	case "response.completed":
		return buildOpenAIChunk(ctx.MessageID, model, "", nil, "stop")
	}

	return nil, nil
}

func extractOpenAI2Text(content interface{}) string {
	arr, ok := content.([]interface{})
	if !ok {
		return ""
	}
	var parts []string
	for _, part := range arr {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}
		if partMap["type"] == "input_text" || partMap["type"] == "output_text" {
			if text, ok := partMap["text"].(string); ok {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "")
}
