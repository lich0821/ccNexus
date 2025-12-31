import { t } from '../i18n/index.js';
import { escapeHtml } from '../utils/format.js';
import { addEndpoint, updateEndpoint, removeEndpoint, testEndpoint, testEndpointLight, updatePort } from './config.js';
import { setTestState, clearTestState, saveEndpointTestStatus } from './endpoints.js';

let currentEditIndex = -1;

// Show error toast
function showError(message) {
    const toast = document.getElementById('errorToast');
    const messageEl = document.getElementById('errorToastMessage');

    messageEl.textContent = message;
    toast.classList.add('show');

    setTimeout(() => {
        toast.classList.remove('show');
    }, 3000);
}

// Show notification
export function showNotification(message, type = 'info') {
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

// Confirm dialog
let confirmResolve = null;

export function showConfirm(message) {
    return new Promise((resolve) => {
        confirmResolve = resolve;
        document.getElementById('confirmMessage').textContent = message;
        document.getElementById('confirmDialog').classList.add('active');
    });
}

export function acceptConfirm() {
    document.getElementById('confirmDialog').classList.remove('active');
    if (confirmResolve) {
        confirmResolve(true);
        confirmResolve = null;
    }
}

export function cancelConfirm() {
    document.getElementById('confirmDialog').classList.remove('active');
    if (confirmResolve) {
        confirmResolve(false);
        confirmResolve = null;
    }
}

// Close action dialog
export function showCloseActionDialog() {
    document.getElementById('closeActionDialog').classList.add('active');
}

export function quitApplication() {
    document.getElementById('closeActionDialog').classList.remove('active');
    window.go.main.App.Quit();
}

export function minimizeToTray() {
    document.getElementById('closeActionDialog').classList.remove('active');
    window.go.main.App.HideWindow();
}

// Toggle password visibility
export function togglePasswordVisibility() {
    const input = document.getElementById('endpointKey');
    const icon = document.getElementById('eyeIcon');

    if (input.type === 'password') {
        input.type = 'text';
        icon.innerHTML = '<path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path><line x1="1" y1="1" x2="23" y2="23"></line>';
    } else {
        input.type = 'password';
        icon.innerHTML = '<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle>';
    }
}

// Endpoint Modal
export function showAddEndpointModal() {
    currentEditIndex = -1;
    document.getElementById('modalTitle').textContent = '➕ ' + t('modal.addEndpoint');
    document.getElementById('endpointName').value = '';
    document.getElementById('endpointUrl').value = '';
    document.getElementById('endpointKey').value = '';
    document.getElementById('endpointTransformer').value = 'claude';
    document.getElementById('endpointModel').value = '';
    document.getElementById('endpointRemark').value = '';
    handleTransformerChange();
    document.getElementById('endpointModal').classList.add('active');
}

export async function editEndpoint(index) {
    currentEditIndex = index;
    const configStr = await window.go.main.App.GetConfig();
    const config = JSON.parse(configStr);
    const ep = config.endpoints[index];

    document.getElementById('modalTitle').textContent = '✏️ ' + t('modal.editEndpoint');
    document.getElementById('endpointName').value = ep.name;
    document.getElementById('endpointUrl').value = ep.apiUrl;
    document.getElementById('endpointKey').value = ep.apiKey;
    document.getElementById('endpointTransformer').value = ep.transformer || 'claude';
    document.getElementById('endpointModel').value = ep.model || '';
    document.getElementById('endpointRemark').value = ep.remark || '';

    handleTransformerChange();
    document.getElementById('endpointModal').classList.add('active');
}

export async function saveEndpoint() {
    const name = document.getElementById('endpointName').value.trim();
    const url = document.getElementById('endpointUrl').value.trim();
    const key = document.getElementById('endpointKey').value.trim();
    const transformer = document.getElementById('endpointTransformer').value;
    const model = document.getElementById('endpointModel').value.trim();
    const remark = document.getElementById('endpointRemark').value.trim();

    if (!name || !url || !key) {
        showError(t('modal.requiredFields'));
        return;
    }

    if (transformer !== 'claude' && !model) {
        showError(t('modal.modelRequired').replace('{transformer}', transformer));
        return;
    }

    // Check for duplicate endpoint name
    const configStr = await window.go.main.App.GetConfig();
    const config = JSON.parse(configStr);
    const existingEndpoint = config.endpoints.find((ep, idx) =>
        ep.name === name && idx !== currentEditIndex
    );

    if (existingEndpoint) {
        showError(`Endpoint name "${name}" already exists. Please use a different name.`);
        return;
    }

    try {
        if (currentEditIndex === -1) {
            await addEndpoint(name, url, key, transformer, model, remark);
        } else {
            await updateEndpoint(currentEditIndex, name, url, key, transformer, model, remark);
        }

        closeModal();
        window.loadConfig();
    } catch (error) {
        showError(t('modal.saveFailed').replace('{error}', error));
    }
}

export async function deleteEndpoint(index) {
    try {
        const config = await window.go.main.App.GetConfig();
        const endpoints = JSON.parse(config).endpoints;
        const endpointName = endpoints[index].name;

        const confirmed = await showConfirm(t('modal.confirmDelete').replace('{name}', endpointName));
        if (!confirmed) {
            return;
        }

        await removeEndpoint(index);
        window.loadConfig();
    } catch (error) {
        console.error('Delete failed:', error);
        showError(t('modal.deleteFailed').replace('{error}', error));
    }
}

export function closeModal() {
    document.getElementById('endpointModal').classList.remove('active');
}

export function handleTransformerChange() {
    const transformer = document.getElementById('endpointTransformer').value;
    const modelRequired = document.getElementById('modelRequired');
    const modelInput = document.getElementById('endpointModel');
    const modelHelpText = document.getElementById('modelHelpText');

    // Clear fetched models when transformer changes
    clearFetchedModels();

    if (transformer === 'claude') {
        modelRequired.style.display = 'none';
        modelInput.placeholder = 'e.g., claude-3-5-sonnet-20241022';
        modelHelpText.textContent = t('modal.modelHelpClaude');
    } else if (transformer === 'openai') {
        modelRequired.style.display = 'inline';
        modelInput.placeholder = 'e.g., gpt-4-turbo';
        modelHelpText.textContent = t('modal.modelHelpOpenAI');
    } else if (transformer === 'openai2') {
        modelRequired.style.display = 'inline';
        modelInput.placeholder = 'e.g., gpt-4.1';
        modelHelpText.textContent = t('modal.modelHelpOpenAI2');
    } else if (transformer === 'gemini') {
        modelRequired.style.display = 'inline';
        modelInput.placeholder = 'e.g., gemini-pro';
        modelHelpText.textContent = t('modal.modelHelpGemini');
    }
}

// Store fetched models for filtering
let fetchedModels = [];

// Fetch models from API
export async function fetchModels() {
    const apiUrl = document.getElementById('endpointUrl').value.trim();
    const apiKey = document.getElementById('endpointKey').value.trim();
    const transformer = document.getElementById('endpointTransformer').value;
    const fetchBtn = document.getElementById('fetchModelsBtn');
    const fetchIcon = document.getElementById('fetchModelsIcon');
    const modelInput = document.getElementById('endpointModel');
    const dropdown = document.getElementById('modelDropdown');

    // Validate inputs
    if (!apiUrl) {
        showNotification(t('modal.fetchModelsNoUrl'), 'error');
        return;
    }
    if (!apiKey) {
        showNotification(t('modal.fetchModelsNoKey'), 'error');
        return;
    }

    // Show loading state
    fetchBtn.disabled = true;
    fetchIcon.textContent = '⏳';

    try {
        const resultStr = await window.go.main.App.FetchModels(apiUrl, apiKey, transformer);
        const result = JSON.parse(resultStr);

        if (result.success && result.models && result.models.length > 0) {
            fetchedModels = result.models;
            renderModelDropdown(fetchedModels, dropdown, modelInput);
            dropdown.classList.add('show');

            showNotification(t('modal.fetchModelsSuccess').replace('{count}', result.models.length), 'success');
        } else {
            const msg = result.message?.includes('no_models_found') ? t('modal.fetchModelsEmpty') : t('modal.fetchModelsFailed');
            showNotification(msg, 'error');
        }
    } catch (error) {
        console.error('Failed to fetch models:', error);
        showNotification(t('modal.fetchModelsFailed') + ': ' + error, 'error');
    } finally {
        fetchBtn.disabled = false;
        fetchIcon.textContent = t('modal.fetchModelsBtn');
    }
}

// Render model dropdown
function renderModelDropdown(models, dropdown, input) {
    dropdown.innerHTML = '';
    models.forEach(model => {
        const item = document.createElement('div');
        item.className = 'model-dropdown-item';
        item.textContent = model;
        item.onclick = () => {
            input.value = model;
            dropdown.classList.remove('show');
        };
        dropdown.appendChild(item);
    });

}


// Initialize model input events
export function initModelInputEvents() {
    const modelInput = document.getElementById('endpointModel');
    const dropdown = document.getElementById('modelDropdown');
    if (!modelInput || !dropdown) return;

    // Show dropdown on focus if has models
    modelInput.addEventListener('focus', () => {
        if (fetchedModels.length > 0) {
            renderModelDropdown(fetchedModels, dropdown, modelInput);
            dropdown.classList.add('show');
        }
    });

    // Hide dropdown on click outside
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.model-select-container')) {
            dropdown.classList.remove('show');
        }
    });

}

// Toggle model dropdown
export function toggleModelDropdown() {
    const dropdown = document.getElementById('modelDropdown');
    const modelInput = document.getElementById('endpointModel');
    if (!dropdown || fetchedModels.length === 0) return;

    if (dropdown.classList.contains('show')) {
        dropdown.classList.remove('show');
    } else {
        renderModelDropdown(fetchedModels, dropdown, modelInput);
        dropdown.classList.add('show');
    }
}

// Clear fetched models (call when transformer changes)
export function clearFetchedModels() {
    fetchedModels = [];
    const dropdown = document.getElementById('modelDropdown');
    if (dropdown) {
        dropdown.innerHTML = '';
        dropdown.classList.remove('show');
    }
}

// Port Modal
export async function showEditPortModal() {
    const configStr = await window.go.main.App.GetConfig();
    const config = JSON.parse(configStr);

    document.getElementById('portInput').value = config.port;
    document.getElementById('portModal').classList.add('active');
}

export async function savePort() {
    const port = parseInt(document.getElementById('portInput').value);

    if (!port || port < 1 || port > 65535) {
        showNotification(t('modal.portInvalid'), 'error');
        return;
    }

    try {
        await updatePort(port);
        closePortModal();
        window.loadConfig();
        showNotification(t('modal.portUpdateSuccess'), 'success');
    } catch (error) {
        showNotification(t('modal.portUpdateFailed').replace('{error}', error), 'error');
    }
}

export function closePortModal() {
    document.getElementById('portModal').classList.remove('active');
}


// ========== 加群二维码URL配置 ==========
// 上传到图床后填写URL，过期时直接替换图床文件即可自动更新
const CHAT_QRCODE_URL = 'https://gitee.com/hea7en/images/raw/master/group/chat.png';

// 添加时间戳破除缓存
function getQRCodeUrlWithTimestamp(url) {
    const ts = new Date().getTime();
    return url + (url.includes('?') ? '&' : '?') + 't=' + ts;
}
// ===================================

// Welcome Modal
export async function showWelcomeModal() {
    document.getElementById('welcomeModal').classList.add('active');

    try {
        const version = await window.go.main.App.GetVersion();
        document.querySelector('#welcomeModal .modal-header h2').textContent = t('welcome.titleWithVersion').replace('{version}', version);
    } catch (error) {
        console.error('Failed to load version:', error);
    }

    // 通过 Go 后端获取加群二维码图片（绕过 CORS 限制，添加时间戳破除缓存）
    try {
        const urlWithTimestamp = getQRCodeUrlWithTimestamp(CHAT_QRCODE_URL);
        const base64Data = await window.go.main.App.FetchImageAsBase64(urlWithTimestamp);
        if (base64Data) {
            const img = document.getElementById('chatQRCodeImg');
            const tip = document.getElementById('chatQRCodeTip');
            if (img) {
                img.src = base64Data;
            }
            if (tip) {
                tip.textContent = t('welcome.chatGroupTip');
            }
        }
    } catch (error) {
        console.error('Failed to load chat QR code:', error);
    }
}

export function closeWelcomeModal() {
    const dontShowAgain = document.getElementById('dontShowAgain').checked;
    if (dontShowAgain) {
        localStorage.setItem('ccNexus_welcomeShown', 'true');
    }
    document.getElementById('welcomeModal').classList.remove('active');
}

// Changelog Modal
export async function showChangelogModal() {
    const modal = document.getElementById('changelogModal');
    const content = document.getElementById('changelogContent');
    if (!modal || !content) return;

    content.innerHTML = `<p>${t('changelog.loading')}</p>`;
    modal.classList.add('active');

    try {
        const lang = await window.go.main.App.GetLanguage();
        const changelogJson = await window.go.main.App.GetChangelog(lang);
        const changelog = JSON.parse(changelogJson);

        let html = '<div class="changelog-timeline">';
        changelog.forEach((item, index) => {
            const position = index % 2 === 0 ? 'left' : 'right';
            html += `
                <div class="timeline-item ${position}">
                    <div class="timeline-dot"></div>
                    <div class="timeline-content">
                        <div class="timeline-header">
                            <span class="timeline-version">${item.version}</span>
                            <span class="timeline-date">${item.date}</span>
                        </div>
                        <ul class="timeline-changes">
                            ${item.changes.map(c => `<li>${c}</li>`).join('')}
                        </ul>
                    </div>
                </div>
            `;
        });
        html += '</div>';
        content.innerHTML = html;
    } catch (error) {
        console.error('Failed to load changelog:', error);
        content.innerHTML = `<p style="color: #e74c3c;">${t('changelog.error')}</p>`;
    }
}

export function closeChangelogModal() {
    document.getElementById('changelogModal').classList.remove('active');
}

export async function showChangelogIfNewVersion() {
    const currentVersion = await window.go.main.App.GetVersion();
    const lastVersion = localStorage.getItem('ccNexus_lastVersion');

    if (lastVersion && lastVersion !== currentVersion) {
        setTimeout(() => showChangelogModal(), 600);
    }
    localStorage.setItem('ccNexus_lastVersion', currentVersion);
}

export function showWelcomeModalIfFirstTime() {
    const hasShown = localStorage.getItem('ccNexus_welcomeShown');
    if (!hasShown) {
        setTimeout(() => {
            showWelcomeModal();
        }, 500);
    }
}

// 判断是否为"不支持测试"的情况
function isTestNotSupported(statusCode, message) {
    // 可能不支持测试的 HTTP 状态码
    const notSupportedCodes = [404, 405, 501];
    // 认证错误关键词（如果包含这些，说明是真正的错误，不是不支持测试）
    const authErrorKeywords = ['unauthorized', 'invalid key', 'invalid_api_key', 'authentication', 'api key', 'api_key', 'forbidden', 'permission', 'access denied'];

    if (notSupportedCodes.includes(statusCode)) {
        const lowerMsg = (message || '').toLowerCase();
        // 排除明显的认证错误
        const isAuthError = authErrorKeywords.some(kw => lowerMsg.includes(kw));
        return !isAuthError;
    }
    return false;
}

// Test Result Modal
export async function testEndpointHandler(index, buttonElement) {
    setTestState(buttonElement, index);

    // 获取端点名称用于保存测试状态（兼容详情视图和简洁视图）
    const endpointItem = buttonElement.closest('.endpoint-item') || buttonElement.closest('.endpoint-item-compact');
    const endpointName = endpointItem ? endpointItem.dataset.name : null;

    // 简洁视图：同时更新 moreBtn
    const moreBtn = endpointItem ? endpointItem.querySelector('[data-action="more"]') : null;
    if (moreBtn) {
        moreBtn.disabled = true;
        moreBtn.innerHTML = '⏳';
    }

    try {
        buttonElement.disabled = true;
        buttonElement.innerHTML = '⏳';

        // 使用轻量级测试（优先零消耗方法）
        const result = await testEndpointLight(index);

        const resultContent = document.getElementById('testResultContent');
        const resultTitle = document.getElementById('testResultTitle');

        if (result.success) {
            resultTitle.innerHTML = t('test.successTitle');
            resultContent.innerHTML = `
                <div style="padding: 15px; background: #d4edda; border: 1px solid #c3e6cb; border-radius: 5px; margin-bottom: 15px;">
                    <strong style="color: #155724;">${t('test.connectionSuccess')}</strong>
                </div>
                <div style="padding: 15px; background: #f8f9fa; border-radius: 5px; font-family: monospace; white-space: pre-line; word-break: break-all;">${escapeHtml(result.message)} (${result.method})</div>
            `;
            // 保存测试成功状态
            if (endpointName) {
                saveEndpointTestStatus(endpointName, true);
            }
        } else if (result.status === 'unknown') {
            // 无法确定状态（如三方站限制测试）
            showNotification(t('test.notSupportedMessage'), 'warning');
            // 保存为未知状态
            if (endpointName) {
                saveEndpointTestStatus(endpointName, 'unknown');
            }
            // 清除测试状态，恢复按钮
            clearTestState();
            // 刷新端点列表以更新图标
            if (window.loadConfig) {
                window.loadConfig();
            }
            return; // 不显示测试结果弹窗
        } else {
            resultTitle.innerHTML = t('test.failedTitle');
            resultContent.innerHTML = `
                <div style="padding: 15px; background: #f8d7da; border: 1px solid #f5c6cb; border-radius: 5px; margin-bottom: 15px;">
                    <strong style="color: #721c24;">${t('test.connectionFailed')}</strong>
                </div>
                <div style="padding: 15px; background: #f8f9fa; border-radius: 5px; font-family: monospace; white-space: pre-line; word-break: break-all;"><strong>Error:</strong><br>${escapeHtml(result.message)}</div>
            `;
            // 保存测试失败状态
            if (endpointName) {
                saveEndpointTestStatus(endpointName, false);
            }
        }

        document.getElementById('testResultModal').classList.add('active');
        // 刷新端点列表以更新图标
        if (window.loadConfig) {
            window.loadConfig();
        }

    } catch (error) {
        console.error('Test failed:', error);

        const resultContent = document.getElementById('testResultContent');
        const resultTitle = document.getElementById('testResultTitle');

        resultTitle.innerHTML = t('test.failedTitle');
        resultContent.innerHTML = `
            <div style="padding: 15px; background: #f8d7da; border: 1px solid #f5c6cb; border-radius: 5px; margin-bottom: 15px;">
                <strong style="color: #721c24;">${t('test.testError')}</strong>
            </div>
            <div style="padding: 15px; background: #f8f9fa; border-radius: 5px; font-family: monospace; white-space: pre-line;">${escapeHtml(error.toString())}</div>
        `;

        // 保存测试失败状态（异常情况）
        if (endpointName) {
            saveEndpointTestStatus(endpointName, false);
        }

        document.getElementById('testResultModal').classList.add('active');
        // 刷新端点列表以更新图标
        if (window.loadConfig) {
            window.loadConfig();
        }
    }
}

export function closeTestResultModal() {
    document.getElementById('testResultModal').classList.remove('active');
    clearTestState();
}

// External URLs
export function openGitHub() {
    if (window.go?.main?.App) {
        window.go.main.App.OpenURL('https://github.com/lich0821/ccNexus');
    }
}

export function openArticle() {
    if (window.go?.main?.App) {
        window.go.main.App.OpenURL('https://mp.weixin.qq.com/s/ohtkyIMd5YC7So1q-gE0og');
    }
}
