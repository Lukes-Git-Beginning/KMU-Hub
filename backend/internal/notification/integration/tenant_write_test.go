package integration

// Writes into the integration tables must carry tenant_id explicitly: the
// column is NOT NULL since migration 000115 and FORCE row level security checks
// it since 000122. Before this was wired, all four INSERTs omitted the column,
// so every write in this module failed at the constraint — including the live
// POST /api/v1/integrations/link route.
//
// Two levels of proof:
//   - TestIntegrationWrites_RefuseWithoutTenant runs without a database. The
//     repository holds a nil pool, so reaching pool.Exec would panic; passing
//     proves the refusal happens before any DB contact.
//   - TestIntegrationWrites_LandInCallerTenant needs a database and proves the
//     value that lands is the caller's tenant, by reading the row back from the
//     other tenant's context and finding nothing.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestIntegrationWrites_RefuseWithoutTenant(t *testing.T) {
	t.Parallel()

	// nil pool: any DB contact panics, so a clean ErrTenantMissing is proof the
	// guard runs first.
	repo := NewPostgresRepository(nil)
	ctx := context.Background()

	writes := []struct {
		name string
		call func() error
	}{
		{"CreateMapping", func() error {
			return repo.CreateMapping(ctx, &ChannelMapping{ID: uuid.New(), ConfigID: uuid.New(), Modules: []string{"crm"}})
		}},
		{"CreateAccountLink", func() error {
			return repo.CreateAccountLink(ctx, &AccountLink{ID: uuid.New(), Platform: PlatformSlack, ExternalUserID: "U1", KMUHubUserID: uuid.New()})
		}},
		{"CreateLinkToken", func() error {
			return repo.CreateLinkToken(ctx, &LinkToken{ID: uuid.New(), Platform: PlatformSlack, ExternalUserID: "U1", TokenHash: "hash"})
		}},
		{"LogDelivery", func() error {
			return repo.LogDelivery(ctx, &DeliveryLogEntry{ID: uuid.New(), NotificationID: uuid.New(), MappingID: uuid.New(), Platform: PlatformSlack, Status: DeliveryStatusSent})
		}},
	}

	for _, w := range writes {
		w := w
		t.Run(w.name, func(t *testing.T) {
			if err := w.call(); !errors.Is(err, ErrTenantMissing) {
				t.Fatalf("want ErrTenantMissing, got %v", err)
			}
		})
	}
}

func TestIntegrationWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("integ-write-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	configID := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             testutil.TenantA,
		"platform":              PlatformSlack,
		"credentials_vault_key": "slack/" + uuid.New().String(),
		"created_by":            userID,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", configID)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	now := time.Now()

	mapping := &ChannelMapping{
		ID:           uuid.New(),
		ConfigID:     configID,
		ChannelID:    "C" + uuid.New().String()[:8],
		ChannelName:  "general",
		Modules:      []string{"crm"},
		IsActive:     true,
		PlatformData: []byte("{}"),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.CreateMapping(ctxA, mapping); err != nil {
		t.Fatalf("CreateMapping: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "integration_channel_mappings", mapping.ID)

	link := &AccountLink{
		ID:             uuid.New(),
		Platform:       PlatformSlack,
		ExternalUserID: uuid.New().String()[:20],
		KMUHubUserID:   userID,
		LinkedAt:       now,
	}
	if err := repo.CreateAccountLink(ctxA, link); err != nil {
		t.Fatalf("CreateAccountLink: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "integration_account_links", link.ID)

	token := &LinkToken{
		ID:             uuid.New(),
		Platform:       PlatformSlack,
		ExternalUserID: uuid.New().String()[:20],
		TokenHash:      uuid.New().String(),
		ExpiresAt:      now.Add(5 * time.Minute),
		CreatedAt:      now,
	}
	if err := repo.CreateLinkToken(ctxA, token); err != nil {
		t.Fatalf("CreateLinkToken: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "integration_link_tokens", token.ID)

	delivery := &DeliveryLogEntry{
		ID:             uuid.New(),
		NotificationID: uuid.New(),
		MappingID:      mapping.ID,
		Platform:       PlatformSlack,
		Status:         DeliveryStatusSent,
		CreatedAt:      now,
	}
	if err := repo.LogDelivery(ctxA, delivery); err != nil {
		t.Fatalf("LogDelivery: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "integration_delivery_log", delivery.ID)

	rows := []struct {
		table string
		id    uuid.UUID
	}{
		{"integration_channel_mappings", mapping.ID},
		{"integration_account_links", link.ID},
		{"integration_link_tokens", token.ID},
		{"integration_delivery_log", delivery.ID},
	}

	for _, row := range rows {
		row := row
		t.Run(row.table, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, row.table, row.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, row.table, row.id, 0)
		})
	}
}
