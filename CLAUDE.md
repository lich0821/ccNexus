# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

ccNexus 是一个智能 API 端点轮换代理，专为 Claude Code 和 Codex CLI 设计。

**核心功能：**
- 多端点轮换与自动故障转移
- API 格式转换（Claude ↔ OpenAI ↔ Gemini）
- Codex Token Pool 管理（自动轮换、刷新、失效隔离）
- 实时统计与监控
- WebDAV 云同步

**两种运行模式：**
- **桌面模式**：基于 Wails v2 的跨平台 GUI 应用（`cmd/desktop/`）
- **服务器模式**：无头 HTTP API 代理（`cmd/server/`）

## 常用开发命令

### 开发与构建
```bash
# 桌面应用开发模式（支持热重载）
cd cmd/desktop && wails dev

# 构建桌面应用（指定平台）
wails build -platform linux/amd64    # Linux
wails build -platform darwin/amd64   # macOS
wails build -platform windows/amd64  # Windows

# 构建服务器
cd cmd/server && go build -ldflags="-s -w" -o ccnexus-server .

# 运行服务器
cd cmd/server && go run main.go
```

### 测试
```bash
# 运行所有测试
go test ./... -count=1

# 运行特定目录的测试
cd internal/proxy && go test -v ./...
cd internal/transformer/convert && go test -v ./...
```

### Docker
```bash
# 构建镜像
docker build -f cmd/server/Dockerfile -t ccnexus .

# 使用 docker-compose
cd cmd/server && docker-compose up -d
```

### 代码质量
```bash
go fmt ./...    # 格式化代码
go vet ./...    # 静态分析
go mod tidy     # 清理依赖
```

## 核心架构

### 目录结构
```
ccNexus/
├── cmd/
│   ├── desktop/          # 桌面应用入口（Wails）
│   │   ├── frontend/     # Vue.js 前端
│   │   └── main.go       # 桌面应用入口
│   └── server/           # 服务器模式入口
│       └── main.go       # 服务器入口
└── internal/
    ├── proxy/            # HTTP 代理核心
    ├── transformer/      # API 格式转换器
    ├── storage/          # SQLite 数据存储
    ├── config/           # 配置管理
    ├── webdav/           # WebDAV 同步
    ├── logger/           # 日志系统
    └── tray/             # 系统托盘（桌面模式）
```

### 关键组件

**代理层** (`internal/proxy/proxy.go`)
- 管理多个 API 端点，自动故障转移
- 跟踪当前端点和活动请求
- 使用连接池优化的 HTTP 客户端
- 处理流式和非流式响应

**转换器** (`internal/transformer/`)
- 在不同 API 格式之间转换请求和响应
- 支持流式传输的增量转换
- 处理工具调用和函数调用
- 类型定义：`internal/transformer/types.go`

**存储层** (`internal/storage/sqlite.go`)
- SQLite WAL 模式数据库
- 管理端点、凭证、使用统计、应用配置
- 线程安全操作

### 关键文件路径
- 数据库：`~/.ccNexus/ccnexus.db`
- 配置常量：`internal/config/config.go`（第 13-20 行：认证模式和端点 URL）
- 代理路由：`internal/proxy/proxy.go`（第 108-114 行）

## 端点配置

### 认证模式（`internal/config/config.go`）
- `api_key`：标准 API 密钥认证
- `token_pool`：Token 池（自动轮换）
- `codex_token_pool`：Codex Token Pool（使用 ChatGPT 后端）

### 转换器类型
- `claude`：Claude API
- `openai`：OpenAI Chat API
- `openai2`：OpenAI Response API
- `gemini`：Google Gemini API

### 端点配置规则
在 `internal/config/config.go` 的 `ApplyEndpointAuthModeRules` 函数中定义：
- Codex Token Pool 自动设置 API URL 和转换器
- Token Pool 模式会清空 APIKey
- URL 标准化处理

## API 端点

代理服务器提供以下端点（`internal/proxy/proxy.go` 第 108-114 行）：
- `/` - 主代理路由（所有 API 请求）
- `/v1/messages/count_tokens` - Token 计数
- `/v1/models` - 模型列表（带缓存）
- `/health` - 健康检查
- `/stats` - 统计数据

## 环境变量

服务器模式支持以下环境变量（`cmd/server/main.go`）：
- `CCNEXUS_PORT` - 覆盖默认端口
- `CCNEXUS_LOG_LEVEL` - 日志级别
- `CCNEXUS_DB_PATH` - 数据库路径
- `CCNEXUS_DATA_DIR` - 数据目录
- `CCNEXUS_BASIC_AUTH_USERNAME` - Basic Auth 用户名
- `CCNEXUS_BASIC_AUTH_PASSWORD` - Basic Auth 密码

## 依赖

- Go 1.24+
- Wails v2（桌面模式）
- Node.js 18+（前端开发）
- SQLite（modernc.org/sqlite，纯 Go 实现）

## 代码规范

**静态函数命名**：所有静态函数必须使用 `__` 前缀表示内部可见性

```c
// 符合规范
static int __internal_helper_function(int param) {
    return param + 1;
}

// 不符合规范
static int internal_helper_function(int param) {
    return param + 1;
}
```

**变量声明**：所有局部变量必须在函数体开头声明，并在声明时显式初始化

```c
// 符合规范
int function_name(void) {
    int ret = 0;
    int value = 0;
    char buffer[256] = {0};
    char *ptr = NULL;

    /* 可执行语句 */
    ret = do_something();
}

// 不符合规范
int function_name(void) {
    int ret = 0;
    ret = do_something();
    int value = 0;  /* 错误：在可执行语句后声明 */
}
```
