package report

import (
	"context"
	"time"

	"github.com/google/uuid"
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

func (r *PostgresRepository) GetPipelineReport(ctx context.Context, filter PipelineFilter) (*models.PipelineReport, error) {
	// Build query based on filter — always tenant-scoped via deals.tenant_id
	query := `
		SELECT
			ps.id AS stage_id,
			ps.name AS stage_name,
			COUNT(d.id)::int AS deal_count,
			COALESCE(SUM(d.value), 0) AS total_value,
			COALESCE(SUM(d.value * (ps.probability / 100.0)), 0) AS weighted_value
		FROM pipeline_stages ps
		LEFT JOIN deals d ON d.stage_id = ps.id
			AND d.tenant_id = $1
			AND d.created_at >= $2
			AND d.created_at <= $3
	`

	args := []any{filter.TenantID, filter.StartDate, filter.EndDate}

	if filter.OwnerID != nil {
		query += " AND d.owner_id = $4"
		args = append(args, *filter.OwnerID)
	}

	query += `
		GROUP BY ps.id, ps.name, ps.sort_order
		ORDER BY ps.sort_order ASC
	`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report := &models.PipelineReport{
		Stages: make([]*models.PipelineStageValue, 0),
	}

	for rows.Next() {
		var stage models.PipelineStageValue
		var totalValueStr, weightedValueStr string
		if scanErr := rows.Scan(&stage.StageID, &stage.StageName, &stage.DealCount, &totalValueStr, &weightedValueStr); scanErr != nil {
			return nil, scanErr
		}
		// Money is scanned as a string and parsed into decimal (ADR-0007); totals are
		// accumulated in decimal to avoid float rounding error across stages.
		stage.TotalValue, _ = decimal.NewFromString(totalValueStr)
		stage.WeightedValue, _ = decimal.NewFromString(weightedValueStr)
		report.Stages = append(report.Stages, &stage)
		report.TotalDeals += stage.DealCount
		report.TotalValue = report.TotalValue.Add(stage.TotalValue)
		report.TotalWeightedValue = report.TotalWeightedValue.Add(stage.WeightedValue)
	}

	return report, rows.Err()
}

func (r *PostgresRepository) GetConversionReport(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*models.ConversionReport, error) {
	// Query stage transitions with metrics — tenant-scoped via deal_stage_history join deals
	query := `
		WITH stage_transitions AS (
			SELECT
				dsh.from_stage_id,
				dsh.to_stage_id,
				ps_from.name AS from_stage,
				ps_to.name AS to_stage,
				dsh.changed_at,
				LAG(dsh.changed_at) OVER (PARTITION BY dsh.deal_id ORDER BY dsh.changed_at) AS prev_changed_at
			FROM deal_stage_history dsh
			JOIN deals d ON dsh.deal_id = d.id AND d.tenant_id = $1
			LEFT JOIN pipeline_stages ps_from ON dsh.from_stage_id = ps_from.id
			JOIN pipeline_stages ps_to ON dsh.to_stage_id = ps_to.id
			WHERE dsh.changed_at >= $2 AND dsh.changed_at <= $3
		)
		SELECT
			COALESCE(from_stage, 'New') AS from_stage,
			to_stage,
			COUNT(*)::int AS converted_count,
			EXTRACT(EPOCH FROM AVG(changed_at - COALESCE(prev_changed_at, changed_at))) / 86400.0 AS average_days
		FROM stage_transitions
		GROUP BY from_stage, to_stage
		ORDER BY from_stage, to_stage
	`

	rows, err := r.pool.Query(ctx, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report := &models.ConversionReport{
		Metrics: make([]*models.ConversionMetric, 0),
	}

	totalTransitions := 0
	for rows.Next() {
		var metric models.ConversionMetric
		if scanErr := rows.Scan(&metric.FromStage, &metric.ToStage, &metric.ConvertedCount, &metric.AverageDays); scanErr != nil {
			return nil, scanErr
		}
		totalTransitions += int(metric.ConvertedCount)
		report.Metrics = append(report.Metrics, &metric)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Calculate conversion rates relative to total
	for _, m := range report.Metrics {
		if totalTransitions > 0 {
			m.ConversionRate = float64(m.ConvertedCount) / float64(totalTransitions) * 100
		}
	}

	// Calculate overall win rate and average deal cycle
	winQuery := `
		SELECT
			COUNT(CASE WHEN ps.name = 'Won' THEN 1 END)::float / NULLIF(COUNT(*), 0) * 100 AS win_rate,
			COALESCE(AVG(EXTRACT(EPOCH FROM (d.updated_at - d.created_at)) / 86400.0), 0) AS avg_cycle
		FROM deals d
		JOIN pipeline_stages ps ON d.stage_id = ps.id
		WHERE d.tenant_id = $1 AND d.created_at >= $2 AND d.created_at <= $3
	`

	var winRate, avgCycle *float64
	if scanErr := r.pool.QueryRow(ctx, winQuery, tenantID, startDate, endDate).Scan(&winRate, &avgCycle); scanErr != nil {
		return nil, scanErr
	}
	if winRate != nil {
		report.OverallWinRate = *winRate
	}
	if avgCycle != nil {
		report.AverageDealCycle = *avgCycle
	}

	return report, nil
}

func (r *PostgresRepository) GetActivityReport(ctx context.Context, filter ActivityFilter) (*models.ActivityReport, error) {
	query := `
		SELECT
			activity_type::text,
			COUNT(*)::int AS total_count,
			COUNT(CASE WHEN is_completed THEN 1 END)::int AS completed_count,
			COUNT(CASE WHEN NOT is_completed AND due_date < NOW() THEN 1 END)::int AS overdue_count
		FROM activities
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at <= $3
	`

	args := []any{filter.TenantID, filter.StartDate, filter.EndDate}

	if filter.UserID != nil {
		query += " AND (created_by = $4 OR assigned_to = $4)"
		args = append(args, *filter.UserID)
	}

	query += `
		GROUP BY activity_type
		ORDER BY activity_type
	`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report := &models.ActivityReport{
		Metrics: make([]*models.ActivityMetric, 0),
	}

	totalCompleted := int32(0)
	for rows.Next() {
		var metric models.ActivityMetric
		if scanErr := rows.Scan(&metric.ActivityType, &metric.TotalCount, &metric.CompletedCount, &metric.OverdueCount); scanErr != nil {
			return nil, scanErr
		}
		report.Metrics = append(report.Metrics, &metric)
		report.TotalActivities += metric.TotalCount
		totalCompleted += metric.CompletedCount
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if report.TotalActivities > 0 {
		report.CompletionRate = float64(totalCompleted) / float64(report.TotalActivities) * 100
	}

	return report, nil
}
