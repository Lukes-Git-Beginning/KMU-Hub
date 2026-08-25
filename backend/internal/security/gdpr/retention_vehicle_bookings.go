package gdpr

// VehicleBookingRetentionHandler implements RetentionHandler for resource_type
// "vehicle_bookings" — feat-retention-handler-fuhrpark-operational (A6
// Teilentscheidung, 2026-08-24). vehicle_bookings (migration 000300) reserves
// a pool vehicle for a user_id and has no retention mapping.
//
// Delete only, clock ends_at: a finished vehicle reservation has no value
// once its window has passed — unlike trip_logs (a Fahrtenbuch, potentially a
// tax record under Paragraph 147 AO, decided separately in
// decide-retention-policy-hgb-ao-domains) a booking is not evidence of
// anything, only a scheduling artefact. Frist 3 Jahre, oriented on the
// regular civil limitation period (Paragraph 195 BGB) within which a claim
// arising from a vehicle handover could still surface.
import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// VehicleBookingRetentionHandler is the vehicle_bookings entry in the
// retention registry.
type VehicleBookingRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewVehicleBookingRetentionHandler wires the handler.
func NewVehicleBookingRetentionHandler(pool *pgxpool.Pool) *VehicleBookingRetentionHandler {
	return &VehicleBookingRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for vehicle
// bookings.
func (h *VehicleBookingRetentionHandler) ResourceType() string { return "vehicle_bookings" }

// Table names the governed table for the admin view.
func (h *VehicleBookingRetentionHandler) Table() string { return "vehicle_bookings" }

// DateColumn discloses the retention clock.
func (h *VehicleBookingRetentionHandler) DateColumn() string { return "ends_at" }

// SupportsAction reports that vehicle bookings can only be hard-deleted — a
// finished reservation has no residual value once anonymized, the whole row
// is scheduling noise once its window has passed.
func (h *VehicleBookingRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists vehicle bookings whose ends_at is past the cutoff. Never
// modifies data.
func (h *VehicleBookingRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM vehicle_bookings
		  WHERE tenant_id = $1 AND ends_at < $2
		  ORDER BY ends_at ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("vehicle booking retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("vehicle booking retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vehicle booking retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids}, nil
}

// Apply deletes the given vehicle bookings. Idempotent by construction — a
// second run finds nothing left to delete.
func (h *VehicleBookingRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	if action != models.RetentionActionDelete {
		return 0, fmt.Errorf("vehicle booking retention: unsupported action %q", action)
	}
	tag, err := h.pool.Exec(ctx,
		`DELETE FROM vehicle_bookings WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("vehicle booking retention: delete: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
