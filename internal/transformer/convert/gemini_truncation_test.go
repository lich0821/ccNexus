package convert

import (
	"testing"

	"github.com/lich0821/ccNexus/internal/transformer"
)

const geminiTruncatedResp = `{"candidates":[{"content":{"role":"model","parts":[{"text":"Once upon a"}]},"finishReason":"MAX_TOKENS"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":3,"totalTokenCount":8}}`

const geminiTruncatedChunk = "data: " + geminiTruncatedResp + "\n\n"

func TestGeminiRespToClaudeMaxTokensReportsMaxTokens(t *testing.T) {
	out, err := GeminiRespToClaude([]byte(geminiTruncatedResp))
	if err != nil {
		t.Fatalf("GeminiRespToClaude failed: %v", err)
	}

	assertContains(t, string(out), `"stop_reason":"max_tokens"`, "Expected stop_reason max_tokens for a Gemini MAX_TOKENS finish")
}

func TestGeminiStreamToClaudeMaxTokensReportsMaxTokens(t *testing.T) {
	ctx := transformer.NewStreamContext()
	ctx.ModelName = "gemini-3.8-flash"

	out, err := GeminiStreamToClaude([]byte(geminiTruncatedChunk), ctx)
	if err != nil {
		t.Fatalf("GeminiStreamToClaude failed: %v", err)
	}

	assertContains(t, string(out), `"stop_reason":"max_tokens"`, "Expected stop_reason max_tokens for a Gemini MAX_TOKENS finish")
}

func TestGeminiRespToOpenAIMaxTokensReportsLength(t *testing.T) {
	out, err := GeminiRespToOpenAI([]byte(geminiTruncatedResp), "gemini-3.8-flash")
	if err != nil {
		t.Fatalf("GeminiRespToOpenAI failed: %v", err)
	}

	assertContains(t, string(out), `"finish_reason":"length"`, "Expected finish_reason length for a Gemini MAX_TOKENS finish")
}

func TestGeminiStreamToOpenAIMaxTokensReportsLength(t *testing.T) {
	ctx := transformer.NewStreamContext()

	out, err := GeminiStreamToOpenAI([]byte(geminiTruncatedChunk), ctx, "gemini-3.8-flash")
	if err != nil {
		t.Fatalf("GeminiStreamToOpenAI failed: %v", err)
	}

	assertContains(t, string(out), `"finish_reason":"length"`, "Expected finish_reason length for a Gemini MAX_TOKENS finish")
}

func TestClaudeRespToGeminiMaxTokensReportsMaxTokens(t *testing.T) {
	claudeResp := `{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"Once upon a"}],"stop_reason":"max_tokens","usage":{"input_tokens":5,"output_tokens":3}}`

	out, err := ClaudeRespToGemini([]byte(claudeResp))
	if err != nil {
		t.Fatalf("ClaudeRespToGemini failed: %v", err)
	}

	assertContains(t, string(out), `"finishReason":"MAX_TOKENS"`, "Expected finishReason MAX_TOKENS for a Claude max_tokens stop")
}
