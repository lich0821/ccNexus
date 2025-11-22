import { t } from '../i18n/index.js';
import { changeLanguage } from './ui.js';

// Show settings modal
export async function showSettingsModal() {
    const modal = document.getElementById('settingsModal');
    if (!modal) return;

    // Load current config
    await loadCurrentSettings();

    // Show modal
    modal.classList.add('active');
}

// Close settings modal
export function closeSettingsModal() {
    const modal = document.getElementById('settingsModal');
    if (modal) {
        modal.classList.remove('active');
    }
}

// Load current settings from backend
async function loadCurrentSettings() {
    try {
        const configStr = await window.go.main.App.GetConfig();
        const config = JSON.parse(configStr);

        // Set close window behavior
        const closeWindowBehavior = config.closeWindowBehavior || 'ask';
        const behaviorSelect = document.getElementById('settingsCloseWindowBehavior');
        if (behaviorSelect) {
            behaviorSelect.value = closeWindowBehavior;
        }

        // Set language
        const language = config.language || 'zh-CN';
        const languageSelect = document.getElementById('settingsLanguage');
        if (languageSelect) {
            languageSelect.value = language;
        }
    } catch (error) {
        console.error('Failed to load settings:', error);
    }
}

// Save settings
export async function saveSettings() {
    try {
        // Get values from form
        const closeWindowBehavior = document.getElementById('settingsCloseWindowBehavior').value;
        const language = document.getElementById('settingsLanguage').value;

        // Save close window behavior
        await window.go.main.App.SetCloseWindowBehavior(closeWindowBehavior);

        // Get current config
        const configStr = await window.go.main.App.GetConfig();
        const config = JSON.parse(configStr);

        // Update language if changed
        if (config.language !== language) {
            config.language = language;
            await window.go.main.App.UpdateConfig(JSON.stringify(config));

            // Apply language change immediately (will reload page)
            changeLanguage(language);
        }

        // Close modal
        closeSettingsModal();
    } catch (error) {
        console.error('Failed to save settings:', error);
        showNotification(t('settings.saveFailed') + ': ' + error, 'error');
    }
}

// Show notification (reuse from webdav.js if available, or implement simple version)
function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 15px 20px;
        background: ${type === 'success' ? '#10b981' : type === 'error' ? '#ef4444' : '#3b82f6'};
        color: white;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        z-index: 10000;
        animation: slideInRight 0.3s ease-out;
    `;

    document.body.appendChild(notification);

    // Auto remove after 3 seconds
    setTimeout(() => {
        notification.style.animation = 'slideOutRight 0.3s ease-out';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}
