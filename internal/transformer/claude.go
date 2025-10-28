package transformer

// ClaudeTransformer is a pass-through transformer for Claude API
type ClaudeTransformer struct{}

// NewClaudeTransformer creates a new Claude transformer
func NewClaudeTransformer() *ClaudeTransformer {
	return &ClaudeTransformer{}
}

// TransformRequest passes through the request without modification
func (t *ClaudeTransformer) TransformRequest(claudeReq []byte) ([]byte, error) {
	return claudeReq, nil
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
