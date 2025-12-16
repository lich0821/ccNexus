package convert

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/transformer"
)

// OpenAIReqToGemini converts OpenAI Chat request to Gemini request
func OpenAIReqToGemini(openaiReq []byte, model string) ([]byte, error) {
	var req transformer.OpenAIRequest
	if err := json.Unmarshal(openaiReq, &req); err != nil {
		return nil, err
	}

	geminiReq := map[string]interface{}{}

	// Convert messages
	var contents []map[string]interface{}
	var systemInstruction string
	toolCallIDToName := make(map[string]string) // Map tool_call_id to function name

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if content, ok := msg.Content.(string); ok {
				systemInstruction += content + "\n"
			}
			continue
		}

		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		var parts []map[string]interface{}
		switch content := msg.Content.(type) {
		case string:
			parts = append(parts, map[string]interface{}{"text": content})
		case []interface{}:
			parts = convertOpenAIContentToGeminiParts(content)
		}

		// Handle tool_calls
		for _, tc := range msg.ToolCalls {
			if tc.ID != "" && tc.Function.Name != "" {
				toolCallIDToName[tc.ID] = tc.Function.Name
			}
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			parts = append(parts, map[string]interface{}{
				"functionCall": map[string]interface{}{"name": tc.Function.Name, "args": args},
			})
		}

		// Handle tool message
		if msg.Role == "tool" {
			funcName := toolCallIDToName[msg.ToolCallID]
			parts = []map[string]interface{}{
				{
					"functionResponse": map[string]interface{}{
						"name":     funcName,
						"response": map[string]interface{}{"result": msg.Content},
					},
				},
			}
			role = "user"
		}

		contents = append(contents, map[string]interface{}{"role": role, "parts": parts})
	}

	geminiReq["contents"] = contents

	if systemInstruction != "" {
		geminiReq["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{{"text": strings.TrimSpace(systemInstruction)}},
		}
	}

	// Generation config
	genConfig := map[string]interface{}{}
	if req.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxTokens
	} else if req.MaxCompletionTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxCompletionTokens
	}
	if req.Temperature != nil {
		genConfig["temperature"] = *req.Temperature
	}
	if len(genConfig) > 0 {
		geminiReq["generationConfig"] = genConfig
	}

	// Convert tools
	if len(req.Tools) > 0 {
		var funcDecls []map[string]interface{}
		for _, tool := range req.Tools {
			if tool.Type == "function" {
				funcDecls = append(funcDecls, map[string]interface{}{
					"name":        tool.Function.Name,
					"description": tool.Function.Description,
					"parameters":  cleanSchemaForGemini(tool.Function.Parameters),
				})
			}
		}
		if len(funcDecls) > 0 {
			geminiReq["tools"] = []map[string]interface{}{{"functionDeclarations": funcDecls}}
			// Add toolConfig to enable function calling
			geminiReq["toolConfig"] = map[string]interface{}{
				"functionCallingConfig": map[string]interface{}{
					"mode": "AUTO",
				},
			}
		}
	}

	return json.Marshal(geminiReq)
}

// GeminiRespToOpenAI converts Gemini response to OpenAI Chat response
func GeminiRespToOpenAI(geminiResp []byte, model string) ([]byte, error) {
	var resp transformer.GeminiResponse
	if err := json.Unmarshal(geminiResp, &resp); err != nil {
		return nil, err
	}

	var textContent string
	var toolCalls []map[string]interface{}
	finishReason := "stop"

	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				textContent += part.Text
			}
			if part.FunctionCall != nil {
				args, _ := json.Marshal(part.FunctionCall.Args)
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   fmt.Sprintf("call_%d", len(toolCalls)),
					"type": "function",
					"function": map[string]interface{}{
						"name":      part.FunctionCall.Name,
						"arguments": string(args),
					},
				})
				finishReason = "tool_calls"
			}
		}
	}

	message := map[string]interface{}{"role": "assistant", "content": textContent}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}

	var usage map[string]interface{}
	if resp.UsageMetadata != nil {
		usage = map[string]interface{}{
			"prompt_tokens":     resp.UsageMetadata.PromptTokenCount,
			"completion_tokens": resp.UsageMetadata.CandidatesTokenCount,
			"total_tokens":      resp.UsageMetadata.TotalTokenCount,
		}
	}

	openaiResp := map[string]interface{}{
		"id":      "gemini-resp",
		"object":  "chat.completion",
		"model":   model,
		"choices": []map[string]interface{}{{"index": 0, "message": message, "finish_reason": finishReason}},
	}
	if usage != nil {
		openaiResp["usage"] = usage
	}

	return json.Marshal(openaiResp)
}

// GeminiStreamToOpenAI converts Gemini stream chunk to OpenAI Chat stream chunk
func GeminiStreamToOpenAI(event []byte, ctx *transformer.StreamContext, model string) ([]byte, error) {
	_, jsonData := parseSSE(event)
	if jsonData == "" || jsonData == "[DONE]" {
		if jsonData == "[DONE]" {
			return []byte("data: [DONE]\n\n"), nil
		}
		return nil, nil
	}

	var resp transformer.GeminiResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		return nil, nil
	}

	if len(resp.Candidates) == 0 {
		return nil, nil
	}

	var result strings.Builder
	candidate := resp.Candidates[0]
	hasToolCall := false

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			chunk, _ := buildOpenAIChunk("gemini-chunk", model, part.Text, nil, "")
			result.Write(chunk)
		}
		if part.FunctionCall != nil {
			hasToolCall = true
			args, _ := json.Marshal(part.FunctionCall.Args)
			toolCall := []map[string]interface{}{
				{
					"index": ctx.ContentIndex,
					"id":    fmt.Sprintf("call_%d", ctx.ContentIndex),
					"type":  "function",
					"function": map[string]interface{}{
						"name":      part.FunctionCall.Name,
						"arguments": string(args),
					},
				},
			}
			chunk, _ := buildOpenAIChunk("gemini-chunk", model, "", toolCall, "")
			result.Write(chunk)
			ctx.ContentIndex++
		}
	}

	// Check for finish
	if candidate.FinishReason != "" {
		finishReason := "stop"
		if hasToolCall || candidate.FinishReason == "TOOL_CODE" {
			finishReason = "tool_calls"
		}
		chunk, _ := buildOpenAIChunk("gemini-chunk", model, "", nil, finishReason)
		result.Write(chunk)
		result.WriteString("data: [DONE]\n\n")
	}

	return []byte(result.String()), nil
}

// OpenAIStreamToGemini converts OpenAI Chat stream chunk to Gemini stream format
func OpenAIStreamToGemini(event []byte, ctx *transformer.StreamContext) ([]byte, error) {
	_, jsonData := parseSSE(event)
	if jsonData == "" || jsonData == "[DONE]" {
		if jsonData == "[DONE]" {
			return []byte("data: [DONE]\n\n"), nil
		}
		return nil, nil
	}

	var chunk transformer.OpenAIStreamChunk
	if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
		return nil, nil
	}

	if len(chunk.Choices) == 0 {
		return nil, nil
	}

	delta := chunk.Choices[0].Delta
	if delta.Content != "" {
		geminiChunk := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{"content": map[string]interface{}{"role": "model", "parts": []map[string]interface{}{{"text": delta.Content}}}},
			},
		}
		d, _ := json.Marshal(geminiChunk)
		return []byte(fmt.Sprintf("data: %s\n\n", d)), nil
	}

	return nil, nil
}

// Helper function
func convertOpenAIContentToGeminiParts(content []interface{}) []map[string]interface{} {
	var parts []map[string]interface{}
	for _, item := range content {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		switch m["type"] {
		case "text":
			parts = append(parts, map[string]interface{}{"text": m["text"]})
		case "image_url":
			if urlObj, ok := m["image_url"].(map[string]interface{}); ok {
				if url, ok := urlObj["url"].(string); ok && strings.HasPrefix(url, "data:") {
					urlParts := strings.SplitN(url, ",", 2)
					if len(urlParts) == 2 {
						mimeType := strings.TrimPrefix(strings.Split(urlParts[0], ";")[0], "data:")
						parts = append(parts, map[string]interface{}{
							"inlineData": map[string]interface{}{"mimeType": mimeType, "data": urlParts[1]},
						})
					}
				}
			}
		}
	}
	return parts
}
