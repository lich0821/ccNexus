package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/storage"
)

const testIdleTimeout = 200 * time.Millisecond

// streamSSE writes count events spaced by interval, so the whole stream outlives
// any timeout shorter than count*interval while never going idle for long.
func streamSSE(w http.ResponseWriter, count int, interval time.Duration) {
	w.Header().Set("Content-Type", "text/event-stream")
	flusher := w.(http.Flusher)
	for i := 0; i < count; i++ {
		fmt.Fprintf(w, "event: ping\ndata: {\"type\":\"ping\",\"n\":%d}\n\n", i)
		flusher.Flush()
		time.Sleep(interval)
	}
	fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	flusher.Flush()
}

func TestSendRequestKeepsActiveStreamPastIdleTimeout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamSSE(w, 8, testIdleTimeout/4)
	}))
	defer upstream.Close()

	req, err := http.NewRequest(http.MethodPost, upstream.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := sendRequestWithIdleTimeout(context.Background(), req, &http.Client{}, nil, testIdleTimeout)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("active stream was cut off: %v", err)
	}
	if !strings.Contains(string(body), "message_stop") {
		t.Fatalf("stream incomplete: %q", body)
	}
}

func TestSendRequestAbortsStalledStream(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer upstream.Close()
	defer close(release)

	req, err := http.NewRequest(http.MethodPost, upstream.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := sendRequestWithIdleTimeout(context.Background(), req, &http.Client{}, nil, testIdleTimeout)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	start := time.Now()
	_, err = io.ReadAll(resp.Body)
	if !errors.Is(err, errUpstreamIdleTimeout) {
		t.Fatalf("expected idle timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("stalled stream took %s to abort", elapsed)
	}
}

func TestSendRequestAbortsWhenNoResponseHeaders(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer upstream.Close()
	defer close(release)

	req, err := http.NewRequest(http.MethodPost, upstream.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	_, err = sendRequestWithIdleTimeout(context.Background(), req, &http.Client{}, nil, testIdleTimeout)
	if !errors.Is(err, errUpstreamIdleTimeout) {
		t.Fatalf("expected idle timeout error, got %v", err)
	}
}

func TestProxiedStreamOutlivesServerWriteTimeout(t *testing.T) {
	const writeTimeout = 200 * time.Millisecond
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamSSE(w, 8, writeTimeout/4)
	}))
	defer upstream.Close()

	db, err := storage.NewSQLiteStorage(filepath.Join(t.TempDir(), "write-timeout.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	cfg := config.DefaultConfig()
	cfg.UpdateEndpoints([]config.Endpoint{{
		Name: "Claude", APIUrl: upstream.URL, APIKey: "x", AuthMode: config.AuthModeAPIKey,
		Enabled: true, Transformer: "claude", Model: "claude-test",
	}})
	p := New(cfg, storage.NewStatsStorageAdapter(db), db, "test")

	server := httptest.NewUnstartedServer(http.HandlerFunc(p.handleProxy))
	server.Config.WriteTimeout = writeTimeout
	server.Start()
	defer server.Close()

	resp, err := http.Post(server.URL+"/v1/messages", "application/json",
		strings.NewReader(`{"model":"claude-test","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("proxied stream was cut off: %v", err)
	}
	if !strings.Contains(string(body), "message_stop") {
		t.Fatalf("proxied stream incomplete: %q", body)
	}
}
