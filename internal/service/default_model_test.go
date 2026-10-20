package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lich0821/ccNexus/internal/config"
)

func newGeminiPathRecorder(t *testing.T) (*httptest.Server, chan string) {
	t.Helper()
	paths := make(chan string, 4)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(upstream.Close)
	return upstream, paths
}

func TestEndpointGeminiDefaultsToSupportedModel(t *testing.T) {
	upstream, paths := newGeminiPathRecorder(t)

	cfg := config.DefaultConfig()
	cfg.UpdateEndpoints([]config.Endpoint{{
		Name: "gemini-test", APIUrl: upstream.URL, APIKey: "secret",
		AuthMode: config.AuthModeAPIKey, Transformer: "gemini",
	}})
	service := NewEndpointService(cfg, nil, nil)
	service.TestEndpoint(0)

	want := "/v1beta/models/gemini-3.8-flash:generateContent"
	if got := <-paths; got != want {
		t.Fatalf("path=%q, want %q", got, want)
	}
}

func TestMinimalRequestGeminiDefaultsToSupportedModel(t *testing.T) {
	upstream, paths := newGeminiPathRecorder(t)

	service := NewEndpointService(config.DefaultConfig(), nil, nil)
	if _, err := service.testMinimalRequest(upstream.URL, "secret", "gemini", "", nil); err != nil {
		t.Fatalf("testMinimalRequest: %v", err)
	}

	want := "/v1beta/models/gemini-3.8-flash:generateContent"
	if got := <-paths; got != want {
		t.Fatalf("path=%q, want %q", got, want)
	}
}
