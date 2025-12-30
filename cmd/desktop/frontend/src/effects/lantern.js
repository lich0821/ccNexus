/**
 * Lantern Effect - 春节氛围效果模块
 * 包含：灯笼、红包、剪纸福字、鞭炮、中国结、倒福字
 */

export class Lantern {
    constructor(config, canvas) {
        this.canvas = canvas;
        this.config = config;
        this.reset();
    }

    reset() {
        const w = this.canvas?.width || window.innerWidth;
        const h = this.canvas?.height || window.innerHeight;

        this.x = Math.random() * w;
        this.y = Math.random() * -100;

        // 固定基础尺寸
        this.size = 14;

        // 颜色
        this.color = this.getRandomColor();

        // 下落
        const speedFactor = 0.35 + (this.size / 50);
        this.speedY = (Math.random() * 0.4 + 0.3) * (this.config.speed || 1.0) * speedFactor;
        this.speedX = (Math.random() - 0.5) * (this.config.wind || 0.25) * 1.5;

        // 摇晃
        this.swingAngle = Math.random() * Math.PI * 2;
        this.swingSpeed = Math.random() * 0.01 + 0.005;
        this.swingRadius = Math.random() * 0.8 + 0.3;

        // 透明度
        this.opacity = (Math.random() * 0.2 + 0.5) * (this.config.opacity || 0.75);

        // 灯光闪烁
        this.glowPhase = Math.random() * Math.PI * 2;
        this.glowSpeed = Math.random() * 0.12 + 0.06;
        this.glowMin = 0.2;
        this.glowMax = 1.0;
    }

    getRandomColor() {
        const colors = [
            { body: '#E53935', light: '#FFCDD2', glow: '#FF5252' },
            { body: '#C62828', light: '#EF9A9A', glow: '#EF5350' },
            { body: '#FF6F00', light: '#FFE0B2', glow: '#FFB74D' },
            { body: '#F9A825', light: '#FFF9C4', glow: '#FFEE58' },
        ];
        const weights = [45, 30, 15, 10];
        let r = Math.random() * 100;
        for (let i = 0; i < colors.length; i++) {
            r -= weights[i];
            if (r <= 0) return colors[i];
        }
        return colors[0];
    }

    update() {
        this.swingAngle += this.swingSpeed;
        this.y += this.speedY;
        this.x += this.speedX + Math.sin(this.swingAngle) * this.swingRadius;
        this.glowPhase += this.glowSpeed;

        const h = this.canvas?.height || window.innerHeight;
        const w = this.canvas?.width || window.innerWidth;

        if (this.y > h + 40) {
            this.y = -40;
            this.x = Math.random() * w;
        }
        if (this.x > w + 25) this.x = -25;
        else if (this.x < -25) this.x = w + 25;
    }

    draw(ctx) {
        if (!ctx) return;

        ctx.save();
        ctx.globalAlpha = this.opacity;
        ctx.translate(this.x, this.y);
        ctx.rotate(Math.sin(this.swingAngle) * 0.04);

        const s = this.size;
        const glow = this.glowMin + (Math.sin(this.glowPhase) * 0.5 + 0.5) * (this.glowMax - this.glowMin);

        // 根据类型绘制不同造型
        if (this.type === 0) {
            this.drawRoundLantern(ctx, s, glow);
        } else if (this.type === 1) {
            this.drawGourdLantern(ctx, s, glow);
        } else if (this.type === 2) {
            this.drawOvalLantern(ctx, s, glow);
        } else if (this.type === 3) {
            this.drawPumpkinLantern(ctx, s, glow);
        } else if (this.type === 4) {
            this.drawDiamondLantern(ctx, s, glow);
        } else if (this.type === 5) {
            this.drawRedEnvelope(ctx, s, glow);
        } else if (this.type === 6) {
            this.drawPaperCutFu(ctx, s, glow);
        } else if (this.type === 7) {
            this.drawFirecracker(ctx, s, glow);
        } else if (this.type === 8) {
            this.drawChineseKnot(ctx, s, glow);
        } else if (this.type === 9) {
            this.drawFuCharacter(ctx, s, glow);
        }

        ctx.restore();
    }

    // 圆形灯笼
    drawRoundLantern(ctx, s, glow) {
        s = s * 1.4;  // 放大
        // 外层光晕
        const outerGlow = ctx.createRadialGradient(0, 0, 0, 0, 0, s * 1.6);
        outerGlow.addColorStop(0, `rgba(255, 200, 100, ${0.3 * glow})`);
        outerGlow.addColorStop(1, 'rgba(255, 100, 50, 0)');
        ctx.fillStyle = outerGlow;
        ctx.beginPath();
        ctx.arc(0, 0, s * 1.6, 0, Math.PI * 2);
        ctx.fill();

        // 挂绳
        ctx.strokeStyle = '#6D4C41';
        ctx.lineWidth = 0.8;
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.7);
        ctx.lineTo(0, -s * 0.45);
        ctx.stroke();

        // 顶部装饰
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.arc(0, -s * 0.45, s * 0.12, 0, Math.PI * 2);
        ctx.fill();

        // 圆形主体
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.4, 0, Math.PI * 2);
        const bodyGrad = ctx.createRadialGradient(s * 0.1, -s * 0.1, 0, 0, 0, s * 0.4);
        bodyGrad.addColorStop(0, this.color.light);
        bodyGrad.addColorStop(0.5, this.color.body);
        bodyGrad.addColorStop(1, '#8B0000');
        ctx.fillStyle = bodyGrad;
        ctx.fill();

        // 内部灯光
        const innerGlow = ctx.createRadialGradient(0, 0, 0, 0, 0, s * 0.35);
        innerGlow.addColorStop(0, `rgba(255, 255, 230, ${1.0 * glow})`);
        innerGlow.addColorStop(0.3, `rgba(255, 240, 180, ${0.7 * glow})`);
        innerGlow.addColorStop(1, 'rgba(255, 180, 100, 0)');
        ctx.fillStyle = innerGlow;
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.32, 0, Math.PI * 2);
        ctx.fill();

        // 高光
        ctx.fillStyle = `rgba(255, 255, 255, ${0.4 * glow})`;
        ctx.beginPath();
        ctx.arc(s * 0.12, -s * 0.12, s * 0.08, 0, Math.PI * 2);
        ctx.fill();

        // 底部装饰
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.arc(0, s * 0.45, s * 0.1, 0, Math.PI * 2);
        ctx.fill();

        // 流苏
        this.drawTassel(ctx, s * 0.5, s, glow);
    }

    // 葫芦形灯笼
    drawGourdLantern(ctx, s, glow) {
        s = s * 1.4;  // 放大

        // 外层光晕
        const outerGlow = ctx.createRadialGradient(0, 0, 0, 0, 0, s * 1.6);
        outerGlow.addColorStop(0, `rgba(255, 200, 100, ${0.28 * glow})`);
        outerGlow.addColorStop(1, 'rgba(255, 100, 50, 0)');
        ctx.fillStyle = outerGlow;
        ctx.beginPath();
        ctx.arc(0, 0, s * 1.6, 0, Math.PI * 2);
        ctx.fill();

        // 挂绳
        ctx.strokeStyle = '#6D4C41';
        ctx.lineWidth = 0.8;
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.85);
        ctx.lineTo(0, -s * 0.55);
        ctx.stroke();

        // 顶部装饰
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.arc(0, -s * 0.55, s * 0.1, 0, Math.PI * 2);
        ctx.fill();

        // 上半部分（小圆）
        ctx.beginPath();
        ctx.arc(0, -s * 0.28, s * 0.22, 0, Math.PI * 2);
        const topGrad = ctx.createRadialGradient(s * 0.05, -s * 0.32, 0, 0, -s * 0.28, s * 0.22);
        topGrad.addColorStop(0, this.color.light);
        topGrad.addColorStop(0.5, this.color.body);
        topGrad.addColorStop(1, '#8B0000');
        ctx.fillStyle = topGrad;
        ctx.fill();

        // 下半部分（大圆）
        ctx.beginPath();
        ctx.arc(0, s * 0.18, s * 0.35, 0, Math.PI * 2);
        const bottomGrad = ctx.createRadialGradient(s * 0.08, s * 0.1, 0, 0, s * 0.18, s * 0.35);
        bottomGrad.addColorStop(0, this.color.light);
        bottomGrad.addColorStop(0.5, this.color.body);
        bottomGrad.addColorStop(1, '#8B0000');
        ctx.fillStyle = bottomGrad;
        ctx.fill();

        // 内部灯光 - 上
        const innerGlowTop = ctx.createRadialGradient(0, -s * 0.28, 0, 0, -s * 0.28, s * 0.18);
        innerGlowTop.addColorStop(0, `rgba(255, 255, 230, ${0.9 * glow})`);
        innerGlowTop.addColorStop(1, 'rgba(255, 180, 100, 0)');
        ctx.fillStyle = innerGlowTop;
        ctx.beginPath();
        ctx.arc(0, -s * 0.28, s * 0.18, 0, Math.PI * 2);
        ctx.fill();

        // 内部灯光 - 下
        const innerGlowBottom = ctx.createRadialGradient(0, s * 0.18, 0, 0, s * 0.18, s * 0.3);
        innerGlowBottom.addColorStop(0, `rgba(255, 255, 230, ${1.0 * glow})`);
        innerGlowBottom.addColorStop(0.4, `rgba(255, 240, 180, ${0.6 * glow})`);
        innerGlowBottom.addColorStop(1, 'rgba(255, 180, 100, 0)');
        ctx.fillStyle = innerGlowBottom;
        ctx.beginPath();
        ctx.arc(0, s * 0.18, s * 0.28, 0, Math.PI * 2);
        ctx.fill();

        // 高光
        ctx.fillStyle = `rgba(255, 255, 255, ${0.35 * glow})`;
        ctx.beginPath();
        ctx.arc(s * 0.1, s * 0.08, s * 0.08, 0, Math.PI * 2);
        ctx.fill();

        // 底部装饰
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.arc(0, s * 0.55, s * 0.08, 0, Math.PI * 2);
        ctx.fill();

        // 流苏
        this.drawTassel(ctx, s * 0.58, s, glow);
    }

    // 椭圆形灯笼
    drawOvalLantern(ctx, s, glow) {
        s = s * 1.3;  // 放大
        // 外层光晕
        const outerGlow = ctx.createRadialGradient(0, 0, 0, 0, 0, s * 1.8);
        outerGlow.addColorStop(0, `rgba(255, 200, 100, ${0.25 * glow})`);
        outerGlow.addColorStop(0.5, `rgba(255, 150, 50, ${0.1 * glow})`);
        outerGlow.addColorStop(1, 'rgba(255, 100, 50, 0)');
        ctx.fillStyle = outerGlow;
        ctx.beginPath();
        ctx.arc(0, 0, s * 1.8, 0, Math.PI * 2);
        ctx.fill();

        // 挂绳
        ctx.strokeStyle = '#6D4C41';
        ctx.lineWidth = 0.8;
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.9);
        ctx.lineTo(0, -s * 0.55);
        ctx.stroke();

        // 顶部金色装饰
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.rect(-s * 0.22, -s * 0.58, s * 0.44, s * 0.12);
        ctx.fill();

        // 椭圆形主体
        ctx.beginPath();
        ctx.ellipse(0, 0, s * 0.4, s * 0.5, 0, 0, Math.PI * 2);
        const bodyGrad = ctx.createRadialGradient(s * 0.1, -s * 0.1, 0, 0, 0, s * 0.5);
        bodyGrad.addColorStop(0, this.color.light);
        bodyGrad.addColorStop(0.5, this.color.body);
        bodyGrad.addColorStop(1, '#8B0000');
        ctx.fillStyle = bodyGrad;
        ctx.fill();

        // 内部灯光
        const innerGlow = ctx.createRadialGradient(0, 0, 0, 0, 0, s * 0.42);
        innerGlow.addColorStop(0, `rgba(255, 255, 230, ${1.0 * glow})`);
        innerGlow.addColorStop(0.25, `rgba(255, 240, 180, ${0.8 * glow})`);
        innerGlow.addColorStop(0.5, `rgba(255, 220, 120, ${0.5 * glow})`);
        innerGlow.addColorStop(1, 'rgba(255, 180, 100, 0)');
        ctx.fillStyle = innerGlow;
        ctx.beginPath();
        ctx.ellipse(0, 0, s * 0.38, s * 0.48, 0, 0, Math.PI * 2);
        ctx.fill();

        // 横向装饰线
        ctx.strokeStyle = 'rgba(139, 0, 0, 0.4)';
        ctx.lineWidth = 0.5;
        ctx.beginPath();
        ctx.ellipse(0, -s * 0.2, s * 0.35, s * 0.08, 0, 0, Math.PI * 2);
        ctx.stroke();
        ctx.beginPath();
        ctx.ellipse(0, s * 0.2, s * 0.35, s * 0.08, 0, 0, Math.PI * 2);
        ctx.stroke();

        // 高光点
        ctx.fillStyle = `rgba(255, 255, 255, ${0.4 * glow})`;
        ctx.beginPath();
        ctx.arc(s * 0.15, -s * 0.2, s * 0.08, 0, Math.PI * 2);
        ctx.fill();

        // 底部金色装饰
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.rect(-s * 0.22, s * 0.46, s * 0.44, s * 0.1);
        ctx.fill();

        // 流苏
        this.drawTassel(ctx, s * 0.56, s, glow);
    }

    // 南瓜形灯笼
    drawPumpkinLantern(ctx, s, glow) {
        s = s * 1.5;  // 放大

        // 外层光晕
        const outerGlow = ctx.createRadialGradient(0, 0, 0, 0, 0, s * 1.7);
        outerGlow.addColorStop(0, `rgba(255, 200, 100, ${0.28 * glow})`);
        outerGlow.addColorStop(1, 'rgba(255, 100, 50, 0)');
        ctx.fillStyle = outerGlow;
        ctx.beginPath();
        ctx.arc(0, 0, s * 1.7, 0, Math.PI * 2);
        ctx.fill();

        // 挂绳
        ctx.strokeStyle = '#6D4C41';
        ctx.lineWidth = 0.8;
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.65);
        ctx.lineTo(0, -s * 0.4);
        ctx.stroke();

        // 顶部装饰
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.arc(0, -s * 0.4, s * 0.1, 0, Math.PI * 2);
        ctx.fill();

        // 南瓜形主体 - 多瓣结构
        const petalCount = 5;
        for (let i = 0; i < petalCount; i++) {
            const angle = (i / petalCount) * Math.PI * 2 - Math.PI / 2;
            const px = Math.cos(angle) * s * 0.15;

            ctx.beginPath();
            ctx.ellipse(px, 0, s * 0.28, s * 0.38, 0, 0, Math.PI * 2);

            const petalGrad = ctx.createRadialGradient(px + s * 0.08, -s * 0.08, 0, px, 0, s * 0.3);
            petalGrad.addColorStop(0, this.color.light);
            petalGrad.addColorStop(0.5, this.color.body);
            petalGrad.addColorStop(1, '#8B0000');
            ctx.fillStyle = petalGrad;
            ctx.fill();
        }

        // 内部灯光
        const innerGlow = ctx.createRadialGradient(0, 0, 0, 0, 0, s * 0.35);
        innerGlow.addColorStop(0, `rgba(255, 255, 230, ${1.0 * glow})`);
        innerGlow.addColorStop(0.3, `rgba(255, 240, 180, ${0.7 * glow})`);
        innerGlow.addColorStop(1, 'rgba(255, 180, 100, 0)');
        ctx.fillStyle = innerGlow;
        ctx.beginPath();
        ctx.ellipse(0, 0, s * 0.32, s * 0.36, 0, 0, Math.PI * 2);
        ctx.fill();

        // 高光
        ctx.fillStyle = `rgba(255, 255, 255, ${0.4 * glow})`;
        ctx.beginPath();
        ctx.arc(s * 0.1, -s * 0.12, s * 0.08, 0, Math.PI * 2);
        ctx.fill();

        // 底部装饰
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.arc(0, s * 0.42, s * 0.08, 0, Math.PI * 2);
        ctx.fill();

        // 流苏
        this.drawTassel(ctx, s * 0.46, s, glow);
    }

    // 菱形灯笼
    drawDiamondLantern(ctx, s, glow) {
        s = s * 1.5;  // 放大

        // 外层光晕
        const outerGlow = ctx.createRadialGradient(0, 0, 0, 0, 0, s * 1.6);
        outerGlow.addColorStop(0, `rgba(255, 200, 100, ${0.3 * glow})`);
        outerGlow.addColorStop(1, 'rgba(255, 100, 50, 0)');
        ctx.fillStyle = outerGlow;
        ctx.beginPath();
        ctx.arc(0, 0, s * 1.6, 0, Math.PI * 2);
        ctx.fill();

        // 挂绳
        ctx.strokeStyle = '#6D4C41';
        ctx.lineWidth = 0.8;
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.75);
        ctx.lineTo(0, -s * 0.5);
        ctx.stroke();

        // 顶部装饰
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.arc(0, -s * 0.5, s * 0.08, 0, Math.PI * 2);
        ctx.fill();

        // 菱形主体
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.45);
        ctx.lineTo(s * 0.35, 0);
        ctx.lineTo(0, s * 0.45);
        ctx.lineTo(-s * 0.35, 0);
        ctx.closePath();

        const bodyGrad = ctx.createRadialGradient(s * 0.08, -s * 0.1, 0, 0, 0, s * 0.45);
        bodyGrad.addColorStop(0, this.color.light);
        bodyGrad.addColorStop(0.5, this.color.body);
        bodyGrad.addColorStop(1, '#8B0000');
        ctx.fillStyle = bodyGrad;
        ctx.fill();

        // 内部灯光
        const innerGlow = ctx.createRadialGradient(0, 0, 0, 0, 0, s * 0.3);
        innerGlow.addColorStop(0, `rgba(255, 255, 230, ${1.0 * glow})`);
        innerGlow.addColorStop(0.4, `rgba(255, 240, 180, ${0.6 * glow})`);
        innerGlow.addColorStop(1, 'rgba(255, 180, 100, 0)');
        ctx.fillStyle = innerGlow;
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.35);
        ctx.lineTo(s * 0.25, 0);
        ctx.lineTo(0, s * 0.35);
        ctx.lineTo(-s * 0.25, 0);
        ctx.closePath();
        ctx.fill();

        // 装饰线
        ctx.strokeStyle = '#D4AF37';
        ctx.lineWidth = 0.5;
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.45);
        ctx.lineTo(0, s * 0.45);
        ctx.stroke();
        ctx.beginPath();
        ctx.moveTo(-s * 0.35, 0);
        ctx.lineTo(s * 0.35, 0);
        ctx.stroke();

        // 高光
        ctx.fillStyle = `rgba(255, 255, 255, ${0.4 * glow})`;
        ctx.beginPath();
        ctx.arc(s * 0.08, -s * 0.15, s * 0.06, 0, Math.PI * 2);
        ctx.fill();

        // 底部装饰
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.arc(0, s * 0.5, s * 0.08, 0, Math.PI * 2);
        ctx.fill();

        // 流苏
        this.drawTassel(ctx, s * 0.52, s, glow);
    }

    // 通用流苏绘制
    drawTassel(ctx, startY, s, glow) {
        ctx.strokeStyle = this.color.body;
        ctx.lineWidth = 0.8;
        ctx.lineCap = 'round';

        // 中心结
        ctx.fillStyle = '#D4AF37';
        ctx.beginPath();
        ctx.arc(0, startY + s * 0.08, s * 0.06, 0, Math.PI * 2);
        ctx.fill();

        // 3条流苏
        const swing = Math.sin(this.swingAngle - 0.2) * 0.06;
        for (let i = -1; i <= 1; i++) {
            ctx.beginPath();
            ctx.moveTo(0, startY + s * 0.08);
            const endX = i * s * 0.12 + Math.sin(swing) * s * 0.05;
            ctx.lineTo(endX, startY + s * 0.35);
            ctx.stroke();
        }
    }

    // 红包
    drawRedEnvelope(ctx, s, glow) {
        s = s * 1.4;

        // 红包主体
        ctx.beginPath();
        ctx.roundRect(-s * 0.35, -s * 0.5, s * 0.7, s * 0.9, s * 0.05);
        ctx.fillStyle = '#E53935';
        ctx.fill();
        ctx.strokeStyle = '#B71C1C';
        ctx.lineWidth = 0.8;
        ctx.stroke();

        // 金色封口
        ctx.beginPath();
        ctx.moveTo(-s * 0.35, -s * 0.15);
        ctx.lineTo(0, s * 0.05);
        ctx.lineTo(s * 0.35, -s * 0.15);
        ctx.lineTo(s * 0.35, -s * 0.5);
        ctx.lineTo(-s * 0.35, -s * 0.5);
        ctx.closePath();
        ctx.fillStyle = '#FFD700';
        ctx.fill();

        // 中心圆形装饰
        ctx.beginPath();
        ctx.arc(0, s * 0.15, s * 0.18, 0, Math.PI * 2);
        ctx.fillStyle = '#FFD700';
        ctx.fill();

        // 福字简化
        ctx.fillStyle = '#E53935';
        ctx.font = `bold ${s * 0.22}px serif`;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('福', 0, s * 0.16);
    }

    // 剪纸风格福字（窗花）
    drawPaperCutFu(ctx, s, glow) {
        s = s * 1.8;

        // 圆形窗花背景
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.5, 0, Math.PI * 2);
        ctx.fillStyle = '#E53935';
        ctx.fill();

        // 内圈花边装饰
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.42, 0, Math.PI * 2);
        ctx.strokeStyle = '#FFCDD2';
        ctx.lineWidth = 2;
        ctx.stroke();

        // 外圈花边装饰
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.48, 0, Math.PI * 2);
        ctx.strokeStyle = '#B71C1C';
        ctx.lineWidth = 1;
        ctx.stroke();

        // 倒福字
        ctx.save();
        ctx.rotate(Math.PI);
        ctx.fillStyle = '#FFCDD2';
        ctx.font = `bold ${s * 0.58}px serif`;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('福', 0, 0);
        ctx.restore();
    }

    // 鞭炮
    drawFirecracker(ctx, s, glow) {
        s = s * 2;

        // 引线
        ctx.strokeStyle = '#5D4037';
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.85);
        ctx.quadraticCurveTo(s * 0.2, -s * 0.7, 0, -s * 0.5);
        ctx.stroke();

        // 火花
        const sparkGrad = ctx.createRadialGradient(0, -s * 0.88, 0, 0, -s * 0.88, s * 0.12);
        sparkGrad.addColorStop(0, `rgba(255, 255, 200, ${glow})`);
        sparkGrad.addColorStop(0.4, `rgba(255, 200, 50, ${glow * 0.8})`);
        sparkGrad.addColorStop(1, `rgba(255, 100, 0, 0)`);
        ctx.fillStyle = sparkGrad;
        ctx.beginPath();
        ctx.arc(0, -s * 0.88, s * 0.12, 0, Math.PI * 2);
        ctx.fill();

        // 小火星
        ctx.fillStyle = `rgba(255, 220, 100, ${glow * 0.9})`;
        ctx.beginPath();
        ctx.arc(-s * 0.06, -s * 0.95, s * 0.025, 0, Math.PI * 2);
        ctx.arc(s * 0.05, -s * 0.92, s * 0.02, 0, Math.PI * 2);
        ctx.fill();

        // 鞭炮串（3个圆柱形）
        for (let i = 0; i < 3; i++) {
            const y = -s * 0.45 + i * s * 0.38;
            const x = (i === 1) ? s * 0.08 : -s * 0.08;

            ctx.save();
            ctx.translate(x, y);

            // 鞭炮主体（圆柱）
            ctx.beginPath();
            ctx.ellipse(0, 0, s * 0.12, s * 0.2, 0, 0, Math.PI * 2);
            const bodyGrad = ctx.createLinearGradient(-s * 0.12, 0, s * 0.12, 0);
            bodyGrad.addColorStop(0, '#B71C1C');
            bodyGrad.addColorStop(0.3, '#E53935');
            bodyGrad.addColorStop(0.7, '#E53935');
            bodyGrad.addColorStop(1, '#B71C1C');
            ctx.fillStyle = bodyGrad;
            ctx.fill();

            // 金色顶部
            ctx.beginPath();
            ctx.ellipse(0, -s * 0.17, s * 0.1, s * 0.04, 0, 0, Math.PI * 2);
            ctx.fillStyle = '#FFD700';
            ctx.fill();

            // 金色底部
            ctx.beginPath();
            ctx.ellipse(0, s * 0.17, s * 0.1, s * 0.04, 0, 0, Math.PI * 2);
            ctx.fillStyle = '#DAA520';
            ctx.fill();

            ctx.restore();
        }

        // 连接绳
        ctx.strokeStyle = '#E53935';
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        ctx.moveTo(-s * 0.08, -s * 0.45);
        ctx.lineTo(s * 0.08, -s * 0.07);
        ctx.lineTo(-s * 0.08, s * 0.31);
        ctx.stroke();
    }

    // 中国结
    drawChineseKnot(ctx, s, glow) {
        s = s * 1.8;

        // 主体菱形
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.4);
        ctx.lineTo(s * 0.35, 0);
        ctx.lineTo(0, s * 0.4);
        ctx.lineTo(-s * 0.35, 0);
        ctx.closePath();
        ctx.fillStyle = '#C62828';
        ctx.fill();
        ctx.strokeStyle = '#8B0000';
        ctx.lineWidth = 1.5;
        ctx.stroke();

        // 内部小菱形
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.2);
        ctx.lineTo(s * 0.18, 0);
        ctx.lineTo(0, s * 0.2);
        ctx.lineTo(-s * 0.18, 0);
        ctx.closePath();
        ctx.fillStyle = '#FFD700';
        ctx.fill();

        // 上方挂绳
        ctx.strokeStyle = '#C62828';
        ctx.lineWidth = 2.5;
        ctx.beginPath();
        ctx.moveTo(0, -s * 0.4);
        ctx.lineTo(0, -s * 0.65);
        ctx.stroke();

        // 下方流苏
        ctx.strokeStyle = '#C62828';
        ctx.lineWidth = 2;
        for (let i = -1; i <= 1; i++) {
            ctx.beginPath();
            ctx.moveTo(0, s * 0.4);
            ctx.lineTo(i * s * 0.15, s * 0.75);
            ctx.stroke();
        }
    }

    // 倒福字
    drawFuCharacter(ctx, s, glow) {
        s = s * 1.8;

        // 红色菱形背景
        ctx.save();
        ctx.rotate(Math.PI / 4);
        ctx.beginPath();
        ctx.rect(-s * 0.38, -s * 0.38, s * 0.76, s * 0.76);
        ctx.fillStyle = '#E53935';
        ctx.fill();
        ctx.strokeStyle = '#FFD700';
        ctx.lineWidth = 2;
        ctx.stroke();
        ctx.restore();

        // 倒置的福字
        ctx.save();
        ctx.rotate(Math.PI);
        ctx.fillStyle = '#FFD700';
        ctx.font = `bold ${s * 0.5}px serif`;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('福', 0, 0);
        ctx.restore();
    }
}

export function createLanterns(config, canvas) {
    const lanterns = [];
    const count = config.lanternCount || 12;

    for (let i = 0; i < count; i++) {
        const lantern = new Lantern(config, canvas);
        lantern.type = i % 10;  // 轮询分配10种类型
        lantern.y = Math.random() * (canvas?.height || window.innerHeight);
        lanterns.push(lantern);
    }

    return lanterns;
}
