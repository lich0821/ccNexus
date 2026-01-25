package convert

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lich0821/ccNexus/internal/transformer"
)

func TestOpenAI2RespToClaudeWithThinking(t *testing.T) {
	openai2Resp := `{
		"id": "resp_1",
		"object": "response",
		"status": "completed",
		"output": [{
			"type": "message",
			"role": "assistant",
			"content": [{
				"type": "output_text",
				"text": "<think>Reason</think>Answer"
			}]
		}],
		"usage": {
			"input_tokens": 3,
			"output_tokens": 5,
			"total_tokens": 8
		}
	}`

	claudeRespBytes, err := OpenAI2RespToClaude([]byte(openai2Resp))
	if err != nil {
		t.Fatalf("OpenAI2RespToClaude failed: %v", err)
	}

	var claudeResp map[string]interface{}
	if err := json.Unmarshal(claudeRespBytes, &claudeResp); err != nil {
		t.Fatalf("Failed to unmarshal Claude response: %v", err)
	}

	content, ok := claudeResp["content"].([]interface{})
	if !ok {
		t.Fatalf("Expected content to be an array, got %T", claudeResp["content"])
	}
	if len(content) != 2 {
		t.Fatalf("Expected 2 content blocks, got %d", len(content))
	}
	if content[0].(map[string]interface{})["type"] != "thinking" {
		t.Fatalf("Expected first block thinking, got %v", content[0])
	}
	if content[1].(map[string]interface{})["type"] != "text" {
		t.Fatalf("Expected second block text, got %v", content[1])
	}
}

func TestOpenAI2StreamToClaudeWithThinking(t *testing.T) {
	ctx := transformer.NewStreamContext()
	ctx.ModelName = "claude-3-sonnet-20240229"

	chunks := []string{
		`data: {"type":"response.created","response":{"id":"resp_1","object":"response","status":"in_progress"}}`,
		`data: {"type":"response.output_text.delta","delta":"<think>Reason</think>Hello"}`,
		`data: {"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed"}}`,
		`data: [DONE]`,
	}

	var allEvents []string
	for _, chunk := range chunks {
		events, err := OpenAI2StreamToClaude([]byte(chunk), ctx)
		if err != nil {
			t.Fatalf("OpenAI2StreamToClaude failed: %v", err)
		}
		if events != nil {
			allEvents = append(allEvents, string(events))
		}
	}

	fullEvents := strings.Join(allEvents, "")
	if !strings.Contains(fullEvents, "\"type\":\"thinking\"") {
		t.Fatalf("Expected thinking block start, but not found")
	}
	if !strings.Contains(fullEvents, "\"thinking\":\"Reason\"") {
		t.Fatalf("Expected thinking delta 'Reason', but not found")
	}
	if !strings.Contains(fullEvents, "\"text\":\"Hello\"") {
		t.Fatalf("Expected text delta 'Hello', but not found")
	}
	if strings.Contains(fullEvents, "<think>") || strings.Contains(fullEvents, "</think>") {
		t.Fatalf("Unexpected think tags leaked into output")
	}
}
