package cc

import (
	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// OpenAITransformer transforms Claude Code requests to OpenAI Chat format
type OpenAITransformer struct {
	model string
}

// NewOpenAITransformer creates a new transformer
func NewOpenAITransformer(model string) *OpenAITransformer {
	return &OpenAITransformer{model: model}
}

func (t *OpenAITransformer) Name() string {
	return "cc_openai"
}

func (t *OpenAITransformer) TransformRequest(req []byte) ([]byte, error) {
	return convert.ClaudeReqToOpenAI(req, t.model)
}

func (t *OpenAITransformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	if isStreaming {
		return nil, nil
	}
	return convert.OpenAIRespToClaude(resp)
}

func (t *OpenAITransformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if isStreaming {
		return convert.OpenAIStreamToClaude(resp, ctx)
	}
	return convert.OpenAIRespToClaude(resp)
}
