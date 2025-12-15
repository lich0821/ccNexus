package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/storage"
)

// testEndpoint tests an endpoint's connectivity
func (h *Handler) testEndpoint(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get endpoint
	endpoints, err := h.storage.GetEndpoints()
	if err != nil {
		logger.Error("Failed to get endpoints: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get endpoints")
		return
	}

	var endpoint *storage.Endpoint
	for i := range endpoints {
		if endpoints[i].Name == name {
			endpoint = &endpoints[i]
			break
		}
	}

	if endpoint == nil {
		WriteError(w, http.StatusNotFound, "Endpoint not found")
		return
	}

	// Test the endpoint
	start := time.Now()
	response, err := h.sendTestRequest(endpoint)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"latency": latency,
			"error":   err.Error(),
		})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"latency":  latency,
		"response": response,
	})
}

// sendTestRequest sends a test request to an endpoint
func (h *Handler) sendTestRequest(endpoint *storage.Endpoint) (string, error) {
	var reqBody []byte
	var url string
	var err error

	switch endpoint.Transformer {
	case "claude":
		url = fmt.Sprintf("%s/v1/messages", endpoint.APIUrl)
		reqBody, err = json.Marshal(map[string]interface{}{
			"model": "claude-3-5-sonnet-20241022",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "你是什么模型?",
				},
			},
			"max_tokens": 16,
		})
	case "openai", "openai2":
		url = fmt.Sprintf("%s/v1/chat/completions", endpoint.APIUrl)
		model := endpoint.Model
		if model == "" {
			model = "gpt-4"
		}
		reqBody, err = json.Marshal(map[string]interface{}{
			"model": model,
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "你是什么模型?",
				},
			},
			"max_tokens": 16,
		})
	case "gemini":
		model := endpoint.Model
		if model == "" {
			model = "gemini-pro"
		}
		url = fmt.Sprintf("%s/v1beta/models/%s:generateContent", endpoint.APIUrl, model)
		reqBody, err = json.Marshal(map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{
							"text": "你是什么模型?",
						},
					},
				},
			},
		})
	default:
		return "", fmt.Errorf("unsupported transformer: %s", endpoint.Transformer)
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication based on transformer
	switch endpoint.Transformer {
	case "claude":
		req.Header.Set("x-api-key", endpoint.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case "openai", "openai2":
		req.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
	case "gemini":
		// Gemini uses API key in URL query parameter
		q := req.URL.Query()
		q.Add("key", endpoint.APIKey)
		req.URL.RawQuery = q.Encode()
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to extract the actual message
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), nil
	}

	// Extract message based on transformer
	switch endpoint.Transformer {
	case "claude":
		if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
			if block, ok := content[0].(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					return text, nil
				}
			}
		}
	case "openai", "openai2":
		if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						return content, nil
					}
				}
			}
		}
	case "gemini":
		if candidates, ok := result["candidates"].([]interface{}); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]interface{}); ok {
				if content, ok := candidate["content"].(map[string]interface{}); ok {
					if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
						if part, ok := parts[0].(map[string]interface{}); ok {
							if text, ok := part["text"].(string); ok {
								return text, nil
							}
						}
					}
				}
			}
		}
	}

	return string(body), nil
}

// handleFetchModels fetches available models from a provider
func (h *Handler) handleFetchModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		APIUrl      string `json:"apiUrl"`
		APIKey      string `json:"apiKey"`
		Transformer string `json:"transformer"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	models, err := h.fetchModelsFromProvider(req.APIUrl, req.APIKey, req.Transformer)
	if err != nil {
		logger.Error("Failed to fetch models: %v", err)
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch models: %v", err))
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"models": models,
	})
}

// fetchModelsFromProvider fetches available models from a provider
func (h *Handler) fetchModelsFromProvider(apiUrl, apiKey, transformer string) ([]string, error) {
	var url string
	var authHeader string

	switch transformer {
	case "openai", "openai2":
		url = fmt.Sprintf("%s/v1/models", apiUrl)
		authHeader = "Bearer " + apiKey
	case "claude":
		// Claude doesn't have a models endpoint, return known models
		return []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
		}, nil
	case "gemini":
		// Gemini models are typically known
		return []string{
			"gemini-pro",
			"gemini-pro-vision",
			"gemini-ultra",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported transformer: %s", transformer)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", authHeader)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Data))
	for _, model := range result.Data {
		models = append(models, model.ID)
	}

	return models, nil
}
