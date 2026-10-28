package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lich0821/ccNexus/internal/config"
)

func TestBuildProxyRequestVersionedBaseURL(t *testing.T) {
	for _, tc := range []struct {
		basePath, wantPath string
	}{
		{"", "/v1/chat/completions"},
		{"/proxy", "/proxy/v1/chat/completions"},
		{"/v1", "/v1/chat/completions"},
		{"/api/coding/paas/v4", "/api/coding/paas/v4/chat/completions"},
		{"/custom/v123", "/custom/v123/chat/completions"},
	} {
		t.Run(tc.basePath, func(t *testing.T) {
			endpoint := config.Endpoint{APIUrl: "https://example.com" + tc.basePath, Transformer: "openai"}
			incoming := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?trace=1", nil)
			outgoing, err := buildProxyRequest(incoming, endpoint, "test-key", []byte(`{}`), "cx_chat_openai", "test-model", nil)
			if err != nil {
				t.Fatal(err)
			}
			if outgoing.URL.Path != tc.wantPath {
				t.Fatalf("path = %q, want %q", outgoing.URL.Path, tc.wantPath)
			}
			if outgoing.URL.RawQuery != "trace=1" {
				t.Fatalf("query = %q, want trace=1", outgoing.URL.RawQuery)
			}
		})
	}
}

func TestFetchModelsFromVersionedBaseURL(t *testing.T) {
	for _, basePath := range []string{"/v1", "/api/coding/paas/v4"} {
		t.Run(basePath, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != basePath+"/models" {
					t.Errorf("path = %q, want %q", r.URL.Path, basePath+"/models")
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write([]byte(`{"data":[{"id":"test-model","object":"model"}]}`))
			}))
			defer upstream.Close()
			p := &Proxy{httpClient: upstream.Client()}
			models, err := p.fetchModelsFromEndpoint(config.Endpoint{
				APIUrl: upstream.URL + basePath, Transformer: "openai", AuthMode: config.AuthModeAPIKey,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(models) != 1 || models[0].ID != "test-model" {
				t.Fatalf("models = %#v", models)
			}
		})
	}
}
