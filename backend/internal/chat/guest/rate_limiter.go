package guest

import (
	"sync"
	"time"
)

// GuestRateLimiter implements a per-session sliding window rate limiter
type GuestRateLimiter struct {
	mu      sync.Mutex
	windows map[string][]time.Time // session_id -> timestamps of recent messages
	limit   int                    // max messages per window (default 30)
	window  time.Duration          // window duration (default 1 minute)
}

// NewGuestRateLimiter creates a new rate limiter with the given limits
func NewGuestRateLimiter(limit int, window time.Duration) *GuestRateLimiter {
	return &GuestRateLimiter{
		windows: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
}

// Allow checks if a message is allowed for the given session and records it.
// Returns true if the message is within the rate limit.
func (rl *GuestRateLimiter) Allow(sessionID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Prune expired entries
	timestamps := rl.windows[sessionID]
	pruned := timestamps[:0]
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}

	if len(pruned) >= rl.limit {
		rl.windows[sessionID] = pruned
		return false
	}

	rl.windows[sessionID] = append(pruned, now)
	return true
}
