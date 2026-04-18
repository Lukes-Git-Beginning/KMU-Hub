// Package executor implements the aggregation engine behind the berichte
// service. Each `query_config.kind` maps to a private method that delegates to
// a downstream domain repository and reshapes the result into a
// berichte.ReportResult.
//
// The downstream repositories are injected as narrow interfaces (CRMReportsRepo,
// FinanceRepo, HelpdeskRepo, InventarRepo, DatevBridge) so the executor can be
// tested without pulling in the full service graph. The cmd/berichte binary
// wires the concrete implementations at startup.
//
// Downstream dependencies may be nil; the executor returns an empty ReportResult
// with a "downstream_not_available" warning rather than panicking, which lets
// the gateway serve the module even when the upstream service is being
// rebooted.
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

// ============================================================================
// Downstream interfaces
// ============================================================================

// MonthlyRevenue is one row from the finance module's monthly revenue report.
type MonthlyRevenue struct {
	Month      time.Time
	Revenue    float64
	InvoiceCnt int
}

// InvoiceSummary is one row from the finance module's open-invoice list.
type InvoiceSummary struct {
	ID         uuid.UUID
	Number     string
	CustomerID uuid.UUID
	Customer   string
	Status     string
	Amount     float64
	Currency   string
	DueDate    *time.Time
	OverdueDay int
}

// FinanceRepo covers the finance-side aggregations berichte consumes.
type FinanceRepo interface {
	GetRevenueByMonth(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]MonthlyRevenue, error)
	ListInvoicesByStatus(ctx context.Context, tenantID uuid.UUID, statuses []string) ([]InvoiceSummary, error)
}

// CRMPipelineReport is one row per stage in a pipeline overview.
type CRMPipelineReport struct {
	Stages []CRMPipelineStage
}

// CRMPipelineStage describes one stage within a pipeline report.
type CRMPipelineStage struct {
	Stage    string
	DealCnt  int
	Volume   float64
	Currency string
}

// CRMConversionReport captures stage-to-stage funnel metrics.
type CRMConversionReport struct {
	Stages []CRMConversionStage
}

// CRMConversionStage describes one step in a conversion funnel.
type CRMConversionStage struct {
	FromStage     string
	ToStage       string
	EnteredCnt    int
	ConvertedCnt  int
	ConvertedRate float64
}

// CRMActivityReport captures activity counts per sales user.
type CRMActivityReport struct {
	Users []CRMActivityUser
}

// CRMActivityUser describes one row of the activity-by-user report.
type CRMActivityUser struct {
	UserID    uuid.UUID
	UserName  string
	Calls     int
	Emails    int
	Notes     int
	Meetings  int
	TotalEvts int
}

// CRMReportsRepo covers the crm/report.Repository subset berichte consumes.
type CRMReportsRepo interface {
	GetPipelineReport(ctx context.Context, tenantID uuid.UUID) (*CRMPipelineReport, error)
	GetConversionReport(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*CRMConversionReport, error)
	GetActivityReport(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*CRMActivityReport, error)
}

// HelpdeskSLARow is one SLA row per queue.
type HelpdeskSLARow struct {
	Queue          string
	TicketsTotal   int
	FirstResponse  int // within SLA
	Resolution     int // within SLA
	FirstResponsePct float64
	ResolutionPct    float64
	AvgFirstRespMin  float64
	AvgResolveMin    float64
}

// HelpdeskSLAReport bundles SLA rows.
type HelpdeskSLAReport struct {
	Queues []HelpdeskSLARow
}

// HelpdeskRepo covers the helpdesk module's SLA aggregation for berichte.
type HelpdeskRepo interface {
	GetSLAReport(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*HelpdeskSLAReport, error)
}

// StockWarning is one row of the inventory low-stock report.
type StockWarning struct {
	SKU         string
	Name        string
	OnHand      int
	MinQuantity int
	Location    string
}

// InventarRepo covers the inventar module's stock-warning aggregation.
// The inventar module lands in sprint 2; nil is accepted here.
type InventarRepo interface {
	GetStockWarnings(ctx context.Context, tenantID uuid.UUID) ([]StockWarning, error)
}

// BWAEntry is one line of the DATEV BWA bridge.
type BWAEntry struct {
	Code   string
	Label  string
	Amount float64
}

// BWAData bundles BWA entries for a period.
type BWAData struct {
	Period  string
	Entries []BWAEntry
	Totals  map[string]float64
}

// DatevBridge exposes the DATEV-export bridge needed for BWA reports.
type DatevBridge interface {
	GetBWAData(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*BWAData, error)
}

// ============================================================================
// Executor
// ============================================================================

// Clock abstracts time.Now for deterministic tests.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// Deps bundles downstream dependencies. Any of them may be nil; the executor
// will return an empty result with a "downstream_not_available" warning for the
// affected kind.
type Deps struct {
	Finance     FinanceRepo
	CRMReports  CRMReportsRepo
	Helpdesk    HelpdeskRepo
	Inventar    InventarRepo
	DatevBridge DatevBridge
	Clock       Clock
}

// Executor runs aggregations based on a Definition's query_config.kind.
type Executor struct {
	finance   FinanceRepo
	crm       CRMReportsRepo
	helpdesk  HelpdeskRepo
	inventar  InventarRepo
	datev     DatevBridge
	clock     Clock
}

// New creates a new executor. The caller owns the downstream lifetimes.
func New(deps Deps) *Executor {
	clk := deps.Clock
	if clk == nil {
		clk = realClock{}
	}
	return &Executor{
		finance:  deps.Finance,
		crm:      deps.CRMReports,
		helpdesk: deps.Helpdesk,
		inventar: deps.Inventar,
		datev:    deps.DatevBridge,
		clock:    clk,
	}
}

// ============================================================================
// queryConfig parsing
// ============================================================================

type queryConfig struct {
	Kind   string `json:"kind"`
	Period string `json:"period"` // last_7_days | last_30_days | last_90_days | last_12_months | current_month | ""
}

func parseQueryConfig(raw []byte) (*queryConfig, error) {
	if len(raw) == 0 {
		return nil, berichte.ErrInvalidQueryConfig
	}
	var cfg queryConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse query config: %w", err)
	}
	if cfg.Kind == "" {
		return nil, berichte.ErrInvalidQueryConfig
	}
	return &cfg, nil
}

func (e *Executor) periodRange(period string) (time.Time, time.Time) {
	now := e.clock.Now()
	switch period {
	case "last_7_days":
		return now.AddDate(0, 0, -7), now
	case "last_30_days":
		return now.AddDate(0, 0, -30), now
	case "last_90_days":
		return now.AddDate(0, 0, -90), now
	case "last_12_months":
		return now.AddDate(-1, 0, 0), now
	case "current_month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, now
	default:
		return now.AddDate(0, 0, -30), now
	}
}

// ============================================================================
// Public entry points
// ============================================================================

// Run dispatches to the appropriate kind handler. Returns
// berichte.ErrInvalidQueryConfig for unknown kinds.
func (e *Executor) Run(ctx context.Context, def *berichte.Definition, _ json.RawMessage) (*berichte.ReportResult, error) {
	cfg, err := parseQueryConfig(def.QueryConfig)
	if err != nil {
		return nil, err
	}

	switch cfg.Kind {
	case "revenue_by_month":
		return e.revenueByMonth(ctx, def, cfg)
	case "invoices_open":
		return e.invoicesOpen(ctx, def, cfg)
	case "pipeline":
		return e.pipeline(ctx, def, cfg)
	case "conversion":
		return e.conversion(ctx, def, cfg)
	case "activity_by_user":
		return e.activityByUser(ctx, def, cfg)
	case "helpdesk_sla":
		return e.helpdeskSLA(ctx, def, cfg)
	case "stock_warnings":
		return e.stockWarnings(ctx, def, cfg)
	case "datev_bwa":
		return e.datevBWA(ctx, def, cfg)
	default:
		return nil, berichte.ErrInvalidQueryConfig
	}
}

// DashboardKPIs produces a compact KPI list for the berichte dashboard. The
// initial implementation derives four cross-module KPIs from the available
// downstream repositories; missing repositories yield zero-value KPIs so the
// dashboard stays predictable.
func (e *Executor) DashboardKPIs(ctx context.Context, tenantID uuid.UUID, modules []string) ([]berichte.KPI, error) {
	active := make(map[string]bool, len(modules))
	for _, m := range modules {
		active[m] = true
	}
	// Empty slice means "all modules".
	if len(modules) == 0 {
		active = map[string]bool{"finanzen": true, "crm": true, "helpdesk": true, "inventar": true}
	}

	var out []berichte.KPI

	if active["finanzen"] && e.finance != nil {
		from, to := e.periodRange("current_month")
		rows, err := e.finance.GetRevenueByMonth(ctx, tenantID, from, to)
		if err == nil && len(rows) > 0 {
			total := 0.0
			for _, r := range rows {
				total += r.Revenue
			}
			out = append(out, berichte.KPI{
				ID:       "revenue_current_month",
				Label:    "Umsatz laufender Monat",
				Value:    fmt.Sprintf("%.2f", total),
				Unit:     "EUR",
				ModuleID: "finanzen",
			})
		}
	}

	if active["crm"] && e.crm != nil {
		rep, err := e.crm.GetPipelineReport(ctx, tenantID)
		if err == nil && rep != nil {
			volume := 0.0
			for _, s := range rep.Stages {
				volume += s.Volume
			}
			out = append(out, berichte.KPI{
				ID:       "pipeline_volume",
				Label:    "Pipeline-Volumen",
				Value:    fmt.Sprintf("%.2f", volume),
				Unit:     "EUR",
				ModuleID: "crm",
			})
		}
	}

	if active["helpdesk"] && e.helpdesk != nil {
		from, to := e.periodRange("last_30_days")
		rep, err := e.helpdesk.GetSLAReport(ctx, tenantID, from, to)
		if err == nil && rep != nil && len(rep.Queues) > 0 {
			total := 0.0
			for _, q := range rep.Queues {
				total += q.FirstResponsePct
			}
			avg := total / float64(len(rep.Queues))
			out = append(out, berichte.KPI{
				ID:       "helpdesk_first_response",
				Label:    "Helpdesk Ersantwort-SLA",
				Value:    fmt.Sprintf("%.1f", avg),
				Unit:     "%",
				ModuleID: "helpdesk",
			})
		}
	}

	if active["inventar"] && e.inventar != nil {
		warnings, err := e.inventar.GetStockWarnings(ctx, tenantID)
		if err == nil {
			out = append(out, berichte.KPI{
				ID:       "stock_warnings_count",
				Label:    "Bestands-Warnungen",
				Value:    fmt.Sprintf("%d", len(warnings)),
				ModuleID: "inventar",
			})
		}
	}

	return out, nil
}

// ============================================================================
// Kind implementations
// ============================================================================

func (e *Executor) revenueByMonth(ctx context.Context, def *berichte.Definition, cfg *queryConfig) (*berichte.ReportResult, error) {
	if e.finance == nil {
		return emptyResult(def, "downstream_not_available"), nil
	}
	from, to := e.periodRange(cfg.Period)
	rows, err := e.finance.GetRevenueByMonth(ctx, def.TenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("revenue_by_month: %w", err)
	}

	result := &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "month", Label: "Monat", Kind: "date"},
			{Name: "revenue", Label: "Umsatz", Kind: "currency"},
			{Name: "invoice_count", Label: "Rechnungen", Kind: "number"},
		},
		Rows:   make([]map[string]any, 0, len(rows)),
		Series: []berichte.Series{{Label: "Umsatz", DataPoints: make([]berichte.DataPoint, 0, len(rows))}},
		Totals: map[string]any{},
	}
	var total float64
	var count int
	for _, r := range rows {
		label := r.Month.Format("2006-01")
		result.Rows = append(result.Rows, map[string]any{
			"month":         label,
			"revenue":       r.Revenue,
			"invoice_count": r.InvoiceCnt,
		})
		result.Series[0].DataPoints = append(result.Series[0].DataPoints,
			berichte.DataPoint{Label: label, Value: r.Revenue})
		total += r.Revenue
		count += r.InvoiceCnt
	}
	result.Totals["revenue"] = total
	result.Totals["invoice_count"] = count
	result.Meta.RowCount = len(result.Rows)
	result.Meta.DefinitionID = def.ID
	result.Meta.GeneratedAt = e.clock.Now()
	return result, nil
}

func (e *Executor) invoicesOpen(ctx context.Context, def *berichte.Definition, _ *queryConfig) (*berichte.ReportResult, error) {
	if e.finance == nil {
		return emptyResult(def, "downstream_not_available"), nil
	}
	rows, err := e.finance.ListInvoicesByStatus(ctx, def.TenantID, []string{"sent", "overdue"})
	if err != nil {
		return nil, fmt.Errorf("invoices_open: %w", err)
	}
	result := &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "number", Label: "Rechnungsnr.", Kind: "string"},
			{Name: "customer", Label: "Kunde", Kind: "string"},
			{Name: "status", Label: "Status", Kind: "string"},
			{Name: "amount", Label: "Betrag", Kind: "currency"},
			{Name: "due_date", Label: "Faellig", Kind: "date"},
			{Name: "overdue_days", Label: "Tage faellig", Kind: "number"},
		},
		Rows:   make([]map[string]any, 0, len(rows)),
		Totals: map[string]any{},
	}
	var total float64
	for _, r := range rows {
		dueStr := ""
		if r.DueDate != nil {
			dueStr = r.DueDate.Format("2006-01-02")
		}
		result.Rows = append(result.Rows, map[string]any{
			"number":       r.Number,
			"customer":     r.Customer,
			"status":       r.Status,
			"amount":       r.Amount,
			"currency":     r.Currency,
			"due_date":     dueStr,
			"overdue_days": r.OverdueDay,
		})
		total += r.Amount
	}
	result.Totals["amount"] = total
	result.Meta.RowCount = len(result.Rows)
	result.Meta.DefinitionID = def.ID
	result.Meta.GeneratedAt = e.clock.Now()
	return result, nil
}

func (e *Executor) pipeline(ctx context.Context, def *berichte.Definition, _ *queryConfig) (*berichte.ReportResult, error) {
	if e.crm == nil {
		return emptyResult(def, "downstream_not_available"), nil
	}
	rep, err := e.crm.GetPipelineReport(ctx, def.TenantID)
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	if rep == nil {
		return emptyResult(def, ""), nil
	}
	result := &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "stage", Label: "Stage", Kind: "string"},
			{Name: "deal_count", Label: "Deals", Kind: "number"},
			{Name: "volume", Label: "Volumen", Kind: "currency"},
		},
		Rows:   make([]map[string]any, 0, len(rep.Stages)),
		Series: []berichte.Series{{Label: "Volumen", DataPoints: make([]berichte.DataPoint, 0, len(rep.Stages))}},
		Totals: map[string]any{},
	}
	var totalVolume float64
	var totalDeals int
	for _, s := range rep.Stages {
		result.Rows = append(result.Rows, map[string]any{
			"stage":      s.Stage,
			"deal_count": s.DealCnt,
			"volume":     s.Volume,
			"currency":   s.Currency,
		})
		result.Series[0].DataPoints = append(result.Series[0].DataPoints,
			berichte.DataPoint{Label: s.Stage, Value: s.Volume})
		totalVolume += s.Volume
		totalDeals += s.DealCnt
	}
	result.Totals["volume"] = totalVolume
	result.Totals["deal_count"] = totalDeals
	result.Meta.RowCount = len(result.Rows)
	result.Meta.DefinitionID = def.ID
	result.Meta.GeneratedAt = e.clock.Now()
	return result, nil
}

func (e *Executor) conversion(ctx context.Context, def *berichte.Definition, cfg *queryConfig) (*berichte.ReportResult, error) {
	if e.crm == nil {
		return emptyResult(def, "downstream_not_available"), nil
	}
	from, to := e.periodRange(cfg.Period)
	rep, err := e.crm.GetConversionReport(ctx, def.TenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("conversion: %w", err)
	}
	if rep == nil {
		return emptyResult(def, ""), nil
	}
	result := &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "from_stage", Label: "Von", Kind: "string"},
			{Name: "to_stage", Label: "Nach", Kind: "string"},
			{Name: "entered", Label: "Eingang", Kind: "number"},
			{Name: "converted", Label: "Konvertiert", Kind: "number"},
			{Name: "rate", Label: "Quote", Kind: "percent"},
		},
		Rows:   make([]map[string]any, 0, len(rep.Stages)),
		Series: []berichte.Series{{Label: "Konversionsrate", DataPoints: make([]berichte.DataPoint, 0, len(rep.Stages))}},
	}
	for _, s := range rep.Stages {
		label := s.FromStage + " -> " + s.ToStage
		result.Rows = append(result.Rows, map[string]any{
			"from_stage": s.FromStage,
			"to_stage":   s.ToStage,
			"entered":    s.EnteredCnt,
			"converted":  s.ConvertedCnt,
			"rate":       s.ConvertedRate,
		})
		result.Series[0].DataPoints = append(result.Series[0].DataPoints,
			berichte.DataPoint{Label: label, Value: s.ConvertedRate})
	}
	result.Meta.RowCount = len(result.Rows)
	result.Meta.DefinitionID = def.ID
	result.Meta.GeneratedAt = e.clock.Now()
	return result, nil
}

func (e *Executor) activityByUser(ctx context.Context, def *berichte.Definition, cfg *queryConfig) (*berichte.ReportResult, error) {
	if e.crm == nil {
		return emptyResult(def, "downstream_not_available"), nil
	}
	from, to := e.periodRange(cfg.Period)
	rep, err := e.crm.GetActivityReport(ctx, def.TenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("activity_by_user: %w", err)
	}
	if rep == nil {
		return emptyResult(def, ""), nil
	}
	result := &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "user", Label: "Vertriebler", Kind: "string"},
			{Name: "calls", Label: "Calls", Kind: "number"},
			{Name: "emails", Label: "E-Mails", Kind: "number"},
			{Name: "notes", Label: "Notizen", Kind: "number"},
			{Name: "meetings", Label: "Meetings", Kind: "number"},
			{Name: "total", Label: "Gesamt", Kind: "number"},
		},
		Rows:   make([]map[string]any, 0, len(rep.Users)),
		Totals: map[string]any{},
	}
	totalEvts := 0
	for _, u := range rep.Users {
		result.Rows = append(result.Rows, map[string]any{
			"user_id":  u.UserID.String(),
			"user":     u.UserName,
			"calls":    u.Calls,
			"emails":   u.Emails,
			"notes":    u.Notes,
			"meetings": u.Meetings,
			"total":    u.TotalEvts,
		})
		totalEvts += u.TotalEvts
	}
	result.Totals["total"] = totalEvts
	result.Meta.RowCount = len(result.Rows)
	result.Meta.DefinitionID = def.ID
	result.Meta.GeneratedAt = e.clock.Now()
	return result, nil
}

func (e *Executor) helpdeskSLA(ctx context.Context, def *berichte.Definition, cfg *queryConfig) (*berichte.ReportResult, error) {
	if e.helpdesk == nil {
		return emptyResult(def, "downstream_not_available"), nil
	}
	from, to := e.periodRange(cfg.Period)
	rep, err := e.helpdesk.GetSLAReport(ctx, def.TenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("helpdesk_sla: %w", err)
	}
	if rep == nil {
		return emptyResult(def, ""), nil
	}
	result := &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "queue", Label: "Queue", Kind: "string"},
			{Name: "tickets_total", Label: "Tickets", Kind: "number"},
			{Name: "first_response_pct", Label: "FR-Quote", Kind: "percent"},
			{Name: "resolution_pct", Label: "Loesungs-Quote", Kind: "percent"},
			{Name: "avg_first_response_min", Label: "Ø FR (min)", Kind: "number"},
			{Name: "avg_resolution_min", Label: "Ø Loesung (min)", Kind: "number"},
		},
		Rows: make([]map[string]any, 0, len(rep.Queues)),
		Series: []berichte.Series{
			{Label: "Ersantwort %", DataPoints: make([]berichte.DataPoint, 0, len(rep.Queues))},
			{Label: "Loesungs %", DataPoints: make([]berichte.DataPoint, 0, len(rep.Queues))},
		},
	}
	for _, q := range rep.Queues {
		result.Rows = append(result.Rows, map[string]any{
			"queue":                  q.Queue,
			"tickets_total":          q.TicketsTotal,
			"first_response_pct":     q.FirstResponsePct,
			"resolution_pct":         q.ResolutionPct,
			"avg_first_response_min": q.AvgFirstRespMin,
			"avg_resolution_min":     q.AvgResolveMin,
		})
		result.Series[0].DataPoints = append(result.Series[0].DataPoints,
			berichte.DataPoint{Label: q.Queue, Value: q.FirstResponsePct})
		result.Series[1].DataPoints = append(result.Series[1].DataPoints,
			berichte.DataPoint{Label: q.Queue, Value: q.ResolutionPct})
	}
	result.Meta.RowCount = len(result.Rows)
	result.Meta.DefinitionID = def.ID
	result.Meta.GeneratedAt = e.clock.Now()
	return result, nil
}

func (e *Executor) stockWarnings(ctx context.Context, def *berichte.Definition, _ *queryConfig) (*berichte.ReportResult, error) {
	if e.inventar == nil {
		return emptyResult(def, "downstream_not_available"), nil
	}
	rows, err := e.inventar.GetStockWarnings(ctx, def.TenantID)
	if err != nil {
		return nil, fmt.Errorf("stock_warnings: %w", err)
	}
	result := &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "sku", Label: "SKU", Kind: "string"},
			{Name: "name", Label: "Bezeichnung", Kind: "string"},
			{Name: "on_hand", Label: "Bestand", Kind: "number"},
			{Name: "min_quantity", Label: "Mindestmenge", Kind: "number"},
			{Name: "location", Label: "Standort", Kind: "string"},
		},
		Rows: make([]map[string]any, 0, len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, map[string]any{
			"sku":          r.SKU,
			"name":         r.Name,
			"on_hand":      r.OnHand,
			"min_quantity": r.MinQuantity,
			"location":     r.Location,
		})
	}
	result.Meta.RowCount = len(result.Rows)
	result.Meta.DefinitionID = def.ID
	result.Meta.GeneratedAt = e.clock.Now()
	return result, nil
}

func (e *Executor) datevBWA(ctx context.Context, def *berichte.Definition, cfg *queryConfig) (*berichte.ReportResult, error) {
	if e.datev == nil {
		return emptyResult(def, "downstream_not_available"), nil
	}
	from, to := e.periodRange(cfg.Period)
	data, err := e.datev.GetBWAData(ctx, def.TenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("datev_bwa: %w", err)
	}
	if data == nil {
		return emptyResult(def, ""), nil
	}
	result := &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "code", Label: "Konto", Kind: "string"},
			{Name: "label", Label: "Bezeichnung", Kind: "string"},
			{Name: "amount", Label: "Betrag", Kind: "currency"},
		},
		Rows:   make([]map[string]any, 0, len(data.Entries)),
		Totals: map[string]any{"period": data.Period},
	}
	for _, e := range data.Entries {
		result.Rows = append(result.Rows, map[string]any{
			"code":   e.Code,
			"label":  e.Label,
			"amount": e.Amount,
		})
	}
	for k, v := range data.Totals {
		result.Totals[k] = v
	}
	result.Meta.RowCount = len(result.Rows)
	result.Meta.DefinitionID = def.ID
	result.Meta.GeneratedAt = e.clock.Now()
	return result, nil
}

// ============================================================================
// Helpers
// ============================================================================

func emptyResult(def *berichte.Definition, warning string) *berichte.ReportResult {
	if warning != "" {
		slog.Warn("berichte executor downstream missing", "definition_id", def.ID, "warning", warning)
	}
	return &berichte.ReportResult{
		Columns: []berichte.Column{},
		Rows:    []map[string]any{},
		Meta: berichte.ReportMeta{
			DefinitionID: def.ID,
			GeneratedAt:  time.Now(),
			RowCount:     0,
			Warning:      warning,
		},
	}
}

// compile-time check: Executor implements berichte.Executor.
var _ berichte.Executor = (*Executor)(nil)
