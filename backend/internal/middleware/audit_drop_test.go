package middleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/metrics"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

type fakeSecurityClient struct {
	securityv1.SecurityServiceClient
	calls atomic.Int64
	err   error
}

func (f *fakeSecurityClient) CreateAuditEntry(_ context.Context, _ *securityv1.CreateAuditEntryRequest, _ ...grpc.CallOption) (*securityv1.CreateAuditEntryResponse, error) {
	f.calls.Add(1)
	return nil, f.err
}

// syncBuffer is a bytes.Buffer safe for the slog handler and the test reader.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func captureLogs(t *testing.T) *syncBuffer {
	t.Helper()
	buf := &syncBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

func auditedRequest(h http.Handler) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)
}

func auditOKHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestAuditLogger_FailedWriteIsWarnedThrottledAndCounted(t *testing.T) {
	logs := captureLogs(t)
	reg := metrics.NewRegistry()
	client := &fakeSecurityClient{err: errors.New("auth down")}
	a := NewAuditLogger(func() (securityv1.SecurityServiceClient, error) { return client, nil })
	a.SetMetrics(reg)
	now := time.Date(2026, 10, 21, 3, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		a.processEvent(auditEvent{method: http.MethodPost, path: "/api/v1/contacts", statusCode: 200})
	}
	if got := counterValue(t, reg.AuditEventsDropped.WithLabelValues(dropWriteFailed)); got != 3 {
		t.Fatalf("counter = %v, want 3", got)
	}
	out := logs.String()
	if n := strings.Count(out, "level=WARN msg=\"audit event lost\""); n != 1 {
		t.Fatalf("want exactly 1 throttled Warn line, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, "reason=write_failed") || !strings.Contains(out, "auth down") {
		t.Fatalf("warn line lacks reason/error:\n%s", out)
	}

	now = now.Add(dropWarnInterval + time.Second)
	a.processEvent(auditEvent{method: http.MethodPost, path: "/api/v1/contacts", statusCode: 200})
	out = logs.String()
	if n := strings.Count(out, "level=WARN msg=\"audit event lost\""); n != 2 {
		t.Fatalf("want a second Warn line after the interval, got %d", n)
	}
	if !strings.Contains(out, "suppressed_since_last_warn=2") {
		t.Fatalf("second line must report the 2 suppressed events:\n%s", out)
	}
}

func TestAuditLogger_UnavailableServiceIsWarnedAndCounted(t *testing.T) {
	logs := captureLogs(t)
	reg := metrics.NewRegistry()
	a := NewAuditLogger(func() (securityv1.SecurityServiceClient, error) { return nil, errors.New("no conn") })
	a.SetMetrics(reg)

	a.processEvent(auditEvent{method: http.MethodPost, path: "/api/v1/contacts", statusCode: 200})

	if got := counterValue(t, reg.AuditEventsDropped.WithLabelValues(dropServiceUnavailable)); got != 1 {
		t.Fatalf("counter = %v, want 1", got)
	}
	if !strings.Contains(logs.String(), "level=WARN msg=\"audit event lost\" reason=service_unavailable") {
		t.Fatalf("expected Warn for unavailable service:\n%s", logs.String())
	}
}

func TestAuditLogger_FullBufferIsCounted(t *testing.T) {
	captureLogs(t)
	reg := metrics.NewRegistry()
	a := NewAuditLogger(func() (securityv1.SecurityServiceClient, error) { return &fakeSecurityClient{}, nil })
	a.SetMetrics(reg)
	h := a.Middleware(auditOKHandler())

	// No workers started: the 1000-slot buffer fills, the next request is dropped.
	for i := 0; i < cap(a.events)+1; i++ {
		auditedRequest(h)
	}
	if got := counterValue(t, reg.AuditEventsDropped.WithLabelValues(dropBufferFull)); got != 1 {
		t.Fatalf("counter = %v, want 1", got)
	}
}

func TestAuditLogger_SendAfterCloseDoesNotPanic(t *testing.T) {
	captureLogs(t)
	reg := metrics.NewRegistry()
	client := &fakeSecurityClient{}
	a := NewAuditLogger(func() (securityv1.SecurityServiceClient, error) { return client, nil })
	a.SetMetrics(reg)
	a.Start(2)
	h := a.Middleware(auditOKHandler())

	auditedRequest(h)
	a.Close()
	a.Close() // idempotent

	auditedRequest(h) // used to panic: send on closed channel

	if got := counterValue(t, reg.AuditEventsDropped.WithLabelValues(dropClosed)); got != 1 {
		t.Fatalf("closed counter = %v, want 1", got)
	}
	if got := client.calls.Load(); got != 1 {
		t.Fatalf("queued event must be flushed before shutdown, calls = %d", got)
	}
}

func counterValue(t *testing.T, c interface{ Write(*dto.Metric) error }) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatal(err)
	}
	return m.GetCounter().GetValue()
}
