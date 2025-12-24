package chat

import (
	"encoding/json"

	"github.com/lich0821/ccNexus/internal/transformer"
)

// OpenAITransformer is a passthrough transformer for Codex Chat → OpenAI Chat
type OpenAITransformer struct {
	model          string
	modelRedirects map[string]string
}

// NewOpenAITransformer creates a new passthrough transformer
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
	return "cx_chat_openai"
}

func (t *OpenAITransformer) TransformRequest(req []byte) ([]byte, error) {
	// Apply model redirects or global override
	if len(t.modelRedirects) > 0 || t.model != "" {
		var data map[string]interface{}
		if err := json.Unmarshal(req, &data); err == nil {
			if reqModel, ok := data["model"].(string); ok && reqModel != "" {
				if redirect, found := t.modelRedirects[reqModel]; found {
					data["model"] = redirect
					return json.Marshal(data)
				}
			}
			if t.model != "" {
				data["model"] = t.model
				return json.Marshal(data)
			}
		}
	}
	return req, nil
}

func (t *OpenAITransformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	return resp, nil
}

func (t *OpenAITransformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	return resp, nil
}
