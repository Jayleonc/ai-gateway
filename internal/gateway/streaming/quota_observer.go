package streaming

import (
	"log"

	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
)

// QuotaObserver 基于 quota.QuotaStore 的观察者实现
type QuotaObserver struct {
	quotaStore quota.QuotaStore
}

// NewQuotaObserver 创建配额观察者
func NewQuotaObserver(quotaStore quota.QuotaStore) *QuotaObserver {
	return &QuotaObserver{
		quotaStore: quotaStore,
	}
}

// OnFirstChunk 首个 chunk 时不做任何配额操作
func (qo *QuotaObserver) OnFirstChunk(ctx *StreamingContext) error {
	return nil
}

// OnChunk 在每个 chunk 时估算 token 并尝试消耗配额
func (qo *QuotaObserver) OnChunk(ctx *StreamingContext, delta *Delta) (stop bool, err error) {
	if qo.quotaStore == nil {
		return false, nil
	}

	// 粗估 token：len(text)/4（简单启发式）
	// 也可以用 len(text) 或其他策略
	estimatedTokens := int64(len(delta.Text) / 4)
	if estimatedTokens == 0 && len(delta.Text) > 0 {
		estimatedTokens = 1
	}

	// 尝试消耗配额
	if !qo.quotaStore.TryConsume(ctx.APIKeyID, estimatedTokens) {
		// 配额耗尽，标记应该中断
		ctx.MarkEnd(EndReasonQuotaExceeded, nil)
		return true, nil
	}

	// 累加已确认 token
	ctx.AddConfirmedTokens(estimatedTokens)
	return false, nil
}

// OnEnd 流结束时记录日志（可扩展为写 metering）
func (qo *QuotaObserver) OnEnd(ctx *StreamingContext) error {
	log.Printf(
		"[QuotaObserver] request_id=%s api_key_id=%s confirmed_tokens=%d end_reason=%s",
		ctx.RequestID, ctx.APIKeyID, ctx.ConfirmedTokens, ctx.EndReason,
	)
	return nil
}
