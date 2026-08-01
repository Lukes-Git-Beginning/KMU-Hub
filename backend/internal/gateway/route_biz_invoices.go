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

	response.Proto(w, http.StatusCreated, resp.Invoice)
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
	// Schedule detail view: the invoices a recurring schedule has emitted.
	if rid := r.URL.Query().Get("recurring_id"); rid != "" {
		if _, parseErr := uuid.Parse(rid); parseErr != nil {
			response.JSON(w, http.StatusBadRequest, struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}{
				Error: "invalid recurring_id: must be a valid UUID",
				Code:  "INVALID_RECURRING_ID",
			})
			return
		}
		req.RecurringId = &rid
	}

	resp, err := client.ListInvoices(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	invoicesJSON, err := hrMarshalSlice(resp.Invoices)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"invoices": invoicesJSON,
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

	response.Proto(w, http.StatusOK, resp.Invoice)
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

	response.Proto(w, http.StatusOK, resp.Invoice)
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

	response.Proto(w, http.StatusOK, resp.Invoice)
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

	response.Proto(w, http.StatusOK, resp.Invoice)
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

	response.Proto(w, http.StatusOK, resp.Invoice)
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
// An invoice that does not meet EN 16931 comes back as FailedPrecondition (409)
// naming every unmet requirement — the RPC no longer falls back to the plain PDF,
// which looked like a successful e-invoice and carried no invoice data.
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

// eInvoiceContentTypes are the values the format query parameter accepts, and
// the Content-Type that goes with each. xrechnung is a bare UBL 2.1 document;
// zugferd is a PDF with the CII XML embedded and declared (see
// GenerateZUGFeRDInvoicePDF).
var eInvoiceContentTypes = map[string]string{
	"xrechnung": "application/xml",
	"zugferd":   "application/pdf",
}

// HandleGenerateEInvoice renders an invoice as an outbound e-invoice. An
// invoice that does not meet EN 16931 — or, when buyer_reference (the
// Leitweg-ID, BT-10) is supplied, the stricter German CIUS for public-sector
// buyers — comes back as 409 listing every unmet requirement; no partial
// document is returned in its place.
func (b *BizRoutes) HandleGenerateEInvoice(w http.ResponseWriter, r *http.Request) {
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

	format := r.URL.Query().Get("format")
	contentType, ok := eInvoiceContentTypes[format]
	if !ok {
		response.Error(w, http.StatusBadRequest, "format must be xrechnung or zugferd")
		return
	}

	resp, err := client.GenerateEInvoice(r.Context(), &bizv1.GenerateEInvoiceRequest{
		Id:             id,
		TenantId:       tenantID,
		Format:         format,
		BuyerReference: r.URL.Query().Get("buyer_reference"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondFile(w, resp.Data, resp.Filename, contentType)
}
