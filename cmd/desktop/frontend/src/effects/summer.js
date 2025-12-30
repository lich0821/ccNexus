/**
 * Summer Effect - 夏天氛围效果模块
 * 包含：西瓜、冰淇淋、阳光、向日葵、冰棍
 */

export class SummerElement {
    constructor(config, canvas) {
        this.config = config;
        this.canvas = canvas;
        this.reset();
    }

    reset() {
        const w = this.canvas?.width || window.innerWidth;
        const h = this.canvas?.height || window.innerHeight;

        this.x = Math.random() * w;
        this.y = Math.random() * -100;
        this.type = this.getRandomType();
        this.size = 10;

        const speedFactor = 0.35 + (this.size / 50);
        this.speedY = (Math.random() * 0.4 + 0.3) * (this.config.speed || 1.0) * speedFactor;
        this.speedX = (Math.random() - 0.5) * (this.config.wind || 0.3) * 1.5;

        this.swingAngle = Math.random() * Math.PI * 2;
        this.swingSpeed = Math.random() * 0.02 + 0.01;
        this.swingRadius = Math.random() * 1.2 + 0.6;

        this.opacity = (Math.random() * 0.2 + 0.6) * (this.config.opacity || 0.75);
        this.rotation = Math.random() * Math.PI * 2;
        this.rotationSpeed = (Math.random() - 0.5) * 0.04;
    }

    getRandomType() {
        // 0=西瓜, 1=冰淇淋, 2=阳光, 3=向日葵, 4=冰棍
        const weights = [30, 15, 10, 15, 30];
        let r = Math.random() * 100;
        for (let i = 0; i < weights.length; i++) {
            r -= weights[i];
            if (r <= 0) return i;
        }
        return 0;
    }

    update() {
        this.swingAngle += this.swingSpeed;
        this.rotation += this.rotationSpeed;
        this.y += this.speedY;
        this.x += this.speedX + Math.sin(this.swingAngle) * this.swingRadius;

        const h = this.canvas?.height || window.innerHeight;
        const w = this.canvas?.width || window.innerWidth;

        if (this.y > h + 50) {
            this.y = -50;
            this.x = Math.random() * w;
        }
        if (this.x > w + 30) this.x = -30;
        else if (this.x < -30) this.x = w + 30;
    }

    draw(ctx) {
        if (!ctx) return;
        ctx.save();
        ctx.globalAlpha = this.opacity;
        ctx.translate(this.x, this.y);

        switch (this.type) {
            case 0: this.drawWatermelon(ctx); break;
            case 1: this.drawIceCream(ctx); break;
            case 2: this.drawSunlight(ctx); break;
            case 3: this.drawSunflower(ctx); break;
            case 4: this.drawPopsicle(ctx); break;
        }

        ctx.restore();
    }

    drawWatermelon(ctx) {
        ctx.rotate(this.rotation);
        const s = this.size * 1.2;

        // 绿皮（外弧）
        ctx.beginPath();
        ctx.arc(0, 0, s, Math.PI, 0, false);
        ctx.closePath();
        ctx.fillStyle = '#4CAF50';
        ctx.fill();

        // 白边
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.9, Math.PI, 0, false);
        ctx.closePath();
        ctx.fillStyle = '#C8E6C9';
        ctx.fill();

        // 红色果肉
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.82, Math.PI, 0, false);
        ctx.closePath();
        ctx.fillStyle = '#EF5350';
        ctx.fill();

        // 黑色籽
        ctx.fillStyle = '#3E2723';
        const seeds = [
            { x: -s * 0.4, y: -s * 0.35 },
            { x: 0, y: -s * 0.5 },
            { x: s * 0.4, y: -s * 0.35 },
            { x: -s * 0.2, y: -s * 0.2 },
            { x: s * 0.2, y: -s * 0.2 }
        ];
        for (const seed of seeds) {
            ctx.beginPath();
            ctx.ellipse(seed.x, seed.y, s * 0.05, s * 0.08, 0, 0, Math.PI * 2);
            ctx.fill();
        }
    }

    drawIceCream(ctx) {
        ctx.rotate(this.rotation);
        const s = this.size * 1.2;

        // 甜筒（截断底部）
        ctx.beginPath();
        ctx.moveTo(-s * 0.4, -s * 0.1);
        ctx.lineTo(-s * 0.15, s * 0.9);
        ctx.lineTo(s * 0.15, s * 0.9);
        ctx.lineTo(s * 0.4, -s * 0.1);
        ctx.closePath();
        ctx.fillStyle = '#D4A056';
        ctx.fill();
        ctx.strokeStyle = '#A67C3D';
        ctx.lineWidth = 0.8;
        ctx.stroke();

        // 甜筒纹理（交叉线）
        ctx.strokeStyle = '#C4903C';
        ctx.lineWidth = 0.6;
        for (let i = 0; i < 3; i++) {
            ctx.beginPath();
            ctx.moveTo(-s * 0.35 + i * s * 0.1, s * 0.1);
            ctx.lineTo(s * 0.0 - i * s * 0.03, s * 0.8);
            ctx.stroke();
            ctx.beginPath();
            ctx.moveTo(s * 0.35 - i * s * 0.1, s * 0.1);
            ctx.lineTo(-s * 0.0 + i * s * 0.03, s * 0.8);
            ctx.stroke();
        }

        // 冰淇淋（螺旋奶油）- 加深颜色和描边
        ctx.fillStyle = '#FFF3CD';
        ctx.strokeStyle = '#E0C9A6';
        ctx.lineWidth = 0.5;
        // 底层
        ctx.beginPath();
        ctx.arc(0, -s * 0.15, s * 0.42, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        // 中层
        ctx.beginPath();
        ctx.arc(0, -s * 0.45, s * 0.35, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        // 顶层
        ctx.beginPath();
        ctx.arc(0, -s * 0.7, s * 0.25, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        // 尖顶
        ctx.beginPath();
        ctx.arc(0, -s * 0.88, s * 0.12, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
    }

    drawSunlight(ctx) {
        ctx.rotate(this.rotation);
        const s = this.size * 1.2;

        // 光芒（三角形射线）
        ctx.fillStyle = '#FFD54F';
        for (let i = 0; i < 8; i++) {
            ctx.save();
            ctx.rotate((Math.PI * 2 / 8) * i);
            ctx.beginPath();
            ctx.moveTo(-s * 0.12, -s * 0.45);
            ctx.lineTo(0, -s * 0.95);
            ctx.lineTo(s * 0.12, -s * 0.45);
            ctx.closePath();
            ctx.fill();
            ctx.restore();
        }

        // 太阳主体
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.5, 0, Math.PI * 2);
        ctx.fillStyle = '#FFCA28';
        ctx.fill();

        // 内圈
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.38, 0, Math.PI * 2);
        ctx.fillStyle = '#FFE082';
        ctx.fill();
    }

    drawSunflower(ctx) {
        ctx.rotate(this.rotation);
        const s = this.size * 0.9;

        // 茎（绿色）
        ctx.fillStyle = '#4CAF50';
        ctx.beginPath();
        ctx.moveTo(-s * 0.12, s * 0.35);
        ctx.lineTo(-s * 0.1, s * 1.8);
        ctx.lineTo(s * 0.1, s * 1.8);
        ctx.lineTo(s * 0.12, s * 0.35);
        ctx.closePath();
        ctx.fill();

        // 左叶子
        ctx.fillStyle = '#66BB6A';
        ctx.beginPath();
        ctx.moveTo(-s * 0.1, s * 0.9);
        ctx.quadraticCurveTo(-s * 0.8, s * 0.5, -s * 0.65, s * 1.0);
        ctx.quadraticCurveTo(-s * 0.4, s * 1.15, -s * 0.1, s * 1.0);
        ctx.closePath();
        ctx.fill();

        // 右叶子
        ctx.beginPath();
        ctx.moveTo(s * 0.1, s * 1.2);
        ctx.quadraticCurveTo(s * 0.8, s * 0.85, s * 0.65, s * 1.35);
        ctx.quadraticCurveTo(s * 0.4, s * 1.5, s * 0.1, s * 1.35);
        ctx.closePath();
        ctx.fill();

        // 花瓣（黄色椭圆）
        ctx.fillStyle = '#FFC107';
        for (let i = 0; i < 12; i++) {
            ctx.save();
            ctx.rotate((Math.PI * 2 / 12) * i);
            ctx.beginPath();
            ctx.ellipse(0, -s * 0.55, s * 0.18, s * 0.38, 0, 0, Math.PI * 2);
            ctx.fill();
            ctx.restore();
        }

        // 花盘（棕色）
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.35, 0, Math.PI * 2);
        ctx.fillStyle = '#795548';
        ctx.fill();

        // 花盘内圈
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.24, 0, Math.PI * 2);
        ctx.fillStyle = '#5D4037';
        ctx.fill();
    }

    drawPopsicle(ctx) {
        ctx.rotate(this.rotation);
        const s = this.size * 1.2;

        // 冰棍棒（短一点）
        ctx.fillStyle = '#D7CCC8';
        ctx.beginPath();
        ctx.moveTo(-s * 0.12, s * 0.5);
        ctx.lineTo(-s * 0.12, s * 1.0);
        ctx.quadraticCurveTo(-s * 0.12, s * 1.15, 0, s * 1.15);
        ctx.quadraticCurveTo(s * 0.12, s * 1.15, s * 0.12, s * 1.0);
        ctx.lineTo(s * 0.12, s * 0.5);
        ctx.closePath();
        ctx.fill();
        ctx.strokeStyle = '#BCAAA4';
        ctx.lineWidth = 0.5;
        ctx.stroke();

        // 冰棍主体（大一点）
        ctx.beginPath();
        ctx.moveTo(-s * 0.42, s * 0.5);
        ctx.lineTo(-s * 0.42, -s * 0.55);
        ctx.quadraticCurveTo(-s * 0.42, -s * 0.85, 0, -s * 0.85);
        ctx.quadraticCurveTo(s * 0.42, -s * 0.85, s * 0.42, -s * 0.55);
        ctx.lineTo(s * 0.42, s * 0.5);
        ctx.closePath();
        ctx.fillStyle = '#FF7043';
        ctx.fill();
        ctx.strokeStyle = '#E64A19';
        ctx.lineWidth = 0.8;
        ctx.stroke();

        // 咬痕（半圆缺口）
        ctx.fillStyle = '#FFF8E1';
        ctx.beginPath();
        ctx.arc(s * 0.3, -s * 0.5, s * 0.2, 0, Math.PI * 2);
        ctx.fill();

        // 高光
        ctx.fillStyle = 'rgba(255,255,255,0.4)';
        ctx.beginPath();
        ctx.ellipse(-s * 0.18, -s * 0.3, s * 0.1, s * 0.3, 0, 0, Math.PI * 2);
        ctx.fill();
    }
}

export function createSummerElements(config, canvas) {
    const elements = [];
    const count = config.summerCount || 12;
    const w = canvas?.width || window.innerWidth;
    const h = canvas?.height || window.innerHeight;

    for (let i = 0; i < count; i++) {
        const element = new SummerElement(config, canvas);
        const sectionWidth = w / count;
        element.x = sectionWidth * i + Math.random() * sectionWidth;
        element.y = Math.random() * h;
        elements.push(element);
    }
    return elements;
}
