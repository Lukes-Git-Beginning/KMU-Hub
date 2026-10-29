package recording

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository for recordings
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateRecording(ctx context.Context, rec *Recording) error {
	snapshotJSON, err := marshalConsentSnapshot(rec.ConsentSnapshot)
	if err != nil {
		return fmt.Errorf("marshal consent snapshot: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO recordings (id, tenant_id, call_id, meeting_id, started_by, consent_snapshot, status,
		  egress_id, file_url, file_size_bytes, duration_seconds, retention_expires_at, created_at, room_name)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NULLIF($14, ''))`,
		rec.ID, rec.TenantID, rec.CallID, rec.MeetingID, rec.StartedBy, snapshotJSON, rec.Status,
		rec.EgressID, rec.FileURL, rec.FileSizeBytes, rec.DurationSeconds,
		rec.RetentionExpiresAt, rec.CreatedAt, rec.RoomName,
	)
	return err
}

func (r *PostgresRepository) GetRecording(ctx context.Context, id uuid.UUID) (*Recording, error) {
	var rec Recording
	var snapshotJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, call_id, meeting_id, started_by, consent_snapshot, status,
		        egress_id, file_url, file_size_bytes, duration_seconds, retention_expires_at, created_at
		 FROM recordings WHERE id = $1`,
		id,
	).Scan(&rec.ID, &rec.TenantID, &rec.CallID, &rec.MeetingID, &rec.StartedBy, &snapshotJSON, &rec.Status,
		&rec.EgressID, &rec.FileURL, &rec.FileSizeBytes, &rec.DurationSeconds,
		&rec.RetentionExpiresAt, &rec.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(snapshotJSON) > 0 {
		if err := json.Unmarshal(snapshotJSON, &rec.ConsentSnapshot); err != nil {
			return nil, fmt.Errorf("unmarshal consent snapshot: %w", err)
		}
	}
	return &rec, nil
}

func (r *PostgresRepository) UpdateRecording(ctx context.Context, rec *Recording) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE recordings SET status = $1, egress_id = $2, file_url = $3, file_size_bytes = $4,
		  duration_seconds = $5, retention_expires_at = $6
		 WHERE id = $7`,
		rec.Status, rec.EgressID, rec.FileURL, rec.FileSizeBytes,
		rec.DurationSeconds, rec.RetentionExpiresAt, rec.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ListRecordingsByCall(ctx context.Context, callID uuid.UUID) ([]Recording, error) {
	return r.listRecordings(ctx,
		`SELECT id, tenant_id, call_id, meeting_id, started_by, consent_snapshot, status, egress_id, file_url,
		        file_size_bytes, duration_seconds, retention_expires_at, created_at
		 FROM recordings WHERE call_id = $1 ORDER BY created_at DESC`,
		callID,
	)
}

func (r *PostgresRepository) ListRecordingsByMeeting(ctx context.Context, meetingID uuid.UUID) ([]Recording, error) {
	return r.listRecordings(ctx,
		`SELECT id, tenant_id, call_id, meeting_id, started_by, consent_snapshot, status, egress_id, file_url,
		        file_size_bytes, duration_seconds, retention_expires_at, created_at
		 FROM recordings WHERE meeting_id = $1 ORDER BY created_at DESC`,
		meetingID,
	)
}

func (r *PostgresRepository) GetRecordingByEgressID(ctx context.Context, egressID string) (*Recording, error) {
	var rec Recording
	var snapshotJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, call_id, meeting_id, started_by, consent_snapshot, status,
		        egress_id, file_url, file_size_bytes, duration_seconds, retention_expires_at, created_at
		 FROM recordings WHERE egress_id = $1`,
		egressID,
	).Scan(&rec.ID, &rec.TenantID, &rec.CallID, &rec.MeetingID, &rec.StartedBy, &snapshotJSON, &rec.Status,
		&rec.EgressID, &rec.FileURL, &rec.FileSizeBytes, &rec.DurationSeconds,
		&rec.RetentionExpiresAt, &rec.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(snapshotJSON) > 0 {
		if err := json.Unmarshal(snapshotJSON, &rec.ConsentSnapshot); err != nil {
			return nil, fmt.Errorf("unmarshal consent snapshot: %w", err)
		}
	}
	return &rec, nil
}

func (r *PostgresRepository) TagRecordingWithConsents(ctx context.Context, recordingID uuid.UUID, snapshot []ParticipantConsentInfo) error {
	snapshotJSON, err := marshalConsentSnapshot(snapshot)
	if err != nil {
		return fmt.Errorf("marshal consent snapshot: %w", err)
	}
	// Ensure we never write a NULL — use empty JSON array as sentinel
	if snapshotJSON == nil {
		snapshotJSON = []byte("[]")
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE recordings SET consent_snapshot = $1 WHERE id = $2`,
		snapshotJSON, recordingID,
	)
	return err
}

func (r *PostgresRepository) DeleteRecording(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM recordings WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetConsent inserts or updates a participant's consent response. tenant_id is
// not a caller-supplied parameter: it is derived from the parent recording via
// subquery, so a cross-tenant call (RecordingID belonging to another tenant)
// resolves to NULL under that tenant's RLS view of `recordings` and fails the
// NOT NULL constraint instead of silently writing under the wrong tenant.
func (r *PostgresRepository) SetConsent(ctx context.Context, consent *RecordingConsent) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO recording_consents (recording_id, user_id, consented, responded_at, tenant_id)
		 VALUES ($1, $2, $3, $4, (SELECT tenant_id FROM recordings WHERE id = $1))
		 ON CONFLICT (recording_id, user_id) DO UPDATE SET consented = $3, responded_at = $4`,
		consent.RecordingID, consent.UserID, consent.Consented, consent.RespondedAt,
	)
	return err
}

func (r *PostgresRepository) GetConsents(ctx context.Context, recordingID uuid.UUID) ([]RecordingConsent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT recording_id, user_id, consented, responded_at
		 FROM recording_consents WHERE recording_id = $1
		 ORDER BY responded_at ASC`,
		recordingID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []RecordingConsent
	for rows.Next() {
		var c RecordingConsent
		if scanErr := rows.Scan(&c.RecordingID, &c.UserID, &c.Consented, &c.RespondedAt); scanErr != nil {
			return nil, scanErr
		}
		consents = append(consents, c)
	}
	return consents, rows.Err()
}

func (r *PostgresRepository) GetConsentsWithUser(ctx context.Context, recordingID uuid.UUID) ([]RecordingConsentWithUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT rc.recording_id, rc.user_id, rc.consented, rc.responded_at,
		        COALESCE(u.first_name, ''), COALESCE(u.last_name, '')
		 FROM recording_consents rc
		 LEFT JOIN users u ON rc.user_id = u.id
		 WHERE rc.recording_id = $1
		 ORDER BY rc.responded_at ASC`,
		recordingID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []RecordingConsentWithUser
	for rows.Next() {
		var c RecordingConsentWithUser
		if scanErr := rows.Scan(
			&c.RecordingID, &c.UserID, &c.Consented, &c.RespondedAt,
			&c.FirstName, &c.LastName,
		); scanErr != nil {
			return nil, scanErr
		}
		consents = append(consents, c)
	}
	return consents, rows.Err()
}

func (r *PostgresRepository) CountPendingConsents(ctx context.Context, recordingID uuid.UUID, participantIDs []uuid.UUID) (int, error) {
	// Count participants who have NOT responded to consent for this recording
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM unnest($2::uuid[]) AS pid(id)
		 WHERE NOT EXISTS (
		   SELECT 1 FROM recording_consents
		   WHERE recording_id = $1 AND user_id = pid.id
		 )`,
		recordingID, participantIDs,
	).Scan(&count)
	return count, err
}

func (r *PostgresRepository) ListExpiredRecordings(ctx context.Context, before time.Time) ([]Recording, error) {
	return r.listRecordings(ctx,
		`SELECT id, tenant_id, call_id, meeting_id, started_by, consent_snapshot, status, egress_id, file_url,
		        file_size_bytes, duration_seconds, retention_expires_at, created_at
		 FROM recordings
		 WHERE retention_expires_at < $1 AND status = 'completed'
		 ORDER BY retention_expires_at ASC`,
		before,
	)
}

func (r *PostgresRepository) ListRecordingsWithAccess(ctx context.Context, userID uuid.UUID, callID, meetingID *uuid.UUID) ([]Recording, error) {
	// Returns recordings where the user was a participant (via call_participants or meeting_attendees),
	// optionally filtered by call ID and/or meeting ID.
	query := `
		SELECT DISTINCT r.id, r.tenant_id, r.call_id, r.meeting_id, r.started_by, r.consent_snapshot,
		       r.status, r.egress_id, r.file_url,
		       r.file_size_bytes, r.duration_seconds, r.retention_expires_at, r.created_at
		FROM recordings r
		LEFT JOIN call_participants cp ON r.call_id = cp.call_id AND cp.user_id = $1
		LEFT JOIN meeting_attendees ma ON r.meeting_id = ma.meeting_id AND ma.user_id = $1
		WHERE (cp.user_id IS NOT NULL OR ma.user_id IS NOT NULL)
		  AND r.status IN ('completed', 'processing')`

	args := []any{userID}

	if callID != nil {
		query += fmt.Sprintf(" AND r.call_id = $%d", len(args)+1)
		args = append(args, *callID)
	}
	if meetingID != nil {
		query += fmt.Sprintf(" AND r.meeting_id = $%d", len(args)+1)
		args = append(args, *meetingID)
	}

	query += " ORDER BY r.created_at DESC"

	return r.listRecordings(ctx, query, args...)
}

func (r *PostgresRepository) GetRecordingParticipants(ctx context.Context, recordingID uuid.UUID) ([]uuid.UUID, error) {
	// Returns user IDs who participated in the call or meeting associated with this recording
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT u_id FROM (
		   SELECT cp.user_id AS u_id
		   FROM recordings r
		   JOIN call_participants cp ON r.call_id = cp.call_id
		   WHERE r.id = $1
		   UNION
		   SELECT ma.user_id AS u_id
		   FROM recordings r
		   JOIN meeting_attendees ma ON r.meeting_id = ma.meeting_id
		   WHERE r.id = $1
		 ) sub`,
		recordingID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if scanErr := rows.Scan(&uid); scanErr != nil {
			return nil, scanErr
		}
		userIDs = append(userIDs, uid)
	}
	return userIDs, rows.Err()
}

// listRecordings is a helper to scan recording rows.
// All queries passed here must SELECT: id, tenant_id, call_id, meeting_id, started_by, consent_snapshot,
// status, egress_id, file_url, file_size_bytes, duration_seconds, retention_expires_at, created_at
func (r *PostgresRepository) listRecordings(ctx context.Context, query string, args ...any) ([]Recording, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recordings []Recording
	for rows.Next() {
		var rec Recording
		var snapshotJSON []byte
		if scanErr := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.CallID, &rec.MeetingID, &rec.StartedBy, &snapshotJSON, &rec.Status,
			&rec.EgressID, &rec.FileURL, &rec.FileSizeBytes, &rec.DurationSeconds,
			&rec.RetentionExpiresAt, &rec.CreatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		if len(snapshotJSON) > 0 {
			if unmarshalErr := json.Unmarshal(snapshotJSON, &rec.ConsentSnapshot); unmarshalErr != nil {
				return nil, fmt.Errorf("unmarshal consent snapshot: %w", unmarshalErr)
			}
		}
		recordings = append(recordings, rec)
	}
	return recordings, rows.Err()
}

// MarkInitiatorConsent stamps the initiator's confirmation and moves the
// recording from pending to active. Only the call that performs that move gets
// claimed=true together with the room name captured at start — the caller starts
// the egress exactly then. A repeated (or concurrent) stamp on the same row finds
// no pending row and returns claimed=false: the row lock serialises the UPDATEs,
// the loser re-checks status = 'pending' and matches nothing.
// The move is gated in the same statement on participant consent: every user in
// consent_snapshot other than the initiator needs a recording_consents row with
// consented = true for this recording. If one is missing or a no, the row stays
// pending and ErrConsentPending is returned — no window between check and claim.
// The sentinel UUID 00000000-0000-0000-0000-000000000001 is used as initiator_consent_id.
// tenant_id and started_by are enforced in the WHERE clause: a recording of
// another user or tenant returns ErrNotFound.
func (r *PostgresRepository) MarkInitiatorConsent(ctx context.Context, recordingID, userID, tenantID uuid.UUID) (string, bool, error) {
	sentinelConsentID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	var roomName string
	err := r.pool.QueryRow(ctx,
		`UPDATE recordings
		 SET status = 'active',
		     pre_recording_consent_at = NOW(),
		     initiator_consent_id = $1
		 WHERE id = $2
		   AND started_by = $3
		   AND tenant_id = $4
		   AND status = 'pending'
		   AND NOT EXISTS (
		     SELECT 1
		     FROM jsonb_array_elements(COALESCE(recordings.consent_snapshot, '[]'::jsonb)) AS p(entry)
		     WHERE (p.entry->>'user_id')::uuid <> recordings.started_by
		       AND NOT EXISTS (
		         SELECT 1 FROM recording_consents rc
		         WHERE rc.recording_id = recordings.id
		           AND rc.user_id = (p.entry->>'user_id')::uuid
		           AND rc.consented
		       )
		   )
		 RETURNING COALESCE(room_name, '')`,
		sentinelConsentID, recordingID, userID, tenantID,
	).Scan(&roomName)
	if err == nil {
		return roomName, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, err
	}

	var status string
	err = r.pool.QueryRow(ctx,
		`SELECT status FROM recordings WHERE id = $1 AND started_by = $2 AND tenant_id = $3`,
		recordingID, userID, tenantID,
	).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, ErrNotFound
	}
	if err != nil {
		return "", false, err
	}
	if status == RecordingStatusPending {
		return "", false, ErrConsentPending
	}
	return "", false, nil
}

// GetPreConsentStatus returns true if pre_recording_consent_at is set on the recording.
// tenantID is enforced in the WHERE clause.
func (r *PostgresRepository) GetPreConsentStatus(ctx context.Context, recordingID, tenantID uuid.UUID) (bool, error) {
	var stamped bool
	err := r.pool.QueryRow(ctx,
		`SELECT pre_recording_consent_at IS NOT NULL
		 FROM recordings
		 WHERE id = $1 AND tenant_id = $2`,
		recordingID, tenantID,
	).Scan(&stamped)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	return stamped, err
}

// marshalConsentSnapshot serializes the participant snapshot to JSONB.
// Returns nil for an empty slice so the column stays NULL for old recordings.
func marshalConsentSnapshot(snapshot []ParticipantConsentInfo) ([]byte, error) {
	if len(snapshot) == 0 {
		return nil, nil
	}
	return json.Marshal(snapshot)
}
