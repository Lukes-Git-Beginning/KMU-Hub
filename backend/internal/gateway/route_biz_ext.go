package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// BizExtRoutes handles extended biz/HR routes.
// Currently: time-tracking-to-invoice conversion.
type BizExtRoutes struct {
	workTimeRepo timetracking.WorkTimeRepository
	invoiceRepo  invoice.Repository
	invoiceSvc   *invoice.Service
}

// NewBizExtRoutes creates BizExtRoutes using repository abstractions.
func NewBizExtRoutes(pool *pgxpool.Pool) *BizExtRoutes {
	workTimeRepo := timetracking.NewPostgresWorkTimeRepo(pool)
	invoiceRepo := invoice.NewPostgresRepository(pool)
	numSeqRepo := quote.NewPostgresNumberSequenceRepo(pool)
	settingsRepo := quote.NewPostgresCompanySettingsRepo(pool)
	invoiceSvc := invoice.NewService(invoiceRepo, numSeqRepo, settingsRepo, nil)

	return &BizExtRoutes{
		workTimeRepo: workTimeRepo,
		invoiceRepo:  invoiceRepo,
		invoiceSvc:   invoiceSvc,
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

	// Aggregate completed work time entries via HR repository
	totalMinutes, entryIDs, err := b.workTimeRepo.AggregateWorkTimeForInvoice(r.Context(), tenantID, employeeID, dateFrom, dateTo)
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
	ttSource := buildTimeTrackingSource(employeeID, req.DateFrom, req.DateTo, totalMinutes, entryIDs)
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

	// Attach time_tracking_source to the invoice record via invoice repository
	if updateErr := b.invoiceRepo.LinkTimeTracking(r.Context(), inv.ID, ttSourceJSON); updateErr != nil {
		// Non-fatal: invoice is created, audit trail attachment failed
		_ = updateErr
	}

	inv.TimeTrackingSource = ttSourceJSON

	response.JSON(w, http.StatusCreated, inv)
}

// ============================================================================
// Internal helpers
// ============================================================================

type timeTrackingSource struct {
	EmployeeID   string    `json:"employee_id"`
	DateFrom     string    `json:"date_from"`
	DateTo       string    `json:"date_to"`
	TotalMinutes int       `json:"total_minutes"`
	EntryIDs     []string  `json:"entry_ids"`
	CreatedAt    time.Time `json:"created_at"`
}

func buildTimeTrackingSource(employeeID uuid.UUID, dateFrom, dateTo string, totalMinutes int, entryIDs []string) timeTrackingSource {
	return timeTrackingSource{
		EmployeeID:   employeeID.String(),
		DateFrom:     dateFrom,
		DateTo:       dateTo,
		TotalMinutes: totalMinutes,
		EntryIDs:     entryIDs,
		CreatedAt:    time.Now(),
	}
}

// ============================================================================
// Route registration helper (called from route_hr.go within existing block)
// ============================================================================

// registerTimeExtRoutes adds time-to-invoice route into an existing /hr/time subrouter.
func (b *BizExtRoutes) registerTimeExtRoutes(r chi.Router) {
	r.With(middleware.RequirePermission("hr", "write")).Post("/create-invoice", b.HandleCreateInvoiceFromTime)
}

