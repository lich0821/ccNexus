package proxy

import (
	"encoding/json"
	"testing"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/transformer/convert"
)

func TestEnsureCodexResponsesPayload(t *testing.T) {
	raw := []byte(`{"model":"gpt-4.1","stream":true}`)
	out := ensureCodexResponsesPayload(raw)

	var payload map[string]interface{}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	store, ok := payload["store"].(bool)
	if !ok || store {
		t.Fatalf("expected store=false, got %#v", payload["store"])
	}
	stream, ok := payload["stream"].(bool)
	if !ok || !stream {
		t.Fatalf("expected stream=true, got %#v", payload["stream"])
	}
	if instructions, ok := payload["instructions"].(string); !ok || instructions != "" {
		t.Fatalf("expected instructions empty string, got %#v", payload["instructions"])
	}
}

func TestEnsureCodexResponsesPayloadOverridesStoreAndStream(t *testing.T) {
	raw := []byte(`{"model":"gpt-4.1","store":true}`)
	out := ensureCodexResponsesPayload(raw)

	var payload map[string]interface{}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	store, ok := payload["store"].(bool)
	if !ok || store {
		t.Fatalf("expected store=false, got %#v", payload["store"])
	}
	stream, ok := payload["stream"].(bool)
	if !ok || !stream {
		t.Fatalf("expected stream=true, got %#v", payload["stream"])
	}
}

func TestNormalizeTargetPathForBaseURLOnCodexBackend(t *testing.T) {
	got := normalizeTargetPathForBaseURL("https://chatgpt.com/backend-api/codex", "/v1/responses")
	if got != "/responses" {
		t.Fatalf("expected /responses, got %s", got)
	}
}

func TestOverrideModelInPayload(t *testing.T) {
	raw := []byte(`{"model":"gpt-5.3-codex","stream":true}`)
	out := overrideModelInPayload(raw, "gpt-5.2-codex")

	var payload map[string]interface{}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if payload["model"] != "gpt-5.2-codex" {
		t.Fatalf("expected model override to gpt-5.2-codex, got %#v", payload["model"])
	}
}

func TestOpenAI2EndpointsShouldOverrideClientModel(t *testing.T) {
	endpoint := config.Endpoint{
		Name:        "Responses",
		APIUrl:      "https://api.example.com",
		AuthMode:    config.AuthModeAPIKey,
		Transformer: "openai2",
		Model:       "gpt-5.4",
	}

	transformer, err := prepareTransformerForClient(ClientFormatOpenAIResponses, endpoint, endpoint.Model)
	if err != nil {
		t.Fatalf("prepare transformer failed: %v", err)
	}

	body := []byte(`{"model":"glm-5.1","input":"hello"}`)
	transformed, err := transformer.TransformRequest(body)
	if err != nil {
		t.Fatalf("transform request failed: %v", err)
	}

	overridden := overrideModelInPayload(transformed, endpoint.Model)

	var payload map[string]interface{}
	if err := json.Unmarshal(overridden, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if payload["model"] != "gpt-5.4" {
		t.Fatalf("expected endpoint model override to gpt-5.4, got %#v", payload["model"])
	}
}

func TestPrepareTransformerAllowsEmptyEndpointModel(t *testing.T) {
	endpoint := config.Endpoint{
		Name:        "Chat",
		APIUrl:      "https://api.example.com",
		AuthMode:    config.AuthModeAPIKey,
		Transformer: "openai",
	}

	transformer, err := prepareTransformerForClient(ClientFormatOpenAIChat, endpoint, "glm-5.1")
	if err != nil {
		t.Fatalf("prepare transformer failed: %v", err)
	}

	transformed, err := transformer.TransformRequest([]byte(`{"model":"glm-5.1","messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatalf("transform request failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(transformed, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if payload["model"] != "glm-5.1" {
		t.Fatalf("expected passthrough model glm-5.1, got %#v", payload["model"])
	}
}

func TestResolveAttemptModelNamePrefersEndpointModel(t *testing.T) {
	reqCtx := &proxyRequestContext{
		requestModel:  "glm-5.1",
		modelOverride: "gpt-4.1",
	}
	endpoint := config.Endpoint{Name: "Primary", Model: "gpt-5.4"}

	got := resolveAttemptModelName(reqCtx, endpoint)
	if got != "gpt-5.4" {
		t.Fatalf("expected endpoint model to win, got %q", got)
	}
}

func TestResolveAttemptModelNameFallsBackToRequestOverride(t *testing.T) {
	reqCtx := &proxyRequestContext{
		requestModel:  "glm-5.1",
		modelOverride: "gpt-4.1",
	}
	endpoint := config.Endpoint{Name: "Primary"}

	got := resolveAttemptModelName(reqCtx, endpoint)
	if got != "gpt-4.1" {
		t.Fatalf("expected model override fallback, got %q", got)
	}
}

func TestClaudeToOpenAIUsesRequestModelWhenEndpointModelEmpty(t *testing.T) {
	body := []byte(`{"model":"glm-5.1","messages":[{"role":"user","content":"hello"}]}`)

	out, err := convert.OpenAIReqToClaude(body, "glm-5.1")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if payload["model"] != "glm-5.1" {
		t.Fatalf("expected request model passthrough, got %#v", payload["model"])
	}
}

func TestShouldHandleAsStreamingResponseForCodexWithoutContentType(t *testing.T) {
	endpoint := config.Endpoint{
		Name:        "TokenPool",
		APIUrl:      "https://chatgpt.com/backend-api/codex",
		Transformer: "openai2",
	}
	if !shouldHandleAsStreamingResponse("", true, endpoint, "cx_chat_openai2") {
		t.Fatal("expected stream=true Codex response with empty content-type to be treated as streaming")
	}
	if shouldHandleAsStreamingResponse("", false, endpoint, "cx_chat_openai2") {
		t.Fatal("expected non-stream client request to not be treated as streaming when content-type is empty")
	}
	if !shouldHandleAsStreamingResponse("text/event-stream", false, endpoint, "cx_chat_openai2") {
		t.Fatal("expected text/event-stream content-type to be treated as streaming")
	}
}
