package share_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/document/share"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// shareWorld is one tenant with an owner and a recipient. Tenants are minted per
// test so parallel packages never collide on shared constants.
type shareWorld struct {
	pool      *pgxpool.Pool
	repo      *share.PostgresRepository
	svc       *share.Service
	tenant    uuid.UUID
	owner     uuid.UUID
	recipient uuid.UUID
	ctx       context.Context
}

func seedUser(t *testing.T, pool *pgxpool.Pool, tenant uuid.UUID, first string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenant,
		"email":         "share-" + first + "-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": first, "last_name": "Muster",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id
}

func newShareWorld(t *testing.T) *shareWorld {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "document share test")

	repo := share.NewPostgresRepository(pool)
	return &shareWorld{
		pool: pool, repo: repo, svc: share.NewService(repo), tenant: tenant,
		owner:     seedUser(t, pool, tenant, "Olga"),
		recipient: seedUser(t, pool, tenant, "Rudi"),
		ctx:       testutil.WithTenantCtx(context.Background(), tenant),
	}
}

func (w *shareWorld) share(t *testing.T, entityType string, entityID, with uuid.UUID, permission string) *models.DocumentShare {
	t.Helper()
	got, err := w.svc.ShareEntity(w.ctx, share.ShareInput{
		TenantID: w.tenant, EntityType: entityType, EntityID: entityID,
		SharedWithUserID: with, Permission: permission, SharedBy: w.owner,
	})
	if err != nil {
		t.Fatalf("ShareEntity: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, w.pool, "document_shares", got.ID) })
	return got
}

func TestRepository_ShareRoundTripsWithUserNames(t *testing.T) {
	w := newShareWorld(t)
	fileID := uuid.New()

	created := w.share(t, models.ShareEntityFile, fileID, w.recipient, models.SharePermissionWrite)

	got, err := w.repo.GetUserPermission(w.ctx, models.ShareEntityFile, fileID, w.recipient)
	if err != nil {
		t.Fatalf("GetUserPermission: %v", err)
	}
	if got.ID != created.ID || got.EntityID != fileID || got.EntityType != models.ShareEntityFile {
		t.Fatalf("identity = %s/%s/%s, want %s/%s/file", got.ID, got.EntityType, got.EntityID, created.ID, fileID)
	}
	if got.SharedWithUserID != w.recipient || got.SharedBy != w.owner || got.Permission != models.SharePermissionWrite {
		t.Fatalf("columns = with %s by %s perm %q", got.SharedWithUserID, got.SharedBy, got.Permission)
	}
	if got.SharedWithUserName != "Rudi Muster" || got.SharedByName != "Olga Muster" {
		t.Fatalf("names = %q / %q, want %q / %q", got.SharedWithUserName, got.SharedByName, "Rudi Muster", "Olga Muster")
	}
	if time.Since(got.CreatedAt) > time.Minute {
		t.Fatalf("CreatedAt = %s, want a fresh timestamp", got.CreatedAt)
	}
}

func TestService_ShareTwiceWithSameUserIsRejected(t *testing.T) {
	w := newShareWorld(t)
	fileID := uuid.New()
	w.share(t, models.ShareEntityFile, fileID, w.recipient, models.SharePermissionRead)

	_, err := w.svc.ShareEntity(w.ctx, share.ShareInput{
		TenantID: w.tenant, EntityType: models.ShareEntityFile, EntityID: fileID,
		SharedWithUserID: w.recipient, Permission: models.SharePermissionWrite, SharedBy: w.owner,
	})
	if !errors.Is(err, share.ErrAlreadyShared) {
		t.Fatalf("second ShareEntity = %v, want ErrAlreadyShared", err)
	}
}

func TestRepository_CreateRejectsUnknownRecipient(t *testing.T) {
	w := newShareWorld(t)

	err := w.repo.Create(w.ctx, &models.DocumentShare{
		ID: uuid.New(), TenantID: w.tenant, EntityType: models.ShareEntityFile, EntityID: uuid.New(),
		SharedWithUserID: uuid.New(), Permission: models.SharePermissionRead, SharedBy: w.owner,
		CreatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("Create with a non-existent recipient succeeded, want a foreign-key error")
	}
}

func TestRepository_CreateRejectsInvalidPermissionAtDatabase(t *testing.T) {
	w := newShareWorld(t)

	err := w.repo.Create(w.ctx, &models.DocumentShare{
		ID: uuid.New(), TenantID: w.tenant, EntityType: models.ShareEntityFile, EntityID: uuid.New(),
		SharedWithUserID: w.recipient, Permission: "admin", SharedBy: w.owner, CreatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("Create with permission=admin succeeded, want the CHECK constraint to reject it")
	}
}

func TestRepository_ListByEntityIsScopedToEntityAndNewestFirst(t *testing.T) {
	w := newShareWorld(t)
	fileID := uuid.New()
	third := seedUser(t, w.pool, w.tenant, "Tina")

	first := w.share(t, models.ShareEntityFile, fileID, w.recipient, models.SharePermissionRead)
	time.Sleep(5 * time.Millisecond)
	second := w.share(t, models.ShareEntityFile, fileID, third, models.SharePermissionWrite)
	// Same id, other entity type, must not leak into the file listing.
	w.share(t, models.ShareEntityFolder, fileID, w.recipient, models.SharePermissionRead)
	// Different entity entirely.
	w.share(t, models.ShareEntityFile, uuid.New(), w.recipient, models.SharePermissionRead)

	got, err := w.svc.ListByEntity(w.ctx, models.ShareEntityFile, fileID)
	if err != nil {
		t.Fatalf("ListByEntity: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (file shares of this entity only)", len(got))
	}
	if got[0].ID != second.ID || got[1].ID != first.ID {
		t.Fatalf("order = %s, %s; want newest first (%s, %s)", got[0].ID, got[1].ID, second.ID, first.ID)
	}
	if got[0].SharedWithUserName != "Tina Muster" {
		t.Fatalf("SharedWithUserName = %q, want %q", got[0].SharedWithUserName, "Tina Muster")
	}
}

func TestRepository_ListByEntityWithoutSharesIsEmptyNotError(t *testing.T) {
	w := newShareWorld(t)

	got, err := w.svc.ListByEntity(w.ctx, models.ShareEntityFolder, uuid.New())
	if err != nil {
		t.Fatalf("ListByEntity: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

func TestRepository_ListSharedWithUserFiltersByRecipientAndType(t *testing.T) {
	w := newShareWorld(t)
	other := seedUser(t, w.pool, w.tenant, "Otto")

	mine := w.share(t, models.ShareEntityFile, uuid.New(), w.recipient, models.SharePermissionRead)
	w.share(t, models.ShareEntityFolder, uuid.New(), w.recipient, models.SharePermissionRead)
	w.share(t, models.ShareEntityFile, uuid.New(), other, models.SharePermissionRead)

	got, err := w.svc.ListSharedWithMe(w.ctx, w.recipient, models.ShareEntityFile)
	if err != nil {
		t.Fatalf("ListSharedWithMe: %v", err)
	}
	if len(got) != 1 || got[0].ID != mine.ID {
		t.Fatalf("got %d shares, want exactly the recipient's file share %s", len(got), mine.ID)
	}
}

func TestService_CheckAccessReturnsGrantedPermissionOnly(t *testing.T) {
	w := newShareWorld(t)
	fileID := uuid.New()
	other := seedUser(t, w.pool, w.tenant, "Otto")
	w.share(t, models.ShareEntityFile, fileID, w.recipient, models.SharePermissionWrite)

	ok, perm, err := w.svc.CheckAccess(w.ctx, models.ShareEntityFile, fileID, w.recipient)
	if err != nil || !ok || perm != models.SharePermissionWrite {
		t.Fatalf("recipient = %v/%q/%v, want true/write/nil", ok, perm, err)
	}
	ok, perm, err = w.svc.CheckAccess(w.ctx, models.ShareEntityFile, fileID, other)
	if err != nil || ok || perm != "" {
		t.Fatalf("unrelated user = %v/%q/%v, want false/\"\"/nil", ok, perm, err)
	}
	ok, _, err = w.svc.CheckAccess(w.ctx, models.ShareEntityFolder, fileID, w.recipient)
	if err != nil || ok {
		t.Fatalf("same id as folder = %v/%v, want no access", ok, err)
	}
}

func TestService_UnshareRevokesAccess(t *testing.T) {
	w := newShareWorld(t)
	fileID := uuid.New()
	created := w.share(t, models.ShareEntityFile, fileID, w.recipient, models.SharePermissionRead)

	if err := w.svc.UnshareEntity(w.ctx, created.ID); err != nil {
		t.Fatalf("UnshareEntity: %v", err)
	}
	ok, _, err := w.svc.CheckAccess(w.ctx, models.ShareEntityFile, fileID, w.recipient)
	if err != nil || ok {
		t.Fatalf("CheckAccess after revoke = %v/%v, want no access", ok, err)
	}
	if _, err := w.repo.GetUserPermission(w.ctx, models.ShareEntityFile, fileID, w.recipient); !errors.Is(err, share.ErrShareNotFound) {
		t.Fatalf("GetUserPermission after revoke = %v, want ErrShareNotFound", err)
	}
	if err := w.svc.UnshareEntity(w.ctx, created.ID); !errors.Is(err, share.ErrShareNotFound) {
		t.Fatalf("second UnshareEntity = %v, want ErrShareNotFound", err)
	}
}

// The repository carries no tenant_id predicate of its own; RLS is the only wall.
// Reading and revoking with another tenant's context must see nothing.
func TestRepository_ForeignTenantSeesAndRevokesNothing(t *testing.T) {
	w := newShareWorld(t)
	fileID := uuid.New()
	created := w.share(t, models.ShareEntityFile, fileID, w.recipient, models.SharePermissionRead)

	foreign := uuid.New()
	testutil.EnsureTenant(t, w.pool, foreign, "document share foreign")
	fctx := testutil.WithTenantCtx(context.Background(), foreign)

	if got, err := w.svc.ListByEntity(fctx, models.ShareEntityFile, fileID); err != nil || len(got) != 0 {
		t.Fatalf("foreign ListByEntity = %d/%v, want 0/nil", len(got), err)
	}
	if got, err := w.svc.ListSharedWithMe(fctx, w.recipient, models.ShareEntityFile); err != nil || len(got) != 0 {
		t.Fatalf("foreign ListSharedWithMe = %d/%v, want 0/nil", len(got), err)
	}
	if ok, _, err := w.svc.CheckAccess(fctx, models.ShareEntityFile, fileID, w.recipient); err != nil || ok {
		t.Fatalf("foreign CheckAccess = %v/%v, want no access", ok, err)
	}
	if _, err := w.repo.GetUserPermission(fctx, models.ShareEntityFile, fileID, w.recipient); !errors.Is(err, share.ErrShareNotFound) {
		t.Fatalf("foreign GetUserPermission = %v, want ErrShareNotFound", err)
	}
	if err := w.svc.UnshareEntity(fctx, created.ID); !errors.Is(err, share.ErrShareNotFound) {
		t.Fatalf("foreign UnshareEntity = %v, want ErrShareNotFound", err)
	}

	// The owner tenant's share survived the foreign delete attempt.
	if ok, _, err := w.svc.CheckAccess(w.ctx, models.ShareEntityFile, fileID, w.recipient); err != nil || !ok {
		t.Fatalf("own CheckAccess after foreign delete = %v/%v, want access intact", ok, err)
	}
}

func TestRepository_CancelledContextSurfacesErrors(t *testing.T) {
	w := newShareWorld(t)
	ctx, cancel := context.WithCancel(w.ctx)
	cancel()

	if err := w.repo.Delete(ctx, uuid.New()); err == nil || errors.Is(err, share.ErrShareNotFound) {
		t.Fatalf("Delete = %v, want a context error", err)
	}
	if _, err := w.repo.ListByEntity(ctx, models.ShareEntityFile, uuid.New()); err == nil {
		t.Fatal("ListByEntity with a cancelled context returned no error")
	}
	if _, err := w.repo.ListSharedWithUser(ctx, w.recipient, models.ShareEntityFile); err == nil {
		t.Fatal("ListSharedWithUser with a cancelled context returned no error")
	}
	if _, _, err := w.repo.HasAccess(ctx, models.ShareEntityFile, uuid.New(), w.recipient); err == nil {
		t.Fatal("HasAccess with a cancelled context returned no error")
	}
	if _, err := w.repo.GetUserPermission(ctx, models.ShareEntityFile, uuid.New(), w.recipient); err == nil || errors.Is(err, share.ErrShareNotFound) {
		t.Fatalf("GetUserPermission = %v, want a context error", err)
	}
}
