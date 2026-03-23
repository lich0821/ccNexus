# /v1/models API 使用说明

## 功能概述
ccNexus 现已支持 OpenAI 兼容的 `/v1/models` API，聚合所有后端端点的模型列表。

## 快速开始

### 启动服务
```bash
go run ./cmd/server
```

### 获取模型列表
```bash
curl http://localhost:3000/v1/models
```

### 强制刷新缓存
```bash
curl http://localhost:3000/v1/models?refresh=true
```

## 响应示例
```json
{
  "object": "list",
  "data": [
    {
      "id": "claude-sonnet-4-20250514",
      "object": "model",
      "created": 1700000000,
      "owned_by": "anthropic",
      "endpoint_id": "Claude Official"
    }
  ]
}
```

## 配置项
在 `config.json` 中添加：
```json
{
  "modelsCacheTTL": 30  // 缓存时间（分钟），默认30
}
```

## 支持的端点
- **openai/openai2**: 自动查询后端 /v1/models
- **gemini**: 自动查询后端 /v1beta/models
- **claude**: 使用配置的 model 字段（无API查询）

## 特性
- ✅ 聚合多后端模型列表
- ✅ 自动缓存（30分钟，可配置）
- ✅ 支持刷新参数
- ✅ 失败降级（返回默认模型）
