package streaming

import (
	"log"
	"time"
)

// StreamReader 定义流读取接口
type StreamReader interface {
	Next() (*Delta, error)
	Close() error
}

// Runtime 管理 streaming 请求的生命周期
// 真正的“权力中心”，但权力被严格限制
// Runtime 是唯一一个：
// - 知道整个生命周期
// - 能决定什么时候 stop
// - 能保证 OnEnd 一定被调用
// - 它做的一切，都是过程管理，不是业务判断。
// 它不关心 quota 是怎么算的，也不关心 token 是怎么估的。
type Runtime struct {
	ctx       *StreamingContext
	observers *ObserverGroup
}

// NewRuntime 创建 streaming runtime
func NewRuntime(ctx *StreamingContext, observers *ObserverGroup) *Runtime {
	return &Runtime{
		ctx:       ctx,
		observers: observers,
	}
}

// Run 执行 streaming 流程
// 返回是否应该中断流
func (r *Runtime) Run(reader StreamReader) error {
	defer reader.Close()

	// 读取第一个 chunk
	firstDelta, err := reader.Next()
	if err != nil {
		if r.ctx.EndReason == "" {
			r.ctx.SetEndReason(EndReasonInternalError, err)
		}
		r.endOnce()
		return err
	}

	// 标记首个 chunk 已发生
	r.ctx.MarkFirstChunk()
	r.ctx.IncrementChunkCount()

	// 调用 OnFirstChunk
	if r.observers != nil {
		if err := r.observers.OnFirstChunk(r.ctx); err != nil {
			if r.ctx.EndReason == "" {
				r.ctx.SetEndReason(EndReasonInternalError, err)
			}
			r.endOnce()
			return err
		}
	}

	// 处理首个 chunk（和后续 chunk 一样）
	stop, err := r.handleChunk(firstDelta)
	if err != nil {
		if r.ctx.EndReason == "" {
			r.ctx.SetEndReason(EndReasonInternalError, err)
		}
		r.endOnce()
		return err
	}
	if stop {
		if r.ctx.EndReason == "" {
			r.ctx.SetEndReason(EndReasonStop, nil)
		}
		r.endOnce()
		return nil
	}

	// 读取后续 chunk
	for {
		delta, err := reader.Next()
		if err != nil {
			// 流结束
			break
		}

		r.ctx.IncrementChunkCount()
		stop, err := r.handleChunk(delta)
		if err != nil {
			if r.ctx.EndReason == "" {
				r.ctx.SetEndReason(EndReasonInternalError, err)
			}
			r.endOnce()
			return err
		}
		if stop {
			if r.ctx.EndReason == "" {
				r.ctx.SetEndReason(EndReasonStop, nil)
			}
			r.endOnce()
			return nil
		}
	}

	// 正常结束
	if r.ctx.EndReason == "" {
		r.ctx.SetEndReason(EndReasonStop, nil)
	}
	r.endOnce()

	return nil
}

func (r *Runtime) endOnce() {
	if r.ctx != nil && r.ctx.Ended {
		return
	}
	if r.ctx != nil {
		r.ctx.MarkEnd()
	}
	if r.observers != nil {
		if err := r.observers.OnEnd(r.ctx); err != nil {
			log.Printf("[Runtime] OnEnd error: %v", err)
		}
	}
}

// handleChunk 处理单个 chunk
func (r *Runtime) handleChunk(delta *Delta) (stop bool, err error) {
	if r.observers == nil {
		return false, nil
	}

	stop, err = r.observers.OnChunk(r.ctx, delta)
	return stop, err
}

// GetContext 获取 streaming context
func (r *Runtime) GetContext() *StreamingContext {
	return r.ctx
}

// LogSummary 输出流摘要日志
func (r *Runtime) LogSummary() {
	duration := time.Duration(0)
	if r.ctx.EndAt != nil && !r.ctx.StartAt.IsZero() {
		duration = r.ctx.EndAt.Sub(r.ctx.StartAt)
	}

	firstChunkLatency := time.Duration(0)
	if r.ctx.FirstChunkAt != nil && !r.ctx.StartAt.IsZero() {
		firstChunkLatency = r.ctx.FirstChunkAt.Sub(r.ctx.StartAt)
	}

	log.Printf(
		"[StreamingRuntime] request_id=%s api_key_id=%s provider=%s model=%s "+
			"chunks=%d confirmed_tokens=%d end_reason=%s duration=%v first_chunk_latency=%v",
		r.ctx.RequestID, r.ctx.APIKeyID, r.ctx.Provider, r.ctx.Model,
		r.ctx.ChunkCount, r.ctx.ConfirmedTokens, r.ctx.EndReason,
		duration, firstChunkLatency,
	)
}
