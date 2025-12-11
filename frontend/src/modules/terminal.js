import { DetectTerminals, GetTerminalConfig, SaveTerminalConfig, AddProjectDir, RemoveProjectDir, LaunchTerminal, SelectDirectory } from '../../wailsjs/go/main/App';
import { t } from '../i18n/index.js';
import { showNotification } from './modal.js';

// 翻译后端错误消息
function translateError(error) {
    const errorStr = error.toString();
    const errorKey = `terminal.errors.${errorStr}`;
    const translated = t(errorKey);
    return translated !== errorKey ? translated : errorStr;
}

let terminals = [];
let terminalConfig = { selectedTerminal: 'cmd', projectDirs: [] };

export function initTerminal() {
    window.showTerminalModal = showTerminalModal;
    window.closeTerminalModal = closeTerminalModal;
    window.onTerminalChange = onTerminalChange;
    window.addProjectDir = addProjectDir;
    window.removeProjectDir = removeProjectDir;
    window.launchTerminal = launchTerminal;
}

async function showTerminalModal() {
    const modal = document.getElementById('terminalModal');
    modal.style.display = 'flex';

    // Load terminals and config
    await loadTerminals();
    await loadTerminalConfig();
    renderProjectDirs();
}

function closeTerminalModal() {
    document.getElementById('terminalModal').style.display = 'none';
}

async function loadTerminals() {
    try {
        const data = await DetectTerminals();
        terminals = JSON.parse(data);
        renderTerminalSelect();
    } catch (err) {
        console.error('Failed to detect terminals:', err);
    }
}

async function loadTerminalConfig() {
    try {
        const data = await GetTerminalConfig();
        terminalConfig = JSON.parse(data);
        // Update select value
        const select = document.getElementById('terminalSelect');
        if (select && terminalConfig.selectedTerminal) {
            select.value = terminalConfig.selectedTerminal;
        }
    } catch (err) {
        console.error('Failed to load terminal config:', err);
    }
}

function renderTerminalSelect() {
    const select = document.getElementById('terminalSelect');
    if (!select) return;

    select.innerHTML = terminals.map(term =>
        `<option value="${term.id}" ${term.id === terminalConfig.selectedTerminal ? 'selected' : ''}>${term.name}</option>`
    ).join('');
}

async function onTerminalChange() {
    const select = document.getElementById('terminalSelect');
    terminalConfig.selectedTerminal = select.value;
    try {
        await SaveTerminalConfig(terminalConfig.selectedTerminal, terminalConfig.projectDirs);
    } catch (err) {
        console.error('Failed to save terminal config:', err);
    }
}

function renderProjectDirs() {
    const container = document.getElementById('projectDirList');
    if (!container) return;

    if (!terminalConfig.projectDirs || terminalConfig.projectDirs.length === 0) {
        container.innerHTML = `<div class="empty-tip">${t('terminal.noDirs')}</div>`;
        return;
    }

    container.innerHTML = terminalConfig.projectDirs.map(dir => `
        <div class="project-dir-item">
            <span class="dir-path" title="${dir}">${dir}</span>
            <div class="dir-actions">
                <button class="btn btn-sm btn-primary" onclick="window.launchTerminal('${dir.replace(/\\/g, '\\\\')}')">▶ ${t('terminal.launch')}</button>
                <button class="btn btn-sm btn-danger" onclick="window.removeProjectDir('${dir.replace(/\\/g, '\\\\')}')">🗑️ ${t('terminal.delete')}</button>
            </div>
        </div>
    `).join('');
}

async function addProjectDir() {
    try {
        const dir = await SelectDirectory();
        if (!dir) return;

        await AddProjectDir(dir);
        terminalConfig.projectDirs.push(dir);
        renderProjectDirs();
    } catch (err) {
        console.error('Failed to add project dir:', err);
        showNotification(translateError(err), 'error');
    }
}

async function removeProjectDir(dir) {
    const confirmed = await showConfirmDialog(t('terminal.confirmDelete'));
    if (!confirmed) return;

    try {
        await RemoveProjectDir(dir);
        terminalConfig.projectDirs = terminalConfig.projectDirs.filter(d => d !== dir);
        renderProjectDirs();
    } catch (err) {
        console.error('Failed to remove project dir:', err);
    }
}

function showConfirmDialog(message) {
    return new Promise((resolve) => {
        const modal = document.createElement('div');
        modal.id = 'terminalConfirmModal';
        modal.className = 'modal active';
        modal.style.zIndex = '1002';
        modal.innerHTML = `
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
                    <button class="btn-confirm-delete" id="confirmYes">${t('common.yes')}</button>
                    <button class="btn-confirm-cancel" id="confirmNo">${t('common.no')}</button>
                </div>
            </div>
        `;
        document.body.appendChild(modal);

        modal.querySelector('#confirmYes').onclick = () => {
            modal.remove();
            resolve(true);
        };
        modal.querySelector('#confirmNo').onclick = () => {
            modal.remove();
            resolve(false);
        };
        modal.onclick = (e) => {
            if (e.target === modal) {
                modal.remove();
                resolve(false);
            }
        };
    });
}

async function launchTerminal(dir) {
    try {
        await LaunchTerminal(dir);
    } catch (err) {
        console.error('Failed to launch terminal:', err);
        showNotification(t('terminal.launchFailed') + ': ' + translateError(err), 'error');
    }
}
