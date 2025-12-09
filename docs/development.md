# 开发指南

## 环境准备

- Go 1.22+
- Node.js 18+
- Wails CLI v2

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 检查环境依赖
wails doctor
```

## 开发模式

```bash
# 安装前端依赖
cd frontend && npm install && cd ..

# 启动开发模式（支持热重载）
wails dev
```

## 构建发布

```bash
npm run build           # 当前平台
npm run build:prod      # 生产环境优化
npm run build:windows   # Windows
npm run build:macos     # macOS
npm run build:linux     # Linux
```

构建产物位于 `build/bin/` 目录。

## 项目结构

```
ccNexus/
├── main.go                 # 应用入口
├── app.go                  # 核心应用逻辑
├── internal/
│   ├── proxy/              # HTTP 代理核心
│   ├── transformer/        # API 格式转换器
│   ├── storage/            # SQLite 数据存储
│   ├── config/             # 配置管理
│   ├── webdav/             # WebDAV 同步
│   ├── logger/             # 日志系统
│   └── tray/               # 系统托盘
└── frontend/               # 前端代码
    ├── src/modules/        # 功能模块
    ├── src/i18n/           # 国际化
    └── src/themes/         # 主题样式
```
