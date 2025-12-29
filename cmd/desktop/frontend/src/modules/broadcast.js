// ========== 广播横幅模块 ==========

import { t, getLanguage } from '../i18n/index.js';

// 广播URL配置
const BROADCAST_URL = 'https://gitee.com/hea7en/images/raw/master/group/message.json';

// 状态
let currentIndex = 0;
let messages = [];
let config = { carouselInterval: 10, refreshInterval: 60 };
let carouselTimer = null;
let refreshTimer = null;
let isHidden = false;

// 图标映射
const ICONS = {
    info: '📢',
    warning: '⚠️',
    error: '❌',
    success: '✅'
};

// 初始化广播
export async function initBroadcast() {
    await fetchAndRender();
    // 定时刷新
    if (refreshTimer) clearInterval(refreshTimer);
    refreshTimer = setInterval(fetchAndRender, config.refreshInterval * 1000);
}

// 获取并渲染
async function fetchAndRender() {
    try {
        const url = BROADCAST_URL + '?t=' + Date.now();
        const json = await window.go.main.App.FetchBroadcast(url);
        if (!json) return hideBanner();

        const data = JSON.parse(json);
        if (!data.enabled || !data.messages || data.messages.length === 0) {
            return hideBanner();
        }

        // 更新配置
        if (data.config) {
            config = { ...config, ...data.config };
        }

        // 过滤有效消息
        messages = filterValidMessages(data.messages);
        if (messages.length === 0) return hideBanner();

        currentIndex = 0;
        renderBanner();
        startCarousel();
    } catch (e) {
        hideBanner();
    }
}

// 过滤有效消息（检查时间范围）
function filterValidMessages(msgs) {
    const now = new Date();
    return msgs.filter(msg => {
        if (msg.startTime && parseTime(msg.startTime) > now) return false;
        if (msg.endTime && parseTime(msg.endTime) < now) return false;
        return true;
    });
}

// 解析时间字符串，支持 "2025-12-01 00:00:00" 格式
function parseTime(str) {
    return new Date(str.replace(' ', 'T'));
}

// 渲染横幅
function renderBanner() {
    if (isHidden || messages.length === 0) return;

    const banner = document.getElementById('broadcast-banner');
    if (!banner) return;

    const msg = messages[currentIndex];
    const lang = getLanguage();
    const content = lang === 'zh-CN' ? msg.content : (msg.content_en || msg.content);
    const type = msg.type || 'info';
    const icon = ICONS[type] || ICONS.info;

    banner.className = `broadcast-banner ${type}`;
    banner.innerHTML = `
        <span class="broadcast-banner-icon">${icon}</span>
        <div class="broadcast-banner-text-wrapper">
            <span class="broadcast-banner-text" ${msg.link ? 'style="cursor:pointer"' : ''}>${content} <span class="broadcast-banner-close" title="关闭">✕</span></span>
        </div>
    `;

    // 绑定事件
    banner.querySelector('.broadcast-banner-close').onclick = (e) => {
        e.stopPropagation();
        closeBanner();
    };
    if (msg.link) {
        banner.querySelector('.broadcast-banner-text').onclick = () => {
            window.go.main.App.OpenURL(msg.link);
        };
    }

    banner.classList.remove('hidden');

    // 检查是否需要滚动（内容超出wrapper时）
    setTimeout(() => {
        const wrapper = banner.querySelector('.broadcast-banner-text-wrapper');
        const textEl = banner.querySelector('.broadcast-banner-text');
        if (wrapper && textEl && textEl.scrollWidth > wrapper.clientWidth) {
            // 根据文字长度计算滚动时间，每100px约2秒
            const duration = Math.max(10, Math.ceil(textEl.scrollWidth / 50));
            textEl.style.setProperty('--scroll-duration', `${duration}s`);
            textEl.classList.add('scroll');
        }
    }, 100);
}

// 启动轮播
function startCarousel() {
    if (carouselTimer) clearInterval(carouselTimer);
    if (messages.length <= 1) return;

    carouselTimer = setInterval(() => {
        currentIndex = (currentIndex + 1) % messages.length;
        renderBanner();
    }, config.carouselInterval * 1000);
}

// 关闭横幅
function closeBanner() {
    isHidden = true;
    hideBanner();
    if (carouselTimer) clearInterval(carouselTimer);
}

// 隐藏横幅
function hideBanner() {
    const banner = document.getElementById('broadcast-banner');
    if (banner) banner.classList.add('hidden');
}
