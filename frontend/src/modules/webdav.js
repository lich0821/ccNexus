// WebDAV management
import { t } from '../i18n/index.js';

// Global variables to store WebDAV config
let currentWebDAVConfig = {
    url: '',
    username: '',
    password: ''
};

// Track if connection test passed
let connectionTestPassed = false;

// Show a generic modal
function showModal(title, content) {
    // Remove existing generic modal if any
    const existingModal = document.getElementById('genericModal');
    if (existingModal) {
        existingModal.remove();
    }

    // Create modal element
    const modal = document.createElement('div');
    modal.id = 'genericModal';
    modal.className = 'modal active';
    modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h2>${title}</h2>
                <button class="modal-close" onclick="window.closeDataSyncDialog()">&times;</button>
            </div>
            <div class="modal-body">
                ${content}
            </div>
        </div>
    `;

    document.body.appendChild(modal);

    // Do NOT close modal when clicking outside (like history modal)
}

// Show a sub-modal on top of existing modal
function showSubModal(title, content) {
    // Remove existing sub modal if any
    const existingModal = document.getElementById('subModal');
    if (existingModal) {
        existingModal.remove();
    }

    // Create modal element
    const modal = document.createElement('div');
    modal.id = 'subModal';
    modal.className = 'modal active';
    modal.style.zIndex = '1001';
    modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h2>${title}</h2>
                <button class="modal-close" onclick="window.closeSubModal()">&times;</button>
            </div>
            <div class="modal-body">
                ${content}
            </div>
        </div>
    `;

    document.body.appendChild(modal);

    // Do NOT close modal when clicking outside (like history modal)
}

// Show a confirm modal on top of sub modal
function showConfirmModal(title, content, allowClickOutsideClose = true) {
    const existingModal = document.getElementById('confirmModal');
    if (existingModal) {
        existingModal.remove();
    }

    const modal = document.createElement('div');
    modal.id = 'confirmModal';
    modal.className = 'modal active';
    modal.style.zIndex = '1002';
    modal.innerHTML = content;

    document.body.appendChild(modal);

    if (allowClickOutsideClose) {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                hideConfirmModal();
            }
        });
    }
}

// Hide the confirm modal
function hideConfirmModal() {
    const modal = document.getElementById('confirmModal');
    if (modal) {
        modal.classList.remove('active');
        setTimeout(() => modal.remove(), 300);
    }
}

// Hide the sub modal
function hideSubModal() {
    const modal = document.getElementById('subModal');
    if (modal) {
        modal.classList.remove('active');
        setTimeout(() => modal.remove(), 300);
    }
}

// Global function to close sub modal
window.closeSubModal = function() {
    hideSubModal();
};

// Hide the generic modal
function hideModal() {
    const modal = document.getElementById('genericModal');
    if (modal) {
        modal.classList.remove('active');
        setTimeout(() => modal.remove(), 300);
    }
}

// Load WebDAV config from backend
async function loadWebDAVConfig() {
    try {
        const configStr = await window.go.main.App.GetConfig();
        const config = JSON.parse(configStr);

        if (config.webdav) {
            currentWebDAVConfig = {
                url: config.webdav.url || '',
                username: config.webdav.username || '',
                password: config.webdav.password || ''
            };
        }
    } catch (error) {
        console.error('Failed to load WebDAV config:', error);
    }
}

// Show Data Sync Dialog (main entry point)
export async function showDataSyncDialog() {
    // Reset connection test flag when opening dialog
    connectionTestPassed = false;

    // Load current config
    await loadWebDAVConfig();

    const content = `
        <div class="data-sync-dialog">
            <div class="data-sync-section">
                <h3>🌐 ${t('webdav.serverConfig')}</h3>
                <div class="webdav-settings">
                    <div class="form-group">
                        <label><span class="required-mark">*</span>${t('webdav.serverUrl')}</label>
                        <input type="text" id="dataSyncUrl" class="form-input"
                               placeholder="https://dav.example.com/remote.php/dav/files/username/"
                               value="${currentWebDAVConfig.url}">
                        <small style="color: #888; font-size: 12px; margin-top: 5px;">${t('webdav.serverUrlHelp')}</small>
                    </div>
                    <div class="form-row" style="gap: 20px;">
                        <div class="form-group" style="flex: 1;">
                            <label><span class="required-mark">*</span>${t('webdav.username')}</label>
                            <input type="text" id="dataSyncUsername" class="form-input"
                                   placeholder="${t('webdav.usernamePlaceholder')}"
                                   value="${currentWebDAVConfig.username}">
                        </div>
                        <div class="form-group" style="flex: 1;">
                            <label><span class="required-mark">*</span>${t('webdav.password')}</label>
                            <input type="password" id="dataSyncPassword" class="form-input"
                                   placeholder="${t('webdav.passwordPlaceholder')}"
                                   value="${currentWebDAVConfig.password}">
                        </div>
                    </div>
                </div>
            </div>

            <div class="data-sync-section">
                <h3>🔧 ${t('webdav.operations')}</h3>
                <div class="data-sync-actions">
                    <button class="btn btn-secondary" onclick="window.testDataSyncConnection()">
                        🔍 ${t('webdav.testConnection')}
                    </button>
                    <button class="btn btn-secondary" onclick="window.saveDataSyncConfig()">
                        💾 ${t('webdav.saveConfig')}
                    </button>
                    <button class="btn btn-primary" onclick="window.backupToWebDAVFromDialog()">
                        📤 ${t('webdav.backup')}
                    </button>
                    <button class="btn btn-primary" onclick="window.openBackupManagerFromDialog()">
                        📂 ${t('webdav.backupManager')}
                    </button>
                </div>
            </div>
        </div>
    `;

    showModal(`🔄 ${t('webdav.dataSync')}`, content);
}

// Save WebDAV config from dialog
window.saveDataSyncConfig = async function() {
    const url = document.getElementById('dataSyncUrl')?.value.trim() || '';
    const username = document.getElementById('dataSyncUsername')?.value.trim() || '';
    const password = document.getElementById('dataSyncPassword')?.value.trim() || '';

    // Validate required fields
    if (!url) {
        showNotification(t('webdav.urlRequired'), 'error');
        return;
    }
    if (!username) {
        showNotification(t('webdav.usernameRequired'), 'error');
        return;
    }
    if (!password) {
        showNotification(t('webdav.passwordRequired'), 'error');
        return;
    }

    // Check if connection test passed
    if (!connectionTestPassed) {
        showNotification(t('webdav.testRequired'), 'error');
        return;
    }

    try {
        await updateWebDAVConfig(url, username, password);
        currentWebDAVConfig = { url, username, password };
        connectionTestPassed = false; // Reset after save
        showNotification(t('webdav.configSaved'), 'success');
    } catch (error) {
        showNotification(t('webdav.configSaveFailed') + ': ' + error, 'error');
    }
};

// Test connection from dialog
window.testDataSyncConnection = async function() {
    const url = document.getElementById('dataSyncUrl')?.value.trim() || '';
    const username = document.getElementById('dataSyncUsername')?.value.trim() || '';
    const password = document.getElementById('dataSyncPassword')?.value.trim() || '';

    // Validate required fields
    if (!url) {
        showNotification(t('webdav.urlRequired'), 'error');
        return;
    }
    if (!username) {
        showNotification(t('webdav.usernameRequired'), 'error');
        return;
    }
    if (!password) {
        showNotification(t('webdav.passwordRequired'), 'error');
        return;
    }

    try {
        // Test connection without saving
        const resultStr = await window.go.main.App.TestWebDAVConnection(url, username, password);
        const result = JSON.parse(resultStr);
        if (result.success) {
            connectionTestPassed = true;
            showNotification(t('webdav.connectionSuccess'), 'success');
        } else {
            connectionTestPassed = false;
            showNotification(t('webdav.connectionFailed'), 'error');
        }
    } catch (error) {
        connectionTestPassed = false;
        showNotification(t('webdav.connectionFailed') + ': ' + error, 'error');
    }
};

// Backup from dialog
window.backupToWebDAVFromDialog = async function() {
    await backupToWebDAV();
};

// Open backup manager from dialog
window.openBackupManagerFromDialog = async function() {
    await openBackupManager();
};

// Close dialog
window.closeDataSyncDialog = function() {
    hideModal();
};

// Update WebDAV configuration
export async function updateWebDAVConfig(url, username, password) {
    await window.go.main.App.UpdateWebDAVConfig(url, username, password);
}

// Test WebDAV connection (deprecated - use direct call with parameters)
export async function testWebDAVConnection(url, username, password) {
    const resultStr = await window.go.main.App.TestWebDAVConnection(url, username, password);
    return JSON.parse(resultStr);
}

// Generate default backup filename
function generateBackupFilename() {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    const hours = String(now.getHours()).padStart(2, '0');
    const minutes = String(now.getMinutes()).padStart(2, '0');
    const seconds = String(now.getSeconds()).padStart(2, '0');

    return `ccNexus-${year}${month}${day}${hours}${minutes}${seconds}.db`;
}

// Backup to WebDAV
export async function backupToWebDAV() {
    const filename = await promptFilename(t('webdav.enterBackupName'), generateBackupFilename());

    if (!filename) {
        return;
    }

    try {
        await window.go.main.App.BackupToWebDAV(filename);
        showNotification(t('webdav.backupSuccess'), 'success');
    } catch (error) {
        showNotification(t('webdav.backupFailed') + ': ' + error, 'error');
    }
}

// Restore from WebDAV
export async function restoreFromWebDAV(filename) {
    // Detect conflict first
    const conflictStr = await window.go.main.App.DetectWebDAVConflict(filename);
    const conflictResult = JSON.parse(conflictStr);

    if (!conflictResult.success) {
        showNotification(t('webdav.conflictDetectionFailed') + ': ' + conflictResult.message, 'error');
        return;
    }

    const conflicts = conflictResult.conflicts || [];

    // If there are conflicts, show conflict dialog
    if (conflicts.length > 0) {
        const choice = await showConflictDialog(conflicts);
        if (!choice) {
            return; // User cancelled
        }

        try {
            await window.go.main.App.RestoreFromWebDAV(filename, choice);
            if (choice === 'remote') {
                showNotification(t('webdav.restoreSuccess'), 'success');
                // Reload config
                window.location.reload();
            }
        } catch (error) {
            showNotification(t('webdav.restoreFailed') + ': ' + error, 'error');
        }
    } else {
        // No conflict, restore directly
        try {
            await window.go.main.App.RestoreFromWebDAV(filename, 'remote');
            showNotification(t('webdav.restoreSuccess'), 'success');
            // Reload config
            window.location.reload();
        } catch (error) {
            showNotification(t('webdav.restoreFailed') + ': ' + error, 'error');
        }
    }
}

// List WebDAV backups
export async function listWebDAVBackups() {
    const resultStr = await window.go.main.App.ListWebDAVBackups();
    return JSON.parse(resultStr);
}

// Delete WebDAV backups
export async function deleteWebDAVBackups(filenames) {
    try {
        await window.go.main.App.DeleteWebDAVBackups(filenames);
        showNotification(t('webdav.deleteSuccess'), 'success');
    } catch (error) {
        showNotification(t('webdav.deleteFailed') + ': ' + error, 'error');
    }
}

// Show backup manager
export async function openBackupManager() {
    const result = await listWebDAVBackups();

    if (!result.success) {
        showNotification(result.message, 'error');
        return;
    }

    const backups = result.backups || [];

    const content = `
        <div class="backup-manager">
            <div class="backup-manager-header">
                <div class="backup-manager-actions">
                    <button class="btn btn-secondary btn-sm" onclick="window.refreshBackupList()">🔄 ${t('webdav.refresh')}</button>
                    <button class="btn btn-danger btn-sm" onclick="window.deleteSelectedBackups()" ${backups.length === 0 ? 'disabled' : ''}>🗑️ ${t('webdav.deleteSelected')}</button>
                </div>
            </div>
            <div class="backup-list-container">
                ${backups.length === 0 ?
                    `<div class="empty-state">${t('webdav.noBackups')}</div>` :
                    renderBackupList(backups)
                }
            </div>
        </div>
    `;

    showSubModal('📂 ' + t('webdav.backupManager'), content);

    // Set up global functions for backup manager
    window.refreshBackupList = async () => {
        openBackupManager();
    };

    window.deleteSelectedBackups = async () => {
        const checkboxes = document.querySelectorAll('.backup-checkbox:checked');
        const selectedFiles = Array.from(checkboxes).map(cb => cb.dataset.filename);

        if (selectedFiles.length === 0) {
            showNotification(t('webdav.selectBackupsToDelete'), 'warning');
            return;
        }

        const confirmed = await confirmAction(
            t('webdav.confirmDelete').replace('{count}', selectedFiles.length)
        );

        if (!confirmed) {
            return;
        }

        await deleteWebDAVBackups(selectedFiles);
        openBackupManager();
    };

    window.restoreBackup = async (filename) => {
        const confirmed = await confirmAction(
            t('webdav.confirmRestore').replace('{filename}', filename)
        );

        if (!confirmed) {
            return;
        }

        hideSubModal();
        await restoreFromWebDAV(filename);
    };

    window.deleteSingleBackup = async (filename) => {
        const confirmed = await confirmAction(
            t('webdav.confirmDelete').replace('{count}', '1')
        );

        if (!confirmed) {
            return;
        }

        await deleteWebDAVBackups([filename]);
        openBackupManager();
    };

    window.closeBackupManager = () => {
        hideSubModal();
    };
}

// Render backup list
function renderBackupList(backups) {
    return `
        <table class="backup-table">
            <thead>
                <tr>
                    <th width="35"><input type="checkbox" id="selectAllBackups" onchange="window.toggleAllBackups(this)"></th>
                    <th>${t('webdav.filename')}</th>
                    <th width="110">${t('webdav.actions')}</th>
                </tr>
            </thead>
            <tbody>
                ${backups.map(backup => `
                    <tr>
                        <td><input type="checkbox" class="backup-checkbox" data-filename="${backup.filename}"></td>
                        <td>
                            <div style="font-weight: 500; margin-bottom: 4px; word-break: break-all;">${backup.filename}</div>
                            <div style="font-size: 11px; color: #888;">${formatFileSize(backup.size)}</div>
                            <div style="font-size: 11px; color: #888;">${formatDateTime(backup.modTime)}</div>
                        </td>
                        <td>
                            <div style="display: flex; flex-direction: column; gap: 4px;">
                                <button class="btn btn-primary btn-sm" onclick="window.restoreBackup('${backup.filename}')">↩️ ${t('webdav.restore')}</button>
                                <button class="btn btn-danger btn-sm" onclick="window.deleteSingleBackup('${backup.filename}')">🗑️ ${t('webdav.delete')}</button>
                            </div>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
}

// Toggle all backups
window.toggleAllBackups = function(checkbox) {
    const checkboxes = document.querySelectorAll('.backup-checkbox');
    checkboxes.forEach(cb => cb.checked = checkbox.checked);
};

// Format file size
function formatFileSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
}

// Format date time
function formatDateTime(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleString();
}

// Show conflict dialog
async function showConflictDialog(conflicts) {
    return new Promise((resolve) => {
        // Build conflict details HTML
        const conflictDetailsHTML = conflicts.map(conflict => {
            const fields = conflict.conflictFields || [];
            const fieldLabels = {
                'apiUrl': t('webdav.apiUrl'),
                'apiKey': t('webdav.apiKey'),
                'enabled': t('webdav.enabled'),
                'transformer': t('webdav.transformer'),
                'model': t('webdav.model'),
                'remark': t('webdav.remark')
            };

            return `
                <div class="conflict-endpoint">
                    <div class="conflict-endpoint-header">
                        <strong>📍 ${conflict.endpointName}</strong>
                        <span class="conflict-badge">${fields.length} ${fields.length === 1 ? t('webdav.conflict') : t('webdav.conflicts')}</span>
                    </div>
                    <div class="conflict-endpoint-body">
                        <div class="conflict-fields">
                            ${fields.map(field => `
                                <div class="conflict-field-item">
                                    <span class="conflict-field-name">${fieldLabels[field] || field}:</span>
                                    <div class="conflict-field-values">
                                        <div class="conflict-value-local">
                                            <span class="conflict-value-label">${t('webdav.local')}:</span>
                                            <code>${formatFieldValue(conflict.localEndpoint[field])}</code>
                                        </div>
                                        <div class="conflict-value-remote">
                                            <span class="conflict-value-label">${t('webdav.remote')}:</span>
                                            <code>${formatFieldValue(conflict.remoteEndpoint[field])}</code>
                                        </div>
                                    </div>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                </div>
            `;
        }).join('');

        const content = `
            <div class="conflict-dialog-content">
                <button class="conflict-close-btn" onclick="window.resolveConflict(null)">×</button>
                <div class="conflict-header">
                    <span class="conflict-icon">⚠️</span>
                    <span class="conflict-title">${t('webdav.conflictTitle')}</span>
                </div>
                <div class="conflict-divider"></div>
                <div class="conflict-body">
                    <p class="conflict-message">
                        ${t('webdav.conflictDetected')}
                        <strong>${conflicts.length}</strong> ${conflicts.length > 1 ? t('webdav.endpointsHave') : t('webdav.endpointHas')}
                    </p>
                    <div class="conflict-details-container">
                        ${conflictDetailsHTML}
                    </div>
                    <div class="conflict-strategy-info">
                        <p><strong>${t('webdav.useRemote')}:</strong> ${t('webdav.useRemoteDesc')}</p>
                        <p><strong>${t('webdav.keepLocal')}:</strong> ${t('webdav.keepLocalDesc')}</p>
                    </div>
                </div>
                <div class="conflict-footer">
                    <button class="btn btn-primary" onclick="window.resolveConflict('remote')">${t('webdav.useRemote')}</button>
                    <button class="btn btn-secondary" onclick="window.resolveConflict('keep_local')">${t('webdav.keepLocal')}</button>
                </div>
            </div>
        `;

        showConfirmModal('', content, false);

        window.resolveConflict = (choice) => {
            hideConfirmModal();
            delete window.resolveConflict;
            resolve(choice);
        };
    });
}

// Format field value for display
function formatFieldValue(value) {
    if (value === null || value === undefined || value === '') {
        return '<em>empty</em>';
    }
    if (typeof value === 'boolean') {
        return value ? '✓ Enabled' : '✗ Disabled';
    }
    // Handle numeric boolean (0/1) for enabled field
    if (typeof value === 'number' && (value === 0 || value === 1)) {
        return value === 1 ? '✓ Enabled' : '✗ Disabled';
    }
    if (typeof value === 'string' && value.length > 50) {
        return value.substring(0, 47) + '...';
    }
    return String(value);
}

// Prompt for filename
async function promptFilename(message, defaultValue) {
    return new Promise((resolve) => {
        const content = `
            <div class="prompt-dialog">
                <div class="prompt-body">
                    <input type="text" id="promptInput" class="form-input" value="${defaultValue || ''}" />
                </div>
                <div class="prompt-actions">
                    <button class="btn btn-primary" onclick="window.submitPrompt()">${t('common.ok')}</button>
                    <button class="btn btn-secondary" onclick="window.cancelPrompt()">${t('common.cancel')}</button>
                </div>
            </div>
        `;

        showSubModal('📝 ' + t('webdav.filename'), content);

        // Focus input
        setTimeout(() => {
            const input = document.getElementById('promptInput');
            if (input) {
                input.focus();
                input.select();
            }
        }, 100);

        window.submitPrompt = () => {
            const input = document.getElementById('promptInput');
            const value = input?.value.trim();
            hideSubModal();
            delete window.submitPrompt;
            delete window.cancelPrompt;
            resolve(value || null);
        };

        window.cancelPrompt = () => {
            hideSubModal();
            delete window.submitPrompt;
            delete window.cancelPrompt;
            resolve(null);
        };
    });
}

// Confirm action
async function confirmAction(message) {
    return new Promise((resolve) => {
        const content = `
            <div class="confirm-dialog-content">
                <div class="confirm-body">
                    <div class="confirm-icon">
                        <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                            <path d="M12 9v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                    </div>
                    <div class="confirm-content">
                        <h4 class="confirm-title">${t('common.confirm')}</h4>
                        <p class="confirm-message">${message}</p>
                    </div>
                </div>
                <div class="confirm-divider"></div>
                <div class="confirm-footer">
                    <button class="btn-confirm-delete" onclick="window.confirmYes()">${t('common.yes')}</button>
                    <button class="btn-confirm-cancel" onclick="window.confirmNo()">${t('common.no')}</button>
                </div>
            </div>
        `;

        showConfirmModal('', content);

        window.confirmYes = () => {
            hideConfirmModal();
            delete window.confirmYes;
            delete window.confirmNo;
            resolve(true);
        };

        window.confirmNo = () => {
            hideConfirmModal();
            delete window.confirmYes;
            delete window.confirmNo;
            resolve(false);
        };
    });
}

// Show notification
function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;

    // Add to body
    document.body.appendChild(notification);

    // Show notification
    setTimeout(() => notification.classList.add('show'), 10);

    // Hide and remove after 3 seconds
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

