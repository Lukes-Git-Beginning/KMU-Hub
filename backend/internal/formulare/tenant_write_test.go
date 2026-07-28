package formulare

// Exercises the real write path (PostgresRepository, not testutil.SeedRow) so a
// forgotten tenant_id column would show up as a genuine INSERT/constraint failure
// instead of being silently bypassed by a system-context fixture. Complements
// tenant_isolation_phase2_test.go, which only proves RLS SELECT filtering on
// hand-seeded rows.
//
// Also covers the ListActiveWebhooksForSchema fix: that query used to filter
// only on form_schema_id (no tenant_id predicate), relying entirely on RLS.
// Since form_schema_id is a UUID a caller cannot forge, RLS alone made this
// safe, but it violated the project rule that every SELECT carries its own
// tenant_id predicate — same defense-in-depth class as the ListInventurCounts
// fix (wp-branchen-module, iteration 56).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestFormulareWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Formulare Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Formulare Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	schema := &FormSchema{
		ID:          uuid.New(),
		TenantID:    tenantWrite,
		Title:       "Kontaktformular",
		Description: "",
		Fields:      []byte(`[]`),
		Status:      FormSchemaStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.CreateSchema(ctxA, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	webhook := &FormWebhook{
		ID:           uuid.New(),
		FormSchemaID: schema.ID,
		TenantID:     tenantWrite,
		URL:          "https://example.com/hook",
		Events:       []string{"submission.created"},
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.CreateWebhook(ctxA, webhook); err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_webhooks", webhook.ID)

	submission := &FormSubmission{
		ID:           uuid.New(),
		FormSchemaID: &schema.ID,
		TenantID:     tenantWrite,
		Answers:      []byte(`{"name":"Test"}`),
		Status:       FormSubmissionStatusNew,
		SubmittedAt:  now,
	}
	if err := repo.CreateSubmission(ctxA, submission, nil); err != nil {
		t.Fatalf("CreateSubmission: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_submissions", submission.ID)

	rows := []struct {
		table string
		id    uuid.UUID
	}{
		{"form_schemas", schema.ID},
		{"form_webhooks", webhook.ID},
		{"form_submissions", submission.ID},
	}

	for _, row := range rows {
		t.Run(row.table, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, row.table, row.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, row.table, row.id, 0)
		})
	}

	t.Run("ListActiveWebhooksForSchema is tenant-scoped", func(t *testing.T) {
		own, err := repo.ListActiveWebhooksForSchema(ctxA, tenantWrite, schema.ID)
		if err != nil {
			t.Fatalf("ListActiveWebhooksForSchema (own tenant): %v", err)
		}
		if len(own) != 1 {
			t.Fatalf("expected 1 active webhook for own tenant, got %d", len(own))
		}

		// A foreign tenant passing the correct (known) form_schema_id must
		// still see nothing — the query must not rely on RLS alone.
		foreign, err := repo.ListActiveWebhooksForSchema(ctxB, tenantOther, schema.ID)
		if err != nil {
			t.Fatalf("ListActiveWebhooksForSchema (foreign tenant): %v", err)
		}
		if len(foreign) != 0 {
			t.Fatalf("expected 0 active webhooks for foreign tenant, got %d", len(foreign))
		}
	})
}
