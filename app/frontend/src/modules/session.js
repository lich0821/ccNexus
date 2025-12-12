import { GetSessions, DeleteSession, RenameSession, GetSessionData } from '../../wailsjs/go/main/App';
import { t } from '../i18n/index.js';
import { showNotification } from './modal.js';

let currentProjectDir = '';
let sessions = [];
let selectedSessions = {}; // 按目录存储选中的会话

export function initSession() {
    window.showSessionModal = showSessionModal;
    window.closeSessionModal = closeSessionModal;
    window.selectSession = selectSession;
    window.confirmSessionSelection = confirmSessionSelection;
    window.deleteSession = deleteSession;
    window.renameSession = renameSession;
    window.viewSessionDetail = viewSessionDetail;
    window.closeSessionDetailModal = closeSessionDetailModal;
}

// 获取选中的会话
export function getSelectedSession(dir) {
    return selectedSessions[dir] || null;
}

// 清除选中的会话
export function clearSelectedSession(dir) {
    if (dir) {
        delete selectedSessions[dir];
    } else if (currentProjectDir) {
        delete selectedSessions[currentProjectDir];
    }
}

export async function showSessionModal(projectDir) {
    currentProjectDir = projectDir;
    const modal = document.getElementById('sessionModal');
    modal.style.display = 'flex';
    await loadSessions();
}

export function closeSessionModal() {
    document.getElementById('sessionModal').style.display = 'none';
    // 不清空 currentProjectDir，保留选中状态
    sessions = [];
}

async function loadSessions() {
    const listContainer = document.getElementById('sessionList');
    listContainer.innerHTML = `<div class="session-loading">${t('session.loading')}</div>`;

    try {
        const result = JSON.parse(await GetSessions(currentProjectDir));
        if (!result.success) {
            listContainer.innerHTML = `<div class="session-empty">${t('session.loadError')}</div>`;
            return;
        }

        sessions = result.sessions || [];
        renderSessionList();
    } catch (err) {
        console.error('Failed to load sessions:', err);
        listContainer.innerHTML = `<div class="session-empty">${t('session.loadError')}</div>`;
    }
}

function renderSessionList() {
    const listContainer = document.getElementById('sessionList');

    if (sessions.length === 0) {
        listContainer.innerHTML = `<div class="session-empty">${t('session.noSessions')}</div>`;
        return;
    }

    const currentSelected = selectedSessions[currentProjectDir];

    listContainer.innerHTML = sessions.map((s, index) => {
        const serialNumber = index + 1; // 序号从1开始
        const time = formatTime(s.modTime);
        const fullTime = formatFullTime(s.modTime);
        const size = formatSize(s.size);
        const summary = s.summary || t('session.noSummary');
        const displaySummary = s.alias || summary;
        const tooltipTitle = s.alias
            ? `${s.alias}\n${t('session.modTime')}: ${fullTime}\n${t('session.size')}: ${size}\n${t('session.summary')}: ${summary}`
            : `${t('session.session')} ${serialNumber}\n${t('session.modTime')}: ${fullTime}\n${t('session.size')}: ${size}\n${t('session.summary')}: ${summary}`;

        return `
        <div class="session-item ${currentSelected && currentSelected.sessionId === s.sessionId ? 'selected' : ''}"
             data-index="${index}"
             data-session-id="${s.sessionId}"
             title="${tooltipTitle}">
            <div class="session-info">
                <span class="session-serial">${serialNumber}</span>
                <span class="session-summary" title="${displaySummary}">${displaySummary.length > 50 ? displaySummary.substring(0, 47) + '...' : displaySummary}</span>
                <span class="session-time">${time}</span>
                <span class="session-size">${size}</span>
            </div>
            <div class="session-actions">
                <button class="session-btn session-btn-view" title="${t('session.view')}">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                        <circle cx="12" cy="12" r="3"/>
                    </svg>
                </button>
                <button class="session-btn session-btn-rename" title="${t('session.rename')}">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg>
                </button>
                <button class="session-btn session-btn-delete" title="${t('session.delete')}">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"/>
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                    </svg>
                </button>
            </div>
        </div>
    `}).join('');

    // 绑定事件
    listContainer.querySelectorAll('.session-item').forEach(item => {
        const index = parseInt(item.dataset.index);
        const session = sessions[index];

        // 点击会话信息区域选择会话
        item.querySelector('.session-info').onclick = () => window.selectSession(session.sessionId);

        // 查看按钮
        item.querySelector('.session-btn-view').onclick = (e) => {
            e.stopPropagation();
            window.viewSessionDetail(session.sessionId);
        };

        // 重命名按钮
        item.querySelector('.session-btn-rename').onclick = (e) => {
            e.stopPropagation();
            window.renameSession(session.sessionId);
        };

        // 删除按钮
        item.querySelector('.session-btn-delete').onclick = (e) => {
            e.stopPropagation();
            window.deleteSession(session.sessionId);
        };
    });
}

function formatTime(timestamp) {
    const date = new Date(timestamp * 1000);
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${month}/${day} ${hours}:${minutes}`;
}

function formatFullTime(timestamp) {
    const date = new Date(timestamp * 1000);
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${year}-${month}-${day} ${hours}:${minutes}`;
}

function formatSize(bytes) {
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

// 选择会话
function selectSession(sessionId) {
    const session = sessions.find(s => s.sessionId === sessionId);
    if (!session) return;

    const sessionIndex = sessions.findIndex(s => s.sessionId === sessionId);
    const serialNumber = sessionIndex + 1;

    // 按目录存储选中的会话
    selectedSessions[currentProjectDir] = {
        sessionId: sessionId,
        info: {
            alias: session.alias,
            summary: session.summary,
            serialNumber: serialNumber
        }
    };

    // 仅更新UI显示选中状态，不关闭窗口，不弹toast
    renderSessionList();
}

// 确认选择会话
function confirmSessionSelection() {
    // 关闭会话模态框，返回启动器
    closeSessionModal();
    // 触发启动器界面更新（通过自定义事件）
    window.dispatchEvent(new CustomEvent('sessionSelected'));
}

async function deleteSession(sessionId) {
    const confirmed = await showConfirmDialog(t('session.confirmDelete'));
    if (!confirmed) return;

    try {
        await DeleteSession(currentProjectDir, sessionId);
        showNotification(t('session.deleted'), 'success');
        await loadSessions();
    } catch (err) {
        console.error('Failed to delete session:', err);
        showNotification(t('session.deleteFailed'), 'error');
    }
}

async function renameSession(sessionId) {
    const session = sessions.find(s => s.sessionId === sessionId);
    const currentName = session?.alias || '';

    const newName = await showPromptDialog(t('session.renamePrompt'), currentName);
    if (newName === null) return;

    try {
        await RenameSession(currentProjectDir, sessionId, newName);
        showNotification(t('session.renamed'), 'success');
        await loadSessions();
    } catch (err) {
        console.error('Failed to rename session:', err);
        showNotification(t('session.renameFailed'), 'error');
    }
}

function showConfirmDialog(message) {
    return new Promise(resolve => {
        const modal = document.createElement('div');
        modal.id = 'sessionConfirmModal';
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
                        <h4 class="confirm-title">${t('common.confirmDeleteTitle')}</h4>
                        <p class="confirm-message">${message}</p>
                    </div>
                </div>
                <div class="confirm-divider"></div>
                <div class="confirm-footer">
                    <button class="btn-confirm-delete" id="confirmYes">${t('common.delete')}</button>
                    <button class="btn-confirm-cancel" id="confirmNo">${t('common.cancel')}</button>
                </div>
            </div>
        `;
        document.body.appendChild(modal);

        modal.querySelector('#confirmYes').onclick = () => { modal.remove(); resolve(true); };
        modal.querySelector('#confirmNo').onclick = () => { modal.remove(); resolve(false); };
        modal.onclick = (e) => { if (e.target === modal) { modal.remove(); resolve(false); } };
    });
}

function showPromptDialog(message, defaultValue = '') {
    return new Promise(resolve => {
        const modal = document.createElement('div');
        modal.id = 'sessionPromptModal';
        modal.className = 'modal active';
        modal.style.zIndex = '1002';
        modal.innerHTML = `
            <div class="modal-content">
                <div class="modal-header">
                    <h2>📝 ${t('session.rename')}</h2>
                    <button class="modal-close" id="promptClose">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="prompt-dialog">
                        <p>${message}</p>
                        <div class="prompt-body">
                            <input type="text" id="promptInput" class="form-input" value="${defaultValue}" />
                        </div>
                        <div class="prompt-actions">
                            <button class="btn btn-primary" id="promptOk">${t('common.ok')}</button>
                            <button class="btn btn-secondary" id="promptCancel">${t('common.cancel')}</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        document.body.appendChild(modal);

        const input = modal.querySelector('#promptInput');
        setTimeout(() => {
            input.focus();
            input.select();
        }, 100);

        const closeModal = () => {
            modal.classList.remove('active');
            setTimeout(() => modal.remove(), 300);
        };

        modal.querySelector('#promptOk').onclick = () => {
            const value = input.value.trim();
            closeModal();
            resolve(value || null);
        };
        modal.querySelector('#promptCancel').onclick = () => {
            closeModal();
            resolve(null);
        };
        modal.querySelector('#promptClose').onclick = () => {
            closeModal();
            resolve(null);
        };
        input.onkeydown = (e) => {
            if (e.key === 'Enter') {
                const value = input.value.trim();
                closeModal();
                resolve(value || null);
            }
        };
    });
}

// 查看会话详情
async function viewSessionDetail(sessionId) {
    const session = sessions.find(s => s.sessionId === sessionId);
    if (!session) return;

    const modal = document.createElement('div');
    modal.id = 'sessionDetailModal';
    modal.className = 'modal active';
    modal.style.zIndex = '1002';

    const displayName = session.alias || session.summary || t('session.noSummary');

    modal.innerHTML = `
        <div class="modal-content session-detail-content">
            <div class="modal-header">
                <h2>💬 ${t('session.detail')}</h2>
                <button class="modal-close" onclick="closeSessionDetailModal()">&times;</button>
            </div>
            <div class="modal-body">
                <div class="session-detail-messages" id="sessionDetailMessages">
                    <div class="session-loading">${t('session.loading')}</div>
                </div>
            </div>
        </div>
    `;

    document.body.appendChild(modal);

    // 加载会话数据
    try {
        const result = JSON.parse(await GetSessionData(currentProjectDir, sessionId));
        if (!result.success) {
            document.getElementById('sessionDetailMessages').innerHTML =
                `<div class="session-empty">${t('session.loadDetailError')}</div>`;
            return;
        }

        const messages = result.data || [];
        renderMessages(messages);
    } catch (err) {
        console.error('Failed to load session data:', err);
        document.getElementById('sessionDetailMessages').innerHTML =
            `<div class="session-empty">${t('session.loadDetailError')}</div>`;
    }
}

// 渲染消息列表
function renderMessages(messages) {
    const container = document.getElementById('sessionDetailMessages');

    if (messages.length === 0) {
        container.innerHTML = `<div class="session-empty">${t('session.noMessages')}</div>`;
        return;
    }

    container.innerHTML = messages.map(msg => {
        const isUser = msg.type === 'user';
        const label = isUser ? t('session.user') : t('session.assistant');
        const content = msg.content.trim().replace(/\n/g, '<br>');

        return `
            <div class="message-card ${isUser ? 'message-user' : 'message-assistant'}">
                <div class="message-label">${label}</div>
                <div class="message-content">${content}</div>
            </div>
        `;
    }).join('');
}

// 关闭会话详情模态窗口
function closeSessionDetailModal() {
    const modal = document.getElementById('sessionDetailModal');
    if (modal) {
        modal.classList.remove('active');
        setTimeout(() => modal.remove(), 300);
    }
}
