package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// Quote Handlers
// ============================================================================

type createQuoteRequest struct {
	Customer   *bizv1.CustomerSnapshot `json:"customer"    validate:"required"`
	LineItems  []*bizv1.LineItem       `json:"line_items"  validate:"required,min=1"`
	TaxMode    string                  `json:"tax_mode"    validate:"required,oneof=standard reverse_charge kleinunternehmer"`
	ValidUntil string                  `json:"valid_until" validate:"omitempty,datetime=2006-01-02"`
	Notes      string                  `json:"notes"`
	DealID     string                  `json:"deal_id"     validate:"omitempty,uuid"`
}

func (b *BizRoutes) HandleCreateQuote(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[createQuoteRequest](w, r)
	if !ok {
		return
	}
	if !validateCustomerVAT(w, req.Customer.GetUstIdNr()) {
		return
	}

	resp, err := client.CreateQuote(r.Context(), &bizv1.CreateQuoteRequest{
		TenantId:   tenantID,
		Customer:   req.Customer,
		LineItems:  req.LineItems,
		TaxMode:    taxModeToProto(req.TaxMode),
		ValidUntil: req.ValidUntil,
		Notes:      req.Notes,
		DealId:     req.DealID,
		CreatedBy:  userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.Quote)
}

func (b *BizRoutes) HandleListQuotes(w http.ResponseWriter, r *http.Request) {
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

	req := &bizv1.ListQuotesRequest{
		TenantId: tenantID,
		Status:   quoteStatusToProto(r.URL.Query().Get("status")),
		DealId:   r.URL.Query().Get("deal_id"),
		Page:     int32(page),
		PerPage:  int32(pageSize),
	}

	resp, err := client.ListQuotes(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"quotes": resp.Quotes,
		"total":  resp.Total,
	})
}

func (b *BizRoutes) HandleGetQuote(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.GetQuote(r.Context(), &bizv1.GetQuoteRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Quote)
}

type updateQuoteRequest struct {
	Customer   *bizv1.CustomerSnapshot `json:"customer"`
	LineItems  []*bizv1.LineItem       `json:"line_items"`
	TaxMode    string                  `json:"tax_mode"    validate:"omitempty,oneof=standard reverse_charge kleinunternehmer"`
	ValidUntil string                  `json:"valid_until" validate:"omitempty,datetime=2006-01-02"`
	Notes      string                  `json:"notes"`
}

func (b *BizRoutes) HandleUpdateQuote(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[updateQuoteRequest](w, r)
	if !ok {
		return
	}

	if req.Customer != nil && !validateCustomerVAT(w, req.Customer.GetUstIdNr()) {
		return
	}

	resp, err := client.UpdateQuote(r.Context(), &bizv1.UpdateQuoteRequest{
		Id:         id,
		TenantId:   tenantID,
		Customer:   req.Customer,
		LineItems:  req.LineItems,
		TaxMode:    taxModeToProto(req.TaxMode),
		ValidUntil: req.ValidUntil,
		Notes:      req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Quote)
}

func (b *BizRoutes) HandleDeleteQuote(w http.ResponseWriter, r *http.Request) {
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

	_, err = client.DeleteQuote(r.Context(), &bizv1.DeleteQuoteRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (b *BizRoutes) HandleSendQuote(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.SendQuote(r.Context(), &bizv1.SendQuoteRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Quote)
}

func (b *BizRoutes) HandleAcceptQuote(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.AcceptQuote(r.Context(), &bizv1.AcceptQuoteRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Quote)
}

func (b *BizRoutes) HandleRejectQuote(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.RejectQuote(r.Context(), &bizv1.RejectQuoteRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Quote)
}

type convertQuoteRequest struct {
	InvoiceDate  string `json:"invoice_date"  validate:"required,datetime=2006-01-02"`
	DueDate      string `json:"due_date"      validate:"omitempty,datetime=2006-01-02"`
	DeliveryDate string `json:"delivery_date" validate:"omitempty,datetime=2006-01-02"`
	PaymentTerms string `json:"payment_terms"`
}

func (b *BizRoutes) HandleConvertQuoteToInvoice(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[convertQuoteRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ConvertQuoteToInvoice(r.Context(), &bizv1.ConvertQuoteToInvoiceRequest{
		Id:           id,
		TenantId:     tenantID,
		InvoiceDate:  req.InvoiceDate,
		DueDate:      req.DueDate,
		DeliveryDate: req.DeliveryDate,
		PaymentTerms: req.PaymentTerms,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.Invoice)
}

func (b *BizRoutes) HandleGenerateQuotePDF(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.GenerateQuotePDF(r.Context(), &bizv1.GenerateQuotePDFRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondPDF(w, resp.PdfData, resp.Filename)
}

// ============================================================================
// Deal-to-Quote Handler
// ============================================================================

// HandleCreateQuoteFromDeal is a thin proxy that delegates the cross-service
// orchestration (fetch deal, contact, company; assemble customer snapshot) to
// the biz gRPC service via CreateQuoteFromDeal. The HTTP contract is unchanged.
func (b *BizRoutes) HandleCreateQuoteFromDeal(w http.ResponseWriter, r *http.Request) {
	bizClient, err := b.getBizClient()
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
	dealID := chi.URLParam(r, "dealId")

	resp, err := bizClient.CreateQuoteFromDeal(r.Context(), &bizv1.CreateQuoteFromDealRequest{
		TenantId:  tenantID,
		DealId:    dealID,
		CreatedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.GetQuote())
}
