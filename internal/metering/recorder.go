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

type InMemoryRecorder struct {
	mu      sync.RWMutex
	records map[string]*UsageRecord
}

func (r *InMemoryRecorder) Key(record *UsageRecord) string {
	if record == nil {
		return ""
	}
	return record.RequestID
}

// NewRecorder 创建记录器
func NewRecorder() Recorder {
	return &InMemoryRecorder{}
}

// Save 保存记录
func (r *InMemoryRecorder) Save(ctx context.Context, record *UsageRecord) error {
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
func (r *InMemoryRecorder) BatchSave(ctx context.Context, records []*UsageRecord) error {
	for _, record := range records {
		if err := r.Save(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (r *InMemoryRecorder) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.records)
}

func (r *InMemoryRecorder) GetByRequestID(requestID string) *UsageRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.records == nil {
		return nil
	}
	return r.records[requestID]
}

func (r *InMemoryRecorder) All() []*UsageRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.records) == 0 {
		return nil
	}
	out := make([]*UsageRecord, 0, len(r.records))
	for _, v := range r.records {
		out = append(out, v)
	}
	return out
}
