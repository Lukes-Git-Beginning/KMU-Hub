package gdpr

// Retention handler for resource_type "form_webhook_deliveries".
//
// Every form submission with at least one active webhook enqueues a delivery
// row whose payload is a full copy of the submission answers (see
// insertSubmissionTx in formulare/postgres_repository.go) — the same personal
// data as form_submissions, but outside the reach of
// FormSubmissionRetentionHandler: the FK from deliveries to submissions is
// ON DELETE CASCADE, so a submission DELETE cleans up after itself, yet
// nothing ever removed a delivery whose submission stayed. Until this handler
// the table only grew.
//
// Generic retention framework rather than a dedicated cleanup job: the table
// carries tenant_id, a retention clock and a policy that an admin may want to
// see next to form_submissions, and the framework already provides dry run,
// run reports and the tenant walk. A job would have rebuilt all three.
//
// Scope is FINAL deliveries only (status 'delivered' or 'dead'). 'pending'
// and 'failed' rows still belong to the WebhookWorker retry loop; deleting
// one would silently drop a delivery that is still owed.
//
// Delete only: there is nothing to anonymize that is worth keeping — a
// delivery without its payload is no longer a delivery.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// FormWebhookDeliveryRetentionHandler is the form_webhook_deliveries entry in
// the retention registry.
type FormWebhookDeliveryRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewFormWebhookDeliveryRetentionHandler wires the handler.
func NewFormWebhookDeliveryRetentionHandler(pool *pgxpool.Pool) *FormWebhookDeliveryRetentionHandler {
	return &FormWebhookDeliveryRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for webhook
// deliveries.
func (h *FormWebhookDeliveryRetentionHandler) ResourceType() string {
	return "form_webhook_deliveries"
}

// Table names the governed table for the admin view.
func (h *FormWebhookDeliveryRetentionHandler) Table() string { return "form_webhook_deliveries" }

// DateColumn discloses the retention clock. created_at, not delivered_at:
// a 'dead' delivery never gets a delivered_at, and created_at is set for
// every row. The retry window is minutes to hours, far below any sensible
// retention_days, so created_at is within that window of the final attempt.
func (h *FormWebhookDeliveryRetentionHandler) DateColumn() string {
	return "created_at (nur status delivered/dead)"
}

// SupportsAction reports that deliveries can only be hard-deleted.
func (h *FormWebhookDeliveryRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists final deliveries past their retention cutoff. Never modifies
// data.
func (h *FormWebhookDeliveryRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM form_webhook_deliveries
		  WHERE tenant_id = $1 AND status IN ('delivered', 'dead') AND created_at < $2
		  ORDER BY created_at ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("form webhook delivery retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("form webhook delivery retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("form webhook delivery retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids, Skipped: make([]RetentionSkip, 0)}, nil
}

// Apply deletes the given deliveries. The status filter is repeated here so a
// row that went back to 'pending' between Plan and Apply is never deleted.
// Idempotent: a second run over the same ids finds nothing left.
func (h *FormWebhookDeliveryRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	if action != models.RetentionActionDelete {
		return 0, fmt.Errorf("form webhook delivery retention: unsupported action %q", action)
	}
	tag, err := h.pool.Exec(ctx,
		`DELETE FROM form_webhook_deliveries
		  WHERE tenant_id = $1 AND id = ANY($2) AND status IN ('delivered', 'dead')`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("form webhook delivery retention: delete: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

var _ RetentionHandler = (*FormWebhookDeliveryRetentionHandler)(nil)
