package streaming

import (
	"io"
	"testing"
	"time"

	"github.com/Jayleonc/ai-gateway/internal/metering"
	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
)

func TestMeteringObserver_OnEnd_NormalStreaming_Completed(t *testing.T) {
	rec := metering.NewRecorder()
	mem, ok := rec.(*metering.InMemoryRecorder)
	if !ok {
		t.Fatalf("expected InMemoryRecorder, got %T", rec)
	}

	ctx := &StreamingContext{
		RequestID: "req_normal",
		APIKeyID:  "key_normal",
		Provider:  "openai",
		Model:     "gpt-test",
		StartAt:   time.Now(),
	}

	og := NewObserverGroup(NewMeteringObserver(rec))
	rt := NewRuntime(ctx, og)

	chunks := []*Delta{{Text: "Hello"}, {Text: " "}, {Text: "world"}, {Text: "!"}}
	reader := NewStubStreamReader(chunks, 0)

	if err := rt.Run(reader); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if mem.Count() != 1 {
		t.Fatalf("expected 1 record, got %d", mem.Count())
	}

	record := mem.GetByRequestID("req_normal")
	if record == nil {
		t.Fatalf("expected record")
	}
	if record.Status != "completed" {
		t.Fatalf("expected status=completed, got %q", record.Status)
	}
	if record.EndReason != string(EndReasonStop) {
		t.Fatalf("expected end_reason=%q, got %q", EndReasonStop, record.EndReason)
	}
	if record.ChunkCount != len(chunks) {
		t.Fatalf("expected chunk_count=%d, got %d", len(chunks), record.ChunkCount)
	}
}

func TestMeteringObserver_OnEnd_QuotaExceeded_Partial(t *testing.T) {
	rec := metering.NewRecorder()
	mem, ok := rec.(*metering.InMemoryRecorder)
	if !ok {
		t.Fatalf("expected InMemoryRecorder, got %T", rec)
	}

	store := quota.NewInMemoryStore()
	store.SetRemaining("key_quota", 0)

	ctx := &StreamingContext{
		RequestID: "req_quota",
		APIKeyID:  "key_quota",
		Provider:  "openai",
		Model:     "gpt-test",
		StartAt:   time.Now(),
	}

	og := NewObserverGroup(
		NewQuotaObserver(store),
		NewMeteringObserver(rec),
	)
	rt := NewRuntime(ctx, og)

	chunks := []*Delta{{Text: "Hello"}, {Text: "world"}}
	reader := NewStubStreamReader(chunks, 0)

	if err := rt.Run(reader); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if mem.Count() != 1 {
		t.Fatalf("expected 1 record, got %d", mem.Count())
	}

	record := mem.GetByRequestID("req_quota")
	if record == nil {
		t.Fatalf("expected record")
	}
	if record.Status != "partial" {
		t.Fatalf("expected status=partial, got %q", record.Status)
	}
	if record.EndReason != string(EndReasonQuotaExceeded) {
		t.Fatalf("expected end_reason=%q, got %q", EndReasonQuotaExceeded, record.EndReason)
	}
}

func TestMeteringObserver_OnEnd_BeforeFirstChunk_NoRecord(t *testing.T) {
	rec := metering.NewRecorder()
	mem, ok := rec.(*metering.InMemoryRecorder)
	if !ok {
		t.Fatalf("expected InMemoryRecorder, got %T", rec)
	}

	ctx := &StreamingContext{
		RequestID: "req_fail_before_first",
		APIKeyID:  "key_fail",
		Provider:  "openai",
		Model:     "gpt-test",
		StartAt:   time.Now(),
	}

	og := NewObserverGroup(NewMeteringObserver(rec))
	rt := NewRuntime(ctx, og)

	reader := NewStubStreamReader(nil, 0)
	if err := rt.Run(reader); err == nil {
		t.Fatalf("expected non-nil error")
	} else if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}

	if mem.Count() != 0 {
		t.Fatalf("expected 0 record, got %d", mem.Count())
	}
}
