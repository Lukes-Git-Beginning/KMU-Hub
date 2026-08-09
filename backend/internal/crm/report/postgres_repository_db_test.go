package report

// The three aggregation queries in postgres_repository.go had 0% DB coverage
// before this file: service_test.go only exercises the mock Repository, which
// proves the service forwards whatever the repo returns and nothing about the
// SQL itself. GetPipelineReport in particular has never run against a real
// schema — it grouped/ordered by "ps.position", a column that does not exist
// (pipeline_stages has sort_order, see migration 000008); every real call
// would have failed with a SQL error. Fixed in this commit; the tests below
// are what would have caught it, plus the tenant-scoping and empty-result
// guarantees the backlog unit asks for.
//
// Skips cleanly without DATABASE_URL (CI runners without Postgres).

import (
	"context"
	"fmt"
	"maps"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func newReportFixture(t *testing.T) (*pgxpool.Pool, uuid.UUID, uuid.UUID, context.Context) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Report Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("report-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     tenantID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	return pool, tenantID, userID, testutil.WithTenantCtx(context.Background(), tenantID)
}

func seedStage(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, cols map[string]any) uuid.UUID {
	t.Helper()
	base := map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"name": "Stage-" + uuid.NewString()[:8],
		"sort_order": 1, "probability": "10.00",
	}
	maps.Copy(base, cols)
	id := testutil.SeedRow(t, pool, "pipeline_stages", base)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "pipeline_stages", id) })
	return id
}

func seedDeal(t *testing.T, pool *pgxpool.Pool, tenantID, userID, stageID uuid.UUID, cols map[string]any) uuid.UUID {
	t.Helper()
	base := map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"name": "Deal-" + uuid.NewString()[:8],
		"value":      "1000.00",
		"stage_id":   stageID,
		"created_by": userID,
	}
	maps.Copy(base, cols)
	id := testutil.SeedRow(t, pool, "deals", base)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "deals", id) })
	return id
}

func seedActivity(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, cols map[string]any) uuid.UUID {
	t.Helper()
	base := map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"activity_type": "call",
		"subject":       "Activity-" + uuid.NewString()[:8],
		"created_by":    userID,
	}
	maps.Copy(base, cols)
	id := testutil.SeedRow(t, pool, "activities", base)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "activities", id) })
	return id
}

func seedStageHistory(t *testing.T, pool *pgxpool.Pool, tenantID, dealID uuid.UUID, cols map[string]any) uuid.UUID {
	t.Helper()
	base := map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "deal_id": dealID,
	}
	maps.Copy(base, cols)
	id := testutil.SeedRow(t, pool, "deal_stage_history", base)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "deal_stage_history", id) })
	return id
}

// ============================================================================
// GetPipelineReport
// ============================================================================

// TestGetPipelineReport_ScopedByTenantOrderedBySortOrderAndWeightsByProbability
// is the test that would have caught the "ps.position" bug (SQL error on
// every call) and also pins the weighted-value formula (value * probability
// / 100) and the sort_order-based ordering. A second tenant's stage and deal
// sit in the same tables to prove the join is tenant-scoped, not just the
// session's RLS.
func TestGetPipelineReport_ScopedByTenantOrderedBySortOrderAndWeightsByProbability(t *testing.T) {
	pool, tenantA, userA, ctxA := newReportFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Report Tenant B")
	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "email": fmt.Sprintf("report-b-%s@test.invalid", uuid.New()),
		"password_hash": "x", "tenant_id": tenantB,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userB) })
	repo := NewPostgresRepository(pool)

	dateFrom := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	inRange := dateFrom.AddDate(0, 0, 10)

	stageLast := seedStage(t, pool, tenantA, map[string]any{"sort_order": 2, "probability": "50.00"})
	stageFirst := seedStage(t, pool, tenantA, map[string]any{"sort_order": 1, "probability": "10.00"})
	seedDeal(t, pool, tenantA, userA, stageFirst, map[string]any{"value": "1000.00", "created_at": inRange})
	seedDeal(t, pool, tenantA, userA, stageLast, map[string]any{"value": "2000.00", "created_at": inRange})

	// Same date window, a stage and a deal for a different tenant — must not
	// leak into tenant A's stages, counts or totals.
	stageB := seedStage(t, pool, tenantB, map[string]any{"sort_order": 1, "probability": "90.00"})
	seedDeal(t, pool, tenantB, userB, stageB, map[string]any{"value": "999999.00", "created_at": inRange})

	report, err := repo.GetPipelineReport(ctxA, PipelineFilter{
		TenantID: tenantA, StartDate: dateFrom, EndDate: dateTo,
	})
	if err != nil {
		t.Fatalf("GetPipelineReport: %v", err)
	}

	if len(report.Stages) != 2 {
		t.Fatalf("Stages count = %d, want 2 (tenant A's own stages only)", len(report.Stages))
	}
	// sort_order ASC: stageFirst (1) before stageLast (2).
	if report.Stages[0].StageID != stageFirst || report.Stages[1].StageID != stageLast {
		t.Errorf("Stages order = [%s, %s], want [stageFirst, stageLast] by sort_order ASC", report.Stages[0].StageID, report.Stages[1].StageID)
	}
	if report.Stages[0].DealCount != 1 || !report.Stages[0].TotalValue.Equal(mustDecimalReport(t, "1000")) {
		t.Errorf("stageFirst = count %d value %s, want count 1 value 1000", report.Stages[0].DealCount, report.Stages[0].TotalValue)
	}
	// weighted = 1000 * 10/100 = 100
	if !report.Stages[0].WeightedValue.Equal(mustDecimalReport(t, "100")) {
		t.Errorf("stageFirst.WeightedValue = %s, want 100 (value * probability/100)", report.Stages[0].WeightedValue)
	}
	// weighted = 2000 * 50/100 = 1000
	if !report.Stages[1].WeightedValue.Equal(mustDecimalReport(t, "1000")) {
		t.Errorf("stageLast.WeightedValue = %s, want 1000 (value * probability/100)", report.Stages[1].WeightedValue)
	}
	if report.TotalDeals != 2 {
		t.Errorf("TotalDeals = %d, want 2 (tenant B's deal must not leak in)", report.TotalDeals)
	}
	if !report.TotalValue.Equal(mustDecimalReport(t, "3000")) {
		t.Errorf("TotalValue = %s, want 3000", report.TotalValue)
	}
}

func TestGetPipelineReport_OwnerFilterAppliesWithinTenant(t *testing.T) {
	pool, tenantA, ownerA, ctxA := newReportFixture(t)
	ownerOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "email": fmt.Sprintf("report-owner-%s@test.invalid", uuid.New()),
		"password_hash": "x", "tenant_id": tenantA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", ownerOther) })
	repo := NewPostgresRepository(pool)

	dateFrom := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	inRange := dateFrom.AddDate(0, 0, 5)

	stage := seedStage(t, pool, tenantA, nil)
	seedDeal(t, pool, tenantA, ownerA, stage, map[string]any{"value": "500.00", "owner_id": ownerA, "created_at": inRange})
	seedDeal(t, pool, tenantA, ownerA, stage, map[string]any{"value": "700.00", "owner_id": ownerOther, "created_at": inRange})

	report, err := repo.GetPipelineReport(ctxA, PipelineFilter{
		TenantID: tenantA, OwnerID: &ownerA, StartDate: dateFrom, EndDate: dateTo,
	})
	if err != nil {
		t.Fatalf("GetPipelineReport with owner filter: %v", err)
	}
	if report.TotalDeals != 1 {
		t.Fatalf("TotalDeals = %d, want 1 (only ownerA's deal)", report.TotalDeals)
	}
	if !report.TotalValue.Equal(mustDecimalReport(t, "500")) {
		t.Errorf("TotalValue = %s, want 500", report.TotalValue)
	}
}

// TestGetPipelineReport_TenantWithNoStagesReturnsEmptyListNotNil covers the
// case with zero pipeline_stages rows for the tenant: the join produces zero
// rows, and the repo pre-allocates Stages as an empty slice (see line 59) —
// this pins that it stays that way and never regresses to nil, which would
// serialize as `null` on the wire instead of `[]`.
func TestGetPipelineReport_TenantWithNoStagesReturnsEmptyListNotNil(t *testing.T) {
	_, tenantID, _, ctx := newReportFixture(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	repo := NewPostgresRepository(pool)

	report, err := repo.GetPipelineReport(ctx, PipelineFilter{
		TenantID: tenantID, StartDate: time.Now().AddDate(0, -1, 0), EndDate: time.Now(),
	})
	if err != nil {
		t.Fatalf("GetPipelineReport on a tenant with no stages: %v", err)
	}
	if report.Stages == nil {
		t.Fatal("Stages is nil, want an empty non-nil slice")
	}
	if len(report.Stages) != 0 {
		t.Errorf("Stages count = %d, want 0", len(report.Stages))
	}
	if report.TotalDeals != 0 {
		t.Errorf("TotalDeals = %d, want 0", report.TotalDeals)
	}
}

func TestGetPipelineReport_CanceledContextReturnsError(t *testing.T) {
	_, tenantID, _, ctx := newReportFixture(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	repo := NewPostgresRepository(pool)

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err := repo.GetPipelineReport(canceledCtx, PipelineFilter{
		TenantID: tenantID, StartDate: time.Now().AddDate(0, -1, 0), EndDate: time.Now(),
	})
	if err == nil {
		t.Fatal("GetPipelineReport with a canceled context: got nil error, want one")
	}
}

// ============================================================================
// GetConversionReport
// ============================================================================

// TestGetConversionReport_ScopedByTenantWithAverageDays seeds deal_stage_history
// rows directly (bypassing the trigger) so changed_at is fully controlled,
// and checks both the tenant scoping of the join through deals and the
// average-days computation. A second tenant's transition in the same date
// window and with the same stage names must not appear in tenant A's metrics.
func TestGetConversionReport_ScopedByTenantWithAverageDays(t *testing.T) {
	pool, tenantA, userA, ctxA := newReportFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Report Tenant B")
	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "email": fmt.Sprintf("report-conv-b-%s@test.invalid", uuid.New()),
		"password_hash": "x", "tenant_id": tenantB,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userB) })
	repo := NewPostgresRepository(pool)

	dateFrom := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)

	stageNew := seedStage(t, pool, tenantA, map[string]any{"name": "New"})
	stageQualified := seedStage(t, pool, tenantA, map[string]any{"name": "Qualified"})
	dealA := seedDeal(t, pool, tenantA, userA, stageNew, nil)

	firstTransition := dateFrom.AddDate(0, 0, 5)
	secondTransition := firstTransition.AddDate(0, 0, 4) // +4 days
	seedStageHistory(t, pool, tenantA, dealA, map[string]any{
		"from_stage_id": nil, "to_stage_id": stageNew, "changed_at": firstTransition,
	})
	seedStageHistory(t, pool, tenantA, dealA, map[string]any{
		"from_stage_id": stageNew, "to_stage_id": stageQualified, "changed_at": secondTransition,
	})

	// Tenant B: same window, same stage name "Qualified" reused via its own
	// stage row — must not be counted in tenant A's metrics.
	stageBNew := seedStage(t, pool, tenantB, map[string]any{"name": "New"})
	stageBQualified := seedStage(t, pool, tenantB, map[string]any{"name": "Qualified"})
	dealB := seedDeal(t, pool, tenantB, userB, stageBNew, nil)
	seedStageHistory(t, pool, tenantB, dealB, map[string]any{
		"from_stage_id": stageBNew, "to_stage_id": stageBQualified, "changed_at": firstTransition,
	})

	report, err := repo.GetConversionReport(ctxA, tenantA, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetConversionReport: %v", err)
	}

	if len(report.Metrics) != 2 {
		t.Fatalf("Metrics count = %d, want 2 (New->Qualified transition counted once per from/to pair; tenant B's transition excluded)", len(report.Metrics))
	}

	var qualified *struct {
		count       int32
		averageDays float64
	}
	for _, m := range report.Metrics {
		if m.FromStage == "New" && m.ToStage == "Qualified" {
			qualified = &struct {
				count       int32
				averageDays float64
			}{m.ConvertedCount, m.AverageDays}
		}
	}
	if qualified == nil {
		t.Fatal("no New->Qualified metric found")
	}
	if qualified.count != 1 {
		t.Errorf("New->Qualified ConvertedCount = %d, want 1", qualified.count)
	}
	if qualified.averageDays < 3.9 || qualified.averageDays > 4.1 {
		t.Errorf("New->Qualified AverageDays = %.2f, want ~4.0", qualified.averageDays)
	}
}

func TestGetConversionReport_TenantWithNoTransitionsReturnsEmptyMetricsNotNil(t *testing.T) {
	_, tenantID, _, ctx := newReportFixture(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	repo := NewPostgresRepository(pool)

	report, err := repo.GetConversionReport(ctx, tenantID, time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		t.Fatalf("GetConversionReport on a tenant with no transitions: %v", err)
	}
	if report.Metrics == nil {
		t.Fatal("Metrics is nil, want an empty non-nil slice")
	}
	if len(report.Metrics) != 0 {
		t.Errorf("Metrics count = %d, want 0", len(report.Metrics))
	}
	if report.OverallWinRate != 0 {
		t.Errorf("OverallWinRate = %v, want 0 on an empty tenant", report.OverallWinRate)
	}
}

func TestGetConversionReport_CanceledContextReturnsError(t *testing.T) {
	_, tenantID, _, ctx := newReportFixture(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	repo := NewPostgresRepository(pool)

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err := repo.GetConversionReport(canceledCtx, tenantID, time.Now().AddDate(0, -1, 0), time.Now())
	if err == nil {
		t.Fatal("GetConversionReport with a canceled context: got nil error, want one")
	}
}

// ============================================================================
// GetActivityReport
// ============================================================================

func TestGetActivityReport_ScopedByTenantWithCompletedAndOverdueCounts(t *testing.T) {
	pool, tenantA, userA, ctxA := newReportFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Report Tenant B")
	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "email": fmt.Sprintf("report-act-b-%s@test.invalid", uuid.New()),
		"password_hash": "x", "tenant_id": tenantB,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userB) })
	repo := NewPostgresRepository(pool)

	dateFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	inRange := dateFrom.AddDate(0, 0, 10)
	pastDue := dateFrom.AddDate(0, 0, -400) // long overdue, still due before now

	seedActivity(t, pool, tenantA, userA, map[string]any{
		"activity_type": "call", "is_completed": true, "created_at": inRange,
	})
	seedActivity(t, pool, tenantA, userA, map[string]any{
		"activity_type": "call", "is_completed": false, "due_date": pastDue, "created_at": inRange,
	})
	// Same window, different tenant — must not leak into tenant A's counts.
	seedActivity(t, pool, tenantB, userB, map[string]any{
		"activity_type": "call", "is_completed": true, "created_at": inRange,
	})

	report, err := repo.GetActivityReport(ctxA, ActivityFilter{
		TenantID: tenantA, StartDate: dateFrom, EndDate: dateTo,
	})
	if err != nil {
		t.Fatalf("GetActivityReport: %v", err)
	}
	if report.TotalActivities != 2 {
		t.Fatalf("TotalActivities = %d, want 2 (tenant B's activity must not leak in)", report.TotalActivities)
	}
	var call *ActivityMetricLite
	for _, m := range report.Metrics {
		if m.ActivityType == "call" {
			call = &ActivityMetricLite{m.TotalCount, m.CompletedCount, m.OverdueCount}
		}
	}
	if call == nil {
		t.Fatal("no call metric found")
	}
	if call.TotalCount != 2 || call.CompletedCount != 1 || call.OverdueCount != 1 {
		t.Errorf("call metric = %+v, want {Total:2 Completed:1 Overdue:1}", call)
	}
}

// TestGetActivityReport_UserFilterMatchesCreatedByOrAssignedTo covers the two
// branches of the "(created_by = $4 OR assigned_to = $4)" filter: an activity
// the user created and one merely assigned to them must both count, while an
// activity belonging to someone else must not.
func TestGetActivityReport_UserFilterMatchesCreatedByOrAssignedTo(t *testing.T) {
	pool, tenantA, userA, ctxA := newReportFixture(t)
	otherUser := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "email": fmt.Sprintf("report-act-other-%s@test.invalid", uuid.New()),
		"password_hash": "x", "tenant_id": tenantA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", otherUser) })
	repo := NewPostgresRepository(pool)

	dateFrom := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	inRange := dateFrom.AddDate(0, 0, 5)

	seedActivity(t, pool, tenantA, userA, map[string]any{"created_at": inRange}) // created_by = userA
	seedActivity(t, pool, tenantA, otherUser, map[string]any{"assigned_to": userA, "created_at": inRange})
	seedActivity(t, pool, tenantA, otherUser, map[string]any{"created_at": inRange}) // neither created nor assigned to userA

	report, err := repo.GetActivityReport(ctxA, ActivityFilter{
		TenantID: tenantA, UserID: &userA, StartDate: dateFrom, EndDate: dateTo,
	})
	if err != nil {
		t.Fatalf("GetActivityReport with user filter: %v", err)
	}
	if report.TotalActivities != 2 {
		t.Fatalf("TotalActivities = %d, want 2 (created_by OR assigned_to userA)", report.TotalActivities)
	}
}

func TestGetActivityReport_TenantWithNoActivitiesReturnsEmptyListNotNil(t *testing.T) {
	_, tenantID, _, ctx := newReportFixture(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	repo := NewPostgresRepository(pool)

	report, err := repo.GetActivityReport(ctx, ActivityFilter{
		TenantID: tenantID, StartDate: time.Now().AddDate(0, -1, 0), EndDate: time.Now(),
	})
	if err != nil {
		t.Fatalf("GetActivityReport on a tenant with no activities: %v", err)
	}
	if report.Metrics == nil {
		t.Fatal("Metrics is nil, want an empty non-nil slice")
	}
	if len(report.Metrics) != 0 {
		t.Errorf("Metrics count = %d, want 0", len(report.Metrics))
	}
	if report.TotalActivities != 0 || report.CompletionRate != 0 {
		t.Errorf("TotalActivities/CompletionRate = %d/%v, want 0/0", report.TotalActivities, report.CompletionRate)
	}
}

func TestGetActivityReport_CanceledContextReturnsError(t *testing.T) {
	_, tenantID, _, ctx := newReportFixture(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	repo := NewPostgresRepository(pool)

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err := repo.GetActivityReport(canceledCtx, ActivityFilter{
		TenantID: tenantID, StartDate: time.Now().AddDate(0, -1, 0), EndDate: time.Now(),
	})
	if err == nil {
		t.Fatal("GetActivityReport with a canceled context: got nil error, want one")
	}
}

// ActivityMetricLite is a plain copy of the three ActivityMetric count fields
// used to build a local pointer in the table-scan loop above without aliasing
// the loop variable.
type ActivityMetricLite struct {
	TotalCount, CompletedCount, OverdueCount int32
}

func mustDecimalReport(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal %q: %v", s, err)
	}
	return d
}
