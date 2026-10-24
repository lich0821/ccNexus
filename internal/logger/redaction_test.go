package logger

import (
	"strings"
	"testing"
)

func TestRedactSensitiveRedactsCommonSecrets(t *testing.T) {
	message := `Authorization: Bearer bearer-secret apiKey=top-secret password=hunter2 {"access_token":"access-secret","refreshToken":"refresh-secret","backup_s3_secretKey":"s3-secret"} https://url-user:url-pass@example.com/v1?api_key=query-secret&region=us`

	redacted := RedactSensitive(message)

	for _, secret := range []string{"bearer-secret", "top-secret", "hunter2", "access-secret", "refresh-secret", "s3-secret", "url-user", "url-pass", "query-secret"} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("redacted log still contains %q: %s", secret, redacted)
		}
	}
	if !strings.Contains(redacted, redactedSecret) || !strings.Contains(redacted, "region=us") {
		t.Fatalf("redacted log lost expected context: %s", redacted)
	}
}

func TestLoggerStoresRedactedMessages(t *testing.T) {
	log := &Logger{entries: make([]LogEntry, 0), maxSize: 10, minLevel: DEBUG, consoleLevel: ERROR}

	log.Log(INFO, "api_key=%s password=%s", "stored-secret", "stored-password")

	entries := log.GetLogs()
	if len(entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(entries))
	}
	if strings.Contains(entries[0].Message, "stored-secret") || strings.Contains(entries[0].Message, "stored-password") {
		t.Fatalf("stored log leaked secret: %s", entries[0].Message)
	}
}
