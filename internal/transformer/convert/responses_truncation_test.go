package convert

import (
	"testing"

	"github.com/lich0821/ccNexus/internal/transformer"
)

const claudeTruncatedResp = `{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"Once upon a"}],"stop_reason":"max_tokens","usage":{"input_tokens":5,"output_tokens":3}}`

const responsesTruncatedResp = `{"id":"resp_1","object":"response","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},` +
	`"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Once upon a"}]}],"usage":{"input_tokens":5,"output_tokens":3,"total_tokens":8}}`

func TestClaudeRespToOpenAI2MaxTokensReportsIncomplete(t *testing.T) {
	out, err := ClaudeRespToOpenAI2([]byte(claudeTruncatedResp))
	if err != nil {
		t.Fatalf("ClaudeRespToOpenAI2 failed: %v", err)
	}

	assertContains(t, string(out), `"status":"incomplete"`, "Expected status incomplete for a Claude max_tokens stop")
	assertContains(t, string(out), `"reason":"max_output_tokens"`, "Expected incomplete_details.reason max_output_tokens")
}

func TestClaudeStreamToOpenAI2MaxTokensReportsIncomplete(t *testing.T) {
	ctx := transformer.NewStreamContext()
	ctx.MessageID = "msg_1"

	delta := "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"max_tokens\"},\"usage\":{\"output_tokens\":3}}\n\n"
	if _, err := ClaudeStreamToOpenAI2([]byte(delta), ctx); err != nil {
		t.Fatalf("ClaudeStreamToOpenAI2 message_delta failed: %v", err)
	}
	out, err := ClaudeStreamToOpenAI2([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"), ctx)
	if err != nil {
		t.Fatalf("ClaudeStreamToOpenAI2 message_stop failed: %v", err)
	}

	assertContains(t, string(out), `"type":"response.incomplete"`, "Expected a response.incomplete event for a Claude max_tokens stop")
	assertContains(t, string(out), `"reason":"max_output_tokens"`, "Expected incomplete_details.reason max_output_tokens")
}

func TestGeminiRespToOpenAI2MaxTokensReportsIncomplete(t *testing.T) {
	out, err := GeminiRespToOpenAI2([]byte(geminiTruncatedResp))
	if err != nil {
		t.Fatalf("GeminiRespToOpenAI2 failed: %v", err)
	}

	assertContains(t, string(out), `"status":"incomplete"`, "Expected status incomplete for a Gemini MAX_TOKENS finish")
	assertContains(t, string(out), `"reason":"max_output_tokens"`, "Expected incomplete_details.reason max_output_tokens")
}

func TestGeminiStreamToOpenAI2MaxTokensReportsIncomplete(t *testing.T) {
	ctx := transformer.NewStreamContext()
	ctx.ModelName = "gemini-3.8-flash"

	out, err := GeminiStreamToOpenAI2([]byte(geminiTruncatedChunk), ctx)
	if err != nil {
		t.Fatalf("GeminiStreamToOpenAI2 failed: %v", err)
	}

	assertContains(t, string(out), `"type":"response.incomplete"`, "Expected a response.incomplete event for a Gemini MAX_TOKENS finish")
	assertContains(t, string(out), `"reason":"max_output_tokens"`, "Expected incomplete_details.reason max_output_tokens")
}

func TestOpenAI2RespToClaudeIncompleteReportsMaxTokens(t *testing.T) {
	out, err := OpenAI2RespToClaude([]byte(responsesTruncatedResp))
	if err != nil {
		t.Fatalf("OpenAI2RespToClaude failed: %v", err)
	}

	assertContains(t, string(out), `"stop_reason":"max_tokens"`, "Expected stop_reason max_tokens for an incomplete Responses result")
}

func TestOpenAI2StreamToClaudeIncompleteReportsMaxTokens(t *testing.T) {
	ctx := transformer.NewStreamContext()
	ctx.MessageID = "resp_1"

	event := "data: {\"type\":\"response.incomplete\",\"response\":" + responsesTruncatedResp + "}\n\n"
	out, err := OpenAI2StreamToClaude([]byte(event), ctx)
	if err != nil {
		t.Fatalf("OpenAI2StreamToClaude failed: %v", err)
	}

	assertContains(t, string(out), `"stop_reason":"max_tokens"`, "Expected stop_reason max_tokens for a response.incomplete event")
}
