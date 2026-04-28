package idempotency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/idempotency"
)

// mockRepository is an in-memory implementation of Repository for unit tests.
type mockRepository struct {
	records map[string]*idempotency.Record
}

func newMockRepository() *mockRepository {
	return &mockRepository{records: make(map[string]*idempotency.Record)}
}

func (m *mockRepository) Reserve(
	ctx context.Context,
	key string,
	tenantID, userID uuid.UUID,
	method, path, hash string,
) (*idempotency.Record, error) {
	existing, ok := m.records[key]
	if !ok {
		now := time.Now()
		m.records[key] = &idempotency.Record{
			Key:         key,
			TenantID:    tenantID,
			UserID:      userID,
			Method:      method,
			Path:        path,
			RequestHash: hash,
			CreatedAt:   now,
			ExpiresAt:   now.Add(24 * time.Hour),
		}
		return nil, nil
	}

	if existing.RequestHash != hash {
		return nil, idempotency.ErrConflict
	}
	if existing.CompletedAt == nil {
		return nil, idempotency.ErrInFlight
	}
	return existing, nil
}

func (m *mockRepository) Get(ctx context.Context, key string) (*idempotency.Record, error) {
	rec, ok := m.records[key]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return rec, nil
}

func (m *mockRepository) Complete(ctx context.Context, key string, status int, body []byte) error {
	rec, ok := m.records[key]
	if !ok {
		return pgx.ErrNoRows
	}
	rec.ResponseStatus = &status
	rec.ResponseBody = body
	now := time.Now()
	rec.CompletedAt = &now
	return nil
}

func (m *mockRepository) Cleanup(ctx context.Context) (int, error) {
	now := time.Now()
	count := 0
	for k, rec := range m.records {
		if rec.ExpiresAt.Before(now) {
			delete(m.records, k)
			count++
		}
	}
	return count, nil
}

func TestReserve_FreshKey(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	userID := uuid.New()

	rec, err := repo.Reserve(context.Background(), "key-1", tenantID, userID, "POST", "/api/v1/crm/contacts", "hash1")
	require.NoError(t, err)
	assert.Nil(t, rec, "fresh reserve should return nil record (no replay)")
}

func TestReserve_RaceCondition_InFlight(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	userID := uuid.New()

	// First reserve succeeds
	_, err := repo.Reserve(context.Background(), "key-inflight", tenantID, userID, "POST", "/api/v1/test", "hash-abc")
	require.NoError(t, err)

	// Second reserve with same key+hash, not yet completed → ErrInFlight
	_, err = repo.Reserve(context.Background(), "key-inflight", tenantID, userID, "POST", "/api/v1/test", "hash-abc")
	assert.True(t, errors.Is(err, idempotency.ErrInFlight))
}

func TestReserve_Replay_ReturnsCachedRecord(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	userID := uuid.New()

	_, err := repo.Reserve(context.Background(), "key-replay", tenantID, userID, "POST", "/api/v1/test", "hash-abc")
	require.NoError(t, err)

	status := 201
	body := []byte(`{"id":"abc"}`)
	require.NoError(t, repo.Complete(context.Background(), "key-replay", status, body))

	// Replay: same key+hash after completion → returns existing record
	rec, err := repo.Reserve(context.Background(), "key-replay", tenantID, userID, "POST", "/api/v1/test", "hash-abc")
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, 201, *rec.ResponseStatus)
	assert.Equal(t, body, rec.ResponseBody)
}

func TestReserve_Conflict_DifferentHash(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	userID := uuid.New()

	_, err := repo.Reserve(context.Background(), "key-conflict", tenantID, userID, "POST", "/api/v1/test", "hash-original")
	require.NoError(t, err)

	// Same key, different body → ErrConflict
	_, err = repo.Reserve(context.Background(), "key-conflict", tenantID, userID, "POST", "/api/v1/test", "hash-different")
	assert.True(t, errors.Is(err, idempotency.ErrConflict))
}

func TestCleanup_DeletesExpired(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	userID := uuid.New()

	_, err := repo.Reserve(context.Background(), "key-expired", tenantID, userID, "POST", "/api/v1/test", "hash1")
	require.NoError(t, err)

	// Manually expire the record
	repo.records["key-expired"].ExpiresAt = time.Now().Add(-1 * time.Hour)

	// Keep a live record
	_, err = repo.Reserve(context.Background(), "key-live", tenantID, userID, "POST", "/api/v1/test", "hash2")
	require.NoError(t, err)

	count, err := repo.Cleanup(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	_, err = repo.Get(context.Background(), "key-expired")
	assert.True(t, errors.Is(err, pgx.ErrNoRows), "expired key should be deleted")

	_, err = repo.Get(context.Background(), "key-live")
	assert.NoError(t, err, "live key should remain")
}

func TestComplete_StoresResponse(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	userID := uuid.New()

	_, err := repo.Reserve(context.Background(), "key-complete", tenantID, userID, "POST", "/api/v1/test", "hash1")
	require.NoError(t, err)

	err = repo.Complete(context.Background(), "key-complete", 200, []byte(`{"ok":true}`))
	require.NoError(t, err)

	rec, err := repo.Get(context.Background(), "key-complete")
	require.NoError(t, err)
	assert.NotNil(t, rec.CompletedAt)
	assert.Equal(t, 200, *rec.ResponseStatus)
}
