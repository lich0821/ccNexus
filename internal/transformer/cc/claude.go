package cc

import (
	"github.com/lich0821/ccNexus/internal/transformer"
)

// ClaudeTransformer is a passthrough transformer for Claude Code → Claude endpoint
type ClaudeTransformer struct {
	model string
}

// NewClaudeTransformer creates a new passthrough transformer
func NewClaudeTransformer() *ClaudeTransformer {
	return &ClaudeTransformer{}
}

// NewClaudeTransformerWithModel creates a transformer with model override
func NewClaudeTransformerWithModel(model string) *ClaudeTransformer {
	return &ClaudeTransformer{model: model}
}

func (t *ClaudeTransformer) Name() string {
	return "cc_claude"
}

func (t *ClaudeTransformer) TransformRequest(req []byte) ([]byte, error) {
	return req, nil
}

func (t *ClaudeTransformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	return resp, nil
}

func (t *ClaudeTransformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	return resp, nil
}
