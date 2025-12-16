package responses

import (
	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// ClaudeTransformer transforms Codex Responses requests to Claude format
type ClaudeTransformer struct {
	model string
}

// NewClaudeTransformer creates a new transformer
func NewClaudeTransformer(model string) *ClaudeTransformer {
	return &ClaudeTransformer{model: model}
}

func (t *ClaudeTransformer) Name() string {
	return "cx_resp_claude"
}

func (t *ClaudeTransformer) TransformRequest(req []byte) ([]byte, error) {
	return convert.OpenAI2ReqToClaude(req, t.model)
}

func (t *ClaudeTransformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	if isStreaming {
		return nil, nil
	}
	return convert.ClaudeRespToOpenAI2(resp)
}

func (t *ClaudeTransformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if isStreaming {
		return convert.ClaudeStreamToOpenAI2(resp, ctx)
	}
	return convert.ClaudeRespToOpenAI2(resp)
}
