package convert

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/transformer"
)

// ClaudeReqToGemini converts Claude request to Gemini request
func ClaudeReqToGemini(claudeReq []byte, model string) ([]byte, error) {
	var req transformer.ClaudeRequest
	if err := json.Unmarshal(claudeReq, &req); err != nil {
		return nil, err
	}

	geminiReq := map[string]interface{}{}

	// Convert system prompt
	if req.System != nil {
		systemText := extractSystemText(req.System)
		if systemText != "" {
			geminiReq["systemInstruction"] = map[string]interface{}{
				"parts": []map[string]interface{}{{"text": systemText}},
			}
		}
	}

	// Convert messages to contents
	var contents []map[string]interface{}
	toolUseIDToName := make(map[string]string) // Map tool_use_id to function name
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		var parts []map[string]interface{}
		switch content := msg.Content.(type) {
		case string:
			parts = append(parts, map[string]interface{}{"text": content})
		case []interface{}:
			parts = convertClaudeContentToGeminiParts(content, toolUseIDToName)
		}

		contents = append(contents, map[string]interface{}{"role": role, "parts": parts})
	}
	geminiReq["contents"] = contents

	// Generation config
	genConfig := map[string]interface{}{}
	if req.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		genConfig["temperature"] = req.Temperature
	}
	if len(genConfig) > 0 {
		geminiReq["generationConfig"] = genConfig
	}

	// Convert tools
	if len(req.Tools) > 0 {
		var funcDecls []map[string]interface{}
		for _, tool := range req.Tools {
			funcDecls = append(funcDecls, map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  cleanSchemaForGemini(tool.InputSchema),
			})
		}
		geminiReq["tools"] = []map[string]interface{}{{"functionDeclarations": funcDecls}}
		// Add toolConfig to enable function calling
		geminiReq["toolConfig"] = map[string]interface{}{
			"functionCallingConfig": map[string]interface{}{
				"mode": "AUTO",
			},
		}
	}

	return json.Marshal(geminiReq)
}

// GeminiReqToClaude converts Gemini request to Claude request
func GeminiReqToClaude(geminiReq []byte, model string) ([]byte, error) {
	var req transformer.GeminiRequest
	if err := json.Unmarshal(geminiReq, &req); err != nil {
		return nil, err
	}

	claudeReq := map[string]interface{}{
		"model":      model,
		"max_tokens": 8192,
	}

	// Convert system instruction
	if req.SystemInstruction != nil && len(req.SystemInstruction.Parts) > 0 {
		var systemParts []string
		for _, part := range req.SystemInstruction.Parts {
			if part.Text != "" {
				systemParts = append(systemParts, part.Text)
			}
		}
		if len(systemParts) > 0 {
			claudeReq["system"] = strings.Join(systemParts, "\n")
		}
	}

	// Convert contents to messages
	var messages []map[string]interface{}
	for _, content := range req.Contents {
		role := content.Role
		if role == "model" {
			role = "assistant"
		}

		var contentBlocks []map[string]interface{}
		for _, part := range content.Parts {
			if part.Thought && part.Text != "" {
				// Convert Gemini thought to Claude thinking format
				thinkingBlock := map[string]interface{}{
					"type":     "thinking",
					"thinking": part.Text,
				}
				if part.ThoughtSignature != "" {
					thinkingBlock["signature"] = part.ThoughtSignature
				}
				contentBlocks = append(contentBlocks, thinkingBlock)
			} else if part.Text != "" {
				contentBlocks = append(contentBlocks, map[string]interface{}{"type": "text", "text": part.Text})
			}
			if part.FunctionCall != nil {
				contentBlocks = append(contentBlocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    fmt.Sprintf("call_%s", part.FunctionCall.Name),
					"name":  part.FunctionCall.Name,
					"input": part.FunctionCall.Args,
				})
			}
			if part.FunctionResponse != nil {
				contentBlocks = append(contentBlocks, map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": fmt.Sprintf("call_%s", part.FunctionResponse.Name),
					"content":     part.FunctionResponse.Response,
				})
			}
		}

		if len(contentBlocks) == 1 && contentBlocks[0]["type"] == "text" {
			messages = append(messages, map[string]interface{}{"role": role, "content": contentBlocks[0]["text"]})
		} else {
			messages = append(messages, map[string]interface{}{"role": role, "content": contentBlocks})
		}
	}
	claudeReq["messages"] = messages

	// Convert generation config
	if req.GenerationConfig != nil {
		if req.GenerationConfig.MaxOutputTokens != nil {
			claudeReq["max_tokens"] = *req.GenerationConfig.MaxOutputTokens
		}
		if req.GenerationConfig.Temperature != nil {
			claudeReq["temperature"] = *req.GenerationConfig.Temperature
		}
	}

	// Convert tools
	if len(req.Tools) > 0 {
		var tools []map[string]interface{}
		for _, tool := range req.Tools {
			for _, fd := range tool.FunctionDeclarations {
				tools = append(tools, map[string]interface{}{
					"name":         fd.Name,
					"description":  fd.Description,
					"input_schema": fd.Parameters,
				})
			}
		}
		if len(tools) > 0 {
			claudeReq["tools"] = tools
		}
	}

	return json.Marshal(claudeReq)
}

// ClaudeRespToGemini converts Claude response to Gemini response
func ClaudeRespToGemini(claudeResp []byte) ([]byte, error) {
	var resp transformer.ClaudeResponse
	if err := json.Unmarshal(claudeResp, &resp); err != nil {
		return nil, err
	}

	var parts []map[string]interface{}
	for _, block := range resp.Content {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		switch blockMap["type"] {
		case "text":
			parts = append(parts, map[string]interface{}{"text": blockMap["text"]})
		case "thinking":
			// Convert Claude thinking to Gemini thought format
			part := map[string]interface{}{
				"text":    blockMap["thinking"],
				"thought": true,
			}
			if sig, ok := blockMap["signature"].(string); ok && sig != "" {
				part["thoughtSignature"] = sig
			}
			parts = append(parts, part)
		case "tool_use":
			parts = append(parts, map[string]interface{}{
				"functionCall": map[string]interface{}{
					"name": blockMap["name"],
					"args": blockMap["input"],
				},
			})
		}
	}

	finishReason := "STOP"
	if resp.StopReason == "tool_use" {
		finishReason = "TOOL_CODE"
	}

	geminiResp := map[string]interface{}{
		"candidates": []map[string]interface{}{
			{
				"content":      map[string]interface{}{"role": "model", "parts": parts},
				"finishReason": finishReason,
			},
		},
		"usageMetadata": map[string]interface{}{
			"promptTokenCount":     resp.Usage.InputTokens,
			"candidatesTokenCount": resp.Usage.OutputTokens,
			"totalTokenCount":      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	return json.Marshal(geminiResp)
}

// GeminiRespToClaude converts Gemini response to Claude response
func GeminiRespToClaude(geminiResp []byte) ([]byte, error) {
	var resp transformer.GeminiResponse
	if err := json.Unmarshal(geminiResp, &resp); err != nil {
		return nil, err
	}

	content := make([]map[string]interface{}, 0) // Initialize as empty array, not nil
	stopReason := "end_turn"

	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		for _, part := range candidate.Content.Parts {
			if part.Thought && part.Text != "" {
				// Convert Gemini thought to Claude thinking format
				thinkingBlock := map[string]interface{}{
					"type":     "thinking",
					"thinking": part.Text,
				}
				if part.ThoughtSignature != "" {
					thinkingBlock["signature"] = part.ThoughtSignature
				}
				content = append(content, thinkingBlock)
			} else if part.Text != "" {
				content = append(content, map[string]interface{}{"type": "text", "text": part.Text})
			}
			if part.FunctionCall != nil {
				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    fmt.Sprintf("call_%s", part.FunctionCall.Name),
					"name":  part.FunctionCall.Name,
					"input": part.FunctionCall.Args,
				})
				stopReason = "tool_use"
			}
		}
	}

	var inputTokens, outputTokens int
	if resp.UsageMetadata != nil {
		inputTokens = resp.UsageMetadata.PromptTokenCount
		outputTokens = resp.UsageMetadata.CandidatesTokenCount
	}

	claudeResp := map[string]interface{}{
		"id":          "gemini-resp",
		"type":        "message",
		"role":        "assistant",
		"content":     content,
		"stop_reason": stopReason,
		"usage": map[string]interface{}{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		},
	}

	return json.Marshal(claudeResp)
}

// ClaudeStreamToGemini converts Claude SSE event to Gemini stream format
func ClaudeStreamToGemini(event []byte, ctx *transformer.StreamContext) ([]byte, error) {
	eventType, jsonData := parseSSE(event)
	if jsonData == "" {
		return nil, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, nil
	}

	switch eventType {
	case "content_block_delta":
		delta, ok := data["delta"].(map[string]interface{})
		if !ok {
			return nil, nil
		}
		if delta["type"] == "text_delta" {
			text, _ := delta["text"].(string)
			chunk := map[string]interface{}{
				"candidates": []map[string]interface{}{
					{"content": map[string]interface{}{"role": "model", "parts": []map[string]interface{}{{"text": text}}}},
				},
			}
			d, _ := json.Marshal(chunk)
			return []byte(fmt.Sprintf("data: %s\n\n", d)), nil
		}

	case "message_stop":
		return []byte("data: [DONE]\n\n"), nil
	}

	return nil, nil
}

// GeminiStreamToClaude converts Gemini stream chunk to Claude SSE event
func GeminiStreamToClaude(event []byte, ctx *transformer.StreamContext) ([]byte, error) {
	_, jsonData := parseSSE(event)
	if jsonData == "" || jsonData == "[DONE]" {
		if jsonData == "[DONE]" {
			var result []byte
			if ctx.ContentBlockStarted {
				result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ContentIndex})...)
				ctx.ContentBlockStarted = false
			}
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

	var resp transformer.GeminiResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		return nil, nil
	}

	var result []byte

	// message_start
	if !ctx.MessageStartSent {
		ctx.MessageStartSent = true
		ctx.MessageID = "gemini-msg"
		result = append(result, buildClaudeEvent("message_start", map[string]interface{}{
			"message": map[string]interface{}{
				"id": ctx.MessageID, "type": "message", "role": "assistant", "content": []interface{}{},
				"model": ctx.ModelName, "stop_reason": nil, "stop_sequence": nil,
				"usage": map[string]interface{}{"input_tokens": 0, "output_tokens": 0},
			},
		})...)
	}

	if len(resp.Candidates) == 0 {
		return result, nil
	}

	candidate := resp.Candidates[0]
	hasFunctionCall := false
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			if !ctx.ContentBlockStarted {
				ctx.ContentBlockStarted = true
				result = append(result, buildClaudeEvent("content_block_start", map[string]interface{}{
					"index": ctx.ContentIndex, "content_block": map[string]interface{}{"type": "text", "text": ""},
				})...)
			}
			result = append(result, buildClaudeEvent("content_block_delta", map[string]interface{}{
				"index": ctx.ContentIndex, "delta": map[string]interface{}{"type": "text_delta", "text": part.Text},
			})...)
		}
		if part.FunctionCall != nil {
			hasFunctionCall = true
			// Close text block first if open
			if ctx.ContentBlockStarted {
				result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ContentIndex})...)
				ctx.ContentBlockStarted = false
				ctx.ContentIndex++
			}
			// Handle function call
			result = append(result, buildClaudeEvent("content_block_start", map[string]interface{}{
				"index": ctx.ContentIndex,
				"content_block": map[string]interface{}{
					"type": "tool_use", "id": fmt.Sprintf("call_%s", part.FunctionCall.Name), "name": part.FunctionCall.Name,
				},
			})...)
			args, _ := json.Marshal(part.FunctionCall.Args)
			result = append(result, buildClaudeEvent("content_block_delta", map[string]interface{}{
				"index": ctx.ContentIndex, "delta": map[string]interface{}{"type": "input_json_delta", "partial_json": string(args)},
			})...)
			result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ContentIndex})...)
			ctx.ContentIndex++
		}
	}

	// Check for finish
	if candidate.FinishReason != "" {
		if ctx.ContentBlockStarted {
			result = append(result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ContentIndex})...)
			ctx.ContentBlockStarted = false
		}
		stopReason := "end_turn"
		if hasFunctionCall || candidate.FinishReason == "TOOL_CODE" {
			stopReason = "tool_use"
		}
		result = append(result, buildClaudeEvent("message_delta", map[string]interface{}{
			"delta": map[string]interface{}{"stop_reason": stopReason, "stop_sequence": nil},
			"usage": map[string]interface{}{"output_tokens": 0},
		})...)
		result = append(result, buildClaudeEvent("message_stop", map[string]interface{}{})...)
		ctx.FinishReasonSent = true
	}

	return result, nil
}

// Helper function
func convertClaudeContentToGeminiParts(content []interface{}, toolUseIDToName map[string]string) []map[string]interface{} {
	var parts []map[string]interface{}
	for _, block := range content {
		m, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		switch m["type"] {
		case "text":
			parts = append(parts, map[string]interface{}{"text": m["text"]})
		case "thinking":
			// Convert Claude thinking to Gemini thought format
			part := map[string]interface{}{
				"text":    m["thinking"],
				"thought": true,
			}
			if sig, ok := m["signature"].(string); ok && sig != "" {
				part["thoughtSignature"] = sig
			}
			parts = append(parts, part)
		case "tool_use":
			id, _ := m["id"].(string)
			name, _ := m["name"].(string)
			if id != "" && name != "" {
				toolUseIDToName[id] = name
			}
			parts = append(parts, map[string]interface{}{
				"functionCall": map[string]interface{}{"name": name, "args": m["input"]},
			})
		case "tool_result":
			toolUseID, _ := m["tool_use_id"].(string)
			funcName := toolUseIDToName[toolUseID]
			parts = append(parts, map[string]interface{}{
				"functionResponse": map[string]interface{}{
					"name":     funcName,
					"response": map[string]interface{}{"result": m["content"]},
				},
			})
		case "image":
			if source, ok := m["source"].(map[string]interface{}); ok {
				if source["type"] == "base64" {
					parts = append(parts, map[string]interface{}{
						"inlineData": map[string]interface{}{
							"mimeType": source["media_type"],
							"data":     source["data"],
						},
					})
				}
			}
		}
	}
	return parts
}
