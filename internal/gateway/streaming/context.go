package streaming

import (
	"time"
)

// Delta 表示一次 chunk 的增量信息
type Delta struct {
	Text string
	// 可扩展：role、tool_calls 等
}

// EndReason 表示流结束的原因
type EndReason string

const (
	EndReasonStop             EndReason = "stop"
	EndReasonLength           EndReason = "length"
	EndReasonQuotaExceeded    EndReason = "quota_exceeded"
	EndReasonError            EndReason = "error"
	EndReasonClientDisconnect EndReason = "client_disconnect"
)

// StreamingContext 表示一次 streaming 请求的生命周期状态
type StreamingContext struct {
	// 身份信息
	RequestID string
	APIKeyID  string
	Provider  string
	Model     string

	// 时间戳
	StartAt      time.Time
	FirstChunkAt *time.Time
	EndAt        *time.Time

	// 计量
	ChunkCount      int
	ConfirmedTokens int64

	// 结束状态
	EndReason EndReason
	Err       error
}

// IsStarted 检查是否已开始交付（FirstChunk 已发生）
func (sc *StreamingContext) IsStarted() bool {
	return sc.FirstChunkAt != nil
}

// MarkFirstChunk 标记首个 chunk 已发生
func (sc *StreamingContext) MarkFirstChunk() {
	if sc.FirstChunkAt == nil {
		now := time.Now()
		sc.FirstChunkAt = &now
	}
}

// MarkEnd 标记流结束
func (sc *StreamingContext) MarkEnd(reason EndReason, err error) {
	if sc.EndAt == nil {
		now := time.Now()
		sc.EndAt = &now
	}
	sc.EndReason = reason
	sc.Err = err
}

// AddConfirmedTokens 累加已确认 token
func (sc *StreamingContext) AddConfirmedTokens(delta int64) {
	sc.ConfirmedTokens += delta
}

// IncrementChunkCount 增加 chunk 计数
func (sc *StreamingContext) IncrementChunkCount() {
	sc.ChunkCount++
}
