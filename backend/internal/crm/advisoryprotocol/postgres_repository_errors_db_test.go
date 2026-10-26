package advisoryprotocol

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRepository_CancelledContextSurfacesOnEveryQuery(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx, cancel := context.WithCancel(testutil.WithTenantCtx(context.Background(), uuid.New()))
	cancel()
	tid, id := uuid.New(), uuid.New()
	p := fullProtocol(id, tid, uuid.New(), uuid.New())

	assert.Error(t, repo.Create(ctx, p))
	_, err := repo.GetByID(ctx, id, tid)
	assert.Error(t, err)
	_, err = repo.ListByContact(ctx, uuid.New(), tid)
	assert.Error(t, err)
	assert.Error(t, repo.Update(ctx, p))
	assert.Error(t, repo.Delete(ctx, id, tid))
	assert.Error(t, repo.HandOver(ctx, id, tid, time.Now()))
	_, err = repo.ContactExists(ctx, uuid.New(), tid)
	assert.Error(t, err)
	_, err = repo.GetReferralReport(ctx, tid)
	assert.Error(t, err)
}

func TestArrayHelpers_NilBecomesEmptySlice(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, textArrayOrEmpty(nil))
	assert.Empty(t, textArrayOrEmpty(nil))
	assert.Equal(t, []string{"a"}, textArrayOrEmpty([]string{"a"}))
	assert.NotNil(t, orEmpty(nil))
	assert.Empty(t, orEmpty(nil))
	assert.Equal(t, []string{"a"}, orEmpty([]string{"a"}))
}
