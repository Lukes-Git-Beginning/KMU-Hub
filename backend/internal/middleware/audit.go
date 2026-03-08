package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

// AuditLogger writes audit entries via the security gRPC service.
// It wraps handlers for security-relevant endpoints (login, CRUD, GDPR).
type AuditLogger struct {
	getClient func() (securityv1.SecurityServiceClient, error)
}

// NewAuditLogger creates a new AuditLogger that lazily obtains a gRPC client.
func NewAuditLogger(getClient func() (securityv1.SecurityServiceClient, error)) *AuditLogger {
	return &AuditLogger{getClient: getClient}
}

// Middleware returns HTTP middleware that logs audit events for mutating requests.
// It fires asynchronously to avoid adding latency.
func (a *AuditLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only audit mutating methods on security-relevant paths
		if !shouldAudit(r) {
			next.ServeHTTP(w, r)
			return
		}

		sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(sw, r)

		// Fire-and-forget audit log
		go a.logEvent(r, sw.statusCode)
	})
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

func (a *AuditLogger) logEvent(r *http.Request, statusCode int) {
	client, err := a.getClient()
	if err != nil {
		slog.Debug("audit logger: security service unavailable", "error", err)
		return
	}

	userID := GetUserID(r.Context())
	result := "success"
	if statusCode >= 400 {
		result = "failure"
	}

	action := deriveAction(r)

	_, err = client.CreateAuditEntry(r.Context(), &securityv1.CreateAuditEntryRequest{
		UserId:     userID,
		Action:     action,
		Target:     r.URL.Path,
		TargetType: "http_endpoint",
		IpAddress:  auditClientIP(r),
		UserAgent:  r.UserAgent(),
		Result:     result,
	})
	if err != nil {
		slog.Debug("audit logger: failed to log event", "action", action, "error", err)
	}
}

func deriveAction(r *http.Request) string {
	path := r.URL.Path

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
	switch r.Method {
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
