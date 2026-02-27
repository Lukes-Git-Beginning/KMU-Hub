package lexware

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	lexwareMaxTokens  = 2
	lexwareRefillRate = 500 * time.Millisecond
)

type RateLimiter struct {
	mu             sync.Mutex
	tokens         int
	maxTokens      int
	refillInterval time.Duration
	lastRefill     time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		tokens:         lexwareMaxTokens,
		maxTokens:      lexwareMaxTokens,
		refillInterval: lexwareRefillRate,
		lastRefill:     time.Now(),
	}
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		rl.mu.Lock()
		rl.refill()
		if rl.tokens > 0 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}
		waitUntil := rl.lastRefill.Add(rl.refillInterval)
		rl.mu.Unlock()

		waitDuration := time.Until(waitUntil)
		if waitDuration <= 0 {
			continue
		}

		slog.Info("lexware rate limiter waiting", "wait_ms", waitDuration.Milliseconds())

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}
	}
}

func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillInterval)
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}
}
