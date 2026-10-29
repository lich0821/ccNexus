package config

import (
	"net/url"
	"path"
	"strings"
)

// NormalizeAPIPathForBaseURL avoids appending a second API version when the
// configured base URL already ends in a version, such as /v1 or /api/paas/v4.
func NormalizeAPIPathForBaseURL(baseURL, apiPath string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return apiPath
	}
	cleanPath := path.Clean(strings.TrimSpace(parsed.Path))
	segment := path.Base(cleanPath)
	version := strings.TrimPrefix(segment, "v")
	if version != segment && version != "" && strings.Trim(version, "0123456789") == "" && strings.HasPrefix(apiPath, "/v1/") {
		return strings.TrimPrefix(apiPath, "/v1")
	}

	if strings.HasSuffix(cleanPath, "/backend-api/codex") {
		switch apiPath {
		case "/v1/responses":
			return "/responses"
		case "/v1/responses/compact":
			return "/responses/compact"
		}
	}
	return apiPath
}
