package convert

import (
	"encoding/json"
	"testing"
)

func openAIReqToGeminiBody(t *testing.T, req string) map[string]interface{} {
	t.Helper()
	out, err := OpenAIReqToGemini([]byte(req), "gemini-test")
	if err != nil {
		t.Fatalf("OpenAIReqToGemini failed: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("converted request is not JSON: %v", err)
	}
	return body
}

func systemInstructionText(t *testing.T, body map[string]interface{}) string {
	t.Helper()
	si, ok := body["systemInstruction"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a systemInstruction, got %v", body["systemInstruction"])
	}
	parts, _ := si["parts"].([]interface{})
	if len(parts) != 1 {
		t.Fatalf("expected one systemInstruction part, got %v", parts)
	}
	return parts[0].(map[string]interface{})["text"].(string)
}

// Gemini only accepts the roles user and model in contents.
func assertOnlyGeminiRoles(t *testing.T, body map[string]interface{}) {
	t.Helper()
	contents, _ := body["contents"].([]interface{})
	for _, c := range contents {
		role := c.(map[string]interface{})["role"]
		if role != "user" && role != "model" {
			t.Errorf("Gemini contents carry the unsupported role %v", role)
		}
	}
}

func TestOpenAIReqToGeminiDeveloperMessageBecomesSystemInstruction(t *testing.T) {
	body := openAIReqToGeminiBody(t, `{
		"model": "gpt-test",
		"messages": [
			{"role": "developer", "content": "Answer in French."},
			{"role": "user", "content": "hi"}
		]
	}`)

	if got := systemInstructionText(t, body); got != "Answer in French." {
		t.Errorf("expected the developer message as systemInstruction, got %q", got)
	}
	assertOnlyGeminiRoles(t, body)
	if contents := body["contents"].([]interface{}); len(contents) != 1 {
		t.Errorf("expected only the user message in contents, got %v", contents)
	}
}

func TestOpenAIReqToGeminiSystemMessageWithContentPartsIsKept(t *testing.T) {
	body := openAIReqToGeminiBody(t, `{
		"model": "gpt-test",
		"messages": [
			{"role": "system", "content": [{"type": "text", "text": "Be brief."}, {"type": "text", "text": "No emojis."}]},
			{"role": "user", "content": "hi"}
		]
	}`)

	if got := systemInstructionText(t, body); got != "Be brief.\nNo emojis." {
		t.Errorf("expected the system parts joined, got %q", got)
	}
	assertOnlyGeminiRoles(t, body)
}

func TestOpenAIReqToGeminiJoinsSystemAndDeveloperMessages(t *testing.T) {
	body := openAIReqToGeminiBody(t, `{
		"model": "gpt-test",
		"messages": [
			{"role": "system", "content": "Be brief."},
			{"role": "developer", "content": [{"type": "text", "text": "No emojis."}]},
			{"role": "user", "content": "hi"}
		]
	}`)

	if got := systemInstructionText(t, body); got != "Be brief.\nNo emojis." {
		t.Errorf("expected both instructions in order, got %q", got)
	}
}
