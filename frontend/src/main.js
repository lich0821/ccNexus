import './style.css'

let currentEditIndex = -1;

// Load data on startup
window.addEventListener('DOMContentLoaded', () => {
    initApp();
    loadConfig();
    loadStats();

    // Refresh stats every 5 seconds
    setInterval(loadStats, 5000);
});

function initApp() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="header">
            <h1>🚀 ccNexus</h1>
            <p>Smart API endpoint rotation proxy for Claude Code</p>
        </div>

        <div class="container">
            <!-- Statistics -->
            <div class="card">
                <h2>📊 Statistics</h2>
                <div class="stats-grid">
                    <div class="stat-box">
                        <div class="label">Total Requests</div>
                        <div class="value" id="totalRequests">0</div>
                    </div>
                    <div class="stat-box">
                        <div class="label">Active Endpoints</div>
                        <div class="value" id="totalEndpoints">0</div>
                    </div>
                    <div class="stat-box" style="cursor: pointer;" onclick="window.showEditPortModal()">
                        <div class="label">Proxy Port (Click to Edit)</div>
                        <div class="value" id="proxyPort">3000</div>
                    </div>
                </div>
            </div>

            <!-- Endpoints -->
            <div class="card">
                <h2>🔗 Endpoints</h2>
                <div id="endpointList" class="endpoint-list">
                    <div class="loading">Loading endpoints...</div>
                </div>
                <button class="btn btn-primary" onclick="window.showAddEndpointModal()" style="margin-top: 15px;">
                    ➕ Add Endpoint
                </button>
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

        renderEndpoints(config.endpoints);
    } catch (error) {
        console.error('Failed to load config:', error);
    }
}

async function loadStats() {
    try {
        const statsStr = await window.go.main.App.GetStats();
        const stats = JSON.parse(statsStr);

        document.getElementById('totalRequests').textContent = stats.totalRequests;
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
        const item = document.createElement('div');
        item.className = 'endpoint-item';
        item.innerHTML = `
            <div class="endpoint-info">
                <h3>${ep.name}</h3>
                <p>🌐 ${ep.apiUrl}</p>
                <p>🔑 ${maskApiKey(ep.apiKey)}</p>
            </div>
            <div style="display: flex; gap: 10px;">
                <button class="btn btn-secondary" data-action="edit" data-index="${index}">Edit</button>
                <button class="btn btn-danger" data-action="delete" data-index="${index}">Delete</button>
            </div>
        `;

        // Add event listeners
        const editBtn = item.querySelector('[data-action="edit"]');
        const deleteBtn = item.querySelector('[data-action="delete"]');

        editBtn.addEventListener('click', () => {
            const idx = parseInt(editBtn.getAttribute('data-index'));
            window.editEndpoint(idx);
        });
        deleteBtn.addEventListener('click', () => {
            const idx = parseInt(deleteBtn.getAttribute('data-index'));
            window.deleteEndpoint(idx);
        });

        container.appendChild(item);
    });
}

function maskApiKey(key) {
    if (key.length <= 8) return '***';
    return key.substring(0, 8) + '***';
}

window.showAddEndpointModal = function() {
    currentEditIndex = -1;
    document.getElementById('modalTitle').textContent = 'Add Endpoint';
    document.getElementById('endpointName').value = '';
    document.getElementById('endpointUrl').value = '';
    document.getElementById('endpointKey').value = '';
    document.getElementById('endpointModal').classList.add('active');
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
    document.getElementById('endpointModal').classList.add('active');
}

window.saveEndpoint = async function() {
    const name = document.getElementById('endpointName').value.trim();
    const url = document.getElementById('endpointUrl').value.trim();
    const key = document.getElementById('endpointKey').value.trim();

    if (!name || !url || !key) {
        alert('Please fill in all fields');
        return;
    }

    try {
        if (currentEditIndex === -1) {
            await window.go.main.App.AddEndpoint(name, url, key);
        } else {
            await window.go.main.App.UpdateEndpoint(currentEditIndex, name, url, key);
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
