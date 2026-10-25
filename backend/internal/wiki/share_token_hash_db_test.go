package wiki

// Share tokens are stored only as their SHA-256 since 000351. These tests pin
// the two halves of that against the real database: a freshly minted secret
// never reaches the table, and a token minted before the migration -- whose
// hash the migration computed in SQL -- still redeems through the Go lookup.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/linksecret"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestWikiShareToken_StoredOnlyAsHash(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Wiki Share Hash Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	sysCtx := database.WithSystemContext(context.Background())
	now := time.Now().UTC()

	article := &Article{
		ID: uuid.New(), TenantID: tenantID,
		Title: "Hash-Test-" + uuid.New().String()[:8], Slug: "hash-test-" + uuid.New().String()[:8],
		Content: []byte(`{"type":"doc"}`), AuthorID: uuid.New(), Published: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateArticle(ctx, article); err != nil {
		t.Fatalf("CreateArticle: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_articles", article.ID)

	t.Run("a new secret reaches the caller once and the table only as hash", func(t *testing.T) {
		share, err := svc.CreateShareToken(ctx, CreateShareTokenInput{TenantID: tenantID, ArticleID: article.ID})
		if err != nil {
			t.Fatalf("CreateShareToken: %v", err)
		}
		defer testutil.CleanupRow(t, pool, "wiki_share_tokens", share.ID)
		if share.Token == "" {
			t.Fatal("create response carries no token")
		}

		var row, hash string
		if err := pool.QueryRow(sysCtx,
			`SELECT row_to_json(w)::text, token_hash FROM wiki_share_tokens w WHERE id = $1`, share.ID,
		).Scan(&row, &hash); err != nil {
			t.Fatalf("read stored row: %v", err)
		}
		if strings.Contains(row, share.Token) {
			t.Fatalf("stored row contains the clear secret: %s", row)
		}
		if hash != linksecret.Hash(share.Token) {
			t.Fatalf("token_hash = %s, want linksecret.Hash of the secret", hash)
		}

		listed, err := svc.ListShareTokens(ctx, tenantID, article.ID)
		if err != nil {
			t.Fatalf("ListShareTokens: %v", err)
		}
		if len(listed) != 1 || listed[0].ID != share.ID || listed[0].Token != "" {
			t.Fatalf("listing = %+v, want the one token without its secret", listed)
		}

		if _, err := svc.RedeemShareToken(context.Background(), share.Token); err != nil {
			t.Fatalf("RedeemShareToken: %v", err)
		}
		if _, err := svc.RedeemShareToken(context.Background(), hash); err == nil {
			t.Fatal("the stored hash itself redeemed as a token")
		}
	})

	t.Run("a token hashed by the migration backfill still redeems", func(t *testing.T) {
		// A pre-000351 token: the legacy uuid format, hashed with the exact
		// expression the migration ran over the clear column.
		legacy := uuid.New().String()
		id := uuid.New()
		if _, err := pool.Exec(ctx,
			`INSERT INTO wiki_share_tokens (id, tenant_id, article_id, token_hash, permissions)
			 VALUES ($1, $2, $3, encode(sha256(convert_to($4::text, 'UTF8')), 'hex'), '{read}')`,
			id, tenantID, article.ID, legacy,
		); err != nil {
			t.Fatalf("insert backfilled token: %v", err)
		}
		defer testutil.CleanupRow(t, pool, "wiki_share_tokens", id)

		got, err := svc.RedeemShareToken(context.Background(), legacy)
		if err != nil {
			t.Fatalf("RedeemShareToken (backfilled): %v", err)
		}
		if got.Title != article.Title {
			t.Fatalf("redeemed %q, want %q", got.Title, article.Title)
		}
	})

	t.Run("the clear column is gone", func(t *testing.T) {
		var n int
		if err := pool.QueryRow(context.Background(),
			`SELECT count(*) FROM information_schema.columns
			  WHERE table_name IN ('wiki_share_tokens', 'form_share_tokens') AND column_name = 'token'`,
		).Scan(&n); err != nil {
			t.Fatalf("read columns: %v", err)
		}
		if n != 0 {
			t.Fatalf("%d share-token tables still have a clear token column", n)
		}
	})
}
