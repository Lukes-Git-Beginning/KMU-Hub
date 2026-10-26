package gdpr

// CallSessionRetentionHandler implements RetentionHandler for resource_type
// "call_sessions" — Lauf 17 decision "Paket Logs/Buchungen" (2026-10-25):
// 24 months after the call ended. The twenty-four months are the
// retention_days an admin sets on the policy row, not a constant here.
//
// Clock is COALESCE(ended_at, created_at): ended_at is the end of the call;
// a row that never reached 'ended' (the server died mid-call) has none, and
// its start is the best age it has. call_participants goes with the session
// through ON DELETE CASCADE, so one handler governs both tables.
//
// Delete-only. A session carries who called whom and for how long; blanking
// it leaves nothing worth keeping, the same reasoning as guest_sessions.
// dialer_call_sessions.call_session_id is ON DELETE SET NULL, so dialer logs
// survive with the link cleared.
//
// Recordings are not touched. recordings.call_id has no ON DELETE action, so
// a session that still has a recording row cannot be deleted; such sessions
// are reported as skipped and fall due again once the recording's own expiry
// (recording.Service.CleanupExpiredRecordings, which deletes the row) has
// run. The check sits in Apply as well, so a recording started between Plan
// and Apply does not fail the whole batch on the foreign key.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// CallSessionRetentionHandler is the call_sessions entry in the retention
// registry.
type CallSessionRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewCallSessionRetentionHandler wires the handler.
func NewCallSessionRetentionHandler(pool *pgxpool.Pool) *CallSessionRetentionHandler {
	return &CallSessionRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for call sessions.
func (h *CallSessionRetentionHandler) ResourceType() string { return "call_sessions" }

// Table names the governed table for the admin view.
func (h *CallSessionRetentionHandler) Table() string { return "call_sessions" }

// DateColumn discloses the clock — see the file header.
func (h *CallSessionRetentionHandler) DateColumn() string {
	return "ended_at (sonst created_at)"
}

// SupportsAction reports that call sessions can only be hard-deleted.
func (h *CallSessionRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists call sessions that ended before cutoff, minus those still
// referenced by a recording. Never modifies data.
func (h *CallSessionRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT cs.id,
		        EXISTS (SELECT 1 FROM recordings r
		                 WHERE r.call_id = cs.id AND r.tenant_id = cs.tenant_id)
		   FROM call_sessions cs
		  WHERE cs.tenant_id = $1 AND COALESCE(cs.ended_at, cs.created_at) < $2
		  ORDER BY COALESCE(cs.ended_at, cs.created_at) ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("call session retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	skipped := make([]RetentionSkip, 0)
	for rows.Next() {
		var (
			id           uuid.UUID
			hasRecording bool
		)
		if scanErr := rows.Scan(&id, &hasRecording); scanErr != nil {
			return nil, fmt.Errorf("call session retention: scan candidate: %w", scanErr)
		}
		if hasRecording {
			skipped = append(skipped, RetentionSkip{
				RecordID: id,
				Reason:   "recording still present; it expires on its own schedule and the call is retired afterwards",
			})
			continue
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("call session retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids, Skipped: skipped}, nil
}

// Apply deletes the given call sessions (participants cascade). Idempotent: a
// second run over the same ids finds nothing left.
func (h *CallSessionRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	if action != models.RetentionActionDelete {
		return 0, fmt.Errorf("call session retention: unsupported action %q", action)
	}
	tag, err := h.pool.Exec(ctx,
		`DELETE FROM call_sessions cs
		  WHERE cs.tenant_id = $1 AND cs.id = ANY($2)
		    AND NOT EXISTS (SELECT 1 FROM recordings r
		                     WHERE r.call_id = cs.id AND r.tenant_id = cs.tenant_id)`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("call session retention: delete: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

var _ RetentionHandler = (*CallSessionRetentionHandler)(nil)
