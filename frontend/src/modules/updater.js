import { CheckForUpdates, GetUpdateSettings, SetUpdateSettings, SkipVersion, DownloadUpdate, GetDownloadProgress, CancelDownload, InstallUpdate, ApplyUpdate, SendUpdateNotification } from '../../wailsjs/go/main/App';
import { t } from '../i18n/index.js';
import { showNotification } from './modal.js';

let downloadInterval = null;
let updateCheckInterval = null;

// 显示更新红点
function showUpdateBadge() {
    document.getElementById('updateBadge')?.classList.add('show');
    document.getElementById('checkUpdateBadge')?.classList.add('show');
}

// 隐藏关于按钮红点
export function hideAboutBadge() {
    document.getElementById('updateBadge')?.classList.remove('show');
}

// 隐藏检查更新按钮红点
function hideCheckUpdateBadge() {
    document.getElementById('checkUpdateBadge')?.classList.remove('show');
}

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
    hideCheckUpdateBadge();
    try {
        const resultStr = await CheckForUpdates();
        const result = JSON.parse(resultStr);

        if (!result.success) {
            if (!silent) {
                showNotification(t('update.checkFailed') + ': ' + result.error, 'error');
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

            // 显示红点提示
            showUpdateBadge();
            showUpdateNotification(info);
        } else {
            if (!silent) {
                showNotification(t('update.upToDate'), 'success');
            }
        }
    } catch (error) {
        console.error('[Updater] Failed to check for updates:', error);
        if (!silent) {
            showNotification(t('update.checkFailed') + ': ' + error.message, 'error');
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
        <div class="modal-content update-modal-content">
            <div class="modal-header">
                <h2>🎉 ${t('update.newVersionAvailable')}</h2>
                <button class="modal-close">&times;</button>
            </div>
            <div class="modal-body">
                <div class="update-version-info">
                    <div class="update-version-item">
                        <span class="update-version-label">${t('update.currentVersion')}</span>
                        <span class="update-version-value">v${info.currentVersion}</span>
                    </div>
                    <div class="update-version-arrow">→</div>
                    <div class="update-version-item">
                        <span class="update-version-label">${t('update.latestVersion')}</span>
                        <span class="update-version-value update-version-new">${info.latestVersion}</span>
                    </div>
                </div>
                <div class="update-release-date">
                    📅 ${t('update.releaseDate')}: ${info.releaseDate}
                </div>
                <div class="update-changelog">
                    <div class="update-changelog-title">${t('update.changelog')}</div>
                    <div class="update-changelog-content">${formatChangelog(info.changelog)}</div>
                </div>
                <div id="download-progress-container" style="display: none;">
                    <div class="update-progress-bar">
                        <div id="progress-fill" class="update-progress-fill"></div>
                    </div>
                    <div class="update-progress-row">
                        <p id="progress-text" class="update-progress-text">${t('update.downloading')} 0%</p>
                        <button id="btn-cancel-download" class="btn btn-secondary btn-small">${t('common.cancel')}</button>
                    </div>
                </div>
            </div>
            <div class="modal-footer">
                <button id="btn-skip-version" class="btn btn-secondary">${t('update.skipVersion')}</button>
                <button id="btn-remind-later" class="btn btn-secondary">${t('update.remindLater')}</button>
                <button id="btn-download-update" class="btn btn-primary">${t('update.downloadUpdate')}</button>
            </div>
        </div>
    `;

    document.body.appendChild(modal);

    // Attach event listeners
    modal.querySelector('.modal-close').addEventListener('click', async () => {
        if (downloadInterval) {
            clearInterval(downloadInterval);
            downloadInterval = null;
        }
        await CancelDownload();
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
    if (!markdown || markdown.trim() === '') {
        return `<p style="color: #999;">${t('update.noChangelog')}</p>`;
    }
    // Simple markdown to HTML conversion
    return markdown
        .replace(/\*\*Full Changelog\*\*:\s*/gm, '')
        .replace(/^### (.+)$/gm, '<h5>$1</h5>')
        .replace(/^## (.+)$/gm, '<h4>$1</h4>')
        .replace(/^# (.+)$/gm, '<h3>$1</h3>')
        .replace(/^\* (.+)$/gm, '<li>$1</li>')
        .replace(/^- (.+)$/gm, '<li>$1</li>')
        .replace(/(https?:\/\/[^\s<]+)/g, '<a href="$1" target="_blank" style="color: #667eea;">$1</a>')
        .replace(/\n\n/g, '</p><p>')
        .replace(/^(.+)$/gm, '<p>$1</p>');
}

// Start download
async function startDownload(info) {
    // Clear any existing interval
    if (downloadInterval) {
        clearInterval(downloadInterval);
        downloadInterval = null;
    }

    const downloadBtn = document.getElementById('btn-download-update');
    const skipBtn = document.getElementById('btn-skip-version');
    const remindBtn = document.getElementById('btn-remind-later');
    const progressContainer = document.getElementById('download-progress-container');

    // Hide buttons and show progress
    if (downloadBtn) downloadBtn.style.display = 'none';
    if (skipBtn) skipBtn.style.display = 'none';
    if (remindBtn) remindBtn.style.display = 'none';
    progressContainer.style.display = 'block';

    // Add cancel button listener
    const cancelBtn = document.getElementById('btn-cancel-download');
    if (cancelBtn) {
        cancelBtn.addEventListener('click', async () => {
            await CancelDownload();
            if (downloadInterval) {
                clearInterval(downloadInterval);
            }
            const modal = document.getElementById('updateModal');
            if (modal) modal.remove();
        });
    }

    try {
        // Extract filename from URL
        const url = new URL(info.downloadUrl);
        const filename = url.pathname.split('/').pop();

        // Start download
        await DownloadUpdate(info.downloadUrl, filename);

        // Wait a bit before starting to poll
        await new Promise(resolve => setTimeout(resolve, 100));

        // Poll download progress
        let downloadStarted = false;
        downloadInterval = setInterval(async () => {
            const progressStr = await GetDownloadProgress();
            const progress = JSON.parse(progressStr);

            if (progress.status === 'downloading') {
                downloadStarted = true;
                updateProgressBar(progress);
            } else if (progress.status === 'completed') {
                clearInterval(downloadInterval);
                showInstallButton(progress.filePath);
            } else if (downloadStarted && (progress.status === 'failed' || progress.status === 'cancelled')) {
                clearInterval(downloadInterval);
                if (progress.status === 'failed') {
                    showError(progress.error, info);
                }
            }
        }, 200);
    } catch (error) {
        console.error('Failed to start download:', error);
        showError(error.message, info);
    }
}

// Update progress bar
function updateProgressBar(progress) {
    const progressFill = document.getElementById('progress-fill');
    const progressText = document.getElementById('progress-text');

    if (progressFill && progressText) {
        progressFill.style.width = progress.progress + '%';
        progressText.textContent = t('update.downloading') + ' ' + Math.round(progress.progress) + '%';
    }
}

// Show install button
function showInstallButton(filePath) {
    const progressContainer = document.getElementById('download-progress-container');
    progressContainer.innerHTML = `<p class="success-message center">🎉 ${t('update.downloadComplete')}</p>`;

    const isWindows = navigator.platform.indexOf('Win') > -1;
    const btnText = isWindows ? t('update.applyUpdate') : t('update.installUpdate');
    const modalFooter = document.querySelector('#updateModal .modal-footer');
    modalFooter.innerHTML = `<button id="btn-install-update" class="btn btn-primary">${btnText}</button>`;

    document.getElementById('btn-install-update').addEventListener('click', async () => {
        const btn = document.getElementById('btn-install-update');
        btn.disabled = true;
        btn.textContent = t('update.applying');

        try {
            const resultStr = await InstallUpdate(filePath);
            const result = JSON.parse(resultStr);

            if (result.success) {
                if (result.exePath) {
                    // Windows: 直接应用更新
                    const applyResult = JSON.parse(await ApplyUpdate(result.exePath));
                    if (applyResult.success) {
                        const modal = document.getElementById('updateModal');
                        if (modal) modal.remove();
                    } else {
                        showNotification(t('update.applyFailed') + ': ' + applyResult.error, 'error');
                        btn.disabled = false;
                        btn.textContent = btnText;
                    }
                } else {
                    // 其他平台: 显示手动安装说明
                    showInstallInstructions(result);
                }
            } else {
                showNotification(t('update.installFailed') + ': ' + result.error, 'error');
                btn.disabled = false;
                btn.textContent = btnText;
            }
        } catch (error) {
            showNotification(t('update.installFailed') + ': ' + error.message, 'error');
            btn.disabled = false;
            btn.textContent = btnText;
        }
    });
}

// Show installation instructions (非Windows平台)
function showInstallInstructions(result) {
    const progressContainer = document.getElementById('download-progress-container');
    const modalFooter = document.querySelector('#updateModal .modal-footer');
    const instructions = t('update.' + result.message);
    progressContainer.innerHTML = `
        <div class="install-instructions">
            <p class="success-message">${t('update.extractComplete')}</p>
            <p class="install-path">${t('update.extractPath')}: ${result.path}</p>
            <div class="instructions-text">${instructions}</div>
        </div>
    `;
    modalFooter.innerHTML = '';
}

// Show error
function showError(errorMsg, info) {
    const progressContainer = document.getElementById('download-progress-container');
    progressContainer.innerHTML = `
        <div class="error-container" style="background: var(--bg-tertiary);">
            <p class="error-message" style="color: #dc3545;">${t('update.downloadFailed')}</p>
            <p class="error-detail" style="color: var(--text-secondary);">${errorMsg}</p>
        </div>
    `;

    const modalFooter = document.querySelector('#updateModal .modal-footer');
    modalFooter.innerHTML = `<button id="btn-retry-download" class="btn btn-primary">${t('common.retry')}</button>`;

    document.getElementById('btn-retry-download').addEventListener('click', () => {
        if (info) {
            progressContainer.innerHTML = `
                <div class="update-progress-bar">
                    <div id="progress-fill" class="update-progress-fill"></div>
                </div>
                <div class="update-progress-row">
                    <p id="progress-text" class="update-progress-text">${t('update.downloading')} 0%</p>
                    <button id="btn-cancel-download" class="btn btn-secondary btn-small">${t('common.cancel')}</button>
                </div>
            `;
            modalFooter.innerHTML = '';
            startDownload(info);
        }
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
