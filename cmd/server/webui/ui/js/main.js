import { router } from './router.js';
import { state } from './state.js';
import { setLanguage, getLanguage, initLanguage, loadTranslations, t } from './utils/i18n.js';
import { dashboard } from './components/dashboard.js';
import { endpoints } from './components/endpoints.js';
import { stats } from './components/stats.js';
import { testing } from './components/testing.js';
import zhCN from './i18n/zh-CN.js';
import en from './i18n/en.js';

// 加载翻译
loadTranslations({ 'zh-CN': zhCN, 'en': en });
initLanguage();

// Initialize theme
function initTheme() {
    const savedTheme = localStorage.getItem('theme') || 'light';
    document.body.classList.toggle('dark-theme', savedTheme === 'dark');

    const themeToggle = document.getElementById('theme-toggle');
    themeToggle.addEventListener('click', () => {
        const isDark = document.body.classList.toggle('dark-theme');
        localStorage.setItem('theme', isDark ? 'dark' : 'light');
        themeToggle.querySelector('.icon').textContent = isDark ? '☀️' : '🌙';
    });

    // Set initial icon
    themeToggle.querySelector('.icon').textContent = savedTheme === 'dark' ? '☀️' : '🌙';
}

// Update sidebar translations
function updateSidebarTranslations() {
    // Update subtitle
    const subtitle = document.getElementById('sidebar-subtitle');
    if (subtitle) {
        subtitle.textContent = t('dashboard.subtitle');
    }

    // Update navigation menu items
    document.querySelectorAll('.nav-label').forEach(el => {
        const key = el.getAttribute('data-i18n');
        if (key) {
            el.textContent = t(key);
        }
    });
}

// Initialize language toggle
function initLanguageToggle() {
    const langToggle = document.getElementById('lang-toggle');
    const langLabels = {
        'zh-CN': '🇨🇳',
        'en': '🇺🇸'
    };

    // 设置初始图标
    langToggle.querySelector('.icon').textContent = langLabels[getLanguage()];

    langToggle.addEventListener('click', () => {
        const currentLang = getLanguage();
        const newLang = currentLang === 'zh-CN' ? 'en' : 'zh-CN';
        setLanguage(newLang);
        langToggle.querySelector('.icon').textContent = langLabels[newLang];
        // 更新侧边栏翻译
        updateSidebarTranslations();
        // 重新加载当前视图
        const currentView = state.get('currentView');
        if (currentView) {
            router.navigate(currentView);
        }
    });
}

// Initialize real-time updates
function initRealtime() {
    const eventSource = new EventSource('/api/events');

    eventSource.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);

            if (data.type === 'stats') {
                state.update('stats', data.stats);
                state.update('currentEndpoint', data.currentEndpoint);

                // Update dashboard if it's the current view
                if (state.get('currentView') === 'dashboard') {
                    // Dashboard will handle its own updates via state subscription
                }
            }
        } catch (error) {
            console.error('Failed to parse SSE event:', error);
        }
    };

    eventSource.onerror = (error) => {
        console.error('SSE connection error:', error);
        // Attempt to reconnect after 5 seconds
        setTimeout(() => {
            if (eventSource.readyState === EventSource.CLOSED) {
                initRealtime();
            }
        }, 5000);
    };
}

// Initialize application
function init() {
    // Register routes
    router.register('dashboard', dashboard);
    router.register('endpoints', endpoints);
    router.register('stats', stats);
    router.register('testing', testing);

    // Initialize theme
    initTheme();

    // Initialize language toggle
    initLanguageToggle();

    // Initialize sidebar translations
    updateSidebarTranslations();

    // Initialize router
    router.init();

    // Initialize real-time updates
    initRealtime();

    console.log('ccNexus Admin initialized');
}

// Start application when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
