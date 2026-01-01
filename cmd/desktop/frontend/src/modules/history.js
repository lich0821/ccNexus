import { formatTokens } from '../utils/format.js';
import { t } from '../i18n/index.js';
import { showConfirm } from './modal.js';
import { showNotification } from './modal.js';

let currentArchiveMonth = null;
let archivesList = [];

// Load list of available archives
export async function loadArchiveList() {
    try {
        const result = await window.go.main.App.ListArchives();
        const data = JSON.parse(result);

        if (!data.success) {
            console.error('Failed to load archives:', data.message);
            return [];
        }

        archivesList = data.archives || [];
        return archivesList;
    } catch (error) {
        console.error('Failed to load archive list:', error);
        return [];
    }
}

// Load archive data for a specific month
export async function loadArchiveData(month) {
    try {
        const result = await window.go.main.App.GetArchiveData(month);
        const data = JSON.parse(result);

        if (!data.success) {
            console.error('Failed to load archive:', data.message);
            showError(data.message);
            return null;
        }

        currentArchiveMonth = month;
        return data.archive;
    } catch (error) {
        console.error('Failed to load archive data:', error);
        showError(t('history.loadFailed'));
        return null;
    }
}

// Show history statistics modal
export async function showHistoryModal() {
    const modal = document.getElementById('historyModal');
    if (!modal) return;

    // Show modal
    modal.style.display = 'flex';

    // Load archives list
    const archives = await loadArchiveList();

    // Populate month selector
    populateMonthSelector(archives);

    // Load first archive if available
    if (archives.length > 0) {
        await loadAndDisplayArchive(archives[0]);
    } else {
        showNoDataMessage();
    }
}

// Close history statistics modal
export function closeHistoryModal() {
    const modal = document.getElementById('historyModal');
    if (modal) {
        modal.style.display = 'none';
    }
}

// Legacy functions for backward compatibility
export function showHistoryView() {
    showHistoryModal();
}

export function hideHistoryView() {
    closeHistoryModal();
}

// Populate month selector dropdown
function populateMonthSelector(archives) {
    const selector = document.getElementById('historyMonthSelect');
    if (!selector) return;

    // Clear existing options
    selector.innerHTML = '';

    if (archives.length === 0) {
        const option = document.createElement('option');
        option.value = '';
        option.textContent = t('history.noData');
        selector.appendChild(option);
        selector.disabled = true;
        updateDeleteButtonState(false);
        return;
    }

    selector.disabled = false;
    updateDeleteButtonState(true);

    // Add options for each archive
    archives.forEach(month => {
        const option = document.createElement('option');
        option.value = month;
        option.textContent = formatMonthDisplay(month);
        selector.appendChild(option);
    });

    // Add change event listener
    selector.onchange = async (e) => {
        const selectedMonth = e.target.value;
        if (selectedMonth) {
            await loadAndDisplayArchive(selectedMonth);
        }
    };
}

// Format month for display (YYYY-MM -> YYYY年MM月 or YYYY-MM)
function formatMonthDisplay(month) {
    const lang = localStorage.getItem('language') || 'zh-CN';
    const [year, monthNum] = month.split('-');

    if (lang === 'zh-CN') {
        return `${year}年${monthNum}月`;
    } else {
        const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
                           'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
        return `${monthNames[parseInt(monthNum) - 1]} ${year}`;
    }
}

// Load archive trend data for a specific month
async function loadArchiveTrend(month) {
    try {
        const result = await window.go.main.App.GetArchiveTrend(month);
        const data = JSON.parse(result);

        if (!data.success) {
            console.error('Failed to load archive trend:', data.message);
            return null;
        }

        return {
            requestsTrend: data.trend || 0,
            errorsTrend: data.errorsTrend || 0,
            tokensTrend: data.tokensTrend || 0
        };
    } catch (error) {
        console.error('Failed to load archive trend:', error);
        return null;
    }
}

// Format trend value for display
function formatTrend(value) {
    const absValue = Math.abs(value);
    const formattedValue = absValue.toFixed(1);

    if (value > 0) {
        return {
            text: `↑ ${formattedValue}%`,
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

// Load and display archive data
async function loadAndDisplayArchive(month) {
    const archive = await loadArchiveData(month);
    if (!archive) return;

    // Update summary cards
    updateSummaryCards(archive.summary);

    // Load and display trend
    const trend = await loadArchiveTrend(month);
    if (trend) {
        updateTrendDisplay(trend);
    }

    // Render daily details table
    renderDailyTable(archive.endpoints);
}

// Update trend display
function updateTrendDisplay(trend) {
    const requestsTrend = formatTrend(trend.requestsTrend);
    const tokensTrend = formatTrend(trend.tokensTrend);

    const requestsEl = document.getElementById('historyRequestsTrend');
    const tokensEl = document.getElementById('historyTokensTrend');

    if (requestsEl) {
        requestsEl.textContent = requestsTrend.text;
        requestsEl.className = 'trend ' + requestsTrend.className;
    }

    if (tokensEl) {
        tokensEl.textContent = tokensTrend.text;
        tokensEl.className = 'trend ' + tokensTrend.className;
    }
}

// Update summary statistics cards
function updateSummaryCards(summary) {
    // Total requests
    const totalRequestsEl = document.getElementById('historyTotalRequests');
    if (totalRequestsEl) {
        totalRequestsEl.textContent = summary.totalRequests || 0;
    }

    // Success/Failed
    const successEl = document.getElementById('historySuccess');
    const failedEl = document.getElementById('historyFailed');
    if (successEl && failedEl) {
        const success = (summary.totalRequests || 0) - (summary.totalErrors || 0);
        successEl.textContent = success;
        failedEl.textContent = summary.totalErrors || 0;
    }

    // Tokens
    const totalTokens = (summary.totalInputTokens || 0) + (summary.totalOutputTokens || 0);
    const totalTokensEl = document.getElementById('historyTotalTokens');
    const inputTokensEl = document.getElementById('historyInputTokens');
    const outputTokensEl = document.getElementById('historyOutputTokens');

    if (totalTokensEl) {
        totalTokensEl.textContent = formatTokens(totalTokens);
    }
    if (inputTokensEl) {
        inputTokensEl.textContent = formatTokens(summary.totalInputTokens || 0);
    }
    if (outputTokensEl) {
        outputTokensEl.textContent = formatTokens(summary.totalOutputTokens || 0);
    }
}

// Render daily details table
function renderDailyTable(endpoints) {
    const tbody = document.querySelector('#historyDailyTable tbody');
    if (!tbody) return;

    // Clear existing rows
    tbody.innerHTML = '';

    // Collect all daily data
    const dailyDataMap = new Map();

    for (const [endpointName, endpointData] of Object.entries(endpoints)) {
        for (const [date, daily] of Object.entries(endpointData.dailyHistory || {})) {
            if (!dailyDataMap.has(date)) {
                dailyDataMap.set(date, {
                    date: date,
                    requests: 0,
                    errors: 0,
                    inputTokens: 0,
                    outputTokens: 0
                });
            }

            const dayData = dailyDataMap.get(date);
            dayData.requests += daily.requests || 0;
            dayData.errors += daily.errors || 0;
            dayData.inputTokens += daily.inputTokens || 0;
            dayData.outputTokens += daily.outputTokens || 0;
        }
    }

    // Sort by date
    const sortedDates = Array.from(dailyDataMap.keys()).sort();

    // Create table rows
    sortedDates.forEach(date => {
        const data = dailyDataMap.get(date);
        const totalTokens = data.inputTokens + data.outputTokens;
        const row = document.createElement('tr');

        row.innerHTML = `
            <td>${date}</td>
            <td>${data.requests}</td>
            <td>${data.errors}</td>
            <td>${formatTokens(data.inputTokens)}</td>
            <td>${formatTokens(data.outputTokens)}</td>
            <td>${formatTokens(totalTokens)}</td>
        `;

        tbody.appendChild(row);
    });

    // Show "no data" message if empty
    if (sortedDates.length === 0) {
        const row = document.createElement('tr');
        row.innerHTML = `<td colspan="6" style="text-align: center; padding: 20px;">${t('history.noData')}</td>`;
        tbody.appendChild(row);
    }
}

// Show error message
function showError(message) {
    const errorEl = document.getElementById('historyError');
    if (errorEl) {
        errorEl.textContent = message;
        errorEl.style.display = 'block';

        setTimeout(() => {
            errorEl.style.display = 'none';
        }, 5000);
    }
}

// Show no data message
function showNoDataMessage() {
    const tbody = document.querySelector('#historyDailyTable tbody');
    if (tbody) {
        tbody.innerHTML = `<tr><td colspan="6" style="text-align: center; padding: 20px;">${t('history.noArchives')}</td></tr>`;
    }

    // Clear summary cards
    updateSummaryCards({
        totalRequests: 0,
        totalErrors: 0,
        totalInputTokens: 0,
        totalOutputTokens: 0
    });
}

// Get current archive month
export function getCurrentArchiveMonth() {
    return currentArchiveMonth;
}

// Get archives list
export function getArchivesList() {
    return archivesList;
}

// Format month for display in confirm message
function formatMonthForConfirm(month) {
    const lang = localStorage.getItem('language') || 'zh-CN';
    const [year, monthNum] = month.split('-');

    if (lang === 'zh-CN') {
        return `${year}年${monthNum}月`;
    } else {
        const monthNames = ['January', 'February', 'March', 'April', 'May', 'June',
                           'July', 'August', 'September', 'October', 'November', 'December'];
        return `${monthNames[parseInt(monthNum) - 1]} ${year}`;
    }
}

// Delete archive for current selected month
export async function deleteHistoryArchive() {
    const selector = document.getElementById('historyMonthSelect');
    if (!selector || !selector.value) {
        showNotification(t('history.noMonthSelected'), 'error');
        return;
    }

    const month = selector.value;
    const monthDisplay = formatMonthForConfirm(month);

    // Show confirm dialog
    const confirmed = await showConfirm(t('history.confirmDelete').replace('{month}', monthDisplay));
    if (!confirmed) {
        return;
    }

    try {
        const result = await window.go.main.App.DeleteArchive(month);
        const data = JSON.parse(result);

        if (data.success) {
            showNotification(t('history.deleteSuccess'), 'success');

            // Reload archives list
            const archives = await loadArchiveList();
            populateMonthSelector(archives);

            // Load first archive if available, otherwise show no data
            if (archives.length > 0) {
                await loadAndDisplayArchive(archives[0]);
            } else {
                showNoDataMessage();
                // Update delete button state
                updateDeleteButtonState(false);
            }
        } else {
            showNotification(data.message || t('history.deleteFailed'), 'error');
        }
    } catch (error) {
        console.error('Failed to delete archive:', error);
        showNotification(t('history.deleteFailed'), 'error');
    }
}

// Update delete button visibility/state
function updateDeleteButtonState(hasData) {
    const deleteBtn = document.getElementById('historyDeleteBtn');
    if (deleteBtn) {
        deleteBtn.style.display = hasData ? 'inline-flex' : 'none';
    }
}
