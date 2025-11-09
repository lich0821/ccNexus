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
                        <button class="header-link" onclick="window.openGitHub()" title="GitHub Repository">
                            <svg width="24" height="24" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
                            </svg>
                        </button>
                        <button class="header-link" onclick="window.showWelcomeModal()" title="About ccNexus">
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
                <h2>📊 ${t('statistics.title')}</h2>
                <div class="stats-grid">
                    <div class="stat-box">
                        <div class="label">${t('statistics.endpoints')}</div>
                        <div class="value">
                            <span id="activeEndpoints">0</span>
                            <span style="font-size: 20px; opacity: 0.7;"> / </span>
                            <span id="totalEndpoints" style="font-size: 20px; opacity: 0.7;">0</span>
                        </div>
                        <div style="font-size: 12px; opacity: 0.8; margin-top: 5px;">${t('statistics.activeTotal')}</div>
                    </div>
                    <div class="stat-box">
                        <div class="label">${t('statistics.totalRequests')}</div>
                        <div class="value">
                            <span id="totalRequests">0</span>
                        </div>
                        <div style="font-size: 12px; opacity: 0.8; margin-top: 5px;">
                            <span id="successRequests">0</span> ${t('statistics.success')} /
                            <span id="failedRequests">0</span> ${t('statistics.failed')}
                        </div>
                    </div>
                    <div class="stat-box">
                        <div class="label">${t('statistics.totalTokens')}</div>
                        <div class="value">
                            <span id="totalTokens">0</span>
                        </div>
                        <div style="font-size: 12px; opacity: 0.8; margin-top: 5px;">
                            ${t('statistics.in')}: <span id="totalInputTokens">0</span> /
                            ${t('statistics.out')}: <span id="totalOutputTokens">0</span>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Endpoints -->
            <div class="card">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                    <h2 style="margin: 0;">🔗 ${t('endpoints.title')}</h2>
                    <button class="btn btn-primary" onclick="window.showAddEndpointModal()">
                        ➕ ${t('header.addEndpoint')}
                    </button>
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
                    <h2 id="modalTitle">${t('modal.addEndpoint')}</h2>
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
                    <h2>${t('modal.changePort')}</h2>
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
                        <p style="margin-top: 10px; color: #666; font-size: 14px;">扫码关注公众号，了解更多</p>
                    </div>

                    <div style="display: flex; gap: 15px; justify-content: center; margin-top: 20px;">
                        <button class="btn btn-primary" onclick="window.openArticle()">
                            📖 阅读介绍
                        </button>
                        <button class="btn btn-secondary" onclick="window.openGitHub()">
                            🔗 GitHub Repository
                        </button>
                    </div>

                    <div style="margin-top: 25px; padding-top: 20px; border-top: 1px solid #eee;">
                        <label style="display: flex; align-items: center; justify-content: center; cursor: pointer;">
                            <input type="checkbox" id="dontShowAgain" style="margin-right: 8px;">
                            <span style="font-size: 14px; color: #666;">${t('welcome.dontShow')}</span>
                        </label>
                    </div>
                </div>
                <div class="modal-footer">
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
            <div class="modal-content" style="max-width: min(400px, 90vw);">
                <div class="modal-header">
                    <h2 id="confirmTitle">⚠️ 确认操作</h2>
                </div>
                <div class="modal-body">
                    <p id="confirmMessage" style="font-size: 15px; line-height: 1.6; color: #333;"></p>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary" onclick="window.cancelConfirm()">取消</button>
                    <button class="btn btn-danger" onclick="window.acceptConfirm()">确定</button>
                </div>
            </div>
        </div>
    `;

    setupModalEventListeners();
}

function setupModalEventListeners() {
    // Close modals on background click
    document.getElementById('endpointModal').addEventListener('click', (e) => {
        if (e.target.id === 'endpointModal') {
            window.closeModal();
        }
    });

    document.getElementById('portModal').addEventListener('click', (e) => {
        if (e.target.id === 'portModal') {
            window.closePortModal();
        }
    });

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
