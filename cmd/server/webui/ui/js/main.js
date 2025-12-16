import { router } from './router.js';
import { state } from './state.js';
import { dashboard } from './components/dashboard.js';
import { endpoints } from './components/endpoints.js';
import { stats } from './components/stats.js';
import { testing } from './components/testing.js';

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
