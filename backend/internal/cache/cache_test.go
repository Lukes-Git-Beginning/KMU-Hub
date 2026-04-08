package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func setupTestClient(t *testing.T) (*Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rc.Close() })
	return NewClient(rc), mr
}

func TestGet_CacheMiss_FetchesFromDB(t *testing.T) {
	c, _ := setupTestClient(t)
	ctx := context.Background()
	fetchCalled := false

	val, err := Get(ctx, c, "test:miss", time.Minute, func(ctx context.Context) (testItem, error) {
		fetchCalled = true
		return testItem{Name: "hello", Value: 42}, nil
	})

	require.NoError(t, err)
	assert.True(t, fetchCalled)
	assert.Equal(t, "hello", val.Name)
	assert.Equal(t, 42, val.Value)
}

func TestGet_CacheHit_DoesNotFetch(t *testing.T) {
	c, _ := setupTestClient(t)
	ctx := context.Background()

	// Populate cache
	_, err := Get(ctx, c, "test:hit", time.Minute, func(ctx context.Context) (testItem, error) {
		return testItem{Name: "cached", Value: 1}, nil
	})
	require.NoError(t, err)

	// Second call should hit cache
	fetchCalled := false
	val, err := Get(ctx, c, "test:hit", time.Minute, func(ctx context.Context) (testItem, error) {
		fetchCalled = true
		return testItem{}, nil
	})

	require.NoError(t, err)
	assert.False(t, fetchCalled)
	assert.Equal(t, "cached", val.Name)
	assert.Equal(t, 1, val.Value)
}

func TestGet_RedisError_FallsThrough(t *testing.T) {
	c, mr := setupTestClient(t)
	ctx := context.Background()

	mr.Close() // Force Redis errors

	fetchCalled := false
	val, err := Get(ctx, c, "test:err", time.Minute, func(ctx context.Context) (testItem, error) {
		fetchCalled = true
		return testItem{Name: "fallback", Value: 99}, nil
	})

	require.NoError(t, err)
	assert.True(t, fetchCalled)
	assert.Equal(t, "fallback", val.Name)
}

func TestGet_FetchError_ReturnsError(t *testing.T) {
	c, _ := setupTestClient(t)
	ctx := context.Background()
	expectedErr := errors.New("db failure")

	_, err := Get(ctx, c, "test:fetcherr", time.Minute, func(ctx context.Context) (testItem, error) {
		return testItem{}, expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
}

func TestGet_NilClient_FallsThrough(t *testing.T) {
	ctx := context.Background()
	fetchCalled := false

	val, err := Get(ctx, nil, "test:nil", time.Minute, func(ctx context.Context) (testItem, error) {
		fetchCalled = true
		return testItem{Name: "direct", Value: 7}, nil
	})

	require.NoError(t, err)
	assert.True(t, fetchCalled)
	assert.Equal(t, "direct", val.Name)
}

func TestGet_NilRedis_FallsThrough(t *testing.T) {
	c := NewClient(nil)
	ctx := context.Background()
	fetchCalled := false

	val, err := Get(ctx, c, "test:nilredis", time.Minute, func(ctx context.Context) (testItem, error) {
		fetchCalled = true
		return testItem{Name: "direct", Value: 8}, nil
	})

	require.NoError(t, err)
	assert.True(t, fetchCalled)
	assert.Equal(t, "direct", val.Name)
}

func TestDelete_RemovesKey(t *testing.T) {
	c, _ := setupTestClient(t)
	ctx := context.Background()

	// Populate cache
	_, _ = Get(ctx, c, "test:del", time.Minute, func(ctx context.Context) (testItem, error) {
		return testItem{Name: "to-delete", Value: 1}, nil
	})
	Delete(ctx, c, "test:del")

	// Should fetch again (cache miss)
	fetchCalled := false
	_, _ = Get(ctx, c, "test:del", time.Minute, func(ctx context.Context) (testItem, error) {
		fetchCalled = true
		return testItem{}, nil
	})
	assert.True(t, fetchCalled)
}

func TestDelete_NilClient_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		Delete(context.Background(), nil, "key")
	})
}

func TestDelete_NilRedis_NoPanic(t *testing.T) {
	c := NewClient(nil)
	assert.NotPanics(t, func() {
		Delete(context.Background(), c, "key")
	})
}

func TestTTLWithJitter_WithinBounds(t *testing.T) {
	base := 10 * time.Minute
	min := time.Duration(float64(base) * 0.85)
	max := time.Duration(float64(base) * 1.15)

	for i := 0; i < 1000; i++ {
		jittered := ttlWithJitter(base)
		assert.GreaterOrEqual(t, jittered, min, "jittered TTL below minimum")
		assert.LessOrEqual(t, jittered, max, "jittered TTL above maximum")
	}
}

func TestTTLWithJitter_ZeroDuration(t *testing.T) {
	assert.Equal(t, time.Duration(0), ttlWithJitter(0))
}
