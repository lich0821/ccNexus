package proxy

import (
	"strings"
	"testing"
)

func TestExtractTokenUsageSupportsClaudeAndOpenAIFormats(t *testing.T) {
	claudeResp := []byte(`{"usage":{"input_tokens":12,"output_tokens":34}}`)
	in, out := extractTokenUsage(claudeResp)
	if in != 12 || out != 34 {
		t.Fatalf("claude usage parse failed: in=%d out=%d", in, out)
	}

	openAIResp := []byte(`{"usage":{"prompt_tokens":56,"completion_tokens":78,"total_tokens":134}}`)
	in, out = extractTokenUsage(openAIResp)
	if in != 56 || out != 78 {
		t.Fatalf("openai usage parse failed: in=%d out=%d", in, out)
	}
}

func TestExtractTokensFromEventSupportsResponsesAndOpenAIChunk(t *testing.T) {
	p := &Proxy{}
	in, out := 0, 0

	responsesCompleted := []byte("data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":101,\"output_tokens\":202,\"total_tokens\":303}}}\n\n")
	p.extractTokensFromEvent(responsesCompleted, &in, &out)
	if in != 101 || out != 202 {
		t.Fatalf("responses usage parse failed: in=%d out=%d", in, out)
	}

	openAIChunk := []byte("data: {\"id\":\"cmpl-1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":null}],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":9,\"total_tokens\":16}}\n\n")
	p.extractTokensFromEvent(openAIChunk, &in, &out)
	if in != 7 || out != 9 {
		t.Fatalf("openai chunk usage parse failed: in=%d out=%d", in, out)
	}
}

func TestExtractTextFromEventSupportsResponsesAndOpenAIFormats(t *testing.T) {
	p := &Proxy{}
	var output strings.Builder

	responsesDelta := []byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n")
	openAIChunk := []byte("data: {\"id\":\"cmpl-1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" world\"},\"finish_reason\":null}]}\n\n")
	claudeDelta := []byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"!\"}}\n\n")

	p.extractTextFromEvent(responsesDelta, &output)
	p.extractTextFromEvent(openAIChunk, &output)
	p.extractTextFromEvent(claudeDelta, &output)

	if got := output.String(); got != "hello world!" {
		t.Fatalf("unexpected extracted text: %q", got)
	}
}
