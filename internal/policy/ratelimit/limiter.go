package ratelimit

import (
	"context"

	"github.com/Jayleonc/ai-gateway/internal/identity"
)

// Limiter 限流器接口
type Limiter interface {
	Allow(ctx context.Context, reqCtx *identity.RequestContext) (bool, error)
}

// limiter 限流器实现
type limiter struct{}

// NewLimiter 创建限流器
func NewLimiter() Limiter {
	return &limiter{}
}

// Allow 判断是否允许请求
func (l *limiter) Allow(ctx context.Context, reqCtx *identity.RequestContext) (bool, error) {
	// TODO: implement rate limiting
	return true, nil
}
