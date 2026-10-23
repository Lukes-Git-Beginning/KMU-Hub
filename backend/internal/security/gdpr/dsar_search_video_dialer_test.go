package gdpr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSearchByQuery_UserVideoAndDialerParticipation_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	sys := testutil.WithSystemCtx(context.Background())

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Video Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Video Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Vera", "Videoteilnehmerin", true)
	colleagueID := seedDSARUser(t, pool, tenantOwn, "Kurt", "Kollege", true)
	foreignUserID := seedDSARUser(t, pool, tenantOther, "Vera", "Videoteilnehmerin", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	defer testutil.CleanupRow(t, pool, "users", colleagueID)
	defer testutil.CleanupRow(t, pool, "users", foreignUserID)

	now := time.Now().UTC().Truncate(time.Second)

	// --- calls ----------------------------------------------------------------
	initiatedCall := testutil.SeedRow(t, pool, "call_sessions", map[string]any{
		"tenant_id": tenantOwn, "call_type": "one_to_one", "room_name": "room-" + uuid.NewString(),
		"initiator_id": subjectID, "status": "ended", "duration_seconds": 125,
		"created_at": now.Add(-3 * time.Hour),
	})
	joinedCall := testutil.SeedRow(t, pool, "call_sessions", map[string]any{
		"tenant_id": tenantOwn, "call_type": "group", "room_name": "room-" + uuid.NewString(),
		"initiator_id": colleagueID, "status": "ended", "created_at": now.Add(-2 * time.Hour),
	})
	colleagueOnlyCall := testutil.SeedRow(t, pool, "call_sessions", map[string]any{
		"tenant_id": tenantOwn, "call_type": "one_to_one", "room_name": "room-" + uuid.NewString(),
		"initiator_id": colleagueID, "status": "ended", "created_at": now.Add(-time.Hour),
	})
	foreignCall := testutil.SeedRow(t, pool, "call_sessions", map[string]any{
		"tenant_id": tenantOther, "call_type": "one_to_one", "room_name": "room-" + uuid.NewString(),
		"initiator_id": foreignUserID, "status": "ended", "created_at": now.Add(-time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "call_sessions", initiatedCall)
	defer testutil.CleanupRow(t, pool, "call_sessions", joinedCall)
	defer testutil.CleanupRow(t, pool, "call_sessions", colleagueOnlyCall)
	defer testutil.CleanupRow(t, pool, "call_sessions", foreignCall)
	_, err := pool.Exec(sys,
		`INSERT INTO call_participants (call_id, user_id, tenant_id, joined_at) VALUES ($1, $2, $3, $4)`,
		joinedCall, subjectID, tenantOwn, now.Add(-2*time.Hour))
	require.NoError(t, err)

	// --- meetings -------------------------------------------------------------
	newMeeting := func(tenantID, organizer uuid.UUID, title string, start time.Time) uuid.UUID {
		return testutil.SeedRow(t, pool, "meetings", map[string]any{
			"tenant_id": tenantID, "title": title, "organizer_id": organizer,
			"scheduled_start": start, "scheduled_end": start.Add(time.Hour),
		})
	}
	organised := newMeeting(tenantOwn, subjectID, "Quartalsplanung", now.Add(-48*time.Hour))
	invited := newMeeting(tenantOwn, colleagueID, "Kundenworkshop", now.Add(-24*time.Hour))
	colleagueOnly := newMeeting(tenantOwn, colleagueID, "Nur für Kurt", now.Add(-12*time.Hour))
	foreignMeeting := newMeeting(tenantOther, foreignUserID, "Fremdes Meeting", now.Add(-24*time.Hour))
	defer testutil.CleanupRow(t, pool, "meetings", organised)
	defer testutil.CleanupRow(t, pool, "meetings", invited)
	defer testutil.CleanupRow(t, pool, "meetings", colleagueOnly)
	defer testutil.CleanupRow(t, pool, "meetings", foreignMeeting)

	_, err = pool.Exec(sys,
		`INSERT INTO meeting_attendees (meeting_id, user_id, rsvp_status, tenant_id) VALUES ($1, $2, 'accepted', $3)`,
		invited, subjectID, tenantOwn)
	require.NoError(t, err)
	_, err = pool.Exec(sys,
		`INSERT INTO meeting_cohosts (tenant_id, meeting_id, user_id, granted_by) VALUES ($1, $2, $3, $4)`,
		tenantOwn, invited, subjectID, colleagueID)
	require.NoError(t, err)
	roomID := testutil.SeedRow(t, pool, "meeting_breakout_rooms", map[string]any{
		"tenant_id": tenantOwn, "meeting_id": invited, "room_name": "br-" + uuid.NewString(),
		"label": "Gruppe Nord", "created_by": colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "meeting_breakout_rooms", roomID)
	_, err = pool.Exec(sys,
		`INSERT INTO meeting_breakout_assignments (tenant_id, meeting_id, breakout_room_id, user_id, assigned_by)
		 VALUES ($1, $2, $3, $4, $5)`,
		tenantOwn, invited, roomID, subjectID, colleagueID)
	require.NoError(t, err)

	mineItem := testutil.SeedRow(t, pool, "meeting_action_items", map[string]any{
		"tenant_id": tenantOwn, "meeting_id": invited, "description": "Folien bis Freitag verschicken",
		"assignee_id": subjectID,
	})
	colleagueItem := testutil.SeedRow(t, pool, "meeting_action_items", map[string]any{
		"tenant_id": tenantOwn, "meeting_id": invited, "description": "Raum buchen", "assignee_id": colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "meeting_action_items", mineItem)
	defer testutil.CleanupRow(t, pool, "meeting_action_items", colleagueItem)

	// --- dialer ---------------------------------------------------------------
	campaignID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"tenant_id": tenantOwn, "name": "Herbstaktion", "created_by": colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campaignID)
	statusMine := testutil.SeedRow(t, pool, "dialer_agent_status_log", map[string]any{
		"tenant_id": tenantOwn, "user_id": subjectID, "campaign_id": campaignID,
		"status": "on_call", "previous_status": "available", "changed_at": now.Add(-time.Hour),
	})
	statusColleague := testutil.SeedRow(t, pool, "dialer_agent_status_log", map[string]any{
		"tenant_id": tenantOwn, "user_id": colleagueID, "status": "break", "changed_at": now,
	})
	defer testutil.CleanupRow(t, pool, "dialer_agent_status_log", statusMine)
	defer testutil.CleanupRow(t, pool, "dialer_agent_status_log", statusColleague)

	// Cap check: one more agent with more entries than the cap.
	capUserID := seedDSARUser(t, pool, tenantOwn, "Cap", "Vielwechsler", true)
	defer testutil.CleanupRow(t, pool, "users", capUserID)
	_, err = pool.Exec(sys,
		`INSERT INTO dialer_agent_status_log (tenant_id, user_id, status, changed_at)
		 SELECT $1, $2, 'available', $3::timestamptz - (g || ' minutes')::interval
		 FROM generate_series(1, $4::int) g`,
		tenantOwn, capUserID, now, dsarMaxRows+20)
	require.NoError(t, err)
	defer func() {
		_, _ = pool.Exec(sys, `DELETE FROM dialer_agent_status_log WHERE user_id = $1`, capUserID)
	}()

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Videoteilnehmerin")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	person := persons[0]

	calls := recordMaps(dsarModule(t, person, "Anrufe und Videocalls"))
	require.Len(t, calls, 2, "initiated + joined call, never the colleague's other call")
	byRole := map[string]map[string]string{}
	for _, c := range calls {
		byRole[c["Rolle"]] = c
	}
	assert.Equal(t, "one_to_one", byRole["Initiator"]["Art"])
	assert.Equal(t, "125", byRole["Initiator"]["Dauer (s)"])
	assert.Equal(t, "group", byRole["Teilnehmer"]["Art"])
	assert.NotEmpty(t, byRole["Teilnehmer"]["Beigetreten"])

	meetings := recordMaps(dsarModule(t, person, "Meeting-Teilnahmen"))
	got := map[string]string{}
	for _, m := range meetings {
		got[m["Titel"]+"|"+m["Rolle"]] = m["Antwort / Raum"]
	}
	assert.Equal(t, map[string]string{
		"Quartalsplanung|Organisator":  "",
		"Kundenworkshop|Teilnehmer":    "accepted",
		"Kundenworkshop|Co-Host":       "",
		"Kundenworkshop|Breakout-Raum": "Gruppe Nord",
	}, got, "organiser, attendee, co-host and breakout rows; not the colleague-only meeting")

	items := recordMaps(dsarModule(t, person, "Meeting-Aktionspunkte"))
	require.Len(t, items, 1)
	assert.Equal(t, "Folien bis Freitag verschicken", items[0]["Aktionspunkt"])
	assert.Equal(t, "Kundenworkshop", items[0]["Meeting"])

	statuses := recordMaps(dsarModule(t, person, "Dialer-Agent-Statushistorie"))
	require.Len(t, statuses, 1, "only the subject's own status change")
	assert.Equal(t, "on_call", statuses[0]["Status"])
	assert.Equal(t, "available", statuses[0]["Vorheriger Status"])
	assert.Equal(t, "Herbstaktion", statuses[0]["Kampagne"])

	capPersons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Vielwechsler")
	require.NoError(t, err)
	require.Len(t, capPersons, 1)
	capped := dsarModule(t, capPersons[0], "Dialer-Agent-Statushistorie")
	assert.Len(t, capped.Records, dsarMaxRows, fmt.Sprintf("status history capped at %d newest rows", dsarMaxRows))

	// --- tenant isolation -----------------------------------------------------
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Videoteilnehmerin")
	require.NoError(t, err)
	require.Len(t, foreign, 1, "the other tenant's own user of the same name")
	for _, c := range recordMaps(dsarModule(t, foreign[0], "Anrufe und Videocalls")) {
		assert.Equal(t, "Initiator", c["Rolle"], "only the other tenant's own call")
	}
	for _, m := range recordMaps(dsarModule(t, foreign[0], "Meeting-Teilnahmen")) {
		assert.Equal(t, "Fremdes Meeting", m["Titel"])
	}
	assert.NotContains(t, moduleNames(foreign[0]), "Meeting-Aktionspunkte")
	assert.NotContains(t, moduleNames(foreign[0]), "Dialer-Agent-Statushistorie")

	// A forged tenant id must not reach the first tenant's rows either.
	forged, err := SearchByQuery(ctxOther, pool, tenantOther, "Kollege")
	require.NoError(t, err)
	assert.Empty(t, forged)
}
