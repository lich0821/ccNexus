package convert

import (
	"encoding/json"
	"testing"
)

const claudeReqWithImage = `{
	"model": "claude-sonnet-4-6",
	"max_tokens": 1024,
	"messages": [{"role": "user", "content": [
		{"type": "text", "text": "what is in this screenshot?"},
		{"type": "image", "source": {"type": "base64", "media_type": "image/png", "data": "QUJD"}}
	]}]
}`

func TestClaudeReqToOpenAISendsImagesAsImageURLParts(t *testing.T) {
	out, err := ClaudeReqToOpenAI([]byte(claudeReqWithImage), "gpt-test")
	if err != nil {
		t.Fatalf("ClaudeReqToOpenAI failed: %v", err)
	}
	var body struct {
		Messages []struct {
			Role    string      `json:"role"`
			Content interface{} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("converted request is not JSON: %v", err)
	}

	parts, ok := body.Messages[0].Content.([]interface{})
	if !ok || len(parts) != 2 {
		t.Fatalf("expected a content list with a text and an image part, got %v", body.Messages[0].Content)
	}
	text := parts[0].(map[string]interface{})
	if text["type"] != "text" || text["text"] != "what is in this screenshot?" {
		t.Errorf("expected the text part first, got %v", text)
	}
	image := parts[1].(map[string]interface{})
	url, _ := image["image_url"].(map[string]interface{})["url"].(string)
	if image["type"] != "image_url" || url != "data:image/png;base64,QUJD" {
		t.Errorf("expected an image_url part with a data URL, got %v", image)
	}
}

func TestClaudeReqToOpenAIKeepsStringContentWithoutImages(t *testing.T) {
	out, err := ClaudeReqToOpenAI([]byte(`{"model": "m", "messages": [{"role": "user", "content": [
		{"type": "text", "text": "hello "}, {"type": "text", "text": "world"}
	]}]}`), "gpt-test")
	if err != nil {
		t.Fatalf("ClaudeReqToOpenAI failed: %v", err)
	}
	var body struct {
		Messages []struct {
			Content interface{} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("converted request is not JSON: %v", err)
	}
	if body.Messages[0].Content != "hello world" {
		t.Errorf("expected the text joined into a string as before, got %v", body.Messages[0].Content)
	}
}

func TestClaudeReqToOpenAI2SendsImagesAsInputImage(t *testing.T) {
	out, err := ClaudeReqToOpenAI2([]byte(claudeReqWithImage), "gpt-test")
	if err != nil {
		t.Fatalf("ClaudeReqToOpenAI2 failed: %v", err)
	}
	var body struct {
		Input []struct {
			Role    string                   `json:"role"`
			Content []map[string]interface{} `json:"content"`
		} `json:"input"`
	}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("converted request is not JSON: %v", err)
	}

	if len(body.Input) != 1 || len(body.Input[0].Content) != 2 {
		t.Fatalf("expected one message with a text and an image part, got %v", body.Input)
	}
	if body.Input[0].Content[0]["type"] != "input_text" {
		t.Errorf("expected the text part first, got %v", body.Input[0].Content[0])
	}
	image := body.Input[0].Content[1]
	if image["type"] != "input_image" || image["image_url"] != "data:image/png;base64,QUJD" {
		t.Errorf("expected an input_image part with a data URL, got %v", image)
	}
}

func TestClaudeImageURLSourceIsForwarded(t *testing.T) {
	out, err := ClaudeReqToOpenAI([]byte(`{"model": "m", "messages": [{"role": "user", "content": [
		{"type": "image", "source": {"type": "url", "url": "https://example.com/a.png"}}
	]}]}`), "gpt-test")
	if err != nil {
		t.Fatalf("ClaudeReqToOpenAI failed: %v", err)
	}
	var body struct {
		Messages []struct {
			Content []map[string]interface{} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("converted request is not JSON: %v", err)
	}
	if len(body.Messages) != 1 || len(body.Messages[0].Content) != 1 {
		t.Fatalf("expected one message with one image part, got %v", body.Messages)
	}
	url, _ := body.Messages[0].Content[0]["image_url"].(map[string]interface{})["url"].(string)
	if url != "https://example.com/a.png" {
		t.Errorf("expected the image URL to be forwarded, got %q", url)
	}
}
