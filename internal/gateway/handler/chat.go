package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	"github.com/Jayleonc/ai-gateway/internal/identity"
	"github.com/Jayleonc/ai-gateway/internal/policy"
	"github.com/Jayleonc/ai-gateway/internal/provider"
	"github.com/Jayleonc/ai-gateway/pkg/openai"
)

// ChatHandler 聊天补全处理器
type ChatHandler struct {
	authenticator    identity.Authenticator
	policyEngine     policy.Engine
	providerRegistry provider.Registry
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(auth identity.Authenticator, pe policy.Engine, pr provider.Registry) *ChatHandler {
	return &ChatHandler{
		authenticator:    auth,
		policyEngine:     pe,
		providerRegistry: pr,
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
