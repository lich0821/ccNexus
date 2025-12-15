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

4. 数据与迁移
	- 仍支持从旧的 JSON 配置迁移到 SQLite（路径位于数据目录）。
	- 将主机目录挂载到 `/data` 可持久化数据库与配置。

5. 使用快速指引
	- 端口占用时可改成 `HOST_PORT:3000`（例如 `3021:3000`）。
	- 构建运行：`docker compose up -d --build`。
	- 启动后更新数据库中的 endpoint key/model 到真实值，或通过配置文件/环境变量完成覆盖。

此版本专注于 API 代理，无任何桌面/前端界面。
