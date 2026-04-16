import { api } from '../api.js';
import { state } from '../state.js';
import { notifications } from '../utils/notifications.js';
import { getTransformerLabel, getStatusBadge } from '../utils/formatters.js';
import { t } from '../utils/i18n.js';

class Endpoints {
    constructor() {
        this.container = document.getElementById('view-container');
        this.endpoints = [];
        this.tokenPools = {};
        this.currentEndpoint = null;
        this.draggedIndex = null;
        this.currentTokenPoolEndpoint = null;
        // 监听语言切换
        window.addEventListener('languageChanged', () => {
            if (state.get('currentView') === 'endpoints') {
                this.render();
            }
        });
    }

    async render() {
        this.container.innerHTML = `
            <div class="endpoints">
                <div class="flex-between mb-3">
                    <h1>${t('endpoints.title')}</h1>
                    <button class="btn btn-primary" id="add-endpoint-btn">
                        <span>+ ${t('endpoints.addEndpoint')}</span>
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
            notifications.error(`${t('endpoints.failedToLoad')}: ${error.message}`);
        }
    }

    renderTable() {
        const container = document.getElementById('endpoints-table');

        if (this.endpoints.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-state-icon">🔗</div>
                    <div class="empty-state-title">${t('endpoints.noEndpoints')}</div>
                    <div class="empty-state-message">${t('endpoints.noEndpointsMessage')}</div>
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
                            <th>${t('common.name')}</th>
                            <th>${t('endpoints.apiUrl')}</th>
                            <th>${t('endpoints.transformer')}</th>
                            <th>${t('endpoints.model')}</th>
                            <th>${t('endpoints.tokenPool')}</th>
                            <th>${t('common.status')}</th>
                            <th>${t('common.actions')}</th>
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
        let testStatusTitle = t('endpoints.notTested');

        if (testStatus === true) {
            testStatusIcon = '✅';
            testStatusTitle = t('endpoints.testPassed');
        } else if (testStatus === false) {
            testStatusIcon = '❌';
            testStatusTitle = t('endpoints.testFailed');
        }

        return `
            <tr data-endpoint="${this.escapeHtml(ep.name)}" data-index="${index}" draggable="true" style="cursor: move;">
                <td style="cursor: grab; text-align: center;">⋮⋮</td>
                <td>
                    <strong>${this.escapeHtml(ep.name)}</strong>
                    <span title="${testStatusTitle}" style="margin-left: 5px;">${testStatusIcon}</span>
                    ${isCurrentEndpoint ? `<span class="badge badge-primary" style="margin-left: 5px;">${t('endpoints.current')}</span>` : ''}
                </td>
                <td>
                    <code style="font-size: 12px;">${this.escapeHtml(ep.apiUrl)}</code>
                    <button class="btn-icon copy-btn" data-copy="${this.escapeHtml(ep.apiUrl)}" title="${t('endpoints.copyUrl')}">
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
                            <button class="btn btn-sm btn-secondary switch-btn" data-name="${this.escapeHtml(ep.name)}" title="${t('endpoints.switchToEndpoint')}">
                                ${t('common.switch')}
                            </button>
                        ` : ''}
                        <button class="btn btn-sm btn-secondary test-btn" data-name="${this.escapeHtml(ep.name)}">
                            ${t('common.test')}
                        </button>
                        <button class="btn btn-sm btn-secondary token-pool-btn" data-name="${this.escapeHtml(ep.name)}">
                            ${t('endpoints.tokenPoolManagement')}
                        </button>
                        <label class="toggle-switch">
                            <input type="checkbox" class="toggle-endpoint" data-name="${this.escapeHtml(ep.name)}" ${ep.enabled ? 'checked' : ''}>
                            <span class="toggle-slider"></span>
                        </label>
                        <button class="btn btn-sm btn-secondary edit-btn" data-name="${this.escapeHtml(ep.name)}">
                            ${t('common.edit')}
                        </button>
                        <button class="btn btn-sm btn-secondary clone-btn" data-name="${this.escapeHtml(ep.name)}">
                            ${t('common.clone')}
                        </button>
                        <button class="btn btn-sm btn-danger delete-btn" data-name="${this.escapeHtml(ep.name)}">
                            ${t('common.delete')}
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
                <div>${t('endpoints.total')}: <strong>${pool.total}</strong></div>
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

        // Clone buttons
        document.querySelectorAll('.clone-btn').forEach(btn => {
            btn.addEventListener('click', () => this.cloneEndpoint(btn.dataset.name));
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

            notifications.success(t('notifications.endpointsReordered'));
            await this.loadEndpoints();
        } catch (error) {
            notifications.error(`${t('endpoints.failedToReorder')}: ${error.message}`);
            await this.loadEndpoints(); // Reload to reset order
        }
    }

    async switchEndpoint(name) {
        try {
            await api.switchEndpoint(name);
            notifications.success(`${t('notifications.endpointSwitched')} ${name}`);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error(`${t('endpoints.failedToSwitch')}: ${error.message}`);
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
            notifications.error(t('endpoints.failedToCopy'));
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

    showEndpointModal(endpoint, isClone = false) {
        const isEdit = !!endpoint && !isClone;
        const modalContainer = document.getElementById('modal-container');

        // For clone mode: show masked value like edit mode
        const apiKeyValue = endpoint ? '****' : '';
        const apiKeyPlaceholder = 'sk-...';
        const apiKeyHint = isEdit || isClone ? `<small class="text-muted">${t('endpoints.keepExistingKey')}</small>` : '';
        const cloneHiddenInput = isClone ? '<input type="hidden" name="isClone" value="true">' : '';
        const cloneFromValue = endpoint?.cloneFrom || '';
        const cloneFromInput = isClone && cloneFromValue ? `<input type="hidden" name="cloneFrom" value="${cloneFromValue}">` : '';

        modalContainer.innerHTML = `
            <div class="modal-overlay">
                <div class="modal">
                    <div class="modal-header">
                        <h3 class="modal-title">${isClone ? t('endpoints.cloneEndpoint') : (isEdit ? t('common.edit') : t('common.add'))} ${t('endpoints.title')}</h3>
                        <button class="modal-close" id="close-modal">×</button>
                    </div>
                    <div class="modal-body">
                        <form id="endpoint-form">
                            ${cloneHiddenInput}
                            ${cloneFromInput}
                            <div class="form-group">
                                <label class="form-label">${t('common.name')} *</label>
                                <input type="text" class="form-input" name="name" value="${endpoint ? this.escapeHtml(endpoint.name) : ''}" required ${isEdit ? 'readonly' : ''}>
                            </div>
                            <div class="form-group">
                                <label class="form-label">${t('endpoints.apiUrl')} *</label>
                                <input type="text" class="form-input" name="apiUrl" value="${endpoint ? this.escapeHtml(endpoint.apiUrl) : ''}" placeholder="${t('endpoints.apiUrlPlaceholder')}" required>
                            </div>
                            <div class="form-group">
                                <label class="form-label">${t('endpoints.apiKey')} *</label>
                                <input type="password" class="form-input" name="apiKey" value="${apiKeyValue}" placeholder="${apiKeyPlaceholder}" required>
                                ${apiKeyHint}
                            </div>
                            <div class="form-group">
                                <label class="form-label">${t('endpoints.transformer')} *</label>
                                <select class="form-select" name="transformer" required>
                                    <option value="claude" ${endpoint?.transformer === 'claude' ? 'selected' : ''}>${t('transformers.claude')}</option>
                                    <option value="openai" ${endpoint?.transformer === 'openai' ? 'selected' : ''}>${t('transformers.openai')}</option>
                                    <option value="openai2" ${endpoint?.transformer === 'openai2' ? 'selected' : ''}>${t('transformers.openai2')}</option>
                                    <option value="gemini" ${endpoint?.transformer === 'gemini' ? 'selected' : ''}>${t('transformers.gemini')}</option>
                                    <option value="deepseek" ${endpoint?.transformer === 'deepseek' ? 'selected' : ''}>${t('transformers.deepseek')}</option>
                                </select>
                            </div>
                            <div class="form-group">
                                <label class="form-label">${t('endpoints.model')}</label>
                                <div style="display: flex; gap: 8px;">
                                    <input type="text" class="form-input" name="model" id="model-input" value="${endpoint ? this.escapeHtml(endpoint.model || '') : ''}" placeholder="${t('endpoints.modelPlaceholder')}" style="flex: 1;">
                                    <button type="button" class="btn btn-secondary" id="fetch-models-btn" style="white-space: nowrap;">
                                        ${t('endpoints.fetchModels')}
                                    </button>
                                </div>
                                <small class="text-muted">${t('endpoints.fetchModelsHint')}</small>
                            </div>
                            <div class="form-group">
                                <label class="form-label">${t('endpoints.remark')}</label>
                                <textarea class="form-textarea" name="remark">${endpoint ? this.escapeHtml(endpoint.remark || '') : ''}</textarea>
                            </div>
                            <div class="form-group">
                                <label>
                                    <input type="checkbox" class="form-checkbox" name="enabled" ${endpoint?.enabled !== false ? 'checked' : ''}>
                                    ${t('common.enabled')}
                                </label>
                            </div>
                        </form>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" id="cancel-btn">${t('common.cancel')}</button>
                        <button class="btn btn-primary" id="save-btn">${isEdit ? t('common.update') : t('common.create')}</button>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('close-modal').addEventListener('click', () => this.closeModal());
        document.getElementById('cancel-btn').addEventListener('click', () => this.closeModal());
        document.getElementById('save-btn').addEventListener('click', () => {
            const isClone = !!document.querySelector('input[name="isClone"]');
            this.saveEndpoint(isEdit, endpoint?.name, isClone);
        });
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
            notifications.error(t('endpoints.enterApiUrlAndKey'));
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
                notifications.info(t('endpoints.noModelsFound'));
            }
        } catch (error) {
            notifications.error(`${t('endpoints.failedToFetchModels')}: ${error.message}`);
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
                    <h3 class="modal-title">${t('endpoints.selectModel')}</h3>
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
                    <button class="btn btn-secondary" id="cancel-model-btn">${t('common.cancel')}</button>
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
                notifications.success(`${t('notifications.modelSelected')} ${selectedModel}`);
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

    async saveEndpoint(isEdit, originalName, isClone = false) {
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

        // If editing and API key is ****, don't send it (keep existing)
        if ((isEdit || isClone) && data.apiKey === '****') {
            delete data.apiKey;
        }

        // For clone mode, add cloneFrom field if available
        const cloneFromInput = document.querySelector('input[name="cloneFrom"]');
        if (isClone && cloneFromInput && cloneFromInput.value) {
            data.cloneFrom = cloneFromInput.value;
        }

        try {
            if (isEdit) {
                await api.updateEndpoint(originalName, data);
                notifications.success(t('notifications.endpointUpdated'));
            } else {
                await api.createEndpoint(data);
                notifications.success(t('notifications.endpointCreated'));
            }

            this.closeModal();
            await this.loadEndpoints();
        } catch (error) {
            notifications.error(`${t('endpoints.failedToSave')}: ${error.message}`);
        }
    }

    async toggleEndpoint(name, enabled) {
        try {
            await api.toggleEndpoint(name, enabled);
            notifications.success(enabled ? t('notifications.endpointEnabled') : t('notifications.endpointDisabled'));
            await this.loadEndpoints();
        } catch (error) {
            notifications.error(`${t('endpoints.failedToToggle')}: ${error.message}`);
            await this.loadEndpoints(); // Reload to reset toggle state
        }
    }

    async testEndpoint(name) {
        try {
            notifications.info(t('endpoints.testing'));
            const result = await api.testEndpoint(name);

            if (result.success) {
                this.saveTestStatus(name, true);
                notifications.success(`${t('notifications.testSuccessful')} ${result.latency}ms`);
                this.showTestResultModal(name, result);
                await this.loadEndpoints(); // Refresh to show test status
            } else {
                this.saveTestStatus(name, false);
                notifications.error(`${t('notifications.testFailed')} ${result.error}`);
                await this.loadEndpoints(); // Refresh to show test status
            }
        } catch (error) {
            this.saveTestStatus(name, false);
            notifications.error(`${t('endpoints.failedToTest')}: ${error.message}`);
            await this.loadEndpoints(); // Refresh to show test status
        }
    }

    showTestResultModal(name, result) {
        const modalContainer = document.getElementById('modal-container');

        modalContainer.innerHTML = `
            <div class="modal-overlay">
                <div class="modal">
                    <div class="modal-header">
                        <h3 class="modal-title">${t('endpoints.testResult')}: ${this.escapeHtml(name)}</h3>
                        <button class="modal-close" id="close-modal">×</button>
                    </div>
                    <div class="modal-body">
                        <div class="mb-2">
                            <strong>${t('common.status')}:</strong> <span class="badge badge-success">${t('common.success')}</span>
                        </div>
                        <div class="mb-2">
                            <strong>${t('endpoints.latency')}:</strong> ${result.latency}ms
                        </div>
                        <div class="mb-2">
                            <strong>${t('endpoints.response')}:</strong>
                            <div class="code-block mt-1">${this.escapeHtml(result.response || t('endpoints.noResponse'))}</div>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-primary" id="close-btn">${t('common.close')}</button>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('close-modal').addEventListener('click', () => this.closeModal());
        document.getElementById('close-btn').addEventListener('click', () => this.closeModal());
    }

    async deleteEndpoint(name) {
        if (!confirm(t('endpoints.confirmDelete').replace('{name}', name))) {
            return;
        }

        try {
            await api.deleteEndpoint(name);
            notifications.success(t('notifications.endpointDeleted'));
            await this.loadEndpoints();
        } catch (error) {
            notifications.error(`${t('endpoints.failedToDelete')}: ${error.message}`);
        }
    }

    async cloneEndpoint(name) {
        const endpoint = this.endpoints.find(ep => ep.name === name);
        if (!endpoint) {
            notifications.error(t('endpoints.failedToClone'));
            return;
        }

        // Extract base name and add (Copy) suffix
        const baseName = name.replace(/\(Copy\)(?:\s+\d+)?$/, '').trim();
        let newName = `${baseName} (Copy)`;
        let counter = 1;
        while (this.endpoints.some(ep => ep.name === newName)) {
            newName = `${baseName} (Copy) ${counter}`;
            counter++;
        }

        // Create cloned endpoint - don't include apiKey, use cloneFrom instead
        const clonedEndpoint = {
            name: newName,
            apiUrl: endpoint.apiUrl,
            transformer: endpoint.transformer,
            model: endpoint.model,
            remark: endpoint.remark,
            enabled: endpoint.enabled,
            cloneFrom: name  // Reference to source endpoint
        };

        try {
            this.showEndpointModal(clonedEndpoint, true);
            notifications.success(t('notifications.endpointCloned'));
        } catch (error) {
            notifications.error(`${t('endpoints.failedToClone')}: ${error.message}`);
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
                            <h3 class="modal-title">${t('endpoints.tokenPoolTitle')} ${this.escapeHtml(endpointName)}</h3>
                            <button class="modal-close" id="close-modal">×</button>
                        </div>
                        <div class="modal-body">
                            <div class="mb-2" style="font-size: 13px;">
                                <strong>${t('endpoints.total')}:</strong> ${stats.total || 0}
                                <span style="margin-left: 12px;"><strong>${t('endpoints.active')}:</strong> ${stats.active || 0}</span>
                                <span style="margin-left: 12px;"><strong>${t('endpoints.expiring')}:</strong> ${stats.expiring || 0}</span>
                                <span style="margin-left: 12px;"><strong>${t('endpoints.needRefresh')}:</strong> ${stats.needRefresh || 0}</span>
                                <span style="margin-left: 12px;"><strong>${t('endpoints.expired')}:</strong> ${stats.expired || 0}</span>
                                <span style="margin-left: 12px;"><strong>${t('endpoints.invalid')}:</strong> ${stats.invalid || 0}</span>
                            </div>

                            <div class="form-group">
                                <label class="form-label">${t('endpoints.batchImportJson')}</label>
                                <textarea class="form-textarea" id="token-import-json" style="min-height: 140px;" placeholder='${t('endpoints.jsonPasteHint')}'></textarea>
                                <label style="display: inline-flex; gap: 8px; align-items: center; margin-top: 8px;">
                                    <input type="checkbox" id="token-import-overwrite">
                                    ${t('endpoints.overwriteExisting')}
                                </label>
                                <div style="margin-top: 8px;">
                                    <button class="btn btn-primary" id="token-import-btn">${t('common.import')}</button>
                                </div>
                            </div>

                            <div class="table-container">
                                <table class="table">
                                    <thead>
                                        <tr>
                                            <th>${t('endpoints.id')}</th>
                                            <th>${t('endpoints.account')}</th>
                                            <th>${t('endpoints.email')}</th>
                                            <th>${t('common.status')}</th>
                                            <th>${t('endpoints.expiresAt')}</th>
                                            <th>${t('endpoints.lastError')}</th>
                                            <th>${t('common.actions')}</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        ${this.renderCredentialRows(credentials)}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                        <div class="modal-footer">
                            <button class="btn btn-secondary" id="refresh-token-pool-btn">${t('common.refresh')}</button>
                            <button class="btn btn-secondary" id="close-token-pool-btn">${t('common.close')}</button>
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
            notifications.error(`${t('endpoints.failedToLoadTokenPool')}: ${error.message}`);
        }
    }

    renderCredentialRows(credentials) {
        if (!credentials || credentials.length === 0) {
            return `<tr><td colspan="7" class="text-center text-muted">${t('endpoints.noCredentials')}</td></tr>`;
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
                            ${t('common.enabled')}
                        </label>
                        <button class="btn btn-sm btn-secondary token-update-btn" data-id="${cred.id}">${t('common.update')}</button>
                        <button class="btn btn-sm btn-secondary token-activate-btn" data-id="${cred.id}">${t('endpoints.activate')}</button>
                        <button class="btn btn-sm btn-danger token-delete-btn" data-id="${cred.id}">${t('common.delete')}</button>
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
            notifications.warning(t('endpoints.pleasePasteJson'));
            return;
        }

        let payload;
        try {
            payload = JSON.parse(raw);
        } catch {
            notifications.error(t('endpoints.invalidJson'));
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
            notifications.success(t('notifications.importDone').replace('{created}', result.created || 0).replace('{updated}', result.updated || 0).replace('{skipped}', result.skipped || 0).replace('{failed}', result.failed || 0));
            jsonInput.value = '';
            await this.showTokenPoolModal(endpointName);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error(`${t('endpoints.failedToImport')}: ${error.message}`);
        }
    }

    async updateCredentialEnabled(endpointName, credentialId, enabled) {
        try {
            await api.updateEndpointCredential(endpointName, credentialId, { enabled });
            notifications.success(enabled ? t('notifications.credentialEnabled') : t('notifications.credentialDisabled'));
            await this.showTokenPoolModal(endpointName);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error(`${t('endpoints.failedToUpdateCredential')}: ${error.message}`);
            await this.showTokenPoolModal(endpointName);
        }
    }

    async activateCredential(endpointName, credentialId) {
        try {
            await api.updateEndpointCredential(endpointName, credentialId, { status: 'active' });
            notifications.success(t('notifications.credentialActivated'));
            await this.showTokenPoolModal(endpointName);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error(`${t('endpoints.failedToActivateCredential')}: ${error.message}`);
        }
    }

    async updateCredentialToken(endpointName, credentialId) {
        const accessToken = prompt(t('endpoints.enterAccessToken'));
        if (!accessToken) {
            return;
        }

        const expiresAt = prompt(t('endpoints.enterExpiresAt'), '');
        const payload = {
            accessToken: accessToken.trim(),
            status: 'active'
        };
        if (expiresAt && expiresAt.trim()) {
            payload.expiresAt = expiresAt.trim();
        }

        try {
            await api.updateEndpointCredential(endpointName, credentialId, payload);
            notifications.success(t('notifications.tokenUpdated'));
            await this.showTokenPoolModal(endpointName);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error(`${t('endpoints.failedToUpdateToken')}: ${error.message}`);
        }
    }

    async deleteCredential(endpointName, credentialId) {
        if (!confirm(t('endpoints.confirmDeleteCredential').replace('{id}', credentialId))) {
            return;
        }

        try {
            await api.deleteEndpointCredential(endpointName, credentialId);
            notifications.success(t('notifications.credentialDeleted'));
            await this.showTokenPoolModal(endpointName);
            await this.loadEndpoints();
        } catch (error) {
            notifications.error(`${t('endpoints.failedToDeleteCredential')}: ${error.message}`);
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