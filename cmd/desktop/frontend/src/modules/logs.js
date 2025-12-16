import { t } from '../i18n/index.js';

let logPanelExpanded = true;

export async function loadLogs() {
    try {
        if (!window.go?.main?.App) return;

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

    // Check if user is at the bottom before updating content
    // Allow 50px tolerance for "near bottom" detection
    const isAtBottom = textarea.scrollHeight - textarea.scrollTop - textarea.clientHeight < 50;

    const logText = logs.map(log => {
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

    // Only auto-scroll to bottom if user was already at the bottom
    if (isAtBottom) {
        textarea.scrollTop = textarea.scrollHeight;
    }
}

export function toggleLogPanel() {
    const panel = document.getElementById('logPanel');
    const icon = document.getElementById('logToggleIcon');
    const text = document.getElementById('logToggleText');

    logPanelExpanded = !logPanelExpanded;

    if (logPanelExpanded) {
        panel.style.display = 'block';
        icon.textContent = '🔼';
        text.textContent = t('logs.collapse');
    } else {
        panel.style.display = 'none';
        icon.textContent = '🔽';
        text.textContent = t('logs.expand');
    }
}

export async function changeLogLevel() {
    const level = parseInt(document.getElementById('logLevel').value);
    try {
        await window.go.main.App.SetLogLevel(level);
        loadLogs();
    } catch (error) {
        console.error('Failed to change log level:', error);
        alert('Failed to change log level: ' + error);
    }
}

export function copyLogs() {
    const textarea = document.getElementById('logContent');
    textarea.select();
    document.execCommand('copy');

    const btn = event.target.closest('button');
    const originalText = btn.innerHTML;
    btn.innerHTML = '✅ Copied!';
    setTimeout(() => {
        btn.innerHTML = originalText;
    }, 1500);
}

export async function clearLogs() {
    try {
        await window.go.main.App.ClearLogs();
        loadLogs();
    } catch (error) {
        console.error('Failed to clear logs:', error);
        alert('Failed to clear logs: ' + error);
    }
}
