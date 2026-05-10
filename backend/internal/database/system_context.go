package database

import "context"

// systemContextKey marks a context as operating in system-bypass mode.
// BeginRLSTx reads it and sets app.role='system', which the RLS policies
// installed by enable_tenant_rls() treat as a tenant-filter bypass via
// is_system_context() returning true.
//
// Use exactly one path: WithSystemContext at the entry point of any code
// that runs without a per-request tenant — schedulers, pollers, audit-log
// inserts, migration helpers, healthcheck probes that touch RLS-enabled
// tables.
type contextKey int

const (
	systemContextKey contextKey = iota
)

// WithSystemContext marks ctx as a system operation. Subsequent calls to
// IsSystemContext(ctx) return true, and BeginRLSTx will set app.role='system'
// on the spawned transaction so RLS policies admit the operation regardless
// of tenant.
func WithSystemContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, systemContextKey, true)
}

// IsSystemContext reports whether ctx (or one of its parents) was wrapped
// with WithSystemContext.
func IsSystemContext(ctx context.Context) bool {
	v, _ := ctx.Value(systemContextKey).(bool)
	return v
}
