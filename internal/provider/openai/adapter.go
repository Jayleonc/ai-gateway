package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	"github.com/Jayleonc/ai-gateway/internal/provider"
	openaitypes "github.com/Jayleonc/ai-gateway/pkg/openai"
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
	if req == nil {
		return nil, coreerrors.ErrBadRequest
	}
	if strings.TrimSpace(a.apiKey) == "" {
		return nil, coreerrors.ErrUnauthorized
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, coreerrors.ErrBadRequest
	}
	if req.Stream {
		return nil, coreerrors.ErrBadRequest
	}

	openaiReq := &openaitypes.ChatCompletionRequest{
		Model:  req.Model,
		Stream: false,
	}
	if len(req.Messages) > 0 {
		openaiReq.Messages = make([]openaitypes.Message, 0, len(req.Messages))
		for _, m := range req.Messages {
			openaiReq.Messages = append(openaiReq.Messages, openaitypes.Message{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(a.baseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr openaitypes.ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Error != nil && apiErr.Error.Message != "" {
			return nil, coreerrors.NewAPIError("openai_error", apiErr.Error.Message, apiErr.Error.Type)
		}
		return nil, coreerrors.ErrProviderError
	}

	var openaiResp openaitypes.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, err
	}

	out := &provider.ChatResponse{
		ID:    openaiResp.ID,
		Model: openaiResp.Model,
	}
	if len(openaiResp.Choices) > 0 {
		out.Choices = make([]provider.ChatChoice, 0, len(openaiResp.Choices))
		for _, c := range openaiResp.Choices {
			var msg *provider.Message
			if c.Message != nil {
				msg = &provider.Message{Role: c.Message.Role, Content: c.Message.Content}
			}
			out.Choices = append(out.Choices, provider.ChatChoice{
				Index:        c.Index,
				Message:      msg,
				FinishReason: c.FinishReason,
			})
		}
	}
	if openaiResp.Usage != nil {
		out.Usage = &provider.Usage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		}
	}

	return out, nil
}

// ChatStream 执行流式聊天补全
func (a *Adapter) ChatStream(ctx context.Context, req *provider.ChatRequest) (provider.StreamReader, error) {
	if req == nil {
		return nil, coreerrors.ErrBadRequest
	}
	if strings.TrimSpace(a.apiKey) == "" {
		return nil, coreerrors.ErrUnauthorized
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, coreerrors.ErrBadRequest
	}
	if !req.Stream {
		return nil, coreerrors.ErrBadRequest
	}

	openaiReq := (&Transformer{}).TransformChatRequest(req)
	openaiReq.Stream = true

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(a.baseURL, "/") + openaitypes.EndpointChatCompletions
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{Timeout: 0}
	r, err := NewStreamReader(ctx, client, httpReq)
	if err != nil {
		return nil, err
	}
	return &providerStreamReader{r: r}, nil
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
