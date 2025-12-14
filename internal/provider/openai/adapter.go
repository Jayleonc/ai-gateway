package openai

import (
	"context"

	"github.com/Jayleonc/ai-gateway/internal/provider"
)

// Adapter OpenAI Provider 适配器
type Adapter struct {
	apiKey  string
	baseURL string
}

// NewAdapter 创建 OpenAI 适配器
func NewAdapter(apiKey, baseURL string) *Adapter {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &Adapter{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

// Name 返回 Provider 名称
func (a *Adapter) Name() string {
	return "openai"
}

// SupportedModels 返回支持的模型
func (a *Adapter) SupportedModels() []string {
	return []string{
		"gpt-4",
		"gpt-4-turbo",
		"gpt-4-turbo-preview",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-16k",
	}
}

// Chat 执行聊天补全
func (a *Adapter) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	// TODO: implement real OpenAI API call
	return &provider.ChatResponse{
		ID:    "stub-chat-id",
		Model: req.Model,
		Choices: []provider.ChatChoice{
			{
				Index: 0,
				Message: &provider.Message{
					Role:    "assistant",
					Content: "This is a stub response",
				},
				FinishReason: "stop",
			},
		},
		Usage: &provider.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

// ChatStream 执行流式聊天补全
func (a *Adapter) ChatStream(ctx context.Context, req *provider.ChatRequest) (provider.StreamReader, error) {
	// TODO: implement real streaming
	return &stubStreamReader{}, nil
}

// Complete 执行文本补全
func (a *Adapter) Complete(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	// TODO: implement real OpenAI API call
	return &provider.CompletionResponse{
		ID:    "stub-completion-id",
		Model: req.Model,
		Choices: []provider.CompletionChoice{
			{
				Index:        0,
				Text:         "This is a stub completion",
				FinishReason: "stop",
			},
		},
		Usage: &provider.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

// 确保实现 Provider 接口
var _ provider.Provider = (*Adapter)(nil)

// stubStreamReader 占位流读取器
type stubStreamReader struct {
	done bool
}

func (s *stubStreamReader) Recv() (*provider.StreamEvent, error) {
	if s.done {
		return nil, nil
	}
	s.done = true
	return &provider.StreamEvent{
		ID:    "stub-stream-id",
		Model: "gpt-3.5-turbo",
		Delta: &provider.Message{
			Role:    "assistant",
			Content: "Stub stream response",
		},
		FinishReason: "stop",
	}, nil
}

func (s *stubStreamReader) Close() error {
	return nil
}
