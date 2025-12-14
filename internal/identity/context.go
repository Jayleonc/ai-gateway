package identity

import "time"

// RequestContext 请求上下文，贯穿整个请求生命周期
type RequestContext struct {
	RequestID  string
	ReceivedAt time.Time

	// 身份信息
	APIKeyID   string
	ProjectID  string
	TenantID   string
	CostCenter string

	// 权限信息
	AllowedModels []string
	RateLimit     int
	QuotaLimit    int64

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
