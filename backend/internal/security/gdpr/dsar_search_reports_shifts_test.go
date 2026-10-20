package gdpr

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSearchByQuery_UserReportsAndShifts_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Reports Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Reports Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Rita", "Rapportschreiberin", true)
	colleagueID := seedDSARUser(t, pool, tenantOwn, "Kurt", "Kollege", true)
	foreignUserID := seedDSARUser(t, pool, tenantOther, "Rita", "Rapportschreiberin", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	defer testutil.CleanupRow(t, pool, "users", colleagueID)
	defer testutil.CleanupRow(t, pool, "users", foreignUserID)

	now := time.Now().UTC().Truncate(time.Second)

	// --- work reports ---------------------------------------------------------
	authored := testutil.SeedRow(t, pool, "work_reports", map[string]any{
		"tenant_id": tenantOwn, "title": "Dachstuhl Nord", "author_id": subjectID,
		"reviewer_id": colleagueID, "status": "approved", "lat": "48.137154", "lon": "11.576124",
		"project_name": "Neubau Gartenstraße", "work_start": "07:30", "work_end": "16:00", "break_minutes": 45,
		"signature_data": "data:image/png;base64,SECRETSIGNATUREBYTES", "signed_by": "Rita R.", "signed_at": now,
		"description": "Sparren verlegt",
	})
	reviewed := testutil.SeedRow(t, pool, "work_reports", map[string]any{
		"tenant_id": tenantOwn, "title": "Keller Süd", "author_id": colleagueID,
		"reviewer_id": subjectID, "status": "rejected", "review_note": "Fotos fehlen",
	})
	colleagueOnly := testutil.SeedRow(t, pool, "work_reports", map[string]any{
		"tenant_id": tenantOwn, "title": "Nur Kurt", "author_id": colleagueID,
	})
	foreignReport := testutil.SeedRow(t, pool, "work_reports", map[string]any{
		"tenant_id": tenantOther, "title": "Fremder Rapport", "author_id": foreignUserID,
	})
	defer testutil.CleanupRow(t, pool, "work_reports", authored)
	defer testutil.CleanupRow(t, pool, "work_reports", reviewed)
	defer testutil.CleanupRow(t, pool, "work_reports", colleagueOnly)
	defer testutil.CleanupRow(t, pool, "work_reports", foreignReport)

	// --- shifts ---------------------------------------------------------------
	newShift := func(tenantID uuid.UUID, title string, start time.Time) uuid.UUID {
		return testutil.SeedRow(t, pool, "shifts", map[string]any{
			"tenant_id": tenantID, "title": title, "start_time": start, "end_time": start.Add(8 * time.Hour),
			"location": "Werkhalle 2",
		})
	}
	mineShift := newShift(tenantOwn, "Frühschicht", now.Add(-48*time.Hour))
	swapShift := newShift(tenantOwn, "Spätschicht", now.Add(-24*time.Hour))
	colleagueShift := newShift(tenantOwn, "Nachtschicht", now.Add(-12*time.Hour))
	foreignShift := newShift(tenantOther, "Fremde Schicht", now.Add(-24*time.Hour))
	defer testutil.CleanupRow(t, pool, "shifts", mineShift)
	defer testutil.CleanupRow(t, pool, "shifts", swapShift)
	defer testutil.CleanupRow(t, pool, "shifts", colleagueShift)
	defer testutil.CleanupRow(t, pool, "shifts", foreignShift)

	mineAssign := testutil.SeedRow(t, pool, "shift_assignments", map[string]any{
		"tenant_id": tenantOwn, "shift_id": mineShift, "employee_id": subjectID,
	})
	colleagueAssign := testutil.SeedRow(t, pool, "shift_assignments", map[string]any{
		"tenant_id": tenantOwn, "shift_id": colleagueShift, "employee_id": colleagueID,
	})
	swapAssign := testutil.SeedRow(t, pool, "shift_assignments", map[string]any{
		"tenant_id": tenantOwn, "shift_id": swapShift, "employee_id": colleagueID,
	})
	foreignAssign := testutil.SeedRow(t, pool, "shift_assignments", map[string]any{
		"tenant_id": tenantOther, "shift_id": foreignShift, "employee_id": foreignUserID,
	})
	defer testutil.CleanupRow(t, pool, "shift_assignments", mineAssign)
	defer testutil.CleanupRow(t, pool, "shift_assignments", colleagueAssign)
	defer testutil.CleanupRow(t, pool, "shift_assignments", swapAssign)
	defer testutil.CleanupRow(t, pool, "shift_assignments", foreignAssign)

	// The subject is the swap partner of a request the colleague filed.
	swapMine := testutil.SeedRow(t, pool, "shift_swap_requests", map[string]any{
		"tenant_id": tenantOwn, "shift_id": swapShift, "assignment_id": swapAssign,
		"requested_by_employee_id": colleagueID, "swap_with_employee_id": subjectID,
		"reason": "Arzttermin",
	})
	// A request between the colleague and somebody else must not show up.
	otherPartner := seedDSARUser(t, pool, tenantOwn, "Olga", "Andere", true)
	defer testutil.CleanupRow(t, pool, "users", otherPartner)
	swapOther := testutil.SeedRow(t, pool, "shift_swap_requests", map[string]any{
		"tenant_id": tenantOwn, "shift_id": colleagueShift, "assignment_id": colleagueAssign,
		"requested_by_employee_id": colleagueID, "swap_with_employee_id": otherPartner,
		"reason": "Privat",
	})
	defer testutil.CleanupRow(t, pool, "shift_swap_requests", swapMine)
	defer testutil.CleanupRow(t, pool, "shift_swap_requests", swapOther)

	// --- week approvals -------------------------------------------------------
	weekOwn := testutil.SeedRow(t, pool, "hr_week_approvals", map[string]any{
		"tenant_id": tenantOwn, "employee_id": subjectID, "week_start": "2026-10-12",
		"status": "rejected", "rejection_reason": "Pausen fehlen", "approved_by": colleagueID, "approved_at": now,
	})
	weekDecided := testutil.SeedRow(t, pool, "hr_week_approvals", map[string]any{
		"tenant_id": tenantOwn, "employee_id": colleagueID, "week_start": "2026-10-05",
		"status": "approved", "approved_by": subjectID, "approved_at": now,
	})
	weekColleague := testutil.SeedRow(t, pool, "hr_week_approvals", map[string]any{
		"tenant_id": tenantOwn, "employee_id": colleagueID, "week_start": "2026-09-28",
		"status": "open",
	})
	defer testutil.CleanupRow(t, pool, "hr_week_approvals", weekOwn)
	defer testutil.CleanupRow(t, pool, "hr_week_approvals", weekDecided)
	defer testutil.CleanupRow(t, pool, "hr_week_approvals", weekColleague)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Rapportschreiberin")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	person := persons[0]

	reports := recordMaps(dsarModule(t, person, "Rapporte"))
	require.Len(t, reports, 2, "authored + reviewed report, never the colleague-only one")
	byTitle := map[string]map[string]string{}
	for _, r := range reports {
		byTitle[r["Titel"]] = r
	}
	assert.Equal(t, "Autor", byTitle["Dachstuhl Nord"]["Rolle"]) //nolint:misspell // German UI label
	assert.Equal(t, "48.137154", byTitle["Dachstuhl Nord"]["Breitengrad"])
	assert.Equal(t, "11.576124", byTitle["Dachstuhl Nord"]["Längengrad"])
	assert.Equal(t, "07:30", byTitle["Dachstuhl Nord"]["Arbeitsbeginn"])
	assert.Equal(t, "45", byTitle["Dachstuhl Nord"]["Pause (min)"])
	assert.Contains(t, byTitle["Dachstuhl Nord"]["Unterschrift"], "Unterschrift vorhanden von Rita R.")
	assert.Equal(t, "Prüfer", byTitle["Keller Süd"]["Rolle"])
	assert.Equal(t, "Fotos fehlen", byTitle["Keller Süd"]["Prüfnotiz"])
	assert.Empty(t, byTitle["Keller Süd"]["Unterschrift"])
	for _, r := range reports {
		for _, v := range r {
			assert.NotContains(t, v, "SECRETSIGNATUREBYTES", "signature image must never be disclosed")
		}
	}

	shifts := recordMaps(dsarModule(t, person, "Schichtzuteilungen"))
	require.Len(t, shifts, 1, "only the subject's own assignment")
	assert.Equal(t, "Frühschicht", shifts[0]["Schicht"])
	assert.Equal(t, "Werkhalle 2", shifts[0]["Ort"])

	swaps := recordMaps(dsarModule(t, person, "Schicht-Tauschanfragen"))
	require.Len(t, swaps, 1, "only the request that names the subject")
	assert.Equal(t, "Tauschpartner", swaps[0]["Rolle"])
	assert.Equal(t, "Arzttermin", swaps[0]["Grund"])
	assert.Equal(t, "Spätschicht", swaps[0]["Schicht"])

	weeks := recordMaps(dsarModule(t, person, "Wochenfreigaben"))
	require.Len(t, weeks, 2, "own week + the week the subject approved; not the colleague's open week")
	roles := map[string]string{}
	for _, w := range weeks {
		roles[w["Woche ab"]] = w["Rolle"] + "|" + w["Ablehnungsgrund"]
	}
	assert.Equal(t, map[string]string{
		"12.10.2026": "Mitarbeiter|Pausen fehlen",
		"05.10.2026": "Freigebender|",
	}, roles)

	// --- tenant isolation -----------------------------------------------------
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Rapportschreiberin")
	require.NoError(t, err)
	require.Len(t, foreign, 1, "the other tenant's own user of the same name")
	for _, r := range recordMaps(dsarModule(t, foreign[0], "Rapporte")) {
		assert.Equal(t, "Fremder Rapport", r["Titel"])
	}
	for _, s := range recordMaps(dsarModule(t, foreign[0], "Schichtzuteilungen")) {
		assert.Equal(t, "Fremde Schicht", s["Schicht"])
	}
	assert.NotContains(t, moduleNames(foreign[0]), "Schicht-Tauschanfragen")
	assert.NotContains(t, moduleNames(foreign[0]), "Wochenfreigaben")

	// A forged tenant id must not reach the first tenant's rows either.
	forged, err := SearchByQuery(ctxOther, pool, tenantOther, "Kollege")
	require.NoError(t, err)
	assert.Empty(t, forged)
}
