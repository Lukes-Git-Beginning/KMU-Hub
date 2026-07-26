package berichte

import (
	"time"

	"github.com/google/uuid"
)

// Definition represents a report definition stored in report_definitions.
type Definition struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Module        string     `json:"module"`         // finanzen|crm|helpdesk|inventar|produktion|cross
	Kind          string     `json:"kind"`           // system|custom
	QueryConfig   []byte     `json:"query_config"`   // raw JSONB aggregation spec
	DefaultFormat string     `json:"default_format"` // pdf|csv|xlsx
	CreatedBy     *uuid.UUID `json:"created_by,omitempty"`
	IsPublished   bool       `json:"is_published"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Document represents a multi-page report authoring document stored in
// report_documents ("Schicht 4"). Rows and Settings stay opaque JSONB: the
// block tree (rows -> columns -> blocks) is owned by the frontend.
type Document struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Module      string     `json:"module"` // finanzen|crm|helpdesk|inventar|produktion|cross
	Status      string     `json:"status"` // draft|final|released|archived
	Rows        []byte     `json:"rows"`   // raw JSONB array of rows
	Settings    []byte     `json:"settings"`
	TemplateID  *string    `json:"template_id,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ReleasedAt  *time.Time `json:"released_at,omitempty"`
}

// CacheEntry represents a cached report result.
type CacheEntry struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	DefinitionID uuid.UUID `json:"definition_id"`
	ParamsHash   string    `json:"params_hash"`
	Result       []byte    `json:"result"` // raw JSONB of ReportResult
	RowCount     int       `json:"row_count"`
	ComputedAt   time.Time `json:"computed_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Schedule represents a scheduled report in report_schedules.
type Schedule struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	DefinitionID   uuid.UUID  `json:"definition_id"`
	Name           string     `json:"name"`
	CronExpression string     `json:"cron_expression"`
	Recipients     []string   `json:"recipients"`
	Format         string     `json:"format"` // pdf|csv|xlsx
	Params         []byte     `json:"params"` // raw JSONB
	Active         bool       `json:"active"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	LastRunStatus  *string    `json:"last_run_status,omitempty"` // success|failed|skipped
	LastRunError   *string    `json:"last_run_error,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Run represents an execution of a report, tracked in report_runs.
type Run struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	DefinitionID uuid.UUID  `json:"definition_id"`
	ScheduleID   *uuid.UUID `json:"schedule_id,omitempty"`
	Trigger      string     `json:"trigger"` // manual|scheduled|api
	Params       []byte     `json:"params"`
	DurationMs   *int       `json:"duration_ms,omitempty"`
	RowCount     *int       `json:"row_count,omitempty"`
	Status       string     `json:"status"` // success|failed
	Error        *string    `json:"error,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// Column describes a column in a report table-style result.
type Column struct {
	Name  string `json:"name"`  // stable machine key ("revenue_eur")
	Label string `json:"label"` // human label ("Umsatz (EUR)")
	Kind  string `json:"kind"`  // number|currency|string|date|percent
}

// DataPoint is a single labelled value in a chart series.
type DataPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// Series is a named collection of DataPoints for charting.
type Series struct {
	Label      string      `json:"label"`
	DataPoints []DataPoint `json:"data_points"`
}

// ReportMeta captures metadata about a generated report.
type ReportMeta struct {
	GeneratedAt  time.Time `json:"generated_at"`
	RowCount     int       `json:"row_count"`
	DefinitionID uuid.UUID `json:"definition_id"`
	FromCache    bool      `json:"from_cache"`
	Warning      string    `json:"warning,omitempty"` // e.g. "downstream_not_available"
}

// ReportResult is the uniform shape returned by an Executor run.
type ReportResult struct {
	Columns []Column         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Series  []Series         `json:"series,omitempty"`
	Totals  map[string]any   `json:"totals,omitempty"`
	Meta    ReportMeta       `json:"meta"`
}

// KPI is a dashboard-level metric rendered on the Berichte page.
type KPI struct {
	ID            string   `json:"id"`
	Label         string   `json:"label"`
	Value         string   `json:"value"` // rendered for display flexibility
	Unit          string   `json:"unit,omitempty"`
	ChangePercent *float64 `json:"change_percent,omitempty"`
	ModuleID      string   `json:"module_id"`
}
