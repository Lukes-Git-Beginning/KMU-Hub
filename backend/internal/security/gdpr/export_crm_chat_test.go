package gdpr

// Integration coverage for CRMExportHandler.ExportUserData and
// ChatExportHandler.ExportUserData — the two largest of the six export
// handlers that had no test at all (only AuthExportHandler was covered).
//
// Both handlers scope their queries through `JOIN users u ON u.id = $1`
// plus `<table>.tenant_id = u.tenant_id`, so the tenant boundary is enforced
// by RLS on `users` rather than by an explicit predicate on the payload
// table. These tests pin exactly that: under a foreign tenant context the
// join resolves to zero rows and the export comes back empty instead of
// leaking rows.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// ---------------------------------------------------------------------------
// Payload shapes (the handlers marshal anonymous structs, so the assertions
// need their own mirror of the JSON contract).
// ---------------------------------------------------------------------------

type crmExportPayload struct {
	CreatedContacts []struct {
		ID        uuid.UUID `json:"id"`
		FirstName string    `json:"first_name"`
		LastName  string    `json:"last_name"`
		Email     *string   `json:"email"`
		Position  *string   `json:"position"`
	} `json:"created_contacts"`
	CreatedCompanies []struct {
		ID       uuid.UUID `json:"id"`
		Name     string    `json:"name"`
		Industry *string   `json:"industry"`
	} `json:"created_companies"`
	AssignedActivities []struct {
		ID           uuid.UUID  `json:"id"`
		ActivityType string     `json:"activity_type"`
		Subject      string     `json:"subject"`
		DueDate      *time.Time `json:"due_date"`
	} `json:"assigned_activities"`
}

type chatExportPayload struct {
	Messages []struct {
		ID        uuid.UUID `json:"id"`
		ChannelID uuid.UUID `json:"channel_id"`
		Content   string    `json:"content"`
		IsDeleted bool      `json:"is_deleted"`
	} `json:"messages"`
	Memberships []struct {
		ChannelID   uuid.UUID  `json:"channel_id"`
		ChannelName string     `json:"channel_name"`
		Role        string     `json:"role"`
		JoinedAt    time.Time  `json:"joined_at"`
		LastReadAt  *time.Time `json:"last_read_at"`
	} `json:"memberships"`
}

// seedExportUser creates a user in the given tenant with a collision-free email.
func seedExportUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, prefix string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("%s-%s@export.invalid", prefix, uuid.New()),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Export",
		"last_name":     "Subject",
	})
}

// seedChannelMembership inserts into channel_memberships, which has a composite
// primary key (channel_id, user_id) and no id column — testutil.SeedRow cannot
// be used because it relies on RETURNING id.
func seedChannelMembership(t *testing.T, pool *pgxpool.Pool, tenantID, channelID, userID uuid.UUID, role string) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO channel_memberships (channel_id, user_id, role, tenant_id, last_read_at)
		 VALUES ($1, $2, $3::channel_role, $4, now())`,
		channelID, userID, role, tenantID,
	)
	if err != nil {
		t.Fatalf("seed channel_memberships: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CRMExportHandler
// ---------------------------------------------------------------------------

func TestCRMExportHandler_ExportUserData_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Export Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Export Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := seedExportUser(t, pool, tenantOwn, "crm-export")
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Erika",
		"last_name":  "Musterfrau",
		"email":      fmt.Sprintf("erika-%s@export.invalid", uuid.New()),
		"position":   "Einkaufsleitung",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	companyID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Musterfirma GmbH",
		"industry":   "Maschinenbau",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyID)

	dueDate := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	activityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id":     tenantOwn,
		"activity_type": "call",
		"subject":       "Rueckruf Angebot",
		"description":   "Angebot 2026-114 nachfassen",
		"assigned_to":   userID,
		"created_by":    userID,
		"due_date":      dueDate,
	})
	defer testutil.CleanupRow(t, pool, "activities", activityID)

	h := NewCRMExportHandler(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	// --- own tenant: all three lists must carry the seeded row ---------------
	raw, err := h.ExportUserData(ctxOwn, userID)
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	var own crmExportPayload
	require.NoError(t, json.Unmarshal(raw, &own))

	require.Len(t, own.CreatedContacts, 1, "export must contain the contact created by the user")
	assert.Equal(t, contactID, own.CreatedContacts[0].ID)
	assert.Equal(t, "Erika", own.CreatedContacts[0].FirstName)
	assert.Equal(t, "Musterfrau", own.CreatedContacts[0].LastName)
	require.NotNil(t, own.CreatedContacts[0].Position)
	assert.Equal(t, "Einkaufsleitung", *own.CreatedContacts[0].Position)

	require.Len(t, own.CreatedCompanies, 1, "export must contain the company created by the user")
	assert.Equal(t, companyID, own.CreatedCompanies[0].ID)
	assert.Equal(t, "Musterfirma GmbH", own.CreatedCompanies[0].Name)

	require.Len(t, own.AssignedActivities, 1, "export must contain the activity assigned to the user")
	assert.Equal(t, activityID, own.AssignedActivities[0].ID)
	assert.Equal(t, "call", own.AssignedActivities[0].ActivityType, "activity_type must be rendered as text, not as an enum oid")
	assert.Equal(t, "Rueckruf Angebot", own.AssignedActivities[0].Subject)
	require.NotNil(t, own.AssignedActivities[0].DueDate, "nullable due_date must survive the scan when set")

	// --- foreign tenant: nothing may leak -----------------------------------
	rawOther, err := h.ExportUserData(ctxOther, userID)
	require.NoError(t, err, "cross-tenant export returns an empty payload, not an error")

	var other crmExportPayload
	require.NoError(t, json.Unmarshal(rawOther, &other))
	assert.Empty(t, other.CreatedContacts, "foreign tenant must not see the contact")
	assert.Empty(t, other.CreatedCompanies, "foreign tenant must not see the company")
	assert.Empty(t, other.AssignedActivities, "foreign tenant must not see the activity")
	assert.NotContains(t, string(rawOther), contactID.String(), "no contact id may appear in a cross-tenant export")
	assert.NotContains(t, string(rawOther), companyID.String(), "no company id may appear in a cross-tenant export")
	assert.NotContains(t, string(rawOther), activityID.String(), "no activity id may appear in a cross-tenant export")

	// --- unknown user -------------------------------------------------------
	// Divergence from AuthExportHandler, which returns an error for a user that
	// does not resolve: the CRM handler's `JOIN users u ON u.id = $1` simply
	// matches nothing, so the export succeeds with three empty lists. Pinned
	// here as the current contract; see JOURNAL Iteration 38.
	rawUnknown, err := h.ExportUserData(ctxOwn, uuid.New())
	require.NoError(t, err)
	var unknown crmExportPayload
	require.NoError(t, json.Unmarshal(rawUnknown, &unknown))
	assert.Empty(t, unknown.CreatedContacts)
	assert.Empty(t, unknown.CreatedCompanies)
	assert.Empty(t, unknown.AssignedActivities)
}

func TestCRMExportHandler_QueryError(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	// A closed pool makes the very first Query fail — covers the error branch
	// that a happy-path-only test never reaches.
	pool := testutil.PoolFromEnv(t)
	pool.Close()

	h := NewCRMExportHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)

	data, err := h.ExportUserData(ctx, uuid.New())
	require.Error(t, err, "a dead pool must surface as an error, not as an empty export")
	assert.Nil(t, data)
	assert.True(t, strings.HasPrefix(err.Error(), "crm export:"), "error must be wrapped with the module prefix, got %q", err.Error())
}

// ---------------------------------------------------------------------------
// ChatExportHandler
// ---------------------------------------------------------------------------

func TestChatExportHandler_ExportUserData_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Chat Export Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Chat Export Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := seedExportUser(t, pool, tenantOwn, "chat-export")
	defer testutil.CleanupRow(t, pool, "users", userID)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "projekt-alpha",
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

	// A message from a colleague in the same channel and tenant must stay out
	// of the export (Datensparsamkeit: only messages the subject authored).
	otherUserID := seedExportUser(t, pool, tenantOwn, "chat-export-other")
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	foreignMessageID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantOwn,
		"channel_id": channelID,
		"content":    "Fremde Nachricht.",
		"created_by": otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "messages", foreignMessageID)

	seedChannelMembership(t, pool, tenantOwn, channelID, userID, "owner")

	h := NewChatExportHandler(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	// --- own tenant ---------------------------------------------------------
	raw, err := h.ExportUserData(ctxOwn, userID)
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	var own chatExportPayload
	require.NoError(t, json.Unmarshal(raw, &own))

	require.Len(t, own.Messages, 1, "export must contain exactly the message authored by the user")
	assert.Equal(t, messageID, own.Messages[0].ID)
	assert.Equal(t, channelID, own.Messages[0].ChannelID)
	assert.Equal(t, "Angebot ist raus.", own.Messages[0].Content)
	assert.False(t, own.Messages[0].IsDeleted)
	assert.NotContains(t, string(raw), "Fremde Nachricht.", "messages of other authors must not be exported")

	require.Len(t, own.Memberships, 1, "export must contain the channel membership")
	assert.Equal(t, channelID, own.Memberships[0].ChannelID)
	assert.Equal(t, "projekt-alpha", own.Memberships[0].ChannelName, "membership must be joined against channels.name")
	assert.Equal(t, "owner", own.Memberships[0].Role)
	assert.False(t, own.Memberships[0].JoinedAt.IsZero())
	assert.NotNil(t, own.Memberships[0].LastReadAt, "nullable last_read_at must survive the scan when set")

	// --- foreign tenant -----------------------------------------------------
	rawOther, err := h.ExportUserData(ctxOther, userID)
	require.NoError(t, err, "cross-tenant export returns an empty payload, not an error")

	var other chatExportPayload
	require.NoError(t, json.Unmarshal(rawOther, &other))
	assert.Empty(t, other.Messages, "foreign tenant must not see the message")
	assert.Empty(t, other.Memberships, "foreign tenant must not see the membership")
	assert.NotContains(t, string(rawOther), "Angebot ist raus.", "no message content may appear in a cross-tenant export")
	assert.NotContains(t, string(rawOther), "projekt-alpha", "no channel name may appear in a cross-tenant export")

	// --- unknown user (same divergence as the CRM handler) ------------------
	rawUnknown, err := h.ExportUserData(ctxOwn, uuid.New())
	require.NoError(t, err)
	var unknown chatExportPayload
	require.NoError(t, json.Unmarshal(rawUnknown, &unknown))
	assert.Empty(t, unknown.Messages)
	assert.Empty(t, unknown.Memberships)
}

func TestChatExportHandler_QueryError(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	pool.Close()

	h := NewChatExportHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)

	data, err := h.ExportUserData(ctx, uuid.New())
	require.Error(t, err, "a dead pool must surface as an error, not as an empty export")
	assert.Nil(t, data)
	assert.True(t, strings.HasPrefix(err.Error(), "chat export:"), "error must be wrapped with the module prefix, got %q", err.Error())
}
