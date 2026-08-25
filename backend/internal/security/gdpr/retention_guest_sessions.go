package gdpr

// GuestSessionRetentionHandler implements RetentionHandler for resource_type
// "guest_sessions" — A6 Teilentscheidung, decided 2026-08-24: time-based
// deletion WITHOUT DSAR wiring, 90 days after last_activity_at. A guest is
// neither a contacts nor a users row, so a subject-access request would need
// a third lookup path that nobody has built (Lauf 10) — but that gap does
// not block retention here, it argues for it: the shorter guest data lives,
// the smaller the unresolved DSAR exposure.
//
// Clock is last_activity_at, not created_at (a long-running conversation
// would otherwise fall due mid-conversation) and not expires_at (that only
// bounds the session token, not the support relationship). Delete-only: a
// guest_sessions row without display_name, email, ip_address and user_agent
// retains nothing worth anonymizing — the row exists purely to carry that
// contact data plus a token, and once it is due, none of it has a reason to
// stay in any form.
//
// The pre-existing idx_guest_sessions_cleanup (migration 000054) only
// serves expires_at WHERE is_active = true and was never used by any
// cleanup job. It does not fit this handler's query — a session can be
// long past its 90-day relevance while still is_active = true, and an
// inactive session is just as due as an active one — so migration 000325
// adds idx_guest_sessions_tenant_last_activity instead.
//
// The chat messages a guest sent are explicitly NOT covered here — they
// hang off channels/messages and carry their own retention question
// (commercial-letter or not), tracked by decide-retention-policy-hgb-ao-domains.
// This handler only clears the session metadata row; messages.guest_session_id
// references ON DELETE SET NULL (migration 000054), so a message survives
// its guest session with the link severed, not with the message deleted.
import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// GuestSessionRetentionHandler is the guest_sessions entry in the retention
// registry.
type GuestSessionRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewGuestSessionRetentionHandler wires the handler.
func NewGuestSessionRetentionHandler(pool *pgxpool.Pool) *GuestSessionRetentionHandler {
	return &GuestSessionRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for guest
// sessions.
func (h *GuestSessionRetentionHandler) ResourceType() string { return "guest_sessions" }

// Table names the governed table for the admin view.
func (h *GuestSessionRetentionHandler) Table() string { return "guest_sessions" }

// DateColumn discloses the clock this handler runs on — see the file header
// for why it is last_activity_at and neither created_at nor expires_at.
func (h *GuestSessionRetentionHandler) DateColumn() string { return "last_activity_at" }

// SupportsAction reports that guest sessions can only be hard-deleted — a
// session with its contact fields already blanked has nothing left to
// anonymize.
func (h *GuestSessionRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists guest sessions whose last_activity_at has passed cutoff,
// regardless of is_active. Never modifies data.
func (h *GuestSessionRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM guest_sessions
		  WHERE tenant_id = $1 AND last_activity_at < $2
		  ORDER BY last_activity_at ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("guest session retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("guest session retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("guest session retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids, Skipped: make([]RetentionSkip, 0)}, nil
}

// Apply deletes the given guest sessions. Idempotent by construction — a
// second run over the same ids finds nothing left to delete.
func (h *GuestSessionRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	if action != models.RetentionActionDelete {
		return 0, fmt.Errorf("guest session retention: unsupported action %q", action)
	}
	tag, err := h.pool.Exec(ctx,
		`DELETE FROM guest_sessions WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("guest session retention: delete: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

var _ RetentionHandler = (*GuestSessionRetentionHandler)(nil)
