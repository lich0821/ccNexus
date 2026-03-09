package convert

import (
	"encoding/json"
	"testing"

	"github.com/lich0821/ccNexus/internal/transformer"
)

func TestOpenAIReqToOpenAI2DefaultsToolChoiceAutoWhenToolsPresent(t *testing.T) {
	openaiReq := `{
		"model":"gpt-4.1",
		"stream":true,
		"messages":[{"role":"user","content":"test"}],
		"tools":[{"type":"function","function":{"name":"Write","description":"Write file","parameters":{"type":"object"}}}]
	}`

	reqBytes, err := OpenAIReqToOpenAI2([]byte(openaiReq), "gpt-4.1")
	if err != nil {
		t.Fatalf("OpenAIReqToOpenAI2 failed: %v", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		t.Fatalf("unmarshal transformed req failed: %v", err)
	}

	if req["tool_choice"] != "auto" {
		t.Fatalf("expected tool_choice=auto, got %#v", req["tool_choice"])
	}
	if _, ok := req["store"]; ok {
		t.Fatalf("did not expect store in generic openai2 conversion, got %#v", req["store"])
	}
	if _, ok := req["instructions"]; ok {
		t.Fatalf("did not expect instructions without system prompt, got %#v", req["instructions"])
	}
}

func TestOpenAI2RespToOpenAIPreservesTotalTokens(t *testing.T) {
	openai2Resp := `{
		"id":"resp_123",
		"object":"response",
		"status":"completed",
		"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],
		"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":99}
	}`

	respBytes, err := OpenAI2RespToOpenAI([]byte(openai2Resp), "gpt-4.1")
	if err != nil {
		t.Fatalf("OpenAI2RespToOpenAI failed: %v", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal transformed response failed: %v", err)
	}

	usage, ok := resp["usage"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected usage object, got %#v", resp["usage"])
	}

	if usage["total_tokens"] != float64(99) {
		t.Fatalf("expected total_tokens=99, got %#v", usage["total_tokens"])
	}
}

func TestOpenAI2StreamToOpenAIIncludesUsageOnCompleted(t *testing.T) {
	ctx := transformer.NewStreamContext()

	created := `data: {"type":"response.created","response":{"id":"resp_1","object":"response","status":"in_progress"}}`
	if out, err := OpenAI2StreamToOpenAI([]byte(created), ctx, "gpt-4.1"); err != nil {
		t.Fatalf("response.created failed: %v", err)
	} else if out != nil {
		t.Fatalf("expected nil output for response.created, got %s", string(out))
	}

	completed := `data: {"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed","usage":{"input_tokens":7,"output_tokens":3,"total_tokens":42}}}`
	out, err := OpenAI2StreamToOpenAI([]byte(completed), ctx, "gpt-4.1")
	if err != nil {
		t.Fatalf("response.completed failed: %v", err)
	}
	if out == nil {
		t.Fatal("expected transformed chunk, got nil")
	}

	_, jsonData := parseSSE(out)
	var chunk map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
		t.Fatalf("unmarshal chunk failed: %v, raw=%s", err, jsonData)
	}

	usage, ok := chunk["usage"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected usage in final chunk, got %#v", chunk["usage"])
	}
	if usage["prompt_tokens"] != float64(7) {
		t.Fatalf("expected prompt_tokens=7, got %#v", usage["prompt_tokens"])
	}
	if usage["completion_tokens"] != float64(3) {
		t.Fatalf("expected completion_tokens=3, got %#v", usage["completion_tokens"])
	}
	if usage["total_tokens"] != float64(42) {
		t.Fatalf("expected total_tokens=42, got %#v", usage["total_tokens"])
	}
}
