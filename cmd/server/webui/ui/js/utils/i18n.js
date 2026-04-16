// i18n 国际化模块
const translations = {};

let currentLanguage = 'zh-CN'; // 默认使用中文

export function setLanguage(lang) {
    if (translations[lang]) {
        currentLanguage = lang;
        localStorage.setItem('ccnexus-language', lang);
        // 触发语言切换事件，让所有组件重新渲染
        window.dispatchEvent(new CustomEvent('languageChanged', { detail: { language: lang } }));
    }
}

export function getLanguage() {
    return currentLanguage;
}

export function t(key) {
    const keys = key.split('.');
    let value = translations[currentLanguage];

    for (const k of keys) {
        if (value && typeof value === 'object') {
            value = value[k];
        } else {
            return key; // 找不到翻译时返回 key
        }
    }

    return value || key;
}

export function loadTranslations(langStrings) {
    Object.assign(translations, langStrings);
}

export function initLanguage() {
    const saved = localStorage.getItem('ccnexus-language');
    if (saved && translations[saved]) {
        currentLanguage = saved;
    }
}
