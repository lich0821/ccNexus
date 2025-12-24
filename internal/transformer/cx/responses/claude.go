package responses

import (
	"encoding/json"

	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// ClaudeTransformer transforms Codex Responses requests to Claude format
type ClaudeTransformer struct {
	model          string
	modelRedirects map[string]string
}

// NewClaudeTransformer creates a new transformer
func NewClaudeTransformer(model string) *ClaudeTransformer {
	return &ClaudeTransformer{model: model}
}

// NewClaudeTransformerWithRedirects creates a transformer with model and model redirects
func NewClaudeTransformerWithRedirects(model string, redirects map[string]string) *ClaudeTransformer {
	return &ClaudeTransformer{
		model:          model,
		modelRedirects: redirects,
	}
}

func (t *ClaudeTransformer) Name() string {
	return "cx_resp_claude"
}

func (t *ClaudeTransformer) TransformRequest(req []byte) ([]byte, error) {
	// Apply model redirects before conversion
	targetModel := t.model
	if len(t.modelRedirects) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(req, &data); err == nil {
			if reqModel, ok := data["model"].(string); ok && reqModel != "" {
				if redirect, found := t.modelRedirects[reqModel]; found {
					targetModel = redirect
				}
			}
		}
	}
	return convert.OpenAI2ReqToClaude(req, targetModel)
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
