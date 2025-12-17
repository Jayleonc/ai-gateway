package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	"github.com/Jayleonc/ai-gateway/internal/gateway/middleware"
	"github.com/Jayleonc/ai-gateway/internal/identity"
	"github.com/Jayleonc/ai-gateway/internal/metering"
	"github.com/Jayleonc/ai-gateway/internal/policy"
	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
	"github.com/Jayleonc/ai-gateway/internal/provider"
	openaitypes "github.com/Jayleonc/ai-gateway/pkg/openai"
)

type stubAuthenticator struct {
	reqCtx *identity.RequestContext
	err    error
}

func (a *stubAuthenticator) Authenticate(ctx context.Context, apiKey string) (*identity.RequestContext, error) {
	return a.reqCtx, a.err
}

type stubPolicyEngine struct {
	decision *policy.Decision
	err      error
}

func (e *stubPolicyEngine) Evaluate(ctx context.Context, reqCtx *identity.RequestContext, req *policy.EvaluateRequest) (*policy.Decision, error) {
	return e.decision, e.err
}

type stubProvider struct {
	name          string
	supported     []string
	chatStreamErr error
	chatStreamR   provider.StreamReader
}

func (p *stubProvider) Name() string { return p.name }

func (p *stubProvider) SupportedModels() []string { return p.supported }

func (p *stubProvider) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return nil, coreerrors.ErrNotImplemented
}

func (p *stubProvider) ChatStream(ctx context.Context, req *provider.ChatRequest) (provider.StreamReader, error) {
	if p.chatStreamErr != nil {
		return nil, p.chatStreamErr
	}
	return p.chatStreamR, nil
}

func (p *stubProvider) Complete(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return nil, coreerrors.ErrNotImplemented
}

type sliceStreamReader struct {
	events []*provider.StreamEvent
	i      int
}

func (r *sliceStreamReader) Recv() (*provider.StreamEvent, error) {
	if r.i >= len(r.events) {
		return nil, io.EOF
	}
	e := r.events[r.i]
	r.i++
	return e, nil
}

func (r *sliceStreamReader) Close() error { return nil }

func newTestRouter(h *ChatHandler, requestID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.RequestIDKey, requestID)
		c.Next()
	})
	r.POST("/v1/chat/completions", h.Handle)
	return r
}

func doStreamRequest(t *testing.T, r http.Handler, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestStreaming_BeforeFirstChunk_Provider401_ReturnsJSON(t *testing.T) {
	auth := &stubAuthenticator{reqCtx: &identity.RequestContext{APIKeyID: "key_401"}}
	pe := &stubPolicyEngine{decision: &policy.Decision{Allowed: true, TargetProvider: "p", TargetModel: "m"}}
	pr := provider.NewRegistry()
	pr.Register(&stubProvider{name: "p", supported: []string{"m"}, chatStreamErr: coreerrors.ErrInvalidAPIKey})

	h := NewChatHandler(auth, pe, pr, nil)
	r := newTestRouter(h, "req_401")

	w := doStreamRequest(t, r, map[string]any{
		"model":  "m",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	})

	res := w.Result()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status=401, got %d", res.StatusCode)
	}
	ct := res.Header.Get("Content-Type")
	if strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected JSON response, got Content-Type=%q", ct)
	}

	var er openaitypes.ErrorResponse
	if err := json.NewDecoder(res.Body).Decode(&er); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if er.Error == nil || er.Error.Type == "" {
		t.Fatalf("expected openai-style error body, got %#v", er)
	}
}

func TestStreaming_BeforeFirstChunk_Provider429_ReturnsJSON(t *testing.T) {
	auth := &stubAuthenticator{reqCtx: &identity.RequestContext{APIKeyID: "key_429"}}
	pe := &stubPolicyEngine{decision: &policy.Decision{Allowed: true, TargetProvider: "p", TargetModel: "m"}}
	pr := provider.NewRegistry()
	pr.Register(&stubProvider{name: "p", supported: []string{"m"}, chatStreamErr: coreerrors.ErrQuotaExceeded})

	h := NewChatHandler(auth, pe, pr, nil)
	r := newTestRouter(h, "req_429")

	w := doStreamRequest(t, r, map[string]any{
		"model":  "m",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	})

	res := w.Result()
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected status=429, got %d", res.StatusCode)
	}
	ct := res.Header.Get("Content-Type")
	if strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected JSON response, got Content-Type=%q", ct)
	}

	var er openaitypes.ErrorResponse
	if err := json.NewDecoder(res.Body).Decode(&er); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if er.Error == nil || er.Error.Type != coreerrors.OpenAIErrorTypeRateLimit {
		t.Fatalf("expected rate_limit_error body, got %#v", er)
	}
}

func TestStreaming_QuotaExceeded_Partial_WithDone_UsageRecord(t *testing.T) {
	auth := &stubAuthenticator{reqCtx: &identity.RequestContext{APIKeyID: "key_quota"}}
	pe := &stubPolicyEngine{decision: &policy.Decision{Allowed: true, TargetProvider: "p", TargetModel: "m"}}
	pr := provider.NewRegistry()

	sr := &sliceStreamReader{events: []*provider.StreamEvent{
		{Delta: &provider.Message{Role: "assistant", Content: "1234"}},
		{Delta: &provider.Message{Role: "assistant", Content: "5678"}},
	}}
	pr.Register(&stubProvider{name: "p", supported: []string{"m"}, chatStreamR: sr})

	qs := quota.NewInMemoryStore()
	qs.SetRemaining("key_quota", 1)

	rec := metering.NewRecorder()
	mem := rec.(*metering.InMemoryRecorder)

	h := NewChatHandlerWithQuota(auth, pe, pr, qs, rec)
	r := newTestRouter(h, "req_quota")

	w := doStreamRequest(t, r, map[string]any{
		"model":  "m",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	})

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status=200, got %d", res.StatusCode)
	}
	body := w.Body.String()
	if !strings.Contains(body, "data: 1234\n\n") {
		t.Fatalf("expected first chunk delivered, got body=%q", body)
	}
	if strings.Contains(body, "data: 5678\n\n") {
		t.Fatalf("expected second chunk blocked by quota, got body=%q", body)
	}
	if !strings.Contains(body, "data: [DONE]\n\n") {
		t.Fatalf("expected DONE, got body=%q", body)
	}

	record := mem.GetByRequestID("req_quota")
	if record == nil {
		t.Fatalf("expected usage record")
	}
	if record.Status != "partial" {
		t.Fatalf("expected status=partial, got %q", record.Status)
	}
	if record.EndReason != "quota_exceeded" {
		t.Fatalf("expected end_reason=quota_exceeded, got %q", record.EndReason)
	}
}

func TestStreaming_Normal_Completed_WithDone_UsageRecord(t *testing.T) {
	auth := &stubAuthenticator{reqCtx: &identity.RequestContext{APIKeyID: "key_ok"}}
	pe := &stubPolicyEngine{decision: &policy.Decision{Allowed: true, TargetProvider: "p", TargetModel: "m"}}
	pr := provider.NewRegistry()

	sr := &sliceStreamReader{events: []*provider.StreamEvent{
		{Delta: &provider.Message{Role: "assistant", Content: "a"}},
		{Delta: &provider.Message{Role: "assistant", Content: "b"}},
		{Delta: &provider.Message{Role: "assistant", Content: "c"}},
	}}
	pr.Register(&stubProvider{name: "p", supported: []string{"m"}, chatStreamR: sr})

	rec := metering.NewRecorder()
	mem := rec.(*metering.InMemoryRecorder)

	h := NewChatHandler(auth, pe, pr, rec)

	r := newTestRouter(h, "req_ok")

	w := doStreamRequest(t, r, map[string]any{
		"model":  "m",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	})

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status=200, got %d", res.StatusCode)
	}
	body := w.Body.String()
	if !strings.Contains(body, "data: a\n\n") || !strings.Contains(body, "data: b\n\n") || !strings.Contains(body, "data: c\n\n") {
		t.Fatalf("expected multiple chunks, got body=%q", body)
	}
	if !strings.Contains(body, "data: [DONE]\n\n") {
		t.Fatalf("expected DONE, got body=%q", body)
	}

	record := mem.GetByRequestID("req_ok")
	if record == nil {
		t.Fatalf("expected usage record")
	}
	if record.Status != "completed" {
		t.Fatalf("expected status=completed, got %q", record.Status)
	}
	if record.EndReason != "stop" {
		t.Fatalf("expected end_reason=stop, got %q", record.EndReason)
	}
}

var _ identity.Authenticator = (*stubAuthenticator)(nil)
var _ policy.Engine = (*stubPolicyEngine)(nil)
var _ provider.Provider = (*stubProvider)(nil)
var _ provider.StreamReader = (*sliceStreamReader)(nil)
