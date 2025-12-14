package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

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
		c.JSON(http.StatusBadRequest, openai.ErrorResponse{
			Error: &openai.ErrorDetail{
				Message: "invalid request body",
				Type:    "invalid_request_error",
			},
		})
		return
	}

	// 2. 认证
	apiKey := extractAPIKey(c)
	reqCtx, err := h.authenticator.Authenticate(c.Request.Context(), apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, openai.ErrorResponse{
			Error: &openai.ErrorDetail{
				Message: "invalid api key",
				Type:    "authentication_error",
			},
		})
		return
	}

	// 3. 策略评估
	decision, err := h.policyEngine.Evaluate(c.Request.Context(), reqCtx, &policy.EvaluateRequest{
		Model: req.Model,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, openai.ErrorResponse{
			Error: &openai.ErrorDetail{
				Message: "policy evaluation failed",
				Type:    "server_error",
			},
		})
		return
	}
	if !decision.Allowed {
		c.JSON(http.StatusTooManyRequests, openai.ErrorResponse{
			Error: &openai.ErrorDetail{
				Message: decision.DenyReason,
				Type:    "rate_limit_error",
			},
		})
		return
	}

	// 4. 获取 Provider
	p, ok := h.providerRegistry.Get(decision.TargetProvider)
	if !ok {
		c.JSON(http.StatusBadGateway, openai.ErrorResponse{
			Error: &openai.ErrorDetail{
				Message: "provider not available",
				Type:    "server_error",
			},
		})
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
		c.JSON(http.StatusBadGateway, openai.ErrorResponse{
			Error: &openai.ErrorDetail{
				Message: "provider error",
				Type:    "server_error",
			},
		})
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
