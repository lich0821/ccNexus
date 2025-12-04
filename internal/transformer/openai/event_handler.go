package openai

import (
	"fmt"

	"github.com/lich0821/ccNexus/internal/logger"
)

// EventHandler interface for handling events
type EventHandler interface {
	Handle(event *SSEEvent, state *StreamState) ([]*SSEEvent, error)
}

// MessageStartHandler handles message start events
type MessageStartHandler struct{}

func (h *MessageStartHandler) Handle(event *SSEEvent, state *StreamState) ([]*SSEEvent, error) {
	if event.Data == nil {
		return []*SSEEvent{event}, nil
	}

	eventType, _ := event.Data["object"].(string)
	if eventType != "chat.completion.chunk" {
		return []*SSEEvent{event}, nil
	}

	// Extract usage from first event
	if usage, ok := event.Data["usage"].(map[string]interface{}); ok {
		if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
			state.InputTokens = int(promptTokens)
		}
		if completionTokens, ok := usage["completion_tokens"].(float64); ok {
			state.OutputTokens = int(completionTokens)
		}
	}

	if !state.MessageStarted {
		if id, ok := event.Data["id"].(string); ok {
			state.MessageID = id
		}
		if model, ok := event.Data["model"].(string); ok {
			state.ModelName = model
		}

		state.MessageStarted = true

		// Send message_start event
		return []*SSEEvent{{
			Event: "message_start",
			Data: map[string]interface{}{
				"type": "message_start",
				"message": map[string]interface{}{
					"id":      state.MessageID,
					"type":    "message",
					"role":    "assistant",
					"content": []interface{}{},
					"model":   state.ModelName,
					"usage": map[string]interface{}{
						"input_tokens":  state.InputTokens,
						"output_tokens": state.OutputTokens,
					},
				},
			},
		}}, nil
	}

	return nil, nil
}

// ContentDeltaHandler handles content delta events
type ContentDeltaHandler struct{}

func (h *ContentDeltaHandler) Handle(event *SSEEvent, state *StreamState) ([]*SSEEvent, error) {
	if event.Data == nil {
		return nil, nil
	}

	// Extract usage from every event
	if usage, ok := event.Data["usage"].(map[string]interface{}); ok {
		if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
			state.InputTokens = int(promptTokens)
		}
		if completionTokens, ok := usage["completion_tokens"].(float64); ok {
			state.OutputTokens = int(completionTokens)
		}
	}

	choices, ok := event.Data["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, nil
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		logger.Warn("ContentDeltaHandler: invalid choice type")
		return nil, fmt.Errorf("invalid choice type")
	}

	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	// Check if this is a tool call
	if _, hasToolCalls := delta["tool_calls"]; hasToolCalls {
		return nil, nil
	}

	var events []*SSEEvent

	// Handle text content
	if content, ok := delta["content"].(string); ok && content != "" {
		// Close other blocks
		if state.CurrentBlock != "text" {
			if closeEvent := state.CloseCurrentBlock(); closeEvent != nil {
				events = append(events, closeEvent)
			}
		}

		// Start text block
		if state.CurrentBlock != "text" {
			events = append(events, &SSEEvent{
				Event: "content_block_start",
				Data: map[string]interface{}{
					"type":  "content_block_start",
					"index": state.CurrentBlockIndex,
					"content_block": map[string]interface{}{
						"type": "text",
						"text": "",
					},
				},
			})
			state.CurrentBlock = "text"
		}

		// Send text delta
		events = append(events, &SSEEvent{
			Event: "content_block_delta",
			Data: map[string]interface{}{
				"type":  "content_block_delta",
				"index": state.CurrentBlockIndex,
				"delta": map[string]interface{}{
					"type": "text_delta",
					"text": content,
				},
			},
		})
	}

	return events, nil
}

// ToolCallHandler handles tool call events
type ToolCallHandler struct{}

func (h *ToolCallHandler) Handle(event *SSEEvent, state *StreamState) ([]*SSEEvent, error) {
	if event.Data == nil {
		return nil, nil
	}

	choices, ok := event.Data["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, nil
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		logger.Warn("ToolCallHandler: invalid choice type")
		return nil, fmt.Errorf("invalid choice type")
	}

	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	toolCalls, ok := delta["tool_calls"].([]interface{})
	if !ok || len(toolCalls) == 0 {
		return nil, nil
	}

	var events []*SSEEvent

	for _, tc := range toolCalls {
		toolCall, ok := tc.(map[string]interface{})
		if !ok {
			logger.Warn("ToolCallHandler: invalid tool_call type")
			return nil, fmt.Errorf("invalid tool_call type")
		}

		indexFloat, ok := toolCall["index"].(float64)
		if !ok {
			logger.Warn("ToolCallHandler: invalid or missing index")
			return nil, fmt.Errorf("invalid or missing index")
		}
		openaiIndex := int(indexFloat)

		// Get or create tool call context
		ctx := state.GetToolCall(openaiIndex)
		isNew := ctx == nil

		if isNew {
			// Close previous block
			if closeEvent := state.CloseCurrentBlock(); closeEvent != nil {
				events = append(events, closeEvent)
			}

			id, _ := toolCall["id"].(string)
			function, ok := toolCall["function"].(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := function["name"].(string)
			ctx = state.StartToolCall(openaiIndex, id, name)

			// Send content_block_start
			events = append(events, &SSEEvent{
				Event: "content_block_start",
				Data: map[string]interface{}{
					"type":  "content_block_start",
					"index": ctx.AnthropicIndex,
					"content_block": map[string]interface{}{
						"type":  "tool_use",
						"id":    ctx.ID,
						"name":  ctx.Name,
						"input": map[string]interface{}{},
					},
				},
			})

			state.CurrentBlock = "tool_use"
			state.CurrentBlockIndex = ctx.AnthropicIndex
		}

		// Accumulate arguments
		if function, ok := toolCall["function"].(map[string]interface{}); ok {
			if args, ok := function["arguments"].(string); ok && args != "" {
				ctx.ArgumentsJSON += args

				// Send input_json_delta
				events = append(events, &SSEEvent{
					Event: "content_block_delta",
					Data: map[string]interface{}{
						"type":  "content_block_delta",
						"index": ctx.AnthropicIndex,
						"delta": map[string]interface{}{
							"type":         "input_json_delta",
							"partial_json": args,
						},
					},
				})
			}
		}
	}

	return events, nil
}

// FinishReasonHandler handles finish_reason events
type FinishReasonHandler struct{}

func (h *FinishReasonHandler) Handle(event *SSEEvent, state *StreamState) ([]*SSEEvent, error) {
	if event.Data == nil {
		return nil, nil
	}

	choices, ok := event.Data["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, nil
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	finishReason, ok := choice["finish_reason"].(string)
	if !ok || finishReason == "" {
		return nil, nil
	}

	var events []*SSEEvent

	// Close current block
	if closeEvent := state.CloseCurrentBlock(); closeEvent != nil {
		events = append(events, closeEvent)
	}

	// Map finish_reason
	stopReason := "end_turn"
	switch finishReason {
	case "stop":
		stopReason = "end_turn"
	case "length":
		stopReason = "max_tokens"
	case "tool_calls":
		stopReason = "tool_use"
	}

	// Send message_delta
	events = append(events, &SSEEvent{
		Event: "message_delta",
		Data: map[string]interface{}{
			"type": "message_delta",
			"delta": map[string]interface{}{
				"stop_reason": stopReason,
			},
			"usage": map[string]interface{}{
				"output_tokens": state.OutputTokens,
			},
		},
	})

	// Send message_stop
	events = append(events, &SSEEvent{
		Event: "message_stop",
		Data: map[string]interface{}{
			"type": "message_stop",
		},
	})

	return events, nil
}

// EventProcessor processes events through handler chain
type EventProcessor struct {
	handlers []EventHandler
}

// NewEventProcessor creates an event processor
func NewEventProcessor() *EventProcessor {
	return &EventProcessor{
		handlers: []EventHandler{
			&MessageStartHandler{},
			&ContentDeltaHandler{},
			&ToolCallHandler{},
			&FinishReasonHandler{},
		},
	}
}

// Process processes an event
func (p *EventProcessor) Process(event *SSEEvent, state *StreamState) ([]*SSEEvent, error) {
	for _, handler := range p.handlers {
		events, err := handler.Handle(event, state)
		if err != nil {
			return nil, err
		}
		if len(events) > 0 {
			return events, nil
		}
	}
	return nil, nil
}
