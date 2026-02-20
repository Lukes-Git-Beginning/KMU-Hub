package condition

import (
	"fmt"
	"time"
)

// ExprEnv provides a typed environment for expr-lang expression evaluation.
// Each trigger type populates the relevant sub-env struct so that expressions
// can reference typed fields like deal.Value > 10000 or invoice.DaysOverdue > 30.
type ExprEnv struct {
	// Event metadata
	EventType string    `expr:"event_type"`
	Module    string    `expr:"module"`
	Timestamp time.Time `expr:"timestamp"`

	// CRM entities
	Deal    *DealEnv    `expr:"deal"`
	Contact *ContactEnv `expr:"contact"`
	Company *CompanyEnv `expr:"company"`

	// Finance entities
	Invoice *InvoiceEnv `expr:"invoice"`
	Quote   *QuoteEnv   `expr:"quote"`

	// HR entities
	Leave *LeaveEnv `expr:"leave"`
	Shift *ShiftEnv `expr:"shift"`

	// Work entities
	Task    *TaskEnv    `expr:"task"`
	Project *ProjectEnv `expr:"project"`

	// Email
	Email *EmailEnv `expr:"email"`

	// Chained action outputs from previous steps
	Prev map[string]any `expr:"prev"`
}

// DealEnv provides typed fields for CRM deal triggers.
type DealEnv struct {
	ID        string  `expr:"id"`
	Name      string  `expr:"name"`
	Value     float64 `expr:"value"`
	StageName string  `expr:"stage_name"`
	StageOld  string  `expr:"stage_old"`
	OwnerID   string  `expr:"owner_id"`
}

// ContactEnv provides typed fields for CRM contact triggers.
type ContactEnv struct {
	ID    string `expr:"id"`
	Name  string `expr:"name"`
	Email string `expr:"email"`
	Phone string `expr:"phone"`
	Tags  string `expr:"tags"`
}

// CompanyEnv provides typed fields for CRM company triggers.
type CompanyEnv struct {
	ID       string `expr:"id"`
	Name     string `expr:"name"`
	Industry string `expr:"industry"`
}

// InvoiceEnv provides typed fields for finance invoice triggers.
type InvoiceEnv struct {
	ID          string    `expr:"id"`
	Number      string    `expr:"number"`
	Total       float64   `expr:"total"`
	Status      string    `expr:"status"`
	DueDate     time.Time `expr:"due_date"`
	DaysOverdue int       `expr:"days_overdue"`
}

// QuoteEnv provides typed fields for finance quote triggers.
type QuoteEnv struct {
	ID     string  `expr:"id"`
	Number string  `expr:"number"`
	Total  float64 `expr:"total"`
	Status string  `expr:"status"`
}

// LeaveEnv provides typed fields for HR leave triggers.
type LeaveEnv struct {
	EmployeeName string    `expr:"employee_name"`
	Type         string    `expr:"type"`
	Days         float64   `expr:"days"`
	StartDate    time.Time `expr:"start_date"`
	EndDate      time.Time `expr:"end_date"`
}

// ShiftEnv provides typed fields for HR shift triggers.
type ShiftEnv struct {
	EmployeeID string    `expr:"employee_id"`
	StartTime  time.Time `expr:"start_time"`
	EndTime    time.Time `expr:"end_time"`
	DurationH  float64   `expr:"duration_h"`
}

// TaskEnv provides typed fields for work task triggers.
type TaskEnv struct {
	ID        string `expr:"id"`
	Title     string `expr:"title"`
	Status    string `expr:"status"`
	StatusOld string `expr:"status_old"`
	Priority  string `expr:"priority"`
	ProjectID string `expr:"project_id"`
}

// ProjectEnv provides typed fields for work project triggers.
type ProjectEnv struct {
	ID     string `expr:"id"`
	Name   string `expr:"name"`
	Key    string `expr:"key"`
	Status string `expr:"status"`
}

// EmailEnv provides typed fields for email triggers.
type EmailEnv struct {
	ID      string `expr:"id"`
	From    string `expr:"from"`
	To      string `expr:"to"`
	Subject string `expr:"subject"`
	HasCRM  bool   `expr:"has_crm"`
}

// BuildEnvFromEvent constructs a typed ExprEnv from raw event data.
// It populates the relevant sub-env struct based on the trigger type's
// module prefix (e.g., "crm.deal.*" populates the Deal field).
func BuildEnvFromEvent(triggerType string, eventPayload map[string]any) *ExprEnv {
	env := &ExprEnv{
		EventType: triggerType,
		Module:    extractModule(triggerType),
		Timestamp: time.Now(),
		Prev:      make(map[string]any),
	}

	if eventPayload == nil {
		return env
	}

	module := env.Module
	switch module {
	case "crm":
		env.Deal = buildDealEnv(eventPayload)
		env.Contact = buildContactEnv(eventPayload)
		env.Company = buildCompanyEnv(eventPayload)
	case "biz":
		env.Invoice = buildInvoiceEnv(eventPayload)
		env.Quote = buildQuoteEnv(eventPayload)
	case "hr":
		env.Leave = buildLeaveEnv(eventPayload)
		env.Shift = buildShiftEnv(eventPayload)
	case "work":
		env.Task = buildTaskEnv(eventPayload)
		env.Project = buildProjectEnv(eventPayload)
	case "email":
		env.Email = buildEmailEnv(eventPayload)
	}

	return env
}

// extractModule extracts the module prefix from a trigger type string.
// e.g., "crm.deal.stage_changed" -> "crm"
func extractModule(triggerType string) string {
	for i, c := range triggerType {
		if c == '.' {
			return triggerType[:i]
		}
	}
	return triggerType
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getFloat64(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return 0
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func buildDealEnv(m map[string]any) *DealEnv {
	return &DealEnv{
		ID:        getString(m, "deal_id"),
		Name:      getString(m, "deal_name"),
		Value:     getFloat64(m, "deal_value"),
		StageName: getString(m, "stage_name"),
		StageOld:  getString(m, "stage_old"),
		OwnerID:   getString(m, "owner_id"),
	}
}

func buildContactEnv(m map[string]any) *ContactEnv {
	return &ContactEnv{
		ID:    getString(m, "contact_id"),
		Name:  getString(m, "contact_name"),
		Email: getString(m, "contact_email"),
		Phone: getString(m, "contact_phone"),
		Tags:  getString(m, "contact_tags"),
	}
}

func buildCompanyEnv(m map[string]any) *CompanyEnv {
	return &CompanyEnv{
		ID:       getString(m, "company_id"),
		Name:     getString(m, "company_name"),
		Industry: getString(m, "company_industry"),
	}
}

func buildInvoiceEnv(m map[string]any) *InvoiceEnv {
	return &InvoiceEnv{
		ID:          getString(m, "invoice_id"),
		Number:      getString(m, "invoice_number"),
		Total:       getFloat64(m, "total"),
		Status:      getString(m, "status"),
		DaysOverdue: getInt(m, "days_overdue"),
	}
}

func buildQuoteEnv(m map[string]any) *QuoteEnv {
	return &QuoteEnv{
		ID:     getString(m, "quote_id"),
		Number: getString(m, "quote_number"),
		Total:  getFloat64(m, "total"),
		Status: getString(m, "status"),
	}
}

func buildLeaveEnv(m map[string]any) *LeaveEnv {
	return &LeaveEnv{
		EmployeeName: getString(m, "employee_name"),
		Type:         getString(m, "leave_type"),
		Days:         getFloat64(m, "days"),
	}
}

func buildShiftEnv(m map[string]any) *ShiftEnv {
	return &ShiftEnv{
		EmployeeID: getString(m, "employee_id"),
		DurationH:  getFloat64(m, "duration_h"),
	}
}

func buildTaskEnv(m map[string]any) *TaskEnv {
	return &TaskEnv{
		ID:        getString(m, "task_id"),
		Title:     getString(m, "task_title"),
		Status:    getString(m, "status"),
		StatusOld: getString(m, "status_old"),
		Priority:  getString(m, "priority"),
		ProjectID: getString(m, "project_id"),
	}
}

func buildProjectEnv(m map[string]any) *ProjectEnv {
	return &ProjectEnv{
		ID:     getString(m, "project_id"),
		Name:   getString(m, "project_name"),
		Key:    getString(m, "project_key"),
		Status: getString(m, "status"),
	}
}

func buildEmailEnv(m map[string]any) *EmailEnv {
	return &EmailEnv{
		ID:      getString(m, "email_id"),
		From:    getString(m, "from"),
		To:      getString(m, "to"),
		Subject: getString(m, "subject"),
		HasCRM:  getBool(m, "has_crm"),
	}
}
