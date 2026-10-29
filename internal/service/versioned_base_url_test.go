package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lich0821/ccNexus/internal/config"
)

func TestMinimalRequestUsesConfiguredAPIVersion(t *testing.T) {
	for _, tc := range []struct {
		transformer, wantPath string
	}{
		{"openai", "/api/coding/paas/v4/chat/completions"},
		{"openai2", "/api/coding/paas/v4/responses"},
		{"claude", "/api/coding/paas/v4/messages"},
	} {
		t.Run(tc.transformer, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.wantPath {
					t.Errorf("path = %q, want %q", r.URL.Path, tc.wantPath)
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write([]byte(`{}`))
			}))
			defer upstream.Close()
			service := NewEndpointService(config.DefaultConfig(), nil, nil)
			status, err := service.testMinimalRequest(upstream.URL+"/api/coding/paas/v4", "test-key", tc.transformer, "test-model", nil)
			if err != nil || status != http.StatusOK {
				t.Fatalf("status = %d, err = %v", status, err)
			}
		})
	}
}
