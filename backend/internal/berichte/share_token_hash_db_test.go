package berichte

// Report share tokens are stored only as their SHA-256 since 000367, the same
// line wiki and forms drew in 000351. A new secret reaches the caller once and
// the table never, the list carries no secret at all, and a link minted before
// the migration -- whose hash the migration computed in SQL -- still opens the
// report.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/linksecret"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestReportShareToken_StoredOnlyAsHash(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close) // registered first, so it runs after the row cleanups

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte Share Hash Tenant")
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	sysCtx := database.WithSystemContext(context.Background())
	svc := NewService(NewPostgresRepository(pool), nil, Options{})

	doc, err := svc.CreateDocument(ctx, CreateDocumentInput{TenantID: tenant, Title: "Geteilter Bericht"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_documents", doc.ID) })

	t.Run("the clear token column is gone", func(t *testing.T) {
		var n int
		if err := pool.QueryRow(sysCtx,
			`SELECT count(*) FROM information_schema.columns
			  WHERE table_name = 'report_share_tokens' AND column_name = 'token'`,
		).Scan(&n); err != nil {
			t.Fatalf("read columns: %v", err)
		}
		if n != 0 {
			t.Fatal("report_share_tokens still has a clear-text token column")
		}
	})

	t.Run("a new secret reaches the caller once and the table only as hash", func(t *testing.T) {
		link, err := svc.CreateShareToken(ctx, CreateShareTokenInput{TenantID: tenant, DocumentID: doc.ID})
		if err != nil {
			t.Fatalf("CreateShareToken: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_share_tokens", link.ID) })
		if link.Token == "" {
			t.Fatal("create response carries no token")
		}

		var row, hash string
		if err := pool.QueryRow(sysCtx,
			`SELECT row_to_json(s)::text, token_hash FROM report_share_tokens s WHERE id = $1`, link.ID,
		).Scan(&row, &hash); err != nil {
			t.Fatalf("read stored row: %v", err)
		}
		if strings.Contains(row, link.Token) {
			t.Fatalf("stored row contains the clear secret: %s", row)
		}
		if hash != linksecret.Hash(link.Token) {
			t.Fatalf("token_hash = %s, want linksecret.Hash of the secret", hash)
		}

		listed, err := svc.ListShareTokens(ctx, tenant, doc.ID)
		if err != nil {
			t.Fatalf("ListShareTokens: %v", err)
		}
		if len(listed) != 1 || listed[0].ID != link.ID || listed[0].Token != "" {
			t.Fatalf("listing = %+v, want the one link without its secret", listed)
		}

		got, err := svc.GetSharedDocument(context.Background(), link.Token, "")
		if err != nil {
			t.Fatalf("GetSharedDocument: %v", err)
		}
		if got.ID != doc.ID {
			t.Fatalf("shared document = %s, want %s", got.ID, doc.ID)
		}
		if _, err := svc.GetSharedDocument(context.Background(), hash, ""); !errors.Is(err, ErrShareNotFound) {
			t.Fatalf("the stored hash itself as token = %v, want ErrShareNotFound", err)
		}
	})

	t.Run("a link hashed by the migration backfill still opens the report", func(t *testing.T) {
		legacy, err := linksecret.New()
		if err != nil {
			t.Fatalf("linksecret.New: %v", err)
		}
		id := uuid.New()
		if _, err := pool.Exec(ctx,
			`INSERT INTO report_share_tokens (id, tenant_id, document_id, token_hash, expires_at)
			 VALUES ($1, $2, $3, encode(sha256(convert_to($4::text, 'UTF8')), 'hex'), NOW() + INTERVAL '1 day')`,
			id, tenant, doc.ID, legacy,
		); err != nil {
			t.Fatalf("insert backfilled link: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_share_tokens", id) })

		got, err := svc.GetSharedDocument(context.Background(), legacy, "")
		if err != nil {
			t.Fatalf("GetSharedDocument (backfilled): %v", err)
		}
		if got.ID != doc.ID || got.TenantID != tenant {
			t.Fatalf("backfilled link opened %s/%s, want %s/%s", got.TenantID, got.ID, tenant, doc.ID)
		}
	})
}
