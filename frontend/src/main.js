import './style.css'

let currentEditIndex = -1;
let endpointStats = {};
let logPanelExpanded = true;

// Load data on startup
window.addEventListener('DOMContentLoaded', () => {
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
});

function initApp() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="header">
            <div style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
                <div>
                    <h1>🚀 ccNexus</h1>
                    <p>Smart API endpoint rotation proxy for Claude Code</p>
                </div>
                <div style="display: flex; gap: 15px; align-items: center;">
                    <div class="port-display" onclick="window.showEditPortModal()" title="Click to edit port">
                        <span style="color: #666; font-size: 14px;">Proxy Port: </span>
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
                    </div>
                </div>
            </div>
        </div>

        <div class="container">
            <!-- Statistics -->
            <div class="card">
                <h2>📊 Statistics</h2>
                <div class="stats-grid">
                    <div class="stat-box">
                        <div class="label">Endpoints</div>
                        <div class="value">
                            <span id="activeEndpoints">0</span>
                            <span style="font-size: 20px; opacity: 0.7;"> / </span>
                            <span id="totalEndpoints" style="font-size: 20px; opacity: 0.7;">0</span>
                        </div>
                        <div style="font-size: 12px; opacity: 0.8; margin-top: 5px;">Active / Total</div>
                    </div>
                    <div class="stat-box">
                        <div class="label">Total Requests</div>
                        <div class="value">
                            <span id="totalRequests">0</span>
                        </div>
                        <div style="font-size: 12px; opacity: 0.8; margin-top: 5px;">
                            <span id="successRequests">0</span> success /
                            <span id="failedRequests">0</span> failed
                        </div>
                    </div>
                    <div class="stat-box">
                        <div class="label">Total Tokens</div>
                        <div class="value">
                            <span id="totalTokens">0</span>
                        </div>
                        <div style="font-size: 12px; opacity: 0.8; margin-top: 5px;">
                            In: <span id="totalInputTokens">0</span> /
                            Out: <span id="totalOutputTokens">0</span>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Endpoints -->
            <div class="card">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                    <h2 style="margin: 0;">🔗 Endpoints</h2>
                    <button class="btn btn-primary" onclick="window.showAddEndpointModal()">
                        ➕ Add Endpoint
                    </button>
                </div>
                <div id="endpointList" class="endpoint-list">
                    <div class="loading">Loading endpoints...</div>
                </div>
            </div>

            <!-- Logs Panel -->
            <div class="card">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                    <div style="display: flex; align-items: center; gap: 15px;">
                        <h2 style="margin: 0;">📋 Logs</h2>
                        <select id="logLevel" class="log-level-select" onchange="window.changeLogLevel()">
                            <option value="0">🔍 DEBUG+</option>
                            <option value="1" selected>ℹ️ INFO+</option>
                            <option value="2">⚠️ WARN+</option>
                            <option value="3">❌ ERROR</option>
                        </select>
                    </div>
                    <div style="display: flex; gap: 10px;">
                        <button class="btn btn-secondary btn-sm" onclick="window.copyLogs()">
                            📋 Copy
                        </button>
                        <button class="btn btn-secondary btn-sm" onclick="window.toggleLogPanel()">
                            <span id="logToggleIcon">▼</span> <span id="logToggleText">Collapse</span>
                        </button>
                        <button class="btn btn-secondary btn-sm" onclick="window.clearLogs()">
                            🗑️ Clear
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
                    <h2 id="modalTitle">Add Endpoint</h2>
                </div>
                <div class="form-group">
                    <label>Name</label>
                    <input type="text" id="endpointName" placeholder="e.g., Claude Official">
                </div>
                <div class="form-group">
                    <label>API URL</label>
                    <input type="text" id="endpointUrl" placeholder="e.g., api.anthropic.com">
                </div>
                <div class="form-group">
                    <label>API Key</label>
                    <input type="password" id="endpointKey" placeholder="sk-ant-api03-...">
                </div>
                <div class="form-group">
                    <label>Transformer</label>
                    <select id="endpointTransformer" onchange="window.handleTransformerChange()">
                        <option value="claude">Claude (Default)</option>
                        <option value="openai">OpenAI</option>
                        <option value="gemini">Gemini</option>
                    </select>
                    <p style="color: #666; font-size: 12px; margin-top: 5px;">
                        Select the API format for this endpoint
                    </p>
                </div>
                <div class="form-group" id="modelFieldGroup" style="display: none;">
                    <label>Model</label>
                    <input type="text" id="endpointModel" placeholder="e.g., gpt-4-turbo">
                    <p style="color: #666; font-size: 12px; margin-top: 5px;">
                        Required for non-Claude transformers
                    </p>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary" onclick="window.closeModal()">Cancel</button>
                    <button class="btn btn-primary" onclick="window.saveEndpoint()">Save</button>
                </div>
            </div>
        </div>

        <!-- Edit Port Modal -->
        <div id="portModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2>Edit Proxy Port</h2>
                </div>
                <div class="form-group">
                    <label>Port (1-65535)</label>
                    <input type="number" id="portInput" min="1" max="65535" placeholder="3000">
                </div>
                <p style="color: #666; font-size: 14px; margin-top: 10px;">
                    ⚠️ Note: Changing the port requires restarting the application.
                </p>
                <div class="modal-footer">
                    <button class="btn btn-secondary" onclick="window.closePortModal()">Cancel</button>
                    <button class="btn btn-primary" onclick="window.savePort()">Save</button>
                </div>
            </div>
        </div>

        <!-- Welcome Modal -->
        <div id="welcomeModal" class="modal">
            <div class="modal-content" style="max-width: 600px;">
                <div class="modal-header">
                    <h2>👋 Welcome to ccNexus!</h2>
                </div>
                <div style="padding: 20px 0;">
                    <p style="font-size: 16px; line-height: 1.6; margin-bottom: 20px;">
                        ccNexus is a smart API endpoint rotation proxy for Claude Code.
                        It helps you manage multiple API endpoints with automatic failover and load balancing.
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
                            <span style="font-size: 14px; color: #666;">Don't show this again</span>
                        </label>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-primary" onclick="window.closeWelcomeModal()">Get Started</button>
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
                <p>No endpoints configured</p>
                <p>Click "Add Endpoint" to get started</p>
            </div>
        `;
        return;
    }

    // Clear container first
    container.innerHTML = '';

    // Create endpoint items
    endpoints.forEach((ep, index) => {
        const stats = endpointStats[ep.name] || { requests: 0, errors: 0, inputTokens: 0, outputTokens: 0 };
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
                <p style="color: #666; font-size: 14px; margin-top: 5px;">🔄 Transformer: ${transformer}${model ? ` (${model})` : ''}</p>
                <p style="color: #666; font-size: 14px; margin-top: 3px;">📊 Requests: ${stats.requests} | Errors: ${stats.errors}</p>
                <p style="color: #666; font-size: 14px; margin-top: 3px;">🎯 Tokens: ${formatTokens(totalTokens)} (In: ${formatTokens(stats.inputTokens)}, Out: ${formatTokens(stats.outputTokens)})</p>
            </div>
            <div style="display: flex; flex-direction: column; gap: 10px; align-items: flex-end;">
                <label class="toggle-switch">
                    <input type="checkbox" data-index="${index}" ${enabled ? 'checked' : ''}>
                    <span class="toggle-slider"></span>
                </label>
                <div style="display: flex; gap: 10px;">
                    <button class="btn btn-secondary" data-action="edit" data-index="${index}">Edit</button>
                    <button class="btn btn-danger" data-action="delete" data-index="${index}">Delete</button>
                </div>
            </div>
        `;

        // Add event listeners
        const editBtn = item.querySelector('[data-action="edit"]');
        const deleteBtn = item.querySelector('[data-action="delete"]');
        const toggleSwitch = item.querySelector('input[type="checkbox"]');

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
    document.getElementById('modalTitle').textContent = 'Add Endpoint';
    document.getElementById('endpointName').value = '';
    document.getElementById('endpointUrl').value = '';
    document.getElementById('endpointKey').value = '';
    document.getElementById('endpointTransformer').value = 'claude';
    document.getElementById('endpointModel').value = '';
    document.getElementById('modelFieldGroup').style.display = 'none';
    document.getElementById('endpointModal').classList.add('active');
}

window.handleTransformerChange = function() {
    const transformer = document.getElementById('endpointTransformer').value;
    const modelFieldGroup = document.getElementById('modelFieldGroup');

    if (transformer === 'claude') {
        modelFieldGroup.style.display = 'none';
    } else {
        modelFieldGroup.style.display = 'block';
    }
}

window.editEndpoint = async function(index) {
    currentEditIndex = index;
    const configStr = await window.go.main.App.GetConfig();
    const config = JSON.parse(configStr);
    const ep = config.endpoints[index];

    document.getElementById('modalTitle').textContent = 'Edit Endpoint';
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
window.showWelcomeModal = function() {
    document.getElementById('welcomeModal').classList.add('active');
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
        text.textContent = 'Collapse';
    } else {
        panel.style.display = 'none';
        icon.textContent = '▶';
        text.textContent = 'Expand';
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
