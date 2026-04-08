package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

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

// AuditLogger writes audit entries via the security gRPC service.
// Uses a buffered channel and worker pool to prevent goroutine explosion.
type AuditLogger struct {
	getClient func() (securityv1.SecurityServiceClient, error)
	events    chan auditEvent
}

// NewAuditLogger creates a new AuditLogger with a buffered event channel.
// Call Start() to launch workers before using the middleware.
func NewAuditLogger(getClient func() (securityv1.SecurityServiceClient, error)) *AuditLogger {
	return &AuditLogger{
		getClient: getClient,
		events:    make(chan auditEvent, 1000),
	}
}

// Start launches the worker pool that processes audit events.
func (a *AuditLogger) Start(workers int) {
	for i := 0; i < workers; i++ {
		go a.worker()
	}
}

func (a *AuditLogger) worker() {
	for event := range a.events {
		a.processEvent(event)
	}
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
			slog.Warn("audit event dropped, channel full")
		}
	})
}

func (a *AuditLogger) processEvent(event auditEvent) {
	client, err := a.getClient()
	if err != nil {
		slog.Debug("audit logger: security service unavailable", "error", err)
		return
	}

	result := "success"
	if event.statusCode >= 400 {
		result = "failure"
	}

	action := deriveActionFromMethod(event.method, event.path)

	_, err = client.CreateAuditEntry(context.Background(), &securityv1.CreateAuditEntryRequest{
		UserId:     event.userID,
		Action:     action,
		Target:     event.path,
		TargetType: "http_endpoint",
		IpAddress:  event.clientIP,
		UserAgent:  event.userAgent,
		Result:     result,
	})
	if err != nil {
		slog.Debug("audit logger: failed to log event", "action", action, "error", err)
	}
}

// Close drains the event channel. Call during graceful shutdown.
func (a *AuditLogger) Close() {
	close(a.events)
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
	return r.RemoteAddr
}
