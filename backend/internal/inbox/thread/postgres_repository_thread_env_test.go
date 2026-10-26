package thread

// DB tests for the thread-message half of the repository (the canned-response
// half lives in postgres_repository_canned_test.go). They run as kmuhub_app via
// DATABASE_URL, so RLS is live: a wrong tenant filter shows up as a wrong row.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedInboxMessage creates an inbox message for tenantID (the tenant already
// exists via newTestPool); thread rows cascade away with it.
func seedInboxMessage(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": "thread-" + uuid.NewString() + "@example.test",
		"password_hash": "hash", "first_name": "Thread", "last_name": "Owner",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
	messageID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id": tenantID, "user_id": userID, "channel": "email",
		"source_id": "src-" + uuid.NewString(), "received_at": time.Now().UTC(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_messages", messageID) })
	return messageID
}

func TestSeedInboundIfEmpty_SeedsOnceAndIsIdempotent(t *testing.T) {
	repo, pool, ctx, tenantID := newTestPool(t)
	messageID := seedInboxMessage(t, pool, tenantID)
	at := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)

	require.NoError(t, repo.SeedInboundIfEmpty(ctx, tenantID, messageID, "Erika Muster", "Erste Nachricht", at))
	require.NoError(t, repo.SeedInboundIfEmpty(ctx, tenantID, messageID, "Anderer", "Zweiter Versuch", at))

	got, err := repo.ListThreadMessages(ctx, tenantID, messageID)
	require.NoError(t, err)
	require.Len(t, got, 1, "a second seed must not add a row")
	assert.Equal(t, DirectionInbound, got[0].Direction)
	assert.Equal(t, "Erste Nachricht", got[0].Body)
	assert.Equal(t, "Erika Muster", got[0].AuthorName, "no users row: the stored sender name is the fallback")
	assert.Nil(t, got[0].AuthorID)
	assert.Equal(t, tenantID, got[0].TenantID)
	assert.True(t, at.Equal(got[0].CreatedAt), "explicit createdAt is kept, got %s", got[0].CreatedAt)
}

func TestSeedInboundIfEmpty_ZeroTimeDefaultsToNow(t *testing.T) {
	repo, pool, ctx, tenantID := newTestPool(t)
	messageID := seedInboxMessage(t, pool, tenantID)
	before := time.Now().UTC().Add(-time.Minute)

	require.NoError(t, repo.SeedInboundIfEmpty(ctx, tenantID, messageID, "", "Body", time.Time{}))

	got, err := repo.ListThreadMessages(ctx, tenantID, messageID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.True(t, got[0].CreatedAt.After(before), "zero createdAt falls back to now, got %s", got[0].CreatedAt)
	assert.Empty(t, got[0].AuthorName, "blank sender is stored as NULL and listed as empty string")
}

func TestSeedInboundIfEmpty_IgnoresOtherTenantsRows(t *testing.T) {
	repo, pool, ctx, tenantID := newTestPool(t)
	messageID := seedInboxMessage(t, pool, tenantID)

	// A foreign tenant's row on the same message id must neither count as
	// "thread not empty" nor show up in our listing.
	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Other Inbox Thread Env Tenant")
	foreign := testutil.SeedRow(t, pool, "inbox_thread_messages", map[string]any{
		"tenant_id": otherTenantID, "message_id": messageID, "direction": DirectionInbound, "body": "fremd",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_thread_messages", foreign) })

	require.NoError(t, repo.SeedInboundIfEmpty(ctx, tenantID, messageID, "Wir", "Unsere Nachricht", time.Time{}))

	got, err := repo.ListThreadMessages(ctx, tenantID, messageID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Unsere Nachricht", got[0].Body)

	otherCtx := testutil.WithTenantCtx(context.Background(), otherTenantID)
	theirs, err := repo.ListThreadMessages(otherCtx, otherTenantID, messageID)
	require.NoError(t, err)
	require.Len(t, theirs, 1)
	assert.Equal(t, "fremd", theirs[0].Body)
}

func TestAppendThreadMessage_GeneratesIDAndTimestamp(t *testing.T) {
	repo, pool, ctx, tenantID := newTestPool(t)
	messageID := seedInboxMessage(t, pool, tenantID)
	before := time.Now().UTC().Add(-time.Minute)

	m := &ThreadMessage{TenantID: tenantID, MessageID: messageID, Direction: DirectionInternal, Body: "Notiz", AuthorName: "Extern"}
	require.NoError(t, repo.AppendThreadMessage(ctx, m))

	assert.NotEqual(t, uuid.Nil, m.ID)
	assert.True(t, m.CreatedAt.After(before))
	got, err := repo.ListThreadMessages(ctx, tenantID, messageID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, m.ID, got[0].ID)
	assert.Equal(t, DirectionInternal, got[0].Direction)
	assert.Equal(t, "Extern", got[0].AuthorName)
}

func TestAppendThreadMessage_RejectsUnknownDirection(t *testing.T) {
	repo, pool, ctx, tenantID := newTestPool(t)
	messageID := seedInboxMessage(t, pool, tenantID)

	err := repo.AppendThreadMessage(ctx, &ThreadMessage{TenantID: tenantID, MessageID: messageID, Direction: "sideways", Body: "x"})

	assert.Error(t, err, "the direction CHECK constraint must surface as an error")
}

func TestListThreadMessages_OrderedByCreatedAtAndNamedFromUser(t *testing.T) {
	repo, pool, ctx, tenantID := newTestPool(t)
	messageID := seedInboxMessage(t, pool, tenantID)
	authorID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": "author-" + uuid.NewString() + "@example.test",
		"password_hash": "hash", "first_name": "Max", "last_name": "Beispiel",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", authorID) })

	base := time.Now().UTC().Add(-time.Hour)
	// Insert newest first so insertion order and created_at order differ.
	require.NoError(t, repo.AppendThreadMessage(ctx, &ThreadMessage{
		TenantID: tenantID, MessageID: messageID, AuthorID: &authorID, AuthorName: "Stale Name",
		Direction: DirectionOutbound, Body: "zweite", CreatedAt: base.Add(10 * time.Minute),
	}))
	require.NoError(t, repo.AppendThreadMessage(ctx, &ThreadMessage{
		TenantID: tenantID, MessageID: messageID, Direction: DirectionInbound, AuthorName: "Kunde",
		Body: "erste", CreatedAt: base,
	}))

	got, err := repo.ListThreadMessages(ctx, tenantID, messageID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "erste", got[0].Body)
	assert.Equal(t, "Kunde", got[0].AuthorName)
	assert.Equal(t, "zweite", got[1].Body)
	assert.Equal(t, "Max Beispiel", got[1].AuthorName, "the live users name wins over the stored author_name")
	require.NotNil(t, got[1].AuthorID)
	assert.Equal(t, authorID, *got[1].AuthorID)
}

func TestListThreadMessages_UnknownMessageIsEmpty(t *testing.T) {
	repo, _, ctx, tenantID := newTestPool(t)

	got, err := repo.ListThreadMessages(ctx, tenantID, uuid.New())

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCannedResponses_UpdateAndDeleteRespectTenant(t *testing.T) {
	repo, pool, ctx, tenantID := newTestPool(t)
	c := &CannedResponse{TenantID: tenantID, Name: "Gruss", Body: "Hallo"}
	require.NoError(t, repo.CreateCannedResponse(ctx, c))

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Other Canned Env Tenant")
	otherCtx := testutil.WithTenantCtx(context.Background(), otherTenantID)

	// Same ID, foreign tenant: neither update nor delete may touch the row.
	foreign := &CannedResponse{ID: c.ID, TenantID: otherTenantID, Name: "gekapert", Body: "x"}
	assert.ErrorIs(t, repo.UpdateCannedResponse(otherCtx, foreign), ErrCannedResponseNotFound)
	assert.ErrorIs(t, repo.DeleteCannedResponse(otherCtx, otherTenantID, c.ID), ErrCannedResponseNotFound)
	// Argument tenant != context tenant: RLS still hides the row.
	assert.ErrorIs(t, repo.DeleteCannedResponse(otherCtx, tenantID, c.ID), ErrCannedResponseNotFound)

	got, err := repo.GetCannedResponse(ctx, tenantID, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "Gruss", got.Name)
	assert.Equal(t, "Hallo", got.Body)
}

func TestNullString(t *testing.T) {
	assert.Nil(t, nullString(""))
	got := nullString("x")
	require.NotNil(t, got)
	assert.Equal(t, "x", *got)
}

// Under the system context RLS admits every row, so only the explicit
// tenant_id predicates in the queries keep tenants apart. This pins them
// independently of the policy.
func TestTenantPredicates_HoldWithoutRLS(t *testing.T) {
	repo, pool, _, tenantID := newTestPool(t)
	sysCtx := testutil.WithSystemCtx(context.Background())
	messageID := seedInboxMessage(t, pool, tenantID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Other Predicate Tenant")
	foreign := testutil.SeedRow(t, pool, "inbox_thread_messages", map[string]any{
		"tenant_id": otherTenantID, "message_id": messageID, "direction": DirectionInbound, "body": "fremd",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_thread_messages", foreign) })

	// The foreign row must not make our thread look non-empty ...
	require.NoError(t, repo.SeedInboundIfEmpty(sysCtx, tenantID, messageID, "Wir", "Unsere Nachricht", time.Time{}))

	// ... and must not leak into our listing.
	got, err := repo.ListThreadMessages(sysCtx, tenantID, messageID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Unsere Nachricht", got[0].Body)

	// Canned responses: same ID under another tenant argument is not found.
	c := &CannedResponse{TenantID: tenantID, Name: "Gruss", Body: "Hallo"}
	require.NoError(t, repo.CreateCannedResponse(sysCtx, c))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_canned_responses", c.ID) })

	_, err = repo.GetCannedResponse(sysCtx, otherTenantID, c.ID)
	assert.ErrorIs(t, err, ErrCannedResponseNotFound)
	assert.ErrorIs(t, repo.UpdateCannedResponse(sysCtx, &CannedResponse{ID: c.ID, TenantID: otherTenantID, Name: "x", Body: "y"}), ErrCannedResponseNotFound)
	assert.ErrorIs(t, repo.DeleteCannedResponse(sysCtx, otherTenantID, c.ID), ErrCannedResponseNotFound)
	list, err := repo.ListCannedResponses(sysCtx, otherTenantID)
	require.NoError(t, err)
	assert.Empty(t, list)
}
