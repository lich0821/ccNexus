package openai


// ToolCallState represents tool call state
type ToolCallState int

const (
	ToolStateIdle ToolCallState = iota
	ToolStateCollecting
	ToolStateComplete
)

// ToolCallContext represents tool call context
type ToolCallContext struct {
	State          ToolCallState
	ID             string
	Name           string
	ArgumentsJSON  string
	AnthropicIndex int
}

// StreamState represents stream processing state machine
type StreamState struct {
	MessageStarted     bool
	CurrentBlock       string // "text", "thinking", "tool_use", ""
	CurrentBlockIndex  int

	ToolCalls          map[int]*ToolCallContext // OpenAI index -> context
	NextAnthropicIndex int

	MessageID    string
	ModelName    string
	InputTokens  int
	OutputTokens int
}

// NewStreamState creates a new stream state
func NewStreamState() *StreamState {
	return &StreamState{
		ToolCalls:          make(map[int]*ToolCallContext),
		NextAnthropicIndex: 0,
	}
}

// StartToolCall starts a new tool call
func (s *StreamState) StartToolCall(openaiIndex int, id, name string) *ToolCallContext {
	ctx := &ToolCallContext{
		State:          ToolStateCollecting,
		ID:             id,
		Name:           name,
		AnthropicIndex: s.NextAnthropicIndex,
	}
	s.ToolCalls[openaiIndex] = ctx
	s.NextAnthropicIndex++
	return ctx
}

// GetToolCall gets tool call context
func (s *StreamState) GetToolCall(openaiIndex int) *ToolCallContext {
	return s.ToolCalls[openaiIndex]
}

// CloseCurrentBlock closes the current content block
func (s *StreamState) CloseCurrentBlock() *SSEEvent {
	if s.CurrentBlock == "" {
		return nil
	}

	event := &SSEEvent{
		Event: "content_block_stop",
		Data: map[string]interface{}{
			"type":  "content_block_stop",
			"index": s.CurrentBlockIndex,
		},
	}

	s.CurrentBlock = ""
	return event
}
