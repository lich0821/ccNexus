package chat

import (
	"encoding/json"

	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// GeminiTransformer transforms Codex Chat requests to Gemini format
type GeminiTransformer struct {
	model          string
	modelRedirects map[string]string
}

// NewGeminiTransformer creates a new transformer
func NewGeminiTransformer(model string) *GeminiTransformer {
	return &GeminiTransformer{model: model}
}

// NewGeminiTransformerWithRedirects creates a transformer with model and model redirects
func NewGeminiTransformerWithRedirects(model string, redirects map[string]string) *GeminiTransformer {
	return &GeminiTransformer{
		model:          model,
		modelRedirects: redirects,
	}
}

func (t *GeminiTransformer) Name() string {
	return "cx_chat_gemini"
}

func (t *GeminiTransformer) TransformRequest(req []byte) ([]byte, error) {
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
	return convert.OpenAIReqToGemini(req, targetModel)
}

func (t *GeminiTransformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	if isStreaming {
		return nil, nil
	}
	return convert.GeminiRespToOpenAI(resp, t.model)
}

func (t *GeminiTransformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if isStreaming {
		return convert.GeminiStreamToOpenAI(resp, ctx, t.model)
	}
	return convert.GeminiRespToOpenAI(resp, t.model)
}
