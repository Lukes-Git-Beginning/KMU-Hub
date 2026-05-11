// Package sysctx carries the marker that lets RLS policies bypass the
// tenant filter for code paths that legitimately have no per-request
// tenant — schedulers, pollers, audit-log inserts, login/refresh flows
// that read users before a JWT is issued.
//
// The marker lives here, not in internal/database, so packages that the
// database pool transitively depends on (notably internal/middleware
// → internal/auth) can set it without creating an import cycle.
//
// Use exactly one path: sysctx.With at the entry point. Pool hooks and
// BeginRLSTx call sysctx.Is and set app.role='system' so RLS policies
// admit the operation regardless of tenant.
package sysctx

import "context"

type contextKey int

const systemContextKey contextKey = iota

// With marks ctx as a system operation. Subsequent calls to Is(ctx)
// return true; the database pool's PrepareConn hook then sets
// app.role='system' on the acquired connection.
func With(ctx context.Context) context.Context {
	return context.WithValue(ctx, systemContextKey, true)
}

// Is reports whether ctx (or one of its parents) was wrapped with With.
func Is(ctx context.Context) bool {
	v, _ := ctx.Value(systemContextKey).(bool)
	return v
}
