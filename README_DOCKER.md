## Headless Docker Service Summary

本次调整将 ccNexus 从 Wails 桌面应用改造为纯后端 HTTP 服务，并提供容器化运行方式。核心改动要点：

1. 新增无头入口
	- 新增 [app/cmd/server/main.go](app/cmd/server/main.go) 作为 headless 入口：仅启动 HTTP 代理（无 GUI），支持优雅退出，读取 `CCNEXUS_DATA_DIR`、`CCNEXUS_DB_PATH`、`CCNEXUS_PORT`、`CCNEXUS_LOG_LEVEL` 环境变量。
	- 若存储中无任何 endpoint，会自动写入默认示例 endpoint，避免 “no endpoints configured” 直接退出。请尽快替换为真实 API 配置。

2. 镜像与构建
	- [Dockerfile](Dockerfile) 仅构建后端二进制 `ccnexus-server`，移除前端构建。暴露端口仅 `3000`（HTTP API）。
	- 构建阶段执行 `go mod tidy` 以生成 `go.sum`，并启用 CGO 支持 SQLite。

3. 运行与编排
	- [docker-compose.yml](docker-compose.yml) 仅映射 API 端口（示例 `3021:3000`），挂载数据卷 `/data`，健康检查指向 `/health`。
	- 默认环境：`CCNEXUS_DATA_DIR=/data`，`CCNEXUS_DB_PATH=/data/ccnexus.db`，`CCNEXUS_PORT=3000`。

4. 使用快速指引
	- 端口占用时可改成 `HOST_PORT:3000`（例如 `3021:3000`）。
	- 构建运行：`docker compose up -d --build`。
	- 启动后更新数据库中的 endpoint key/model 到真实值，或通过配置文件/环境变量完成覆盖。

此版本专注于 API 代理，并提供 Web 管理界面用于端点管理和监控。

## 文件结构

```
ccNexus/
├── app/
│   ├── cmd/
│   │   ├── server/
│   │   │   ├── main.go              # 主程序（不需要修改）
│   │   │   └── webui_plugin.go      # Web UI 插件接口
│   │   └── webui/                   # 🔌 Web UI 插件（整个文件夹）
│   │       ├── webui.go
│   │       ├── api/
│   │       └── ui/
```
---

## Web 管理界面

ccNexus 现已内置 Web 管理界面，提供可视化的端点管理和监控功能。

### 访问方式

启动服务后，通过浏览器访问：

```
http://localhost:3021/ui/
```

> 注意：端口号根据您的 docker-compose.yml 配置而定（默认映射为 `3021:3000`）

### 功能特性

- **仪表盘**：实时显示请求数、成功率、token 使用量等关键指标
- **端点管理**：通过 Web 界面添加、编辑、删除、启用/禁用 API 端点
- **统计数据**：查看每日、每周、每月的详细统计信息和趋势对比
- **测试功能**：在线测试端点连通性，查看响应时间和返回内容
- **实时监控**：通过 Server-Sent Events 实现数据自动刷新（每 5 秒）
- **深色/浅色主题**：支持主题切换，设置自动保存

### REST API 端点

除了 Web 界面，还可以直接调用 REST API：

#### 端点管理
- `GET /api/endpoints` - 列出所有端点
- `POST /api/endpoints` - 创建新端点
- `PUT /api/endpoints/:name` - 更新端点
- `DELETE /api/endpoints/:name` - 删除端点
- `PATCH /api/endpoints/:name/toggle` - 启用/禁用端点
- `POST /api/endpoints/:name/test` - 测试端点连通性
- `POST /api/endpoints/reorder` - 重新排序端点
- `GET /api/endpoints/current` - 获取当前活动端点
- `POST /api/endpoints/switch` - 切换到指定端点
- `POST /api/endpoints/fetch-models` - 获取可用模型列表

#### 统计数据
- `GET /api/stats/summary` - 总体统计
- `GET /api/stats/daily` - 今日统计
- `GET /api/stats/weekly` - 本周统计
- `GET /api/stats/monthly` - 本月统计
- `GET /api/stats/trends` - 趋势对比数据

#### 配置管理
- `GET /api/config` - 获取配置
- `PUT /api/config` - 更新配置
- `GET /api/config/port` - 获取代理端口
- `PUT /api/config/port` - 更新代理端口
- `GET /api/config/log-level` - 获取日志级别
- `PUT /api/config/log-level` - 设置日志级别

#### 实时更新
- `GET /api/events` - Server-Sent Events 流（用于实时监控）

### 使用示例

#### 通过 Web 界面添加端点

1. 访问 `http://localhost:3021/ui/`
2. 点击左侧导航栏的"Endpoints"（端点）
3. 点击右上角"Add Endpoint"（添加端点）按钮
4. 填写表单：
   - **Name**（名称）：为端点起一个易识别的名称，如 "Claude Official"
   - **API URL**：API 服务地址，如 `https://api.anthropic.com`
   - **API Key**：您的 API 密钥，如 `sk-ant-...`
   - **Transformer**（转换器）：选择 API 类型（claude/openai/gemini/deepseek）
   - **Model**（模型）：指定模型名称（Claude 可留空，OpenAI 需填写如 `gpt-4`）
   - **Remark**（备注）：可选的说明信息
   - **Enabled**（启用）：勾选以立即启用该端点
5. 点击"Create"（创建）保存

#### 通过 API 添加端点

```bash
curl -X POST http://localhost:3021/api/endpoints \
  -H "Content-Type: application/json" \
  -d '{
	"name": "Claude Official",
	"apiUrl": "https://api.anthropic.com",
	"apiKey": "sk-ant-your-key-here",
	"transformer": "claude",
	"model": "",
	"enabled": true,
	"remark": "官方 Claude API"
  }'
```
#### 查看统计数据

通过 Web 界面：
1. 点击左侧导航栏的"Statistics"（统计）
2. 选择时间范围：Daily（每日）/ Weekly（每周）/ Monthly（每月）
3. 查看各端点的请求数、错误数、token 使用量等详细数据


### 技术特点

- **零依赖前端**：使用原生 JavaScript，无需 npm、webpack 等构建工具
- **嵌入式部署**：前端文件嵌入 Go 二进制，单一可执行文件即可运行
- **实时更新**：通过 SSE 实现数据自动刷新，无需手动刷新页面
- **响应式设计**：支持桌面、平板、手机等各种设备
- **API 密钥保护**：在界面中自动掩码显示（仅显示最后 4 位）

### 安全建议

- **生产环境**：建议配置反向代理（如 Nginx）并启用 HTTPS
- **访问控制**：可通过反向代理添加 HTTP Basic Auth 或其他认证机制
- **CORS 配置**：当前 CORS 对所有来源开放，生产环境建议限制允许的域名
- **防火墙**：确保仅允许可信 IP 访问管理端口

### 故障排除

#### UI 无法访问
- 检查容器是否正常运行：`docker ps`
- 查看容器日志：`docker compose logs ccnexus`
- 确认端口映射正确：检查 docker-compose.yml 中的 ports 配置
- 验证防火墙规则是否允许访问

#### API 返回错误
- 查看详细日志：`docker compose logs -f ccnexus`
- 检查数据库文件权限：确保 `/data` 目录可写
- 验证端点配置：通过 Web 界面或 API 检查端点设置是否正确
- **OpenAI 端点需填写 model**：`transformer=openai` 时若 `model` 为空会导致启动反复报错。
  - 直接在宿主修复 DB（假设宿主挂载 `/data/ccnexus`，错误端点 id=5）：
	- 备份：`cp /data/ccnexus.db /data/ccnexus.db.bak-$(date +%Y%m%d%H%M%S)`
	- 临时进入工具容器：`docker run --rm -it -v /data/ccnexus:/data alpine sh`
	- 安装 sqlite：`apk add --no-cache sqlite`
	- 查看端点：`sqlite3 /data/ccnexus.db "SELECT id,name,transformer,model FROM endpoints;"`
	- 方案A补模型：`sqlite3 /data/ccnexus.db "UPDATE endpoints SET model='gpt-4o' WHERE id=5;"`
	- 方案B删除端点：`sqlite3 /data/ccnexus.db "DELETE FROM endpoints WHERE id=5;"`
	- 退出容器 `exit` 后重启服务：`docker compose restart` 或 `docker restart <容器名>`

### 开发与定制

Web UI 使用原生技术栈，修改非常简单：

1. 编辑 `app/ui/` 目录下的文件（HTML/CSS/JS）
2. 重新构建 Docker 镜像：`docker compose up -d --build`
3. 刷新浏览器查看效果

无需安装 Node.js、npm 或任何前端构建工具！

---

## Web UI 插件模式（可插拔）

- **目录结构**：完整插件位于 `app/cmd/webui/`，入口适配在 `app/cmd/server/webui_plugin.go`。
- **直接启用（默认）**：保留目录后 `docker compose up -d --build` 即包含 Web UI。
- **移除插件**：删除 `app/cmd/webui` 与 `app/cmd/server/webui_plugin.go`，重新构建后只保留代理功能。
- **重新添加**：将备份的 `webui` 目录与 `webui_plugin.go` 复制回原位，再次构建即可。
---

## Web UI 快速开始速览

- **访问入口**：生产 `http://localhost:3021/ui/`（或 `/admin` 重定向），测试 `http://localhost:3022/ui/`。
- **常用操作**：
  - 添加端点：`/ui/#endpoints` → Add Endpoint → 填写名称/API URL/API Key/transformer/model。
  - 测试端点：在端点列表点 Test，或 `/ui/#testing` 选择端点后 Send Test Request。
  - 查看统计：`/ui/#stats` 选择 Daily/Weekly/Monthly 查看趋势。
  - 切换/启用/禁用：在端点列表使用 Switch 或开关；Delete 可移除端点。
- **API 示例**：
  - 列表端点：`curl http://localhost:3021/api/endpoints`
  - 添加端点：`curl -X POST http://localhost:3021/api/endpoints -H "Content-Type: application/json" -d '{"name":"OpenAI","apiUrl":"api.openai.com","apiKey":"sk-...","transformer":"openai","model":"gpt-4"}'`
  - 测试端点：`curl -X POST http://localhost:3021/api/endpoints/OpenAI/test`
- **容器运维快捷命令**：
  - 查看日志：`docker logs -f ccnexus`（测试实例：`ccnexus2`）。
  - 重启：`docker compose restart`（测试用 `-f docker-compose.test.yml`）。
  - 重建：`docker compose up -d --build`（测试用 `-f docker-compose.test.yml`）。
  - 进入容器：`docker exec -it ccnexus sh`（测试实例 `ccnexus2`）。

---