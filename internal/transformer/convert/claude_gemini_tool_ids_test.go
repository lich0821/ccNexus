package convert

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/lich0821/ccNexus/internal/transformer"
)

const geminiParallelSameToolResp = `{"candidates":[{"content":{"role":"model","parts":[` +
	`{"functionCall":{"name":"read_file","args":{"path":"a.txt"}}},` +
	`{"functionCall":{"name":"read_file","args":{"path":"b.txt"}}}]},"finishReason":"STOP"}]}`

func TestGeminiRespToClaudeGivesParallelSameToolCallsDistinctIDs(t *testing.T) {
	out, err := GeminiRespToClaude([]byte(geminiParallelSameToolResp))
	if err != nil {
		t.Fatalf("GeminiRespToClaude failed: %v", err)
	}

	var resp struct {
		Content []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"content"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("invalid output: %v", err)
	}

	var ids []string
	for _, block := range resp.Content {
		if block.Type == "tool_use" {
			ids = append(ids, block.ID)
		}
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 tool_use blocks, got %d", len(ids))
	}
	if ids[0] == "" || ids[0] == ids[1] {
		t.Errorf("tool_use ids must be non-empty and distinct, got %q and %q", ids[0], ids[1])
	}
}

func TestGeminiStreamToClaudeGivesParallelSameToolCallsDistinctIDs(t *testing.T) {
	ctx := transformer.NewStreamContext()
	ctx.ModelName = "gemini-3.8-flash"

	out, err := GeminiStreamToClaude([]byte("data: "+geminiParallelSameToolResp+"\n\n"), ctx)
	if err != nil {
		t.Fatalf("GeminiStreamToClaude failed: %v", err)
	}

	matches := regexp.MustCompile(`"id":"(call_[^"]*)"`).FindAllStringSubmatch(string(out), -1)
	var ids []string
	for _, m := range matches {
		ids = append(ids, m[1])
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 tool_use ids in the stream, got %d: %s", len(ids), out)
	}
	if ids[0] == "" || ids[0] == ids[1] {
		t.Errorf("tool_use ids must be non-empty and distinct, got %q and %q", ids[0], ids[1])
	}
}
