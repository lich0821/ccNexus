/**
 * Firework Effect - 烟花效果模块
 */

/**
 * 螺旋粒子 - 带二次爆炸
 */
class SpiralSpark {
    constructor(x, y, angle, speed, color, angularSpeed, canExplode = false) {
        this.x = x;
        this.y = y;
        this.color = color;
        this.angle = angle;
        this.angularSpeed = angularSpeed;

        // 初始速度向外
        this.vx = Math.cos(angle) * speed;
        this.vy = Math.sin(angle) * speed;

        this.gravity = 0.04;
        this.friction = 0.98;
        this.life = 1.0;
        this.decay = 0.012;
        this.size = Math.random() * 1 + 0.8;
        this.initialSize = this.size;
        this.trail = [];
        this.maxTrailLength = 15;

        // 二次爆炸
        this.canExplode = canExplode;
        this.explodeTime = Math.random() * 0.2 + 0.4;
        this.hasExploded = false;
    }

    update() {
        this.trail.unshift({ x: this.x, y: this.y, life: this.life });
        if (this.trail.length > this.maxTrailLength) this.trail.pop();

        // 旋转力
        this.angle += this.angularSpeed;
        const rotateForce = 0.15;
        this.vx += Math.cos(this.angle + Math.PI / 2) * rotateForce;
        this.vy += Math.sin(this.angle + Math.PI / 2) * rotateForce;

        // 重力和摩擦
        this.vy += this.gravity;
        this.vx *= this.friction;
        this.vy *= this.friction;

        this.x += this.vx;
        this.y += this.vy;

        this.life -= this.decay;
        this.size = this.initialSize * (0.3 + this.life * 0.7);

        return this.life > 0;
    }

    draw(ctx) {
        if (this.life <= 0) return;
        const { r, g, b } = this.color;
        const alpha = Math.pow(this.life, 0.7);

        // 绘制尾迹 - 更细更淡
        if (this.trail.length > 2) {
            for (let i = 0; i < this.trail.length - 1; i++) {
                const p1 = this.trail[i];
                const p2 = this.trail[i + 1];
                const trailAlpha = alpha * (1 - i / this.trail.length) * 0.5;
                const width = this.size * (1 - i / this.trail.length) * 0.8;

                ctx.beginPath();
                ctx.moveTo(p1.x, p1.y);
                ctx.lineTo(p2.x, p2.y);
                ctx.strokeStyle = `rgba(${r}, ${g}, ${b}, ${trailAlpha})`;
                ctx.lineWidth = width;
                ctx.lineCap = 'round';
                ctx.stroke();
            }
        }

        // 发光效果
        ctx.beginPath();
        ctx.arc(this.x, this.y, this.size * 2, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(${r}, ${g}, ${b}, ${alpha * 0.25})`;
        ctx.fill();

        ctx.beginPath();
        ctx.arc(this.x, this.y, this.size * 0.8, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(255, 255, 255, ${alpha * 0.85})`;
        ctx.fill();
    }

    shouldExplode() {
        if (!this.canExplode || this.hasExploded) return false;
        if (this.life < this.explodeTime) {
            this.hasExploded = true;
            return true;
        }
        return false;
    }

    getSecondaryExplosion() {
        const sparks = [];
        const count = Math.floor(Math.random() * 4 + 5);
        for (let i = 0; i < count; i++) {
            const angle = (Math.PI * 2 / count) * i + Math.random() * 0.3;
            const speed = Math.random() * 1.5 + 1;
            sparks.push(new SpiralSmallSpark(this.x, this.y, angle, speed, this.color));
        }
        return sparks;
    }
}

/**
 * 螺旋小火花 - 二次爆炸产生
 */
class SpiralSmallSpark {
    constructor(x, y, angle, speed, color) {
        this.x = x;
        this.y = y;
        this.color = color;
        this.vx = Math.cos(angle) * speed;
        this.vy = Math.sin(angle) * speed;
        this.gravity = 0.05;
        this.friction = 0.97;
        this.life = 1.0;
        this.decay = 0.025;
        this.size = Math.random() * 0.8 + 0.5;
        this.initialSize = this.size;
        this.trail = [];
        this.maxTrailLength = 10;
    }

    update() {
        this.trail.unshift({ x: this.x, y: this.y });
        if (this.trail.length > this.maxTrailLength) this.trail.pop();
        this.vy += this.gravity;
        this.vx *= this.friction;
        this.vy *= this.friction;
        this.x += this.vx;
        this.y += this.vy;
        this.life -= this.decay;
        this.size = this.initialSize * (0.3 + this.life * 0.7);
        return this.life > 0;
    }

    draw(ctx) {
        if (this.life <= 0) return;
        const { r, g, b } = this.color;
        const alpha = Math.pow(this.life, 0.6);
        if (this.trail.length > 2) {
            ctx.beginPath();
            ctx.moveTo(this.trail[0].x, this.trail[0].y);
            for (let i = 1; i < this.trail.length; i++) {
                ctx.lineTo(this.trail[i].x, this.trail[i].y);
            }
            ctx.strokeStyle = `rgba(${r}, ${g}, ${b}, ${alpha * 0.4})`;
            ctx.lineWidth = this.size * 0.6;
            ctx.lineCap = 'round';
            ctx.stroke();
        }
        ctx.beginPath();
        ctx.arc(this.x, this.y, this.size * 1.5, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(${r}, ${g}, ${b}, ${alpha * 0.2})`;
        ctx.fill();
        ctx.beginPath();
        ctx.arc(this.x, this.y, this.size * 0.5, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(255, 255, 255, ${alpha * 0.8})`;
        ctx.fill();
    }

    shouldExplode() { return false; }
    getSecondaryExplosion() { return []; }
}

/**
 * 交叉星主火花 - 会二次爆炸
 */
class CrossetteSpark {
    constructor(x, y, angle, speed, color) {
        this.x = x;
        this.y = y;
        this.color = color;
        const speedVar = 1 + (Math.random() - 0.5) * 0.3;
        this.vx = Math.cos(angle) * speed * speedVar;
        this.vy = Math.sin(angle) * speed * speedVar;
        this.gravity = 0.04;
        this.friction = 0.985;
        this.life = 1.0;
        this.decay = 0.012;
        this.size = Math.random() * 1.5 + 1.5;
        this.initialSize = this.size;
        this.trail = [];
        this.maxTrailLength = 18;
        this.explodeTime = Math.random() * 0.2 + 0.5;
        this.hasExploded = false;
    }

    update() {
        this.trail.unshift({ x: this.x, y: this.y, life: this.life });
        if (this.trail.length > this.maxTrailLength) this.trail.pop();
        this.vy += this.gravity;
        this.vx *= this.friction;
        this.vy *= this.friction;
        this.x += this.vx;
        this.y += this.vy;
        this.life -= this.decay;
        this.size = this.initialSize * (0.4 + this.life * 0.6);
        return this.life > 0;
    }

    shouldExplode() {
        if (this.hasExploded) return false;
        if (this.life < this.explodeTime) {
            this.hasExploded = true;
            return true;
        }
        return false;
    }

    getSecondaryExplosion() {
        const sparks = [];
        const count = Math.floor(Math.random() * 5 + 6);
        for (let i = 0; i < count; i++) {
            const angle = (Math.PI * 2 / count) * i + Math.random() * 0.4;
            const speed = Math.random() * 1.8 + 1.2;
            sparks.push(new CrossetteSmallSpark(this.x, this.y, angle, speed, this.color));
        }
        return sparks;
    }

    draw(ctx) {
        if (this.life <= 0) return;
        const { r, g, b } = this.color;
        const alpha = Math.pow(this.life, 0.7);
        if (this.trail.length > 2) {
            for (let i = 0; i < this.trail.length - 1; i++) {
                const p1 = this.trail[i];
                const p2 = this.trail[i + 1];
                const trailAlpha = alpha * (1 - i / this.trail.length) * 0.6;
                const width = this.size * (1 - i / this.trail.length);
                ctx.beginPath();
                ctx.moveTo(p1.x, p1.y);
                ctx.lineTo(p2.x, p2.y);
                ctx.strokeStyle = `rgba(${r}, ${g}, ${b}, ${trailAlpha})`;
                ctx.lineWidth = width;
                ctx.lineCap = 'round';
                ctx.stroke();
            }
        }
        ctx.beginPath();
        ctx.arc(this.x, this.y, this.size * 2.5, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(${r}, ${g}, ${b}, ${alpha * 0.25})`;
        ctx.fill();
        ctx.beginPath();
        ctx.arc(this.x, this.y, this.size, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(255, 255, 255, ${alpha * 0.9})`;
        ctx.fill();
    }
}

/**
 * 交叉星小火花 - 二次爆炸产生
 */
class CrossetteSmallSpark {
    constructor(x, y, angle, speed, color) {
        this.x = x;
        this.y = y;
        this.color = color;
        this.vx = Math.cos(angle) * speed;
        this.vy = Math.sin(angle) * speed;
        this.gravity = 0.055;
        this.friction = 0.97;
        this.life = 1.0;
        this.decay = 0.022;
        this.size = Math.random() * 1 + 0.8;
        this.initialSize = this.size;
        this.trail = [];
        this.maxTrailLength = 12;
    }

    update() {
        this.trail.unshift({ x: this.x, y: this.y });
        if (this.trail.length > this.maxTrailLength) this.trail.pop();
        this.vy += this.gravity;
        this.vx *= this.friction;
        this.vy *= this.friction;
        this.x += this.vx;
        this.y += this.vy;
        this.life -= this.decay;
        this.size = this.initialSize * (0.3 + this.life * 0.7);
        return this.life > 0;
    }

    draw(ctx) {
        if (this.life <= 0) return;
        const { r, g, b } = this.color;
        const alpha = Math.pow(this.life, 0.6);
        if (this.trail.length > 2) {
            ctx.beginPath();
            ctx.moveTo(this.trail[0].x, this.trail[0].y);
            for (let i = 1; i < this.trail.length; i++) {
                ctx.lineTo(this.trail[i].x, this.trail[i].y);
            }
            ctx.strokeStyle = `rgba(${r}, ${g}, ${b}, ${alpha * 0.35})`;
            ctx.lineWidth = this.size * 0.7;
            ctx.lineCap = 'round';
            ctx.stroke();
        }
        ctx.beginPath();
        ctx.arc(this.x, this.y, this.size * 1.8, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(${r}, ${g}, ${b}, ${alpha * 0.2})`;
        ctx.fill();
        ctx.beginPath();
        ctx.arc(this.x, this.y, this.size * 0.7, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(255, 255, 255, ${alpha * 0.85})`;
        ctx.fill();
    }
}

/**
 * 烟花火花粒子
 */
class Spark {
    constructor(x, y, angle, speed, color, type, config) {
        this.x = x;
        this.y = y;
        this.color = color;
        this.type = type;
        this.config = config;

        const speedVariation = 1 + (Math.random() - 0.5) * 0.2;
        this.vx = Math.cos(angle) * speed * speedVariation;
        this.vy = Math.sin(angle) * speed * speedVariation;

        this.gravity = type === 'willow' ? 0.06 : 0.045;
        this.friction = type === 'willow' ? 0.985 : 0.975;
        this.life = 1.0;

        if (type === 'willow') {
            this.decay = Math.random() * 0.004 + 0.006;
        } else if (type === 'peony') {
            this.decay = Math.random() * 0.006 + 0.010;
        } else {
            this.decay = Math.random() * 0.008 + 0.012;
        }

        this.initialSize = type === 'star' ? (Math.random() * 1.5 + 1.5) : (Math.random() * 1 + 0.8);
        this.size = this.initialSize;

        this.trail = [];
        this.maxTrailLength = type === 'peony' ? 18 : (type === 'willow' ? 25 : 10);

        this.twinkle = Math.random() * Math.PI * 2;
        this.twinkleSpeed = Math.random() * 0.2 + 0.1;
        this.colorShift = type === 'willow' || type === 'peony';

        this.canExplode = type === 'chrysanthemum' && Math.random() < 0.05;
        this.explodeTime = Math.random() * 0.3 + 0.4;
        this.hasExploded = false;
    }

    update() {
        this.trail.unshift({ x: this.x, y: this.y, life: this.life, size: this.size });
        if (this.trail.length > this.maxTrailLength) {
            this.trail.pop();
        }

        const gravityFactor = 1 + (1 - this.life) * 0.5;
        this.vy += this.gravity * gravityFactor;
        this.vx *= this.friction;
        this.vy *= this.friction;

        this.x += this.vx;
        this.y += this.vy;
        this.life -= this.decay;
        this.size = this.initialSize * (0.3 + this.life * 0.7);
        this.twinkle += this.twinkleSpeed;

        return this.life > 0;
    }

    draw(ctx) {
        if (this.life <= 0) return;

        let { r, g, b } = this.color;

        if (this.colorShift && this.life < 0.6) {
            const shift = (0.6 - this.life) / 0.6;
            r = Math.min(255, r + shift * 50);
            g = Math.max(0, g - shift * 100);
            b = Math.max(0, b - shift * 80);
        }

        const alpha = Math.pow(this.life, 0.7);
        const twinkleFactor = this.type === 'star' ? (0.6 + Math.sin(this.twinkle) * 0.4) : (0.9 + Math.sin(this.twinkle) * 0.1);

        // 绘制尾迹
        if (this.trail.length > 2) {
            ctx.beginPath();
            ctx.moveTo(this.x, this.y);

            for (let i = 0; i < this.trail.length; i++) {
                const point = this.trail[i];
                if (i === 0) {
                    ctx.moveTo(point.x, point.y);
                } else {
                    ctx.lineTo(point.x, point.y);
                }
            }

            const lastPoint = this.trail[this.trail.length - 1];
            const trailGradient = ctx.createLinearGradient(this.x, this.y, lastPoint.x, lastPoint.y);
            trailGradient.addColorStop(0, `rgba(${r}, ${g}, ${b}, ${alpha * 0.7})`);
            trailGradient.addColorStop(0.5, `rgba(${r}, ${g}, ${b}, ${alpha * 0.3})`);
            trailGradient.addColorStop(1, `rgba(${r}, ${g}, ${b}, 0)`);

            ctx.strokeStyle = trailGradient;
            ctx.lineWidth = this.size * 1.2;
            ctx.lineCap = 'round';
            ctx.lineJoin = 'round';
            ctx.stroke();
        }

        // 绘制火花本体
        ctx.save();
        ctx.translate(this.x, this.y);

        const glowSize = this.size * 4;
        const glow = ctx.createRadialGradient(0, 0, 0, 0, 0, glowSize);
        glow.addColorStop(0, `rgba(255, 255, 255, ${alpha * 0.5 * twinkleFactor})`);
        glow.addColorStop(0.15, `rgba(${r}, ${g}, ${b}, ${alpha * 0.4 * twinkleFactor})`);
        glow.addColorStop(0.5, `rgba(${r}, ${g}, ${b}, ${alpha * 0.15})`);
        glow.addColorStop(1, `rgba(${r}, ${g}, ${b}, 0)`);

        ctx.beginPath();
        ctx.arc(0, 0, glowSize, 0, Math.PI * 2);
        ctx.fillStyle = glow;
        ctx.fill();

        ctx.beginPath();
        ctx.arc(0, 0, this.size, 0, Math.PI * 2);
        const coreGradient = ctx.createRadialGradient(0, 0, 0, 0, 0, this.size);
        coreGradient.addColorStop(0, `rgba(255, 255, 255, ${alpha * twinkleFactor})`);
        coreGradient.addColorStop(1, `rgba(${r}, ${g}, ${b}, ${alpha * 0.8})`);
        ctx.fillStyle = coreGradient;
        ctx.fill();

        ctx.restore();
    }

    shouldExplode() {
        if (!this.canExplode || this.hasExploded) return false;
        if (this.life < this.explodeTime) {
            this.hasExploded = true;
            return true;
        }
        return false;
    }

    getSecondaryExplosion() {
        const sparks = [];
        const sparkCount = Math.floor(Math.random() * 4 + 3);

        for (let i = 0; i < sparkCount; i++) {
            const angle = (Math.PI * 2 / sparkCount) * i + Math.random() * 0.5;
            const speed = Math.random() * 0.8 + 0.3;
            sparks.push(new Spark(this.x, this.y, angle, speed, this.color, 'star', this.config));
        }
        return sparks;
    }
}

/**
 * 烟花火箭（上升阶段）
 */
class Rocket {
    constructor(x, targetY, color, burstType, config, canvas) {
        this.x = x;
        this.y = canvas?.height || window.innerHeight;
        this.targetY = targetY;
        this.color = color;
        this.burstType = burstType;
        this.config = config;

        this.vy = -(Math.random() * 2 + 4);
        this.vx = (Math.random() - 0.5) * 0.5;

        this.trail = [];
        this.maxTrailLength = 15;
        this.exploded = false;
        this.flameColor = { r: 255, g: 180, b: 80 };
    }

    update() {
        this.trail.unshift({ x: this.x, y: this.y });
        if (this.trail.length > this.maxTrailLength) {
            this.trail.pop();
        }

        this.vy *= 0.99;
        this.y += this.vy;
        this.x += this.vx;

        if (this.y <= this.targetY || this.vy > -1) {
            this.exploded = true;
        }

        return !this.exploded;
    }

    draw(ctx) {
        if (this.trail.length > 1) {
            for (let i = 0; i < this.trail.length - 1; i++) {
                const point = this.trail[i];
                const nextPoint = this.trail[i + 1];
                const alpha = 1 - (i / this.trail.length);
                const width = (1 - i / this.trail.length) * 3;

                ctx.beginPath();
                ctx.moveTo(point.x, point.y);
                ctx.lineTo(nextPoint.x, nextPoint.y);

                const { r, g, b } = this.flameColor;
                ctx.strokeStyle = `rgba(${r}, ${g}, ${b}, ${alpha * 0.8})`;
                ctx.lineWidth = width;
                ctx.lineCap = 'round';
                ctx.stroke();
            }
        }

        ctx.save();
        ctx.translate(this.x, this.y);

        const glow = ctx.createRadialGradient(0, 0, 0, 0, 0, 8);
        glow.addColorStop(0, 'rgba(255, 255, 255, 0.9)');
        glow.addColorStop(0.3, 'rgba(255, 200, 100, 0.6)');
        glow.addColorStop(1, 'rgba(255, 150, 50, 0)');

        ctx.beginPath();
        ctx.arc(0, 0, 8, 0, Math.PI * 2);
        ctx.fillStyle = glow;
        ctx.fill();

        ctx.beginPath();
        ctx.arc(0, 0, 2, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(255, 255, 255, 1)';
        ctx.fill();

        ctx.restore();
    }
}

/**
 * 烟花主类
 */
export class Firework {
    constructor(config, canvas) {
        this.config = config;
        this.canvas = canvas;
        this.width = canvas?.width || window.innerWidth;
        this.height = canvas?.height || window.innerHeight;

        this.launchX = Math.random() * this.width * 0.8 + this.width * 0.1;
        this.burstY = Math.random() * this.height * 0.4 + this.height * 0.1;

        const types = ['chrysanthemum', 'peony', 'willow', 'star'];
        this.burstType = types[Math.floor(Math.random() * types.length)];
        this.color = this.getSpringFestivalColor();

        this.rocket = new Rocket(this.launchX, this.burstY, this.color, this.burstType, config, canvas);
        this.sparks = [];
        this.phase = 'launch';
        this.life = 1.0;

        this.isSpecial = true;
        const specialTypes = [
            { type: 'double', weight: 10 },  //双层
            { type: 'ring', weight: 10 },    //环形
            { type: 'starShape', weight: 10 },   //星形
            { type: 'heart', weight: 10 },    //心形
            { type: 'dahlia', weight: 10 },  //菊冠
            { type: 'crossette', weight: 10 },   //交叉星
            { type: 'phoenix', weight: 10 },  //凤凰展翅
            { type: 'saturn', weight: 10 },  //土星环
            { type: 'peonyFlower', weight: 10 },  //牡丹花
            { type: 'spiral', weight: 10 }   //旋转
        ];
        const totalWeight = specialTypes.reduce((sum, item) => sum + item.weight, 0);
        let random = Math.random() * totalWeight;
        for (const item of specialTypes) {
            random -= item.weight;
            if (random <= 0) {
                this.specialType = item.type;
                break;
            }
        }
    }

    getSpringFestivalColor() {
        const colors = [
            { r: 255, g: 50, b: 50 },
            { r: 255, g: 80, b: 80 },
            { r: 255, g: 215, b: 0 },
            { r: 255, g: 180, b: 50 },
            { r: 255, g: 100, b: 150 },
            { r: 255, g: 255, b: 100 },
            { r: 150, g: 255, b: 150 },
            { r: 100, g: 200, b: 255 },
            { r: 200, g: 100, b: 255 },
            { r: 255, g: 255, b: 255 },
        ];

        const weights = [20, 15, 20, 15, 8, 8, 5, 4, 3, 2];
        const totalWeight = weights.reduce((a, b) => a + b, 0);
        let random = Math.random() * totalWeight;

        for (let i = 0; i < colors.length; i++) {
            random -= weights[i];
            if (random <= 0) {
                return colors[i];
            }
        }
        return colors[0];
    }

    createBurst() {
        const x = this.rocket.x;
        const y = this.rocket.y;

        switch (this.specialType) {
            case 'double': this.createDoubleBurst(x, y); break;
            case 'ring': this.createRingBurst(x, y); break;
            case 'heart': this.createHeartBurst(x, y); break;
            case 'starShape': this.createStarShapeBurst(x, y); break;
            case 'dahlia': this.createDahliaBurst(x, y); break;
            case 'crossette': this.createCrossetteBurst(x, y); break;
            case 'phoenix': this.createPhoenixBurst(x, y); break;
            case 'saturn': this.createSaturnBurst(x, y); break;
            case 'peonyFlower': this.createPeonyFlowerBurst(x, y); break;
            case 'spiral': this.createSpiralBurst(x, y); break;
            default: this.createDoubleBurst(x, y);
        }
    }

    createDoubleBurst(x, y) {
        const innerColor = this.getSpringFestivalColor();
        const innerCount = Math.floor(Math.random() * 15 + 25);
        const outerCount = Math.floor(Math.random() * 20 + 30);

        for (let i = 0; i < innerCount; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 1.5 + 1.5;
            this.sparks.push(new Spark(x, y, angle, speed, innerColor, 'star', this.config));
        }

        for (let i = 0; i < outerCount; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 2 + 3;
            this.sparks.push(new Spark(x, y, angle, speed, this.color, 'peony', this.config));
        }
    }

    createRingBurst(x, y) {
        const ringCount = Math.floor(Math.random() * 20 + 35);
        const centerCount = Math.floor(Math.random() * 10 + 15);

        for (let i = 0; i < ringCount; i++) {
            const angle = (Math.PI * 2 / ringCount) * i;
            const speed = 3.5 + Math.random() * 0.5;
            this.sparks.push(new Spark(x, y, angle, speed, this.color, 'peony', this.config));
        }

        for (let i = 0; i < centerCount; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 1.5;
            this.sparks.push(new Spark(x, y, angle, speed, { r: 255, g: 255, b: 255 }, 'star', this.config));
        }
    }

    createHeartBurst(x, y) {
        const outerCount = Math.floor(Math.random() * 30 + 60);
        for (let i = 0; i < outerCount; i++) {
            const t = (i / outerCount) * Math.PI * 2;
            const heartX = 16 * Math.pow(Math.sin(t), 3);
            const heartY = -(13 * Math.cos(t) - 5 * Math.cos(2 * t) - 2 * Math.cos(3 * t) - Math.cos(4 * t));

            const angle = Math.atan2(heartY, heartX);
            const distance = Math.sqrt(heartX * heartX + heartY * heartY);
            const speed = distance * 0.28 + Math.random() * 0.4;

            const heartColor = Math.random() < 0.7 ?
                { r: 255, g: 50 + Math.random() * 50, b: 80 + Math.random() * 50 } :
                { r: 255, g: 150 + Math.random() * 50, b: 180 + Math.random() * 50 };

            this.sparks.push(new Spark(x, y, angle, speed, heartColor, 'peony', this.config));
        }

        const innerCount = Math.floor(Math.random() * 20 + 30);
        for (let i = 0; i < innerCount; i++) {
            const t = (i / innerCount) * Math.PI * 2;
            const heartX = 16 * Math.pow(Math.sin(t), 3);
            const heartY = -(13 * Math.cos(t) - 5 * Math.cos(2 * t) - 2 * Math.cos(3 * t) - Math.cos(4 * t));

            const angle = Math.atan2(heartY, heartX);
            const distance = Math.sqrt(heartX * heartX + heartY * heartY);
            const speed = distance * 0.18 + Math.random() * 0.3;

            const innerColor = { r: 255, g: 100 + Math.random() * 80, b: 120 + Math.random() * 60 };
            this.sparks.push(new Spark(x, y, angle, speed, innerColor, 'star', this.config));
        }

        for (let i = 0; i < 20; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 1.5 + 0.5;
            this.sparks.push(new Spark(x, y, angle, speed, { r: 255, g: 255, b: 255 }, 'star', this.config));
        }
    }

    createStarShapeBurst(x, y) {
        const points = Math.random() < 0.5 ? 5 : 6;
        const count = Math.floor(Math.random() * 20 + 40);

        // 星角
        for (let i = 0; i < points; i++) {
            const baseAngle = (Math.PI * 2 / points) * i - Math.PI / 2;

            for (let j = 0; j < 8; j++) {
                const angle = baseAngle + (Math.random() - 0.5) * 0.1545;
                const speed = 1.545 + j * 0.412 + Math.random() * 0.309;
                this.sparks.push(new Spark(x, y, angle, speed, this.color, 'peony', this.config));
            }
        }

        // 内部填充
        for (let i = 0; i < count * 0.5; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 2.06 + 1.03;
            const innerColor = this.getSpringFestivalColor();
            this.sparks.push(new Spark(x, y, angle, speed, innerColor, 'star', this.config));
        }
    }

    createDahliaBurst(x, y) {
        const rayCount = Math.floor(Math.random() * 8 + 12);

        for (let i = 0; i < rayCount; i++) {
            const baseAngle = (Math.PI * 2 / rayCount) * i;

            for (let j = 0; j < 6; j++) {
                const angle = baseAngle + (Math.random() - 0.5) * 0.1;
                const speed = 2 + j * 0.5;

                const colorFade = j / 6;
                const fadedColor = {
                    r: Math.floor(this.color.r + (255 - this.color.r) * (1 - colorFade)),
                    g: Math.floor(this.color.g + (255 - this.color.g) * (1 - colorFade) * 0.5),
                    b: Math.floor(this.color.b)
                };

                this.sparks.push(new Spark(x, y, angle, speed, fadedColor, 'chrysanthemum', this.config));
            }
        }

        for (let i = 0; i < 20; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 1.5;
            this.sparks.push(new Spark(x, y, angle, speed, { r: 255, g: 220, b: 100 }, 'star', this.config));
        }
    }

    // 交叉星烟花
    createCrossetteBurst(x, y) {
        this.crossetteSparks = [];
        const mainCount = Math.floor(Math.random() * 8 + 18);
        for (let i = 0; i < mainCount; i++) {
            const angle = (Math.PI * 2 / mainCount) * i + (Math.random() - 0.5) * 0.15;
            const speed = Math.random() * 1.5 + 2.8;
            const color = Math.random() < 0.8 ? this.color : this.getSpringFestivalColor();
            this.crossetteSparks.push(new CrossetteSpark(x, y, angle, speed, color));
        }
        for (let i = 0; i < 12; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 1.5 + 0.5;
            this.crossetteSparks.push(new CrossetteSmallSpark(x, y, angle, speed, { r: 255, g: 255, b: 255 }));
        }
        this.isCrossette = true;
    }

    // 凤凰展翅烟花 - 使用Spark类保持烟花效果
    createPhoenixBurst(x, y) {
        // 左翼 - 向左上弧形展开
        for (let i = 0; i < 15; i++) {
            const t = i / 14;
            const angle = -Math.PI * 0.85 + t * Math.PI * 0.5;
            const speed = 3.5 + t * 2 + Math.random() * 0.8;
            const color = { r: 255, g: Math.floor(100 + t * 100), b: Math.floor(30 + t * 20) };
            this.sparks.push(new Spark(x, y, angle, speed, color, 'willow', this.config));
        }

        // 右翼 - 向右上弧形展开
        for (let i = 0; i < 15; i++) {
            const t = i / 14;
            const angle = -Math.PI * 0.15 - t * Math.PI * 0.5;
            const speed = 3.5 + t * 2 + Math.random() * 0.8;
            const color = { r: 255, g: Math.floor(100 + t * 100), b: Math.floor(30 + t * 20) };
            this.sparks.push(new Spark(x, y, angle, speed, color, 'willow', this.config));
        }

        // 尾羽 - 向下飘散
        for (let i = 0; i < 12; i++) {
            const angle = Math.PI * 0.5 + (Math.random() - 0.5) * 0.6;
            const speed = Math.random() * 2 + 3;
            const color = { r: 255, g: Math.floor(Math.random() * 60 + 50), b: Math.floor(Math.random() * 40 + 30) };
            this.sparks.push(new Spark(x, y, angle, speed, color, 'willow', this.config));
        }

        // 中心爆炸 - 金色火花
        for (let i = 0; i < 25; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 2.5 + 1.5;
            this.sparks.push(new Spark(x, y, angle, speed, { r: 255, g: 200, b: 50 }, 'star', this.config));
        }
    }

    // 土星环烟花
    createSaturnBurst(x, y) {
        // 土星球体 - 中心金黄色爆炸
        for (let i = 0; i < 30; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 2.5 + 1.5;
            const color = { r: 255, g: Math.floor(180 + Math.random() * 40), b: Math.floor(50 + Math.random() * 30) };
            this.sparks.push(new Spark(x, y, angle, speed, color, 'peony', this.config));
        }

        // 土星环 - 倾斜椭圆形（约30度倾斜）
        const tilt = 0.3; // 倾斜系数
        for (let i = 0; i < 40; i++) {
            const t = (i / 40) * Math.PI * 2;
            const ringX = Math.cos(t);
            const ringY = Math.sin(t) * tilt;
            const angle = Math.atan2(ringY, ringX);
            const speed = 4 + Math.random() * 1;
            const color = { r: 200 + Math.floor(Math.random() * 55), g: 220 + Math.floor(Math.random() * 35), b: 255 };
            this.sparks.push(new Spark(x, y, angle, speed, color, 'star', this.config));
        }

        // 外环 - 更大更淡
        for (let i = 0; i < 30; i++) {
            const t = (i / 30) * Math.PI * 2;
            const ringX = Math.cos(t);
            const ringY = Math.sin(t) * tilt;
            const angle = Math.atan2(ringY, ringX);
            const speed = 5.5 + Math.random() * 1;
            const color = { r: 180 + Math.floor(Math.random() * 40), g: 200 + Math.floor(Math.random() * 30), b: 230 + Math.floor(Math.random() * 25) };
            this.sparks.push(new Spark(x, y, angle, speed, color, 'star', this.config));
        }
    }

    // 牡丹绽放烟花
    createPeonyFlowerBurst(x, y) {
        // 内层花瓣 - 深红/粉色
        for (let i = 0; i < 12; i++) {
            const angle = (i / 12) * Math.PI * 2;
            const speed = 2 + Math.random() * 0.5;
            const color = { r: 255, g: Math.floor(50 + Math.random() * 60), b: Math.floor(80 + Math.random() * 50) };
            this.sparks.push(new Spark(x, y, angle, speed, color, 'peony', this.config));
        }

        // 中层花瓣 - 粉红色，错开角度
        for (let i = 0; i < 18; i++) {
            const angle = (i / 18) * Math.PI * 2 + Math.PI / 18;
            const speed = 3.2 + Math.random() * 0.6;
            const color = { r: 255, g: Math.floor(100 + Math.random() * 80), b: Math.floor(120 + Math.random() * 60) };
            this.sparks.push(new Spark(x, y, angle, speed, color, 'peony', this.config));
        }

        // 外层花瓣 - 浅粉/白色
        for (let i = 0; i < 24; i++) {
            const angle = (i / 24) * Math.PI * 2;
            const speed = 4.5 + Math.random() * 0.8;
            const color = { r: 255, g: Math.floor(180 + Math.random() * 50), b: Math.floor(200 + Math.random() * 40) };
            this.sparks.push(new Spark(x, y, angle, speed, color, 'peony', this.config));
        }

        // 花蕊 - 金黄色中心
        for (let i = 0; i < 15; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 1.5 + 0.8;
            this.sparks.push(new Spark(x, y, angle, speed, { r: 255, g: 220, b: 50 }, 'star', this.config));
        }
    }

    // 螺旋烟花 - 带二次爆炸
    createSpiralBurst(x, y) {
        const colors = [
            { r: 255, g: 50, b: 50 },    // 红
            { r: 255, g: 200, b: 50 },   // 金
            { r: 50, g: 255, b: 150 },   // 青
            { r: 150, g: 100, b: 255 }   // 紫
        ];

        // 内层 - 均匀分布，带旋转
        for (let i = 0; i < 12; i++) {
            const angle = (i / 12) * Math.PI * 2;
            const speed = 2 + Math.random() * 0.5;
            const color = colors[i % colors.length];
            this.sparks.push(new SpiralSpark(x, y, angle, speed, color, 0.05, false));
        }

        // 中层 - 错开角度，带旋转
        for (let i = 0; i < 18; i++) {
            const angle = (i / 18) * Math.PI * 2 + Math.PI / 18;
            const speed = 3.2 + Math.random() * 0.6;
            const color = colors[i % colors.length];
            this.sparks.push(new SpiralSpark(x, y, angle, speed, color, 0.04, false));
        }

        // 外层 - 普通烟花粒子
        for (let i = 0; i < 24; i++) {
            const angle = (i / 24) * Math.PI * 2;
            const speed = 4.5 + Math.random() * 0.8;
            const color = colors[i % colors.length];
            const canExplode = Math.random() < 0.9; // 90%概率二次爆炸
            this.sparks.push(new SpiralSpark(x, y, angle, speed, color, 0.03, canExplode));
        }

        // 花蕊 - 金黄色中心
        for (let i = 0; i < 15; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = Math.random() * 1.5 + 0.8;
            this.sparks.push(new Spark(x, y, angle, speed, { r: 255, g: 220, b: 50 }, 'star', this.config));
        }
    }

    update() {
        if (this.phase === 'launch') {
            const alive = this.rocket.update();
            if (!alive) {
                this.phase = 'burst';
                this.createBurst();
            }
        } else if (this.phase === 'burst' || this.phase === 'fade') {
            this.phase = 'fade';

            // 交叉星效果更新
            if (this.isCrossette && this.crossetteSparks) {
                const newSparks = [];
                this.crossetteSparks = this.crossetteSparks.filter(spark => {
                    const alive = spark.update();
                    if (spark instanceof CrossetteSpark && spark.shouldExplode()) {
                        newSparks.push(...spark.getSecondaryExplosion());
                    }
                    return alive;
                });
                this.crossetteSparks.push(...newSparks);
                if (this.crossetteSparks.length === 0) {
                    this.life = 0;
                }
            } else {
                const newSparks = [];
                this.sparks = this.sparks.filter(spark => {
                    const alive = spark.update();

                    if (spark.shouldExplode()) {
                        newSparks.push(...spark.getSecondaryExplosion());
                    }

                    return alive;
                });

                this.sparks.push(...newSparks);

                if (this.sparks.length === 0) {
                    this.life = 0;
                }
            }
        }

        return this.life > 0;
    }

    draw(ctx) {
        if (!ctx) return;

        if (this.phase === 'launch') {
            this.rocket.draw(ctx);
        } else {
            // 交叉星效果绘制
            if (this.isCrossette && this.crossetteSparks) {
                for (const spark of this.crossetteSparks) {
                    spark.draw(ctx);
                }
            } else {
                for (const spark of this.sparks) {
                    spark.draw(ctx);
                }
            }
        }
    }
}

/**
 * 创建烟花数组
 */
export function createFireworks(config, canvas, count = 2) {
    const fireworks = [];
    for (let i = 0; i < count; i++) {
        fireworks.push(new Firework(config, canvas));
    }
    return fireworks;
}
