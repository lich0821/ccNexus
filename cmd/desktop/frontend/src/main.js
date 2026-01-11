import './style.css'
import './effects/festival-effects.css'
import '../wailsjs/runtime/runtime.js'
import { setLanguage } from './i18n/index.js'
import { initUI, changeLanguage } from './modules/ui.js'
import { loadConfig } from './modules/config.js'
import { loadStats, switchStatsPeriod, loadStatsByPeriod, getCurrentPeriod } from './modules/stats.js'
import { renderEndpoints, toggleEndpointPanel, initEndpointSuccessListener, checkAllEndpointsOnStartup, switchEndpointViewMode, initEndpointViewMode, isDropdownOpen } from './modules/endpoints.js'
import { loadLogs, toggleLogPanel, changeLogLevel, copyLogs, clearLogs } from './modules/logs.js'
import { showDataSyncDialog } from './modules/webdav.js'
import { initTips } from './modules/tips.js'
import { initTerminal } from './modules/terminal.js'
import { initSession } from './modules/session.js'
import { showSettingsModal, closeSettingsModal, saveSettings, applyTheme, initTheme, showAutoThemeConfigModal, closeAutoThemeConfigModal, saveAutoThemeConfig } from './modules/settings.js'
import { checkUpdatesOnStartup, checkForUpdates, initUpdateSettings } from './modules/updater.js'
import { initBroadcast } from './modules/broadcast.js'
import { initSponsor, showSponsorModal, closeSponsorModal, openSponsorLink } from './modules/sponsor.js'
import {
    showAddEndpointModal,
    editEndpoint,
    saveEndpoint,
    deleteEndpoint,
    closeModal,
    handleTransformerChange,
    fetchModels,
    initModelInputEvents,
    toggleModelDropdown,
    showEditPortModal,
    savePort,
    closePortModal,
    showWelcomeModal,
    closeWelcomeModal,
    showWelcomeModalIfFirstTime,
    showChangelogModal,
    closeChangelogModal,
    showChangelogIfNewVersion,
    testEndpointHandler,
    closeTestResultModal,
    openGitHub,
    openArticle,
    togglePasswordVisibility,
    acceptConfirm,
    cancelConfirm,
    showCloseActionDialog,
    quitApplication,
    minimizeToTray
} from './modules/modal.js'

// Load data on startup
window.addEventListener('DOMContentLoaded', async () => {
    // Wait for Wails runtime to be ready
    while (!window.go) {
        await new Promise(resolve => setTimeout(resolve, 100));
    }

    // Initialize language
    const lang = await window.go.main.App.GetLanguage();
    setLanguage(lang);

    // Initialize theme (supports auto mode)
    await initTheme();

    // Initialize UI
    initUI();

    // Initialize endpoint view mode
    initEndpointViewMode();

    // Initialize terminal module
    initTerminal();

    // Initialize session module
    initSession();

    // Initialize model input events
    initModelInputEvents();

    // Load and display version
    try {
        const version = await window.go.main.App.GetVersion();
        document.getElementById('appVersion').textContent = version;
    } catch (error) {
        console.error('Failed to get version:', error);
    }

    // Load initial data
    await loadConfigAndRender();
    loadStatsByPeriod('daily'); // Load today's stats by default

    // Restore log level from config
    try {
        const logLevel = await window.go.main.App.GetLogLevel();
        document.getElementById('logLevel').value = logLevel;
    } catch (error) {
        console.error('Failed to get log level:', error);
    }

    loadLogs();

    // Initialize tips
    initTips();

    // Initialize endpoint success listener
    initEndpointSuccessListener();

    // Check all endpoints on startup (zero-cost methods only)
    checkAllEndpointsOnStartup();

    // Refresh stats every 3 seconds
    setInterval(async () => {
        await loadStats(); // Refresh cumulative stats for endpoint cards
        const currentPeriod = getCurrentPeriod(); // Get current selected period
        await loadStatsByPeriod(currentPeriod); // Refresh period stats (daily/weekly/monthly)
        // 如果下拉菜单打开，跳过渲染避免菜单消失
        if (isDropdownOpen()) {
            return;
        }
        const config = await window.go.main.App.GetConfig();
        if (config) {
            renderEndpoints(JSON.parse(config).endpoints);
        }
    }, 3000);

    // Refresh logs every 2 seconds
    setInterval(loadLogs, 2000);

    // Show welcome modal on first launch
    showWelcomeModalIfFirstTime();
    // showChangelogIfNewVersion(); // 暂时禁用自动弹窗

    // Check for updates on startup
    checkUpdatesOnStartup();

    // Initialize broadcast banner
    initBroadcast();

    // Initialize sponsor module
    initSponsor();

    // Initialize update settings
    initUpdateSettings();

    // Listen for close dialog event from backend
    if (window.runtime) {
        window.runtime.EventsOn('show-close-dialog', () => {
            showCloseActionDialog();
        });
    }

    // Handle Cmd/Ctrl+W to hide window
    window.addEventListener('keydown', (e) => {
        if ((e.metaKey || e.ctrlKey) && e.key === 'w') {
            e.preventDefault();
            window.runtime.WindowHide();
        }
    });
});

// Helper function to load config and render endpoints
async function loadConfigAndRender() {
    const config = await loadConfig();
    if (config) {
        renderEndpoints(config.endpoints);
    }
}

// Expose functions to window for onclick handlers
window.loadConfig = loadConfigAndRender;
window.showAddEndpointModal = showAddEndpointModal;
window.editEndpoint = editEndpoint;
window.saveEndpoint = saveEndpoint;
window.deleteEndpoint = deleteEndpoint;
window.closeModal = closeModal;
window.handleTransformerChange = handleTransformerChange;
window.fetchModels = fetchModels;
window.toggleModelDropdown = toggleModelDropdown;
window.showEditPortModal = showEditPortModal;
window.savePort = savePort;
window.closePortModal = closePortModal;
window.showWelcomeModal = showWelcomeModal;
window.closeWelcomeModal = closeWelcomeModal;
window.showChangelogModal = showChangelogModal;
window.closeChangelogModal = closeChangelogModal;
window.testEndpoint = testEndpointHandler;
window.closeTestResultModal = closeTestResultModal;
window.openGitHub = openGitHub;
window.openArticle = openArticle;
window.toggleLogPanel = toggleLogPanel;
window.changeLogLevel = changeLogLevel;
window.copyLogs = copyLogs;
window.clearLogs = clearLogs;
window.changeLanguage = changeLanguage;
window.togglePasswordVisibility = togglePasswordVisibility;
window.acceptConfirm = acceptConfirm;
window.checkForUpdates = checkForUpdates;
window.cancelConfirm = cancelConfirm;
window.showCloseActionDialog = showCloseActionDialog;
window.quitApplication = quitApplication;
window.minimizeToTray = minimizeToTray;
window.showDataSyncDialog = showDataSyncDialog;
window.switchStatsPeriod = switchStatsPeriod;
window.toggleEndpointPanel = toggleEndpointPanel;
window.switchEndpointViewMode = switchEndpointViewMode;
window.showSettingsModal = showSettingsModal;
window.closeSettingsModal = closeSettingsModal;
window.saveSettings = saveSettings;
window.showAutoThemeConfigModal = showAutoThemeConfigModal;
window.closeAutoThemeConfigModal = closeAutoThemeConfigModal;
window.saveAutoThemeConfig = saveAutoThemeConfig;
window.showSponsorModal = showSponsorModal;
window.closeSponsorModal = closeSponsorModal;
window.openSponsorLink = openSponsorLink;

// History modal functions
window.closeHistoryModal = async () => {
    const { closeHistoryModal } = await import('./modules/history.js');
    closeHistoryModal();
};

window.deleteHistoryArchive = async () => {
    const { deleteHistoryArchive } = await import('./modules/history.js');
    deleteHistoryArchive();
};
