package transformer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/lich0821/ccNexus/internal/logger"
)

// ToolChainHandler handles tool chain execution
type ToolChainHandler struct {
	apiURL        string
	apiKey        string
	originalReq   []byte
	toolMessages  []map[string]interface{}
	assistantMsgs []map[string]interface{}
	maxDepth      int
	currentDepth  int
}

// NewToolChainHandler creates a new tool chain handler
func NewToolChainHandler(apiURL, apiKey string, originalReq []byte) *ToolChainHandler {
	return &ToolChainHandler{
		apiURL:        apiURL,
		apiKey:        apiKey,
		originalReq:   originalReq,
		toolMessages:  make([]map[string]interface{}, 0),
		assistantMsgs: make([]map[string]interface{}, 0),
		maxDepth:      5, // Prevent infinite recursion
		currentDepth:  0,
	}
}

// AddToolCall adds a tool call to the assistant message
func (h *ToolChainHandler) AddToolCall(toolID, toolName string, input map[string]interface{}) {
	h.assistantMsgs = append(h.assistantMsgs, map[string]interface{}{
		"type":  "tool_use",
		"id":    toolID,
		"name":  toolName,
		"input": input,
	})
}

// AddToolResult adds a tool result to the user message
func (h *ToolChainHandler) AddToolResult(toolID string, result interface{}) {
	h.toolMessages = append(h.toolMessages, map[string]interface{}{
		"type":        "tool_result",
		"tool_use_id": toolID,
		"content":     result,
	})
}

// HasToolCalls checks if there are any tool calls
func (h *ToolChainHandler) HasToolCalls() bool {
	return len(h.assistantMsgs) > 0
}

// HasToolResults checks if there are any tool results
func (h *ToolChainHandler) HasToolResults() bool {
	return len(h.toolMessages) > 0
}

// ExecuteChain executes the tool chain by making a recursive API call
func (h *ToolChainHandler) ExecuteChain() (io.Reader, error) {
	if h.currentDepth >= h.maxDepth {
		return nil, fmt.Errorf("maximum tool chain depth (%d) exceeded", h.maxDepth)
	}

	// Parse original request
	var req map[string]interface{}
	if err := json.Unmarshal(h.originalReq, &req); err != nil {
		return nil, fmt.Errorf("failed to parse original request: %w", err)
	}

	// Get existing messages
	messages, ok := req["messages"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid messages format in request")
	}

	// Add assistant message with tool_use
	if len(h.assistantMsgs) > 0 {
		messages = append(messages, map[string]interface{}{
			"role":    "assistant",
			"content": h.assistantMsgs,
		})
	}

	// Add user message with tool_result
	if len(h.toolMessages) > 0 {
		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": h.toolMessages,
		})
	}

	req["messages"] = messages

	// Construct new request body
	newReqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	logger.Debug("[ToolChain] Making recursive API call (depth: %d)", h.currentDepth+1)

	// Make recursive API call
	httpReq, err := http.NewRequest("POST", h.apiURL, bytes.NewReader(newReqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("x-api-key", h.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("recursive API call failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("recursive API call failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

// Reset resets the tool chain handler for reuse
func (h *ToolChainHandler) Reset() {
	h.toolMessages = make([]map[string]interface{}, 0)
	h.assistantMsgs = make([]map[string]interface{}, 0)
	h.currentDepth = 0
}

// SetMaxDepth sets the maximum recursion depth
func (h *ToolChainHandler) SetMaxDepth(depth int) {
	h.maxDepth = depth
}

// GetCurrentDepth returns the current recursion depth
func (h *ToolChainHandler) GetCurrentDepth() int {
	return h.currentDepth
}

// IncrementDepth increments the recursion depth counter
func (h *ToolChainHandler) IncrementDepth() {
	h.currentDepth++
}
