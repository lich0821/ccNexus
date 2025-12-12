package transformer

// Transformer defines the interface for API format transformation
type Transformer interface {
	// TransformRequest converts Claude format request to target API format
	TransformRequest(claudeReq []byte) (targetReq []byte, err error)

	// TransformResponse converts target API format response to Claude format
	TransformResponse(targetResp []byte, isStreaming bool) (claudeResp []byte, err error)

	// Name returns the transformer name
	Name() string
}
