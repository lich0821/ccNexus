package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/storage"
)

// handleConfig handles GET and PUT for full configuration
func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getConfig(w, r)
	case http.MethodPut:
		h.updateConfig(w, r)
	default:
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// getConfig returns the full configuration
func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	WriteSuccess(w, map[string]interface{}{
		"port":       h.config.GetPort(),
		"listenAddr": h.config.GetListenAddr(),
		"logLevel":   h.config.GetLogLevel(),
	})
}

// updateConfig updates the full configuration
func (h *Handler) updateConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Port       int    `json:"port"`
		ListenAddr string `json:"listenAddr"`
		LogLevel   int    `json:"logLevel"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update port if provided
	if req.Port > 0 {
		h.config.UpdatePort(req.Port)
	}

	if strings.TrimSpace(req.ListenAddr) != "" {
		h.config.UpdateListenAddr(req.ListenAddr)
	}

	// Update log level if provided
	if req.LogLevel >= 0 {
		h.config.UpdateLogLevel(req.LogLevel)
	}

	// Save to storage
	adapter := storage.NewConfigStorageAdapter(h.storage)
	if err := h.config.SaveToStorage(adapter); err != nil {
		logger.Error("Failed to save config: %v", err)
		WriteError(w, http.StatusInternalServerError, "Failed to save configuration")
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"message": "Configuration updated successfully",
	})
}

// handleConfigPort handles GET and PUT for port configuration
func (h *Handler) handleConfigPort(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		WriteSuccess(w, map[string]interface{}{
			"port":       h.config.GetPort(),
			"listenAddr": h.config.GetListenAddr(),
		})
	case http.MethodPut:
		var req struct {
			Port       int    `json:"port"`
			ListenAddr string `json:"listenAddr"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Port <= 0 || req.Port > 65535 {
			WriteError(w, http.StatusBadRequest, "Invalid port number")
			return
		}

		if strings.TrimSpace(req.ListenAddr) == "" {
			WriteError(w, http.StatusBadRequest, "Invalid listen address")
			return
		}

		h.config.UpdatePort(req.Port)
		h.config.UpdateListenAddr(req.ListenAddr)

		// Save to storage
		adapter := storage.NewConfigStorageAdapter(h.storage)
		if err := h.config.SaveToStorage(adapter); err != nil {
			logger.Error("Failed to save config: %v", err)
			WriteError(w, http.StatusInternalServerError, "Failed to save configuration")
			return
		}

		WriteSuccess(w, map[string]interface{}{
			"port":       req.Port,
			"listenAddr": req.ListenAddr,
			"message":    "Port and listen address updated successfully (restart required)",
		})
	default:
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleConfigLogLevel handles GET and PUT for log level configuration
func (h *Handler) handleConfigLogLevel(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		WriteSuccess(w, map[string]interface{}{
			"logLevel": h.config.GetLogLevel(),
		})
	case http.MethodPut:
		var req struct {
			LogLevel int `json:"logLevel"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.LogLevel < 0 || req.LogLevel > 3 {
			WriteError(w, http.StatusBadRequest, "Invalid log level (must be 0-3)")
			return
		}

		h.config.UpdateLogLevel(req.LogLevel)

		// Update logger level
		logger.GetLogger().SetMinLevel(logger.LogLevel(req.LogLevel))
		logger.GetLogger().SetConsoleLevel(logger.LogLevel(req.LogLevel))

		// Save to storage
		adapter := storage.NewConfigStorageAdapter(h.storage)
		if err := h.config.SaveToStorage(adapter); err != nil {
			logger.Error("Failed to save config: %v", err)
			WriteError(w, http.StatusInternalServerError, "Failed to save configuration")
			return
		}

		WriteSuccess(w, map[string]interface{}{
			"logLevel": req.LogLevel,
			"message":  "Log level updated successfully",
		})
	default:
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
