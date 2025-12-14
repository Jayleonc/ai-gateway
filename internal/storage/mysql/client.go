package mysql

import (
	"context"

	"github.com/Jayleonc/ai-gateway/internal/storage"
)

// Client MySQL 客户端
type Client struct {
	dsn string
}

// NewClient 创建 MySQL 客户端
func NewClient(dsn string) *Client {
	return &Client{dsn: dsn}
}

// Ping 健康检查
func (c *Client) Ping(ctx context.Context) error {
	// TODO: implement
	return nil
}

// BeginTx 开启事务
func (c *Client) BeginTx(ctx context.Context) (storage.Transaction, error) {
	// TODO: implement
	return nil, nil
}

// WithTx 在事务中执行
func (c *Client) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	// TODO: implement
	return nil
}

// 确保实现接口
var (
	_ storage.HealthChecker = (*Client)(nil)
	_ storage.Transactional = (*Client)(nil)
)
