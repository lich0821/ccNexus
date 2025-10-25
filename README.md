# ccNexus (Claude Code Nexus)

<div align="center">

**A smart API endpoint rotation proxy for Claude Code**

[![Build Status](https://github.com/lich0821/ccNexus/workflows/Build%20and%20Release/badge.svg)](https://github.com/lich0821/ccNexus/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2-blue)](https://wails.io/)

[English](README.md) | [简体中文](README_CN.md)

</div>

## ✨ Features

- 🔄 **Automatic Endpoint Rotation** - Seamlessly switches between endpoints on errors
- 🌐 **Multi-Provider Support** - Use official Claude API and third-party providers
- 🔁 **Smart Retry** - Retries on any non-200 response
- 📊 **Real-time Statistics** - Monitor requests, errors, and endpoint usage
- 💰 **Token Usage Tracking** - Track input/output tokens for each endpoint
- 🎯 **Endpoint Management** - Enable/disable endpoints with toggle switches
- 🔐 **Secure API Key Display** - Shows only last 4 characters of API keys
- 🚦 **Smart Load Balancing** - Distributes requests only to enabled endpoints
- 🖥️ **Desktop GUI** - Beautiful cross-platform interface built with Wails
- 🚀 **Single Binary** - No dependencies, just download and run
- 🔧 **Easy Configuration** - Manage endpoints through GUI or config file
- 💾 **Persistent Config** - Automatically saves configuration
- 🔒 **Local First** - All data stays on your machine

## 🚀 Quick Start

### Download

Download the latest release for your platform:

- **Windows**: `ccNexus-windows-amd64.zip`
- **macOS (Intel)**: `ccNexus-darwin-amd64.zip`
- **macOS (Apple Silicon)**: `ccNexus-darwin-arm64.zip`
- **Linux**: `ccNexus-linux-amd64.tar.gz`

[📥 Download Latest Release](https://github.com/lich0821/ccNexus/releases/latest)

### Installation

#### Windows

1. Extract the ZIP file
2. Double-click `ccNexus.exe`
3. The application will start with a default configuration

#### macOS

1. Extract the ZIP file
2. Move `ccNexus.app` to Applications folder
3. Right-click and select "Open" (first time only)
4. The application will start with a default configuration

#### Linux

```bash
tar -xzf ccNexus-linux-amd64.tar.gz
chmod +x ccNexus
./ccNexus
```

### Configuration

1. **Add Endpoints**: Click "Add Endpoint" button
2. **Fill in Details**:
   - Name: A friendly name (e.g., "Claude Official")
   - API URL: The API server address (e.g., `api.anthropic.com`)
   - API Key: Your API key
3. **Save**: Click "Save" to add the endpoint

### Configure Claude Code

In Claude Code settings:
- **API Base URL**: `http://localhost:3000`
- **API Key**: Any value (will be replaced by proxy)

## 📖 How It Works

```
Claude Code → Proxy (localhost:3000) → Endpoint #1 (non-200 response)
                                     → Endpoint #2 (success) ✅
```

1. **Request Interception**: Proxy receives all API requests
2. **Endpoint Selection**: Uses current available endpoint
3. **Error Detection**: Monitors response status codes
4. **Auto Retry**: Switches endpoint and retries on non-200 responses
5. **Round Robin**: Cycles through all endpoints

## 🎉 What's New in v0.2.0

### 🔐 Enhanced Security
- **Masked API Keys**: API keys now show only the last 4 characters (e.g., `****ABCD`)
- Better protection when sharing screenshots or during presentations

### 📊 Advanced Statistics
- **Per-Endpoint Request Tracking**: See request count and error rate for each endpoint
- **Token Usage Monitoring**: Track input and output tokens consumed by each endpoint
- **Real-time Updates**: Statistics refresh every 5 seconds automatically

### 🎯 Endpoint Control
- **Toggle Switches**: Enable/disable endpoints with a single click
- **Visual Status Indicators**: Quickly identify active (✅) and disabled (❌) endpoints
- **Zero Downtime**: Disable problematic endpoints without stopping the proxy

### 💡 Usage Example

View detailed statistics for each endpoint:
```
📊 Requests: 1,234 | Errors: 5
🎯 Tokens: 45,678 (In: 12,345, Out: 33,333)
```

Disable expensive or rate-limited endpoints temporarily while keeping others active.

## 🔧 Configuration File

Configuration is stored at:
- **Windows**: `%USERPROFILE%\.ccNexus\config.json`
- **macOS/Linux**: `~/.ccNexus/config.json`

Example:

```json
{
  "port": 3000,
  "endpoints": [
    {
      "name": "Claude Official 1",
      "apiUrl": "api.anthropic.com",
      "apiKey": "sk-ant-api03-your-key-1",
      "enabled": true
    },
    {
      "name": "Third Party Provider",
      "apiUrl": "api.example.com",
      "apiKey": "your-key-2",
      "enabled": true
    }
  ]
}
```

## 🛠️ Development

### Prerequisites

- Go 1.22+
- Node.js 18+
- Wails CLI v2

### Setup

```bash
# Clone repository
git clone https://github.com/lich0821/ccNexus.git
cd ccNexus

# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install dependencies
go mod download
cd frontend && npm install && cd ..

# Run in development mode
wails dev
```

### Build

```bash
# Build for current platform
wails build

# Build for specific platform
wails build -platform windows/amd64
wails build -platform darwin/amd64
wails build -platform darwin/arm64
wails build -platform linux/amd64
```

## 📚 Project Structure

```
ccNexus/
├── main.go                 # Application entry point
├── app.go                  # Wails app logic
├── internal/
│   ├── proxy/             # Proxy core logic
│   │   ├── proxy.go       # HTTP proxy with retry
│   │   └── stats.go       # Statistics tracking
│   └── config/            # Configuration management
│       └── config.go      # Config structure
├── frontend/              # Frontend UI
│   ├── index.html
│   └── src/
│       ├── main.js        # UI logic
│       └── style.css      # Styles
└── .github/workflows/
    └── build.yml          # CI/CD pipeline
```

## ❓ FAQ

### Q: Proxy won't start?

**A**: Check if port is in use:
```bash
# macOS/Linux
lsof -i :3000

# Windows
netstat -ano | findstr :3000
```

### Q: Claude Code can't connect?

**A**: Verify:
1. Proxy application is running
2. Claude Code Base URL is `http://localhost:3000`
3. Firewall isn't blocking the connection

### Q: Endpoint switching not working?

**A**: Check:
1. Multiple endpoints are configured
2. API keys are valid
3. View application logs for switching behavior

### Q: How to view detailed logs?

**A**:
- **macOS**: Run app from terminal to see logs
- **Windows**: Check log files in app directory
- **Linux**: Use `./ccNexus 2>&1 | tee ccNexus.log`

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Wails](https://wails.io/) - Amazing Go + Web framework
- [Anthropic](https://www.anthropic.com/) - Claude Code
- All contributors and users

## 📞 Support

- 🐛 [Report Bug](https://github.com/lich0821/ccNexus/issues/new)
- 💡 [Request Feature](https://github.com/lich0821/ccNexus/issues/new)
- 💬 [Discussions](https://github.com/lich0821/ccNexus/discussions)

---

<div align="center">
Made with ❤️ by Chuck
</div>
