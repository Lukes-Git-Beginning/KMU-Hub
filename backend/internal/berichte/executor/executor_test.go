package executor

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

// ============================================================================
// Fixtures and mocks
// ============================================================================

var testTenant = uuid.MustParse("22222222-2222-2222-2222-222222222222")

type fixedClock struct{ t time.Time }

func (f *fixedClock) Now() time.Time { return f.t }

type mockFinance struct {
	revenue []MonthlyRevenue
	open    []InvoiceSummary
	err     error
}

func (m *mockFinance) GetRevenueByMonth(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]MonthlyRevenue, error) {
	return m.revenue, m.err
}

func (m *mockFinance) ListInvoicesByStatus(_ context.Context, _ uuid.UUID, _ []string) ([]InvoiceSummary, error) {
	return m.open, m.err
}

type mockCRM struct {
	pipeline   *CRMPipelineReport
	conversion *CRMConversionReport
	activity   *CRMActivityReport
	err        error

	// Captured by GetConversionReport so a test can assert which period
	// actually reached the downstream (cross sources may override it).
	conversionFrom time.Time
	conversionTo   time.Time
}

func (m *mockCRM) GetPipelineReport(_ context.Context, _ uuid.UUID) (*CRMPipelineReport, error) {
	return m.pipeline, m.err
}

func (m *mockCRM) GetConversionReport(_ context.Context, _ uuid.UUID, from, to time.Time) (*CRMConversionReport, error) {
	m.conversionFrom, m.conversionTo = from, to
	return m.conversion, m.err
}

func (m *mockCRM) GetActivityReport(_ context.Context, _ uuid.UUID, _, _ time.Time) (*CRMActivityReport, error) {
	return m.activity, m.err
}

type mockHelpdesk struct {
	report *HelpdeskSLAReport
	err    error
}

func (m *mockHelpdesk) GetSLAReport(_ context.Context, _ uuid.UUID, _, _ time.Time) (*HelpdeskSLAReport, error) {
	return m.report, m.err
}

type mockInventar struct {
	warnings []StockWarning
	err      error
}

func (m *mockInventar) GetStockWarnings(_ context.Context, _ uuid.UUID) ([]StockWarning, error) {
	return m.warnings, m.err
}

type kpiCall struct {
	tenantID uuid.UUID
	from, to time.Time
}

// mockKPI answers snapshot calls in order: first the current period, then the
// previous one. Series calls are separate: a fixed result (or error) returned
// on every call, since DashboardKPIs only ever asks for one series per
// invocation (unlike the two ordered snapshot calls).
type mockKPI struct {
	snapshots []*KPISnapshot
	errs      []error
	calls     []kpiCall

	series      []KPISeriesPoint
	seriesErr   error
	seriesCalls []kpiSeriesCall
}

type kpiSeriesCall struct {
	tenantID uuid.UUID
	now      time.Time
	periods  int
}

func (m *mockKPI) KPISnapshot(_ context.Context, tenantID uuid.UUID, from, to time.Time) (*KPISnapshot, error) {
	i := len(m.calls)
	m.calls = append(m.calls, kpiCall{tenantID: tenantID, from: from, to: to})
	if i < len(m.errs) && m.errs[i] != nil {
		return nil, m.errs[i]
	}
	if i >= len(m.snapshots) {
		return &KPISnapshot{}, nil
	}
	return m.snapshots[i], nil
}

func (m *mockKPI) KPISeries(_ context.Context, tenantID uuid.UUID, now time.Time, periods int) ([]KPISeriesPoint, error) {
	m.seriesCalls = append(m.seriesCalls, kpiSeriesCall{tenantID: tenantID, now: now, periods: periods})
	if m.seriesErr != nil {
		return nil, m.seriesErr
	}
	return m.series, nil
}

type mockDatev struct {
	data *BWAData
	err  error
}

func (m *mockDatev) GetBWAData(_ context.Context, _ uuid.UUID, _, _ time.Time) (*BWAData, error) {
	return m.data, m.err
}

// ============================================================================
// Helpers
// ============================================================================

func newTestExecutor(deps Deps) *Executor {
	if deps.Clock == nil {
		deps.Clock = &fixedClock{t: time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)}
	}
	return New(deps)
}

func testDefinition(kind string) *berichte.Definition {
	raw, _ := json.Marshal(map[string]any{"kind": kind, "period": "last_30_days"})
	return &berichte.Definition{
		ID:            uuid.New(),
		TenantID:      testTenant,
		Name:          "Test",
		Module:        "cross",
		Kind:          "system",
		QueryConfig:   raw,
		DefaultFormat: "pdf",
	}
}

// ============================================================================
// Tests: Run dispatch
// ============================================================================

func TestRun_UnknownKind(t *testing.T) {
	exec := newTestExecutor(Deps{})
	_, err := exec.Run(context.Background(), testDefinition("wtf"), nil)
	if !errors.Is(err, berichte.ErrInvalidQueryConfig) {
		t.Errorf("expected ErrInvalidQueryConfig, got %v", err)
	}
}

func TestRun_EmptyConfig(t *testing.T) {
	exec := newTestExecutor(Deps{})
	def := &berichte.Definition{ID: uuid.New(), TenantID: testTenant, QueryConfig: nil}
	_, err := exec.Run(context.Background(), def, nil)
	if !errors.Is(err, berichte.ErrInvalidQueryConfig) {
		t.Errorf("expected ErrInvalidQueryConfig, got %v", err)
	}
}

func TestRun_MalformedConfig(t *testing.T) {
	exec := newTestExecutor(Deps{})
	def := &berichte.Definition{ID: uuid.New(), TenantID: testTenant, QueryConfig: []byte("not-json")}
	_, err := exec.Run(context.Background(), def, nil)
	if err == nil {
		t.Error("expected error for malformed config")
	}
}

// ============================================================================
// Tests: revenue_by_month
// ============================================================================

func TestRun_RevenueByMonth_HappyPath(t *testing.T) {
	finance := &mockFinance{
		revenue: []MonthlyRevenue{
			{Month: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Revenue: 10000, InvoiceCnt: 5},
			{Month: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Revenue: 15000, InvoiceCnt: 8},
		},
	}
	exec := newTestExecutor(Deps{Finance: finance})
	result, err := exec.Run(context.Background(), testDefinition("revenue_by_month"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta.RowCount != 2 {
		t.Errorf("expected 2 rows, got %d", result.Meta.RowCount)
	}
	if result.Totals["revenue"] != 25000.0 {
		t.Errorf("expected total 25000, got %v", result.Totals["revenue"])
	}
	if len(result.Series) != 1 || len(result.Series[0].DataPoints) != 2 {
		t.Errorf("unexpected series shape: %+v", result.Series)
	}
}

func TestRun_RevenueByMonth_NoFinance(t *testing.T) {
	exec := newTestExecutor(Deps{})
	result, err := exec.Run(context.Background(), testDefinition("revenue_by_month"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta.Warning != "downstream_not_available" {
		t.Errorf("expected warning, got %q", result.Meta.Warning)
	}
}

func TestRun_RevenueByMonth_PropagatesError(t *testing.T) {
	finance := &mockFinance{err: errors.New("db down")}
	exec := newTestExecutor(Deps{Finance: finance})
	_, err := exec.Run(context.Background(), testDefinition("revenue_by_month"), nil)
	if err == nil {
		t.Error("expected error from downstream")
	}
}

// ============================================================================
// Tests: invoices_open
// ============================================================================

func TestRun_InvoicesOpen_HappyPath(t *testing.T) {
	due := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	finance := &mockFinance{
		open: []InvoiceSummary{
			{Number: "R-001", Customer: "ACME", Status: "sent", Amount: 1000, Currency: "EUR", DueDate: &due, OverdueDay: 0},
			{Number: "R-002", Customer: "Beta", Status: "overdue", Amount: 500, Currency: "EUR", OverdueDay: 10},
		},
	}
	exec := newTestExecutor(Deps{Finance: finance})
	result, err := exec.Run(context.Background(), testDefinition("invoices_open"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Totals["amount"] != 1500.0 {
		t.Errorf("expected total 1500, got %v", result.Totals["amount"])
	}
	if result.Rows[0]["due_date"] != "2026-04-20" {
		t.Errorf("unexpected due_date format: %v", result.Rows[0]["due_date"])
	}
	if result.Rows[1]["due_date"] != "" {
		t.Errorf("expected empty due_date for nil, got %v", result.Rows[1]["due_date"])
	}
}

func TestRun_InvoicesOpen_NoFinance(t *testing.T) {
	exec := newTestExecutor(Deps{})
	result, err := exec.Run(context.Background(), testDefinition("invoices_open"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta.Warning == "" {
		t.Error("expected warning for missing finance repo")
	}
}

// ============================================================================
// Tests: pipeline
// ============================================================================

func TestRun_Pipeline_HappyPath(t *testing.T) {
	crm := &mockCRM{
		pipeline: &CRMPipelineReport{
			Stages: []CRMPipelineStage{
				{Stage: "Lead", DealCnt: 10, Volume: 50000, Currency: "EUR"},
				{Stage: "Qualified", DealCnt: 5, Volume: 25000, Currency: "EUR"},
			},
		},
	}
	exec := newTestExecutor(Deps{CRMReports: crm})
	result, err := exec.Run(context.Background(), testDefinition("pipeline"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Totals["volume"] != 75000.0 {
		t.Errorf("expected volume 75000, got %v", result.Totals["volume"])
	}
	if result.Totals["deal_count"] != 15 {
		t.Errorf("expected deals 15, got %v", result.Totals["deal_count"])
	}
}

func TestRun_Pipeline_NilReport(t *testing.T) {
	crm := &mockCRM{} // nil pipeline
	exec := newTestExecutor(Deps{CRMReports: crm})
	result, err := exec.Run(context.Background(), testDefinition("pipeline"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 0 {
		t.Errorf("expected empty rows, got %d", len(result.Rows))
	}
}

// ============================================================================
// Tests: conversion
// ============================================================================

func TestRun_Conversion_HappyPath(t *testing.T) {
	crm := &mockCRM{
		conversion: &CRMConversionReport{
			Stages: []CRMConversionStage{
				{FromStage: "Lead", ToStage: "Qualified", EnteredCnt: 100, ConvertedCnt: 40, ConvertedRate: 0.4},
			},
		},
	}
	exec := newTestExecutor(Deps{CRMReports: crm})
	result, err := exec.Run(context.Background(), testDefinition("conversion"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(result.Rows))
	}
}

// ============================================================================
// Tests: activity_by_user
// ============================================================================

func TestRun_ActivityByUser_HappyPath(t *testing.T) {
	uid := uuid.New()
	crm := &mockCRM{
		activity: &CRMActivityReport{
			Users: []CRMActivityUser{
				{UserID: uid, UserName: "Luke", Calls: 20, Emails: 30, Notes: 5, Meetings: 4, TotalEvts: 59},
			},
		},
	}
	exec := newTestExecutor(Deps{CRMReports: crm})
	result, err := exec.Run(context.Background(), testDefinition("activity_by_user"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Totals["total"] != 59 {
		t.Errorf("expected total 59, got %v", result.Totals["total"])
	}
	if result.Rows[0]["user_id"] != uid.String() {
		t.Errorf("unexpected user_id: %v", result.Rows[0]["user_id"])
	}
}

// ============================================================================
// Tests: helpdesk_sla
// ============================================================================

func TestRun_HelpdeskSLA_HappyPath(t *testing.T) {
	hd := &mockHelpdesk{
		report: &HelpdeskSLAReport{
			Queues: []HelpdeskSLARow{
				{Queue: "general", TicketsTotal: 100, FirstResponsePct: 0.92, ResolutionPct: 0.85,
					AvgFirstRespMin: 12, AvgResolveMin: 240},
			},
		},
	}
	exec := newTestExecutor(Deps{Helpdesk: hd})
	result, err := exec.Run(context.Background(), testDefinition("helpdesk_sla"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Series) != 2 {
		t.Errorf("expected 2 series, got %d", len(result.Series))
	}
	if len(result.Rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(result.Rows))
	}
}

// ============================================================================
// Tests: stock_warnings
// ============================================================================

func TestRun_StockWarnings_HappyPath(t *testing.T) {
	inv := &mockInventar{
		warnings: []StockWarning{
			{SKU: "SKU-1", Name: "Widget", OnHand: 3, MinQuantity: 10, Location: "A1"},
		},
	}
	exec := newTestExecutor(Deps{Inventar: inv})
	result, err := exec.Run(context.Background(), testDefinition("stock_warnings"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta.RowCount != 1 {
		t.Errorf("expected 1 warning, got %d", result.Meta.RowCount)
	}
}

func TestRun_StockWarnings_NoInventar(t *testing.T) {
	exec := newTestExecutor(Deps{})
	result, err := exec.Run(context.Background(), testDefinition("stock_warnings"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta.Warning == "" {
		t.Error("expected warning when inventar missing")
	}
}

// ============================================================================
// Tests: datev_bwa
// ============================================================================

func TestRun_DatevBWA_HappyPath(t *testing.T) {
	datev := &mockDatev{
		data: &BWAData{
			Period: "2026-04",
			Entries: []BWAEntry{
				{Code: "8400", Label: "Umsatzerloese", Amount: 25000},
			},
			Totals: map[string]float64{"net": 25000},
		},
	}
	exec := newTestExecutor(Deps{DatevBridge: datev})
	result, err := exec.Run(context.Background(), testDefinition("datev_bwa"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Totals["period"] != "2026-04" {
		t.Errorf("expected period 2026-04, got %v", result.Totals["period"])
	}
	if result.Totals["net"] != 25000.0 {
		t.Errorf("expected net total 25000, got %v", result.Totals["net"])
	}
}

func TestRun_DatevBWA_NoBridge(t *testing.T) {
	exec := newTestExecutor(Deps{})
	result, err := exec.Run(context.Background(), testDefinition("datev_bwa"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta.Warning != "downstream_not_available" {
		t.Errorf("expected warning, got %q", result.Meta.Warning)
	}
}

// ============================================================================
// Tests: DashboardKPIs
// ============================================================================

func kpiByID(kpis []berichte.KPI, id string) *berichte.KPI {
	for i := range kpis {
		if kpis[i].ID == id {
			return &kpis[i]
		}
	}
	return nil
}

func TestDashboardKPIs_AllModulesPopulated(t *testing.T) {
	repo := &mockKPI{
		snapshots: []*KPISnapshot{
			{Revenue: decimal.NewFromInt(5000), PipelineVolume: decimal.NewFromInt(10000), OpenTickets: 12, StockWarnings: 3},
			{Revenue: decimal.NewFromInt(4000), PipelineVolume: decimal.NewFromInt(8000), OpenTickets: 15, StockWarnings: 3},
		},
	}

	exec := newTestExecutor(Deps{KPI: repo})
	kpis, err := exec.DashboardKPIs(context.Background(), testTenant, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kpis) != 4 {
		t.Fatalf("expected 4 KPIs, got %d", len(kpis))
	}

	revenue := kpiByID(kpis, "revenue_current_month")
	if revenue == nil || revenue.Value != "5000.00" || revenue.Unit != "EUR" {
		t.Fatalf("unexpected revenue KPI: %+v", revenue)
	}
	if revenue.ChangePercent == nil || *revenue.ChangePercent != 25 {
		t.Errorf("expected revenue change +25%%, got %v", revenue.ChangePercent)
	}

	tickets := kpiByID(kpis, "open_tickets")
	if tickets == nil || tickets.Value != "12" {
		t.Fatalf("unexpected tickets KPI: %+v", tickets)
	}
	if tickets.ChangePercent == nil || *tickets.ChangePercent != -20 {
		t.Errorf("expected tickets change -20%%, got %v", tickets.ChangePercent)
	}

	// stock_warnings has no history, so it must not claim a trend.
	stock := kpiByID(kpis, "stock_warnings_count")
	if stock == nil || stock.Value != "3" {
		t.Fatalf("unexpected stock KPI: %+v", stock)
	}
	if stock.ChangePercent != nil {
		t.Errorf("expected no change_percent for stock warnings, got %v", *stock.ChangePercent)
	}
}

func TestDashboardKPIs_ComparesAgainstSameSpanOfPreviousMonth(t *testing.T) {
	repo := &mockKPI{snapshots: []*KPISnapshot{{}, {}}}
	exec := newTestExecutor(Deps{KPI: repo}) // clock: 2026-04-18 12:00 UTC

	if _, err := exec.DashboardKPIs(context.Background(), testTenant, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.calls) != 2 {
		t.Fatalf("expected 2 snapshot calls, got %d", len(repo.calls))
	}

	cur, prev := repo.calls[0], repo.calls[1]
	if cur.tenantID != testTenant || prev.tenantID != testTenant {
		t.Errorf("snapshots must stay tenant-scoped, got %v / %v", cur.tenantID, prev.tenantID)
	}
	wantCurFrom := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !cur.from.Equal(wantCurFrom) {
		t.Errorf("current period starts %v, want %v", cur.from, wantCurFrom)
	}
	wantPrevFrom := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	wantPrevTo := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	if !prev.from.Equal(wantPrevFrom) || !prev.to.Equal(wantPrevTo) {
		t.Errorf("previous period %v..%v, want %v..%v", prev.from, prev.to, wantPrevFrom, wantPrevTo)
	}
}

// A 31-day month must not let the comparison window bleed into the current one.
func TestDashboardKPIs_PreviousPeriodClampedToMonthStart(t *testing.T) {
	repo := &mockKPI{snapshots: []*KPISnapshot{{}, {}}}
	exec := newTestExecutor(Deps{
		KPI:   repo,
		Clock: &fixedClock{t: time.Date(2026, 3, 31, 23, 0, 0, 0, time.UTC)},
	})

	if _, err := exec.DashboardKPIs(context.Background(), testTenant, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	marchStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if got := repo.calls[1].to; got.After(marchStart) {
		t.Errorf("previous period ends %v, must not pass %v", got, marchStart)
	}
}

func TestDashboardKPIs_ZeroBaselineHasNoChange(t *testing.T) {
	repo := &mockKPI{
		snapshots: []*KPISnapshot{
			{Revenue: decimal.NewFromInt(500)},
			{Revenue: decimal.Zero},
		},
	}
	exec := newTestExecutor(Deps{KPI: repo})

	kpis, err := exec.DashboardKPIs(context.Background(), testTenant, []string{"finanzen"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := kpis[0].ChangePercent; got != nil {
		t.Errorf("expected no change against a zero baseline, got %v", *got)
	}
}

func TestDashboardKPIs_ModuleFilter(t *testing.T) {
	repo := &mockKPI{
		snapshots: []*KPISnapshot{
			{Revenue: decimal.NewFromInt(100), PipelineVolume: decimal.NewFromInt(200)},
			{Revenue: decimal.NewFromInt(100), PipelineVolume: decimal.NewFromInt(200)},
		},
	}
	exec := newTestExecutor(Deps{KPI: repo})
	kpis, err := exec.DashboardKPIs(context.Background(), testTenant, []string{"finanzen"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kpis) != 1 || kpis[0].ModuleID != "finanzen" {
		t.Errorf("expected only finanzen KPI, got %+v", kpis)
	}
}

func TestDashboardKPIs_MissingRepoNoCrash(t *testing.T) {
	exec := newTestExecutor(Deps{})
	kpis, err := exec.DashboardKPIs(context.Background(), testTenant, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kpis == nil {
		t.Fatal("expected an empty slice, not nil — the wire shape must stay []")
	}
	if len(kpis) != 0 {
		t.Errorf("expected 0 KPIs without a KPI repo, got %d", len(kpis))
	}
}

func TestDashboardKPIs_CurrentSnapshotErrorFails(t *testing.T) {
	repo := &mockKPI{errs: []error{errors.New("boom")}}
	exec := newTestExecutor(Deps{KPI: repo})

	if _, err := exec.DashboardKPIs(context.Background(), testTenant, nil); err == nil {
		t.Fatal("expected an error when the current snapshot fails")
	}
}

// Losing only the comparison must not cost the user the numbers themselves.
func TestDashboardKPIs_PreviousSnapshotErrorStillServesKPIs(t *testing.T) {
	repo := &mockKPI{
		snapshots: []*KPISnapshot{{Revenue: decimal.NewFromInt(700)}, nil},
		errs:      []error{nil, errors.New("boom")},
	}
	exec := newTestExecutor(Deps{KPI: repo})

	kpis, err := exec.DashboardKPIs(context.Background(), testTenant, []string{"finanzen"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kpis) != 1 || kpis[0].Value != "700.00" {
		t.Fatalf("unexpected KPIs: %+v", kpis)
	}
	if kpis[0].ChangePercent != nil {
		t.Errorf("expected no change_percent without a previous snapshot, got %v", *kpis[0].ChangePercent)
	}
}

// Each history-capable module KPI carries its own metric out of the shared
// series call — a wiring bug here would show, say, ticket counts under the
// revenue KPI's series.
func TestDashboardKPIs_SeriesPerModule(t *testing.T) {
	period := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockKPI{
		snapshots: []*KPISnapshot{{}, {}},
		series: []KPISeriesPoint{
			{PeriodStart: period, Revenue: decimal.NewFromInt(1000), PipelineVolume: decimal.NewFromInt(2000), OpenTickets: 7},
		},
	}
	exec := newTestExecutor(Deps{KPI: repo})

	kpis, err := exec.DashboardKPIs(context.Background(), testTenant, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	revenue := kpiByID(kpis, "revenue_current_month")
	if len(revenue.Series) != 1 || revenue.Series[0].Value != "1000.00" || !revenue.Series[0].PeriodStart.Equal(period) {
		t.Fatalf("unexpected revenue series: %+v", revenue.Series)
	}
	pipeline := kpiByID(kpis, "pipeline_volume")
	if len(pipeline.Series) != 1 || pipeline.Series[0].Value != "2000.00" {
		t.Fatalf("unexpected pipeline series: %+v", pipeline.Series)
	}
	tickets := kpiByID(kpis, "open_tickets")
	if len(tickets.Series) != 1 || tickets.Series[0].Value != "7" {
		t.Fatalf("unexpected tickets series: %+v", tickets.Series)
	}

	// stock_warnings has no history to bucket — see the executor's doc comment.
	stock := kpiByID(kpis, "stock_warnings_count")
	if stock.Series != nil {
		t.Errorf("expected no series for stock warnings, got %+v", stock.Series)
	}

	if len(repo.seriesCalls) != 1 {
		t.Fatalf("expected exactly 1 series call (one query, not one per KPI), got %d", len(repo.seriesCalls))
	}
	if got := repo.seriesCalls[0].periods; got != 8 {
		t.Errorf("expected 8 periods requested, got %d", got)
	}
}

// A failed series call must not cost the user the current KPI values —
// same graceful-degradation stance as a failed previous-period snapshot.
func TestDashboardKPIs_SeriesErrorStillServesKPIs(t *testing.T) {
	repo := &mockKPI{
		snapshots: []*KPISnapshot{{Revenue: decimal.NewFromInt(700)}, {}},
		seriesErr: errors.New("boom"),
	}
	exec := newTestExecutor(Deps{KPI: repo})

	kpis, err := exec.DashboardKPIs(context.Background(), testTenant, []string{"finanzen"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kpis) != 1 || kpis[0].Value != "700.00" {
		t.Fatalf("unexpected KPIs: %+v", kpis)
	}
	if kpis[0].Series != nil {
		t.Errorf("expected no series after a failed series call, got %+v", kpis[0].Series)
	}
}

// ============================================================================
// Tests: periodRange
// ============================================================================

func TestPeriodRange(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	exec := newTestExecutor(Deps{Clock: &fixedClock{t: now}})

	cases := []struct {
		period   string
		wantFrom time.Time
	}{
		{"last_7_days", now.AddDate(0, 0, -7)},
		{"last_30_days", now.AddDate(0, 0, -30)},
		{"last_90_days", now.AddDate(0, 0, -90)},
		{"last_12_months", now.AddDate(-1, 0, 0)},
		{"current_month", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		{"", now.AddDate(0, 0, -30)},
	}

	for _, c := range cases {
		from, to := exec.periodRange(c.period)
		if !from.Equal(c.wantFrom) {
			t.Errorf("period %q: expected from=%v, got %v", c.period, c.wantFrom, from)
		}
		if !to.Equal(now) {
			t.Errorf("period %q: expected to=%v, got %v", c.period, now, to)
		}
	}
}

// ============================================================================
// Tests: cross
// ============================================================================

func crossDefinition(joinOn string, sources ...map[string]any) *berichte.Definition {
	raw, _ := json.Marshal(map[string]any{
		"kind":    "cross",
		"period":  "last_30_days",
		"join_on": joinOn,
		"sources": sources,
	})
	return &berichte.Definition{
		ID:            uuid.New(),
		TenantID:      testTenant,
		Name:          "Cross",
		Module:        "cross",
		Kind:          "custom",
		QueryConfig:   raw,
		DefaultFormat: "pdf",
	}
}

// crossCRM wires a pipeline report and a conversion report that share stage
// names, which is the realistic join: pipeline calls the dimension "stage",
// conversion calls it "from_stage".
func crossCRM() *mockCRM {
	return &mockCRM{
		pipeline: &CRMPipelineReport{Stages: []CRMPipelineStage{
			{Stage: "lead", DealCnt: 10, Volume: 5000, Currency: "EUR"},
			{Stage: "qualified", DealCnt: 4, Volume: 9000, Currency: "EUR"},
			{Stage: "won", DealCnt: 2, Volume: 12000, Currency: "EUR"},
		}},
		conversion: &CRMConversionReport{Stages: []CRMConversionStage{
			{FromStage: "lead", ToStage: "qualified", EnteredCnt: 10, ConvertedCnt: 4, ConvertedRate: 40},
			{FromStage: "qualified", ToStage: "won", EnteredCnt: 4, ConvertedCnt: 2, ConvertedRate: 50},
			{FromStage: "lost", ToStage: "archived", EnteredCnt: 3, ConvertedCnt: 0, ConvertedRate: 0},
		}},
	}
}

func TestRun_Cross_JoinsOnPerSourceKeys(t *testing.T) {
	exec := newTestExecutor(Deps{CRMReports: crossCRM()})
	def := crossDefinition("stage",
		map[string]any{"kind": "pipeline", "key": "stage"},
		map[string]any{"kind": "conversion", "key": "from_stage"},
	)

	result, err := exec.Run(context.Background(), def, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "won" has no conversion row, "lost" has no pipeline row: inner join keeps
	// the two matching stages and reports the drops instead of hiding them.
	if result.Meta.RowCount != 2 || len(result.Rows) != 2 {
		t.Fatalf("expected 2 joined rows, got %d: %+v", result.Meta.RowCount, result.Rows)
	}
	if result.Totals["cross_unmatched_pipeline"] != 1 {
		t.Errorf("expected 1 unmatched pipeline row, got %v", result.Totals["cross_unmatched_pipeline"])
	}
	if result.Totals["cross_unmatched_conversion"] != 1 {
		t.Errorf("expected 1 unmatched conversion row, got %v", result.Totals["cross_unmatched_conversion"])
	}

	// Row order follows the first source.
	if got := result.Rows[0]["stage"]; got != "lead" {
		t.Errorf("expected first row stage=lead, got %v", got)
	}
	if got := result.Rows[0]["pipeline_volume"]; got != 5000.0 {
		t.Errorf("expected pipeline_volume=5000, got %v", got)
	}
	if got := result.Rows[0]["conversion_rate"]; got != 40.0 {
		t.Errorf("expected conversion_rate=40, got %v", got)
	}
	// Both sources' keys are consumed by the join column, not re-emitted.
	if _, ok := result.Rows[0]["conversion_from_stage"]; ok {
		t.Error("right source key must not be re-emitted as its own column")
	}

	wantCols := []string{
		"stage",
		"pipeline_deal_count", "pipeline_volume",
		"conversion_to_stage", "conversion_entered", "conversion_converted", "conversion_rate",
	}
	if len(result.Columns) != len(wantCols) {
		t.Fatalf("expected %d columns, got %d: %+v", len(wantCols), len(result.Columns), result.Columns)
	}
	for i, want := range wantCols {
		if result.Columns[i].Name != want {
			t.Errorf("column %d: expected %q, got %q", i, want, result.Columns[i].Name)
		}
	}

	// Source totals survive under their alias, so the joined table still shows
	// what each side summed up.
	if result.Totals["pipeline_volume"] != 26000.0 {
		t.Errorf("expected aliased pipeline total 26000, got %v", result.Totals["pipeline_volume"])
	}
	if len(result.Series) != 0 {
		t.Errorf("cross must not guess a chart series, got %+v", result.Series)
	}
	if result.Meta.DefinitionID != def.ID {
		t.Errorf("expected definition id %v, got %v", def.ID, result.Meta.DefinitionID)
	}
}

func TestRun_Cross_RepeatedKeyProducesEveryCombination(t *testing.T) {
	crm := &mockCRM{
		pipeline: &CRMPipelineReport{Stages: []CRMPipelineStage{
			{Stage: "lead", DealCnt: 1, Volume: 100},
			{Stage: "lead", DealCnt: 2, Volume: 200},
		}},
		conversion: &CRMConversionReport{Stages: []CRMConversionStage{
			{FromStage: "lead", ToStage: "qualified", ConvertedRate: 10},
			{FromStage: "lead", ToStage: "lost", ConvertedRate: 90},
		}},
	}
	exec := newTestExecutor(Deps{CRMReports: crm})
	result, err := exec.Run(context.Background(), crossDefinition("stage",
		map[string]any{"kind": "pipeline", "key": "stage"},
		map[string]any{"kind": "conversion", "key": "from_stage"},
	), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 4 {
		t.Fatalf("expected 2x2 = 4 rows, got %d", len(result.Rows))
	}
	if result.Totals["cross_unmatched_pipeline"] != 0 {
		t.Errorf("expected no unmatched rows, got %v", result.Totals["cross_unmatched_pipeline"])
	}
}

// TestRun_Cross_SourceRowCapDropsRowsBeyondLimit pins the PER-SOURCE cap on
// its own. The one matching key sits beyond maxCrossSourceRows on the capped
// side, so the join can only find it if that side was not cut — which makes
// the assertion fail the moment the cap stops working. The output cap cannot
// stand in for it here: without the source cap this join yields exactly one
// row, far below any output limit.
func TestRun_Cross_SourceRowCapDropsRowsBeyondLimit(t *testing.T) {
	const beyondCap = maxCrossSourceRows + 100
	sharedKey := "stage-" + strconv.Itoa(beyondCap-1)

	oversizedStages := func() []CRMPipelineStage {
		out := make([]CRMPipelineStage, 0, beyondCap)
		for i := range beyondCap {
			out = append(out, CRMPipelineStage{Stage: "stage-" + strconv.Itoa(i), DealCnt: i})
		}
		return out
	}
	oversizedConversions := func() []CRMConversionStage {
		out := make([]CRMConversionStage, 0, beyondCap)
		for i := range beyondCap {
			out = append(out, CRMConversionStage{FromStage: "stage-" + strconv.Itoa(i), ConvertedRate: float64(i)})
		}
		return out
	}

	cases := map[string]*mockCRM{
		"left source beyond the cap": {
			pipeline:   &CRMPipelineReport{Stages: oversizedStages()},
			conversion: &CRMConversionReport{Stages: []CRMConversionStage{{FromStage: sharedKey}}},
		},
		"right source beyond the cap": {
			pipeline:   &CRMPipelineReport{Stages: []CRMPipelineStage{{Stage: sharedKey}}},
			conversion: &CRMConversionReport{Stages: oversizedConversions()},
		},
	}

	for name, crm := range cases {
		t.Run(name, func(t *testing.T) {
			exec := newTestExecutor(Deps{CRMReports: crm})
			result, err := exec.Run(context.Background(), crossDefinition("stage",
				map[string]any{"kind": "pipeline", "key": "stage"},
				map[string]any{"kind": "conversion", "key": "from_stage"},
			), nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Rows) != 0 {
				t.Errorf("the matching key sits beyond the per-source cap, expected 0 rows, got %d", len(result.Rows))
			}
			if result.Meta.Warning != warningCrossTruncated {
				t.Errorf("a truncated source must say so, got warning %q", result.Meta.Warning)
			}
		})
	}
}

// TestRun_Cross_OutputRowCapBoundsCartesianProduct pins the SECOND cap: both
// sources sit exactly at the per-source limit (so that cap stays silent), but
// they share a single join value, which without an output cap would expand to
// maxCrossSourceRows^2 rows from one request.
func TestRun_Cross_OutputRowCapBoundsCartesianProduct(t *testing.T) {
	stages := make([]CRMPipelineStage, 0, maxCrossSourceRows)
	conversions := make([]CRMConversionStage, 0, maxCrossSourceRows)
	for i := range maxCrossSourceRows {
		stages = append(stages, CRMPipelineStage{Stage: "same", DealCnt: i})
		conversions = append(conversions, CRMConversionStage{FromStage: "same", ConvertedRate: float64(i)})
	}
	crm := &mockCRM{
		pipeline:   &CRMPipelineReport{Stages: stages},
		conversion: &CRMConversionReport{Stages: conversions},
	}

	exec := newTestExecutor(Deps{CRMReports: crm})
	result, err := exec.Run(context.Background(), crossDefinition("stage",
		map[string]any{"kind": "pipeline", "key": "stage"},
		map[string]any{"kind": "conversion", "key": "from_stage"},
	), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != maxCrossSourceRows {
		t.Errorf("expected the join capped at %d rows, got %d", maxCrossSourceRows, len(result.Rows))
	}
	if result.Meta.Warning != warningCrossTruncated {
		t.Errorf("a truncated result must say so, got warning %q", result.Meta.Warning)
	}
}

func TestRun_Cross_ExactFitIsNotFlaggedTruncated(t *testing.T) {
	stages := make([]CRMPipelineStage, 0, maxCrossSourceRows)
	conversions := make([]CRMConversionStage, 0, maxCrossSourceRows)
	for i := range maxCrossSourceRows {
		key := "stage-" + strconv.Itoa(i)
		stages = append(stages, CRMPipelineStage{Stage: key})
		conversions = append(conversions, CRMConversionStage{FromStage: key})
	}
	crm := &mockCRM{
		pipeline:   &CRMPipelineReport{Stages: stages},
		conversion: &CRMConversionReport{Stages: conversions},
	}

	exec := newTestExecutor(Deps{CRMReports: crm})
	result, err := exec.Run(context.Background(), crossDefinition("stage",
		map[string]any{"kind": "pipeline", "key": "stage"},
		map[string]any{"kind": "conversion", "key": "from_stage"},
	), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != maxCrossSourceRows {
		t.Fatalf("expected %d rows, got %d", maxCrossSourceRows, len(result.Rows))
	}
	if result.Meta.Warning != "" {
		t.Errorf("a join that fits exactly lost nothing, got warning %q", result.Meta.Warning)
	}
}

func TestRun_Cross_SourcePeriodOverride(t *testing.T) {
	crm := crossCRM()
	clock := &fixedClock{t: time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)}
	exec := newTestExecutor(Deps{CRMReports: crm, Clock: clock})

	// Report period is last_30_days; the conversion source overrides it.
	_, err := exec.Run(context.Background(), crossDefinition("stage",
		map[string]any{"kind": "pipeline", "key": "stage"},
		map[string]any{"kind": "conversion", "key": "from_stage", "period": "last_90_days"},
	), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantFrom := clock.t.AddDate(0, 0, -90)
	if !crm.conversionFrom.Equal(wantFrom) {
		t.Errorf("expected the source period to reach the downstream (from=%v), got %v", wantFrom, crm.conversionFrom)
	}
	if !crm.conversionTo.Equal(clock.t) {
		t.Errorf("expected the window to end at now (%v), got %v", clock.t, crm.conversionTo)
	}
}

func TestRun_Cross_DownstreamMissingDegradesInsteadOfFailing(t *testing.T) {
	// No CRM repo wired: both sources come back empty with the downstream
	// warning. That must stay a degraded result, not a config error.
	exec := newTestExecutor(Deps{})
	result, err := exec.Run(context.Background(), crossDefinition("stage",
		map[string]any{"kind": "pipeline", "key": "stage"},
		map[string]any{"kind": "conversion", "key": "from_stage"},
	), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta.Warning != warningDownstreamMissing {
		t.Errorf("expected %q, got %q", warningDownstreamMissing, result.Meta.Warning)
	}
	if len(result.Rows) != 0 {
		t.Errorf("expected no rows, got %d", len(result.Rows))
	}
}

func TestRun_Cross_SourceErrorPropagates(t *testing.T) {
	crm := crossCRM()
	crm.err = errors.New("db down")
	exec := newTestExecutor(Deps{CRMReports: crm})
	_, err := exec.Run(context.Background(), crossDefinition("stage",
		map[string]any{"kind": "pipeline", "key": "stage"},
		map[string]any{"kind": "conversion", "key": "from_stage"},
	), nil)
	if err == nil {
		t.Error("expected the downstream error to surface")
	}
}

func TestRun_Cross_InvalidConfigs(t *testing.T) {
	cases := []struct {
		name    string
		joinOn  string
		sources []map[string]any
	}{
		{
			name:    "single source",
			joinOn:  "stage",
			sources: []map[string]any{{"kind": "pipeline", "key": "stage"}},
		},
		{
			name:   "three sources",
			joinOn: "stage",
			sources: []map[string]any{
				{"kind": "pipeline", "key": "stage"},
				{"kind": "conversion", "key": "from_stage"},
				{"kind": "pipeline", "key": "stage", "alias": "p2"},
			},
		},
		{
			name:   "missing join_on",
			joinOn: "",
			sources: []map[string]any{
				{"kind": "pipeline", "key": "stage"},
				{"kind": "conversion", "key": "from_stage"},
			},
		},
		{
			name:   "colliding aliases",
			joinOn: "stage",
			sources: []map[string]any{
				{"kind": "pipeline", "key": "stage"},
				{"kind": "conversion", "key": "from_stage", "alias": "pipeline"},
			},
		},
		{
			name:   "unknown source kind",
			joinOn: "stage",
			sources: []map[string]any{
				{"kind": "pipeline", "key": "stage"},
				{"kind": "there_is_no_such_report", "key": "stage"},
			},
		},
		{
			name:   "nested cross",
			joinOn: "stage",
			sources: []map[string]any{
				{"kind": "pipeline", "key": "stage"},
				{"kind": "cross", "key": "stage"},
			},
		},
		{
			name:   "join key is not a column of the source",
			joinOn: "stage",
			sources: []map[string]any{
				{"kind": "pipeline", "key": "stage"},
				{"kind": "conversion", "key": "not_a_column"},
			},
		},
		{
			name:   "empty source kind",
			joinOn: "stage",
			sources: []map[string]any{
				{"kind": "pipeline", "key": "stage"},
				{"kind": "", "key": "stage"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			exec := newTestExecutor(Deps{CRMReports: crossCRM()})
			_, err := exec.Run(context.Background(), crossDefinition(c.joinOn, c.sources...), nil)
			if !errors.Is(err, berichte.ErrInvalidQueryConfig) {
				t.Errorf("expected ErrInvalidQueryConfig, got %v", err)
			}
		})
	}
}

func TestRun_Cross_KeyDefaultsToJoinOn(t *testing.T) {
	// Both sources happen to name the dimension the same way: omitting `key`
	// must fall back to join_on rather than failing.
	crm := &mockCRM{
		pipeline: &CRMPipelineReport{Stages: []CRMPipelineStage{{Stage: "lead", DealCnt: 3, Volume: 300}}},
	}
	exec := newTestExecutor(Deps{CRMReports: crm})
	result, err := exec.Run(context.Background(), crossDefinition("stage",
		map[string]any{"kind": "pipeline", "alias": "left"},
		map[string]any{"kind": "pipeline", "alias": "right"},
	), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result.Rows))
	}
	if result.Rows[0]["left_deal_count"] != 3 || result.Rows[0]["right_deal_count"] != 3 {
		t.Errorf("expected both sides under their alias, got %+v", result.Rows[0])
	}
}

// ============================================================================
// Tests: executor implements berichte.Executor
// ============================================================================

func TestExecutor_ImplementsBerichteExecutor(t *testing.T) {
	var _ berichte.Executor = New(Deps{})
}
