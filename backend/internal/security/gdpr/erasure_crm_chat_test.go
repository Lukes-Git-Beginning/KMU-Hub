package gdpr

// Integration coverage for CRMErasureHandler and ChatErasureHandler — the two
// erasure handlers in erasure.go that had no test at all. Auth (Execute),
// Notification (Execute) and Work (Preview) are covered in handlers_test.go,
// AuditErasureHandler is a deliberate no-op and covered there as well.
//
// Two properties get pinned here that a happy-path count assertion would miss:
//
//  1. Preview must not write. Both PreviewErasure implementations are pure
//     COUNT queries; the tests re-read the seeded rows afterwards and assert
//     they are untouched.
//  2. Both handlers ignore their `action` parameter — they always anonymize,
//     never delete, regardless of whether ErasureAnonymize or ErasureDelete is
//     passed in. Today that is harmless because Service.ExecuteErasure derives
//     the action from PreviewErasure, which hardcodes "anonymize" for both
//     modules. The tests pin the current contract rather than bend it; see
//     JOURNAL Iteration 39.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

const erasureLabel = "Geloeschter Benutzer #7"

// ---------------------------------------------------------------------------
// CRMErasureHandler
// ---------------------------------------------------------------------------

func TestCRMErasureHandler_PreviewErasure_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Erasure Preview Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Erasure Preview Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := seedExportUser(t, pool, tenantOwn, "crm-erasure-preview")
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Erika",
		"last_name":  "Musterfrau",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	companyID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Musterfirma GmbH",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyID)

	activityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id":     tenantOwn,
		"activity_type": "call",
		"subject":       "Rueckruf Angebot",
		"description":   "Angebot 2026-114 nachfassen",
		"assigned_to":   userID,
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", activityID)

	h := NewCRMErasureHandler(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	preview, err := h.PreviewErasure(ctxOwn, userID)
	require.NoError(t, err)
	require.NotNil(t, preview)
	assert.Equal(t, "crm", preview.ModuleName)
	assert.Equal(t, string(ErasureAnonymize), preview.Action)
	assert.Equal(t, 3, preview.RecordCount, "one contact + one company + one activity")

	// Preview must not write: the activity still carries its assignment and
	// its description afterwards.
	var assignedTo *uuid.UUID
	var description *string
	require.NoError(t, pool.QueryRow(ctxOwn,
		`SELECT assigned_to, description FROM activities WHERE id = $1`, activityID,
	).Scan(&assignedTo, &description))
	require.NotNil(t, assignedTo, "preview must not clear assigned_to")
	assert.Equal(t, userID, *assignedTo)
	require.NotNil(t, description, "preview must not clear description")
	assert.Equal(t, "Angebot 2026-114 nachfassen", *description)

	// The COUNT queries carry no tenant predicate — the boundary is RLS alone.
	previewOther, err := h.PreviewErasure(ctxOther, userID)
	require.NoError(t, err)
	assert.Equal(t, 0, previewOther.RecordCount, "a foreign tenant must not see any CRM record of this user")

	previewUnknown, err := h.PreviewErasure(ctxOwn, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 0, previewUnknown.RecordCount, "an unknown user has nothing to erase")
}

func TestCRMErasureHandler_ExecuteErasure_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Erasure Execute Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "crm-erasure-exec")
	defer testutil.CleanupRow(t, pool, "users", userID)
	colleagueID := seedExportUser(t, pool, tenantOwn, "crm-erasure-colleague")
	defer testutil.CleanupRow(t, pool, "users", colleagueID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Erika",
		"last_name":  "Musterfrau",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	companyID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Musterfirma GmbH",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyID)

	// Assigned to the subject — first UPDATE clears assignment and description.
	assignedActivityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id":     tenantOwn,
		"activity_type": "call",
		"subject":       "Rueckruf Angebot",
		"description":   "Angebot 2026-114 nachfassen",
		"assigned_to":   userID,
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", assignedActivityID)

	// Created by the subject but assigned to somebody else — second UPDATE
	// clears the description only and must leave the assignment alone.
	createdActivityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id":     tenantOwn,
		"activity_type": "meeting",
		"subject":       "Uebergabe",
		"description":   "Persoenliche Notiz",
		"assigned_to":   colleagueID,
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", createdActivityID)

	// Neither created nor assigned by the subject — must survive untouched.
	foreignActivityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id":     tenantOwn,
		"activity_type": "note",
		"subject":       "Fremde Aktivitaet",
		"description":   "Fremde Beschreibung",
		"assigned_to":   colleagueID,
		"created_by":    colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "activities", foreignActivityID)

	h := NewCRMErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)
	// Two activities touched (assigned+created activity counts once, created-only
	// activity counts once) + one contact + one company = 4.
	assert.Equal(t, 4, affected, "each activity is counted exactly once, regardless of matching both branches")

	// The assigned activity loses both its assignment and its description.
	var assignedTo *uuid.UUID
	var description *string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT assigned_to, description FROM activities WHERE id = $1`, assignedActivityID,
	).Scan(&assignedTo, &description))
	assert.Nil(t, assignedTo, "assigned_to must be cleared for activities assigned to the subject")
	assert.Nil(t, description, "description must be cleared for activities assigned to the subject")

	// The created-only activity keeps its assignment but loses the description.
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT assigned_to, description FROM activities WHERE id = $1`, createdActivityID,
	).Scan(&assignedTo, &description))
	require.NotNil(t, assignedTo, "an activity assigned to somebody else must keep its assignee")
	assert.Equal(t, colleagueID, *assignedTo)
	assert.Nil(t, description, "personal description must be cleared for activities created by the subject")

	// An unrelated activity is not touched at all.
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT assigned_to, description FROM activities WHERE id = $1`, foreignActivityID,
	).Scan(&assignedTo, &description))
	require.NotNil(t, assignedTo)
	assert.Equal(t, colleagueID, *assignedTo)
	require.NotNil(t, description, "an activity of another user must survive the erasure untouched")
	assert.Equal(t, "Fremde Beschreibung", *description)

	// Contact and company are deliberately retained: created_by is a NOT NULL FK
	// and the user row itself is anonymized by AuthErasureHandler.
	var contactRows, companyRows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contacts WHERE id = $1`, contactID).Scan(&contactRows))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM companies WHERE id = $1`, companyID).Scan(&companyRows))
	assert.Equal(t, 1, contactRows, "contacts are counted, not deleted")
	assert.Equal(t, 1, companyRows, "companies are counted, not deleted")
}

func TestCRMErasureHandler_ExecuteErasure_IgnoresAction(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Erasure Action Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "crm-erasure-action")
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Erika",
		"last_name":  "Musterfrau",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	activityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id":     tenantOwn,
		"activity_type": "call",
		"subject":       "Rueckruf Angebot",
		"description":   "Angebot 2026-114 nachfassen",
		"assigned_to":   userID,
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", activityID)

	h := NewCRMErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	// ErasureDelete does NOT delete: the handler ignores its action parameter.
	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureDelete)
	require.NoError(t, err)
	assert.Equal(t, 2, affected, "one activity counted once + one contact")

	var contactRows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contacts WHERE id = $1`, contactID).Scan(&contactRows))
	assert.Equal(t, 1, contactRows, "ErasureDelete must not remove the contact — the handler anonymizes regardless of action")

	var description *string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT description FROM activities WHERE id = $1`, activityID).Scan(&description))
	assert.Nil(t, description, "the anonymizing UPDATE runs for ErasureDelete just as for ErasureAnonymize")
}

func TestCRMErasureHandler_DeadPool(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	pool.Close()

	h := NewCRMErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	userID := uuid.New()

	// Preview swallows the query error and reports an empty, successful preview.
	preview, err := h.PreviewErasure(ctx, userID)
	require.NoError(t, err, "PreviewErasure reports 0 records instead of surfacing a DB error")
	require.NotNil(t, preview)
	assert.Equal(t, "crm", preview.ModuleName)
	assert.Equal(t, 0, preview.RecordCount)
	assert.Equal(t, string(ErasureAnonymize), preview.Action)

	// Execute does surface it.
	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.Error(t, err, "a dead pool must surface as an error, not as a silent no-op erasure")
	assert.Equal(t, 0, affected)
	assert.True(t, strings.HasPrefix(err.Error(), "crm erasure:"), "error must be wrapped with the module prefix, got %q", err.Error())
}

// ---------------------------------------------------------------------------
// ChatErasureHandler
// ---------------------------------------------------------------------------

func TestChatErasureHandler_PreviewErasure_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Chat Erasure Preview Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Chat Erasure Preview Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := seedExportUser(t, pool, tenantOwn, "chat-erasure-preview")
	defer testutil.CleanupRow(t, pool, "users", userID)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "projekt-beta",
		"created_by": userID,
	})
	// Deleting the channel cascades into channel_memberships and messages.
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	messageID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantOwn,
		"channel_id": channelID,
		"content":    "Angebot ist raus.",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "messages", messageID)

	seedChannelMembership(t, pool, tenantOwn, channelID, userID, "owner")

	h := NewChatErasureHandler(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	preview, err := h.PreviewErasure(ctxOwn, userID)
	require.NoError(t, err)
	require.NotNil(t, preview)
	assert.Equal(t, "chat", preview.ModuleName)
	assert.Equal(t, string(ErasureAnonymize), preview.Action)
	assert.Equal(t, 2, preview.RecordCount, "one message + one channel membership")

	// Preview must not write.
	var content string
	var isDeleted bool
	require.NoError(t, pool.QueryRow(ctxOwn,
		`SELECT content, is_deleted FROM messages WHERE id = $1`, messageID,
	).Scan(&content, &isDeleted))
	assert.Equal(t, "Angebot ist raus.", content, "preview must not anonymize the message")
	assert.False(t, isDeleted, "preview must not flag the message as deleted")

	var membershipRows int
	require.NoError(t, pool.QueryRow(ctxOwn,
		`SELECT COUNT(*) FROM channel_memberships WHERE user_id = $1`, userID,
	).Scan(&membershipRows))
	assert.Equal(t, 1, membershipRows, "preview must not remove the membership")

	previewOther, err := h.PreviewErasure(ctxOther, userID)
	require.NoError(t, err)
	assert.Equal(t, 0, previewOther.RecordCount, "a foreign tenant must not see any chat record of this user")

	previewUnknown, err := h.PreviewErasure(ctxOwn, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 0, previewUnknown.RecordCount, "an unknown user has nothing to erase")
}

func TestChatErasureHandler_ExecuteErasure_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Chat Erasure Execute Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "chat-erasure-exec")
	defer testutil.CleanupRow(t, pool, "users", userID)
	colleagueID := seedExportUser(t, pool, tenantOwn, "chat-erasure-colleague")
	defer testutil.CleanupRow(t, pool, "users", colleagueID)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "projekt-gamma",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	messageID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantOwn,
		"channel_id": channelID,
		"content":    "Angebot ist raus.",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "messages", messageID)

	// A colleague's message in the same channel must survive verbatim.
	foreignMessageID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantOwn,
		"channel_id": channelID,
		"content":    "Fremde Nachricht.",
		"created_by": colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "messages", foreignMessageID)

	seedChannelMembership(t, pool, tenantOwn, channelID, userID, "owner")
	seedChannelMembership(t, pool, tenantOwn, channelID, colleagueID, "member")

	h := NewChatErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 2, affected, "one message anonymized + one membership deleted")

	// Message skeleton stays, content is replaced by the bracketed label.
	var content string
	var isDeleted bool
	var editedAt *time.Time
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT content, is_deleted, edited_at FROM messages WHERE id = $1`, messageID,
	).Scan(&content, &isDeleted, &editedAt))
	assert.Equal(t, "["+erasureLabel+"]", content, "message content must be replaced by the bracketed anonymized label")
	assert.True(t, isDeleted, "the anonymized message must be flagged as deleted")
	assert.NotNil(t, editedAt, "edited_at must be stamped by the erasure")

	// The colleague's message is untouched.
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT content, is_deleted FROM messages WHERE id = $1`, foreignMessageID,
	).Scan(&content, &isDeleted))
	assert.Equal(t, "Fremde Nachricht.", content, "a message of another author must survive the erasure untouched")
	assert.False(t, isDeleted)

	// Only the subject's membership is gone.
	var ownMemberships, colleagueMemberships int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM channel_memberships WHERE user_id = $1`, userID).Scan(&ownMemberships))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM channel_memberships WHERE user_id = $1`, colleagueID).Scan(&colleagueMemberships))
	assert.Equal(t, 0, ownMemberships, "the subject's channel membership must be deleted")
	assert.Equal(t, 1, colleagueMemberships, "another member must keep their membership")
}

func TestChatErasureHandler_DeadPool(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	pool.Close()

	h := NewChatErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	userID := uuid.New()

	preview, err := h.PreviewErasure(ctx, userID)
	require.NoError(t, err, "PreviewErasure reports 0 records instead of surfacing a DB error")
	require.NotNil(t, preview)
	assert.Equal(t, "chat", preview.ModuleName)
	assert.Equal(t, 0, preview.RecordCount)
	assert.Equal(t, string(ErasureAnonymize), preview.Action)

	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.Error(t, err, "a dead pool must surface as an error, not as a silent no-op erasure")
	assert.Equal(t, 0, affected)
	assert.True(t, strings.HasPrefix(err.Error(), "chat erasure:"), "error must be wrapped with the module prefix, got %q", err.Error())
}
