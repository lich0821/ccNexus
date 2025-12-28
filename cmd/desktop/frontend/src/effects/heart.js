/**
 * Heart Effect - 情人节爱心效果模块
 * 多种造型爱心从上往下飘落
 */

export class Heart {
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
        this.size = 12;

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
        this.opacity = (Math.random() * 0.15 + 0.55) * (this.config.opacity || 0.85);

        // 旋转
        this.rotation = (Math.random() - 0.5) * 0.3;
    }

    getRandomColor() {
        const colors = [
            { body: '#E91E63', light: '#F8BBD9' },  // 粉红
            { body: '#F44336', light: '#FFCDD2' },  // 红色
            { body: '#E91E63', light: '#FCE4EC' },  // 玫红
            { body: '#FF5722', light: '#FFCCBC' },  // 珊瑚红
            { body: '#9C27B0', light: '#E1BEE7' },  // 紫色
        ];
        const weights = [35, 30, 20, 10, 5];
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
        ctx.rotate(this.rotation + Math.sin(this.swingAngle) * 0.05);

        const s = this.size;

        // 根据类型绘制不同造型
        if (this.type === 0) {
            this.drawClassicHeart(ctx, s);
        } else if (this.type === 1) {
            this.drawBubbleHeart(ctx, s);
        } else if (this.type === 2) {
            this.drawSparkleHeart(ctx, s);
        } else if (this.type === 3) {
            this.drawDoubleHeart(ctx, s);
        } else {
            this.drawOutlineHeart(ctx, s);
        }

        ctx.restore();
    }

    // 绘制爱心路径
    drawHeartPath(ctx, s, offsetX = 0, offsetY = 0) {
        ctx.beginPath();
        ctx.moveTo(offsetX, offsetY + s * 0.3);
        ctx.bezierCurveTo(offsetX, offsetY - s * 0.1, offsetX - s * 0.5, offsetY - s * 0.1, offsetX - s * 0.5, offsetY + s * 0.2);
        ctx.bezierCurveTo(offsetX - s * 0.5, offsetY + s * 0.5, offsetX, offsetY + s * 0.7, offsetX, offsetY + s * 0.9);
        ctx.bezierCurveTo(offsetX, offsetY + s * 0.7, offsetX + s * 0.5, offsetY + s * 0.5, offsetX + s * 0.5, offsetY + s * 0.2);
        ctx.bezierCurveTo(offsetX + s * 0.5, offsetY - s * 0.1, offsetX, offsetY - s * 0.1, offsetX, offsetY + s * 0.3);
        ctx.closePath();
    }

    // 经典爱心
    drawClassicHeart(ctx, s) {
        s = s * 1.4;

        // 爱心主体
        this.drawHeartPath(ctx, s);
        const bodyGrad = ctx.createRadialGradient(s * 0.1, s * 0.2, 0, 0, s * 0.4, s * 0.8);
        bodyGrad.addColorStop(0, this.color.light);
        bodyGrad.addColorStop(0.5, this.color.body);
        bodyGrad.addColorStop(1, '#8B0035');
        ctx.fillStyle = bodyGrad;
        ctx.fill();

        // 高光
        ctx.fillStyle = 'rgba(255, 255, 255, 0.4)';
        ctx.beginPath();
        ctx.ellipse(-s * 0.2, s * 0.15, s * 0.12, s * 0.08, -0.5, 0, Math.PI * 2);
        ctx.fill();
    }

    // 气泡爱心
    drawBubbleHeart(ctx, s) {
        s = s * 1.3;

        // 爱心主体 - 半透明气泡效果
        this.drawHeartPath(ctx, s);
        const bodyGrad = ctx.createRadialGradient(s * 0.15, s * 0.15, 0, 0, s * 0.4, s * 0.9);
        bodyGrad.addColorStop(0, 'rgba(255, 255, 255, 0.8)');
        bodyGrad.addColorStop(0.3, this.color.light);
        bodyGrad.addColorStop(0.7, this.color.body);
        bodyGrad.addColorStop(1, 'rgba(200, 50, 100, 0.8)');
        ctx.fillStyle = bodyGrad;
        ctx.fill();

        // 边框
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.5)';
        ctx.lineWidth = 1;
        this.drawHeartPath(ctx, s);
        ctx.stroke();

        // 高光点
        ctx.fillStyle = 'rgba(255, 255, 255, 0.7)';
        ctx.beginPath();
        ctx.arc(-s * 0.2, s * 0.1, s * 0.1, 0, Math.PI * 2);
        ctx.fill();
        ctx.beginPath();
        ctx.arc(-s * 0.08, s * 0.02, s * 0.05, 0, Math.PI * 2);
        ctx.fill();
    }

    // 闪耀爱心
    drawSparkleHeart(ctx, s) {
        s = s * 1.35;

        // 爱心主体
        this.drawHeartPath(ctx, s);
        const bodyGrad = ctx.createLinearGradient(-s * 0.5, 0, s * 0.5, s);
        bodyGrad.addColorStop(0, this.color.light);
        bodyGrad.addColorStop(0.5, this.color.body);
        bodyGrad.addColorStop(1, '#8B0050');
        ctx.fillStyle = bodyGrad;
        ctx.fill();

        // 闪烁星星
        ctx.fillStyle = 'rgba(255, 255, 255, 0.8)';
        this.drawStar(ctx, -s * 0.25, s * 0.2, s * 0.08);
        this.drawStar(ctx, s * 0.15, s * 0.5, s * 0.06);
        this.drawStar(ctx, -s * 0.1, s * 0.65, s * 0.05);
    }

    // 绘制小星星
    drawStar(ctx, x, y, size) {
        ctx.beginPath();
        for (let i = 0; i < 4; i++) {
            const angle = (i / 4) * Math.PI * 2;
            const px = x + Math.cos(angle) * size;
            const py = y + Math.sin(angle) * size;
            if (i === 0) ctx.moveTo(px, py);
            else ctx.lineTo(px, py);

            const midAngle = angle + Math.PI / 4;
            const mx = x + Math.cos(midAngle) * size * 0.4;
            const my = y + Math.sin(midAngle) * size * 0.4;
            ctx.lineTo(mx, my);
        }
        ctx.closePath();
        ctx.fill();
    }

    // 双心
    drawDoubleHeart(ctx, s) {
        s = s * 1.2;

        // 后面的小爱心
        ctx.save();
        ctx.translate(s * 0.2, -s * 0.1);
        ctx.scale(0.6, 0.6);
        this.drawHeartPath(ctx, s);
        ctx.fillStyle = 'rgba(255, 150, 180, 0.6)';
        ctx.fill();
        ctx.restore();

        // 前面的大爱心
        this.drawHeartPath(ctx, s);
        const bodyGrad = ctx.createRadialGradient(s * 0.1, s * 0.2, 0, 0, s * 0.4, s * 0.8);
        bodyGrad.addColorStop(0, this.color.light);
        bodyGrad.addColorStop(0.5, this.color.body);
        bodyGrad.addColorStop(1, '#8B0035');
        ctx.fillStyle = bodyGrad;
        ctx.fill();

        // 高光
        ctx.fillStyle = 'rgba(255, 255, 255, 0.4)';
        ctx.beginPath();
        ctx.ellipse(-s * 0.18, s * 0.15, s * 0.1, s * 0.06, -0.5, 0, Math.PI * 2);
        ctx.fill();
    }

    // 丘比特爱心
    drawOutlineHeart(ctx, s) {
        s = s * 1.5;

        // 左小翅膀
        ctx.fillStyle = '#FFFFFF';
        ctx.strokeStyle = 'rgba(180, 180, 180, 0.6)';
        ctx.lineWidth = 0.5;
        ctx.beginPath();
        ctx.ellipse(-s * 0.5, s * 0.15, s * 0.2, s * 0.07, -0.8, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        ctx.beginPath();
        ctx.ellipse(-s * 0.6, s * 0.05, s * 0.18, s * 0.06, -1.0, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        ctx.beginPath();
        ctx.ellipse(-s * 0.65, -s * 0.08, s * 0.15, s * 0.05, -1.2, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();

        // 右小翅膀
        ctx.beginPath();
        ctx.ellipse(s * 0.5, s * 0.15, s * 0.2, s * 0.07, 0.8, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        ctx.beginPath();
        ctx.ellipse(s * 0.6, s * 0.05, s * 0.18, s * 0.06, 1.0, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        ctx.beginPath();
        ctx.ellipse(s * 0.65, -s * 0.08, s * 0.15, s * 0.05, 1.2, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();

        // 爱心主体
        this.drawHeartPath(ctx, s);
        const bodyGrad = ctx.createRadialGradient(s * 0.1, s * 0.2, 0, 0, s * 0.4, s * 0.8);
        bodyGrad.addColorStop(0, this.color.light);
        bodyGrad.addColorStop(0.5, this.color.body);
        bodyGrad.addColorStop(1, '#8B0035');
        ctx.fillStyle = bodyGrad;
        ctx.fill();
    }
}

export function createHearts(config, canvas) {
    const hearts = [];
    const count = config.heartCount || 15;
    const w = canvas?.width || window.innerWidth;
    const h = canvas?.height || window.innerHeight;

    for (let i = 0; i < count; i++) {
        const heart = new Heart(config, canvas);
        heart.type = i % 5;
        // 按区域均匀分布x位置，避免聚集
        const sectionWidth = w / count;
        heart.x = sectionWidth * i + Math.random() * sectionWidth;
        heart.y = Math.random() * h;
        hearts.push(heart);
    }

    return hearts;
}
