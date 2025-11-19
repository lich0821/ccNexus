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
                        <button class="header-link" onclick="window.showWelcomeModal()" title="${t('header.about')}">
                            📖
                        </button>
                        <div class="lang-switcher">
                            <svg width="24" height="24" viewBox="0 0 1024 1024" fill="currentColor">
                                <path d="M757.205333 473.173333c5.333333 0 10.453333 2.090667 14.250667 5.717334a19.029333 19.029333 0 0 1 5.888 13.738666v58.154667h141.184c11.093333 0 20.138667 8.704 20.138667 19.413333v232.704a19.797333 19.797333 0 0 1-20.138667 19.413334h-141.184v96.981333a19.754667 19.754667 0 0 1-20.138667 19.370667H716.8a20.565333 20.565333 0 0 1-14.250667-5.674667 19.029333 19.029333 0 0 1-5.888-13.696v-96.981333h-141.141333a20.565333 20.565333 0 0 1-14.250667-5.674667 19.029333 19.029333 0 0 1-5.930666-13.738667v-232.704c0-5.12 2.133333-10.112 5.930666-13.738666a20.565333 20.565333 0 0 1 14.250667-5.674667h141.141333v-58.154667c0-5.162667 2.133333-10.112 5.888-13.738666a20.565333 20.565333 0 0 1 14.250667-5.674667h40.362667zM192.597333 628.394667c22.272 0 40.32 17.365333 40.32 38.826666v38.741334c0 40.618667 32.512 74.368 74.624 77.397333l6.058667 0.213333h80.64c21.930667 0.469333 39.424 17.706667 39.424 38.784 0 21.077333-17.493333 38.314667-39.424 38.784H313.6c-89.088 0-161.28-69.461333-161.28-155.178666v-38.741334c0-21.461333 18.005333-38.826667 40.277333-38.826666z m504.106667 0h-80.64v116.394666h80.64v-116.394666z m161.28 0h-80.64v116.394666h80.64v-116.394666zM320.170667 85.333333c8.234667 0 15.658667 4.778667 18.773333 12.202667H338.773333l161.322667 387.84c2.517333 5.973333 1.706667 12.8-2.005333 18.090667a20.394667 20.394667 0 0 1-16.725334 8.533333h-43.52a20.181333 20.181333 0 0 1-18.688-12.202667L375.850667 395.648H210.901333l-43.264 104.149333A20.181333 20.181333 0 0 1 148.906667 512H105.514667a20.394667 20.394667 0 0 1-16.725334-8.533333 18.773333 18.773333 0 0 1-2.005333-18.090667l161.28-387.84A20.181333 20.181333 0 0 1 266.88 85.333333h53.290667zM716.8 162.901333c42.794667 0 83.84 16.341333 114.090667 45.44a152.234667 152.234667 0 0 1 47.232 109.738667v38.741333c-0.469333 21.077333-18.389333 37.930667-40.32 37.930667s-39.808-16.853333-40.32-37.930667v-38.741333c0-20.608-8.490667-40.32-23.637334-54.869333a82.304 82.304 0 0 0-57.045333-22.741334h-80.64c-21.888-0.469333-39.424-17.706667-39.424-38.784 0-21.077333 17.493333-38.314667 39.424-38.784h80.64z m-423.424 34.304L243.2 318.037333h100.48L293.418667 197.205333z"/>
                            </svg>
                            <div class="lang-menu">
                                <div class="lang-item" onclick="window.changeLanguage('en')">English</div>
                                <div class="lang-item" onclick="window.changeLanguage('zh-CN')">简体中文</div>
                            </div>
                        </div>
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
                            <span class="stat-divider"> / </span>
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
                            <span class="stat-divider"> / </span>
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
                                </div>
                                <div class="stat-value">
                                    <span id="historyTotalRequests">0</span>
                                </div>
                                <div class="stat-detail">
                                    <span id="historySuccess">0</span>
                                    <span class="stat-text"> ${t('statistics.success')}</span>
                                    <span class="stat-divider"> / </span>
                                    <span id="historyFailed">0</span>
                                    <span class="stat-text"> ${t('statistics.failed')}</span>
                                </div>
                            </div>
                            <div class="stat-box">
                                <div class="stat-header">
                                    <div class="stat-label">${t('statistics.totalTokens')}</div>
                                </div>
                                <div class="stat-value">
                                    <span id="historyTotalTokens">0</span>
                                </div>
                                <div class="stat-detail">
                                    <span id="historyInputTokens">0</span>
                                    <span class="stat-text"> ${t('statistics.in')}</span>
                                    <span class="stat-divider"> / </span>
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
                    <h2 style="margin: 0;">🔗 ${t('endpoints.title')}</h2>
                    <div style="display: flex; gap: 10px;">
                        <button class="btn btn-secondary" onclick="window.showDataSyncDialog()">
                            🔄 ${t('webdav.dataSync')}
                        </button>
                        <button class="btn btn-primary" onclick="window.showAddEndpointModal()">
                            ➕ ${t('header.addEndpoint')}
                        </button>
                    </div>
                </div>
                <div id="endpointList" class="endpoint-list">
                    <div class="loading">${t('endpoints.title')}...</div>
                </div>
            </div>

            <!-- Logs Panel -->
            <div class="card">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                    <div style="display: flex; align-items: center; gap: 15px;">
                        <h2 style="margin: 0;">📋 ${t('logs.title')}</h2>
                        <select id="logLevel" class="log-level-select" onchange="window.changeLogLevel()">
                            <option value="0">🔍 ${t('logs.levels.0')}+</option>
                            <option value="1" selected>ℹ️ ${t('logs.levels.1')}+</option>
                            <option value="2">⚠️ ${t('logs.levels.2')}+</option>
                            <option value="3">❌ ${t('logs.levels.3')}</option>
                        </select>
                    </div>
                    <div style="display: flex; gap: 10px;">
                        <button class="btn btn-secondary btn-sm" onclick="window.copyLogs()">
                            📋 ${t('logs.copy')}
                        </button>
                        <button class="btn btn-secondary btn-sm" onclick="window.toggleLogPanel()">
                            <span id="logToggleIcon">▼</span> <span id="logToggleText">${t('logs.collapse')}</span>
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
                            <option value="gemini">Gemini</option>
                        </select>
                        <p style="color: #666; font-size: 12px; margin-top: 5px;">
                            ${t('modal.transformerHelp')}
                        </p>
                    </div>
                    <div class="form-group" id="modelFieldGroup" style="display: block;">
                        <label><span class="required" id="modelRequired" style="display: none; color: #ff4444;">* </span>${t('modal.model')}</label>
                        <input type="text" id="endpointModel" placeholder="${t('modal.modelPlaceholder')}">
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

                    <div style="text-align: center; margin: 30px 0;">
                        <img src="/WeChat.jpg" alt="WeChat QR Code" style="width: 200px; height: 200px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
                        <p style="margin-top: 10px; color: #666; font-size: 14px;">${t('welcome.qrCodeTip')}</p>
                    </div>

                    <div style="display: flex; gap: 15px; justify-content: center; margin-top: 20px;">
                        <button class="btn btn-primary" onclick="window.openArticle()">
                            ${t('welcome.readArticle')}
                        </button>
                        <button class="btn btn-secondary" onclick="window.openGitHub()">
                            ${t('welcome.githubRepo')}
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
                </div>
                <div class="modal-body">
                    <div id="testResultContent" style="font-size: 14px; line-height: 1.6;">
                        <!-- Test result will be inserted here -->
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-primary" onclick="window.closeTestResultModal()">${t('modal.close')}</button>
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
                        <h4 class="confirm-title">确认删除</h4>
                        <p id="confirmMessage" class="confirm-message"></p>
                    </div>
                </div>
                <div class="confirm-divider"></div>
                <div class="confirm-footer">
                    <button class="btn-confirm-delete" onclick="window.acceptConfirm()">删除</button>
                    <button class="btn-confirm-cancel" onclick="window.cancelConfirm()">取消</button>
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
