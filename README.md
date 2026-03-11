<div align="center">

<p align="center">
  <img src="docs/images/ccNexus.svg" alt="Claude Code & Codex CLI 智能端点轮换代理" width="720" />
</p>

[![构建状态](https://github.com/lich0821/ccNexus/workflows/Build%20and%20Release/badge.svg)](https://github.com/lich0821/ccNexus/actions)
[![许可证: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 版本](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2-blue)](https://wails.io/)

[English](docs/README_EN.md) | [简体中文](README.md)

</div>

## 功能特性

- **多端点轮换**：自动故障转移，一个失败自动切换下一个
- **API 格式转换**：支持 Claude、OpenAI、Gemini 格式互转
- **Codex Token Pool**：支持批量导入 `access_token/refresh_token`，自动轮换、自动刷新、失效隔离与状态管理
- **Token Pool 使用统计**：单条凭证请求/错误/Token 统计，支持快捷查看
- **实时统计**：事件驱动的零延迟统计更新，支持今日/昨日/本周/本月四周期快速切换
- **端点筛选**：按类型、可用性、启用状态多选筛选，快速定位端点
- **WebDAV 同步**：多设备间同步配置和数据
- **跨平台**：Windows、macOS、Linux
- **[Docker](docs/README_DOCKER.md)**：纯后端 HTTP 服务，并提供容器化运行

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

点击「添加端点」，填写 API 地址、密钥、选择转换器（claude/openai/gemini/openai2）。

如需使用 Codex Token Pool：
- 认证方式选择 `Codex Token Pool`
- 在 Token Pool 页面导入一批 token JSON（支持 `access_token` + `refresh_token`）
- 系统会自动进行 token 轮换、401 后刷新与状态管理（active/expiring/need_refresh/invalid 等）

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
    <td align="center"><img src="https://gitee.com/hea7en/images/raw/master/group/chat.png" alt="微信群" width="200"></td>
    <td align="center"><img src="cmd/desktop/frontend/public/WeChat.jpg" alt="公众号" width="200"></td>
    <td align="center"><img src="cmd/desktop/frontend/public/ME.png" alt="个人微信" width="200"></td>
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
