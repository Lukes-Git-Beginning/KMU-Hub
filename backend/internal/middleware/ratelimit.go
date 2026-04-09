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
	redis    *redis.Client
	rps      int
	window   time.Duration
	fallback *inMemoryLimiter
}

func NewRateLimiter(redisClient *redis.Client, rps int) *RateLimiter {
	return &RateLimiter{
		redis:    redisClient,
		rps:      rps,
		window:   time.Second,
		fallback: newInMemoryLimiter(rps),
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)

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

	redisKey := fmt.Sprintf("ratelimit:%s", key)

	pipe := rl.redis.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, rl.window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	return incr.Val() <= int64(rl.rps), nil
}

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
