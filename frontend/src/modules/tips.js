// Tips module - displays scrolling tips at the bottom of the page
import { t } from '../i18n/index.js';

let currentTipIndex = 0;

// Initialize tips functionality
export function initTips() {
    const tipElement = document.getElementById('scrollingTip');
    if (!tipElement) return;

    // Listen for animation end event
    tipElement.addEventListener('animationend', () => {
        showNextTip();
    });

    // Start showing first tip
    showNextTip();
}

// Show next tip
function showNextTip() {
    const tips = t('tips');
    if (!tips || tips.length === 0) return;

    // Get a random tip (avoid showing the same tip twice in a row)
    let newIndex;
    do {
        newIndex = Math.floor(Math.random() * tips.length);
    } while (newIndex === currentTipIndex && tips.length > 1);

    currentTipIndex = newIndex;
    const tip = tips[currentTipIndex];

    // Update tip content
    const tipElement = document.getElementById('scrollingTip');
    if (tipElement) {
        // Remove animation class
        tipElement.classList.remove('tip-scroll');

        // Force reflow to restart animation
        void tipElement.offsetWidth;

        // Update content and add animation class
        tipElement.textContent = tip;
        tipElement.classList.add('tip-scroll');
    }
}

// Stop tips (cleanup)
export function stopTips() {
    const tipElement = document.getElementById('scrollingTip');
    if (tipElement) {
        tipElement.removeEventListener('animationend', showNextTip);
    }
}

// Manually trigger next tip (for testing or user interaction)
export function nextTip() {
    showNextTip();
}
