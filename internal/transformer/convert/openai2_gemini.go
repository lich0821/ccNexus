package convert

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/transformer"
)

// OpenAI2ReqToGemini converts OpenAI Responses API request to Gemini request
func OpenAI2ReqToGemini(openai2Req []byte, model string) ([]byte, error) {
	var req transformer.OpenAI2Request
	if err := json.Unmarshal(openai2Req, &req); err != nil {
		return nil, err
	}

	geminiReq := map[string]interface{}{}

	// Convert instructions to system instruction
	if req.Instructions != "" {
		geminiReq["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{{"text": req.Instructions}},
		}
	}

	// Convert input to contents
	contents := convertOpenAI2InputToGeminiContents(req.Input)
	geminiReq["contents"] = contents

	// Generation config
	genConfig := map[string]interface{}{}
	if req.MaxOutputTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxOutputTokens
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
			var params map[string]interface{}
			switch tool.Type {
			case "function":
				params = tool.Parameters
			case "custom":
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
			funcDecls = append(funcDecls, map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  cleanSchemaForGemini(params),
			})
		}
		if len(funcDecls) > 0 {
			geminiReq["tools"] = []map[string]interface{}{{"functionDeclarations": funcDecls}}
		}
	}

	return json.Marshal(geminiReq)
}

// GeminiRespToOpenAI2 converts Gemini response to OpenAI Responses API response
func GeminiRespToOpenAI2(geminiResp []byte) ([]byte, error) {
	var resp transformer.GeminiResponse
	if err := json.Unmarshal(geminiResp, &resp); err != nil {
		return nil, err
	}

	var outputContent []map[string]interface{}
	var functionCalls []map[string]interface{}

	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				outputContent = append(outputContent, map[string]interface{}{
					"type": "output_text",
					"text": part.Text,
				})
			}
			if part.FunctionCall != nil {
				args, _ := json.Marshal(part.FunctionCall.Args)
				functionCalls = append(functionCalls, map[string]interface{}{
					"type":      "function_call",
					"id":        fmt.Sprintf("call_%d", len(functionCalls)),
					"call_id":   fmt.Sprintf("call_%d", len(functionCalls)),
					"name":      part.FunctionCall.Name,
					"arguments": string(args),
				})
			}
		}
	}

	var output []map[string]interface{}
	if len(outputContent) > 0 {
		output = append(output, map[string]interface{}{
			"type":    "message",
			"role":    "assistant",
			"content": outputContent,
		})
	}
	output = append(output, functionCalls...)

	var usage map[string]interface{}
	if resp.UsageMetadata != nil {
		usage = map[string]interface{}{
			"input_tokens":  resp.UsageMetadata.PromptTokenCount,
			"output_tokens": resp.UsageMetadata.CandidatesTokenCount,
			"total_tokens":  resp.UsageMetadata.TotalTokenCount,
		}
	}

	openai2Resp := map[string]interface{}{
		"id":     "gemini-resp",
		"object": "response",
		"status": "completed",
		"output": output,
	}
	if usage != nil {
		openai2Resp["usage"] = usage
	}

	return json.Marshal(openai2Resp)
}

// GeminiStreamToOpenAI2 converts Gemini stream chunk to OpenAI Responses stream event
func GeminiStreamToOpenAI2(event []byte, ctx *transformer.StreamContext) ([]byte, error) {
	_, jsonData := parseSSE(event)
	if jsonData == "" || jsonData == "[DONE]" {
		if jsonData == "[DONE]" {
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
			Code    int    `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(jsonData), &errResp); err == nil && errResp.Error.Message != "" {
		return nil, fmt.Errorf("upstream error: %s", errResp.Error.Message)
	}

	var resp transformer.GeminiResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		return nil, nil
	}

	if len(resp.Candidates) == 0 {
		return nil, nil
	}

	var result strings.Builder
	writeEvent := func(evt map[string]interface{}) {
		d, _ := json.Marshal(evt)
		result.WriteString(fmt.Sprintf("data: %s\n\n", d))
	}

	// Send response.created on first chunk
	if !ctx.MessageStartSent {
		ctx.MessageStartSent = true
		ctx.MessageID = "gemini-resp"
		writeEvent(map[string]interface{}{
			"type": "response.created",
			"response": map[string]interface{}{"id": ctx.MessageID, "object": "response", "status": "in_progress"},
		})
	}

	candidate := resp.Candidates[0]
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
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
			writeEvent(map[string]interface{}{"type": "response.output_text.delta", "output_index": 0, "content_index": 0, "delta": part.Text})
		}
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			callID := fmt.Sprintf("call_%d", ctx.ToolCallCounter)
			ctx.ToolCallCounter++
			writeEvent(map[string]interface{}{
				"type": "response.output_item.added", "output_index": ctx.ToolCallCounter,
				"item": map[string]interface{}{"type": "function_call", "call_id": callID, "name": part.FunctionCall.Name, "arguments": "", "status": "in_progress"},
			})
			writeEvent(map[string]interface{}{"type": "response.function_call_arguments.done", "output_index": ctx.ToolCallCounter, "arguments": string(args)})
			writeEvent(map[string]interface{}{
				"type": "response.output_item.done", "output_index": ctx.ToolCallCounter,
				"item": map[string]interface{}{"type": "function_call", "call_id": callID, "name": part.FunctionCall.Name, "arguments": string(args), "status": "completed"},
			})
		}
	}

	// Check for finish
	if candidate.FinishReason != "" {
		if ctx.ContentBlockStarted {
			writeEvent(map[string]interface{}{"type": "response.output_text.done", "output_index": 0, "content_index": 0})
			writeEvent(map[string]interface{}{"type": "response.content_part.done", "output_index": 0, "content_index": 0, "part": map[string]interface{}{"type": "output_text"}})
			writeEvent(map[string]interface{}{"type": "response.output_item.done", "output_index": 0, "item": map[string]interface{}{"type": "message", "role": "assistant", "status": "completed"}})
			ctx.ContentBlockStarted = false
		}
		writeEvent(map[string]interface{}{
			"type": "response.completed",
			"response": map[string]interface{}{
				"id": ctx.MessageID, "object": "response", "status": "completed",
				"usage": map[string]interface{}{"input_tokens": ctx.InputTokens, "output_tokens": ctx.OutputTokens, "total_tokens": ctx.InputTokens + ctx.OutputTokens},
			},
		})
		result.WriteString("data: [DONE]\n\n")
	}

	return []byte(result.String()), nil
}

// OpenAI2StreamToGemini converts OpenAI Responses stream event to Gemini stream format
func OpenAI2StreamToGemini(event []byte, ctx *transformer.StreamContext) ([]byte, error) {
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
	case "response.output_text.delta":
		chunk := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{"content": map[string]interface{}{"role": "model", "parts": []map[string]interface{}{{"text": evt.Delta}}}},
			},
		}
		d, _ := json.Marshal(chunk)
		return []byte(fmt.Sprintf("data: %s\n\n", d)), nil
	}

	return nil, nil
}

// Helper function
func convertOpenAI2InputToGeminiContents(input interface{}) []map[string]interface{} {
	var contents []map[string]interface{}

	switch v := input.(type) {
	case string:
		contents = append(contents, map[string]interface{}{
			"role":  "user",
			"parts": []map[string]interface{}{{"text": v}},
		})
	case []interface{}:
		var pendingFuncCalls []map[string]interface{}
		var pendingFuncResponses []map[string]interface{}
		callIDToName := make(map[string]string) // Map call_id to function name

		for _, item := range v {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			itemType, _ := itemMap["type"].(string)
			switch itemType {
			case "message":
				// Flush pending function calls
				if len(pendingFuncCalls) > 0 {
					contents = append(contents, map[string]interface{}{"role": "model", "parts": pendingFuncCalls})
					pendingFuncCalls = nil
				}
				// Flush pending function responses
				if len(pendingFuncResponses) > 0 {
					contents = append(contents, map[string]interface{}{"role": "user", "parts": pendingFuncResponses})
					pendingFuncResponses = nil
				}

				role, _ := itemMap["role"].(string)
				if role == "assistant" {
					role = "model"
				}
				parts := convertOpenAI2ContentToGeminiParts(itemMap["content"])
				contents = append(contents, map[string]interface{}{"role": role, "parts": parts})

			case "function_call":
				name, _ := itemMap["name"].(string)
				callID, _ := itemMap["call_id"].(string)
				if callID != "" && name != "" {
					callIDToName[callID] = name
				}
				argsStr, _ := itemMap["arguments"].(string)
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
					args = map[string]interface{}{}
				}
				pendingFuncCalls = append(pendingFuncCalls, map[string]interface{}{
					"functionCall": map[string]interface{}{"name": name, "args": args},
				})

			case "function_call_output":
				// Flush pending function calls first
				if len(pendingFuncCalls) > 0 {
					contents = append(contents, map[string]interface{}{"role": "model", "parts": pendingFuncCalls})
					pendingFuncCalls = nil
				}
				callID, _ := itemMap["call_id"].(string)
				name := callIDToName[callID]
				output, _ := itemMap["output"].(string)
				pendingFuncResponses = append(pendingFuncResponses, map[string]interface{}{
					"functionResponse": map[string]interface{}{"name": name, "response": map[string]interface{}{"result": output}},
				})
			}
		}

		// Flush remaining
		if len(pendingFuncCalls) > 0 {
			contents = append(contents, map[string]interface{}{"role": "model", "parts": pendingFuncCalls})
		}
		if len(pendingFuncResponses) > 0 {
			contents = append(contents, map[string]interface{}{"role": "user", "parts": pendingFuncResponses})
		}
	}

	return contents
}

func convertOpenAI2ContentToGeminiParts(content interface{}) []map[string]interface{} {
	var parts []map[string]interface{}

	arr, ok := content.([]interface{})
	if !ok {
		if str, ok := content.(string); ok {
			return []map[string]interface{}{{"text": str}}
		}
		return parts
	}

	for _, part := range arr {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}
		switch partMap["type"] {
		case "input_text", "output_text":
			parts = append(parts, map[string]interface{}{"text": partMap["text"]})
		}
	}

	return parts
}

