import { formatTokens } from '../utils/format.js';
import { t } from '../i18n/index.js';

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
        return;
    }

    selector.disabled = false;

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

// Load and display archive data
async function loadAndDisplayArchive(month) {
    const archive = await loadArchiveData(month);
    if (!archive) return;

    // Update summary cards
    updateSummaryCards(archive.summary);

    // Render daily details table
    renderDailyTable(archive.endpoints);
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
