package message

// Writes into messages and message_mentions must land in — and stay visible
// only to — the caller's tenant. messages carries tenant_id NOT NULL since
// migration 000106 but had no RLS policy until migration 000253: every
// sibling chat table (channel_memberships, message_mentions, chat_files,
// guest_sessions, ...) was RLS-enabled in the mig 000122 sweep, messages was
// silently dropped from that list. GetByID always takes an explicit tenantID
// and filters on it in SQL, so the application layer already scoped every
// call correctly — but message_mentions.tenant_id lands via CreateMentions,
// and without RLS on the parent messages table a raw query had no
// database-level backstop. This test would have gone red against the
// pre-000253 schema.
//
// message_mentions has a composite primary key (message_id, user_id), no
// `id` column — assertRowCountWhere below generalizes testutil.AssertRowCount
// the same way internal/chat/channel/tenant_write_test.go does.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestMessageWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Message Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Message Write Other Tenant")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("message-write-a-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)
	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("message-write-b-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "message-write-test",
		"created_by": userA,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	// Create
	msg := &models.Message{
		ID: uuid.New(), TenantID: tenantOwn, ChannelID: channelID, Content: "hello",
		Lang: "german", CreatedBy: &userA, CreatedAt: now,
	}
	if err := repo.Create(ctxOwn, msg); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "messages", msg.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "messages", msg.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "messages", msg.ID, 0)

	// Update
	msg.Content = "hello (edited)"
	editedAt := time.Now().UTC()
	msg.EditedAt = &editedAt
	if err := repo.Update(ctxOwn, msg); err != nil {
		t.Fatalf("Update: %v", err)
	}
	testutil.AssertRowCount(t, pool, ctxOther, "messages", msg.ID, 0)

	// CreateMentions
	mentions := []models.Mention{
		{MessageID: msg.ID, TenantID: tenantOwn, UserID: userB, MentionType: models.MentionTypeUser, CreatedAt: now},
	}
	if err := repo.CreateMentions(ctxOwn, msg.ID, mentions); err != nil {
		t.Fatalf("CreateMentions: %v", err)
	}
	assertRowCountWhere(t, pool, ctxOwn, "message_mentions", "message_id = $1 AND user_id = $2", 1, msg.ID, userB)
	assertRowCountWhere(t, pool, ctxOther, "message_mentions", "message_id = $1 AND user_id = $2", 0, msg.ID, userB)

	// CreateWithReplyCount — transactional reply insert + parent reply_count bump.
	reply := &models.Message{
		ID: uuid.New(), TenantID: tenantOwn, ChannelID: channelID, Content: "a reply",
		Lang: "german", CreatedBy: &userA, ParentMessageID: &msg.ID, CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateWithReplyCount(ctxOwn, reply); err != nil {
		t.Fatalf("CreateWithReplyCount: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "messages", reply.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "messages", reply.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "messages", reply.ID, 0)

	// Delete — soft delete (is_deleted = TRUE), row stays but flips.
	if err := repo.Delete(ctxOwn, reply.ID, tenantOwn); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var isDeleted bool
	if err := pool.QueryRow(sysCtx, "SELECT is_deleted FROM messages WHERE id = $1", reply.ID).Scan(&isDeleted); err != nil {
		t.Fatalf("read back is_deleted: %v", err)
	}
	if !isDeleted {
		t.Fatalf("Delete: expected is_deleted = true, got false")
	}
}

// assertRowCountWhere runs `SELECT count(*) FROM <table> WHERE <where>` under
// ctx (which determines RLS visibility) and fails if the count differs.
// Generalizes testutil.AssertRowCount (keys on `id`) for tables with a
// composite primary key, such as message_mentions.
func assertRowCountWhere(t *testing.T, pool *pgxpool.Pool, ctx context.Context, table, where string, expected int, args ...any) {
	t.Helper()
	var count int
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", table, where)
	if err := pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		t.Fatalf("count %s where %s: %v", table, where, err)
	}
	if count != expected {
		t.Fatalf("table %s where %s: expected %d row(s), got %d", table, where, expected, count)
	}
}
