// Format tokens in K or M
export function formatTokens(tokens) {
    if (tokens === 0) return '0';
    if (tokens >= 1000000) {
        return (tokens / 1000000).toFixed(1) + 'M';
    } else if (tokens >= 1000) {
        return (tokens / 1000).toFixed(1) + 'K';
    }
    return tokens.toString();
}

// Mask API key
export function maskApiKey(key) {
    if (key.length <= 4) return '***';
    return '****' + key.substring(key.length - 4);
}

// Escape HTML
export function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
