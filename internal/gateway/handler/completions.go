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

// CompletionsHandler 文本补全处理器
type CompletionsHandler struct {
	authenticator    identity.Authenticator
	policyEngine     policy.Engine
	providerRegistry provider.Registry
}

// NewCompletionsHandler 创建补全处理器
func NewCompletionsHandler(auth identity.Authenticator, pe policy.Engine, pr provider.Registry) *CompletionsHandler {
	return &CompletionsHandler{
		authenticator:    auth,
		policyEngine:     pe,
		providerRegistry: pr,
	}
}

// Handle 处理文本补全请求
func (h *CompletionsHandler) Handle(c *gin.Context) {
	// 1. 解析请求
	var req openai.CompletionRequest
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
		case "quota_exceeded":
			WriteOpenAIError(c, coreerrors.NewAPIErrorWithStatus("quota_exceeded", decision.DenyReason, "rate_limit_error", http.StatusTooManyRequests))
		case "rate_limited":
			WriteOpenAIError(c, coreerrors.NewAPIErrorWithStatus("rate_limited", decision.DenyReason, "rate_limit_error", http.StatusTooManyRequests))
		default:
			WriteOpenAIError(c, coreerrors.NewAPIErrorWithStatus("rate_limited", decision.DenyReason, "rate_limit_error", http.StatusTooManyRequests))
		}
		return
	}

	// 4. 获取 Provider
	p, ok := h.providerRegistry.Get(decision.TargetProvider)
	if !ok {
		WriteOpenAIError(c, coreerrors.ErrProviderNotFound)
		return
	}

	// 5. 调用 Provider
	prompt := ""
	if s, ok := req.Prompt.(string); ok {
		prompt = s
	}

	providerReq := &provider.CompletionRequest{
		Model:  decision.TargetModel,
		Prompt: prompt,
		Stream: req.Stream,
	}

	resp, err := p.Complete(c.Request.Context(), providerReq)
	if err != nil {
		WriteOpenAIError(c, err)
		return
	}

	// 6. 返回响应
	choices := make([]openai.CompletionChoice, len(resp.Choices))
	for i, ch := range resp.Choices {
		choices[i] = openai.CompletionChoice{
			Index:        ch.Index,
			Text:         ch.Text,
			FinishReason: ch.FinishReason,
		}
	}

	c.JSON(http.StatusOK, openai.CompletionResponse{
		ID:      resp.ID,
		Object:  openai.ObjectCompletion,
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: choices,
		Usage:   convertUsage(resp.Usage),
	})
}
