import { t } from '../i18n/index.js';
import { changeLanguage } from './ui.js';

// Auto theme check interval ID
let autoThemeIntervalId = null;

// Apply theme to body element
export function applyTheme(theme) {
    if (theme === 'dark') {
        document.body.classList.add('dark-theme');
    } else {
        document.body.classList.remove('dark-theme');
    }
}

// Get theme based on current time (7:00-19:00 = light, otherwise = dark)
function getTimeBasedTheme() {
    const hour = new Date().getHours();
    return (hour >= 7 && hour < 19) ? 'light' : 'dark';
}

// Check and apply auto theme
export function checkAndApplyAutoTheme() {
    const theme = getTimeBasedTheme();
    applyTheme(theme);
}

// Start auto theme checking (check every minute)
export function startAutoThemeCheck() {
    // Apply immediately
    checkAndApplyAutoTheme();

    // Clear existing interval if any
    if (autoThemeIntervalId) {
        clearInterval(autoThemeIntervalId);
    }

    // Check every minute
    autoThemeIntervalId = setInterval(checkAndApplyAutoTheme, 60000);
}

// Stop auto theme checking
export function stopAutoThemeCheck() {
    if (autoThemeIntervalId) {
        clearInterval(autoThemeIntervalId);
        autoThemeIntervalId = null;
    }
}

// Initialize theme based on settings
export async function initTheme() {
    try {
        const themeAuto = await window.go.main.App.GetThemeAuto();
        if (themeAuto) {
            startAutoThemeCheck();
        } else {
            const theme = await window.go.main.App.GetTheme();
            applyTheme(theme);
        }
    } catch (error) {
        console.error('Failed to init theme:', error);
        const theme = await window.go.main.App.GetTheme();
        applyTheme(theme);
    }
}

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

        // Set theme
        const theme = config.theme || 'light';
        const themeSelect = document.getElementById('settingsTheme');
        if (themeSelect) {
            themeSelect.value = theme;
        }

        // Set theme auto
        const themeAuto = config.themeAuto || false;
        const themeAutoCheckbox = document.getElementById('settingsThemeAuto');
        if (themeAutoCheckbox) {
            themeAutoCheckbox.checked = themeAuto;
            // Disable theme select when auto mode is enabled
            if (themeSelect) {
                themeSelect.disabled = themeAuto;
            }
        }

        // Add event listener for auto checkbox
        if (themeAutoCheckbox) {
            themeAutoCheckbox.onchange = function() {
                if (themeSelect) {
                    themeSelect.disabled = this.checked;
                }
            };
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
        const theme = document.getElementById('settingsTheme').value;
        const themeAuto = document.getElementById('settingsThemeAuto').checked;

        // Save close window behavior
        await window.go.main.App.SetCloseWindowBehavior(closeWindowBehavior);

        // Get current config
        const configStr = await window.go.main.App.GetConfig();
        const config = JSON.parse(configStr);

        // Update theme auto setting
        if (config.themeAuto !== themeAuto) {
            await window.go.main.App.SetThemeAuto(themeAuto);

            if (themeAuto) {
                // Enable auto mode - start checking and apply time-based theme
                startAutoThemeCheck();
            } else {
                // Disable auto mode - stop checking and apply selected theme
                stopAutoThemeCheck();
                await window.go.main.App.SetTheme(theme);
                applyTheme(theme);
            }
        } else if (!themeAuto && config.theme !== theme) {
            // Auto mode not changed, but theme changed (only when auto is off)
            await window.go.main.App.SetTheme(theme);
            applyTheme(theme);
        }

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
