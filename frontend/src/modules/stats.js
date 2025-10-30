import { formatTokens } from '../utils/format.js';

let endpointStats = {};

export function getEndpointStats() {
    return endpointStats;
}

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
