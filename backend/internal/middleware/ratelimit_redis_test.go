package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func redisLimiterHandler(t *testing.T, mr *miniredis.Miniredis, rps int) http.Handler {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	rl := NewRateLimiter(client, rps, false)
	return rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func serveFrom(h http.Handler, addr string) int {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = addr
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// TestRateLimiter_Redis_WindowResetsUnderSustainedTraffic is the regression for the
// counter that never expired: EXPIRE was re-armed on every request, so with less than
// one second between requests the key lived forever and the caller was locked out.
func TestRateLimiter_Redis_WindowResetsUnderSustainedTraffic(t *testing.T) {
	mr := miniredis.RunT(t)
	const rps = 5
	h := redisLimiterHandler(t, mr, rps)

	// 25 requests, 100 ms apart (2.5 s of traffic, never a quiet second): the counter
	// must roll over every second, so each of the three windows admits rps requests.
	allowed := 0
	for i := 0; i < 25; i++ {
		if serveFrom(h, "10.0.0.1:1234") == http.StatusOK {
			allowed++
		}
		mr.FastForward(100 * time.Millisecond)
	}
	assert.Equal(t, 3*rps, allowed, "5 in each of the windows [0,1) [1,2) [2,2.5)")
}

// TestRateLimiter_Redis_TTLSetOnlyByFirstRequest pins the fixed one-second window:
// later requests in the same window must not move the expiry.
func TestRateLimiter_Redis_TTLSetOnlyByFirstRequest(t *testing.T) {
	mr := miniredis.RunT(t)
	h := redisLimiterHandler(t, mr, 100)

	require.Equal(t, http.StatusOK, serveFrom(h, "10.0.0.2:1"))
	mr.FastForward(700 * time.Millisecond)
	require.Equal(t, http.StatusOK, serveFrom(h, "10.0.0.2:1"))

	assert.InDelta(t, 300*time.Millisecond, mr.TTL("ratelimit:10.0.0.2"), float64(time.Millisecond))
	mr.FastForward(300 * time.Millisecond)
	assert.False(t, mr.Exists("ratelimit:10.0.0.2"), "key must expire one window after the first request")
}

func TestRateLimiter_Redis_CountersPerClientAndPrefix(t *testing.T) {
	mr := miniredis.RunT(t)
	h := redisLimiterHandler(t, mr, 1)

	assert.Equal(t, http.StatusOK, serveFrom(h, "10.0.0.3:1"))
	assert.Equal(t, http.StatusTooManyRequests, serveFrom(h, "10.0.0.3:2"))
	assert.Equal(t, http.StatusOK, serveFrom(h, "10.0.0.4:1"), "other client has its own counter")
}
