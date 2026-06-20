package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// BizExtRoutes handles extended biz/HR routes that require the biz gRPC backend.
// Previously these held direct DB access; they are now thin gRPC proxy handlers.
type BizExtRoutes struct {
	registry *ServiceRegistry
}

// NewBizExtRoutes creates BizExtRoutes backed by the gRPC service registry.
func NewBizExtRoutes(registry *ServiceRegistry) *BizExtRoutes {
	return &BizExtRoutes{registry: registry}
}

// getBizClient lazily obtains a gRPC client for the Finance/biz service.
func (b *BizExtRoutes) getBizClient() (bizv1.FinanceServiceClient, error) {
	conn, err := b.registry.GetConnection("biz")
	if err != nil {
		return nil, err
	}
	return bizv1.NewFinanceServiceClient(conn), nil
}

// ============================================================================
// Time-Tracking → Invoice
// ============================================================================

type createInvoiceFromTimeRequest struct {
	EmployeeID      string `json:"employee_id" validate:"required,uuid"`
	CustomerName    string `json:"customer_name" validate:"required"`
	CustomerAddress string `json:"customer_address"`
	CustomerEmail   string `json:"customer_email" validate:"omitempty,email"`
	DateFrom        string `json:"date_from" validate:"required,datetime=2006-01-02"` // YYYY-MM-DD
	DateTo          string `json:"date_to" validate:"required,datetime=2006-01-02"`   // YYYY-MM-DD
	HourlyRate      string `json:"hourly_rate" validate:"required,decimal_gt0"`
	Description     string `json:"description"`
	TaxMode         string `json:"tax_mode" validate:"omitempty,oneof=standard reverse_charge kleinunternehmer"` // default: "standard"
}

// HandleCreateInvoiceFromTime is a thin proxy: it decodes the HTTP request,
// calls the biz gRPC service, and forwards the JSON invoice response.
func (b *BizExtRoutes) HandleCreateInvoiceFromTime(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createInvoiceFromTimeRequest](w, r)
	if !ok {
		return
	}

	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, "biz")
		return
	}

	grpcResp, err := client.CreateInvoiceFromTimeEntries(r.Context(), &bizv1.CreateInvoiceFromTimeEntriesRequest{
		TenantId:        tenantID,
		EmployeeId:      req.EmployeeID,
		CustomerName:    req.CustomerName,
		CustomerAddress: req.CustomerAddress,
		CustomerEmail:   req.CustomerEmail,
		DateFrom:        req.DateFrom,
		DateTo:          req.DateTo,
		HourlyRate:      req.HourlyRate,
		Description:     req.Description,
		TaxMode:         req.TaxMode,
		CreatedBy:       userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// The biz service returns the invoice as JSON bytes — forward directly.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(grpcResp.GetInvoiceJson())
}

// ============================================================================
// Route registration helper (called from route_hr.go within existing block)
// ============================================================================

// registerTimeExtRoutes adds time-to-invoice route into an existing /hr/time subrouter.
func (b *BizExtRoutes) registerTimeExtRoutes(r chi.Router) {
	r.With(middleware.RequirePermission("hr", "write")).Post("/create-invoice", b.HandleCreateInvoiceFromTime)
}
