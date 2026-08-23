package gdpr

// InvitationRetentionHandler implements RetentionHandler for resource_type
// "invitations" — ninth handler on the retention engine from A10
// (retention.go). Fund aus scan-retention-mapping-remaining-services (Lauf
// 11, Iteration 57): the invitations table (email, role, token_hash,
// expires_at, accepted_at — migration 000004) carries personal data (the
// invited email, already an export source in feat-dsar-search-invitations)
// and has no retention handler, even though idx_invitations_expires_at was
// added "for cleanup of expired invitations" from the start (migration
// 000004, line 20) and that cleanup was never built.
//
// Unlike the eight existing handlers, an invitation has TWO distinct age
// states that both warrant cleanup, for different reasons, so ONE handler
// covers both in a single Plan query rather than splitting into two
// resource_types:
//
//   - PENDING (accepted_at IS NULL): never taken up, and once expires_at has
//     passed it is dead weight with no audit value — nobody can accept it
//     any more. Clock: expires_at.
//   - ACCEPTED (accepted_at IS NOT NULL): served its purpose the moment the
//     invited person exists as a users row; the person themself now lives
//     in users, not here. Clock: accepted_at.
//
// Both states delete-only, never anonymize: token_hash is a SHA-256 of the
// invitation token, already pseudonymous (nothing to gain from blanking it),
// and email/role on an invitation with no remaining function has no reason
// to be kept in any form. SupportsAction says so explicitly, same as
// NotificationRetentionHandler.
//
// A single resource_type instead of two ("invitations_pending" /
// "invitations_accepted") because both states share one retention_days
// policy and one operator-facing table — an admin configuring "how long do
// we keep invitations" does not think of these as two different data
// categories, only as two reasons the same row can be done with. DateColumn
// discloses both clocks so the distinction stays visible in the admin view.
import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// InvitationRetentionHandler is the invitations entry in the retention
// registry.
type InvitationRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewInvitationRetentionHandler wires the handler.
func NewInvitationRetentionHandler(pool *pgxpool.Pool) *InvitationRetentionHandler {
	return &InvitationRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for
// invitations.
func (h *InvitationRetentionHandler) ResourceType() string { return "invitations" }

// Table names the governed table for the admin view.
func (h *InvitationRetentionHandler) Table() string { return "invitations" }

// DateColumn discloses both clocks this handler runs — see the file header
// for why an invitation has two.
func (h *InvitationRetentionHandler) DateColumn() string {
	return "expires_at (nie angenommen, abgelaufen) oder accepted_at (angenommen)"
}

// SupportsAction reports that invitations can only be hard-deleted — see the
// file header for why anonymizing one adds nothing.
func (h *InvitationRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists invitations past their retention cutoff in either age state: a
// never-accepted invitation whose expires_at has passed, or an accepted
// invitation whose accepted_at has passed. Never modifies data.
func (h *InvitationRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM invitations
		  WHERE tenant_id = $1
		    AND (
		      (accepted_at IS NULL AND expires_at < $2)
		      OR
		      (accepted_at IS NOT NULL AND accepted_at < $2)
		    )
		  ORDER BY created_at ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("invitation retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("invitation retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("invitation retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids}, nil
}

// Apply deletes the given invitations. Idempotent by construction — a second
// run finds nothing left to delete.
func (h *InvitationRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	if action != models.RetentionActionDelete {
		return 0, fmt.Errorf("invitation retention: unsupported action %q", action)
	}
	tag, err := h.pool.Exec(ctx,
		`DELETE FROM invitations WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("invitation retention: delete: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
