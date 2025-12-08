# ccNexus

<div align="center">

<p align="center">
  <img src="docs/images/ccNexus.svg" alt="Claude Code & Codex CLI Smart Endpoint Rotation Proxy" width="800" />
</p>

[![Build Status](https://github.com/lich0821/ccNexus/workflows/Build%20and%20Release/badge.svg)](https://github.com/lich0821/ccNexus/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2-blue)](https://wails.io/)

[English](README_EN.md) | [简体中文](README.md)

</div>

## Features

- **Multi-Endpoint Rotation**: Automatic failover, switches to next endpoint on failure
- **API Format Conversion**: Supports Claude, OpenAI, Gemini format conversion
- **Real-time Statistics**: Request count, error count, token usage monitoring
- **WebDAV Sync**: Sync configuration and data across devices
- **Cross-Platform**: Windows, macOS, Linux

<table>
  <tr>
    <td align="center"><img src="docs/images/EN-Light.png" alt="Light Theme" width="400"></td>
    <td align="center"><img src="docs/images/EN-Dark.png" alt="Dark Theme" width="400"></td>
  </tr>
</table>

## Quick Start

### 1. Download and Install

[Download Latest Release](https://github.com/lich0821/ccNexus/releases/latest)

- **Windows**: Extract and run `ccNexus.exe`
- **macOS**: Move to Applications, right-click → Open for first run
- **Linux**: `tar -xzf ccNexus-linux-amd64.tar.gz && ./ccNexus`

### 2. Add Endpoints

Click "Add Endpoint", fill in API URL, key, and select transformer (claude/openai/gemini).

### 3. Configure CC

#### Claude Code
`~/.claude/settings.json`
```json
{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "anything, not important",
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:3000",
    "CLAUDE_CODE_MAX_OUTPUT_TOKENS": "64000", // Some models may not support 64k
  }
  // Other settings
}

```

#### Codex CLI
Just configure `~/.codex/config.toml`:
```toml
model_provider = "ccNexus"
model = "gpt-5-codex"
preferred_auth_method = "apikey"

[model_providers.ccNexus]
name = "ccNexus"
base_url = "http://localhost:3000/v1"
wire_api = "responses"  # or "chat"

# Other settings
```

`~/.codex/auth.json` can be ignored.

## Get Help

<table>
  <tr>
    <td align="center"><img src="frontend/public/chat.jpg" alt="WeChat Group" width="200"></td>
    <td align="center"><img src="docs/images/MP.jpg" alt="WeChat Official Account" width="200"></td>
    <td align="center"><img src="docs/images/ME.png" alt="Personal WeChat" width="200"></td>
  </tr>
  <tr>
    <td align="center">Join group for feedback</td>
    <td align="center">Official Account</td>
    <td align="center">Add me if group expired</td>
  </tr>
</table>

## Documentation

- [Configuration Guide](docs/configuration_en.md)
- [Development Guide](docs/development_en.md)
- [FAQ](docs/FAQ_en.md)

## License

[MIT](LICENSE)
