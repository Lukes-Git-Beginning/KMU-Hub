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
		`INSERT INTO meetings (id, title, description, agenda, organizer_id, status,
		  scheduled_start, scheduled_end, actual_start, actual_end, room_name,
		  calendar_event_id, recurring_meeting_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		m.ID, m.Title, m.Description, m.Agenda, m.OrganizerID, m.Status,
		m.ScheduledStart, m.ScheduledEnd, m.ActualStart, m.ActualEnd, m.RoomName,
		m.CalendarEventID, m.RecurringMeetingID, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetMeeting(ctx context.Context, id uuid.UUID) (*Meeting, error) {
	var m Meeting
	err := r.pool.QueryRow(ctx,
		`SELECT id, title, description, agenda, organizer_id, status,
		  scheduled_start, scheduled_end, actual_start, actual_end, room_name,
		  calendar_event_id, recurring_meeting_id, created_at, updated_at
		 FROM meetings WHERE id = $1`,
		id,
	).Scan(
		&m.ID, &m.Title, &m.Description, &m.Agenda, &m.OrganizerID, &m.Status,
		&m.ScheduledStart, &m.ScheduledEnd, &m.ActualStart, &m.ActualEnd, &m.RoomName,
		&m.CalendarEventID, &m.RecurringMeetingID, &m.CreatedAt, &m.UpdatedAt,
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
		`UPDATE meetings SET title=$2, description=$3, agenda=$4, status=$5,
		  scheduled_start=$6, scheduled_end=$7, actual_start=$8, actual_end=$9,
		  room_name=$10, calendar_event_id=$11, recurring_meeting_id=$12, updated_at=$13
		 WHERE id=$1`,
		m.ID, m.Title, m.Description, m.Agenda, m.Status,
		m.ScheduledStart, m.ScheduledEnd, m.ActualStart, m.ActualEnd,
		m.RoomName, m.CalendarEventID, m.RecurringMeetingID, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update meeting: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteMeeting(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM meetings WHERE id=$1`, id)
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

	query := `SELECT m.id, m.title, m.description, m.agenda, m.organizer_id, m.status,
		m.scheduled_start, m.scheduled_end, m.actual_start, m.actual_end, m.room_name,
		m.calendar_event_id, m.recurring_meeting_id, m.created_at, m.updated_at
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
			&m.ID, &m.Title, &m.Description, &m.Agenda, &m.OrganizerID, &m.Status,
			&m.ScheduledStart, &m.ScheduledEnd, &m.ActualStart, &m.ActualEnd, &m.RoomName,
			&m.CalendarEventID, &m.RecurringMeetingID, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan meeting: %w", err)
		}
		meetings = append(meetings, m)
	}
	return meetings, rows.Err()
}

// --- Attendees ---

func (r *PostgresRepository) AddAttendee(ctx context.Context, meetingID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO meeting_attendees (meeting_id, user_id, rsvp_status) VALUES ($1, $2, 'pending')
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
		`INSERT INTO meeting_notes (id, meeting_id, author_id, content, is_private, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
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

func (r *PostgresRepository) CreateActionItem(ctx context.Context, item *MeetingActionItem) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO meeting_action_items (id, meeting_id, description, assignee_id, is_completed, task_id, sort_order, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		item.ID, item.MeetingID, item.Description, item.AssigneeID,
		item.IsCompleted, item.TaskID, item.SortOrder, item.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateActionItem(ctx context.Context, item *MeetingActionItem) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE meeting_action_items SET description=$2, assignee_id=$3, is_completed=$4, sort_order=$5
		 WHERE id=$1`,
		item.ID, item.Description, item.AssigneeID, item.IsCompleted, item.SortOrder,
	)
	if err != nil {
		return fmt.Errorf("update action item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrActionItemNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteActionItem(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM meeting_action_items WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete action item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrActionItemNotFound
	}
	return nil
}

func (r *PostgresRepository) ListActionItems(ctx context.Context, meetingID uuid.UUID) ([]MeetingActionItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, meeting_id, description, assignee_id, is_completed, task_id, sort_order, created_at
		 FROM meeting_action_items WHERE meeting_id=$1 ORDER BY sort_order, created_at`,
		meetingID,
	)
	if err != nil {
		return nil, fmt.Errorf("list action items: %w", err)
	}
	defer rows.Close()

	var items []MeetingActionItem
	for rows.Next() {
		var item MeetingActionItem
		if err := rows.Scan(
			&item.ID, &item.MeetingID, &item.Description, &item.AssigneeID,
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
