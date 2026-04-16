import { api } from '../api.js';
import { notifications } from '../utils/notifications.js';
import { formatNumber, formatTokens } from '../utils/formatters.js';
import { t } from '../utils/i18n.js';

class Stats {
    constructor() {
        this.container = document.getElementById('view-container');
        // 监听语言切换
        window.addEventListener('languageChanged', () => {
            if (state.get('currentView') === 'stats') {
                this.render();
            }
        });
    }

    async render() {
        this.container.innerHTML = `
            <div class="stats">
                <h1>${t('stats.title')}</h1>

                <div class="flex gap-2 mt-3 mb-3">
                    <button class="btn btn-sm btn-primary period-btn active" data-period="daily">${t('stats.daily')}</button>
                    <button class="btn btn-sm btn-secondary period-btn" data-period="weekly">${t('stats.weekly')}</button>
                    <button class="btn btn-sm btn-secondary period-btn" data-period="monthly">${t('stats.monthly')}</button>
                </div>

                <div id="stats-content"></div>
            </div>
        `;

        document.querySelectorAll('.period-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.period-btn').forEach(b => {
                    b.classList.remove('btn-primary', 'active');
                    b.classList.add('btn-secondary');
                });
                btn.classList.remove('btn-secondary');
                btn.classList.add('btn-primary', 'active');
                this.loadStats(btn.dataset.period);
            });
        });

        await this.loadStats('daily');
    }

    async loadStats(period) {
        try {
            let data;
            switch (period) {
                case 'daily':
                    data = await api.getStatsDaily();
                    break;
                case 'weekly':
                    data = await api.getStatsWeekly();
                    break;
                case 'monthly':
                    data = await api.getStatsMonthly();
                    break;
            }

            this.renderStats(data);
        } catch (error) {
            notifications.error(`${t('stats.failedToLoad')}: ${error.message}`);
        }
    }

    renderStats(data) {
        const stats = data.stats || {};
        const container = document.getElementById('stats-content');

        container.innerHTML = `
            <div class="grid grid-cols-4 mb-4">
                <div class="stat-card">
                    <div class="stat-label">${t('stats.totalRequests')}</div>
                    <div class="stat-value">${formatNumber(stats.totalRequests || 0)}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">${t('stats.successful')}</div>
                    <div class="stat-value">${formatNumber(stats.totalSuccess || 0)}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">${t('stats.errors')}</div>
                    <div class="stat-value">${formatNumber(stats.totalErrors || 0)}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">${t('stats.totalTokens')}</div>
                    <div class="stat-value">${formatTokens((stats.totalInputTokens || 0) + (stats.totalOutputTokens || 0))}</div>
                </div>
            </div>

            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">${t('stats.endpointBreakdown')}</h3>
                </div>
                <div class="card-body">
                    ${this.renderEndpointTable(stats.endpoints || {})}
                </div>
            </div>
        `;
    }

    renderEndpointTable(endpoints) {
        const endpointNames = Object.keys(endpoints);

        if (endpointNames.length === 0) {
            return `<div class="empty-state"><p>${t('stats.noDataAvailable')}</p></div>`;
        }

        return `
            <div class="table-container">
                <table class="table">
                    <thead>
                        <tr>
                            <th>${t('stats.endpoint')}</th>
                            <th>${t('stats.requests')}</th>
                            <th>${t('stats.errors')}</th>
                            <th>${t('stats.inputTokens')}</th>
                            <th>${t('stats.outputTokens')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${endpointNames.map(name => {
                            const ep = endpoints[name];
                            return `
                                <tr>
                                    <td><strong>${this.escapeHtml(name)}</strong></td>
                                    <td>${formatNumber(ep.requests || 0)}</td>
                                    <td>${formatNumber(ep.errors || 0)}</td>
                                    <td>${formatTokens(ep.inputTokens || 0)}</td>
                                    <td>${formatTokens(ep.outputTokens || 0)}</td>
                                </tr>
                            `;
                        }).join('')}
                    </tbody>
                </table>
            </div>
        `;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

export const stats = new Stats();
