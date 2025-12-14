package provider

import (
	"context"
	"io"
)

// Provider AI Provider 统一接口
type Provider interface {
	// Name 返回 Provider 标识符
	Name() string

	// SupportedModels 返回支持的模型列表
	SupportedModels() []string

	// Chat 执行聊天补全（非流式）
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream 执行聊天补全（流式）
	ChatStream(ctx context.Context, req *ChatRequest) (StreamReader, error)

	// Complete 执行文本补全（非流式）
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
}

// StreamReader 流式响应读取器
type StreamReader interface {
	// Recv 接收下一个流式事件
	Recv() (*StreamEvent, error)

	// Close 关闭流
	Close() error
}

// 确保 io.EOF 可用于标识流结束
var _ error = io.EOF
