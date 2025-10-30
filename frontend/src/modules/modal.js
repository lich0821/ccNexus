import { t } from '../i18n/index.js';
import { escapeHtml } from '../utils/format.js';
import { addEndpoint, updateEndpoint, removeEndpoint, testEndpoint, updatePort } from './config.js';
import { setTestState, clearTestState } from './endpoints.js';

let currentEditIndex = -1;

// Endpoint Modal
export function showAddEndpointModal() {
    currentEditIndex = -1;
    document.getElementById('modalTitle').textContent = t('modal.addEndpoint');
    document.getElementById('endpointName').value = '';
    document.getElementById('endpointUrl').value = '';
    document.getElementById('endpointKey').value = '';
    document.getElementById('endpointTransformer').value = 'claude';
    document.getElementById('endpointModel').value = '';
    handleTransformerChange();
    document.getElementById('endpointModal').classList.add('active');
}

export async function editEndpoint(index) {
    currentEditIndex = index;
    const configStr = await window.go.main.App.GetConfig();
    const config = JSON.parse(configStr);
    const ep = config.endpoints[index];

    document.getElementById('modalTitle').textContent = t('modal.editEndpoint');
    document.getElementById('endpointName').value = ep.name;
    document.getElementById('endpointUrl').value = ep.apiUrl;
    document.getElementById('endpointKey').value = ep.apiKey;
    document.getElementById('endpointTransformer').value = ep.transformer || 'claude';
    document.getElementById('endpointModel').value = ep.model || '';

    handleTransformerChange();
    document.getElementById('endpointModal').classList.add('active');
}

export async function saveEndpoint() {
    const name = document.getElementById('endpointName').value.trim();
    const url = document.getElementById('endpointUrl').value.trim();
    const key = document.getElementById('endpointKey').value.trim();
    const transformer = document.getElementById('endpointTransformer').value;
    const model = document.getElementById('endpointModel').value.trim();

    if (!name || !url || !key) {
        alert('Please fill in all required fields');
        return;
    }

    if (transformer !== 'claude' && !model) {
        alert('Model field is required for ' + transformer + ' transformer');
        return;
    }

    try {
        if (currentEditIndex === -1) {
            await addEndpoint(name, url, key, transformer, model);
        } else {
            await updateEndpoint(currentEditIndex, name, url, key, transformer, model);
        }

        closeModal();
        window.loadConfig();
    } catch (error) {
        alert('Failed to save endpoint: ' + error);
    }
}

export async function deleteEndpoint(index) {
    try {
        await removeEndpoint(index);
        window.loadConfig();
    } catch (error) {
        console.error('Delete failed:', error);
        alert('Failed to delete endpoint: ' + error);
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
        document.querySelector('#welcomeModal .modal-header h2').textContent = `👋 Welcome to ccNexus v${version}`;
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
        window.go.main.App.OpenURL('https://mp.weixin.qq.com/s/MqUVgWbkcVUNPnZQC--CZQ');
    }
}
