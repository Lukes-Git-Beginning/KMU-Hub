package reaction

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL reaction repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO message_reactions (message_id, tenant_id, user_id, emoji, created_at)
		 SELECT $1, m.tenant_id, $2, $3, $4
		 FROM messages m
		 WHERE m.id = $1
		 ON CONFLICT DO NOTHING`,
		messageID, userID, emoji, time.Now(),
	)
	return err
}

func (r *PostgresRepository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`,
		messageID, userID, emoji,
	)
	return err
}

func (r *PostgresRepository) ReactionExists(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3)`,
		messageID, userID, emoji,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) ListReactions(ctx context.Context, messageID uuid.UUID) ([]Reaction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT message_id, user_id, emoji, created_at
		 FROM message_reactions
		 WHERE message_id = $1
		 ORDER BY created_at ASC`,
		messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []Reaction
	for rows.Next() {
		var rx Reaction
		if scanErr := rows.Scan(&rx.MessageID, &rx.UserID, &rx.Emoji, &rx.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		reactions = append(reactions, rx)
	}
	return reactions, rows.Err()
}

func (r *PostgresRepository) GetReactionSummary(ctx context.Context, messageID, currentUserID uuid.UUID) ([]ReactionSummary, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT emoji,
		        COUNT(*) AS count,
		        array_agg(user_id ORDER BY created_at) AS user_ids,
		        bool_or(user_id = $2) AS current_user_reacted
		 FROM message_reactions
		 WHERE message_id = $1
		 GROUP BY emoji
		 ORDER BY MIN(created_at)`,
		messageID, currentUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ReactionSummary
	for rows.Next() {
		s := ReactionSummary{MessageID: messageID}
		if scanErr := rows.Scan(&s.Emoji, &s.Count, &s.UserIDs, &s.CurrentUserReacted); scanErr != nil {
			return nil, scanErr
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

func (r *PostgresRepository) GetBatchReactionSummary(ctx context.Context, messageIDs []uuid.UUID, currentUserID uuid.UUID) (map[uuid.UUID][]ReactionSummary, error) {
	if len(messageIDs) == 0 {
		return make(map[uuid.UUID][]ReactionSummary), nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT message_id,
		        emoji,
		        COUNT(*) AS count,
		        array_agg(user_id ORDER BY created_at) AS user_ids,
		        bool_or(user_id = $2) AS current_user_reacted
		 FROM message_reactions
		 WHERE message_id = ANY($1)
		 GROUP BY message_id, emoji
		 ORDER BY message_id, MIN(created_at)`,
		messageIDs, currentUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]ReactionSummary)
	for rows.Next() {
		var s ReactionSummary
		if scanErr := rows.Scan(&s.MessageID, &s.Emoji, &s.Count, &s.UserIDs, &s.CurrentUserReacted); scanErr != nil {
			return nil, scanErr
		}
		result[s.MessageID] = append(result[s.MessageID], s)
	}
	return result, rows.Err()
}
