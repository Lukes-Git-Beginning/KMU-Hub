package gdpr

// MeetingNoteRetentionHandler and MeetingChatRetentionHandler implement
// RetentionHandler for resource_type "meeting_notes" and "meeting_chat" —
// the second half of the Lauf 17 decision "DSAR + Frist 2 Jahre" (2026-10-25);
// the DSAR half is meetingNotesModule / meetingChatModule in
// dsar_search_video_dialer.go. The two-year period itself is the
// retention_days an admin sets on the policy row, not a constant here.
//
// Clock is the end of the meeting the row hangs off, COALESCE(actual_end,
// scheduled_end): a note written at the start of a long series of follow-up
// sessions would otherwise fall due while its meeting is still running, and
// created_at of a note says nothing about when the discussion was over. A
// meeting that never started still has a scheduled_end, so a cancelled one
// ages out like any other.
//
// Delete-only. Notes carry free text about the meeting and chat lines carry
// sender_name plus free text; blanking either leaves a row that retains
// nothing worth keeping, the same reasoning as guest_sessions. The meetings
// row itself (title, times, organizer) is not touched — it is the booking
// record, not the discussion. Private notes are not exempt: they are the
// author's own data, but the retention period applies to the content, not to
// who may read it.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// meetingRetentionTable is the meeting-scoped table a handler governs. The
// value is interpolated into SQL, so it only ever comes from the two
// constants below, never from input.
type meetingRetentionTable string

const (
	meetingNotesTable meetingRetentionTable = "meeting_notes"
	meetingChatTable  meetingRetentionTable = "meeting_chat"
)

// planMeetingRows lists the rows of table whose meeting ended before cutoff.
func planMeetingRows(ctx context.Context, pool *pgxpool.Pool, table meetingRetentionTable, tenantID uuid.UUID, cutoff time.Time) (*RetentionPlan, error) {
	rows, err := pool.Query(ctx,
		`SELECT t.id FROM `+string(table)+` t
		   JOIN meetings m ON m.id = t.meeting_id AND m.tenant_id = t.tenant_id
		  WHERE t.tenant_id = $1 AND COALESCE(m.actual_end, m.scheduled_end) < $2
		  ORDER BY COALESCE(m.actual_end, m.scheduled_end) ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("%s retention: list candidates: %w", table, err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("%s retention: scan candidate: %w", table, scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s retention: iterate candidates: %w", table, err)
	}
	return &RetentionPlan{Due: ids, Skipped: make([]RetentionSkip, 0)}, nil
}

// applyMeetingRowsDelete hard-deletes the given rows of table. Idempotent: a
// second run over the same ids finds nothing left.
func applyMeetingRowsDelete(ctx context.Context, pool *pgxpool.Pool, table meetingRetentionTable, tenantID uuid.UUID, ids []uuid.UUID, action string) (int, error) {
	if action != models.RetentionActionDelete {
		return 0, fmt.Errorf("%s retention: unsupported action %q", table, action)
	}
	tag, err := pool.Exec(ctx,
		`DELETE FROM `+string(table)+` WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("%s retention: delete: %w", table, err)
	}
	return int(tag.RowsAffected()), nil
}

// MeetingNoteRetentionHandler is the meeting_notes entry in the retention
// registry.
type MeetingNoteRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewMeetingNoteRetentionHandler wires the handler.
func NewMeetingNoteRetentionHandler(pool *pgxpool.Pool) *MeetingNoteRetentionHandler {
	return &MeetingNoteRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for meeting notes.
func (h *MeetingNoteRetentionHandler) ResourceType() string { return "meeting_notes" }

// Table names the governed table for the admin view.
func (h *MeetingNoteRetentionHandler) Table() string { return "meeting_notes" }

// DateColumn discloses the clock — see the file header.
func (h *MeetingNoteRetentionHandler) DateColumn() string {
	return "meetings.actual_end (sonst scheduled_end) des zugehoerigen Meetings"
}

// SupportsAction reports that meeting notes can only be hard-deleted.
func (h *MeetingNoteRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists notes whose meeting ended before cutoff. Never modifies data.
func (h *MeetingNoteRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	return planMeetingRows(ctx, h.pool, meetingNotesTable, tenantID, cutoff)
}

// Apply deletes the given notes.
func (h *MeetingNoteRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	return applyMeetingRowsDelete(ctx, h.pool, meetingNotesTable, tenantID, ids, action)
}

// MeetingChatRetentionHandler is the meeting_chat entry in the retention
// registry.
type MeetingChatRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewMeetingChatRetentionHandler wires the handler.
func NewMeetingChatRetentionHandler(pool *pgxpool.Pool) *MeetingChatRetentionHandler {
	return &MeetingChatRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for in-call chat.
func (h *MeetingChatRetentionHandler) ResourceType() string { return "meeting_chat" }

// Table names the governed table for the admin view.
func (h *MeetingChatRetentionHandler) Table() string { return "meeting_chat" }

// DateColumn discloses the clock — see the file header.
func (h *MeetingChatRetentionHandler) DateColumn() string {
	return "meetings.actual_end (sonst scheduled_end) des zugehoerigen Meetings"
}

// SupportsAction reports that meeting chat can only be hard-deleted.
func (h *MeetingChatRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists chat lines whose meeting ended before cutoff. Never modifies data.
func (h *MeetingChatRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	return planMeetingRows(ctx, h.pool, meetingChatTable, tenantID, cutoff)
}

// Apply deletes the given chat lines.
func (h *MeetingChatRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	return applyMeetingRowsDelete(ctx, h.pool, meetingChatTable, tenantID, ids, action)
}

var (
	_ RetentionHandler = (*MeetingNoteRetentionHandler)(nil)
	_ RetentionHandler = (*MeetingChatRetentionHandler)(nil)
)
