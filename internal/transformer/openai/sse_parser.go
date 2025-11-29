package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// SSEEvent represents an SSE event
type SSEEvent struct {
	Event string                 `json:"event,omitempty"`
	Data  map[string]interface{} `json:"data,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Retry int                    `json:"retry,omitempty"`
}

// SSEParser parses SSE streams
type SSEParser struct {
	scanner      *bufio.Scanner
	currentEvent *SSEEvent
}

// NewSSEParser creates an SSE parser
func NewSSEParser(reader *bytes.Reader) *SSEParser {
	return &SSEParser{
		scanner:      bufio.NewScanner(reader),
		currentEvent: &SSEEvent{},
	}
}

// Next reads the next complete SSE event
func (p *SSEParser) Next() (*SSEEvent, error) {
	for p.scanner.Scan() {
		line := p.scanner.Text()

		// Empty line indicates end of event
		if line == "" {
			if p.currentEvent.Event != "" || p.currentEvent.Data != nil {
				event := p.currentEvent
				p.currentEvent = &SSEEvent{}
				return event, nil
			}
			continue
		}

		// Parse event fields
		if strings.HasPrefix(line, "event:") {
			p.currentEvent.Event = strings.TrimSpace(line[6:])
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(line[5:])
			if data == "[DONE]" {
				p.currentEvent.Data = map[string]interface{}{"type": "done"}
			} else {
				// Validate JSON before parsing
				if !isValidJSON(data) {
					// Check if it's an error response
					if isErr, errMsg := isOpenAIError(data); isErr {
						return nil, fmt.Errorf("OpenAI API error: %s", errMsg)
					}
					// Skip non-JSON data
					continue
				}
				var dataObj map[string]interface{}
				if err := json.Unmarshal([]byte(data), &dataObj); err == nil {
					p.currentEvent.Data = dataObj
				}
			}
		} else if strings.HasPrefix(line, "id:") {
			p.currentEvent.ID = strings.TrimSpace(line[3:])
		}
	}

	// Return any pending event at EOF
	if p.currentEvent.Event != "" || p.currentEvent.Data != nil {
		event := p.currentEvent
		p.currentEvent = &SSEEvent{}
		return event, nil
	}

	if p.scanner.Err() != nil {
		return nil, p.scanner.Err()
	}
	return nil, fmt.Errorf("EOF")
}

// SSESerializer serializes SSE events
type SSESerializer struct {
	buffer *bytes.Buffer
}

// NewSSESerializer creates an SSE serializer
func NewSSESerializer() *SSESerializer {
	return &SSESerializer{
		buffer: &bytes.Buffer{},
	}
}

// Serialize converts an event to SSE format
func (s *SSESerializer) Serialize(event *SSEEvent) ([]byte, error) {
	s.buffer.Reset()

	if event.Event != "" {
		s.buffer.WriteString("event: " + event.Event + "\n")
	}
	if event.ID != "" {
		s.buffer.WriteString("id: " + event.ID + "\n")
	}
	if event.Retry > 0 {
		s.buffer.WriteString("retry: " + string(rune(event.Retry)) + "\n")
	}
	if event.Data != nil {
		if doneType, ok := event.Data["type"].(string); ok && doneType == "done" {
			s.buffer.WriteString("data: [DONE]\n")
		} else {
			dataJSON, _ := json.Marshal(event.Data)
			s.buffer.WriteString("data: " + string(dataJSON) + "\n")
		}
	}

	s.buffer.WriteString("\n")
	return s.buffer.Bytes(), nil
}
