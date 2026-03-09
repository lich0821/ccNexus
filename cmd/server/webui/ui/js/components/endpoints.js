import { api } from '../api.js';
import { state } from '../state.js';
import { notifications } from '../utils/notifications.js';
import { getTransformerLabel, getStatusBadge } from '../utils/formatters.js';

class Endpoints {
    constructor() {
        this.container = document.getElementById('view-container');
        this.endpoints = [];
        this.tokenPools = {};
        this.currentEndpoint = null;
        this.draggedIndex = null;
        this.currentTokenPoolEndpoint = null;
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
            this.tokenPools = data.tokenPools || {};

            // Get current endpoint
            try {
                const currentData = await api.getCurrentEndpoint();
                this.currentEndpoint = currentData.name || null;
            } catch (error) {
                console.error('Failed to get current endpoint:', error);
                this.currentEndpoint = null;
            }

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
                            <th style="width: 30px;"></th>
                            <th>Name</th>
                            <th>API URL</th>
                            <th>Transformer</th>
                            <th>Model</th>
                            <th>Token Pool</th>
                            <th>Status</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody id="endpoints-tbody">
                        ${this.endpoints.map((ep, index) => this.renderEndpointRow(ep, index)).join('')}
                    </tbody>
                </table>
            </div>
        `;

        // Attach event listeners
        this.attachEventListeners();
        this.attachDragListeners();
    }

    renderEndpointRow(ep, index) {
        const isCurrentEndpoint = ep.name === this.currentEndpoint;
        const testStatus = this.getTestStatus(ep.name);
        let testStatusIcon = '⚠️';
        let testStatusTitle = 'Not tested';

        if (testStatus === true) {
            testStatusIcon = '✅';
            testStatusTitle = 'Test passed';
        } else if (testStatus === false) {
            testStatusIcon = '❌';
            testStatusTitle = 'Test failed';
        }

        return `
            <tr data-endpoint="${this.escapeHtml(ep.name)}" data-index="${index}" draggable="true" style="cursor: move;">
                <td style="cursor: grab; text-align: center;">⋮⋮</td>
                <td>
                    <strong>${this.escapeHtml(ep.name)}</strong>
                    <span title="${testStatusTitle}" style="margin-left: 5px;">${testStatusIcon}</span>
                    ${isCurrentEndpoint ? '<span class="badge badge-primary" style="margin-left: 5px;">Current</span>' : ''}
                </td>
                <td>
                    <code style="font-size: 12px;">${this.escapeHtml(ep.apiUrl)}</code>
                    <button class="btn-icon copy-btn" data-copy="${this.escapeHtml(ep.apiUrl)}" title="Copy URL">
                        📋
                    </button>
                </td>
                <td>${getTransformerLabel(ep.transformer)}</td>
                <td>${this.escapeHtml(ep.model || '-')}</td>
                <td>${this.renderTokenPoolSummary(this.tokenPools[ep.name])}</td>
                <td>${getStatusBadge(ep.enabled)}</td>
                <td>
                    <div class="flex gap-2">
                        ${ep.enabled && !isCurrentEndpoint ? `
                            <button class="btn btn-sm btn-secondary switch-btn" data-name="${this.escapeHtml(ep.name)}" title="Switch to this endpoint">
                                Switch
                            </button>
                        ` : ''}
                        <button class="btn btn-sm btn-secondary test-btn" data-name="${this.escapeHtml(ep.name)}">
                            Test
                        </button>
                        <button class="btn btn-sm btn-secondary token-pool-btn" data-name="${this.escapeHtml(ep.name)}">
                            Token Pool
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

    renderTokenPoolSummary(pool) {
        if (!pool || !pool.total) {
            return '<span class="text-muted">0</span>';
        }

        return `
            <div style="font-size: 12px; line-height: 1.4;">
                <div>Total: <strong>${pool.total}</strong></div>
                <div>A:${pool.active || 0} E:${pool.expiring || 0} X:${pool.expired || 0} I:${pool.invalid || 0}</div>
                <div>C:${pool.cooldown || 0} R:${pool.needRefresh || 0} D:${pool.disabled || 0}</div>
            </div>
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

        // Switch buttons
        document.querySelectorAll('.switch-btn').forEach(btn => {
            btn.addEventListener('click', () => this.switchEndpoint(btn.dataset.name));
        });

        // Token pool buttons
        document.querySelectorAll('.token-pool-btn').forEach(btn => {
            btn.addEventListener('click', () => this.showTokenPoolModal(btn.dataset.name));
        });

        // Copy buttons
        document.querySelectorAll('.copy-btn').forEach(btn => {
            btn.addEventListener('click', () => this.copyToClipboard(btn.dataset.copy, btn));
        });
    }

    attachDragListeners() {
        const rows = document.querySelectorAll('#endpoints-tbody tr[draggable="true"]');

        rows.forEach(row => {
            row.addEventListener('dragstart', (e) => {
                this.draggedIndex = parseInt(row.dataset.index);
                row.style.opacity = '0.5';
            });

            row.addEventListener('dragend', (e) => {
                row.style.opacity = '1';
            });

            row.addEventListener('dragover', (e) => {
                e.preventDefault();
                row.style.borderTop = '2px solid #3b82f6';
            });

            row.addEventListener('dragleave', (e) => {
                row.style.borderTop = '';
            });

            row.addEventListener('drop', async (e) => {
                e.preventDefault();
                row.style.borderTop = '';

                const dropIndex = parseInt(row.dataset.index);
                if (this.draggedIndex !== null && this.draggedIndex !== dropIndex) {
                    await this.reorderEndpoints(this.draggedIndex, dropIndex);
                }
                this.draggedIndex = null;
            });
        });
    }

    async reorderEndpoints(fromIndex, toIndex) {
        try {
            // Reorder the array
            const [movedItem] = this.endpoints.splice(fromIndex, 1);
            this.endpoints.splice(toIndex, 0, movedItem);

            // Send new order to backend
            const names = this.endpoints.map(ep => ep.name);
            await api.reorderEndpoints(names);

            notifications.success('Endpoints reordered successfully');
            await this.loadEndpoints();
        } catch (error) {
            notifications.error('Failed to reorder endpoints: ' + error.message);
            await this.loadEndpoints(); // Reload to reset order
        }
    }

    async switchEndpoint(name) {
        try {
            await api.switchEndpoint(name);
            notifications.success(`Switched to endpoint: ${name}`);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error('Failed to switch endpoint: ' + error.message);
        }
    }

    copyToClipboard(text, button) {
        navigator.clipboard.writeText(text).then(() => {
            const originalText = button.textContent;
            button.textContent = '✓';
            setTimeout(() => {
                button.textContent = originalText;
            }, 1000);
        }).catch(err => {
            notifications.error('Failed to copy to clipboard');
        });
    }

    getTestStatus(endpointName) {
        try {
            const statusMap = JSON.parse(localStorage.getItem('ccNexus_endpointTestStatus') || '{}');
            return statusMap[endpointName];
        } catch {
            return undefined;
        }
    }

    saveTestStatus(endpointName, success) {
        try {
            const statusMap = JSON.parse(localStorage.getItem('ccNexus_endpointTestStatus') || '{}');
            statusMap[endpointName] = success;
            localStorage.setItem('ccNexus_endpointTestStatus', JSON.stringify(statusMap));
        } catch (error) {
            console.error('Failed to save test status:', error);
        }
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
                                <div style="display: flex; gap: 8px;">
                                    <input type="text" class="form-input" name="model" id="model-input" value="${endpoint ? this.escapeHtml(endpoint.model || '') : ''}" placeholder="gpt-4, gemini-pro, etc." style="flex: 1;">
                                    <button type="button" class="btn btn-secondary" id="fetch-models-btn" style="white-space: nowrap;">
                                        Fetch Models
                                    </button>
                                </div>
                                <small class="text-muted">Click "Fetch Models" to load available models from the API</small>
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
        document.getElementById('fetch-models-btn').addEventListener('click', () => this.fetchModels());
    }

    async fetchModels() {
        const apiUrlInput = document.querySelector('input[name="apiUrl"]');
        const apiKeyInput = document.querySelector('input[name="apiKey"]');
        const transformerSelect = document.querySelector('select[name="transformer"]');
        const modelInput = document.getElementById('model-input');
        const fetchBtn = document.getElementById('fetch-models-btn');

        const apiUrl = apiUrlInput.value.trim();
        const apiKey = apiKeyInput.value.trim();
        const transformer = transformerSelect.value;

        if (!apiUrl || !apiKey || apiKey === '****') {
            notifications.error('Please enter API URL and API Key first');
            return;
        }

        try {
            fetchBtn.disabled = true;
            fetchBtn.textContent = 'Fetching...';

            const result = await api.fetchModels(apiUrl, apiKey, transformer);

            if (result.models && result.models.length > 0) {
                // Show model selection modal
                this.showModelSelectionModal(result.models, modelInput);
            } else {
                notifications.info('No models found');
            }
        } catch (error) {
            notifications.error('Failed to fetch models: ' + error.message);
        } finally {
            fetchBtn.disabled = false;
            fetchBtn.textContent = 'Fetch Models';
        }
    }

    showModelSelectionModal(models, modelInput) {
        const modalContainer = document.getElementById('modal-container');
        const currentModal = modalContainer.querySelector('.modal');

        // Create a second modal overlay
        const modelModal = document.createElement('div');
        modelModal.className = 'modal-overlay';
        modelModal.style.zIndex = '1001';
        modelModal.innerHTML = `
            <div class="modal" style="max-width: 500px;">
                <div class="modal-header">
                    <h3 class="modal-title">Select Model</h3>
                    <button class="modal-close" id="close-model-modal">×</button>
                </div>
                <div class="modal-body">
                    <div style="max-height: 400px; overflow-y: auto;">
                        ${models.map(model => `
                            <div class="model-item" style="padding: 10px; border-bottom: 1px solid #e5e7eb; cursor: pointer;" data-model="${this.escapeHtml(model)}">
                                <strong>${this.escapeHtml(model)}</strong>
                            </div>
                        `).join('')}
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary" id="cancel-model-btn">Cancel</button>
                </div>
            </div>
        `;

        modalContainer.appendChild(modelModal);

        // Attach event listeners
        document.getElementById('close-model-modal').addEventListener('click', () => {
            modelModal.remove();
        });

        document.getElementById('cancel-model-btn').addEventListener('click', () => {
            modelModal.remove();
        });

        document.querySelectorAll('.model-item').forEach(item => {
            item.addEventListener('click', () => {
                const selectedModel = item.dataset.model;
                modelInput.value = selectedModel;
                notifications.success(`Model selected: ${selectedModel}`);
                modelModal.remove();
            });

            item.addEventListener('mouseenter', () => {
                item.style.backgroundColor = '#f3f4f6';
            });

            item.addEventListener('mouseleave', () => {
                item.style.backgroundColor = '';
            });
        });
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
                this.saveTestStatus(name, true);
                notifications.success(`Test successful! Latency: ${result.latency}ms`);
                this.showTestResultModal(name, result);
                await this.loadEndpoints(); // Refresh to show test status
            } else {
                this.saveTestStatus(name, false);
                notifications.error(`Test failed: ${result.error}`);
                await this.loadEndpoints(); // Refresh to show test status
            }
        } catch (error) {
            this.saveTestStatus(name, false);
            notifications.error('Test failed: ' + error.message);
            await this.loadEndpoints(); // Refresh to show test status
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

    async showTokenPoolModal(endpointName) {
        this.currentTokenPoolEndpoint = endpointName;

        try {
            const result = await api.getEndpointCredentials(endpointName);
            const credentials = result.credentials || [];
            const stats = result.stats || {};
            const modalContainer = document.getElementById('modal-container');

            modalContainer.innerHTML = `
                <div class="modal-overlay">
                    <div class="modal" style="max-width: 960px; width: 95vw;">
                        <div class="modal-header">
                            <h3 class="modal-title">Token Pool: ${this.escapeHtml(endpointName)}</h3>
                            <button class="modal-close" id="close-modal">×</button>
                        </div>
                        <div class="modal-body">
                            <div class="mb-2" style="font-size: 13px;">
                                <strong>Total:</strong> ${stats.total || 0}
                                <span style="margin-left: 12px;"><strong>Active:</strong> ${stats.active || 0}</span>
                                <span style="margin-left: 12px;"><strong>Expiring:</strong> ${stats.expiring || 0}</span>
                                <span style="margin-left: 12px;"><strong>Need Refresh:</strong> ${stats.needRefresh || 0}</span>
                                <span style="margin-left: 12px;"><strong>Expired:</strong> ${stats.expired || 0}</span>
                                <span style="margin-left: 12px;"><strong>Invalid:</strong> ${stats.invalid || 0}</span>
                            </div>

                            <div class="form-group">
                                <label class="form-label">Batch Import JSON</label>
                                <textarea class="form-textarea" id="token-import-json" style="min-height: 140px;" placeholder='Paste one item / array / {"items":[...]}'></textarea>
                                <label style="display: inline-flex; gap: 8px; align-items: center; margin-top: 8px;">
                                    <input type="checkbox" id="token-import-overwrite">
                                    Overwrite existing account_id/email
                                </label>
                                <div style="margin-top: 8px;">
                                    <button class="btn btn-primary" id="token-import-btn">Import</button>
                                </div>
                            </div>

                            <div class="table-container">
                                <table class="table">
                                    <thead>
                                        <tr>
                                            <th>ID</th>
                                            <th>Account</th>
                                            <th>Email</th>
                                            <th>Status</th>
                                            <th>Expires At</th>
                                            <th>Last Error</th>
                                            <th>Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        ${this.renderCredentialRows(credentials)}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                        <div class="modal-footer">
                            <button class="btn btn-secondary" id="refresh-token-pool-btn">Refresh</button>
                            <button class="btn btn-secondary" id="close-token-pool-btn">Close</button>
                        </div>
                    </div>
                </div>
            `;

            document.getElementById('close-modal').addEventListener('click', () => this.closeModal());
            document.getElementById('close-token-pool-btn').addEventListener('click', () => this.closeModal());
            document.getElementById('refresh-token-pool-btn').addEventListener('click', () => this.showTokenPoolModal(endpointName));
            document.getElementById('token-import-btn').addEventListener('click', () => this.importEndpointCredentials(endpointName));

            document.querySelectorAll('.token-enable-toggle').forEach(toggle => {
                toggle.addEventListener('change', () => this.updateCredentialEnabled(endpointName, toggle.dataset.id, toggle.checked));
            });
            document.querySelectorAll('.token-update-btn').forEach(btn => {
                btn.addEventListener('click', () => this.updateCredentialToken(endpointName, btn.dataset.id));
            });
            document.querySelectorAll('.token-activate-btn').forEach(btn => {
                btn.addEventListener('click', () => this.activateCredential(endpointName, btn.dataset.id));
            });
            document.querySelectorAll('.token-delete-btn').forEach(btn => {
                btn.addEventListener('click', () => this.deleteCredential(endpointName, btn.dataset.id));
            });
        } catch (error) {
            notifications.error('Failed to load token pool: ' + error.message);
        }
    }

    renderCredentialRows(credentials) {
        if (!credentials || credentials.length === 0) {
            return '<tr><td colspan="7" class="text-center text-muted">No credentials imported</td></tr>';
        }

        return credentials.map(cred => `
            <tr>
                <td>${cred.id}</td>
                <td><code>${this.escapeHtml(cred.accountId || '-')}</code></td>
                <td>${this.escapeHtml(cred.email || '-')}</td>
                <td>${this.renderCredentialStatusBadge(cred.status)}</td>
                <td>${this.escapeHtml(this.formatDateTime(cred.expiresAt))}</td>
                <td style="max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${this.escapeHtml(cred.lastError || '')}">
                    ${this.escapeHtml(cred.lastError || '-')}
                </td>
                <td>
                    <div class="flex gap-2">
                        <label style="display: inline-flex; align-items: center; gap: 6px; font-size: 12px;">
                            <input type="checkbox" class="token-enable-toggle" data-id="${cred.id}" ${cred.enabled ? 'checked' : ''}>
                            Enabled
                        </label>
                        <button class="btn btn-sm btn-secondary token-update-btn" data-id="${cred.id}">Update</button>
                        <button class="btn btn-sm btn-secondary token-activate-btn" data-id="${cred.id}">Activate</button>
                        <button class="btn btn-sm btn-danger token-delete-btn" data-id="${cred.id}">Delete</button>
                    </div>
                </td>
            </tr>
        `).join('');
    }

    renderCredentialStatusBadge(status) {
        const normalized = status || 'unknown';
        const colorMap = {
            active: '#10b981',
            expiring: '#f59e0b',
            need_refresh: '#f97316',
            expired: '#ef4444',
            invalid: '#ef4444',
            cooldown: '#6366f1',
            disabled: '#6b7280'
        };
        const color = colorMap[normalized] || '#6b7280';
        return `<span style="display:inline-block;padding:2px 8px;border-radius:999px;background:${color};color:#fff;font-size:12px;">${this.escapeHtml(normalized)}</span>`;
    }

    async importEndpointCredentials(endpointName) {
        const jsonInput = document.getElementById('token-import-json');
        const overwriteInput = document.getElementById('token-import-overwrite');
        const raw = (jsonInput?.value || '').trim();

        if (!raw) {
            notifications.warning('Please paste credential JSON first');
            return;
        }

        let payload;
        try {
            payload = JSON.parse(raw);
        } catch {
            notifications.error('Invalid JSON');
            return;
        }

        let requestBody;
        if (Array.isArray(payload)) {
            requestBody = { items: payload, overwrite: overwriteInput?.checked === true };
        } else if (payload.items && Array.isArray(payload.items)) {
            requestBody = { ...payload, overwrite: overwriteInput?.checked === true };
        } else {
            requestBody = { items: [payload], overwrite: overwriteInput?.checked === true };
        }

        try {
            const result = await api.importEndpointCredentials(endpointName, requestBody);
            notifications.success(`Import done: +${result.created || 0}, updated ${result.updated || 0}, skipped ${result.skipped || 0}, failed ${result.failed || 0}`);
            jsonInput.value = '';
            await this.showTokenPoolModal(endpointName);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error('Import failed: ' + error.message);
        }
    }

    async updateCredentialEnabled(endpointName, credentialId, enabled) {
        try {
            await api.updateEndpointCredential(endpointName, credentialId, { enabled });
            notifications.success(`Credential ${enabled ? 'enabled' : 'disabled'}`);
            await this.showTokenPoolModal(endpointName);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error('Failed to update credential: ' + error.message);
            await this.showTokenPoolModal(endpointName);
        }
    }

    async activateCredential(endpointName, credentialId) {
        try {
            await api.updateEndpointCredential(endpointName, credentialId, { status: 'active' });
            notifications.success('Credential activated');
            await this.showTokenPoolModal(endpointName);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error('Failed to activate credential: ' + error.message);
        }
    }

    async updateCredentialToken(endpointName, credentialId) {
        const accessToken = prompt('New access token');
        if (!accessToken) {
            return;
        }

        const expiresAt = prompt('expiresAt (RFC3339, optional)', '');
        const payload = {
            accessToken: accessToken.trim(),
            status: 'active'
        };
        if (expiresAt && expiresAt.trim()) {
            payload.expiresAt = expiresAt.trim();
        }

        try {
            await api.updateEndpointCredential(endpointName, credentialId, payload);
            notifications.success('Credential token updated');
            await this.showTokenPoolModal(endpointName);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error('Failed to update token: ' + error.message);
        }
    }

    async deleteCredential(endpointName, credentialId) {
        if (!confirm(`Delete credential #${credentialId}?`)) {
            return;
        }

        try {
            await api.deleteEndpointCredential(endpointName, credentialId);
            notifications.success('Credential deleted');
            await this.showTokenPoolModal(endpointName);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error('Failed to delete credential: ' + error.message);
        }
    }

    formatDateTime(value) {
        if (!value) {
            return '-';
        }
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return value;
        }
        return date.toLocaleString();
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
