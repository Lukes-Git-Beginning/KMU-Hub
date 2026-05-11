package dialer

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// ─────────────────────────────────────────────────────────────────────────────
// dialer_campaigns
// ─────────────────────────────────────────────────────────────────────────────

func TestRLS_DialerCampaigns_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-camp-test-a@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Campaign TenantA",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "dialer_campaigns", campID, 0)
}

func TestRLS_DialerCampaigns_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-camp-own-a@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Own Campaign TenantA",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "dialer_campaigns", campID, 1)
}

func TestRLS_DialerCampaigns_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-camp-sys@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "System Campaign",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	sysCtx := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, sysCtx, "dialer_campaigns", campID, 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// dialer_campaign_contacts  (FK on dialer_campaigns + contacts)
// ─────────────────────────────────────────────────────────────────────────────

func TestRLS_DialerCampaignContacts_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-cc-b@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Campaign for CC test",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "TestCC",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ccID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", ccID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "dialer_campaign_contacts", ccID, 0)
}

func TestRLS_DialerCampaignContacts_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-cc-own@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Camp CC Own",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "CCOwn",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ccID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", ccID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "dialer_campaign_contacts", ccID, 1)
}

func TestRLS_DialerCampaignContacts_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-cc-sys@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Camp CC Sys",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "CCSys",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ccID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", ccID)

	sysCtx := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, sysCtx, "dialer_campaign_contacts", ccID, 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// dialer_call_sessions  (FK on dialer_campaign_contacts)
// ─────────────────────────────────────────────────────────────────────────────

func TestRLS_DialerCallSessions_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-cs-b@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Camp CS B",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "CSB",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ccID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", ccID)

	sessionID := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"id":                   uuid.New(),
		"tenant_id":            testutil.TenantA,
		"campaign_contact_id":  ccID,
		"agent_id":             userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", sessionID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "dialer_call_sessions", sessionID, 0)
}

func TestRLS_DialerCallSessions_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-cs-own@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Camp CS Own",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "CSOwnContact",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ccID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", ccID)

	sessionID := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"id":                  uuid.New(),
		"tenant_id":           testutil.TenantA,
		"campaign_contact_id": ccID,
		"agent_id":            userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", sessionID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "dialer_call_sessions", sessionID, 1)
}

func TestRLS_DialerCallSessions_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-cs-sys@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Camp CS Sys",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "CSSysContact",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ccID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", ccID)

	sessionID := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"id":                  uuid.New(),
		"tenant_id":           testutil.TenantA,
		"campaign_contact_id": ccID,
		"agent_id":            userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", sessionID)

	sysCtx := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, sysCtx, "dialer_call_sessions", sessionID, 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// dialer_call_outcomes
// ─────────────────────────────────────────────────────────────────────────────

func TestRLS_DialerCallOutcomes_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	outcomeID := testutil.SeedRow(t, pool, "dialer_call_outcomes", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"label":     "Interested",
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_outcomes", outcomeID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "dialer_call_outcomes", outcomeID, 0)
}

func TestRLS_DialerCallOutcomes_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	outcomeID := testutil.SeedRow(t, pool, "dialer_call_outcomes", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"label":     "Not Interested",
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_outcomes", outcomeID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "dialer_call_outcomes", outcomeID, 1)
}

func TestRLS_DialerCallOutcomes_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	outcomeID := testutil.SeedRow(t, pool, "dialer_call_outcomes", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"label":     "Callback",
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_outcomes", outcomeID)

	sysCtx := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, sysCtx, "dialer_call_outcomes", outcomeID, 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// dialer_call_events  (FK: dialer_call_session_id → dialer_call_sessions)
// ─────────────────────────────────────────────────────────────────────────────

func TestRLS_DialerCallEvents_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-ce-b@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Camp CE B",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "CEB",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ccID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", ccID)

	sessionID := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"id":                  uuid.New(),
		"tenant_id":           testutil.TenantA,
		"campaign_contact_id": ccID,
		"agent_id":            userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", sessionID)

	eventID := testutil.SeedRow(t, pool, "dialer_call_events", map[string]any{
		"id":                      uuid.New(),
		"tenant_id":               testutil.TenantA,
		"dialer_call_session_id":  sessionID,
		"event_type":              "call_started",
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_events", eventID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "dialer_call_events", eventID, 0)
}

func TestRLS_DialerCallEvents_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-ce-own@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Camp CE Own",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "CEOwnContact",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ccID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", ccID)

	sessionID := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"id":                  uuid.New(),
		"tenant_id":           testutil.TenantA,
		"campaign_contact_id": ccID,
		"agent_id":            userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", sessionID)

	eventID := testutil.SeedRow(t, pool, "dialer_call_events", map[string]any{
		"id":                     uuid.New(),
		"tenant_id":              testutil.TenantA,
		"dialer_call_session_id": sessionID,
		"event_type":             "call_ended",
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_events", eventID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "dialer_call_events", eventID, 1)
}

func TestRLS_DialerCallEvents_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-ce-sys@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Camp CE Sys",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "CESysContact",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ccID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", ccID)

	sessionID := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"id":                  uuid.New(),
		"tenant_id":           testutil.TenantA,
		"campaign_contact_id": ccID,
		"agent_id":            userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", sessionID)

	eventID := testutil.SeedRow(t, pool, "dialer_call_events", map[string]any{
		"id":                     uuid.New(),
		"tenant_id":              testutil.TenantA,
		"dialer_call_session_id": sessionID,
		"event_type":             "wrap_up_started",
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_events", eventID)

	sysCtx := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, sysCtx, "dialer_call_events", eventID, 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// dialer_agent_status_log  (FK: user_id → users)
// ─────────────────────────────────────────────────────────────────────────────

func TestRLS_DialerAgentStatusLog_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-asl-b@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	logID := testutil.SeedRow(t, pool, "dialer_agent_status_log", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"user_id":   userID,
		"status":    "available",
	})
	defer testutil.CleanupRow(t, pool, "dialer_agent_status_log", logID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "dialer_agent_status_log", logID, 0)
}

func TestRLS_DialerAgentStatusLog_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-asl-own@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	logID := testutil.SeedRow(t, pool, "dialer_agent_status_log", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"user_id":   userID,
		"status":    "on_call",
	})
	defer testutil.CleanupRow(t, pool, "dialer_agent_status_log", logID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "dialer_agent_status_log", logID, 1)
}

func TestRLS_DialerAgentStatusLog_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "dialer-asl-sys@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	logID := testutil.SeedRow(t, pool, "dialer_agent_status_log", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"user_id":   userID,
		"status":    "offline",
	})
	defer testutil.CleanupRow(t, pool, "dialer_agent_status_log", logID)

	sysCtx := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, sysCtx, "dialer_agent_status_log", logID, 1)
}
