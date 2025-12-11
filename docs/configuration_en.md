# Configuration Guide

## Application Settings

| Setting | Description | Default |
|---------|-------------|---------|
| Proxy Port | Local proxy listening port | `3000` |
| Log Level | 0=Debug, 1=Info, 2=Warn, 3=Error | `1` |
| Language | Chinese / English | `zh-CN` |
| Theme | 12 themes available | `light` |
| Auto Theme | Auto switch based on time (7:00-19:00 light) | Off |
| Window Close Behavior | Close / Minimize to tray / Ask every time | Ask every time |

## Endpoint Configuration

### Transformer Types

| Transformer | Description |
|--------|------|
| `claude` | Claude API |
| `openai` | OpenAI Chat API |
| `openai2` | OpenAI Response API |
| `gemini` | Google Gemini API |

### Configuration Examples

**Claude Endpoint:**
```json
{
  "name": "Claude Official",
  "apiUrl": "https://api.anthropic.com",
  "apiKey": "sk-ant-api03-xxx",
  "enabled": true,
  "transformer": "claude"
}
```

**OpenAI Endpoint:**
```json
{
  "name": "OpenAI Proxy",
  "apiUrl": "https://api.openai.com",
  "apiKey": "sk-xxx",
  "enabled": true,
  "transformer": "openai",
  "model": "gpt-4-turbo"
}
```

**Gemini Endpoint:**
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

## WebDAV Cloud Sync

Supports syncing configuration and statistics via WebDAV protocol, compatible with Nutstore, NextCloud, ownCloud, etc.

**Setup Steps:**
1. Click "WebDAV Cloud Backup" in the interface
2. Fill in WebDAV server URL, username, password
3. Click "Test Connection" to verify configuration
4. Use "Backup" and "Restore" to manage data

## Data Storage Location

- Database: `~/.ccNexus/ccnexus.db`
