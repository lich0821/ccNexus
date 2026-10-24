package logger

import (
	"net/url"
	"regexp"
	"strings"
)

const redactedSecret = "[REDACTED]"

var (
	sensitiveKeyPattern     = `(?:[A-Za-z0-9_.-]*(?:api[-_]?key|access[-_]?token|refresh[-_]?token|id[-_]?token|session[-_]?token|secret[-_]?key|client[-_]?secret|backup[-_]?s3[-_]?(?:secret[-_]?key|secretKey|session[-_]?token|sessionToken)|webdav[-_]?password)[A-Za-z0-9_.-]*|password|passwd|secret|token|cookie|set[-_]?cookie)`
	jsonSensitiveKeyPattern = `(?:` + sensitiveKeyPattern + `|authorization)`
	urlPattern              = regexp.MustCompile(`https?://[^\s"'<>]+`)
	jsonStringPattern       = regexp.MustCompile(`(?i)("` + jsonSensitiveKeyPattern + `"\s*:\s*")([^"\\]*(?:\\.[^"\\]*)*)(")`)
	jsonBarePattern         = regexp.MustCompile(`(?i)("` + jsonSensitiveKeyPattern + `"\s*:\s*)([^",}\]\s]+)`)
	keyValuePattern         = regexp.MustCompile(`(?i)(\b` + sensitiveKeyPattern + `\b\s*[=:]\s*)("[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'|[^&\s,;]+)`)
	apiTokenPattern         = regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{8,}|sk-ant-[A-Za-z0-9_-]{8,}|AIza[0-9A-Za-z_-]{20,}|gh[pousr]_[0-9A-Za-z_]{20,})\b`)
	authorizationPattern    = regexp.MustCompile(`(?i)(\bauthorization\b\s*[:=]\s*)(bearer|basic)?\s*([^\s,;]+)`)
)

// RedactSensitive removes common credential material from log messages before
// they are persisted or printed. It intentionally keeps structural context so
// logs remain useful while API keys, tokens, passwords and URL credentials are
// not exposed.
func RedactSensitive(message string) string {
	if message == "" {
		return ""
	}

	redacted := urlPattern.ReplaceAllStringFunc(message, redactURL)
	redacted = authorizationPattern.ReplaceAllStringFunc(redacted, redactAuthorizationMatch)
	redacted = jsonStringPattern.ReplaceAllString(redacted, `${1}`+redactedSecret+`${3}`)
	redacted = jsonBarePattern.ReplaceAllString(redacted, `${1}`+redactedSecret)
	redacted = keyValuePattern.ReplaceAllString(redacted, `${1}`+redactedSecret)
	redacted = apiTokenPattern.ReplaceAllString(redacted, redactedSecret)
	return redacted
}

func redactAuthorizationMatch(match string) string {
	groups := authorizationPattern.FindStringSubmatch(match)
	if len(groups) != 4 {
		return match
	}
	if groups[2] == "" {
		return groups[1] + redactedSecret
	}
	return groups[1] + groups[2] + " " + redactedSecret
}

func redactURL(raw string) string {
	trimmed := strings.TrimRight(raw, ".,);]")
	suffix := strings.TrimPrefix(raw, trimmed)
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed == nil {
		return raw
	}
	if parsed.User != nil {
		parsed.User = url.UserPassword(redactedSecret, redactedSecret)
	}
	query := parsed.Query()
	for key := range query {
		if isSensitiveKey(key) {
			query.Set(key, redactedSecret)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String() + suffix
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	return strings.Contains(lower, "key") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "password") ||
		strings.Contains(lower, "authorization") ||
		strings.Contains(lower, "passwd")
}
