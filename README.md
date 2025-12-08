<div align="center">

<p align="center">
  <img src="docs/images/ccNexus.svg" alt="Claude Code & Codex CLI 智能端点轮换代理" width="800" />
</p>

[![构建状态](https://github.com/lich0821/ccNexus/workflows/Build%20and%20Release/badge.svg)](https://github.com/lich0821/ccNexus/actions)
[![许可证: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 版本](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2-blue)](https://wails.io/)

[English](README_EN.md) | [简体中文](README.md)

</div>

## 功能特性

- **多端点轮换**：自动故障转移，一个失败自动切换下一个
- **API 格式转换**：支持 Claude、OpenAI、Gemini 格式互转
- **实时统计**：请求数、错误数、Token 用量监控
- **WebDAV 同步**：多设备间同步配置和数据
- **跨平台**：Windows、macOS、Linux

<table>
  <tr>
    <td align="center"><img src="docs/images/CN-Light.png" alt="明亮主题" width="400"></td>
    <td align="center"><img src="docs/images/CN-Dark.png" alt="暗黑主题" width="400"></td>
  </tr>
</table>

## 快速开始

### 1. 下载安装

[下载最新版本](https://github.com/lich0821/ccNexus/releases/latest)

- **Windows**: 解压后运行 `ccNexus.exe`
- **macOS**: 移动到「应用程序」，首次运行右键点击 → 打开
- **Linux**: `tar -xzf ccNexus-linux-amd64.tar.gz && ./ccNexus`

### 2. 添加端点

点击「添加端点」，填写 API 地址、密钥、选择转换器（claude/openai/gemini）。

### 3. 配置 CC

#### Claude Code
`~/.claude/settings.json`
```json
{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "随便写，不重要",
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:3000",
    "CLAUDE_CODE_MAX_OUTPUT_TOKENS": "64000", // 有些模型可能不支持 64k
  }
  // 其他配置
}

```

#### Codex CLI
只需要配置 `~/.codex/config.toml`：
```toml
model_provider = "ccNexus"
model = "gpt-5-codex"
preferred_auth_method = "apikey"

[model_providers.ccNexus]
name = "ccNexus"
base_url = "http://localhost:3000/v1"
wire_api = "responses"  # 或 "chat"

# 其他配置
```

`~/.codex/auth.json` 可以忽略了。

## 获取帮助

<table>
  <tr>
    <td align="center"><img src="frontend/public/chat.jpg" alt="微信群" width="200"></td>
    <td align="center"><img src="docs/images/MP.jpg" alt="公众号" width="200"></td>
    <td align="center"><img src="docs/images/ME.png" alt="个人微信" width="200"></td>
  </tr>
  <tr>
    <td align="center">问题反馈请加群</td>
    <td align="center">公众号</td>
    <td align="center">群过期请加好友</td>
  </tr>
</table>

## 文档

- [详细配置](docs/configuration.md)
- [开发指南](docs/development.md)
- [常见问题](docs/FAQ.md)

## 许可证

[MIT](LICENSE)
