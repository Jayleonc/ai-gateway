package openai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jayleonc/ai-gateway/internal/provider"
)

func TestAdapter_ChatStream_Recv(t *testing.T) {
	sse := "" +
		"data: {\"id\":\"x\",\"model\":\"m\",\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n" +
		"data: [DONE]\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sse))
	}))
	defer srv.Close()

	a := NewAdapter("test-key", srv.URL)
	pr, err := a.ChatStream(context.Background(), &provider.ChatRequest{Model: "m", Stream: true})
	if err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}
	defer pr.Close()

	e, err := pr.Recv()
	if err != nil {
		t.Fatalf("Recv error: %v", err)
	}
	if e == nil || e.Delta == nil {
		t.Fatalf("expected event with delta, got %#v", e)
	}
	if e.Delta.Content != "Hello" {
		t.Fatalf("expected delta content 'Hello', got %q", e.Delta.Content)
	}

	_, err = pr.Recv()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestAdapter_ChatStream_RequiresStreamFlag(t *testing.T) {
	a := NewAdapter("test-key", "http://example.invalid")
	_, err := a.ChatStream(context.Background(), &provider.ChatRequest{Model: "m", Stream: false})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestAdapter_ChatStream_UsesAuthorizationHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	a := NewAdapter("test-key", srv.URL)
	pr, err := a.ChatStream(context.Background(), &provider.ChatRequest{Model: "m", Stream: true})
	if err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}
	defer pr.Close()

	_, err = pr.Recv()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}
