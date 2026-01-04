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
let fullConfig = null; // 完整配置
let effectCheckTimer = null; // 效果切换定时检查器
let isCheckingEffect = false; // 防止并发检查

// 效果名称映射
const EFFECT_NAMES = {
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

        // 配置生效（存在、启用、有效果列表）
        if (config && config.enabled && config.effects && config.effects.length > 0) {
            // 保存完整配置
            fullConfig = config;

            // 找到当前有效的效果
            const activeEffect = findActiveEffect(config.effects);

            if (activeEffect) {
                currentConfig = activeEffect;
                showFestivalToggle(activeEffect);

                if (!isManuallyDisabled) {
                    startEffectByType(activeEffect.effect, activeEffect.config);
                }

                window.addEventListener('resize', handleResize);
                document.addEventListener('visibilitychange', handleVisibilityChange);

                // 启动定时检查（用于效果切换）
                startEffectCheckTimer();

                console.log('[Festival] Effects initialized:', activeEffect.effect);
                return;
            }
        }

        // 配置不生效时，检查主题默认效果
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
    if (effect === 'snow') {
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
    if (!Array.isArray(config.effects) || config.effects.length === 0) return false;

    // 验证 cacheDuration（可选）
    if (config.cacheDuration !== undefined && (typeof config.cacheDuration !== 'number' || config.cacheDuration < 1 || config.cacheDuration > 86400)) {
        return false;
    }

    // 验证每个 effect
    for (const item of config.effects) {
        if (!validateEffectItem(item)) return false;
    }

    return true;
}

/**
 * 验证单个效果配置
 */
function validateEffectItem(item) {
    if (typeof item !== 'object' || item === null) return false;

    // 验证 effect 类型
    const validEffects = ['snow', 'firework', 'lantern', 'heart', 'sakura', 'maple', 'summer'];
    if (!validEffects.includes(item.effect)) return false;

    // 验证时间参数（可选）
    if (item.startTime !== undefined && typeof item.startTime !== 'string') return false;
    if (item.endTime !== undefined && typeof item.endTime !== 'string') return false;

    // 验证 cycle（可选）
    if (item.cycle !== undefined && !['daily', 'weekly', 'monthly'].includes(item.cycle)) return false;

    // 验证 config（可选）
    if (item.config && !validateEffectConfig(item.effect, item.config)) return false;

    return true;
}

/**
 * 验证效果参数配置
 */
function validateEffectConfig(effect, config) {
    if (typeof config !== 'object' || config === null) return false;

    // 通用参数验证
    if (config.speed !== undefined && (typeof config.speed !== 'number' || config.speed < 0.1 || config.speed > 5)) {
        return false;
    }
    if (config.wind !== undefined && (typeof config.wind !== 'number' || config.wind < 0 || config.wind > 2)) {
        return false;
    }
    if (config.opacity !== undefined && (typeof config.opacity !== 'number' || config.opacity < 0 || config.opacity > 1)) {
        return false;
    }

    // 根据效果类型验证特定参数
    if (effect === 'snow') {
        if (config.particleCount !== undefined && (typeof config.particleCount !== 'number' || config.particleCount < 1 || config.particleCount > 200)) {
            return false;
        }
    } else if (effect === 'firework') {
        if (config.launchInterval !== undefined && (typeof config.launchInterval !== 'number' || config.launchInterval < 10 || config.launchInterval > 500)) {
            return false;
        }
        if (config.maxFireworks !== undefined && (typeof config.maxFireworks !== 'number' || config.maxFireworks < 1 || config.maxFireworks > 20)) {
            return false;
        }
        if (config.burstChance !== undefined && (typeof config.burstChance !== 'number' || config.burstChance < 0 || config.burstChance > 1)) {
            return false;
        }
    } else if (effect === 'lantern') {
        if (config.lanternCount !== undefined && (typeof config.lanternCount !== 'number' || config.lanternCount < 1 || config.lanternCount > 50)) {
            return false;
        }
        if (config.swingSpeed !== undefined && (typeof config.swingSpeed !== 'number' || config.swingSpeed < 0.1 || config.swingSpeed > 5)) {
            return false;
        }
        if (config.floatSpeed !== undefined && (typeof config.floatSpeed !== 'number' || config.floatSpeed < 0.1 || config.floatSpeed > 5)) {
            return false;
        }
    } else if (effect === 'heart') {
        if (config.heartCount !== undefined && (typeof config.heartCount !== 'number' || config.heartCount < 1 || config.heartCount > 100)) {
            return false;
        }
    } else if (effect === 'sakura') {
        if (config.sakuraCount !== undefined && (typeof config.sakuraCount !== 'number' || config.sakuraCount < 1 || config.sakuraCount > 100)) {
            return false;
        }
    } else if (effect === 'maple') {
        if (config.mapleCount !== undefined && (typeof config.mapleCount !== 'number' || config.mapleCount < 1 || config.mapleCount > 100)) {
            return false;
        }
    } else if (effect === 'summer') {
        if (config.summerCount !== undefined && (typeof config.summerCount !== 'number' || config.summerCount < 1 || config.summerCount > 100)) {
            return false;
        }
    }

    return true;
}

/**
 * 安全地停止当前动画（在启动新效果前调用）
 * 确保不会有多个动画循环同时运行
 */
function ensureAnimationStopped() {
    if (animationId) {
        cancelAnimationFrame(animationId);
        animationId = null;
    }
    isRunning = false;
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
    ensureAnimationStopped();

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
    ensureAnimationStopped();

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
    ensureAnimationStopped();

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
    ensureAnimationStopped();

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
    ensureAnimationStopped();

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
    ensureAnimationStopped();

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
    ensureAnimationStopped();

    const effectConfig = {
        summerCount: config?.summerCount || 10,
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
    fullConfig = null;
    isCheckingEffect = false;

    // 停止效果切换定时检查
    stopEffectCheckTimer();

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
        if (currentConfig.effect === 'snow') {
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
 * 解析时间字符串，支持 "2025-12-01 00:00:00" 格式
 */
function parseTime(str) {
    return new Date(str.replace(' ', 'T'));
}

/**
 * 从 effects 数组中找到当前有效的效果
 */
function findActiveEffect(effects) {
    const now = new Date();

    for (const item of effects) {
        if (isEffectActive(item, now)) {
            return item;
        }
    }
    return null;
}

/**
 * 判断单个效果是否在有效时间内（支持 cycle 周期）
 */
function isEffectActive(item, now) {
    // 没有设置时间范围，直接有效
    if (!item.startTime && !item.endTime) return true;

    const startTime = item.startTime ? parseTime(item.startTime) : null;
    const endTime = item.endTime ? parseTime(item.endTime) : null;

    // 没有设置 cycle，使用原逻辑（startTime 到 endTime 整段时间内有效）
    if (!item.cycle) {
        if (startTime && startTime > now) return false;
        if (endTime && endTime < now) return false;
        return true;
    }

    // 有 cycle 时，检查日期有效期
    if (startTime) {
        const startDate = new Date(startTime.getFullYear(), startTime.getMonth(), startTime.getDate());
        const nowDate = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        if (nowDate < startDate) return false;
    }
    if (endTime) {
        const endDate = new Date(endTime.getFullYear(), endTime.getMonth(), endTime.getDate());
        const nowDate = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        if (nowDate > endDate) return false;
    }

    // 提取时间部分（时:分:秒）
    const timeStart = startTime ? { h: startTime.getHours(), m: startTime.getMinutes(), s: startTime.getSeconds() } : { h: 0, m: 0, s: 0 };
    const timeEnd = endTime ? { h: endTime.getHours(), m: endTime.getMinutes(), s: endTime.getSeconds() } : { h: 23, m: 59, s: 59 };

    // 当前时间（时:分:秒）转换为秒数便于比较
    const nowSeconds = now.getHours() * 3600 + now.getMinutes() * 60 + now.getSeconds();
    const startSeconds = timeStart.h * 3600 + timeStart.m * 60 + timeStart.s;
    const endSeconds = timeEnd.h * 3600 + timeEnd.m * 60 + timeEnd.s;

    // 检查当前时间是否在时间段内
    if (nowSeconds < startSeconds || nowSeconds > endSeconds) return false;

    // 根据 cycle 类型判断
    if (item.cycle === 'daily') {
        // 每天都有效，只要时间匹配即可
        return true;
    } else if (item.cycle === 'weekly') {
        // 每周固定周几有效（从 startTime 推断）
        const targetDayOfWeek = startTime ? startTime.getDay() : 0;
        return now.getDay() === targetDayOfWeek;
    } else if (item.cycle === 'monthly') {
        // 每月固定几号有效（从 startTime 推断）
        const targetDayOfMonth = startTime ? startTime.getDate() : 1;
        return now.getDate() === targetDayOfMonth;
    }

    return true;
}

/**
 * 启动效果切换定时检查
 */
function startEffectCheckTimer() {
    if (effectCheckTimer) clearInterval(effectCheckTimer);
    effectCheckTimer = setInterval(checkAndSwitchEffect, 60000); // 每分钟检查
}

/**
 * 停止效果切换定时检查
 */
function stopEffectCheckTimer() {
    if (effectCheckTimer) {
        clearInterval(effectCheckTimer);
        effectCheckTimer = null;
    }
}

/**
 * 检查并切换效果
 */
async function checkAndSwitchEffect() {
    // 防止并发执行
    if (isCheckingEffect) return;
    isCheckingEffect = true;

    try {
        // 尝试刷新配置（会自动检查缓存是否过期）
        const newConfig = await fetchFestivalConfig();
        if (newConfig) {
            fullConfig = newConfig;
        }

        // 配置无效或禁用时，停止效果
        if (!fullConfig || !fullConfig.enabled) {
            if (isRunning) {
                console.log('[Festival] Config disabled, stopping effect');
                stopFestivalEffect();
                hideFestivalToggle();
                currentConfig = null;
            }
            return;
        }

        if (isManuallyDisabled) return;

        const newActiveEffect = findActiveEffect(fullConfig.effects);

        // 判断效果是否需要切换
        const currentEffect = currentConfig?.effect;
        const newEffect = newActiveEffect?.effect;

        if (newEffect !== currentEffect) {
            console.log('[Festival] Effect switching:', currentEffect, '->', newEffect);

            // 停止当前效果
            stopFestivalEffect();

            if (newActiveEffect) {
                // 启动新效果
                currentConfig = newActiveEffect;
                showFestivalToggle(newActiveEffect);
                startEffectByType(newActiveEffect.effect, newActiveEffect.config);
            } else {
                // 没有有效效果
                hideFestivalToggle();
                currentConfig = null;
            }
        }
    } finally {
        isCheckingEffect = false;
    }
}
