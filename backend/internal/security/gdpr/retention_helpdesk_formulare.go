package gdpr

// HelpdeskTicketRetentionHandler and FormSubmissionRetentionHandler — fourth
// and fifth handlers on the retention engine from A10 (retention.go), both
// covering surfaces where a person's free text accumulates and outlives the
// purpose it was collected for: closed helpdesk tickets (with their
// messages) and public form submissions.
//
// ticket_messages cascades on ticket delete (migration 000077:
// `ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE`), so a
// hard delete of a closed ticket takes its message trail with it for free.
// Anonymize does not get that for free — it never removes the row, so this
// handler anonymizes ticket_messages.body explicitly, for every message under
// the ticket regardless of the `internal` flag. That flag governs whether a
// DSAR export discloses a staff-only note to the data subject (A6,
// dsar_search.go) — it says nothing about whether the note may keep existing
// past the retention period, which is the question this file answers.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ---------------------------------------------------------------------------
// HelpdeskTicketRetentionHandler
// ---------------------------------------------------------------------------

// HelpdeskTicketRetentionHandler is the helpdesk_tickets entry in the
// retention registry. Scope is CLOSED tickets only — an open or pending
// ticket has not finished its purpose yet, no matter how old it is.
type HelpdeskTicketRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewHelpdeskTicketRetentionHandler wires the handler.
func NewHelpdeskTicketRetentionHandler(pool *pgxpool.Pool) *HelpdeskTicketRetentionHandler {
	return &HelpdeskTicketRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for tickets.
func (h *HelpdeskTicketRetentionHandler) ResourceType() string { return "helpdesk_tickets" }

// Table names the governed table for the admin view.
func (h *HelpdeskTicketRetentionHandler) Table() string { return "tickets" }

// DateColumn discloses the retention clock. resolved_at is the closure
// timestamp: CloseTicket (service.go) is the only path that ever sets
// status='closed', and it always stamps resolved_at in the same call —
// UpdateTicket's generic field patch explicitly refuses to set status to
// closed (service.go:294) so that side effect can never be skipped. For the
// anonymize action, updated_at must also be past the cutoff: Apply bumps it
// on anonymize, the same idempotency technique DialerCallRetentionHandler
// uses for updated_at on dialer_call_sessions.
func (h *HelpdeskTicketRetentionHandler) DateColumn() string {
	return "resolved_at (nur status=closed; bei anonymize zusaetzlich updated_at)"
}

// SupportsAction reports that closed tickets can be both hard-deleted and
// anonymized.
func (h *HelpdeskTicketRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete || action == models.RetentionActionAnonymize
}

// Plan lists closed tickets past their retention cutoff. Never modifies data.
func (h *HelpdeskTicketRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, action string) (*RetentionPlan, error) {
	query := `SELECT id FROM tickets WHERE tenant_id = $1 AND status = 'closed' AND resolved_at < $2`
	if action == models.RetentionActionAnonymize {
		query += ` AND updated_at < $2`
	}
	query += ` ORDER BY resolved_at ASC`

	rows, err := h.pool.Query(ctx, query, tenantID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("helpdesk ticket retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("helpdesk ticket retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("helpdesk ticket retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids}, nil
}

// Apply deletes or anonymizes the given tickets.
//
// Delete relies on the ON DELETE CASCADE from ticket_messages to tickets
// (migration 000077) to take the message trail with it.
//
// Anonymize clears the ticket's two free-text fields (subject, description)
// and the external-requester identity fields, then does the same for every
// message under the ticket — a closed ticket with its subject cleared but
// its messages intact would still carry the same personal data one layer
// down. requester_email is replaced with a bracketed label rather than
// cleared, because chk_tickets_requester_identity (migration 000291) forbids
// an empty requester_email for an external requester. An internal
// requester's requester_id is left untouched: the user account behind it is
// a person handled by its own retention topic, not this ticket's.
func (h *HelpdeskTicketRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, anonymizedLabel string) (int, error) {
	switch action {
	case models.RetentionActionDelete:
		tag, err := h.pool.Exec(ctx,
			`DELETE FROM tickets WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids,
		)
		if err != nil {
			return 0, fmt.Errorf("helpdesk ticket retention: delete: %w", err)
		}
		return int(tag.RowsAffected()), nil
	case models.RetentionActionAnonymize:
		tag, err := h.pool.Exec(ctx,
			`UPDATE tickets
			    SET subject = '[' || $3 || ']',
			        description = '[' || $3 || ']',
			        requester_name = NULL,
			        requester_email = CASE WHEN requester_is_external THEN '[' || $3 || ']' ELSE requester_email END,
			        updated_at = NOW()
			  WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids, anonymizedLabel,
		)
		if err != nil {
			return 0, fmt.Errorf("helpdesk ticket retention: anonymize: %w", err)
		}
		affected := int(tag.RowsAffected())

		if _, err := h.pool.Exec(ctx,
			`UPDATE ticket_messages SET body = '[' || $3 || ']'
			  WHERE tenant_id = $1 AND ticket_id = ANY($2)`,
			tenantID, ids, anonymizedLabel,
		); err != nil {
			return affected, fmt.Errorf("helpdesk ticket retention: anonymize messages: %w", err)
		}
		return affected, nil
	default:
		return 0, fmt.Errorf("helpdesk ticket retention: unsupported action %q", action)
	}
}

// ---------------------------------------------------------------------------
// FormSubmissionRetentionHandler
// ---------------------------------------------------------------------------

// FormSubmissionRetentionHandler is the form_submissions entry in the
// retention registry.
type FormSubmissionRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewFormSubmissionRetentionHandler wires the handler.
func NewFormSubmissionRetentionHandler(pool *pgxpool.Pool) *FormSubmissionRetentionHandler {
	return &FormSubmissionRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for form
// submissions.
func (h *FormSubmissionRetentionHandler) ResourceType() string { return "form_submissions" }

// Table names the governed table for the admin view.
func (h *FormSubmissionRetentionHandler) Table() string { return "form_submissions" }

// DateColumn discloses the retention clock: submitted_at.
func (h *FormSubmissionRetentionHandler) DateColumn() string { return "submitted_at" }

// SupportsAction reports that form submissions can be both hard-deleted and
// anonymized.
func (h *FormSubmissionRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete || action == models.RetentionActionAnonymize
}

// Plan lists submissions past their retention cutoff. form_schema_id is
// irrelevant here on purpose: an anonymous public-form submission collects
// exactly the same personal data (answers, submitted_by, ip_address) as one
// that later got matched to a contact, and the retention clock does not
// treat the two differently. For anonymize, submissions already anonymized
// are excluded via anonymized_at (migration 000317) — submitted_at itself
// never changes, so unlike the other handlers in this run there is no
// existing timestamp column whose bump would give idempotency for free.
func (h *FormSubmissionRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, action string) (*RetentionPlan, error) {
	query := `SELECT id FROM form_submissions WHERE tenant_id = $1 AND submitted_at < $2`
	if action == models.RetentionActionAnonymize {
		query += ` AND anonymized_at IS NULL`
	}
	query += ` ORDER BY submitted_at ASC`

	rows, err := h.pool.Query(ctx, query, tenantID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("form submission retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("form submission retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("form submission retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids}, nil
}

// Apply deletes or anonymizes the given submissions. Anonymize clears the
// answer payload and the two identifying fields (submitted_by, ip_address)
// and stamps anonymized_at, which is what Plan checks on the next run.
func (h *FormSubmissionRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, anonymizedLabel string) (int, error) {
	switch action {
	case models.RetentionActionDelete:
		tag, err := h.pool.Exec(ctx,
			`DELETE FROM form_submissions WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids,
		)
		if err != nil {
			return 0, fmt.Errorf("form submission retention: delete: %w", err)
		}
		return int(tag.RowsAffected()), nil
	case models.RetentionActionAnonymize:
		tag, err := h.pool.Exec(ctx,
			`UPDATE form_submissions
			    SET answers = '{}'::jsonb, submitted_by = '[' || $3 || ']', ip_address = NULL,
			        anonymized_at = NOW()
			  WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids, anonymizedLabel,
		)
		if err != nil {
			return 0, fmt.Errorf("form submission retention: anonymize: %w", err)
		}
		return int(tag.RowsAffected()), nil
	default:
		return 0, fmt.Errorf("form submission retention: unsupported action %q", action)
	}
}
