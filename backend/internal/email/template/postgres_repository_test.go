package template_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/email/template"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"email":         "template-" + name + "-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Template", "last_name": name,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
	return userID
}

func seedTemplate(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, ownerID *uuid.UUID, visibility, name string) uuid.UUID {
	t.Helper()
	now := time.Now().UTC()
	id := testutil.SeedRow(t, pool, "email_templates", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "owner_id": ownerID, "visibility": visibility,
		"name": name, "subject": "Betreff " + name, "body_html": "<p>Hallo {{contact_first_name}}</p>",
		"body_text": "Hallo {{contact_first_name}}", "created_at": now, "updated_at": now,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_templates", id) })
	return id
}

// TestRepository_VisibilityAndTenantIsolation proves the WHERE clause shared
// by GetByID and ListVisible enforces both boundaries at once: a foreign
// tenant sees nothing regardless of visibility, and within the own tenant a
// personal template is invisible to everyone except its owner and admins.
func TestRepository_VisibilityAndTenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	// t.Cleanup, not defer: registered before the row-cleanup helpers below,
	// so LIFO ordering runs it last -- their cleanup queries still need a
	// live pool. A defer here would close the pool before those callbacks run.
	t.Cleanup(func() { pool.Close() })

	repo := template.NewPostgresRepository(pool)

	tenantA := uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "template test tenant A")
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "template test tenant B")

	ownerA1 := seedUser(t, pool, tenantA, "owner-a1")
	otherA2 := seedUser(t, pool, tenantA, "other-a2")
	ownerB1 := seedUser(t, pool, tenantB, "owner-b1")

	personalA := seedTemplate(t, pool, tenantA, &ownerA1, template.VisibilityPersonal, "Personal A")
	sharedA := seedTemplate(t, pool, tenantA, nil, template.VisibilityShared, "Shared A")
	seedTemplate(t, pool, tenantB, &ownerB1, template.VisibilityPersonal, "Personal B")

	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	// Owner sees their own personal template plus the shared one.
	ownList, err := repo.ListVisible(ctxA, tenantA, ownerA1, false)
	if err != nil {
		t.Fatalf("ListVisible(owner): %v", err)
	}
	if len(ownList) != 2 {
		t.Fatalf("owner sees %d templates, want 2 (own personal + shared)", len(ownList))
	}

	// A different tenant member sees only the shared one.
	peerList, err := repo.ListVisible(ctxA, tenantA, otherA2, false)
	if err != nil {
		t.Fatalf("ListVisible(peer): %v", err)
	}
	if len(peerList) != 1 || peerList[0].ID != sharedA {
		t.Fatalf("peer list = %+v, want just the shared template", peerList)
	}

	// An admin sees every template in the tenant, including foreign personal ones.
	adminList, err := repo.ListVisible(ctxA, tenantA, otherA2, true)
	if err != nil {
		t.Fatalf("ListVisible(admin): %v", err)
	}
	if len(adminList) != 2 {
		t.Fatalf("admin sees %d templates, want 2", len(adminList))
	}

	// Foreign tenant sees nothing from tenant A, regardless of visibility.
	crossTenantList, err := repo.ListVisible(ctxB, tenantB, ownerA1, true)
	if err != nil {
		t.Fatalf("ListVisible(cross-tenant): %v", err)
	}
	if len(crossTenantList) != 1 {
		t.Fatalf("tenant B admin sees %d templates, want 1 (only their own)", len(crossTenantList))
	}

	// GetByID: a peer cannot fetch a foreign personal template by id.
	if _, err := repo.GetByID(ctxA, personalA, tenantA, otherA2, false); !errors.Is(err, template.ErrTemplateNotFound) {
		t.Fatalf("peer GetByID personal: err = %v, want ErrTemplateNotFound", err)
	}
	// The owner can.
	if _, err := repo.GetByID(ctxA, personalA, tenantA, ownerA1, false); err != nil {
		t.Fatalf("owner GetByID personal: %v", err)
	}
	// Cross-tenant GetByID never resolves, even for the shared template.
	if _, err := repo.GetByID(ctxB, sharedA, tenantB, ownerB1, true); !errors.Is(err, template.ErrTemplateNotFound) {
		t.Fatalf("cross-tenant GetByID shared: err = %v, want ErrTemplateNotFound", err)
	}
}

func TestRepository_CreateUpdateDelete(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	// t.Cleanup, not defer: registered before the row-cleanup helpers below,
	// so LIFO ordering runs it last -- their cleanup queries still need a
	// live pool. A defer here would close the pool before those callbacks run.
	t.Cleanup(func() { pool.Close() })

	repo := template.NewPostgresRepository(pool)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "template crud tenant")
	ownerID := seedUser(t, pool, tenantID, "crud-owner")
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	id := uuid.New()
	tpl := &models.EmailTemplate{
		ID: id, TenantID: tenantID, OwnerID: &ownerID, Visibility: template.VisibilityPersonal,
		Name: "Angebot", Subject: "Ihr Angebot", BodyHTML: "<p>Hallo</p>", BodyText: "Hallo",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctx, tpl); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_templates", id) })

	fetched, err := repo.GetByID(ctx, id, tenantID, ownerID, false)
	if err != nil {
		t.Fatalf("GetByID after Create: %v", err)
	}
	if fetched.Name != "Angebot" {
		t.Fatalf("fetched name = %q, want Angebot", fetched.Name)
	}

	fetched.Name = "Angebot v2"
	fetched.Visibility = template.VisibilityShared
	fetched.OwnerID = nil
	if err := repo.Update(ctx, fetched); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Now visible to anyone in the tenant, without an owner.
	otherUser := seedUser(t, pool, tenantID, "crud-other")
	afterUpdate, err := repo.GetByID(ctx, id, tenantID, otherUser, false)
	if err != nil {
		t.Fatalf("GetByID after Update: %v", err)
	}
	if afterUpdate.Name != "Angebot v2" || afterUpdate.Visibility != template.VisibilityShared || afterUpdate.OwnerID != nil {
		t.Fatalf("after update = %+v, want name=Angebot v2 visibility=shared owner=nil", afterUpdate)
	}

	if err := repo.Delete(ctx, id, tenantID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, id, tenantID, otherUser, true); !errors.Is(err, template.ErrTemplateNotFound) {
		t.Fatalf("GetByID after Delete: err = %v, want ErrTemplateNotFound", err)
	}
}
