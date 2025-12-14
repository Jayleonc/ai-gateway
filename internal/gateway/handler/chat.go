package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	"github.com/Jayleonc/ai-gateway/internal/gateway/streaming"
	"github.com/Jayleonc/ai-gateway/internal/identity"
	"github.com/Jayleonc/ai-gateway/internal/policy"
	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
	"github.com/Jayleonc/ai-gateway/internal/provider"
	"github.com/Jayleonc/ai-gateway/pkg/openai"
)

// ChatHandler 聊天补全处理器
type ChatHandler struct {
	authenticator    identity.Authenticator
	policyEngine     policy.Engine
	providerRegistry provider.Registry
	quotaStore       quota.QuotaStore
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(auth identity.Authenticator, pe policy.Engine, pr provider.Registry) *ChatHandler {
	return &ChatHandler{
		authenticator:    auth,
		policyEngine:     pe,
		providerRegistry: pr,
		quotaStore:       nil,
	}
}

// NewChatHandlerWithQuota 创建带配额管理的聊天处理器
func NewChatHandlerWithQuota(auth identity.Authenticator, pe policy.Engine, pr provider.Registry, qs quota.QuotaStore) *ChatHandler {
	return &ChatHandler{
		authenticator:    auth,
		policyEngine:     pe,
		providerRegistry: pr,
		quotaStore:       qs,
	}
}

// Handle 处理聊天补全请求
func (h *ChatHandler) Handle(c *gin.Context) {
	// 1. 解析请求
	var req openai.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteOpenAIError(c, coreerrors.NewAPIErrorWithStatus("invalid_request", "invalid request body", "invalid_request_error", http.StatusBadRequest))
		return
	}

	// 2. 认证
	apiKey := extractAPIKey(c)
	reqCtx, err := h.authenticator.Authenticate(c.Request.Context(), apiKey)
	if err != nil {
		WriteOpenAIError(c, err)
		return
	}

	// 3. 策略评估
	decision, err := h.policyEngine.Evaluate(c.Request.Context(), reqCtx, &policy.EvaluateRequest{
		Model: req.Model,
	})
	if err != nil {
		WriteOpenAIError(c, err)
		return
	}
	if !decision.Allowed {
		switch decision.DenyCode {
		case policy.DenyCodeQuotaExceeded:
			WriteOpenAIError(c, coreerrors.NewAPIErrorWithStatus(decision.DenyCode, decision.DenyReason, coreerrors.OpenAIErrorTypeRateLimit, http.StatusTooManyRequests))
		case policy.DenyCodeRateLimited:
			WriteOpenAIError(c, coreerrors.NewAPIErrorWithStatus(decision.DenyCode, decision.DenyReason, coreerrors.OpenAIErrorTypeRateLimit, http.StatusTooManyRequests))
		default:
			WriteOpenAIError(c, coreerrors.NewAPIErrorWithStatus(decision.DenyCode, decision.DenyReason, coreerrors.OpenAIErrorTypeRateLimit, http.StatusTooManyRequests))
		}
		return
	}

	// 4. 获取 Provider
	p, ok := h.providerRegistry.Get(decision.TargetProvider)
	if !ok {
		WriteOpenAIError(c, coreerrors.ErrProviderNotFound)
		return
	}

	// 5. 调用 Provider（stub 响应）
	providerReq := &provider.ChatRequest{
		Model:  decision.TargetModel,
		Stream: req.Stream,
	}
	for _, m := range req.Messages {
		providerReq.Messages = append(providerReq.Messages, provider.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// 检查是否为 streaming 模式
	if req.Stream {
		h.handleStreamingChat(c, reqCtx, decision, p, providerReq)
		return
	}

	resp, err := p.Chat(c.Request.Context(), providerReq)
	if err != nil {
		WriteOpenAIError(c, err)
		return
	}

	// 6. 返回响应
	c.JSON(http.StatusOK, openai.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  openai.ObjectChatCompletion,
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: convertChoices(resp.Choices),
		Usage:   convertUsage(resp.Usage),
	})
}

func extractAPIKey(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}

func convertChoices(choices []provider.ChatChoice) []openai.Choice {
	result := make([]openai.Choice, len(choices))
	for i, c := range choices {
		var msg *openai.Message
		if c.Message != nil {
			msg = &openai.Message{
				Role:    c.Message.Role,
				Content: c.Message.Content,
			}
		}
		result[i] = openai.Choice{
			Index:        c.Index,
			Message:      msg,
			FinishReason: c.FinishReason,
		}
	}
	return result
}

func convertUsage(usage *provider.Usage) *openai.Usage {
	if usage == nil {
		return nil
	}
	return &openai.Usage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

// handleStreamingChat 处理 streaming 模式的聊天请求
func (h *ChatHandler) handleStreamingChat(c *gin.Context, reqCtx *identity.RequestContext, decision *policy.Decision, p provider.Provider, providerReq *provider.ChatRequest) {
	// 创建 streaming context
	sctx := &streaming.StreamingContext{
		RequestID: c.GetString("request_id"),
		APIKeyID:  reqCtx.APIKeyID,
		Provider:  decision.TargetProvider,
		Model:     decision.TargetModel,
		StartAt:   time.Now(),
	}

	// 创建观察者组（包含配额观察者）
	var observers []streaming.StreamObserver
	if h.quotaStore != nil {
		observers = append(observers, streaming.NewQuotaObserver(h.quotaStore))
	}
	observerGroup := streaming.NewObserverGroup(observers...)

	// 创建 runtime
	rt := streaming.NewRuntime(sctx, observerGroup)

	// 创建 stub stream reader（演示用）
	// 实际应用中应该从 provider 的响应读取真实 SSE 流
	stubChunks := []*streaming.Delta{
		{Text: "Hello"},
		{Text: " "},
		{Text: "world"},
		{Text: "!"},
	}
	reader := streaming.NewStubStreamReader(stubChunks, 0)

	// 执行 streaming 流程
	if err := rt.Run(reader); err != nil {
		// 如果是配额超限，返回 429
		if sctx.EndReason == streaming.EndReasonQuotaExceeded {
			WriteOpenAIError(c, coreerrors.NewAPIErrorWithStatus(policy.ErrQuotaExceeded, "quota exceeded during streaming", coreerrors.OpenAIErrorTypeRateLimit, http.StatusTooManyRequests))
			return
		}
		WriteOpenAIError(c, err)
		return
	}

	// 输出摘要日志
	rt.LogSummary()

	// 返回 streaming 响应（简化版：直接返回 JSON）
	// 实际应该返回 SSE 格式的流式响应
	c.JSON(http.StatusOK, gin.H{
		"id":      "chatcmpl-stub",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   decision.TargetModel,
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": "Hello world!",
				},
				"finish_reason": "stop",
			},
		},
		"usage": gin.H{
			"prompt_tokens":     0,
			"completion_tokens": sctx.ConfirmedTokens,
			"total_tokens":      sctx.ConfirmedTokens,
		},
	})
}
