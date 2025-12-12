package convert

import (
	"encoding/json"
	"fmt"
	"strings"
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

// buildOpenAIChunk builds an OpenAI streaming chunk
func buildOpenAIChunk(id, model, content string, toolCalls []map[string]interface{}, finish string) ([]byte, error) {
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
	data, _ := json.Marshal(chunk)
	return []byte(fmt.Sprintf("data: %s\n\n", data)), nil
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
