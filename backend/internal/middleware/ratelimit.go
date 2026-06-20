package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kmuhub/kmuhub/internal/server/response"
)

type RateLimiter struct {
	redis       *redis.Client
	rps         int
	window      time.Duration
	fallback    *inMemoryLimiter
	prefix      string
	behindProxy bool
}

// NewRateLimiter creates a rate limiter with the default "ratelimit" key prefix.
// behindProxy controls whether X-Forwarded-For is trusted (see ClientIPTrusted).
func NewRateLimiter(redisClient *redis.Client, rps int, behindProxy bool) *RateLimiter {
	return NewRateLimiterWithPrefix(redisClient, rps, "ratelimit", behindProxy)
}

// NewRateLimiterWithPrefix creates a rate limiter with a custom Redis key prefix.
// Use this to create scoped limiters with independent counters (e.g. "ratelimit:public").
func NewRateLimiterWithPrefix(redisClient *redis.Client, rps int, prefix string, behindProxy bool) *RateLimiter {
	return &RateLimiter{
		redis:       redisClient,
		rps:         rps,
		window:      time.Second,
		fallback:    newInMemoryLimiter(rps),
		prefix:      prefix,
		behindProxy: behindProxy,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := ClientIPTrusted(r, rl.behindProxy)

		// Try authenticated user ID first
		if userID := GetUserID(r.Context()); userID != "" {
			key = userID
		}

		allowed, err := rl.allow(r.Context(), key)
		if err != nil {
			slog.Warn("rate limiter error, using fallback", "error", err)
			allowed = rl.fallback.allow(key)
		}

		if !allowed {
			w.Header().Set("Retry-After", "1")
			response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(ctx context.Context, key string) (bool, error) {
	if rl.redis == nil {
		return rl.fallback.allow(key), nil
	}

	redisKey := fmt.Sprintf("%s:%s", rl.prefix, key)

	pipe := rl.redis.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, rl.window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	return incr.Val() <= int64(rl.rps), nil
}

// ClientIPTrusted resolves the client IP for security decisions (rate limiting,
// IP filtering) under the configured proxy-trust setting.
//
// When behindProxy is true, the request is assumed to traverse exactly one
// trusted reverse proxy (Caddy) that appends the real peer IP to the END of
// X-Forwarded-For. The LAST entry is therefore authoritative; the leftmost
// entries are attacker-supplied and ignored. This defeats X-Forwarded-For
// spoofing of the rate limiter and IP filter that a leftmost read would allow.
//
// When behindProxy is false, all forwarding headers are attacker-controlled
// (the gateway is directly exposed) and ignored — only the TCP peer
// (RemoteAddr) is trusted.
func ClientIPTrusted(r *http.Request, behindProxy bool) string {
	if behindProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ClientIP extracts the client IP preferring the first X-Forwarded-For value.
// Retained for non-security callers (request logging, captcha attribution) that
// predate the proxy-trust model; security middleware must use ClientIPTrusted.
func ClientIP(r *http.Request) string { return clientIP(r) }

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type inMemoryLimiter struct {
	mu     sync.Mutex
	counts map[string]*bucket
	rps    int
}

type bucket struct {
	count   int
	resetAt time.Time
}

func newInMemoryLimiter(rps int) *inMemoryLimiter {
	return &inMemoryLimiter{
		counts: make(map[string]*bucket),
		rps:    rps,
	}
}

func (l *inMemoryLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.counts[key]
	if !ok || now.After(b.resetAt) {
		l.counts[key] = &bucket{count: 1, resetAt: now.Add(time.Second)}
		return true
	}

	b.count++
	return b.count <= l.rps
}
