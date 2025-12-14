package metering

import "context"

// Recorder 调用记录持久化接口
type Recorder interface {
	Save(ctx context.Context, record *UsageRecord) error
	BatchSave(ctx context.Context, records []*UsageRecord) error
}

// recorder 记录器实现（stub）
type recorder struct{}

// NewRecorder 创建记录器
func NewRecorder() Recorder {
	return &recorder{}
}

// Save 保存记录
func (r *recorder) Save(ctx context.Context, record *UsageRecord) error {
	// TODO: implement persistence
	return nil
}

// BatchSave 批量保存
func (r *recorder) BatchSave(ctx context.Context, records []*UsageRecord) error {
	// TODO: implement batch persistence
	return nil
}
