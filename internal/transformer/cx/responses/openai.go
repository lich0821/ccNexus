package responses

import (
	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// OpenAITransformer transforms Codex Responses requests to OpenAI Chat format
type OpenAITransformer struct {
	model string
}

// NewOpenAITransformer creates a new transformer
func NewOpenAITransformer(model string) *OpenAITransformer {
	return &OpenAITransformer{model: model}
}

func (t *OpenAITransformer) Name() string {
	return "cx_resp_openai"
}

func (t *OpenAITransformer) TransformRequest(req []byte) ([]byte, error) {
	return convert.OpenAI2ReqToOpenAI(req, t.model)
}

func (t *OpenAITransformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	if isStreaming {
		return nil, nil
	}
	return convert.OpenAIRespToOpenAI2(resp)
}

func (t *OpenAITransformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if isStreaming {
		return convert.OpenAIStreamToOpenAI2(resp, ctx)
	}
	return convert.OpenAIRespToOpenAI2(resp)
}
