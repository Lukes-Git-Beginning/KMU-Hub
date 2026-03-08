package middleware

import "net/http"

// SecurityHeaders sets protective HTTP headers on every response.
//
// CSRF note: This API uses Bearer tokens in the Authorization header exclusively,
// not cookies. CSRF attacks exploit automatic cookie submission, which does not
// apply here. No CSRF middleware is needed.
func SecurityHeaders(behindProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			h.Set("X-XSS-Protection", "0")

			// CSP: API-only backend serves no HTML, but set a restrictive policy as defense-in-depth.
			// Tailwind is pre-compiled (no unsafe-eval), fonts/images loaded from same origin.
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

			if behindProxy {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}
