package convert

import (
	"encoding/json"
	"testing"
)

const claudeReqWithSampling = `{
	"model": "claude-sonnet-4-6",
	"max_tokens": 1024,
	"temperature": 0,
	"top_p": 0.9,
	"top_k": 40,
	"stop_sequences": ["END", "STOP"],
	"messages": [{"role": "user", "content": "hi"}]
}`

const claudeReqWithoutSampling = `{
	"model": "claude-sonnet-4-6",
	"max_tokens": 1024,
	"messages": [{"role": "user", "content": "hi"}]
}`

func convertedBody(t *testing.T, out []byte, err error) map[string]interface{} {
	t.Helper()
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("converted request is not JSON: %v", err)
	}
	return body
}

func TestClaudeReqToOpenAIKeepsSamplingParams(t *testing.T) {
	out, err := ClaudeReqToOpenAI([]byte(claudeReqWithSampling), "gpt-test")
	body := convertedBody(t, out, err)

	// temperature 0 is a real value (deterministic output), not "unset"
	if v, ok := body["temperature"]; !ok || v != float64(0) {
		t.Errorf("expected temperature 0 to be kept, got %v (present=%v)", v, ok)
	}
	if body["top_p"] != 0.9 {
		t.Errorf("expected top_p 0.9, got %v", body["top_p"])
	}
	stop, _ := body["stop"].([]interface{})
	if len(stop) != 2 || stop[0] != "END" || stop[1] != "STOP" {
		t.Errorf("expected stop [END STOP], got %v", body["stop"])
	}
	if _, ok := body["top_k"]; ok {
		t.Errorf("OpenAI has no top_k, it should not be forwarded")
	}
}

func TestClaudeReqToOpenAIOmitsUnsetSamplingParams(t *testing.T) {
	out, err := ClaudeReqToOpenAI([]byte(claudeReqWithoutSampling), "gpt-test")
	body := convertedBody(t, out, err)

	for _, key := range []string{"temperature", "top_p", "stop"} {
		if v, ok := body[key]; ok {
			t.Errorf("expected %s to be omitted when the client did not set it, got %v", key, v)
		}
	}
}

func TestClaudeReqToGeminiKeepsSamplingParams(t *testing.T) {
	out, err := ClaudeReqToGemini([]byte(claudeReqWithSampling), "gemini-test")
	body := convertedBody(t, out, err)

	cfg, _ := body["generationConfig"].(map[string]interface{})
	if v, ok := cfg["temperature"]; !ok || v != float64(0) {
		t.Errorf("expected temperature 0 to be kept, got %v (present=%v)", v, ok)
	}
	if cfg["topP"] != 0.9 {
		t.Errorf("expected topP 0.9, got %v", cfg["topP"])
	}
	if cfg["topK"] != float64(40) {
		t.Errorf("expected topK 40, got %v", cfg["topK"])
	}
	stop, _ := cfg["stopSequences"].([]interface{})
	if len(stop) != 2 || stop[0] != "END" || stop[1] != "STOP" {
		t.Errorf("expected stopSequences [END STOP], got %v", cfg["stopSequences"])
	}
}

func TestClaudeReqToGeminiOmitsUnsetSamplingParams(t *testing.T) {
	out, err := ClaudeReqToGemini([]byte(claudeReqWithoutSampling), "gemini-test")
	body := convertedBody(t, out, err)

	cfg, _ := body["generationConfig"].(map[string]interface{})
	for _, key := range []string{"temperature", "topP", "topK", "stopSequences"} {
		if v, ok := cfg[key]; ok {
			t.Errorf("expected %s to be omitted when the client did not set it, got %v", key, v)
		}
	}
}
