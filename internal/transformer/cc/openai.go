package cc

import (
	"encoding/json"

	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// OpenAITransformer transforms Claude Code requests to OpenAI Chat format
type OpenAITransformer struct {
	model          string
	modelRedirects map[string]string
}

// NewOpenAITransformer creates a new transformer
func NewOpenAITransformer(model string) *OpenAITransformer {
	return &OpenAITransformer{model: model}
}

// NewOpenAITransformerWithRedirects creates a transformer with model and model redirects
func NewOpenAITransformerWithRedirects(model string, redirects map[string]string) *OpenAITransformer {
	return &OpenAITransformer{
		model:          model,
		modelRedirects: redirects,
	}
}

func (t *OpenAITransformer) Name() string {
	return "cc_openai"
}

func (t *OpenAITransformer) TransformRequest(req []byte) ([]byte, error) {
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
	return convert.ClaudeReqToOpenAI(req, targetModel)
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
