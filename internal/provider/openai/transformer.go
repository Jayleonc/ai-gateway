package openai

import (
	"github.com/Jayleonc/ai-gateway/internal/provider"
	"github.com/Jayleonc/ai-gateway/pkg/openai"
)

// Transformer 请求/响应转换器
type Transformer struct{}

// NewTransformer 创建转换器
func NewTransformer() *Transformer {
	return &Transformer{}
}

// TransformChatRequest 转换聊天请求
func (t *Transformer) TransformChatRequest(req *provider.ChatRequest) *openai.ChatCompletionRequest {
	messages := make([]openai.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openai.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	return &openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
		Stop:        req.Stop,
		User:        req.User,
	}
}

// TransformChatResponse 转换聊天响应
func (t *Transformer) TransformChatResponse(resp *openai.ChatCompletionResponse) *provider.ChatResponse {
	choices := make([]provider.ChatChoice, len(resp.Choices))
	for i, c := range resp.Choices {
		var msg *provider.Message
		if c.Message != nil {
			msg = &provider.Message{
				Role:    c.Message.Role,
				Content: c.Message.Content,
			}
		}
		choices[i] = provider.ChatChoice{
			Index:        c.Index,
			Message:      msg,
			FinishReason: c.FinishReason,
		}
	}

	var usage *provider.Usage
	if resp.Usage != nil {
		usage = &provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return &provider.ChatResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Choices: choices,
		Usage:   usage,
	}
}
