/**
 * Maple Effect - 枫叶飘落效果模块
 * 秋天氛围枫叶从上往下飘落
 */

export class Maple {
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

        this.size = 7;
        this.color = this.getRandomColor();

        const speedFactor = 0.4 + (this.size / 50);
        this.speedY = (Math.random() * 0.5 + 0.4) * (this.config.speed || 1.0) * speedFactor;
        this.speedX = (Math.random() - 0.5) * (this.config.wind || 0.4) * 1.5;

        this.swingAngle = Math.random() * Math.PI * 2;
        this.swingSpeed = Math.random() * 0.02 + 0.01;
        this.swingRadius = Math.random() * 1.2 + 0.6;

        this.opacity = (Math.random() * 0.2 + 0.7) * (this.config.opacity || 0.85);

        this.rotation = Math.random() * Math.PI * 2;
        this.rotationSpeed = (Math.random() - 0.5) * 0.04;
    }

    getRandomColor() {
        const colors = [
            { body: '#E74C3C', light: '#F1948A', stroke: '#C0392B' },  // 红色
            { body: '#E67E22', light: '#F5B041', stroke: '#D35400' },  // 橙色
            { body: '#F39C12', light: '#F7DC6F', stroke: '#D68910' },  // 金黄
            { body: '#D35400', light: '#E59866', stroke: '#A04000' },  // 深橙
            { body: '#C0392B', light: '#E74C3C', stroke: '#922B21' },  // 深红
        ];
        const weights = [30, 25, 25, 12, 8];
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
        ctx.rotate(this.rotation);

        const s = this.size * 1.5;

        // 绘制枫叶🍁
        // 说明：用点集 + 二次曲线平滑，避免折线+miter 造成“炸刺怪虫”
        const leafScale = s * 1.15;
        ctx.translate(0, -leafScale * 0.10);

        const px = (v) => leafScale * v;
        const py = (v) => leafScale * v;

        const stemAttachY = py(0.62);
        const topY = py(-0.92);

        // 轮廓
        ctx.beginPath();
        ctx.lineJoin = 'round';
        ctx.lineCap = 'round';

        // 从叶柄凹口中心出发，沿右边缘到顶尖，再回到左侧凹口
        ctx.moveTo(0, stemAttachY);

        // 右侧基裂片
        ctx.bezierCurveTo(px(0.12), py(0.54), px(0.32), py(0.52), px(0.54), py(0.62));
        ctx.bezierCurveTo(px(0.70), py(0.72), px(0.72), py(0.60), px(0.58), py(0.50));
        // 右侧基裂片与右大裂片之间的凹陷
        ctx.bezierCurveTo(px(0.48), py(0.46), px(0.40), py(0.44), px(0.32), py(0.44));

        // 右大裂片（主尖）
        ctx.bezierCurveTo(px(0.46), py(0.30), px(0.86), py(0.26), px(1.04), py(0.16));
        ctx.bezierCurveTo(px(1.02), py(0.04), px(0.82), py(-0.02), px(0.70), py(0.02));

        // 右上裂片（次尖）
        ctx.bezierCurveTo(px(0.74), py(-0.04), px(0.84), py(-0.18), px(0.86), py(-0.30));
        ctx.bezierCurveTo(px(0.88), py(-0.46), px(0.64), py(-0.34), px(0.50), py(-0.20));

        // 顶裂片
        // 右侧凹口
        ctx.bezierCurveTo(px(0.44), py(-0.22), px(0.34), py(-0.24), px(0.26), py(-0.28));
        // 右小尖
        ctx.bezierCurveTo(px(0.34), py(-0.46), px(0.42), py(-0.58), px(0.32), py(-0.64));
        // 右小尖与中尖之间凹口
        ctx.bezierCurveTo(px(0.24), py(-0.60), px(0.16), py(-0.62), px(0.12), py(-0.68));
        // 中尖（最高）
        ctx.bezierCurveTo(px(0.08), py(-0.80), px(0.04), py(-0.88), 0, topY);

        // 左侧镜像
        ctx.bezierCurveTo(px(-0.04), py(-0.88), px(-0.08), py(-0.80), px(-0.12), py(-0.68));
        // 左小尖与中尖之间凹口
        ctx.bezierCurveTo(px(-0.16), py(-0.62), px(-0.24), py(-0.60), px(-0.32), py(-0.64));
        // 左小尖
        ctx.bezierCurveTo(px(-0.42), py(-0.58), px(-0.34), py(-0.46), px(-0.26), py(-0.28));
        // 左侧凹口
        ctx.bezierCurveTo(px(-0.34), py(-0.24), px(-0.44), py(-0.22), px(-0.50), py(-0.20));

        // 左上裂片
        ctx.bezierCurveTo(px(-0.64), py(-0.34), px(-0.88), py(-0.46), px(-0.86), py(-0.30));
        ctx.bezierCurveTo(px(-0.84), py(-0.18), px(-0.74), py(-0.04), px(-0.70), py(0.02));

        // 左大裂片（主尖）
        ctx.bezierCurveTo(px(-0.82), py(-0.02), px(-1.02), py(0.04), px(-1.04), py(0.16));
        ctx.bezierCurveTo(px(-0.86), py(0.26), px(-0.46), py(0.30), px(-0.32), py(0.44));

        // 左侧基裂片与左大裂片之间的凹陷
        ctx.bezierCurveTo(px(-0.40), py(0.44), px(-0.48), py(0.46), px(-0.58), py(0.50));
        // 左侧基裂片
        ctx.bezierCurveTo(px(-0.72), py(0.60), px(-0.70), py(0.72), px(-0.54), py(0.62));
        ctx.bezierCurveTo(px(-0.32), py(0.52), px(-0.12), py(0.54), 0, stemAttachY);

        ctx.closePath();

        // 渐变填充
        const grad = ctx.createRadialGradient(-leafScale * 0.18, -leafScale * 0.50, 0, 0, -leafScale * 0.12, leafScale * 1.15);
        grad.addColorStop(0, this.color.light);
        grad.addColorStop(1, this.color.body);
        ctx.fillStyle = grad;
        ctx.fill();

        // 边框
        ctx.strokeStyle = this.color.stroke;
        ctx.lineWidth = 0.5;
        ctx.stroke();

        // 叶脉
        ctx.strokeStyle = this.color.stroke;
        ctx.lineWidth = 0.6;
        ctx.globalAlpha = this.opacity * 0.5;
        ctx.beginPath();
        // 主叶脉
        ctx.moveTo(0, stemAttachY);
        ctx.lineTo(0, topY * 0.92);
        // 放射叶脉
        // 到底部基裂片的叶脉
        ctx.moveTo(0, py(0.50));
        ctx.lineTo(px(0.62), py(0.66));
        ctx.moveTo(0, py(0.50));
        ctx.lineTo(px(-0.62), py(0.66));
        // 到中部大裂片的叶脉
        ctx.moveTo(0, py(0.16));
        ctx.lineTo(px(1.04), py(0.16));
        ctx.moveTo(0, py(0.16));
        ctx.lineTo(px(-1.04), py(0.16));
        ctx.moveTo(0, py(0.02));
        ctx.lineTo(px(0.86), py(-0.30));
        ctx.moveTo(0, py(0.02));
        ctx.lineTo(px(-0.86), py(-0.30));
        ctx.stroke();

        // 叶柄
        ctx.globalAlpha = this.opacity;
        ctx.strokeStyle = this.color.stroke;
        ctx.lineWidth = 1.2;
        ctx.beginPath();
        ctx.moveTo(0, stemAttachY);
        ctx.lineTo(0, stemAttachY + leafScale * 0.40);
        ctx.stroke();

        ctx.restore();
    }
}

export function createMaples(config, canvas) {
    const maples = [];
    const count = config.mapleCount || 10;
    const w = canvas?.width || window.innerWidth;
    const h = canvas?.height || window.innerHeight;

    for (let i = 0; i < count; i++) {
        const maple = new Maple(config, canvas);
        const sectionWidth = w / count;
        maple.x = sectionWidth * i + Math.random() * sectionWidth;
        maple.y = Math.random() * h;
        maples.push(maple);
    }

    return maples;
}
