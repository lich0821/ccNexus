import './style.css'
import { setLanguage, t } from './i18n/index.js'

let currentEditIndex = -1;
let endpointStats = {};
let logPanelExpanded = true;
let currentTestButton = null;
let currentTestButtonOriginalText = '';
let currentTestIndex = -1;

// Load data on startup
window.addEventListener('DOMContentLoaded', async () => {
    // Initialize language
    const lang = await window.go.main.App.GetLanguage();
    setLanguage(lang);

    initApp();
    loadConfig();
    loadStats();
    loadLogs();

    // Refresh stats every 5 seconds
    setInterval(loadStats, 5000);

    // Refresh logs every 2 seconds
    setInterval(loadLogs, 2000);

    // Show welcome modal on first launch
    showWelcomeModalIfFirstTime();

    // Handle Cmd/Ctrl+W to hide window
    window.addEventListener('keydown', (e) => {
        if ((e.metaKey || e.ctrlKey) && e.key === 'w') {
            e.preventDefault();
            window.runtime.WindowHide();
        }
    });
});

function initApp() {
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

        <!-- Add/Edit Endpoint Modal -->
        <div id="endpointModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2 id="modalTitle">${t('modal.addEndpoint')}</h2>
                </div>
                <div class="form-group">
                    <label>${t('modal.name')} <span class="required" style="color: #ff4444;">*</span></label>
                    <input type="text" id="endpointName" placeholder="e.g., Claude Official" required>
                </div>
                <div class="form-group">
                    <label>${t('modal.apiUrl')} <span class="required" style="color: #ff4444;">*</span></label>
                    <input type="text" id="endpointUrl" placeholder="e.g., api.anthropic.com" required>
                </div>
                <div class="form-group">
                    <label>${t('modal.apiKey')} <span class="required" style="color: #ff4444;">*</span></label>
                    <input type="password" id="endpointKey" placeholder="sk-ant-api03-..." required>
                </div>
                <div class="form-group">
                    <label>${t('modal.transformer')} <span class="required" style="color: #ff4444;">*</span></label>
                    <select id="endpointTransformer" onchange="window.handleTransformerChange()" required>
                        <option value="claude">Claude (Default)</option>
                        <option value="openai">OpenAI</option>
                        <option value="gemini">Gemini</option>
                    </select>
                    <p style="color: #666; font-size: 12px; margin-top: 5px;">
                        ${t('modal.transformerHelp')}
                    </p>
                </div>
                <div class="form-group" id="modelFieldGroup" style="display: block;">
                    <label>${t('modal.model')} <span class="required" id="modelRequired" style="display: none; color: #ff4444;">*</span></label>
                    <input type="text" id="endpointModel" placeholder="e.g., claude-3-5-sonnet-20241022">
                    <p style="color: #666; font-size: 12px; margin-top: 5px;" id="modelHelpText">
                        ${t('modal.modelHelp')}
                    </p>
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
                <div class="form-group">
                    <label>${t('modal.port')} (1-65535)</label>
                    <input type="number" id="portInput" min="1" max="65535" placeholder="3000">
                </div>
                <p style="color: #666; font-size: 14px; margin-top: 10px;">
                    ⚠️ ${t('modal.portNote')}
                </p>
                <div class="modal-footer">
                    <button class="btn btn-secondary" onclick="window.closePortModal()">${t('modal.cancel')}</button>
                    <button class="btn btn-primary" onclick="window.savePort()">${t('modal.save')}</button>
                </div>
            </div>
        </div>

        <!-- Welcome Modal -->
        <div id="welcomeModal" class="modal">
            <div class="modal-content" style="max-width: 600px;">
                <div class="modal-header">
                    <h2>👋 ${t('welcome.title')}</h2>
                </div>
                <div style="padding: 20px 0;">
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
            <div class="modal-content" style="max-width: 600px;">
                <div class="modal-header">
                    <h2 id="testResultTitle">🧪 ${t('test.title')}</h2>
                </div>
                <div style="padding: 20px 0;">
                    <div id="testResultContent" style="font-size: 14px; line-height: 1.6;">
                        <!-- Test result will be inserted here -->
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-primary" onclick="window.closeTestResultModal()">${t('modal.close')}</button>
                </div>
            </div>
        </div>
    `;

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

async function loadConfig() {
    try {
        // Check if running in Wails
        if (!window.go || !window.go.main || !window.go.main.App) {
            console.error('Not running in Wails environment');
            document.getElementById('endpointList').innerHTML = `
                <div class="empty-state">
                    <p>⚠️ Please run this app through Wails</p>
                    <p>Use: wails dev or run the built application</p>
                </div>
            `;
            return;
        }

        const configStr = await window.go.main.App.GetConfig();
        const config = JSON.parse(configStr);

        document.getElementById('proxyPort').textContent = config.port;
        document.getElementById('totalEndpoints').textContent = config.endpoints.length;

        // Count active endpoints
        const activeCount = config.endpoints.filter(ep => ep.enabled !== false).length;
        document.getElementById('activeEndpoints').textContent = activeCount;

        renderEndpoints(config.endpoints);
    } catch (error) {
        console.error('Failed to load config:', error);
    }
}

// Format tokens in K or M
function formatTokens(tokens) {
    if (tokens === 0) return '0';
    if (tokens >= 1000000) {
        const m = tokens / 1000000;
        return m.toFixed(1) + 'M';
    } else if (tokens >= 1000) {
        const k = tokens / 1000;
        return k.toFixed(1) + 'K';
    } else {
        return tokens.toString();
    }
}

async function loadStats() {
    try {
        const statsStr = await window.go.main.App.GetStats();
        const stats = JSON.parse(statsStr);

        // Total requests
        document.getElementById('totalRequests').textContent = stats.totalRequests;

        // Calculate success and failed requests
        let totalSuccess = 0;
        let totalFailed = 0;
        let totalInputTokens = 0;
        let totalOutputTokens = 0;

        for (const [name, epStats] of Object.entries(stats.endpoints || {})) {
            totalSuccess += (epStats.requests - epStats.errors);
            totalFailed += epStats.errors;
            totalInputTokens += epStats.inputTokens || 0;
            totalOutputTokens += epStats.outputTokens || 0;
        }

        document.getElementById('successRequests').textContent = totalSuccess;
        document.getElementById('failedRequests').textContent = totalFailed;

        // Total tokens
        const totalTokens = totalInputTokens + totalOutputTokens;
        document.getElementById('totalTokens').textContent = formatTokens(totalTokens);
        document.getElementById('totalInputTokens').textContent = formatTokens(totalInputTokens);
        document.getElementById('totalOutputTokens').textContent = formatTokens(totalOutputTokens);

        // Save endpoint stats globally
        endpointStats = stats.endpoints || {};

        // Trigger re-render if config is already loaded
        const configStr = await window.go.main.App.GetConfig();
        const config = JSON.parse(configStr);
        renderEndpoints(config.endpoints);
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

function renderEndpoints(endpoints) {
    const container = document.getElementById('endpointList');

    if (endpoints.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <p>${t('endpoints.noEndpoints')}</p>
            </div>
        `;
        return;
    }

    // Clear container first
    container.innerHTML = '';

    // Sort endpoints by enabled status, success rate and request count
    const sortedEndpoints = endpoints.map((ep, index) => {
        const stats = endpointStats[ep.name] || { requests: 0, errors: 0, inputTokens: 0, outputTokens: 0 };
        const enabled = ep.enabled !== undefined ? ep.enabled : true;
        return { endpoint: ep, originalIndex: index, stats: stats, enabled: enabled };
    }).sort((a, b) => {
        // Primary sort: by enabled status (enabled first)
        if (a.enabled !== b.enabled) {
            return a.enabled ? -1 : 1;
        }

        // Within same enabled group, sort by performance
        const statsA = a.stats;
        const statsB = b.stats;

        // Special handling: requests = 0 goes to the end of each group
        if (statsA.requests === 0 && statsB.requests === 0) {
            return 0; // Keep original order
        }
        if (statsA.requests === 0) {
            return 1; // A goes after B
        }
        if (statsB.requests === 0) {
            return -1; // B goes after A
        }

        // Calculate success rate
        const successRateA = (statsA.requests - statsA.errors) / statsA.requests;
        const successRateB = (statsB.requests - statsB.errors) / statsB.requests;

        // Secondary sort: by success rate (descending)
        if (successRateA !== successRateB) {
            return successRateB - successRateA;
        }

        // Tertiary sort: by request count (descending)
        return statsB.requests - statsA.requests;
    });

    // Create endpoint items
    sortedEndpoints.forEach(({ endpoint: ep, originalIndex: index, stats }) => {
        const totalTokens = stats.inputTokens + stats.outputTokens;

        // Format tokens in K (thousands) or M (millions)
        const formatTokens = (tokens) => {
            if (tokens === 0) return '0';
            if (tokens >= 1000000) {
                // >= 1M, show in M
                const m = tokens / 1000000;
                return m.toFixed(1) + 'M';
            } else if (tokens >= 1000) {
                // >= 1K, show in K
                const k = tokens / 1000;
                return k.toFixed(1) + 'K';
            } else {
                // < 1K, show original value
                return tokens.toString();
            }
        };

        const enabled = ep.enabled !== undefined ? ep.enabled : true;
        const transformer = ep.transformer || 'claude';
        const model = ep.model || '';

        const item = document.createElement('div');
        item.className = 'endpoint-item';
        item.innerHTML = `
            <div class="endpoint-info">
                <h3>${ep.name} ${enabled ? '✅' : '❌'}</h3>
                <p>🌐 ${ep.apiUrl}</p>
                <p>🔑 ${maskApiKey(ep.apiKey)}</p>
                <p style="color: #666; font-size: 14px; margin-top: 5px;">🔄 ${t('endpoints.transformer')}: ${transformer}${model ? ` (${model})` : ''}</p>
                <p style="color: #666; font-size: 14px; margin-top: 3px;">📊 ${t('endpoints.requests')}: ${stats.requests} | ${t('endpoints.errors')}: ${stats.errors}</p>
                <p style="color: #666; font-size: 14px; margin-top: 3px;">🎯 ${t('endpoints.tokens')}: ${formatTokens(totalTokens)} (${t('statistics.in')}: ${formatTokens(stats.inputTokens)}, ${t('statistics.out')}: ${formatTokens(stats.outputTokens)})</p>
            </div>
            <div class="endpoint-actions">
                <label class="toggle-switch">
                    <input type="checkbox" data-index="${index}" ${enabled ? 'checked' : ''}>
                    <span class="toggle-slider"></span>
                </label>
                <button class="btn btn-secondary" data-action="test" data-index="${index}">${t('endpoints.test')}</button>
                <button class="btn btn-secondary" data-action="edit" data-index="${index}">${t('endpoints.edit')}</button>
                <button class="btn btn-danger" data-action="delete" data-index="${index}">${t('endpoints.delete')}</button>
            </div>
        `;

        // Add event listeners
        const testBtn = item.querySelector('[data-action="test"]');
        const editBtn = item.querySelector('[data-action="edit"]');
        const deleteBtn = item.querySelector('[data-action="delete"]');
        const toggleSwitch = item.querySelector('input[type="checkbox"]');

        // Restore test button state if this endpoint is being tested
        if (currentTestIndex === index) {
            testBtn.disabled = true;
            testBtn.innerHTML = '⏳';
            currentTestButton = testBtn; // Update reference to new button
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
                await window.go.main.App.ToggleEndpoint(idx, newEnabled);
                loadConfig();
            } catch (error) {
                console.error('Failed to toggle endpoint:', error);
                alert('Failed to toggle endpoint: ' + error);
                // Revert checkbox state on error
                e.target.checked = !newEnabled;
            }
        });

        container.appendChild(item);
    });
}

function maskApiKey(key) {
    if (key.length <= 4) return '***';
    return '****' + key.substring(key.length - 4);
}

window.showAddEndpointModal = function() {
    currentEditIndex = -1;
    document.getElementById('modalTitle').textContent = t('modal.addEndpoint');
    document.getElementById('endpointName').value = '';
    document.getElementById('endpointUrl').value = '';
    document.getElementById('endpointKey').value = '';
    document.getElementById('endpointTransformer').value = 'claude';
    document.getElementById('endpointModel').value = '';
    window.handleTransformerChange(); // Update model field visibility and hints
    document.getElementById('endpointModal').classList.add('active');
}

window.handleTransformerChange = function() {
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

window.editEndpoint = async function(index) {
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

    // Show/hide model field based on transformer
    window.handleTransformerChange();

    document.getElementById('endpointModal').classList.add('active');
}

window.saveEndpoint = async function() {
    const name = document.getElementById('endpointName').value.trim();
    const url = document.getElementById('endpointUrl').value.trim();
    const key = document.getElementById('endpointKey').value.trim();
    const transformer = document.getElementById('endpointTransformer').value;
    const model = document.getElementById('endpointModel').value.trim();

    if (!name || !url || !key) {
        alert('Please fill in all required fields');
        return;
    }

    // Validate model field for non-Claude transformers
    if (transformer !== 'claude' && !model) {
        alert('Model field is required for ' + transformer + ' transformer');
        return;
    }

    try {
        if (currentEditIndex === -1) {
            await window.go.main.App.AddEndpoint(name, url, key, transformer, model);
        } else {
            await window.go.main.App.UpdateEndpoint(currentEditIndex, name, url, key, transformer, model);
        }

        window.closeModal();
        loadConfig();
    } catch (error) {
        alert('Failed to save endpoint: ' + error);
    }
}

window.deleteEndpoint = async function(index) {
    console.log('Delete button clicked for index:', index);

    try {
        console.log('Calling RemoveEndpoint...');
        await window.go.main.App.RemoveEndpoint(index);
        console.log('RemoveEndpoint succeeded');
        loadConfig();
    } catch (error) {
        console.error('Delete failed:', error);
        alert('Failed to delete endpoint: ' + error);
    }
}

window.closeModal = function() {
    document.getElementById('endpointModal').classList.remove('active');
}

// Port editing functions
window.showEditPortModal = async function() {
    const configStr = await window.go.main.App.GetConfig();
    const config = JSON.parse(configStr);

    document.getElementById('portInput').value = config.port;
    document.getElementById('portModal').classList.add('active');
}

window.savePort = async function() {
    const port = parseInt(document.getElementById('portInput').value);

    if (!port || port < 1 || port > 65535) {
        alert('Please enter a valid port number (1-65535)');
        return;
    }

    try {
        await window.go.main.App.UpdatePort(port);
        window.closePortModal();
        loadConfig();
        alert('Port updated successfully! Please restart the application for changes to take effect.');
    } catch (error) {
        alert('Failed to update port: ' + error);
    }
}

window.closePortModal = function() {
    document.getElementById('portModal').classList.remove('active');
}

// Welcome modal functions
window.showWelcomeModal = async function() {
    document.getElementById('welcomeModal').classList.add('active');

    // Load and display version number
    try {
        const version = await window.go.main.App.GetVersion();
        document.querySelector('#welcomeModal .modal-header h2').textContent = `👋 Welcome to ccNexus v${version}`;
    } catch (error) {
        console.error('Failed to load version:', error);
    }
}

window.closeWelcomeModal = function() {
    const dontShowAgain = document.getElementById('dontShowAgain').checked;
    if (dontShowAgain) {
        localStorage.setItem('ccNexus_welcomeShown', 'true');
    }
    document.getElementById('welcomeModal').classList.remove('active');
}

function showWelcomeModalIfFirstTime() {
    const hasShown = localStorage.getItem('ccNexus_welcomeShown');
    if (!hasShown) {
        // Show modal after a short delay for better UX
        setTimeout(() => {
            window.showWelcomeModal();
        }, 500);
    }
}

// Open external URLs using Wails runtime
window.openGitHub = function() {
    if (window.go && window.go.main && window.go.main.App) {
        window.go.main.App.OpenURL('https://github.com/lich0821/ccNexus');
    }
}

window.openArticle = function() {
    if (window.go && window.go.main && window.go.main.App) {
        window.go.main.App.OpenURL('https://mp.weixin.qq.com/s/MqUVgWbkcVUNPnZQC--CZQ');
    }
}

// Log panel functions
async function loadLogs() {
    try {
        if (!window.go || !window.go.main || !window.go.main.App) {
            return;
        }

        const level = parseInt(document.getElementById('logLevel').value);
        const logsStr = await window.go.main.App.GetLogsByLevel(level);
        const logs = JSON.parse(logsStr);

        renderLogs(logs);
    } catch (error) {
        console.error('Failed to load logs:', error);
    }
}

function renderLogs(logs) {
    const textarea = document.getElementById('logContent');

    if (logs.length === 0) {
        textarea.value = '';
        return;
    }

    // Show all logs (no limit)
    const recentLogs = logs;

    // Format logs as plain text with date and 24-hour time
    const logText = recentLogs.map(log => {
        const date = new Date(log.timestamp);
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        const seconds = String(date.getSeconds()).padStart(2, '0');
        const timeStr = `${year}${month}${day} ${hours}:${minutes}:${seconds}`;

        return `${timeStr} ${log.icon} ${log.levelStr.padEnd(5)} ${log.message}`;
    }).join('\n');

    textarea.value = logText;

    // Auto-scroll to bottom
    textarea.scrollTop = textarea.scrollHeight;
}

window.toggleLogPanel = function() {
    const panel = document.getElementById('logPanel');
    const icon = document.getElementById('logToggleIcon');
    const text = document.getElementById('logToggleText');

    logPanelExpanded = !logPanelExpanded;

    if (logPanelExpanded) {
        panel.style.display = 'block';
        icon.textContent = '▼';
        text.textContent = t('logs.collapse');
    } else {
        panel.style.display = 'none';
        icon.textContent = '▶';
        text.textContent = t('logs.expand');
    }
}

window.changeLogLevel = async function() {
    const level = parseInt(document.getElementById('logLevel').value);
    try {
        // Set both display and record level
        await window.go.main.App.SetLogLevel(level);
        // Reload logs with new filter
        loadLogs();
    } catch (error) {
        console.error('Failed to change log level:', error);
        alert('Failed to change log level: ' + error);
    }
}

window.copyLogs = function() {
    const textarea = document.getElementById('logContent');
    textarea.select();
    document.execCommand('copy');

    // Visual feedback
    const btn = event.target.closest('button');
    const originalText = btn.innerHTML;
    btn.innerHTML = '✅ Copied!';
    setTimeout(() => {
        btn.innerHTML = originalText;
    }, 1500);
}

window.clearLogs = async function() {
    try {
        await window.go.main.App.ClearLogs();
        loadLogs();
    } catch (error) {
        console.error('Failed to clear logs:', error);
        alert('Failed to clear logs: ' + error);
    }
}

// Test endpoint function
window.testEndpoint = async function(index, buttonElement) {
    // Save button reference, original text, and index
    currentTestButton = buttonElement;
    currentTestButtonOriginalText = buttonElement.innerHTML;
    currentTestIndex = index;

    try {
        // Change button to loading state
        buttonElement.disabled = true;
        buttonElement.innerHTML = '⏳';

        // Call backend to test endpoint
        const resultStr = await window.go.main.App.TestEndpoint(index);
        const result = JSON.parse(resultStr);

        // Display result in modal
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

        // Show modal
        document.getElementById('testResultModal').classList.add('active');

    } catch (error) {
        console.error('Test failed:', error);

        // Display error in modal
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
    // Note: Button state will be restored when modal is closed
}

// Helper function to escape HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Close test result modal
window.closeTestResultModal = function() {
    document.getElementById('testResultModal').classList.remove('active');

    // Restore test button state
    if (currentTestButton) {
        currentTestButton.disabled = false;
        currentTestButton.innerHTML = currentTestButtonOriginalText;
        currentTestButton = null;
        currentTestButtonOriginalText = '';
        currentTestIndex = -1;
    }
}

// Change language
window.changeLanguage = async function(lang) {
    try {
        await window.go.main.App.SetLanguage(lang);
        setLanguage(lang);
        // Reload the page to apply new language
        location.reload();
    } catch (error) {
        console.error('Failed to change language:', error);
    }
}

