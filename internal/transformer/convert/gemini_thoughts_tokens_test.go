package convert

import (
	"strings"
	"testing"

	"github.com/lich0821/ccNexus/internal/transformer"
)

// Gemini reports reasoning tokens in thoughtsTokenCount, outside candidatesTokenCount,
// and bills them as output.
const geminiThinkingResp = `{"candidates":[{"content":{"role":"model","parts":[{"text":"4"}]},"finishReason":"STOP"}],` +
	`"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":3,"thoughtsTokenCount":7,"totalTokenCount":15}}`

func TestGeminiRespToClaudeCountsThoughtTokensAsOutput(t *testing.T) {
	out, err := GeminiRespToClaude([]byte(geminiThinkingResp))
	if err != nil {
		t.Fatalf("GeminiRespToClaude failed: %v", err)
	}

	assertContains(t, string(out), `"output_tokens":10`, "Expected output_tokens to include the 7 thought tokens")
}

func TestGeminiRespToOpenAICountsThoughtTokensAsCompletion(t *testing.T) {
	out, err := GeminiRespToOpenAI([]byte(geminiThinkingResp), "gemini-3.8-flash")
	if err != nil {
		t.Fatalf("GeminiRespToOpenAI failed: %v", err)
	}

	assertContains(t, string(out), `"completion_tokens":10`, "Expected completion_tokens to include the 7 thought tokens")
}

func TestGeminiStreamToClaudeCountsThoughtTokensAsOutput(t *testing.T) {
	ctx := transformer.NewStreamContext()
	ctx.ModelName = "gemini-3.8-flash"

	out, err := GeminiStreamToClaude([]byte("data: "+geminiThinkingResp+"\n\n"), ctx)
	if err != nil {
		t.Fatalf("GeminiStreamToClaude failed: %v", err)
	}

	if !strings.Contains(string(out), `"output_tokens":10`) {
		t.Errorf("Expected output_tokens 10 (3 candidate + 7 thought) in the stream usage, got: %s", out)
	}
}
