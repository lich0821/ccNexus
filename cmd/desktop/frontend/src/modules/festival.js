/**
 * Festival Effects Module - 节日氛围效果模块
 *
 * 通过远程配置文件控制节日效果的开关和参数
 * 支持效果：雪花(christmas)、烟花(firework)
 * 支持10种烟花造型效果
 */

import { Snowflake, createSnowParticles } from '../effects/snow.js';
import { Firework, createFireworks } from '../effects/firework.js';

// 配置
const FESTIVAL_CONFIG_URL = 'https://gitee.com/hea7en/images/raw/master/group/festival.json';
const CONFIG_CACHE_KEY = 'festival_cache';
const CONFIG_CACHE_TIME_KEY = 'festival_cache_time';

// 全局状态
let canvas = null;
let ctx = null;
let particles = [];
let fireworks = [];
let fireworkTimer = 0;
let fireworkConfig = null;
let animationId = null;
let isRunning = false;
let currentConfig = null;
let isManuallyDisabled = false; // 用户手动关闭的状态

// 效果名称映射
const EFFECT_NAMES = {
    'christmas': '飘雪',
    'firework': '烟花'
};

/**
 * 初始化节日效果
 */
export async function initFestivalEffects() {
    try {
        const config = await fetchFestivalConfig();

        if (!config || !config.enabled) {
            console.log('[Festival] Effects disabled or config not available');
            hideFestivalToggle();
            return;
        }

        // 检查时间范围
        if (!isWithinTimeRange(config)) {
            console.log('[Festival] Effects not in valid time range');
            hideFestivalToggle();
            return;
        }

        currentConfig = config;

        // 显示开关控件
        showFestivalToggle(config);

        // 如果用户没有手动关闭，则启动效果
        if (!isManuallyDisabled) {
            if (config.effect === 'christmas') {
                startSnowEffect(config.config);
            } else if (config.effect === 'firework') {
                startFireworkEffect(config.config);
            }
        }

        window.addEventListener('resize', handleResize);
        document.addEventListener('visibilitychange', handleVisibilityChange);

        console.log('[Festival] Effects initialized:', config.effect);
    } catch (error) {
        console.error('[Festival] Failed to initialize:', error);
        hideFestivalToggle();
    }
}

/**
 * 获取节日配置
 */
async function fetchFestivalConfig() {
    try {
        const cachedConfig = localStorage.getItem(CONFIG_CACHE_KEY);
        const cachedTime = localStorage.getItem(CONFIG_CACHE_TIME_KEY);

        if (cachedConfig && cachedTime) {
            const config = JSON.parse(cachedConfig);
            const cacheDuration = (config.cacheDuration || 3600) * 1000;
            const elapsed = Date.now() - parseInt(cachedTime);
            if (elapsed < cacheDuration) {
                return config;
            }
        }

        const url = FESTIVAL_CONFIG_URL + '?t=' + Date.now();
        const json = await window.go.main.App.FetchBroadcast(url);

        if (!json) {
            throw new Error('Empty response');
        }

        const config = JSON.parse(json);

        if (!validateConfig(config)) {
            throw new Error('Invalid config format');
        }

        localStorage.setItem(CONFIG_CACHE_KEY, JSON.stringify(config));
        localStorage.setItem(CONFIG_CACHE_TIME_KEY, Date.now().toString());

        return config;
    } catch (error) {
        console.warn('[Festival] Failed to fetch config:', error.message);

        const cachedConfig = localStorage.getItem(CONFIG_CACHE_KEY);
        if (cachedConfig) {
            return JSON.parse(cachedConfig);
        }

        return null;
    }
}

/**
 * 验证配置格式
 */
function validateConfig(config) {
    if (typeof config !== 'object' || config === null) return false;
    if (typeof config.enabled !== 'boolean') return false;
    if (typeof config.effect !== 'string') return false;

    // 验证时间参数（可选）
    if (config.startTime !== undefined && typeof config.startTime !== 'string') {
        return false;
    }
    if (config.endTime !== undefined && typeof config.endTime !== 'string') {
        return false;
    }

    if (config.cacheDuration !== undefined && (typeof config.cacheDuration !== 'number' || config.cacheDuration < 1 || config.cacheDuration > 86400)) {
        return false;
    }

    if (config.config) {
        const c = config.config;
        if (c.particleCount !== undefined && (typeof c.particleCount !== 'number' || c.particleCount < 1 || c.particleCount > 200)) {
            return false;
        }
        if (c.speed !== undefined && (typeof c.speed !== 'number' || c.speed < 0.1 || c.speed > 5)) {
            return false;
        }
        if (c.wind !== undefined && (typeof c.wind !== 'number' || c.wind < 0 || c.wind > 2)) {
            return false;
        }
        if (c.opacity !== undefined && (typeof c.opacity !== 'number' || c.opacity < 0 || c.opacity > 1)) {
            return false;
        }
    }

    return true;
}

/**
 * 创建 Canvas
 */
function createCanvas() {
    if (canvas) return;

    canvas = document.createElement('canvas');
    canvas.id = 'festival-canvas';
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;

    document.body.appendChild(canvas);
    ctx = canvas.getContext('2d');
}

/**
 * 销毁 Canvas
 */
function destroyCanvas() {
    if (canvas) {
        canvas.remove();
        canvas = null;
        ctx = null;
    }
}

/**
 * 启动飘雪效果
 */
function startSnowEffect(config) {
    const effectConfig = {
        particleCount: config?.particleCount || 50,
        speed: config?.speed || 1.0,
        wind: config?.wind || 0.5,
        opacity: config?.opacity || 0.8
    };

    createCanvas();
    particles = createSnowParticles(effectConfig, canvas);

    isRunning = true;
    animate();

    // 更新开关状态
    updateToggleState();
}

/**
 * 启动烟花效果
 */
function startFireworkEffect(config) {
    fireworkConfig = {
        launchInterval: config?.launchInterval || 130,
        maxFireworks: config?.maxFireworks || 3,
        burstChance: config?.burstChance || 0.25
    };

    createCanvas();
    fireworks = [];
    fireworkTimer = 0;

    for (let i = 0; i < 2; i++) {
        setTimeout(() => {
            if (isRunning) {
                fireworks.push(new Firework(fireworkConfig, canvas));
            }
        }, i * 500);
    }

    isRunning = true;
    animateFireworks();

    // 更新开关状态
    updateToggleState();
}

/**
 * 动画循环（雪花）
 */
function animate() {
    if (!isRunning || !ctx || !canvas) return;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    for (const particle of particles) {
        particle.update();
        particle.draw(ctx);
    }

    animationId = requestAnimationFrame(animate);
}

/**
 * 烟花动画循环
 */
function animateFireworks() {
    if (!isRunning || !ctx || !canvas) return;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    fireworkTimer++;

    if (fireworkTimer >= fireworkConfig.launchInterval && fireworks.length < fireworkConfig.maxFireworks) {
        fireworkTimer = 0;
        fireworks.push(new Firework(fireworkConfig, canvas));

        if (Math.random() < fireworkConfig.burstChance) {
            setTimeout(() => {
                if (isRunning && fireworks.length < fireworkConfig.maxFireworks) {
                    fireworks.push(new Firework(fireworkConfig, canvas));
                }
            }, Math.random() * 200 + 100);
        }
    }

    fireworks = fireworks.filter(firework => {
        const alive = firework.update();
        firework.draw(ctx);
        return alive;
    });

    animationId = requestAnimationFrame(animateFireworks);
}

/**
 * 暂停动画
 */
function pauseAnimation() {
    isRunning = false;
    if (animationId) {
        cancelAnimationFrame(animationId);
        animationId = null;
    }
}

/**
 * 恢复动画
 */
function resumeAnimation() {
    if (!isRunning && currentConfig) {
        isRunning = true;
        if (currentConfig.effect === 'firework') {
            animateFireworks();
        } else if (particles.length > 0) {
            animate();
        }
    }
}

/**
 * 处理窗口大小变化
 */
function handleResize() {
    if (canvas) {
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
    }
}

/**
 * 处理页面可见性变化
 */
function handleVisibilityChange() {
    if (document.hidden) {
        pauseAnimation();
    } else {
        resumeAnimation();
    }
}

/**
 * 销毁节日效果
 */
export function destroyFestivalEffects() {
    pauseAnimation();
    destroyCanvas();
    particles = [];
    fireworks = [];
    fireworkTimer = 0;
    fireworkConfig = null;
    currentConfig = null;

    window.removeEventListener('resize', handleResize);
    document.removeEventListener('visibilitychange', handleVisibilityChange);

    console.log('[Festival] Effects destroyed');
}

/**
 * 获取当前效果状态
 */
export function getFestivalEffectState() {
    return {
        isRunning,
        config: currentConfig,
        particleCount: particles.length
    };
}

/**
 * 清除配置缓存
 */
export function clearFestivalConfigCache() {
    localStorage.removeItem(CONFIG_CACHE_KEY);
    localStorage.removeItem(CONFIG_CACHE_TIME_KEY);
    console.log('[Festival] Config cache cleared');
}

/**
 * 显示节日效果开关控件
 */
function showFestivalToggle(config) {
    const toggle = document.getElementById('festivalToggle');
    const nameSpan = document.getElementById('festivalToggleName');
    const switchSpan = document.getElementById('festivalToggleSwitch');

    if (!toggle || !nameSpan || !switchSpan) return;

    // 设置效果名称
    const effectName = EFFECT_NAMES[config.effect] || config.effect;
    nameSpan.textContent = effectName;

    // 显示控件（初始状态会由启动效果后自动更新）
    toggle.classList.remove('hidden');
}

/**
 * 隐藏节日效果开关控件
 */
function hideFestivalToggle() {
    const toggle = document.getElementById('festivalToggle');
    if (toggle) {
        toggle.classList.add('hidden');
    }
}

/**
 * 更新开关状态显示
 */
function updateToggleState() {
    const toggle = document.getElementById('festivalToggle');
    const switchSpan = document.getElementById('festivalToggleSwitch');

    if (!toggle || !switchSpan) return;

    if (isRunning && !isManuallyDisabled) {
        toggle.classList.add('active');
        toggle.classList.remove('inactive');
        switchSpan.textContent = 'ON';
    } else {
        toggle.classList.remove('active');
        toggle.classList.add('inactive');
        switchSpan.textContent = 'OFF';
    }
}

/**
 * 切换节日效果开关
 */
export function toggleFestivalEffect() {
    if (!currentConfig) return;

    if (isRunning) {
        // 关闭效果
        isManuallyDisabled = true;
        stopFestivalEffect();
    } else {
        // 开启效果
        isManuallyDisabled = false;
        if (currentConfig.effect === 'christmas') {
            startSnowEffect(currentConfig.config);
        } else if (currentConfig.effect === 'firework') {
            startFireworkEffect(currentConfig.config);
        }
    }

    console.log('[Festival] Effect toggled:', isRunning ? 'ON' : 'OFF');
}

/**
 * 停止节日效果（不销毁配置）
 */
function stopFestivalEffect() {
    pauseAnimation();
    destroyCanvas();
    particles = [];
    fireworks = [];
    fireworkTimer = 0;

    // 更新开关状态
    updateToggleState();
}

// 暴露到 window 对象供 UI 调用
window.toggleFestivalEffect = toggleFestivalEffect;

/**
 * 检查配置是否在有效时间范围内
 */
function isWithinTimeRange(config) {
    const now = new Date();

    // 检查开始时间
    if (config.startTime) {
        const startTime = parseTime(config.startTime);
        if (startTime > now) {
            return false;
        }
    }

    // 检查结束时间
    if (config.endTime) {
        const endTime = parseTime(config.endTime);
        if (endTime < now) {
            return false;
        }
    }

    return true;
}

/**
 * 解析时间字符串，支持 "2025-12-01 00:00:00" 格式
 */
function parseTime(str) {
    return new Date(str.replace(' ', 'T'));
}
