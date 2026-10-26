package signature

// Repository behaviour against the app-role database (kmuhub_app, RLS on):
// every statement carries its own tenant predicate, so the tests that prove
// it run in system context (which RLS lets through) to keep the SQL itself
// under test, and the ordinary ones run under the tenant context.

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

type sigFixture struct {
	pool     *pgxpool.Pool
	repo     *PostgresRepository
	tenantID uuid.UUID
	userID   uuid.UUID
	ctx      context.Context
}

func newSigFixture(t *testing.T) *sigFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "signature repo")
	userID := seedSigUser(t, pool, tenantID)
	return &sigFixture{
		pool: pool, repo: NewPostgresRepository(pool), tenantID: tenantID, userID: userID,
		ctx: testutil.WithTenantCtx(context.Background(), tenantID),
	}
}

func seedSigUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": "sig-" + uuid.NewString()[:8] + "@example.local", "password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() {
		testutil.CleanupRow(t, pool, "users", id)
	})
	return id
}

// add stores one signature and registers its cleanup (before the user's, so the FK is free).
func (f *sigFixture) add(t *testing.T, userID uuid.UUID, name, html string, isDefault bool) *models.EmailSignature {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	sig := &models.EmailSignature{
		ID: uuid.New(), TenantID: f.tenantID, UserID: userID, Name: name, HTMLContent: html,
		IsDefault: isDefault, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, f.repo.Create(f.ctx, sig))
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "email_signatures", sig.ID) })
	return sig
}

func TestSignatureRepo_CreateAndGetByID_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)

	sig := f.add(t, f.userID, "Work", "<p>Best,<br>Alice</p>", true)

	got, err := f.repo.GetByID(f.ctx, sig.ID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, sig.ID, got.ID)
	assert.Equal(t, f.tenantID, got.TenantID)
	assert.Equal(t, f.userID, got.UserID)
	assert.Equal(t, "Work", got.Name)
	assert.Equal(t, "<p>Best,<br>Alice</p>", got.HTMLContent)
	assert.True(t, got.IsDefault)
	assert.WithinDuration(t, sig.CreatedAt, got.CreatedAt, time.Millisecond)
	assert.WithinDuration(t, sig.UpdatedAt, got.UpdatedAt, time.Millisecond)

	_, err = f.repo.GetByID(f.ctx, uuid.New(), f.tenantID)
	assert.ErrorIs(t, err, ErrSignatureNotFound)
}

func TestSignatureRepo_CreateRejectsForeignTenant_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)

	other := uuid.New()
	testutil.EnsureTenant(t, f.pool, other, "signature repo other")
	now := time.Now().UTC()
	sig := &models.EmailSignature{
		ID: uuid.New(), TenantID: other, UserID: f.userID, Name: "Foreign", CreatedAt: now, UpdatedAt: now,
	}
	// Stamped with another tenant than the context's: the RLS policy refuses the row.
	if err := f.repo.Create(f.ctx, sig); err == nil {
		testutil.CleanupRow(t, f.pool, "email_signatures", sig.ID)
		t.Fatal("create with a foreign tenant_id succeeded, want an RLS rejection")
	}
	testutil.AssertRowCount(t, f.pool, testutil.WithSystemCtx(context.Background()), "email_signatures", sig.ID, 0)
}

func TestSignatureRepo_ReadsAreTenantScoped_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)

	sig := f.add(t, f.userID, "Scoped", "<p>x</p>", true)
	sysCtx := testutil.WithSystemCtx(context.Background())
	other := uuid.New()

	// System context bypasses RLS, so only the statement's own tenant predicate can hide the row.
	_, err := f.repo.GetByID(sysCtx, sig.ID, other)
	assert.ErrorIs(t, err, ErrSignatureNotFound, "GetByID must filter on the tenant")
	_, err = f.repo.GetDefault(sysCtx, f.userID, other)
	assert.ErrorIs(t, err, ErrSignatureNotFound, "GetDefault must filter on the tenant")
	list, err := f.repo.ListByUser(sysCtx, f.userID, other)
	require.NoError(t, err)
	assert.Empty(t, list, "ListByUser must filter on the tenant")
	count, err := f.repo.CountByUser(sysCtx, f.userID, other)
	require.NoError(t, err)
	assert.Zero(t, count, "CountByUser must filter on the tenant")

	// And the same calls with the right tenant do see the row.
	_, err = f.repo.GetByID(sysCtx, sig.ID, f.tenantID)
	assert.NoError(t, err)
	count, err = f.repo.CountByUser(sysCtx, f.userID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSignatureRepo_GetDefault_PicksTheUsersDefault_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)
	otherUser := seedSigUser(t, f.pool, f.tenantID)

	_, err := f.repo.GetDefault(f.ctx, f.userID, f.tenantID)
	assert.ErrorIs(t, err, ErrSignatureNotFound, "no signature yet")

	f.add(t, f.userID, "Plain", "a", false)
	_, err = f.repo.GetDefault(f.ctx, f.userID, f.tenantID)
	assert.ErrorIs(t, err, ErrSignatureNotFound, "a non-default signature is not the default")

	// Another user's default must not leak into this user's lookup.
	theirs := f.add(t, otherUser, "Theirs", "t", true)
	_, err = f.repo.GetDefault(f.ctx, f.userID, f.tenantID)
	assert.ErrorIs(t, err, ErrSignatureNotFound, "the default of another user must not be returned")

	mine := f.add(t, f.userID, "Mine", "m", true)
	got, err := f.repo.GetDefault(f.ctx, f.userID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, mine.ID, got.ID)
	assert.True(t, got.IsDefault)
	assert.Equal(t, "m", got.HTMLContent)

	got, err = f.repo.GetDefault(f.ctx, otherUser, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, theirs.ID, got.ID)
}

func TestSignatureRepo_ListByUser_OrdersDefaultFirstThenName_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)
	otherUser := seedSigUser(t, f.pool, f.tenantID)

	zulu := f.add(t, f.userID, "Zulu", "z", true)
	bravo := f.add(t, f.userID, "Bravo", "b", false)
	alpha := f.add(t, f.userID, "Alpha", "a", false)
	f.add(t, otherUser, "NotMine", "n", false)

	list, err := f.repo.ListByUser(f.ctx, f.userID, f.tenantID)
	require.NoError(t, err)
	require.Len(t, list, 3, "only the user's own signatures")
	assert.Equal(t, []uuid.UUID{zulu.ID, alpha.ID, bravo.ID}, []uuid.UUID{list[0].ID, list[1].ID, list[2].ID},
		"default first, then by name ascending")
	assert.Equal(t, "z", list[0].HTMLContent)
	assert.Equal(t, f.userID, list[1].UserID)

	empty, err := f.repo.ListByUser(f.ctx, uuid.New(), f.tenantID)
	require.NoError(t, err)
	assert.Empty(t, empty)

	// The count is per user as well: the other user's signature is not included.
	count, err := f.repo.CountByUser(f.ctx, f.userID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestSignatureRepo_Update_WritesFieldsAndBumpsUpdatedAt_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)

	sig := f.add(t, f.userID, "Old", "old", false)
	other := f.add(t, f.userID, "Untouched", "keep", false)

	sig.Name, sig.HTMLContent, sig.IsDefault = "New", "<b>new</b>", true
	require.NoError(t, f.repo.Update(f.ctx, sig))
	assert.True(t, sig.UpdatedAt.After(sig.CreatedAt), "Update stamps UpdatedAt on the struct")

	got, err := f.repo.GetByID(f.ctx, sig.ID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, "New", got.Name)
	assert.Equal(t, "<b>new</b>", got.HTMLContent)
	assert.True(t, got.IsDefault)
	assert.WithinDuration(t, sig.UpdatedAt, got.UpdatedAt, time.Millisecond)
	assert.True(t, got.UpdatedAt.After(got.CreatedAt))

	untouched, err := f.repo.GetByID(f.ctx, other.ID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, "Untouched", untouched.Name, "the update is scoped to its own row")
	assert.False(t, untouched.IsDefault)
}

func TestSignatureRepo_Update_IgnoresForeignTenant_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)

	sig := f.add(t, f.userID, "Mine", "mine", false)
	sysCtx := testutil.WithSystemCtx(context.Background())

	// System context bypasses RLS: only the UPDATE's own tenant predicate keeps the row untouched.
	attempt := *sig
	attempt.TenantID = uuid.New()
	attempt.Name, attempt.HTMLContent = "Hijacked", "evil"
	require.NoError(t, f.repo.Update(sysCtx, &attempt))

	got, err := f.repo.GetByID(f.ctx, sig.ID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, "Mine", got.Name)
	assert.Equal(t, "mine", got.HTMLContent)
}

func TestSignatureRepo_Delete_RemovesOnlyTheTenantsRow_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)

	sig := f.add(t, f.userID, "Gone", "g", false)
	keep := f.add(t, f.userID, "Keep", "k", false)
	sysCtx := testutil.WithSystemCtx(context.Background())

	// A delete naming another tenant removes nothing, even where RLS lets the row through.
	require.NoError(t, f.repo.Delete(sysCtx, sig.ID, uuid.New()))
	testutil.AssertRowCount(t, f.pool, sysCtx, "email_signatures", sig.ID, 1)

	require.NoError(t, f.repo.Delete(f.ctx, sig.ID, f.tenantID))
	testutil.AssertRowCount(t, f.pool, sysCtx, "email_signatures", sig.ID, 0)
	testutil.AssertRowCount(t, f.pool, sysCtx, "email_signatures", keep.ID, 1)

	// Deleting a row that is already gone is not an error (idempotent).
	require.NoError(t, f.repo.Delete(f.ctx, sig.ID, f.tenantID))
}

func TestSignatureRepo_ClearDefaultForUser_ScopesUserAndTenant_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)
	otherUser := seedSigUser(t, f.pool, f.tenantID)
	sysCtx := testutil.WithSystemCtx(context.Background())

	mine := f.add(t, f.userID, "Mine", "m", true)
	mine2 := f.add(t, f.userID, "Mine2", "m2", false)
	theirs := f.add(t, otherUser, "Theirs", "t", true)
	before := mine.UpdatedAt

	// Naming another tenant clears nothing, even in system context.
	require.NoError(t, f.repo.ClearDefaultForUser(sysCtx, f.userID, uuid.New()))
	got, err := f.repo.GetByID(f.ctx, mine.ID, f.tenantID)
	require.NoError(t, err)
	assert.True(t, got.IsDefault, "foreign tenant must not clear the default")

	require.NoError(t, f.repo.ClearDefaultForUser(f.ctx, f.userID, f.tenantID))

	got, err = f.repo.GetByID(f.ctx, mine.ID, f.tenantID)
	require.NoError(t, err)
	assert.False(t, got.IsDefault)
	assert.True(t, got.UpdatedAt.After(before), "clearing the default stamps updated_at")
	got, err = f.repo.GetByID(f.ctx, mine2.ID, f.tenantID)
	require.NoError(t, err)
	assert.False(t, got.IsDefault)
	got, err = f.repo.GetByID(f.ctx, theirs.ID, f.tenantID)
	require.NoError(t, err)
	assert.True(t, got.IsDefault, "another user's default stays")
}

func TestSignatureRepo_ErrorsOnClosedPool_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)
	sig := f.add(t, f.userID, "X", "x", true)

	pool := testutil.PoolFromEnv(t)
	repo := NewPostgresRepository(pool)
	pool.Close()
	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)

	assert.Error(t, repo.Create(ctx, sig))
	_, err := repo.GetByID(ctx, sig.ID, f.tenantID)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrSignatureNotFound, "a connection failure is not a missing row")
	_, err = repo.GetDefault(ctx, f.userID, f.tenantID)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrSignatureNotFound)
	_, err = repo.ListByUser(ctx, f.userID, f.tenantID)
	assert.Error(t, err)
	assert.Error(t, repo.Update(ctx, sig))
	assert.Error(t, repo.Delete(ctx, sig.ID, f.tenantID))
	assert.Error(t, repo.ClearDefaultForUser(ctx, f.userID, f.tenantID))
	_, err = repo.CountByUser(ctx, f.userID, f.tenantID)
	assert.Error(t, err)
}

// TestSignatureService_LifecycleOnRealDB runs the service over the real
// repository: first signature becomes the default, later ones do not, and
// SetDefault moves the flag so that exactly one default remains.
func TestSignatureService_LifecycleOnRealDB_DB(t *testing.T) {
	t.Parallel()
	f := newSigFixture(t)
	svc := NewService(f.repo)

	first, err := svc.Create(f.ctx, f.userID, f.tenantID, "  First  ", "<p>1</p>")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "email_signatures", first.ID) })
	assert.True(t, first.IsDefault, "the user's first signature is the default")
	assert.Equal(t, "First", first.Name, "name is trimmed")

	second, err := svc.Create(f.ctx, f.userID, f.tenantID, "Second", "<p>2</p>")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "email_signatures", second.ID) })
	assert.False(t, second.IsDefault)

	require.NoError(t, svc.SetDefault(f.ctx, second.ID, f.tenantID))
	def, err := svc.GetDefault(f.ctx, f.userID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, second.ID, def.ID)
	stale, err := svc.GetByID(f.ctx, first.ID, f.tenantID)
	require.NoError(t, err)
	assert.False(t, stale.IsDefault, "the previous default was cleared")

	updated, err := svc.Update(f.ctx, first.ID, f.tenantID, "", "<p>edited</p>")
	require.NoError(t, err)
	assert.Equal(t, "First", updated.Name, "an empty name keeps the old one")
	assert.Equal(t, "<p>edited</p>", updated.HTMLContent)

	list, err := svc.ListByUser(f.ctx, f.userID, f.tenantID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, second.ID, list[0].ID, "default first")

	require.NoError(t, svc.Delete(f.ctx, first.ID, f.tenantID))
	_, err = svc.GetByID(f.ctx, first.ID, f.tenantID)
	assert.ErrorIs(t, err, ErrSignatureNotFound)

	// Validation and not-found paths of the service.
	_, err = svc.Create(f.ctx, f.userID, f.tenantID, "   ", "x")
	assert.Error(t, err)
	assert.ErrorIs(t, svc.SetDefault(f.ctx, uuid.New(), f.tenantID), ErrSignatureNotFound)
	_, err = svc.Update(f.ctx, uuid.New(), f.tenantID, "n", "h")
	assert.ErrorIs(t, err, ErrSignatureNotFound)
}
