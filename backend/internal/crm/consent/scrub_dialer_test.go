package consent

// feat-scrub-dependent-pii-dialer-tables: ScrubDependentPII never touched
// dialer_call_sessions.notes/.next_action or dialer_campaign_contacts.notes --
// call notes that can name the contact verbatim. This pins the two new
// blocks through AnonymizeContact (the only path that can exercise them --
// see delete_scrub_dialer_restrict_db_test.go in the contact package for why
// the hard-delete path cannot).

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAnonymizeContact_ScrubsDialerCampaignAndCallSessionNotes(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Consent Dialer Scrub Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Consent Dialer Scrub Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("dialer-scrub-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	otherUserID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     otherTenantID,
		"email":         fmt.Sprintf("dialer-scrub-other-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Dialer",
		"last_name":  "ScrubTest",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	otherContactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  otherTenantID,
		"first_name": "Foreign",
		"last_name":  "ScrubTest",
		"created_by": otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", otherContactID)

	campaignID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"tenant_id":  tenantID,
		"name":       "Dialer Scrub Campaign",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campaignID)

	otherCampaignID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"tenant_id":  otherTenantID,
		"name":       "Foreign Dialer Campaign",
		"created_by": otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", otherCampaignID)

	campaignContactID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"tenant_id":   tenantID,
		"campaign_id": campaignID,
		"contact_id":  contactID,
		"position":    1,
		"notes":       "difficult to reach, calls back after 6pm",
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", campaignContactID)

	otherCampaignContactID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"tenant_id":   otherTenantID,
		"campaign_id": otherCampaignID,
		"contact_id":  otherContactID,
		"position":    1,
		"notes":       "unrelated foreign tenant note",
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", otherCampaignContactID)

	callSessionID := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"tenant_id":           tenantID,
		"campaign_contact_id": campaignContactID,
		"agent_id":            userID,
		"notes":               "customer asked for a callback next week",
		"next_action":         "callback Tuesday 10am",
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", callSessionID)

	otherCallSessionID := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"tenant_id":           otherTenantID,
		"campaign_contact_id": otherCampaignContactID,
		"agent_id":            otherUserID,
		"notes":               "unrelated foreign tenant call note",
		"next_action":         "unrelated foreign next action",
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", otherCallSessionID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)

	if err := repo.AnonymizeContact(ctxOwn, contactID, tenantID); err != nil {
		t.Fatalf("AnonymizeContact: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())

	var ccNotes *string
	if err := pool.QueryRow(sysCtx, "SELECT notes FROM dialer_campaign_contacts WHERE id = $1", campaignContactID).Scan(&ccNotes); err != nil {
		t.Fatalf("read back campaign contact: %v", err)
	}
	if ccNotes != nil {
		t.Fatalf("expected dialer_campaign_contacts.notes to be scrubbed, got %v", *ccNotes)
	}

	var csNotes, csNextAction *string
	if err := pool.QueryRow(sysCtx, "SELECT notes, next_action FROM dialer_call_sessions WHERE id = $1", callSessionID).Scan(&csNotes, &csNextAction); err != nil {
		t.Fatalf("read back call session: %v", err)
	}
	if csNotes != nil || csNextAction != nil {
		t.Fatalf("expected dialer_call_sessions.notes/.next_action to be scrubbed, got notes=%v next_action=%v", csNotes, csNextAction)
	}

	// Negative: own tenant scrubbed above (>0 rows changed), the foreign
	// tenant's rows for a different contact/campaign must survive untouched --
	// zero rows changed there, not "both zero".
	var otherCCNotes *string
	if err := pool.QueryRow(sysCtx, "SELECT notes FROM dialer_campaign_contacts WHERE id = $1", otherCampaignContactID).Scan(&otherCCNotes); err != nil {
		t.Fatalf("read back foreign campaign contact: %v", err)
	}
	if otherCCNotes == nil || *otherCCNotes != "unrelated foreign tenant note" {
		t.Fatalf("expected foreign tenant's dialer_campaign_contacts.notes to survive untouched, got %v", otherCCNotes)
	}

	var otherCSNotes, otherCSNextAction *string
	if err := pool.QueryRow(sysCtx, "SELECT notes, next_action FROM dialer_call_sessions WHERE id = $1", otherCallSessionID).Scan(&otherCSNotes, &otherCSNextAction); err != nil {
		t.Fatalf("read back foreign call session: %v", err)
	}
	if otherCSNotes == nil || *otherCSNotes != "unrelated foreign tenant call note" {
		t.Fatalf("expected foreign tenant's dialer_call_sessions.notes to survive untouched, got %v", otherCSNotes)
	}
	if otherCSNextAction == nil || *otherCSNextAction != "unrelated foreign next action" {
		t.Fatalf("expected foreign tenant's dialer_call_sessions.next_action to survive untouched, got %v", otherCSNextAction)
	}
}
