// 端点筛选功能模块
import { getEndpointTestStatus } from './endpoints.js';
import { t } from '../i18n/index.js';

// 筛选状态（多选数组）
let filterState = {
    types: [],            // ['claude', 'gemini', 'openai', 'openai2']
    availabilities: [],   // ['available', 'unknown', 'unavailable']
    enabledStates: []     // ['enabled', 'disabled']
};

// 获取当前筛选状态
export function getFilterState() {
    return { ...filterState };
}

// 检查是否有激活的筛选
export function isFilterActive() {
    return filterState.types.length > 0 ||
           filterState.availabilities.length > 0 ||
           filterState.enabledStates.length > 0;
}

// 清除所有筛选
export function clearAllFilters() {
    filterState = {
        types: [],
        availabilities: [],
        enabledStates: []
    };

    // 清空所有复选框
    document.querySelectorAll('.filter-dropdown-panel input[type="checkbox"]').forEach(cb => {
        cb.checked = false;
    });

    saveFilterState();
    updateAllBadges();
    applyFilters();
}

// 持久化筛选状态
const FILTER_STATE_KEY = 'ccnexus_filter_state_v3';

function saveFilterState() {
    try {
        localStorage.setItem(FILTER_STATE_KEY, JSON.stringify(filterState));
    } catch (error) {
        console.error('Failed to save filter state:', error);
    }
}

function loadFilterState() {
    try {
        const saved = localStorage.getItem(FILTER_STATE_KEY);
        if (saved) {
            filterState = { ...filterState, ...JSON.parse(saved) };
        }
    } catch (error) {
        console.error('Failed to load filter state:', error);
    }
}

// 初始化筛选下拉框
export function initFilterDropdowns() {
    loadFilterState();

    // 为每个下拉按钮绑定事件
    document.querySelectorAll('.filter-dropdown').forEach(dropdown => {
        const btn = dropdown.querySelector('.filter-dropdown-btn');
        const panel = dropdown.querySelector('.filter-dropdown-panel');
        const filterKey = dropdown.dataset.filter; // 'types' | 'availabilities' | 'enabledStates'

        // 点击按钮切换面板
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            togglePanel(dropdown);
        });

        // 复选框变化时更新临时状态（不立即应用）
        const checkboxes = panel.querySelectorAll('input[type="checkbox"]');
        checkboxes.forEach(cb => {
            // 初始化勾选状态
            cb.checked = filterState[filterKey].includes(cb.value);

            cb.addEventListener('change', () => {
                updateTempState(filterKey, checkboxes);
            });
        });

        // "清空"按钮
        panel.querySelector('.btn-clear-dimension').addEventListener('click', () => {
            checkboxes.forEach(cb => cb.checked = false);
            updateTempState(filterKey, checkboxes);
        });

        // "确定"按钮
        panel.querySelector('.btn-apply').addEventListener('click', () => {
            applyTempState(filterKey, checkboxes);
            closePanel(dropdown);
        });
    });

    // 点击外部关闭所有面板
    document.addEventListener('click', () => {
        closeAllPanels();
    });

    // 阻止面板内点击冒泡
    document.querySelectorAll('.filter-dropdown-panel').forEach(panel => {
        panel.addEventListener('click', (e) => {
            e.stopPropagation();
        });
    });

    // 初始化徽章显示
    updateAllBadges();
}

// 切换面板显示
function togglePanel(dropdown) {
    const panel = dropdown.querySelector('.filter-dropdown-panel');
    const arrow = dropdown.querySelector('.filter-arrow');
    const isOpen = !panel.classList.contains('hidden');

    // 关闭所有其他面板
    closeAllPanels();

    if (!isOpen) {
        panel.classList.remove('hidden');
        arrow.textContent = '▲';
    }
}

// 关闭所有面板
function closeAllPanels() {
    document.querySelectorAll('.filter-dropdown-panel').forEach(panel => {
        panel.classList.add('hidden');
    });
    document.querySelectorAll('.filter-arrow').forEach(arrow => {
        arrow.textContent = '▼';
    });
}

// 关闭单个面板
function closePanel(dropdown) {
    dropdown.querySelector('.filter-dropdown-panel').classList.add('hidden');
    dropdown.querySelector('.filter-arrow').textContent = '▼';
}

// 临时状态（用于"确定"前的预览）
let tempState = {};

// 更新临时状态（仅内存，不持久化，不应用筛选）
function updateTempState(filterKey, checkboxes) {
    tempState[filterKey] = Array.from(checkboxes)
        .filter(cb => cb.checked)
        .map(cb => cb.value);
}

// 应用临时状态到实际筛选
function applyTempState(filterKey, checkboxes) {
    filterState[filterKey] = Array.from(checkboxes)
        .filter(cb => cb.checked)
        .map(cb => cb.value);

    saveFilterState();
    updateBadge(filterKey);
    applyFilters();
}

// 更新徽章显示
function updateBadge(filterKey) {
    const count = filterState[filterKey].length;
    const badge = document.getElementById(`filterBadge${capitalizeFirst(filterKey)}`);
    const btn = badge?.closest('.filter-dropdown-btn');

    if (badge) {
        if (count > 0) {
            badge.textContent = count;
            badge.classList.remove('hidden');
            btn?.classList.add('active');
        } else {
            badge.classList.add('hidden');
            btn?.classList.remove('active');
        }
    }
}

function updateAllBadges() {
    updateBadge('types');
    updateBadge('availabilities');
    updateBadge('enabledStates');
}

function capitalizeFirst(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

// 应用筛选
function applyFilters() {
    // 显示/隐藏筛选激活提示条
    const banner = document.getElementById('filterActiveBanner');
    if (isFilterActive()) {
        banner?.classList.remove('hidden');
    } else {
        banner?.classList.add('hidden');
    }

    // 更新清除筛选按钮显示
    const clearBtn = document.getElementById('filterClearBtn');
    if (clearBtn) {
        if (isFilterActive()) {
            clearBtn.classList.remove('hidden');
        } else {
            clearBtn.classList.add('hidden');
        }
    }

    // 触发端点列表重新渲染
    if (window.loadConfig) {
        window.loadConfig();
    }
}

// 筛选端点（多选逻辑：维度内 OR，维度间 AND）
export function filterEndpoints(endpoints) {
    const { types, availabilities, enabledStates } = filterState;

    return endpoints.filter(ep => {
        // 1. 类型筛选（维度内 OR）
        if (types.length > 0) {
            const epType = ep.transformer || 'claude';
            if (!types.includes(epType)) return false;
        }

        // 2. 可用性筛选（维度内 OR）
        if (availabilities.length > 0) {
            const testStatus = getEndpointTestStatus(ep.name);
            let matchesAvailability = false;

            for (const av of availabilities) {
                if (av === 'available' && testStatus === true) {
                    matchesAvailability = true;
                    break;
                }
                if (av === 'unknown' && (testStatus === undefined || testStatus === 'unknown')) {
                    matchesAvailability = true;
                    break;
                }
                if (av === 'unavailable' && testStatus === false) {
                    matchesAvailability = true;
                    break;
                }
            }

            if (!matchesAvailability) return false;
        }

        // 3. 启用状态筛选（维度内 OR）
        if (enabledStates.length > 0) {
            const isEnabled = ep.enabled !== undefined ? ep.enabled : true;
            let matchesEnabled = false;

            for (const state of enabledStates) {
                if (state === 'enabled' && isEnabled) {
                    matchesEnabled = true;
                    break;
                }
                if (state === 'disabled' && !isEnabled) {
                    matchesEnabled = true;
                    break;
                }
            }

            if (!matchesEnabled) return false;
        }

        return true;
    });
}

// 更新筛选统计
export function updateFilterStats(total, filtered) {
    const statsEl = document.getElementById('filterStats');
    if (statsEl) {
        if (total === filtered) {
            statsEl.textContent = t('endpoints.filterStatsTotal').replace('{total}', total);
        } else {
            statsEl.textContent = t('endpoints.filterStatsFiltered')
                .replace('{filtered}', filtered)
                .replace('{total}', total);
        }
    }
}

// 暴露全局方法
if (typeof window !== 'undefined') {
    window.clearAllFilters = clearAllFilters;
}
