package cc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"

	"github.com/lich0821/ccNexus/internal/transformer"
)

// ClaudeTransformer is a passthrough transformer for Claude Code → Claude endpoint
// with input_tokens fallback for message_delta events
type ClaudeTransformer struct {
	model          string
	modelRedirects map[string]string
}

// NewClaudeTransformer creates a new passthrough transformer
func NewClaudeTransformer() *ClaudeTransformer {
	return &ClaudeTransformer{}
}

// NewClaudeTransformerWithModel creates a transformer with model override
func NewClaudeTransformerWithModel(model string) *ClaudeTransformer {
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
	return "cc_claude"
}

func (t *ClaudeTransformer) TransformRequest(req []byte) ([]byte, error) {
	// Parse request to get model
	var data map[string]interface{}
	if err := json.Unmarshal(req, &data); err != nil {
		return req, nil
	}

	// Step 1: Check ModelRedirects first (highest priority)
	if reqModel, ok := data["model"].(string); ok && reqModel != "" {
		if redirect, found := t.modelRedirects[reqModel]; found {
			data["model"] = redirect
			return json.Marshal(data)
		}
	}

	// Step 2: Apply global model override if set
	if t.model != "" {
		data["model"] = t.model
		return json.Marshal(data)
	}

	// Step 3: No changes needed
	return req, nil
}

func (t *ClaudeTransformer) TransformResponse(resp []byte, isStreaming bool) ([]byte, error) {
	return resp, nil
}

func (t *ClaudeTransformer) TransformResponseWithContext(resp []byte, isStreaming bool, ctx *transformer.StreamContext) ([]byte, error) {
	if ctx == nil {
		return resp, nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(resp))
	var result bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data:") {
			jsonData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(jsonData), &event); err == nil {
				eventType, _ := event["type"].(string)

				if eventType == "message_start" {
					// Cache input_tokens from message_start (only if > 0, keep estimate otherwise)
					if msg, ok := event["message"].(map[string]interface{}); ok {
						if usage, ok := msg["usage"].(map[string]interface{}); ok {
							if input, ok := usage["input_tokens"].(float64); ok && int(input) > 0 {
								ctx.InputTokens = int(input)
							}
						}
					}
				} else if eventType == "message_delta" {
					// Fallback: fill input_tokens if 0
					if usage, ok := event["usage"].(map[string]interface{}); ok {
						if input, ok := usage["input_tokens"].(float64); ok && int(input) == 0 && ctx.InputTokens > 0 {
							usage["input_tokens"] = ctx.InputTokens
							modified, _ := json.Marshal(event)
							result.WriteString("data: ")
							result.Write(modified)
							result.WriteString("\n")
							continue
						}
					}
				}
			}
		}

		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.Bytes(), nil
}
