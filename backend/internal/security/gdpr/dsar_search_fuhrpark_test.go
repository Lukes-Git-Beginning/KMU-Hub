package gdpr

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSearchByQuery_UserFuhrparkDamagesAndLinkedTrips_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Fuhrpark Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Fuhrpark Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Fiona", "Fahrerin", true)
	colleagueID := seedDSARUser(t, pool, tenantOwn, "Kurt", "Kollege", true)
	foreignUserID := seedDSARUser(t, pool, tenantOther, "Fiona", "Fahrerin", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	defer testutil.CleanupRow(t, pool, "users", colleagueID)
	defer testutil.CleanupRow(t, pool, "users", foreignUserID)

	newVehicle := func(tenantID uuid.UUID, plate string) uuid.UUID {
		return testutil.SeedRow(t, pool, "vehicles", map[string]any{
			"tenant_id": tenantID, "license_plate": plate, "make": "VW", "model": "Caddy", "year": 2022,
		})
	}
	ownVehicle := newVehicle(tenantOwn, "M-FF 100")
	foreignVehicle := newVehicle(tenantOther, "B-XX 200")
	defer testutil.CleanupRow(t, pool, "vehicles", ownVehicle)
	defer testutil.CleanupRow(t, pool, "vehicles", foreignVehicle)

	// driver is the free-text name, link the driver_user_id (nil = not linked).
	newTrip := func(tenantID, vehicleID uuid.UUID, driver, purpose string, private bool, link *uuid.UUID) uuid.UUID {
		row := map[string]any{
			"tenant_id": tenantID, "vehicle_id": vehicleID, "date": "2026-10-01",
			"start_location": "München", "end_location": "Augsburg", "purpose": purpose,
			"start_km": 1000, "end_km": 1085, "is_private": private, "driver_name": driver,
			"business_partner": "Müller GmbH",
		}
		if link != nil {
			row["driver_user_id"] = *link
		}
		return testutil.SeedRow(t, pool, "trip_logs", row)
	}
	// Linked: found whatever the free-text name says. Unlinked with the same name: a
	// namesake's trips can't be told apart, so never disclosed.
	tripLinked := newTrip(tenantOwn, ownVehicle, "Fiona Fahrerin", "Kundenbesuch", false, &subjectID)
	tripLinkedAlias := newTrip(tenantOwn, ownVehicle, "F. Fahrerin", "Wochenendfahrt", true, &subjectID)
	tripUnlinkedSameName := newTrip(tenantOwn, ownVehicle, "  fahrerin, FIONA ", "Namensvetter-Fahrt", false, nil)
	tripColleague := newTrip(tenantOwn, ownVehicle, "Kurt Kollege", "Lieferung", false, &colleagueID)
	tripForeign := newTrip(tenantOther, foreignVehicle, "Fiona Fahrerin", "Fremde Fahrt", false, &foreignUserID)
	defer testutil.CleanupRow(t, pool, "trip_logs", tripLinked)
	defer testutil.CleanupRow(t, pool, "trip_logs", tripLinkedAlias)
	defer testutil.CleanupRow(t, pool, "trip_logs", tripUnlinkedSameName)
	defer testutil.CleanupRow(t, pool, "trip_logs", tripColleague)
	defer testutil.CleanupRow(t, pool, "trip_logs", tripForeign)

	newDamage := func(tenantID, vehicleID uuid.UUID, desc string, reporter, resolver *uuid.UUID) uuid.UUID {
		row := map[string]any{
			"tenant_id": tenantID, "vehicle_id": vehicleID, "description": desc,
			"photo_keys": []string{"secret/storage/key.jpg"},
		}
		if reporter != nil {
			row["reported_by"] = *reporter
		}
		if resolver != nil {
			row["resolved_by"] = *resolver
		}
		return testutil.SeedRow(t, pool, "vehicle_damages", row)
	}
	damageReported := newDamage(tenantOwn, ownVehicle, "Kratzer Heckklappe", &subjectID, nil)
	damageResolved := newDamage(tenantOwn, ownVehicle, "Delle Tür", &colleagueID, &subjectID)
	damageColleague := newDamage(tenantOwn, ownVehicle, "Steinschlag", &colleagueID, &colleagueID)
	damageForeign := newDamage(tenantOther, foreignVehicle, "Fremder Schaden", &foreignUserID, nil)
	defer testutil.CleanupRow(t, pool, "vehicle_damages", damageReported)
	defer testutil.CleanupRow(t, pool, "vehicle_damages", damageResolved)
	defer testutil.CleanupRow(t, pool, "vehicle_damages", damageColleague)
	defer testutil.CleanupRow(t, pool, "vehicle_damages", damageForeign)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Fahrerin")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	person := persons[0]

	// Trips come by driver_user_id only: the aliased row is found, the same-name row
	// without a link is not, and neither is the colleague's.
	trips := recordMaps(dsarModule(t, person, "Fahrtenbuch"))
	require.Len(t, trips, 2, "the two linked trips, never the unlinked namesake row")
	tripPurposes := map[string]map[string]string{}
	for _, tr := range trips {
		tripPurposes[tr["Zweck"]] = tr
	}
	require.Contains(t, tripPurposes, "Kundenbesuch")
	require.Contains(t, tripPurposes, "Wochenendfahrt")
	assert.Equal(t, map[string]string{
		"Kennzeichen": "M-FF 100", "Datum": "2026-10-01", "Start": "München", "Ziel": "Augsburg",
		"Zweck": "Kundenbesuch", "Geschäftspartner": "Müller GmbH", "Kilometer": "85",
		"Privatfahrt": "nein", "Fahrer": "Fiona Fahrerin", "Notizen": "",
	}, tripPurposes["Kundenbesuch"])
	assert.Equal(t, "ja", tripPurposes["Wochenendfahrt"]["Privatfahrt"], "private trips are the subject's own data")
	assert.Equal(t, "F. Fahrerin", tripPurposes["Wochenendfahrt"]["Fahrer"])

	damages := recordMaps(dsarModule(t, person, "Fahrzeugschäden"))
	require.Len(t, damages, 2, "reported + resolved by the subject, never the colleague-only damage")
	roles := map[string]string{}
	for _, d := range damages {
		roles[d["Beschreibung"]] = d["Rolle"]
		for _, v := range d {
			assert.NotContains(t, v, "secret/storage", "photo storage keys must never be disclosed")
		}
	}
	assert.Equal(t, map[string]string{"Kratzer Heckklappe": "Melder", "Delle Tür": "Bearbeiter"}, roles)

	// --- tenant isolation -----------------------------------------------------
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Fahrerin")
	require.NoError(t, err)
	require.Len(t, foreign, 1)
	foreignTrips := recordMaps(dsarModule(t, foreign[0], "Fahrtenbuch"))
	require.Len(t, foreignTrips, 1)
	assert.Equal(t, "Fremde Fahrt", foreignTrips[0]["Zweck"])
	for _, d := range recordMaps(dsarModule(t, foreign[0], "Fahrzeugschäden")) {
		assert.Equal(t, "Fremder Schaden", d["Beschreibung"])
	}

	// The colleague sees both role labels on damages and only their own trip.
	colleagueOnly, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Kollege")
	require.NoError(t, err)
	require.Len(t, colleagueOnly, 1)
	colleagueTrips := recordMaps(dsarModule(t, colleagueOnly[0], "Fahrtenbuch"))
	require.Len(t, colleagueTrips, 1)
	assert.Equal(t, "Lieferung", colleagueTrips[0]["Zweck"])
	colleagueRoles := map[string]string{}
	for _, d := range recordMaps(dsarModule(t, colleagueOnly[0], "Fahrzeugschäden")) {
		colleagueRoles[d["Beschreibung"]] = d["Rolle"]
	}
	assert.Equal(t, map[string]string{"Delle Tür": "Melder", "Steinschlag": "Melder und Bearbeiter"}, colleagueRoles)
}
