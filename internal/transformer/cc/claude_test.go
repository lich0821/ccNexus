package cc

import (
	"strings"
	"testing"

	"github.com/lich0821/ccNexus/internal/transformer"
)

func TestClaudeTransformerForwardsLargeSignatureDelta(t *testing.T) {
	// Long extended-thinking blocks produce signature_delta lines well over 64KB.
	signature := strings.Repeat("s", 100*1024)
	event := "event: content_block_delta\n" +
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"` + signature + `"}}` + "\n\n"

	out, err := NewClaudeTransformer().TransformResponseWithContext([]byte(event), true, transformer.NewStreamContext())
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if string(out) != event {
		t.Fatalf("large signature_delta event was not forwarded unchanged: got %d bytes, want %d", len(out), len(event))
	}
}
