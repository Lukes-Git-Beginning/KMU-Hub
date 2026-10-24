package contact

// End-to-end through the real repository for what the desktop contact form
// does: save with every profile field empty (""), and later empty a field that
// was set. The gateway lets "" through (clearable_* rules); this proves the
// service and the CHECK constraints of migration 000314 accept it too.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestService_ProfileFields_EmptyStringCreatesAndClears_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CRM Profile Clear Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         "crm-profile-clear-" + uuid.New().String()[:8] + "@tenant.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	svc := NewService(NewPostgresRepository(pool))
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	str := func(s string) *string { return &s }

	empty := ProfileFields{
		Salutation: str(""), Title: str(""), Mobile: str(""), Department: str(""),
		AddressStreet: str(""), AddressZip: str(""), AddressCity: str(""), AddressCountry: str(""),
		Website: str(""), LinkedIn: str(""), Xing: str(""), Category: str(""), Status: str(""),
	}
	created, err := svc.Create(ctx, CreateInput{
		FirstName: "Anna", LastName: "Berg", CreatedBy: userID, TenantID: tenantID, ProfileFields: empty,
	})
	if err != nil {
		t.Fatalf("create with empty profile fields: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", created.ID)

	set := ProfileFields{Salutation: str("Frau"), Mobile: str("+49 170 1234567"), Category: str("partner")}
	if _, err := svc.Update(ctx, created.ID, UpdateInput{ProfileFields: set}, tenantID); err != nil {
		t.Fatalf("update with values: %v", err)
	}
	got, err := svc.GetByID(ctx, created.ID, tenantID)
	if err != nil || got.Salutation == nil || *got.Salutation != "Frau" {
		t.Fatalf("read back after set: contact %+v, err %v", got, err)
	}

	if _, err := svc.Update(ctx, created.ID, UpdateInput{ProfileFields: empty}, tenantID); err != nil {
		t.Fatalf("update with empty profile fields: %v", err)
	}
	got, err = svc.GetByID(ctx, created.ID, tenantID)
	if err != nil {
		t.Fatalf("read back after clear: %v", err)
	}
	if got.Salutation != nil || got.Mobile != nil || got.Category != nil {
		t.Errorf("\"\" must clear the columns, got salutation=%v mobile=%v category=%v",
			got.Salutation, got.Mobile, got.Category)
	}
}
