package formulare

// Share tokens are stored only as their SHA-256 since 000351 (see
// internal/wiki/share_token_hash_db_test.go for the wiki half). A new secret
// never reaches the table, and a link minted before the migration -- whose
// hash the migration computed in SQL -- still takes submissions.

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/linksecret"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestFormShareLink_StoredOnlyAsHash(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Formulare Share Hash Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	sysCtx := database.WithSystemContext(context.Background())
	now := time.Now().UTC()

	schema := &FormSchema{
		ID: uuid.New(), TenantID: tenantID, Title: "Share Hash Schema",
		Fields: []byte(`[]`), Status: FormSchemaStatusActive, IsPublic: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSchema(ctx, schema); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_schemas", schema.ID)

	t.Run("a new secret reaches the caller once and the table only as hash", func(t *testing.T) {
		link, err := svc.CreateShareLink(ctx, CreateShareLinkInput{TenantID: tenantID, FormSchemaID: schema.ID})
		if err != nil {
			t.Fatalf("CreateShareLink: %v", err)
		}
		defer testutil.CleanupRow(t, pool, "form_share_tokens", link.ID)
		if link.Token == "" {
			t.Fatal("create response carries no token")
		}

		var row, hash string
		if err := pool.QueryRow(sysCtx,
			`SELECT row_to_json(f)::text, token_hash FROM form_share_tokens f WHERE id = $1`, link.ID,
		).Scan(&row, &hash); err != nil {
			t.Fatalf("read stored row: %v", err)
		}
		if strings.Contains(row, link.Token) {
			t.Fatalf("stored row contains the clear secret: %s", row)
		}
		if hash != linksecret.Hash(link.Token) {
			t.Fatalf("token_hash = %s, want linksecret.Hash of the secret", hash)
		}

		listed, err := svc.ListShareLinks(ctx, schema.ID, tenantID)
		if err != nil {
			t.Fatalf("ListShareLinks: %v", err)
		}
		if len(listed) != 1 || listed[0].ID != link.ID || listed[0].Token != "" {
			t.Fatalf("listing = %+v, want the one link without its secret", listed)
		}

		sub, err := svc.SubmitByShareToken(context.Background(), link.Token, []byte(`{"f1":"a"}`), "", "")
		if err != nil {
			t.Fatalf("SubmitByShareToken: %v", err)
		}
		defer testutil.CleanupRow(t, pool, "form_submissions", sub.SubmissionID)
		if _, err := svc.SubmitByShareToken(context.Background(), hash, []byte(`{"f1":"a"}`), "", ""); err == nil {
			t.Fatal("the stored hash itself was accepted as a token")
		}
	})

	t.Run("a link hashed by the migration backfill still takes submissions", func(t *testing.T) {
		legacy, err := linksecret.New()
		if err != nil {
			t.Fatalf("linksecret.New: %v", err)
		}
		id := uuid.New()
		if _, err := pool.Exec(ctx,
			`INSERT INTO form_share_tokens (id, tenant_id, form_schema_id, token_hash)
			 VALUES ($1, $2, $3, encode(sha256(convert_to($4::text, 'UTF8')), 'hex'))`,
			id, tenantID, schema.ID, legacy,
		); err != nil {
			t.Fatalf("insert backfilled link: %v", err)
		}
		defer testutil.CleanupRow(t, pool, "form_share_tokens", id)

		sub, err := svc.SubmitByShareToken(context.Background(), legacy, []byte(`{"f1":"b"}`), "", "")
		if err != nil {
			t.Fatalf("SubmitByShareToken (backfilled): %v", err)
		}
		defer testutil.CleanupRow(t, pool, "form_submissions", sub.SubmissionID)
		if sub.TenantID != tenantID || sub.FormSchemaID != schema.ID {
			t.Fatalf("submission landed in %s/%s, want %s/%s", sub.TenantID, sub.FormSchemaID, tenantID, schema.ID)
		}
	})
}
