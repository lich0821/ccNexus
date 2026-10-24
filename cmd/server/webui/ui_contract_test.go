package webui

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func readUIAsset(t *testing.T, path string) string {
	t.Helper()
	data, err := uiFS.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func requireUIAssetContains(t *testing.T, path string, values ...string) {
	t.Helper()
	content := readUIAsset(t, path)
	for _, value := range values {
		if !strings.Contains(content, value) {
			t.Errorf("%s must contain %q", path, value)
		}
	}
}

func TestWebUIUsesDesktopAppSinglePageShell(t *testing.T) {
	index := readUIAsset(t, "ui/index.html")
	for _, value := range []string{
		`class="app-header"`,
		`id="proxy-port"`,
		`id="statistics-panel"`,
		`id="endpoints-panel"`,
		`class="app-footer"`,
	} {
		if !strings.Contains(index, value) {
			t.Errorf("index.html must contain %q", value)
		}
	}
	for _, value := range []string{`id="sidebar"`, `data-view="testing"`, "chart.umd.min.js"} {
		if strings.Contains(index, value) {
			t.Errorf("index.html must not contain legacy navigation or unsupported asset %q", value)
		}
	}
}

func TestEndpointUIUsesDesktopAppCardControls(t *testing.T) {
	requireUIAssetContains(t, "ui/js/components/endpoints.js",
		"endpoint-card-grid",
		"endpoint-card-compact",
		"endpoint-collapse-btn",
		"endpoint-view-btn",
		"endpoint-filter-transformer",
		"endpoint-filter-availability",
		"endpoint-filter-enabled",
	)
}

func TestEndpointTestUsesTemporaryModelPrompt(t *testing.T) {
	content := readUIAsset(t, "ui/js/components/endpoints.js")
	for _, value := range []string{
		"requestTestModel",
		"await api.testEndpoint(name, testModel)",
		"input.value = configuredModel;",
		"test-model-fetch",
		"endpointName: endpoint.name",
	} {
		if !strings.Contains(content, value) {
			t.Errorf("ui/js/components/endpoints.js must contain %q", value)
		}
	}
	if strings.Contains(content, "return Promise.resolve(configuredModel)") {
		t.Error("WebUI test flow must always open the model prompt")
	}
	requireUIAssetContains(t, "ui/js/api.js",
		"testEndpoint(name, model",
		"/test`, { model })",
	)
}

func TestEndpointUIHasScopedActionsAndAccessibleReordering(t *testing.T) {
	requireUIAssetContains(t, "ui/js/components/endpoints.js",
		"this.actionVersion",
		"this.reorderQueue",
		"move-up-btn",
		"move-down-btn",
		"configured ? t('endpoints.tokenConfigured')",
	)
}

func TestEndpointAuthModesDriveFieldsAndCapabilities(t *testing.T) {
	requireUIAssetContains(t, "ui/js/components/endpoints.js",
		"const usesCodexTokenPool = authMode === 'codex_token_pool'",
		"apiUrl.readOnly = usesCodexTokenPool",
		"transformer.disabled = usesCodexTokenPool",
		"fetchModelsButton.hidden = usesCodexTokenPool",
		"apiUrl.value = 'https://chatgpt.com/backend-api/codex'",
		"transformer.value = 'openai2'",
		"endpoints.codexTokenPoolModeHint",
		"endpoints.codexModelsUnavailable",
	)
}

func TestEndpointAPIKeyIsNotPrefilledInPlaintext(t *testing.T) {
	content := readUIAsset(t, "ui/js/components/endpoints.js")
	for _, value := range []string{
		`type="text" class="form-input" id="endpoint-api-key"`,
		`value=""`,
		`hasApiKey ? t('endpoints.apiKeyPlaceholderConfigured') : t('endpoints.apiKeyPlaceholder')`,
		`t('endpoints.keepExistingKey')`,
	} {
		if !strings.Contains(content, value) {
			t.Errorf("endpoints.js must contain %q", value)
		}
	}
	if strings.Contains(content, "endpoint.apiKey") {
		t.Error("endpoints.js must not prefill or read endpoint.apiKey from API responses")
	}
}

func TestEndpointMutationsConvergeAndCloneReportsAfterSave(t *testing.T) {
	content := readUIAsset(t, "ui/js/components/endpoints.js")
	for _, value := range []string{
		"this.mutationQueue",
		"queueMutation",
		"\\(Copy\\)|（副本）",
	} {
		if !strings.Contains(content, value) {
			t.Errorf("endpoints.js must contain %q", value)
		}
	}
	if strings.Count(content, "queueMutation(") < 8 {
		t.Error("all endpoint and credential mutations must use the shared mutation queue")
	}
	testStart := strings.Index(content, "async testEndpoint(")
	if testStart < 0 {
		t.Fatal("testEndpoint block not found")
	}
	testEnd := strings.Index(content[testStart:], "showTestResultModal(")
	if testEnd < 0 || !strings.Contains(content[testStart:testStart+testEnd], "this.renderTable()") {
		t.Error("successful endpoint tests must update the visible status before opening the result modal")
	}
	cloneStart := strings.Index(content, "\n    cloneEndpoint(")
	if cloneStart < 0 {
		t.Fatal("cloneEndpoint block not found")
	}
	cloneEnd := strings.Index(content[cloneStart:], "async showTokenPoolModal(")
	if cloneEnd < 0 {
		t.Fatal("cloneEndpoint block not found")
	}
	if strings.Contains(content[cloneStart:cloneStart+cloneEnd], "notifications.success") {
		t.Error("opening the clone form must not report a successful clone")
	}
}

func TestStatsDiscardStaleResponses(t *testing.T) {
	requireUIAssetContains(t, "ui/js/components/stats.js",
		"this.requestVersion",
		"requestVersion !== this.requestVersion",
		"aria-pressed",
		"renderLoading",
		"renderError",
	)
}

func TestSinglePageRealtimeRefreshUsesStatsComponent(t *testing.T) {
	workingDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("resolve package directory: %v", err)
	}
	script := `
import { pathToFileURL } from 'node:url';
const elements = new Map();
globalThis.document = { getElementById(id) { return elements.get(id) || null; } };
globalThis.window = { addEventListener() {} };
const statsModule = await import(pathToFileURL(process.argv[1]));
const apiModule = await import(pathToFileURL(process.argv[2]));
const pending = [];
apiModule.api.getStatsDaily = () => new Promise(resolve => pending.push(resolve));
apiModule.api.getEndpoints = async () => ({ endpoints: [] });
statsModule.stats.renderStats = value => { statsModule.stats.lastRendered = value.marker; };
const older = statsModule.stats.refreshRealtime();
const newer = statsModule.stats.refreshRealtime();
pending[1]({ marker: 'newer', stats: {} });
await newer;
pending[0]({ marker: 'older', stats: {} });
await older;
if (statsModule.stats.lastRendered !== 'newer') {
  throw new Error(JSON.stringify({ rendered: statsModule.stats.lastRendered }));
}
`
	cmd := exec.Command(
		"node", "--input-type=module", "-e", script,
		filepath.Join(workingDir, "ui/js/components/stats.js"),
		filepath.Join(workingDir, "ui/js/api.js"),
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stats stale-response harness failed: %v\n%s", err, output)
	}
}

func TestModalNavigationAndThemeAccessibilityContracts(t *testing.T) {
	requireUIAssetContains(t, "ui/js/utils/modal.js",
		"inert = true",
		"removeAttribute('aria-hidden')",
	)
	requireUIAssetContains(t, "ui/js/main.js",
		"themeChanged",
		"document.title =",
		"closeAllModals()",
		"activateModal(overlay",
	)
	requireUIAssetContains(t, "ui/css/main.css", "--bg-gradient-start", "prefers-reduced-motion")
	requireUIAssetContains(t, "ui/css/components.css", ".modal--wide", ".endpoint-card-grid")
}

func TestFrontendFallbackErrorsAreLocalized(t *testing.T) {
	requireUIAssetContains(t, "ui/js/api.js",
		"import { t }",
		"errors.invalidJSONResponse",
		"errors.requestFailed",
	)
}

func TestEndpointStatusLabelsDoNotUseNotificationPunctuation(t *testing.T) {
	workingDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("resolve package directory: %v", err)
	}
	script := `
import { pathToFileURL } from 'node:url';
const en = (await import(pathToFileURL(process.argv[1]))).default;
const zh = (await import(pathToFileURL(process.argv[2]))).default;
if (en.endpoints.testFailed !== 'Test failed' || zh.endpoints.testFailed !== '测试失败') {
  throw new Error(JSON.stringify({ en: en.endpoints.testFailed, zh: zh.endpoints.testFailed }));
}
`
	cmd := exec.Command(
		"node", "--input-type=module", "-e", script,
		filepath.Join(workingDir, "ui/js/i18n/en.js"),
		filepath.Join(workingDir, "ui/js/i18n/zh-CN.js"),
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("endpoint status translation harness failed: %v\n%s", err, output)
	}
}

func TestRealtimeRefreshIsCoalescedAndNetworkErrorsAreLocalized(t *testing.T) {
	requireUIAssetContains(t, "ui/js/main.js",
		"let realtimeDisconnected = false",
		"let realtimeRefreshTimer = null",
		"eventSource.onopen = () =>",
		"notifications.warning(t('notifications.realtimeDisconnected'))",
		"notifications.success(t('notifications.realtimeRestored'))",
		"clearTimeout(realtimeRefreshTimer)",
		"stats.refreshRealtime()",
		"state.update('currentEndpoint'",
	)

	requireUIAssetContains(t, "ui/js/components/stats.js",
		"async refreshRealtime()",
		"this.loadStats(this.period, false)",
	)
	requireUIAssetContains(t, "ui/js/api.js",
		"error instanceof TypeError",
		"errors.networkError",
	)
	requireUIAssetContains(t, "ui/js/i18n/zh-CN.js",
		"networkError:",
		"realtimeDisconnected:",
		"realtimeRestored:",
	)
	requireUIAssetContains(t, "ui/js/i18n/en.js",
		"networkError:",
		"realtimeDisconnected:",
		"realtimeRestored:",
	)
}

func TestThemeModalAndMobileTableRegressions(t *testing.T) {
	modal := readUIAsset(t, "ui/js/utils/modal.js")
	if strings.Contains(modal, "queueMicrotask") {
		t.Error("modal focus must move before the previous layer becomes hidden")
	}
	if !strings.Contains(modal, "preferredTarget?.matches(focusableSelector)") {
		t.Error("disabled or hidden initial focus targets must fall back to a focusable modal control")
	}
	focusIndex := strings.LastIndex(modal, "(target || dialog).focus();")
	syncIndex := strings.LastIndex(modal, "syncModalStack();")
	if focusIndex < 0 || syncIndex < focusIndex {
		t.Error("new modal must receive focus before lower modal layers become hidden")
	}
	requireUIAssetContains(t, "ui/css/components.css",
		".inline-alert.error",
		".endpoint-card-actions",
		"@media (max-width: 700px)",
	)
	components := readUIAsset(t, "ui/css/components.css")
	buttonStart := strings.Index(components, ".endpoint-section-header > .btn {")
	buttonEnd := strings.Index(components[buttonStart:], "}")
	if buttonStart < 0 || buttonEnd < 0 || !strings.Contains(components[buttonStart:buttonStart+buttonEnd], "white-space: nowrap") {
		t.Error("the endpoint primary action must remain on one line at tablet widths")
	}
}
