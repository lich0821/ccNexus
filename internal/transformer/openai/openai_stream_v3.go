package openai

import (
	"github.com/lich0821/ccNexus/internal/transformer"
	"bytes"
	"fmt"
	"time"
)

// transformStreamingResponseV3 uses new architecture to transform streaming response
func (t *OpenAITransformer) transformStreamingResponseV3(openaiStream []byte, ctx *transformer.StreamContext) ([]byte, error) {
	startTime := time.Now()
	metrics := GetGlobalMetrics()

	parser := NewSSEParser(bytes.NewReader(openaiStream))
	serializer := NewSSESerializer()
	processor := NewEventProcessor()

	// Use StreamContext's state instead of creating new one
	if ctx.State == nil {
		ctx.State = NewStreamState()
	}
	state := ctx.State.(*StreamState)

	result := GetBuffer()
	defer PutBuffer(result)

	var eventsProcessed, eventsGenerated int

	for {
		event, err := parser.Next()
		if err != nil {
			if err.Error() != "EOF" {
				return nil, fmt.Errorf("parser error: %w", err)
			}
			break
		}
		if event == nil {
			continue
		}

		// Check for [DONE] marker
		if event.Data != nil {
			if doneType, ok := event.Data["type"].(string); ok && doneType == "done" {
				// Close all open blocks
				if closeEvent := state.CloseCurrentBlock(); closeEvent != nil {
					serialized, _ := serializer.Serialize(closeEvent)
					result.Write(serialized)
				}

				// Send message_delta
				messageDelta := &SSEEvent{
					Event: "message_delta",
					Data: map[string]interface{}{
						"type": "message_delta",
						"delta": map[string]interface{}{
							"stop_reason": "end_turn",
						},
						"usage": map[string]interface{}{
							"output_tokens": state.OutputTokens,
						},
					},
				}
				serialized, _ := serializer.Serialize(messageDelta)
				result.Write(serialized)

				// Send message_stop
				messageStop := &SSEEvent{
					Event: "message_stop",
					Data: map[string]interface{}{
						"type": "message_stop",
					},
				}
				serialized, _ = serializer.Serialize(messageStop)
				result.Write(serialized)

				break
			}
		}

		// Process event
		eventsProcessed++
		outputEvents, err := processor.Process(event, state)
		if err != nil {
			metrics.RecordError()
			return nil, err
		}

		// Serialize output events
		for _, outEvent := range outputEvents {
			serialized, err := serializer.Serialize(outEvent)
			if err != nil {
				metrics.RecordError()
				return nil, err
			}
			result.Write(serialized)
			eventsGenerated++
		}
	}

	duration := time.Since(startTime)
	metrics.RecordEvent(eventsProcessed, eventsGenerated, duration)

	return result.Bytes(), nil
}
