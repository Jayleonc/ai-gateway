package quota

import (
	"context"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	"github.com/Jayleonc/ai-gateway/internal/identity"
)

// Checker 配额检查器接口
type Checker interface {
	Check(ctx context.Context, reqCtx *identity.RequestContext, estimatedTokens int) error
}

// checker 配额检查器实现
type checker struct {
	quotaStore QuotaStore
}

// NewChecker 创建配额检查器
func NewChecker(quotaStore QuotaStore) Checker {
	return &checker{
		quotaStore: quotaStore,
	}
}

// Check 检查配额
func (c *checker) Check(ctx context.Context, reqCtx *identity.RequestContext, estimatedTokens int) error {
	_ = ctx
	if c == nil || c.quotaStore == nil || reqCtx == nil {
		return nil
	}

	key := reqCtx.APIKeyID
	if key == "" {
		return nil
	}

	tokens := int64(estimatedTokens)
	if tokens <= 0 {
		tokens = 1
	}

	if !c.quotaStore.TryConsume(key, tokens) {
		return coreerrors.ErrQuotaExceeded
	}
	return nil
}
