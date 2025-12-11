# 详细配置

## 应用设置

| 设置项 | 说明 | 默认值 |
|--------|------|--------|
| 代理端口 | 本地代理监听端口 | `3000` |
| 日志级别 | 0= 调试，1= 信息，2= 警告，3= 错误 | `1` |
| 界面语言 | 中文 / English | `zh-CN` |
| 主题 | 12 种主题可选 | `light` |
| 自动主题 | 根据时间自动切换（7:00-19:00 浅色） | 关闭 |
| 窗口关闭行为 | 直接关闭 / 最小化到托盘 / 每次询问 | 每次询问 |

## 端点配置

### 转换器类型

| 转换器 | 说明 |
|--------|------|
| `claude` | Claude API |
| `openai` | OpenAI Chat API |
| `openai2` | OpenAI Response API |
| `gemini` | Google Gemini API |

### 配置示例

**Claude 端点：**
```json
{
  "name": "Claude 官方",
  "apiUrl": "https://api.anthropic.com",
  "apiKey": "sk-ant-api03-xxx",
  "enabled": true,
  "transformer": "claude"
}
```

**OpenAI 端点：**
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

## WebDAV 云同步

支持通过 WebDAV 协议同步配置和统计数据，兼容坚果云、NextCloud、ownCloud 等服务。

**配置步骤：**
1. 点击界面上的「WebDAV 云备份」
2. 填写 WebDAV 服务器地址、用户名、密码
3. 点击「测试连接」确认配置正确
4. 使用「备份」和「恢复」功能管理数据

## 数据存储位置

- 数据库：`~/.ccNexus/ccnexus.db`
