package event

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL event repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, event *models.CalendarEvent) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO calendar_events (id, calendar_id, title, description, location, resource_id,
		 start_time, end_time, is_all_day, timezone, rrule, recurrence_end, has_video_call,
		 livekit_room_name, category_id, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		event.ID, event.CalendarID, event.Title, event.Description, event.Location,
		event.ResourceID, event.StartTime, event.EndTime, event.IsAllDay, event.Timezone,
		event.RRule, event.RecurrenceEnd, event.HasVideoCall, event.LiveKitRoomName,
		event.CategoryID, event.CreatedBy, event.CreatedAt, event.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.CalendarEvent, error) {
	var e models.CalendarEvent
	err := r.pool.QueryRow(ctx,
		`SELECT id, calendar_id, title, description, location, resource_id,
		 start_time, end_time, is_all_day, timezone, rrule, recurrence_end,
		 has_video_call, livekit_room_name, category_id, created_by, created_at, updated_at
		 FROM calendar_events WHERE id = $1`, id,
	).Scan(&e.ID, &e.CalendarID, &e.Title, &e.Description, &e.Location, &e.ResourceID,
		&e.StartTime, &e.EndTime, &e.IsAllDay, &e.Timezone, &e.RRule, &e.RecurrenceEnd,
		&e.HasVideoCall, &e.LiveKitRoomName, &e.CategoryID, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *PostgresRepository) Update(ctx context.Context, event *models.CalendarEvent) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE calendar_events SET title=$1, description=$2, location=$3, resource_id=$4,
		 start_time=$5, end_time=$6, is_all_day=$7, timezone=$8, rrule=$9, recurrence_end=$10,
		 has_video_call=$11, livekit_room_name=$12, category_id=$13, updated_at=$14
		 WHERE id=$15`,
		event.Title, event.Description, event.Location, event.ResourceID,
		event.StartTime, event.EndTime, event.IsAllDay, event.Timezone,
		event.RRule, event.RecurrenceEnd, event.HasVideoCall, event.LiveKitRoomName,
		event.CategoryID, event.UpdatedAt, event.ID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM calendar_events WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) ListInRange(ctx context.Context, calendarIDs []uuid.UUID, start, end time.Time, userID uuid.UUID) ([]models.ExpandedEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT e.id, e.calendar_id, e.title, e.description, e.location, e.resource_id,
		        e.start_time, e.end_time, e.is_all_day, e.timezone, e.rrule, e.recurrence_end,
		        e.has_video_call, e.livekit_room_name, e.category_id, e.created_by,
		        e.created_at, e.updated_at,
		        c.name AS calendar_name, c.color AS calendar_color,
		        COALESCE(cm.color_override, c.color) AS display_color,
		        ec.name AS category_name, ec.color AS category_color,
		        res.name AS resource_name,
		        (SELECT COUNT(*) FROM event_attendees ea WHERE ea.event_id = e.id) AS attendee_count,
		        (SELECT ea.rsvp_status FROM event_attendees ea WHERE ea.event_id = e.id AND ea.user_id = $4) AS my_rsvp
		 FROM calendar_events e
		 JOIN calendars c ON e.calendar_id = c.id
		 LEFT JOIN calendar_members cm ON c.id = cm.calendar_id AND cm.user_id = $4
		 LEFT JOIN event_categories ec ON e.category_id = ec.id
		 LEFT JOIN resources res ON e.resource_id = res.id
		 WHERE e.calendar_id = ANY($1)
		   AND e.rrule IS NULL
		   AND e.start_time < $3 AND e.end_time > $2
		 ORDER BY e.start_time ASC`,
		calendarIDs, start, end, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.ExpandedEvent
	for rows.Next() {
		var ev models.ExpandedEvent
		if scanErr := rows.Scan(
			&ev.ID, &ev.CalendarID, &ev.Title, &ev.Description, &ev.Location, &ev.ResourceID,
			&ev.StartTime, &ev.EndTime, &ev.IsAllDay, &ev.Timezone, &ev.RRule, &ev.RecurrenceEnd,
			&ev.HasVideoCall, &ev.LiveKitRoomName, &ev.CategoryID, &ev.CreatedBy,
			&ev.CreatedAt, &ev.UpdatedAt,
			&ev.CalendarName, &ev.CalendarColor, &ev.DisplayColor,
			&ev.CategoryName, &ev.CategoryColor,
			&ev.ResourceName,
			&ev.AttendeeCount, &ev.MyRSVP,
		); scanErr != nil {
			return nil, scanErr
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

func (r *PostgresRepository) ListRecurringOverlapping(ctx context.Context, calendarIDs []uuid.UUID, start, end time.Time) ([]models.CalendarEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, calendar_id, title, description, location, resource_id,
		        start_time, end_time, is_all_day, timezone, rrule, recurrence_end,
		        has_video_call, livekit_room_name, category_id, created_by,
		        created_at, updated_at
		 FROM calendar_events
		 WHERE calendar_id = ANY($1)
		   AND rrule IS NOT NULL
		   AND start_time < $3
		   AND (recurrence_end IS NULL OR recurrence_end > $2)
		 ORDER BY start_time ASC`,
		calendarIDs, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.CalendarEvent
	for rows.Next() {
		var e models.CalendarEvent
		if scanErr := rows.Scan(
			&e.ID, &e.CalendarID, &e.Title, &e.Description, &e.Location, &e.ResourceID,
			&e.StartTime, &e.EndTime, &e.IsAllDay, &e.Timezone, &e.RRule, &e.RecurrenceEnd,
			&e.HasVideoCall, &e.LiveKitRoomName, &e.CategoryID, &e.CreatedBy,
			&e.CreatedAt, &e.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// Attendees

func (r *PostgresRepository) AddAttendee(ctx context.Context, attendee *models.EventAttendee) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO event_attendees (event_id, user_id, rsvp_status, responded_at, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		attendee.EventID, attendee.UserID, attendee.RSVPStatus,
		attendee.RespondedAt, attendee.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyAttendee
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) RemoveAttendee(ctx context.Context, eventID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM event_attendees WHERE event_id = $1 AND user_id = $2`,
		eventID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotAttendee
	}
	return nil
}

func (r *PostgresRepository) UpdateRSVP(ctx context.Context, eventID, userID uuid.UUID, status string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE event_attendees SET rsvp_status = $1, responded_at = $4
		 WHERE event_id = $2 AND user_id = $3`,
		status, eventID, userID, time.Now(),
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotAttendee
	}
	return nil
}

func (r *PostgresRepository) ListAttendees(ctx context.Context, eventID uuid.UUID) ([]models.EventAttendee, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ea.event_id, ea.user_id, ea.rsvp_status, ea.responded_at, ea.created_at,
		        COALESCE(u.first_name, '') AS first_name, COALESCE(u.last_name, '') AS last_name
		 FROM event_attendees ea
		 JOIN users u ON ea.user_id = u.id
		 WHERE ea.event_id = $1
		 ORDER BY ea.created_at ASC`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attendees []models.EventAttendee
	for rows.Next() {
		var a models.EventAttendee
		if scanErr := rows.Scan(&a.EventID, &a.UserID, &a.RSVPStatus, &a.RespondedAt,
			&a.CreatedAt, &a.FirstName, &a.LastName); scanErr != nil {
			return nil, scanErr
		}
		attendees = append(attendees, a)
	}
	return attendees, rows.Err()
}

func (r *PostgresRepository) ListAttendeeEventIDs(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ea.event_id FROM event_attendees ea
		 JOIN calendar_events e ON ea.event_id = e.id
		 WHERE ea.user_id = $1 AND e.start_time < $3 AND e.end_time > $2`,
		userID, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, scanErr
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Exceptions

func (r *PostgresRepository) CreateException(ctx context.Context, exception *models.EventException) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO event_exceptions (id, event_id, original_date, is_cancelled, title, description,
		 location, start_time, end_time, resource_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		exception.ID, exception.EventID, exception.OriginalDate, exception.IsCancelled,
		exception.Title, exception.Description, exception.Location,
		exception.StartTime, exception.EndTime, exception.ResourceID,
		exception.CreatedAt, exception.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrExceptionAlreadyExists
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) ListExceptions(ctx context.Context, eventID uuid.UUID, start, end time.Time) ([]models.EventException, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, event_id, original_date, is_cancelled, title, description, location,
		        start_time, end_time, resource_id, created_at, updated_at
		 FROM event_exceptions
		 WHERE event_id = $1 AND original_date >= $2 AND original_date <= $3
		 ORDER BY original_date ASC`,
		eventID, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exceptions []models.EventException
	for rows.Next() {
		var ex models.EventException
		if scanErr := rows.Scan(&ex.ID, &ex.EventID, &ex.OriginalDate, &ex.IsCancelled,
			&ex.Title, &ex.Description, &ex.Location, &ex.StartTime, &ex.EndTime,
			&ex.ResourceID, &ex.CreatedAt, &ex.UpdatedAt); scanErr != nil {
			return nil, scanErr
		}
		exceptions = append(exceptions, ex)
	}
	return exceptions, rows.Err()
}

func (r *PostgresRepository) DeleteExceptionsAfterDate(ctx context.Context, eventID uuid.UUID, date time.Time) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM event_exceptions WHERE event_id = $1 AND original_date >= $2`,
		eventID, date,
	)
	return err
}

// Reminders

func (r *PostgresRepository) SetReminders(ctx context.Context, eventID uuid.UUID, minutesBefore []int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, delErr := tx.Exec(ctx, `DELETE FROM event_reminders WHERE event_id = $1`, eventID); delErr != nil {
		return delErr
	}

	now := time.Now()
	for _, minutes := range minutesBefore {
		if _, insErr := tx.Exec(ctx,
			`INSERT INTO event_reminders (id, event_id, minutes_before, created_at)
			 VALUES ($1, $2, $3, $4)`,
			uuid.New(), eventID, minutes, now,
		); insErr != nil {
			return insErr
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) ListReminders(ctx context.Context, eventID uuid.UUID) ([]models.EventReminder, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, event_id, minutes_before, created_at
		 FROM event_reminders WHERE event_id = $1
		 ORDER BY minutes_before ASC`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []models.EventReminder
	for rows.Next() {
		var rem models.EventReminder
		if scanErr := rows.Scan(&rem.ID, &rem.EventID, &rem.MinutesBefore, &rem.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		reminders = append(reminders, rem)
	}
	return reminders, rows.Err()
}

func (r *PostgresRepository) ListUpcomingReminders(ctx context.Context, windowStart, windowEnd time.Time) ([]ReminderWithEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.event_id, r.minutes_before, r.created_at,
		        e.id, e.calendar_id, e.title, e.description, e.location, e.resource_id,
		        e.start_time, e.end_time, e.is_all_day, e.timezone, e.rrule, e.recurrence_end,
		        e.has_video_call, e.livekit_room_name, e.category_id, e.created_by,
		        e.created_at, e.updated_at
		 FROM event_reminders r
		 JOIN calendar_events e ON r.event_id = e.id
		 WHERE e.rrule IS NULL
		   AND (e.start_time - make_interval(mins => r.minutes_before)) BETWEEN $1 AND $2`,
		windowStart, windowEnd,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ReminderWithEvent
	for rows.Next() {
		var rwe ReminderWithEvent
		if scanErr := rows.Scan(
			&rwe.Reminder.ID, &rwe.Reminder.EventID, &rwe.Reminder.MinutesBefore, &rwe.Reminder.CreatedAt,
			&rwe.Event.ID, &rwe.Event.CalendarID, &rwe.Event.Title, &rwe.Event.Description,
			&rwe.Event.Location, &rwe.Event.ResourceID, &rwe.Event.StartTime, &rwe.Event.EndTime,
			&rwe.Event.IsAllDay, &rwe.Event.Timezone, &rwe.Event.RRule, &rwe.Event.RecurrenceEnd,
			&rwe.Event.HasVideoCall, &rwe.Event.LiveKitRoomName, &rwe.Event.CategoryID,
			&rwe.Event.CreatedBy, &rwe.Event.CreatedAt, &rwe.Event.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, rwe)
	}
	return result, rows.Err()
}

// Task deadlines

func (r *PostgresRepository) ListTaskDeadlinesInRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]models.TaskDeadlineStub, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, t.title, t.due_date, p.project_key, t.priority
		 FROM tasks t
		 LEFT JOIN projects p ON t.project_id = p.id
		 WHERE t.due_date BETWEEN $2 AND $3
		   AND (t.assignee_id = $1
		        OR EXISTS (
		            SELECT 1 FROM project_members pm
		            WHERE pm.project_id = t.project_id AND pm.user_id = $1
		        ))
		 ORDER BY t.due_date ASC`,
		userID, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stubs []models.TaskDeadlineStub
	for rows.Next() {
		var s models.TaskDeadlineStub
		if scanErr := rows.Scan(&s.TaskID, &s.Title, &s.DueDate, &s.ProjectKey, &s.Priority); scanErr != nil {
			return nil, scanErr
		}
		stubs = append(stubs, s)
	}
	return stubs, rows.Err()
}
