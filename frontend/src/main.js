import './style.css'
import '../wailsjs/runtime/runtime.js'
import { setLanguage } from './i18n/index.js'
import { initUI, changeLanguage } from './modules/ui.js'
import { loadConfig } from './modules/config.js'
import { loadStats } from './modules/stats.js'
import { renderEndpoints } from './modules/endpoints.js'
import { loadLogs, toggleLogPanel, changeLogLevel, copyLogs, clearLogs } from './modules/logs.js'
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
    openArticle
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

    // Initialize UI
    initUI();

    // Load initial data
    await loadConfigAndRender();
    loadStats();
    loadLogs();

    // Refresh stats every 5 seconds
    setInterval(async () => {
        await loadStats();
        const config = await window.go.main.App.GetConfig();
        if (config) {
            renderEndpoints(JSON.parse(config).endpoints);
        }
    }, 5000);

    // Refresh logs every 2 seconds
    setInterval(loadLogs, 2000);

    // Show welcome modal on first launch
    showWelcomeModalIfFirstTime();

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
