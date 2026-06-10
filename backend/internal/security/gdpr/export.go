package gdpr

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DataExportHandler defines the interface for per-module GDPR data export.
// Each module registers a handler to export its user-owned data.
type DataExportHandler interface {
	// ModuleName returns the unique module identifier (e.g., "auth", "crm", "chat").
	ModuleName() string

	// ExportUserData exports all data belonging to the given user from this module.
	// Returns the data as JSON bytes. Returns nil, nil if the module has no data for this user.
	ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error)
}

// ExecuteExport iterates over all registered export handlers, collects each module's
// user data as JSON, and packages everything into a ZIP archive.
// Each module's data becomes a separate JSON file in the archive.
func ExecuteExport(ctx context.Context, userID uuid.UUID, handlers []DataExportHandler) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add metadata file with export information
	metadata := map[string]interface{}{
		"export_type":    "gdpr_data_export",
		"user_id":        userID.String(),
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
		"modules_count":  len(handlers),
		"format_version": "1.0",
	}

	metaJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("gdpr: failed to marshal export metadata: %w", err)
	}

	metaFile, err := zipWriter.Create("_metadata.json")
	if err != nil {
		return nil, fmt.Errorf("gdpr: failed to create metadata entry in zip: %w", err)
	}
	if _, err := metaFile.Write(metaJSON); err != nil {
		return nil, fmt.Errorf("gdpr: failed to write metadata to zip: %w", err)
	}

	// Collect data from each module handler
	for _, handler := range handlers {
		moduleName := handler.ModuleName()

		data, exportErr := handler.ExportUserData(ctx, userID)
		if exportErr != nil {
			slog.Error("gdpr export: module export failed",
				"module", moduleName,
				"user_id", userID,
				"error", exportErr,
			)
			// Write error marker file for this module
			errFile, createErr := zipWriter.Create(fmt.Sprintf("%s/_error.txt", moduleName))
			if createErr != nil {
				continue
			}
			fmt.Fprintf(errFile, "Export failed: %s", exportErr.Error())
			continue
		}

		if len(data) == 0 {
			slog.Info("gdpr export: no data for module",
				"module", moduleName,
				"user_id", userID,
			)
			continue
		}

		fileName := fmt.Sprintf("%s/data.json", moduleName)
		file, createErr := zipWriter.Create(fileName)
		if createErr != nil {
			return nil, fmt.Errorf("gdpr: failed to create zip entry for %s: %w", moduleName, createErr)
		}
		if _, writeErr := file.Write(data); writeErr != nil {
			return nil, fmt.Errorf("gdpr: failed to write data for %s: %w", moduleName, writeErr)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("gdpr: failed to finalize zip archive: %w", err)
	}

	return buf.Bytes(), nil
}

// ---------------------------------------------------------------------------
// Per-module export handlers
// ---------------------------------------------------------------------------

// AuthExportHandler exports authentication-related user data.
// Tables: users (own profile), user_sessions, refresh_tokens, recovery_codes (count only),
//         password_history (count only -- hashes excluded for security).
type AuthExportHandler struct {
	pool *pgxpool.Pool
}

// NewAuthExportHandler creates a new AuthExportHandler with DB access.
func NewAuthExportHandler(pool *pgxpool.Pool) *AuthExportHandler {
	return &AuthExportHandler{pool: pool}
}

func (h *AuthExportHandler) ModuleName() string { return "auth" }

func (h *AuthExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	type userProfile struct {
		ID               uuid.UUID  `json:"id"`
		Email            string     `json:"email"`
		FirstName        string     `json:"first_name"`
		LastName         string     `json:"last_name"`
		IsActive         bool       `json:"is_active"`
		TwoFactorEnabled bool       `json:"two_factor_enabled"`
		TwoFactorEnabledAt *time.Time `json:"two_factor_enabled_at,omitempty"`
		Locale           string     `json:"locale"`
		CreatedAt        time.Time  `json:"created_at"`
		UpdatedAt        time.Time  `json:"updated_at"`
	}

	type sessionEntry struct {
		ID           uuid.UUID  `json:"id"`
		DeviceName   string     `json:"device_name,omitempty"`
		DeviceType   string     `json:"device_type,omitempty"`
		IPAddress    *string    `json:"ip_address,omitempty"`
		Location     string     `json:"location,omitempty"`
		IsCurrent    bool       `json:"is_current"`
		LastActiveAt time.Time  `json:"last_active_at"`
		CreatedAt    time.Time  `json:"created_at"`
	}

	type result struct {
		Profile              *userProfile   `json:"profile"`
		Sessions             []sessionEntry `json:"sessions"`
		ActiveRefreshTokens  int            `json:"active_refresh_tokens_count"`
		PasswordHistoryCount int            `json:"password_history_count"`
	}

	var out result

	// User profile (including 2FA status; secret NOT exported)
	var profile userProfile
	err := h.pool.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, is_active,
		        two_factor_enabled, two_factor_enabled_at, locale, created_at, updated_at
		 FROM users WHERE id = $1`, userID,
	).Scan(
		&profile.ID, &profile.Email, &profile.FirstName, &profile.LastName,
		&profile.IsActive, &profile.TwoFactorEnabled, &profile.TwoFactorEnabledAt,
		&profile.Locale, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("auth export: failed to query user profile: %w", err)
	}
	out.Profile = &profile

	// Sessions (device/IP metadata -- no token hashes)
	rows, err := h.pool.Query(ctx,
		`SELECT id, COALESCE(device_name, ''), COALESCE(device_type, ''),
		        CASE WHEN ip_address IS NOT NULL THEN host(ip_address) ELSE NULL END,
		        COALESCE(location, ''), is_current, last_active_at, created_at
		 FROM user_sessions WHERE user_id = $1 ORDER BY last_active_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("auth export: failed to query sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s sessionEntry
		if scanErr := rows.Scan(
			&s.ID, &s.DeviceName, &s.DeviceType, &s.IPAddress,
			&s.Location, &s.IsCurrent, &s.LastActiveAt, &s.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("auth export: failed to scan session: %w", scanErr)
		}
		out.Sessions = append(out.Sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auth export: session rows error: %w", err)
	}

	// Active refresh token count (no hashes -- security)
	if err := h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1 AND revoked = false AND expires_at > NOW()`,
		userID,
	).Scan(&out.ActiveRefreshTokens); err != nil {
		slog.Warn("auth export: failed to count refresh tokens", "user_id", userID, "error", err)
	}

	// Password history entry count (no hashes -- security)
	if err := h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM password_history WHERE user_id = $1`, userID,
	).Scan(&out.PasswordHistoryCount); err != nil {
		slog.Warn("auth export: failed to count password history", "user_id", userID, "error", err)
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("auth export: failed to marshal: %w", err)
	}
	return data, nil
}

// CRMExportHandler exports CRM-related user data.
// Tables: contacts (created_by), companies (created_by), activities (assigned_to or created_by).
// Datensparsamkeit: contacts/companies of other users are excluded; only user-owned records.
type CRMExportHandler struct {
	pool *pgxpool.Pool
}

// NewCRMExportHandler creates a new CRMExportHandler with DB access.
func NewCRMExportHandler(pool *pgxpool.Pool) *CRMExportHandler {
	return &CRMExportHandler{pool: pool}
}

func (h *CRMExportHandler) ModuleName() string { return "crm" }

func (h *CRMExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	type contactEntry struct {
		ID        uuid.UUID  `json:"id"`
		FirstName string     `json:"first_name"`
		LastName  string     `json:"last_name"`
		Email     *string    `json:"email,omitempty"`
		Phone     *string    `json:"phone,omitempty"`
		Position  *string    `json:"position,omitempty"`
		CreatedAt time.Time  `json:"created_at"`
	}

	type companyEntry struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		Domain    *string   `json:"domain,omitempty"`
		Industry  *string   `json:"industry,omitempty"`
		CreatedAt time.Time `json:"created_at"`
	}

	type activityEntry struct {
		ID           uuid.UUID  `json:"id"`
		ActivityType string     `json:"activity_type"`
		Subject      string     `json:"subject"`
		Description  *string    `json:"description,omitempty"`
		DueDate      *time.Time `json:"due_date,omitempty"`
		CompletedAt  *time.Time `json:"completed_at,omitempty"`
		CreatedAt    time.Time  `json:"created_at"`
	}

	type result struct {
		CreatedContacts  []contactEntry  `json:"created_contacts"`
		CreatedCompanies []companyEntry  `json:"created_companies"`
		AssignedActivities []activityEntry `json:"assigned_activities"`
	}

	var out result

	// Contacts created by this user (tenant-scoped via users.tenant_id)
	contactRows, err := h.pool.Query(ctx,
		`SELECT c.id, c.first_name, c.last_name, c.email, c.phone, c.position, c.created_at
		 FROM contacts c
		 JOIN users u ON u.id = $1
		 WHERE c.created_by = $1 AND c.tenant_id = u.tenant_id
		 ORDER BY c.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("crm export: failed to query contacts: %w", err)
	}
	defer contactRows.Close()
	for contactRows.Next() {
		var c contactEntry
		if scanErr := contactRows.Scan(
			&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone, &c.Position, &c.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("crm export: failed to scan contact: %w", scanErr)
		}
		out.CreatedContacts = append(out.CreatedContacts, c)
	}
	if err := contactRows.Err(); err != nil {
		return nil, fmt.Errorf("crm export: contact rows error: %w", err)
	}

	// Companies created by this user
	companyRows, err := h.pool.Query(ctx,
		`SELECT co.id, co.name, co.domain, co.industry, co.created_at
		 FROM companies co
		 JOIN users u ON u.id = $1
		 WHERE co.created_by = $1 AND co.tenant_id = u.tenant_id
		 ORDER BY co.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("crm export: failed to query companies: %w", err)
	}
	defer companyRows.Close()
	for companyRows.Next() {
		var co companyEntry
		if scanErr := companyRows.Scan(
			&co.ID, &co.Name, &co.Domain, &co.Industry, &co.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("crm export: failed to scan company: %w", scanErr)
		}
		out.CreatedCompanies = append(out.CreatedCompanies, co)
	}
	if err := companyRows.Err(); err != nil {
		return nil, fmt.Errorf("crm export: company rows error: %w", err)
	}

	// Activities assigned to or created by this user
	actRows, err := h.pool.Query(ctx,
		`SELECT a.id, a.activity_type::text, a.subject, a.description, a.due_date, a.completed_at, a.created_at
		 FROM activities a
		 JOIN users u ON u.id = $1
		 WHERE (a.assigned_to = $1 OR a.created_by = $1)
		   AND a.tenant_id = u.tenant_id
		 ORDER BY a.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("crm export: failed to query activities: %w", err)
	}
	defer actRows.Close()
	for actRows.Next() {
		var a activityEntry
		if scanErr := actRows.Scan(
			&a.ID, &a.ActivityType, &a.Subject, &a.Description, &a.DueDate, &a.CompletedAt, &a.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("crm export: failed to scan activity: %w", scanErr)
		}
		out.AssignedActivities = append(out.AssignedActivities, a)
	}
	if err := actRows.Err(); err != nil {
		return nil, fmt.Errorf("crm export: activity rows error: %w", err)
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("crm export: failed to marshal: %w", err)
	}
	return data, nil
}

// ChatExportHandler exports chat messages and channel memberships for the user.
// Tables: messages (created_by), channel_memberships (user_id).
// Datensparsamkeit: only messages authored by the user; channel names included for context.
type ChatExportHandler struct {
	pool *pgxpool.Pool
}

// NewChatExportHandler creates a new ChatExportHandler with DB access.
func NewChatExportHandler(pool *pgxpool.Pool) *ChatExportHandler {
	return &ChatExportHandler{pool: pool}
}

func (h *ChatExportHandler) ModuleName() string { return "chat" }

func (h *ChatExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	type messageEntry struct {
		ID        uuid.UUID  `json:"id"`
		ChannelID uuid.UUID  `json:"channel_id"`
		Content   string     `json:"content"`
		IsDeleted bool       `json:"is_deleted"`
		EditedAt  *time.Time `json:"edited_at,omitempty"`
		CreatedAt time.Time  `json:"created_at"`
	}

	type membershipEntry struct {
		ChannelID   uuid.UUID  `json:"channel_id"`
		ChannelName string     `json:"channel_name"`
		Role        string     `json:"role"`
		JoinedAt    time.Time  `json:"joined_at"`
		LastReadAt  *time.Time `json:"last_read_at,omitempty"`
	}

	type result struct {
		Messages    []messageEntry    `json:"messages"`
		Memberships []membershipEntry `json:"memberships"`
	}

	var out result

	// Messages authored by this user (tenant-scoped)
	msgRows, err := h.pool.Query(ctx,
		`SELECT m.id, m.channel_id, m.content, m.is_deleted, m.edited_at, m.created_at
		 FROM messages m
		 JOIN users u ON u.id = $1
		 WHERE m.created_by = $1 AND m.tenant_id = u.tenant_id
		 ORDER BY m.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("chat export: failed to query messages: %w", err)
	}
	defer msgRows.Close()
	for msgRows.Next() {
		var msg messageEntry
		if scanErr := msgRows.Scan(
			&msg.ID, &msg.ChannelID, &msg.Content, &msg.IsDeleted, &msg.EditedAt, &msg.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("chat export: failed to scan message: %w", scanErr)
		}
		out.Messages = append(out.Messages, msg)
	}
	if err := msgRows.Err(); err != nil {
		return nil, fmt.Errorf("chat export: message rows error: %w", err)
	}

	// Channel memberships (tenant-scoped via channel's tenant_id)
	memRows, err := h.pool.Query(ctx,
		`SELECT cm.channel_id, c.name, cm.role, cm.joined_at, cm.last_read_at
		 FROM channel_memberships cm
		 JOIN channels c ON c.id = cm.channel_id
		 JOIN users u ON u.id = $1
		 WHERE cm.user_id = $1 AND c.tenant_id = u.tenant_id
		 ORDER BY cm.joined_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("chat export: failed to query memberships: %w", err)
	}
	defer memRows.Close()
	for memRows.Next() {
		var mem membershipEntry
		if scanErr := memRows.Scan(
			&mem.ChannelID, &mem.ChannelName, &mem.Role, &mem.JoinedAt, &mem.LastReadAt,
		); scanErr != nil {
			return nil, fmt.Errorf("chat export: failed to scan membership: %w", scanErr)
		}
		out.Memberships = append(out.Memberships, mem)
	}
	if err := memRows.Err(); err != nil {
		return nil, fmt.Errorf("chat export: membership rows error: %w", err)
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("chat export: failed to marshal: %w", err)
	}
	return data, nil
}

// WorkExportHandler exports work/project data for the user.
// Tables: tasks (created_by or assignee_id), time_entries (user_id), task_comments (author_id).
type WorkExportHandler struct {
	pool *pgxpool.Pool
}

// NewWorkExportHandler creates a new WorkExportHandler with DB access.
func NewWorkExportHandler(pool *pgxpool.Pool) *WorkExportHandler {
	return &WorkExportHandler{pool: pool}
}

func (h *WorkExportHandler) ModuleName() string { return "work" }

func (h *WorkExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	type taskEntry struct {
		ID          uuid.UUID  `json:"id"`
		Title       string     `json:"title"`
		Description *string    `json:"description,omitempty"`
		Priority    string     `json:"priority"`
		DueDate     *time.Time `json:"due_date,omitempty"`
		CreatedAt   time.Time  `json:"created_at"`
		CompletedAt *time.Time `json:"completed_at,omitempty"`
	}

	type timeEntry struct {
		ID              uuid.UUID  `json:"id"`
		TaskID          uuid.UUID  `json:"task_id"`
		StartedAt       time.Time  `json:"started_at"`
		EndedAt         *time.Time `json:"ended_at,omitempty"`
		DurationSeconds *int       `json:"duration_seconds,omitempty"`
		Description     *string    `json:"description,omitempty"`
		IsManual        bool       `json:"is_manual"`
		CreatedAt       time.Time  `json:"created_at"`
	}

	type commentEntry struct {
		ID        uuid.UUID `json:"id"`
		TaskID    uuid.UUID `json:"task_id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}

	type result struct {
		CreatedTasks   []taskEntry    `json:"created_tasks"`
		AssignedTasks  []taskEntry    `json:"assigned_tasks"`
		TimeEntries    []timeEntry    `json:"time_entries"`
		Comments       []commentEntry `json:"comments"`
	}

	var out result

	// Tasks created by this user
	createdTaskRows, err := h.pool.Query(ctx,
		`SELECT t.id, t.title, t.description, t.priority, t.due_date, t.created_at, t.completed_at
		 FROM tasks t
		 JOIN users u ON u.id = $1
		 WHERE t.created_by = $1 AND t.tenant_id = u.tenant_id
		 ORDER BY t.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("work export: failed to query created tasks: %w", err)
	}
	defer createdTaskRows.Close()
	for createdTaskRows.Next() {
		var t taskEntry
		if scanErr := createdTaskRows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Priority, &t.DueDate, &t.CreatedAt, &t.CompletedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("work export: failed to scan task: %w", scanErr)
		}
		out.CreatedTasks = append(out.CreatedTasks, t)
	}
	if err := createdTaskRows.Err(); err != nil {
		return nil, fmt.Errorf("work export: created task rows error: %w", err)
	}

	// Tasks assigned to this user (excluding those also created by, to avoid duplicates)
	assignedTaskRows, err := h.pool.Query(ctx,
		`SELECT t.id, t.title, t.description, t.priority, t.due_date, t.created_at, t.completed_at
		 FROM tasks t
		 JOIN users u ON u.id = $1
		 WHERE t.assignee_id = $1 AND t.created_by != $1 AND t.tenant_id = u.tenant_id
		 ORDER BY t.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("work export: failed to query assigned tasks: %w", err)
	}
	defer assignedTaskRows.Close()
	for assignedTaskRows.Next() {
		var t taskEntry
		if scanErr := assignedTaskRows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Priority, &t.DueDate, &t.CreatedAt, &t.CompletedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("work export: failed to scan assigned task: %w", scanErr)
		}
		out.AssignedTasks = append(out.AssignedTasks, t)
	}
	if err := assignedTaskRows.Err(); err != nil {
		return nil, fmt.Errorf("work export: assigned task rows error: %w", err)
	}

	// Time entries logged by this user
	teRows, err := h.pool.Query(ctx,
		`SELECT te.id, te.task_id, te.started_at, te.ended_at, te.duration_seconds,
		        te.description, te.is_manual, te.created_at
		 FROM time_entries te
		 JOIN users u ON u.id = $1
		 WHERE te.user_id = $1 AND te.tenant_id = u.tenant_id
		 ORDER BY te.started_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("work export: failed to query time entries: %w", err)
	}
	defer teRows.Close()
	for teRows.Next() {
		var te timeEntry
		if scanErr := teRows.Scan(
			&te.ID, &te.TaskID, &te.StartedAt, &te.EndedAt, &te.DurationSeconds,
			&te.Description, &te.IsManual, &te.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("work export: failed to scan time entry: %w", scanErr)
		}
		out.TimeEntries = append(out.TimeEntries, te)
	}
	if err := teRows.Err(); err != nil {
		return nil, fmt.Errorf("work export: time entry rows error: %w", err)
	}

	// Task comments authored by this user
	commentRows, err := h.pool.Query(ctx,
		`SELECT tc.id, tc.task_id, tc.content, tc.created_at
		 FROM task_comments tc
		 JOIN tasks t ON t.id = tc.task_id
		 JOIN users u ON u.id = $1
		 WHERE tc.author_id = $1 AND t.tenant_id = u.tenant_id
		 ORDER BY tc.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("work export: failed to query comments: %w", err)
	}
	defer commentRows.Close()
	for commentRows.Next() {
		var c commentEntry
		if scanErr := commentRows.Scan(&c.ID, &c.TaskID, &c.Content, &c.CreatedAt); scanErr != nil {
			return nil, fmt.Errorf("work export: failed to scan comment: %w", scanErr)
		}
		out.Comments = append(out.Comments, c)
	}
	if err := commentRows.Err(); err != nil {
		return nil, fmt.Errorf("work export: comment rows error: %w", err)
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("work export: failed to marshal: %w", err)
	}
	return data, nil
}

// CalendarExportHandler exports calendar events, attendances, and preferences.
// Tables: calendar_events (created_by), event_attendees (user_id), user_calendar_preferences.
type CalendarExportHandler struct {
	pool *pgxpool.Pool
}

// NewCalendarExportHandler creates a new CalendarExportHandler with DB access.
func NewCalendarExportHandler(pool *pgxpool.Pool) *CalendarExportHandler {
	return &CalendarExportHandler{pool: pool}
}

func (h *CalendarExportHandler) ModuleName() string { return "calendar" }

func (h *CalendarExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	type eventEntry struct {
		ID         uuid.UUID  `json:"id"`
		Title      string     `json:"title"`
		Description *string   `json:"description,omitempty"`
		Location   *string    `json:"location,omitempty"`
		StartTime  time.Time  `json:"start_time"`
		EndTime    time.Time  `json:"end_time"`
		IsAllDay   bool       `json:"is_all_day"`
		Timezone   string     `json:"timezone"`
		HasVideo   bool       `json:"has_video_call"`
		CreatedAt  time.Time  `json:"created_at"`
	}

	type attendanceEntry struct {
		EventID    uuid.UUID  `json:"event_id"`
		RsvpStatus string     `json:"rsvp_status"`
		RespondedAt *time.Time `json:"responded_at,omitempty"`
	}

	type prefsEntry struct {
		DefaultView              string `json:"default_view"`
		WeekDays                 int    `json:"week_days"`
		DefaultReminderMinutes   int    `json:"default_reminder_minutes"`
		ShowTaskDeadlines        bool   `json:"show_task_deadlines"`
	}

	type result struct {
		CreatedEvents  []eventEntry      `json:"created_events"`
		Attendances    []attendanceEntry `json:"event_attendances"`
		Preferences    *prefsEntry       `json:"preferences,omitempty"`
	}

	var out result

	// Events created by this user (tenant-scoped)
	evtRows, err := h.pool.Query(ctx,
		`SELECT ce.id, ce.title, ce.description, ce.location,
		        ce.start_time, ce.end_time, ce.is_all_day, ce.timezone, ce.has_video_call, ce.created_at
		 FROM calendar_events ce
		 JOIN users u ON u.id = $1
		 WHERE ce.created_by = $1 AND ce.tenant_id = u.tenant_id
		 ORDER BY ce.start_time DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("calendar export: failed to query events: %w", err)
	}
	defer evtRows.Close()
	for evtRows.Next() {
		var e eventEntry
		if scanErr := evtRows.Scan(
			&e.ID, &e.Title, &e.Description, &e.Location,
			&e.StartTime, &e.EndTime, &e.IsAllDay, &e.Timezone, &e.HasVideo, &e.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("calendar export: failed to scan event: %w", scanErr)
		}
		out.CreatedEvents = append(out.CreatedEvents, e)
	}
	if err := evtRows.Err(); err != nil {
		return nil, fmt.Errorf("calendar export: event rows error: %w", err)
	}

	// Event attendances for this user
	attRows, err := h.pool.Query(ctx,
		`SELECT ea.event_id, ea.rsvp_status, ea.responded_at
		 FROM event_attendees ea
		 JOIN calendar_events ce ON ce.id = ea.event_id
		 JOIN users u ON u.id = $1
		 WHERE ea.user_id = $1 AND ce.tenant_id = u.tenant_id
		 ORDER BY ea.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("calendar export: failed to query attendances: %w", err)
	}
	defer attRows.Close()
	for attRows.Next() {
		var a attendanceEntry
		if scanErr := attRows.Scan(&a.EventID, &a.RsvpStatus, &a.RespondedAt); scanErr != nil {
			return nil, fmt.Errorf("calendar export: failed to scan attendance: %w", scanErr)
		}
		out.Attendances = append(out.Attendances, a)
	}
	if err := attRows.Err(); err != nil {
		return nil, fmt.Errorf("calendar export: attendance rows error: %w", err)
	}

	// Calendar preferences
	var prefs prefsEntry
	scanErr := h.pool.QueryRow(ctx,
		`SELECT default_view, week_days, default_reminder_minutes, show_task_deadlines
		 FROM user_calendar_preferences WHERE user_id = $1`, userID,
	).Scan(&prefs.DefaultView, &prefs.WeekDays, &prefs.DefaultReminderMinutes, &prefs.ShowTaskDeadlines)
	if scanErr == nil {
		out.Preferences = &prefs
	}
	// No error on missing preferences (optional record)

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("calendar export: failed to marshal: %w", err)
	}
	return data, nil
}

// SessionExportHandler exports session history and device metadata.
// This handler is a lightweight complement to AuthExportHandler for session-specific data.
// Deduplicates: if AuthExportHandler is also registered, session data is primary there.
// Register only one of them. This handler is kept for standalone use cases.
type SessionExportHandler struct {
	pool *pgxpool.Pool
}

// NewSessionExportHandler creates a new SessionExportHandler with DB access.
func NewSessionExportHandler(pool *pgxpool.Pool) *SessionExportHandler {
	return &SessionExportHandler{pool: pool}
}

func (h *SessionExportHandler) ModuleName() string { return "sessions" }

func (h *SessionExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	type sessionEntry struct {
		ID           uuid.UUID  `json:"id"`
		DeviceName   string     `json:"device_name,omitempty"`
		DeviceType   string     `json:"device_type,omitempty"`
		IPAddress    *string    `json:"ip_address,omitempty"`
		Location     string     `json:"location,omitempty"`
		IsCurrent    bool       `json:"is_current"`
		LastActiveAt time.Time  `json:"last_active_at"`
		CreatedAt    time.Time  `json:"created_at"`
	}

	type result struct {
		Sessions []sessionEntry `json:"sessions"`
	}

	var out result

	rows, err := h.pool.Query(ctx,
		`SELECT id, COALESCE(device_name, ''), COALESCE(device_type, ''),
		        CASE WHEN ip_address IS NOT NULL THEN host(ip_address) ELSE NULL END,
		        COALESCE(location, ''), is_current, last_active_at, created_at
		 FROM user_sessions WHERE user_id = $1 ORDER BY last_active_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("sessions export: failed to query sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s sessionEntry
		if scanErr := rows.Scan(
			&s.ID, &s.DeviceName, &s.DeviceType, &s.IPAddress,
			&s.Location, &s.IsCurrent, &s.LastActiveAt, &s.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("sessions export: failed to scan session: %w", scanErr)
		}
		out.Sessions = append(out.Sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sessions export: rows error: %w", err)
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("sessions export: failed to marshal: %w", err)
	}
	return data, nil
}

// NotificationExportHandler exports notification preferences and delivery history.
// Tables: notification_preferences, notification_quiet_hours, notifications (recent 90 days).
type NotificationExportHandler struct {
	pool *pgxpool.Pool
}

// NewNotificationExportHandler creates a new NotificationExportHandler with DB access.
func NewNotificationExportHandler(pool *pgxpool.Pool) *NotificationExportHandler {
	return &NotificationExportHandler{pool: pool}
}

func (h *NotificationExportHandler) ModuleName() string { return "notifications" }

func (h *NotificationExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	type prefEntry struct {
		EventTypeKey *string `json:"event_type_key,omitempty"`
		ModuleID     *string `json:"module_id,omitempty"`
		InApp        bool    `json:"in_app"`
		DesktopPush  bool    `json:"desktop_push"`
		Sound        *string `json:"sound,omitempty"`
	}

	type quietHoursEntry struct {
		StartTime   string    `json:"start_time"`
		EndTime     string    `json:"end_time"`
		Timezone    string    `json:"timezone"`
		DaysOfWeek  []int     `json:"days_of_week"`
		Enabled     bool      `json:"enabled"`
		ManualDND   bool      `json:"manual_dnd"`
	}

	type notifEntry struct {
		ID           uuid.UUID  `json:"id"`
		EventTypeKey string     `json:"event_type_key"`
		ModuleID     string     `json:"module_id"`
		Priority     string     `json:"priority"`
		Title        string     `json:"title"`
		IsRead       bool       `json:"is_read"`
		ReadAt       *time.Time `json:"read_at,omitempty"`
		CreatedAt    time.Time  `json:"created_at"`
	}

	type result struct {
		Preferences []prefEntry       `json:"preferences"`
		QuietHours  *quietHoursEntry  `json:"quiet_hours,omitempty"`
		RecentNotifications []notifEntry `json:"recent_notifications_90d"`
	}

	var out result

	// Notification preferences
	prefRows, err := h.pool.Query(ctx,
		`SELECT event_type_key, module_id, in_app, desktop_push, sound
		 FROM notification_preferences WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("notifications export: failed to query preferences: %w", err)
	}
	defer prefRows.Close()
	for prefRows.Next() {
		var p prefEntry
		if scanErr := prefRows.Scan(&p.EventTypeKey, &p.ModuleID, &p.InApp, &p.DesktopPush, &p.Sound); scanErr != nil {
			return nil, fmt.Errorf("notifications export: failed to scan preference: %w", scanErr)
		}
		out.Preferences = append(out.Preferences, p)
	}
	if err := prefRows.Err(); err != nil {
		return nil, fmt.Errorf("notifications export: preferences rows error: %w", err)
	}

	// Quiet hours config
	var qh quietHoursEntry
	if scanErr := h.pool.QueryRow(ctx,
		`SELECT start_time::text, end_time::text, timezone, days_of_week, enabled, manual_dnd
		 FROM notification_quiet_hours WHERE user_id = $1`, userID,
	).Scan(&qh.StartTime, &qh.EndTime, &qh.Timezone, &qh.DaysOfWeek, &qh.Enabled, &qh.ManualDND); scanErr == nil {
		out.QuietHours = &qh
	}

	// Recent notifications (90-day window, no actor details to minimize third-party data)
	notifRows, err := h.pool.Query(ctx,
		`SELECT id, event_type_key, module_id, priority, title, is_read, read_at, created_at
		 FROM notifications
		 WHERE user_id = $1 AND created_at > NOW() - INTERVAL '90 days'
		 ORDER BY created_at DESC
		 LIMIT 1000`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("notifications export: failed to query notifications: %w", err)
	}
	defer notifRows.Close()
	for notifRows.Next() {
		var n notifEntry
		if scanErr := notifRows.Scan(
			&n.ID, &n.EventTypeKey, &n.ModuleID, &n.Priority, &n.Title, &n.IsRead, &n.ReadAt, &n.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("notifications export: failed to scan notification: %w", scanErr)
		}
		out.RecentNotifications = append(out.RecentNotifications, n)
	}
	if err := notifRows.Err(); err != nil {
		return nil, fmt.Errorf("notifications export: notification rows error: %w", err)
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("notifications export: failed to marshal: %w", err)
	}
	return data, nil
}
