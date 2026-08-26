package formulare

// postgres_repository_db_test.go exercises PostgresRepository against the
// real schema instead of the mock repo behind service_test.go. Filed for
// cov-formulare-postgres-repository-real-sql (BACKLOG.yml): before this file
// most of postgres_repository.go's read/update/delete methods (Update/
// SoftDelete/List Schemas, Get/List/UpdateStatus Submissions, Get/Update/
// Delete/List Webhooks, Claim/Mark/List/Get Deliveries, GetFormStats,
// RevokeShareLink) ran only through the in-memory mock, so a broken WHERE
// clause, a wrong placeholder index, or an RLS gap would never have shown up.
//
// RLS boundary coverage for all four tables already lives in
// tenant_isolation_phase2_test.go (raw SQL) and tenant_write_test.go (through
// the repository's write path); this file adds the cross-tenant NotFound
// checks on the read methods themselves plus the one deliberate RLS bypass
// (ClaimPendingDeliveries under a system context, mirroring worker.go's
// database.WithSystemContext wrap).

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresUpdateSchema_AppliesChangesAndCrossTenantIsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Formulare UpdateSchema A")
	testutil.EnsureTenant(t, pool, tenantB, "Formulare UpdateSchema B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	schema := &FormSchema{
		ID: uuid.New(), TenantID: tenantA, Title: "Original",
		Fields: []byte(`[]`), Status: FormSchemaStatusDraft,
		PageCount: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSchema(ctxA, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	schema.Title = "Updated"
	schema.Status = FormSchemaStatusActive
	schema.PageCount = 3
	schema.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateSchema(ctxA, schema); err != nil {
		t.Fatalf("UpdateSchema: %v", err)
	}

	got, err := repo.GetSchema(ctxA, schema.ID, tenantA)
	if err != nil {
		t.Fatalf("GetSchema: %v", err)
	}
	if got.Title != "Updated" || got.Status != FormSchemaStatusActive || got.PageCount != 3 {
		t.Fatalf("update did not persist: %+v", got)
	}

	// A tenant-B row with tenant A's id must not update A's schema.
	foreign := &FormSchema{
		ID: schema.ID, TenantID: tenantB, Title: "Hijacked",
		Fields: []byte(`[]`), Status: FormSchemaStatusDraft, UpdatedAt: time.Now().UTC(),
	}
	if err := repo.UpdateSchema(ctxB, foreign); err == nil {
		t.Fatal("expected UpdateSchema to fail across tenants, got nil error")
	} else if err != ErrSchemaNotFound {
		t.Fatalf("expected ErrSchemaNotFound for cross-tenant update, got %v", err)
	}

	// Untouched: still tenant A's title.
	still, err := repo.GetSchema(ctxA, schema.ID, tenantA)
	if err != nil {
		t.Fatalf("GetSchema after cross-tenant attempt: %v", err)
	}
	if still.Title != "Updated" {
		t.Fatalf("cross-tenant update leaked through: title = %q", still.Title)
	}
}

func TestPostgresSoftDeleteSchema_HidesFromGetAndCrossTenantIsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Formulare SoftDelete A")
	testutil.EnsureTenant(t, pool, tenantB, "Formulare SoftDelete B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	schema := &FormSchema{
		ID: uuid.New(), TenantID: tenantA, Title: "To Delete",
		Fields: []byte(`[]`), Status: FormSchemaStatusDraft, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSchema(ctxA, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	if err := repo.SoftDeleteSchema(ctxB, schema.ID, tenantB); err != ErrSchemaNotFound {
		t.Fatalf("expected ErrSchemaNotFound deleting another tenant's schema, got %v", err)
	}
	if _, err := repo.GetSchema(ctxA, schema.ID, tenantA); err != nil {
		t.Fatalf("schema must survive a foreign-tenant delete attempt, got %v", err)
	}

	if err := repo.SoftDeleteSchema(ctxA, schema.ID, tenantA); err != nil {
		t.Fatalf("SoftDeleteSchema: %v", err)
	}
	if _, err := repo.GetSchema(ctxA, schema.ID, tenantA); err != ErrSchemaNotFound {
		t.Fatalf("expected ErrSchemaNotFound after soft delete, got %v", err)
	}
	if err := repo.SoftDeleteSchema(ctxA, schema.ID, tenantA); err != ErrSchemaNotFound {
		t.Fatalf("expected ErrSchemaNotFound deleting an already-deleted schema, got %v", err)
	}
}

func TestPostgresListSchemas_FiltersByStatusTemplateAndSearch(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// t.Cleanup (not defer) so mk()'s row cleanups, registered later, run
	// before this closes the pool — t.Cleanup is LIFO, defer would fire
	// immediately when this function returns, before those cleanups exist.
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare ListSchemas")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	mk := func(title string, status FormSchemaStatus, template bool) *FormSchema {
		s := &FormSchema{
			ID: uuid.New(), TenantID: tenantID, Title: title,
			Fields: []byte(`[]`), Status: status, IsTemplate: template,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateSchema(ctx, s); err != nil {
			t.Fatalf("CreateSchema(%s): %v", title, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "form_schemas", s.ID) })
		return s
	}

	active := mk("Kontaktformular Support", FormSchemaStatusActive, false)
	mk("Kontaktformular Vertrieb", FormSchemaStatusDraft, false)
	template := mk("Vorlage Feedback", FormSchemaStatusActive, true)

	activeStatus := FormSchemaStatusActive
	isTemplateTrue := true

	list, total, err := repo.ListSchemas(ctx, tenantID, ListSchemasFilter{Status: &activeStatus, Limit: 20})
	if err != nil {
		t.Fatalf("ListSchemas(status=active): %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 active schemas, got total=%d", total)
	}
	ids := map[uuid.UUID]bool{}
	for _, s := range list {
		ids[s.ID] = true
	}
	if !ids[active.ID] || !ids[template.ID] {
		t.Fatalf("expected both active schemas in result, got %+v", list)
	}

	list, total, err = repo.ListSchemas(ctx, tenantID, ListSchemasFilter{IsTemplate: &isTemplateTrue, Limit: 20})
	if err != nil {
		t.Fatalf("ListSchemas(is_template=true): %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != template.ID {
		t.Fatalf("expected only the template schema, got total=%d list=%+v", total, list)
	}

	list, total, err = repo.ListSchemas(ctx, tenantID, ListSchemasFilter{Search: "support", Limit: 20})
	if err != nil {
		t.Fatalf("ListSchemas(search=support): %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != active.ID {
		t.Fatalf("expected only the support schema, got total=%d list=%+v", total, list)
	}

	// Combined: active + not-template + search must intersect correctly.
	notTemplate := false
	list, total, err = repo.ListSchemas(ctx, tenantID, ListSchemasFilter{
		Status: &activeStatus, IsTemplate: &notTemplate, Search: "kontaktformular", Limit: 20,
	})
	if err != nil {
		t.Fatalf("ListSchemas(combined filters): %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != active.ID {
		t.Fatalf("expected exactly the active non-template contact schema, got total=%d list=%+v", total, list)
	}
}

func TestPostgresDuplicateSchema_CopiesFieldsResetsStatusAndPublic(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare DuplicateSchema")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	original := &FormSchema{
		ID: uuid.New(), TenantID: tenantID, Title: "Original", Description: "desc",
		Fields: []byte(`[{"id":"f1","type":"text","label":"L"}]`),
		Status: FormSchemaStatusActive, IsPublic: true, PageCount: 2,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSchema(ctx, original); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", original.ID)

	dup, err := repo.DuplicateSchema(ctx, original.ID, tenantID, "")
	if err != nil {
		t.Fatalf("DuplicateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", dup.ID)

	if dup.ID == original.ID {
		t.Fatal("duplicate must get a new ID")
	}
	if dup.Title != "Original (Copy)" {
		t.Fatalf("expected default copy title, got %q", dup.Title)
	}
	if dup.Status != FormSchemaStatusDraft {
		t.Fatalf("duplicate must reset to draft, got %q", dup.Status)
	}
	if dup.IsPublic {
		t.Fatal("duplicate must not inherit is_public — a copied form is not automatically exposed")
	}
	// JSONB round-trips through Postgres reformat whitespace, so compare
	// decoded content rather than raw bytes.
	var gotFields, wantFields []FormField
	if err := json.Unmarshal(dup.Fields, &gotFields); err != nil {
		t.Fatalf("unmarshal dup.Fields: %v", err)
	}
	if err := json.Unmarshal(original.Fields, &wantFields); err != nil {
		t.Fatalf("unmarshal original.Fields: %v", err)
	}
	if !reflect.DeepEqual(gotFields, wantFields) {
		t.Fatalf("fields not copied: got %+v want %+v", gotFields, wantFields)
	}

	// Cross-tenant duplicate must fail before writing anything.
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "Formulare DuplicateSchema Foreign")
	ctxOther := testutil.WithTenantCtx(context.Background(), otherTenant)
	if _, err := repo.DuplicateSchema(ctxOther, original.ID, otherTenant, ""); err != ErrSchemaNotFound {
		t.Fatalf("expected ErrSchemaNotFound for cross-tenant duplicate, got %v", err)
	}
}

func TestPostgresGetSubmission_CrossTenantReturnsErrSubmissionNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Formulare GetSubmission A")
	testutil.EnsureTenant(t, pool, tenantB, "Formulare GetSubmission B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	sub := &FormSubmission{
		ID: uuid.New(), TenantID: tenantA, Answers: []byte(`{"a":1}`),
		Status: FormSubmissionStatusNew, SubmittedAt: now,
	}
	if err := repo.CreateSubmission(ctxA, sub, nil); err != nil {
		t.Fatalf("CreateSubmission: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_submissions", sub.ID)

	got, err := repo.GetSubmission(ctxA, sub.ID, tenantA)
	if err != nil {
		t.Fatalf("GetSubmission (own tenant): %v", err)
	}
	if got.ID != sub.ID {
		t.Fatalf("expected submission %s, got %s", sub.ID, got.ID)
	}

	if _, err := repo.GetSubmission(ctxB, sub.ID, tenantB); err != ErrSubmissionNotFound {
		t.Fatalf("expected ErrSubmissionNotFound for cross-tenant read, got %v", err)
	}
}

func TestPostgresListSubmissions_FiltersBySchemaAndStatus(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare ListSubmissions")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	schemaA := &FormSchema{ID: uuid.New(), TenantID: tenantID, Title: "Schema A", Fields: []byte(`[]`), Status: FormSchemaStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSchema(ctx, schemaA); err != nil {
		t.Fatalf("CreateSchema A: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schemaA.ID)
	schemaB := &FormSchema{ID: uuid.New(), TenantID: tenantID, Title: "Schema B", Fields: []byte(`[]`), Status: FormSchemaStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSchema(ctx, schemaB); err != nil {
		t.Fatalf("CreateSchema B: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schemaB.ID)

	mkSub := func(schemaID uuid.UUID, status FormSubmissionStatus) *FormSubmission {
		s := &FormSubmission{ID: uuid.New(), FormSchemaID: &schemaID, TenantID: tenantID, Answers: []byte(`{}`), Status: status, SubmittedAt: now}
		if err := repo.CreateSubmission(ctx, s, nil); err != nil {
			t.Fatalf("CreateSubmission: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "form_submissions", s.ID) })
		return s
	}

	subANew := mkSub(schemaA.ID, FormSubmissionStatusNew)
	mkSub(schemaA.ID, FormSubmissionStatusRead)
	mkSub(schemaB.ID, FormSubmissionStatusNew)

	list, total, err := repo.ListSubmissions(ctx, tenantID, ListSubmissionsFilter{FormSchemaID: &schemaA.ID, Limit: 20})
	if err != nil {
		t.Fatalf("ListSubmissions(schema=A): %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 submissions for schema A, got %d", total)
	}
	_ = list

	newStatus := FormSubmissionStatusNew
	list, total, err = repo.ListSubmissions(ctx, tenantID, ListSubmissionsFilter{FormSchemaID: &schemaA.ID, Status: &newStatus, Limit: 20})
	if err != nil {
		t.Fatalf("ListSubmissions(schema=A, status=new): %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != subANew.ID {
		t.Fatalf("expected only subANew, got total=%d list=%+v", total, list)
	}
}

func TestPostgresUpdateSubmissionStatus_ChangesStatusAndCrossTenantIsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Formulare UpdateSubmissionStatus A")
	testutil.EnsureTenant(t, pool, tenantB, "Formulare UpdateSubmissionStatus B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	sub := &FormSubmission{ID: uuid.New(), TenantID: tenantA, Answers: []byte(`{}`), Status: FormSubmissionStatusNew, SubmittedAt: now}
	if err := repo.CreateSubmission(ctxA, sub, nil); err != nil {
		t.Fatalf("CreateSubmission: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_submissions", sub.ID)

	if err := repo.UpdateSubmissionStatus(ctxB, sub.ID, tenantB, FormSubmissionStatusArchived); err != ErrSubmissionNotFound {
		t.Fatalf("expected ErrSubmissionNotFound updating another tenant's submission, got %v", err)
	}

	if err := repo.UpdateSubmissionStatus(ctxA, sub.ID, tenantA, FormSubmissionStatusRead); err != nil {
		t.Fatalf("UpdateSubmissionStatus: %v", err)
	}
	got, err := repo.GetSubmission(ctxA, sub.ID, tenantA)
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if got.Status != FormSubmissionStatusRead {
		t.Fatalf("expected status 'read', got %q", got.Status)
	}
}

func TestPostgresListSubmissionsForExport_FiltersByStatusAndDateRange(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare ExportSubmissions")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	schema := &FormSchema{ID: uuid.New(), TenantID: tenantID, Title: "Export Schema", Fields: []byte(`[]`), Status: FormSchemaStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSchema(ctx, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	mkSubAt := func(status FormSubmissionStatus, at time.Time) *FormSubmission {
		s := &FormSubmission{ID: uuid.New(), FormSchemaID: &schema.ID, TenantID: tenantID, Answers: []byte(`{}`), Status: status, SubmittedAt: at}
		if err := repo.CreateSubmission(ctx, s, nil); err != nil {
			t.Fatalf("CreateSubmission: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "form_submissions", s.ID) })
		return s
	}

	old := mkSubAt(FormSubmissionStatusNew, now.Add(-72*time.Hour))
	inWindow := mkSubAt(FormSubmissionStatusNew, now.Add(-24*time.Hour))
	mkSubAt(FormSubmissionStatusArchived, now.Add(-24*time.Hour)) // wrong status, excluded
	future := mkSubAt(FormSubmissionStatusNew, now.Add(24*time.Hour))
	_ = old
	_ = future

	newStatus := FormSubmissionStatusNew
	after := now.Add(-48 * time.Hour)
	before := now.Add(1 * time.Hour)
	list, err := repo.ListSubmissionsForExport(ctx, schema.ID, tenantID, ExportFilter{
		Status: &newStatus, SubmittedAfter: &after, SubmittedBefore: &before,
	})
	if err != nil {
		t.Fatalf("ListSubmissionsForExport: %v", err)
	}
	if len(list) != 1 || list[0].ID != inWindow.ID {
		t.Fatalf("expected exactly the in-window new submission, got %+v", list)
	}
}

func TestPostgresGetWebhook_CrossTenantReturnsErrWebhookNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Formulare GetWebhook A")
	testutil.EnsureTenant(t, pool, tenantB, "Formulare GetWebhook B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	schema := &FormSchema{ID: uuid.New(), TenantID: tenantA, Title: "Webhook Schema", Fields: []byte(`[]`), Status: FormSchemaStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSchema(ctxA, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	secret := "s3cr3t"
	webhook := &FormWebhook{
		ID: uuid.New(), FormSchemaID: schema.ID, TenantID: tenantA,
		URL: "https://example.com/hook", Secret: &secret,
		Events: []string{"submission.created"}, Active: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateWebhook(ctxA, webhook); err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_webhooks", webhook.ID)

	got, err := repo.GetWebhook(ctxA, webhook.ID, tenantA)
	if err != nil {
		t.Fatalf("GetWebhook (own tenant): %v", err)
	}
	if got.Secret == nil || *got.Secret != secret {
		t.Fatalf("repository read must return the raw secret (masking is a service concern), got %+v", got.Secret)
	}

	if _, err := repo.GetWebhook(ctxB, webhook.ID, tenantB); err != ErrWebhookNotFound {
		t.Fatalf("expected ErrWebhookNotFound for cross-tenant read, got %v", err)
	}
}

func TestPostgresUpdateWebhook_AppliesChanges(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare UpdateWebhook")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	schema := &FormSchema{ID: uuid.New(), TenantID: tenantID, Title: "Update Webhook Schema", Fields: []byte(`[]`), Status: FormSchemaStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSchema(ctx, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	webhook := &FormWebhook{
		ID: uuid.New(), FormSchemaID: schema.ID, TenantID: tenantID,
		URL: "https://example.com/old", Events: []string{"submission.created"},
		Active: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateWebhook(ctx, webhook); err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_webhooks", webhook.ID)

	webhook.URL = "https://example.com/new"
	webhook.Active = false
	webhook.Events = []string{"submission.created", "submission.updated"}
	webhook.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateWebhook(ctx, webhook); err != nil {
		t.Fatalf("UpdateWebhook: %v", err)
	}

	got, err := repo.GetWebhook(ctx, webhook.ID, tenantID)
	if err != nil {
		t.Fatalf("GetWebhook: %v", err)
	}
	if got.URL != "https://example.com/new" || got.Active || len(got.Events) != 2 {
		t.Fatalf("update did not persist: %+v", got)
	}

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "Formulare UpdateWebhook Foreign")
	ctxOther := testutil.WithTenantCtx(context.Background(), otherTenant)
	webhook.TenantID = otherTenant
	if err := repo.UpdateWebhook(ctxOther, webhook); err != ErrWebhookNotFound {
		t.Fatalf("expected ErrWebhookNotFound for cross-tenant update, got %v", err)
	}
}

func TestPostgresDeleteWebhook_RemovesRowAndCrossTenantIsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Formulare DeleteWebhook A")
	testutil.EnsureTenant(t, pool, tenantB, "Formulare DeleteWebhook B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	schema := &FormSchema{ID: uuid.New(), TenantID: tenantA, Title: "Delete Webhook Schema", Fields: []byte(`[]`), Status: FormSchemaStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSchema(ctxA, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	webhook := &FormWebhook{ID: uuid.New(), FormSchemaID: schema.ID, TenantID: tenantA, URL: "https://example.com/hook", Events: []string{"submission.created"}, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateWebhook(ctxA, webhook); err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	if err := repo.DeleteWebhook(ctxB, webhook.ID, tenantB); err != ErrWebhookNotFound {
		t.Fatalf("expected ErrWebhookNotFound deleting another tenant's webhook, got %v", err)
	}
	if err := repo.DeleteWebhook(ctxA, webhook.ID, tenantA); err != nil {
		t.Fatalf("DeleteWebhook: %v", err)
	}
	if _, err := repo.GetWebhook(ctxA, webhook.ID, tenantA); err != ErrWebhookNotFound {
		t.Fatalf("expected ErrWebhookNotFound after delete, got %v", err)
	}
}

func TestPostgresListWebhooks_ReturnsAllForSchemaOrderedByCreation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare ListWebhooks")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	schema := &FormSchema{ID: uuid.New(), TenantID: tenantID, Title: "List Webhooks Schema", Fields: []byte(`[]`), Status: FormSchemaStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSchema(ctx, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	first := &FormWebhook{ID: uuid.New(), FormSchemaID: schema.ID, TenantID: tenantID, URL: "https://example.com/1", Events: []string{"submission.created"}, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateWebhook(ctx, first); err != nil {
		t.Fatalf("CreateWebhook first: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_webhooks", first.ID)
	second := &FormWebhook{ID: uuid.New(), FormSchemaID: schema.ID, TenantID: tenantID, URL: "https://example.com/2", Events: []string{"submission.created"}, Active: false, CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)}
	if err := repo.CreateWebhook(ctx, second); err != nil {
		t.Fatalf("CreateWebhook second: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_webhooks", second.ID)

	list, err := repo.ListWebhooks(ctx, schema.ID, tenantID)
	if err != nil {
		t.Fatalf("ListWebhooks: %v", err)
	}
	if len(list) != 2 || list[0].ID != first.ID || list[1].ID != second.ID {
		t.Fatalf("expected [first, second] ordered by created_at, got %+v", list)
	}
}

// mkFullChain creates schema + active webhook + submission (with the webhook
// enqueued as a pending delivery) for tenantID, returning the delivery row.
func mkFullChain(t *testing.T, repo *PostgresRepository, ctx context.Context, tenantID uuid.UUID) *WebhookDelivery {
	t.Helper()
	now := time.Now().UTC()

	schema := &FormSchema{ID: uuid.New(), TenantID: tenantID, Title: "Delivery Chain Schema " + uuid.New().String(), Fields: []byte(`[]`), Status: FormSchemaStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSchema(ctx, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, repo.pool, "form_schemas", schema.ID) })

	webhook := &FormWebhook{ID: uuid.New(), FormSchemaID: schema.ID, TenantID: tenantID, URL: "https://example.com/chain", Events: []string{"submission.created"}, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateWebhook(ctx, webhook); err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, repo.pool, "form_webhooks", webhook.ID) })

	sub := &FormSubmission{ID: uuid.New(), FormSchemaID: &schema.ID, TenantID: tenantID, Answers: []byte(`{"f":"v"}`), Status: FormSubmissionStatusNew, SubmittedAt: now}
	if err := repo.CreateSubmission(ctx, sub, []uuid.UUID{webhook.ID}); err != nil {
		t.Fatalf("CreateSubmission: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, repo.pool, "form_submissions", sub.ID) })

	deliveries, _, err := repo.ListWebhookDeliveries(ctx, tenantID, ListDeliveriesFilter{SubmissionID: &sub.ID, Limit: 10})
	if err != nil {
		t.Fatalf("ListWebhookDeliveries (seed lookup): %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("expected exactly 1 enqueued delivery, got %d", len(deliveries))
	}
	return deliveries[0]
}

func TestPostgresClaimPendingDeliveries_SystemContextCrossesTenantsTenantContextDoesNot(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Formulare Claim A")
	testutil.EnsureTenant(t, pool, tenantB, "Formulare Claim B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	delA := mkFullChain(t, repo, ctxA, tenantA)
	delB := mkFullChain(t, repo, ctxB, tenantB)

	// A plain tenant-scoped context is what a handler would use, and RLS
	// must confine it to that tenant's own pending rows only.
	claimedA, err := repo.ClaimPendingDeliveries(ctxA, 10)
	if err != nil {
		t.Fatalf("ClaimPendingDeliveries (tenant A ctx): %v", err)
	}
	for _, d := range claimedA {
		if d.ID == delB.ID {
			t.Fatalf("tenant A context claimed tenant B's delivery %s — RLS gap", delB.ID)
		}
	}
	foundA := false
	for _, d := range claimedA {
		if d.ID == delA.ID {
			foundA = true
		}
	}
	if !foundA {
		t.Fatalf("expected tenant A's own pending delivery to be claimable under its own context")
	}
	// ClaimPendingDeliveries only SELECTs FOR UPDATE SKIP LOCKED, it never
	// writes a status — and pool.Query with no explicit BEGIN commits (and
	// releases the row lock) as soon as the statement finishes. So delA is
	// still 'pending' here; no reset needed before claiming again below.

	// The worker runs under database.WithSystemContext (worker.go Run) —
	// that is the one intentional cross-tenant bypass on this table, so it
	// must see both tenants' pending rows in one claim.
	sysCtx := database.WithSystemContext(context.Background())
	claimedSys, err := repo.ClaimPendingDeliveries(sysCtx, 10)
	if err != nil {
		t.Fatalf("ClaimPendingDeliveries (system ctx): %v", err)
	}
	seen := map[uuid.UUID]bool{}
	for _, d := range claimedSys {
		seen[d.ID] = true
	}
	if !seen[delA.ID] || !seen[delB.ID] {
		t.Fatalf("expected system context to see both tenants' pending deliveries, got %+v", claimedSys)
	}
}

func TestPostgresMarkDeliveryResult_UpdatesRowAndReportsMissing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare MarkDeliveryResult")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	delivery := mkFullChain(t, repo, ctx, tenantID)

	now := time.Now().UTC()
	if err := repo.MarkDeliveryResult(ctx, delivery.ID, WebhookDeliveryStatusDelivered, 1, &now, "", 200); err != nil {
		t.Fatalf("MarkDeliveryResult: %v", err)
	}

	got, err := repo.GetWebhookDelivery(ctx, delivery.ID, tenantID)
	if err != nil {
		t.Fatalf("GetWebhookDelivery: %v", err)
	}
	if got.Status != WebhookDeliveryStatusDelivered || got.AttemptCount != 1 || got.DeliveredAt == nil {
		t.Fatalf("mark result did not persist: %+v", got)
	}
	if got.LastResponseCode == nil || *got.LastResponseCode != 200 {
		t.Fatalf("expected last_response_code 200, got %+v", got.LastResponseCode)
	}

	if err := repo.MarkDeliveryResult(ctx, uuid.New(), WebhookDeliveryStatusFailed, 1, nil, "boom", 500); err != ErrDeliveryNotFound {
		t.Fatalf("expected ErrDeliveryNotFound for an unknown delivery id, got %v", err)
	}
}

func TestPostgresListWebhookDeliveries_FiltersByWebhookAndStatus(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare ListDeliveries")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	delivery := mkFullChain(t, repo, ctx, tenantID)

	list, total, err := repo.ListWebhookDeliveries(ctx, tenantID, ListDeliveriesFilter{WebhookID: &delivery.WebhookID, Limit: 20})
	if err != nil {
		t.Fatalf("ListWebhookDeliveries(webhook): %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != delivery.ID {
		t.Fatalf("expected exactly the seeded delivery, got total=%d list=%+v", total, list)
	}

	deadStatus := WebhookDeliveryStatusDead
	list, total, err = repo.ListWebhookDeliveries(ctx, tenantID, ListDeliveriesFilter{WebhookID: &delivery.WebhookID, Status: &deadStatus, Limit: 20})
	if err != nil {
		t.Fatalf("ListWebhookDeliveries(status=dead): %v", err)
	}
	if total != 0 || len(list) != 0 {
		t.Fatalf("expected 0 dead deliveries, got total=%d list=%+v", total, list)
	}
}

func TestPostgresGetWebhookDelivery_CrossTenantReturnsErrDeliveryNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Formulare GetDelivery A")
	testutil.EnsureTenant(t, pool, tenantB, "Formulare GetDelivery B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	delivery := mkFullChain(t, repo, ctxA, tenantA)

	if _, err := repo.GetWebhookDelivery(ctxB, delivery.ID, tenantB); err != ErrDeliveryNotFound {
		t.Fatalf("expected ErrDeliveryNotFound for cross-tenant read, got %v", err)
	}
}

func TestPostgresGetFormStats_CountsByStatusAndTimeWindow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare GetFormStats")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	schema := &FormSchema{ID: uuid.New(), TenantID: tenantID, Title: "Stats Schema", Fields: []byte(`[]`), Status: FormSchemaStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSchema(ctx, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	mkAt := func(status FormSubmissionStatus, at time.Time) {
		s := &FormSubmission{ID: uuid.New(), FormSchemaID: &schema.ID, TenantID: tenantID, Answers: []byte(`{}`), Status: status, SubmittedAt: at}
		if err := repo.CreateSubmission(ctx, s, nil); err != nil {
			t.Fatalf("CreateSubmission: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "form_submissions", s.ID) })
	}

	mkAt(FormSubmissionStatusNew, now.Add(-1*time.Hour))      // within 7d and 30d, new
	mkAt(FormSubmissionStatusRead, now.Add(-10*24*time.Hour)) // within 30d only, not new
	mkAt(FormSubmissionStatusNew, now.Add(-40*24*time.Hour))  // outside both windows

	stats, err := repo.GetFormStats(ctx, schema.ID, tenantID)
	if err != nil {
		t.Fatalf("GetFormStats: %v", err)
	}
	if stats.TotalCount != 3 {
		t.Fatalf("expected total=3, got %d", stats.TotalCount)
	}
	if stats.NewCount != 2 {
		t.Fatalf("expected new_count=2, got %d", stats.NewCount)
	}
	if stats.Last7dCount != 1 {
		t.Fatalf("expected last_7d=1, got %d", stats.Last7dCount)
	}
	if stats.Last30dCount != 2 {
		t.Fatalf("expected last_30d=2, got %d", stats.Last30dCount)
	}
}

func TestPostgresRevokeShareLink_CrossTenantReturnsErrShareLinkNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Formulare RevokeShareLink A")
	testutil.EnsureTenant(t, pool, tenantB, "Formulare RevokeShareLink B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	schema := &FormSchema{ID: uuid.New(), TenantID: tenantA, Title: "Revoke Schema", Fields: []byte(`[]`), Status: FormSchemaStatusActive, IsPublic: true, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSchema(ctxA, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	link := &FormShareLink{ID: uuid.New(), TenantID: tenantA, FormSchemaID: schema.ID, Token: uuid.New().String(), CreatedAt: now}
	if err := repo.CreateShareLink(ctxA, link); err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_share_tokens", link.ID)

	if err := repo.RevokeShareLink(ctxB, link.ID, tenantB, now); err != ErrShareLinkNotFound {
		t.Fatalf("expected ErrShareLinkNotFound revoking another tenant's link, got %v", err)
	}

	if err := repo.RevokeShareLink(ctxA, link.ID, tenantA, now); err != nil {
		t.Fatalf("RevokeShareLink: %v", err)
	}
	links, err := repo.ListShareLinks(ctxA, schema.ID, tenantA)
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}
	if len(links) != 1 || links[0].RevokedAt == nil {
		t.Fatalf("expected the link to show revoked, got %+v", links)
	}
}
