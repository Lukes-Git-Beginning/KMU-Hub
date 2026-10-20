package formulare

// DB-backed proof that the Idempotency-Key of the public submit route works
// under RLS: the reservation is written with the tenant the share link
// resolved to (no JWT, no gateway tenant), as kmuhub_app.

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSubmitByShareToken_IdempotencyKeyDedupesAgainstPostgres(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare Public Idempotency Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.SetIdempotency(idempotency.NewPostgresRepository(pool))

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()
	schema := &FormSchema{
		ID: uuid.New(), TenantID: tenantID, Title: "Public Idempotency Schema",
		Fields: []byte(`[]`), Status: FormSchemaStatusActive, IsPublic: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSchema(ctx, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	link, err := svc.CreateShareLink(ctx, CreateShareLinkInput{TenantID: tenantID, FormSchemaID: schema.ID})
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_share_tokens", link.ID)

	key := "retry-" + uuid.NewString()
	answers := []byte(`{"f1":"x"}`)

	first, err := svc.SubmitByShareToken(context.Background(), link.Token, answers, "", key)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_submissions", first.SubmissionID)
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE tenant_id = $1`, tenantID)
	}()

	second, err := svc.SubmitByShareToken(context.Background(), link.Token, answers, "", key)
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}
	if !second.Replayed || second.SubmissionID != first.SubmissionID || second.TenantID != tenantID {
		t.Fatalf("second = %+v, want replay of %s in tenant %s", second, first.SubmissionID, tenantID)
	}

	links, err := svc.ListShareLinks(ctx, schema.ID, tenantID)
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}
	if len(links) != 1 || links[0].SubmissionCount != 1 {
		t.Fatalf("share link = %#v, want exactly one redemption", links)
	}
}
