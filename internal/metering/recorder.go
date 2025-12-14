package metering

import (
	"context"
	"log"
	"sync"
)

// Recorder 调用记录持久化接口
type Recorder interface {
	Save(ctx context.Context, record *UsageRecord) error
	BatchSave(ctx context.Context, records []*UsageRecord) error
}

// recorder 记录器实现（stub）
type recorder struct {
	mu      sync.RWMutex
	records map[string]*UsageRecord
}

func (r *recorder) Key(record *UsageRecord) string {
	if record == nil {
		return ""
	}
	if record.APIKeyID != "" {
		return record.APIKeyID + ":" + record.RequestID
	}
	return record.RequestID
}

// NewRecorder 创建记录器
func NewRecorder() Recorder {
	return &recorder{}
}

// Save 保存记录
func (r *recorder) Save(ctx context.Context, record *UsageRecord) error {
	k := r.Key(record)
	if record == nil || k == "" {
		log.Println("invalid record")
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.records == nil {
		r.records = make(map[string]*UsageRecord)
	}

	r.records[k] = record

	return nil
}

// BatchSave 批量保存
func (r *recorder) BatchSave(ctx context.Context, records []*UsageRecord) error {
	for _, record := range records {
		if err := r.Save(ctx, record); err != nil {
			return err
		}
	}
	return nil
}
