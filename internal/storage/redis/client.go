package redis

import (
	"context"

	"github.com/Jayleonc/ai-gateway/internal/storage"
)

// Client Redis 客户端
type Client struct {
	addr     string
	password string
	db       int
}

// NewClient 创建 Redis 客户端
func NewClient(addr, password string, db int) *Client {
	return &Client{
		addr:     addr,
		password: password,
		db:       db,
	}
}

// Ping 健康检查
func (c *Client) Ping(ctx context.Context) error {
	// TODO: implement
	return nil
}

// 确保实现接口
var _ storage.HealthChecker = (*Client)(nil)
