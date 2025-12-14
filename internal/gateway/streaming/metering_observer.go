package streaming

import (
	"context"

	"github.com/Jayleonc/ai-gateway/internal/metering"
)

type MeteringObserver struct {
	recorder metering.Recorder
}

func NewMeteringObserver(recorder metering.Recorder) *MeteringObserver {
	return &MeteringObserver{recorder: recorder}
}

func (mo *MeteringObserver) OnFirstChunk(ctx *StreamingContext) error {
	return nil
}

func (mo *MeteringObserver) OnChunk(ctx *StreamingContext, delta *Delta) (stop bool, err error) {
	return false, nil
}

func (mo *MeteringObserver) OnEnd(ctx *StreamingContext) error {
	if mo == nil || mo.recorder == nil || ctx == nil {
		return nil
	}

	if ctx.FirstChunkAt == nil {
		return nil
	}

	record := &metering.UsageRecord{
		RequestID:        ctx.RequestID,
		APIKeyID:         ctx.APIKeyID,
		Provider:         ctx.Provider,
		Model:            ctx.Model,
		ConfirmedTokens:  ctx.ConfirmedTokens,
		ChunkCount:       ctx.ChunkCount,
		StartAt:          ctx.StartAt,
		FirstChunkAt:     ctx.FirstChunkAt,
		EndAt:            ctx.EndAt,
		EndReason:        string(ctx.EndReason),
		RequestedAt:      ctx.StartAt,
		CompletionTokens: 0,
		PromptTokens:     0,
		TotalTokens:      0,
	}

	if ctx.EndAt != nil {
		record.CompletedAt = *ctx.EndAt
		record.Duration = record.CompletedAt.Sub(record.RequestedAt)
	}

	status := "partial"
	if ctx.EndReason == EndReasonStop && ctx.Err == nil {
		status = "completed"
	}
	record.Status = status
	record.Success = status == "completed"

	if ctx.Err != nil {
		record.ErrorMsg = ctx.Err.Error()
	}

	return mo.recorder.Save(context.Background(), record)
}
