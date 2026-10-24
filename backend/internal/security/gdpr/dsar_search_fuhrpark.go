package gdpr

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Fuhrpark damage reports and logbook trips of a matched user — Art. 15 GDPR.
//
// Read-only. vehicle_damages is matched by the reported_by / resolved_by user
// FK, trip_logs by driver_user_id; photo_keys (storage keys) are never selected.
// trip_logs.driver_name is free text, so a name match would hand a namesake's
// trips to the requester: only the FK counts. A trip without the link (external
// driver, entry from before driver_user_id) is not disclosed here and has to be
// pulled by hand.
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

// tripLogsModule discloses the logbook entries linked to the user as driver,
// private trips included: the data subject's own movements are exactly what
// Art. 15 covers. Matched by driver_user_id only, never by driver_name.
func tripLogsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	cols := []string{"Kennzeichen", "Datum", "Start", "Ziel", "Zweck", "Geschäftspartner", "Kilometer", "Privatfahrt", "Fahrer", "Notizen"}
	return userRowsModule(ctx, pool, "trip logs", "Fahrtenbuch", cols,
		`SELECT v.license_plate, t.date::text, t.start_location, t.end_location, t.purpose, t.business_partner,
		        t.km, CASE WHEN t.is_private THEN 'ja' ELSE 'nein' END, t.driver_name, COALESCE(t.notes, '')
		 FROM trip_logs t
		 JOIN vehicles v ON v.id = t.vehicle_id AND v.tenant_id = t.tenant_id
		 WHERE t.tenant_id = $1 AND t.driver_user_id = $2
		 ORDER BY t.date DESC, t.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(cols))
}
