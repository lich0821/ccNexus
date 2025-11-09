import { t } from '../i18n/index.js';
import { formatTokens, maskApiKey } from '../utils/format.js';
import { getEndpointStats } from './stats.js';
import { toggleEndpoint } from './config.js';

let currentTestButton = null;
let currentTestButtonOriginalText = '';
let currentTestIndex = -1;

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

    // Get current endpoint
    let currentEndpointName = '';
    try {
        currentEndpointName = await window.go.main.App.GetCurrentEndpoint();
    } catch (error) {
        console.error('Failed to get current endpoint:', error);
    }

    if (endpoints.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <p>${t('endpoints.noEndpoints')}</p>
            </div>
        `;
        return;
    }

    container.innerHTML = '';

    const endpointStats = getEndpointStats();
    const sortedEndpoints = endpoints.map((ep, index) => {
        const stats = endpointStats[ep.name] || { requests: 0, errors: 0, inputTokens: 0, outputTokens: 0 };
        const enabled = ep.enabled !== undefined ? ep.enabled : true;
        return { endpoint: ep, originalIndex: index, stats, enabled };
    }).sort((a, b) => {
        // Sort by enabled status first (enabled endpoints first)
        if (a.enabled !== b.enabled) return a.enabled ? -1 : 1;

        // Then sort by original index (config file order)
        return a.originalIndex - b.originalIndex;
    });

    sortedEndpoints.forEach(({ endpoint: ep, originalIndex: index, stats }) => {
        const totalTokens = stats.inputTokens + stats.outputTokens;
        const enabled = ep.enabled !== undefined ? ep.enabled : true;
        const transformer = ep.transformer || 'claude';
        const model = ep.model || '';
        const isCurrentEndpoint = ep.name === currentEndpointName;

        const item = document.createElement('div');
        item.className = 'endpoint-item';
        item.innerHTML = `
            <div class="endpoint-info">
                <h3>
                    ${ep.name}
                    ${enabled ? '✅' : '❌'}
                    ${isCurrentEndpoint ? '<span class="current-badge">' + t('endpoints.current') + '</span>' : ''}
                    ${enabled && !isCurrentEndpoint ? '<button class="btn btn-switch" data-action="switch" data-name="' + ep.name + '">' + t('endpoints.switchTo') + '</button>' : ''}
                </h3>
                <p style="display: flex; align-items: center; gap: 8px; min-width: 0;"><span style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">🌐 ${ep.apiUrl}</span> <button class="copy-btn" data-copy="${ep.apiUrl}" aria-label="${t('endpoints.copy')}" title="${t('endpoints.copy')}"><svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" width="1em" height="1em"><path d="M7 4c0-1.1.9-2 2-2h11a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2h-1V8c0-2-1-3-3-3H7V4Z" fill="currentColor"></path><path d="M5 7a2 2 0 0 0-2 2v10c0 1.1.9 2 2 2h10a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2H5Z" fill="currentColor"></path></svg></button></p>
                <p style="display: flex; align-items: center; gap: 8px; min-width: 0;"><span style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">🔑 ${maskApiKey(ep.apiKey)}</span> <button class="copy-btn" data-copy="${ep.apiKey}" aria-label="${t('endpoints.copy')}" title="${t('endpoints.copy')}"><svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" width="1em" height="1em"><path d="M7 4c0-1.1.9-2 2-2h11a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2h-1V8c0-2-1-3-3-3H7V4Z" fill="currentColor"></path><path d="M5 7a2 2 0 0 0-2 2v10c0 1.1.9 2 2 2h10a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2H5Z" fill="currentColor"></path></svg></button></p>
                <p style="color: #666; font-size: 14px; margin-top: 5px;">🔄 ${t('endpoints.transformer')}: ${transformer}${model ? ` (${model})` : ''}</p>
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

        container.appendChild(item);
    });
}
