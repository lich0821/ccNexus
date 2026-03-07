// Configuration management
export async function loadConfig() {
    try {
        if (!window.go?.main?.App) {
            console.error('Not running in Wails environment');
            document.getElementById('endpointList').innerHTML = `
                <div class="empty-state">
                    <p>⚠️ Please run this app through Wails</p>
                    <p>Use: wails dev or run the built application</p>
                </div>
            `;
            return null;
        }

        const configStr = await window.go.main.App.GetConfig();
        const config = JSON.parse(configStr);

        document.getElementById('proxyPort').textContent = config.port;
        document.getElementById('totalEndpoints').textContent = config.endpoints.length;

        const activeCount = config.endpoints.filter(ep => ep.enabled !== false).length;
        document.getElementById('activeEndpoints').textContent = activeCount;
        // Keep the visible stats bar endpoint count in sync with current config.
        const activeDisplayEl = document.getElementById('activeEndpointsDisplay');
        const totalDisplayEl = document.getElementById('totalEndpointsDisplay');
        if (activeDisplayEl) activeDisplayEl.textContent = activeCount;
        if (totalDisplayEl) totalDisplayEl.textContent = config.endpoints.length;

        return config;
    } catch (error) {
        console.error('Failed to load config:', error);
        return null;
    }
}

export async function updatePort(port) {
    await window.go.main.App.UpdatePort(port);
}

export async function addEndpoint(name, url, key, transformer, model, remark) {
    await window.go.main.App.AddEndpoint(name, url, key, transformer, model, remark || '');
}

export async function updateEndpoint(index, name, url, key, transformer, model, remark) {
    await window.go.main.App.UpdateEndpoint(index, name, url, key, transformer, model, remark || '');
}

export async function removeEndpoint(index) {
    await window.go.main.App.RemoveEndpoint(index);
}

export async function toggleEndpoint(index, enabled) {
    await window.go.main.App.ToggleEndpoint(index, enabled);
}

export async function testEndpoint(index) {
    const resultStr = await window.go.main.App.TestEndpoint(index);
    return JSON.parse(resultStr);
}

export async function testEndpointLight(index) {
    const resultStr = await window.go.main.App.TestEndpointLight(index);
    return JSON.parse(resultStr);
}

export async function testAllEndpointsZeroCost() {
    const resultStr = await window.go.main.App.TestAllEndpointsZeroCost();
    return JSON.parse(resultStr);
}
