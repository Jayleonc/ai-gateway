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
		r.ctx.MarkEnd(EndReasonError, err)
		if r.observers != nil {
			_ = r.observers.OnEnd(r.ctx)
		}
		return err
	}

	// 标记首个 chunk 已发生
	r.ctx.MarkFirstChunk()
	r.ctx.IncrementChunkCount()

	// 调用 OnFirstChunk
	if r.observers != nil {
		if err := r.observers.OnFirstChunk(r.ctx); err != nil {
			r.ctx.MarkEnd(EndReasonError, err)
			_ = r.observers.OnEnd(r.ctx)
			return err
		}
	}

	// 处理首个 chunk（和后续 chunk 一样）
	stop, err := r.handleChunk(firstDelta)
	if stop || err != nil {
		r.ctx.MarkEnd(EndReasonError, err)
		if r.observers != nil {
			_ = r.observers.OnEnd(r.ctx)
		}
		return err
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
		if stop || err != nil {
			r.ctx.MarkEnd(EndReasonError, err)
			if r.observers != nil {
				_ = r.observers.OnEnd(r.ctx)
			}
			return err
		}
	}

	// 正常结束
	r.ctx.MarkEnd(EndReasonStop, nil)
	if r.observers != nil {
		if err := r.observers.OnEnd(r.ctx); err != nil {
			log.Printf("[Runtime] OnEnd error: %v", err)
			return err
		}
	}

	return nil
}

// handleChunk 处理单个 chunk
func (r *Runtime) handleChunk(delta *Delta) (stop bool, err error) {
	if r.observers == nil {
		return false, nil
	}

	stop, err = r.observers.OnChunk(r.ctx, delta)
	if stop {
		// 观察者要求中断
		if r.ctx.EndReason == "" {
			r.ctx.MarkEnd(EndReasonStop, nil)
		}
	}
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
