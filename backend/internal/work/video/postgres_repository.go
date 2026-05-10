package video

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository for call sessions
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateCallSession(ctx context.Context, session *CallSession) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO call_sessions (id, tenant_id, call_type, status, room_name, initiator_id, channel_id, created_at, ended_at, duration_seconds)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		session.ID, session.TenantID, session.CallType, session.Status, session.RoomName,
		session.InitiatorID, session.ChannelID, session.CreatedAt,
		session.EndedAt, session.DurationSeconds,
	)
	return err
}

func (r *PostgresRepository) GetCallSession(ctx context.Context, id uuid.UUID) (*CallSession, error) {
	var s CallSession
	err := r.pool.QueryRow(ctx,
		`SELECT id, call_type, status, room_name, initiator_id, channel_id, created_at, ended_at, duration_seconds
		 FROM call_sessions WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.CallType, &s.Status, &s.RoomName, &s.InitiatorID,
		&s.ChannelID, &s.CreatedAt, &s.EndedAt, &s.DurationSeconds)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepository) GetCallSessionByRoomName(ctx context.Context, roomName string) (*CallSession, error) {
	var s CallSession
	err := r.pool.QueryRow(ctx,
		`SELECT id, call_type, status, room_name, initiator_id, channel_id, created_at, ended_at, duration_seconds
		 FROM call_sessions WHERE room_name = $1`,
		roomName,
	).Scan(&s.ID, &s.CallType, &s.Status, &s.RoomName, &s.InitiatorID,
		&s.ChannelID, &s.CreatedAt, &s.EndedAt, &s.DurationSeconds)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepository) UpdateCallSession(ctx context.Context, session *CallSession) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE call_sessions SET status = $1, ended_at = $2, duration_seconds = $3
		 WHERE id = $4`,
		session.Status, session.EndedAt, session.DurationSeconds, session.ID,
	)
	return err
}

func (r *PostgresRepository) ListActiveCallsForUser(ctx context.Context, userID uuid.UUID) ([]CallSession, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cs.id, cs.call_type, cs.status, cs.room_name, cs.initiator_id,
		        cs.channel_id, cs.created_at, cs.ended_at, cs.duration_seconds
		 FROM call_sessions cs
		 JOIN call_participants cp ON cs.id = cp.call_id
		 WHERE cp.user_id = $1 AND cs.status != 'ended' AND cp.left_at IS NULL
		 ORDER BY cs.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []CallSession
	for rows.Next() {
		var s CallSession
		if scanErr := rows.Scan(&s.ID, &s.CallType, &s.Status, &s.RoomName,
			&s.InitiatorID, &s.ChannelID, &s.CreatedAt, &s.EndedAt,
			&s.DurationSeconds); scanErr != nil {
			return nil, scanErr
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *PostgresRepository) AddParticipant(ctx context.Context, p *CallParticipant) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO call_participants (call_id, tenant_id, user_id, joined_at, left_at, has_video, has_audio)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (call_id, user_id) DO UPDATE SET left_at = NULL, joined_at = $4, has_video = $6, has_audio = $7`,
		p.CallID, p.TenantID, p.UserID, p.JoinedAt, p.LeftAt, p.HasVideo, p.HasAudio,
	)
	return err
}

func (r *PostgresRepository) RemoveParticipant(ctx context.Context, callID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE call_participants SET left_at = NOW()
		 WHERE call_id = $1 AND user_id = $2 AND left_at IS NULL`,
		callID, userID,
	)
	return err
}

func (r *PostgresRepository) GetParticipants(ctx context.Context, callID uuid.UUID) ([]CallParticipant, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT call_id, user_id, joined_at, left_at, has_video, has_audio
		 FROM call_participants
		 WHERE call_id = $1
		 ORDER BY joined_at ASC`,
		callID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []CallParticipant
	for rows.Next() {
		var p CallParticipant
		if scanErr := rows.Scan(&p.CallID, &p.UserID, &p.JoinedAt, &p.LeftAt,
			&p.HasVideo, &p.HasAudio); scanErr != nil {
			return nil, scanErr
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func (r *PostgresRepository) CountActiveParticipants(ctx context.Context, callID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM call_participants WHERE call_id = $1 AND left_at IS NULL`,
		callID,
	).Scan(&count)
	return count, err
}
