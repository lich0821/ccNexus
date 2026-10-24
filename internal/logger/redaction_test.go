package logger

import (
	"os"
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

func TestRedactSensitiveRedactsHeaderEnvAndTokenPrefixes(t *testing.T) {
	message := `X-Api-Key: sk-testsecret123 OPENAI_API_KEY="env-secret" Cookie: session=raw-cookie client_secret='quoted-secret' Authorization=Basic basic-secret url=https://example.com?session_token=query-session`

	redacted := RedactSensitive(message)

	for _, secret := range []string{"sk-testsecret123", "env-secret", "raw-cookie", "quoted-secret", "basic-secret", "query-session"} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("redacted log still contains %q: %s", secret, redacted)
		}
	}
	for _, context := range []string{"X-Api-Key:", "OPENAI_API_KEY=", "Cookie:", "client_secret=", "Authorization=Basic", "https://example.com"} {
		if !strings.Contains(redacted, context) {
			t.Fatalf("redacted log lost context %q: %s", context, redacted)
		}
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

func TestDebugFileWritesRedactedMessages(t *testing.T) {
	log := &Logger{entries: make([]LogEntry, 0), maxSize: 10, minLevel: DEBUG, consoleLevel: ERROR}
	debugPath := t.TempDir() + "/debug.log"
	if err := log.EnableDebugFile(debugPath); err != nil {
		t.Fatalf("enable debug file: %v", err)
	}
	t.Cleanup(log.Close)

	log.DebugLog("Authorization: Bearer %s apiKey=%s", "debug-bearer-secret", "debug-api-secret")
	log.Close()

	content, err := os.ReadFile(debugPath)
	if err != nil {
		t.Fatalf("read debug file: %v", err)
	}
	debugLog := string(content)
	if strings.Contains(debugLog, "debug-bearer-secret") || strings.Contains(debugLog, "debug-api-secret") {
		t.Fatalf("debug file leaked secret: %s", debugLog)
	}
	if !strings.Contains(debugLog, redactedSecret) {
		t.Fatalf("debug file did not contain redaction marker: %s", debugLog)
	}
}
