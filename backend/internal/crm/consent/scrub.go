package consent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AnonymizedRequesterEmail is the placeholder ScrubDependentPII writes into
// tickets.requester_email for an external requester whose contact was
// deleted or anonymized. chk_tickets_requester_identity (migration 000291)
// requires a non-empty value there whenever requester_id is NULL, so the
// column cannot simply be cleared like an internal requester's copy can.
const AnonymizedRequesterEmail = "geloescht@deleted.invalid"

// ScrubDependentPII removes personal data left behind in tables contacts
// itself does not own, once a contact is either hard-deleted or anonymized:
// activities notes, consent_records identifiers, and -- for tickets naming
// this contact as requester -- the requester's name and reply address. Runs
// inside the caller's transaction, so contact.PostgresRepository.Delete
// (hard-delete, no RESTRICT hit) and AnonymizeContact (RESTRICT hit) apply
// the identical cleanup instead of drifting apart.
//
// tickets.requester_name/-email are only replaced with a placeholder for
// external requesters (requester_is_external): for internal ones the
// identity lives on the user row (see models.Ticket.RequesterName godoc in
// the helpdesk package) and these two columns are unused fallback storage,
// safe to clear to NULL.
//
// Also scrubbed: dialer_campaign_contacts.notes (direct contact_id column)
// and dialer_call_sessions.notes/.next_action -- call notes and next-step
// text that can carry a caller's name verbatim. dialer_call_sessions has no
// contact_id of its own; it hangs off dialer_campaign_contacts via
// campaign_contact_id, so reaching it takes a subselect rather than a plain
// WHERE contact_id = $1. That join is why these two tables got their own
// unit instead of a one-line addition to the blocks below.
//
// Deliberately NOT scrubbed here:
//   - finance_invoices (customer_name/-address/-email/-ust_id_nr) -- §147
//     Abs. 3 AO requires 10 years retention for invoices
//     (gobdarchive/service.go:270), and sent invoices are GoBD-immutable
//     (locked_at). Scrubbing them would be a compliance violation, not a fix.
//   - deals.notes/name, meetings.title/description/agenda -- free text that
//     CAN name the contact but does not have to; a blind search/replace risks
//     corrupting unrelated content more than it protects. Accepted residual
//     risk, not a code gap.
func ScrubDependentPII(ctx context.Context, tx pgx.Tx, contactID, tenantID uuid.UUID) (int, error) {
	affected := 0

	res, err := tx.Exec(ctx,
		`UPDATE activities SET description = NULL, updated_at = NOW() WHERE contact_id = $1 AND tenant_id = $2`,
		contactID, tenantID,
	)
	if err != nil {
		return 0, fmt.Errorf("scrub dependent pii: clear activity notes: %w", err)
	}
	affected += int(res.RowsAffected())

	res, err = tx.Exec(ctx,
		`UPDATE consent_records SET ip_address = NULL, notes = NULL
		 WHERE contact_id = $1 AND tenant_id = $2 AND (ip_address IS NOT NULL OR COALESCE(notes, '') <> '')`,
		contactID, tenantID,
	)
	if err != nil {
		return 0, fmt.Errorf("scrub dependent pii: scrub consent records: %w", err)
	}
	affected += int(res.RowsAffected())

	res, err = tx.Exec(ctx,
		`UPDATE tickets SET
		   requester_name = CASE WHEN requester_is_external THEN $3 ELSE NULL END,
		   requester_email = CASE WHEN requester_is_external THEN $4 ELSE NULL END,
		   updated_at = NOW()
		 WHERE contact_id = $1 AND tenant_id = $2
		   AND (requester_name IS NOT NULL OR requester_email IS NOT NULL)`,
		contactID, tenantID, AnonymizedFirstName+" "+AnonymizedLastName, AnonymizedRequesterEmail,
	)
	if err != nil {
		return 0, fmt.Errorf("scrub dependent pii: scrub ticket requester identity: %w", err)
	}
	affected += int(res.RowsAffected())

	res, err = tx.Exec(ctx,
		`UPDATE dialer_campaign_contacts SET notes = NULL, updated_at = NOW()
		 WHERE contact_id = $1 AND tenant_id = $2 AND notes IS NOT NULL`,
		contactID, tenantID,
	)
	if err != nil {
		return 0, fmt.Errorf("scrub dependent pii: clear dialer campaign contact notes: %w", err)
	}
	affected += int(res.RowsAffected())

	res, err = tx.Exec(ctx,
		`UPDATE dialer_call_sessions SET notes = NULL, next_action = NULL, updated_at = NOW()
		 WHERE tenant_id = $2
		   AND (notes IS NOT NULL OR next_action IS NOT NULL)
		   AND campaign_contact_id IN (
		     SELECT id FROM dialer_campaign_contacts WHERE contact_id = $1 AND tenant_id = $2
		   )`,
		contactID, tenantID,
	)
	if err != nil {
		return 0, fmt.Errorf("scrub dependent pii: clear dialer call session notes: %w", err)
	}
	affected += int(res.RowsAffected())

	return affected, nil
}
