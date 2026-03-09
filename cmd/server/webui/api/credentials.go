package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/storage"
)

type importCredentialItem struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
	LastRefresh  string `json:"last_refresh"`
	Email        string `json:"email"`
	Type         string `json:"type"`
	Expired      string `json:"expired"`
	Remark       string `json:"remark"`
	Enabled      *bool  `json:"enabled"`
}

type importCredentialsRequest struct {
	Items     []importCredentialItem `json:"items"`
	Overwrite bool                   `json:"overwrite"`
	Remark    string                 `json:"remark"`
}

func (h *Handler) handleEndpointCredentials(w http.ResponseWriter, r *http.Request, endpointName string, parts []string) {
	endpoint, err := h.getEndpointByName(endpointName)
	if err != nil {
		logger.Error("Failed to get endpoint: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get endpoint")
		return
	}
	if endpoint == nil {
		WriteError(w, http.StatusNotFound, "Endpoint not found")
		return
	}

	if len(parts) == 0 || parts[0] == "" {
		switch r.Method {
		case http.MethodGet:
			h.listEndpointCredentials(w, r, endpointName)
		case http.MethodPost:
			h.importEndpointCredentials(w, r, endpointName)
		default:
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}

	switch parts[0] {
	case "import":
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		h.importEndpointCredentials(w, r, endpointName)
		return
	case "stats":
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		h.getEndpointCredentialStats(w, r, endpointName)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		WriteError(w, http.StatusBadRequest, "Invalid credential id")
		return
	}

	switch r.Method {
	case http.MethodPatch:
		h.updateEndpointCredential(w, r, endpointName, id)
	case http.MethodDelete:
		h.deleteEndpointCredential(w, r, endpointName, id)
	default:
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *Handler) listEndpointCredentials(w http.ResponseWriter, r *http.Request, endpointName string) {
	credentials, err := h.storage.GetEndpointCredentials(endpointName)
	if err != nil {
		logger.Error("Failed to get endpoint credentials: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get endpoint credentials")
		return
	}

	stats, err := h.storage.GetTokenPoolStats(endpointName)
	if err != nil {
		logger.Warn("Failed to get credential stats: %v", err)
	}

	for i := range credentials {
		credentials[i].AccessToken = maskToken(credentials[i].AccessToken)
		credentials[i].RefreshToken = maskToken(credentials[i].RefreshToken)
		credentials[i].IDToken = maskToken(credentials[i].IDToken)
	}

	WriteSuccess(w, map[string]interface{}{
		"credentials": credentials,
		"stats":       stats,
	})
}

func (h *Handler) importEndpointCredentials(w http.ResponseWriter, r *http.Request, endpointName string) {
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req, items, err := parseImportCredentialsPayload(rawBody)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	existing, err := h.storage.GetEndpointCredentials(endpointName)
	if err != nil {
		logger.Error("Failed to get existing credentials: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get existing credentials")
		return
	}

	accountIndex := make(map[string]storage.EndpointCredential)
	emailIndex := make(map[string]storage.EndpointCredential)
	for _, cred := range existing {
		if cred.AccountID != "" {
			accountIndex[cred.AccountID] = cred
		}
		if cred.Email != "" {
			emailIndex[cred.Email] = cred
		}
	}

	created := 0
	updated := 0
	skipped := 0
	failed := 0
	errors := make([]string, 0)

	for i, item := range items {
		if strings.TrimSpace(item.AccessToken) == "" {
			failed++
			errors = append(errors, fmt.Sprintf("item[%d]: access_token is required", i))
			continue
		}

		expiresAt, err := parseOptionalRFC3339(item.Expired)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("item[%d]: invalid expired: %v", i, err))
			continue
		}

		lastRefresh, err := parseOptionalRFC3339(item.LastRefresh)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("item[%d]: invalid last_refresh: %v", i, err))
			continue
		}

		enabled := true
		if item.Enabled != nil {
			enabled = *item.Enabled
		}

		cred := storage.EndpointCredential{
			EndpointName: endpointName,
			ProviderType: strings.TrimSpace(item.Type),
			AccountID:    strings.TrimSpace(item.AccountID),
			Email:        strings.TrimSpace(item.Email),
			AccessToken:  strings.TrimSpace(item.AccessToken),
			RefreshToken: strings.TrimSpace(item.RefreshToken),
			IDToken:      strings.TrimSpace(item.IDToken),
			LastRefresh:  lastRefresh,
			ExpiresAt:    expiresAt,
			Status:       "active",
			Enabled:      enabled,
			Remark:       strings.TrimSpace(item.Remark),
		}
		if cred.ProviderType == "" {
			cred.ProviderType = "codex"
		}
		if cred.Remark == "" {
			cred.Remark = strings.TrimSpace(req.Remark)
		}

		existingCred := findExistingCredential(accountIndex, emailIndex, &cred)
		if existingCred == nil {
			if err := h.storage.SaveEndpointCredential(&cred); err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("item[%d]: save failed: %v", i, err))
				continue
			}
			created++
		} else {
			if !req.Overwrite {
				skipped++
				continue
			}

			cred.ID = existingCred.ID
			if item.Enabled == nil {
				cred.Enabled = existingCred.Enabled
			}
			if cred.Remark == "" {
				cred.Remark = existingCred.Remark
			}
			if cred.LastRefresh == nil {
				cred.LastRefresh = existingCred.LastRefresh
			}
			if cred.ExpiresAt == nil {
				cred.ExpiresAt = existingCred.ExpiresAt
			}
			if cred.RefreshToken == "" {
				cred.RefreshToken = existingCred.RefreshToken
			}
			if cred.IDToken == "" {
				cred.IDToken = existingCred.IDToken
			}

			if err := h.storage.UpdateEndpointCredential(&cred); err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("item[%d]: update failed: %v", i, err))
				continue
			}
			updated++
		}

		if cred.AccountID != "" {
			accountIndex[cred.AccountID] = cred
		}
		if cred.Email != "" {
			emailIndex[cred.Email] = cred
		}
	}

	WriteSuccess(w, map[string]interface{}{
		"created":   created,
		"updated":   updated,
		"skipped":   skipped,
		"failed":    failed,
		"processed": len(items),
		"errors":    errors,
	})
}

func (h *Handler) updateEndpointCredential(w http.ResponseWriter, r *http.Request, endpointName string, id int64) {
	var req struct {
		AccessToken  *string `json:"accessToken"`
		RefreshToken *string `json:"refreshToken"`
		IDToken      *string `json:"idToken"`
		Remark       *string `json:"remark"`
		Enabled      *bool   `json:"enabled"`
		ExpiresAt    *string `json:"expiresAt"`
		LastRefresh  *string `json:"lastRefresh"`
		Status       *string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cred, err := h.storage.GetCredentialByID(id)
	if err != nil {
		logger.Error("Failed to get credential: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get credential")
		return
	}
	if cred == nil || cred.EndpointName != endpointName {
		WriteError(w, http.StatusNotFound, "Credential not found")
		return
	}

	if req.AccessToken != nil {
		token := strings.TrimSpace(*req.AccessToken)
		if token == "" {
			WriteError(w, http.StatusBadRequest, "accessToken cannot be empty")
			return
		}
		cred.AccessToken = token
	}
	if req.RefreshToken != nil {
		cred.RefreshToken = strings.TrimSpace(*req.RefreshToken)
	}
	if req.IDToken != nil {
		cred.IDToken = strings.TrimSpace(*req.IDToken)
	}
	if req.Remark != nil {
		cred.Remark = strings.TrimSpace(*req.Remark)
	}
	if req.Enabled != nil {
		cred.Enabled = *req.Enabled
	}
	if req.ExpiresAt != nil {
		expiresAt, err := parseOptionalRFC3339(*req.ExpiresAt)
		if err != nil {
			WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid expiresAt: %v", err))
			return
		}
		cred.ExpiresAt = expiresAt
	}
	if req.LastRefresh != nil {
		lastRefresh, err := parseOptionalRFC3339(*req.LastRefresh)
		if err != nil {
			WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid lastRefresh: %v", err))
			return
		}
		cred.LastRefresh = lastRefresh
	}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if status != "" {
			if !isAllowedCredentialStatus(status) {
				WriteError(w, http.StatusBadRequest, "invalid status")
				return
			}
			cred.Status = status
		}
	}

	if err := h.storage.UpdateEndpointCredential(cred); err != nil {
		logger.Error("Failed to update credential: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to update credential")
		return
	}

	updated, err := h.storage.GetCredentialByID(id)
	if err != nil {
		logger.Error("Failed to reload credential: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to reload credential")
		return
	}
	if updated == nil {
		WriteError(w, http.StatusNotFound, "Credential not found")
		return
	}
	updated.AccessToken = maskToken(updated.AccessToken)
	updated.RefreshToken = maskToken(updated.RefreshToken)
	updated.IDToken = maskToken(updated.IDToken)
	WriteSuccess(w, updated)
}

func (h *Handler) deleteEndpointCredential(w http.ResponseWriter, r *http.Request, endpointName string, id int64) {
	if err := h.storage.DeleteEndpointCredential(endpointName, id); err != nil {
		logger.Error("Failed to delete credential: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to delete credential")
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"message": "Credential deleted successfully",
	})
}

func (h *Handler) getEndpointCredentialStats(w http.ResponseWriter, r *http.Request, endpointName string) {
	stats, err := h.storage.GetTokenPoolStats(endpointName)
	if err != nil {
		logger.Error("Failed to get token pool stats: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to get token pool stats")
		return
	}

	WriteSuccess(w, stats)
}

func (h *Handler) getEndpointByName(name string) (*storage.Endpoint, error) {
	endpoints, err := h.storage.GetEndpoints()
	if err != nil {
		return nil, err
	}
	for i := range endpoints {
		if endpoints[i].Name == name {
			return &endpoints[i], nil
		}
	}
	return nil, nil
}

func findExistingCredential(accountIndex map[string]storage.EndpointCredential, emailIndex map[string]storage.EndpointCredential, cred *storage.EndpointCredential) *storage.EndpointCredential {
	if cred.AccountID != "" {
		if existing, ok := accountIndex[cred.AccountID]; ok {
			return &existing
		}
	}
	if cred.Email != "" {
		if existing, ok := emailIndex[cred.Email]; ok {
			return &existing
		}
	}
	return nil
}

func parseImportCredentialsPayload(rawBody []byte) (*importCredentialsRequest, []importCredentialItem, error) {
	var req importCredentialsRequest
	if err := json.Unmarshal(rawBody, &req); err == nil && len(req.Items) > 0 {
		return &req, req.Items, nil
	}

	var items []importCredentialItem
	if err := json.Unmarshal(rawBody, &items); err == nil && len(items) > 0 {
		return &importCredentialsRequest{Items: items}, items, nil
	}

	var single importCredentialItem
	if err := json.Unmarshal(rawBody, &single); err == nil && strings.TrimSpace(single.AccessToken) != "" {
		return &importCredentialsRequest{Items: []importCredentialItem{single}}, []importCredentialItem{single}, nil
	}

	return nil, nil, fmt.Errorf("request body must be a credential object, array, or {items:[...]}")
}

func parseOptionalRFC3339(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, err
	}
	utc := t.UTC()
	return &utc, nil
}

func isAllowedCredentialStatus(status string) bool {
	switch status {
	case "active", "invalid", "cooldown":
		return true
	default:
		return false
	}
}

func maskToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 10 {
		return "****"
	}
	return token[:6] + "..." + token[len(token)-4:]
}
