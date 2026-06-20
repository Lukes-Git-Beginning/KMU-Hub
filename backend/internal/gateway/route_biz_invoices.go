package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// Invoice Handlers
// ============================================================================

type createInvoiceRequest struct {
	Customer      *bizv1.CustomerSnapshot `json:"customer"       validate:"required"`
	LineItems     []*bizv1.LineItem       `json:"line_items"     validate:"required,min=1"`
	TaxMode       string                  `json:"tax_mode"       validate:"required,oneof=standard reverse_charge kleinunternehmer"`
	InvoiceDate   string                  `json:"invoice_date"   validate:"required,datetime=2006-01-02"`
	DeliveryDate  string                  `json:"delivery_date"  validate:"omitempty,datetime=2006-01-02"`
	DueDate       string                  `json:"due_date"       validate:"omitempty,datetime=2006-01-02"`
	PaymentTerms  string                  `json:"payment_terms"`
	Notes         string                  `json:"notes"`
	SourceQuoteID string                  `json:"source_quote_id" validate:"omitempty,uuid"`
	ContactID     string                  `json:"contact_id"      validate:"omitempty,uuid"`
}

func (b *BizRoutes) HandleCreateInvoice(w http.ResponseWriter, r *http.Request) {
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
	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createInvoiceRequest](w, r)
	if !ok {
		return
	}
	if !validateCustomerVAT(w, req.Customer.GetUstIdNr()) {
		return
	}

	grpcCreateReq := &bizv1.CreateInvoiceRequest{
		TenantId:      tenantID,
		Customer:      req.Customer,
		LineItems:     req.LineItems,
		TaxMode:       taxModeToProto(req.TaxMode),
		InvoiceDate:   req.InvoiceDate,
		DeliveryDate:  req.DeliveryDate,
		DueDate:       req.DueDate,
		PaymentTerms:  req.PaymentTerms,
		Notes:         req.Notes,
		SourceQuoteId: req.SourceQuoteID,
		CreatedBy:     userID,
	}
	if req.ContactID != "" {
		grpcCreateReq.ContactId = &req.ContactID
	}
	resp, err := client.CreateInvoice(r.Context(), grpcCreateReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.Invoice)
}

func (b *BizRoutes) HandleListInvoices(w http.ResponseWriter, r *http.Request) {
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

	req := &bizv1.ListInvoicesRequest{
		TenantId: tenantID,
		Status:   invoiceStatusToProto(r.URL.Query().Get("status")),
		Page:     int32(page),
		PerPage:  int32(pageSize),
	}
	if cid := r.URL.Query().Get("contact_id"); cid != "" {
		if _, parseErr := uuid.Parse(cid); parseErr != nil {
			response.JSON(w, http.StatusBadRequest, struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}{
				Error: "invalid contact_id: must be a valid UUID",
				Code:  "INVALID_CONTACT_ID",
			})
			return
		}
		req.ContactId = &cid
	}

	resp, err := client.ListInvoices(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"invoices": resp.Invoices,
		"total":    resp.Total,
	})
}

func (b *BizRoutes) HandleGetInvoice(w http.ResponseWriter, r *http.Request) {
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
	id := chi.URLParam(r, "id")

	resp, err := client.GetInvoice(r.Context(), &bizv1.GetInvoiceRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Invoice)
}

type updateInvoiceRequest struct {
	Customer     *bizv1.CustomerSnapshot `json:"customer"`
	LineItems    []*bizv1.LineItem       `json:"line_items"`
	TaxMode      string                  `json:"tax_mode"      validate:"omitempty,oneof=standard reverse_charge kleinunternehmer"`
	InvoiceDate  string                  `json:"invoice_date"  validate:"omitempty,datetime=2006-01-02"`
	DeliveryDate string                  `json:"delivery_date" validate:"omitempty,datetime=2006-01-02"`
	DueDate      string                  `json:"due_date"      validate:"omitempty,datetime=2006-01-02"`
	PaymentTerms string                  `json:"payment_terms"`
	Notes        string                  `json:"notes"`
}

func (b *BizRoutes) HandleUpdateInvoice(w http.ResponseWriter, r *http.Request) {
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
	id := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[updateInvoiceRequest](w, r)
	if !ok {
		return
	}

	if req.Customer != nil && !validateCustomerVAT(w, req.Customer.GetUstIdNr()) {
		return
	}

	resp, err := client.UpdateInvoice(r.Context(), &bizv1.UpdateInvoiceRequest{
		Id:           id,
		TenantId:     tenantID,
		Customer:     req.Customer,
		LineItems:    req.LineItems,
		TaxMode:      taxModeToProto(req.TaxMode),
		InvoiceDate:  req.InvoiceDate,
		DeliveryDate: req.DeliveryDate,
		DueDate:      req.DueDate,
		PaymentTerms: req.PaymentTerms,
		Notes:        req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Invoice)
}

func (b *BizRoutes) HandleSendInvoice(w http.ResponseWriter, r *http.Request) {
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
	id := chi.URLParam(r, "id")

	resp, err := client.SendInvoice(r.Context(), &bizv1.SendInvoiceRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Invoice)
}

func (b *BizRoutes) HandleMarkInvoicePaid(w http.ResponseWriter, r *http.Request) {
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
	id := chi.URLParam(r, "id")

	resp, err := client.MarkInvoicePaid(r.Context(), &bizv1.MarkInvoicePaidRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Invoice)
}

func (b *BizRoutes) HandleCancelInvoice(w http.ResponseWriter, r *http.Request) {
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
	id := chi.URLParam(r, "id")

	resp, err := client.CancelInvoice(r.Context(), &bizv1.CancelInvoiceRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Invoice)
}

func (b *BizRoutes) HandleGenerateInvoicePDF(w http.ResponseWriter, r *http.Request) {
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
	id := chi.URLParam(r, "id")

	// ZUGFeRD format: embed Factur-X XML into the PDF
	if r.URL.Query().Get("format") == "zugferd" {
		b.handleZUGFeRDInvoicePDF(w, r, client, tenantID, id)
		return
	}

	resp, err := client.GenerateInvoicePDF(r.Context(), &bizv1.GenerateInvoicePDFRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondPDF(w, resp.PdfData, resp.Filename)
}

// handleZUGFeRDInvoicePDF delegates to the biz service which owns the domain logic.
// Graceful degradation (plain PDF on XML failure) is handled inside the RPC.
func (b *BizRoutes) handleZUGFeRDInvoicePDF(w http.ResponseWriter, r *http.Request, client bizv1.FinanceServiceClient, tenantID, id string) {
	resp, err := client.GenerateZUGFeRDInvoicePDF(r.Context(), &bizv1.GenerateZUGFeRDInvoicePDFRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	respondPDF(w, resp.PdfData, resp.Filename)
}
