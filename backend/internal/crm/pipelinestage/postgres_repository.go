package pipelinestage

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, stage *models.PipelineStage) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO pipeline_stages (id, name, color, sort_order, is_won, is_lost, probability, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		stage.ID, stage.Name, stage.Color, stage.SortOrder, stage.IsWon, stage.IsLost,
		stage.Probability, stage.CreatedAt, stage.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PipelineStage, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, color, sort_order, is_won, is_lost, probability, created_at, updated_at
		 FROM pipeline_stages WHERE id = $1`, id,
	)
	return r.scanStage(row)
}

func (r *PostgresRepository) List(ctx context.Context) ([]*models.PipelineStage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, color, sort_order, is_won, is_lost, probability, created_at, updated_at
		 FROM pipeline_stages ORDER BY sort_order ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []*models.PipelineStage
	for rows.Next() {
		stage, scanErr := r.scanStageFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		stages = append(stages, stage)
	}
	return stages, rows.Err()
}

func (r *PostgresRepository) ListWithStats(ctx context.Context) ([]*models.PipelineStageWithStats, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ps.id, ps.name, ps.color, ps.sort_order, ps.is_won, ps.is_lost, ps.probability,
		        ps.created_at, ps.updated_at,
		        COALESCE(COUNT(d.id), 0) AS deal_count,
		        COALESCE(SUM(d.value), 0) AS total_value
		 FROM pipeline_stages ps
		 LEFT JOIN deals d ON ps.id = d.stage_id
		 GROUP BY ps.id
		 ORDER BY ps.sort_order ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []*models.PipelineStageWithStats
	for rows.Next() {
		var s models.PipelineStageWithStats
		scanErr := rows.Scan(
			&s.ID, &s.Name, &s.Color, &s.SortOrder, &s.IsWon, &s.IsLost, &s.Probability,
			&s.CreatedAt, &s.UpdatedAt, &s.DealCount, &s.TotalValue,
		)
		if scanErr != nil {
			return nil, scanErr
		}
		stages = append(stages, &s)
	}
	return stages, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, stage *models.PipelineStage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE pipeline_stages SET name = $1, color = $2, sort_order = $3, is_won = $4, is_lost = $5,
		 probability = $6, updated_at = $7
		 WHERE id = $8`,
		stage.Name, stage.Color, stage.SortOrder, stage.IsWon, stage.IsLost,
		stage.Probability, stage.UpdatedAt, stage.ID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pipeline_stages WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) Reorder(ctx context.Context, stageIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for i, id := range stageIDs {
		_, execErr := tx.Exec(ctx,
			`UPDATE pipeline_stages SET sort_order = $1, updated_at = NOW() WHERE id = $2`,
			i+1, id,
		)
		if execErr != nil {
			return execErr
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) HasDeals(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM deals WHERE stage_id = $1)`, id,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) HasWonStage(ctx context.Context, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	var err error
	if excludeID != nil {
		err = r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pipeline_stages WHERE is_won = TRUE AND id != $1)`, *excludeID,
		).Scan(&exists)
	} else {
		err = r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pipeline_stages WHERE is_won = TRUE)`,
		).Scan(&exists)
	}
	return exists, err
}

func (r *PostgresRepository) HasLostStage(ctx context.Context, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	var err error
	if excludeID != nil {
		err = r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pipeline_stages WHERE is_lost = TRUE AND id != $1)`, *excludeID,
		).Scan(&exists)
	} else {
		err = r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pipeline_stages WHERE is_lost = TRUE)`,
		).Scan(&exists)
	}
	return exists, err
}

func (r *PostgresRepository) GetNextSortOrder(ctx context.Context) (int, error) {
	var maxOrder int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(sort_order), 0) FROM pipeline_stages`,
	).Scan(&maxOrder)
	if err != nil {
		return 0, err
	}
	return maxOrder + 1, nil
}

func (r *PostgresRepository) CountStages(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pipeline_stages`,
	).Scan(&count)
	return count, err
}

// Helper to scan a single row into PipelineStage
func (r *PostgresRepository) scanStage(row pgx.Row) (*models.PipelineStage, error) {
	var s models.PipelineStage
	var probFloat float64
	err := row.Scan(
		&s.ID, &s.Name, &s.Color, &s.SortOrder, &s.IsWon, &s.IsLost, &probFloat,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStageNotFound
	}
	if err != nil {
		return nil, err
	}
	s.Probability = decimal.NewFromFloat(probFloat)
	return &s, nil
}

// Helper to scan rows iterator
func (r *PostgresRepository) scanStageFromRows(rows pgx.Rows) (*models.PipelineStage, error) {
	var s models.PipelineStage
	var probFloat float64
	err := rows.Scan(
		&s.ID, &s.Name, &s.Color, &s.SortOrder, &s.IsWon, &s.IsLost, &probFloat,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Probability = decimal.NewFromFloat(probFloat)
	return &s, nil
}
