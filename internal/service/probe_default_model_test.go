package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lich0821/ccNexus/internal/config"
)

func newModelRecorder(t *testing.T) (*httptest.Server, chan string) {
	t.Helper()
	models := make(chan string, 4)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &req)
		models <- req.Model
		_, _ = w.Write([]byte(`{"input_tokens":1}`))
	}))
	t.Cleanup(upstream.Close)
	return upstream, models
}

func TestMinimalRequestClaudeDefaultsToSupportedModel(t *testing.T) {
	upstream, models := newModelRecorder(t)

	service := NewEndpointService(config.DefaultConfig(), nil, nil)
	if _, err := service.testMinimalRequest(upstream.URL, "secret", "claude", "", nil); err != nil {
		t.Fatalf("testMinimalRequest: %v", err)
	}

	if got, want := <-models, "claude-sonnet-5-5"; got != want {
		t.Fatalf("model=%q, want %q", got, want)
	}
}

func TestEndpointClaudeDefaultsToSupportedModel(t *testing.T) {
	upstream, models := newModelRecorder(t)

	cfg := config.DefaultConfig()
	cfg.UpdateEndpoints([]config.Endpoint{{
		Name: "claude-test", APIUrl: upstream.URL, APIKey: "secret",
		AuthMode: config.AuthModeAPIKey, Transformer: "claude",
	}})
	NewEndpointService(cfg, nil, nil).TestEndpoint(0)

	if got, want := <-models, "claude-sonnet-5-5"; got != want {
		t.Fatalf("model=%q, want %q", got, want)
	}
}

func TestTokenCountProbeUsesSupportedModel(t *testing.T) {
	upstream, models := newModelRecorder(t)

	service := NewEndpointService(config.DefaultConfig(), nil, nil)
	if _, err := service.testTokenCountAPI(upstream.URL, "secret"); err != nil {
		t.Fatalf("testTokenCountAPI: %v", err)
	}

	if got, want := <-models, "claude-sonnet-5-5"; got != want {
		t.Fatalf("model=%q, want %q", got, want)
	}
}

func TestMinimalRequestOpenAI2DefaultsToSupportedModel(t *testing.T) {
	upstream, models := newModelRecorder(t)

	service := NewEndpointService(config.DefaultConfig(), nil, nil)
	if _, err := service.testMinimalRequest(upstream.URL, "secret", "openai2", "", nil); err != nil {
		t.Fatalf("testMinimalRequest: %v", err)
	}

	if got, want := <-models, "gpt-5.6-sol"; got != want {
		t.Fatalf("model=%q, want %q", got, want)
	}
}

func TestMinimalRequestOpenAIDefaultsToSupportedModel(t *testing.T) {
	upstream, models := newModelRecorder(t)

	service := NewEndpointService(config.DefaultConfig(), nil, nil)
	if _, err := service.testMinimalRequest(upstream.URL, "secret", "openai", "", nil); err != nil {
		t.Fatalf("testMinimalRequest: %v", err)
	}

	if got, want := <-models, "gpt-4.1-mini"; got != want {
		t.Fatalf("model=%q, want %q", got, want)
	}
}

func TestEndpointOpenAIDefaultsToSupportedModel(t *testing.T) {
	upstream, models := newModelRecorder(t)

	cfg := config.DefaultConfig()
	cfg.UpdateEndpoints([]config.Endpoint{{
		Name: "openai-test", APIUrl: upstream.URL, APIKey: "secret",
		AuthMode: config.AuthModeAPIKey, Transformer: "openai",
	}})
	NewEndpointService(cfg, nil, nil).TestEndpoint(0)

	if got, want := <-models, "gpt-4.1-mini"; got != want {
		t.Fatalf("model=%q, want %q", got, want)
	}
}
