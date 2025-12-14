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
		c.JSON(http.StatusBadGateway, openai.ErrorResponse{
			Error: &openai.ErrorDetail{
				Message: "provider error",
				Type:    "server_error",
			},
		})
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
