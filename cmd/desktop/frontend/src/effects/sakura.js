/**
 * Sakura Effect - 樱花飘落效果模块
 * 五瓣樱花从上往下飘落
 */

export class Sakura {
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

        this.size = 10;
        this.color = this.getRandomColor();

        const speedFactor = 0.35 + (this.size / 50);
        this.speedY = (Math.random() * 0.4 + 0.3) * (this.config.speed || 1.0) * speedFactor;
        this.speedX = (Math.random() - 0.5) * (this.config.wind || 0.3) * 1.5;

        this.swingAngle = Math.random() * Math.PI * 2;
        this.swingSpeed = Math.random() * 0.015 + 0.008;
        this.swingRadius = Math.random() * 1.0 + 0.5;

        this.opacity = (Math.random() * 0.2 + 0.7) * (this.config.opacity || 0.85);

        this.rotation = Math.random() * Math.PI * 2;
        this.rotationSpeed = (Math.random() - 0.5) * 0.03;
    }

    getRandomColor() {
        const colors = [
            { body: '#FFB7C5', light: '#FFE4E9', stroke: '#E8A0AD' },  // 淡粉
            { body: '#FFC0CB', light: '#FFE4EC', stroke: '#E0A8B3' },  // 樱粉
            { body: '#FF91A4', light: '#FFBCC9', stroke: '#D87A8C' },  // 深粉
            { body: '#FFD1DC', light: '#FFF0F5', stroke: '#E0B8C3' },  // 浅粉
            { body: '#DDA0DD', light: '#E6D5E6', stroke: '#C088C0' },  // 淡紫
        ];
        const weights = [30, 30, 20, 15, 5];
        let r = Math.random() * 100;
        for (let i = 0; i < colors.length; i++) {
            r -= weights[i];
            if (r <= 0) return colors[i];
        }
        return colors[0];
    }

    update() {
        this.swingAngle += this.swingSpeed;
        this.rotation += this.rotationSpeed;
        this.y += this.speedY;
        this.x += this.speedX + Math.sin(this.swingAngle) * this.swingRadius;

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
        ctx.rotate(this.rotation);

        const s = this.size * 1.5;
        const petalCount = 5;

        // 绘制5片花瓣
        for (let i = 0; i < petalCount; i++) {
            const angle = (i / petalCount) * Math.PI * 2 - Math.PI / 2;
            ctx.save();
            ctx.rotate(angle);

            // 樱花花瓣（带V形缺口）
            ctx.beginPath();
            ctx.moveTo(0, 0);
            // 左边曲线
            ctx.quadraticCurveTo(-s * 0.2, -s * 0.3, -s * 0.15, -s * 0.5);
            // 左边到缺口
            ctx.quadraticCurveTo(-s * 0.1, -s * 0.55, -s * 0.05, -s * 0.48);
            // V形缺口
            ctx.lineTo(0, -s * 0.4);
            ctx.lineTo(s * 0.05, -s * 0.48);
            // 缺口到右边
            ctx.quadraticCurveTo(s * 0.1, -s * 0.55, s * 0.15, -s * 0.5);
            // 右边曲线
            ctx.quadraticCurveTo(s * 0.2, -s * 0.3, 0, 0);
            ctx.closePath();

            const grad = ctx.createLinearGradient(0, 0, 0, -s * 0.5);
            grad.addColorStop(0, this.color.light);
            grad.addColorStop(1, this.color.body);
            ctx.fillStyle = grad;
            ctx.fill();

            ctx.strokeStyle = this.color.stroke;
            ctx.lineWidth = 0.5;
            ctx.stroke();

            ctx.restore();
        }

        // 花蕊
        ctx.fillStyle = '#FFD700';
        ctx.beginPath();
        ctx.arc(0, 0, s * 0.12, 0, Math.PI * 2);
        ctx.fill();

        ctx.restore();
    }
}

export function createSakuras(config, canvas) {
    const sakuras = [];
    const count = config.sakuraCount || 20;
    const w = canvas?.width || window.innerWidth;
    const h = canvas?.height || window.innerHeight;

    for (let i = 0; i < count; i++) {
        const sakura = new Sakura(config, canvas);
        const sectionWidth = w / count;
        sakura.x = sectionWidth * i + Math.random() * sectionWidth;
        sakura.y = Math.random() * h;
        sakuras.push(sakura);
    }

    return sakuras;
}
