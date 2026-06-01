import en from './en';
import zhCN from './zh-CN';

const translations = {
    'en': en,
    'zh-CN': zhCN
};

const defaultLanguage = 'zh-CN';
let currentLanguage = defaultLanguage;

export function setLanguage(lang) {
    if (translations[lang]) {
        currentLanguage = lang;
        return;
    }
    currentLanguage = defaultLanguage;
}

export function getLanguage() {
    return currentLanguage;
}

export function t(key) {
    const keys = key.split('.');
    let value = translations[currentLanguage] || translations[defaultLanguage];

    for (const k of keys) {
        if (value && typeof value === 'object') {
            value = value[k];
        } else {
            value = undefined;
            break;
        }
    }

    if (value !== undefined && value !== null) {
        return value;
    }

    value = translations[defaultLanguage];
    for (const k of keys) {
        if (value && typeof value === 'object') {
            value = value[k];
        } else {
            return key;
        }
    }
    return value || key;
}
