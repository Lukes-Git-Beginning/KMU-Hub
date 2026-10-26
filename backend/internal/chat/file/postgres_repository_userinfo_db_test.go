package file

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresRepository_GetUserInfo(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newFileReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	first, last, err := repo.GetUserInfo(ctxOwn, fx.user)
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if first != "Filip" || last != "Uploader" {
		t.Fatalf("GetUserInfo: expected Filip Uploader, got %q %q", first, last)
	}

	if _, _, err := repo.GetUserInfo(ctxOwn, uuid.New()); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetUserInfo unknown user: expected pgx.ErrNoRows, got %v", err)
	}

	// RLS: a user of another tenant is invisible, not an empty-named success.
	other := uuid.New()
	testutil.EnsureTenant(t, pool, other, "Chat File UserInfo Other Tenant")
	if _, _, err := repo.GetUserInfo(testutil.WithTenantCtx(context.Background(), other), fx.user); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetUserInfo across tenants: expected pgx.ErrNoRows, got %v", err)
	}
}

func TestPostgresRepository_ListChannelFiles_CancelledContext(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newFileReadFixture(t, pool)
	fileID := seedChatFile(t, pool, fx.tenant, fx.channel, fx.user, nil, "cancel.pdf", false, time.Now().UTC())
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", fileID) })

	ctx, cancel := context.WithCancel(testutil.WithTenantCtx(context.Background(), fx.tenant))
	cancel()

	files, total, err := NewPostgresRepository(pool).ListChannelFiles(ctx, fx.channel, 10, 0)
	if err == nil {
		t.Fatalf("ListChannelFiles with cancelled context: expected an error, got %d files", len(files))
	}
	if files != nil || total != 0 {
		t.Fatalf("ListChannelFiles with cancelled context: expected nothing, got %d files, total %d", len(files), total)
	}
}
