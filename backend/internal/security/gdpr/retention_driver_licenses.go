package gdpr

// DriverLicenseRetentionHandler implements RetentionHandler for resource_type
// "driver_licenses" — feat-retention-handler-fuhrpark-operational (A6
// Teilentscheidung, 2026-08-24). driver_licenses (migration 000279) is one
// row per Fuehrerscheinkontrolle, a history, not a status field: the row with
// the latest checked_at for a given driver_id is the current compliance
// state (Halterpflicht) and must never be deleted, no matter its age — an
// employer with only one, decade-old check on file still needs that row to
// show the control happened at all. Frist: 3 Jahre, same civil-limitation
// reasoning as VehicleBookingRetentionHandler.
//
// Plan resolves "latest per driver_id" with a window function over ALL of
// the tenant's rows, not just the ones past cutoff — a driver with a single,
// ancient check must keep it, and that can only be known by looking past the
// cutoff window.
import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// DriverLicenseRetentionHandler is the driver_licenses entry in the
// retention registry.
type DriverLicenseRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewDriverLicenseRetentionHandler wires the handler.
func NewDriverLicenseRetentionHandler(pool *pgxpool.Pool) *DriverLicenseRetentionHandler {
	return &DriverLicenseRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for driver
// license checks.
func (h *DriverLicenseRetentionHandler) ResourceType() string { return "driver_licenses" }

// Table names the governed table for the admin view.
func (h *DriverLicenseRetentionHandler) Table() string { return "driver_licenses" }

// DateColumn discloses the retention clock and the current-state exclusion.
func (h *DriverLicenseRetentionHandler) DateColumn() string {
	return "checked_at (die jeweils juengste Zeile je driver_id bleibt als aktueller Kontrollstand erhalten)"
}

// SupportsAction reports that driver license checks can only be
// hard-deleted — the row is either evidence of a past control or it is not,
// there is nothing left to anonymize once the driver reference is gone.
func (h *DriverLicenseRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists driver license checks past their retention cutoff. A candidate
// that is the newest check for its driver_id is moved to Skipped instead of
// Due — it is the current compliance record and must survive regardless of
// age. Never modifies data.
func (h *DriverLicenseRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id, is_latest FROM (
		     SELECT id, checked_at,
		            ROW_NUMBER() OVER (PARTITION BY driver_id ORDER BY checked_at DESC, id DESC) = 1 AS is_latest
		       FROM driver_licenses
		      WHERE tenant_id = $1
		 ) ranked
		 WHERE checked_at < $2
		 ORDER BY checked_at ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("driver license retention: list candidates: %w", err)
	}
	defer rows.Close()

	plan := &RetentionPlan{Due: make([]uuid.UUID, 0), Skipped: make([]RetentionSkip, 0)}
	for rows.Next() {
		var id uuid.UUID
		var isLatest bool
		if scanErr := rows.Scan(&id, &isLatest); scanErr != nil {
			return nil, fmt.Errorf("driver license retention: scan candidate: %w", scanErr)
		}
		if isLatest {
			plan.Skipped = append(plan.Skipped, RetentionSkip{
				RecordID: id,
				Reason:   "ist die aktuelle Kontrollzeile dieses Fahrers; wird nicht geloescht, auch wenn sie aelter als die Frist ist, weil sie der Nachweis der Halterpflicht ist",
			})
			continue
		}
		plan.Due = append(plan.Due, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("driver license retention: iterate candidates: %w", err)
	}
	return plan, nil
}

// Apply deletes the given driver license checks. Idempotent by
// construction — a second run finds nothing left to delete, and the
// current-record guard in Plan means the newest row per driver never
// reaches here.
func (h *DriverLicenseRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	if action != models.RetentionActionDelete {
		return 0, fmt.Errorf("driver license retention: unsupported action %q", action)
	}
	tag, err := h.pool.Exec(ctx,
		`DELETE FROM driver_licenses WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("driver license retention: delete: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
