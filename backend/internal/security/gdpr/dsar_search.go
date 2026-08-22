package gdpr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/crm/advisoryprotocol"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// ---------------------------------------------------------------------------
// DSAR (Data Subject Access Request) cross-module search — Art. 15 GDPR.
// ---------------------------------------------------------------------------

// DSARField is a single key/value cell within a DSAR record. Order matters, so
// records are modelled as ordered field lists rather than maps.
type DSARField struct {
	Key   string
	Value string
}

// DSARRecord is one logical row of disclosed data.
type DSARRecord struct {
	Fields []DSARField
}

// DSARModule groups a data subject's disclosed records from one module.
type DSARModule struct {
	Module  string
	Columns []string
	Records []DSARRecord
}

// DSARPerson is a matched data subject and the data held about them.
type DSARPerson struct {
	ID      string
	Name    string
	Email   string
	Company string
	Avatar  string
	Modules []DSARModule
}

const (
	dsarMaxSubjects = 10
	dsarMaxRows     = 50
	dsarTimeLayout  = "2006-01-02 15:04"
	dsarDateLayout  = "2006-01-02"
)

// SearchByQuery performs an Art. 15 GDPR cross-module lookup for a person within
// a tenant. It matches contacts and users by name/email and, for each match,
// aggregates the data held about them across CRM, consent, dialer, finance,
// meetings, helpdesk, contracts, email, deals and activities modules. Other
// modules can still be folded in incrementally. All queries are tenant-scoped
// and run behind RLS as defense-in-depth.
func SearchByQuery(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, query string) ([]DSARPerson, error) {
	pattern := "%" + query + "%"
	persons := make([]DSARPerson, 0)

	contacts, err := matchContacts(ctx, pool, tenantID, pattern)
	if err != nil {
		return nil, err
	}
	for _, c := range contacts {
		name := joinName(c.first, c.last)
		person := DSARPerson{
			ID:      c.id.String(),
			Name:    name,
			Email:   c.email,
			Company: c.company,
			Avatar:  initials(c.first, c.last),
			Modules: []DSARModule{{
				Module:  "CRM Kontakte",
				Columns: []string{"Feld", "Wert"},
				Records: []DSARRecord{
					fieldValueRecord("Name", name),
					fieldValueRecord("E-Mail", c.email),
					fieldValueRecord("Telefon", c.phone),
					fieldValueRecord("Position", c.position),
					fieldValueRecord("Unternehmen", c.company),
					fieldValueRecord("Erstellt", c.createdAt.Format(dsarTimeLayout)),
				},
			}},
		}

		customFields, cfErr := customFieldsModule(ctx, pool, tenantID, c.id)
		if cfErr != nil {
			return nil, cfErr
		}
		if customFields != nil {
			person.Modules = append(person.Modules, *customFields)
		}

		tags, tgErr := tagsModule(ctx, pool, tenantID, c.id)
		if tgErr != nil {
			return nil, tgErr
		}
		if tags != nil {
			person.Modules = append(person.Modules, *tags)
		}

		documents, docErr := documentsModule(ctx, pool, tenantID, c.id)
		if docErr != nil {
			return nil, docErr
		}
		if documents != nil {
			person.Modules = append(person.Modules, *documents)
		}

		advisoryProtocols, apErr := advisoryProtocolModules(ctx, pool, tenantID, c.id)
		if apErr != nil {
			return nil, apErr
		}
		person.Modules = append(person.Modules, advisoryProtocols...)

		consent, cErr := consentModule(ctx, pool, c.id)
		if cErr != nil {
			return nil, cErr
		}
		if consent != nil {
			person.Modules = append(person.Modules, *consent)
		}

		calls, dErr := dialerModule(ctx, pool, tenantID, c.id)
		if dErr != nil {
			return nil, dErr
		}
		if calls != nil {
			person.Modules = append(person.Modules, *calls)
		}

		invoices, fErr := financeModule(ctx, pool, tenantID, c.id)
		if fErr != nil {
			return nil, fErr
		}
		if invoices != nil {
			person.Modules = append(person.Modules, *invoices)
		}

		meetings, mErr := meetingsModule(ctx, pool, tenantID, c.id)
		if mErr != nil {
			return nil, mErr
		}
		if meetings != nil {
			person.Modules = append(person.Modules, *meetings)
		}

		tickets, tErr := helpdeskModule(ctx, pool, tenantID, c.id)
		if tErr != nil {
			return nil, tErr
		}
		if tickets != nil {
			person.Modules = append(person.Modules, *tickets)
		}

		helpdeskMessages, hmErr := helpdeskMessagesModule(ctx, pool, tenantID, c.id)
		if hmErr != nil {
			return nil, hmErr
		}
		if helpdeskMessages != nil {
			person.Modules = append(person.Modules, *helpdeskMessages)
		}

		contracts, ctErr := contractsModule(ctx, pool, tenantID, c.id)
		if ctErr != nil {
			return nil, ctErr
		}
		if contracts != nil {
			person.Modules = append(person.Modules, *contracts)
		}

		emails, eErr := emailModule(ctx, pool, tenantID, c.id)
		if eErr != nil {
			return nil, eErr
		}
		if emails != nil {
			person.Modules = append(person.Modules, *emails)
		}

		deals, dlErr := dealsModule(ctx, pool, tenantID, c.id)
		if dlErr != nil {
			return nil, dlErr
		}
		if deals != nil {
			person.Modules = append(person.Modules, *deals)
		}

		formSubmissions, fsErr := formSubmissionsModule(ctx, pool, tenantID, c.email)
		if fsErr != nil {
			return nil, fsErr
		}
		if formSubmissions != nil {
			person.Modules = append(person.Modules, *formSubmissions)
		}

		activities, acErr := activitiesModule(ctx, pool, tenantID, c.id)
		if acErr != nil {
			return nil, acErr
		}
		if activities != nil {
			person.Modules = append(person.Modules, *activities)
		}

		persons = append(persons, person)
	}

	users, err := matchUsers(ctx, pool, tenantID, pattern)
	if err != nil {
		return nil, err
	}
	persons = append(persons, users...)

	return persons, nil
}

// ---------------------------------------------------------------------------
// Subject matching
// ---------------------------------------------------------------------------

type contactRow struct {
	id                                           uuid.UUID
	first, last, email, phone, position, company string
	createdAt                                    time.Time
}

func matchContacts(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, pattern string) ([]contactRow, error) {
	rows, err := pool.Query(ctx,
		`SELECT c.id, c.first_name, c.last_name, COALESCE(c.email, ''),
		        COALESCE(c.phone, ''), COALESCE(c.position, ''), COALESCE(co.name, ''), c.created_at
		 FROM contacts c
		 LEFT JOIN companies co ON co.id = c.company_id
		 WHERE c.tenant_id = $1
		   AND (c.first_name ILIKE $2 OR c.last_name ILIKE $2 OR c.email ILIKE $2
		        OR (c.first_name || ' ' || c.last_name) ILIKE $2)
		 ORDER BY c.created_at DESC
		 LIMIT $3`, tenantID, pattern, dsarMaxSubjects)
	if err != nil {
		return nil, fmt.Errorf("dsar: query contacts: %w", err)
	}
	defer rows.Close()

	var out []contactRow
	for rows.Next() {
		var c contactRow
		if scanErr := rows.Scan(&c.id, &c.first, &c.last, &c.email, &c.phone, &c.position, &c.company, &c.createdAt); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan contact: %w", scanErr)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: contact rows: %w", err)
	}
	return out, nil
}

type dsarUserRow struct {
	id                 uuid.UUID
	email, first, last string
	isActive           bool
	createdAt          time.Time
}

func matchUsers(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, pattern string) ([]DSARPerson, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, COALESCE(email, ''), first_name, last_name, is_active, created_at
		 FROM users
		 WHERE tenant_id = $1
		   AND (first_name ILIKE $2 OR last_name ILIKE $2 OR email ILIKE $2
		        OR (first_name || ' ' || last_name) ILIKE $2)
		 ORDER BY created_at DESC
		 LIMIT $3`, tenantID, pattern, dsarMaxSubjects)
	if err != nil {
		return nil, fmt.Errorf("dsar: query users: %w", err)
	}

	var users []dsarUserRow
	for rows.Next() {
		var u dsarUserRow
		if scanErr := rows.Scan(&u.id, &u.email, &u.first, &u.last, &u.isActive, &u.createdAt); scanErr != nil {
			rows.Close()
			return nil, fmt.Errorf("dsar: scan user: %w", scanErr)
		}
		users = append(users, u)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return nil, fmt.Errorf("dsar: user rows: %w", rowsErr)
	}

	// Enrichment queries run after the user rows are collected and rows.Close()
	// has run, matching the contacts path (matchContacts returns a slice before
	// SearchByQuery layers on per-contact modules) rather than nesting a second
	// pool.Query while the first result set is still open.
	var out []DSARPerson
	for _, u := range users {
		name := joinName(u.first, u.last)
		person := DSARPerson{
			ID:     u.id.String(),
			Name:   name,
			Email:  u.email,
			Avatar: initials(u.first, u.last),
			Modules: []DSARModule{{
				Module:  "Benutzerkonto",
				Columns: []string{"Feld", "Wert"},
				Records: []DSARRecord{
					fieldValueRecord("Name", name),
					fieldValueRecord("E-Mail", u.email),
					fieldValueRecord("Aktiv", boolLabel(u.isActive)),
					fieldValueRecord("Erstellt", u.createdAt.Format(dsarTimeLayout)),
				},
			}},
		}

		chatMessages, cmErr := chatMessagesModule(ctx, pool, tenantID, u.id)
		if cmErr != nil {
			return nil, cmErr
		}
		if chatMessages != nil {
			person.Modules = append(person.Modules, *chatMessages)
		}

		chatMemberships, chErr := chatMembershipsModule(ctx, pool, tenantID, u.id)
		if chErr != nil {
			return nil, chErr
		}
		if chatMemberships != nil {
			person.Modules = append(person.Modules, *chatMemberships)
		}

		tasks, tErr := tasksModule(ctx, pool, tenantID, u.id)
		if tErr != nil {
			return nil, tErr
		}
		if tasks != nil {
			person.Modules = append(person.Modules, *tasks)
		}

		taskComments, tcErr := taskCommentsModule(ctx, pool, tenantID, u.id)
		if tcErr != nil {
			return nil, tcErr
		}
		if taskComments != nil {
			person.Modules = append(person.Modules, *taskComments)
		}

		timeEntries, teErr := timeEntriesModule(ctx, pool, tenantID, u.id)
		if teErr != nil {
			return nil, teErr
		}
		if timeEntries != nil {
			person.Modules = append(person.Modules, *timeEntries)
		}

		calendarEvents, ceErr := calendarEventsModule(ctx, pool, tenantID, u.id)
		if ceErr != nil {
			return nil, ceErr
		}
		if calendarEvents != nil {
			person.Modules = append(person.Modules, *calendarEvents)
		}

		calendarPrefs, cpErr := calendarPreferencesModule(ctx, pool, tenantID, u.id)
		if cpErr != nil {
			return nil, cpErr
		}
		if calendarPrefs != nil {
			person.Modules = append(person.Modules, *calendarPrefs)
		}

		notifications, nErr := notificationsModule(ctx, pool, tenantID, u.id)
		if nErr != nil {
			return nil, nErr
		}
		if notifications != nil {
			person.Modules = append(person.Modules, *notifications)
		}

		notificationPrefs, npErr := notificationPreferencesModule(ctx, pool, tenantID, u.id)
		if npErr != nil {
			return nil, npErr
		}
		if notificationPrefs != nil {
			person.Modules = append(person.Modules, *notificationPrefs)
		}

		quietHours, qhErr := notificationQuietHoursModule(ctx, pool, tenantID, u.id)
		if qhErr != nil {
			return nil, qhErr
		}
		if quietHours != nil {
			person.Modules = append(person.Modules, *quietHours)
		}

		mutes, muErr := notificationMutesModule(ctx, pool, tenantID, u.id)
		if muErr != nil {
			return nil, muErr
		}
		if mutes != nil {
			person.Modules = append(person.Modules, *mutes)
		}

		hrProfile, hpErr := hrProfileModule(ctx, pool, tenantID, u.id)
		if hpErr != nil {
			return nil, hpErr
		}
		if hrProfile != nil {
			person.Modules = append(person.Modules, *hrProfile)
		}

		hrDocuments, hdErr := hrEmployeeDocumentsModule(ctx, pool, tenantID, u.id)
		if hdErr != nil {
			return nil, hdErr
		}
		if hrDocuments != nil {
			person.Modules = append(person.Modules, *hrDocuments)
		}

		hrChangeRequests, hcErr := hrProfileChangeRequestsModule(ctx, pool, tenantID, u.id)
		if hcErr != nil {
			return nil, hcErr
		}
		if hrChangeRequests != nil {
			person.Modules = append(person.Modules, *hrChangeRequests)
		}

		sessions, sessErr := userSessionsModule(ctx, pool, tenantID, u.id)
		if sessErr != nil {
			return nil, sessErr
		}
		if sessions != nil {
			person.Modules = append(person.Modules, *sessions)
		}

		accountSecurity, asErr := accountSecurityModule(ctx, pool, tenantID, u.id)
		if asErr != nil {
			return nil, asErr
		}
		if accountSecurity != nil {
			person.Modules = append(person.Modules, *accountSecurity)
		}

		driverLicenses, dlErr := driverLicensesModule(ctx, pool, tenantID, u.id)
		if dlErr != nil {
			return nil, dlErr
		}
		if driverLicenses != nil {
			person.Modules = append(person.Modules, *driverLicenses)
		}

		vehicleBookings, vbErr := vehicleBookingsModule(ctx, pool, tenantID, u.id)
		if vbErr != nil {
			return nil, vbErr
		}
		if vehicleBookings != nil {
			person.Modules = append(person.Modules, *vehicleBookings)
		}

		out = append(out, person)
	}
	return out, nil
}

// chatMessagesModule discloses a matched user's own chat message content —
// not the messages of other participants in a shared channel. Scope mirrors
// ChatErasureHandler.ExecuteErasure, which anonymizes exactly the rows
// WHERE created_by = userID (erasure.go:296).
func chatMessagesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT c.name, m.content, m.created_at
		 FROM messages m
		 JOIN channels c ON c.id = m.channel_id
		 WHERE m.tenant_id = $1 AND m.created_by = $2 AND NOT m.is_deleted
		 ORDER BY m.created_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows+1)
	if err != nil {
		return nil, fmt.Errorf("dsar: query chat messages: %w", err)
	}
	defer rows.Close()

	type message struct {
		channel, content string
		at               time.Time
	}
	var messages []message
	for rows.Next() {
		var m message
		if scanErr := rows.Scan(&m.channel, &m.content, &m.at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan chat message: %w", scanErr)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: chat message rows: %w", err)
	}
	if len(messages) == 0 {
		return nil, nil
	}

	// Query fetched dsarMaxRows+1 newest-first so truncation, if any, drops
	// the oldest messages rather than silently hiding the most recent ones.
	truncated := len(messages) > dsarMaxRows
	if truncated {
		messages = messages[:dsarMaxRows]
	}
	// Reverse to chronological (oldest-first) order for a readable transcript.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	mod := DSARModule{Module: "Chat-Nachrichten", Columns: []string{"Kanal", "Nachricht", "Datum"}}
	if truncated {
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Kanal", Value: ""},
			{Key: "Nachricht", Value: fmt.Sprintf("(gekürzt auf die %d neuesten Nachrichten)", dsarMaxRows)},
			{Key: "Datum", Value: ""},
		}})
	}
	for _, m := range messages {
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Kanal", Value: m.channel},
			{Key: "Nachricht", Value: m.content},
			{Key: "Datum", Value: m.at.Format(dsarTimeLayout)},
		}})
	}
	return &mod, nil
}

// chatMembershipsModule discloses which channels a matched user belongs to —
// the second table ChatErasureHandler.ExecuteErasure clears, DELETE FROM
// channel_memberships WHERE user_id = userID (erasure.go:308).
func chatMembershipsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT c.name, cm.role::text, cm.joined_at
		 FROM channel_memberships cm
		 JOIN channels c ON c.id = cm.channel_id
		 WHERE cm.tenant_id = $1 AND cm.user_id = $2
		 ORDER BY cm.joined_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query chat memberships: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Chat-Kanalmitgliedschaften", Columns: []string{"Kanal", "Rolle", "Beigetreten"}}
	for rows.Next() {
		var channel, role string
		var joinedAt time.Time
		if scanErr := rows.Scan(&channel, &role, &joinedAt); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan chat membership: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Kanal", Value: channel},
			{Key: "Rolle", Value: role},
			{Key: "Beigetreten", Value: joinedAt.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: chat membership rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// tasksModule discloses tasks a matched user created or was assigned — both
// directions count, since either makes the task the subject's personal data.
// Scope mirrors WorkErasureHandler.ExecuteErasure, which touches exactly the
// rows WHERE assignee_id = userID OR created_by = userID (erasure.go:385).
func tasksModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT t.title, COALESCE(ps.name, ''), (t.assignee_id IS NOT DISTINCT FROM $2), (t.created_by = $2), t.created_at
		 FROM tasks t
		 LEFT JOIN project_statuses ps ON ps.id = t.status_id
		 WHERE t.tenant_id = $1 AND (t.assignee_id = $2 OR t.created_by = $2)
		 ORDER BY t.created_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query tasks: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Aufgaben", Columns: []string{"Titel", "Status", "Rolle", "Erstellt"}}
	for rows.Next() {
		var title, status string
		var isAssignee, isCreator bool
		var createdAt time.Time
		if scanErr := rows.Scan(&title, &status, &isAssignee, &isCreator, &createdAt); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan task: %w", scanErr)
		}
		var roles []string
		if isCreator {
			roles = append(roles, "Ersteller")
		}
		if isAssignee {
			roles = append(roles, "Zugewiesen")
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Titel", Value: title},
			{Key: "Status", Value: status},
			{Key: "Rolle", Value: strings.Join(roles, ", ")},
			{Key: "Erstellt", Value: createdAt.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: task rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// taskCommentsModule discloses a matched user's own task comments — the
// second table WorkErasureHandler.ExecuteErasure anonymizes, WHERE
// author_id = userID (erasure.go:406).
func taskCommentsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT t.title, tc.content, tc.created_at
		 FROM task_comments tc
		 JOIN tasks t ON t.id = tc.task_id
		 WHERE tc.tenant_id = $1 AND tc.author_id = $2
		 ORDER BY tc.created_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows+1)
	if err != nil {
		return nil, fmt.Errorf("dsar: query task comments: %w", err)
	}
	defer rows.Close()

	type comment struct {
		task, content string
		at            time.Time
	}
	var comments []comment
	for rows.Next() {
		var c comment
		if scanErr := rows.Scan(&c.task, &c.content, &c.at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan task comment: %w", scanErr)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: task comment rows: %w", err)
	}
	if len(comments) == 0 {
		return nil, nil
	}

	// Query fetched dsarMaxRows+1 newest-first so truncation, if any, drops
	// the oldest comments rather than silently hiding the most recent ones.
	truncated := len(comments) > dsarMaxRows
	if truncated {
		comments = comments[:dsarMaxRows]
	}
	for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
		comments[i], comments[j] = comments[j], comments[i]
	}

	mod := DSARModule{Module: "Aufgaben-Kommentare", Columns: []string{"Aufgabe", "Kommentar", "Datum"}}
	if truncated {
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Aufgabe", Value: ""},
			{Key: "Kommentar", Value: fmt.Sprintf("(gekürzt auf die %d neuesten Kommentare)", dsarMaxRows)},
			{Key: "Datum", Value: ""},
		}})
	}
	for _, c := range comments {
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Aufgabe", Value: c.task},
			{Key: "Kommentar", Value: c.content},
			{Key: "Datum", Value: c.at.Format(dsarTimeLayout)},
		}})
	}
	return &mod, nil
}

// timeEntriesModule discloses a matched user's tracked work time — the third
// table WorkErasureHandler.ExecuteErasure deletes outright, WHERE
// user_id = userID (erasure.go:394). Individual entries can accumulate into
// the thousands for a long-tenured user, so this module aggregates by month
// rather than truncating a row list — the aggregation is named in the module
// title so it never reads as a raw, complete entry list.
func timeEntriesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT to_char(date_trunc('month', started_at), 'YYYY-MM'),
		        COUNT(*), COALESCE(SUM(duration_seconds), 0)
		 FROM time_entries
		 WHERE tenant_id = $1 AND user_id = $2
		 GROUP BY 1
		 ORDER BY 1 DESC`, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("dsar: query time entries: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Zeiterfassung (aggregiert pro Monat)", Columns: []string{"Monat", "Eintraege", "Dauer"}}
	for rows.Next() {
		var month string
		var count, totalSeconds int
		if scanErr := rows.Scan(&month, &count, &totalSeconds); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan time entry: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Monat", Value: month},
			{Key: "Eintraege", Value: strconv.Itoa(count)},
			{Key: "Dauer", Value: fmt.Sprintf("%.1f Std.", float64(totalSeconds)/3600)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: time entry rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// calendarEventsModule discloses calendar events a matched user created or
// attends. Both event_attendees (participation) and calendar_events
// (created_by) are the two tables CalendarErasureHandler.ExecuteErasure
// touches for events (erasure.go:461); the third, user_calendar_preferences,
// is its own module below. Only the subject's own role and RSVP are
// disclosed — an event with other attendees does not list who else is on it,
// same rule as chat rooms and calendar entries with multiple participants.
func calendarEventsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT e.title, e.start_time, COALESCE(e.location, ''),
		        (e.created_by = $2) AS is_creator,
		        ea.rsvp_status
		 FROM calendar_events e
		 LEFT JOIN event_attendees ea ON ea.event_id = e.id AND ea.user_id = $2
		 WHERE e.tenant_id = $1 AND (e.created_by = $2 OR ea.user_id = $2)
		 ORDER BY e.start_time DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query calendar events: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Kalendertermine", Columns: []string{"Titel", "Beginn", "Ort", "Rolle"}}
	for rows.Next() {
		var title, location string
		var start time.Time
		var isCreator bool
		var rsvpStatus *string
		if scanErr := rows.Scan(&title, &start, &location, &isCreator, &rsvpStatus); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan calendar event: %w", scanErr)
		}
		var roles []string
		if isCreator {
			roles = append(roles, "Ersteller")
		}
		if rsvpStatus != nil {
			roles = append(roles, "Teilnehmer ("+*rsvpStatus+")")
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Titel", Value: title},
			{Key: "Beginn", Value: start.Format(dsarTimeLayout)},
			{Key: "Ort", Value: location},
			{Key: "Rolle", Value: strings.Join(roles, ", ")},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: calendar event rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// calendarPreferencesModule discloses a matched user's calendar defaults —
// the third table CalendarErasureHandler.ExecuteErasure deletes outright
// (erasure.go:490). One row per user at most, so it renders like the
// "Benutzerkonto" field/value module rather than a record list.
func calendarPreferencesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	var defaultView, subdivisionCode *string
	var weekDays, reminderMinutes, alldayReminderMinutes int
	var showTaskDeadlines bool
	err := pool.QueryRow(ctx,
		`SELECT default_view, week_days, default_reminder_minutes, default_allday_reminder_minutes,
		        subdivision_code, show_task_deadlines
		 FROM user_calendar_preferences
		 WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID,
	).Scan(&defaultView, &weekDays, &reminderMinutes, &alldayReminderMinutes, &subdivisionCode, &showTaskDeadlines)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("dsar: query calendar preferences: %w", err)
	}

	view := ""
	if defaultView != nil {
		view = *defaultView
	}
	subdivision := ""
	if subdivisionCode != nil {
		subdivision = *subdivisionCode
	}
	mod := DSARModule{Module: "Kalendereinstellungen", Columns: []string{"Feld", "Wert"}, Records: []DSARRecord{
		fieldValueRecord("Standardansicht", view),
		fieldValueRecord("Wochentage", strconv.Itoa(weekDays)),
		fieldValueRecord("Erinnerung (Minuten)", strconv.Itoa(reminderMinutes)),
		fieldValueRecord("Erinnerung ganztaegig (Minuten)", strconv.Itoa(alldayReminderMinutes)),
		fieldValueRecord("Feiertagsregion", subdivision),
		fieldValueRecord("Aufgaben-Termine anzeigen", boolLabel(showTaskDeadlines)),
	}}
	return &mod, nil
}

// notificationsModule discloses a matched user's delivered notifications —
// the first table NotificationErasureHandler.ExecuteErasure deletes
// (erasure.go:560).
func notificationsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT title, COALESCE(body, ''), created_at, is_read
		 FROM notifications
		 WHERE tenant_id = $1 AND user_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows+1)
	if err != nil {
		return nil, fmt.Errorf("dsar: query notifications: %w", err)
	}
	defer rows.Close()

	type notif struct {
		title, body string
		at          time.Time
		read        bool
	}
	var notifs []notif
	for rows.Next() {
		var n notif
		if scanErr := rows.Scan(&n.title, &n.body, &n.at, &n.read); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan notification: %w", scanErr)
		}
		notifs = append(notifs, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: notification rows: %w", err)
	}
	if len(notifs) == 0 {
		return nil, nil
	}

	truncated := len(notifs) > dsarMaxRows
	if truncated {
		notifs = notifs[:dsarMaxRows]
	}

	mod := DSARModule{Module: "Benachrichtigungen", Columns: []string{"Titel", "Text", "Datum", "Gelesen"}}
	for _, n := range notifs {
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Titel", Value: n.title},
			{Key: "Text", Value: n.body},
			{Key: "Datum", Value: n.at.Format(dsarTimeLayout)},
			{Key: "Gelesen", Value: boolLabel(n.read)},
		}})
	}
	if truncated {
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Titel", Value: fmt.Sprintf("(gekürzt auf die %d neuesten Benachrichtigungen)", dsarMaxRows)},
			{Key: "Text", Value: ""},
			{Key: "Datum", Value: ""},
			{Key: "Gelesen", Value: ""},
		}})
	}
	return &mod, nil
}

// notificationPreferencesModule discloses a matched user's per-event-type
// and per-module delivery settings — the second table
// NotificationErasureHandler.ExecuteErasure deletes (erasure.go:567).
func notificationPreferencesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT COALESCE(event_type_key, ''), COALESCE(module_id, ''), in_app, desktop_push, COALESCE(sound, '')
		 FROM notification_preferences
		 WHERE tenant_id = $1 AND user_id = $2
		 ORDER BY module_id, event_type_key
		 LIMIT $3`, tenantID, userID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query notification preferences: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Benachrichtigungseinstellungen", Columns: []string{"Ereignistyp", "Modul", "In-App", "Desktop", "Ton"}}
	for rows.Next() {
		var eventType, moduleID, sound string
		var inApp, desktopPush bool
		if scanErr := rows.Scan(&eventType, &moduleID, &inApp, &desktopPush, &sound); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan notification preference: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Ereignistyp", Value: eventType},
			{Key: "Modul", Value: moduleID},
			{Key: "In-App", Value: boolLabel(inApp)},
			{Key: "Desktop", Value: boolLabel(desktopPush)},
			{Key: "Ton", Value: sound},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: notification preference rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// notificationQuietHoursModule discloses a matched user's do-not-disturb
// schedule — the third table NotificationErasureHandler.ExecuteErasure
// deletes (erasure.go:574). At most one row per user (UNIQUE(user_id)), so
// it renders as a field/value module.
func notificationQuietHoursModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	var startTime, endTime, timezone string
	var enabled, manualDND bool
	err := pool.QueryRow(ctx,
		`SELECT start_time::text, end_time::text, timezone, enabled, manual_dnd
		 FROM notification_quiet_hours
		 WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID,
	).Scan(&startTime, &endTime, &timezone, &enabled, &manualDND)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("dsar: query quiet hours: %w", err)
	}

	mod := DSARModule{Module: "Ruhezeiten", Columns: []string{"Feld", "Wert"}, Records: []DSARRecord{
		fieldValueRecord("Aktiviert", boolLabel(enabled)),
		fieldValueRecord("Von", startTime),
		fieldValueRecord("Bis", endTime),
		fieldValueRecord("Zeitzone", timezone),
		fieldValueRecord("Manuell aktiv", boolLabel(manualDND)),
	}}
	return &mod, nil
}

// notificationMutesModule discloses which resources a matched user has
// muted — the fourth table NotificationErasureHandler.ExecuteErasure
// deletes (erasure.go:581).
func notificationMutesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT module_id, resource_id, created_at
		 FROM notification_mutes
		 WHERE tenant_id = $1 AND user_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query notification mutes: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Stummgeschaltete Ressourcen", Columns: []string{"Modul", "Ressource", "Seit"}}
	for rows.Next() {
		var moduleID, resourceID string
		var createdAt time.Time
		if scanErr := rows.Scan(&moduleID, &resourceID, &createdAt); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan notification mute: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Modul", Value: moduleID},
			{Key: "Ressource", Value: resourceID},
			{Key: "Seit", Value: createdAt.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: notification mute rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// hrProfileModule discloses the matched user's employment record --
// department, contract terms, address, emergency contact, hourly rate,
// exit data and the is_minor flag. hr_employee_profiles carries a UNIQUE
// constraint on user_id (uq_hr_employee_user), so at most one row exists.
//
// is_minor is disclosed as a plain boolean like any other field: the schema
// has no separate guardian/representative contact to route the disclosure
// through, and Art. 15 does not carve out a different mechanism for a minor
// employee's own record -- the special handling GDPR requires for minors is
// about consent and parental notice at data collection, not about narrowing
// what a later access request returns.
func hrProfileModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	var (
		department, position, contractType, emergencyName, emergencyPhone                   *string
		addressStreet, addressCity, addressPostalCode, addressCountry, exitType, exitReason *string
		managerFirst, managerLast                                                           *string
		workDaysPerWeek, annualLeaveDays                                                    int
		startDate                                                                           time.Time
		lastWorkDay, exitDate                                                               *time.Time
		hourlyRate                                                                          *float64
		status                                                                              string
		isMinor                                                                             bool
	)
	err := pool.QueryRow(ctx,
		`SELECT hep.department, hep.position_title, hep.contract_type, hep.work_days_per_week,
		        hep.annual_leave_days, m.first_name, m.last_name, hep.start_date,
		        hep.emergency_contact_name, hep.emergency_contact_phone, hep.address_street,
		        hep.address_city, hep.address_postal_code, hep.address_country, hep.hourly_rate,
		        hep.status, hep.last_work_day, hep.exit_date, hep.exit_type, hep.exit_reason,
		        hep.is_minor
		 FROM hr_employee_profiles hep
		 LEFT JOIN users m ON m.id = hep.manager_user_id
		 WHERE hep.tenant_id = $1 AND hep.user_id = $2`, tenantID, userID,
	).Scan(&department, &position, &contractType, &workDaysPerWeek, &annualLeaveDays,
		&managerFirst, &managerLast, &startDate, &emergencyName, &emergencyPhone, &addressStreet,
		&addressCity, &addressPostalCode, &addressCountry, &hourlyRate, &status, &lastWorkDay,
		&exitDate, &exitType, &exitReason, &isMinor)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("dsar: query hr employee profile: %w", err)
	}

	manager := ""
	if managerFirst != nil || managerLast != nil {
		manager = joinName(derefOrEmpty(managerFirst), derefOrEmpty(managerLast))
	}
	address := strings.TrimSpace(strings.Join([]string{
		derefOrEmpty(addressStreet), derefOrEmpty(addressPostalCode) + " " + derefOrEmpty(addressCity),
		derefOrEmpty(addressCountry),
	}, ", "))

	return &DSARModule{
		Module:  "Personalprofil",
		Columns: []string{"Feld", "Wert"},
		Records: []DSARRecord{
			fieldValueRecord("Abteilung", derefOrEmpty(department)),
			fieldValueRecord("Position", derefOrEmpty(position)),
			fieldValueRecord("Vertragsart", derefOrEmpty(contractType)),
			fieldValueRecord("Arbeitstage pro Woche", strconv.Itoa(workDaysPerWeek)),
			fieldValueRecord("Urlaubsanspruch (Tage/Jahr)", strconv.Itoa(annualLeaveDays)),
			fieldValueRecord("Vorgesetzte(r)", manager),
			fieldValueRecord("Eintrittsdatum", startDate.Format(dsarDateLayout)),
			fieldValueRecord("Notfallkontakt", derefOrEmpty(emergencyName)),
			fieldValueRecord("Notfallkontakt Telefon", derefOrEmpty(emergencyPhone)),
			fieldValueRecord("Adresse", address),
			fieldValueRecord("Stundenlohn", moneyOrEmpty(hourlyRate)),
			fieldValueRecord("Status", status),
			fieldValueRecord("Letzter Arbeitstag", derefDateOrEmpty(lastWorkDay)),
			fieldValueRecord("Austrittsdatum", derefDateOrEmpty(exitDate)),
			fieldValueRecord("Austrittsart", derefOrEmpty(exitType)),
			fieldValueRecord("Austrittsgrund", derefOrEmpty(exitReason)),
			fieldValueRecord("Minderjährig", boolLabel(isMinor)),
		},
	}, nil
}

func derefDateOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(dsarDateLayout)
}

// hrEmployeeDocumentsModule discloses metadata only for personnel documents
// -- title, filename, category, size, upload/expiry dates -- never the file
// content, path or a download URL, same rule as documentsModule for contact
// files: a disclosure export leaves the building, and a link inside it would
// be a route to the document store that bypasses its own access checks.
//
// Unlike every other table this file reads, hr_employee_documents carries a
// role-gated RLS policy (hr_document_access, migration 000127/000128) on top
// of tenant_isolation: a plain tenant-scoped caller only sees rows their own
// HR role or self-access would allow, not everything in the tenant. A DSAR
// search is a legal disclosure obligation triggered by a security/compliance
// permission already checked upstream (DSARSearch RPC), not an HR workflow
// -- it must see every document regardless of the caller's HR role, so the
// read runs under sysctx like RunScheduledRetention does in the same
// package (retention_scheduler.go) for the identical reason. The explicit
// tenant_id and employee_id predicates below still do the real scoping.
func hrEmployeeDocumentsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(sysctx.With(ctx),
		`SELECT hd.title, hd.file_name, dc.name, hd.file_size, hd.created_at, hd.expires_at
		 FROM hr_employee_documents hd
		 JOIN hr_document_categories dc ON dc.id = hd.category_id
		 WHERE hd.tenant_id = $1 AND hd.employee_id = $2
		 ORDER BY hd.created_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query hr employee documents: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{
		Module:  "Personaldokumente (Metadaten)",
		Columns: []string{"Titel", "Dateiname", "Kategorie", "Größe", "Hochgeladen am", "Läuft ab am"},
	}
	for rows.Next() {
		var title, fileName, category, fileSize string
		var uploadedAt time.Time
		var expiresAt *time.Time
		if scanErr := rows.Scan(&title, &fileName, &category, &fileSize, &uploadedAt, &expiresAt); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan hr employee document: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Titel", Value: title},
			{Key: "Dateiname", Value: fileName},
			{Key: "Kategorie", Value: category},
			{Key: "Größe", Value: fileSize},
			{Key: "Hochgeladen am", Value: uploadedAt.Format(dsarTimeLayout)},
			{Key: "Läuft ab am", Value: derefDateOrEmpty(expiresAt)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: hr employee document rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// hrProfileChangeRequestsModule discloses the matched user's self-service
// profile change history -- what they asked to change, the old and new
// value, and how it was decided. Both drawers (personal, job) share one
// table and are disclosed together; the drawer value tells them apart.
func hrProfileChangeRequestsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT drawer, field_label, old_value, new_value, status, reason, created_at, decided_at
		 FROM hr_profile_change_requests
		 WHERE tenant_id = $1 AND user_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query hr profile change requests: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{
		Module: "Änderungsanträge (Profil)",
		Columns: []string{
			"Bereich", "Feld", "Alter Wert", "Neuer Wert", "Status", "Grund", "Erstellt am", "Entschieden am",
		},
	}
	for rows.Next() {
		var drawer, fieldLabel, oldValue, newValue, status, reason string
		var createdAt time.Time
		var decidedAt *time.Time
		if scanErr := rows.Scan(&drawer, &fieldLabel, &oldValue, &newValue, &status, &reason, &createdAt, &decidedAt); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan hr profile change request: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Bereich", Value: drawer},
			{Key: "Feld", Value: fieldLabel},
			{Key: "Alter Wert", Value: oldValue},
			{Key: "Neuer Wert", Value: newValue},
			{Key: "Status", Value: status},
			{Key: "Grund", Value: reason},
			{Key: "Erstellt am", Value: createdAt.Format(dsarTimeLayout)},
			{Key: "Entschieden am", Value: derefDateTimeOrEmpty(decidedAt)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: hr profile change request rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// userSessionsModule discloses a matched user's login/session history --
// device, IP address, location, last activity and whether the underlying
// refresh token is still active. host(ip_address) mirrors the pattern
// AuthRepository already uses for reading this inet column
// (internal/auth/postgres_repository.go) -- binding it as anything but text
// trips the pgx inet/net.IPNet mismatch that pattern exists to avoid.
// Status is computed in SQL against now() rather than in Go so the
// active/expired cutoff can't drift from the database's own clock, the same
// authority every session-expiry check in the auth package already defers
// to.
func userSessionsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT COALESCE(us.device_name, ''), COALESCE(us.device_type, ''),
		        COALESCE(host(us.ip_address), ''), COALESCE(us.location, ''), us.last_active_at,
		        CASE
		          WHEN rt.id IS NULL THEN 'Beendet'
		          WHEN rt.revoked THEN 'Widerrufen'
		          WHEN rt.expires_at < now() THEN 'Abgelaufen'
		          ELSE 'Aktiv'
		        END AS status
		 FROM user_sessions us
		 LEFT JOIN refresh_tokens rt ON rt.id = us.refresh_token_id
		 WHERE us.tenant_id = $1 AND us.user_id = $2
		 ORDER BY us.last_active_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query user sessions: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{
		Module:  "Sitzungsverlauf",
		Columns: []string{"Gerät", "Typ", "IP-Adresse", "Standort", "Letzte Aktivität", "Status"},
	}
	for rows.Next() {
		var deviceName, deviceType, ip, location, status string
		var lastActive time.Time
		if scanErr := rows.Scan(&deviceName, &deviceType, &ip, &location, &lastActive, &status); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan user session: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Gerät", Value: deviceName},
			{Key: "Typ", Value: deviceType},
			{Key: "IP-Adresse", Value: ip},
			{Key: "Standort", Value: location},
			{Key: "Letzte Aktivität", Value: lastActive.Format(dsarTimeLayout)},
			{Key: "Status", Value: status},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: user session rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// accountSecurityModule discloses account-level security status for a
// matched user: 2FA enrollment, password change history and recovery code
// usage -- counts and timestamps only, never a password hash, a TOTP secret
// or a usable recovery code value.
//
// Unlike most modules in this file, this one never returns nil for an empty
// result: two_factor_enabled is a NOT NULL column on users itself, so "never
// changed a password, never enabled 2FA" is itself the disclosed status, not
// an absence of data -- the same reasoning that keeps the base
// "Benutzerkonto" module in matchUsers unconditional.
//
// Deviates from the unit's premise: two_factor_policy is a per-ROLE
// enforcement setting (unique on tenant_id+role_name), not a per-user
// status, so it names something the tenant configured for a role, not
// personal data of this user -- it is deliberately left out. The user's own
// 2FA state lives on the users row instead (two_factor_enabled,
// two_factor_enabled_at).
func accountSecurityModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	var twoFactorEnabled bool
	var twoFactorEnabledAt *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT two_factor_enabled, two_factor_enabled_at FROM users WHERE tenant_id = $1 AND id = $2`,
		tenantID, userID,
	).Scan(&twoFactorEnabled, &twoFactorEnabledAt); err != nil {
		return nil, fmt.Errorf("dsar: query user two-factor status: %w", err)
	}

	var passwordChanges int
	var lastPasswordChange *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT count(*), max(created_at) FROM password_history WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	).Scan(&passwordChanges, &lastPasswordChange); err != nil {
		return nil, fmt.Errorf("dsar: query password history: %w", err)
	}

	var recoveryCodesTotal, recoveryCodesUsed int
	if err := pool.QueryRow(ctx,
		`SELECT count(*), count(*) FILTER (WHERE used_at IS NOT NULL)
		 FROM recovery_codes WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	).Scan(&recoveryCodesTotal, &recoveryCodesUsed); err != nil {
		return nil, fmt.Errorf("dsar: query recovery codes: %w", err)
	}

	return &DSARModule{
		Module:  "Kontosicherheit",
		Columns: []string{"Feld", "Wert"},
		Records: []DSARRecord{
			fieldValueRecord("Zwei-Faktor-Authentifizierung", boolLabel(twoFactorEnabled)),
			fieldValueRecord("Aktiviert am", derefDateTimeOrEmpty(twoFactorEnabledAt)),
			fieldValueRecord("Passwortänderungen (Anzahl)", strconv.Itoa(passwordChanges)),
			fieldValueRecord("Letzte Passwortänderung", derefDateTimeOrEmpty(lastPasswordChange)),
			fieldValueRecord("Wiederherstellungscodes (gesamt)", strconv.Itoa(recoveryCodesTotal)),
			fieldValueRecord("Wiederherstellungscodes (genutzt)", strconv.Itoa(recoveryCodesUsed)),
		},
	}, nil
}

// driverLicensesModule discloses a matched user's driver license checks
// (Führerscheinkontrolle) when they are on record as a fuhrpark driver.
// driver_licenses.driver_id is CASCADE to users, so the disclosure and the
// deletion agree on what exists -- unlike matchUsers itself before this
// module, which knew nothing about this table.
func driverLicensesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT license_classes, expiry_date, checked_at, next_check_due_date, COALESCE(notes, '')
		 FROM driver_licenses
		 WHERE tenant_id = $1 AND driver_id = $2
		 ORDER BY checked_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query driver licenses: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{
		Module:  "Führerscheinkontrolle",
		Columns: []string{"Führerscheinklassen", "Ablaufdatum", "Geprüft am", "Nächste Prüfung fällig", "Notizen"},
	}
	for rows.Next() {
		var classes []string
		var expiry *time.Time
		var checkedAt, nextCheckDue time.Time
		var notes string
		if scanErr := rows.Scan(&classes, &expiry, &checkedAt, &nextCheckDue, &notes); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan driver license: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Führerscheinklassen", Value: strings.Join(classes, ", ")},
			{Key: "Ablaufdatum", Value: derefDateOrEmpty(expiry)},
			{Key: "Geprüft am", Value: checkedAt.Format(dsarDateLayout)},
			{Key: "Nächste Prüfung fällig", Value: nextCheckDue.Format(dsarDateLayout)},
			{Key: "Notizen", Value: notes},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: driver license rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// vehicleBookingsModule discloses a matched user's pool vehicle reservations
// -- both bookings made FOR them (vehicle_bookings.user_id) and bookings
// they made for someone else (vehicle_bookings.created_by). The two columns
// name different roles (the person the vehicle is reserved for vs. the
// person who booked it) and can point at different users, so each record
// carries its role explicitly rather than assuming they're the same person.
func vehicleBookingsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT v.license_plate, v.make, v.model, vb.starts_at, vb.ends_at, vb.purpose, vb.status,
		        vb.user_id = $2, vb.created_by = $2
		 FROM vehicle_bookings vb
		 JOIN vehicles v ON v.id = vb.vehicle_id
		 WHERE vb.tenant_id = $1 AND (vb.user_id = $2 OR vb.created_by = $2)
		 ORDER BY vb.starts_at DESC
		 LIMIT $3`, tenantID, userID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query vehicle bookings: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{
		Module:  "Fahrzeugbuchungen",
		Columns: []string{"Fahrzeug", "Beginn", "Ende", "Zweck", "Status", "Rolle"},
	}
	for rows.Next() {
		var plate, make_, model, purpose, status string
		var starts, ends time.Time
		var reservedForSubject, bookedBySubject bool
		if scanErr := rows.Scan(&plate, &make_, &model, &starts, &ends, &purpose, &status,
			&reservedForSubject, &bookedBySubject); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan vehicle booking: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Fahrzeug", Value: fmt.Sprintf("%s %s (%s)", make_, model, plate)},
			{Key: "Beginn", Value: starts.Format(dsarTimeLayout)},
			{Key: "Ende", Value: ends.Format(dsarTimeLayout)},
			{Key: "Zweck", Value: purpose},
			{Key: "Status", Value: status},
			{Key: "Rolle", Value: vehicleBookingRoleLabel(reservedForSubject, bookedBySubject)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: vehicle booking rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func vehicleBookingRoleLabel(reservedForSubject, bookedBySubject bool) string {
	switch {
	case reservedForSubject && bookedBySubject:
		return "Fahrer und Buchender"
	case reservedForSubject:
		return "Fahrer"
	case bookedBySubject:
		return "Buchender"
	default:
		return ""
	}
}

func derefDateTimeOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(dsarTimeLayout)
}

// ---------------------------------------------------------------------------
// Per-module aggregation for a matched contact
// ---------------------------------------------------------------------------

func consentModule(ctx context.Context, pool *pgxpool.Pool, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT consent_type::text, granted, legal_basis::text,
		        COALESCE(granted_at, created_at), revoked_at
		 FROM consent_records
		 WHERE contact_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`, contactID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query consent: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Einwilligungen", Columns: []string{"Typ", "Status", "Rechtsgrundlage", "Datum"}}
	for rows.Next() {
		var ctype, basis string
		var granted bool
		var at time.Time
		var revoked *time.Time
		if scanErr := rows.Scan(&ctype, &granted, &basis, &at, &revoked); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan consent: %w", scanErr)
		}
		status := "Erteilt"
		if !granted {
			status = "Verweigert"
		}
		if revoked != nil {
			status = "Widerrufen"
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Typ", Value: ctype},
			{Key: "Status", Value: status},
			{Key: "Rechtsgrundlage", Value: basis},
			{Key: "Datum", Value: at.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: consent rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// customFieldsModule reads a contact's custom field values. There is no
// tenant_id on contact_custom_field_values itself -- its RLS policy scopes
// via a join back to contacts, so this query joins the same way rather than
// relying on RLS alone (defense-in-depth, matching every other module here).
func customFieldsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT cfd.field_label, cfv.value
		 FROM contact_custom_field_values cfv
		 JOIN custom_field_definitions cfd ON cfd.id = cfv.field_id
		 JOIN contacts c ON c.id = cfv.contact_id
		 WHERE cfv.contact_id = $1 AND c.tenant_id = $2
		 ORDER BY cfd.sort_order, cfd.field_label
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query custom fields: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Benutzerdefinierte Felder", Columns: []string{"Feld", "Wert"}}
	for rows.Next() {
		var label string
		var valueJSON []byte
		if scanErr := rows.Scan(&label, &valueJSON); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan custom field: %w", scanErr)
		}
		mod.Records = append(mod.Records, fieldValueRecord(label, formatCustomFieldValue(valueJSON)))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: custom field rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// formatCustomFieldValue renders a contact_custom_field_values.value JSONB
// cell as plain text for the disclosure. field_type on the definition
// (text/number/date/boolean/select/multiselect) governs how the value was
// written, but decoding by Go type covers all of them without needing to
// look up the definition's field_type separately.
func formatCustomFieldValue(raw []byte) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case bool:
		return boolLabel(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", val)
	}
}

func tagsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT t.name, ct.created_at
		 FROM contact_tags ct
		 JOIN tags t ON t.id = ct.tag_id
		 WHERE ct.contact_id = $1 AND ct.tenant_id = $2
		 ORDER BY ct.created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query tags: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Tags", Columns: []string{"Tag", "Zugewiesen"}}
	for rows.Next() {
		var name string
		var at time.Time
		if scanErr := rows.Scan(&name, &at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan tag: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Tag", Value: name},
			{Key: "Zugewiesen", Value: at.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: tag rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// documentsModule discloses metadata only for files linked to a contact via
// document_entity_links (entity_type="contact", the literal used throughout
// internal/gateway/route_crm_contact_files.go and internal/document/file --
// there is no shared constant for it). No storage_key, thumbnail_key or
// content_text leaves this function: an Art. 15 export leaves the building,
// and a path or signed URL in it would be a document-store access path that
// bypasses the permission check a real download goes through.
//
// Soft-deleted files (is_deleted) are excluded: from the data subject's
// perspective they no longer exist once moved to trash, same as the file
// list UI already treats them.
func documentsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT f.filename, f.mime_type, f.file_size, f.created_at
		 FROM document_files f
		 JOIN document_entity_links del ON del.file_id = f.id
		 WHERE del.entity_type = 'contact' AND del.entity_id = $1 AND del.tenant_id = $2
		   AND f.tenant_id = $2 AND NOT f.is_deleted
		 ORDER BY f.created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query documents: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Dokumente", Columns: []string{"Dateiname", "Typ", "Größe", "Hochgeladen am"}}
	for rows.Next() {
		var filename, mimeType string
		var size int64
		var at time.Time
		if scanErr := rows.Scan(&filename, &mimeType, &size, &at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan document: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Dateiname", Value: filename},
			{Key: "Typ", Value: mimeType},
			{Key: "Größe", Value: fmt.Sprintf("%d Bytes", size)},
			{Key: "Hochgeladen am", Value: at.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: document rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// advisoryProtocolModules discloses every advisory session document tied to
// a contact via the RESTRICT FK enforced by IsInUse (postgres_repository.go:549)
// -- of the 14 tables that FK onto contacts, this is the only one with no
// DSAR module, and it carries the most sensitive data in the whole schema:
// birth date, marital status, tax status, income, assets, liabilities, risk
// profile, existing insurance, discussed products.
//
// Reuses advisoryprotocol.PostgresRepository.ListByContact instead of
// duplicating its ~50-column tenant-scoped scan; that repository query is
// already the mechanism every field here is read through elsewhere in the
// codebase.
//
// internal_notes is deliberately excluded, the same rule as the internal
// helpdesk flag in helpdeskMessagesModule: it is the advisor's own working
// assessment of the client, not communication addressed to them or a fact
// about their situation.
//
// A contact can have several protocols (consultation history). Each becomes
// its own module -- one module per protocol keeps ~40 disclosed fields
// readable per session instead of forcing them into one table's columns --
// ordered oldest first so the sequence reads as a timeline.
func advisoryProtocolModules(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) ([]DSARModule, error) {
	repo := advisoryprotocol.NewPostgresRepository(pool)
	protocols, err := repo.ListByContact(ctx, contactID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("dsar: query advisory protocols: %w", err)
	}
	if len(protocols) == 0 {
		return nil, nil
	}

	// ListByContact orders newest first for the editor UI; a disclosure reads
	// better as a consultation history, oldest first.
	for i, j := 0, len(protocols)-1; i < j; i, j = i+1, j-1 {
		protocols[i], protocols[j] = protocols[j], protocols[i]
	}

	modules := make([]DSARModule, 0, len(protocols))
	for i, p := range protocols {
		date := derefOrEmpty(p.Date)
		if date == "" {
			date = "ohne Datum"
		}
		modules = append(modules, DSARModule{
			Module:  fmt.Sprintf("Beratungsprotokoll %d (%s, %s)", i+1, date, advisoryStatusLabel(p.Status)),
			Columns: []string{"Feld", "Wert"},
			Records: advisoryProtocolRecords(p),
		})
	}
	return modules, nil
}

func advisoryStatusLabel(status string) string {
	if status == "finalized" {
		return "abgeschlossen"
	}
	return "Entwurf"
}

func advisoryProtocolRecords(p *advisoryprotocol.Protocol) []DSARRecord {
	products := make([]string, len(p.Products))
	for i, prod := range p.Products {
		rec := "empfohlen"
		if !prod.Recommended {
			rec = "nicht empfohlen"
		}
		products[i] = fmt.Sprintf("%s (SRI %d, %s)", prod.Name, prod.RiskClass, rec)
	}

	return []DSARRecord{
		fieldValueRecord("Datum", derefOrEmpty(p.Date)),
		fieldValueRecord("Zeit", strings.TrimSpace(strings.Trim(p.TimeFrom+"–"+p.TimeTo, "–"))),
		fieldValueRecord("Ort", p.Location),
		fieldValueRecord("Berater", p.Advisor),
		fieldValueRecord("Anlass", p.Occasion),
		fieldValueRecord("Anlass-Notiz", p.OccasionNote),
		fieldValueRecord("Kundenkategorie", p.CustomerCategory),
		fieldValueRecord("Geburtsdatum", derefOrEmpty(p.BirthDate)),
		fieldValueRecord("Familienstand", p.MaritalStatus),
		fieldValueRecord("Steuerstatus", p.TaxStatus),
		fieldValueRecord("Bekannte Anlageklassen", strings.Join(p.KnownAssetClasses, ", ")),
		fieldValueRecord("Bisherige Transaktionen", p.PastTransactions),
		fieldValueRecord("Finanzbildung", p.FinancialEducation),
		fieldValueRecord("Berufserfahrung", p.ProfessionalExperience),
		fieldValueRecord("Selbsteinschätzung (1-5)", derefIntOrEmpty(p.SelfAssessment)),
		fieldValueRecord("Monatliches Nettoeinkommen", moneyOrEmpty(p.MonthlyNetIncome)),
		fieldValueRecord("Laufende Verbindlichkeiten", moneyOrEmpty(p.RecurringLiabilities)),
		fieldValueRecord("Liquide Mittel", moneyOrEmpty(p.LiquidAssets)),
		fieldValueRecord("Bestehende Anlagen", moneyOrEmpty(p.CurrentInvestments)),
		fieldValueRecord("Immobilien", p.RealEstate),
		fieldValueRecord("Bestehende Versicherungen", p.ExistingInsurance),
		fieldValueRecord("Maximale Verlusttragfähigkeit (absolut)", moneyOrEmpty(p.MaxLossCapacityAbs)),
		fieldValueRecord("Maximale Verlusttragfähigkeit (%)", percentOrEmpty(p.MaxLossCapacityPct)),
		fieldValueRecord("Anlagezweck", strings.Join(p.InvestmentPurpose, ", ")),
		fieldValueRecord("Anlagehorizont", p.Horizon),
		fieldValueRecord("Risikobereitschaft", p.RiskTolerance),
		fieldValueRecord("Risikotragfähigkeit", p.RiskCapacity),
		fieldValueRecord("Risikoklasse (SRI 1-7)", strconv.Itoa(p.RiskClass)),
		fieldValueRecord("ESG-Präferenz", boolLabel(p.EsgPreference)),
		fieldValueRecord("ESG-Details", p.EsgDetails),
		fieldValueRecord("Einmalbetrag", moneyOrEmpty(p.OneTimeAmount)),
		fieldValueRecord("Monatliche Sparrate", moneyOrEmpty(p.MonthlySavings)),
		fieldValueRecord("Besprochene Produkte", strings.Join(products, "; ")),
		fieldValueRecord("Empfehlungszusammenfassung", p.RecommendationSummary),
		fieldValueRecord("Geeignetheitsbegründung", p.SuitabilityReasoning),
		fieldValueRecord("Zielbezug", p.GoalReference),
		fieldValueRecord("Alternativen", p.Alternatives),
		fieldValueRecord("Nicht empfohlen", p.NotRecommended),
		fieldValueRecord("Wesentliche Anliegen", p.MainConcerns),
		fieldValueRecord("Erteilte Warnhinweise", strings.Join(p.WarningsGiven, ", ")),
		fieldValueRecord("Dokument ausgehändigt", boolLabel(p.DocumentDelivered)),
		fieldValueRecord("Ausgehändigt am", derefOrEmpty(p.DocumentDeliveredDate)),
		fieldValueRecord("Übermittlungsform", p.DeliveryForm),
		fieldValueRecord("Berater-Unterschrift", p.AdvisorSignature),
		fieldValueRecord("Kundenbestätigung", boolLabel(p.CustomerConfirmation)),
		fieldValueRecord("Dokumentenverzicht", boolLabel(p.DocumentWaiver)),
		fieldValueRecord("Folgetermin", derefOrEmpty(p.FollowupDate)),
	}
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefIntOrEmpty(v *int) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(*v)
}

func moneyOrEmpty(v *float64) string {
	if v == nil {
		return ""
	}
	// lean: EUR hardcoded, same reasoning as financeModule below -- the
	// product is DACH/EUR-only and advisory_protocols carries no currency
	// column of its own.
	return fmt.Sprintf("%.2f EUR", *v)
}

func percentOrEmpty(v *float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.2f %%", *v)
}

// formFieldMeta is a partial decode of one entry in form_schemas.fields --
// enough to label an answer for disclosure without importing the formulare
// package's full FormField (which also carries options/conditional logic a
// data subject export has no use for).
type formFieldMeta struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

// formSubmissionsModule discloses public form submissions matched to a
// contact by comparing an "email"-typed field's answer against the contact's
// own address. This is a heuristic, not a foreign key: form_submissions
// carries no contact_id at all, and submitted_by is not the respondent's
// identity -- it holds the staff user who logged an authenticated
// submission, or nothing whatsoever for a public share-link submission
// (formulare/form_share.go SubmitByShareToken never sets it). The
// respondent's own data lives only inside the JSONB answers, keyed by
// whichever field id the form author chose, which is why the match has to
// go through the schema's field definitions instead of a column.
//
// A submission whose schema has since been deleted (form_schema_id set NULL
// by form_schemas' ON DELETE SET NULL) is excluded: without the schema there
// is no way to tell which answer, if any, held an email address.
func formSubmissionsModule(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, contactEmail string) (*DSARModule, error) {
	if contactEmail == "" {
		return nil, nil
	}
	rows, err := pool.Query(ctx,
		`SELECT fs.title, fs.fields, fsub.answers, fsub.submitted_at
		 FROM form_submissions fsub
		 JOIN form_schemas fs ON fs.id = fsub.form_schema_id
		 WHERE fsub.tenant_id = $1 AND fs.tenant_id = $1
		   AND EXISTS (
		     SELECT 1 FROM jsonb_array_elements(fs.fields) AS field(elem)
		     WHERE field.elem ->> 'type' = 'email'
		       AND LOWER(fsub.answers ->> (field.elem ->> 'id')) = LOWER($2)
		   )
		 ORDER BY fsub.submitted_at DESC
		 LIMIT $3`, tenantID, contactEmail, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query form submissions: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Formulareinreichungen", Columns: []string{"Formular", "Datum", "Feld", "Wert"}}
	for rows.Next() {
		var title string
		var fieldsRaw, answersRaw []byte
		var at time.Time
		if scanErr := rows.Scan(&title, &fieldsRaw, &answersRaw, &at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan form submission: %w", scanErr)
		}

		var defs []formFieldMeta
		if err := json.Unmarshal(fieldsRaw, &defs); err != nil {
			return nil, fmt.Errorf("dsar: decode form fields: %w", err)
		}
		var answers map[string]json.RawMessage
		if err := json.Unmarshal(answersRaw, &answers); err != nil {
			// The disclosure must still surface stored data even if it is not
			// the JSON object shape every submission is expected to have.
			mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
				{Key: "Formular", Value: title},
				{Key: "Datum", Value: at.Format(dsarTimeLayout)},
				{Key: "Feld", Value: "(unstrukturiert)"},
				{Key: "Wert", Value: string(answersRaw)},
			}})
			continue
		}

		for _, def := range defs {
			raw, ok := answers[def.ID]
			if !ok {
				continue
			}
			label := def.Label
			if label == "" {
				label = def.ID
			}
			mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
				{Key: "Formular", Value: title},
				{Key: "Datum", Value: at.Format(dsarTimeLayout)},
				{Key: "Feld", Value: label},
				{Key: "Wert", Value: formatCustomFieldValue(raw)},
			}})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: form submission rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func dialerModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT dcs.created_at, COALESCE(dcs.duration_seconds, 0), COALESCE(dcs.notes, '')
		 FROM dialer_call_sessions dcs
		 JOIN dialer_campaign_contacts dcc ON dcc.id = dcs.campaign_contact_id
		 WHERE dcc.contact_id = $1 AND dcc.tenant_id = $2
		 ORDER BY dcs.created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query dialer: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Anrufe", Columns: []string{"Datum", "Dauer (s)", "Notiz"}}
	for rows.Next() {
		var at time.Time
		var dur int
		var notes string
		if scanErr := rows.Scan(&at, &dur, &notes); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan dialer: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Datum", Value: at.Format(dsarTimeLayout)},
			{Key: "Dauer (s)", Value: fmt.Sprintf("%d", dur)},
			{Key: "Notiz", Value: notes},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: dialer rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func financeModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT invoice_number, status, invoice_date, gross_total::float8
		 FROM finance_invoices
		 WHERE contact_id = $1 AND tenant_id = $2
		 ORDER BY invoice_date DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query finance invoices: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Rechnungen", Columns: []string{"Rechnungsnummer", "Status", "Datum", "Betrag"}}
	for rows.Next() {
		var number, status string
		var at time.Time
		var gross float64
		if scanErr := rows.Scan(&number, &status, &at, &gross); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan finance invoice: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Rechnungsnummer", Value: number},
			{Key: "Status", Value: status},
			{Key: "Datum", Value: at.Format(dsarDateLayout)},
			// lean: EUR hardcoded — finance_invoices carries no currency column
			// because the product is DACH/EUR-only; revisit if multi-currency
			// billing lands.
			{Key: "Betrag", Value: fmt.Sprintf("%.2f EUR", gross)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: finance invoice rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func meetingsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT title, status, scheduled_start, scheduled_end
		 FROM meetings
		 WHERE contact_id = $1 AND tenant_id = $2
		 ORDER BY scheduled_start DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query meetings: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Meetings", Columns: []string{"Titel", "Status", "Beginn", "Ende"}}
	for rows.Next() {
		var title, status string
		var start, end time.Time
		if scanErr := rows.Scan(&title, &status, &start, &end); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan meeting: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Titel", Value: title},
			{Key: "Status", Value: status},
			{Key: "Beginn", Value: start.Format(dsarTimeLayout)},
			{Key: "Ende", Value: end.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: meeting rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func helpdeskModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT subject, status, priority, created_at
		 FROM tickets
		 WHERE contact_id = $1 AND tenant_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query tickets: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Helpdesk-Tickets", Columns: []string{"Betreff", "Status", "Priorität", "Erstellt"}}
	for rows.Next() {
		var subject, status, priority string
		var at time.Time
		if scanErr := rows.Scan(&subject, &status, &priority, &at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan ticket: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Betreff", Value: subject},
			{Key: "Status", Value: status},
			{Key: "Priorität", Value: priority},
			{Key: "Erstellt", Value: at.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: ticket rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// helpdeskMessagesModule discloses the customer-facing conversation on a
// contact's tickets — what the person wrote and what they were told, which is
// the actual personal content behind the ticket metadata in helpdeskModule.
// Internal notes (internal = true) are excluded on purpose: they are agent
// working material and routinely contain a colleague's assessment OF the
// data subject, not communication addressed to them.
func helpdeskMessagesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT t.subject, tm.body, tm.created_at
		 FROM ticket_messages tm
		 JOIN tickets t ON t.id = tm.ticket_id
		 WHERE t.contact_id = $1 AND tm.tenant_id = $2 AND NOT tm.internal
		 ORDER BY tm.created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows+1)
	if err != nil {
		return nil, fmt.Errorf("dsar: query helpdesk messages: %w", err)
	}
	defer rows.Close()

	type message struct {
		subject, body string
		at            time.Time
	}
	var messages []message
	for rows.Next() {
		var m message
		if scanErr := rows.Scan(&m.subject, &m.body, &m.at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan helpdesk message: %w", scanErr)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: helpdesk message rows: %w", err)
	}
	if len(messages) == 0 {
		return nil, nil
	}

	// Query fetched dsarMaxRows+1 newest-first so truncation, if any, drops
	// the oldest messages rather than silently hiding the most recent ones.
	truncated := len(messages) > dsarMaxRows
	if truncated {
		messages = messages[:dsarMaxRows]
	}
	// Reverse to chronological (oldest-first) order for a readable transcript.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	mod := DSARModule{Module: "Helpdesk-Nachrichten", Columns: []string{"Ticket", "Nachricht", "Datum"}}
	if truncated {
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Ticket", Value: ""},
			{Key: "Nachricht", Value: fmt.Sprintf("(gekürzt auf die %d neuesten Nachrichten)", dsarMaxRows)},
			{Key: "Datum", Value: ""},
		}})
	}
	for _, m := range messages {
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Ticket", Value: m.subject},
			{Key: "Nachricht", Value: m.body},
			{Key: "Datum", Value: m.at.Format(dsarTimeLayout)},
		}})
	}
	return &mod, nil
}

// contractsModule resolves via contract_parties: contracts carries no
// contact_id of its own, a contact is one of possibly several parties to a
// contract (contact/company/external).
func contractsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT c.contract_number, c.title, c.status, c.starts_on
		 FROM contracts c
		 JOIN contract_parties cp ON cp.contract_id = c.id
		 WHERE cp.contact_id = $1 AND cp.tenant_id = $2
		 ORDER BY c.starts_on DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query contracts: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Verträge", Columns: []string{"Vertragsnummer", "Titel", "Status", "Beginn"}}
	for rows.Next() {
		var number, title, status string
		var startsOn time.Time
		if scanErr := rows.Scan(&number, &title, &status, &startsOn); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan contract: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Vertragsnummer", Value: number},
			{Key: "Titel", Value: title},
			{Key: "Status", Value: status},
			{Key: "Beginn", Value: startsOn.Format(dsarDateLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: contract rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// emailModule resolves via email_contact_links: the junction table carries no
// tenant_id of its own, so tenant scoping happens on the joined email_messages
// row instead.
func emailModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT em.date, COALESCE(em.subject, ''), COALESCE(NULLIF(em.from_name, ''), em.from_email)
		 FROM email_messages em
		 JOIN email_contact_links ecl ON ecl.message_id = em.id
		 WHERE ecl.contact_id = $1 AND em.tenant_id = $2
		 ORDER BY em.date DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query emails: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "E-Mails", Columns: []string{"Datum", "Betreff", "Von"}}
	for rows.Next() {
		var subject, from string
		var at time.Time
		if scanErr := rows.Scan(&at, &subject, &from); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan email: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Datum", Value: at.Format(dsarTimeLayout)},
			{Key: "Betreff", Value: subject},
			{Key: "Von", Value: from},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: email rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func dealsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT d.name, d.value::float8, d.currency, ps.name, d.created_at
		 FROM deals d
		 JOIN pipeline_stages ps ON ps.id = d.stage_id
		 WHERE d.contact_id = $1 AND d.tenant_id = $2
		 ORDER BY d.created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query deals: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Deals", Columns: []string{"Bezeichnung", "Wert", "Phase", "Erstellt"}}
	for rows.Next() {
		var name, currency, stage string
		var value float64
		var at time.Time
		if scanErr := rows.Scan(&name, &value, &currency, &stage, &at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan deal: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Bezeichnung", Value: name},
			{Key: "Wert", Value: fmt.Sprintf("%.2f %s", value, currency)},
			{Key: "Phase", Value: stage},
			{Key: "Erstellt", Value: at.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: deal rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func activitiesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT created_at, activity_type::text, subject, COALESCE(description, '')
		 FROM activities
		 WHERE contact_id = $1 AND tenant_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query activities: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Aktivitäten", Columns: []string{"Datum", "Typ", "Betreff", "Notiz"}}
	for rows.Next() {
		var activityType, subject, note string
		var at time.Time
		if scanErr := rows.Scan(&at, &activityType, &subject, &note); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan activity: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Datum", Value: at.Format(dsarTimeLayout)},
			{Key: "Typ", Value: activityType},
			{Key: "Betreff", Value: subject},
			{Key: "Notiz", Value: note},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: activity rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func fieldValueRecord(field, value string) DSARRecord {
	return DSARRecord{Fields: []DSARField{
		{Key: "Feld", Value: field},
		{Key: "Wert", Value: value},
	}}
}

func joinName(first, last string) string {
	name := strings.TrimSpace(first + " " + last)
	if name == "" {
		return "—"
	}
	return name
}

func initials(first, last string) string {
	var b strings.Builder
	if first != "" {
		b.WriteString(string([]rune(first)[0:1]))
	}
	if last != "" {
		b.WriteString(string([]rune(last)[0:1]))
	}
	if b.Len() == 0 {
		return "?"
	}
	return strings.ToUpper(b.String())
}

func boolLabel(v bool) string {
	if v {
		return "Ja"
	}
	return "Nein"
}
