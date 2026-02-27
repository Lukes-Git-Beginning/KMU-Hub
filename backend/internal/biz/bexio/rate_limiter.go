package bexio

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	defaultRateLimit    = 50
	defaultRateWindow   = 10 * time.Second
	headerRateLimit     = "X-RateLimit-Limit"
	headerRateRemaining = "X-RateLimit-Remaining"
	headerRateReset     = "X-RateLimit-Reset"
)

// RateLimiter enforces Bexio API rate limits using a token bucket that adapts
// based on X-RateLimit-* response headers.
type RateLimiter struct {
	mu        sync.Mutex
	limit     int
	remaining int
	resetAt   time.Time
}

// NewRateLimiter creates a rate limiter with default Bexio limits.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limit:     defaultRateLimit,
		remaining: defaultRateLimit,
		resetAt:   time.Now().Add(defaultRateWindow),
	}
}

// Wait blocks until a request slot is available or the context is cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		rl.mu.Lock()

		// If the rate window has passed, reset
		if time.Now().After(rl.resetAt) {
			rl.remaining = rl.limit
			rl.resetAt = time.Now().Add(defaultRateWindow)
		}

		if rl.remaining > 0 {
			rl.remaining--
			rl.mu.Unlock()
			return nil
		}

		waitUntil := rl.resetAt
		rl.mu.Unlock()

		waitDuration := time.Until(waitUntil)
		if waitDuration <= 0 {
			continue
		}

		slog.Info("bexio rate limiter waiting",
			"wait_ms", waitDuration.Milliseconds(),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
			// Continue loop to try acquiring a slot
		}
	}
}

// UpdateFromHeaders reads Bexio rate limit response headers and adjusts internal state.
func (rl *RateLimiter) UpdateFromHeaders(headers http.Header) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if v := headers.Get(headerRateLimit); v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit > 0 {
			rl.limit = limit
		}
	}

	if v := headers.Get(headerRateRemaining); v != "" {
		if remaining, err := strconv.Atoi(v); err == nil {
			rl.remaining = remaining
		}
	}

	if v := headers.Get(headerRateReset); v != "" {
		if resetSecs, err := strconv.ParseInt(v, 10, 64); err == nil {
			rl.resetAt = time.Unix(resetSecs, 0)
		}
	}
}
