package middleware

import (
	"net/http"

	"github.com/kmuhub/kmuhub/internal/clientctx"
)

// userAgentMaxLen bounds what we carry into a session row. Real agents are
// well under 512 bytes; anything longer is a caller trying to write a large
// value into the database through an unauthenticated endpoint (login).
const userAgentMaxLen = 512

// ClientInfo stores the caller's IP and user agent in the request context so
// downstream services can record them. Must run before any handler that
// forwards to a gRPC service — TenantOutboundUnaryInterceptor picks the values
// up from the context, not from the HTTP request.
//
// behindProxy has the same meaning as everywhere else in this package: when
// true the last X-Forwarded-For entry is the real peer, when false forwarding
// headers are attacker-controlled and only RemoteAddr is trusted
// (see ClientIPTrusted).
func ClientInfo(behindProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ua := r.UserAgent()
			if len(ua) > userAgentMaxLen {
				ua = ua[:userAgentMaxLen]
			}

			ctx := clientctx.With(r.Context(), clientctx.Info{
				IP:        ClientIPTrusted(r, behindProxy),
				UserAgent: ua,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
