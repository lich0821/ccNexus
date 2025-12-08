import { t } from '../i18n/index.js';

export function initUI() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="header">
            <div style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
                <div>
                    <h1>🚀 ${t('app.title')}</h1>
                    <p>${t('header.title')}</p>
                </div>
                <div style="display: flex; gap: 15px; align-items: center;">
                    <div class="port-display" onclick="window.showEditPortModal()" title="${t('header.port')}">
                        <span style="color: #666; font-size: 14px;">${t('header.port')}: </span>
                        <span class="port-number" id="proxyPort">3000</span>
                    </div>
                    <div style="display: flex; gap: 10px;">
                        <button class="header-link" onclick="window.openGitHub()" title="${t('header.githubRepo')}">
                            <svg width="24" height="24" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
                            </svg>
                        </button>
                        <button class="header-link about-btn" id="aboutBtn" onclick="window.showWelcomeModal()" title="${t('header.about')}">
                            📖
                            <span class="update-badge" id="updateBadge"></span>
                        </button>
                        <button class="header-link" onclick="window.showSettingsModal()" title="${t('settings.title')}">
                            ⚙️
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <div class="container">
            <!-- Statistics -->
            <div class="card">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                    <h2 style="margin: 0;">📊 ${t('statistics.title')}</h2>
                    <div class="stats-tabs">
                        <button class="stats-tab-btn active" data-period="daily" onclick="window.switchStatsPeriod('daily')">
                            📅 ${t('statistics.daily')}
                        </button>
                        <button class="stats-tab-btn" data-period="yesterday" onclick="window.switchStatsPeriod('yesterday')">
                            📆 ${t('statistics.yesterday')}
                        </button>
                        <button class="stats-tab-btn" data-period="weekly" onclick="window.switchStatsPeriod('weekly')">
                            📊 ${t('statistics.weekly')}
                        </button>
                        <button class="stats-tab-btn" data-period="monthly" onclick="window.switchStatsPeriod('monthly')">
                            📈 ${t('statistics.monthly')}
                        </button>
                        <button class="stats-tab-btn" data-period="history" onclick="window.switchStatsPeriod('history')">
                            📚 ${t('statistics.history')}
                        </button>
                    </div>
                </div>

                <!-- Current Stats View -->
                <div id="currentStatsView">
                    <div class="stats-grid">
                    <div class="stat-box">
                        <div class="stat-header">
                            <div class="stat-label">${t('statistics.endpoints')}</div>
                        </div>
                        <div class="stat-value">
                            <span id="activeEndpointsDisplay" class="stat-primary">0</span>
                            <span class="stat-secondary"> / </span>
                            <span id="totalEndpointsDisplay" class="stat-secondary">0</span>
                        </div>
                        <div class="stat-detail">${t('statistics.activeTotal')}</div>
                    </div>
                    <div class="stat-box">
                        <div class="stat-header">
                            <div class="stat-label">${t('statistics.totalRequests')}</div>
                            <span class="trend" id="requestsTrend">→ 0%</span>
                        </div>
                        <div class="stat-value">
                            <span id="periodTotalRequests">0</span>
                        </div>
                        <div class="stat-detail">
                            <span id="periodSuccess">0</span>
                            <span class="stat-text"> ${t('statistics.success')}</span>
                            <span class="stat-divider">/</span>
                            <span id="periodFailed">0</span>
                            <span class="stat-text"> ${t('statistics.failed')}</span>
                        </div>
                    </div>
                    <div class="stat-box">
                        <div class="stat-header">
                            <div class="stat-label">${t('statistics.totalTokens')}</div>
                            <span class="trend" id="tokensTrend">→ 0%</span>
                        </div>
                        <div class="stat-value">
                            <span id="periodTotalTokens">0</span>
                        </div>
                        <div class="stat-detail">
                            <span id="periodInputTokens">0</span>
                            <span class="stat-text"> ${t('statistics.in')}</span>
                            <span class="stat-divider">/</span>
                            <span id="periodOutputTokens">0</span>
                            <span class="stat-text"> ${t('statistics.out')}</span>
                        </div>
                    </div>
                </div>

                <!-- Hidden cumulative stats for endpoint cards -->
                <div style="display: none;">
                    <span id="activeEndpoints">0</span>
                    <span id="totalEndpoints">0</span>
                    <span id="totalRequests">0</span>
                    <span id="successRequests">0</span>
                    <span id="failedRequests">0</span>
                    <span id="totalTokens">0</span>
                    <span id="totalInputTokens">0</span>
                    <span id="totalOutputTokens">0</span>
                </div>
                </div>
            </div>

            <!-- History Modal (弹窗) -->
            <div id="historyModal" class="modal" style="display: none;">
                <div class="modal-content">
                    <div class="modal-header">
                        <h2>📚 ${t('history.title')}</h2>
                        <button class="modal-close" onclick="window.closeHistoryModal()">&times;</button>
                    </div>
                    <div class="modal-body">
                        <div class="history-selector">
                            <label>${t('history.selectMonth')}:</label>
                            <select id="historyMonthSelect"></select>
                        </div>

                        <div class="stats-grid">
                            <div class="stat-box">
                                <div class="stat-header">
                                    <div class="stat-label">${t('statistics.totalRequests')}</div>
                                    <span class="trend" id="historyRequestsTrend">→ 0%</span>
                                </div>
                                <div class="stat-value">
                                    <span id="historyTotalRequests">0</span>
                                </div>
                                <div class="stat-detail">
                                    <span id="historySuccess">0</span>
                                    <span class="stat-text"> ${t('statistics.success')}</span>
                                    <span class="stat-divider">/</span>
                                    <span id="historyFailed">0</span>
                                    <span class="stat-text"> ${t('statistics.failed')}</span>
                                </div>
                            </div>
                            <div class="stat-box">
                                <div class="stat-header">
                                    <div class="stat-label">${t('statistics.totalTokens')}</div>
                                    <span class="trend" id="historyTokensTrend">→ 0%</span>
                                </div>
                                <div class="stat-value">
                                    <span id="historyTotalTokens">0</span>
                                </div>
                                <div class="stat-detail">
                                    <span id="historyInputTokens">0</span>
                                    <span class="stat-text"> ${t('statistics.in')}</span>
                                    <span class="stat-divider">/</span>
                                    <span id="historyOutputTokens">0</span>
                                    <span class="stat-text"> ${t('statistics.out')}</span>
                                </div>
                            </div>
                        </div>

                        <div class="history-details">
                            <h3>${t('history.dailyDetails')}</h3>
                            <div class="table-container">
                                <table id="historyDailyTable">
                                    <thead>
                                        <tr>
                                            <th>${t('history.date')}</th>
                                            <th>${t('history.requests')}</th>
                                            <th>${t('history.errors')}</th>
                                            <th>${t('history.inputTokens')}</th>
                                            <th>${t('history.outputTokens')}</th>
                                            <th>${t('history.totalTokens')}</th>
                                        </tr>
                                    </thead>
                                    <tbody></tbody>
                                </table>
                            </div>
                        </div>

                        <div id="historyError" class="error-message" style="display: none;"></div>
                    </div>
                </div>
            </div>

            <!-- Endpoints -->
            <div class="card">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                    <div style="display: flex; align-items: center; gap: 15px;">
                        <h2 style="margin: 0;">🔗 ${t('endpoints.title')}</h2>
                        <button class="endpoint-toggle-btn" onclick="window.toggleEndpointPanel()">
                            <span id="endpointToggleIcon">🔼</span> <span id="endpointToggleText">${t('endpoints.collapse')}</span>
                        </button>
                    </div>
                    <div style="display: flex; gap: 10px;">
                        <button class="btn btn-secondary" onclick="window.showDataSyncDialog()">
                            🔄 ${t('webdav.dataSync')}
                        </button>
                        <button class="btn btn-primary" onclick="window.showAddEndpointModal()">
                            ➕ ${t('header.addEndpoint')}
                        </button>
                    </div>
                </div>
                <div id="endpointPanel" class="endpoint-panel">
                    <div id="endpointList" class="endpoint-list">
                        <div class="loading">${t('endpoints.title')}...</div>
                    </div>
                </div>
            </div>

            <!-- Logs Panel -->
            <div class="card">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                    <div style="display: flex; align-items: center; gap: 15px;">
                        <h2 style="margin: 0;">📋 ${t('logs.title')}</h2>
                        <button class="endpoint-toggle-btn" onclick="window.toggleLogPanel()">
                            <span id="logToggleIcon">🔼</span> <span id="logToggleText">${t('logs.collapse')}</span>
                        </button>
                    </div>
                    <div style="display: flex; gap: 10px;">
                        <select id="logLevel" class="log-level-select-btn" onchange="window.changeLogLevel()">
                            <option value="0">🔍 ${t('logs.levels.0')}</option>
                            <option value="1" selected>ℹ️ ${t('logs.levels.1')}</option>
                            <option value="2">⚠️ ${t('logs.levels.2')}</option>
                            <option value="3">❌ ${t('logs.levels.3')}</option>
                        </select>
                        <button class="btn btn-secondary btn-sm" onclick="window.copyLogs()">
                            📋 ${t('logs.copy')}
                        </button>
                        <button class="btn btn-secondary btn-sm" onclick="window.clearLogs()">
                            🗑️ ${t('logs.clear')}
                        </button>
                    </div>
                </div>
                <div id="logPanel" class="log-panel">
                    <textarea id="logContent" class="log-textarea" readonly></textarea>
                </div>
            </div>
        </div>

        <!-- Footer -->
        <div class="footer">
            <div class="footer-content">
                <div class="footer-left">
                    <span style="opacity: 0.8;">© 2025 ccNexus</span>
                </div>
                <div class="footer-center">
                    <div class="tips-container">
                        <span id="scrollingTip" class="tip-scroll"></span>
                    </div>
                </div>
                <div class="footer-right">
                    <span style="opacity: 0.7; margin-right: 5px;">v</span>
                    <span id="appVersion" style="font-weight: 500;">1.0.0</span>
                </div>
            </div>
        </div>

        <!-- Add/Edit Endpoint Modal -->
        <div id="endpointModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2 id="modalTitle">➕ ${t('modal.addEndpoint')}</h2>
                    <button class="modal-close" onclick="window.closeModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="form-group">
                        <label><span class="required" style="color: #ff4444;">* </span>${t('modal.name')}</label>
                        <input type="text" id="endpointName" placeholder="${t('modal.namePlaceholder')}">
                    </div>
                    <div class="form-group">
                        <label><span class="required" style="color: #ff4444;">* </span>${t('modal.apiUrl')}</label>
                        <input type="text" id="endpointUrl" placeholder="${t('modal.apiUrlPlaceholder')}">
                    </div>
                    <div class="form-group">
                        <label><span class="required" style="color: #ff4444;">* </span>${t('modal.apiKey')}</label>
                        <div class="password-input-wrapper">
                            <input type="password" id="endpointKey" placeholder="${t('modal.apiKeyPlaceholder')}">
                            <button type="button" class="password-toggle" onclick="window.togglePasswordVisibility()" title="${t('modal.togglePassword')}">
                                <svg id="eyeIcon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
                                    <circle cx="12" cy="12" r="3"></circle>
                                </svg>
                            </button>
                        </div>
                    </div>
                    <div class="form-group">
                        <label><span class="required" style="color: #ff4444;">* </span>${t('modal.transformer')}</label>
                        <select id="endpointTransformer" onchange="window.handleTransformerChange()">
                            <option value="claude">Claude (Default)</option>
                            <option value="openai">OpenAI</option>
                            <option value="openai2">OpenAI2 (Responses API)</option>
                            <option value="gemini">Gemini</option>
                        </select>
                        <p style="color: #666; font-size: 12px; margin-top: 5px;">
                            ${t('modal.transformerHelp')}
                        </p>
                    </div>
                    <div class="form-group" id="modelFieldGroup" style="display: block;">
                        <label><span class="required" id="modelRequired" style="display: none; color: #ff4444;">* </span>${t('modal.model')}</label>
                        <div class="model-input-wrapper">
                            <div class="model-select-container">
                                <input type="text" id="endpointModel" placeholder="${t('modal.modelPlaceholder')}" autocomplete="off">
                                <button type="button" class="model-dropdown-toggle" onclick="window.toggleModelDropdown()">
                                    <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor">
                                        <path d="M2 4L6 8L10 4" stroke="currentColor" stroke-width="2" fill="none"/>
                                    </svg>
                                </button>
                                <div class="model-dropdown" id="modelDropdown"></div>
                            </div>
                            <button type="button" class="btn btn-secondary" id="fetchModelsBtn" onclick="window.fetchModels()" title="${t('modal.fetchModels')}">
                                <span id="fetchModelsIcon">${t('modal.fetchModelsBtn')}</span>
                            </button>
                        </div>
                        <p style="color: #666; font-size: 12px; margin-top: 5px;" id="modelHelpText">
                            ${t('modal.modelHelp')}
                        </p>
                    </div>
                    <div class="form-group">
                        <label>${t('modal.remark')}</label>
                        <input type="text" id="endpointRemark" placeholder="${t('modal.remarkHelp')}">
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary" onclick="window.closeModal()">${t('modal.cancel')}</button>
                    <button class="btn btn-primary" onclick="window.saveEndpoint()">${t('modal.save')}</button>
                </div>
            </div>
        </div>

        <!-- Edit Port Modal -->
        <div id="portModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2>⚙️ ${t('modal.changePort')}</h2>
                    <button class="modal-close" onclick="window.closePortModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="form-group">
                        <label>${t('modal.port')} (1-65535)</label>
                        <input type="number" id="portInput" min="1" max="65535" placeholder="3000">
                    </div>
                    <p style="color: #666; font-size: 14px; margin-top: 10px;">
                        ⚠️ ${t('modal.portNote')}
                    </p>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary" onclick="window.closePortModal()">${t('modal.cancel')}</button>
                    <button class="btn btn-primary" onclick="window.savePort()">${t('modal.save')}</button>
                </div>
            </div>
        </div>

        <!-- Welcome Modal -->
        <div id="welcomeModal" class="modal">
            <div class="modal-content" style="max-width: min(600px, 90vw);">
                <div class="modal-header">
                    <h2>👋 ${t('welcome.title')}</h2>
                </div>
                <div class="modal-body">
                    <p style="font-size: 16px; line-height: 1.6; margin-bottom: 20px;">
                        ${t('welcome.message')}
                    </p>

                    <div style="display: flex; justify-content: center; gap: 30px; margin: 30px 0;">
                        <div style="text-align: center;">
                            <img src="/WeChat.jpg" alt="WeChat QR Code" style="width: 200px; height: 200px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
                            <p style="margin-top: 10px; color: #666; font-size: 14px;">${t('welcome.qrCodeTip')}</p>
                        </div>
                        <div style="text-align: center;">
                            <img src="/chat.jpg" alt="Chat Group QR Code" style="width: 200px; height: 200px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
                            <p style="margin-top: 10px; color: #666; font-size: 14px;">${t('welcome.chatGroupTip')}</p>
                        </div>
                    </div>

                    <div style="display: flex; gap: 15px; justify-content: center; margin-top: 20px;">
                        <button class="btn btn-primary" onclick="window.openArticle()">
                            ${t('welcome.readArticle')}
                        </button>
                        <button class="btn btn-secondary" onclick="window.showChangelogModal()">
                            ${t('welcome.changelog')}
                        </button>
                        <button class="btn btn-secondary check-update-btn" onclick="window.checkForUpdates()">
                            🔄 ${t('update.checkForUpdates')}
                            <span class="update-badge" id="checkUpdateBadge"></span>
                        </button>
                    </div>
                </div>
                <div class="modal-footer" style="display: flex; justify-content: flex-end; align-items: center; gap: 20px;">
                    <label style="display: flex; align-items: center; cursor: pointer;">
                        <input type="checkbox" id="dontShowAgain" style="margin-right: 8px;">
                        <span style="font-size: 14px; color: #666;">${t('welcome.dontShow')}</span>
                    </label>
                    <button class="btn btn-primary" onclick="window.closeWelcomeModal()">${t('welcome.getStarted')}</button>
                </div>
            </div>
        </div>

        <!-- Test Result Modal -->
        <div id="testResultModal" class="modal">
            <div class="modal-content" style="max-width: min(600px, 90vw);">
                <div class="modal-header">
                    <h2 id="testResultTitle">🧪 ${t('test.title')}</h2>
                    <button class="modal-close" onclick="window.closeTestResultModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <div id="testResultContent" style="font-size: 14px; line-height: 1.6;">
                        <!-- Test result will be inserted here -->
                    </div>
                </div>
            </div>
        </div>

        <!-- Changelog Modal -->
        <div id="changelogModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2>📋 ${t('changelog.title')}</h2>
                    <button class="modal-close" onclick="window.closeChangelogModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <div id="changelogContent" style="font-size: 14px; line-height: 1.8;">
                    </div>
                </div>
            </div>
        </div>

        <!-- Error Toast -->
        <div id="errorToast" class="error-toast">
            <div class="error-toast-content">
                <span class="error-toast-icon">⚠️</span>
                <span id="errorToastMessage"></span>
            </div>
        </div>

        <!-- Confirm Dialog -->
        <div id="confirmDialog" class="modal">
            <div class="confirm-dialog-content">
                <div class="confirm-body">
                    <div class="confirm-icon">
                        <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                            <path d="M12 9v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                    </div>
                    <div class="confirm-content">
                        <h4 class="confirm-title">${t('common.confirmDeleteTitle')}</h4>
                        <p id="confirmMessage" class="confirm-message"></p>
                    </div>
                </div>
                <div class="confirm-divider"></div>
                <div class="confirm-footer">
                    <button class="btn-confirm-delete" onclick="window.acceptConfirm()">${t('common.delete')}</button>
                    <button class="btn-confirm-cancel" onclick="window.cancelConfirm()">${t('common.cancel')}</button>
                </div>
            </div>
        </div>

        <!-- Close Action Dialog -->
        <div id="closeActionDialog" class="modal">
            <div class="confirm-dialog-content">
                <div class="confirm-body">
                    <div class="confirm-icon" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);">
                        <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                            <path d="M6 18L18 6M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                    </div>
                    <div class="confirm-content">
                        <h4 class="confirm-title">关闭窗口</h4>
                        <p class="confirm-message">您希望如何处理？</p>
                    </div>
                </div>
                <div class="confirm-divider"></div>
                <div class="confirm-footer">
                    <button class="btn-confirm-delete" onclick="window.quitApplication()">退出程序</button>
                    <button class="btn-confirm-cancel" onclick="window.minimizeToTray()">最小化到托盘</button>
                </div>
            </div>
        </div>

        <!-- Settings Modal -->
        <div id="settingsModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2>⚙️ ${t('settings.title')}</h2>
                    <button class="modal-close" onclick="window.closeSettingsModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="form-group">
                        <label>🌍 ${t('settings.language')}</label>
                        <select id="settingsLanguage">
                            <option value="zh-CN">${t('settings.languages.zh-CN')}</option>
                            <option value="en">${t('settings.languages.en')}</option>
                        </select>
                        <p style="color: #666; font-size: 12px; margin-top: 5px;">
                            ${t('settings.languageHelp')}
                        </p>
                    </div>
                    <div class="form-group">
                        <label>🌓 ${t('settings.theme')}</label>
                        <div style="display: flex; align-items: center; gap: 12px;">
                            <select id="settingsTheme" style="flex: 1;">
                                <option value="light">${t('settings.themes.light')}</option>
                                <option value="dark">${t('settings.themes.dark')}</option>
                                <option value="green">${t('settings.themes.green')}</option>
                                <option value="starry">${t('settings.themes.starry')}</option>
                                <option value="sakura">${t('settings.themes.sakura')}</option>
                                <option value="sunset">${t('settings.themes.sunset')}</option>
                                <option value="ocean">${t('settings.themes.ocean')}</option>
                                <option value="mocha">${t('settings.themes.mocha')}</option>
                                <option value="cyberpunk">${t('settings.themes.cyberpunk')}</option>
                                <option value="aurora">${t('settings.themes.aurora')}</option>
                                <option value="holographic">${t('settings.themes.holographic')}</option>
                                <option value="quantum">${t('settings.themes.quantum')}</option>
                            </select>
                            <div style="display: flex; align-items: center; gap: 8px; white-space: nowrap;" title="${t('settings.themeAutoHelp')}">
                                <span style="font-size: 13px; color: var(--text-secondary);">${t('settings.themeAuto')}</span>
                                <label class="toggle-switch" style="width: 40px; height: 20px; margin-top: 7px;">
                                    <input type="checkbox" id="settingsThemeAuto">
                                    <span class="toggle-slider" style="border-radius: 20px;"></span>
                                </label>
                            </div>
                        </div>
                        <p style="color: #666; font-size: 12px; margin-top: 5px;">
                            ${t('settings.themeHelp')}
                        </p>
                    </div>
                    <div class="form-group">
                        <label>✨ ${t('settings.closeWindowBehavior')}</label>
                        <select id="settingsCloseWindowBehavior">
                            <option value="quit">${t('settings.closeWindowOptions.quit')}</option>
                            <option value="ask">${t('settings.closeWindowOptions.ask')}</option>
                            <option value="minimize">${t('settings.closeWindowOptions.minimize')}</option>
                        </select>
                        <p style="color: #666; font-size: 12px; margin-top: 5px;">
                            ${t('settings.closeWindowBehaviorHelp')}
                        </p>
                    </div>
                    <div class="form-group">
                        <label>🔄 ${t('update.autoCheck')}</label>
                        <select id="check-interval">
                            <option value="1">${t('update.everyHour')}</option>
                            <option value="24">${t('update.everyDay')}</option>
                            <option value="168">${t('update.everyWeek')}</option>
                            <option value="720">${t('update.everyMonth')}</option>
                            <option value="0">${t('update.noAutoCheck')}</option>
                        </select>
                        <p style="color: #666; font-size: 12px; margin-top: 5px;">
                            ${t('update.autoCheckHelp')}
                        </p>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary" onclick="window.closeSettingsModal()">${t('settings.cancel')}</button>
                    <button class="btn btn-primary" onclick="window.saveSettings()">${t('settings.save')}</button>
                </div>
            </div>
        </div>

        <!-- Auto Theme Config Modal -->
        <div id="autoThemeConfigModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2>🌓 ${t('settings.autoThemeConfigTitle')}</h2>
                    <button class="modal-close" onclick="window.closeAutoThemeConfigModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <p style="color: var(--text-secondary); font-size: 14px; margin-bottom: 20px;">
                        ${t('settings.autoThemeConfigDesc')}
                    </p>
                    <div class="form-group">
                        <label>☀️ ${t('settings.lightThemeLabel')}</label>
                        <select id="autoLightTheme">
                            <option value="light">${t('settings.themes.light')}</option>
                            <option value="green">${t('settings.themes.green')}</option>
                            <option value="sakura">${t('settings.themes.sakura')}</option>
                            <option value="sunset">${t('settings.themes.sunset')}</option>
                            <option value="ocean">${t('settings.themes.ocean')}</option>
                            <option value="mocha">${t('settings.themes.mocha')}</option>
                        </select>
                        <p style="color: var(--text-secondary); font-size: 12px; margin-top: 5px;">
                            ${t('settings.lightThemeHelp')}
                        </p>
                    </div>
                    <div class="form-group">
                        <label>🌙 ${t('settings.darkThemeLabel')}</label>
                        <select id="autoDarkTheme">
                            <option value="dark">${t('settings.themes.dark')}</option>
                            <option value="starry">${t('settings.themes.starry')}</option>
                            <option value="cyberpunk">${t('settings.themes.cyberpunk')}</option>
                            <option value="aurora">${t('settings.themes.aurora')}</option>
                            <option value="holographic">${t('settings.themes.holographic')}</option>
                            <option value="quantum">${t('settings.themes.quantum')}</option>
                        </select>
                        <p style="color: var(--text-secondary); font-size: 12px; margin-top: 5px;">
                            ${t('settings.darkThemeHelp')}
                        </p>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary" onclick="window.closeAutoThemeConfigModal()">${t('settings.cancel')}</button>
                    <button class="btn btn-primary" onclick="window.saveAutoThemeConfig()">${t('settings.save')}</button>
                </div>
            </div>
        </div>
    `;

    setupModalEventListeners();
}

function setupModalEventListeners() {
    // Close modals on background click (endpointModal and portModal do NOT close on background click)

    document.getElementById('welcomeModal').addEventListener('click', (e) => {
        if (e.target.id === 'welcomeModal') {
            window.closeWelcomeModal();
        }
    });

    document.getElementById('testResultModal').addEventListener('click', (e) => {
        if (e.target.id === 'testResultModal') {
            window.closeTestResultModal();
        }
    });
}

export async function changeLanguage(lang) {
    try {
        await window.go.main.App.SetLanguage(lang);
        location.reload();
    } catch (error) {
        console.error('Failed to change language:', error);
    }
}
