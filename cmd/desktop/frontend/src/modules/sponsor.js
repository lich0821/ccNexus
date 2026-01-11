// ========== 赞助模块 ==========

import { t } from '../i18n/index.js';

// 赞助配置URL
const SPONSOR_URL = 'https://gitee.com/hea7en/images/raw/master/group/sponsor.json';

// 状态
let sponsorConfig = null;

// 初始化赞助模块
export async function initSponsor() {
    await fetchSponsorConfig();
}

// 获取赞助配置
async function fetchSponsorConfig() {
    try {
        const url = SPONSOR_URL + '?t=' + Date.now();
        const json = await window.go.main.App.FetchBroadcast(url);
        if (!json) return hideRibbon();

        const data = JSON.parse(json);
        if (!data.enabled) {
            return hideRibbon();
        }

        sponsorConfig = data;
        showRibbon();
    } catch (e) {
        hideRibbon();
    }
}

// 显示横幅
function showRibbon() {
    const ribbon = document.querySelector('.ribbon-banner');
    if (ribbon) ribbon.classList.remove('hidden');
}

// 隐藏横幅
function hideRibbon() {
    const ribbon = document.querySelector('.ribbon-banner');
    if (ribbon) ribbon.classList.add('hidden');
}

// 显示赞助弹窗
export function showSponsorModal() {
    if (!sponsorConfig || !sponsorConfig.items) return;

    const modal = document.getElementById('sponsorModal');
    if (!modal) return;

    // 渲染赞助卡片
    const grid = modal.querySelector('.sponsor-grid');
    if (grid) {
        grid.innerHTML = sponsorConfig.items.map(item => `
            <div class="sponsor-card" onclick="window.openSponsorLink('${item.link}')">
                <div class="sponsor-icon">${item.icon}</div>
                <div class="sponsor-name">${item.name}</div>
                <div class="sponsor-desc">${item.description}</div>
                <div class="sponsor-amount">${item.amount}</div>
            </div>
        `).join('');
    }

    // 设置标题
    const title = modal.querySelector('.modal-header h2');
    if (title) title.textContent = sponsorConfig.title || t('sponsor.title');

    modal.classList.add('active');
}

// 关闭赞助弹窗
export function closeSponsorModal() {
    const modal = document.getElementById('sponsorModal');
    if (modal) modal.classList.remove('active');
}

// 打开赞助链接
export function openSponsorLink(link) {
    if (link && window.go?.main?.App) {
        window.go.main.App.OpenURL(link);
    }
}
