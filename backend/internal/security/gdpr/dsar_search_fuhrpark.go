package gdpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Fuhrpark logbook and damage reports of a matched user — Art. 15 GDPR.
//
// Read-only. trip_logs.driver_name is free text, not a user FK: a trip is
// attributed by case-insensitive equality with "Vorname Nachname",
// "Nachname Vorname" or "Nachname, Vorname". A namesake in the same tenant
// would show up, hence the "(Namensabgleich)" suffix in the module title.
// The logbook is bound by § 147 AO / § 6 EStG: disclosure is owed, erasure
// is not. vehicle_damages is matched by the reported_by / resolved_by user
// FK; photo_keys (storage keys) are never selected.
// ---------------------------------------------------------------------------

// fuhrparkModules collects the logbook and damage modules for one user;
// empty ones are skipped.
func fuhrparkModules(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID, first, last string) ([]DSARModule, error) {
	var out []DSARModule
	add := func(mod *DSARModule, err error) error {
		if err != nil {
			return fmt.Errorf("dsar: fuhrpark: %w", err)
		}
		if mod != nil {
			out = append(out, *mod)
		}
		return nil
	}

	if names := driverNameVariants(first, last); len(names) > 0 {
		cols := []string{"Datum", "Kennzeichen", "Fahrer", "Von", "Nach", "Zweck", "Geschäftspartner", "Kilometer", "Privatfahrt", "Notizen"}
		if err := add(userRowsModule(ctx, pool, "trip logs", "Fahrtenbuch (Namensabgleich)", cols,
			`SELECT t.date, v.license_plate, t.driver_name, t.start_location, t.end_location, t.purpose,
			        t.business_partner, t.km, t.is_private, COALESCE(t.notes, '')
			 FROM trip_logs t
			 JOIN vehicles v ON v.id = t.vehicle_id AND v.tenant_id = t.tenant_id
			 WHERE t.tenant_id = $1 AND LOWER(BTRIM(t.driver_name)) = ANY($2::text[])
			 ORDER BY t.date DESC, t.created_at DESC
			 LIMIT $3`, []any{tenantID, names, dsarMaxRows}, valuesRecord(cols))); err != nil {
			return nil, err
		}
	}

	cols := []string{"Kennzeichen", "Rolle", "Beschreibung", "Schweregrad", "Status", "Notizen", "Gemeldet am", "Gelöst am"}
	if err := add(userRowsModule(ctx, pool, "vehicle damages", "Fahrzeugschäden", cols,
		`SELECT v.license_plate,
		        CASE WHEN d.reported_by = $2 AND d.resolved_by = $2 THEN 'Melder und Bearbeiter'
		             WHEN d.reported_by = $2 THEN 'Melder' ELSE 'Bearbeiter' END,
		        d.description, d.severity, d.status, COALESCE(d.notes, ''), d.created_at, d.resolved_at
		 FROM vehicle_damages d
		 JOIN vehicles v ON v.id = d.vehicle_id AND v.tenant_id = d.tenant_id
		 WHERE d.tenant_id = $1 AND (d.reported_by = $2 OR d.resolved_by = $2)
		 ORDER BY d.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(cols))); err != nil {
		return nil, err
	}
	return out, nil
}

// driverNameVariants lists the lower-cased spellings a driver_name may carry
// for a user; empty if the user has no name to match on.
func driverNameVariants(first, last string) []string {
	f, l := strings.ToLower(strings.TrimSpace(first)), strings.ToLower(strings.TrimSpace(last))
	if f == "" || l == "" {
		return nil
	}
	return []string{f + " " + l, l + " " + f, l + ", " + f}
}
