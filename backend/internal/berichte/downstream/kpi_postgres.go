// Package downstream implements the berichte executor's cross-module
// aggregation interfaces against Postgres. It lives outside
// internal/berichte/executor so the executor keeps its narrow interfaces and
// stays unit-testable without a database.
//
// The queries read other modules' tables (finance_invoices, deals, tickets,
// stock_warnings) directly. That is deliberate for a reporting module sitting
// on the same database — routing one dashboard through four owning services
// would cost four gRPC round-trips for numbers that a single query answers.
// Every predicate is tenant-scoped explicitly on top of RLS.
package downstream

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/berichte/executor"
)

// PostgresKPIRepo aggregates dashboard KPIs from the module tables.
type PostgresKPIRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresKPIRepo creates a KPI repository over the given pool.
func NewPostgresKPIRepo(pool *pgxpool.Pool) *PostgresKPIRepo {
	return &PostgresKPIRepo{pool: pool}
}

// kpiSnapshotQuery collects all four dashboard aggregates in one round-trip.
//
// Monetary sums are cast to text and parsed as decimal rather than scanned as
// float64 (ADR-0007). The invoice status set mirrors the finance dashboard's
// "invoiced" definition so both views agree.
//
// Revenue counts what was invoiced inside the period. Pipeline volume and open
// tickets are point-in-time values reconstructed as of the period end, so the
// same query serves both the current and the previous period. Stock warnings
// carry no history and are therefore always current.
//
// The ::timestamptz casts are load-bearing, not decoration: $2/$3 are compared
// against invoice_date (date) as well as created_at (timestamptz). Without a
// cast Postgres infers the parameter as `date` from its first use, pgx then
// sends the bounds date-truncated, and every point-in-time predicate silently
// clips to midnight — pipeline volume and open tickets read 0 all day.
const kpiSnapshotQuery = `
	SELECT
		COALESCE((
			SELECT SUM(gross_total) FROM finance_invoices
			WHERE tenant_id = $1
			  AND status IN ('sent', 'paid', 'overdue')
			  AND invoice_date >= ($2::timestamptz)::date
			  AND invoice_date <= ($3::timestamptz)::date
		), 0)::text AS revenue,
		COALESCE((
			SELECT SUM(value) FROM deals
			WHERE tenant_id = $1
			  AND created_at <= $3::timestamptz
			  AND (closed_at IS NULL OR closed_at > $3::timestamptz)
		), 0)::text AS pipeline_volume,
		(
			SELECT COUNT(*) FROM tickets
			WHERE tenant_id = $1
			  AND status <> 'merged'
			  AND created_at <= $3::timestamptz
			  AND (resolved_at IS NULL OR resolved_at > $3::timestamptz)
		)::int AS open_tickets,
		(
			SELECT COUNT(*) FROM stock_warnings
			WHERE tenant_id = $1 AND status = 'active'
		)::int AS stock_warnings`

// KPISnapshot returns the cross-module aggregates for [from, to].
func (r *PostgresKPIRepo) KPISnapshot(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*executor.KPISnapshot, error) {
	var revenueRaw, pipelineRaw string
	snap := &executor.KPISnapshot{}

	if err := r.pool.QueryRow(ctx, kpiSnapshotQuery, tenantID, from, to).
		Scan(&revenueRaw, &pipelineRaw, &snap.OpenTickets, &snap.StockWarnings); err != nil {
		return nil, fmt.Errorf("kpi snapshot: %w", err)
	}

	revenue, err := decimal.NewFromString(revenueRaw)
	if err != nil {
		return nil, fmt.Errorf("kpi snapshot: parse revenue %q: %w", revenueRaw, err)
	}
	pipeline, err := decimal.NewFromString(pipelineRaw)
	if err != nil {
		return nil, fmt.Errorf("kpi snapshot: parse pipeline volume %q: %w", pipelineRaw, err)
	}

	snap.Revenue = revenue
	snap.PipelineVolume = pipeline
	return snap, nil
}

// kpiSeriesQuery buckets the same three history-capable metrics from
// kpiSnapshotQuery (stock_warnings has no history, see the executor's doc
// comment) into `periods` calendar months ending in the month of $2, in one
// round trip via a shared `periods` CTE instead of one query per bucket.
//
// Each bucket's upper bound is exclusive: the 1st of the following month for
// a past bucket, or the day after $2 for the current (partial) month — so the
// last point matches KPISnapshot's "current" reading exactly (month-to-date
// as of $2). Revenue buckets on invoice_date (a flow within the bucket);
// pipeline volume and open tickets are point-in-time as of the bucket's
// upper bound (a stock value, same convention as kpiSnapshotQuery).
const kpiSeriesQuery = `
	WITH periods AS (
		SELECT
			gs::date AS period_start,
			LEAST(gs + INTERVAL '1 month', $2::timestamptz + INTERVAL '1 day') AS period_end
		FROM generate_series(
			date_trunc('month', $2::timestamptz) - ($3::int - 1) * INTERVAL '1 month',
			date_trunc('month', $2::timestamptz),
			INTERVAL '1 month'
		) AS gs
	)
	SELECT
		p.period_start,
		COALESCE((
			SELECT SUM(gross_total) FROM finance_invoices
			WHERE tenant_id = $1
			  AND status IN ('sent', 'paid', 'overdue')
			  AND invoice_date >= p.period_start
			  AND invoice_date < p.period_end::date
		), 0)::text AS revenue,
		COALESCE((
			SELECT SUM(value) FROM deals
			WHERE tenant_id = $1
			  AND created_at < p.period_end
			  AND (closed_at IS NULL OR closed_at >= p.period_end)
		), 0)::text AS pipeline_volume,
		(
			SELECT COUNT(*) FROM tickets
			WHERE tenant_id = $1
			  AND status <> 'merged'
			  AND created_at < p.period_end
			  AND (resolved_at IS NULL OR resolved_at >= p.period_end)
		)::int AS open_tickets
	FROM periods p
	ORDER BY p.period_start`

// KPISeries returns the last `periods` calendar-month buckets ending in the
// month of `now`, oldest first.
func (r *PostgresKPIRepo) KPISeries(ctx context.Context, tenantID uuid.UUID, now time.Time, periods int) ([]executor.KPISeriesPoint, error) {
	rows, err := r.pool.Query(ctx, kpiSeriesQuery, tenantID, now, periods)
	if err != nil {
		return nil, fmt.Errorf("kpi series: %w", err)
	}
	defer rows.Close()

	out := make([]executor.KPISeriesPoint, 0, periods)
	for rows.Next() {
		var p executor.KPISeriesPoint
		var revenueRaw, pipelineRaw string
		if err := rows.Scan(&p.PeriodStart, &revenueRaw, &pipelineRaw, &p.OpenTickets); err != nil {
			return nil, fmt.Errorf("kpi series: scan: %w", err)
		}
		revenue, err := decimal.NewFromString(revenueRaw)
		if err != nil {
			return nil, fmt.Errorf("kpi series: parse revenue %q: %w", revenueRaw, err)
		}
		pipeline, err := decimal.NewFromString(pipelineRaw)
		if err != nil {
			return nil, fmt.Errorf("kpi series: parse pipeline volume %q: %w", pipelineRaw, err)
		}
		p.Revenue = revenue
		p.PipelineVolume = pipeline
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kpi series: %w", err)
	}
	return out, nil
}
