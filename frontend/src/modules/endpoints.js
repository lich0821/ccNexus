import { t } from '../i18n/index.js';
import { formatTokens, maskApiKey } from '../utils/format.js';
import { getEndpointStats } from './stats.js';
import { toggleEndpoint, moveEndpoint } from './config.js';

let currentTestButton = null;
let currentTestButtonOriginalText = '';
let currentTestIndex = -1;

export function getTestState() {
    return { currentTestButton, currentTestIndex };
}

export function clearTestState() {
    if (currentTestButton) {
        currentTestButton.disabled = false;
        currentTestButton.innerHTML = currentTestButtonOriginalText;
        currentTestButton = null;
        currentTestButtonOriginalText = '';
        currentTestIndex = -1;
    }
}

export function setTestState(button, index) {
    currentTestButton = button;
    currentTestButtonOriginalText = button.innerHTML;
    currentTestIndex = index;
}

export async function renderEndpoints(endpoints) {
    const container = document.getElementById('endpointList');

    // Get current endpoint
    let currentEndpointName = '';
    try {
        currentEndpointName = await window.go.main.App.GetCurrentEndpoint();
    } catch (error) {
        console.error('Failed to get current endpoint:', error);
    }

    if (endpoints.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <p>${t('endpoints.noEndpoints')}</p>
            </div>
        `;
        return;
    }

    container.innerHTML = '';

    const endpointStats = getEndpointStats();

    let draggedIndex = null;
    let autoScrollInterval = null;

    endpoints.forEach((ep, index) => {
        const stats = endpointStats[ep.name] || { requests: 0, errors: 0, inputTokens: 0, outputTokens: 0 };
        const totalTokens = stats.inputTokens + stats.outputTokens;
        const enabled = ep.enabled !== undefined ? ep.enabled : true;
        const transformer = ep.transformer || 'claude';
        const model = ep.model || '';
        const isCurrentEndpoint = ep.name === currentEndpointName;

        const item = document.createElement('div');
        item.className = 'endpoint-item';
        item.draggable = true;
        item.setAttribute('data-index', index);
        item.innerHTML = `
            <div class="drag-handle">⋮⋮</div>
            <div class="endpoint-info">
                <h3>
                    ${ep.name}
                    ${enabled ? '✅' : '❌'}
                    ${isCurrentEndpoint ? '<span class="current-badge">' + t('endpoints.current') + '</span>' : ''}
                    ${enabled && !isCurrentEndpoint ? '<button class="btn btn-switch" data-action="switch" data-name="' + ep.name + '">' + t('endpoints.switchTo') + '</button>' : ''}
                </h3>
                <p style="display: flex; align-items: center; gap: 8px; min-width: 0;"><span style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">🌐 ${ep.apiUrl}</span> <button class="copy-btn" data-copy="${ep.apiUrl}" aria-label="${t('endpoints.copy')}" title="${t('endpoints.copy')}"><svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" width="1em" height="1em"><path d="M7 4c0-1.1.9-2 2-2h11a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2h-1V8c0-2-1-3-3-3H7V4Z" fill="currentColor"></path><path d="M5 7a2 2 0 0 0-2 2v10c0 1.1.9 2 2 2h10a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2H5Z" fill="currentColor"></path></svg></button></p>
                <p style="display: flex; align-items: center; gap: 8px; min-width: 0;"><span style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">🔑 ${maskApiKey(ep.apiKey)}</span> <button class="copy-btn" data-copy="${ep.apiKey}" aria-label="${t('endpoints.copy')}" title="${t('endpoints.copy')}"><svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" width="1em" height="1em"><path d="M7 4c0-1.1.9-2 2-2h11a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2h-1V8c0-2-1-3-3-3H7V4Z" fill="currentColor"></path><path d="M5 7a2 2 0 0 0-2 2v10c0 1.1.9 2 2 2h10a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2H5Z" fill="currentColor"></path></svg></button></p>
                <p style="color: #666; font-size: 14px; margin-top: 5px;">🔄 ${t('endpoints.transformer')}: ${transformer}${model ? ` (${model})` : ''}</p>
                <p style="color: #666; font-size: 14px; margin-top: 3px;">📊 ${t('endpoints.requests')}: ${stats.requests} | ${t('endpoints.errors')}: ${stats.errors}</p>
                <p style="color: #666; font-size: 14px; margin-top: 3px;">🎯 ${t('endpoints.tokens')}: ${formatTokens(totalTokens)} (${t('statistics.in')}: ${formatTokens(stats.inputTokens)}, ${t('statistics.out')}: ${formatTokens(stats.outputTokens)})</p>
            </div>
            <div class="endpoint-actions">
                <label class="toggle-switch">
                    <input type="checkbox" data-index="${index}" ${enabled ? 'checked' : ''}>
                    <span class="toggle-slider"></span>
                </label>
                <button class="btn btn-secondary" data-action="test" data-index="${index}">${t('endpoints.test')}</button>
                <button class="btn btn-secondary" data-action="edit" data-index="${index}">${t('endpoints.edit')}</button>
                <button class="btn btn-danger" data-action="delete" data-index="${index}">${t('endpoints.delete')}</button>
            </div>
        `;

        const testBtn = item.querySelector('[data-action="test"]');
        const editBtn = item.querySelector('[data-action="edit"]');
        const deleteBtn = item.querySelector('[data-action="delete"]');
        const toggleSwitch = item.querySelector('input[type="checkbox"]');

        if (currentTestIndex === index) {
            testBtn.disabled = true;
            testBtn.innerHTML = '⏳';
            currentTestButton = testBtn;
        }

        testBtn.addEventListener('click', () => {
            const idx = parseInt(testBtn.getAttribute('data-index'));
            window.testEndpoint(idx, testBtn);
        });
        editBtn.addEventListener('click', () => {
            const idx = parseInt(editBtn.getAttribute('data-index'));
            window.editEndpoint(idx);
        });
        deleteBtn.addEventListener('click', () => {
            const idx = parseInt(deleteBtn.getAttribute('data-index'));
            window.deleteEndpoint(idx);
        });
        toggleSwitch.addEventListener('change', async (e) => {
            const idx = parseInt(e.target.getAttribute('data-index'));
            const newEnabled = e.target.checked;
            try {
                await toggleEndpoint(idx, newEnabled);
                window.loadConfig();
            } catch (error) {
                console.error('Failed to toggle endpoint:', error);
                alert('Failed to toggle endpoint: ' + error);
                e.target.checked = !newEnabled;
            }
        });

        // Copy button event listeners
        const copyButtons = item.querySelectorAll('.copy-btn');
        copyButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                const textToCopy = btn.getAttribute('data-copy');
                copyToClipboard(textToCopy, btn);
            });
        });

        // Switch button event listener
        const switchBtn = item.querySelector('[data-action="switch"]');
        if (switchBtn) {
            switchBtn.addEventListener('click', async () => {
                const name = switchBtn.getAttribute('data-name');
                try {
                    switchBtn.disabled = true;
                    switchBtn.innerHTML = '⏳';
                    await window.go.main.App.SwitchToEndpoint(name);
                    window.loadConfig();
                } catch (error) {
                    console.error('Failed to switch endpoint:', error);
                    alert(t('endpoints.switchFailed') + ': ' + error);
                } finally {
                    if (switchBtn) {
                        switchBtn.disabled = false;
                        switchBtn.innerHTML = t('endpoints.switchTo');
                    }
                }
            });
        }

        // 拖拽事件监听器
        item.addEventListener('dragstart', (e) => {
            draggedIndex = parseInt(item.getAttribute('data-index'));
            item.classList.add('dragging');
            e.dataTransfer.effectAllowed = 'move';
            e.dataTransfer.setData('text/html', item.innerHTML);

            // 开始自动滚动
            startAutoScroll();
        });

        item.addEventListener('dragend', (e) => {
            item.classList.remove('dragging');

            // 清除所有拖拽相关的类
            container.querySelectorAll('.endpoint-item').forEach(el => {
                el.classList.remove('drag-over', 'drag-over-bottom');
            });

            // 停止自动滚动
            stopAutoScroll();
        });

        item.addEventListener('dragover', (e) => {
            e.preventDefault();
            e.dataTransfer.dropEffect = 'move';

            // 如果是正在被拖拽的元素，不处理
            if (item.classList.contains('dragging')) {
                return;
            }

            const currentIndex = parseInt(item.getAttribute('data-index'));

            // 清除所有拖拽反馈（排除正在拖拽的元素）
            container.querySelectorAll('.endpoint-item:not(.dragging)').forEach(el => {
                el.classList.remove('drag-over', 'drag-over-bottom');
            });

            // 如果拖拽到自己位置，不显示反馈
            if (draggedIndex === currentIndex) {
                return;
            }

            const box = item.getBoundingClientRect();
            const midpoint = box.top + box.height / 2;

            if (e.clientY < midpoint) {
                // 拖到元素上方
                item.classList.add('drag-over');
            } else {
                // 拖到元素下方
                item.classList.add('drag-over', 'drag-over-bottom');
            }
        });

        item.addEventListener('dragleave', (e) => {
            item.classList.remove('drag-over', 'drag-over-bottom');
        });

        item.addEventListener('drop', async (e) => {
            e.preventDefault();
            item.classList.remove('drag-over', 'drag-over-bottom');

            if (draggedIndex === null || draggedIndex === index) {
                return;
            }

            try {
                await moveEndpoint(draggedIndex, index);
                window.loadConfig();
            } catch (error) {
                console.error('Failed to move endpoint:', error);
                alert('Failed to move endpoint: ' + error);
            }
        });

        container.appendChild(item);
    });
}

// 自动滚动变量
let scrollInterval = null;
let mouseY = 0;

// 开始自动滚动
function startAutoScroll() {
    const scrollZone = 100;
    const scrollSpeed = 10;

    scrollInterval = setInterval(() => {
        const containerEl = document.querySelector('.container');
        if (!containerEl) return;

        const rect = containerEl.getBoundingClientRect();

        if (mouseY < rect.top + scrollZone) {
            containerEl.scrollTop -= scrollSpeed;
        } else if (mouseY > rect.bottom - scrollZone) {
            containerEl.scrollTop += scrollSpeed;
        }
    }, 50);

    document.addEventListener('dragover', updateMouseY);
}

// 停止自动滚动
function stopAutoScroll() {
    if (scrollInterval) {
        clearInterval(scrollInterval);
        scrollInterval = null;
    }
    document.removeEventListener('dragover', updateMouseY);
}

// 更新鼠标位置
function updateMouseY(e) {
    mouseY = e.clientY;
}

// 辅助函数：获取拖拽元素应该放置的位置
function getDragAfterElement(container, y) {
    const draggableElements = [...container.querySelectorAll('.endpoint-item:not(.dragging)')];

    return draggableElements.reduce((closest, child) => {
        const box = child.getBoundingClientRect();
        const offset = y - box.top - box.height / 2;

        if (offset < 0 && offset > closest.offset) {
            return { offset: offset, element: child };
        } else {
            return closest;
        }
    }, { offset: Number.NEGATIVE_INFINITY }).element;
}
