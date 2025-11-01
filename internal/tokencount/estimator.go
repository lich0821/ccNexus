package tokencount

import (
	"encoding/json"
	"strings"
)

// CountTokensRequest matches Anthropic's official API specification
type CountTokensRequest struct {
	Model    string         `json:"model" binding:"required"`
	Messages []MessageParam `json:"messages" binding:"required"`
	System   any            `json:"system,omitempty"`
	Tools    []Tool         `json:"tools,omitempty"`
}

type MessageParam struct {
	Role    string `json:"role" binding:"required"`
	Content any    `json:"content" binding:"required"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

type CountTokensResponse struct {
	InputTokens int `json:"input_tokens"`
}

// EstimateInputTokens estimates input tokens for a request
func EstimateInputTokens(req *CountTokensRequest) int {
	tokens := 10 // Base request overhead

	// System prompt
	if req.System != nil {
		tokens += estimateAny(req.System) + 5
	}

	// Messages
	for _, msg := range req.Messages {
		tokens += 10 + estimateAny(msg.Content)
	}

	// Tools
	if len(req.Tools) > 0 {
		tokens += estimateTools(req.Tools)
	}

	return tokens
}

// EstimateOutputTokens estimates tokens for output text
func EstimateOutputTokens(text string) int {
	return estimateText(text)
}

func estimateAny(v any) int {
	switch val := v.(type) {
	case string:
		return estimateText(val)
	case []any:
		tokens := 0
		for _, item := range val {
			tokens += estimateBlock(item)
		}
		return tokens
	default:
		if data, err := json.Marshal(v); err == nil {
			return len(data) / 4
		}
		return 0
	}
}

func estimateBlock(block any) int {
	m, ok := block.(map[string]any)
	if !ok {
		return 10
	}

	blockType, _ := m["type"].(string)
	switch blockType {
	case "text":
		if text, ok := m["text"].(string); ok {
			return estimateText(text)
		}
	case "image":
		return 1500
	case "document":
		return 500
	case "tool_use":
		if input, ok := m["input"]; ok {
			if data, err := json.Marshal(input); err == nil {
				return len(data) / 4
			}
		}
	case "tool_result":
		if content, ok := m["content"].(string); ok {
			return estimateText(content)
		}
	}

	if data, err := json.Marshal(block); err == nil {
		return len(data) / 4
	}
	return 10
}

func estimateText(text string) int {
	if text == "" {
		return 0
	}

	runes := []rune(text)
	count := len(runes)
	if count == 0 {
		return 0
	}

	// Sample first 500 chars for Chinese detection
	sample := count
	if sample > 500 {
		sample = 500
	}

	chinese := 0
	for i := 0; i < sample; i++ {
		if r := runes[i]; r >= 0x4E00 && r <= 0x9FFF {
			chinese++
		}
	}

	ratio := float64(chinese) / float64(sample)
	charsPerToken := 4.0 - 2.5*ratio // English: 4, Chinese: 1.5

	tokens := int(float64(count) / charsPerToken)
	if tokens < 1 {
		return 1
	}
	return tokens
}

func estimateTools(tools []Tool) int {
	n := len(tools)
	base, perTool := getToolOverhead(n)
	tokens := base

	for _, tool := range tools {
		tokens += estimateToolName(tool.Name)
		tokens += estimateText(tool.Description)
		tokens += estimateSchema(tool.InputSchema, n)
		tokens += perTool
	}

	return tokens
}

func getToolOverhead(count int) (base, perTool int) {
	if count == 1 {
		return 0, 400
	}
	if count <= 5 {
		return 150, 150
	}
	return 250, 80
}

func estimateToolName(name string) int {
	if name == "" {
		return 0
	}

	tokens := len(name)/2 + strings.Count(name, "_")

	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			tokens++
		}
	}
	tokens /= 2

	if tokens < 2 {
		return 2
	}
	return tokens
}

func estimateSchema(schema any, toolCount int) int {
	if schema == nil {
		return 0
	}

	data, err := json.Marshal(schema)
	if err != nil {
		return 0
	}

	var density float64
	if toolCount == 1 {
		density = 1.6
	} else if toolCount <= 5 {
		density = 1.9
	} else {
		density = 2.2
	}

	tokens := int(float64(len(data)) / density)

	if strings.Contains(string(data), "$schema") {
		if toolCount == 1 {
			tokens += 15
		} else {
			tokens += 8
		}
	}

	min := 80
	if toolCount > 5 {
		min = 40
	}
	if tokens < min {
		tokens = min
	}

	return tokens
}
