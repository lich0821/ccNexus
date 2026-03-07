package convert

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/transformer"
)

// cleanSchemaForGemini removes fields not supported by Gemini API
func cleanSchemaForGemini(schema interface{}) interface{} {
	m, ok := schema.(map[string]interface{})
	if !ok {
		return schema
	}
	// Remove unsupported fields
	delete(m, "additionalProperties")
	delete(m, "$schema")
	if props, ok := m["properties"].(map[string]interface{}); ok {
		for k, v := range props {
			props[k] = cleanSchemaForGemini(v)
		}
	}
	if items, ok := m["items"]; ok {
		m["items"] = cleanSchemaForGemini(items)
	}
	return m
}

// parseSSE parses SSE event data
func parseSSE(data []byte) (eventType, jsonData string) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			jsonData = strings.TrimPrefix(line, "data: ")
		}
	}
	return
}

// buildClaudeEvent builds a Claude SSE event
func buildClaudeEvent(eventType string, data map[string]interface{}) []byte {
	data["type"] = eventType
	jsonData, _ := json.Marshal(data)
	return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, jsonData))
}

// buildOpenAIChunk builds an OpenAI streaming chunk without usage.
func buildOpenAIChunk(id, model, content string, toolCalls []map[string]interface{}, finish string) ([]byte, error) {
	return buildOpenAIChunkWithUsage(id, model, content, toolCalls, finish, nil)
}

// buildOpenAIChunkWithUsage builds an OpenAI streaming chunk with optional usage.
func buildOpenAIChunkWithUsage(id, model, content string, toolCalls []map[string]interface{}, finish string, usage map[string]interface{}) ([]byte, error) {
	delta := map[string]interface{}{}
	if content != "" {
		delta["content"] = content
	}
	if len(toolCalls) > 0 {
		delta["tool_calls"] = toolCalls
	}

	var finishReason interface{} = nil
	if finish != "" {
		finishReason = finish
	}

	chunk := map[string]interface{}{
		"id": id, "object": "chat.completion.chunk", "model": model,
		"choices": []map[string]interface{}{{"index": 0, "delta": delta, "finish_reason": finishReason}},
	}
	if usage != nil {
		chunk["usage"] = usage
	}
	data, _ := json.Marshal(chunk)
	return []byte(fmt.Sprintf("data: %s\n\n", data)), nil
}

// syncGeminiUsageMetadata stores Gemini usage metadata in stream context for later usage emission.
func syncGeminiUsageMetadata(resp *transformer.GeminiResponse, ctx *transformer.StreamContext) {
	if resp == nil || resp.UsageMetadata == nil || ctx == nil {
		return
	}
	if resp.UsageMetadata.PromptTokenCount > 0 {
		ctx.InputTokens = resp.UsageMetadata.PromptTokenCount
	}
	if resp.UsageMetadata.CandidatesTokenCount > 0 {
		ctx.OutputTokens = resp.UsageMetadata.CandidatesTokenCount
	}
}

func currentOpenAIUsage(ctx *transformer.StreamContext) map[string]interface{} {
	if ctx == nil || (ctx.InputTokens == 0 && ctx.OutputTokens == 0) {
		return nil
	}
	return map[string]interface{}{
		"prompt_tokens":     ctx.InputTokens,
		"completion_tokens": ctx.OutputTokens,
		"total_tokens":      ctx.InputTokens + ctx.OutputTokens,
	}
}

func currentClaudeUsage(ctx *transformer.StreamContext) map[string]interface{} {
	if ctx == nil {
		return map[string]interface{}{"input_tokens": 0, "output_tokens": 0}
	}
	return map[string]interface{}{
		"input_tokens":  ctx.InputTokens,
		"output_tokens": ctx.OutputTokens,
	}
}

// extractSystemText extracts text from Claude system prompt
func extractSystemText(system interface{}) string {
	switch s := system.(type) {
	case string:
		return s
	case []interface{}:
		var parts []string
		for _, block := range s {
			if m, ok := block.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}
