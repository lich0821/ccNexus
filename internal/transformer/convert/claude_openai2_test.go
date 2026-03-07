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

func TestOpenAI2StreamToClaudeCompletesWithoutDone(t *testing.T) {
	ctx := transformer.NewStreamContext()
	ctx.ModelName = "claude-3-sonnet-20240229"

	chunks := []string{
		`data: {"type":"response.created","response":{"id":"resp_1","object":"response","status":"in_progress"}}`,
		`data: {"type":"response.output_text.delta","delta":"hello"}`,
		`data: {"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed"}}`,
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
	if !strings.Contains(fullEvents, "\"type\":\"message_delta\"") {
		t.Fatalf("Expected message_delta in transformed events, got: %s", fullEvents)
	}
	if !strings.Contains(fullEvents, "event: message_stop") {
		t.Fatalf("Expected message_stop when response.completed arrives without [DONE], got: %s", fullEvents)
	}
}

func TestClaudeReqToOpenAI2PreservesToolChain(t *testing.T) {
	claudeReq := `{
		"model": "claude-sonnet-4-20250514",
		"stream": false,
		"messages": [
			{"role":"user","content":"请写文件"},
			{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Write","input":{"file_path":"/tmp/a.txt","content":"hello"}}]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}
		],
		"tools": [
			{"name":"Write","description":"Write file","input_schema":{"type":"object","properties":{"file_path":{"type":"string"},"content":{"type":"string"}},"required":["file_path","content"]}}
		]
	}`

	reqBytes, err := ClaudeReqToOpenAI2([]byte(claudeReq), "gpt-4.1")
	if err != nil {
		t.Fatalf("ClaudeReqToOpenAI2 failed: %v", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		t.Fatalf("unmarshal transformed req failed: %v", err)
	}

	input, ok := req["input"].([]interface{})
	if !ok {
		t.Fatalf("input should be array, got %T", req["input"])
	}
	if len(input) != 3 {
		t.Fatalf("expected 3 input items, got %d", len(input))
	}

	functionCall, ok := input[1].(map[string]interface{})
	if !ok || functionCall["type"] != "function_call" {
		t.Fatalf("expected input[1] function_call, got %#v", input[1])
	}
	if functionCall["call_id"] != "toolu_1" {
		t.Fatalf("expected call_id toolu_1, got %#v", functionCall["call_id"])
	}
	if _, hasID := functionCall["id"]; hasID {
		t.Fatalf("function_call.id should not be set for upstream compatibility, got %#v", functionCall["id"])
	}

	argsStr, _ := functionCall["arguments"].(string)
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
		t.Fatalf("function arguments is not valid json: %v, raw=%s", err, argsStr)
	}
	if args["file_path"] != "/tmp/a.txt" {
		t.Fatalf("unexpected function arguments: %#v", args)
	}

	functionOutput, ok := input[2].(map[string]interface{})
	if !ok || functionOutput["type"] != "function_call_output" {
		t.Fatalf("expected input[2] function_call_output, got %#v", input[2])
	}
	if functionOutput["call_id"] != "toolu_1" {
		t.Fatalf("expected output call_id toolu_1, got %#v", functionOutput["call_id"])
	}
	if functionOutput["output"] != "ok" {
		t.Fatalf("expected output ok, got %#v", functionOutput["output"])
	}

	if strings.Contains(string(reqBytes), "[Tool Call:") || strings.Contains(string(reqBytes), "[Tool Result:") {
		t.Fatalf("found legacy pseudo tool text in transformed request: %s", string(reqBytes))
	}
}

func TestOpenAI2RespToClaudeFallbackToItemID(t *testing.T) {
	openai2Resp := `{
		"id":"resp_1",
		"object":"response",
		"status":"completed",
		"output":[{"type":"function_call","id":"fc_123","name":"Write","arguments":"{\"file_path\":\"/tmp/a.txt\"}"}],
		"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}
	}`

	claudeRespBytes, err := OpenAI2RespToClaude([]byte(openai2Resp))
	if err != nil {
		t.Fatalf("OpenAI2RespToClaude failed: %v", err)
	}

	var claudeResp map[string]interface{}
	if err := json.Unmarshal(claudeRespBytes, &claudeResp); err != nil {
		t.Fatalf("unmarshal claude resp failed: %v", err)
	}

	content, ok := claudeResp["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("unexpected content: %#v", claudeResp["content"])
	}

	toolUse, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("tool_use item type invalid: %#v", content[0])
	}
	if toolUse["type"] != "tool_use" {
		t.Fatalf("expected tool_use type, got %#v", toolUse["type"])
	}
	if toolUse["id"] != "fc_123" {
		t.Fatalf("expected tool_use id from item.id fallback, got %#v", toolUse["id"])
	}
}

func TestClaudeReqToOpenAI2MapsToolChoiceAnyToRequired(t *testing.T) {
	claudeReq := `{
		"model": "claude-sonnet-4-20250514",
		"stream": true,
		"messages": [{"role":"user","content":"test"}],
		"tools": [{"name":"Write","description":"Write file","input_schema":{"type":"object"}}],
		"tool_choice": {"type":"any"}
	}`

	reqBytes, err := ClaudeReqToOpenAI2([]byte(claudeReq), "gpt-4.1")
	if err != nil {
		t.Fatalf("ClaudeReqToOpenAI2 failed: %v", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		t.Fatalf("unmarshal transformed req failed: %v", err)
	}

	if req["tool_choice"] != "required" {
		t.Fatalf("expected tool_choice=required, got %#v", req["tool_choice"])
	}
}

func TestClaudeReqToOpenAI2MapsNamedToolChoice(t *testing.T) {
	claudeReq := `{
		"model": "claude-sonnet-4-20250514",
		"stream": true,
		"messages": [{"role":"user","content":"test"}],
		"tools": [{"name":"Write","description":"Write file","input_schema":{"type":"object"}}],
		"tool_choice": {"type":"tool","name":"Write"}
	}`

	reqBytes, err := ClaudeReqToOpenAI2([]byte(claudeReq), "gpt-4.1")
	if err != nil {
		t.Fatalf("ClaudeReqToOpenAI2 failed: %v", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		t.Fatalf("unmarshal transformed req failed: %v", err)
	}

	toolChoice, ok := req["tool_choice"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected object tool_choice, got %#v", req["tool_choice"])
	}
	if toolChoice["type"] != "function" || toolChoice["name"] != "Write" {
		t.Fatalf("unexpected tool_choice mapping: %#v", toolChoice)
	}
}

func TestClaudeReqToOpenAI2DefaultsToolChoiceRequiredWhenToolsPresent(t *testing.T) {
	claudeReq := `{
		"model": "claude-sonnet-4-20250514",
		"stream": true,
		"messages": [{"role":"user","content":"test"}],
		"tools": [{"name":"Write","description":"Write file","input_schema":{"type":"object"}}]
	}`

	reqBytes, err := ClaudeReqToOpenAI2([]byte(claudeReq), "gpt-4.1")
	if err != nil {
		t.Fatalf("ClaudeReqToOpenAI2 failed: %v", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		t.Fatalf("unmarshal transformed req failed: %v", err)
	}

	if req["tool_choice"] != "required" {
		t.Fatalf("expected tool_choice=required, got %#v", req["tool_choice"])
	}
}

func TestClaudeReqToOpenAI2DefaultsToolChoiceAutoAfterToolResult(t *testing.T) {
	claudeReq := `{
		"model": "claude-sonnet-4-20250514",
		"stream": true,
		"messages": [
			{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Read","input":{"file_path":"/tmp/a"}}]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}
		],
		"tools": [{"name":"Read","description":"Read file","input_schema":{"type":"object"}}]
	}`

	reqBytes, err := ClaudeReqToOpenAI2([]byte(claudeReq), "gpt-4.1")
	if err != nil {
		t.Fatalf("ClaudeReqToOpenAI2 failed: %v", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		t.Fatalf("unmarshal transformed req failed: %v", err)
	}

	if req["tool_choice"] != "auto" {
		t.Fatalf("expected tool_choice=auto after tool_result, got %#v", req["tool_choice"])
	}
}
