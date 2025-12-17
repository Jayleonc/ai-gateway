package gateway_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	"github.com/Jayleonc/ai-gateway/internal/gateway"
	"github.com/Jayleonc/ai-gateway/internal/gateway/middleware"
	"github.com/Jayleonc/ai-gateway/internal/identity"
	"github.com/Jayleonc/ai-gateway/internal/metering"
	"github.com/Jayleonc/ai-gateway/internal/policy"
	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
	"github.com/Jayleonc/ai-gateway/internal/provider"
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
	name      string
	supported []string
	reader    provider.StreamReader
}

func (p *stubProvider) Name() string { return p.name }

func (p *stubProvider) SupportedModels() []string { return p.supported }

func (p *stubProvider) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return nil, coreerrors.ErrNotImplemented
}

func (p *stubProvider) ChatStream(ctx context.Context, req *provider.ChatRequest) (provider.StreamReader, error) {
	return p.reader, nil
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

func doJSON(t *testing.T, h http.Handler, method, path string, requestID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test")
	req.Header.Set(middleware.RequestIDKey, requestID)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func doGET(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestInternalUsageQuery_NormalStreaming(t *testing.T) {
	requestID := "req_ok"
	apiKeyID := "key_ok"

	rec := metering.NewRecorder()
	mem := rec.(*metering.InMemoryRecorder)
	q := metering.NewInMemoryUsageQuery(mem)

	pr := provider.NewRegistry()
	sr := &sliceStreamReader{events: []*provider.StreamEvent{
		{Delta: &provider.Message{Role: "assistant", Content: "a"}},
		{Delta: &provider.Message{Role: "assistant", Content: "b"}},
		{Delta: &provider.Message{Role: "assistant", Content: "c"}},
	}}
	pr.Register(&stubProvider{name: "p", supported: []string{"m"}, reader: sr})

	auth := &stubAuthenticator{reqCtx: &identity.RequestContext{APIKeyID: apiKeyID}}
	pe := &stubPolicyEngine{decision: &policy.Decision{Allowed: true, TargetProvider: "p", TargetModel: "m"}}

	r := gateway.SetupRouter(&gateway.RouterConfig{
		Authenticator:    auth,
		PolicyEngine:     pe,
		ProviderRegistry: pr,
		MeteringRecorder: rec,
		UsageQuery:       q,
	})

	w := doJSON(t, r, http.MethodPost, "/v1/chat/completions", requestID, map[string]any{
		"model":    "m",
		"stream":   true,
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected status=200, got %d body=%q", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got == "" {
		t.Fatalf("expected SSE body")
	}

	g := doGET(t, r, "/internal/usage/"+requestID)
	if g.Code != http.StatusOK {
		t.Fatalf("expected status=200, got %d body=%q", g.Code, g.Body.String())
	}

	var view map[string]any
	if err := json.Unmarshal(g.Body.Bytes(), &view); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if view["request_id"] != requestID {
		t.Fatalf("expected request_id=%q got=%v", requestID, view["request_id"])
	}
	if view["api_key_id"] != apiKeyID {
		t.Fatalf("expected api_key_id=%q got=%v", apiKeyID, view["api_key_id"])
	}
	if view["status"] != "completed" {
		t.Fatalf("expected status=completed got=%v", view["status"])
	}
	if view["end_reason"] != "stop" {
		t.Fatalf("expected end_reason=stop got=%v", view["end_reason"])
	}

	l := doGET(t, r, "/internal/usage?api_key_id="+apiKeyID+"&limit=50")
	if l.Code != http.StatusOK {
		t.Fatalf("expected status=200, got %d body=%q", l.Code, l.Body.String())
	}
	var list []map[string]any
	if err := json.Unmarshal(l.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list) < 1 {
		t.Fatalf("expected at least 1 record")
	}
}

func TestInternalUsageQuery_QuotaExceeded_Partial(t *testing.T) {
	requestID := "req_quota"
	apiKeyID := "key_quota"

	rec := metering.NewRecorder()
	mem := rec.(*metering.InMemoryRecorder)
	q := metering.NewInMemoryUsageQuery(mem)

	pr := provider.NewRegistry()
	sr := &sliceStreamReader{events: []*provider.StreamEvent{
		{Delta: &provider.Message{Role: "assistant", Content: "1234"}},
		{Delta: &provider.Message{Role: "assistant", Content: "5678"}},
	}}
	pr.Register(&stubProvider{name: "p", supported: []string{"m"}, reader: sr})

	qs := quota.NewInMemoryStore()
	qs.SetRemaining(apiKeyID, 1)

	auth := &stubAuthenticator{reqCtx: &identity.RequestContext{APIKeyID: apiKeyID}}
	pe := &stubPolicyEngine{decision: &policy.Decision{Allowed: true, TargetProvider: "p", TargetModel: "m"}}

	r := gateway.SetupRouter(&gateway.RouterConfig{
		Authenticator:    auth,
		PolicyEngine:     pe,
		ProviderRegistry: pr,
		QuotaStore:       qs,
		MeteringRecorder: rec,
		UsageQuery:       q,
	})

	w := doJSON(t, r, http.MethodPost, "/v1/chat/completions", requestID, map[string]any{
		"model":    "m",
		"stream":   true,
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected status=200, got %d body=%q", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("data: 1234\n\n")) {
		t.Fatalf("expected first chunk delivered, got body=%q", body)
	}
	if bytes.Contains([]byte(body), []byte("data: 5678\n\n")) {
		t.Fatalf("expected second chunk blocked by quota, got body=%q", body)
	}
	if !bytes.Contains([]byte(body), []byte("data: [DONE]\n\n")) {
		t.Fatalf("expected DONE, got body=%q", body)
	}

	g := doGET(t, r, "/internal/usage/"+requestID)
	if g.Code != http.StatusOK {
		t.Fatalf("expected status=200, got %d body=%q", g.Code, g.Body.String())
	}

	var view map[string]any
	if err := json.Unmarshal(g.Body.Bytes(), &view); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if view["status"] != "partial" {
		t.Fatalf("expected status=partial got=%v", view["status"])
	}
	if view["end_reason"] != "quota_exceeded" {
		t.Fatalf("expected end_reason=quota_exceeded got=%v", view["end_reason"])
	}
}

var _ identity.Authenticator = (*stubAuthenticator)(nil)
var _ policy.Engine = (*stubPolicyEngine)(nil)
var _ provider.Provider = (*stubProvider)(nil)
var _ provider.StreamReader = (*sliceStreamReader)(nil)
