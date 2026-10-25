package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kmuhub/kmuhub/internal/metrics"
	"github.com/kmuhub/kmuhub/internal/safego"
	"github.com/kmuhub/kmuhub/internal/sysctx"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

// auditEvent captures the data needed to log an audit entry.
type auditEvent struct {
	method     string
	path       string
	userAgent  string
	clientIP   string
	userID     string
	statusCode int
}

// Drop reasons, used as log field and as label of audit_events_dropped_total.
const (
	dropBufferFull         = "buffer_full"
	dropServiceUnavailable = "service_unavailable"
	dropWriteFailed        = "write_failed"
	dropClosed             = "closed"
	dropWorkerPanic        = "worker_panic"
)

const (
	// dropWarnInterval throttles the Warn line per reason: while the security
	// service is down every event fails, one line per event would flood the log.
	dropWarnInterval = 30 * time.Second
	// closeDrainTimeout bounds how long Close waits for the workers to flush.
	closeDrainTimeout = 5 * time.Second
)

// AuditLogger writes audit entries via the security gRPC service.
// Uses a buffered channel and worker pool to prevent goroutine explosion.
type AuditLogger struct {
	getClient func() (securityv1.SecurityServiceClient, error)
	events    chan auditEvent
	// events is never closed: Middleware may still send while Close runs, and a
	// send on a closed channel panics. done signals the workers instead.
	done      chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup

	metrics *metrics.Registry // optional
	now     func() time.Time

	dropMu sync.Mutex
	drops  map[string]*dropState
}

type dropState struct {
	lastWarn   time.Time
	suppressed int
}

// NewAuditLogger creates a new AuditLogger with a buffered event channel.
// Call Start() to launch workers before using the middleware.
func NewAuditLogger(getClient func() (securityv1.SecurityServiceClient, error)) *AuditLogger {
	return &AuditLogger{
		getClient: getClient,
		events:    make(chan auditEvent, 1000),
		done:      make(chan struct{}),
		now:       time.Now,
		drops:     make(map[string]*dropState),
	}
}

// SetMetrics makes dropped events visible as audit_events_dropped_total.
func (a *AuditLogger) SetMetrics(reg *metrics.Registry) { a.metrics = reg }

// Start launches the worker pool that processes audit events.
func (a *AuditLogger) Start(workers int) {
	a.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go a.worker()
	}
}

func (a *AuditLogger) worker() {
	defer a.wg.Done()
	for {
		select {
		case event := <-a.events:
			a.safeProcessEvent(event)
		case <-a.done:
			// Flush what is already queued, then stop.
			for {
				select {
				case event := <-a.events:
					a.safeProcessEvent(event)
				default:
					return
				}
			}
		}
	}
}

// safeProcessEvent keeps a panic while writing one event (nil client, nil
// response from a misbehaving stub) from killing the whole process: that event
// is counted as lost and the worker moves on to the next one.
func (a *AuditLogger) safeProcessEvent(event auditEvent) {
	if !safego.Do("audit event worker", func() { a.processEvent(event) }) {
		a.recordDrop(dropWorkerPanic, nil)
	}
}

// recordDrop counts a lost audit event and warns at most once per
// dropWarnInterval and reason. Lost audit entries must never be silent.
func (a *AuditLogger) recordDrop(reason string, err error) {
	if a.metrics != nil {
		a.metrics.AuditEventsDropped.WithLabelValues(reason).Inc()
	}

	a.dropMu.Lock()
	st := a.drops[reason]
	if st == nil {
		st = &dropState{}
		a.drops[reason] = st
	}
	now := a.now()
	if !st.lastWarn.IsZero() && now.Sub(st.lastWarn) < dropWarnInterval {
		st.suppressed++
		a.dropMu.Unlock()
		return
	}
	suppressed := st.suppressed
	st.lastWarn, st.suppressed = now, 0
	a.dropMu.Unlock()

	slog.Warn("audit event lost", "reason", reason, "suppressed_since_last_warn", suppressed, "error", err)
}

// Middleware returns HTTP middleware that logs audit events for mutating requests.
func (a *AuditLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !shouldAudit(r) {
			next.ServeHTTP(w, r)
			return
		}

		sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(sw, r)

		select {
		case <-a.done:
			a.recordDrop(dropClosed, nil)
			return
		default:
		}

		// Non-blocking send to worker pool
		select {
		case a.events <- auditEvent{
			method:     r.Method,
			path:       r.URL.Path,
			userAgent:  r.UserAgent(),
			clientIP:   auditClientIP(r),
			userID:     GetUserID(r.Context()),
			statusCode: sw.statusCode,
		}:
		default:
			a.recordDrop(dropBufferFull, nil)
		}
	})
}

func (a *AuditLogger) processEvent(event auditEvent) {
	client, err := a.getClient()
	if err != nil {
		a.recordDrop(dropServiceUnavailable, err)
		return
	}

	result := "success"
	if event.statusCode >= 400 {
		result = "failure"
	}

	action := deriveActionFromMethod(event.method, event.path)

	_, err = client.CreateAuditEntry(eventCtx(), &securityv1.CreateAuditEntryRequest{
		UserId:     event.userID,
		Action:     action,
		Target:     event.path,
		TargetType: "http_endpoint",
		IpAddress:  event.clientIP,
		UserAgent:  event.userAgent,
		Result:     result,
	})
	if err != nil {
		a.recordDrop(dropWriteFailed, err)
	}
}

// Close stops the workers after they have flushed the queued events, waiting
// at most closeDrainTimeout. Safe to call more than once and while requests
// are still arriving: those are counted as dropped instead of panicking.
func (a *AuditLogger) Close() {
	a.closeOnce.Do(func() { close(a.done) })

	flushed := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(flushed)
	}()
	select {
	case <-flushed:
	case <-time.After(closeDrainTimeout):
		slog.Warn("audit logger: workers did not flush in time", "queued", len(a.events))
	}
}

// eventCtx returns the context used by the worker for audit-log writes.
// Wrapped with sysctx.With so the audit_log SELECT used for hash-chain
// previous-hash lookup bypasses the per-tenant RLS policy — the worker
// has no per-request tenant.
func eventCtx() context.Context {
	return sysctx.With(context.Background())
}

func shouldAudit(r *http.Request) bool {
	path := r.URL.Path

	// Always audit auth actions
	if strings.HasPrefix(path, "/api/v1/auth/") {
		return r.Method == http.MethodPost
	}

	// Audit mutating operations on core resources
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return false
	}

	auditPrefixes := []string{
		"/api/v1/contacts",
		"/api/v1/companies",
		"/api/v1/deals",
		"/api/v1/security/",
		"/api/v1/documents",
		"/api/v1/files/",
	}
	for _, prefix := range auditPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func deriveActionFromMethod(method string, path string) string {
	// Auth actions
	if strings.Contains(path, "/auth/login") {
		return "user.login"
	}
	if strings.Contains(path, "/auth/register") {
		return "user.register"
	}
	if strings.Contains(path, "/auth/logout") {
		return "user.logout"
	}

	// Generic CRUD mapping
	switch method {
	case http.MethodPost:
		return "resource.create"
	case http.MethodPut, http.MethodPatch:
		return "resource.update"
	case http.MethodDelete:
		return "resource.delete"
	default:
		return "resource.access"
	}
}

func auditClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// RemoteAddr is host:port ("[::1]:52502"), which the inet column refuses, so every entry
	// of a request without proxy headers was lost.
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
