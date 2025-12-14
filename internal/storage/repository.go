package storage

import "context"

// Transaction 事务接口
type Transaction interface {
	Commit() error
	Rollback() error
}

// Transactional 支持事务的存储
type Transactional interface {
	BeginTx(ctx context.Context) (Transaction, error)
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// HealthChecker 健康检查接口
type HealthChecker interface {
	Ping(ctx context.Context) error
}
