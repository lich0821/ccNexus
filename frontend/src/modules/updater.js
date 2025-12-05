import { CheckForUpdates, GetUpdateSettings, SetUpdateSettings, SkipVersion, DownloadUpdate, GetDownloadProgress, InstallUpdate, SendUpdateNotification } from '../../wailsjs/go/main/App';
import { t } from '../i18n/index.js';

let downloadInterval = null;
let updateCheckInterval = null;

// Check for updates on startup
export async function checkUpdatesOnStartup() {
    try {
        const settingsStr = await GetUpdateSettings();
        const settings = JSON.parse(settingsStr);

        if (settings.checkInterval === 0) {
            stopAutoCheck();
            return;
        }

        if (settings.lastCheckTime) {
            const lastCheck = new Date(settings.lastCheckTime);
            const now = new Date();
            const hoursSinceCheck = (now - lastCheck) / (1000 * 60 * 60);

            if (hoursSinceCheck < settings.checkInterval) {
                startAutoCheck(settings.checkInterval);
                return;
            }
        }

        await checkForUpdates(true);
        startAutoCheck(settings.checkInterval);
    } catch (error) {
        console.error('[Updater] Failed to check updates on startup:', error);
    }
}

// Start automatic update checking
function startAutoCheck(intervalHours) {
    stopAutoCheck();
    if (intervalHours > 0) {
        updateCheckInterval = setInterval(() => {
            checkForUpdates(true);
        }, intervalHours * 60 * 60 * 1000);
    }
}

// Stop automatic update checking
function stopAutoCheck() {
    if (updateCheckInterval) {
        clearInterval(updateCheckInterval);
        updateCheckInterval = null;
    }
}

// Check for updates manually
export async function checkForUpdates(silent = false) {
    try {
        const resultStr = await CheckForUpdates();
        const result = JSON.parse(resultStr);

        if (!result.success) {
            if (!silent) {
                alert(t('update.checkFailed') + ': ' + result.error);
            }
            return;
        }

        const info = result.info;

        if (info.hasUpdate) {
            const settingsStr = await GetUpdateSettings();
            const settings = JSON.parse(settingsStr);

            if (settings.skippedVersion === info.latestVersion) {
                return;
            }

            showUpdateNotification(info);
        } else {
            if (!silent) {
                alert(t('update.upToDate'));
            }
        }
    } catch (error) {
        console.error('[Updater] Failed to check for updates:', error);
        if (!silent) {
            alert(t('update.checkFailed') + ': ' + error.message);
        }
    }
}

// Show update notification
function showUpdateNotification(info) {
    if (document.getElementById('updateModal')) {
        return;
    }

    const title = 'ccNexus ' + t('update.newVersionAvailable');
    const message = t('update.latestVersion') + ': ' + info.latestVersion;
    SendUpdateNotification(title, message).catch(err => console.error('Failed to send notification:', err));

    const modal = document.createElement('div');
    modal.id = 'updateModal';
    modal.className = 'modal active';
    modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h2>${t('update.newVersionAvailable')}</h2>
                <button class="modal-close">&times;</button>
            </div>
            <div class="modal-body">
                <div class="version-comparison">
                    <p><strong>${t('update.currentVersion')}:</strong> ${info.currentVersion}</p>
                    <p><strong>${t('update.latestVersion')}:</strong> ${info.latestVersion}</p>
                    <p><strong>${t('update.releaseDate')}:</strong> ${info.releaseDate}</p>
                </div>
                <div class="changelog">
                    <h4>${t('update.changelog')}:</h4>
                    <div class="changelog-content">${formatChangelog(info.changelog)}</div>
                </div>
                <div id="download-progress-container" class="hidden">
                    <div class="progress-bar">
                        <div id="progress-fill" class="progress-fill" style="width: 0%; height: 20px; background: #4CAF50; transition: width 0.3s;"></div>
                    </div>
                    <p id="progress-text">0%</p>
                </div>
            </div>
            <div class="modal-footer">
                <button id="btn-download-update" class="btn btn-primary">${t('update.downloadUpdate')}</button>
                <button id="btn-skip-version" class="btn btn-secondary">${t('update.skipVersion')}</button>
                <button id="btn-remind-later" class="btn btn-secondary">${t('update.remindLater')}</button>
            </div>
        </div>
    `;

    document.body.appendChild(modal);

    // Attach event listeners
    modal.querySelector('.modal-close').addEventListener('click', () => {
        modal.remove();
    });

    document.getElementById('btn-download-update').addEventListener('click', () => {
        startDownload(info);
    });

    document.getElementById('btn-skip-version').addEventListener('click', async () => {
        await SkipVersion(info.latestVersion);
        modal.remove();
    });

    document.getElementById('btn-remind-later').addEventListener('click', () => {
        modal.remove();
    });
}

// Format changelog from markdown
function formatChangelog(markdown) {
    // Simple markdown to HTML conversion
    return markdown
        .replace(/^### (.+)$/gm, '<h5>$1</h5>')
        .replace(/^## (.+)$/gm, '<h4>$1</h4>')
        .replace(/^# (.+)$/gm, '<h3>$1</h3>')
        .replace(/^\* (.+)$/gm, '<li>$1</li>')
        .replace(/^- (.+)$/gm, '<li>$1</li>')
        .replace(/\n\n/g, '</p><p>')
        .replace(/^(.+)$/gm, '<p>$1</p>');
}

// Start download
async function startDownload(info) {
    const downloadBtn = document.getElementById('btn-download-update');
    const skipBtn = document.getElementById('btn-skip-version');
    const remindBtn = document.getElementById('btn-remind-later');
    const progressContainer = document.getElementById('download-progress-container');

    // Hide buttons and show progress
    downloadBtn.style.display = 'none';
    skipBtn.style.display = 'none';
    remindBtn.style.display = 'none';
    progressContainer.classList.remove('hidden');

    try {
        // Extract filename from URL
        const url = new URL(info.downloadUrl);
        const filename = url.pathname.split('/').pop();

        // Start download
        await DownloadUpdate(info.downloadUrl, filename);

        // Wait a bit before starting to poll
        await new Promise(resolve => setTimeout(resolve, 100));

        // Poll download progress
        downloadInterval = setInterval(async () => {
            const progressStr = await GetDownloadProgress();
            const progress = JSON.parse(progressStr);

            updateProgressBar(progress);

            if (progress.status === 'completed') {
                clearInterval(downloadInterval);
                showInstallButton(progress.filePath);
            } else if (progress.status === 'failed') {
                clearInterval(downloadInterval);
                showError(progress.error);
            }
        }, 200);
    } catch (error) {
        console.error('Failed to start download:', error);
        showError(error.message);
    }
}

// Update progress bar
function updateProgressBar(progress) {
    const progressFill = document.getElementById('progress-fill');
    const progressText = document.getElementById('progress-text');

    if (progressFill && progressText) {
        progressFill.style.width = progress.progress + '%';
        progressText.textContent = Math.round(progress.progress) + '%';
    }
}

// Show install button
function showInstallButton(filePath) {
    const progressContainer = document.getElementById('download-progress-container');
    progressContainer.innerHTML = `
        <p class="success-message">${t('update.downloadComplete')}</p>
        <button id="btn-install-update" class="btn-primary">${t('update.installUpdate')}</button>
    `;

    document.getElementById('btn-install-update').addEventListener('click', async () => {
        try {
            const resultStr = await InstallUpdate(filePath);
            const result = JSON.parse(resultStr);

            if (result.success) {
                showInstallInstructions(result);
            } else {
                alert(t('update.installFailed') + ': ' + result.error);
            }
        } catch (error) {
            alert(t('update.installFailed') + ': ' + error.message);
        }
    });
}

// Show installation instructions
function showInstallInstructions(result) {
    const progressContainer = document.getElementById('download-progress-container');
    const instructions = t('update.' + result.message);

    progressContainer.innerHTML = `
        <div class="install-instructions">
            <p class="success-message">${t('update.extractComplete')}</p>
            <p class="install-path">${t('update.extractPath')}: ${result.path}</p>
            <div class="instructions-text">${instructions}</div>
            <button id="btn-close-modal" class="btn btn-secondary">${t('common.close')}</button>
        </div>
    `;

    document.getElementById('btn-close-modal').addEventListener('click', () => {
        const modal = document.getElementById('updateModal');
        if (modal) modal.remove();
    });
}

// Show error
function showError(errorMsg) {
    const progressContainer = document.getElementById('download-progress-container');
    progressContainer.innerHTML = `
        <p class="error-message" style="color: red;">${t('update.downloadFailed')}: ${errorMsg}</p>
        <button id="btn-close-error" class="btn btn-secondary">${t('common.close')}</button>
    `;

    document.getElementById('btn-close-error').addEventListener('click', () => {
        const modal = document.getElementById('updateModal');
        if (modal) modal.remove();
    });
}

// Initialize update settings UI
export function initUpdateSettings() {
    const checkIntervalSelect = document.getElementById('check-interval');

    // Load current settings
    loadUpdateSettings();

    // Save settings on change
    if (checkIntervalSelect) {
        checkIntervalSelect.addEventListener('change', saveUpdateSettings);
    }
}

// Load update settings
async function loadUpdateSettings() {
    try {
        const settingsStr = await GetUpdateSettings();
        const settings = JSON.parse(settingsStr);

        const checkIntervalSelect = document.getElementById('check-interval');

        if (checkIntervalSelect) {
            checkIntervalSelect.value = settings.checkInterval.toString();
        }
    } catch (error) {
        console.error('Failed to load update settings:', error);
    }
}

// Save update settings
async function saveUpdateSettings() {
    try {
        const checkIntervalSelect = document.getElementById('check-interval');
        const checkInterval = checkIntervalSelect ? parseInt(checkIntervalSelect.value) : 24;
        const autoCheck = checkInterval > 0;

        await SetUpdateSettings(autoCheck, checkInterval);

        if (checkInterval > 0) {
            startAutoCheck(checkInterval);
        } else {
            stopAutoCheck();
        }
    } catch (error) {
        console.error('Failed to save update settings:', error);
    }
}
