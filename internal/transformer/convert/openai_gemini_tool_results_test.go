package convert

import (
	"encoding/json"
	"testing"
)

func geminiContents(t *testing.T, req string) []map[string]interface{} {
	t.Helper()
	out, err := OpenAIReqToGemini([]byte(req), "gemini-test")
	if err != nil {
		t.Fatalf("OpenAIReqToGemini failed: %v", err)
	}
	var body struct {
		Contents []map[string]interface{} `json:"contents"`
	}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("converted request is not JSON: %v", err)
	}
	return body.Contents
}

func partsOf(t *testing.T, content map[string]interface{}) []interface{} {
	t.Helper()
	parts, _ := content["parts"].([]interface{})
	return parts
}

func functionResponseName(t *testing.T, part interface{}) string {
	t.Helper()
	fr, ok := part.(map[string]interface{})["functionResponse"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a functionResponse part, got %v", part)
	}
	return fr["name"].(string)
}

const parallelToolCallsPrefix = `{"role": "user", "content": "weather and time?"},
	{"role": "assistant", "content": null, "tool_calls": [
		{"id": "call_a", "type": "function", "function": {"name": "get_weather", "arguments": "{\"city\":\"Paris\"}"}},
		{"id": "call_b", "type": "function", "function": {"name": "get_time", "arguments": "{\"city\":\"Paris\"}"}}
	]}`

// Gemini wants the results of one parallel function call turn in a single content, with as many
// functionResponse parts as there were functionCall parts; separate contents are rejected.
func TestOpenAIReqToGeminiGroupsParallelToolResultsInOneContent(t *testing.T) {
	contents := geminiContents(t, `{"model": "gpt-test", "messages": [`+parallelToolCallsPrefix+`,
		{"role": "tool", "tool_call_id": "call_a", "content": "sunny"},
		{"role": "tool", "tool_call_id": "call_b", "content": "12:00"}
	]}`)

	if len(contents) != 3 {
		t.Fatalf("expected user, model and one tool-result content, got %d: %v", len(contents), contents)
	}
	if contents[2]["role"] != "user" {
		t.Errorf("expected the results in a user content, got role %v", contents[2]["role"])
	}
	parts := partsOf(t, contents[2])
	if len(parts) != 2 {
		t.Fatalf("expected 2 functionResponse parts, got %d", len(parts))
	}
	if functionResponseName(t, parts[0]) != "get_weather" || functionResponseName(t, parts[1]) != "get_time" {
		t.Errorf("expected the results in call order, got %v", parts)
	}
}

func TestOpenAIReqToGeminiKeepsASingleToolResultAndLaterTurnsApart(t *testing.T) {
	contents := geminiContents(t, `{"model": "gpt-test", "messages": [
		{"role": "user", "content": "weather?"},
		{"role": "assistant", "content": null, "tool_calls": [
			{"id": "call_a", "type": "function", "function": {"name": "get_weather", "arguments": "{}"}}
		]},
		{"role": "tool", "tool_call_id": "call_a", "content": "sunny"},
		{"role": "user", "content": "and tomorrow?"},
		{"role": "assistant", "content": null, "tool_calls": [
			{"id": "call_b", "type": "function", "function": {"name": "get_forecast", "arguments": "{}"}}
		]},
		{"role": "tool", "tool_call_id": "call_b", "content": "rain"}
	]}`)

	if len(contents) != 6 {
		t.Fatalf("expected 6 contents (results of different turns are not merged), got %d", len(contents))
	}
	for _, i := range []int{2, 5} {
		if n := len(partsOf(t, contents[i])); n != 1 {
			t.Errorf("content %d: expected a single functionResponse part, got %d", i, n)
		}
	}
	if contents[3]["role"] != "user" {
		t.Errorf("the user message after a tool result must stay its own content")
	}
}
