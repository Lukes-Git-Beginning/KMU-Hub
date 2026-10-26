package account

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// These tests run against DATABASE_URL as kmuhub_app (RLS enforced) and cover
// the SQL the mock-based service tests never reach.

func envAccountRepo(t *testing.T) (*PostgresRepository, *pgxpool.Pool) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	return NewPostgresRepository(pool), pool
}

// envAccountTenant creates a tenant plus one user and removes what the test wrote.
func envAccountTenant(t *testing.T, pool *pgxpool.Pool) (tenantID, userID uuid.UUID, ctx context.Context) {
	t.Helper()
	tenantID, userID = uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "acct-env-"+tenantID.String()[:8])
	sys := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(sys, `INSERT INTO users (id, email, password_hash, tenant_id) VALUES ($1, $2, 'x', $3)`,
		userID, userID.String()+"@acct-env.test", tenantID)
	require.NoError(t, err)
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM email_accounts WHERE tenant_id = $1`,
			`DELETE FROM users WHERE tenant_id = $1`,
			`DELETE FROM tenants WHERE id = $1`,
		} {
			if _, err := pool.Exec(sys, q, tenantID); err != nil {
				t.Logf("cleanup %q: %v", q, err)
			}
		}
	})
	return tenantID, userID, testutil.WithTenantCtx(context.Background(), tenantID)
}

func envAccount(tenantID, userID uuid.UUID, addr string, createdAt time.Time) *models.EmailAccount {
	return &models.EmailAccount{
		ID: uuid.New(), TenantID: tenantID, UserID: userID,
		EmailAddress: addr, DisplayName: "Env " + addr,
		IMAPHost: "imap.example.com", IMAPPort: 993, SMTPHost: "smtp.example.com", SMTPPort: 587,
		Username: addr, PasswordEncrypted: "enc:" + addr,
		UseSSL: true, SyncEnabled: true,
		CreatedAt: createdAt, UpdatedAt: createdAt,
	}
}

func TestPostgresRepository_CreateGetRoundtrip(t *testing.T) {
	repo, pool := envAccountRepo(t)
	tenantID, userID, ctx := envAccountTenant(t, pool)

	acct := envAccount(tenantID, userID, "a@env.test", time.Now().UTC().Truncate(time.Microsecond))
	acct.IsDefault = true
	require.NoError(t, repo.Create(ctx, acct))

	got, err := repo.GetByID(ctx, acct.ID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, acct.EmailAddress, got.EmailAddress)
	assert.Equal(t, acct.DisplayName, got.DisplayName)
	assert.Equal(t, 993, got.IMAPPort)
	assert.Equal(t, 587, got.SMTPPort)
	assert.Equal(t, "enc:a@env.test", got.PasswordEncrypted)
	assert.True(t, got.UseSSL)
	assert.True(t, got.SyncEnabled)
	assert.True(t, got.IsDefault)
	assert.Nil(t, got.LastSyncAt)

	_, err = repo.GetByID(ctx, uuid.New(), tenantID)
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

func TestPostgresRepository_TenantBoundary(t *testing.T) {
	repo, pool := envAccountRepo(t)
	tenantA, userA, ctxA := envAccountTenant(t, pool)
	tenantB, _, ctxB := envAccountTenant(t, pool)

	acct := envAccount(tenantA, userA, "a@env.test", time.Now().UTC())
	acct.IsDefault = true
	require.NoError(t, repo.Create(ctxA, acct))

	// Reads under the foreign tenant find nothing, whichever key is used.
	_, err := repo.GetByID(ctxB, acct.ID, tenantB)
	assert.ErrorIs(t, err, ErrAccountNotFound)
	_, err = repo.GetByID(ctxB, acct.ID, tenantA)
	assert.ErrorIs(t, err, ErrAccountNotFound, "RLS hides the row even when the caller passes the owner tenant id")
	_, err = repo.GetByUserIDAndTenant(ctxB, userA, tenantB)
	assert.ErrorIs(t, err, ErrAccountNotFound)

	list, err := repo.ListByUserAndTenant(ctxB, userA, tenantB)
	require.NoError(t, err)
	assert.Empty(t, list)
	active, err := repo.ListActive(ctxB, tenantB)
	require.NoError(t, err)
	assert.Empty(t, active)

	// A foreign-tenant delete and update change nothing.
	require.NoError(t, repo.Delete(ctxB, acct.ID, tenantB))
	foreign := *acct
	foreign.TenantID = tenantB
	foreign.DisplayName = "hijacked"
	require.NoError(t, repo.Update(ctxB, &foreign))
	got, err := repo.GetByID(ctxA, acct.ID, tenantA)
	require.NoError(t, err)
	assert.Equal(t, "Env a@env.test", got.DisplayName)
}

func TestPostgresRepository_ListByUserOrderAndDefault(t *testing.T) {
	repo, pool := envAccountRepo(t)
	tenantID, userID, ctx := envAccountTenant(t, pool)

	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	first := envAccount(tenantID, userID, "first@env.test", base)
	second := envAccount(tenantID, userID, "second@env.test", base.Add(time.Minute))
	third := envAccount(tenantID, userID, "third@env.test", base.Add(2*time.Minute))
	first.IsDefault = true
	// Insert out of order: the list must come back oldest first regardless.
	for _, a := range []*models.EmailAccount{third, first, second} {
		require.NoError(t, repo.Create(ctx, a))
	}

	list, err := repo.ListByUserAndTenant(ctx, userID, tenantID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, []string{"first@env.test", "second@env.test", "third@env.test"},
		[]string{list[0].EmailAddress, list[1].EmailAddress, list[2].EmailAddress})

	def, err := repo.GetByUserIDAndTenant(ctx, userID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, def.ID)

	// SetDefault moves the flag in one statement: exactly one default remains.
	require.NoError(t, repo.SetDefault(ctx, second.ID, userID, tenantID))
	def, err = repo.GetByUserIDAndTenant(ctx, userID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, second.ID, def.ID)
	list, err = repo.ListByUserAndTenant(ctx, userID, tenantID)
	require.NoError(t, err)
	defaults := 0
	for _, a := range list {
		if a.IsDefault {
			defaults++
		}
	}
	assert.Equal(t, 1, defaults)
}

func TestPostgresRepository_SetDefaultLeavesOtherUsersAlone(t *testing.T) {
	repo, pool := envAccountRepo(t)
	tenantID, userID, ctx := envAccountTenant(t, pool)

	otherUser := uuid.New()
	sys := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(sys, `INSERT INTO users (id, email, password_hash, tenant_id) VALUES ($1, $2, 'x', $3)`,
		otherUser, otherUser.String()+"@acct-env.test", tenantID)
	require.NoError(t, err)

	mine := envAccount(tenantID, userID, "mine@env.test", time.Now().UTC())
	theirs := envAccount(tenantID, otherUser, "theirs@env.test", time.Now().UTC())
	theirs.IsDefault = true
	require.NoError(t, repo.Create(ctx, mine))
	require.NoError(t, repo.Create(ctx, theirs))

	require.NoError(t, repo.SetDefault(ctx, mine.ID, userID, tenantID))

	got, err := repo.GetByID(ctx, theirs.ID, tenantID)
	require.NoError(t, err)
	assert.True(t, got.IsDefault, "another user's default must survive")
}

func TestPostgresRepository_UpdateAndDelete(t *testing.T) {
	repo, pool := envAccountRepo(t)
	tenantID, userID, ctx := envAccountTenant(t, pool)

	acct := envAccount(tenantID, userID, "u@env.test", time.Now().UTC().Add(-time.Hour))
	require.NoError(t, repo.Create(ctx, acct))

	synced := time.Now().UTC().Truncate(time.Microsecond)
	acct.DisplayName = "Renamed"
	acct.IMAPPort = 143
	acct.UseSSL = false
	acct.SyncEnabled = false
	acct.LastSyncAt = &synced
	before := acct.UpdatedAt
	require.NoError(t, repo.Update(ctx, acct))
	assert.True(t, acct.UpdatedAt.After(before), "Update stamps updated_at")

	got, err := repo.GetByID(ctx, acct.ID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.DisplayName)
	assert.Equal(t, 143, got.IMAPPort)
	assert.False(t, got.UseSSL)
	assert.False(t, got.SyncEnabled)
	require.NotNil(t, got.LastSyncAt)
	assert.WithinDuration(t, synced, *got.LastSyncAt, time.Second)

	require.NoError(t, repo.Delete(ctx, acct.ID, tenantID))
	_, err = repo.GetByID(ctx, acct.ID, tenantID)
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

func TestPostgresRepository_ListActiveAndAll(t *testing.T) {
	repo, pool := envAccountRepo(t)
	tenantA, userA, ctxA := envAccountTenant(t, pool)
	tenantB, userB, ctxB := envAccountTenant(t, pool)

	on := envAccount(tenantA, userA, "on@env.test", time.Now().UTC())
	off := envAccount(tenantA, userA, "off@env.test", time.Now().UTC())
	off.SyncEnabled = false
	onB := envAccount(tenantB, userB, "onb@env.test", time.Now().UTC())
	require.NoError(t, repo.Create(ctxA, on))
	require.NoError(t, repo.Create(ctxA, off))
	require.NoError(t, repo.Create(ctxB, onB))

	active, err := repo.ListActive(ctxA, tenantA)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, on.ID, active[0].ID, "sync-disabled account is excluded")

	// ListAllActive is the system-wide sweep: it needs the system context and
	// then sees every tenant's enabled accounts, still without the disabled one.
	all, err := repo.ListAllActive(testutil.WithSystemCtx(context.Background()))
	require.NoError(t, err)
	ids := map[uuid.UUID]bool{}
	for _, a := range all {
		ids[a.ID] = true
	}
	assert.True(t, ids[on.ID])
	assert.True(t, ids[onB.ID])
	assert.False(t, ids[off.ID])

	// Under a tenant context RLS narrows even the cross-tenant query.
	scoped, err := repo.ListAllActive(ctxA)
	require.NoError(t, err)
	for _, a := range scoped {
		assert.Equal(t, tenantA, a.TenantID)
	}
}
