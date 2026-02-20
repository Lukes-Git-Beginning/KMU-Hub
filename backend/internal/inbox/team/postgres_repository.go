package team

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL team inbox repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateTeamInbox(ctx context.Context, inbox *models.TeamInbox) error {
	query := `
		INSERT INTO team_inboxes (id, name, description, assignment_mode, visibility,
			next_assignee_index, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`

	_, err := r.pool.Exec(ctx, query,
		inbox.ID, inbox.Name, inbox.Description, inbox.AssignmentMode, inbox.Visibility,
		inbox.NextAssigneeIndex, inbox.CreatedBy,
	)
	return err
}

func (r *PostgresRepository) UpdateTeamInbox(ctx context.Context, inbox *models.TeamInbox) error {
	query := `
		UPDATE team_inboxes SET
			name = $2, description = $3, assignment_mode = $4, visibility = $5,
			updated_at = NOW()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		inbox.ID, inbox.Name, inbox.Description, inbox.AssignmentMode, inbox.Visibility,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTeamInboxNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteTeamInbox(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM team_inboxes WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTeamInboxNotFound
	}
	return nil
}

func (r *PostgresRepository) GetTeamInbox(ctx context.Context, id uuid.UUID) (*models.TeamInbox, error) {
	query := `
		SELECT id, name, description, assignment_mode, visibility,
			next_assignee_index, created_by, created_at, updated_at
		FROM team_inboxes WHERE id = $1`

	inbox := &models.TeamInbox{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&inbox.ID, &inbox.Name, &inbox.Description, &inbox.AssignmentMode, &inbox.Visibility,
		&inbox.NextAssigneeIndex, &inbox.CreatedBy, &inbox.CreatedAt, &inbox.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrTeamInboxNotFound
	}
	return inbox, err
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.TeamInbox, error) {
	query := `
		SELECT t.id, t.name, t.description, t.assignment_mode, t.visibility,
			t.next_assignee_index, t.created_by, t.created_at, t.updated_at
		FROM team_inboxes t
		INNER JOIN team_inbox_members m ON t.id = m.team_inbox_id
		WHERE m.user_id = $1
		ORDER BY t.name`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inboxes []*models.TeamInbox
	for rows.Next() {
		inbox := &models.TeamInbox{}
		if err := rows.Scan(
			&inbox.ID, &inbox.Name, &inbox.Description, &inbox.AssignmentMode, &inbox.Visibility,
			&inbox.NextAssigneeIndex, &inbox.CreatedBy, &inbox.CreatedAt, &inbox.UpdatedAt,
		); err != nil {
			return nil, err
		}
		inboxes = append(inboxes, inbox)
	}

	return inboxes, rows.Err()
}

func (r *PostgresRepository) AddMember(ctx context.Context, member *models.TeamInboxMember) error {
	query := `
		INSERT INTO team_inbox_members (team_inbox_id, user_id, role, created_at)
		VALUES ($1, $2, $3, NOW())`

	_, err := r.pool.Exec(ctx, query, member.TeamInboxID, member.UserID, member.Role)
	if err != nil {
		// Check for unique violation (already a member)
		if isDuplicateKeyError(err) {
			return ErrAlreadyMember
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, teamInboxID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM team_inbox_members WHERE team_inbox_id = $1 AND user_id = $2",
		teamInboxID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotTeamMember
	}
	return nil
}

func (r *PostgresRepository) ListMembers(ctx context.Context, teamInboxID uuid.UUID) ([]*models.TeamInboxMember, error) {
	query := `
		SELECT team_inbox_id, user_id, role, created_at
		FROM team_inbox_members
		WHERE team_inbox_id = $1
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, teamInboxID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*models.TeamInboxMember
	for rows.Next() {
		m := &models.TeamInboxMember{}
		if err := rows.Scan(&m.TeamInboxID, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, rows.Err()
}

func (r *PostgresRepository) IsMember(ctx context.Context, teamInboxID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM team_inbox_members WHERE team_inbox_id = $1 AND user_id = $2)",
		teamInboxID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) GetMemberRole(ctx context.Context, teamInboxID, userID uuid.UUID) (string, error) {
	var role string
	err := r.pool.QueryRow(ctx,
		"SELECT role FROM team_inbox_members WHERE team_inbox_id = $1 AND user_id = $2",
		teamInboxID, userID,
	).Scan(&role)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return role, err
}

func (r *PostgresRepository) IncrementAssigneeIndex(ctx context.Context, teamInboxID uuid.UUID) (int, error) {
	var newIndex int
	err := r.pool.QueryRow(ctx,
		"UPDATE team_inboxes SET next_assignee_index = next_assignee_index + 1, updated_at = NOW() WHERE id = $1 RETURNING next_assignee_index",
		teamInboxID,
	).Scan(&newIndex)
	if err == pgx.ErrNoRows {
		return 0, ErrTeamInboxNotFound
	}
	return newIndex, err
}

func (r *PostgresRepository) GetMemberCount(ctx context.Context, teamInboxID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM team_inbox_members WHERE team_inbox_id = $1",
		teamInboxID,
	).Scan(&count)
	return count, err
}

func (r *PostgresRepository) CountAdmins(ctx context.Context, teamInboxID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM team_inbox_members WHERE team_inbox_id = $1 AND role = 'admin'",
		teamInboxID,
	).Scan(&count)
	return count, err
}

// isDuplicateKeyError checks if a pgx error is a unique constraint violation (SQLSTATE 23505).
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps the error; check error string for SQLSTATE 23505
	return contains(err.Error(), "23505") || contains(err.Error(), "duplicate key")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
