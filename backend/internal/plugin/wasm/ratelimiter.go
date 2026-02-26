package wasm

import (
	"sync"
	"time"
)

// RateLimiter implements a per-plugin token bucket rate limiter
type RateLimiter struct {
	buckets    map[string]*tokenBucket
	mu         sync.Mutex
	maxTokens  int
	refillRate time.Duration
}

type tokenBucket struct {
	tokens     int
	maxTokens  int
	lastRefill time.Time
	refillRate time.Duration
}

// NewRateLimiter creates a new rate limiter
// maxPerMinute is the maximum number of host API calls per minute per plugin
func NewRateLimiter(maxPerMinute int) *RateLimiter {
	if maxPerMinute <= 0 {
		maxPerMinute = 100
	}
	return &RateLimiter{
		buckets:    make(map[string]*tokenBucket),
		maxTokens:  maxPerMinute,
		refillRate: time.Minute / time.Duration(maxPerMinute),
	}
}

// Allow checks if a request from the given plugin is allowed
func (rl *RateLimiter) Allow(pluginID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[pluginID]
	if !exists {
		bucket = &tokenBucket{
			tokens:     rl.maxTokens,
			maxTokens:  rl.maxTokens,
			lastRefill: time.Now(),
			refillRate: rl.refillRate,
		}
		rl.buckets[pluginID] = bucket
	}

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / bucket.refillRate)
	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > bucket.maxTokens {
			bucket.tokens = bucket.maxTokens
		}
		bucket.lastRefill = now
	}

	if bucket.tokens <= 0 {
		return false
	}

	bucket.tokens--
	return true
}

// Reset resets the rate limiter for a specific plugin
func (rl *RateLimiter) Reset(pluginID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.buckets, pluginID)
}
