package identity

import "time"

// RequestContext 请求上下文，贯穿整个请求生命周期
//
// RequestContext 携带 identity Allowance（使用许可）的运行时视图，
// 会在 policy、routing、handler 等链路中向下传递。
type RequestContext struct {
	RequestID  string
	ReceivedAt time.Time

	// 身份信息
	APIKeyID   string
	ProjectID  string
	TenantID   string
	CostCenter string

	// 权限信息（Allowance 的运行态表达）
	AllowedModels []string
	QuotaLimit    int64
	RateLimit     int

	// 元数据
	Labels map[string]string
}

// NewRequestContext 创建请求上下文
func NewRequestContext(requestID string) *RequestContext {
	return &RequestContext{
		RequestID:  requestID,
		ReceivedAt: time.Now(),
		Labels:     make(map[string]string),
	}
}
