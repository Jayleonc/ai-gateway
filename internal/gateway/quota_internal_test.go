package gateway_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jayleonc/ai-gateway/internal/gateway"
	"github.com/Jayleonc/ai-gateway/internal/metering"
	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
)

func TestInternalQuotaSnapshot(t *testing.T) {
	apiKeyID := "gw-key-001"

	recorder := metering.NewRecorder()
	mem := recorder.(*metering.InMemoryRecorder)
	usageQuery := metering.NewInMemoryUsageQuery(mem)

	qs := quota.NewInMemoryStore()
	qs.SetRemaining(apiKeyID, 7)

	now := time.Now().UTC()
	r1 := &metering.UsageRecord{
		RequestID:       "req_1",
		APIKeyID:        apiKeyID,
		ConfirmedTokens: 3,
		RequestedAt:     now.Add(-2 * time.Minute),
		CompletedAt:     now.Add(-2 * time.Minute),
	}
	r2 := &metering.UsageRecord{
		RequestID:       "req_2",
		APIKeyID:        apiKeyID,
		ConfirmedTokens: 5,
		RequestedAt:     now.Add(-1 * time.Minute),
		CompletedAt:     now.Add(-1 * time.Minute),
	}
	_ = mem.Save(context.Background(), r1)
	_ = mem.Save(context.Background(), r2)

	r := gateway.SetupRouter(&gateway.RouterConfig{
		QuotaStore: qs,
		UsageQuery: usageQuery,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/quota/"+apiKeyID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status=200 got=%d body=%q", w.Code, w.Body.String())
	}

	var out struct {
		QuotaTotal     int64      `json:"quota_total"`
		QuotaUsed      int64      `json:"quota_used"`
		QuotaRemaining int64      `json:"quota_remaining"`
		LastUpdatedAt  *time.Time `json:"last_updated_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.QuotaUsed != 8 {
		t.Fatalf("expected quota_used=8 got=%d", out.QuotaUsed)
	}
	if out.QuotaRemaining != 7 {
		t.Fatalf("expected quota_remaining=7 got=%d", out.QuotaRemaining)
	}
	if out.QuotaTotal != 15 {
		t.Fatalf("expected quota_total=15 got=%d", out.QuotaTotal)
	}
	if out.LastUpdatedAt == nil {
		t.Fatalf("expected last_updated_at set")
	}
}
