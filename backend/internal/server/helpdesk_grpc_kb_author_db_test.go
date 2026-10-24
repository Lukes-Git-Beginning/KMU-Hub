package server

// DB-backed proof that CreateKBArticle stores the calling user as author.
// The handler used to pass uuid.Nil, so every article carried 00000000-... as
// author_id (fix-helpdesk-kb-article-author-id-uuid-nil).
//
// The tests go through HelpdeskGRPCServer with the real postgres repository
// under the kmuhub_app role, so RLS is active.

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/helpdesk"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
	helpdeskv1 "github.com/kmuhub/kmuhub/proto/helpdesk/v1"
)

func TestHelpdeskGRPC_CreateKBArticle_StoresCallerAsAuthor(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "KB Author Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	svc := helpdesk.NewService(helpdesk.NewPostgresRepository(pool), slog.Default())
	srv := NewHelpdeskGRPCServer(svc, nil, nil)

	authorX := uuid.New()
	ctx := context.WithValue(helpdeskTenantContext(tenantID), middleware.UserIDKey, authorX.String())

	t.Run("author is the calling user", func(t *testing.T) {
		resp, err := srv.CreateKBArticle(ctx, &helpdeskv1.CreateKBArticleRequest{
			TenantId: tenantID.String(), Title: "Passwort zurücksetzen", Content: "Schritte ...",
			Category: "konto", Status: string(helpdesk.KBArticleStatusPublished),
		})
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "helpdesk_kb_articles", uuid.MustParse(resp.GetId())) })
		require.Equal(t, authorX.String(), resp.GetAuthorId())

		// What the next reader sees must carry the same author.
		list, err := srv.ListKBArticle(ctx, &helpdeskv1.ListKBArticleRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		require.Len(t, list.GetArticles(), 1)
		require.Equal(t, authorX.String(), list.GetArticles()[0].GetAuthorId())
	})

	t.Run("two callers get their own author", func(t *testing.T) {
		authorY := uuid.New()
		ctxY := context.WithValue(helpdeskTenantContext(tenantID), middleware.UserIDKey, authorY.String())
		resp, err := srv.CreateKBArticle(ctxY, &helpdeskv1.CreateKBArticleRequest{
			TenantId: tenantID.String(), Title: "Rechnung nachsenden", Status: string(helpdesk.KBArticleStatusDraft),
		})
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "helpdesk_kb_articles", uuid.MustParse(resp.GetId())) })
		require.Equal(t, authorY.String(), resp.GetAuthorId())
	})

	t.Run("without caller the call is unauthenticated and writes nothing", func(t *testing.T) {
		_, err := srv.CreateKBArticle(helpdeskTenantContext(tenantID), &helpdeskv1.CreateKBArticleRequest{
			TenantId: tenantID.String(), Title: "Ohne Aufrufer",
		})
		require.Equal(t, codes.Unauthenticated, status.Code(err))

		list, err := srv.ListKBArticle(ctx, &helpdeskv1.ListKBArticleRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		for _, a := range list.GetArticles() {
			require.NotEqual(t, "Ohne Aufrufer", a.GetTitle())
		}
	})
}
