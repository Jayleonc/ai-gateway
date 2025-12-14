package quota

import (
	"context"

	"github.com/Jayleonc/ai-gateway/internal/identity"
)

// Checker 配额检查器接口
type Checker interface {
	Check(ctx context.Context, reqCtx *identity.RequestContext, estimatedTokens int) error
}

// checker 配额检查器实现
type checker struct{}

// NewChecker 创建配额检查器
func NewChecker() Checker {
	return &checker{}
}

// Check 检查配额
func (c *checker) Check(ctx context.Context, reqCtx *identity.RequestContext, estimatedTokens int) error {
	// TODO: implement quota checking
	return nil
}
