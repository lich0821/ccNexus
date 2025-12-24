package responses

import (
	"encoding/json"

	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// OpenAITransformer transforms Codex Responses requests to OpenAI Chat format
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
	return "cx_resp_openai"
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
	return convert.OpenAI2ReqToOpenAI(req, targetModel)
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
