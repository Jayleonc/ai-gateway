package routing

import (
	"context"

	"github.com/Jayleonc/ai-gateway/internal/identity"
)

// RouteResult 路由结果
type RouteResult struct {
	Provider string
	Model    string
}

// Router 路由决策器接口
type Router interface {
	Route(ctx context.Context, reqCtx *identity.RequestContext, model string) (*RouteResult, error)
}

// router 路由决策器实现
type router struct{}

// NewRouter 创建路由决策器
func NewRouter() Router {
	return &router{}
}

// Route 路由决策
func (r *router) Route(ctx context.Context, reqCtx *identity.RequestContext, model string) (*RouteResult, error) {
	// TODO: implement routing logic
	// For now, default to openai provider
	return &RouteResult{
		Provider: "openai",
		Model:    model,
	}, nil
}
