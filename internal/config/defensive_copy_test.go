package config

import "testing"

func TestMutableConfigGettersReturnDefensiveCopies(t *testing.T) {
	cfg := DefaultConfig()
	cfg.UpdateEndpoints([]Endpoint{{Name: "primary", APIKey: "endpoint-secret"}})
	cfg.UpdateWebDAV(&WebDAVConfig{URL: "https://webdav.example", Username: "alice", Password: "webdav-secret"})
	cfg.UpdateBackup(&BackupConfig{
		Provider: "s3",
		Local:    &LocalBackupConfig{Dir: "/backup"},
		S3: &S3BackupConfig{
			Endpoint:     "https://s3.example",
			Bucket:       "bucket",
			AccessKey:    "access-key",
			SecretKey:    "s3-secret",
			SessionToken: "s3-session",
		},
	})
	cfg.UpdateUpdate(&UpdateConfig{AutoCheck: true, CheckInterval: 12, SkippedVersion: "1.0.0"})
	cfg.UpdateTerminal(&TerminalConfig{SelectedTerminal: "iterm", ProjectDirs: []string{"/project-a"}, ClaudeCommand: "claude"})
	cfg.UpdateProxy(&ProxyConfig{URL: "http://proxy.example"})
	cfg.UpdateCodexProxy(&ProxyConfig{URL: "http://codex-proxy.example"})

	endpoints := cfg.GetEndpoints()
	endpoints[0].APIKey = "mutated"
	if got := cfg.GetEndpoints()[0].APIKey; got != "endpoint-secret" {
		t.Fatalf("GetEndpoints leaked mutable slice, got api key %q", got)
	}

	webdav := cfg.GetWebDAV()
	webdav.Password = "mutated"
	if got := cfg.GetWebDAV().Password; got != "webdav-secret" {
		t.Fatalf("GetWebDAV leaked mutable pointer, got password %q", got)
	}

	backup := cfg.GetBackup()
	backup.Local.Dir = "/mutated"
	backup.S3.SecretKey = "mutated"
	backup.S3.SessionToken = "mutated-session"
	backupAgain := cfg.GetBackup()
	if backupAgain.Local.Dir != "/backup" || backupAgain.S3.SecretKey != "s3-secret" || backupAgain.S3.SessionToken != "s3-session" {
		t.Fatalf("GetBackup leaked nested mutable pointers: %+v", backupAgain)
	}

	update := cfg.GetUpdate()
	update.SkippedVersion = "mutated"
	if got := cfg.GetUpdate().SkippedVersion; got != "1.0.0" {
		t.Fatalf("GetUpdate leaked mutable pointer, got skipped version %q", got)
	}

	terminal := cfg.GetTerminal()
	terminal.ProjectDirs[0] = "/mutated"
	terminal.ClaudeCommand = "mutated"
	terminalAgain := cfg.GetTerminal()
	if terminalAgain.ProjectDirs[0] != "/project-a" || terminalAgain.ClaudeCommand != "claude" {
		t.Fatalf("GetTerminal leaked mutable state: %+v", terminalAgain)
	}

	proxy := cfg.GetProxy()
	proxy.URL = "http://mutated"
	if got := cfg.GetProxy().URL; got != "http://proxy.example" {
		t.Fatalf("GetProxy leaked mutable pointer, got URL %q", got)
	}

	codexProxy := cfg.GetCodexProxy()
	codexProxy.URL = "http://mutated"
	if got := cfg.GetCodexProxy().URL; got != "http://codex-proxy.example" {
		t.Fatalf("GetCodexProxy leaked mutable pointer, got URL %q", got)
	}
}

func TestMutableConfigUpdatesStoreDefensiveCopies(t *testing.T) {
	cfg := DefaultConfig()

	endpoints := []Endpoint{{Name: "primary", APIKey: "endpoint-secret"}}
	cfg.UpdateEndpoints(endpoints)
	endpoints[0].APIKey = "mutated"
	if got := cfg.GetEndpoints()[0].APIKey; got != "endpoint-secret" {
		t.Fatalf("UpdateEndpoints stored caller slice, got api key %q", got)
	}

	webdav := &WebDAVConfig{URL: "https://webdav.example", Password: "webdav-secret"}
	cfg.UpdateWebDAV(webdav)
	webdav.Password = "mutated"
	if got := cfg.GetWebDAV().Password; got != "webdav-secret" {
		t.Fatalf("UpdateWebDAV stored caller pointer, got password %q", got)
	}

	backup := &BackupConfig{
		Provider: "s3",
		Local:    &LocalBackupConfig{Dir: "/backup"},
		S3:       &S3BackupConfig{SecretKey: "s3-secret", SessionToken: "s3-session"},
	}
	cfg.UpdateBackup(backup)
	backup.Local.Dir = "/mutated"
	backup.S3.SecretKey = "mutated"
	backupAgain := cfg.GetBackup()
	if backupAgain.Local.Dir != "/backup" || backupAgain.S3.SecretKey != "s3-secret" {
		t.Fatalf("UpdateBackup stored caller nested pointers: %+v", backupAgain)
	}

	update := &UpdateConfig{SkippedVersion: "1.0.0"}
	cfg.UpdateUpdate(update)
	update.SkippedVersion = "mutated"
	if got := cfg.GetUpdate().SkippedVersion; got != "1.0.0" {
		t.Fatalf("UpdateUpdate stored caller pointer, got skipped version %q", got)
	}

	terminal := &TerminalConfig{SelectedTerminal: "iterm", ProjectDirs: []string{"/project-a"}}
	cfg.UpdateTerminal(terminal)
	terminal.ProjectDirs[0] = "/mutated"
	terminalAgain := cfg.GetTerminal()
	if terminalAgain.ProjectDirs[0] != "/project-a" {
		t.Fatalf("UpdateTerminal stored caller slice: %+v", terminalAgain)
	}

	proxy := &ProxyConfig{URL: "http://proxy.example"}
	cfg.UpdateProxy(proxy)
	proxy.URL = "http://mutated"
	if got := cfg.GetProxy().URL; got != "http://proxy.example" {
		t.Fatalf("UpdateProxy stored caller pointer, got URL %q", got)
	}

	codexProxy := &ProxyConfig{URL: "http://codex-proxy.example"}
	cfg.UpdateCodexProxy(codexProxy)
	codexProxy.URL = "http://mutated"
	if got := cfg.GetCodexProxy().URL; got != "http://codex-proxy.example" {
		t.Fatalf("UpdateCodexProxy stored caller pointer, got URL %q", got)
	}
}
