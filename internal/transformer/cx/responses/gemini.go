package responses

import (
	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// GeminiTransformer transforms Codex Responses requests to Gemini format
type GeminiTransformer struct {
	model string
}

// NewGeminiTransformer creates a new transformer
func NewGeminiTransformer(model string) *GeminiTransformer {
	return &GeminiTransformer{model: model}
}

func (t *GeminiTransformer) Name() string {
	return "cx_resp_gemini"
}

func (t *GeminiTransformer) TransformRequest(req []byte) ([]byte, error) {
	return convert.OpenAI2ReqToGemini(req, t.model)
}

func (t *GeminiTransformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	if isStreaming {
		return nil, nil
	}
	return convert.GeminiRespToOpenAI2(resp)
}

func (t *GeminiTransformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if isStreaming {
		return convert.GeminiStreamToOpenAI2(resp, ctx)
	}
	return convert.GeminiRespToOpenAI2(resp)
}
