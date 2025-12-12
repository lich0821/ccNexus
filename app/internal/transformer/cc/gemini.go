package cc

import (
	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// GeminiTransformer transforms Claude Code requests to Gemini format
type GeminiTransformer struct {
	model string
}

// NewGeminiTransformer creates a new transformer
func NewGeminiTransformer(model string) *GeminiTransformer {
	return &GeminiTransformer{model: model}
}

func (t *GeminiTransformer) Name() string {
	return "cc_gemini"
}

func (t *GeminiTransformer) TransformRequest(req []byte) ([]byte, error) {
	return convert.ClaudeReqToGemini(req, t.model)
}

func (t *GeminiTransformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	if isStreaming {
		return nil, nil
	}
	return convert.GeminiRespToClaude(resp)
}

func (t *GeminiTransformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if isStreaming {
		return convert.GeminiStreamToClaude(resp, ctx)
	}
	return convert.GeminiRespToClaude(resp)
}
