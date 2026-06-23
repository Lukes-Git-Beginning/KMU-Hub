package meeting

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL meeting repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateMeeting(ctx context.Context, m *Meeting) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO meetings (id, tenant_id, title, description, agenda, organizer_id, status,
		  scheduled_start, scheduled_end, actual_start, actual_end, room_name,
		  calendar_event_id, recurring_meeting_id, contact_id, deal_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		m.ID, m.TenantID, m.Title, m.Description, m.Agenda, m.OrganizerID, m.Status,
		m.ScheduledStart, m.ScheduledEnd, m.ActualStart, m.ActualEnd, m.RoomName,
		m.CalendarEventID, m.RecurringMeetingID, m.ContactID, m.DealID, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetMeeting(ctx context.Context, id, tenantID uuid.UUID) (*Meeting, error) {
	var m Meeting
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, title, description, agenda, organizer_id, status, locked,
		  scheduled_start, scheduled_end, actual_start, actual_end, room_name,
		  calendar_event_id, recurring_meeting_id, contact_id, deal_id, created_at, updated_at
		 FROM meetings WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(
		&m.ID, &m.TenantID, &m.Title, &m.Description, &m.Agenda, &m.OrganizerID, &m.Status, &m.Locked,
		&m.ScheduledStart, &m.ScheduledEnd, &m.ActualStart, &m.ActualEnd, &m.RoomName,
		&m.CalendarEventID, &m.RecurringMeetingID, &m.ContactID, &m.DealID, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get meeting: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) UpdateMeeting(ctx context.Context, m *Meeting) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE meetings SET title=$3, description=$4, agenda=$5, status=$6,
		  scheduled_start=$7, scheduled_end=$8, actual_start=$9, actual_end=$10,
		  room_name=$11, calendar_event_id=$12, recurring_meeting_id=$13, updated_at=$14,
		  locked=$15, contact_id=$16, deal_id=$17
		 WHERE id=$1 AND tenant_id=$2`,
		m.ID, m.TenantID, m.Title, m.Description, m.Agenda, m.Status,
		m.ScheduledStart, m.ScheduledEnd, m.ActualStart, m.ActualEnd,
		m.RoomName, m.CalendarEventID, m.RecurringMeetingID, m.UpdatedAt,
		m.Locked, m.ContactID, m.DealID,
	)
	if err != nil {
		return fmt.Errorf("update meeting: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteMeeting(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM meetings WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete meeting: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ListMeetings(ctx context.Context, filter MeetingFilter) ([]Meeting, error) {
	var conditions []string
	var args []any
	argIdx := 1

	// Tenant isolation is mandatory
	conditions = append(conditions, fmt.Sprintf("m.tenant_id = $%d", argIdx))
	args = append(args, filter.TenantID)
	argIdx++

	if filter.OrganizerID != nil {
		conditions = append(conditions, fmt.Sprintf("m.organizer_id = $%d", argIdx))
		args = append(args, *filter.OrganizerID)
		argIdx++
	}
	if filter.AttendeeID != nil {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM meeting_attendees ma WHERE ma.meeting_id = m.id AND ma.user_id = $%d)", argIdx))
		args = append(args, *filter.AttendeeID)
		argIdx++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("m.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.StartAfter != nil {
		conditions = append(conditions, fmt.Sprintf("m.scheduled_start >= $%d", argIdx))
		args = append(args, *filter.StartAfter)
		argIdx++
	}
	if filter.StartBefore != nil {
		conditions = append(conditions, fmt.Sprintf("m.scheduled_start <= $%d", argIdx))
		args = append(args, *filter.StartBefore)
		argIdx++
	}

	query := `SELECT m.id, m.tenant_id, m.title, m.description, m.agenda, m.organizer_id, m.status, m.locked,
		m.scheduled_start, m.scheduled_end, m.actual_start, m.actual_end, m.room_name,
		m.calendar_event_id, m.recurring_meeting_id, m.contact_id, m.deal_id, m.created_at, m.updated_at
		FROM meetings m`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY m.scheduled_start DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list meetings: %w", err)
	}
	defer rows.Close()

	var meetings []Meeting
	for rows.Next() {
		var m Meeting
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.Title, &m.Description, &m.Agenda, &m.OrganizerID, &m.Status, &m.Locked,
			&m.ScheduledStart, &m.ScheduledEnd, &m.ActualStart, &m.ActualEnd, &m.RoomName,
			&m.CalendarEventID, &m.RecurringMeetingID, &m.ContactID, &m.DealID, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan meeting: %w", err)
		}
		meetings = append(meetings, m)
	}
	return meetings, rows.Err()
}

// GetMeetingByRoomName looks up a meeting by its LiveKit room_name without a
// tenant filter. Must be called under WithSystemContext so RLS is bypassed.
func (r *PostgresRepository) GetMeetingByRoomName(ctx context.Context, roomName string) (*Meeting, error) {
	var m Meeting
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, title, description, agenda, organizer_id, status, locked,
		  scheduled_start, scheduled_end, actual_start, actual_end, room_name,
		  calendar_event_id, recurring_meeting_id, contact_id, deal_id, created_at, updated_at
		 FROM meetings WHERE room_name = $1`,
		roomName,
	).Scan(
		&m.ID, &m.TenantID, &m.Title, &m.Description, &m.Agenda, &m.OrganizerID, &m.Status, &m.Locked,
		&m.ScheduledStart, &m.ScheduledEnd, &m.ActualStart, &m.ActualEnd, &m.RoomName,
		&m.CalendarEventID, &m.RecurringMeetingID, &m.ContactID, &m.DealID, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get meeting by room name: %w", err)
	}
	return &m, nil
}

// ListStaleMeetings returns all in_progress meetings whose scheduled_end is
// strictly before cutoff (caller adds grace before passing). No tenant filter —
// must be called under WithSystemContext so RLS is bypassed.
func (r *PostgresRepository) ListStaleMeetings(ctx context.Context, cutoff time.Time) ([]Meeting, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, title, description, agenda, organizer_id, status, locked,
		  scheduled_start, scheduled_end, actual_start, actual_end, room_name,
		  calendar_event_id, recurring_meeting_id, contact_id, deal_id, created_at, updated_at
		 FROM meetings
		 WHERE status = 'in_progress' AND scheduled_end < $1`,
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("list stale meetings: %w", err)
	}
	defer rows.Close()

	var meetings []Meeting
	for rows.Next() {
		var m Meeting
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.Title, &m.Description, &m.Agenda, &m.OrganizerID, &m.Status, &m.Locked,
			&m.ScheduledStart, &m.ScheduledEnd, &m.ActualStart, &m.ActualEnd, &m.RoomName,
			&m.CalendarEventID, &m.RecurringMeetingID, &m.ContactID, &m.DealID, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stale meeting: %w", err)
		}
		meetings = append(meetings, m)
	}
	return meetings, rows.Err()
}

// --- Attendees ---

func (r *PostgresRepository) AddAttendee(ctx context.Context, meetingID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO meeting_attendees (meeting_id, user_id, rsvp_status, tenant_id)
		 VALUES ($1, $2, 'pending', (SELECT tenant_id FROM meetings WHERE id = $1))
		 ON CONFLICT (meeting_id, user_id) DO NOTHING`,
		meetingID, userID,
	)
	return err
}

func (r *PostgresRepository) RemoveAttendee(ctx context.Context, meetingID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM meeting_attendees WHERE meeting_id=$1 AND user_id=$2`,
		meetingID, userID,
	)
	return err
}

func (r *PostgresRepository) UpdateAttendeeRSVP(ctx context.Context, meetingID, userID uuid.UUID, rsvp string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE meeting_attendees SET rsvp_status=$3 WHERE meeting_id=$1 AND user_id=$2`,
		meetingID, userID, rsvp,
	)
	if err != nil {
		return fmt.Errorf("update attendee RSVP: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) GetAttendees(ctx context.Context, meetingID uuid.UUID) ([]MeetingAttendee, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT meeting_id, user_id, rsvp_status FROM meeting_attendees WHERE meeting_id=$1`,
		meetingID,
	)
	if err != nil {
		return nil, fmt.Errorf("get attendees: %w", err)
	}
	defer rows.Close()

	var attendees []MeetingAttendee
	for rows.Next() {
		var a MeetingAttendee
		if err := rows.Scan(&a.MeetingID, &a.UserID, &a.RSVPStatus); err != nil {
			return nil, fmt.Errorf("scan attendee: %w", err)
		}
		attendees = append(attendees, a)
	}
	return attendees, rows.Err()
}

// --- Notes ---

func (r *PostgresRepository) SaveNotes(ctx context.Context, notes *MeetingNotes) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO meeting_notes (id, meeting_id, author_id, content, is_private, created_at, updated_at, tenant_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, (SELECT tenant_id FROM meetings WHERE id = $2))
		 ON CONFLICT (meeting_id, author_id) WHERE is_private = $5
		 DO UPDATE SET content = EXCLUDED.content, updated_at = EXCLUDED.updated_at`,
		notes.ID, notes.MeetingID, notes.AuthorID, notes.Content, notes.IsPrivate,
		notes.CreatedAt, notes.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save notes: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetNotes(ctx context.Context, meetingID, authorID uuid.UUID) (*MeetingNotes, error) {
	var n MeetingNotes
	err := r.pool.QueryRow(ctx,
		`SELECT id, meeting_id, author_id, content, is_private, created_at, updated_at
		 FROM meeting_notes WHERE meeting_id=$1 AND author_id=$2 LIMIT 1`,
		meetingID, authorID,
	).Scan(&n.ID, &n.MeetingID, &n.AuthorID, &n.Content, &n.IsPrivate, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get notes: %w", err)
	}
	return &n, nil
}

func (r *PostgresRepository) GetAllNotes(ctx context.Context, meetingID uuid.UUID) ([]MeetingNotes, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, meeting_id, author_id, content, is_private, created_at, updated_at
		 FROM meeting_notes WHERE meeting_id=$1 AND is_private=false ORDER BY created_at`,
		meetingID,
	)
	if err != nil {
		return nil, fmt.Errorf("get all notes: %w", err)
	}
	defer rows.Close()

	var notes []MeetingNotes
	for rows.Next() {
		var n MeetingNotes
		if err := rows.Scan(&n.ID, &n.MeetingID, &n.AuthorID, &n.Content, &n.IsPrivate, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan notes: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (r *PostgresRepository) GetPreviousMeetingNotes(ctx context.Context, recurringMeetingID uuid.UUID, beforeDate time.Time) (*MeetingNotes, error) {
	// Find the most recent completed meeting with same recurring_meeting_id before the given date,
	// then return its first public notes entry
	var n MeetingNotes
	err := r.pool.QueryRow(ctx,
		`SELECT mn.id, mn.meeting_id, mn.author_id, mn.content, mn.is_private, mn.created_at, mn.updated_at
		 FROM meeting_notes mn
		 INNER JOIN meetings m ON mn.meeting_id = m.id
		 WHERE m.recurring_meeting_id = $1
		   AND m.scheduled_start < $2
		   AND m.status = 'completed'
		   AND mn.is_private = false
		 ORDER BY m.scheduled_start DESC, mn.created_at ASC
		 LIMIT 1`,
		recurringMeetingID, beforeDate,
	).Scan(&n.ID, &n.MeetingID, &n.AuthorID, &n.Content, &n.IsPrivate, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoPreviousNotes
	}
	if err != nil {
		return nil, fmt.Errorf("get previous meeting notes: %w", err)
	}
	return &n, nil
}

// --- Action Items ---

func (r *PostgresRepository) GetActionItemByID(ctx context.Context, id, tenantID uuid.UUID) (*MeetingActionItem, error) {
	var item MeetingActionItem
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, meeting_id, description, assignee_id, is_completed, task_id, sort_order, created_at
		 FROM meeting_action_items WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
	).Scan(
		&item.ID, &item.TenantID, &item.MeetingID, &item.Description, &item.AssigneeID,
		&item.IsCompleted, &item.TaskID, &item.SortOrder, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrActionItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get action item by id: %w", err)
	}
	return &item, nil
}

func (r *PostgresRepository) CreateActionItem(ctx context.Context, item *MeetingActionItem) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO meeting_action_items (id, tenant_id, meeting_id, description, assignee_id, is_completed, task_id, sort_order, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		item.ID, item.TenantID, item.MeetingID, item.Description, item.AssigneeID,
		item.IsCompleted, item.TaskID, item.SortOrder, item.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateActionItem(ctx context.Context, item *MeetingActionItem, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE meeting_action_items SET description=$2, assignee_id=$3, is_completed=$4, sort_order=$5
		 WHERE id=$1 AND tenant_id=$6`,
		item.ID, item.Description, item.AssigneeID, item.IsCompleted, item.SortOrder, tenantID,
	)
	if err != nil {
		return fmt.Errorf("update action item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrActionItemNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteActionItem(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM meeting_action_items WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete action item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrActionItemNotFound
	}
	return nil
}

func (r *PostgresRepository) ListActionItems(ctx context.Context, meetingID, tenantID uuid.UUID) ([]MeetingActionItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, meeting_id, description, assignee_id, is_completed, task_id, sort_order, created_at
		 FROM meeting_action_items WHERE meeting_id=$1 AND tenant_id=$2 ORDER BY sort_order, created_at`,
		meetingID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list action items: %w", err)
	}
	defer rows.Close()

	var items []MeetingActionItem
	for rows.Next() {
		var item MeetingActionItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.MeetingID, &item.Description, &item.AssigneeID,
			&item.IsCompleted, &item.TaskID, &item.SortOrder, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan action item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) UpdateActionItemTaskID(ctx context.Context, itemID, taskID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE meeting_action_items SET task_id=$2 WHERE id=$1`,
		itemID, taskID,
	)
	if err != nil {
		return fmt.Errorf("update action item task_id: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrActionItemNotFound
	}
	return nil
}

// --- Chat ---

func (r *PostgresRepository) SaveChatMessage(ctx context.Context, msg *MeetingChatMessage) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO meeting_chat (id, tenant_id, meeting_id, sender_id, sender_name, message, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		msg.ID, msg.TenantID, msg.MeetingID, msg.SenderID, msg.SenderName, msg.Message, msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save chat message: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListChatMessages(ctx context.Context, meetingID, tenantID uuid.UUID, limit int) ([]MeetingChatMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, meeting_id, sender_id, sender_name, message, created_at
		 FROM meeting_chat
		 WHERE meeting_id = $1 AND tenant_id = $2
		 ORDER BY created_at ASC
		 LIMIT $3`,
		meetingID, tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}
	defer rows.Close()

	var msgs []MeetingChatMessage
	for rows.Next() {
		var m MeetingChatMessage
		if err := rows.Scan(&m.ID, &m.TenantID, &m.MeetingID, &m.SenderID, &m.SenderName, &m.Message, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// --- Co-hosts ---

// AddCoHost inserts a co-host row. Idempotent via ON CONFLICT DO NOTHING.
func (r *PostgresRepository) AddCoHost(ctx context.Context, tenantID, meetingID, userID, grantedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO meeting_cohosts (tenant_id, meeting_id, user_id, granted_by)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (meeting_id, user_id) DO NOTHING`,
		tenantID, meetingID, userID, grantedBy,
	)
	if err != nil {
		return fmt.Errorf("add co-host: %w", err)
	}
	return nil
}

// RemoveCoHost deletes the co-host row. Idempotent — no error when not found.
func (r *PostgresRepository) RemoveCoHost(ctx context.Context, tenantID, meetingID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM meeting_cohosts WHERE tenant_id=$1 AND meeting_id=$2 AND user_id=$3`,
		tenantID, meetingID, userID,
	)
	if err != nil {
		return fmt.Errorf("remove co-host: %w", err)
	}
	return nil
}

// IsCoHost reports whether userID has co-host rights for meetingID.
func (r *PostgresRepository) IsCoHost(ctx context.Context, tenantID, meetingID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
		   SELECT 1 FROM meeting_cohosts
		   WHERE tenant_id=$1 AND meeting_id=$2 AND user_id=$3
		 )`,
		tenantID, meetingID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is co-host: %w", err)
	}
	return exists, nil
}

// ListCoHosts returns all co-host entries for a meeting.
func (r *PostgresRepository) ListCoHosts(ctx context.Context, tenantID, meetingID uuid.UUID) ([]MeetingCoHost, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, meeting_id, user_id, granted_by, created_at
		 FROM meeting_cohosts
		 WHERE tenant_id=$1 AND meeting_id=$2
		 ORDER BY created_at`,
		tenantID, meetingID,
	)
	if err != nil {
		return nil, fmt.Errorf("list co-hosts: %w", err)
	}
	defer rows.Close()

	var cohosts []MeetingCoHost
	for rows.Next() {
		var c MeetingCoHost
		if err := rows.Scan(&c.ID, &c.TenantID, &c.MeetingID, &c.UserID, &c.GrantedBy, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan co-host: %w", err)
		}
		cohosts = append(cohosts, c)
	}
	return cohosts, rows.Err()
}

// --- Lock ---

// SetLocked updates the locked flag on a meeting row.
func (r *PostgresRepository) SetLocked(ctx context.Context, tenantID, meetingID uuid.UUID, locked bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE meetings SET locked=$3, updated_at=now() WHERE id=$1 AND tenant_id=$2`,
		meetingID, tenantID, locked,
	)
	if err != nil {
		return fmt.Errorf("set locked: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
