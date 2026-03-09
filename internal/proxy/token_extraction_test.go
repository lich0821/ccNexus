package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/transformer"
)

type passthroughResponseTransformer struct{}

func (t *passthroughResponseTransformer) Name() string { return "test_passthrough" }

func (t *passthroughResponseTransformer) TransformRequest(claudeReq []byte) ([]byte, error) {
	return claudeReq, nil
}

func (t *passthroughResponseTransformer) TransformResponse(targetResp []byte, isStreaming bool) ([]byte, error) {
	return targetResp, nil
}

func (t *passthroughResponseTransformer) TransformResponseWithContext(targetResp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	return targetResp, nil
}

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

	stringUsageResp := []byte(`{"usage":{"prompt_tokens":"11","completion_tokens":"22","total_tokens":"33"}}`)
	in, out = extractTokenUsage(stringUsageResp)
	if in != 11 || out != 22 {
		t.Fatalf("string usage parse failed: in=%d out=%d", in, out)
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

func TestExtractResponseOutputTextSupportsCommonFormats(t *testing.T) {
	openAIResp := []byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"pong"}}]}`)
	if got := extractResponseOutputText(openAIResp); got != "pong" {
		t.Fatalf("unexpected openai text: %q", got)
	}

	claudeResp := []byte(`{"content":[{"type":"text","text":"hello"},{"type":"text","text":" world"}]}`)
	if got := extractResponseOutputText(claudeResp); got != "hello world" {
		t.Fatalf("unexpected claude text: %q", got)
	}

	responsesResp := []byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`)
	if got := extractResponseOutputText(responsesResp); got != "ok" {
		t.Fatalf("unexpected responses text: %q", got)
	}
}

func TestHandleNonStreamingResponseExtractsUsageFromSSEPayloadFallback(t *testing.T) {
	endpoint := config.Endpoint{
		Name: "TokenPool",
	}
	rawSSE := strings.Join([]string{
		`data: {"type":"response.completed","response":{"usage":{"input_tokens":13,"output_tokens":8,"total_tokens":21}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(rawSSE)),
	}
	rec := httptest.NewRecorder()
	p := &Proxy{}

	in, out, err := p.handleNonStreamingResponse(rec, resp, endpoint, &passthroughResponseTransformer{})
	if err != nil {
		t.Fatalf("handleNonStreamingResponse failed: %v", err)
	}
	if in != 13 || out != 8 {
		t.Fatalf("expected usage from SSE fallback in=13 out=8, got in=%d out=%d", in, out)
	}
}
