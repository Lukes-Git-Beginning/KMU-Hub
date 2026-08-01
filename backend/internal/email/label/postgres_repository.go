package label

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

const labelColumns = `id, tenant_id, name, color, created_at, updated_at`

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL email label repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func scanLabel(row pgx.Row) (*models.EmailLabel, error) {
	var l models.EmailLabel
	if err := row.Scan(&l.ID, &l.TenantID, &l.Name, &l.Color, &l.CreatedAt, &l.UpdatedAt); err != nil {
		return nil, err
	}
	return &l, nil
}

func (p *PostgresRepository) Create(ctx context.Context, l *models.EmailLabel) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO email_labels (`+labelColumns+`) VALUES ($1, $2, $3, $4, $5, $6)`,
		l.ID, l.TenantID, l.Name, l.Color, l.CreatedAt, l.UpdatedAt,
	)
	if isDuplicateError(err) {
		return ErrDuplicateName
	}
	return err
}

func (p *PostgresRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailLabel, error) {
	l, err := scanLabel(p.pool.QueryRow(ctx,
		`SELECT `+labelColumns+` FROM email_labels WHERE id = $1 AND tenant_id = $2`, id, tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return l, nil
}

func (p *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailLabel, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT `+labelColumns+` FROM email_labels WHERE tenant_id = $1 ORDER BY name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labels := make([]*models.EmailLabel, 0)
	for rows.Next() {
		l, err := scanLabel(rows)
		if err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

func (p *PostgresRepository) Update(ctx context.Context, l *models.EmailLabel) error {
	l.UpdatedAt = time.Now().UTC()
	tag, err := p.pool.Exec(ctx,
		`UPDATE email_labels SET name = $3, color = $4, updated_at = $5 WHERE id = $1 AND tenant_id = $2`,
		l.ID, l.TenantID, l.Name, l.Color, l.UpdatedAt,
	)
	if err != nil {
		if isDuplicateError(err) {
			return ErrDuplicateName
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes the label and, in the same transaction, strips its id from
// every message of the tenant that still carries it -- label_ids has no
// foreign key to enforce that on its own.
func (p *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `DELETE FROM email_labels WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx,
		`UPDATE email_messages SET label_ids = array_remove(label_ids, $1), updated_at = NOW()
		  WHERE tenant_id = $2 AND $1 = ANY(label_ids)`,
		id, tenantID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (p *PostgresRepository) LabelIDsBelongToTenant(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	var count int
	err := p.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_labels WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, ids,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	// ids may repeat; count distinct ids covered instead of len(ids).
	return count == len(uniqueUUIDs(ids)), nil
}

func (p *PostgresRepository) AssignToMessage(ctx context.Context, tenantID, messageID uuid.UUID, labelIDs []uuid.UUID) error {
	if labelIDs == nil {
		labelIDs = []uuid.UUID{}
	}
	tag, err := p.pool.Exec(ctx,
		`UPDATE email_messages SET label_ids = $3, updated_at = NOW() WHERE id = $1 AND tenant_id = $2`,
		messageID, tenantID, labelIDs,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// isDuplicateError checks whether a pgx error is a unique-constraint violation.
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps pg error codes; the unique_violation code is "23505"
	type pgErr interface{ SQLState() string }
	var pe pgErr
	return errors.As(err, &pe) && pe.SQLState() == "23505"
}
