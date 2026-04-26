package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kmuhub/kmuhub/internal/health"
)

type stubChecker struct {
	name   string
	status health.Status
}

func (s stubChecker) Check(_ context.Context) health.CheckResult {
	return health.CheckResult{Name: s.name, Status: s.status}
}

func TestHealthHandler_GetReturnsJSONBody(t *testing.T) {
	r := chi.NewRouter()
	RegisterHealth(r, "/health", []health.Checker{stubChecker{name: "stub", status: health.StatusHealthy}})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"healthy"`) {
		t.Fatalf("expected status field in body, got %q", rec.Body.String())
	}
}

func TestHealthHandler_HeadReturnsStatusOnly(t *testing.T) {
	r := chi.NewRouter()
	RegisterHealth(r, "/health", []health.Checker{stubChecker{name: "stub", status: health.StatusHealthy}})

	req := httptest.NewRequest(http.MethodHead, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body on HEAD, got %q", rec.Body.String())
	}
}

func TestHealthHandler_HeadUnhealthyReturns503(t *testing.T) {
	r := chi.NewRouter()
	RegisterHealth(r, "/health", []health.Checker{stubChecker{name: "stub", status: health.StatusUnhealthy}})

	req := httptest.NewRequest(http.MethodHead, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body on HEAD, got %q", rec.Body.String())
	}
}
