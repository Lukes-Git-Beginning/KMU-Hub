package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// BizExtRoutes handles extended biz/HR routes with direct DB access.
// Currently: time-tracking-to-invoice conversion.
type BizExtRoutes struct {
	pool       *pgxpool.Pool
	invoiceSvc *invoice.Service
}

// NewBizExtRoutes creates BizExtRoutes using direct DB connections.
func NewBizExtRoutes(pool *pgxpool.Pool) *BizExtRoutes {
	invoiceRepo := invoice.NewPostgresRepository(pool)
	numSeqRepo := quote.NewPostgresNumberSequenceRepo(pool)
	settingsRepo := quote.NewPostgresCompanySettingsRepo(pool)
	invoiceSvc := invoice.NewService(invoiceRepo, numSeqRepo, settingsRepo, nil)

	return &BizExtRoutes{
		pool:       pool,
		invoiceSvc: invoiceSvc,
	}
}

// ============================================================================
// Time-Tracking → Invoice
// ============================================================================

type createInvoiceFromTimeRequest struct {
	EmployeeID      string `json:"employee_id"`
	CustomerName    string `json:"customer_name"`
	CustomerAddress string `json:"customer_address"`
	CustomerEmail   string `json:"customer_email"`
	DateFrom        string `json:"date_from"` // YYYY-MM-DD
	DateTo          string `json:"date_to"`   // YYYY-MM-DD
	HourlyRate      string `json:"hourly_rate"`
	Description     string `json:"description"`
	TaxMode         string `json:"tax_mode"` // default: "standard"
}

// HandleCreateInvoiceFromTime aggregates completed work time entries for an employee
// in the given date range and creates a draft invoice.
func (b *BizExtRoutes) HandleCreateInvoiceFromTime(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := getTenantID(r)
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid tenant id")
		return
	}

	userIDStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req createInvoiceFromTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	employeeID, err := uuid.Parse(req.EmployeeID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid employee_id")
		return
	}

	dateFrom, err := time.Parse("2006-01-02", req.DateFrom)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid date_from, expected YYYY-MM-DD")
		return
	}
	dateTo, err := time.Parse("2006-01-02", req.DateTo)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid date_to, expected YYYY-MM-DD")
		return
	}
	// Include the full last day
	dateTo = dateTo.Add(24*time.Hour - time.Second)

	hourlyRate, err := decimal.NewFromString(req.HourlyRate)
	if err != nil || hourlyRate.IsNegative() {
		response.Error(w, http.StatusBadRequest, "invalid hourly_rate")
		return
	}

	if req.CustomerName == "" {
		response.Error(w, http.StatusBadRequest, "customer_name is required")
		return
	}

	taxMode := req.TaxMode
	if taxMode == "" {
		taxMode = models.TaxModeStandard
	}

	// Aggregate completed work time entries
	totalMinutes, entryIDs, err := b.aggregateWorkTime(r.Context(), tenantID, employeeID, dateFrom, dateTo)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, fmt.Sprintf("aggregate work time: %s", err.Error()))
		return
	}

	if totalMinutes == 0 {
		response.Error(w, http.StatusUnprocessableEntity, "no completed work time entries found for the given period")
		return
	}

	// Convert minutes to hours (2 decimal places)
	hours := decimal.NewFromInt(int64(totalMinutes)).Div(decimal.NewFromInt(60)).Round(2)
	lineTotal := hours.Mul(hourlyRate)

	description := req.Description
	if description == "" {
		description = fmt.Sprintf("Arbeitszeit %s–%s", req.DateFrom, req.DateTo)
	}

	lineItem := models.LineItem{
		ID:          uuid.New().String(),
		Position:    1,
		Description: description,
		Quantity:    hours,
		UnitPrice:   hourlyRate,
		TaxRate:     decimal.NewFromInt(19),
		LineTotal:   lineTotal,
	}

	// Build time_tracking_source audit trail
	type timeTrackingSource struct {
		EmployeeID   string    `json:"employee_id"`
		DateFrom     string    `json:"date_from"`
		DateTo       string    `json:"date_to"`
		TotalMinutes int       `json:"total_minutes"`
		EntryIDs     []string  `json:"entry_ids"`
		CreatedAt    time.Time `json:"created_at"`
	}
	ttSource := timeTrackingSource{
		EmployeeID:   employeeID.String(),
		DateFrom:     req.DateFrom,
		DateTo:       req.DateTo,
		TotalMinutes: totalMinutes,
		EntryIDs:     entryIDs,
		CreatedAt:    time.Now(),
	}
	ttSourceJSON, _ := json.Marshal(ttSource)

	// Create draft invoice
	inv, err := b.invoiceSvc.Create(r.Context(), invoice.CreateInput{
		TenantID:        tenantID,
		CustomerName:    req.CustomerName,
		CustomerAddress: req.CustomerAddress,
		CustomerEmail:   req.CustomerEmail,
		TaxMode:         taxMode,
		LineItems:       []models.LineItem{lineItem},
		InvoiceDate:     time.Now(),
		UserID:          userID,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, fmt.Sprintf("create invoice: %s", err.Error()))
		return
	}

	// Attach time_tracking_source to the invoice record
	if updateErr := b.setTimeTrackingSource(r.Context(), inv.ID, ttSourceJSON); updateErr != nil {
		// Non-fatal: invoice is created, audit trail attachment failed
		_ = updateErr
	}

	inv.TimeTrackingSource = ttSourceJSON

	response.JSON(w, http.StatusCreated, inv)
}

// aggregateWorkTime returns total net_work_minutes and entry IDs for completed
// work time entries for the given employee in the date range.
func (b *BizExtRoutes) aggregateWorkTime(ctx context.Context, tenantID, employeeID uuid.UUID, from, to time.Time) (int, []string, error) {
	rows, err := b.pool.Query(ctx,
		`SELECT id, net_work_minutes
		 FROM hr_work_time_entries
		 WHERE tenant_id = $1
		   AND employee_id = $2
		   AND status = 'completed'
		   AND clock_in >= $3
		   AND clock_in <= $4
		   AND net_work_minutes IS NOT NULL`,
		tenantID, employeeID, from, to,
	)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var total int
	var entryIDs []string
	for rows.Next() {
		var entryID uuid.UUID
		var netMinutes int
		if err := rows.Scan(&entryID, &netMinutes); err != nil {
			return 0, nil, err
		}
		total += netMinutes
		entryIDs = append(entryIDs, entryID.String())
	}
	return total, entryIDs, rows.Err()
}

// setTimeTrackingSource persists the time_tracking_source JSONB on the invoice.
func (b *BizExtRoutes) setTimeTrackingSource(ctx context.Context, invoiceID uuid.UUID, src json.RawMessage) error {
	_, err := b.pool.Exec(ctx,
		`UPDATE finance_invoices SET time_tracking_source = $1 WHERE id = $2`,
		src, invoiceID,
	)
	return err
}

// ============================================================================
// Route registration helper (called from route_hr.go within existing block)
// ============================================================================

// registerTimeExtRoutes adds time-to-invoice route into an existing /hr/time subrouter.
func (b *BizExtRoutes) registerTimeExtRoutes(r chi.Router) {
	r.With(middleware.RequirePermission("hr", "write")).Post("/create-invoice", b.HandleCreateInvoiceFromTime)
}
