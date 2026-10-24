package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/proxy"
	"github.com/lich0821/ccNexus/internal/storage"
)

func newEndpointTestHandler(t *testing.T) *Handler {
	t.Helper()

	db, err := storage.NewSQLiteStorage(filepath.Join(t.TempDir(), "webui.db"))
	if err != nil {
		t.Fatalf("create test storage: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	cfg := config.DefaultConfig()
	p := proxy.New(cfg, storage.NewStatsStorageAdapter(db), db, "test")
	return NewHandler(cfg, p, db)
}

func saveEndpointForTest(t *testing.T, h *Handler, endpoint storage.Endpoint) {
	t.Helper()
	if err := h.storage.SaveEndpoint(&endpoint); err != nil {
		t.Fatalf("save endpoint: %v", err)
	}
}

func TestListEndpointsDoesNotReturnAPIKeyInPlaintext(t *testing.T) {
	h := newEndpointTestHandler(t)
	saveEndpointForTest(t, h, storage.Endpoint{
		Name:        "saved",
		APIUrl:      "https://url-user:url-pass@example.com/v1?api_key=query-secret&region=us",
		APIKey:      "top-secret",
		AuthMode:    config.AuthModeAPIKey,
		Transformer: "openai",
	})

	rec := httptest.NewRecorder()
	h.listEndpoints(rec, httptest.NewRequest(http.MethodGet, "/api/endpoints", nil))

	var response struct {
		Data struct {
			Endpoints []map[string]interface{} `json:"endpoints"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data.Endpoints) != 1 {
		t.Fatalf("expected one endpoint, got %d", len(response.Data.Endpoints))
	}
	endpoint := response.Data.Endpoints[0]
	if _, ok := endpoint["apiKey"]; ok || strings.Contains(rec.Body.String(), "top-secret") {
		t.Fatalf("response leaked plaintext apiKey: %s", rec.Body.String())
	}
	if hasAPIKey, _ := endpoint["hasApiKey"].(bool); !hasAPIKey {
		t.Fatalf("expected hasApiKey=true: %s", rec.Body.String())
	}
	for _, secret := range []string{"url-user", "url-pass", "query-secret"} {
		if strings.Contains(rec.Body.String(), secret) {
			t.Fatalf("response leaked URL secret %q: %s", secret, rec.Body.String())
		}
	}
	safeURL, err := url.Parse(endpoint["apiUrl"].(string))
	if err != nil {
		t.Fatalf("parse redacted apiUrl: %v", err)
	}
	if safeURL.User != nil || safeURL.Query().Get("api_key") != "[REDACTED]" || safeURL.Query().Get("region") != "us" {
		t.Fatalf("unexpected redacted apiUrl: %s", safeURL)
	}
}

func TestUpdateEndpointPreservesHiddenURLSecretsWhenEditingVisibleFields(t *testing.T) {
	h := newEndpointTestHandler(t)
	const storedURL = "https://url-user:url-pass@example.com/v1?api_key=query-secret&region=us"
	saveEndpointForTest(t, h, storage.Endpoint{
		Name: "saved", APIUrl: storedURL, APIKey: "top-secret",
		AuthMode: config.AuthModeAPIKey, Enabled: true, Transformer: "openai",
	})

	rec := httptest.NewRecorder()
	h.updateEndpoint(rec, httptest.NewRequest(http.MethodPut, "/api/endpoints/saved", strings.NewReader(`{"apiUrl":"https://example.com/v2?api_key=%5BREDACTED%5D&region=eu","apiKey":"","authMode":"api_key","enabled":true,"transformer":"openai"}`)), "saved")
	if rec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}

	endpoints, err := h.storage.GetEndpoints()
	if err != nil || len(endpoints) != 1 {
		t.Fatalf("load endpoint: count=%d err=%v", len(endpoints), err)
	}
	updatedURL, err := url.Parse(endpoints[0].APIUrl)
	if err != nil {
		t.Fatalf("parse updated URL: %v", err)
	}
	password, _ := updatedURL.User.Password()
	if updatedURL.User.Username() != "url-user" || password != "url-pass" || updatedURL.Path != "/v2" || updatedURL.Query().Get("api_key") != "query-secret" || updatedURL.Query().Get("region") != "eu" {
		t.Fatalf("unexpected merged URL: %s", updatedURL)
	}
}

func TestCloneEndpointPreservesHiddenURLSecretsWhenEditingVisibleFields(t *testing.T) {
	h := newEndpointTestHandler(t)
	const storedURL = "https://url-user:url-pass@example.com/v1?api_key=query-secret&region=us"
	saveEndpointForTest(t, h, storage.Endpoint{
		Name: "source", APIUrl: storedURL, APIKey: "top-secret",
		AuthMode: config.AuthModeAPIKey, Enabled: true, Transformer: "openai",
	})

	body := `{"name":"clone","apiUrl":"https://example.com/v2?api_key=%5BREDACTED%5D&region=eu","apiKey":"","authMode":"api_key","enabled":true,"transformer":"openai","cloneFrom":"source"}`
	rec := httptest.NewRecorder()
	h.createEndpoint(rec, httptest.NewRequest(http.MethodPost, "/api/endpoints", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("clone status=%d body=%s", rec.Code, rec.Body.String())
	}

	endpoints, err := h.storage.GetEndpoints()
	if err != nil || len(endpoints) != 2 {
		t.Fatalf("load endpoints: count=%d err=%v", len(endpoints), err)
	}
	clonedURL, err := url.Parse(endpoints[1].APIUrl)
	if err != nil {
		t.Fatalf("parse cloned URL: %v", err)
	}
	password, _ := clonedURL.User.Password()
	if clonedURL.User.Username() != "url-user" || password != "url-pass" || clonedURL.Path != "/v2" || clonedURL.Query().Get("api_key") != "query-secret" || clonedURL.Query().Get("region") != "eu" {
		t.Fatalf("unexpected cloned URL: %s", clonedURL)
	}
}

func TestUpdateEndpointDoesNotCarryHiddenURLSecretsToDifferentHost(t *testing.T) {
	h := newEndpointTestHandler(t)
	saveEndpointForTest(t, h, storage.Endpoint{
		Name: "saved", APIUrl: "https://url-user:url-pass@example.com/v1?api_key=query-secret", APIKey: "top-secret",
		AuthMode: config.AuthModeAPIKey, Enabled: true, Transformer: "openai",
	})

	rec := httptest.NewRecorder()
	h.updateEndpoint(rec, httptest.NewRequest(http.MethodPut, "/api/endpoints/saved", strings.NewReader(`{"apiUrl":"https://other.example/v1?region=eu","apiKey":"","authMode":"api_key","enabled":true,"transformer":"openai"}`)), "saved")
	if rec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}
	endpoints, err := h.storage.GetEndpoints()
	if err != nil || len(endpoints) != 1 {
		t.Fatalf("load endpoint: count=%d err=%v", len(endpoints), err)
	}
	updatedURL, err := url.Parse(endpoints[0].APIUrl)
	if err != nil {
		t.Fatalf("parse updated URL: %v", err)
	}
	if updatedURL.Host != "other.example" || updatedURL.User != nil || updatedURL.Query().Get("api_key") != "" {
		t.Fatalf("old URL secrets crossed hosts: %s", updatedURL)
	}
}

func TestEndpointResponseNormalizesLegacyAuthMode(t *testing.T) {
	response := newEndpointResponse(storage.Endpoint{AuthMode: ""})
	if response.AuthMode != config.AuthModeAPIKey {
		t.Fatalf("expected legacy authMode to normalize to %q, got %q", config.AuthModeAPIKey, response.AuthMode)
	}
}

func TestCurrentEndpointAndSwitchUseProxyState(t *testing.T) {
	h := newEndpointTestHandler(t)
	h.config.UpdateEndpoints([]config.Endpoint{
		{Name: "first", APIUrl: "https://first.example", Enabled: true},
		{Name: "second", APIUrl: "https://second.example", Enabled: true},
	})
	if err := h.proxy.SetCurrentEndpoint("second"); err != nil {
		t.Fatalf("set initial current endpoint: %v", err)
	}

	rec := httptest.NewRecorder()
	h.handleCurrentEndpoint(rec, httptest.NewRequest(http.MethodGet, "/api/endpoints/current", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"second"`) {
		t.Fatalf("current endpoint ignored proxy state: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.handleSwitchEndpoint(rec, httptest.NewRequest(http.MethodPost, "/api/endpoints/switch", strings.NewReader(`{"name":"first"}`)))
	if rec.Code != http.StatusOK || h.proxy.GetCurrentEndpointName() != "first" {
		t.Fatalf("switch did not update proxy state: status=%d body=%s current=%q", rec.Code, rec.Body.String(), h.proxy.GetCurrentEndpointName())
	}
}

func TestReloadConfigKeepsSharedConfigPointer(t *testing.T) {
	h := newEndpointTestHandler(t)
	shared := h.config
	saveEndpointForTest(t, h, storage.Endpoint{
		Name:        "saved",
		APIUrl:      "https://example.com",
		APIKey:      "secret",
		AuthMode:    config.AuthModeAPIKey,
		Enabled:     true,
		Transformer: "openai",
	})

	if err := h.reloadConfig(); err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if h.config != shared {
		t.Fatal("reloadConfig replaced the shared config used by dynamic middleware")
	}
	if endpoints := shared.GetEndpoints(); len(endpoints) != 1 || endpoints[0].Name != "saved" {
		t.Fatalf("shared config did not receive stored endpoints: %+v", endpoints)
	}
}

func TestUpdateEndpointAPIKeyContract(t *testing.T) {
	h := newEndpointTestHandler(t)
	saveEndpointForTest(t, h, storage.Endpoint{
		Name:        "saved",
		APIUrl:      "https://example.com",
		APIKey:      "top-secret",
		AuthMode:    config.AuthModeAPIKey,
		Transformer: "openai",
	})

	update := func(body string, wantStatus int) {
		t.Helper()
		rec := httptest.NewRecorder()
		h.updateEndpoint(rec, httptest.NewRequest(http.MethodPut, "/api/endpoints/saved", strings.NewReader(body)), "saved")
		if rec.Code != wantStatus {
			t.Fatalf("update status=%d, want %d body=%s", rec.Code, wantStatus, rec.Body.String())
		}
	}
	storedKey := func() string {
		t.Helper()
		endpoints, err := h.storage.GetEndpoints()
		if err != nil || len(endpoints) != 1 {
			t.Fatalf("load endpoint: count=%d err=%v", len(endpoints), err)
		}
		return endpoints[0].APIKey
	}

	update(`{"apiUrl":"https://example.com","apiKey":"","authMode":"api_key","enabled":true,"transformer":"openai"}`, http.StatusOK)
	if got := storedKey(); got != "top-secret" {
		t.Fatalf("empty apiKey must preserve stored key, got %q", got)
	}

	update(`{"apiUrl":"https://example.com","apiKey":"","clearApiKey":true,"authMode":"api_key","enabled":true,"transformer":"openai"}`, http.StatusOK)
	if got := storedKey(); got != "" {
		t.Fatalf("clearApiKey must clear stored key, got %q", got)
	}
	endpoints, err := h.storage.GetEndpoints()
	if err != nil || len(endpoints) != 1 || endpoints[0].Enabled {
		t.Fatalf("clearing an api_key endpoint must disable it: endpoints=%+v err=%v", endpoints, err)
	}
	if err := h.reloadConfig(); err != nil {
		t.Fatalf("cleared disabled endpoint must remain reloadable: %v", err)
	}
	rec := httptest.NewRecorder()
	h.toggleEndpoint(rec, httptest.NewRequest(http.MethodPatch, "/api/endpoints/saved/toggle", strings.NewReader(`{"enabled":true}`)), "saved")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("enabling keyless api_key endpoint status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestEndpointRoutesRejectReservedNamesAndExtraSegments(t *testing.T) {
	h := newEndpointTestHandler(t)
	h.config.UpdateBasicAuth(false, "", "")

	for _, name := range []string{"current", "switch", "reorder", "fetch-models"} {
		rec := httptest.NewRecorder()
		body := fmt.Sprintf(`{"name":%q,"apiUrl":"https://example.com","apiKey":"secret","authMode":"api_key","enabled":true,"transformer":"openai"}`, name)
		h.createEndpoint(rec, httptest.NewRequest(http.MethodPost, "/api/endpoints", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reserved name %q status=%d body=%s", name, rec.Code, rec.Body.String())
		}
	}

	saveEndpointForTest(t, h, storage.Endpoint{
		Name: "kept", APIUrl: "https://example.com", APIKey: "secret",
		AuthMode: config.AuthModeAPIKey, Enabled: true, Transformer: "openai",
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/endpoints/kept/garbage", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("extra endpoint path status=%d body=%s", rec.Code, rec.Body.String())
	}
	endpoints, err := h.storage.GetEndpoints()
	if err != nil || len(endpoints) != 1 || endpoints[0].Name != "kept" {
		t.Fatalf("extra path changed endpoint: endpoints=%+v err=%v", endpoints, err)
	}
}

func TestConcurrentEndpointCreatesKeepUniqueOrderAndConfigSnapshot(t *testing.T) {
	h := newEndpointTestHandler(t)
	const count = 24
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan string, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			body := fmt.Sprintf(`{"name":"endpoint-%02d","apiUrl":"https://example.com","apiKey":"secret","authMode":"api_key","enabled":true,"transformer":"openai"}`, i)
			rec := httptest.NewRecorder()
			h.createEndpoint(rec, httptest.NewRequest(http.MethodPost, "/api/endpoints", strings.NewReader(body)))
			if rec.Code != http.StatusOK {
				errs <- fmt.Sprintf("create %d status=%d body=%s", i, rec.Code, rec.Body.String())
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	endpoints, err := h.storage.GetEndpoints()
	if err != nil || len(endpoints) != count {
		t.Fatalf("stored endpoints count=%d err=%v", len(endpoints), err)
	}
	for i, endpoint := range endpoints {
		if endpoint.SortOrder != i {
			t.Fatalf("endpoint %q sortOrder=%d, want %d", endpoint.Name, endpoint.SortOrder, i)
		}
	}
	if configured := h.config.GetEndpoints(); len(configured) != count {
		t.Fatalf("proxy config snapshot has %d endpoints, want %d", len(configured), count)
	}
}

func TestProviderURLNormalizesMissingScheme(t *testing.T) {
	if got, want := providerURL("api.anthropic.com", "/v1/messages"), "https://api.anthropic.com/v1/messages"; got != want {
		t.Fatalf("providerURL()=%q, want %q", got, want)
	}
}

func TestUpdateEndpointModelContract(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantModel string
	}{
		{
			name:      "omitted model preserves stored model",
			body:      `{"apiUrl":"https://example.com","authMode":"api_key","enabled":true,"transformer":"openai"}`,
			wantModel: "gpt-old",
		},
		{
			name:      "empty model clears stored model",
			body:      `{"apiUrl":"https://example.com","authMode":"api_key","enabled":true,"transformer":"openai","model":""}`,
			wantModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newEndpointTestHandler(t)
			saveEndpointForTest(t, h, storage.Endpoint{
				Name:        "saved",
				APIUrl:      "https://example.com",
				APIKey:      "top-secret",
				AuthMode:    config.AuthModeAPIKey,
				Transformer: "openai",
				Model:       "gpt-old",
			})

			rec := httptest.NewRecorder()
			h.updateEndpoint(rec, httptest.NewRequest(http.MethodPut, "/api/endpoints/saved", strings.NewReader(tt.body)), "saved")
			if rec.Code != http.StatusOK {
				t.Fatalf("update failed: status=%d body=%s", rec.Code, rec.Body.String())
			}

			endpoints, err := h.storage.GetEndpoints()
			if err != nil || len(endpoints) != 1 {
				t.Fatalf("load endpoint: count=%d err=%v", len(endpoints), err)
			}
			if got := endpoints[0].Model; got != tt.wantModel {
				t.Fatalf("expected model %q, got %q", tt.wantModel, got)
			}
		})
	}
}

func TestUpdateEndpointRenamesStoredEndpoint(t *testing.T) {
	h := newEndpointTestHandler(t)
	saveEndpointForTest(t, h, storage.Endpoint{
		Name: "old-name", APIUrl: "https://example.com", APIKey: "top-secret",
		AuthMode: config.AuthModeAPIKey, Enabled: true, Transformer: "openai", Model: "gpt-test",
	})

	rec := httptest.NewRecorder()
	h.updateEndpoint(rec, httptest.NewRequest(http.MethodPut, "/api/endpoints/old-name", strings.NewReader(
		`{"name":"new-name","apiUrl":"https://example.com","authMode":"api_key","enabled":true,"transformer":"openai","model":"gpt-test"}`,
	)), "old-name")
	if rec.Code != http.StatusOK {
		t.Fatalf("rename failed: status=%d body=%s", rec.Code, rec.Body.String())
	}

	endpoints, err := h.storage.GetEndpoints()
	if err != nil || len(endpoints) != 1 {
		t.Fatalf("load endpoints: count=%d err=%v", len(endpoints), err)
	}
	if endpoints[0].Name != "new-name" {
		t.Fatalf("stored endpoint name = %q, want new-name", endpoints[0].Name)
	}
}

func TestEncodedEndpointNameWithSlashIsAddressable(t *testing.T) {
	h := newEndpointTestHandler(t)
	h.config.UpdateBasicAuth(false, "", "")
	saveEndpointForTest(t, h, storage.Endpoint{
		Name: "team/primary", APIUrl: "https://example.com", APIKey: "top-secret",
		AuthMode: config.AuthModeAPIKey, Transformer: "openai",
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/endpoints/team%2Fprimary", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"team/primary"`) {
		t.Fatalf("encoded endpoint name was not addressable: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDeleteMissingEndpointReturnsNotFound(t *testing.T) {
	h := newEndpointTestHandler(t)
	rec := httptest.NewRecorder()
	h.deleteEndpoint(rec, httptest.NewRequest(http.MethodDelete, "/api/endpoints/missing", nil), "missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete missing endpoint status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestReorderEndpointsRequiresCompleteUniqueNames(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "partial", body: `{"names":["first"]}`},
		{name: "duplicate", body: `{"names":["first","first"]}`},
		{name: "unknown", body: `{"names":["first","missing"]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newEndpointTestHandler(t)
			for _, name := range []string{"first", "second"} {
				saveEndpointForTest(t, h, storage.Endpoint{
					Name: name, APIUrl: "https://example.com", APIKey: "key",
					AuthMode: config.AuthModeAPIKey, Transformer: "openai",
				})
			}

			rec := httptest.NewRecorder()
			h.handleReorderEndpoints(rec, httptest.NewRequest(http.MethodPost, "/api/endpoints/reorder", strings.NewReader(tt.body)))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateEndpointRejectsUnsupportedTransformer(t *testing.T) {
	h := newEndpointTestHandler(t)
	rec := httptest.NewRecorder()
	h.createEndpoint(rec, httptest.NewRequest(http.MethodPost, "/api/endpoints", strings.NewReader(
		`{"name":"deepseek","apiUrl":"https://example.com","apiKey":"secret","authMode":"api_key","enabled":true,"transformer":"deepseek"}`,
	)))

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "unsupported transformer") {
		t.Fatalf("expected explicit unsupported transformer error, status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateEndpointNormalizesMissingScheme(t *testing.T) {
	h := newEndpointTestHandler(t)
	rec := httptest.NewRecorder()
	h.createEndpoint(rec, httptest.NewRequest(http.MethodPost, "/api/endpoints", strings.NewReader(
		`{"name":"normalized","apiUrl":"api.example.com/","apiKey":"secret","authMode":"api_key","enabled":true,"transformer":"openai"}`,
	)))
	if rec.Code != http.StatusOK {
		t.Fatalf("create endpoint status=%d body=%s", rec.Code, rec.Body.String())
	}
	endpoints, err := h.storage.GetEndpoints()
	if err != nil || len(endpoints) != 1 {
		t.Fatalf("load endpoints: count=%d err=%v", len(endpoints), err)
	}
	if endpoints[0].APIUrl != "https://api.example.com" {
		t.Fatalf("apiUrl=%q, want normalized https URL", endpoints[0].APIUrl)
	}
}

func TestFetchModelsUsesStoredEndpointKey(t *testing.T) {
	authorization := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization <- r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-test"}]}`))
	}))
	defer upstream.Close()

	h := newEndpointTestHandler(t)
	saveEndpointForTest(t, h, storage.Endpoint{
		Name:        "saved",
		APIUrl:      upstream.URL,
		APIKey:      "top-secret",
		AuthMode:    config.AuthModeAPIKey,
		Transformer: "openai",
	})

	rec := httptest.NewRecorder()
	h.handleFetchModels(rec, httptest.NewRequest(http.MethodPost, "/api/endpoints/fetch-models", strings.NewReader(`{"endpointName":"saved"}`)))

	if rec.Code != http.StatusOK {
		t.Fatalf("fetch models failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	select {
	case got := <-authorization:
		if got != "Bearer top-secret" {
			t.Fatalf("unexpected authorization header %q", got)
		}
	default:
		t.Fatal("provider did not receive request")
	}
	if strings.Contains(rec.Body.String(), "top-secret") {
		t.Fatalf("response leaked stored key: %s", rec.Body.String())
	}
}

func TestSendTestRequestUsesTransformerProtocol(t *testing.T) {
	t.Run("openai2 responses", func(t *testing.T) {
		type capture struct {
			path string
			body map[string]interface{}
		}
		requests := make(chan capture, 1)
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			requests <- capture{path: r.URL.Path, body: body}
			_, _ = w.Write([]byte(`{"output_text":"ok"}`))
		}))
		defer upstream.Close()

		h := newEndpointTestHandler(t)
		got, err := h.sendTestRequest(&storage.Endpoint{
			Name: "responses", APIUrl: upstream.URL + "/v1", APIKey: "secret",
			AuthMode: config.AuthModeAPIKey, Transformer: "openai2", Model: "gpt-test",
		})
		if err != nil {
			t.Fatalf("send test request: %v", err)
		}
		request := <-requests
		if request.path != "/v1/responses" || request.body["input"] == nil {
			t.Fatalf("unexpected Responses request: path=%s body=%v", request.path, request.body)
		}
		if got != "ok" {
			t.Fatalf("unexpected response text %q", got)
		}
	})

	t.Run("codex token pool headers and response redaction", func(t *testing.T) {
		type capture struct {
			authorization string
			accountID     string
			originator    string
			version       string
			sessionID     string
			userAgent     string
		}
		requests := make(chan capture, 1)
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- capture{
				authorization: r.Header.Get("Authorization"),
				accountID:     r.Header.Get("Chatgpt-Account-Id"),
				originator:    r.Header.Get("Originator"),
				version:       r.Header.Get("Version"),
				sessionID:     r.Header.Get("Session_id"),
				userAgent:     r.Header.Get("User-Agent"),
			}
			_, _ = w.Write([]byte(`{"output_text":"access-secret refresh-secret id-secret url-user url-pass"}`))
		}))
		defer upstream.Close()

		h := newEndpointTestHandler(t)
		saveCredentialForTest(t, h, "codex-pool")
		got, err := h.sendTestRequest(&storage.Endpoint{
			Name: "codex-pool", APIUrl: strings.Replace(upstream.URL, "http://", "http://url-user:url-pass@", 1), AuthMode: config.AuthModeTokenPool,
			Transformer: "openai2", Model: "gpt-test",
		})
		if err != nil {
			t.Fatalf("send token pool test request: %v", err)
		}
		request := <-requests
		if request.authorization != "Bearer access-secret" || request.accountID != "account-1" || request.originator != "codex_cli_rs" || request.version == "" || request.sessionID == "" || request.userAgent == "" {
			t.Fatalf("incomplete Codex request headers: %+v", request)
		}
		for _, secret := range []string{"access-secret", "refresh-secret", "id-secret", "url-user", "url-pass"} {
			if strings.Contains(got, secret) {
				t.Fatalf("successful test response leaked %q: %q", secret, got)
			}
		}
	})

	t.Run("claude configured model", func(t *testing.T) {
		models := make(chan string, 1)
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			model, _ := body["model"].(string)
			models <- model
			_, _ = w.Write([]byte(`{"content":[{"text":"ok"}]}`))
		}))
		defer upstream.Close()

		h := newEndpointTestHandler(t)
		got, err := h.sendTestRequest(&storage.Endpoint{
			Name: "claude", APIUrl: upstream.URL, APIKey: "secret",
			AuthMode: config.AuthModeAPIKey, Transformer: "claude", Model: "claude-test",
		})
		if err != nil {
			t.Fatalf("send test request: %v", err)
		}
		if model := <-models; model != "claude-test" {
			t.Fatalf("expected configured Claude model, got %q", model)
		}
		if got != "ok" {
			t.Fatalf("unexpected response text %q", got)
		}
	})

	t.Run("gemini header authentication", func(t *testing.T) {
		type capture struct {
			path     string
			queryKey string
			header   string
		}
		requests := make(chan capture, 1)
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- capture{
				path:     r.URL.Path,
				queryKey: r.URL.Query().Get("key"),
				header:   r.Header.Get("x-goog-api-key"),
			}
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
		}))
		defer upstream.Close()

		h := newEndpointTestHandler(t)
		got, err := h.sendTestRequest(&storage.Endpoint{
			Name: "gemini", APIUrl: upstream.URL, APIKey: "top-secret",
			AuthMode: config.AuthModeAPIKey, Transformer: "gemini", Model: "gemini-test",
		})
		if err != nil {
			t.Fatalf("send test request: %v", err)
		}
		request := <-requests
		if request.path != "/v1beta/models/gemini-test:generateContent" || request.header != "top-secret" || request.queryKey != "" {
			t.Fatalf("unexpected Gemini request: %+v", request)
		}
		if got != "ok" {
			t.Fatalf("unexpected response text %q", got)
		}
	})
}

func TestSendTestRequestRequiresConfiguredModel(t *testing.T) {
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"content":[{"text":"unexpected"}]}`))
	}))
	defer upstream.Close()

	_, err := newEndpointTestHandler(t).sendTestRequest(&storage.Endpoint{
		Name: "claude", APIUrl: upstream.URL, APIKey: "secret",
		AuthMode: config.AuthModeAPIKey, Transformer: "claude", Model: "  ",
	})
	if err == nil || !strings.Contains(err.Error(), "model is required") {
		t.Fatalf("expected missing model error, got %v", err)
	}
	if requests != 0 {
		t.Fatalf("missing model must not send an upstream request, got %d", requests)
	}
}

func TestEndpointTestUsesTemporaryModelWithoutSaving(t *testing.T) {
	models := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		models <- body["model"].(string)
		_, _ = w.Write([]byte(`{"content":[{"text":"ok"}]}`))
	}))
	defer upstream.Close()

	h := newEndpointTestHandler(t)
	saveEndpointForTest(t, h, storage.Endpoint{
		Name: "temporary-model", APIUrl: upstream.URL, APIKey: "secret",
		AuthMode: config.AuthModeAPIKey, Transformer: "claude",
	})

	rec := httptest.NewRecorder()
	h.testEndpoint(rec, httptest.NewRequest(http.MethodPost, "/api/endpoints/temporary-model/test", strings.NewReader(`{"model":"claude-test-only"}`)), "temporary-model")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"success":true`) {
		t.Fatalf("temporary-model test failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := <-models; got != "claude-test-only" {
		t.Fatalf("test model=%q, want claude-test-only", got)
	}
	endpoints, err := h.storage.GetEndpoints()
	if err != nil || len(endpoints) != 1 {
		t.Fatalf("reload endpoints: count=%d err=%v", len(endpoints), err)
	}
	if endpoints[0].Model != "" {
		t.Fatalf("temporary test model was persisted: %q", endpoints[0].Model)
	}
}

func TestSendTestRequestRejectsUnexpectedSuccessBodies(t *testing.T) {
	const secret = "top-secret"
	tests := []struct {
		name      string
		body      string
		wantError string
	}{
		{name: "invalid JSON", body: secret, wantError: "failed to parse response"},
		{name: "missing content", body: `{"echo":"` + secret + `"}`, wantError: "missing expected response content"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			defer upstream.Close()

			got, err := newEndpointTestHandler(t).sendTestRequest(&storage.Endpoint{
				Name: "openai", APIUrl: upstream.URL, APIKey: "auth-key",
				AuthMode: config.AuthModeAPIKey, Transformer: "openai", Model: "gpt-test",
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected %q error, got response=%q error=%v", tt.wantError, got, err)
			}
			if strings.Contains(got, secret) || strings.Contains(err.Error(), secret) {
				t.Fatalf("response leaked provider body: response=%q error=%v", got, err)
			}
		})
	}
}

func TestFetchModelsFromProviderProtocols(t *testing.T) {
	tests := []struct {
		name        string
		transformer string
		response    string
		baseSuffix  string
		wantPath    string
		wantHeader  string
		wantModels  []string
	}{
		{
			name:        "claude",
			transformer: "claude",
			response:    `{"data":[{"id":"claude-a"},{"id":"claude-b"}]}`,
			baseSuffix:  "/v1",
			wantPath:    "/v1/models",
			wantHeader:  "x-api-key",
			wantModels:  []string{"claude-a", "claude-b"},
		},
		{
			name:        "gemini",
			transformer: "gemini",
			response:    `{"models":[{"name":"models/gemini-a"},{"name":"gemini-b"}]}`,
			baseSuffix:  "/v1beta",
			wantPath:    "/v1beta/models",
			wantHeader:  "x-goog-api-key",
			wantModels:  []string{"gemini-a", "gemini-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type capture struct {
				path       string
				rawQuery   string
				authHeader string
			}
			requests := make(chan capture, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- capture{
					path:       r.URL.Path,
					rawQuery:   r.URL.RawQuery,
					authHeader: r.Header.Get(tt.wantHeader),
				}
				_, _ = w.Write([]byte(tt.response))
			}))
			defer upstream.Close()

			models, err := newEndpointTestHandler(t).fetchModelsFromProvider(upstream.URL+tt.baseSuffix, "top-secret", tt.transformer)
			if err != nil {
				t.Fatalf("fetch models: %v", err)
			}
			var request capture
			select {
			case request = <-requests:
			default:
				t.Fatal("provider did not receive request")
			}
			if request.path != tt.wantPath || request.rawQuery != "" || request.authHeader != "top-secret" {
				t.Fatalf("unexpected provider request: %+v", request)
			}
			if strings.Join(models, ",") != strings.Join(tt.wantModels, ",") {
				t.Fatalf("unexpected models: %v", models)
			}
		})
	}
}

func TestFetchModelsErrorsRedactKnownSecrets(t *testing.T) {
	const apiKey = "top-secret"
	t.Run("provider body", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("denied " + apiKey))
		}))
		defer upstream.Close()

		_, err := newEndpointTestHandler(t).fetchModelsFromProvider(upstream.URL, apiKey, "openai")
		if err == nil || strings.Contains(err.Error(), apiKey) {
			t.Fatalf("expected redacted provider error, got %v", err)
		}
	})

	t.Run("client error", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		apiURL := strings.Replace(upstream.URL, "http://", "http://url-user:url-pass@", 1) + "/" + apiKey + "?access_token=query-secret"
		upstream.Close()

		_, err := newEndpointTestHandler(t).fetchModelsFromProvider(apiURL, apiKey, "openai")
		if err == nil {
			t.Fatal("expected client error")
		}
		for _, secret := range []string{apiKey, "url-user", "url-pass", "query-secret"} {
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("client error leaked %q: %v", secret, err)
			}
		}
	})
}

func TestFetchModelsRequestDoesNotEchoAPIKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/endpoints/fetch-models", bytes.NewBufferString(`{"apiUrl":"bad-url","apiKey":"top-secret","transformer":"openai"}`))
	rec := httptest.NewRecorder()
	newEndpointTestHandler(t).handleFetchModels(rec, req)
	if strings.Contains(rec.Body.String(), "top-secret") {
		t.Fatalf("response leaked request key: %s", rec.Body.String())
	}
}
