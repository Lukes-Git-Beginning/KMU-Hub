package gdpr

// DialerAgentStatusLogRetentionHandler implements RetentionHandler for
// resource_type "dialer_agent_status_log" — Lauf 17 decision "Paket
// Logs/Buchungen" (2026-10-25): 12 months after the status change. The twelve
// months are the retention_days an admin sets on the policy row, not a
// constant here.
//
// Clock is changed_at, the moment the agent's status flipped. The table is
// append-only (one row per status change, never updated), so changed_at is
// both the event time and the age of the row. Delete-only: user_id is NOT
// NULL and is the very thing the log records, so a row without it retains
// nothing worth keeping.
//
// lean: no (tenant_id, changed_at) index — the daily run scans the table; add
// one when the log grows to a size where the Plan query shows up in slow-query
// logs.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// DialerAgentStatusLogRetentionHandler is the dialer_agent_status_log entry in
// the retention registry.
type DialerAgentStatusLogRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewDialerAgentStatusLogRetentionHandler wires the handler.
func NewDialerAgentStatusLogRetentionHandler(pool *pgxpool.Pool) *DialerAgentStatusLogRetentionHandler {
	return &DialerAgentStatusLogRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for the agent
// status log.
func (h *DialerAgentStatusLogRetentionHandler) ResourceType() string {
	return "dialer_agent_status_log"
}

// Table names the governed table for the admin view.
func (h *DialerAgentStatusLogRetentionHandler) Table() string { return "dialer_agent_status_log" }

// DateColumn discloses the clock — see the file header.
func (h *DialerAgentStatusLogRetentionHandler) DateColumn() string { return "changed_at" }

// SupportsAction reports that status log rows can only be hard-deleted.
func (h *DialerAgentStatusLogRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists status log rows whose changed_at has passed cutoff. Never
// modifies data.
func (h *DialerAgentStatusLogRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM dialer_agent_status_log
		  WHERE tenant_id = $1 AND changed_at < $2
		  ORDER BY changed_at ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("dialer agent status log retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("dialer agent status log retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dialer agent status log retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids, Skipped: make([]RetentionSkip, 0)}, nil
}

// Apply deletes the given status log rows. Idempotent: a second run over the
// same ids finds nothing left.
func (h *DialerAgentStatusLogRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	if action != models.RetentionActionDelete {
		return 0, fmt.Errorf("dialer agent status log retention: unsupported action %q", action)
	}
	tag, err := h.pool.Exec(ctx,
		`DELETE FROM dialer_agent_status_log WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("dialer agent status log retention: delete: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

var _ RetentionHandler = (*DialerAgentStatusLogRetentionHandler)(nil)
