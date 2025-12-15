// Utility functions for formatting data

export function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    }
    if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

export function formatTokens(tokens) {
    return formatNumber(tokens);
}

export function formatPercentage(value) {
    const sign = value >= 0 ? '+' : '';
    return `${sign}${value.toFixed(1)}%`;
}

export function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString();
}

export function formatDateTime(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString();
}

export function formatLatency(ms) {
    if (ms < 1000) {
        return `${ms}ms`;
    }
    return `${(ms / 1000).toFixed(2)}s`;
}

export function getTransformerLabel(transformer) {
    const labels = {
        'claude': 'Claude',
        'openai': 'OpenAI',
        'openai2': 'OpenAI Responses',
        'gemini': 'Gemini',
        'deepseek': 'DeepSeek'
    };
    return labels[transformer] || transformer;
}

export function getStatusBadge(enabled) {
    if (enabled) {
        return '<span class="badge badge-success">Enabled</span>';
    }
    return '<span class="badge badge-danger">Disabled</span>';
}

export function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
