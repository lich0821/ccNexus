import { api } from '../api.js';
import { notifications } from '../utils/notifications.js';

class Testing {
    constructor() {
        this.container = document.getElementById('view-container');
        this.endpoints = [];
    }

    async render() {
        this.container.innerHTML = `
            <div class="testing">
                <h1>Endpoint Testing</h1>

                <div class="card mt-3">
                    <div class="card-body">
                        <div class="form-group">
                            <label class="form-label">Select Endpoint</label>
                            <select class="form-select" id="test-endpoint-select">
                                <option value="">Loading...</option>
                            </select>
                        </div>

                        <div class="form-group">
                            <button class="btn btn-primary" id="test-btn">Run Test</button>
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
                select.innerHTML = '<option value="">No enabled endpoints</option>';
                return;
            }

            select.innerHTML = enabledEndpoints.map(ep =>
                `<option value="${this.escapeHtml(ep.name)}">${this.escapeHtml(ep.name)}</option>`
            ).join('');
        } catch (error) {
            notifications.error('Failed to load endpoints: ' + error.message);
        }
    }

    async runTest() {
        const select = document.getElementById('test-endpoint-select');
        const endpointName = select.value;

        if (!endpointName) {
            notifications.warning('Please select an endpoint');
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
                            <span class="badge badge-success">Success</span>
                            <span class="text-muted ml-2">Latency: ${result.latency}ms</span>
                        </div>
                        <div>
                            <strong>Response:</strong>
                            <div class="code-block mt-1">${this.escapeHtml(result.response || 'No response')}</div>
                        </div>
                    </div>
                `;
                notifications.success('Test completed successfully');
            } else {
                resultDiv.innerHTML = `
                    <div class="card" style="background-color: var(--bg-secondary);">
                        <div class="mb-2">
                            <span class="badge badge-danger">Failed</span>
                        </div>
                        <div>
                            <strong>Error:</strong>
                            <div class="code-block mt-1">${this.escapeHtml(result.error || 'Unknown error')}</div>
                        </div>
                    </div>
                `;
                notifications.error('Test failed');
            }
        } catch (error) {
            resultDiv.innerHTML = `
                <div class="card" style="background-color: var(--bg-secondary);">
                    <div class="mb-2">
                        <span class="badge badge-danger">Error</span>
                    </div>
                    <div>
                        <strong>Error:</strong>
                        <div class="code-block mt-1">${this.escapeHtml(error.message)}</div>
                    </div>
                </div>
            `;
            notifications.error('Test failed: ' + error.message);
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

export const testing = new Testing();
