package metering

import (
	"context"
	"time"
)

// UsageRecord 使用量记录
type UsageRecord struct {
	RequestID   string
	RequestedAt time.Time
	CompletedAt time.Time
	Duration    time.Duration

	// 归属信息
	APIKeyID   string
	ProjectID  string
	TenantID   string
	CostCenter string
	Labels     map[string]string

	// 调用信息
	Provider string
	Model    string
	Endpoint string

	// Token 统计
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int

	// 成本
	Cost *Cost

	// 状态
	Success   bool
	ErrorCode string
	ErrorMsg  string
}

// Cost 成本信息
type Cost struct {
	PromptCost     float64
	CompletionCost float64
	TotalCost      float64
	Currency       string
}

// Collector 使用量收集器接口
type Collector interface {
	Record(ctx context.Context, record *UsageRecord) error
}

// collector 收集器实现
type collector struct {
	calculator Calculator
	recorder   Recorder
}

// NewCollector 创建收集器
func NewCollector(calc Calculator, rec Recorder) Collector {
	return &collector{
		calculator: calc,
		recorder:   rec,
	}
}

// Record 记录使用量
func (c *collector) Record(ctx context.Context, record *UsageRecord) error {
	// 计算成本
	if record.Cost == nil && c.calculator != nil {
		cost, err := c.calculator.Calculate(record.Provider, record.Model, record.PromptTokens, record.CompletionTokens)
		if err == nil {
			record.Cost = cost
		}
	}

	// 异步记录（这里简化为同步）
	if c.recorder != nil {
		return c.recorder.Save(ctx, record)
	}

	return nil
}
