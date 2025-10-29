package transformer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lich0821/ccNexus/internal/logger"
)

// ClaudeTransformer handles Claude API with optional model override
type ClaudeTransformer struct{
	model string // Optional model override
}

// NewClaudeTransformer creates a new Claude transformer
func NewClaudeTransformer() *ClaudeTransformer {
	return &ClaudeTransformer{}
}

// NewClaudeTransformerWithModel creates a new Claude transformer with model override
func NewClaudeTransformerWithModel(model string) *ClaudeTransformer {
	return &ClaudeTransformer{
		model: strings.TrimSpace(model),
	}
}

// TransformRequest handles Claude API request with optional model override
func (t *ClaudeTransformer) TransformRequest(claudeReq []byte) ([]byte, error) {
	// If no model override, pass through as-is
	if t.model == "" {
		return claudeReq, nil
	}

	// Parse the request to modify the model field
	var req map[string]interface{}
	if err := json.Unmarshal(claudeReq, &req); err != nil {
		return nil, fmt.Errorf("failed to parse Claude request: %w", err)
	}

	// Override the model field
	if existingModel, exists := req["model"]; exists {
		logger.Debug("[Claude] Overriding model: %s → %s", existingModel, t.model)
	}
	req["model"] = t.model

	return json.Marshal(req)
}

// TransformResponse passes through the response without modification
func (t *ClaudeTransformer) TransformResponse(targetResp []byte, isStreaming bool) ([]byte, error) {
	return targetResp, nil
}

// Name returns the transformer name
func (t *ClaudeTransformer) Name() string {
	return "claude"
}

func init() {
	Register(NewClaudeTransformer())
}
