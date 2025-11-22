import { t } from '../i18n/index.js';
import { escapeHtml } from '../utils/format.js';
import { addEndpoint, updateEndpoint, removeEndpoint, testEndpoint, updatePort } from './config.js';
import { setTestState, clearTestState } from './endpoints.js';

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

// Confirm dialog
let confirmResolve = null;

function showConfirm(message) {
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

export async function quitApplication() {
    document.getElementById('closeActionDialog').classList.remove('active');
    // Save user preference ("quit" = quit directly)
    try {
        await window.go.main.App.SetCloseWindowBehavior('quit');
    } catch (error) {
        console.error('Failed to save close window behavior:', error);
    }
    window.go.main.App.Quit();
}

export async function minimizeToTray() {
    document.getElementById('closeActionDialog').classList.remove('active');
    // Save user preference ("minimize" = minimize to tray)
    try {
        await window.go.main.App.SetCloseWindowBehavior('minimize');
    } catch (error) {
        console.error('Failed to save close window behavior:', error);
    }
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

    if (transformer === 'claude') {
        modelRequired.style.display = 'none';
        modelInput.placeholder = 'e.g., claude-3-5-sonnet-20241022';
        modelHelpText.textContent = t('modal.modelHelpClaude');
    } else if (transformer === 'openai') {
        modelRequired.style.display = 'inline';
        modelInput.placeholder = 'e.g., gpt-4-turbo';
        modelHelpText.textContent = t('modal.modelHelpOpenAI');
    } else if (transformer === 'gemini') {
        modelRequired.style.display = 'inline';
        modelInput.placeholder = 'e.g., gemini-pro';
        modelHelpText.textContent = t('modal.modelHelpGemini');
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
        alert('Please enter a valid port number (1-65535)');
        return;
    }

    try {
        await updatePort(port);
        closePortModal();
        window.loadConfig();
        alert('Port updated successfully! Please restart the application for changes to take effect.');
    } catch (error) {
        alert('Failed to update port: ' + error);
    }
}

export function closePortModal() {
    document.getElementById('portModal').classList.remove('active');
}

// Welcome Modal
export async function showWelcomeModal() {
    document.getElementById('welcomeModal').classList.add('active');

    try {
        const version = await window.go.main.App.GetVersion();
        document.querySelector('#welcomeModal .modal-header h2').textContent = t('welcome.titleWithVersion').replace('{version}', version);
    } catch (error) {
        console.error('Failed to load version:', error);
    }
}

export function closeWelcomeModal() {
    const dontShowAgain = document.getElementById('dontShowAgain').checked;
    if (dontShowAgain) {
        localStorage.setItem('ccNexus_welcomeShown', 'true');
    }
    document.getElementById('welcomeModal').classList.remove('active');
}

export function showWelcomeModalIfFirstTime() {
    const hasShown = localStorage.getItem('ccNexus_welcomeShown');
    if (!hasShown) {
        setTimeout(() => {
            showWelcomeModal();
        }, 500);
    }
}

// Test Result Modal
export async function testEndpointHandler(index, buttonElement) {
    setTestState(buttonElement, index);

    try {
        buttonElement.disabled = true;
        buttonElement.innerHTML = '⏳';

        const result = await testEndpoint(index);

        const resultContent = document.getElementById('testResultContent');
        const resultTitle = document.getElementById('testResultTitle');

        if (result.success) {
            resultTitle.innerHTML = '✅ Test Successful';
            resultContent.innerHTML = `
                <div style="padding: 15px; background: #d4edda; border: 1px solid #c3e6cb; border-radius: 5px; margin-bottom: 15px;">
                    <strong style="color: #155724;">Connection successful!</strong>
                </div>
                <div style="padding: 15px; background: #f8f9fa; border-radius: 5px; font-family: monospace; white-space: pre-line; word-break: break-all;">${escapeHtml(result.message)}</div>
            `;
        } else {
            resultTitle.innerHTML = '❌ Test Failed';
            resultContent.innerHTML = `
                <div style="padding: 15px; background: #f8d7da; border: 1px solid #f5c6cb; border-radius: 5px; margin-bottom: 15px;">
                    <strong style="color: #721c24;">Connection failed</strong>
                </div>
                <div style="padding: 15px; background: #f8f9fa; border-radius: 5px; font-family: monospace; white-space: pre-line; word-break: break-all;"><strong>Error:</strong><br>${escapeHtml(result.message)}</div>
            `;
        }

        document.getElementById('testResultModal').classList.add('active');

    } catch (error) {
        console.error('Test failed:', error);

        const resultContent = document.getElementById('testResultContent');
        const resultTitle = document.getElementById('testResultTitle');

        resultTitle.innerHTML = '❌ Test Failed';
        resultContent.innerHTML = `
            <div style="padding: 15px; background: #f8d7da; border: 1px solid #f5c6cb; border-radius: 5px; margin-bottom: 15px;">
                <strong style="color: #721c24;">Test error</strong>
            </div>
            <div style="padding: 15px; background: #f8f9fa; border-radius: 5px; font-family: monospace; white-space: pre-line;">${escapeHtml(error.toString())}</div>
        `;

        document.getElementById('testResultModal').classList.add('active');
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
