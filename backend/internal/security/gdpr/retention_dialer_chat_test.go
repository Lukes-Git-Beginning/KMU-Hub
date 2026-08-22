package gdpr

// Coverage for DialerCallRetentionHandler and ChatMessageRetentionHandler
// (Lauf 10, second and third handlers on the retention engine from A10).
// The cases that matter: candidates outside the retention window never show
// up as Due, delete and anonymize both work, anonymize is idempotent via the
// same updated_at/edited_at bump technique ContactRetentionHandler uses, and
// deleting a chat reply corrects its parent's reply_count.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// ---------------------------------------------------------------------------
// DialerCallRetentionHandler
// ---------------------------------------------------------------------------

func TestDialerCallRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewDialerCallRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.True(t, handler.SupportsAction(models.RetentionActionAnonymize))
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "dialer_calls", handler.ResourceType())
	assert.Equal(t, "dialer_call_sessions", handler.Table())
}

func TestDialerCallRetentionHandler_PlanExcludesFreshSessions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dialer Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	agentID := seedExportUser(t, pool, tenantID, "dialer-retention-plan-agent")
	defer testutil.CleanupRow(t, pool, "users", agentID)

	contactID, campaignID, campaignContactID := seedDialerCampaignContact(t, pool, tenantID, agentID)
	defer testutil.CleanupRow(t, pool, "contacts", contactID)
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campaignID)
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", campaignContactID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	fresh := time.Now().UTC().AddDate(0, 0, -2)

	oldSession := seedDialerCallSession(t, pool, tenantID, campaignContactID, agentID, old)
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", oldSession)

	freshSession := seedDialerCallSession(t, pool, tenantID, campaignContactID, agentID, fresh)
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", freshSession)

	handler := NewDialerCallRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{oldSession}, plan.Due)
	assert.Empty(t, plan.Skipped)
}

func TestDialerCallRetentionHandler_ApplyDeleteCascadesEvents(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dialer Retention Apply Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	agentID := seedExportUser(t, pool, tenantID, "dialer-retention-apply-del-agent")
	defer testutil.CleanupRow(t, pool, "users", agentID)

	contactID, campaignID, campaignContactID := seedDialerCampaignContact(t, pool, tenantID, agentID)
	defer testutil.CleanupRow(t, pool, "contacts", contactID)
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campaignID)
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", campaignContactID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	sessionID := seedDialerCallSession(t, pool, tenantID, campaignContactID, agentID, old)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	testutil.SeedRow(t, pool, "dialer_call_events", map[string]any{
		"tenant_id":              tenantID,
		"dialer_call_session_id": sessionID,
		"event_type":             "initiated",
	})

	handler := NewDialerCallRetentionHandler(pool)
	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{sessionID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var sessionCount, eventCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM dialer_call_sessions WHERE id = $1`, sessionID).Scan(&sessionCount))
	assert.Zero(t, sessionCount, "deleted session must be gone")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM dialer_call_events WHERE dialer_call_session_id = $1`, sessionID).Scan(&eventCount))
	assert.Zero(t, eventCount, "cascade must take the event trail with it")
}

func TestDialerCallRetentionHandler_ApplyAnonymizeIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dialer Retention Apply Anonymize Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	agentID := seedExportUser(t, pool, tenantID, "dialer-retention-apply-anon-agent")
	defer testutil.CleanupRow(t, pool, "users", agentID)

	contactID, campaignID, campaignContactID := seedDialerCampaignContact(t, pool, tenantID, agentID)
	defer testutil.CleanupRow(t, pool, "contacts", contactID)
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campaignID)
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", campaignContactID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	sessionID := seedDialerCallSession(t, pool, tenantID, campaignContactID, agentID, old)
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", sessionID)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewDialerCallRetentionHandler(pool)

	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{sessionID}, models.RetentionActionAnonymize, "Geloeschter Anruf 1")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var notes string
	var nextAction *string
	require.NoError(t, pool.QueryRow(ctx, `SELECT notes, next_action FROM dialer_call_sessions WHERE id = $1`, sessionID).Scan(&notes, &nextAction))
	assert.Equal(t, "[Geloeschter Anruf 1]", notes)
	assert.Nil(t, nextAction)

	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionAnonymize)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, sessionID, "anonymize must bump updated_at, taking the row out of the next Plan")
}

// seedDialerCampaignContact returns (contactID, campaignID, campaignContactID)
// so the caller can clean them up in FK-safe order (see the defer stack in
// each test: the last deferred cleanup runs first).
func seedDialerCampaignContact(t *testing.T, pool *pgxpool.Pool, tenantID, agentID uuid.UUID) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Dialer",
		"last_name":  "Retention",
		"created_by": agentID,
	})
	campaignID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"tenant_id":  tenantID,
		"name":       "Retention Test Campaign",
		"created_by": agentID,
	})
	campaignContactID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"tenant_id":   tenantID,
		"campaign_id": campaignID,
		"contact_id":  contactID,
		"position":    1,
	})
	return contactID, campaignID, campaignContactID
}

func seedDialerCallSession(t *testing.T, pool *pgxpool.Pool, tenantID, campaignContactID, agentID uuid.UUID, at time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"tenant_id":           tenantID,
		"campaign_contact_id": campaignContactID,
		"agent_id":            agentID,
		"notes":               "Ursprüngliche Notiz",
		"created_at":          at,
		"updated_at":          at,
	})
}

// ---------------------------------------------------------------------------
// ChatMessageRetentionHandler
// ---------------------------------------------------------------------------

func TestChatMessageRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewChatMessageRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.True(t, handler.SupportsAction(models.RetentionActionAnonymize))
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "chat_messages", handler.ResourceType())
	assert.Equal(t, "messages", handler.Table())
}

func TestChatMessageRetentionHandler_PlanExcludesFreshAndEditedMessages(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Chat Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := seedExportUser(t, pool, tenantID, "chat-retention-plan-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	channelID := seedRetentionChannel(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	fresh := time.Now().UTC().AddDate(0, 0, -2)

	oldMessage := seedRetentionMessage(t, pool, tenantID, channelID, userID, "Alte Nachricht", old, nil, nil)
	defer testutil.CleanupRow(t, pool, "messages", oldMessage)

	freshMessage := seedRetentionMessage(t, pool, tenantID, channelID, userID, "Neue Nachricht", fresh, nil, nil)
	defer testutil.CleanupRow(t, pool, "messages", freshMessage)

	// Old creation date but recently edited — must be treated as still relevant.
	editedMessage := seedRetentionMessage(t, pool, tenantID, channelID, userID, "Bearbeitete Nachricht", old, &fresh, nil)
	defer testutil.CleanupRow(t, pool, "messages", editedMessage)

	handler := NewChatMessageRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{oldMessage}, plan.Due)
}

func TestChatMessageRetentionHandler_ApplyAnonymizeReusesErasurePattern(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Chat Retention Apply Anonymize Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := seedExportUser(t, pool, tenantID, "chat-retention-apply-anon-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	channelID := seedRetentionChannel(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	messageID := seedRetentionMessage(t, pool, tenantID, channelID, userID, "Persoenliche Nachricht", old, nil, nil)
	defer testutil.CleanupRow(t, pool, "messages", messageID)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewChatMessageRetentionHandler(pool)

	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{messageID}, models.RetentionActionAnonymize, "Geloeschte Nachricht 1")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var content string
	var isDeleted bool
	require.NoError(t, pool.QueryRow(ctx, `SELECT content, is_deleted FROM messages WHERE id = $1`, messageID).Scan(&content, &isDeleted))
	assert.Equal(t, "[Geloeschte Nachricht 1]", content)
	assert.True(t, isDeleted)

	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionAnonymize)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, messageID, "anonymize must bump edited_at, taking the row out of the next Plan")
}

func TestChatMessageRetentionHandler_ApplyDeleteFixesUpParentReplyCount(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Chat Retention Apply Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := seedExportUser(t, pool, tenantID, "chat-retention-apply-del-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	channelID := seedRetentionChannel(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	parentID := seedRetentionMessage(t, pool, tenantID, channelID, userID, "Elternnachricht", old, nil, nil)
	defer testutil.CleanupRow(t, pool, "messages", parentID)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	_, err := pool.Exec(ctx, `UPDATE messages SET reply_count = 1 WHERE id = $1`, parentID)
	require.NoError(t, err)

	replyID := seedRetentionMessage(t, pool, tenantID, channelID, userID, "Antwort", old, nil, &parentID)

	handler := NewChatMessageRetentionHandler(pool)
	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{replyID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var replyCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT reply_count FROM messages WHERE id = $1`, parentID).Scan(&replyCount))
	assert.Zero(t, replyCount, "deleting the only reply must bring the parent's counter back to zero")

	var goneCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM messages WHERE id = $1`, replyID).Scan(&goneCount))
	assert.Zero(t, goneCount)
}

func seedRetentionChannel(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantID,
		"name":       "retention-test-" + uuid.New().String(),
		"created_by": createdBy,
	})
}

func seedRetentionMessage(t *testing.T, pool *pgxpool.Pool, tenantID, channelID, createdBy uuid.UUID, content string, createdAt time.Time, editedAt *time.Time, parentMessageID *uuid.UUID) uuid.UUID {
	t.Helper()
	cols := map[string]any{
		"tenant_id":  tenantID,
		"channel_id": channelID,
		"content":    content,
		"created_by": createdBy,
		"created_at": createdAt,
	}
	if editedAt != nil {
		cols["edited_at"] = *editedAt
	}
	if parentMessageID != nil {
		cols["parent_message_id"] = *parentMessageID
	}
	return testutil.SeedRow(t, pool, "messages", cols)
}
