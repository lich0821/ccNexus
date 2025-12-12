package responses

import (
	"github.com/lich0821/ccNexus/internal/transformer"
)

// OpenAI2Transformer is a passthrough transformer for Codex Responses → OpenAI Responses
type OpenAI2Transformer struct {
	model string
}

// NewOpenAI2Transformer creates a new passthrough transformer
func NewOpenAI2Transformer(model string) *OpenAI2Transformer {
	return &OpenAI2Transformer{model: model}
}

func (t *OpenAI2Transformer) Name() string {
	return "cx_resp_openai2"
}

func (t *OpenAI2Transformer) TransformRequest(req []byte) ([]byte, error) {
	return req, nil
}

func (t *OpenAI2Transformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	return resp, nil
}

func (t *OpenAI2Transformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	return resp, nil
}
