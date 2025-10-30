package transformer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/lich0821/ccNexus/internal/logger"
)

// ClaudeTransformer handles Claude API with optional model override
type ClaudeTransformer struct{
	model         string // Optional model override
	originalModel string // Original model from request
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
	// Parse to extract original model
	var temp map[string]interface{}
	if err := json.Unmarshal(claudeReq, &temp); err != nil {
		return nil, fmt.Errorf("failed to parse Claude request: %w", err)
	}

	// Save original model for response restoration
	if model, ok := temp["model"].(string); ok {
		t.originalModel = model
	}

	// If no model override, pass through as-is
	if t.model == "" {
		return claudeReq, nil
	}

	// Override model if configured
	result := string(claudeReq)
	logger.Debug("[Claude] Overriding model: %s → %s", t.originalModel, t.model)
	// Use regex to replace model value while preserving order
	re := regexp.MustCompile(`"model":"[^"]*"`)
	result = re.ReplaceAllString(result, `"model":"`+t.model+`"`)

	return []byte(result), nil
}

// TransformResponse normalizes the response for compatibility
func (t *ClaudeTransformer) TransformResponse(targetResp []byte, isStreaming bool) ([]byte, error) {
	result := string(targetResp)

	// For streaming responses, fix content: null -> content: []
	if isStreaming && strings.Contains(result, `"content":null`) {
		result = strings.ReplaceAll(result, `"content":null`, `"content":[]`)
	}

	// Restore original model name if it was overridden
	if t.model != "" && t.originalModel != "" {
		// Replace any occurrence of the overridden model with original
		result = strings.ReplaceAll(result, `"model":"`+t.model+`"`, `"model":"`+t.originalModel+`"`)
	}

	return []byte(result), nil
}

// Name returns the transformer name
func (t *ClaudeTransformer) Name() string {
	return "claude"
}

func init() {
	Register(NewClaudeTransformer())
}
