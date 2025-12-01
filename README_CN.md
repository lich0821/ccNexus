# ccNexus

<div align="center">

**Claude Code 智能端点轮换代理**

[![构建状态](https://github.com/lich0821/ccNexus/workflows/Build%20and%20Release/badge.svg)](https://github.com/lich0821/ccNexus/actions)
[![许可证: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 版本](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2-blue)](https://wails.io/)

[English](README.md) | [简体中文](README_CN.md)

</div>

## 📖 项目简介

ccNexus 是一个专为 Claude Code 设计的智能 API 端点轮换代理工具。它可以帮助你管理多个 API 端点，实现自动故障转移、负载均衡，并支持将 Claude API 请求转换为 OpenAI 或 Gemini 格式，让你能够使用各种兼容的 API 服务。

### 为什么需要 ccNexus？

- **多端点管理**：同时配置多个 API 端点，一个失败自动切换到下一个
- **API 格式转换**：支持 Claude、OpenAI、Gemini 三种 API 格式互转
- **使用统计**：实时监控请求数、错误数、Token 用量
- **数据安全**：所有数据本地存储，安全可靠

## 📸 应用界面

<p align="center">
  <img src="docs/images/CN-Light.png" alt="明亮主题" width="45%">
  <br/>默认主题
</p>
<p align="center">
  <img src="docs/images/CN-Dark.png" alt="暗黑主题" width="45%">
  <br/>暗黑主题
</p>

## 📖 获取帮助

<p align="center">
  <img src="frontend/public/chat.jpg" alt="微信群" width="45%">
  <br/>问题反馈请加群
</p>
<p align="center">
  <img src="docs/images/ME.png" alt="个人微信" width="45%">
  <br/>若群聊过期，请加好友拉你入群
</p>

## ✨ 功能特性

### 核心功能

| 功能 | 说明 |
|------|------|
| 🔄 **自动端点轮换** | 请求失败时自动切换到下一个可用端点，实现无缝故障转移 |
| 🔀 **多格式支持** | 支持 Claude、OpenAI 和 Gemini API 格式转换 |
| 🔁 **智能重试** | 自动重试失败请求，最多重试 `端点数 × 2` 次 |
| 📊 **实时统计** | 监控请求数、错误数、Token 使用量 |
| 📈 **历史数据** | 基于 SQLite 的统计数据存储，支持按月查看历史归档 |
| ☁️ **WebDAV 同步** | 支持通过 WebDAV 在多设备间同步配置和统计数据 |

### 界面功能

| 功能 | 说明 |
|------|------|
| 🖥️ **跨平台桌面应用** | 支持 Windows、macOS、Linux |
| 🎨 **多主题支持** | 12 种主题可选：默认、深色、护眼、星空、樱花粉、暖阳橙、海洋蓝、摩卡棕、赛博朋克、暗夜极光、全息蓝、量子紫 |
| 🌙 **自动主题切换** | 根据时间自动在浅色和深色主题间切换（7:00-19:00 浅色） |
| 🌐 **中英文界面** | 支持中文和英文界面切换 |
| 📋 **系统托盘** | 支持最小化到系统托盘运行 |
| 📝 **实时日志** | 查看代理运行日志，支持按级别过滤 |

## 🚀 快速开始

### 下载安装

[📥 下载最新版本](https://github.com/lich0821/ccNexus/releases/latest)

#### Windows
1. 下载 `ccNexus-windows-amd64.zip`
2. 解压到任意目录
3. 双击运行 `ccNexus.exe`

#### macOS
1. 下载 `ccNexus-darwin-amd64.zip` 或 `ccNexus-darwin-arm64.zip`（M 系列芯片）
2. 解压后将 `ccNexus.app` 移动到「应用程序」文件夹
3. 首次运行：右键点击 → 打开（绕过 Gatekeeper）

#### Linux
```bash
tar -xzf ccNexus-linux-amd64.tar.gz
./ccNexus
```

### 配置步骤

#### 1. 添加 API 端点

点击界面上的「添加端点」按钮，填写以下信息：

| 字段 | 说明 | 示例 |
|------|------|------|
| **名称** | 端点的友好名称 | `Claude 官方` |
| **API 地址** | API 服务器地址 | `https://api.anthropic.com` |
| **API 密钥** | 你的 API 密钥 | `sk-ant-api03-...` |
| **转换器** | API 格式类型 | `claude` / `openai` / `gemini` |
| **模型** | 目标模型（非 Claude 必填） | `gpt-4-turbo` / `gemini-pro` |
| **备注** | 可选的备注说明 | `主力端点` |

#### 2. 配置 Claude Code

在 Claude Code 的配置文件 settings.json 中设置以下参数（默认在系统的用户目录下）：

```
API Base URL: http://localhost:3000
API Key: 任意值（代理会使用端点配置的密钥）
```

#### 3. 开始使用

配置完成后，Claude Code 的所有请求都会通过 ccNexus 代理转发到你配置的端点。

## 📖 工作原理

```
┌─────────────┐     ┌─────────────────────────────────────────────────┐
│ Claude Code │────▶│              ccNexus 代理                        │
└─────────────┘     │  localhost:3000                                  │
                    │                                                  │
                    │  ┌─────────────┐   失败    ┌─────────────┐      │
                    │  │  端点 #1    │─────────▶│  端点 #2    │      │
                    │  │  (Claude)   │          │  (OpenAI)   │      │
                    │  └─────────────┘          └─────────────┘      │
                    │         │                        │              │
                    │         │ 成功                   │ 成功         │
                    │         ▼                        ▼              │
                    │  ┌─────────────────────────────────────┐       │
                    │  │           返回响应给 Claude Code     │       │
                    │  └─────────────────────────────────────┘       │
                    └─────────────────────────────────────────────────┘
```

**工作流程：**
1. Claude Code 发送请求到本地代理（默认端口 3000）
2. 代理按顺序尝试已启用的端点
3. 如果当前端点失败，自动切换到下一个端点重试
4. 根据端点配置的转换器，自动转换请求/响应格式
5. 返回成功响应给 Claude Code

## 🔧 详细配置

### 应用设置

| 设置项 | 说明 | 默认值 |
|--------|------|--------|
| **代理端口** | 本地代理监听端口 | `3000` |
| **日志级别** | 0=调试, 1=信息, 2=警告, 3=错误 | `1` (信息) |
| **界面语言** | 中文 / English | `zh-CN` |
| **主题** | 12 种主题可选 | `light` |
| **自动主题** | 根据时间自动切换主题 | 关闭 |
| **窗口关闭行为** | 直接关闭 / 最小化到托盘 / 每次询问 | 每次询问 |

### 端点配置详解

#### 转换器类型

| 转换器 | 说明 | 模型字段 |
|--------|------|----------|
| `claude` | Claude 原生 API（直通） | 可选（覆盖请求中的模型） |
| `openai` | OpenAI 兼容 API | **必填**（如 `gpt-4-turbo`） |
| `gemini` | Google Gemini API | **必填**（如 `gemini-pro`） |

#### 配置示例

**Claude 官方端点：**
```json
{
  "name": "Claude 官方",
  "apiUrl": "https://api.anthropic.com",
  "apiKey": "sk-ant-api03-xxx",
  "enabled": true,
  "transformer": "claude"
}
```

**OpenAI 兼容端点：**
```json
{
  "name": "OpenAI 代理",
  "apiUrl": "https://api.openai.com",
  "apiKey": "sk-xxx",
  "enabled": true,
  "transformer": "openai",
  "model": "gpt-4-turbo"
}
```

**Gemini 端点：**
```json
{
  "name": "Gemini",
  "apiUrl": "https://generativelanguage.googleapis.com",
  "apiKey": "AIza-xxx",
  "enabled": true,
  "transformer": "gemini",
  "model": "gemini-pro"
}
```

### WebDAV 云同步

ccNexus 支持通过 WebDAV 协议同步配置和统计数据，兼容以下服务：
- 坚果云
- NextCloud
- ownCloud
- 其他标准 WebDAV 服务

**配置步骤：**
1. 点击界面上的「WebDAV 云备份」
2. 填写 WebDAV 服务器地址、用户名、密码
3. 点击「测试连接」确认配置正确
4. 使用「备份」和「恢复」功能管理数据

## 📊 统计功能

### 统计维度

| 维度 | 说明 |
|------|------|
| **今日** | 当天的请求统计 |
| **昨日** | 昨天的请求统计 |
| **本周** | 本周的累计统计 |
| **本月** | 本月的累计统计 |
| **历史** | 按月查看历史归档数据 |

### 统计指标

- **请求数**：成功和失败的请求总数
- **错误数**：失败的请求数量
- **Token 数**：输入和输出的 Token 用量（估算值）
- **成功率**：请求成功的百分比

## 🛠️ 开发指南

### 环境要求

- Go 1.22+
- Node.js 18+
- Wails CLI v2

### 开发运行

```bash
# 克隆项目
git clone https://github.com/lich0821/ccNexus.git
cd ccNexus

# 开发模式运行（自动安装依赖）
node run.mjs
```

### 构建发布

```bash
# 当前平台构建
npm run build

# 优化构建（生产环境）
npm run build:prod

# 指定平台构建
npm run build:windows    # Windows
npm run build:macos      # macOS
npm run build:linux      # Linux
```

### 项目结构

```
ccNexus/
├── main.go                    # 应用入口
├── app.go                     # 核心应用逻辑
├── wails.json                 # Wails 配置
│
├── internal/                  # Go 后端模块
│   ├── proxy/                 # HTTP 代理核心
│   │   ├── proxy.go          # 代理服务器
│   │   ├── handler.go        # 请求处理
│   │   ├── streaming.go      # SSE 流式响应
│   │   └── stats.go          # 统计记录
│   ├── transformer/           # API 格式转换器
│   │   ├── claude/           # Claude API
│   │   ├── openai/           # OpenAI API
│   │   └── gemini/           # Gemini API
│   ├── storage/               # SQLite 数据存储
│   ├── config/                # 配置管理
│   ├── webdav/                # WebDAV 同步
│   ├── logger/                # 日志系统
│   └── tray/                  # 系统托盘
│
└── frontend/                  # 前端代码
    ├── src/
    │   ├── modules/          # 功能模块
    │   ├── i18n/             # 国际化
    │   └── themes/           # 主题样式
    └── wailsjs/              # Wails 绑定
```

## ❓ 常见问题

### 安装和启动

**Q: Windows 提示「Windows 已保护你的电脑」怎么办？**

A: 点击「更多信息」→「仍要运行」。这是因为应用没有数字签名，不影响使用。

**Q: macOS 提示「无法打开，因为无法验证开发者」怎么办？**

A: 右键点击应用 → 选择「打开」→ 在弹出的对话框中点击「打开」。或者在「系统偏好设置」→「安全性与隐私」中允许运行。

**Q: 启动后端口被占用怎么办？**

A: 点击界面顶部的端口号，修改为其他未被占用的端口（如 3001），然后重启应用。

### 端点配置

**Q: 如何判断应该使用哪种转换器？**

A:
- 使用 Claude 官方 API 或兼容 Claude 格式的服务 → 选择 `claude`
- 使用 OpenAI API 或兼容 OpenAI 格式的服务 → 选择 `openai`
- 使用 Google Gemini API → 选择 `gemini`

**Q: 为什么 OpenAI/Gemini 转换器必须填写模型？**

A: 因为 Claude Code 发送的请求中包含 Claude 模型名称，代理需要知道应该转换为目标 API 的哪个模型。

**Q: 端点测试成功但实际使用失败？**

A: 测试只验证连接是否正常。实际使用时可能因为：
- API 密钥权限不足
- 模型名称错误
- API 配额用尽
- 请检查日志获取详细错误信息

### 使用问题

**Q: 如何查看请求日志？**

A: 点击界面底部的「日志」区域展开，可以查看实时日志。支持按级别（调试/信息/警告/错误）过滤。

**Q: Token 统计准确吗？**

A: Token 数量是估算值，基于文本长度计算，可能与实际计费有差异。仅供参考。

**Q: 如何备份配置？**

A: 两种方式：
1. 使用 WebDAV 云同步功能
2. 手动复制 `~/.ccNexus/ccnexus.db` 文件

**Q: 多个端点的轮换顺序是什么？**

A: 按照端点列表的顺序轮换。你可以通过拖拽调整端点顺序。

### 其他问题

**Q: 数据存储在哪里？安全吗？**

A: 所有数据存储在本地 `~/.ccNexus/` 目录下，API 密钥不会发送给任何第三方服务。

**Q: 支持哪些操作系统？**

A: 支持 Windows 10+、macOS 10.15+、Linux（需要 GTK3）。

**Q: 如何更新到新版本？**

A: 下载新版本覆盖安装即可，配置数据会自动保留。

## 📄 许可证

本项目采用 [MIT 许可证](LICENSE) 开源。

---

<div align="center">
Made with ❤️ by Chuck
</div>
