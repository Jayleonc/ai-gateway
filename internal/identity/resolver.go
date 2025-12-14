package identity

import "context"

// Resolver 身份解析器，解析项目/归属/成本中心
type Resolver interface {
	Resolve(ctx context.Context, keyInfo *APIKeyInfo) (*RequestContext, error)
}

// resolver 解析器实现
type resolver struct{}

// NewResolver 创建解析器
func NewResolver() Resolver {
	return &resolver{}
}

// Resolve 解析身份信息
func (r *resolver) Resolve(ctx context.Context, keyInfo *APIKeyInfo) (*RequestContext, error) {
	if keyInfo == nil {
		return nil, nil
	}

	reqCtx := &RequestContext{
		APIKeyID:      keyInfo.ID,
		ProjectID:     keyInfo.ProjectID,
		TenantID:      keyInfo.TenantID,
		CostCenter:    keyInfo.CostCenter,
		AllowedModels: keyInfo.AllowedModels,
		RateLimit:     keyInfo.RateLimit,
		QuotaLimit:    keyInfo.QuotaLimit,
		Labels:        keyInfo.Labels,
	}

	return reqCtx, nil
}
