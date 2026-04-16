import { api } from '../api.js';
import { state } from '../state.js';
import { notifications } from '../utils/notifications.js';
import { formatNumber, formatTokens } from '../utils/formatters.js';
import { t } from '../utils/i18n.js';

class Dashboard {
    constructor() {
        this.container = document.getElementById('view-container');
        // 监听语言切换
        window.addEventListener('languageChanged', () => {
            if (state.get('currentView') === 'dashboard') {
                this.render();
            }
        });
    }

    async render() {
        this.container.innerHTML = `
            <div class="dashboard">
                <h1>${t('dashboard.title')}</h1>
                <div id="stats-cards" class="grid grid-cols-4 mt-3">
                    <div class="stat-card">
                        <div class="stat-label">${t('dashboard.totalRequests')}</div>
                        <div class="stat-value" id="stat-requests">-</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">${t('dashboard.successRate')}</div>
                        <div class="stat-value" id="stat-success">-</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">${t('dashboard.inputTokens')}</div>
                        <div class="stat-value" id="stat-input-tokens">-</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">${t('dashboard.outputTokens')}</div>
                        <div class="stat-value" id="stat-output-tokens">-</div>
                    </div>
                </div>

                <div class="grid grid-cols-2 mt-4">
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">${t('dashboard.activeEndpoints')}</h3>
                        </div>
                        <div class="card-body">
                            <div id="endpoints-list"></div>
                        </div>
                    </div>

                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">${t('dashboard.recentActivity')}</h3>
                        </div>
                        <div class="card-body">
                            <canvas id="activity-chart"></canvas>
                        </div>
                    </div>
                </div>
            </div>
        `;

        await this.loadData();
    }

    async loadData() {
        try {
            // Load stats
            const stats = await api.getStatsSummary();
            this.updateStats(stats);

            // Load endpoints
            const endpointsData = await api.getEndpoints();
            this.updateEndpoints(endpointsData.endpoints);

            // Load daily stats for chart
            const dailyStats = await api.getStatsDaily();
            this.renderChart(dailyStats);
        } catch (error) {
            notifications.error('Failed to load dashboard data: ' + error.message);
        }
    }

    updateStats(stats) {
        const totalRequests = stats.TotalRequests || 0;
        const totalErrors = stats.TotalErrors || 0;
        const successRate = totalRequests > 0
            ? ((totalRequests - totalErrors) / totalRequests * 100).toFixed(1)
            : 0;

        document.getElementById('stat-requests').textContent = formatNumber(totalRequests);
        document.getElementById('stat-success').textContent = successRate + '%';
        document.getElementById('stat-input-tokens').textContent = formatTokens(stats.TotalInputTokens || 0);
        document.getElementById('stat-output-tokens').textContent = formatTokens(stats.TotalOutputTokens || 0);
    }

    updateEndpoints(endpoints) {
        const container = document.getElementById('endpoints-list');

        if (!endpoints || endpoints.length === 0) {
            container.innerHTML = `<div class="empty-state"><p>${t('dashboard.noEndpoints')}</p></div>`;
            return;
        }

        const enabledEndpoints = endpoints.filter(ep => ep.enabled);

        if (enabledEndpoints.length === 0) {
            container.innerHTML = `<div class="empty-state"><p>${t('dashboard.noEnabledEndpoints')}</p></div>`;
            return;
        }

        container.innerHTML = `
            <div class="table-container">
                <table class="table">
                    <thead>
                        <tr>
                            <th>${t('common.name')}</th>
                            <th>${t('endpoints.transformer')}</th>
                            <th>${t('common.status')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${enabledEndpoints.map(ep => `
                            <tr>
                                <td>${this.escapeHtml(ep.name)}</td>
                                <td>${this.escapeHtml(ep.transformer)}</td>
                                <td>
                                    <span class="status-indicator online"></span>
                                    <span class="badge badge-success">${t('common.active')}</span>
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        `;
    }

    renderChart(dailyStats) {
        const canvas = document.getElementById('activity-chart');
        const ctx = canvas.getContext('2d');

        // Simple bar chart showing requests
        const stats = dailyStats.stats || {};
        const endpoints = Object.keys(stats.endpoints || {});
        const requests = endpoints.map(ep => stats.endpoints[ep].requests || 0);

        new Chart(ctx, {
            type: 'bar',
            data: {
                labels: endpoints,
                datasets: [{
                    label: 'Requests',
                    data: requests,
                    backgroundColor: '#3b82f6',
                    borderColor: '#2563eb',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: true,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true
                    }
                }
            }
        });
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

export const dashboard = new Dashboard();
