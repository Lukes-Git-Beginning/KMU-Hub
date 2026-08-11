package thread

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func newTestPool(t *testing.T) (repo *PostgresRepository, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool = testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID = uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Canned Response Tenant")

	return NewPostgresRepository(pool), pool, testutil.WithTenantCtx(context.Background(), tenantID), tenantID
}

func TestCannedResponses_CreateGetListUpdateDelete(t *testing.T) {
	repo, _, ctx, tenantID := newTestPool(t)

	c := &CannedResponse{
		TenantID: tenantID,
		Name:     "Willkommen",
		Body:     "Hallo, danke fuer Ihre Nachricht.",
	}
	require.NoError(t, repo.CreateCannedResponse(ctx, c))
	require.NotEqual(t, uuid.Nil, c.ID)

	got, err := repo.GetCannedResponse(ctx, tenantID, c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.Name, got.Name)
	assert.Equal(t, c.Body, got.Body)

	c2 := &CannedResponse{TenantID: tenantID, Name: "Abschluss", Body: "Vielen Dank."}
	require.NoError(t, repo.CreateCannedResponse(ctx, c2))

	list, err := repo.ListCannedResponses(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// ORDER BY name ASC: "Abschluss" sorts before "Willkommen".
	assert.Equal(t, "Abschluss", list[0].Name)
	assert.Equal(t, "Willkommen", list[1].Name)

	got.Name = "Willkommen (neu)"
	got.Body = "Hallo, danke fuer Ihre Nachricht! Wir melden uns."
	require.NoError(t, repo.UpdateCannedResponse(ctx, got))

	updated, err := repo.GetCannedResponse(ctx, tenantID, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "Willkommen (neu)", updated.Name)

	require.NoError(t, repo.DeleteCannedResponse(ctx, tenantID, c.ID))
	_, err = repo.GetCannedResponse(ctx, tenantID, c.ID)
	assert.ErrorIs(t, err, ErrCannedResponseNotFound)
}

func TestGetCannedResponse_NotFound(t *testing.T) {
	repo, _, ctx, tenantID := newTestPool(t)

	_, err := repo.GetCannedResponse(ctx, tenantID, uuid.New())

	assert.ErrorIs(t, err, ErrCannedResponseNotFound)
}

func TestUpdateCannedResponse_NotFound(t *testing.T) {
	repo, _, ctx, tenantID := newTestPool(t)

	c := &CannedResponse{ID: uuid.New(), TenantID: tenantID, Name: "Ghost", Body: "n/a"}

	err := repo.UpdateCannedResponse(ctx, c)

	assert.ErrorIs(t, err, ErrCannedResponseNotFound)
}

func TestDeleteCannedResponse_NotFound(t *testing.T) {
	repo, _, ctx, tenantID := newTestPool(t)

	err := repo.DeleteCannedResponse(ctx, tenantID, uuid.New())

	assert.ErrorIs(t, err, ErrCannedResponseNotFound)
}

func TestCannedResponses_CrossTenantIsolation(t *testing.T) {
	repo, pool, ctx, tenantID := newTestPool(t)

	c := &CannedResponse{TenantID: tenantID, Name: "Nur fuer uns", Body: "geheim"}
	require.NoError(t, repo.CreateCannedResponse(ctx, c))

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Other Inbox Canned Response Tenant")
	otherCtx := testutil.WithTenantCtx(context.Background(), otherTenantID)

	_, err := repo.GetCannedResponse(otherCtx, otherTenantID, c.ID)
	assert.ErrorIs(t, err, ErrCannedResponseNotFound)

	list, err := repo.ListCannedResponses(otherCtx, otherTenantID)
	require.NoError(t, err)
	assert.Empty(t, list)
}
