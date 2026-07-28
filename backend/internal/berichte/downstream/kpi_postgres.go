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
