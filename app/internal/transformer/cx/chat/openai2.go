package chat

import (
	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// OpenAI2Transformer transforms Codex Chat requests to OpenAI Responses format
type OpenAI2Transformer struct {
	model string
}

// NewOpenAI2Transformer creates a new transformer
func NewOpenAI2Transformer(model string) *OpenAI2Transformer {
	return &OpenAI2Transformer{model: model}
}

func (t *OpenAI2Transformer) Name() string {
	return "cx_chat_openai2"
}

func (t *OpenAI2Transformer) TransformRequest(req []byte) ([]byte, error) {
	return convert.OpenAIReqToOpenAI2(req, t.model)
}

func (t *OpenAI2Transformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	if isStreaming {
		return nil, nil
	}
	return convert.OpenAI2RespToOpenAI(resp, t.model)
}

func (t *OpenAI2Transformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if isStreaming {
		return convert.OpenAI2StreamToOpenAI(resp, ctx, t.model)
	}
	return convert.OpenAI2RespToOpenAI(resp, t.model)
}
