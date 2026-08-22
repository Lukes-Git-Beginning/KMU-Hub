package gdpr

// DialerCallRetentionHandler and ChatMessageRetentionHandler — second and
// third handlers on the retention engine from A10 (retention.go), covering
// two surfaces where personal data accumulates without an active decision to
// enter it: dialer call logs and chat messages.
//
// Object storage: recordings (call_sessions -> recordings, MinIO-backed)
// are NOT reachable from either handler. dialer_call_sessions.call_session_id
// only ever points AT call_sessions (no reverse FK), so deleting or
// anonymizing a dialer call session never touches call_sessions or the
// recordings row that might reference it. Recordings already have their own
// independent expiry mechanism (recording.Service.CleanupExpiredRecordings,
// retention_expires_at, MinIO object removal) — this unit does not
// duplicate it.
//
// Both handlers use the raw pool directly, the same style already
// established by ChatErasureHandler in erasure.go, rather than going through
// the dialer/chat package repositories — there is no existing business logic
// worth reusing here, unlike ContactRetentionHandler's use of IsInUse.

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ---------------------------------------------------------------------------
// DialerCallRetentionHandler
// ---------------------------------------------------------------------------

// DialerCallRetentionHandler is the dialer_call_sessions entry in the
// retention registry.
type DialerCallRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewDialerCallRetentionHandler wires the handler.
func NewDialerCallRetentionHandler(pool *pgxpool.Pool) *DialerCallRetentionHandler {
	return &DialerCallRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for dialer calls.
func (h *DialerCallRetentionHandler) ResourceType() string { return "dialer_calls" }

// Table names the governed table for the admin view. dialer_call_events
// cascades on delete (migration 000065) and is not named separately.
func (h *DialerCallRetentionHandler) Table() string { return "dialer_call_sessions" }

// DateColumn discloses the retention clock. updated_at rather than
// created_at, deliberately: anonymize bumps it, which is what gives Apply
// idempotency for free — the same mechanism ContactRetentionHandler uses.
// In practice the two rarely drift apart: updated_at only otherwise changes
// a few minutes after created_at, when the agent logs the call outcome.
func (h *DialerCallRetentionHandler) DateColumn() string { return "updated_at" }

// SupportsAction reports that dialer calls can be both hard-deleted and
// anonymized. There is no RESTRICT-style block like contacts have — every
// candidate is Due regardless of action.
func (h *DialerCallRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete || action == models.RetentionActionAnonymize
}

// Plan lists dialer call sessions past their retention cutoff. Never
// modifies data.
func (h *DialerCallRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM dialer_call_sessions WHERE tenant_id = $1 AND updated_at < $2 ORDER BY created_at ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("dialer call retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("dialer call retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dialer call retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids}, nil
}

// Apply deletes or anonymizes the given dialer call sessions. Delete relies
// on the ON DELETE CASCADE from dialer_call_events to dialer_call_sessions
// (migration 000065) to take the per-call event trail with it. Anonymize
// clears the two free-text fields an agent can put personal remarks about
// the contact into — notes and next_action — and leaves outcome, duration
// and timestamps intact, since those are call statistics, not personal
// data about the contact beyond what the outcome label (a tenant-configured
// category) already discloses.
func (h *DialerCallRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, anonymizedLabel string) (int, error) {
	switch action {
	case models.RetentionActionDelete:
		tag, err := h.pool.Exec(ctx,
			`DELETE FROM dialer_call_sessions WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids,
		)
		if err != nil {
			return 0, fmt.Errorf("dialer call retention: delete: %w", err)
		}
		return int(tag.RowsAffected()), nil
	case models.RetentionActionAnonymize:
		tag, err := h.pool.Exec(ctx,
			`UPDATE dialer_call_sessions
			    SET notes = '[' || $3 || ']', next_action = NULL, updated_at = NOW()
			  WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids, anonymizedLabel,
		)
		if err != nil {
			return 0, fmt.Errorf("dialer call retention: anonymize: %w", err)
		}
		return int(tag.RowsAffected()), nil
	default:
		return 0, fmt.Errorf("dialer call retention: unsupported action %q", action)
	}
}

// ---------------------------------------------------------------------------
// ChatMessageRetentionHandler
// ---------------------------------------------------------------------------

// ChatMessageRetentionHandler is the chat_messages entry in the retention
// registry. ResourceType is deliberately distinct from Table (messages):
// "messages" is generic enough to collide with a future email- or
// inbox-message policy, so the policy-facing name is more specific than the
// table it maps to.
type ChatMessageRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewChatMessageRetentionHandler wires the handler.
func NewChatMessageRetentionHandler(pool *pgxpool.Pool) *ChatMessageRetentionHandler {
	return &ChatMessageRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for chat messages.
func (h *ChatMessageRetentionHandler) ResourceType() string { return "chat_messages" }

// Table names the governed table for the admin view.
func (h *ChatMessageRetentionHandler) Table() string { return "messages" }

// DateColumn discloses the retention clock: the later of created_at and
// edited_at. A message an author or moderator recently edited is treated as
// still relevant, the same reasoning ContactRetentionHandler applies to a
// contact with recent activity. It also gives anonymize idempotency for
// free, since anonymizing bumps edited_at.
func (h *ChatMessageRetentionHandler) DateColumn() string {
	return "created_at (oder edited_at, falls spaeter)"
}

// SupportsAction reports that chat messages can be both hard-deleted and
// anonymized.
func (h *ChatMessageRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete || action == models.RetentionActionAnonymize
}

// Plan lists messages past their retention cutoff. Never modifies data.
func (h *ChatMessageRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM messages
		  WHERE tenant_id = $1
		    AND GREATEST(created_at, COALESCE(edited_at, created_at)) < $2
		  ORDER BY created_at ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("chat message retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("chat message retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("chat message retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids}, nil
}

// Apply anonymizes or hard-deletes the given messages.
//
// Anonymize reuses the exact mechanism ChatErasureHandler.ExecuteErasure
// already applies for the GDPR-erasure-request path (content replaced with
// a bracketed label, is_deleted set, edited_at bumped) — one anonymization
// mechanism for chat content, not two.
//
// Delete hard-removes the rows. A deleted message that was itself a thread
// reply leaves its parent's reply_count stale (a denormalized display
// counter) unless corrected; a deleted message that had replies turns them
// into top-level messages via ON DELETE SET NULL on parent_message_id
// (migration 000016) — the schema's own answer to "the parent of a reply
// disappears". The RETURNING clause collects which deleted rows were replies
// so their parent's counter can be corrected; a correction failure is
// logged and does not fail the run, same as the existing manual-delete path
// in message.Service.Delete.
func (h *ChatMessageRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, anonymizedLabel string) (int, error) {
	switch action {
	case models.RetentionActionAnonymize:
		tag, err := h.pool.Exec(ctx,
			`UPDATE messages
			    SET content = '[' || $3 || ']', is_deleted = true, edited_at = NOW()
			  WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids, anonymizedLabel,
		)
		if err != nil {
			return 0, fmt.Errorf("chat message retention: anonymize: %w", err)
		}
		return int(tag.RowsAffected()), nil
	case models.RetentionActionDelete:
		return h.deleteWithReplyCountFixup(ctx, tenantID, ids)
	default:
		return 0, fmt.Errorf("chat message retention: unsupported action %q", action)
	}
}

func (h *ChatMessageRetentionHandler) deleteWithReplyCountFixup(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (int, error) {
	rows, err := h.pool.Query(ctx,
		`DELETE FROM messages WHERE tenant_id = $1 AND id = ANY($2) RETURNING parent_message_id`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("chat message retention: delete: %w", err)
	}

	affected := 0
	parents := make([]uuid.UUID, 0)
	for rows.Next() {
		var parentID *uuid.UUID
		if scanErr := rows.Scan(&parentID); scanErr != nil {
			rows.Close()
			return affected, fmt.Errorf("chat message retention: scan deleted row: %w", scanErr)
		}
		affected++
		if parentID != nil {
			parents = append(parents, *parentID)
		}
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return affected, fmt.Errorf("chat message retention: iterate deleted rows: %w", rowsErr)
	}

	for _, parentID := range parents {
		if _, decErr := h.pool.Exec(ctx,
			`UPDATE messages SET reply_count = GREATEST(reply_count - 1, 0) WHERE id = $1`, parentID,
		); decErr != nil {
			slog.Error("chat message retention: reply_count fixup failed",
				"parent_message_id", parentID, "error", decErr)
		}
	}
	return affected, nil
}
