package gdpr

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// timeEntryRetentionDays is the conservative minimum a time entry survives a
// GDPR erasure request before WorkErasureHandler hard-deletes it. Sec. 16(2)
// ArbZG obliges employers to retain records of working hours for two years;
// no ArbZG-specific column or constant exists elsewhere in the codebase for
// task-level time tracking, so this figure is the documentation duty applied
// conservatively rather than a value read out of an existing retention table.
const timeEntryRetentionDays = 730

// ErasureAction defines the type of erasure operation for a module.
type ErasureAction string

const (
	// ErasureAnonymize replaces personal data with anonymized labels.
	ErasureAnonymize ErasureAction = "anonymize"
	// ErasureDelete removes records entirely.
	ErasureDelete ErasureAction = "delete"
	// ErasureRetain keeps records unchanged (e.g., audit logs for compliance).
	ErasureRetain ErasureAction = "retain"
)

// ErasureHandler defines the interface for per-module GDPR right-to-erasure execution.
// Each module registers a handler to preview and execute user data erasure.
type ErasureHandler interface {
	// ModuleName returns the unique module identifier (e.g., "auth", "crm", "chat").
	ModuleName() string

	// PreviewErasure returns a preview of what erasure will affect in this module.
	// Does not modify any data.
	PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error)

	// ExecuteErasure performs the actual data erasure/anonymization for the given user.
	// The anonymizedLabel is used to replace personal data (e.g., "Geloeschter Benutzer #42").
	// The action specifies the erasure type (anonymize, delete, retain).
	// Returns the number of records affected.
	ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error)
}

// ---------------------------------------------------------------------------
// Per-module erasure handlers
// ---------------------------------------------------------------------------

// AuthErasureHandler handles erasure for authentication data.
// Anonymizes user profile (name/email), clears 2FA secrets, deletes sessions,
// recovery codes, refresh tokens, password history, password-reset tokens,
// and app-specific passwords.
type AuthErasureHandler struct {
	pool *pgxpool.Pool
}

// NewAuthErasureHandler creates a new AuthErasureHandler with DB access.
func NewAuthErasureHandler(pool *pgxpool.Pool) *AuthErasureHandler {
	return &AuthErasureHandler{pool: pool}
}

func (h *AuthErasureHandler) ModuleName() string { return "auth" }

func (h *AuthErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	var count int
	// Count: 1 user profile + sessions + refresh_tokens + recovery_codes + password_history
	// + password_reset_tokens + app_specific_passwords
	if err := h.pool.QueryRow(ctx,
		`SELECT
		   1 +
		   (SELECT COUNT(*) FROM user_sessions WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM recovery_codes WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM password_history WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM password_reset_tokens WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM app_specific_passwords WHERE user_id = $1)
		 FROM users WHERE id = $1`, userID,
	).Scan(&count); err != nil {
		// User might not exist — return 0
		return &models.ModuleErasurePreview{
			ModuleName:  "auth",
			RecordCount: 0,
			Action:      string(ErasureAnonymize),
		}, nil
	}

	return &models.ModuleErasurePreview{
		ModuleName:  "auth",
		RecordCount: count,
		Action:      string(ErasureAnonymize),
	}, nil
}

func (h *AuthErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("auth erasure: failed to begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	affected := 0

	// 1. Delete sessions
	res, err := tx.Exec(ctx, `DELETE FROM user_sessions WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("auth erasure: failed to delete sessions: %w", err)
	}
	affected += int(res.RowsAffected())

	// 2. Revoke and delete refresh tokens
	res, err = tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("auth erasure: failed to delete refresh tokens: %w", err)
	}
	affected += int(res.RowsAffected())

	// 3. Delete recovery codes
	res, err = tx.Exec(ctx, `DELETE FROM recovery_codes WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("auth erasure: failed to delete recovery codes: %w", err)
	}
	affected += int(res.RowsAffected())

	// 4. Delete password history
	res, err = tx.Exec(ctx, `DELETE FROM password_history WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("auth erasure: failed to delete password history: %w", err)
	}
	affected += int(res.RowsAffected())

	// 4a. Delete open password-reset tokens. A stale token left behind after
	// erasure would let anyone still holding the link set a password on an
	// already-anonymized account.
	res, err = tx.Exec(ctx, `DELETE FROM password_reset_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("auth erasure: failed to delete password reset tokens: %w", err)
	}
	affected += int(res.RowsAffected())

	// 4b. Delete app-specific passwords (CalDAV/CardDAV Basic Auth credentials
	// the user generated). These authenticate independently of the primary
	// password/2FA cleared below and must not survive erasure.
	res, err = tx.Exec(ctx, `DELETE FROM app_specific_passwords WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("auth erasure: failed to delete app-specific passwords: %w", err)
	}
	affected += int(res.RowsAffected())

	// 5. Anonymize user profile: clear PII, clear 2FA secrets, deactivate account
	res, err = tx.Exec(ctx,
		`UPDATE users SET
		   email = $2 || '@deleted.invalid',
		   first_name = $2,
		   last_name = 'ANONYMIZED',
		   password_hash = '',
		   is_active = false,
		   two_factor_enabled = false,
		   two_factor_secret_encrypted = NULL,
		   two_factor_pending_secret = NULL,
		   two_factor_enabled_at = NULL,
		   updated_at = NOW()
		 WHERE id = $1`, userID, anonymizedLabel,
	)
	if err != nil {
		return 0, fmt.Errorf("auth erasure: failed to anonymize user: %w", err)
	}
	affected += int(res.RowsAffected())

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return 0, fmt.Errorf("auth erasure: failed to commit: %w", commitErr)
	}

	slog.Info("gdpr auth erasure: complete",
		"user_id", userID,
		"records_affected", affected,
	)
	return affected, nil
}

// CRMErasureHandler handles erasure for CRM data.
// Anonymizes activities owned by the user. Contacts, companies and deal/pipeline
// history are retained unchanged: they are business records belonging to the
// tenant, not personal data of this user, and their created_by FK stays valid
// via the anonymized user sentinel (AuthErasureHandler). Preview and execute
// therefore both report only activities as affected -- contacts/companies are
// deliberately excluded from the count since nothing about them changes.
type CRMErasureHandler struct {
	pool *pgxpool.Pool
}

// NewCRMErasureHandler creates a new CRMErasureHandler with DB access.
func NewCRMErasureHandler(pool *pgxpool.Pool) *CRMErasureHandler {
	return &CRMErasureHandler{pool: pool}
}

func (h *CRMErasureHandler) ModuleName() string { return "crm" }

func (h *CRMErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	var count int
	if err := h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM activities WHERE assigned_to = $1 OR created_by = $1`, userID,
	).Scan(&count); err != nil {
		return &models.ModuleErasurePreview{
			ModuleName:  "crm",
			RecordCount: 0,
			Action:      string(ErasureAnonymize),
		}, nil
	}

	return &models.ModuleErasurePreview{
		ModuleName:  "crm",
		RecordCount: count,
		Action:      string(ErasureAnonymize),
	}, nil
}

func (h *CRMErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("crm erasure: failed to begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	affected := 0

	// Anonymize activities assigned to and/or created by this user in a single UPDATE,
	// so a row that is both (assigned to self, created by self) is only counted once.
	// activities.created_by is NOT NULL FK; the user record persists as anonymized sentinel,
	// so FK integrity is maintained regardless of which branch touches the row.
	res, err := tx.Exec(ctx,
		`UPDATE activities SET
		   assigned_to = CASE WHEN assigned_to = $1 THEN NULL ELSE assigned_to END,
		   description = NULL,
		   updated_at = NOW()
		 WHERE assigned_to = $1 OR created_by = $1`, userID,
	)
	if err != nil {
		return 0, fmt.Errorf("crm erasure: failed to anonymize activities: %w", err)
	}
	affected += int(res.RowsAffected())

	// contacts.created_by and companies.created_by are NOT NULL FKs to users(id).
	// The user record is anonymized (not deleted) by AuthErasureHandler, so the FK
	// reference remains valid with the anonymized user as creator. Nothing about
	// these rows changes, so they are deliberately not counted as affected --
	// counting them without touching them made the preview lie about impact.

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return 0, fmt.Errorf("crm erasure: failed to commit: %w", commitErr)
	}

	slog.Info("gdpr crm erasure: complete",
		"user_id", userID,
		"records_affected", affected,
	)
	return affected, nil
}

// ChatErasureHandler handles erasure for chat data.
// Anonymizes message content and removes channel memberships.
type ChatErasureHandler struct {
	pool *pgxpool.Pool
}

// NewChatErasureHandler creates a new ChatErasureHandler with DB access.
func NewChatErasureHandler(pool *pgxpool.Pool) *ChatErasureHandler {
	return &ChatErasureHandler{pool: pool}
}

func (h *ChatErasureHandler) ModuleName() string { return "chat" }

func (h *ChatErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	var count int
	if err := h.pool.QueryRow(ctx,
		`SELECT
		   (SELECT COUNT(*) FROM messages WHERE created_by = $1) +
		   (SELECT COUNT(*) FROM channel_memberships WHERE user_id = $1)
		`, userID,
	).Scan(&count); err != nil {
		return &models.ModuleErasurePreview{
			ModuleName:  "chat",
			RecordCount: 0,
			Action:      string(ErasureAnonymize),
		}, nil
	}

	return &models.ModuleErasurePreview{
		ModuleName:  "chat",
		RecordCount: count,
		Action:      string(ErasureAnonymize),
	}, nil
}

func (h *ChatErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("chat erasure: failed to begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	affected := 0

	// Anonymize message content (keep message skeleton for thread integrity)
	res, err := tx.Exec(ctx,
		`UPDATE messages SET
		   content = '[' || $2 || ']',
		   is_deleted = true,
		   edited_at = NOW()
		 WHERE created_by = $1`, userID, anonymizedLabel,
	)
	if err != nil {
		return 0, fmt.Errorf("chat erasure: failed to anonymize messages: %w", err)
	}
	affected += int(res.RowsAffected())

	// Remove channel memberships
	res, err = tx.Exec(ctx,
		`DELETE FROM channel_memberships WHERE user_id = $1`, userID,
	)
	if err != nil {
		return 0, fmt.Errorf("chat erasure: failed to delete channel memberships: %w", err)
	}
	affected += int(res.RowsAffected())

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return 0, fmt.Errorf("chat erasure: failed to commit: %w", commitErr)
	}

	slog.Info("gdpr chat erasure: complete",
		"user_id", userID,
		"records_affected", affected,
	)
	return affected, nil
}

// WorkErasureHandler handles erasure for work/project data.
// Anonymizes task assignments and comments; deletes personal time entries.
type WorkErasureHandler struct {
	pool *pgxpool.Pool
}

// NewWorkErasureHandler creates a new WorkErasureHandler with DB access.
func NewWorkErasureHandler(pool *pgxpool.Pool) *WorkErasureHandler {
	return &WorkErasureHandler{pool: pool}
}

func (h *WorkErasureHandler) ModuleName() string { return "work" }

func (h *WorkErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	var count int
	if err := h.pool.QueryRow(ctx,
		`SELECT
		   (SELECT COUNT(*) FROM tasks WHERE assignee_id = $1 OR created_by = $1) +
		   (SELECT COUNT(*) FROM time_entries WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM task_comments WHERE author_id = $1)
		`, userID,
	).Scan(&count); err != nil {
		return &models.ModuleErasurePreview{
			ModuleName:  "work",
			RecordCount: 0,
			Action:      string(ErasureAnonymize),
		}, nil
	}

	return &models.ModuleErasurePreview{
		ModuleName:  "work",
		RecordCount: count,
		Action:      string(ErasureAnonymize),
	}, nil
}

func (h *WorkErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("work erasure: failed to begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	affected := 0

	// Unassign tasks and count tasks created by the subject in a single UPDATE,
	// so a task that is both (assigned to self, created by self) is only counted
	// once. tasks.created_by is NOT NULL FK; the user record persists as
	// anonymized sentinel, so FK integrity is maintained regardless of which
	// branch touches the row.
	res, err := tx.Exec(ctx,
		`UPDATE tasks SET
		   assignee_id = CASE WHEN assignee_id = $1 THEN NULL ELSE assignee_id END,
		   updated_at = NOW()
		 WHERE assignee_id = $1 OR created_by = $1`, userID,
	)
	if err != nil {
		return 0, fmt.Errorf("work erasure: failed to unassign tasks: %w", err)
	}
	affected += int(res.RowsAffected())

	// Delete only time entries older than the retention window; entries still
	// inside it survive untouched. Sec. 16(2) ArbZG requires employers to keep
	// working-time records for two years, and that duty does not lapse because
	// the employee later requests erasure. This does not leak identity: the
	// users row is anonymized by AuthErasureHandler, registered before this
	// handler (cmd/auth/main.go), so the surviving user_id FK already points at
	// an anonymized sentinel -- the same reasoning tasks.created_by above and
	// contacts/companies (CRMErasureHandler) already rely on.
	res, err = tx.Exec(ctx,
		`DELETE FROM time_entries WHERE user_id = $1 AND started_at < $2`,
		userID, time.Now().UTC().AddDate(0, 0, -timeEntryRetentionDays),
	)
	if err != nil {
		return 0, fmt.Errorf("work erasure: failed to delete time entries: %w", err)
	}
	affected += int(res.RowsAffected())

	// Anonymize task comments (keep for task history, remove author identity)
	res, err = tx.Exec(ctx,
		`UPDATE task_comments SET
		   content = '[' || $2 || ']',
		   updated_at = NOW()
		 WHERE author_id = $1`, userID, anonymizedLabel,
	)
	if err != nil {
		return 0, fmt.Errorf("work erasure: failed to anonymize task comments: %w", err)
	}
	affected += int(res.RowsAffected())

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return 0, fmt.Errorf("work erasure: failed to commit: %w", commitErr)
	}

	slog.Info("gdpr work erasure: complete",
		"user_id", userID,
		"records_affected", affected,
	)
	return affected, nil
}

// CalendarErasureHandler handles erasure for calendar data.
// Deletes the user's own personal calendars (cascading to their events,
// attendees, exceptions and reminders) unless a booking_pages row still
// depends on the calendar, in which case it is kept intact to protect the
// public_bookings customer records attached to it. Any remaining event the
// user created elsewhere (e.g. on a colleague's shared calendar) is
// anonymized, not deleted, so the event survives for other attendees.
// Removes attendee records, calendar memberships, CalDAV push registrations,
// personal event categories and calendar preferences.
type CalendarErasureHandler struct {
	pool *pgxpool.Pool
}

// NewCalendarErasureHandler creates a new CalendarErasureHandler with DB access.
func NewCalendarErasureHandler(pool *pgxpool.Pool) *CalendarErasureHandler {
	return &CalendarErasureHandler{pool: pool}
}

func (h *CalendarErasureHandler) ModuleName() string { return "calendar" }

func (h *CalendarErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	var count int
	if err := h.pool.QueryRow(ctx,
		`SELECT
		   (SELECT COUNT(*) FROM calendars c WHERE c.owner_id = $1 AND c.calendar_type = 'personal'
		      AND NOT EXISTS (SELECT 1 FROM booking_pages bp WHERE bp.calendar_id = c.id)) +
		   (SELECT COUNT(*) FROM calendar_events WHERE created_by = $1) +
		   (SELECT COUNT(*) FROM event_attendees WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM calendar_members WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM caldav_push_subscriptions WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM event_categories WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM user_calendar_preferences WHERE user_id = $1)
		`, userID,
	).Scan(&count); err != nil {
		return &models.ModuleErasurePreview{
			ModuleName:  "calendar",
			RecordCount: 0,
			Action:      string(ErasureDelete),
		}, nil
	}

	return &models.ModuleErasurePreview{
		ModuleName:  "calendar",
		RecordCount: count,
		Action:      string(ErasureDelete),
	}, nil
}

func (h *CalendarErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("calendar erasure: failed to begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	affected := 0

	// Pre-count rows that the personal-calendar cascade below can silently
	// remove before their own DELETE/UPDATE statements run, so the reported
	// total stays in sync with what PreviewErasure showed.
	var eventCount, attendeeCount, memberCount int
	if err = tx.QueryRow(ctx,
		`SELECT
		   (SELECT COUNT(*) FROM calendar_events WHERE created_by = $1),
		   (SELECT COUNT(*) FROM event_attendees WHERE user_id = $1),
		   (SELECT COUNT(*) FROM calendar_members WHERE user_id = $1)
		`, userID,
	).Scan(&eventCount, &attendeeCount, &memberCount); err != nil {
		return 0, fmt.Errorf("calendar erasure: failed to count affected rows: %w", err)
	}

	// Delete the user's personal calendars. calendar_id ON DELETE CASCADE takes
	// their events, attendee records, exceptions and reminders with them in the
	// same statement -- already counted above, not via RowsAffected here.
	// Calendars still referenced by a booking_pages row are excluded: deleting
	// them would cascade-delete the booking page and, with it, the
	// public_bookings rows holding real customer name/email/phone data that
	// has nothing to do with this erasure subject.
	res, err := tx.Exec(ctx,
		`DELETE FROM calendars c
		 WHERE c.owner_id = $1 AND c.calendar_type = 'personal'
		   AND NOT EXISTS (SELECT 1 FROM booking_pages bp WHERE bp.calendar_id = c.id)`,
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf("calendar erasure: failed to delete personal calendars: %w", err)
	}
	affected += int(res.RowsAffected())

	// Anonymize any event the user created that is still standing (on a
	// shared/booking-page calendar, or their own if retained above): the event
	// stays for other attendees or the booking page, only the
	// organizer-authored content disappears.
	if _, err = tx.Exec(ctx,
		`UPDATE calendar_events SET
		   title = '[' || $2 || ']',
		   description = NULL,
		   location = NULL,
		   updated_at = NOW()
		 WHERE created_by = $1`, userID, anonymizedLabel,
	); err != nil {
		return 0, fmt.Errorf("calendar erasure: failed to anonymize remaining events: %w", err)
	}
	affected += eventCount

	// Remove attendee records (participation data is personal)
	if _, err = tx.Exec(ctx,
		`DELETE FROM event_attendees WHERE user_id = $1`, userID,
	); err != nil {
		return 0, fmt.Errorf("calendar erasure: failed to delete attendances: %w", err)
	}
	affected += attendeeCount

	// Remove membership in other users' shared calendars.
	if _, err = tx.Exec(ctx,
		`DELETE FROM calendar_members WHERE user_id = $1`, userID,
	); err != nil {
		return 0, fmt.Errorf("calendar erasure: failed to delete calendar memberships: %w", err)
	}
	affected += memberCount

	// Remove CalDAV push registrations (device sync callbacks). CASCADE on
	// users(id) never fires because erasure anonymizes the users row in place
	// instead of deleting it.
	res, err = tx.Exec(ctx,
		`DELETE FROM caldav_push_subscriptions WHERE user_id = $1`, userID,
	)
	if err != nil {
		return 0, fmt.Errorf("calendar erasure: failed to delete push subscriptions: %w", err)
	}
	affected += int(res.RowsAffected())

	// Remove personal event categories. ON DELETE SET NULL on
	// calendar_events.category_id clears the reference on any surviving event
	// automatically.
	res, err = tx.Exec(ctx,
		`DELETE FROM event_categories WHERE user_id = $1`, userID,
	)
	if err != nil {
		return 0, fmt.Errorf("calendar erasure: failed to delete event categories: %w", err)
	}
	affected += int(res.RowsAffected())

	// Delete calendar preferences
	res, err = tx.Exec(ctx,
		`DELETE FROM user_calendar_preferences WHERE user_id = $1`, userID,
	)
	if err != nil {
		return 0, fmt.Errorf("calendar erasure: failed to delete calendar preferences: %w", err)
	}
	affected += int(res.RowsAffected())

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return 0, fmt.Errorf("calendar erasure: failed to commit: %w", commitErr)
	}

	slog.Info("gdpr calendar erasure: complete",
		"user_id", userID,
		"records_affected", affected,
	)
	return affected, nil
}

// NotificationErasureHandler handles erasure for notification data.
// Deletes notification preferences, quiet hours, mute rules, and history.
type NotificationErasureHandler struct {
	pool *pgxpool.Pool
}

// NewNotificationErasureHandler creates a new NotificationErasureHandler with DB access.
func NewNotificationErasureHandler(pool *pgxpool.Pool) *NotificationErasureHandler {
	return &NotificationErasureHandler{pool: pool}
}

func (h *NotificationErasureHandler) ModuleName() string { return "notifications" }

func (h *NotificationErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	var count int
	if err := h.pool.QueryRow(ctx,
		`SELECT
		   (SELECT COUNT(*) FROM notifications WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM notification_preferences WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM notification_quiet_hours WHERE user_id = $1) +
		   (SELECT COUNT(*) FROM notification_mutes WHERE user_id = $1)
		`, userID,
	).Scan(&count); err != nil {
		return &models.ModuleErasurePreview{
			ModuleName:  "notifications",
			RecordCount: 0,
			Action:      string(ErasureDelete),
		}, nil
	}

	return &models.ModuleErasurePreview{
		ModuleName:  "notifications",
		RecordCount: count,
		Action:      string(ErasureDelete),
	}, nil
}

func (h *NotificationErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("notifications erasure: failed to begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	affected := 0

	// Delete notification history
	res, err := tx.Exec(ctx, `DELETE FROM notifications WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("notifications erasure: failed to delete notifications: %w", err)
	}
	affected += int(res.RowsAffected())

	// Delete preferences
	res, err = tx.Exec(ctx, `DELETE FROM notification_preferences WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("notifications erasure: failed to delete preferences: %w", err)
	}
	affected += int(res.RowsAffected())

	// Delete quiet hours config
	res, err = tx.Exec(ctx, `DELETE FROM notification_quiet_hours WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("notifications erasure: failed to delete quiet hours: %w", err)
	}
	affected += int(res.RowsAffected())

	// Delete mute rules
	res, err = tx.Exec(ctx, `DELETE FROM notification_mutes WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("notifications erasure: failed to delete mutes: %w", err)
	}
	affected += int(res.RowsAffected())

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return 0, fmt.Errorf("notifications erasure: failed to commit: %w", commitErr)
	}

	slog.Info("gdpr notifications erasure: complete",
		"user_id", userID,
		"records_affected", affected,
	)
	return affected, nil
}

// AuditErasureHandler is a no-op handler for audit logs.
// Audit logs are retained for compliance requirements (cannot be erased per DSGVO Art. 17(3)(e)).
type AuditErasureHandler struct{}

func (h *AuditErasureHandler) ModuleName() string { return "audit" }

func (h *AuditErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	// Audit logs are retained for legal compliance -- no erasure possible.
	return &models.ModuleErasurePreview{
		ModuleName:  "audit",
		RecordCount: 0,
		Action:      string(ErasureRetain),
	}, nil
}

func (h *AuditErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	// No-op: audit logs must be retained for legal compliance (DSGVO Art. 17(3)(e)).
	// The user_id reference in audit_log remains for legal traceability.
	slog.Info("gdpr audit erasure: retained per DSGVO Art. 17(3)(e)", "user_id", userID)
	return 0, nil
}
