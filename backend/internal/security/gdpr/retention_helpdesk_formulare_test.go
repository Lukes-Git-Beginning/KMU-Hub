package gdpr

// Coverage for HelpdeskTicketRetentionHandler and FormSubmissionRetentionHandler
// (Lauf 10, fourth and fifth handlers on the retention engine from A10). The
// cases that matter: only CLOSED tickets past the cutoff are Due (an open
// ticket never is, no matter how old), deleting a closed ticket cascades its
// messages, anonymizing one clears both the ticket and its messages while
// keeping chk_tickets_requester_identity satisfied for an external requester,
// and both anonymize paths are idempotent -- tickets via updated_at, form
// submissions via the dedicated anonymized_at column (migration 000317).

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
// HelpdeskTicketRetentionHandler
// ---------------------------------------------------------------------------

func TestHelpdeskTicketRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewHelpdeskTicketRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.True(t, handler.SupportsAction(models.RetentionActionAnonymize))
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "helpdesk_tickets", handler.ResourceType())
	assert.Equal(t, "tickets", handler.Table())
}

func TestHelpdeskTicketRetentionHandler_PlanOnlyMatchesClosedPastCutoff(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	agentID := seedExportUser(t, pool, tenantID, "helpdesk-retention-plan-agent")
	defer testutil.CleanupRow(t, pool, "users", agentID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	fresh := time.Now().UTC().AddDate(0, 0, -2)

	closedOld := seedTicket(t, pool, tenantID, agentID, "closed", &old)
	defer testutil.CleanupRow(t, pool, "tickets", closedOld)

	closedFresh := seedTicket(t, pool, tenantID, agentID, "closed", &fresh)
	defer testutil.CleanupRow(t, pool, "tickets", closedFresh)

	// Open ticket, old creation date but no resolved_at -- must never be Due,
	// closed status is the gate, not age.
	openOld := seedTicket(t, pool, tenantID, agentID, "open", nil)
	defer testutil.CleanupRow(t, pool, "tickets", openOld)
	_, err := pool.Exec(context.Background(), `UPDATE tickets SET created_at = $1, updated_at = $1 WHERE id = $2`, old, openOld)
	require.NoError(t, err)

	handler := NewHelpdeskTicketRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{closedOld}, plan.Due)
}

func TestHelpdeskTicketRetentionHandler_ApplyDeleteCascadesMessages(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk Retention Apply Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	agentID := seedExportUser(t, pool, tenantID, "helpdesk-retention-apply-del-agent")
	defer testutil.CleanupRow(t, pool, "users", agentID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	ticketID := seedTicket(t, pool, tenantID, agentID, "closed", &old)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	msgID := testutil.SeedRow(t, pool, "ticket_messages", map[string]any{
		"tenant_id": tenantID,
		"ticket_id": ticketID,
		"author_id": agentID,
		"body":      "Danke, Fall erledigt.",
		"internal":  false,
	})

	handler := NewHelpdeskTicketRetentionHandler(pool)
	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{ticketID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var ticketCount, messageCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM tickets WHERE id = $1`, ticketID).Scan(&ticketCount))
	assert.Zero(t, ticketCount, "deleted ticket must be gone")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM ticket_messages WHERE id = $1`, msgID).Scan(&messageCount))
	assert.Zero(t, messageCount, "cascade must take the message trail with it")
}

func TestHelpdeskTicketRetentionHandler_ApplyAnonymizeClearsTicketAndMessagesAndIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk Retention Apply Anonymize Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	agentID := seedExportUser(t, pool, tenantID, "helpdesk-retention-apply-anon-agent")
	defer testutil.CleanupRow(t, pool, "users", agentID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	ticketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"tenant_id":             tenantID,
		"subject":               "Vertragsdaten falsch",
		"description":           "Meine Adresse stimmt nicht mehr.",
		"status":                "closed",
		"requester_id":          nil,
		"requester_is_external": true,
		"requester_email":       "kunde@example.com",
		"requester_name":        "Erika Musterfrau",
		"resolved_at":           old,
	})
	defer testutil.CleanupRow(t, pool, "tickets", ticketID)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	msgID := testutil.SeedRow(t, pool, "ticket_messages", map[string]any{
		"tenant_id": tenantID,
		"ticket_id": ticketID,
		"author_id": agentID,
		"body":      "Bitte um Aenderung meiner Adresse.",
		"internal":  false,
	})
	internalMsgID := testutil.SeedRow(t, pool, "ticket_messages", map[string]any{
		"tenant_id": tenantID,
		"ticket_id": ticketID,
		"author_id": agentID,
		"body":      "Interne Notiz ueber die Kundin.",
		"internal":  true,
	})

	handler := NewHelpdeskTicketRetentionHandler(pool)
	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{ticketID}, models.RetentionActionAnonymize, "Geloeschtes Ticket 1")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var subject, description, requesterEmail string
	var requesterName *string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT subject, description, requester_email, requester_name FROM tickets WHERE id = $1`, ticketID,
	).Scan(&subject, &description, &requesterEmail, &requesterName))
	assert.Equal(t, "[Geloeschtes Ticket 1]", subject)
	assert.Equal(t, "[Geloeschtes Ticket 1]", description)
	assert.Equal(t, "[Geloeschtes Ticket 1]", requesterEmail, "must stay non-empty, chk_tickets_requester_identity forbids an empty external email")
	assert.Nil(t, requesterName)

	var customerBody, internalBody string
	require.NoError(t, pool.QueryRow(ctx, `SELECT body FROM ticket_messages WHERE id = $1`, msgID).Scan(&customerBody))
	require.NoError(t, pool.QueryRow(ctx, `SELECT body FROM ticket_messages WHERE id = $1`, internalMsgID).Scan(&internalBody))
	assert.Equal(t, "[Geloeschtes Ticket 1]", customerBody, "retention anonymizes personal data regardless of the disclosure-only internal flag")
	assert.Equal(t, "[Geloeschtes Ticket 1]", internalBody)

	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionAnonymize)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, ticketID, "anonymize must bump updated_at, taking the row out of the next Plan")
}

// seedTicket seeds a minimal ticket with an internal requester (agentID) and
// the given status. resolvedAt is nil for open tickets, a fixed timestamp for
// closed ones -- CloseTicket always sets both together, so a hand-seeded
// closed ticket without resolved_at would not exist in production data.
func seedTicket(t *testing.T, pool *pgxpool.Pool, tenantID, agentID uuid.UUID, status string, resolvedAt *time.Time) uuid.UUID {
	t.Helper()
	cols := map[string]any{
		"tenant_id":    tenantID,
		"subject":      "Testfall " + uuid.New().String(),
		"status":       status,
		"requester_id": agentID,
	}
	if resolvedAt != nil {
		cols["resolved_at"] = *resolvedAt
		cols["updated_at"] = *resolvedAt
	}
	return testutil.SeedRow(t, pool, "tickets", cols)
}

// ---------------------------------------------------------------------------
// FormSubmissionRetentionHandler
// ---------------------------------------------------------------------------

func TestFormSubmissionRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewFormSubmissionRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.True(t, handler.SupportsAction(models.RetentionActionAnonymize))
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "form_submissions", handler.ResourceType())
	assert.Equal(t, "form_submissions", handler.Table())
}

func TestFormSubmissionRetentionHandler_PlanIncludesSubmissionsWithoutContactLink(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Form Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	fresh := time.Now().UTC().AddDate(0, 0, -2)

	// No form_schema_id, no requester link of any kind -- an anonymous public
	// form submission. Must still be Due, retention does not care whether it
	// was ever matched to a contact.
	oldSubmission := seedFormSubmission(t, pool, tenantID, old)
	defer testutil.CleanupRow(t, pool, "form_submissions", oldSubmission)

	freshSubmission := seedFormSubmission(t, pool, tenantID, fresh)
	defer testutil.CleanupRow(t, pool, "form_submissions", freshSubmission)

	handler := NewFormSubmissionRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{oldSubmission}, plan.Due)
}

func TestFormSubmissionRetentionHandler_ApplyAnonymizeIsIdempotentViaDedicatedColumn(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Form Retention Apply Anonymize Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	submissionID := seedFormSubmission(t, pool, tenantID, old)
	defer testutil.CleanupRow(t, pool, "form_submissions", submissionID)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewFormSubmissionRetentionHandler(pool)

	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{submissionID}, models.RetentionActionAnonymize, "Geloeschte Einreichung 1")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var answers []byte
	var submittedBy *string
	var ipAddress *string
	var anonymizedAt *time.Time
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT answers, submitted_by, ip_address::text, anonymized_at FROM form_submissions WHERE id = $1`, submissionID,
	).Scan(&answers, &submittedBy, &ipAddress, &anonymizedAt))
	assert.JSONEq(t, `{}`, string(answers))
	require.NotNil(t, submittedBy)
	assert.Equal(t, "[Geloeschte Einreichung 1]", *submittedBy)
	assert.Nil(t, ipAddress)
	assert.NotNil(t, anonymizedAt)

	// submitted_at never changes, so the cutoff alone would find this row
	// again on every future run without the anonymized_at gate.
	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionAnonymize)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, submissionID, "anonymized_at must take the row out of the next anonymize Plan")

	// A delete run must still find it -- anonymized_at only gates anonymize.
	planDelete, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Contains(t, planDelete.Due, submissionID)
}

func seedFormSubmission(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, submittedAt time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "form_submissions", map[string]any{
		"tenant_id":    tenantID,
		"answers":      `{"email":"person@example.com","message":"Bitte um Rueckruf."}`,
		"submitted_by": "person@example.com",
		"submitted_at": submittedAt,
	})
}
