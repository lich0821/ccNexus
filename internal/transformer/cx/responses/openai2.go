package responses

import (
	"encoding/json"

	"github.com/lich0821/ccNexus/internal/transformer"
)

// OpenAI2Transformer is a passthrough transformer for Codex Responses → OpenAI Responses
type OpenAI2Transformer struct {
	model          string
	modelRedirects map[string]string
}

// NewOpenAI2Transformer creates a new passthrough transformer
func NewOpenAI2Transformer(model string) *OpenAI2Transformer {
	return &OpenAI2Transformer{model: model}
}

// NewOpenAI2TransformerWithRedirects creates a transformer with model and model redirects
func NewOpenAI2TransformerWithRedirects(model string, redirects map[string]string) *OpenAI2Transformer {
	return &OpenAI2Transformer{
		model:          model,
		modelRedirects: redirects,
	}
}

func (t *OpenAI2Transformer) Name() string {
	return "cx_resp_openai2"
}

func (t *OpenAI2Transformer) TransformRequest(req []byte) ([]byte, error) {
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

func (t *OpenAI2Transformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	return resp, nil
}

func (t *OpenAI2Transformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	return resp, nil
}
