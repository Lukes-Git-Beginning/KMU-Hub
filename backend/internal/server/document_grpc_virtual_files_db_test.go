package server

// DB-backed proof that ListVirtualFiles acts as the calling user. The handler
// used to pass uuid.Nil ("gateway handles auth"), but the repository filters
// on channel membership, mailbox owner and task access, so the virtual file
// view was empty for every user (fix-document-virtual-files-nil-user).
//
// The tests go through DocumentGRPCServer with a real postgres repository
// under the kmuhub_app role, so RLS is active and a mock could not hide the
// Nil-user filter.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/document/virtual"
	"github.com/kmuhub/kmuhub/internal/testutil"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

func virtualFilesServerForTest(pool *pgxpool.Pool) *DocumentGRPCServer {
	svc := virtual.NewService(virtual.NewPostgresRepository(pool))
	return NewDocumentGRPCServer(nil, nil, nil, nil, nil, svc, nil, nil)
}

func seedVirtualFilesUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, label string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("vf-%s-%s@test.invalid", label, uuid.New().String()[:8]),
		"password_hash": "x",
		"first_name":    "Virtual",
		"last_name":     label,
	})
}

func TestDocumentGRPCServer_ListVirtualFiles_ActsAsCaller(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Virtual Files Caller Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	member := seedVirtualFilesUser(t, pool, tenantID, "member")
	defer testutil.CleanupRow(t, pool, "users", member)
	outsider := seedVirtualFilesUser(t, pool, tenantID, "outsider")
	defer testutil.CleanupRow(t, pool, "users", outsider)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id": tenantID, "name": "vf-" + uuid.New().String()[:8], "created_by": member,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO channel_memberships (channel_id, user_id, tenant_id) VALUES ($1, $2, $3)`,
		channelID, member, tenantID)
	require.NoError(t, err)

	fileID := testutil.SeedRow(t, pool, "chat_files", map[string]any{
		"tenant_id": tenantID, "channel_id": channelID, "filename": "angebot.pdf",
		"mime_type": "application/pdf", "file_size": int64(2048),
		"storage_key": "channels/" + uuid.New().String(), "uploaded_by": member,
	})
	defer testutil.CleanupRow(t, pool, "chat_files", fileID)

	srv := virtualFilesServerForTest(pool)
	req := &documentv1.ListVirtualFilesRequest{Page: 1, PerPage: 20}

	t.Run("channel member sees the chat file", func(t *testing.T) {
		resp, err := srv.ListVirtualFiles(ctxWithTenantAndUser(tenantID, member), req)
		require.NoError(t, err)
		require.EqualValues(t, 1, resp.Total)
		require.Len(t, resp.Files, 1)
		require.Equal(t, fileID.String(), resp.Files[0].Id)
		require.Equal(t, "angebot.pdf", resp.Files[0].Filename)
	})

	t.Run("non-member does not see it", func(t *testing.T) {
		resp, err := srv.ListVirtualFiles(ctxWithTenantAndUser(tenantID, outsider), req)
		require.NoError(t, err)
		require.EqualValues(t, 0, resp.Total)
		require.Empty(t, resp.Files)
	})

	t.Run("missing caller is Unauthenticated, not an empty list", func(t *testing.T) {
		_, err := srv.ListVirtualFiles(ctxWithTenant(tenantID), req)
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
}
