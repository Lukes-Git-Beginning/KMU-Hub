package gdpr

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Video, meeting and dialer participation of a matched user — Art. 15 GDPR.
//
// Read-only. Meeting notes and meeting chat are a separate, undecided topic
// (BACKLOG-NEXT) and deliberately not part of this file.
//
// dialer_agent_status_log grows without bound (one row per agent status
// change). Disclosure is capped at the dsarMaxRows most recent changes per
// agent, newest first; a capped list is recognisable by holding exactly
// dsarMaxRows records.
// ---------------------------------------------------------------------------

func callParticipationModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Rolle", "Art", "Status", "Gestartet", "Beigetreten", "Verlassen", "Dauer (s)"}
	return userRowsModule(ctx, pool, "call participation", "Anrufe und Videocalls", columns,
		`SELECT cs.initiator_id = $2, cs.call_type, cs.status, cs.created_at,
		        cp.joined_at, cp.left_at, cs.duration_seconds
		 FROM call_sessions cs
		 LEFT JOIN call_participants cp
		        ON cp.call_id = cs.id AND cp.tenant_id = cs.tenant_id AND cp.user_id = $2
		 WHERE cs.tenant_id = $1 AND (cs.initiator_id = $2 OR cp.user_id IS NOT NULL)
		 ORDER BY cs.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var initiator bool
			var callType, status string
			var startedAt time.Time
			var joinedAt, leftAt *time.Time
			var duration *int
			if err := rows.Scan(&initiator, &callType, &status, &startedAt, &joinedAt, &leftAt, &duration); err != nil {
				return DSARRecord{}, err
			}
			role := "Teilnehmer"
			if initiator {
				role = "Initiator"
			}
			durationText := ""
			if duration != nil {
				durationText = fmt.Sprintf("%d", *duration)
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Rolle", Value: role},
				{Key: "Art", Value: callType},
				{Key: "Status", Value: status},
				{Key: "Gestartet", Value: startedAt.Format(dsarTimeLayout)},
				{Key: "Beigetreten", Value: derefDateTimeOrEmpty(joinedAt)},
				{Key: "Verlassen", Value: derefDateTimeOrEmpty(leftAt)},
				{Key: "Dauer (s)", Value: durationText},
			}}, nil
		})
}

// meetingParticipationModule lists every way the user is tied to a meeting:
// organiser, invited attendee, co-host, or assigned to a breakout room.
func meetingParticipationModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Titel", "Rolle", "Antwort / Raum", "Beginn"}
	return userRowsModule(ctx, pool, "meeting participation", "Meeting-Teilnahmen", columns,
		`SELECT title, role, detail, scheduled_start FROM (
		   SELECT m.title, 'Organisator' AS role, '' AS detail, m.scheduled_start
		     FROM meetings m WHERE m.tenant_id = $1 AND m.organizer_id = $2
		   UNION ALL
		   SELECT m.title, 'Teilnehmer', a.rsvp_status, m.scheduled_start
		     FROM meeting_attendees a
		     JOIN meetings m ON m.id = a.meeting_id AND m.tenant_id = a.tenant_id
		    WHERE a.tenant_id = $1 AND a.user_id = $2
		   UNION ALL
		   SELECT m.title, 'Co-Host', '', m.scheduled_start
		     FROM meeting_cohosts c
		     JOIN meetings m ON m.id = c.meeting_id AND m.tenant_id = c.tenant_id
		    WHERE c.tenant_id = $1 AND c.user_id = $2
		   UNION ALL
		   SELECT m.title, 'Breakout-Raum', r.label, m.scheduled_start
		     FROM meeting_breakout_assignments b
		     JOIN meeting_breakout_rooms r ON r.id = b.breakout_room_id AND r.tenant_id = b.tenant_id
		     JOIN meetings m ON m.id = b.meeting_id AND m.tenant_id = b.tenant_id
		    WHERE b.tenant_id = $1 AND b.user_id = $2
		 ) t
		 ORDER BY scheduled_start DESC, role
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var title, role, detail string
			var start time.Time
			if err := rows.Scan(&title, &role, &detail, &start); err != nil {
				return DSARRecord{}, err
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Titel", Value: title},
				{Key: "Rolle", Value: role},
				{Key: "Antwort / Raum", Value: detail},
				{Key: "Beginn", Value: start.Format(dsarTimeLayout)},
			}}, nil
		})
}

func meetingActionItemsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Meeting", "Aktionspunkt", "Erledigt", "Angelegt"}
	return userRowsModule(ctx, pool, "meeting action items", "Meeting-Aktionspunkte", columns,
		`SELECT m.title, ai.description, ai.is_completed, ai.created_at
		 FROM meeting_action_items ai
		 JOIN meetings m ON m.id = ai.meeting_id AND m.tenant_id = ai.tenant_id
		 WHERE ai.tenant_id = $1 AND ai.assignee_id = $2
		 ORDER BY ai.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var title, description string
			var done bool
			var createdAt time.Time
			if err := rows.Scan(&title, &description, &done, &createdAt); err != nil {
				return DSARRecord{}, err
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Meeting", Value: title},
				{Key: "Aktionspunkt", Value: description},
				{Key: "Erledigt", Value: boolLabel(done)},
				{Key: "Angelegt", Value: createdAt.Format(dsarTimeLayout)},
			}}, nil
		})
}

// agentStatusHistoryModule discloses the dialer status changes recorded for
// the user as agent (working-time-adjacent employee data). Capped, see the
// file comment.
func agentStatusHistoryModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Zeitpunkt", "Status", "Vorheriger Status", "Kampagne"}
	return userRowsModule(ctx, pool, "dialer agent status log", "Dialer-Agent-Statushistorie", columns,
		`SELECT l.changed_at, l.status, COALESCE(l.previous_status, ''), COALESCE(c.name, '')
		 FROM dialer_agent_status_log l
		 LEFT JOIN dialer_campaigns c ON c.id = l.campaign_id AND c.tenant_id = l.tenant_id
		 WHERE l.tenant_id = $1 AND l.user_id = $2
		 ORDER BY l.changed_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var changedAt time.Time
			var status, previous, campaign string
			if err := rows.Scan(&changedAt, &status, &previous, &campaign); err != nil {
				return DSARRecord{}, err
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Zeitpunkt", Value: changedAt.Format(dsarTimeLayout)},
				{Key: "Status", Value: status},
				{Key: "Vorheriger Status", Value: previous},
				{Key: "Kampagne", Value: campaign},
			}}, nil
		})
}

// videoDialerModules collects the four modules for one user; empty ones are
// skipped.
func videoDialerModules(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) ([]DSARModule, error) {
	var out []DSARModule
	for _, load := range []func(context.Context, *pgxpool.Pool, uuid.UUID, uuid.UUID) (*DSARModule, error){
		callParticipationModule, meetingParticipationModule, meetingActionItemsModule, agentStatusHistoryModule,
	} {
		mod, err := load(ctx, pool, tenantID, userID)
		if err != nil {
			return nil, err
		}
		if mod != nil {
			out = append(out, *mod)
		}
	}
	return out, nil
}
