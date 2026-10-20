package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/middleware"
)

// memIdemRepo is a minimal in-memory idempotency.Repository: first Reserve wins,
// a different hash is a conflict, a stored response is replayed.
type memIdemRepo struct {
	mu   sync.Mutex
	recs map[string]*idempotency.Record
}

func newMemIdemRepo() *memIdemRepo {
	return &memIdemRepo{recs: map[string]*idempotency.Record{}}
}

func (m *memIdemRepo) Reserve(_ context.Context, key string, tenantID, userID uuid.UUID, method, path, hash string) (*idempotency.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := tenantID.String() + "/" + key
	rec, ok := m.recs[k]
	if !ok {
		m.recs[k] = &idempotency.Record{Key: key, TenantID: tenantID, UserID: userID, Method: method, Path: path, RequestHash: hash}
		return nil, nil
	}
	if rec.RequestHash != hash {
		return nil, idempotency.ErrConflict
	}
	if rec.ResponseStatus == nil {
		return nil, idempotency.ErrInFlight
	}
	cp := *rec
	return &cp, nil
}

func (m *memIdemRepo) Get(context.Context, uuid.UUID, string) (*idempotency.Record, error) {
	return nil, idempotency.ErrKeyMissing
}

func (m *memIdemRepo) Complete(_ context.Context, tenantID uuid.UUID, key string, status int, body []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.recs[tenantID.String()+"/"+key]
	if !ok {
		return idempotency.ErrKeyMissing
	}
	rec.ResponseStatus = &status
	rec.ResponseBody = body
	return nil
}

func (m *memIdemRepo) Cleanup(context.Context) (int, error)                { return 0, nil }
func (m *memIdemRepo) CleanupWithLock(context.Context, int64) (int, error) { return 0, nil }

// uploadWithKey sends one upload as the given tenant. Complete runs in a goroutine
// after the response, so an immediate retry can still see the key in flight (409):
// it is re-sent until the key has settled.
func uploadWithKey(t *testing.T, h *FileUploadHandler, tenantID, userID, channelID uuid.UUID, key, filename string) *httptest.ResponseRecorder {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		req := newMultipartUploadRequestWithMessageID(t, userID.String(), channelID.String(), "",
			filename, "application/pdf", []byte("fake-pdf"))
		req = req.WithContext(context.WithValue(req.Context(), middleware.TenantIDKey, tenantID.String()))
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		rec := httptest.NewRecorder()
		h.HandleUploadFile(rec, req)
		if rec.Code != http.StatusConflict || time.Now().After(deadline) {
			return rec
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestFileUpload_IdempotencyKeyDeduplicatesRetries(t *testing.T) {
	h, repo, _, _ := newTestFileUploadHandler()
	h.SetIdempotency(newMemIdemRepo())
	tenantID, userID, channelID := uuid.New(), uuid.New(), uuid.New()
	repo.members[repo.memberKey(channelID, userID)] = true

	first := uploadWithKey(t, h, tenantID, userID, channelID, "key-1", "a.pdf")
	require.Equal(t, http.StatusCreated, first.Code)
	retry := uploadWithKey(t, h, tenantID, userID, channelID, "key-1", "a.pdf")

	require.Equal(t, http.StatusCreated, retry.Code)
	require.Equal(t, "true", retry.Header().Get("Idempotency-Replayed"))
	require.Len(t, repo.files, 1, "retry must not store a second file")
	var a, b map[string]interface{}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &a))
	require.NoError(t, json.Unmarshal(retry.Body.Bytes(), &b))
	require.Equal(t, a["id"], b["id"])
}

func TestFileUpload_IdempotencyKeyReusedForOtherUploadIsRejected(t *testing.T) {
	h, repo, _, _ := newTestFileUploadHandler()
	h.SetIdempotency(newMemIdemRepo())
	tenantID, userID, channelID := uuid.New(), uuid.New(), uuid.New()
	repo.members[repo.memberKey(channelID, userID)] = true

	require.Equal(t, http.StatusCreated, uploadWithKey(t, h, tenantID, userID, channelID, "key-1", "a.pdf").Code)
	other := uploadWithKey(t, h, tenantID, userID, channelID, "key-1", "b.pdf")

	require.Equal(t, http.StatusUnprocessableEntity, other.Code)
	require.Len(t, repo.files, 1)
}

func TestFileUpload_WithoutIdempotencyKeyEveryUploadIsStored(t *testing.T) {
	h, repo, _, _ := newTestFileUploadHandler()
	h.SetIdempotency(newMemIdemRepo())
	tenantID, userID, channelID := uuid.New(), uuid.New(), uuid.New()
	repo.members[repo.memberKey(channelID, userID)] = true

	require.Equal(t, http.StatusCreated, uploadWithKey(t, h, tenantID, userID, channelID, "", "a.pdf").Code)
	require.Equal(t, http.StatusCreated, uploadWithKey(t, h, tenantID, userID, channelID, "", "a.pdf").Code)
	require.Len(t, repo.files, 2)
}
