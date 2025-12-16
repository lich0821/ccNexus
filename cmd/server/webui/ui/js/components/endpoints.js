import { api } from '../api.js';
import { state } from '../state.js';
import { notifications } from '../utils/notifications.js';
import { getTransformerLabel, getStatusBadge } from '../utils/formatters.js';

class Endpoints {
    constructor() {
        this.container = document.getElementById('view-container');
        this.endpoints = [];
    }

    async render() {
        this.container.innerHTML = `
            <div class="endpoints">
                <div class="flex-between mb-3">
                    <h1>Endpoints</h1>
                    <button class="btn btn-primary" id="add-endpoint-btn">
                        <span>+ Add Endpoint</span>
                    </button>
                </div>

                <div class="card">
                    <div class="card-body">
                        <div id="endpoints-table"></div>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('add-endpoint-btn').addEventListener('click', () => this.showAddModal());

        await this.loadEndpoints();
    }

    async loadEndpoints() {
        try {
            const data = await api.getEndpoints();
            this.endpoints = data.endpoints || [];
            this.renderTable();
        } catch (error) {
            notifications.error('Failed to load endpoints: ' + error.message);
        }
    }

    renderTable() {
        const container = document.getElementById('endpoints-table');

        if (this.endpoints.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-state-icon">🔗</div>
                    <div class="empty-state-title">No Endpoints</div>
                    <div class="empty-state-message">Add your first endpoint to get started</div>
                </div>
            `;
            return;
        }

        container.innerHTML = `
            <div class="table-container">
                <table class="table">
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>API URL</th>
                            <th>Transformer</th>
                            <th>Model</th>
                            <th>Status</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${this.endpoints.map(ep => this.renderEndpointRow(ep)).join('')}
                    </tbody>
                </table>
            </div>
        `;

        // Attach event listeners
        this.attachEventListeners();
    }

    renderEndpointRow(ep) {
        return `
            <tr data-endpoint="${this.escapeHtml(ep.name)}">
                <td><strong>${this.escapeHtml(ep.name)}</strong></td>
                <td><code>${this.escapeHtml(ep.apiUrl)}</code></td>
                <td>${getTransformerLabel(ep.transformer)}</td>
                <td>${this.escapeHtml(ep.model || '-')}</td>
                <td>${getStatusBadge(ep.enabled)}</td>
                <td>
                    <div class="flex gap-2">
                        <button class="btn btn-sm btn-secondary test-btn" data-name="${this.escapeHtml(ep.name)}">
                            Test
                        </button>
                        <label class="toggle-switch">
                            <input type="checkbox" class="toggle-endpoint" data-name="${this.escapeHtml(ep.name)}" ${ep.enabled ? 'checked' : ''}>
                            <span class="toggle-slider"></span>
                        </label>
                        <button class="btn btn-sm btn-secondary edit-btn" data-name="${this.escapeHtml(ep.name)}">
                            Edit
                        </button>
                        <button class="btn btn-sm btn-danger delete-btn" data-name="${this.escapeHtml(ep.name)}">
                            Delete
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }

    attachEventListeners() {
        // Test buttons
        document.querySelectorAll('.test-btn').forEach(btn => {
            btn.addEventListener('click', () => this.testEndpoint(btn.dataset.name));
        });

        // Toggle switches
        document.querySelectorAll('.toggle-endpoint').forEach(toggle => {
            toggle.addEventListener('change', () => this.toggleEndpoint(toggle.dataset.name, toggle.checked));
        });

        // Edit buttons
        document.querySelectorAll('.edit-btn').forEach(btn => {
            btn.addEventListener('click', () => this.showEditModal(btn.dataset.name));
        });

        // Delete buttons
        document.querySelectorAll('.delete-btn').forEach(btn => {
            btn.addEventListener('click', () => this.deleteEndpoint(btn.dataset.name));
        });
    }

    showAddModal() {
        this.showEndpointModal(null);
    }

    showEditModal(name) {
        const endpoint = this.endpoints.find(ep => ep.name === name);
        if (endpoint) {
            this.showEndpointModal(endpoint);
        }
    }

    showEndpointModal(endpoint) {
        const isEdit = !!endpoint;
        const modalContainer = document.getElementById('modal-container');

        modalContainer.innerHTML = `
            <div class="modal-overlay">
                <div class="modal">
                    <div class="modal-header">
                        <h3 class="modal-title">${isEdit ? 'Edit' : 'Add'} Endpoint</h3>
                        <button class="modal-close" id="close-modal">×</button>
                    </div>
                    <div class="modal-body">
                        <form id="endpoint-form">
                            <div class="form-group">
                                <label class="form-label">Name *</label>
                                <input type="text" class="form-input" name="name" value="${endpoint ? this.escapeHtml(endpoint.name) : ''}" required ${isEdit ? 'readonly' : ''}>
                            </div>
                            <div class="form-group">
                                <label class="form-label">API URL *</label>
                                <input type="text" class="form-input" name="apiUrl" value="${endpoint ? this.escapeHtml(endpoint.apiUrl) : ''}" placeholder="https://api.example.com" required>
                            </div>
                            <div class="form-group">
                                <label class="form-label">API Key *</label>
                                <input type="password" class="form-input" name="apiKey" value="${endpoint ? '****' : ''}" placeholder="sk-..." required>
                                ${endpoint ? '<small class="text-muted">Leave as **** to keep existing key</small>' : ''}
                            </div>
                            <div class="form-group">
                                <label class="form-label">Transformer *</label>
                                <select class="form-select" name="transformer" required>
                                    <option value="claude" ${endpoint?.transformer === 'claude' ? 'selected' : ''}>Claude</option>
                                    <option value="openai" ${endpoint?.transformer === 'openai' ? 'selected' : ''}>OpenAI</option>
                                    <option value="openai2" ${endpoint?.transformer === 'openai2' ? 'selected' : ''}>OpenAI Responses</option>
                                    <option value="gemini" ${endpoint?.transformer === 'gemini' ? 'selected' : ''}>Gemini</option>
                                    <option value="deepseek" ${endpoint?.transformer === 'deepseek' ? 'selected' : ''}>DeepSeek</option>
                                </select>
                            </div>
                            <div class="form-group">
                                <label class="form-label">Model</label>
                                <input type="text" class="form-input" name="model" value="${endpoint ? this.escapeHtml(endpoint.model || '') : ''}" placeholder="gpt-4, gemini-pro, etc.">
                            </div>
                            <div class="form-group">
                                <label class="form-label">Remark</label>
                                <textarea class="form-textarea" name="remark">${endpoint ? this.escapeHtml(endpoint.remark || '') : ''}</textarea>
                            </div>
                            <div class="form-group">
                                <label>
                                    <input type="checkbox" class="form-checkbox" name="enabled" ${endpoint?.enabled !== false ? 'checked' : ''}>
                                    Enabled
                                </label>
                            </div>
                        </form>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" id="cancel-btn">Cancel</button>
                        <button class="btn btn-primary" id="save-btn">${isEdit ? 'Update' : 'Create'}</button>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('close-modal').addEventListener('click', () => this.closeModal());
        document.getElementById('cancel-btn').addEventListener('click', () => this.closeModal());
        document.getElementById('save-btn').addEventListener('click', () => this.saveEndpoint(isEdit, endpoint?.name));
    }

    async saveEndpoint(isEdit, originalName) {
        const form = document.getElementById('endpoint-form');
        const formData = new FormData(form);

        const data = {
            name: formData.get('name'),
            apiUrl: formData.get('apiUrl'),
            apiKey: formData.get('apiKey'),
            transformer: formData.get('transformer'),
            model: formData.get('model'),
            remark: formData.get('remark'),
            enabled: formData.get('enabled') === 'on'
        };

        // If editing and API key is ****, don't send it
        if (isEdit && data.apiKey === '****') {
            delete data.apiKey;
        }

        try {
            if (isEdit) {
                await api.updateEndpoint(originalName, data);
                notifications.success('Endpoint updated successfully');
            } else {
                await api.createEndpoint(data);
                notifications.success('Endpoint created successfully');
            }

            this.closeModal();
            await this.loadEndpoints();
        } catch (error) {
            notifications.error('Failed to save endpoint: ' + error.message);
        }
    }

    async toggleEndpoint(name, enabled) {
        try {
            await api.toggleEndpoint(name, enabled);
            notifications.success(`Endpoint ${enabled ? 'enabled' : 'disabled'}`);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error('Failed to toggle endpoint: ' + error.message);
            await this.loadEndpoints(); // Reload to reset toggle state
        }
    }

    async testEndpoint(name) {
        try {
            notifications.info('Testing endpoint...');
            const result = await api.testEndpoint(name);

            if (result.success) {
                notifications.success(`Test successful! Latency: ${result.latency}ms`);
                this.showTestResultModal(name, result);
            } else {
                notifications.error(`Test failed: ${result.error}`);
            }
        } catch (error) {
            notifications.error('Test failed: ' + error.message);
        }
    }

    showTestResultModal(name, result) {
        const modalContainer = document.getElementById('modal-container');

        modalContainer.innerHTML = `
            <div class="modal-overlay">
                <div class="modal">
                    <div class="modal-header">
                        <h3 class="modal-title">Test Result: ${this.escapeHtml(name)}</h3>
                        <button class="modal-close" id="close-modal">×</button>
                    </div>
                    <div class="modal-body">
                        <div class="mb-2">
                            <strong>Status:</strong> <span class="badge badge-success">Success</span>
                        </div>
                        <div class="mb-2">
                            <strong>Latency:</strong> ${result.latency}ms
                        </div>
                        <div class="mb-2">
                            <strong>Response:</strong>
                            <div class="code-block mt-1">${this.escapeHtml(result.response || 'No response')}</div>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-primary" id="close-btn">Close</button>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('close-modal').addEventListener('click', () => this.closeModal());
        document.getElementById('close-btn').addEventListener('click', () => this.closeModal());
    }

    async deleteEndpoint(name) {
        if (!confirm(`Are you sure you want to delete endpoint "${name}"?`)) {
            return;
        }

        try {
            await api.deleteEndpoint(name);
            notifications.success('Endpoint deleted successfully');
            await this.loadEndpoints();
        } catch (error) {
            notifications.error('Failed to delete endpoint: ' + error.message);
        }
    }

    closeModal() {
        document.getElementById('modal-container').innerHTML = '';
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

export const endpoints = new Endpoints();
