/**
 * Snow Effect - 雪花效果模块
 */

/**
 * 雪花粒子类
 */
export class Snowflake {
    constructor(config, canvas) {
        this.config = config;
        this.canvas = canvas;
        this.reset();
    }

    reset() {
        const { particleCount, speed, wind, opacity } = this.config;

        // 随机初始位置
        this.x = Math.random() * (this.canvas?.width || window.innerWidth);
        this.y = Math.random() * -100;

        // 随机大小 (2-8px)
        const sizeRand = Math.random();
        if (sizeRand < 0.6) {
            this.size = Math.random() * 2 + 2;
        } else if (sizeRand < 0.9) {
            this.size = Math.random() * 2 + 4;
        } else {
            this.size = Math.random() * 3 + 6;
        }

        // 随机速度
        const speedFactor = 0.5 + (this.size / 10);
        this.speedY = (Math.random() * 0.8 + 0.3) * speed * speedFactor;
        this.speedX = (Math.random() - 0.5) * wind * 2;

        // 随机透明度
        this.opacity = (Math.random() * 0.4 + 0.4) * opacity * (1 - this.size / 20);

        // 摇摆参数
        this.swing = Math.random() * Math.PI * 2;
        this.swingSpeed = Math.random() * 0.02 + 0.01;
        this.swingRadius = Math.random() * 1.5 + 0.5;

        // 旋转参数
        this.rotation = Math.random() * Math.PI * 2;
        this.rotationSpeed = (Math.random() - 0.5) * 0.03;

        // 雪花类型
        const typeRand = Math.random();
        if (this.size < 4) {
            this.type = 0;
        } else if (typeRand < 0.5) {
            this.type = 1;
        } else if (typeRand < 0.8) {
            this.type = 2;
        } else {
            this.type = 0;
        }

        this.branches = this.type === 1 ? 6 : (this.type === 2 ? 8 : 0);
    }

    update() {
        this.swing += this.swingSpeed;
        this.rotation += this.rotationSpeed;
        this.y += this.speedY;
        this.x += this.speedX + Math.sin(this.swing) * this.swingRadius;

        if (this.y > (this.canvas?.height || window.innerHeight) + 10) {
            this.y = -10;
            this.x = Math.random() * (this.canvas?.width || window.innerWidth);
        }

        if (this.x > (this.canvas?.width || window.innerWidth) + 10) {
            this.x = -10;
        } else if (this.x < -10) {
            this.x = (this.canvas?.width || window.innerWidth) + 10;
        }
    }

    draw(ctx) {
        if (!ctx) return;

        ctx.save();
        ctx.translate(this.x, this.y);
        ctx.rotate(this.rotation);

        if (this.type === 0) {
            this.drawGlowDot(ctx);
        } else if (this.type === 1) {
            this.drawCrystal(ctx);
        } else {
            this.drawStar(ctx);
        }

        ctx.restore();
    }

    drawGlowDot(ctx) {
        const gradient = ctx.createRadialGradient(0, 0, 0, 0, 0, this.size);
        gradient.addColorStop(0, `rgba(255, 255, 255, ${this.opacity})`);
        gradient.addColorStop(0.4, `rgba(255, 255, 255, ${this.opacity * 0.6})`);
        gradient.addColorStop(1, `rgba(255, 255, 255, 0)`);

        ctx.beginPath();
        ctx.arc(0, 0, this.size, 0, Math.PI * 2);
        ctx.fillStyle = gradient;
        ctx.fill();
    }

    drawCrystal(ctx) {
        const armLength = this.size;
        const branchLength = armLength * 0.4;

        ctx.strokeStyle = `rgba(255, 255, 255, ${this.opacity})`;
        ctx.lineWidth = Math.max(1, this.size / 5);
        ctx.lineCap = 'round';

        for (let i = 0; i < 6; i++) {
            const angle = (Math.PI / 3) * i;

            ctx.save();
            ctx.rotate(angle);

            ctx.beginPath();
            ctx.moveTo(0, 0);
            ctx.lineTo(armLength, 0);
            ctx.stroke();

            const branchPos = armLength * 0.65;
            ctx.beginPath();
            ctx.moveTo(branchPos, 0);
            ctx.lineTo(branchPos + branchLength * 0.7, -branchLength * 0.5);
            ctx.moveTo(branchPos, 0);
            ctx.lineTo(branchPos + branchLength * 0.7, branchLength * 0.5);
            ctx.stroke();

            const smallBranchPos = armLength * 0.35;
            ctx.beginPath();
            ctx.moveTo(smallBranchPos, 0);
            ctx.lineTo(smallBranchPos + branchLength * 0.4, -branchLength * 0.3);
            ctx.moveTo(smallBranchPos, 0);
            ctx.lineTo(smallBranchPos + branchLength * 0.4, branchLength * 0.3);
            ctx.stroke();

            ctx.restore();
        }

        const centerGradient = ctx.createRadialGradient(0, 0, 0, 0, 0, this.size * 0.3);
        centerGradient.addColorStop(0, `rgba(255, 255, 255, ${this.opacity * 0.8})`);
        centerGradient.addColorStop(1, `rgba(255, 255, 255, 0)`);
        ctx.beginPath();
        ctx.arc(0, 0, this.size * 0.3, 0, Math.PI * 2);
        ctx.fillStyle = centerGradient;
        ctx.fill();
    }

    drawStar(ctx) {
        const outerRadius = this.size;
        const innerRadius = this.size * 0.4;
        const points = 8;

        ctx.beginPath();
        for (let i = 0; i < points * 2; i++) {
            const radius = i % 2 === 0 ? outerRadius : innerRadius;
            const angle = (Math.PI / points) * i - Math.PI / 2;
            const x = Math.cos(angle) * radius;
            const y = Math.sin(angle) * radius;

            if (i === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        }
        ctx.closePath();

        const gradient = ctx.createRadialGradient(0, 0, 0, 0, 0, outerRadius);
        gradient.addColorStop(0, `rgba(255, 255, 255, ${this.opacity})`);
        gradient.addColorStop(0.5, `rgba(240, 248, 255, ${this.opacity * 0.7})`);
        gradient.addColorStop(1, `rgba(220, 240, 255, ${this.opacity * 0.3})`);

        ctx.fillStyle = gradient;
        ctx.fill();
    }
}

/**
 * 创建雪花粒子数组
 */
export function createSnowParticles(config, canvas) {
    const particles = [];
    for (let i = 0; i < config.particleCount; i++) {
        const snowflake = new Snowflake(config, canvas);
        snowflake.y = Math.random() * (canvas?.height || window.innerHeight);
        particles.push(snowflake);
    }
    return particles;
}
