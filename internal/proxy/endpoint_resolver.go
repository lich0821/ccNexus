package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/logger"
)

// EndpointResolver 负责从 HTTP 请求中解析客户端指定的端点
// 按优先级解析：HTTP Header → 特殊模型名格式 → 查询参数
type EndpointResolver struct {
	getEndpointsFunc func() []config.Endpoint // 动态获取端点列表的函数
}

// NewEndpointResolver 创建新的端点解析器
// getEndpointsFunc 用于动态获取最新的端点列表
func NewEndpointResolver(endpoints []config.Endpoint) *EndpointResolver {
	// 闭包捕获端点切片
	eps := endpoints
	return &EndpointResolver{
		getEndpointsFunc: func() []config.Endpoint {
			return eps
		},
	}
}

// NewEndpointResolverWithFunc 创建一个使用动态函数的解析器
func NewEndpointResolverWithFunc(getEndpointsFunc func() []config.Endpoint) *EndpointResolver {
	return &EndpointResolver{
		getEndpointsFunc: getEndpointsFunc,
	}
}

// ResolveEndpoint 从请求中解析端点，按优先级处理
// 返回：解析到的端点（可能为 nil），模型覆盖值（可能为空），错误信息
func (r *EndpointResolver) ResolveEndpoint(req *http.Request, bodyBytes []byte) (*config.Endpoint, string, error) {
	// 获取最新的端点列表
	endpoints := r.getEndpointsFunc()

	// 优先级 1: HTTP 头部
	if endpointName := r.parseEndpointFromHeader(req); endpointName != "" {
		endpoint := r.findEndpointByName(endpointName, endpoints)
		if endpoint == nil {
			return nil, "", fmt.Errorf("指定的端点 '%s' 不存在或未启用", endpointName)
		}
		logger.Debug("[Resolver] 通过 HTTP 头部指定端点: %s", endpointName)
		return endpoint, "", nil
	}

	// 优先级 2: 特殊模型名格式
	var streamReq struct {
		Model string `json:"model"`
	}
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &streamReq)
	}
	modelName := strings.TrimSpace(streamReq.Model)

	if modelName != "" && strings.HasPrefix(modelName, "@") {
		endpointName, modelOverride := r.parseEndpointFromModel(modelName)
		endpoint := r.findEndpointByName(endpointName, endpoints)
		if endpoint == nil {
			return nil, "", fmt.Errorf("指定的端点 '%s' 不存在或未启用", endpointName)
		}
		logger.Debug("[Resolver] 通过模型名格式指定端点: %s, 模型: %s", endpointName, modelOverride)
		return endpoint, modelOverride, nil
	}

	// 优先级 3: 查询参数
	if endpointName := r.parseEndpointFromQuery(req); endpointName != "" {
		endpoint := r.findEndpointByName(endpointName, endpoints)
		if endpoint == nil {
			return nil, "", fmt.Errorf("指定的端点 '%s' 不存在或未启用", endpointName)
		}
		logger.Debug("[Resolver] 通过查询参数指定端点: %s", endpointName)
		return endpoint, "", nil
	}

	// 没有指定端点，使用默认轮询机制
	return nil, "", nil
}

// parseEndpointFromHeader 从 HTTP 头部解析端点
// 支持的头部: X-CCN-Endpoint, X-Endpoint-Name
func (r *EndpointResolver) parseEndpointFromHeader(req *http.Request) string {
	// 优先检查 X-CCN-Endpoint
	if name := strings.TrimSpace(req.Header.Get("X-CCN-Endpoint")); name != "" {
		return name
	}
	// 其次检查 X-Endpoint-Name
	if name := strings.TrimSpace(req.Header.Get("X-Endpoint-Name")); name != "" {
		return name
	}
	return ""
}

// parseEndpointFromModel 从模型名解析端点（支持 @endpoint 格式）
// 格式：
//   @endpoint-name/model-name → 返回 (endpoint-name, model-name)
//   @endpoint-name → 返回 (endpoint-name, "")
func (r *EndpointResolver) parseEndpointFromModel(model string) (string, string) {
	model = strings.TrimSpace(model)
	if !strings.HasPrefix(model, "@") {
		return "", ""
	}

	// 移除 @ 前缀
	model = model[1:]

	// 查找斜杠分隔符
	slashIndex := strings.Index(model, "/")
	if slashIndex == -1 {
		// 格式: @endpoint-name
		endpointName := strings.TrimSpace(model)
		return endpointName, ""
	}

	// 格式: @endpoint-name/model-name
	endpointName := strings.TrimSpace(model[:slashIndex])
	modelName := strings.TrimSpace(model[slashIndex+1:])
	return endpointName, modelName
}

// parseEndpointFromQuery 从查询参数解析端点
// 支持的参数: endpoint, ep
func (r *EndpointResolver) parseEndpointFromQuery(req *http.Request) string {
	// 优先检查 endpoint
	if name := strings.TrimSpace(req.URL.Query().Get("endpoint")); name != "" {
		return name
	}
	// 其次检查 ep
	if name := strings.TrimSpace(req.URL.Query().Get("ep")); name != "" {
		return name
	}
	return ""
}

// findEndpointByName 根据名称查找端点（不区分大小写）
// 只返回已启用的端点
func (r *EndpointResolver) findEndpointByName(name string, endpoints []config.Endpoint) *config.Endpoint {
	targetName := strings.ToLower(strings.TrimSpace(name))

	for i := range endpoints {
		endpoint := &endpoints[i]
		if !endpoint.Enabled {
			continue
		}
		if strings.ToLower(strings.TrimSpace(endpoint.Name)) == targetName {
			return endpoint
		}
	}
	return nil
}