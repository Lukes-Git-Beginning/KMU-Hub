package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func newTestClient(baseURL string) *Client {
	return NewClient(Config{
		Provider:       "ollama",
		BaseURL:        baseURL,
		Model:          "test-model",
		TimeoutSeconds: 5,
	})
}

func TestSummarize_Success(t *testing.T) {
	var gotBody chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message chatMessage `json:"message"`
			}{
				{Message: chatMessage{Role: "assistant", Content: "  Kurzfassung der Notizen.  "}},
			},
		})
	}))
	defer srv.Close()

	got, err := newTestClient(srv.URL).Summarize(context.Background(), "Wichtige Notizen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Kurzfassung der Notizen." {
		t.Errorf("summary = %q, want trimmed content", got)
	}
	if gotBody.Model != "test-model" {
		t.Errorf("model = %q, want test-model", gotBody.Model)
	}
	if len(gotBody.Messages) != 2 || gotBody.Messages[0].Role != "system" || gotBody.Messages[1].Content != "Wichtige Notizen" {
		t.Errorf("unexpected messages: %+v", gotBody.Messages)
	}
}

func TestSummarize_EmptyInput(t *testing.T) {
	got, err := newTestClient("http://unused.invalid").Summarize(context.Background(), "   ")
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("err = %v, want ErrEmptyInput", err)
	}
	if got != "" {
		t.Errorf("summary = %q, want empty", got)
	}
}

func TestSummarize_NoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(chatResponse{})
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).Summarize(context.Background(), "notes")
	if !errors.Is(err, ErrNoCompletion) {
		t.Fatalf("err = %v, want ErrNoCompletion", err)
	}
}

func TestSummarize_RetriesOn5xxThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message chatMessage `json:"message"`
			}{
				{Message: chatMessage{Content: "ok"}},
			},
		})
	}))
	defer srv.Close()

	got, err := newTestClient(srv.URL).Summarize(context.Background(), "notes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ok" {
		t.Errorf("summary = %q, want ok", got)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("calls = %d, want 2 (one 5xx + one success)", calls)
	}
}

func TestSummarize_4xxFailsWithoutRetry(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).Summarize(context.Background(), "notes")
	if err == nil {
		t.Fatal("expected error on 4xx")
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("calls = %d, want 1 (no retry on 4xx)", calls)
	}
}
