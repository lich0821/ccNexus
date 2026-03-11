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

type noUsageStreamTransformer struct{}

func (t *noUsageStreamTransformer) Name() string {
	return "test_no_usage"
}

func (t *noUsageStreamTransformer) TransformRequest(claudeReq []byte) ([]byte, error) {
	return claudeReq, nil
}

func (t *noUsageStreamTransformer) TransformResponse(targetResp []byte, isStreaming bool) ([]byte, error) {
	return targetResp, nil
}

func (t *noUsageStreamTransformer) TransformResponseWithContext(targetResp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if !isStreaming {
		return targetResp, nil
	}
	// Simulate transformer that drops usage data.
	return []byte("data: {\"type\":\"response.completed\"}\n\n"), nil
}

func TestHandleStreamingResponseExtractsUsageFromOriginalEvent(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UpdateEndpoints([]config.Endpoint{
		{
			Name:        "TokenPool",
			APIUrl:      "https://example.com",
			APIKey:      "x",
			AuthMode:    config.AuthModeAPIKey,
			Enabled:     true,
			Transformer: "openai2",
			Model:       "gpt-4.1",
		},
	})

	p := &Proxy{config: cfg}
	endpoint := cfg.GetEndpoints()[0]
	originalSSE := strings.Join([]string{
		`data: {"type":"response.completed","response":{"usage":{"input_tokens":7,"output_tokens":5,"total_tokens":12}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(originalSSE)),
	}
	rec := httptest.NewRecorder()

	in, out, _ := p.handleStreamingResponse(
		rec,
		resp,
		endpoint,
		&noUsageStreamTransformer{},
		"cc_openai2",
		false,
		"gpt-4.1",
		[]byte(`{}`),
		0,
	)

	if in != 7 || out != 5 {
		t.Fatalf("expected tokens from original stream usage in=7 out=5, got in=%d out=%d", in, out)
	}
}
