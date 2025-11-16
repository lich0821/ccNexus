import { formatTokens } from '../utils/format.js';

let endpointStats = {};
let currentPeriod = 'daily'; // 'daily', 'weekly', 'monthly'

export function getEndpointStats() {
    return endpointStats;
}

export function getCurrentPeriod() {
    return currentPeriod;
}

// Load statistics (legacy function for backward compatibility)
export async function loadStats() {
    try {
        const statsStr = await window.go.main.App.GetStats();
        const stats = JSON.parse(statsStr);

        document.getElementById('totalRequests').textContent = stats.totalRequests;

        let totalSuccess = 0;
        let totalFailed = 0;
        let totalInputTokens = 0;
        let totalOutputTokens = 0;

        for (const epStats of Object.values(stats.endpoints || {})) {
            totalSuccess += (epStats.requests - epStats.errors);
            totalFailed += epStats.errors;
            totalInputTokens += epStats.inputTokens || 0;
            totalOutputTokens += epStats.outputTokens || 0;
        }

        document.getElementById('successRequests').textContent = totalSuccess;
        document.getElementById('failedRequests').textContent = totalFailed;

        const totalTokens = totalInputTokens + totalOutputTokens;
        document.getElementById('totalTokens').textContent = formatTokens(totalTokens);
        document.getElementById('totalInputTokens').textContent = formatTokens(totalInputTokens);
        document.getElementById('totalOutputTokens').textContent = formatTokens(totalOutputTokens);

        endpointStats = stats.endpoints || {};

        return stats;
    } catch (error) {
        console.error('Failed to load stats:', error);
        return null;
    }
}

// Load statistics by period (daily, yesterday, weekly, monthly)
export async function loadStatsByPeriod(period = 'daily') {
    try {
        currentPeriod = period;

        let statsStr;
        switch (period) {
            case 'daily':
                statsStr = await window.go.main.App.GetStatsDaily();
                break;
            case 'yesterday':
                statsStr = await window.go.main.App.GetStatsYesterday();
                break;
            case 'weekly':
                statsStr = await window.go.main.App.GetStatsWeekly();
                break;
            case 'monthly':
                statsStr = await window.go.main.App.GetStatsMonthly();
                break;
            default:
                statsStr = await window.go.main.App.GetStatsDaily();
        }

        const stats = JSON.parse(statsStr);

        // Update UI elements
        document.getElementById('periodTotalRequests').textContent = stats.totalRequests || 0;
        document.getElementById('periodSuccess').textContent = stats.totalSuccess || 0;
        document.getElementById('periodFailed').textContent = stats.totalErrors || 0;

        const totalTokens = (stats.totalInputTokens || 0) + (stats.totalOutputTokens || 0);
        document.getElementById('periodTotalTokens').textContent = formatTokens(totalTokens);
        document.getElementById('periodInputTokens').textContent = formatTokens(stats.totalInputTokens || 0);
        document.getElementById('periodOutputTokens').textContent = formatTokens(stats.totalOutputTokens || 0);

        // Update endpoint stats (active / total)
        const activeEndpoints = stats.activeEndpoints || 0;
        const totalEndpoints = stats.totalEndpoints || 0;
        document.getElementById('activeEndpointsDisplay').textContent = activeEndpoints;
        document.getElementById('totalEndpointsDisplay').textContent = totalEndpoints;

        // Load and display trend
        await loadTrend();

        // Store endpoint stats for today
        endpointStats = stats.endpoints || {};

        return stats;
    } catch (error) {
        console.error('Failed to load stats by period:', error);
        return null;
    }
}

// Load trend comparison data
async function loadTrend() {
    try {
        const trendStr = await window.go.main.App.GetStatsTrend();
        const trend = JSON.parse(trendStr);

        // Display daily trend
        if (trend.daily) {
            const requestsTrend = formatTrend(trend.daily.trend);
            const errorsTrend = formatTrend(trend.daily.errorsTrend);
            const tokensTrend = formatTrend(trend.daily.tokensTrend);

            const requestsEl = document.getElementById('requestsTrend');
            const errorsEl = document.getElementById('errorsTrend');
            const tokensEl = document.getElementById('tokensTrend');

            if (requestsEl) {
                requestsEl.textContent = requestsTrend.text;
                requestsEl.className = 'trend ' + requestsTrend.className;
            }

            if (errorsEl) {
                // For errors, negative trend is good
                errorsEl.textContent = errorsTrend.text;
                errorsEl.className = 'trend ' + (trend.daily.errorsTrend < 0 ? 'trend-down' : trend.daily.errorsTrend > 0 ? 'trend-up' : 'trend-flat');
            }

            if (tokensEl) {
                tokensEl.textContent = tokensTrend.text;
                tokensEl.className = 'trend ' + tokensTrend.className;
            }
        }
    } catch (error) {
        console.error('Failed to load trend:', error);
    }
}

// Format trend value for display
function formatTrend(value) {
    const absValue = Math.abs(value);
    const formattedValue = absValue.toFixed(1);

    if (value > 0) {
        return {
            text: `↑ +${formattedValue}%`,
            className: 'trend-up'
        };
    } else if (value < 0) {
        return {
            text: `↓ ${formattedValue}%`,
            className: 'trend-down'
        };
    } else {
        return {
            text: '→ 0%',
            className: 'trend-flat'
        };
    }
}

// Switch statistics period
export function switchStatsPeriod(period) {
    // Handle history modal separately
    if (period === 'history') {
        // Open history modal without changing active tab
        import('./history.js').then(module => {
            module.showHistoryModal();
        });
        return;
    }

    currentPeriod = period;

    // Update tab buttons
    const tabs = document.querySelectorAll('.stats-tab-btn');
    tabs.forEach(tab => {
        if (tab.dataset.period === period) {
            tab.classList.add('active');
        } else {
            tab.classList.remove('active');
        }
    });

    // Load stats for the selected period
    loadStatsByPeriod(period);
}
