import { api } from '../api.js';
import { notifications } from '../utils/notifications.js';
import { t } from '../utils/i18n.js';

class Testing {
    constructor() {
        this.container = document.getElementById('view-container');
        this.endpoints = [];
        // 监听语言切换
        window.addEventListener('languageChanged', () => {
            if (state.get('currentView') === 'testing') {
                this.render();
            }
        });
    }

    async render() {
        this.container.innerHTML = `
            <div class="testing">
                <h1>${t('testing.title')}</h1>

                <div class="card mt-3">
                    <div class="card-body">
                        <div class="form-group">
                            <label class="form-label">${t('testing.selectEndpoint')}</label>
                            <select class="form-select" id="test-endpoint-select">
                                <option value="">${t('common.loading')}</option>
                            </select>
                        </div>

                        <div class="form-group">
                            <button class="btn btn-primary" id="test-btn">${t('testing.runTest')}</button>
                        </div>

                        <div id="test-result" class="mt-3" style="display: none;"></div>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('test-btn').addEventListener('click', () => this.runTest());

        await this.loadEndpoints();
    }

    async loadEndpoints() {
        try {
            const data = await api.getEndpoints();
            this.endpoints = data.endpoints || [];

            const select = document.getElementById('test-endpoint-select');
            const enabledEndpoints = this.endpoints.filter(ep => ep.enabled);

            if (enabledEndpoints.length === 0) {
                select.innerHTML = `<option value="">${t('testing.noEnabledEndpoints')}</option>`;
                return;
            }

            select.innerHTML = enabledEndpoints.map(ep =>
                `<option value="${this.escapeHtml(ep.name)}">${this.escapeHtml(ep.name)}</option>`
            ).join('');
        } catch (error) {
            notifications.error(`${t('testing.failedToLoadEndpoints')}: ${error.message}`);
        }
    }

    async runTest() {
        const select = document.getElementById('test-endpoint-select');
        const endpointName = select.value;

        if (!endpointName) {
            notifications.warning(t('testing.pleaseSelectEndpoint'));
            return;
        }

        const resultDiv = document.getElementById('test-result');
        resultDiv.style.display = 'block';
        resultDiv.innerHTML = '<div class="flex-center"><div class="spinner"></div></div>';

        try {
            const result = await api.testEndpoint(endpointName);

            if (result.success) {
                resultDiv.innerHTML = `
                    <div class="card" style="background-color: var(--bg-secondary);">
                        <div class="mb-2">
                            <span class="badge badge-success">${t('common.success')}</span>
                            <span class="text-muted ml-2">${t('testing.latency')}: ${result.latency}ms</span>
                        </div>
                        <div>
                            <strong>${t('testing.response')}:</strong>
                            <div class="code-block mt-1">${this.escapeHtml(result.response || t('testing.noResponse'))}</div>
                        </div>
                    </div>
                `;
                notifications.success(t('testing.testCompletedSuccessfully'));
            } else {
                resultDiv.innerHTML = `
                    <div class="card" style="background-color: var(--bg-secondary);">
                        <div class="mb-2">
                            <span class="badge badge-danger">${t('testing.testFailed')}</span>
                        </div>
                        <div>
                            <strong>${t('testing.error')}:</strong>
                            <div class="code-block mt-1">${this.escapeHtml(result.error || t('testing.unknownError'))}</div>
                        </div>
                    </div>
                `;
                notifications.error(t('testing.testFailed'));
            }
        } catch (error) {
            resultDiv.innerHTML = `
                <div class="card" style="background-color: var(--bg-secondary);">
                    <div class="mb-2">
                        <span class="badge badge-danger">${t('common.error')}</span>
                    </div>
                    <div>
                        <strong>${t('testing.error')}:</strong>
                        <div class="code-block mt-1">${this.escapeHtml(error.message)}</div>
                    </div>
                </div>
            `;
            notifications.error(`${t('testing.testFailed')}: ${error.message}`);
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

export const testing = new Testing();
