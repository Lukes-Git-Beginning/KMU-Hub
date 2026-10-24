package gdpr

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Fuhrpark damage reports of a matched user — Art. 15 GDPR.
//
// Read-only. vehicle_damages is matched by the reported_by / resolved_by user
// FK; photo_keys (storage keys) are never selected.
//
// lean: the logbook (trip_logs) is not disclosed. driver_name is free text
// without a user FK, and a name match hands a namesake's trips to the requester.
// Disclose by that FK once trip_logs carries one (backend-gaps.md, Owner-FK-Gap
// fuhrpark); until then the reviewer pulls a driver's trips by hand.
// ---------------------------------------------------------------------------

// vehicleDamagesModule discloses damage reports the user filed or resolved.
func vehicleDamagesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	cols := []string{"Kennzeichen", "Rolle", "Beschreibung", "Schweregrad", "Status", "Notizen", "Gemeldet am", "Gelöst am"}
	return userRowsModule(ctx, pool, "vehicle damages", "Fahrzeugschäden", cols,
		`SELECT v.license_plate,
		        CASE WHEN d.reported_by = $2 AND d.resolved_by = $2 THEN 'Melder und Bearbeiter'
		             WHEN d.reported_by = $2 THEN 'Melder' ELSE 'Bearbeiter' END,
		        d.description, d.severity, d.status, COALESCE(d.notes, ''), d.created_at, d.resolved_at
		 FROM vehicle_damages d
		 JOIN vehicles v ON v.id = d.vehicle_id AND v.tenant_id = d.tenant_id
		 WHERE d.tenant_id = $1 AND (d.reported_by = $2 OR d.resolved_by = $2)
		 ORDER BY d.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(cols))
}
