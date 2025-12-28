/**
 * Festival Effects Module - 节日氛围效果模块
 *
 * 通过远程配置文件控制节日效果的开关和参数
 * 支持效果：雪花(snow)、烟花(firework)、灯笼(lantern)、爱心(heart)、樱花(sakura)、枫叶(maple)、夏天(summer)
 * 支持10种烟花造型效果
 */

import { Snowflake, createSnowParticles } from '../effects/snow.js';
import { Firework, createFireworks } from '../effects/firework.js';
import { Lantern, createLanterns } from '../effects/lantern.js';
import { Heart, createHearts } from '../effects/heart.js';
import { Sakura, createSakuras } from '../effects/sakura.js';
import { Maple, createMaples } from '../effects/maple.js';
import { SummerElement, createSummerElements } from '../effects/summer.js';

// 配置
const FESTIVAL_CONFIG_URL = 'https://gitee.com/hea7en/images/raw/master/group/festival.json';
const CONFIG_CACHE_KEY = 'festival_cache';
const CONFIG_CACHE_TIME_KEY = 'festival_cache_time';

// 全局状态
let canvas = null;
let ctx = null;
let particles = [];
let fireworks = [];
let lanterns = [];
let hearts = [];
let sakuras = [];
let maples = [];
let summers = [];
let fireworkTimer = 0;
let fireworkConfig = null;
let animationId = null;
let isRunning = false;
let currentConfig = null;
let isManuallyDisabled = false; // 用户手动关闭的状态

// 效果名称映射
const EFFECT_NAMES = {
    'christmas': '飘雪',
    'snow': '飘雪',
    'firework': '烟花',
    'lantern': '灯笼',
    'heart': '爱心',
    'sakura': '樱花',
    'maple': '枫叶',
    'summer': '夏天'
};

/**
 * 初始化节日效果
 */
export async function initFestivalEffects() {
    try {
        const config = await fetchFestivalConfig();

        // 配置生效（存在、启用、时间范围内），优先使用配置的效果
        if (config && config.enabled && isWithinTimeRange(config)) {
            currentConfig = config;
            showFestivalToggle(config);

            if (!isManuallyDisabled) {
                startEffectByType(config.effect, config.config);
            }

            window.addEventListener('resize', handleResize);
            document.addEventListener('visibilitychange', handleVisibilityChange);

            console.log('[Festival] Effects initialized:', config.effect);
            return;
        }

        // 配置不生效（不存在、未启用、时间范围外）时，检查主题默认效果
        if (document.body.classList.contains('sakura-theme')) {
            currentConfig = {
                enabled: true,
                effect: 'sakura',
                config: {}
            };
            showFestivalToggle(currentConfig);

            if (!isManuallyDisabled) {
                startSakuraEffect(currentConfig.config);
            }

            window.addEventListener('resize', handleResize);
            document.addEventListener('visibilitychange', handleVisibilityChange);

            console.log('[Festival] Sakura theme detected, using sakura effect');
            return;
        }

        if (document.body.classList.contains('ocean-theme')) {
            currentConfig = {
                enabled: true,
                effect: 'summer',
                config: {}
            };
            showFestivalToggle(currentConfig);

            if (!isManuallyDisabled) {
                startSummerEffect(currentConfig.config);
            }

            window.addEventListener('resize', handleResize);
            document.addEventListener('visibilitychange', handleVisibilityChange);

            console.log('[Festival] Ocean theme detected, using summer effect');
            return;
        }

        console.log('[Festival] Effects disabled or config not available');
        hideFestivalToggle();
    } catch (error) {
        console.error('[Festival] Failed to initialize:', error);
        hideFestivalToggle();
    }
}

/**
 * 根据效果类型启动对应效果
 */
function startEffectByType(effect, config) {
    if (effect === 'christmas' || effect === 'snow') {
        startSnowEffect(config);
    } else if (effect === 'firework') {
        startFireworkEffect(config);
    } else if (effect === 'lantern') {
        startLanternEffect(config);
    } else if (effect === 'heart') {
        startHeartEffect(config);
    } else if (effect === 'sakura') {
        startSakuraEffect(config);
    } else if (effect === 'maple') {
        startMapleEffect(config);
    } else if (effect === 'summer') {
        startSummerEffect(config);
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
 * 启动灯笼效果
 */
function startLanternEffect(config) {
    const effectConfig = {
        lanternCount: config?.lanternCount || 12,
        swingSpeed: config?.swingSpeed || 1.0,
        floatSpeed: config?.floatSpeed || 0.5,
        opacity: config?.opacity || 0.85
    };

    createCanvas();
    lanterns = createLanterns(effectConfig, canvas);

    isRunning = true;
    animateLanterns();

    // 更新开关状态
    updateToggleState();
}

/**
 * 启动爱心效果
 */
function startHeartEffect(config) {
    const effectConfig = {
        heartCount: config?.heartCount || 15,
        speed: config?.speed || 1.0,
        wind: config?.wind || 0.25,
        opacity: config?.opacity || 0.85
    };

    createCanvas();
    hearts = createHearts(effectConfig, canvas);

    isRunning = true;
    animateHearts();

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
 * 灯笼动画循环
 */
function animateLanterns() {
    if (!isRunning || !ctx || !canvas) return;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    for (const lantern of lanterns) {
        lantern.update();
        lantern.draw(ctx);
    }

    animationId = requestAnimationFrame(animateLanterns);
}

/**
 * 爱心动画循环
 */
function animateHearts() {
    if (!isRunning || !ctx || !canvas) return;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    for (const heart of hearts) {
        heart.update();
        heart.draw(ctx);
    }

    animationId = requestAnimationFrame(animateHearts);
}

/**
 * 启动樱花效果
 */
function startSakuraEffect(config) {
    const effectConfig = {
        sakuraCount: config?.sakuraCount || 20,
        speed: config?.speed || 1.0,
        wind: config?.wind || 0.3,
        opacity: config?.opacity || 0.85
    };

    createCanvas();
    sakuras = createSakuras(effectConfig, canvas);

    isRunning = true;
    animateSakuras();

    updateToggleState();
}

/**
 * 樱花动画循环
 */
function animateSakuras() {
    if (!isRunning || !ctx || !canvas) return;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    for (const sakura of sakuras) {
        sakura.update();
        sakura.draw(ctx);
    }

    animationId = requestAnimationFrame(animateSakuras);
}

/**
 * 启动枫叶效果
 */
function startMapleEffect(config) {
    const effectConfig = {
        mapleCount: config?.mapleCount || 10,
        speed: config?.speed || 1.0,
        wind: config?.wind || 0.4,
        opacity: config?.opacity || 0.85
    };

    createCanvas();
    maples = createMaples(effectConfig, canvas);

    isRunning = true;
    animateMaples();

    updateToggleState();
}

/**
 * 枫叶动画循环
 */
function animateMaples() {
    if (!isRunning || !ctx || !canvas) return;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    for (const maple of maples) {
        maple.update();
        maple.draw(ctx);
    }

    animationId = requestAnimationFrame(animateMaples);
}

/**
 * 启动夏天效果
 */
function startSummerEffect(config) {
    const effectConfig = {
        summerCount: config?.summerCount || 12,
        speed: config?.speed || 1.0,
        wind: config?.wind || 0.3,
        opacity: config?.opacity || 0.85
    };

    createCanvas();
    summers = createSummerElements(effectConfig, canvas);

    isRunning = true;
    animateSummer();

    updateToggleState();
}

/**
 * 夏天动画循环
 */
function animateSummer() {
    if (!isRunning || !ctx || !canvas) return;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    for (const element of summers) {
        element.update();
        element.draw(ctx);
    }

    animationId = requestAnimationFrame(animateSummer);
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
        } else if (currentConfig.effect === 'lantern') {
            animateLanterns();
        } else if (currentConfig.effect === 'heart') {
            animateHearts();
        } else if (currentConfig.effect === 'sakura') {
            animateSakuras();
        } else if (currentConfig.effect === 'maple') {
            animateMaples();
        } else if (currentConfig.effect === 'summer') {
            animateSummer();
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
    lanterns = [];
    hearts = [];
    sakuras = [];
    maples = [];
    summers = [];
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
        if (currentConfig.effect === 'christmas' || currentConfig.effect === 'snow') {
            startSnowEffect(currentConfig.config);
        } else if (currentConfig.effect === 'firework') {
            startFireworkEffect(currentConfig.config);
        } else if (currentConfig.effect === 'lantern') {
            startLanternEffect(currentConfig.config);
        } else if (currentConfig.effect === 'heart') {
            startHeartEffect(currentConfig.config);
        } else if (currentConfig.effect === 'sakura') {
            startSakuraEffect(currentConfig.config);
        } else if (currentConfig.effect === 'maple') {
            startMapleEffect(currentConfig.config);
        } else if (currentConfig.effect === 'summer') {
            startSummerEffect(currentConfig.config);
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
    lanterns = [];
    hearts = [];
    sakuras = [];
    maples = [];
    summers = [];
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
