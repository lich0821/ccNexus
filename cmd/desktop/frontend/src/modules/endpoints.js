import { t } from '../i18n/index.js';
import { escapeHtml, formatTokens, maskApiKey } from '../utils/format.js';
import { getEndpointStats } from './stats.js';
import { toggleEndpoint, testAllEndpointsZeroCost } from './config.js';
import { filterEndpoints, isFilterActive, updateFilterStats } from './filters.js';

// 提取基础名称，移除副本后缀
function extractBaseName(name) {
    // 移除类似 "(Copy)", "(副本)", "(Copy) 1", "(副本) 1" 等后缀
    // 使用固定的模式匹配，避免在函数内部调用 t()
    const copyPattern = /\(Copy\)(?:\s+\d+)?$/;
    const chineseCopyPattern = /\(副本\)(?:\s+\d+)?$/;
    
    let baseName = name.replace(copyPattern, '').trim();
    baseName = baseName.replace(chineseCopyPattern, '').trim();
    
    return baseName;
}

const ENDPOINT_TEST_STATUS_KEY = 'ccNexus_endpointTestStatus';
const ENDPOINT_VIEW_MODE_KEY = 'ccNexus_endpointViewMode';

// 获取端点测试状态
export function getEndpointTestStatus(endpointName) {
    try {
        const statusMap = JSON.parse(localStorage.getItem(ENDPOINT_TEST_STATUS_KEY) || '{}');
        return statusMap[endpointName]; // true=成功, false=失败, undefined=未测试
    } catch {
        return undefined;
    }
}

// 保存端点测试状态
export function saveEndpointTestStatus(endpointName, success) {
    try {
        const statusMap = JSON.parse(localStorage.getItem(ENDPOINT_TEST_STATUS_KEY) || '{}');
        statusMap[endpointName] = success;
        localStorage.setItem(ENDPOINT_TEST_STATUS_KEY, JSON.stringify(statusMap));
    } catch (error) {
        console.error('Failed to save endpoint test status:', error);
    }
}

// 获取端点视图模式
export function getEndpointViewMode() {
    try {
        return localStorage.getItem(ENDPOINT_VIEW_MODE_KEY) || 'detail';
    } catch {
        return 'detail';
    }
}

// 保存端点视图模式
export function saveEndpointViewMode(mode) {
    try {
        localStorage.setItem(ENDPOINT_VIEW_MODE_KEY, mode);
    } catch (error) {
        console.error('Failed to save endpoint view mode:', error);
    }
}

// 切换视图模式
export function switchEndpointViewMode(mode) {
    saveEndpointViewMode(mode);

    // 更新按钮状态
    const buttons = document.querySelectorAll('.view-mode-btn');
    buttons.forEach(btn => {
        btn.classList.toggle('active', btn.dataset.view === mode);
    });

    // 更新列表样式
    const container = document.getElementById('endpointList');
    if (mode === 'compact') {
        container.classList.add('compact-view');
    } else {
        container.classList.remove('compact-view');
    }

    // 重新渲染端点列表
    window.loadConfig();
}

// 初始化视图模式
export function initEndpointViewMode() {
    const mode = getEndpointViewMode();
    const buttons = document.querySelectorAll('.view-mode-btn');
    buttons.forEach(btn => {
        btn.classList.toggle('active', btn.dataset.view === mode);
    });
}

let currentTestButton = null;
let currentTestButtonOriginalText = '';
let currentTestIndex = -1;
let endpointPanelExpanded = true;
let tokenPoolCurrentIndex = -1;
let tokenPoolErrorCache = new Map();
let tokenPoolUsageCache = new Map();

function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    document.body.appendChild(notification);
    setTimeout(() => notification.classList.add('show'), 10);
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

function copyToClipboard(text, button) {
    navigator.clipboard.writeText(text).then(() => {
        const originalHTML = button.innerHTML;
        button.innerHTML = '<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" width="1em" height="1em"><path d="M20 6L9 17l-5-5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>';
        setTimeout(() => { button.innerHTML = originalHTML; }, 1000);
    });
}

export function getTestState() {
    return { currentTestButton, currentTestIndex };
}

export function clearTestState() {
    if (currentTestButton) {
        currentTestButton.disabled = false;
        currentTestButton.innerHTML = currentTestButtonOriginalText;

        // 恢复简洁视图的 moreBtn
        const endpointItem = currentTestButton.closest('.endpoint-item-compact');
        if (endpointItem) {
            const moreBtn = endpointItem.querySelector('[data-action="more"]');
            if (moreBtn) {
                moreBtn.disabled = false;
                moreBtn.innerHTML = '⋯';
            }
        }

        currentTestButton = null;
        currentTestButtonOriginalText = '';
        currentTestIndex = -1;
    }
}

export function setTestState(button, index) {
    currentTestButton = button;
    currentTestButtonOriginalText = button.innerHTML;
    currentTestIndex = index;
}

export async function renderEndpoints(endpoints) {
    const container = document.getElementById('endpointList');

    // Get current endpoint from backend
    let currentEndpointName = '';
    try {
        currentEndpointName = await window.go.main.App.GetCurrentEndpoint();
    } catch (error) {
        console.error('Failed to get current endpoint:', error);
    }

    // 应用筛选
    const filteredEndpoints = filterEndpoints(endpoints);
    const isFiltered = isFilterActive();

    // 更新筛选统计
    updateFilterStats(endpoints.length, filteredEndpoints.length);

    // 空状态处理
    if (filteredEndpoints.length === 0) {
        container.innerHTML = `
            <div class="empty-state" style="text-align: center; padding: 60px 20px; color: #999;">
                <div style="font-size: 48px; margin-bottom: 15px;">🔍</div>
                <p style="font-size: 16px; margin-bottom: 20px;">
                    ${isFiltered ? t('endpoints.noMatchingEndpoints') : t('endpoints.noEndpoints')}
                </p>
                ${isFiltered ? `
                    <button class="btn btn-primary" onclick="window.clearAllFilters()">
                        🔄 ${t('endpoints.clearFilters')}
                    </button>
                ` : `
                    <button class="btn btn-primary" onclick="window.showAddEndpointModal()">
                        ➕ ${t('header.addEndpoint')}
                    </button>
                `}
            </div>
        `;
        return;
    }

    container.innerHTML = '';

    const endpointStats = getEndpointStats();
    // Display endpoints in config file order (no sorting by enabled status)
    // Keep original index from full endpoints array to avoid index mismatch after filtering
    const endpointIndexMap = new Map(endpoints.map((ep, idx) => [ep, idx]));
    const sortedEndpoints = filteredEndpoints.map((ep) => {
        const originalIndex = endpointIndexMap.has(ep)
            ? endpointIndexMap.get(ep)
            : endpoints.findIndex(item => item.name === ep.name);
        const stats = endpointStats[ep.name] || { requests: 0, errors: 0, inputTokens: 0, outputTokens: 0 };
        const enabled = ep.enabled !== undefined ? ep.enabled : true;
        return { endpoint: ep, originalIndex, stats, enabled };
    });

    // 检查视图模式
    const viewMode = getEndpointViewMode();
    if (viewMode === 'compact') {
        container.classList.add('compact-view');
        renderCompactView(sortedEndpoints, container, currentEndpointName, isFiltered);
        return;
    } else {
        container.classList.remove('compact-view');
    }

    sortedEndpoints.forEach(({ endpoint: ep, originalIndex: index, stats }) => {
        const totalTokens = stats.inputTokens + stats.outputTokens;
        const enabled = ep.enabled !== undefined ? ep.enabled : true;
        const transformer = ep.transformer || 'claude';
        const model = ep.model || '';
        const authMode = ep.authMode || 'api_key';
        const isCurrentEndpoint = ep.name === currentEndpointName;

        const item = document.createElement('div');
        item.className = 'endpoint-item';
        item.dataset.name = ep.name;
        item.dataset.index = index;

        // 筛选激活时禁用拖拽
        if (isFiltered) {
            item.draggable = false;
            item.style.cursor = 'default';
            item.title = t('endpoints.dragDisabledDuringFilter');
        } else {
            item.draggable = true;
            setupDragAndDrop(item, container);
        }

        // 获取测试状态：true=成功显示✅，false=失败显示❌，undefined/unknown=未测试/未知显示⚠️
        const testStatus = getEndpointTestStatus(ep.name);
        let testStatusIcon = '⚠️';
        let testStatusTip = t('endpoints.testTipUnknown');
        if (testStatus === true) {
            testStatusIcon = '✅';
            testStatusTip = t('endpoints.testTipSuccess');
        } else if (testStatus === false) {
            testStatusIcon = '❌';
            testStatusTip = t('endpoints.testTipFailed');
        }

        item.innerHTML = `
            <div class="endpoint-info">
                <h3>
                    <span title="${testStatusTip}" style="cursor: help">${testStatusIcon}</span>
                    ${ep.name}
                    ${!enabled ? '<span class="disabled-badge">' + t('endpoints.disabled') + '</span>' : ''}
                    ${isCurrentEndpoint ? '<span class="current-badge">' + t('endpoints.current') + '</span>' : ''}
                    ${enabled && !isCurrentEndpoint ? '<button class="btn btn-switch" data-action="switch" data-name="' + ep.name + '">' + t('endpoints.switchTo') + '</button>' : ''}
                </h3>
                <p style="display: flex; align-items: center; gap: 8px; min-width: 0;"><span style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">🌐 ${ep.apiUrl}</span> <button class="copy-btn" data-copy="${ep.apiUrl}" aria-label="${t('endpoints.copy')}" title="${t('endpoints.copy')}"><svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" width="1em" height="1em"><path d="M7 4c0-1.1.9-2 2-2h11a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2h-1V8c0-2-1-3-3-3H7V4Z" fill="currentColor"></path><path d="M5 7a2 2 0 0 0-2 2v10c0 1.1.9 2 2 2h10a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2H5Z" fill="currentColor"></path></svg></button></p>
                ${authMode === 'api_key'
                    ? `<p style="display: flex; align-items: center; gap: 8px; min-width: 0;"><span style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">🔑 ${maskApiKey(ep.apiKey)}</span> <button class="copy-btn" data-copy="${ep.apiKey}" aria-label="${t('endpoints.copy')}" title="${t('endpoints.copy')}"><svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" width="1em" height="1em"><path d="M7 4c0-1.1.9-2 2-2h11a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2h-1V8c0-2-1-3-3-3H7V4Z" fill="currentColor"></path><path d="M5 7a2 2 0 0 0-2 2v10c0 1.1.9 2 2 2h10a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2H5Z" fill="currentColor"></path></svg></button></p>`
                    : `<p style="color: #666; font-size: 14px; margin-top: 3px;">🪪 Using credential pool</p>`}
                <p style="color: #666; font-size: 14px; margin-top: 5px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">🔄 ${t('endpoints.transformer')}: ${transformer}${model ? ` (${model})` : ''}</p>
                <p style="color: #666; font-size: 14px; margin-top: 3px;">📊 ${t('endpoints.requests')}: ${stats.requests} | ${t('endpoints.errors')}: ${stats.errors}</p>
                <p style="color: #666; font-size: 14px; margin-top: 3px;">🎯 ${t('endpoints.tokens')}: ${formatTokens(totalTokens)} (${t('statistics.in')}: ${formatTokens(stats.inputTokens)}, ${t('statistics.out')}: ${formatTokens(stats.outputTokens)})</p>
                ${ep.remark ? `<p style="color: #888; font-size: 13px; margin-top: 5px; font-style: italic;" title="${ep.remark}">💬 ${ep.remark.length > 20 ? ep.remark.substring(0, 20) + '...' : ep.remark}</p>` : ''}
            </div>
            <div class="endpoint-actions">
                <label class="toggle-switch">
                    <input type="checkbox" data-index="${index}" ${enabled ? 'checked' : ''}>
                    <span class="toggle-slider"></span>
                </label>
                <button class="btn-card btn-secondary" data-action="test" data-index="${index}">${t('endpoints.test')}</button>
                <button class="btn-card btn-secondary" data-action="copy" data-index="${index}">${t('endpoints.copy')}</button>
                <button class="btn-card btn-secondary" data-action="edit" data-index="${index}">${t('endpoints.edit')}</button>
                <button class="btn-card btn-danger" data-action="delete" data-index="${index}">${t('endpoints.delete')}</button>
            </div>
        `;

        const testBtn = item.querySelector('[data-action="test"]');
        const editBtn = item.querySelector('[data-action="edit"]');
        const deleteBtn = item.querySelector('[data-action="delete"]');
        const toggleSwitch = item.querySelector('input[type="checkbox"]');
        const copyBtns = item.querySelectorAll('.copy-btn');

        if (currentTestIndex === index) {
            testBtn.disabled = true;
            testBtn.innerHTML = '⏳';
            currentTestButton = testBtn;
        }

        testBtn.addEventListener('click', () => {
            const idx = parseInt(testBtn.getAttribute('data-index'));
            window.testEndpoint(idx, testBtn);
        });
        const copyBtn = item.querySelector('[data-action="copy"]');
        copyBtn.addEventListener('click', () => {
            const idx = parseInt(copyBtn.getAttribute('data-index'));
            copyEndpointConfig(idx, copyBtn);
        });
        editBtn.addEventListener('click', () => {
            const idx = parseInt(editBtn.getAttribute('data-index'));
            window.editEndpoint(idx);
        });
        deleteBtn.addEventListener('click', () => {
            const idx = parseInt(deleteBtn.getAttribute('data-index'));
            window.deleteEndpoint(idx);
        });
        toggleSwitch.addEventListener('change', async (e) => {
            const idx = parseInt(e.target.getAttribute('data-index'));
            const newEnabled = e.target.checked;
            try {
                await toggleEndpoint(idx, newEnabled);
                window.loadConfig();
            } catch (error) {
                console.error('Failed to toggle endpoint:', error);
                alert('Failed to toggle endpoint: ' + error);
                e.target.checked = !newEnabled;
            }
        });
        copyBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                copyToClipboard(btn.getAttribute('data-copy'), btn);
            });
        });

        // Add switch button event listener
        const switchBtn = item.querySelector('[data-action="switch"]');
        if (switchBtn) {
            switchBtn.addEventListener('click', async () => {
                const name = switchBtn.getAttribute('data-name');
                try {
                    switchBtn.disabled = true;
                    switchBtn.innerHTML = '⏳';
                    await window.go.main.App.SwitchToEndpoint(name);
                    window.loadConfig(); // Refresh display
                } catch (error) {
                    console.error('Failed to switch endpoint:', error);
                    alert(t('endpoints.switchFailed') + ': ' + error);
                } finally {
                    if (switchBtn) {
                        switchBtn.disabled = false;
                        switchBtn.innerHTML = t('endpoints.switchTo');
                    }
                }
            });
        }

        // Add drag and drop event listeners
        setupDragAndDrop(item, container);

        container.appendChild(item);
    });
}

function ensureTokenPoolModal() {
    let modal = document.getElementById('tokenPoolModal');
    if (modal) {
        return modal;
    }

    modal = document.createElement('div');
    modal.id = 'tokenPoolModal';
    modal.className = 'modal';
    modal.innerHTML = `
        <div class="modal-content" style="max-width: min(1100px, 96vw); width: 96vw; height: min(82vh, 760px);">
            <div class="modal-header">
                <h2 id="tokenPoolTitle">🪪 Token Pool</h2>
            </div>
            <div class="modal-body" style="padding: 6px 20px 16px 20px;">
                <div id="tokenPoolHint" style="font-size: 12px; color: #6b7280; margin-bottom: 8px; display: none;"></div>
                <div id="tokenPoolStats" style="display: block; margin-bottom: 12px;"></div>
                <div class="token-pool-proxy-bar">
                    <label for="tokenPoolProxyUrl" class="token-pool-proxy-label" style="font-size: 13px; font-weight: 600;" title="Only applies to Codex requests and credential refresh.">Codex Proxy</label>
                    <input id="tokenPoolProxyUrl" class="form-input" type="text" placeholder="${t('settings.proxyUrlPlaceholder')}">
                    <button class="btn btn-secondary" id="tokenPoolProxySaveBtn">Save</button>
                    <button class="btn btn-secondary" id="tokenPoolProxyClearBtn">Clear</button>
                </div>

                <div class="form-group">
                    <div class="token-pool-import-header">
                        <label>Batch Import JSON</label>
                        <div class="token-pool-overwrite">
                            <label class="toggle-switch" style="margin: 0;">
                                <input type="checkbox" id="tokenPoolOverwrite">
                                <span class="toggle-slider"></span>
                            </label>
                            <label for="tokenPoolOverwrite" class="token-pool-overwrite-label">Overwrite existing account/email</label>
                        </div>
                    </div>
                    <textarea id="tokenPoolImportInput" style="width: 100%; min-height: 140px; padding: 10px; border: 1px solid #d1d5db; border-radius: 8px;" placeholder='Paste one object / array / {"items":[...]}'></textarea>
                    <div style="margin-top: 8px;">
                        <button class="btn btn-primary" id="tokenPoolImportBtn">Import</button>
                        <button class="btn btn-secondary" id="tokenPoolImportFilesBtn">Import Files</button>
                        <button class="btn btn-secondary" id="tokenPoolRefreshBtn">Refresh</button>
                        <button class="btn btn-secondary" id="tokenPoolRateRefreshBtn">Refresh Limits</button>
                    </div>
                </div>

                <div style="overflow-x: auto;">
                    <table style="width: 100%; border-collapse: collapse; font-size: 13px;">
                        <thead>
                            <tr style="background: rgba(148, 163, 184, 0.15);">
                                <th style="padding: 8px; text-align: center;">Account</th>
                                <th style="padding: 8px; text-align: center;">Email</th>
                                <th style="padding: 8px; text-align: center;">Status</th>
                                <th style="padding: 8px; text-align: center;">Expires At</th>
                                <th style="padding: 8px; text-align: center; width: 220px; max-width: 220px;">Rate Limits</th>
                                <th style="padding: 8px; text-align: center;">Last Error</th>
                                <th style="padding: 8px; text-align: center;">Actions</th>
                            </tr>
                        </thead>
                        <tbody id="tokenPoolTableBody"></tbody>
                    </table>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" id="tokenPoolCloseBtn">Close</button>
            </div>
        </div>
    `;

    document.body.appendChild(modal);

    const closeModal = () => {
        modal.classList.remove('active');
    };

    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            closeModal();
        }
    });
    modal.querySelector('.modal-body')?.addEventListener('click', () => {
        closeAllTokenPoolActionMenus(modal);
    });
    modal.querySelector('#tokenPoolCloseBtn').addEventListener('click', closeModal);
    modal.querySelector('#tokenPoolImportBtn').addEventListener('click', handleTokenPoolImport);
    modal.querySelector('#tokenPoolImportFilesBtn').addEventListener('click', handleTokenPoolFileImport);
    modal.querySelector('#tokenPoolRefreshBtn').addEventListener('click', async () => {
        await loadTokenPoolData(tokenPoolCurrentIndex);
    });
    modal.querySelector('#tokenPoolRateRefreshBtn').addEventListener('click', async () => {
        await refreshTokenPoolRateLimits(tokenPoolCurrentIndex);
    });
    modal.querySelector('#tokenPoolProxySaveBtn').addEventListener('click', async () => {
        await saveTokenPoolProxySetting();
    });
    modal.querySelector('#tokenPoolProxyClearBtn').addEventListener('click', async () => {
        await saveTokenPoolProxySetting(true);
    });

    return modal;
}

async function loadTokenPoolProxySetting() {
    const modal = ensureTokenPoolModal();
    const input = modal.querySelector('#tokenPoolProxyUrl');
    if (!input) {
        return;
    }

    try {
        const proxyUrl = await window.go.main.App.GetCodexProxyURL();
        input.value = proxyUrl || '';
    } catch (error) {
        const message = error?.message || String(error);
        showNotification(`Failed to load Codex proxy: ${message}`, 'error');
    }
}

async function saveTokenPoolProxySetting(clear = false) {
    const modal = ensureTokenPoolModal();
    const input = modal.querySelector('#tokenPoolProxyUrl');
    if (!input) {
        return;
    }

    const proxyUrl = clear ? '' : input.value.trim();
    try {
        await window.go.main.App.SetCodexProxyURL(proxyUrl);
        input.value = proxyUrl;
        showNotification(proxyUrl ? 'Codex proxy updated' : 'Codex proxy cleared', 'success');
    } catch (error) {
        const message = error?.message || String(error);
        showNotification(`Failed to save Codex proxy: ${message}`, 'error');
    }
}

function parseAppJSON(value) {
    if (typeof value === 'string') {
        return JSON.parse(value);
    }
    return value;
}

function maskTokenPoolAccountID(accountId) {
    const raw = (accountId || '').trim();
    if (!raw) {
        return '-';
    }
    return `${raw.slice(0, 8)}*`;
}

function maskTokenPoolEmail(email) {
    const raw = (email || '').trim();
    if (!raw || !raw.includes('@')) {
        return raw || '-';
    }
    const [local, domain] = raw.split('@');
    if (!local || !domain) {
        return raw;
    }
    const localMasked = local.length <= 2
        ? `${local[0] || ''}*`
        : `${local[0]}*${local.slice(-2)}`;
    const domainParts = domain.split('.');
    const tld = domainParts.length > 1 ? domainParts[domainParts.length - 1] : '';
    const firstLabel = domainParts[0] || '';
    const domainMasked = firstLabel
        ? `${firstLabel[0]}*${tld ? tld : ''}`
        : `*${tld ? tld : ''}`;
    return `${localMasked}@${domainMasked}`;
}

function ensureTokenPoolErrorModal() {
    let modal = document.getElementById('tokenPoolErrorModal');
    if (modal) {
        return modal;
    }

    modal = document.createElement('div');
    modal.id = 'tokenPoolErrorModal';
    modal.className = 'modal';
    modal.innerHTML = `
        <div class="modal-content token-pool-error-modal-content">
            <div class="modal-header">
                <h2>🧾 Last Error</h2>
            </div>
            <div class="modal-body">
                <pre id="tokenPoolErrorText" class="token-pool-error-pre"></pre>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" id="tokenPoolErrorCloseBtn">Close</button>
            </div>
        </div>
    `;

    document.body.appendChild(modal);
    const closeModal = () => modal.classList.remove('active');
    modal.addEventListener('click', (event) => {
        if (event.target === modal) {
            closeModal();
        }
    });
    modal.querySelector('#tokenPoolErrorCloseBtn')?.addEventListener('click', closeModal);
    return modal;
}

function showTokenPoolErrorDialog(errorText) {
    const modal = ensureTokenPoolErrorModal();
    const textEl = modal.querySelector('#tokenPoolErrorText');
    if (textEl) {
        textEl.textContent = (errorText || '').trim() || '-';
    }
    modal.classList.add('active');
}

function ensureTokenPoolUsageModal() {
    let modal = document.getElementById('tokenPoolUsageModal');
    if (modal) {
        return modal;
    }

    modal = document.createElement('div');
    modal.id = 'tokenPoolUsageModal';
    modal.className = 'modal';
    modal.innerHTML = `
        <div class="modal-content token-pool-usage-modal-content">
            <div class="modal-header">
                <h2 id="tokenPoolUsageTitle">📊 Usage</h2>
            </div>
            <div class="modal-body">
                <div id="tokenPoolUsageBody" class="token-pool-usage-body"></div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" id="tokenPoolUsageCloseBtn">Close</button>
            </div>
        </div>
    `;

    document.body.appendChild(modal);
    const closeModal = () => modal.classList.remove('active');
    modal.addEventListener('click', (event) => {
        if (event.target === modal) {
            closeModal();
        }
    });
    modal.querySelector('#tokenPoolUsageCloseBtn')?.addEventListener('click', closeModal);
    return modal;
}

function showTokenPoolUsageDialog(label, usage) {
    const modal = ensureTokenPoolUsageModal();
    const title = modal.querySelector('#tokenPoolUsageTitle');
    const body = modal.querySelector('#tokenPoolUsageBody');
    if (title) {
        title.textContent = `📊 Usage${label ? `: ${label}` : ''}`;
    }

    if (!usage) {
        body.innerHTML = `<div class="token-pool-usage-empty">No usage yet</div>`;
    } else {
        const totalTokens = (usage.inputTokens || 0) + (usage.outputTokens || 0);
        const updatedAt = usage.updatedAt ? formatTokenPoolTime(usage.updatedAt) : '-';
        body.innerHTML = `
            <div class="token-pool-usage-grid">
                <div>Requests</div><div>${usage.requests || 0}</div>
                <div>Errors</div><div>${usage.errors || 0}</div>
                <div>Total Tokens</div><div>${formatTokens(totalTokens)}</div>
                <div>Input Tokens</div><div>${formatTokens(usage.inputTokens || 0)}</div>
                <div>Output Tokens</div><div>${formatTokens(usage.outputTokens || 0)}</div>
                <div>Updated</div><div>${escapeHtml(updatedAt)}</div>
            </div>
        `;
    }

    modal.classList.add('active');
}

function showTokenPoolUpdateTokenDialog() {
    return new Promise((resolve) => {
        const modal = document.createElement('div');
        modal.id = 'tokenPoolUpdateTokenModal';
        modal.className = 'modal active';
        modal.style.zIndex = '1002';
        modal.innerHTML = `
            <div class="modal-content">
                <div class="modal-header">
                    <h2>🔑 Update Token</h2>
                </div>
                <div class="modal-body">
                    <div class="prompt-dialog">
                        <p><span class="required">*</span>Access token</p>
                        <div class="prompt-body">
                            <textarea id="tokenPoolUpdateAccess" class="form-input" rows="4" placeholder="Paste access_token here"></textarea>
                        </div>
                        <p style="margin-top: 12px;">expiresAt (RFC3339, optional)</p>
                        <div class="prompt-body">
                            <input type="text" id="tokenPoolUpdateExpires" class="form-input" placeholder="2026-03-18T09:22:23Z" />
                        </div>
                        <div class="prompt-actions">
                            <button class="btn btn-primary" id="tokenPoolUpdateOk">OK</button>
                            <button class="btn btn-secondary" id="tokenPoolUpdateCancel">Cancel</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        document.body.appendChild(modal);

        const accessEl = modal.querySelector('#tokenPoolUpdateAccess');
        const expiresEl = modal.querySelector('#tokenPoolUpdateExpires');
        setTimeout(() => accessEl?.focus(), 50);

        const closeModal = (value) => {
            modal.classList.remove('active');
            setTimeout(() => modal.remove(), 200);
            resolve(value);
        };

        const handleSubmit = () => {
            const token = (accessEl?.value || '').trim();
            if (!token) {
                showNotification('Access token is required', 'warning');
                accessEl?.focus();
                return;
            }
            const expiresAt = (expiresEl?.value || '').trim();
            closeModal({ token, expiresAt });
        };

        modal.querySelector('#tokenPoolUpdateOk')?.addEventListener('click', handleSubmit);
        modal.querySelector('#tokenPoolUpdateCancel')?.addEventListener('click', () => closeModal(null));
        modal.addEventListener('click', (event) => {
            if (event.target === modal) {
                closeModal(null);
            }
        });

        modal.addEventListener('keydown', (event) => {
            if (event.key === 'Escape') {
                closeModal(null);
            }
        });
    });
}

function formatTokenPoolTime(value) {
    if (!value) {
        return '-';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return value;
    }
    const pad = (num) => String(num).padStart(2, '0');
    const year = date.getFullYear();
    const month = pad(date.getMonth() + 1);
    const day = pad(date.getDate());
    const hours = pad(date.getHours());
    const minutes = pad(date.getMinutes());
    const seconds = pad(date.getSeconds());
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
}

function renderTokenPoolStatus(status, enabled = true, rateLimits = null) {
    if (!enabled) {
        return `<span style="display: inline-block; padding: 2px 8px; border-radius: 999px; background: #6b7280; color: #fff; font-size: 12px;">disabled</span>`;
    }
    const rateStatus = (rateLimits?.status || '').trim();
    const normalized = rateStatus && rateStatus !== 'ok' ? rateStatus : (status || 'active');
    const colors = {
        active: '#10b981',
        expiring: '#f59e0b',
        need_refresh: '#f97316',
        expired: '#ef4444',
        invalid: '#ef4444',
        unauthorized: '#ef4444',
        blocked: '#ef4444',
        error: '#ef4444',
        network: '#f59e0b',
        upstream: '#f59e0b',
        parse_error: '#6366f1',
        empty: '#6366f1',
        missing_token: '#6b7280',
        invalid: '#ef4444',
        cooldown: '#6366f1'
    };
    const color = colors[normalized] || '#6b7280';
    return `<span style="display: inline-block; padding: 2px 8px; border-radius: 999px; background: ${color}; color: #fff; font-size: 12px;">${escapeHtml(normalized)}</span>`;
}

function renderTokenPoolStats(stats = {}) {
    const items = [
        ['Total', stats.total || 0],
        ['Active', stats.active || 0],
        ['Expiring', stats.expiring || 0],
        ['Need Refresh', stats.needRefresh || 0],
        ['Expired', stats.expired || 0],
        ['Invalid', stats.invalid || 0],
        ['Cooldown', stats.cooldown || 0],
        ['Disabled', stats.disabled || 0]
    ];

    const parts = items.map(([label, value]) => {
        const highlight = value > 0 && label !== 'Total' && label !== 'Active' ? ' token-pool-stat-alert' : '';
        return `<span class="token-pool-stat${highlight}"><strong>${label}</strong> ${value}</span>`;
    });

    return `<div class="token-pool-stats-line">${parts.join('<span class="token-pool-stat-sep">·</span>')}</div>`;
}

function renderTokenPoolRows(credentials = []) {
    if (!credentials.length) {
        return `<tr><td colspan="7" style="padding: 16px; text-align: center; color: #6b7280;">No credentials</td></tr>`;
    }

    const latestUsed = credentials.reduce((max, cred) => {
        if (!cred.lastUsedAt) {
            return max;
        }
        const t = Date.parse(cred.lastUsedAt);
        if (Number.isNaN(t)) {
            return max;
        }
        return t > max ? t : max;
    }, 0);

    return credentials.map((cred) => `
        <tr class="${latestUsed && cred.lastUsedAt && Date.parse(cred.lastUsedAt) === latestUsed ? 'token-pool-row-active' : ''}" style="border-top: 1px solid rgba(148, 163, 184, 0.2);">
            <td style="padding: 8px;"><code title="${escapeHtml(cred.accountId || '')}">${escapeHtml(maskTokenPoolAccountID(cred.accountId))}</code></td>
            <td style="padding: 8px;"><span title="${escapeHtml(cred.email || '')}">${escapeHtml(maskTokenPoolEmail(cred.email))}</span></td>
            <td style="padding: 8px;">
                <div class="token-pool-status-cell">
                    ${renderTokenPoolStatus(cred.status, cred.enabled, cred.rateLimits)}
                </div>
            </td>
            <td style="padding: 8px; white-space: nowrap;">${escapeHtml(formatTokenPoolTime(cred.expiresAt))}</td>
            <td style="padding: 8px; width: 220px; max-width: 220px;">${renderTokenPoolRateLimits(cred.rateLimits)}</td>
            <td style="padding: 8px;">
                ${tokenPoolErrorCache.has(String(cred.id))
                    ? `<button type="button" class="btn btn-secondary token-pool-error-view" data-error-id="${cred.id}" style="padding: 4px 8px; font-size: 12px;">View</button>`
                    : '-'}
            </td>
            <td style="padding: 8px;">
                <div class="token-pool-actions">
                    <button type="button" class="btn btn-secondary token-pool-toggle-action" data-id="${cred.id}" data-enabled="${cred.enabled ? '1' : '0'}" style="padding: 4px 8px; font-size: 12px;">${cred.enabled ? 'Disable' : 'Enable'}</button>
                    <div class="token-pool-more-wrap">
                        <button type="button" class="btn btn-secondary token-pool-more-toggle" data-id="${cred.id}" style="padding: 4px 8px; font-size: 12px;">More</button>
                        <div class="token-pool-more-menu">
                            <button type="button" class="token-pool-activate" data-id="${cred.id}">Activate</button>
                            <button type="button" class="token-pool-rate-refresh" data-id="${cred.id}">Refresh limits</button>
                            <button type="button" class="token-pool-refresh-token" data-id="${cred.id}">Refresh token</button>
                            <button type="button" class="token-pool-usage" data-id="${cred.id}">Usage</button>
                            <button type="button" class="token-pool-update" data-id="${cred.id}">Update token</button>
                            <button type="button" class="token-pool-delete" data-id="${cred.id}">Delete</button>
                        </div>
                    </div>
                </div>
            </td>
        </tr>
    `).join('');
}

function formatTokenPoolWindowLabel(windowMinutes) {
    if (!windowMinutes || windowMinutes <= 0) {
        return '';
    }
    if (windowMinutes % (60 * 24) === 0) {
        return `${windowMinutes / (60 * 24)}d`;
    }
    if (windowMinutes % 60 === 0) {
        return `${windowMinutes / 60}h`;
    }
    return `${windowMinutes}m`;
}

function renderTokenPoolRateLimits(rateLimits) {
    if (!rateLimits) {
        return '-';
    }
    const status = (rateLimits.status || '').trim();
    const updatedAt = rateLimits.updatedAt ? formatTokenPoolTime(rateLimits.updatedAt) : '-';
    const data = rateLimits.data || {};
    const snapshot = data.snapshot || {};
    const primary = snapshot.primary || {};
    const secondary = snapshot.secondary || {};
    const usedPercent = typeof primary.usedPercent === 'number' ? Math.round(primary.usedPercent) : null;
    const primaryWindowMinutes = primary.windowMinutes;
    const secondaryUsedPercent = typeof secondary.usedPercent === 'number' ? Math.round(secondary.usedPercent) : null;
    const secondaryWindowMinutes = secondary.windowMinutes;
    const primaryLabel = formatTokenPoolWindowLabel(primaryWindowMinutes);
    const secondaryLabel = formatTokenPoolWindowLabel(secondaryWindowMinutes);
    const parts = [];
    if (primaryLabel || usedPercent !== null) {
        const pct = usedPercent !== null ? `${usedPercent}%` : '';
        parts.push(`${pct}@${primaryLabel || 'window'}`.replace(/^@/, '').trim());
    }
    if (secondaryLabel || secondaryUsedPercent !== null) {
        const pct = secondaryUsedPercent !== null ? `${secondaryUsedPercent}%` : '';
        parts.push(`${pct}@${secondaryLabel || 'short'}`.replace(/^@/, '').trim());
    }
    const summary = parts.join(' · ');
    const credits = snapshot.credits || {};
    const creditText = credits.unlimited
        ? 'unlimited'
        : credits.balance
            ? `balance ${credits.balance}`
            : credits.hasCredits
                ? 'has credits'
                : '';
    const primaryReset = primary.resetsAt ? formatTokenPoolTime(new Date(primary.resetsAt * 1000).toISOString()) : '';
    const secondaryReset = secondary.resetsAt ? formatTokenPoolTime(new Date(secondary.resetsAt * 1000).toISOString()) : '';
    const metaParts = [];
    if (creditText) {
        metaParts.push(creditText);
    }
    if (!summary && status === 'ok') {
        metaParts.push('ok');
    }

    const detailLines = [];
    if (summary) {
        detailLines.push(summary);
    }
    if (primaryReset) {
        detailLines.push(`reset ${primaryLabel || 'primary'} ${primaryReset}`.trim());
    }
    if (secondaryReset) {
        detailLines.push(`reset ${secondaryLabel || 'secondary'} ${secondaryReset}`.trim());
    }
    if (creditText) {
        detailLines.push(`credits ${creditText}`);
    }
    if (updatedAt) {
        detailLines.push(`updated ${updatedAt}`);
    }
    const detailTitle = detailLines.join(' · ');

    const errorText = (rateLimits.error || '').trim();
    if (status && status !== 'ok') {
        return '-';
    }

    if (!summary && !metaParts.length) {
        return '-';
    }

    const mainLine = escapeHtml(summary || metaParts.shift() || '');
    const metaLine = metaParts.length ? escapeHtml(metaParts.join(' · ')) : '';
    return `
        <div class="token-pool-rate-cell" title="${escapeHtml(detailTitle)}">
            <div class="token-pool-rate-main">${mainLine}</div>
            ${metaLine ? `<div class="token-pool-rate-meta">${metaLine}</div>` : ''}
        </div>
    `;
}

function closeAllTokenPoolActionMenus(scope = document) {
    scope.querySelectorAll('.token-pool-more-menu.show').forEach((menu) => {
        menu.classList.remove('show');
    });
}

function setTokenPoolHint(modal, text) {
    const hintEl = modal.querySelector('#tokenPoolHint');
    if (!hintEl) {
        return;
    }
    const message = (text || '').trim();
    hintEl.textContent = message;
    hintEl.style.display = message ? 'block' : 'none';
}

async function loadTokenPoolData(index) {
    if (index < 0) {
        return;
    }

    const modal = ensureTokenPoolModal();
    setTokenPoolHint(modal, 'Loading token pool...');
    const raw = await window.go.main.App.GetEndpointCredentials(index);
    const parsed = parseAppJSON(raw);
    if (!parsed.success) {
        setTokenPoolHint(modal, `Load failed: ${parsed.error || 'unknown error'}`);
        throw new Error(parsed.error || 'Failed to load token pool');
    }

    const payload = parsed.data || {};
    const credentials = payload.credentials || [];
    const stats = payload.stats || {};

    tokenPoolErrorCache = new Map();
    tokenPoolUsageCache = new Map();
    credentials.forEach((cred) => {
        const primaryError = (cred.lastError || '').trim();
        const rateErr = (cred.rateLimits && cred.rateLimits.status && cred.rateLimits.status !== 'ok')
            ? (cred.rateLimits.error || '').trim()
            : '';
        const displayError = primaryError || rateErr;
        if (displayError) {
            tokenPoolErrorCache.set(String(cred.id), displayError);
        }
        if (cred.usage) {
            tokenPoolUsageCache.set(String(cred.id), cred.usage);
        }
    });

    const statsEl = modal.querySelector('#tokenPoolStats');
    const bodyEl = modal.querySelector('#tokenPoolTableBody');
    statsEl.innerHTML = renderTokenPoolStats(stats);
    bodyEl.innerHTML = renderTokenPoolRows(credentials);
    setTokenPoolHint(modal, '');

    bodyEl.querySelectorAll('.token-pool-toggle-action').forEach((button) => {
        button.addEventListener('click', async () => {
            const credentialID = Number(button.dataset.id);
            const currentlyEnabled = button.dataset.enabled === '1';
            const targetEnabled = !currentlyEnabled;
            try {
                await window.go.main.App.SetEndpointCredentialEnabled(tokenPoolCurrentIndex, credentialID, targetEnabled);
                showNotification(`Credential ${targetEnabled ? 'enabled' : 'disabled'}`, 'success');
                await loadTokenPoolData(tokenPoolCurrentIndex);
                if (window.loadConfig) {
                    window.loadConfig();
                }
            } catch (error) {
                const message = error?.message || String(error);
                showNotification(`Failed: ${message}`, 'error');
                await loadTokenPoolData(tokenPoolCurrentIndex);
            }
            closeAllTokenPoolActionMenus(bodyEl);
        });
    });

    bodyEl.querySelectorAll('.token-pool-activate').forEach((button) => {
        button.addEventListener('click', async () => {
            try {
                await window.go.main.App.ActivateEndpointCredential(tokenPoolCurrentIndex, Number(button.dataset.id));
                showNotification('Credential activated', 'success');
                await loadTokenPoolData(tokenPoolCurrentIndex);
                if (window.loadConfig) {
                    window.loadConfig();
                }
            } catch (error) {
                const message = error?.message || String(error);
                showNotification(`Failed: ${message}`, 'error');
            }
        });
    });

    bodyEl.querySelectorAll('.token-pool-more-toggle').forEach((button) => {
        button.addEventListener('click', (event) => {
            event.preventDefault();
            event.stopPropagation();

            const wrap = button.closest('.token-pool-more-wrap');
            const menu = wrap?.querySelector('.token-pool-more-menu');
            if (!menu) {
                return;
            }

            const shouldOpen = !menu.classList.contains('show');
            closeAllTokenPoolActionMenus(bodyEl);
            if (shouldOpen) {
                menu.classList.add('show');
            }
        });
    });

    bodyEl.querySelectorAll('.token-pool-more-menu').forEach((menu) => {
        menu.addEventListener('click', (event) => {
            event.stopPropagation();
        });
    });

    bodyEl.querySelectorAll('.token-pool-error-view').forEach((button) => {
        button.addEventListener('click', (event) => {
            event.preventDefault();
            event.stopPropagation();
            const errorId = button.dataset.errorId;
            const errorText = errorId ? tokenPoolErrorCache.get(String(errorId)) || '' : '';
            showTokenPoolErrorDialog(errorText);
        });
    });

    bodyEl.querySelectorAll('.token-pool-rate-error-view').forEach((button) => {
        button.addEventListener('click', (event) => {
            event.preventDefault();
            event.stopPropagation();
            showTokenPoolErrorDialog(button.dataset.error || '');
        });
    });

    bodyEl.querySelectorAll('.token-pool-usage').forEach((button) => {
        button.addEventListener('click', (event) => {
            event.preventDefault();
            event.stopPropagation();
            const credentialID = String(button.dataset.id || '');
            const row = button.closest('tr');
            const accountText = row?.querySelector('td code')?.textContent?.trim() || '';
            const label = accountText || '';
            const usage = tokenPoolUsageCache.get(credentialID) || null;
            showTokenPoolUsageDialog(label, usage);
            closeAllTokenPoolActionMenus(bodyEl);
        });
    });

    bodyEl.querySelectorAll('.token-pool-update').forEach((button) => {
        button.addEventListener('click', async () => {
            const result = await showTokenPoolUpdateTokenDialog();
            if (!result) {
                return;
            }
            try {
                await window.go.main.App.UpdateEndpointCredentialToken(
                    tokenPoolCurrentIndex,
                    Number(button.dataset.id),
                    result.token,
                    result.expiresAt
                );
                showNotification('Credential updated', 'success');
                await loadTokenPoolData(tokenPoolCurrentIndex);
                if (window.loadConfig) {
                    window.loadConfig();
                }
            } catch (error) {
                const message = error?.message || String(error);
                showNotification(`Failed: ${message}`, 'error');
            }
            closeAllTokenPoolActionMenus(bodyEl);
        });
    });

    bodyEl.querySelectorAll('.token-pool-rate-refresh').forEach((button) => {
        button.addEventListener('click', async () => {
            const credentialID = Number(button.dataset.id);
            if (!Number.isFinite(credentialID) || credentialID <= 0) {
                showNotification('Invalid credential id', 'error');
                return;
            }
            const row = button.closest('tr');
            const accountText = row?.querySelector('td code')?.textContent?.trim() || '';
            const label = accountText ? `${accountText} (#${credentialID})` : `#${credentialID}`;
            const modal = ensureTokenPoolModal();
            try {
                setTokenPoolHint(modal, `Refreshing rate limits for ${label}...`);
                const raw = await window.go.main.App.FetchCodexRateLimitsForCredential(tokenPoolCurrentIndex, credentialID);
                const result = parseAppJSON(raw);
                if (!result.success) {
                    throw new Error(result.error || 'Failed to refresh rate limits');
                }
                const data = result.data || {};
                const detail = `updated ${data.updated || 0}, failed ${data.failed || 0}, skipped ${data.skipped || 0}`;
                await loadTokenPoolData(tokenPoolCurrentIndex);
                const refreshedRow = modal.querySelector(`.token-pool-rate-refresh[data-id="${credentialID}"]`)?.closest('tr');
                const rateMain = refreshedRow?.querySelector('.token-pool-rate-main')?.textContent?.trim();
                const rateStatus = refreshedRow?.querySelector('.token-pool-rate-status')?.textContent?.trim();
                const rateSummary = rateMain || rateStatus || '-';
                showNotification(`Rate limits refreshed for ${label}: ${rateSummary} (${detail})`, 'success');
                setTokenPoolHint(modal, `Rate limits refreshed for ${label}: ${rateSummary} (${detail})`);
            } catch (error) {
                const message = error?.message || String(error);
                showNotification(`Rate limits refresh failed: ${message}`, 'error');
                setTokenPoolHint(modal, `Rate limits refresh failed: ${message}`);
            } finally {
                closeAllTokenPoolActionMenus(bodyEl);
            }
        });
    });

    bodyEl.querySelectorAll('.token-pool-refresh-token').forEach((button) => {
        button.addEventListener('click', async () => {
            const credentialID = Number(button.dataset.id);
            if (!Number.isFinite(credentialID) || credentialID <= 0) {
                showNotification('Invalid credential id', 'error');
                return;
            }
            const row = button.closest('tr');
            const accountText = row?.querySelector('td code')?.textContent?.trim() || '';
            const label = accountText ? `${accountText} (#${credentialID})` : `#${credentialID}`;
            const modal = ensureTokenPoolModal();
            try {
                setTokenPoolHint(modal, `Refreshing token for ${label}...`);
                const raw = await window.go.main.App.RefreshEndpointCredential(tokenPoolCurrentIndex, credentialID);
                const result = parseAppJSON(raw);
                if (!result.success) {
                    throw new Error(result.error || 'Failed to refresh token');
                }
                showNotification(`Token refreshed for ${label}`, 'success');
                await loadTokenPoolData(tokenPoolCurrentIndex);
                setTokenPoolHint(modal, `Token refreshed for ${label}`);
            } catch (error) {
                const message = error?.message || String(error);
                showNotification(`Token refresh failed: ${message}`, 'error');
                setTokenPoolHint(modal, `Token refresh failed: ${message}`);
                await loadTokenPoolData(tokenPoolCurrentIndex);
            } finally {
                closeAllTokenPoolActionMenus(bodyEl);
            }
        });
    });

    bodyEl.querySelectorAll('.token-pool-delete').forEach((button) => {
        button.addEventListener('click', async () => {
            try {
                const credentialID = Number(button.dataset.id);
                if (!Number.isFinite(credentialID) || credentialID <= 0) {
                    throw new Error(`invalid credential id: ${button.dataset.id}`);
                }

                console.info('[TokenPool] delete clicked', {
                    endpointIndex: tokenPoolCurrentIndex,
                    credentialID
                });

                showNotification(`Deleting credential #${credentialID}...`, 'info');
                await window.go.main.App.DeleteEndpointCredential(tokenPoolCurrentIndex, credentialID);
                showNotification('Credential deleted', 'success');
                await loadTokenPoolData(tokenPoolCurrentIndex);
                if (window.loadConfig) {
                    window.loadConfig();
                }
            } catch (error) {
                const message = error?.message || String(error);
                console.error('[TokenPool] delete failed', {
                    endpointIndex: tokenPoolCurrentIndex,
                    credentialID: button.dataset.id,
                    error
                });
                showNotification(`Failed: ${message}`, 'error');
            }
            closeAllTokenPoolActionMenus(bodyEl);
        });
    });
}

async function refreshTokenPoolRateLimits(index) {
    if (index < 0) {
        return;
    }

    const modal = ensureTokenPoolModal();
    const button = modal.querySelector('#tokenPoolRateRefreshBtn');
    try {
        if (button) {
            button.disabled = true;
            button.textContent = 'Refreshing...';
        }
        setTokenPoolHint(modal, 'Fetching rate limits...');

        const raw = await window.go.main.App.FetchCodexRateLimits(index);
        const result = parseAppJSON(raw);
        if (!result.success) {
            throw new Error(result.error || 'Failed to fetch rate limits');
        }

        const data = result.data || {};
        showNotification(`Rate limits refreshed: ${data.updated || 0} updated, ${data.failed || 0} failed, ${data.skipped || 0} skipped`, 'success');
        await loadTokenPoolData(index);
    } catch (error) {
        const message = error?.message || String(error);
        showNotification(`Rate limits refresh failed: ${message}`, 'error');
        setTokenPoolHint(modal, `Rate limits refresh failed: ${message}`);
    } finally {
        if (button) {
            button.disabled = false;
            button.textContent = 'Refresh Limits';
        }
    }
}

async function handleTokenPoolImport() {
    if (tokenPoolCurrentIndex < 0) {
        return;
    }

    const modal = ensureTokenPoolModal();
    const input = modal.querySelector('#tokenPoolImportInput');
    const overwrite = modal.querySelector('#tokenPoolOverwrite')?.checked === true;
    const raw = (input?.value || '').trim();

    if (!raw) {
        showNotification('Please paste JSON first', 'warning');
        return;
    }

    try {
        JSON.parse(raw);
    } catch {
        showNotification('Invalid JSON', 'error');
        return;
    }

    try {
        const resultRaw = await window.go.main.App.ImportEndpointCredentials(tokenPoolCurrentIndex, raw, overwrite);
        const result = parseAppJSON(resultRaw);
        if (!result.success) {
            throw new Error(result.error || 'Import failed');
        }

        const data = result.data || {};
        showNotification(
            `Import: +${data.created || 0}, updated ${data.updated || 0}, skipped ${data.skipped || 0}, failed ${data.failed || 0}`,
            'success'
        );
        input.value = '';
        await loadTokenPoolData(tokenPoolCurrentIndex);
        if (window.loadConfig) {
            window.loadConfig();
        }
    } catch (error) {
        const message = error?.message || String(error);
        showNotification(`Import failed: ${message}`, 'error');
    }
}

async function handleTokenPoolFileImport() {
    if (tokenPoolCurrentIndex < 0) {
        return;
    }

    const modal = ensureTokenPoolModal();
    const overwrite = modal.querySelector('#tokenPoolOverwrite')?.checked === true;

    try {
        setTokenPoolHint(modal, 'Opening file picker...');
        const resultRaw = await window.go.main.App.ImportEndpointCredentialsFromFiles(tokenPoolCurrentIndex, overwrite);
        const result = parseAppJSON(resultRaw);
        if (!result.success) {
            throw new Error(result.error || 'Import failed');
        }

        const data = result.data || {};
        if ((data.processed || 0) === 0 && (data.failed || 0) === 0) {
            showNotification('No files selected', 'warning');
            setTokenPoolHint(modal, 'No files selected.');
            return;
        }

        showNotification(
            `Import files: +${data.created || 0}, updated ${data.updated || 0}, skipped ${data.skipped || 0}, failed ${data.failed || 0}`,
            (data.failed || 0) > 0 ? 'warning' : 'success'
        );
        await loadTokenPoolData(tokenPoolCurrentIndex);
        if (window.loadConfig) {
            window.loadConfig();
        }
    } catch (error) {
        const message = error?.message || String(error);
        if (message.toLowerCase().includes('no files selected')) {
            showNotification('No files selected', 'warning');
            setTokenPoolHint(modal, 'No files selected.');
            return;
        }
        showNotification(`Import files failed: ${message}`, 'error');
        setTokenPoolHint(modal, `Import files failed: ${message}`);
    }
}

export async function openTokenPoolModal(index, endpointName = '') {
    tokenPoolCurrentIndex = index;
    const modal = ensureTokenPoolModal();
    const title = modal.querySelector('#tokenPoolTitle');
    title.textContent = `🪪 Token Pool${endpointName ? `: ${endpointName}` : ''}`;
    modal.classList.add('active');
    await loadTokenPoolProxySetting();

    try {
        await loadTokenPoolData(index);
    } catch (error) {
        const message = error?.message || String(error);
        showNotification(`Failed to load token pool: ${message}`, 'error');
    }
}

// 克隆端点配置（创建副本）
function copyEndpointConfig(index, button) {
    const allEndpoints = window.config?.endpoints || [];
    
    if (index < 0 || index >= allEndpoints.length) {
        const errorMsg = `Invalid index ${index} for cloning endpoint. Total endpoints: ${allEndpoints.length} at ${new Date().toISOString()}`;
        console.error(errorMsg);
        if (typeof window.logError === 'function') {
            window.logError(errorMsg);
        }
        showNotification(t('endpoints.cloneFailed') + ': ' + (t('endpoints.invalidIndex') || `Invalid index ${index}`), 'error');
        return;
    }
    
    const endpoint = allEndpoints[index];
    
    if (endpoint) {
        const clonedEndpoint = { ...endpoint };
        
        const baseName = extractBaseName(endpoint.name);
        const copySuffix = '(Copy)';
        
        let newName = `${baseName}${copySuffix}`;
        let counter = 1;
        while (allEndpoints.some(ep => ep.name === newName)) {
            newName = `${baseName}${copySuffix} ${counter}`;
            counter++;
        }
        clonedEndpoint.name = newName;
        
        if (clonedEndpoint.authMode === "token_pool") {
            delete clonedEndpoint.apiKey;
        }
        
        const originalHTML = button.innerHTML;
        button.innerHTML = '<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" width="1em" height="1em"><path d="M20 6L9 17l-5-5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>';
        setTimeout(() => { button.innerHTML = originalHTML; }, 1000);
        
        showNotification(t('endpoints.cloned') || 'Endpoint cloned successfully', 'success');
        
        window.clonedEndpointData = clonedEndpoint;
        
        if (typeof window.showAddEndpointModalWithPreset === 'function') {
            try {
                window.showAddEndpointModalWithPreset(clonedEndpoint);
            } catch (error) {
                const errorMsg = `Error calling showAddEndpointModalWithPreset at ${new Date().toISOString()}: ${error.message}\nStack: ${error.stack}`;
                console.error(errorMsg);
                try {
                    if (typeof window.logError === 'function') {
                        window.logError(errorMsg);
                    }
                } catch (logErr) {
                    console.error('Failed to call logError:', logErr);
                }
                showNotification(t('endpoints.cloneFailed') + ': ' + error.message || `Failed to clone endpoint: ${error.message}`, 'error');
            }
        } else {
            const errorMsg = `showAddEndpointModalWithPreset function is not available at ${new Date().toISOString()}`;
            console.error(errorMsg);
            if (typeof window.logError === 'function') {
                window.logError(errorMsg);
            }
            showNotification(t('endpoints.cloneFailed') + ': ' + t('endpoints.functionUnavailable') || 'Failed to clone endpoint: Function not available', 'error');
        }
    } else {
        const errorMsg = `Failed to clone endpoint: endpoint data not found at index ${index} at ${new Date().toISOString()}`;
        console.error(errorMsg);
        if (typeof window.logError === 'function') {
            window.logError(errorMsg);
        }
        showNotification(t('endpoints.cloneFailed') + ': ' + (t('endpoints.noEndpointAtIdxWithIndex') || `No endpoint found at index ${index}`), 'error');
    }
}

export function toggleEndpointPanel() {
    const panel = document.getElementById('endpointPanel');
    const icon = document.getElementById('endpointToggleIcon');
    const text = document.getElementById('endpointToggleText');

    endpointPanelExpanded = !endpointPanelExpanded;

    if (endpointPanelExpanded) {
        panel.style.display = 'block';
        icon.textContent = '🔼';
        text.textContent = t('endpoints.collapse');
    } else {
        panel.style.display = 'none';
        icon.textContent = '🔽';
        text.textContent = t('endpoints.expand');
    }
}

// Drag and drop state
let draggedElement = null;
let draggedOverElement = null;
let draggedOriginalName = null;
let autoScrollInterval = null;

// Auto scroll when dragging near edges
function autoScroll(e) {
    const scrollContainer = document.querySelector('.container');
    const scrollThreshold = 80;
    const scrollSpeed = 10;

    const rect = scrollContainer.getBoundingClientRect();
    const distanceFromTop = e.clientY - rect.top;
    const distanceFromBottom = rect.bottom - e.clientY;

    if (distanceFromTop < scrollThreshold) {
        scrollContainer.scrollTop -= scrollSpeed;
    } else if (distanceFromBottom < scrollThreshold) {
        scrollContainer.scrollTop += scrollSpeed;
    }
}

// Setup drag and drop for an endpoint item
function setupDragAndDrop(item, container) {
    item.addEventListener('dragstart', (e) => {
        draggedElement = item;
        draggedOriginalName = item.dataset.name;
        item.classList.add('dragging');
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/html', item.innerHTML);

        // Start auto-scroll interval
        autoScrollInterval = setInterval(() => {
            if (window.lastDragEvent) {
                autoScroll(window.lastDragEvent);
            }
        }, 50);
    });

    item.addEventListener('dragend', (e) => {
        item.classList.remove('dragging');
        const allItems = container.querySelectorAll('.endpoint-item');
        allItems.forEach(i => i.classList.remove('drag-over'));
        draggedElement = null;
        draggedOverElement = null;
        draggedOriginalName = null;

        // Clear auto-scroll
        if (autoScrollInterval) {
            clearInterval(autoScrollInterval);
            autoScrollInterval = null;
        }
        window.lastDragEvent = null;
    });

    item.addEventListener('dragover', (e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        window.lastDragEvent = e; // Store for auto-scroll

        if (draggedElement && draggedElement !== item) {
            if (draggedOverElement && draggedOverElement !== item) {
                draggedOverElement.classList.remove('drag-over');
            }
            item.classList.add('drag-over');
            draggedOverElement = item;
        }
    });

    item.addEventListener('dragleave', (e) => {
        // Only remove if we're actually leaving the element
        if (!item.contains(e.relatedTarget)) {
            item.classList.remove('drag-over');
            if (draggedOverElement === item) {
                draggedOverElement = null;
            }
        }
    });

    item.addEventListener('drop', async (e) => {
        e.preventDefault();
        e.stopPropagation();

        if (draggedElement && draggedElement !== item) {
            // Use dataset.name to identify positions, not DOM order
            const draggedName = draggedElement.dataset.name;
            const targetName = item.dataset.name;

            // Get all items and build current order by name
            const allItems = Array.from(container.querySelectorAll('.endpoint-item'));
            const currentOrder = allItems.map(el => el.dataset.name);

            // Find positions by name (stable, not affected by scrolling)
            const fromIndex = currentOrder.indexOf(draggedName);
            const toIndex = currentOrder.indexOf(targetName);

            // Calculate new order
            const newOrder = [...currentOrder];
            newOrder.splice(fromIndex, 1);
            newOrder.splice(toIndex, 0, draggedName);

            // Compare arrays: if order hasn't changed, don't do anything
            const orderChanged = !currentOrder.every((name, idx) => name === newOrder[idx]);

            if (!orderChanged) {
                item.classList.remove('drag-over');
                return;
            }

            // Save to backend
            try {
                await window.go.main.App.ReorderEndpoints(newOrder);
                window.loadConfig();
            } catch (error) {
                console.error('Failed to reorder endpoints:', error);
                alert(t('endpoints.reorderFailed') + ': ' + error);
                window.loadConfig();
            }
        }

        item.classList.remove('drag-over');
    });
}

// 初始化端点成功事件监听
export function initEndpointSuccessListener() {
    if (window.runtime && window.runtime.EventsOn) {
        window.runtime.EventsOn('endpoint:success', (endpointName) => {
            // 更新测试状态为成功
            saveEndpointTestStatus(endpointName, true);
            // 刷新端点列表显示
            if (window.loadConfig) {
                window.loadConfig();
            }
        });
    }
}

// 清除所有端点测试状态
export function clearAllEndpointTestStatus() {
    try {
        localStorage.removeItem(ENDPOINT_TEST_STATUS_KEY);
    } catch (error) {
        console.error('Failed to clear endpoint test status:', error);
    }
}

// 启动时零消耗检测所有端点
export async function checkAllEndpointsOnStartup() {
    try {
        // 先清除所有状态
        clearAllEndpointTestStatus();

        const results = await testAllEndpointsZeroCost();
        for (const [name, status] of Object.entries(results)) {
            if (status === 'ok') {
                saveEndpointTestStatus(name, true);
            } else if (status === 'invalid_key') {
                saveEndpointTestStatus(name, false);
            }
            // 'unknown' 保持未设置状态，显示 ⚠️
        }
        // 刷新端点列表显示
        if (window.loadConfig) {
            window.loadConfig();
        }
    } catch (error) {
        console.error('Failed to check endpoints on startup:', error);
    }
}

// 渲染简洁视图
function renderCompactView(sortedEndpoints, container, currentEndpointName, isFiltered) {
    sortedEndpoints.forEach(({ endpoint: ep, originalIndex: index, stats }) => {
        const enabled = ep.enabled !== undefined ? ep.enabled : true;
        const transformer = ep.transformer || 'claude';
        const model = ep.model || '';
        const authMode = ep.authMode || 'api_key';
        const isCurrentEndpoint = ep.name === currentEndpointName;

        // 获取测试状态
        const testStatus = getEndpointTestStatus(ep.name);
        let testStatusIcon = '⚠️';
        let testStatusTip = t('endpoints.testTipUnknown');
        if (testStatus === true) {
            testStatusIcon = '✅';
            testStatusTip = t('endpoints.testTipSuccess');
        } else if (testStatus === false) {
            testStatusIcon = '❌';
            testStatusTip = t('endpoints.testTipFailed');
        }

        const item = document.createElement('div');
        item.className = 'endpoint-item-compact';
        item.dataset.name = ep.name;
        item.dataset.index = index;

        // 筛选激活时禁用拖拽
        if (isFiltered) {
            item.draggable = false;
            item.style.cursor = 'default';
            item.title = t('endpoints.dragDisabledDuringFilter');
        } else {
            item.draggable = true;
            setupCompactDragAndDrop(item, container);
        }

        // 截断 URL 显示
        const displayUrl = ep.apiUrl.length > 40 ? ep.apiUrl.substring(0, 40) + '...' : ep.apiUrl;

        // 构建统计详情提示
        const totalTokens = stats.inputTokens + stats.outputTokens;
        let statsTooltip = `${t('endpoints.requests')}: ${stats.requests} | ${t('endpoints.errors')}: ${stats.errors}\n${t('statistics.in')}: ${formatTokens(stats.inputTokens)} | ${t('statistics.out')}: ${formatTokens(stats.outputTokens)}`;
        if (model) {
            statsTooltip += `\n${t('modal.model')}: ${model}`;
        }
        if (ep.remark) {
            statsTooltip += `\n${t('modal.remark')}: ${ep.remark}`;
        }

        item.innerHTML = `
            <div class="drag-handle" title="${t('endpoints.dragToReorder')}">
                <div class="drag-handle-dots"><span></span><span></span></div>
                <div class="drag-handle-dots"><span></span><span></span></div>
                <div class="drag-handle-dots"><span></span><span></span></div>
            </div>
            <span class="compact-status" title="${testStatusTip}" style="cursor: help">${testStatusIcon}</span>
            <span class="compact-name" title="${ep.name}">${ep.name}</span>
            ${isCurrentEndpoint ? '<span class="btn btn-primary compact-badge-btn">' + t('endpoints.current') + '</span>' : (enabled ? '<button class="btn btn-primary compact-badge-btn" data-action="switch" data-name="' + ep.name + '">' + t('endpoints.switchTo') + '</button>' : '<span class="btn btn-primary compact-badge-btn compact-badge-disabled">' + t('endpoints.disabled') + '</span>')}
            <span class="compact-url" title="${ep.apiUrl}"><span class="compact-url-icon">🌐</span>${displayUrl}</span>
            <span class="compact-transformer">🔄 ${transformer}</span>
            <span class="compact-stats" title="${statsTooltip}">📊 ${stats.requests} | 🎯 ${formatTokens(stats.inputTokens + stats.outputTokens)}</span>
            <div class="compact-actions">
                <label class="toggle-switch">
                    <input type="checkbox" data-index="${index}" ${enabled ? 'checked' : ''}>
                    <span class="toggle-slider"></span>
                </label>
                <div class="compact-more-dropdown">
                    <button class="compact-btn" data-action="more" title="${t('endpoints.moreActions')}">⋯</button>
                    <div class="compact-more-menu">
                        <button data-action="test" data-index="${index}">🧪 ${t('endpoints.test')}</button>
                        <button data-action="edit" data-index="${index}">✏️ ${t('endpoints.edit')}</button>
                        <button data-action="copy" data-index="${index}">📋 ${t('endpoints.copy')}</button>
                        <button data-action="delete" data-index="${index}" class="danger">🗑️ ${t('endpoints.delete')}</button>
                    </div>
                </div>
            </div>
        `;

        // 绑定事件
        bindCompactItemEvents(item, index, enabled);

        // 设置拖拽
        setupCompactDragAndDrop(item, container);

        container.appendChild(item);
    });

    // 点击其他地方关闭下拉菜单（先移除旧监听器，避免重复绑定）
    document.removeEventListener('click', closeAllDropdowns);
    document.addEventListener('click', closeAllDropdowns);
}

// 绑定简洁视图项目事件
function bindCompactItemEvents(item, index, enabled) {
    const toggleSwitch = item.querySelector('input[type="checkbox"]');
    const switchBtn = item.querySelector('[data-action="switch"]');
    const moreBtn = item.querySelector('[data-action="more"]');
    const moreMenu = item.querySelector('.compact-more-menu');
    const testBtn = item.querySelector('[data-action="test"]');
    const editBtn = item.querySelector('[data-action="edit"]');
    const deleteBtn = item.querySelector('[data-action="delete"]');

    // 如果当前正在测试这个端点，显示加载状态
    if (currentTestIndex === index) {
        moreBtn.innerHTML = '⏳';
        moreBtn.disabled = true;
        currentTestButton = testBtn;
    }

    // 启用/禁用开关
    toggleSwitch.addEventListener('change', async (e) => {
        const idx = parseInt(e.target.getAttribute('data-index'));
        const newEnabled = e.target.checked;
        try {
            await toggleEndpoint(idx, newEnabled);
            window.loadConfig();
        } catch (error) {
            console.error('Failed to toggle endpoint:', error);
            alert('Failed to toggle endpoint: ' + error);
            e.target.checked = !newEnabled;
        }
    });

    // 切换按钮
    if (switchBtn) {
        switchBtn.addEventListener('click', async () => {
            const name = switchBtn.getAttribute('data-name');
            try {
                switchBtn.disabled = true;
                switchBtn.innerHTML = '⏳';
                await window.go.main.App.SwitchToEndpoint(name);
                window.loadConfig(); // Refresh display
            } catch (error) {
                console.error('Failed to switch endpoint:', error);
                alert(t('endpoints.switchFailed') + ': ' + error);
            } finally {
                if (switchBtn) {
                    switchBtn.disabled = false;
                    switchBtn.innerHTML = t('endpoints.switchTo');
                }
            }
        });
    }

    // 更多操作按钮
    moreBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        const isOpen = moreMenu.classList.contains('show');
        closeAllDropdowns();
        if (!isOpen) {
            moreMenu.classList.add('show');
        }
    });

    // 测试按钮
    testBtn.addEventListener('click', () => {
        closeAllDropdowns();
        const idx = parseInt(testBtn.getAttribute('data-index'));
        window.testEndpoint(idx, testBtn);
    });

    // 复制按钮
    const copyBtn = item.querySelector('[data-action="copy"]');
    copyBtn.addEventListener('click', () => {
        closeAllDropdowns();
        const idx = parseInt(copyBtn.getAttribute('data-index'));
        copyEndpointConfig(idx, copyBtn);
});

    // 编辑按钮
    editBtn.addEventListener('click', () => {
        closeAllDropdowns();
        const idx = parseInt(editBtn.getAttribute('data-index'));
        window.editEndpoint(idx);
    });

    // 删除按钮
    deleteBtn.addEventListener('click', () => {
        closeAllDropdowns();
        const idx = parseInt(deleteBtn.getAttribute('data-index'));
        window.deleteEndpoint(idx);
    });
}

// 关闭所有下拉菜单
function closeAllDropdowns() {
    document.querySelectorAll('.compact-more-menu.show').forEach(menu => {
        menu.classList.remove('show');
    });
}

// 检查是否有下拉菜单正在显示
export function isDropdownOpen() {
    return document.querySelectorAll('.compact-more-menu.show').length > 0;
}

// 拖拽占位符元素
let dragPlaceholder = null;
let draggedItemHeight = 0;

// 创建占位符（指示线）
function createPlaceholder() {
    const placeholder = document.createElement('div');
    placeholder.className = 'drag-placeholder';
    return placeholder;
}

// 更新其他元素的位置
function updateItemPositions(container, draggedElement, placeholder) {
    const allItems = Array.from(container.querySelectorAll('.endpoint-item-compact'));
    const draggedIndex = allItems.indexOf(draggedElement);

    // 计算占位符在端点元素中的目标索引
    let targetIndex = 0;
    let currentNode = placeholder.previousSibling;
    while (currentNode) {
        if (currentNode.classList && currentNode.classList.contains('endpoint-item-compact')) {
            targetIndex++;
        }
        currentNode = currentNode.previousSibling;
    }

    allItems.forEach((item, index) => {
        let offset = 0;

        if (item === draggedElement) {
            // 被拖拽元素视觉上移动到占位符位置
            offset = (targetIndex - draggedIndex) * (draggedItemHeight + 8);
        } else if (draggedIndex < targetIndex) {
            // 向下拖拽：draggedIndex 和 targetIndex 之间的元素向上移
            if (index > draggedIndex && index < targetIndex) {
                offset = -(draggedItemHeight + 8);
            }
        } else if (draggedIndex > targetIndex) {
            // 向上拖拽：targetIndex 和 draggedIndex 之间的元素向下移
            if (index >= targetIndex && index < draggedIndex) {
                offset = draggedItemHeight + 8;
            }
        }

        item.style.transform = offset !== 0 ? `translateY(${offset}px)` : '';
    });
}

// 根据鼠标位置移动占位符
function movePlaceholderByMousePosition(e, container, draggedElement, dragPlaceholder) {
    if (!draggedElement || !dragPlaceholder) return;

    const allItems = Array.from(container.querySelectorAll('.endpoint-item-compact'));
    const mouseY = e.clientY;

    // 找到最接近鼠标位置的元素
    let closestItem = null;
    let closestDistance = Infinity;
    let insertBefore = true;

    allItems.forEach(item => {
        if (item === draggedElement) return;

        const rect = item.getBoundingClientRect();
        const itemMiddle = rect.top + rect.height / 2;
        const distance = Math.abs(mouseY - itemMiddle);

        if (distance < closestDistance) {
            closestDistance = distance;
            closestItem = item;
            insertBefore = mouseY < itemMiddle;
        }
    });

    // 移动占位符
    if (closestItem) {
        const targetPosition = insertBefore ? closestItem : closestItem.nextSibling;
        if (targetPosition !== dragPlaceholder && targetPosition !== dragPlaceholder.nextSibling) {
            container.insertBefore(dragPlaceholder, targetPosition);
            updateItemPositions(container, draggedElement, dragPlaceholder);
        }
    } else if (allItems.length === 1) {
        // 只有一个元素（被拖拽的元素）
        if (dragPlaceholder.parentNode !== container) {
            container.appendChild(dragPlaceholder);
        }
    }
}

// 简洁视图的拖拽设置
function setupCompactDragAndDrop(item, container) {
    item.addEventListener('dragstart', (e) => {
        draggedElement = item;
        draggedOriginalName = item.dataset.name;
        draggedItemHeight = item.offsetHeight;
        item.classList.add('dragging');
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/html', item.innerHTML);

        // 创建并插入占位符（指示线）
        dragPlaceholder = createPlaceholder();
        item.parentNode.insertBefore(dragPlaceholder, item.nextSibling);

        // 在容器上添加事件监听
        container.addEventListener('dragover', handleContainerDragOver);
        container.addEventListener('drop', handleContainerDrop);

        autoScrollInterval = setInterval(() => {
            if (window.lastDragEvent) {
                autoScroll(window.lastDragEvent);
            }
        }, 50);
    });

    item.addEventListener('dragend', () => {
        item.classList.remove('dragging');
        const allItems = container.querySelectorAll('.endpoint-item-compact');
        allItems.forEach(i => {
            i.classList.remove('drag-over');
            i.style.transform = '';
        });

        // 清理容器的 cursor 样式
        container.style.cursor = '';

        // 移除容器的事件监听
        container.removeEventListener('dragover', handleContainerDragOver);
        container.removeEventListener('drop', handleContainerDrop);

        // 移除占位符
        if (dragPlaceholder && dragPlaceholder.parentNode) {
            dragPlaceholder.parentNode.removeChild(dragPlaceholder);
            dragPlaceholder = null;
        }

        draggedElement = null;
        draggedOverElement = null;
        draggedOriginalName = null;
        draggedItemHeight = 0;

        if (autoScrollInterval) {
            clearInterval(autoScrollInterval);
            autoScrollInterval = null;
        }
        window.lastDragEvent = null;
    });

    // 在端点元素上禁止 drop（但允许事件冒泡到容器，让占位符能正常移动）
    item.addEventListener('dragover', (e) => {
        e.preventDefault();
        // 移除 stopPropagation()，让事件冒泡到容器
        e.dataTransfer.dropEffect = 'none';
    });
}

// 容器的 dragover 处理函数
function handleContainerDragOver(e) {
    e.preventDefault();
    window.lastDragEvent = e;

    const container = e.currentTarget;

    // 检查鼠标是否在端点元素上
    const isOverEndpointItem = e.target.closest('.endpoint-item-compact');

    if (isOverEndpointItem) {
        // 在端点元素上：显示禁止图标，但仍然移动占位符
        e.dataTransfer.dropEffect = 'none';
        container.style.cursor = 'no-drop';
    } else {
        // 在空白区域或占位符上：显示允许图标
        e.dataTransfer.dropEffect = 'move';
        container.style.cursor = 'grabbing';
    }

    // 始终更新占位符位置，让其他元素自动移开
    movePlaceholderByMousePosition(e, container, draggedElement, dragPlaceholder);
}

// 容器的 drop 处理函数
async function handleContainerDrop(e) {
    if (e.target.closest('.endpoint-item-compact')) {
        return;
    }
    e.preventDefault();
    e.stopPropagation();

    const container = e.currentTarget;
    if (draggedElement && dragPlaceholder) {
        const draggedName = draggedElement.dataset.name;
        const allItems = Array.from(container.querySelectorAll('.endpoint-item-compact'));
        const currentOrder = allItems.map(el => el.dataset.name);
        const allChildren = Array.from(container.children);
        const placeholderIndex = allChildren.indexOf(dragPlaceholder);

        let targetIndex = 0;
        for (let i = 0; i < placeholderIndex; i++) {
            if (allChildren[i].classList.contains('endpoint-item-compact')) {
                targetIndex++;
            }
        }

        const draggedIndex = currentOrder.indexOf(draggedName);
        if (draggedIndex < targetIndex) {
            targetIndex--;
        }

        const newOrder = [...currentOrder];
        newOrder.splice(draggedIndex, 1);
        newOrder.splice(targetIndex, 0, draggedName);

        const orderChanged = !currentOrder.every((name, idx) => name === newOrder[idx]);
        if (!orderChanged) return;

        try {
            await window.go.main.App.ReorderEndpoints(newOrder);
            window.loadConfig();
        } catch (error) {
            console.error('Failed to reorder endpoints:', error);
            alert(t('endpoints.reorderFailed') + ': ' + error);
            window.loadConfig();
        }
    }
}

// Incremental endpoint stats update - updates only the numbers in the endpoint card without re-rendering
export function updateEndpointStatsIncremental(endpointName, data) {
    // Find endpoint card by name (works for both detail and compact views)
    const endpointCard = document.querySelector(`[data-name="${endpointName}"]`);
    if (!endpointCard) {
        return; // Endpoint not found or filtered out
    }

    const totalTokens = (data.inputTokens || 0) + (data.outputTokens || 0);

    // Update stats in detail view
    const paragraphs = endpointCard.querySelectorAll('p');
    for (const p of paragraphs) {
        const text = p.textContent;

        // Update requests/errors line
        if (text.includes('📊') && text.includes(t('endpoints.requests'))) {
            p.innerHTML = `📊 ${t('endpoints.requests')}: ${data.requests} | ${t('endpoints.errors')}: ${data.errors}`;
        }

        // Update tokens line
        if (text.includes('🎯') && text.includes(t('endpoints.tokens'))) {
            p.innerHTML = `🎯 ${t('endpoints.tokens')}: ${formatTokens(totalTokens)} (${t('statistics.in')}: ${formatTokens(data.inputTokens)}, ${t('statistics.out')}: ${formatTokens(data.outputTokens)})`;
        }
    }

    // Update stats in compact view
    const compactStats = endpointCard.querySelector('.compact-stats');
    if (compactStats) {
        compactStats.textContent = `📊 ${data.requests} | 🎯 ${formatTokens(totalTokens)}`;

        // Update tooltip
        const tooltip = `${t('endpoints.requests')}: ${data.requests} | ${t('endpoints.errors')}: ${data.errors}\n${t('statistics.in')}: ${formatTokens(data.inputTokens)} | ${t('statistics.out')}: ${formatTokens(data.outputTokens)}`;
        compactStats.title = tooltip;
    }
}