package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	"github.com/Jayleonc/ai-gateway/internal/gateway/middleware"
	"github.com/Jayleonc/ai-gateway/internal/gateway/streaming"
	"github.com/Jayleonc/ai-gateway/internal/identity"
	"github.com/Jayleonc/ai-gateway/internal/metering"
	"github.com/Jayleonc/ai-gateway/internal/policy"
	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
	"github.com/Jayleonc/ai-gateway/internal/provider"
	transportopenai "github.com/Jayleonc/ai-gateway/internal/transport/openai"
	"github.com/Jayleonc/ai-gateway/pkg/openai"
)

// ChatHandler 聊天补全处理器
type ChatHandler struct {
	authenticator    identity.Authenticator
	policyEngine     policy.Engine
	providerRegistry provider.Registry
	quotaStore       quota.QuotaStore
	meteringRecorder metering.Recorder
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(auth identity.Authenticator, pe policy.Engine, pr provider.Registry, rec metering.Recorder) *ChatHandler {
	return &ChatHandler{
		authenticator:    auth,
		policyEngine:     pe,
		providerRegistry: pr,
		quotaStore:       nil,
		meteringRecorder: rec,
	}
}

// NewChatHandlerWithQuota 创建带配额管理的聊天处理器
func NewChatHandlerWithQuota(auth identity.Authenticator, pe policy.Engine, pr provider.Registry, qs quota.QuotaStore, rec metering.Recorder) *ChatHandler {
	return &ChatHandler{
		authenticator:    auth,
		policyEngine:     pe,
		providerRegistry: pr,
		quotaStore:       qs,
		meteringRecorder: rec,
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

type providerToStreamingReader struct {
	r provider.StreamReader
}

func (r *providerToStreamingReader) Next() (*streaming.Delta, error) {
	for {
		e, err := r.r.Recv()
		if err != nil {
			return nil, err
		}
		if e == nil {
			return nil, io.EOF
		}
		if e.Delta == nil {
			continue
		}
		return &streaming.Delta{Text: e.Delta.Content}, nil
	}
}

func (r *providerToStreamingReader) Close() error {
	return r.r.Close()
}

var _ streaming.StreamReader = (*providerToStreamingReader)(nil)

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
		RequestID: middleware.GetRequestID(c),
		APIKeyID:  reqCtx.APIKeyID,
		Provider:  decision.TargetProvider,
		Model:     decision.TargetModel,
		StartAt:   time.Now(),
	}

	writer := c.Writer
	flusher, _ := writer.(http.Flusher)

	// 创建观察者组（包含配额观察者）
	var observers []streaming.StreamObserver
	if h.quotaStore != nil {
		observers = append(observers, streaming.NewQuotaObserver(h.quotaStore))
	}
	observers = append(observers, &streamWriteObserver{w: writer, flusher: flusher})
	if h.meteringRecorder != nil {
		observers = append(observers, streaming.NewMeteringObserver(h.meteringRecorder))
	}
	observerGroup := streaming.NewObserverGroup(observers...)

	// 创建 runtime
	rt := streaming.NewRuntime(sctx, observerGroup)

	// 创建 stub stream reader（演示用）
	// 实际应用中应该从 provider 的响应读取真实 SSE 流
	providerReader, err := p.ChatStream(c.Request.Context(), providerReq)
	if err != nil {
		plan := transportopenai.MapStreamEnd(sctx, err)
		switch plan.Action {
		case transportopenai.StreamEndActionWriteJSONError:
			if plan.Error != nil {
				c.JSON(plan.Error.HTTPStatus, plan.Error.Body)
				return
			}
			c.Status(http.StatusInternalServerError)
			return
		case transportopenai.StreamEndActionCloseStream:
			c.Status(http.StatusOK)
			return
		case transportopenai.StreamEndActionNone:
			// continue
		default:
			c.Status(http.StatusInternalServerError)
			return
		}
	}
	reader := &providerToStreamingReader{r: providerReader}

	// 执行 streaming 流程
	err = rt.Run(reader)

	// 输出摘要日志（无论成功或失败）
	rt.LogSummary()

	plan := transportopenai.MapStreamEnd(sctx, err)
	switch plan.Action {
	case transportopenai.StreamEndActionWriteJSONError:
		if plan.Error != nil {
			c.JSON(plan.Error.HTTPStatus, plan.Error.Body)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	case transportopenai.StreamEndActionCloseStream:
		return
	case transportopenai.StreamEndActionNone:
		return
	default:
		c.Status(http.StatusInternalServerError)
		return
	}
}

type streamWriteObserver struct {
	w       gin.ResponseWriter
	flusher http.Flusher
}

func (o *streamWriteObserver) OnFirstChunk(ctx *streaming.StreamingContext) error {
	if o == nil || o.w == nil {
		return nil
	}

	o.w.Header().Set("Content-Type", "text/event-stream")
	o.w.Header().Set("Cache-Control", "no-cache")
	o.w.Header().Set("Connection", "keep-alive")
	o.w.WriteHeader(http.StatusOK)
	if o.flusher != nil {
		o.flusher.Flush()
	}
	return nil
}

func (o *streamWriteObserver) OnChunk(ctx *streaming.StreamingContext, d *streaming.Delta) (stop bool, err error) {
	if o == nil || o.w == nil || d == nil {
		return false, nil
	}

	if _, err := fmt.Fprintf(o.w, "data: %s\n\n", d.Text); err != nil {
		return false, err
	}
	if o.flusher != nil {
		o.flusher.Flush()
	}
	return false, nil
}

func (o *streamWriteObserver) OnEnd(ctx *streaming.StreamingContext) error {
	if o == nil || o.w == nil {
		return nil
	}
	if ctx == nil || !ctx.IsStarted() {
		return nil
	}

	if _, err := fmt.Fprint(o.w, "data: [DONE]\n\n"); err != nil {
		return err
	}
	if o.flusher != nil {
		o.flusher.Flush()
	}
	return nil
}

var _ streaming.StreamObserver = (*streamWriteObserver)(nil)
