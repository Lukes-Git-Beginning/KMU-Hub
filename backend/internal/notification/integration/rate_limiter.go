package integration

import (
	"sync"
	"time"
)

// RateLimiter enforces per-channel send rate limits for external platform APIs.
// Slack: 1 msg/sec/channel. Teams: 4 msg/sec (global).
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]time.Time // key: platform:channelID -> last send time
}

// NewRateLimiter creates a new per-channel rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]time.Time),
	}
}

// minInterval returns the minimum time between messages for a platform.
func minInterval(platform string) time.Duration {
	switch platform {
	case PlatformSlack:
		return time.Second // 1 msg/sec/channel
	case PlatformTeams:
		return 250 * time.Millisecond // 4 msg/sec global
	default:
		return time.Second
	}
}

// bucketKey returns the rate limit bucket key.
// Slack is per-channel; Teams is global (all channels share one bucket).
func bucketKey(platform, channelID string) string {
	if platform == PlatformTeams {
		return "teams:global"
	}
	return platform + ":" + channelID
}

// Allow checks if a message can be sent now and records the send time.
// Returns true if allowed, false if rate limited.
func (rl *RateLimiter) Allow(platform, channelID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := bucketKey(platform, channelID)
	interval := minInterval(platform)
	now := time.Now()

	lastSend, exists := rl.buckets[key]
	if !exists || now.Sub(lastSend) >= interval {
		rl.buckets[key] = now
		return true
	}
	return false
}

// Wait returns how long the caller should wait before sending.
// Returns 0 if the caller can send immediately.
func (rl *RateLimiter) Wait(platform, channelID string) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := bucketKey(platform, channelID)
	interval := minInterval(platform)
	now := time.Now()

	lastSend, exists := rl.buckets[key]
	if !exists {
		return 0
	}

	elapsed := now.Sub(lastSend)
	if elapsed >= interval {
		return 0
	}
	return interval - elapsed
}
