package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
	"github.com/lich0821/ccNexus/internal/storage"
)

const codexRateLimitTimeout = 30 * time.Second

type codexRateLimitStatusPayload struct {
	PlanType             string                           `json:"plan_type"`
	RateLimit            *codexRateLimitStatusDetails     `json:"rate_limit"`
	Credits              *codexCreditStatusDetails        `json:"credits"`
	AdditionalRateLimits []codexAdditionalRateLimitDetail `json:"additional_rate_limits"`
}

type codexRateLimitStatusDetails struct {
	PrimaryWindow   *codexRateLimitWindowSnapshot `json:"primary_window"`
	SecondaryWindow *codexRateLimitWindowSnapshot `json:"secondary_window"`
}

type codexRateLimitWindowSnapshot struct {
	UsedPercent        float64 `json:"used_percent"`
	LimitWindowSeconds int64   `json:"limit_window_seconds"`
	ResetAfterSeconds  int64   `json:"reset_after_seconds"`
	ResetAt            int64   `json:"reset_at"`
}

type codexCreditStatusDetails struct {
	HasCredits bool    `json:"has_credits"`
	Unlimited  bool    `json:"unlimited"`
	Balance    *string `json:"balance"`
}

type codexAdditionalRateLimitDetail struct {
	LimitName      string                       `json:"limit_name"`
	MeteredFeature string                       `json:"metered_feature"`
	RateLimit      *codexRateLimitStatusDetails `json:"rate_limit"`
}

type codexRateLimitEvent struct {
	Type             string                      `json:"type"`
	PlanType         string                      `json:"plan_type"`
	RateLimits       *codexRateLimitEventDetails `json:"rate_limits"`
	Credits          *codexRateLimitEventCredits `json:"credits"`
	MeteredLimitName string                      `json:"metered_limit_name"`
	LimitName        string                      `json:"limit_name"`
}

type codexRateLimitEventDetails struct {
	Primary   *codexRateLimitEventWindow `json:"primary"`
	Secondary *codexRateLimitEventWindow `json:"secondary"`
}

type codexRateLimitEventWindow struct {
	UsedPercent   float64 `json:"used_percent"`
	WindowMinutes *int64  `json:"window_minutes"`
	ResetAt       *int64  `json:"reset_at"`
}

type codexRateLimitEventCredits struct {
	HasCredits bool    `json:"has_credits"`
	Unlimited  bool    `json:"unlimited"`
	Balance    *string `json:"balance"`
}

type CodexRateLimitFetchResult struct {
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

func (p *Proxy) FetchCodexRateLimits(endpoint config.Endpoint, credentialID int64) (CodexRateLimitFetchResult, error) {
	if p == nil || p.storage == nil {
		return CodexRateLimitFetchResult{}, fmt.Errorf("token storage unavailable")
	}
	if config.NormalizeAuthMode(endpoint.AuthMode) != config.AuthModeCodexTokenPool {
		return CodexRateLimitFetchResult{}, fmt.Errorf("codex token pool required")
	}

	credentials := make([]storage.EndpointCredential, 0)
	if credentialID > 0 {
		cred, err := p.storage.GetCredentialByID(credentialID)
		if err != nil {
			return CodexRateLimitFetchResult{}, fmt.Errorf("failed to load credential: %w", err)
		}
		if cred == nil || cred.EndpointName != endpoint.Name {
			return CodexRateLimitFetchResult{}, fmt.Errorf("credential not found")
		}
		credentials = append(credentials, *cred)
	} else {
		list, err := p.storage.GetEndpointCredentials(endpoint.Name)
		if err != nil {
			return CodexRateLimitFetchResult{}, fmt.Errorf("failed to load credentials: %w", err)
		}
		credentials = append(credentials, list...)
	}

	result := CodexRateLimitFetchResult{}
	for i := range credentials {
		cred := &credentials[i]
		if !cred.Enabled || strings.TrimSpace(cred.AccessToken) == "" {
			result.Skipped++
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), codexRateLimitTimeout)
		data, status, err := p.fetchCodexRateLimitsForCredential(ctx, endpoint, cred, false)
		cancel()

		if err != nil {
			if status == "unauthorized" && strings.TrimSpace(cred.RefreshToken) != "" {
				refreshed, refreshErr := p.refreshCredential(endpoint, cred)
				if refreshErr == nil && refreshed != nil {
					ctx, cancel := context.WithTimeout(context.Background(), codexRateLimitTimeout)
					data, status, err = p.fetchCodexRateLimitsForCredential(ctx, endpoint, refreshed, true)
					cancel()
				} else {
					err = refreshErr
				}
			}
		}

		storeErr := p.storeCredentialRateLimits(cred.ID, data, status, err)
		if storeErr != nil {
			logger.Warn("[%s] Failed to store codex rate limits (id=%d): %v", endpoint.Name, cred.ID, storeErr)
		}
		if err != nil {
			result.Failed++
		} else {
			result.Updated++
		}
	}

	return result, nil
}

func (p *Proxy) fetchCodexRateLimitsForCredential(ctx context.Context, endpoint config.Endpoint, credential *storage.EndpointCredential, retrying bool) (*storage.CodexRateLimitsData, string, error) {
	if credential == nil {
		return nil, "invalid", fmt.Errorf("credential is nil")
	}
	token := strings.TrimSpace(credential.AccessToken)
	if token == "" {
		return nil, "missing_token", fmt.Errorf("access token is empty")
	}

	baseURL := normalizeCodexRateLimitBaseURL(endpoint.APIUrl)
	url := buildCodexRateLimitURL(baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "error", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", codexUserAgent)
	req.Header.Set("Version", codexClientVersion)
	if accountID := strings.TrimSpace(credential.AccountID); accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}

	resp, err := p.codexRateLimitHTTPClient().Do(req)
	if err != nil {
		return nil, "network", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "network", err
	}
	if resp.StatusCode != http.StatusOK {
		errMsg := truncateForLog(string(body), 300)
		if isLikelyHTMLResponse(body) {
			return nil, "blocked", fmt.Errorf("blocked (%d): %s", resp.StatusCode, errMsg)
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, "unauthorized", fmt.Errorf("unauthorized (%d): %s", resp.StatusCode, errMsg)
		}
		if resp.StatusCode >= 500 && !retrying {
			return nil, "upstream", fmt.Errorf("upstream (%d): %s", resp.StatusCode, errMsg)
		}
		return nil, "error", fmt.Errorf("unexpected status (%d): %s", resp.StatusCode, errMsg)
	}

	var payload codexRateLimitStatusPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "parse_error", fmt.Errorf("decode response failed: %w", err)
	}

	data := buildCodexRateLimitData(payload)
	if data == nil {
		return nil, "empty", fmt.Errorf("rate limit payload empty")
	}
	return data, "ok", nil
}

func (p *Proxy) captureCodexRateLimitsFromHeaders(endpoint config.Endpoint, credentialID int64, headers http.Header) {
	if p == nil || p.storage == nil || credentialID <= 0 {
		return
	}
	if config.NormalizeAuthMode(endpoint.AuthMode) != config.AuthModeCodexTokenPool {
		return
	}
	data := parseCodexRateLimitsFromHeaders(headers)
	if data == nil {
		return
	}
	if err := p.storeCredentialRateLimits(credentialID, data, "ok", nil); err != nil {
		logger.Debug("[%s] Failed to persist rate limits from headers: %v", endpoint.Name, err)
	}
}

func (p *Proxy) captureCodexRateLimitsFromEvent(endpoint config.Endpoint, credentialID int64, eventData []byte) {
	if p == nil || p.storage == nil || credentialID <= 0 {
		return
	}
	if config.NormalizeAuthMode(endpoint.AuthMode) != config.AuthModeCodexTokenPool {
		return
	}
	data := parseCodexRateLimitsFromEvent(eventData)
	if data == nil {
		return
	}
	if err := p.storeCredentialRateLimits(credentialID, data, "ok", nil); err != nil {
		logger.Debug("[%s] Failed to persist rate limits from event: %v", endpoint.Name, err)
	}
}

func (p *Proxy) storeCredentialRateLimits(credentialID int64, data *storage.CodexRateLimitsData, status string, err error) error {
	if p == nil || p.storage == nil || credentialID <= 0 {
		return nil
	}
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	if data != nil && data.Source == "" {
		data.Source = "api"
	}
	return p.storage.UpsertCredentialRateLimits(credentialID, data, status, errMsg, time.Now().UTC())
}

func codexRateLimitHTTPClient() *http.Client {
	return &http.Client{Timeout: codexRateLimitTimeout}
}

func (p *Proxy) codexRateLimitHTTPClient() *http.Client {
	client := codexRateLimitHTTPClient()
	if p != nil && p.httpClient != nil {
		client.Transport = p.httpClient.Transport
	}
	if p == nil || p.config == nil {
		return client
	}

	proxyCfg := p.config.GetProxy()
	codexProxyCfg := p.config.GetCodexProxy()
	proxyURL := ""
	if codexProxyCfg != nil && strings.TrimSpace(codexProxyCfg.URL) != "" {
		proxyURL = codexProxyCfg.URL
	} else if proxyCfg != nil && strings.TrimSpace(proxyCfg.URL) != "" {
		proxyURL = proxyCfg.URL
	}
	if strings.TrimSpace(proxyURL) == "" {
		return client
	}
	transport, err := CreateProxyTransport(proxyURL)
	if err != nil {
		logger.Warn("Failed to create proxy transport for rate limits: %v", err)
		return client
	}
	client.Transport = transport
	logger.Debug("Using proxy for rate limits: %s", proxyURL)
	return client
}

func normalizeCodexRateLimitBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "https://chatgpt.com/backend-api"
	}
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed == nil {
		return strings.TrimSuffix(trimmed, "/")
	}
	cleanPath := path.Clean(strings.TrimSpace(parsed.Path))
	if cleanPath == "." || cleanPath == "/" {
		cleanPath = ""
	}
	cleanPath = strings.TrimSuffix(cleanPath, "/")
	cleanPath = strings.TrimSuffix(cleanPath, "/v1")
	if strings.Contains(cleanPath, "/backend-api") {
		idx := strings.Index(cleanPath, "/backend-api")
		cleanPath = cleanPath[:idx+len("/backend-api")]
	}
	parsed.Path = cleanPath
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimSuffix(parsed.String(), "/")
}

func buildCodexRateLimitURL(baseURL string) string {
	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if strings.Contains(base, "/backend-api") {
		return base + "/wham/usage"
	}
	return base + "/api/codex/usage"
}

func buildCodexRateLimitData(payload codexRateLimitStatusPayload) *storage.CodexRateLimitsData {
	planType := strings.TrimSpace(payload.PlanType)
	if payload.RateLimit == nil && payload.Credits == nil && len(payload.AdditionalRateLimits) == 0 {
		return nil
	}
	snapshots := make(map[string]storage.CodexRateLimitSnapshot)

	baseSnapshot := makeCodexRateLimitSnapshot("codex", "", payload.RateLimit, payload.Credits, planType)
	snapshots["codex"] = baseSnapshot

	for _, extra := range payload.AdditionalRateLimits {
		limitID := strings.TrimSpace(extra.MeteredFeature)
		if limitID == "" {
			limitID = strings.TrimSpace(extra.LimitName)
		}
		if limitID == "" {
			continue
		}
		snapshots[limitID] = makeCodexRateLimitSnapshot(limitID, extra.LimitName, extra.RateLimit, nil, planType)
	}

	if len(snapshots) == 0 {
		return nil
	}

	primary := snapshots["codex"]
	data := &storage.CodexRateLimitsData{
		Snapshot:  &primary,
		ByLimitID: snapshots,
		Source:    "api",
	}
	return data
}

func makeCodexRateLimitSnapshot(limitID, limitName string, rateLimit *codexRateLimitStatusDetails, credits *codexCreditStatusDetails, planType string) storage.CodexRateLimitSnapshot {
	primary, secondary := mapCodexRateLimitWindows(rateLimit)
	var creditSnap *storage.CodexCreditsSnapshot
	if credits != nil {
		balance := ""
		if credits.Balance != nil {
			balance = strings.TrimSpace(*credits.Balance)
		}
		creditSnap = &storage.CodexCreditsSnapshot{
			HasCredits: credits.HasCredits,
			Unlimited:  credits.Unlimited,
			Balance:    balance,
		}
	}
	return storage.CodexRateLimitSnapshot{
		LimitID:   strings.TrimSpace(limitID),
		LimitName: strings.TrimSpace(limitName),
		Primary:   primary,
		Secondary: secondary,
		Credits:   creditSnap,
		PlanType:  planType,
	}
}

func mapCodexRateLimitWindows(details *codexRateLimitStatusDetails) (*storage.CodexRateLimitWindow, *storage.CodexRateLimitWindow) {
	if details == nil {
		return nil, nil
	}
	primary := mapCodexRateLimitWindow(details.PrimaryWindow)
	secondary := mapCodexRateLimitWindow(details.SecondaryWindow)
	return primary, secondary
}

func mapCodexRateLimitWindow(snapshot *codexRateLimitWindowSnapshot) *storage.CodexRateLimitWindow {
	if snapshot == nil {
		return nil
	}
	windowMinutes := windowMinutesFromSeconds(snapshot.LimitWindowSeconds)
	resetsAt := snapshot.ResetAt
	return &storage.CodexRateLimitWindow{
		UsedPercent:   snapshot.UsedPercent,
		WindowMinutes: windowMinutes,
		ResetsAt:      toInt64Ptr(resetsAt),
	}
}

func windowMinutesFromSeconds(seconds int64) *int64 {
	if seconds <= 0 {
		return nil
	}
	minutes := (seconds + 59) / 60
	return &minutes
}

func toInt64Ptr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

func parseCodexRateLimitsFromHeaders(headers http.Header) *storage.CodexRateLimitsData {
	if headers == nil {
		return nil
	}
	limits := make(map[string]storage.CodexRateLimitSnapshot)

	getSnapshot := func(limitID string) *storage.CodexRateLimitSnapshot {
		limitID = normalizeLimitID(limitID)
		if existing, ok := limits[limitID]; ok {
			return &existing
		}
		snapshot := storage.CodexRateLimitSnapshot{LimitID: limitID}
		limits[limitID] = snapshot
		return &snapshot
	}

	updateSnapshot := func(limitID string, snap storage.CodexRateLimitSnapshot) {
		if limitID == "" {
			return
		}
		limits[normalizeLimitID(limitID)] = snap
	}

	for name, values := range headers {
		lower := strings.ToLower(strings.TrimSpace(name))
		if !strings.HasPrefix(lower, "x-") {
			continue
		}

		if lower == "x-codex-credits-has-credits" || lower == "x-codex-credits-unlimited" || lower == "x-codex-credits-balance" {
			snap := getSnapshot("codex")
			credits := snap.Credits
			if credits == nil {
				credits = &storage.CodexCreditsSnapshot{}
			}
			switch lower {
			case "x-codex-credits-has-credits":
				if v, ok := parseBool(values); ok {
					credits.HasCredits = v
				}
			case "x-codex-credits-unlimited":
				if v, ok := parseBool(values); ok {
					credits.Unlimited = v
				}
			case "x-codex-credits-balance":
				if v := firstHeader(values); v != "" {
					credits.Balance = v
				}
			}
			snap.Credits = credits
			updateSnapshot("codex", *snap)
			continue
		}

		if strings.HasSuffix(lower, "-limit-name") {
			limitID := strings.TrimPrefix(strings.TrimSuffix(lower, "-limit-name"), "x-")
			if limitID == "" {
				continue
			}
			snap := getSnapshot(limitID)
			if v := firstHeader(values); v != "" {
				snap.LimitName = v
			}
			updateSnapshot(limitID, *snap)
			continue
		}

		limitID, window, field, ok := parseRateLimitHeaderKey(lower)
		if !ok {
			continue
		}
		snap := getSnapshot(limitID)
		var windowRef **storage.CodexRateLimitWindow
		if window == "primary" {
			windowRef = &snap.Primary
		} else {
			windowRef = &snap.Secondary
		}
		if *windowRef == nil {
			*windowRef = &storage.CodexRateLimitWindow{}
		}
		target := *windowRef
		switch field {
		case "used_percent":
			if v, ok := parseFloat(values); ok {
				target.UsedPercent = v
			}
		case "window_minutes":
			if v, ok := parseInt64(values); ok {
				target.WindowMinutes = &v
			}
		case "reset_at":
			if v, ok := parseInt64(values); ok {
				target.ResetsAt = &v
			}
		}
		updateSnapshot(limitID, *snap)
	}

	if len(limits) == 0 {
		return nil
	}
	for id, snap := range limits {
		if snap.LimitID == "" {
			snap.LimitID = id
			limits[id] = snap
		}
	}

	primary, ok := limits["codex"]
	if !ok {
		for _, snap := range limits {
			primary = snap
			break
		}
	}
	data := &storage.CodexRateLimitsData{
		Snapshot:  &primary,
		ByLimitID: limits,
		Source:    "headers",
	}
	return data
}

func parseRateLimitHeaderKey(lower string) (limitID, window, field string, ok bool) {
	switch {
	case strings.HasSuffix(lower, "-primary-used-percent"):
		return strings.TrimPrefix(strings.TrimSuffix(lower, "-primary-used-percent"), "x-"), "primary", "used_percent", true
	case strings.HasSuffix(lower, "-primary-window-minutes"):
		return strings.TrimPrefix(strings.TrimSuffix(lower, "-primary-window-minutes"), "x-"), "primary", "window_minutes", true
	case strings.HasSuffix(lower, "-primary-reset-at"):
		return strings.TrimPrefix(strings.TrimSuffix(lower, "-primary-reset-at"), "x-"), "primary", "reset_at", true
	case strings.HasSuffix(lower, "-secondary-used-percent"):
		return strings.TrimPrefix(strings.TrimSuffix(lower, "-secondary-used-percent"), "x-"), "secondary", "used_percent", true
	case strings.HasSuffix(lower, "-secondary-window-minutes"):
		return strings.TrimPrefix(strings.TrimSuffix(lower, "-secondary-window-minutes"), "x-"), "secondary", "window_minutes", true
	case strings.HasSuffix(lower, "-secondary-reset-at"):
		return strings.TrimPrefix(strings.TrimSuffix(lower, "-secondary-reset-at"), "x-"), "secondary", "reset_at", true
	default:
		return "", "", "", false
	}
}

func parseCodexRateLimitsFromEvent(eventData []byte) *storage.CodexRateLimitsData {
	scanner := bufio.NewScanner(bytes.NewReader(eventData))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		jsonData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if jsonData == "" || jsonData == "[DONE]" {
			continue
		}

		var evt codexRateLimitEvent
		if err := json.Unmarshal([]byte(jsonData), &evt); err != nil {
			continue
		}
		if evt.Type != "codex.rate_limits" {
			continue
		}

		limitID := strings.TrimSpace(evt.MeteredLimitName)
		if limitID == "" {
			limitID = strings.TrimSpace(evt.LimitName)
		}
		if limitID == "" {
			limitID = "codex"
		}

		snapshot := storage.CodexRateLimitSnapshot{
			LimitID:   limitID,
			LimitName: strings.TrimSpace(evt.LimitName),
			PlanType:  strings.TrimSpace(evt.PlanType),
		}
		if evt.RateLimits != nil {
			snapshot.Primary = mapCodexRateLimitEventWindow(evt.RateLimits.Primary)
			snapshot.Secondary = mapCodexRateLimitEventWindow(evt.RateLimits.Secondary)
		}
		if evt.Credits != nil {
			balance := ""
			if evt.Credits.Balance != nil {
				balance = strings.TrimSpace(*evt.Credits.Balance)
			}
			snapshot.Credits = &storage.CodexCreditsSnapshot{
				HasCredits: evt.Credits.HasCredits,
				Unlimited:  evt.Credits.Unlimited,
				Balance:    balance,
			}
		}

		data := &storage.CodexRateLimitsData{
			Snapshot:  &snapshot,
			ByLimitID: map[string]storage.CodexRateLimitSnapshot{limitID: snapshot},
			Source:    "sse",
		}
		return data
	}
	return nil
}

func mapCodexRateLimitEventWindow(window *codexRateLimitEventWindow) *storage.CodexRateLimitWindow {
	if window == nil {
		return nil
	}
	return &storage.CodexRateLimitWindow{
		UsedPercent:   window.UsedPercent,
		WindowMinutes: window.WindowMinutes,
		ResetsAt:      window.ResetAt,
	}
}

func normalizeLimitID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(strings.ReplaceAll(trimmed, "_", "-"))
}

func parseBool(values []string) (bool, bool) {
	value := strings.TrimSpace(firstHeader(values))
	if value == "" {
		return false, false
	}
	parsed, err := strconv.ParseBool(value)
	return parsed, err == nil
}

func parseInt64(values []string) (int64, bool) {
	value := strings.TrimSpace(firstHeader(values))
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return parsed, err == nil
}

func parseFloat(values []string) (float64, bool) {
	value := strings.TrimSpace(firstHeader(values))
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	return parsed, err == nil
}

func firstHeader(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func isLikelyHTMLResponse(body []byte) bool {
	trimmed := strings.ToLower(strings.TrimSpace(string(body)))
	if trimmed == "" {
		return false
	}
	return strings.HasPrefix(trimmed, "<!doctype html") ||
		strings.HasPrefix(trimmed, "<html") ||
		strings.Contains(trimmed, "<head>") ||
		strings.Contains(trimmed, "<body")
}
