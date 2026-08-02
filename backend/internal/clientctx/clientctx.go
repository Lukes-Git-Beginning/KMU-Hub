// Package clientctx carries the transport metadata of the originating HTTP
// request — client IP and user agent — down to the services that record it.
//
// Only the auth service uses it today (user_sessions rows carry the device a
// login came from), and it exists as its own package for the same reason
// sysctx does: internal/middleware imports internal/auth, so the keys cannot
// live in middleware without an import cycle.
//
// The value travels HTTP request → gateway context (middleware.ClientInfo) →
// gRPC metadata (TenantOutboundUnaryInterceptor) → service context
// (TenantInboundUnaryInterceptor). Every hop is optional: an absent value
// yields the zero Info, never an error. A session row with an empty device is
// still a usable session row.
package clientctx

import "context"

type contextKey int

const infoKey contextKey = iota

// Info is the request metadata worth persisting alongside a session.
type Info struct {
	// IP is the client address as resolved under the configured proxy-trust
	// setting (middleware.ClientIPTrusted). May be empty.
	IP string
	// UserAgent is the raw User-Agent header. May be empty.
	UserAgent string
}

// With attaches the client info to ctx. An Info with both fields empty is
// still attached — From then returns it unchanged, which is the same result
// as not attaching at all.
func With(ctx context.Context, info Info) context.Context {
	return context.WithValue(ctx, infoKey, info)
}

// From returns the client info attached to ctx, or the zero Info if none was.
func From(ctx context.Context) Info {
	info, _ := ctx.Value(infoKey).(Info)
	return info
}
