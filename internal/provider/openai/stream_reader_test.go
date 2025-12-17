package openai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamReader_Next_Done(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader("{}"))
	sr, err := NewStreamReader(context.Background(), srv.Client(), req)
	if err != nil {
		t.Fatalf("NewStreamReader error: %v", err)
	}
	defer sr.Close()

	_, err = sr.Next()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestStreamReader_Next_DeltaContent(t *testing.T) {
	sse := "" +
		"data: {\"id\":\"x\",\"model\":\"m\",\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n" +
		"data: {\"id\":\"x\",\"model\":\"m\",\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n" +
		"data: [DONE]\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sse))
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader("{}"))
	sr, err := NewStreamReader(context.Background(), srv.Client(), req)
	if err != nil {
		t.Fatalf("NewStreamReader error: %v", err)
	}
	defer sr.Close()

	d1, err := sr.Next()
	if err != nil {
		t.Fatalf("Next error: %v", err)
	}
	if d1 == nil || d1.Text != "Hello" {
		t.Fatalf("expected 'Hello', got %#v", d1)
	}

	d2, err := sr.Next()
	if err != nil {
		t.Fatalf("Next error: %v", err)
	}
	if d2 == nil || d2.Text != " world" {
		t.Fatalf("expected ' world', got %#v", d2)
	}

	_, err = sr.Next()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestStreamReader_IgnoresNonDataFields(t *testing.T) {
	sse := "" +
		": comment\n" +
		"event: message\n" +
		"id: 1\n" +
		"data: {\"id\":\"x\",\"model\":\"m\",\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n" +
		"data: [DONE]\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sse))
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader("{}"))
	sr, err := NewStreamReader(context.Background(), srv.Client(), req)
	if err != nil {
		t.Fatalf("NewStreamReader error: %v", err)
	}
	defer sr.Close()

	d, err := sr.Next()
	if err != nil {
		t.Fatalf("Next error: %v", err)
	}
	if d == nil || d.Text != "Hi" {
		t.Fatalf("expected 'Hi', got %#v", d)
	}
}
