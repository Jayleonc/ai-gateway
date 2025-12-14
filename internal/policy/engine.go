package policy

import (
	"context"

	"github.com/Jayleonc/ai-gateway/internal/identity"
	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
	"github.com/Jayleonc/ai-gateway/internal/policy/ratelimit"
	"github.com/Jayleonc/ai-gateway/internal/policy/routing"
)

// Engine 策略引擎接口
type Engine interface {
	Evaluate(ctx context.Context, reqCtx *identity.RequestContext, req *EvaluateRequest) (*Decision, error)
}

// engine 策略引擎实现
type engine struct {
	quotaChecker quota.Checker
	rateLimiter  ratelimit.Limiter
	router       routing.Router
}

// NewEngine 创建策略引擎
func NewEngine(qc quota.Checker, rl ratelimit.Limiter, r routing.Router) Engine {
	return &engine{
		quotaChecker: qc,
		rateLimiter:  rl,
		router:       r,
	}
}

// Evaluate 评估请求
func (e *engine) Evaluate(ctx context.Context, reqCtx *identity.RequestContext, req *EvaluateRequest) (*Decision, error) {
	// 1. 配额检查
	if err := e.quotaChecker.Check(ctx, reqCtx, req.EstimatedTokens); err != nil {
		return NewDenyDecision(DenyCodeQuotaExceeded, err.Error()), nil
	}

	// 2. 限流判断
	allowed, err := e.rateLimiter.Allow(ctx, reqCtx)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return NewDenyDecision(DenyCodeRateLimited, "rate limit exceeded"), nil
	}

	// 3. 路由决策
	routeResult, err := e.router.Route(ctx, reqCtx, req.Model)
	if err != nil {
		return nil, err
	}

	return NewAllowDecision(routeResult.Provider, routeResult.Model), nil
}
