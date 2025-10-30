package transformer

// OpenAI API structures

// OpenAIToolCall represents a tool call in OpenAI format
type OpenAIToolCall struct {
	Index    *int   `json:"index,omitempty"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// OpenAITool represents a tool definition in OpenAI format
type OpenAITool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description,omitempty"`
		Parameters  map[string]interface{} `json:"parameters"`
	} `json:"function"`
}

// OpenAIMessage represents a message in OpenAI format
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

// OpenAIRequest represents an OpenAI API request
type OpenAIRequest struct {
	Model               string          `json:"model"`
	Messages            []OpenAIMessage `json:"messages"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	Temperature         float64         `json:"temperature,omitempty"`
	Stream              bool            `json:"stream,omitempty"`
	StreamOptions       *StreamOptions  `json:"stream_options,omitempty"`
	EnableThinking      bool            `json:"enable_thinking,omitempty"` // For models that support reasoning/thinking
	Tools               []OpenAITool    `json:"tools,omitempty"`
	ToolChoice          interface{}     `json:"tool_choice,omitempty"`
}

// StreamOptions represents OpenAI stream options
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// OpenAIResponse represents an OpenAI API response
type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string           `json:"role"`
			Content   string           `json:"content"`
			ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAIStreamChunk represents a streaming response chunk
type OpenAIStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role             string           `json:"role,omitempty"`
			Content          string           `json:"content,omitempty"`
			ReasoningContent string           `json:"reasoning_content,omitempty"` // For models with reasoning/thinking
			ToolCalls        []OpenAIToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// Claude API structures

// ClaudeMessage represents a message in Claude format
type ClaudeMessage struct {
	Role         string      `json:"role"`
	Content      interface{} `json:"content"` // Can be string or array of content blocks
	CacheControl interface{} `json:"cache_control,omitempty"` // Prompt caching (ignored in transformation)
}

// ClaudeRequest represents a Claude API request
type ClaudeRequest struct {
	Model       string          `json:"model"`
	Messages    []ClaudeMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	System      interface{}     `json:"system,omitempty"`  // Can be string or array of system messages
	Thinking    interface{}     `json:"thinking,omitempty"` // Claude's thinking/extended thinking parameter
	Tools       []ClaudeTool    `json:"tools,omitempty"`
	ToolChoice  interface{}     `json:"tool_choice,omitempty"`
}

// ClaudeTool represents a tool definition in Claude format
type ClaudeTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ClaudeResponse represents a Claude API response
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ClaudeStreamEvent represents a Claude streaming event
type ClaudeStreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta,omitempty"`
	ContentBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content_block,omitempty"`
	Message struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Model      string `json:"model"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

// StreamContext holds the state for a single streaming response
// This allows multiple concurrent streams to be processed independently
type StreamContext struct {
	MessageStartSent     bool
	ContentBlockStarted  bool
	ThinkingBlockStarted bool // Track if thinking block has been started
	ToolBlockStarted     bool // Track if tool_use block has been started
	ToolBlockPending     bool // Track if tool_use block is pending (waiting for first arguments)
	MessageID            string
	ModelName            string
	InputTokens          int
	OutputTokens         int
	ContentIndex         int
	ThinkingIndex        int   // Index for thinking content block
	ToolIndex            int   // Current tool_use content block index (from OpenAI)
	LastToolIndex        int   // Last assigned Anthropic tool block index (incremental counter)
	FinishReasonSent     bool
	EnableThinking       bool            // Whether thinking is enabled for this request
	CurrentToolCall      *OpenAIToolCall // Current tool call being processed
	ToolCallBuffer       string          // Buffer for accumulating tool call arguments
}

// NewStreamContext creates a new stream context with default values
func NewStreamContext() *StreamContext {
	return &StreamContext{
		MessageStartSent:     false,
		ContentBlockStarted:  false,
		ThinkingBlockStarted: false,
		ToolBlockStarted:     false,
		ToolBlockPending:     false,
		MessageID:            "",
		ModelName:            "",
		InputTokens:          0,
		OutputTokens:         0,
		ContentIndex:         0,
		ThinkingIndex:        0,
		ToolIndex:            0,
		LastToolIndex:        0,
		FinishReasonSent:     false,
		EnableThinking:       false,
		CurrentToolCall:      nil,
		ToolCallBuffer:       "",
	}
}

// Gemini API structures

// GeminiPart represents a part in Gemini format
type GeminiPart struct {
	Text         string                 `json:"text,omitempty"`
	FunctionCall *GeminiFunctionCall    `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

// GeminiFunctionCall represents a function call in Gemini format
type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// GeminiFunctionResponse represents a function response in Gemini format
type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

// GeminiContent represents content in Gemini format
type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiTool represents a tool definition in Gemini format
type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

// GeminiFunctionDeclaration represents a function declaration in Gemini format
type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// GeminiRequest represents a Gemini API request
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	SystemInstruction *GeminiContent        `json:"systemInstruction,omitempty"`
	Tools            []GeminiTool           `json:"tools,omitempty"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

// GeminiGenerationConfig represents generation configuration in Gemini format
type GeminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// GeminiResponse represents a Gemini API response
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []GeminiPart `json:"parts"`
			Role  string       `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
		Index        int    `json:"index"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// GeminiStreamChunk represents a streaming response chunk from Gemini
type GeminiStreamChunk struct {
	Candidates []struct {
		Content struct {
			Parts []GeminiPart `json:"parts"`
			Role  string       `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason,omitempty"`
		Index        int    `json:"index"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata,omitempty"`
}
