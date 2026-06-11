package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestFileRoutes_NoMountConflictWithChatFiles reproduces the E2E startup panic
// "chi: attempting to Mount() a handler on an existing path, '/api/v1/files'":
// route_chat.go mounts /api/v1/files for per-file operations, so the presign
// routes must register as concrete paths, not a second subrouter mount.
func TestFileRoutes_NoMountConflictWithChatFiles(t *testing.T) {
	r := chi.NewRouter()

	noop := func(next http.Handler) http.Handler { return next }

	// Simulate the chat file routes that already own the /api/v1/files mount.
	r.Route("/api/v1/files", func(r chi.Router) {
		r.Get("/{id}/download", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot) // marker for the subrouter
		})
	})

	fr := NewFileRoutes(NewServiceRegistry(nil))

	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("registering file routes panicked: %v", rec)
		}
	}()
	fr.RegisterRoutes(r, noop)

	// The static presign path must be dispatched to the presign handler
	// (503 service unavailable without a document connection), not to the
	// chat subrouter's catch-all.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/presign-upload", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code == http.StatusTeapot {
		t.Fatalf("presign-upload was routed into the chat files subrouter")
	}
	if rec.Code == http.StatusNotFound {
		t.Fatalf("presign-upload route not registered (404)")
	}

	// The chat subrouter must keep working.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/files/abc/download", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusTeapot {
		t.Fatalf("chat file download route broken, got status %d", rec.Code)
	}
}
