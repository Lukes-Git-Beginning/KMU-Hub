package holiday

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) BulkUpsert(ctx context.Context, holidays []models.PublicHoliday) error {
	if len(holidays) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	query := `
		INSERT INTO public_holidays (id, date, name, local_name, country_code, is_global, subdivision_codes, holiday_type, year, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (date, country_code, name) DO UPDATE SET
			local_name = EXCLUDED.local_name,
			is_global = EXCLUDED.is_global,
			subdivision_codes = EXCLUDED.subdivision_codes,
			holiday_type = EXCLUDED.holiday_type`

	for _, h := range holidays {
		createdAt := h.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		_, execErr := tx.Exec(ctx, query,
			h.ID, h.Date, h.Name, h.LocalName, h.CountryCode,
			h.IsGlobal, h.SubdivisionCodes, h.HolidayType, h.Year, createdAt,
		)
		if execErr != nil {
			return fmt.Errorf("upsert holiday %q: %w", h.Name, execErr)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) ListByDateRange(ctx context.Context, start, end time.Time, countryCode string, subdivisionCode *string) ([]models.PublicHoliday, error) {
	var query string
	var args []any

	if subdivisionCode != nil {
		query = `
			SELECT id, date, name, local_name, country_code, is_global, subdivision_codes, holiday_type, year, created_at
			FROM public_holidays
			WHERE date >= $1 AND date <= $2
			  AND country_code = $3
			  AND (is_global = true OR $4 = ANY(subdivision_codes))
			ORDER BY date`
		args = []any{start, end, countryCode, *subdivisionCode}
	} else {
		query = `
			SELECT id, date, name, local_name, country_code, is_global, subdivision_codes, holiday_type, year, created_at
			FROM public_holidays
			WHERE date >= $1 AND date <= $2
			  AND country_code = $3
			ORDER BY date`
		args = []any{start, end, countryCode}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list holidays: %w", err)
	}
	defer rows.Close()

	var holidays []models.PublicHoliday
	for rows.Next() {
		var h models.PublicHoliday
		if scanErr := rows.Scan(
			&h.ID, &h.Date, &h.Name, &h.LocalName, &h.CountryCode,
			&h.IsGlobal, &h.SubdivisionCodes, &h.HolidayType, &h.Year, &h.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan holiday: %w", scanErr)
		}
		holidays = append(holidays, h)
	}
	return holidays, rows.Err()
}

func (r *PostgresRepository) HasDataForYear(ctx context.Context, year int, countryCode string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM public_holidays WHERE year = $1 AND country_code = $2)`,
		year, countryCode,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check holiday data: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) DeleteByYear(ctx context.Context, year int, countryCode string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM public_holidays WHERE year = $1 AND country_code = $2`,
		year, countryCode,
	)
	if err != nil {
		return fmt.Errorf("delete holidays: %w", err)
	}
	return nil
}
