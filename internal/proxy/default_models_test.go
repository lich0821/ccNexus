package proxy

import (
	"testing"

	"github.com/lich0821/ccNexus/internal/config"
)

func TestDefaultModelsClaudeFallbackIsSupportedModel(t *testing.T) {
	models := (&Proxy{}).getDefaultModels(config.Endpoint{Name: "claude", Transformer: "claude"})
	if len(models) != 1 || models[0].ID != "claude-sonnet-4-6" {
		t.Fatalf("models=%+v, want claude-sonnet-4-6", models)
	}
}
