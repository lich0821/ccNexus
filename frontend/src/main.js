import './style.css'
import '../wailsjs/runtime/runtime.js'
import { setLanguage } from './i18n/index.js'
import { initUI, changeLanguage } from './modules/ui.js'
import { loadConfig } from './modules/config.js'
import { loadStats, switchStatsPeriod, loadStatsByPeriod, getCurrentPeriod } from './modules/stats.js'
import { renderEndpoints, toggleEndpointPanel } from './modules/endpoints.js'
import { loadLogs, toggleLogPanel, changeLogLevel, copyLogs, clearLogs } from './modules/logs.js'
import { showDataSyncDialog } from './modules/webdav.js'
import { initTips } from './modules/tips.js'
import { showSettingsModal, closeSettingsModal, saveSettings, applyTheme } from './modules/settings.js'
import {
    showAddEndpointModal,
    editEndpoint,
    saveEndpoint,
    deleteEndpoint,
    closeModal,
    handleTransformerChange,
    showEditPortModal,
    savePort,
    closePortModal,
    showWelcomeModal,
    closeWelcomeModal,
    showWelcomeModalIfFirstTime,
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

    // Initialize theme
    const theme = await window.go.main.App.GetTheme();
    applyTheme(theme);

    // Initialize UI
    initUI();

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

    // Refresh stats every 3 seconds
    setInterval(async () => {
        await loadStats(); // Refresh cumulative stats for endpoint cards
        const currentPeriod = getCurrentPeriod(); // Get current selected period
        await loadStatsByPeriod(currentPeriod); // Refresh period stats (daily/weekly/monthly)
        const config = await window.go.main.App.GetConfig();
        if (config) {
            renderEndpoints(JSON.parse(config).endpoints);
        }
    }, 3000);

    // Refresh logs every 2 seconds
    setInterval(loadLogs, 2000);

    // Show welcome modal on first launch
    showWelcomeModalIfFirstTime();

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
window.showEditPortModal = showEditPortModal;
window.savePort = savePort;
window.closePortModal = closePortModal;
window.showWelcomeModal = showWelcomeModal;
window.closeWelcomeModal = closeWelcomeModal;
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
window.cancelConfirm = cancelConfirm;
window.showCloseActionDialog = showCloseActionDialog;
window.quitApplication = quitApplication;
window.minimizeToTray = minimizeToTray;
window.showDataSyncDialog = showDataSyncDialog;
window.switchStatsPeriod = switchStatsPeriod;
window.toggleEndpointPanel = toggleEndpointPanel;
window.showSettingsModal = showSettingsModal;
window.closeSettingsModal = closeSettingsModal;
window.saveSettings = saveSettings;

// History modal functions
window.closeHistoryModal = async () => {
    const { closeHistoryModal } = await import('./modules/history.js');
    closeHistoryModal();
};
