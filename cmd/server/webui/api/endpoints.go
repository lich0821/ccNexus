package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/proxy"
	"github.com/lich0821/ccNexus/internal/storage"
)

type endpointResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	APIUrl      string    `json:"apiUrl"`
	HasAPIKey   bool      `json:"hasApiKey"`
	AuthMode    string    `json:"authMode"`
	Enabled     bool      `json:"enabled"`
	Transformer string    `json:"transformer"`
	Model       string    `json:"model"`
	Remark      string    `json:"remark"`
	SortOrder   int       `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type createEndpointRequest struct {
	Name        string `json:"name"`
	APIUrl      string `json:"apiUrl"`
	APIKey      string `json:"apiKey"`
	AuthMode    string `json:"authMode"`
	Enabled     bool   `json:"enabled"`
	Transformer string `json:"transformer"`
	Model       string `json:"model"`
	Remark      string `json:"remark"`
	CloneFrom   string `json:"cloneFrom"`
}

type updateEndpointRequest struct {
	Name        string  `json:"name"`
	APIUrl      string  `json:"apiUrl"`
	APIKey      string  `json:"apiKey"`
	ClearAPIKey bool    `json:"clearApiKey"`
	AuthMode    string  `json:"authMode"`
	Enabled     bool    `json:"enabled"`
	Transformer string  `json:"transformer"`
	Model       *string `json:"model"`
	Remark      string  `json:"remark"`
}

func newEndpointResponse(endpoint storage.Endpoint) endpointResponse {
	return endpointResponse{
		ID:          endpoint.ID,
		Name:        endpoint.Name,
		APIUrl:      proxy.RedactURLSecrets(endpoint.APIUrl),
		HasAPIKey:   strings.TrimSpace(endpoint.APIKey) != "",
		AuthMode:    config.NormalizeAuthMode(endpoint.AuthMode),
		Enabled:     endpoint.Enabled,
		Transformer: endpoint.Transformer,
		Model:       endpoint.Model,
		Remark:      endpoint.Remark,
		SortOrder:   endpoint.SortOrder,
		CreatedAt:   endpoint.CreatedAt,
		UpdatedAt:   endpoint.UpdatedAt,
	}
}

func normalizeEndpointForSave(endpoint *storage.Endpoint, allowEmptyAPIKey bool) error {
	normalized := config.Endpoint{
		Name:        endpoint.Name,
		APIUrl:      endpoint.APIUrl,
		APIKey:      endpoint.APIKey,
		AuthMode:    endpoint.AuthMode,
		Enabled:     endpoint.Enabled,
		Transformer: endpoint.Transformer,
		Model:       endpoint.Model,
		Remark:      endpoint.Remark,
	}
	if normalized.Transformer == "" {
		normalized.Transformer = "claude"
	}
	config.ApplyEndpointAuthModeRules(&normalized)
	normalized.APIUrl = normalizeAPIUrl(normalized.APIUrl)
	if !isSupportedEndpointTransformer(normalized.Transformer) {
		return fmt.Errorf("unsupported transformer: %s", normalized.Transformer)
	}
	if normalized.AuthMode == config.AuthModeAPIKey && strings.TrimSpace(normalized.APIKey) == "" && !allowEmptyAPIKey {
		return fmt.Errorf("apiKey is required in api_key mode")
	}

	endpoint.APIUrl = normalized.APIUrl
	endpoint.APIKey = normalized.APIKey
	endpoint.AuthMode = normalized.AuthMode
	endpoint.Transformer = normalized.Transformer
	return nil
}

func isSupportedEndpointTransformer(transformer string) bool {
	switch transformer {
	case "claude", "openai", "openai2", "gemini":
		return true
	default:
		return false
	}
}

func mergeHiddenAPIURL(submitted, stored string) string {
	submittedURL, submittedErr := url.Parse(strings.TrimSpace(submitted))
	storedURL, storedErr := url.Parse(strings.TrimSpace(stored))
	if submittedErr != nil || storedErr != nil || submittedURL == nil || storedURL == nil ||
		!strings.EqualFold(submittedURL.Scheme, storedURL.Scheme) ||
		!strings.EqualFold(submittedURL.Host, storedURL.Host) {
		return submitted
	}

	if submittedURL.User == nil {
		submittedURL.User = storedURL.User
	}
	query := submittedURL.Query()
	storedQuery := storedURL.Query()
	for key, values := range query {
		storedValues := storedQuery[key]
		for i, value := range values {
			if value != "[REDACTED]" || len(storedValues) == 0 {
				continue
			}
			if i < len(storedValues) {
				values[i] = storedValues[i]
			} else {
				values[i] = storedValues[0]
			}
		}
		query[key] = values
	}
	submittedURL.RawQuery = query.Encode()
	return submittedURL.String()
}

// handleEndpoints handles GET (list) and POST (create) for endpoints
func (h *Handler) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listEndpoints(w, r)
	case http.MethodPost:
		h.createEndpoint(w, r)
	default:
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleEndpointByName handles GET, PUT, DELETE, PATCH for specific endpoint
func (h *Handler) handleEndpointByName(w http.ResponseWriter, r *http.Request) {
	// Split the escaped path first so encoded slashes remain part of the name.
	path := strings.TrimPrefix(r.URL.EscapedPath(), "/api/endpoints/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		WriteError(w, http.StatusBadRequest, "Endpoint name required")
		return
	}

	name, err := url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(name) == "" {
		WriteError(w, http.StatusBadRequest, "Invalid endpoint name")
		return
	}

	// Handle /test and /toggle sub-paths
	if len(parts) > 1 {
		switch parts[1] {
		case "test":
			if len(parts) != 2 {
				http.NotFound(w, r)
				return
			}
			h.testEndpoint(w, r, name)
			return
		case "toggle":
			if len(parts) != 2 {
				http.NotFound(w, r)
				return
			}
			h.toggleEndpoint(w, r, name)
			return
		case "credentials":
			h.handleEndpointCredentials(w, r, name, parts[2:])
			return
		default:
			http.NotFound(w, r)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.getEndpoint(w, r, name)
	case http.MethodPut:
		h.updateEndpoint(w, r, name)
	case http.MethodDelete:
		h.deleteEndpoint(w, r, name)
	default:
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// listEndpoints returns all endpoints
func (h *Handler) listEndpoints(w http.ResponseWriter, r *http.Request) {
	endpoints, err := h.storage.GetEndpoints()
	if err != nil {
		logger.Error("Failed to get endpoints: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get endpoints")
		return
	}

	responseEndpoints := make([]endpointResponse, len(endpoints))
	for i := range endpoints {
		responseEndpoints[i] = newEndpointResponse(endpoints[i])
	}

	tokenPools, err := h.storage.GetAllTokenPoolStats()
	if err != nil {
		logger.Warn("Failed to get token pool stats: %v", err)
		tokenPools = map[string]storage.TokenPoolStats{}
	}

	WriteSuccess(w, map[string]interface{}{
		"endpoints":  responseEndpoints,
		"tokenPools": tokenPools,
	})
}

// getEndpoint returns a specific endpoint
func (h *Handler) getEndpoint(w http.ResponseWriter, r *http.Request, name string) {
	endpoints, err := h.storage.GetEndpoints()
	if err != nil {
		logger.Error("Failed to get endpoints: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get endpoints")
		return
	}

	for _, ep := range endpoints {
		if ep.Name == name {
			WriteSuccess(w, newEndpointResponse(ep))
			return
		}
	}

	WriteError(w, http.StatusNotFound, "Endpoint not found")
}

// createEndpoint creates a new endpoint
func (h *Handler) createEndpoint(w http.ResponseWriter, r *http.Request) {
	var req createEndpointRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	h.endpointMu.Lock()
	defer h.endpointMu.Unlock()

	// Preserve secrets that are intentionally omitted from the source endpoint response.
	if req.CloneFrom != "" {
		endpoints, err := h.storage.GetEndpoints()
		if err == nil {
			for _, ep := range endpoints {
				if ep.Name == req.CloneFrom {
					if req.APIKey == "" {
						req.APIKey = ep.APIKey
					}
					req.APIUrl = mergeHiddenAPIURL(req.APIUrl, ep.APIUrl)
					break
				}
			}
		}
	}

	endpoint := &storage.Endpoint{
		Name:        strings.TrimSpace(req.Name),
		APIUrl:      req.APIUrl,
		APIKey:      req.APIKey,
		AuthMode:    req.AuthMode,
		Enabled:     req.Enabled,
		Transformer: req.Transformer,
		Model:       req.Model,
		Remark:      req.Remark,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if isReservedEndpointName(endpoint.Name) {
		WriteError(w, http.StatusBadRequest, "Endpoint name is reserved")
		return
	}
	if endpoint.Name == "" || (strings.TrimSpace(endpoint.APIUrl) == "" && config.NormalizeAuthMode(endpoint.AuthMode) != config.AuthModeCodexTokenPool) {
		WriteError(w, http.StatusBadRequest, "Name and apiUrl are required")
		return
	}
	if err := normalizeEndpointForSave(endpoint, false); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if endpoint.APIUrl == "" {
		WriteError(w, http.StatusBadRequest, "Name and apiUrl are required")
		return
	}

	// Get current endpoints to determine sort order
	endpoints, err := h.storage.GetEndpoints()
	if err != nil {
		logger.Error("Failed to get endpoints: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get endpoints")
		return
	}

	// Check if endpoint with same name exists
	for _, ep := range endpoints {
		if ep.Name == endpoint.Name {
			WriteError(w, http.StatusConflict, "Endpoint with this name already exists")
			return
		}
	}

	endpoint.SortOrder = len(endpoints)

	if err := h.storage.SaveEndpoint(endpoint); err != nil {
		logger.Error("Failed to save endpoint: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to save endpoint")
		return
	}

	// Update proxy config
	if err := h.reloadConfig(); err != nil {
		logger.Error("Failed to reload config: %v", err)
	}

	WriteSuccess(w, newEndpointResponse(*endpoint))
}

// updateEndpoint updates an existing endpoint
func (h *Handler) updateEndpoint(w http.ResponseWriter, r *http.Request, name string) {
	var req updateEndpointRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	h.endpointMu.Lock()
	defer h.endpointMu.Unlock()

	// Get existing endpoint
	endpoints, err := h.storage.GetEndpoints()
	if err != nil {
		logger.Error("Failed to get endpoints: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get endpoints")
		return
	}

	var existing *storage.Endpoint
	for i := range endpoints {
		if endpoints[i].Name == name {
			existing = &endpoints[i]
			break
		}
	}

	if existing == nil {
		WriteError(w, http.StatusNotFound, "Endpoint not found")
		return
	}

	// Update fields
	if updatedName := strings.TrimSpace(req.Name); updatedName != "" {
		if isReservedEndpointName(updatedName) {
			WriteError(w, http.StatusBadRequest, "Endpoint name is reserved")
			return
		}
		for _, endpoint := range endpoints {
			if endpoint.Name == updatedName && endpoint.Name != name {
				WriteError(w, http.StatusConflict, "Endpoint with this name already exists")
				return
			}
		}
		existing.Name = updatedName
	}
	if req.APIUrl != "" {
		existing.APIUrl = normalizeAPIUrl(mergeHiddenAPIURL(req.APIUrl, existing.APIUrl))
	}
	if req.ClearAPIKey {
		existing.APIKey = ""
	} else if req.APIKey != "" {
		existing.APIKey = req.APIKey
	}
	if req.AuthMode != "" {
		existing.AuthMode = config.NormalizeAuthMode(req.AuthMode)
	}
	existing.Enabled = req.Enabled
	if req.Transformer != "" {
		existing.Transformer = req.Transformer
	}
	if req.Model != nil {
		existing.Model = *req.Model
	}
	existing.Remark = req.Remark
	if req.ClearAPIKey && existing.AuthMode == config.AuthModeAPIKey {
		existing.Enabled = false
	}
	if err := normalizeEndpointForSave(existing, req.ClearAPIKey); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	existing.UpdatedAt = time.Now()

	if err := h.storage.UpdateEndpointByName(name, existing); err != nil {
		if errors.Is(err, storage.ErrEndpointNotFound) {
			WriteError(w, http.StatusNotFound, "Endpoint not found")
			return
		}
		logger.Error("Failed to update endpoint: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to update endpoint")
		return
	}

	// Update proxy config
	if err := h.reloadConfig(); err != nil {
		logger.Error("Failed to reload config: %v", err)
	}

	WriteSuccess(w, newEndpointResponse(*existing))
}

// deleteEndpoint deletes an endpoint
func (h *Handler) deleteEndpoint(w http.ResponseWriter, r *http.Request, name string) {
	h.endpointMu.Lock()
	defer h.endpointMu.Unlock()

	if err := h.storage.DeleteEndpoint(name); err != nil {
		if errors.Is(err, storage.ErrEndpointNotFound) {
			WriteError(w, http.StatusNotFound, "Endpoint not found")
			return
		}
		logger.Error("Failed to delete endpoint: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to delete endpoint")
		return
	}

	// Update proxy config
	if err := h.reloadConfig(); err != nil {
		logger.Error("Failed to reload config: %v", err)
	}

	WriteSuccess(w, map[string]interface{}{
		"message": "Endpoint deleted successfully",
	})
}

// toggleEndpoint enables or disables an endpoint
func (h *Handler) toggleEndpoint(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	h.endpointMu.Lock()
	defer h.endpointMu.Unlock()

	// Get existing endpoint
	endpoints, err := h.storage.GetEndpoints()
	if err != nil {
		logger.Error("Failed to get endpoints: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get endpoints")
		return
	}

	var existing *storage.Endpoint
	for i := range endpoints {
		if endpoints[i].Name == name {
			existing = &endpoints[i]
			break
		}
	}

	if existing == nil {
		WriteError(w, http.StatusNotFound, "Endpoint not found")
		return
	}
	if req.Enabled && config.NormalizeAuthMode(existing.AuthMode) == config.AuthModeAPIKey && strings.TrimSpace(existing.APIKey) == "" {
		WriteError(w, http.StatusBadRequest, "apiKey is required to enable this endpoint")
		return
	}

	existing.Enabled = req.Enabled
	existing.UpdatedAt = time.Now()

	if err := h.storage.UpdateEndpoint(existing); err != nil {
		logger.Error("Failed to update endpoint: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to update endpoint")
		return
	}

	// Update proxy config
	if err := h.reloadConfig(); err != nil {
		logger.Error("Failed to reload config: %v", err)
	}

	WriteSuccess(w, map[string]interface{}{
		"enabled": existing.Enabled,
	})
}

// handleCurrentEndpoint returns the current active endpoint
func (h *Handler) handleCurrentEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	name := h.proxy.GetCurrentEndpointName()
	if name == "" {
		WriteError(w, http.StatusNotFound, "No enabled endpoints")
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"name": name,
	})
}

// handleSwitchEndpoint switches to a specific endpoint
func (h *Handler) handleSwitchEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	h.endpointMu.Lock()
	defer h.endpointMu.Unlock()

	if err := h.proxy.SetCurrentEndpoint(req.Name); err != nil {
		WriteError(w, http.StatusNotFound, "Endpoint not found or not enabled")
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"message": "Endpoint switched successfully",
		"name":    req.Name,
	})
}

// handleReorderEndpoints reorders endpoints
func (h *Handler) handleReorderEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Names []string `json:"names"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	h.endpointMu.Lock()
	defer h.endpointMu.Unlock()

	// Get all endpoints
	endpoints, err := h.storage.GetEndpoints()
	if err != nil {
		logger.Error("Failed to get endpoints: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get endpoints")
		return
	}

	if len(req.Names) != len(endpoints) {
		WriteError(w, http.StatusBadRequest, "Endpoint order must contain every endpoint exactly once")
		return
	}
	existingNames := make(map[string]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		existingNames[endpoint.Name] = struct{}{}
	}
	seen := make(map[string]struct{}, len(req.Names))
	for _, name := range req.Names {
		if _, ok := existingNames[name]; !ok {
			WriteError(w, http.StatusBadRequest, "Endpoint order contains an unknown endpoint")
			return
		}
		if _, duplicate := seen[name]; duplicate {
			WriteError(w, http.StatusBadRequest, "Endpoint order contains a duplicate endpoint")
			return
		}
		seen[name] = struct{}{}
	}

	if err := h.storage.ReorderEndpoints(req.Names); err != nil {
		logger.Error("Failed to reorder endpoints: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to reorder endpoints")
		return
	}

	// Update proxy config
	if err := h.reloadConfig(); err != nil {
		logger.Error("Failed to reload config: %v", err)
	}

	WriteSuccess(w, map[string]interface{}{
		"message": "Endpoints reordered successfully",
	})
}

// reloadConfig reloads the configuration from storage and updates the proxy
func (h *Handler) reloadConfig() error {
	adapter := storage.NewConfigStorageAdapter(h.storage)
	loaded, err := config.LoadFromStorage(adapter)
	if err != nil {
		return err
	}
	if err := h.proxy.UpdateConfig(loaded); err != nil {
		return err
	}
	h.config.UpdateEndpoints(loaded.GetEndpoints())
	return nil
}

// normalizeAPIUrl ensures the API URL has the correct format
func normalizeAPIUrl(apiUrl string) string {
	apiUrl = strings.TrimSuffix(strings.TrimSpace(apiUrl), "/")
	if apiUrl != "" && !strings.HasPrefix(apiUrl, "http://") && !strings.HasPrefix(apiUrl, "https://") {
		apiUrl = "https://" + apiUrl
	}
	return apiUrl
}

func isReservedEndpointName(name string) bool {
	switch strings.TrimSpace(name) {
	case "current", "switch", "reorder", "fetch-models":
		return true
	default:
		return false
	}
}
