package chat

import (
	"encoding/json"

	"github.com/lich0821/ccNexus/internal/transformer"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

// OpenAI2Transformer transforms Codex Chat requests to OpenAI Responses format
type OpenAI2Transformer struct {
	model          string
	modelRedirects map[string]string
}

// NewOpenAI2Transformer creates a new transformer
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
	return "cx_chat_openai2"
}

func (t *OpenAI2Transformer) TransformRequest(req []byte) ([]byte, error) {
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
	return convert.OpenAIReqToOpenAI2(req, targetModel)
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
