package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// Recurring Invoice Handlers (Abo-Rechnungen) — Migration 000246
// ============================================================================

type createRecurringRequest struct {
	Title            string                  `json:"title"`
	Customer         *bizv1.CustomerSnapshot `json:"customer"           validate:"required"`
	LineItems        []*bizv1.LineItem       `json:"line_items"         validate:"required,min=1"`
	TaxMode          string                  `json:"tax_mode"           validate:"required,oneof=standard reverse_charge kleinunternehmer"`
	Currency         string                  `json:"currency"           validate:"omitempty,len=3"`
	Interval         string                  `json:"interval"           validate:"required,oneof=weekly monthly quarterly yearly"`
	StartDate        string                  `json:"start_date"         validate:"required,datetime=2006-01-02"`
	EndDate          string                  `json:"end_date"           validate:"omitempty,datetime=2006-01-02"`
	PaymentTermsDays int32                   `json:"payment_terms_days" validate:"omitempty,gte=0,lte=365"`
	Notes            string                  `json:"notes"`
}

// updateRecurringRequest mirrors Partial<CreateRecurringInvoiceRequest>: every
// field is a pointer so "absent" and "cleared" stay distinguishable.
type updateRecurringRequest struct {
	Title            *string                 `json:"title"`
	Customer         *bizv1.CustomerSnapshot `json:"customer"`
	LineItems        []*bizv1.LineItem       `json:"line_items"`
	TaxMode          *string                 `json:"tax_mode"           validate:"omitempty,oneof=standard reverse_charge kleinunternehmer"`
	Currency         *string                 `json:"currency"           validate:"omitempty,len=3"`
	Interval         *string                 `json:"interval"           validate:"omitempty,oneof=weekly monthly quarterly yearly"`
	StartDate        *string                 `json:"start_date"         validate:"omitempty,datetime=2006-01-02"`
	EndDate          *string                 `json:"end_date"           validate:"omitempty,datetime=2006-01-02"`
	PaymentTermsDays *int32                  `json:"payment_terms_days" validate:"omitempty,gte=0,lte=365"`
	Notes            *string                 `json:"notes"`
}

func (b *BizRoutes) HandleListRecurringInvoices(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListRecurringInvoices(r.Context(), &bizv1.ListRecurringInvoicesRequest{
		TenantId: tenantID,
		Status:   r.URL.Query().Get("status"),
		Page:     int32(page),
		PerPage:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// The frontend types this as {recurring: RecurringInvoice[], total: number};
	// an empty schedule list must serialize as [], never null.
	response.ProtoListWrapped(w, http.StatusOK, "recurring", resp.GetRecurring(), map[string]any{
		"total": resp.GetTotal(),
	})
}

func (b *BizRoutes) HandleCreateRecurringInvoice(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	req, ok := decodeAndValidate[createRecurringRequest](w, r)
	if !ok {
		return
	}
	if !validateCustomerVAT(w, req.Customer.GetUstIdNr()) {
		return
	}

	resp, err := client.CreateRecurringInvoice(r.Context(), &bizv1.CreateRecurringInvoiceRequest{
		TenantId:         tenantID,
		Title:            req.Title,
		Customer:         req.Customer,
		LineItems:        req.LineItems,
		TaxMode:          taxModeToProto(req.TaxMode),
		Currency:         req.Currency,
		Interval:         req.Interval,
		StartDate:        req.StartDate,
		EndDate:          req.EndDate,
		PaymentTermsDays: req.PaymentTermsDays,
		Notes:            req.Notes,
		CreatedBy:        middleware.GetUserID(r.Context()),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (b *BizRoutes) HandleGetRecurringInvoice(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.GetRecurringInvoice(r.Context(), &bizv1.GetRecurringInvoiceRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (b *BizRoutes) HandleUpdateRecurringInvoice(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	req, ok := decodeAndValidate[updateRecurringRequest](w, r)
	if !ok {
		return
	}
	if req.Customer != nil && !validateCustomerVAT(w, req.Customer.GetUstIdNr()) {
		return
	}

	grpcReq := &bizv1.UpdateRecurringInvoiceRequest{
		TenantId:  tenantID,
		Id:        chi.URLParam(r, "id"),
		Title:     req.Title,
		Customer:  req.Customer,
		LineItems: req.LineItems,
		Currency:  req.Currency,
		Interval:  req.Interval,
		StartDate: req.StartDate,
		Notes:     req.Notes,
	}
	if req.TaxMode != nil {
		mode := taxModeToProto(*req.TaxMode)
		grpcReq.TaxMode = &mode
	}
	if req.PaymentTermsDays != nil {
		grpcReq.PaymentTermsDays = req.PaymentTermsDays
	}
	// An explicit null end_date clears the end date; an absent one leaves it alone.
	if req.EndDate != nil {
		grpcReq.EndDate = req.EndDate
		if *req.EndDate == "" {
			grpcReq.ClearEndDate = true
		}
	}

	resp, err := client.UpdateRecurringInvoice(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (b *BizRoutes) HandleDeleteRecurringInvoice(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	if _, err := client.DeleteRecurringInvoice(r.Context(), &bizv1.DeleteRecurringInvoiceRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
	}); err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, struct{}{})
}

// HandlePauseRecurringInvoice suspends a schedule; generation stays blocked until
// it is resumed.
func (b *BizRoutes) HandlePauseRecurringInvoice(w http.ResponseWriter, r *http.Request) {
	b.setRecurringStatus(w, r, "paused")
}

// HandleResumeRecurringInvoice reactivates a paused schedule. A schedule past its
// end date stays ended.
func (b *BizRoutes) HandleResumeRecurringInvoice(w http.ResponseWriter, r *http.Request) {
	b.setRecurringStatus(w, r, "active")
}

func (b *BizRoutes) setRecurringStatus(w http.ResponseWriter, r *http.Request, newStatus string) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.SetRecurringInvoiceStatus(r.Context(), &bizv1.SetRecurringInvoiceStatusRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
		Status:   newStatus,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// HandleGenerateRecurringInvoice emits the invoice for the schedule's current
// period. Idempotent per period: a repeated call returns the same invoice.
func (b *BizRoutes) HandleGenerateRecurringInvoice(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.GenerateRecurringInvoice(r.Context(), &bizv1.GenerateRecurringInvoiceRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
		UserId:   middleware.GetUserID(r.Context()),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}
