package logger

import (
	"net/url"
	"regexp"
	"strings"
)

const redactedSecret = "[REDACTED]"

var (
	sensitiveKeyPattern  = `(?:api[-_]?key|authorization|password|passwd|secret|token|access[-_]?token|refresh[-_]?token|id[-_]?token|session[-_]?token|secret[-_]?key|client[-_]?secret|backup[-_]?s3[-_]?secretKey|backup[-_]?s3[-_]?sessionToken|webdav[-_]?password)`
	urlPattern           = regexp.MustCompile(`https?://[^\s"'<>]+`)
	jsonStringPattern    = regexp.MustCompile(`(?i)("` + sensitiveKeyPattern + `"\s*:\s*")([^"\\]*(?:\\.[^"\\]*)*)(")`)
	jsonBarePattern      = regexp.MustCompile(`(?i)("` + sensitiveKeyPattern + `"\s*:\s*)([^",}\]\s]+)`)
	keyValuePattern      = regexp.MustCompile(`(?i)(\b` + sensitiveKeyPattern + `\b\s*[=:]\s*)([^&\s,;]+)`)
	authorizationPattern = regexp.MustCompile(`(?i)(\bauthorization\b\s*[:=]\s*)(bearer|basic)?\s*([^\s,;]+)`)
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
