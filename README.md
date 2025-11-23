# ccNexus

<div align="center">

**Smart API endpoint rotation proxy for Claude Code**

[![Build Status](https://github.com/lich0821/ccNexus/workflows/Build%20and%20Release/badge.svg)](https://github.com/lich0821/ccNexus/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2-blue)](https://wails.io/)

[English](README.md) | [简体中文](README_CN.md)

</div>

## 📸 Screenshot

<p align="center">
  <img src="docs/images/EN-Light.png" alt="Light" width="45%">
  <img src="docs/images/EN-Dark.png" alt="Dark" width="45%">
</p>

## ✨ Features

- 🔄 **Auto Endpoint Rotation** - Seamless failover on errors
- 🔀 **Multi-Format Support** - Claude, OpenAI, and Gemini API formats
- 🔁 **Smart Retry** - Automatic retry with endpoint switching
- 📊 **Real-time Stats** - Monitor requests, errors, and token usage
- 📈 **Historical Data** - SQLite-based statistics with monthly archives
- 🖥️ **Desktop GUI** - Cross-platform interface with light/dark themes
- 🚀 **Single Binary** - No dependencies required
- 🔒 **Local First** - All data stays on your machine

## 🚀 Quick Start

[📥 Download Latest Release](https://github.com/lich0821/ccNexus/releases/latest)

### Installation

**Windows**: Extract ZIP and run `ccNexus.exe`
**macOS**: Extract ZIP, move to Applications, right-click → Open
**Linux**: `tar -xzf ccNexus-linux-amd64.tar.gz && ./ccNexus`

### Setup

1. Click "Add Endpoint" and configure:
   - **Name**: Friendly identifier
   - **API URL**: e.g., `api.anthropic.com`
   - **API Key**: Your API key
   - **Transformer**: Claude/OpenAI/Gemini
   - **Model**: Required for OpenAI/Gemini (e.g., `gpt-4-turbo`)

2. Configure Claude Code:
   - **API Base URL**: `http://localhost:3000`
   - **API Key**: Any value

## 📖 How It Works

```
Claude Code → Proxy (localhost:3000) → Endpoint #1 (fails) → Endpoint #2 (success) ✅
```

Proxy intercepts requests, forwards to enabled endpoints with round-robin rotation, and automatically retries on failures.

## 🔧 Configuration

**Data Location**: `~/.ccNexus/` (Windows: `%USERPROFILE%\.ccNexus\`)

**Files**:
- `ccnexus.db` - SQLite database (config + stats)
- `config.json` - Legacy config (auto-migrated on first run)

**Settings**:
- `port`: Proxy port (default: 3000)
- `logLevel`: 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR

## 🛠️ Development

**Prerequisites**: Go 1.22+, Node.js 18+

```bash
# Clone and run
git clone https://github.com/lich0821/ccNexus.git
cd ccNexus
node run.mjs  # Auto-installs Wails CLI and dependencies

# Build
npm run build              # Current platform
npm run build:prod         # Optimized build
npm run build:windows      # Windows
npm run build:macos        # macOS
npm run build:linux        # Linux
```

## 📚 Architecture

```
ccNexus/
├── main.go & app.go           # Application entry
├── internal/
│   ├── proxy/                 # HTTP proxy with retry logic
│   ├── storage/               # SQLite persistence + migration
│   ├── transformer/           # API format converters (Claude/OpenAI/Gemini)
│   ├── config/                # Configuration management
│   └── logger/                # Multi-level logging
└── frontend/                  # Vanilla JS UI
```

<div align="center">
Made with ❤️ by Chuck
</div>
