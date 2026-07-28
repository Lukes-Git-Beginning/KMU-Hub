package vault

// wp-security: closes the write-surface gap for vault secrets. Update and
// Delete took a bare row ID with no tenant_id predicate (RLS was the only
// backstop); GetByKeyName and List ran without a tenant filter at all, so
// SetSecret/DeleteSecret/DeleteByKeyName resolved rows through an RLS-only
// lookup instead of an explicit tenant scope. All four now carry an explicit
// tenant_id predicate/filter, fixed in the same commit as this test.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestVaultSecretWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Vault Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Vault Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	// key_name carries a GLOBAL unique index (not per-tenant), so the foreign
	// probe below must use its own key name to fail on RLS, not on the
	// constraint.
	keyName := "wp-security-vault-" + uuid.New().String()[:8]
	now := time.Now().UTC()
	secret := &models.VaultSecret{
		ID:             uuid.New(),
		TenantID:       tenantOwn,
		KeyName:        keyName,
		EncryptedValue: "ciphertext",
		KeyVersion:     1,
		Description:    "wp-security test secret",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	foreignAttempt := *secret
	foreignAttempt.ID = uuid.New()
	foreignAttempt.KeyName = keyName + "-foreign-attempt"
	if err := repo.Create(ctxOther, &foreignAttempt); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "vault_secrets", foreignAttempt.ID, 0)

	if err := repo.Create(ctxOwn, secret); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "vault_secrets", secret.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "vault_secrets", secret.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "vault_secrets", secret.ID, 0)

	// GetByKeyName is explicitly tenant-scoped: a foreign tenant asking for
	// the caller tenant's key name must not resolve it.
	if _, err := repo.GetByKeyName(ctxOther, tenantOther, keyName); err == nil {
		t.Fatalf("GetByKeyName (foreign ctx): expected not-found, got nil")
	}
	got, err := repo.GetByKeyName(ctxOwn, tenantOwn, keyName)
	if err != nil {
		t.Fatalf("GetByKeyName (own ctx): %v", err)
	}
	if got.ID != secret.ID {
		t.Fatalf("GetByKeyName (own ctx): got wrong secret %s", got.ID)
	}

	// List only returns the caller tenant's secrets.
	listOther, err := repo.List(ctxOther, tenantOther)
	if err != nil {
		t.Fatalf("List (foreign ctx): %v", err)
	}
	for _, s := range listOther {
		if s.ID == secret.ID {
			t.Fatalf("List (foreign ctx): leaked foreign-tenant secret %s", s.ID)
		}
	}

	// Update carries an explicit tenant_id predicate — a foreign-tenant call
	// must affect zero rows, not mutate the victim's secret.
	foreignUpdate := *secret
	foreignUpdate.EncryptedValue = "hacked-ciphertext"
	foreignUpdate.TenantID = tenantOther
	foreignUpdate.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreignUpdate); err != nil {
		t.Fatalf("Update (foreign ctx): unexpected error: %v", err)
	}
	got, err = repo.GetByKeyName(ctxOwn, tenantOwn, keyName)
	if err != nil {
		t.Fatalf("GetByKeyName after foreign update: %v", err)
	}
	if got.EncryptedValue == "hacked-ciphertext" {
		t.Fatalf("a foreign-tenant write reached the vault secret")
	}

	ownUpdate := *secret
	ownUpdate.EncryptedValue = "rotated-ciphertext"
	ownUpdate.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOwn, &ownUpdate); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByKeyName(ctxOwn, tenantOwn, keyName)
	if err != nil {
		t.Fatalf("GetByKeyName after own update: %v", err)
	}
	if got.EncryptedValue != "rotated-ciphertext" {
		t.Fatalf("own-tenant write did not land: encrypted_value=%q", got.EncryptedValue)
	}

	// Delete carries an explicit tenantID predicate.
	if err := repo.Delete(ctxOther, tenantOther, secret.ID); err == nil {
		t.Fatalf("Delete (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "vault_secrets", secret.ID, 1)

	if err := repo.Delete(ctxOwn, tenantOwn, secret.ID); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "vault_secrets", secret.ID, 0)
}
