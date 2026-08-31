package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/storage"
	"github.com/lich0821/ccNexus/internal/transformer/cx/responses"
)

type switchTestRoundTripper func(*http.Request) (*http.Response, error)

func (f switchTestRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type switchTestBody struct {
	context  context.Context
	canceled chan struct{}
	release  chan struct{}
}

func (b *switchTestBody) Read([]byte) (int, error) {
	<-b.context.Done()
	close(b.canceled)
	<-b.release
	return 0, context.Cause(b.context)
}

func (b *switchTestBody) Close() error { return nil }

func waitForSwitchTestSignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func TestHandleStreamingResponseRequiresResponsesCompletion(t *testing.T) {
	const completed = `data: {"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":1}}}`
	doneText := "data:" + " [DONE]"
	embeddedDone := `data: {"type":"response.output_text.delta","delta":"` + doneText + ` is documentation"}` + "\n\n"

	cfg := config.DefaultConfig()
	endpoint := config.Endpoint{
		Name:        "OpenAIResponses",
		APIUrl:      "https://example.com",
		APIKey:      "x",
		AuthMode:    config.AuthModeAPIKey,
		Enabled:     true,
		Transformer: "openai2",
		Model:       "gpt-5.6-sol",
	}
	cfg.UpdateEndpoints([]config.Endpoint{endpoint})
	p := &Proxy{config: cfg}
	trans := responses.NewOpenAI2Transformer(endpoint.Model)

	tests := []struct {
		name     string
		upstream string
		wantBody string
		wantErr  bool
	}{
		{
			name:     "completion without trailing blank line",
			upstream: completed,
			wantBody: completed + "\n\n",
		},
		{
			name:     "clean EOF before completion",
			upstream: `data: {"type":"response.in_progress"}` + "\n\n",
			wantBody: `data: {"type":"response.in_progress"}` + "\n\n",
			wantErr:  true,
		},
		{
			name:     "completion with trailing blank line",
			upstream: completed + "\n\n",
			wantBody: completed + "\n\n",
		},
		{
			name:     "completion marker inside JSON delta",
			upstream: embeddedDone + completed + "\n\n",
			wantBody: embeddedDone + completed + "\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(tt.upstream)),
			}
			rec := httptest.NewRecorder()

			_, _, _, err := p.handleStreamingResponse(
				rec,
				resp,
				endpoint,
				trans,
				trans.Name(),
				false,
				endpoint.Model,
				[]byte(`{}`),
				0,
			)

			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "response.completed") {
					t.Fatalf("expected missing response.completed error, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("expected completed stream, got error: %v", err)
			}
			if got := rec.Body.String(); got != tt.wantBody {
				t.Fatalf("unexpected forwarded stream:\nwant %q\n got %q", tt.wantBody, got)
			}
		})
	}
}

func TestHandleStreamingResponseAllowsLargeSSEEvent(t *testing.T) {
	endpoint := config.Endpoint{
		Name:        "OpenAIResponses",
		APIUrl:      "https://example.com",
		APIKey:      "x",
		AuthMode:    config.AuthModeAPIKey,
		Enabled:     true,
		Transformer: "openai2",
		Model:       "gpt-image-2",
	}
	payload := strings.Repeat("a", 3*1024*1024)
	completed := `data: {"type":"response.completed","response":{"output":"` + payload + `"}}` + "\n\n"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(completed)),
	}
	rec := httptest.NewRecorder()

	_, _, _, err := (&Proxy{}).handleStreamingResponse(
		rec,
		resp,
		endpoint,
		responses.NewOpenAI2Transformer(endpoint.Model),
		"cx_resp_openai2",
		false,
		endpoint.Model,
		[]byte(`{}`),
		0,
	)
	if err != nil {
		t.Fatalf("large SSE event failed: %v", err)
	}
	if rec.Body.String() != completed {
		t.Fatalf("large SSE event was not forwarded unchanged")
	}
}

func TestHandleStreamingResponseAllowsNonCurrentSpecifiedEndpoint(t *testing.T) {
	endpointA := config.Endpoint{
		Name: "A", APIUrl: "https://a.example", APIKey: "key-a", AuthMode: config.AuthModeAPIKey,
		Enabled: true, Transformer: "openai2", Model: "gpt-test",
	}
	endpointB := config.Endpoint{
		Name: "B", APIUrl: "https://b.example", APIKey: "key-b", AuthMode: config.AuthModeAPIKey,
		Enabled: true, Transformer: "openai2", Model: "gpt-test",
	}
	cfg := config.DefaultConfig()
	cfg.UpdateEndpoints([]config.Endpoint{endpointA, endpointB})
	p := &Proxy{config: cfg}
	if err := p.SetCurrentEndpoint("B"); err != nil {
		t.Fatalf("set global endpoint: %v", err)
	}
	completed := "data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(completed)),
	}
	rec := httptest.NewRecorder()
	_, _, _, err := p.handleStreamingResponse(
		rec, resp, endpointA, responses.NewOpenAI2Transformer(endpointA.Model),
		"cx_resp_openai2", false, endpointA.Model, []byte(`{}`), 0,
	)
	if err != nil {
		t.Fatalf("non-current explicitly routed stream failed: %v", err)
	}
	if rec.Body.String() != completed {
		t.Fatalf("unexpected forwarded stream: %q", rec.Body.String())
	}
}

func TestRapidEndpointSwitchDoesNotRecordStreamingFailure(t *testing.T) {
	db, err := storage.NewSQLiteStorage(filepath.Join(t.TempDir(), "switch.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	endpointA := config.Endpoint{
		Name: "A", APIUrl: "https://a.example", AuthMode: config.AuthModeCodexTokenPool,
		Enabled: true, Transformer: "openai2", Model: "gpt-test",
	}
	endpointB := config.Endpoint{
		Name: "B", APIUrl: "https://b.example", APIKey: "key-b", AuthMode: config.AuthModeAPIKey,
		Enabled: true, Transformer: "openai2", Model: "gpt-test",
	}
	credential := storage.EndpointCredential{
		EndpointName: "A", ProviderType: "codex", AccountID: "account-a",
		AccessToken: "access-a", Status: "active", Enabled: true,
	}
	if err := db.SaveEndpointCredential(&credential); err != nil {
		t.Fatalf("save credential: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.UpdateEndpoints([]config.Endpoint{endpointA, endpointB})
	p := New(cfg, storage.NewStatsStorageAdapter(db), db, "test")
	started := make(chan struct{})
	canceled := make(chan struct{})
	release := make(chan struct{})
	p.httpClient = &http.Client{Transport: switchTestRoundTripper(func(request *http.Request) (*http.Response, error) {
		close(started)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       &switchTestBody{context: request.Context(), canceled: canceled, release: release},
		}, nil
	})}

	body := []byte(`{"model":"gpt-test","stream":true,"input":"hello"}`)
	reqCtx := &proxyRequestContext{
		httpRequest: httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(string(body))),
		bodyBytes:   body, clientFormat: ClientFormatOpenAIResponses, streamRequested: true,
		requestModel: endpointA.Model, requestStart: time.Now(), endpoints: []config.Endpoint{endpointA, endpointB},
		refreshedCredentialAttempts: make(map[int64]bool),
	}
	done := make(chan struct{})
	go func() {
		p.runEndpointAttempt(httptest.NewRecorder(), reqCtx, &endpointAttempt{endpoint: endpointA})
		close(done)
	}()

	waitForSwitchTestSignal(t, started, "request start")
	if err := p.SetCurrentEndpoint("B"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	waitForSwitchTestSignal(t, canceled, "stream cancellation")
	if err := p.SetCurrentEndpoint("A"); err != nil {
		t.Fatalf("switch back to A: %v", err)
	}
	close(release)
	waitForSwitchTestSignal(t, done, "stream completion")
	assertEndpointSwitchRecordedNoFailure(t, p, db, credential.ID)
}

func TestEndpointSwitchBeforeHeadersDoesNotRecordFailure(t *testing.T) {
	db, err := storage.NewSQLiteStorage(filepath.Join(t.TempDir(), "switch-before-headers.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	endpointA := config.Endpoint{
		Name: "A", APIUrl: "https://a.example", AuthMode: config.AuthModeCodexTokenPool,
		Enabled: true, Transformer: "openai2", Model: "gpt-test",
	}
	endpointB := config.Endpoint{
		Name: "B", APIUrl: "https://b.example", APIKey: "key-b", AuthMode: config.AuthModeAPIKey,
		Enabled: true, Transformer: "openai2", Model: "gpt-test",
	}
	credential := storage.EndpointCredential{
		EndpointName: "A", ProviderType: "codex", AccountID: "account-a",
		AccessToken: "access-a", Status: "active", Enabled: true,
	}
	if err := db.SaveEndpointCredential(&credential); err != nil {
		t.Fatalf("save credential: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.UpdateEndpoints([]config.Endpoint{endpointA, endpointB})
	p := New(cfg, storage.NewStatsStorageAdapter(db), db, "test")
	started := make(chan struct{})
	canceled := make(chan struct{})
	release := make(chan struct{})
	p.httpClient = &http.Client{Transport: switchTestRoundTripper(func(request *http.Request) (*http.Response, error) {
		close(started)
		<-request.Context().Done()
		close(canceled)
		<-release
		return nil, context.Cause(request.Context())
	})}

	body := []byte(`{"model":"gpt-test","stream":true,"input":"hello"}`)
	reqCtx := &proxyRequestContext{
		httpRequest: httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(string(body))),
		bodyBytes:   body, clientFormat: ClientFormatOpenAIResponses, streamRequested: true,
		requestModel: endpointA.Model, requestStart: time.Now(), endpoints: []config.Endpoint{endpointA, endpointB},
		refreshedCredentialAttempts: make(map[int64]bool),
	}
	done := make(chan struct{})
	go func() {
		p.runEndpointAttempt(httptest.NewRecorder(), reqCtx, &endpointAttempt{endpoint: endpointA})
		close(done)
	}()
	waitForSwitchTestSignal(t, started, "request start")
	if err := p.SetCurrentEndpoint("B"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	waitForSwitchTestSignal(t, canceled, "request cancellation")
	if err := p.SetCurrentEndpoint("A"); err != nil {
		t.Fatalf("switch back to A: %v", err)
	}
	close(release)
	waitForSwitchTestSignal(t, done, "request completion")
	assertEndpointSwitchRecordedNoFailure(t, p, db, credential.ID)
}

func TestRapidEndpointSwitchDoesNotRecordAggregatedFailure(t *testing.T) {
	db, err := storage.NewSQLiteStorage(filepath.Join(t.TempDir(), "switch-aggregate.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	endpointA := config.Endpoint{
		Name: "A", APIUrl: config.CodexTokenPoolAPIURL, AuthMode: config.AuthModeCodexTokenPool,
		Enabled: true, Transformer: "openai2", Model: "gpt-test",
	}
	endpointB := config.Endpoint{
		Name: "B", APIUrl: "https://b.example", APIKey: "key-b", AuthMode: config.AuthModeAPIKey,
		Enabled: true, Transformer: "openai2", Model: "gpt-test",
	}
	credential := storage.EndpointCredential{
		EndpointName: "A", ProviderType: "codex", AccountID: "account-a",
		AccessToken: "access-a", Status: "active", Enabled: true,
	}
	if err := db.SaveEndpointCredential(&credential); err != nil {
		t.Fatalf("save credential: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.UpdateEndpoints([]config.Endpoint{endpointA, endpointB})
	p := New(cfg, storage.NewStatsStorageAdapter(db), db, "test")
	started := make(chan struct{})
	canceled := make(chan struct{})
	release := make(chan struct{})
	p.httpClient = &http.Client{Transport: switchTestRoundTripper(func(request *http.Request) (*http.Response, error) {
		close(started)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       &switchTestBody{context: request.Context(), canceled: canceled, release: release},
		}, nil
	})}

	body := []byte(`{"model":"gpt-test","stream":false,"input":"hello"}`)
	reqCtx := &proxyRequestContext{
		httpRequest: httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(string(body))),
		bodyBytes:   body, clientFormat: ClientFormatOpenAIResponses, streamRequested: false,
		requestModel: endpointA.Model, requestStart: time.Now(), endpoints: []config.Endpoint{endpointA, endpointB},
		refreshedCredentialAttempts: make(map[int64]bool),
	}
	done := make(chan struct{})
	go func() {
		p.runEndpointAttempt(httptest.NewRecorder(), reqCtx, &endpointAttempt{endpoint: endpointA})
		close(done)
	}()
	waitForSwitchTestSignal(t, started, "request start")
	if err := p.SetCurrentEndpoint("B"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	waitForSwitchTestSignal(t, canceled, "aggregate cancellation")
	if err := p.SetCurrentEndpoint("A"); err != nil {
		t.Fatalf("switch back to A: %v", err)
	}
	close(release)
	waitForSwitchTestSignal(t, done, "aggregate completion")
	assertEndpointSwitchRecordedNoFailure(t, p, db, credential.ID)
}

func assertEndpointSwitchRecordedNoFailure(t *testing.T, p *Proxy, db *storage.SQLiteStorage, credentialID int64) {
	t.Helper()
	stored, err := db.GetCredentialByID(credentialID)
	if err != nil || stored == nil {
		t.Fatalf("load credential: credential=%v err=%v", stored, err)
	}
	if stored.FailureCount != 0 || stored.LastError != "" {
		t.Fatalf("endpoint switch penalized credential: failureCount=%d lastError=%q", stored.FailureCount, stored.LastError)
	}
	usage, err := db.GetCredentialUsageByEndpoint("A")
	if err != nil {
		t.Fatalf("load credential usage: %v", err)
	}
	if got := usage[credentialID]; got != nil && got.Errors != 0 {
		t.Fatalf("endpoint switch recorded credential usage error: %+v", got)
	}
	_, stats := p.stats.GetStats()
	if got := stats["A"]; got != nil && got.Errors != 0 {
		t.Fatalf("endpoint switch recorded endpoint error: %+v", got)
	}
}
