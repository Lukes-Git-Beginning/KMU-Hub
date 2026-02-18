package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// BizRoutes handles HTTP routes for the Biz (Finance) backend service.
type BizRoutes struct {
	registry *ServiceRegistry
}

// NewBizRoutes creates a new BizRoutes with the given service registry.
func NewBizRoutes(registry *ServiceRegistry) *BizRoutes {
	return &BizRoutes{registry: registry}
}

// ServiceName returns the backend service name.
func (b *BizRoutes) ServiceName() string { return "biz" }

// getBizClient lazily obtains a gRPC client for the Biz service.
func (b *BizRoutes) getBizClient() (bizv1.FinanceServiceClient, error) {
	conn, err := b.registry.GetConnection("biz")
	if err != nil {
		return nil, err
	}
	return bizv1.NewFinanceServiceClient(conn), nil
}

// getCRMClient lazily obtains a gRPC client for the CRM service.
func (b *BizRoutes) getCRMClient() (crmv1.CRMServiceClient, error) {
	conn, err := b.registry.GetConnection("crm")
	if err != nil {
		return nil, err
	}
	return crmv1.NewCRMServiceClient(conn), nil
}

// getTenantID extracts the tenant identifier from the request context.
// In single-tenant mode, the user ID is used as the tenant identifier.
// Multi-tenant support will extract tenant from JWT claims in a future phase.
func getTenantID(r *http.Request) string {
	return middleware.GetUserID(r.Context())
}

// RegisterRoutes registers all Finance HTTP routes.
func (b *BizRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Company Settings
	r.Route("/api/v1/finance/settings", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleGetCompanySettings)
		r.With(middleware.RequirePermission("finance", "admin")).Put("/", b.HandleUpdateCompanySettings)
	})

	// Quotes
	r.Route("/api/v1/finance/quotes", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "write")).Post("/", b.HandleCreateQuote)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListQuotes)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}", b.HandleGetQuote)
		r.With(middleware.RequirePermission("finance", "write")).Put("/{id}", b.HandleUpdateQuote)
		r.With(middleware.RequirePermission("finance", "delete")).Delete("/{id}", b.HandleDeleteQuote)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/send", b.HandleSendQuote)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/accept", b.HandleAcceptQuote)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/reject", b.HandleRejectQuote)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/convert", b.HandleConvertQuoteToInvoice)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}/pdf", b.HandleGenerateQuotePDF)
	})

	// Invoices
	r.Route("/api/v1/finance/invoices", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "write")).Post("/", b.HandleCreateInvoice)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListInvoices)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}", b.HandleGetInvoice)
		r.With(middleware.RequirePermission("finance", "write")).Put("/{id}", b.HandleUpdateInvoice)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/send", b.HandleSendInvoice)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/pay", b.HandleMarkInvoicePaid)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/cancel", b.HandleCancelInvoice)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}/pdf", b.HandleGenerateInvoicePDF)
		// Payments nested under invoices
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/payments", b.HandleRecordPayment)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}/payments", b.HandleListPayments)
	})

	// Credit Notes
	r.Route("/api/v1/finance/credit-notes", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "write")).Post("/", b.HandleCreateCreditNote)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListCreditNotes)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}", b.HandleGetCreditNote)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/send", b.HandleSendCreditNote)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}/pdf", b.HandleGenerateCreditNotePDF)
	})

	// Payments (top-level for delete by payment ID)
	r.Route("/api/v1/finance/payments", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "delete")).Delete("/{id}", b.HandleDeletePayment)
	})

	// Dunning
	r.Route("/api/v1/finance/dunning", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListDunnings)
		r.With(middleware.RequirePermission("finance", "write")).Post("/detect", b.HandleCreateDunning)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/send", b.HandleSendDunning)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/escalate", b.HandleEscalateDunning)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}/pdf", b.HandleGenerateDunningPDF)
		r.With(middleware.RequirePermission("finance", "read")).Get("/config", b.HandleGetDunningConfig)
		r.With(middleware.RequirePermission("finance", "admin")).Put("/config", b.HandleUpdateDunningConfig)
	})

	// Dashboard
	r.Route("/api/v1/finance/dashboard", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleGetFinanceDashboard)
	})

	// DATEV Export
	r.Route("/api/v1/finance/export", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Post("/datev", b.HandleExportDATEV)
	})

	// Deal-to-Quote conversion
	r.Route("/api/v1/finance/deals/{dealId}/quote", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "write")).Post("/", b.HandleCreateQuoteFromDeal)
	})
}

// ============================================================================
// Company Settings Handlers
// ============================================================================

func (b *BizRoutes) HandleGetCompanySettings(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	resp, err := client.GetCompanySettings(r.Context(), &bizv1.GetCompanySettingsRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Settings)
}

type updateCompanySettingsRequest struct {
	Name                    string `json:"name"`
	Street                  string `json:"street"`
	PLZ                     string `json:"plz"`
	City                    string `json:"city"`
	Country                 string `json:"country"`
	Steuernummer            string `json:"steuernummer"`
	UstIdNr                 string `json:"ust_id_nr"`
	Handelsregister         string `json:"handelsregister"`
	BankName                string `json:"bank_name"`
	IBAN                    string `json:"iban"`
	BIC                     string `json:"bic"`
	LogoURL                 string `json:"logo_url"`
	AccentColor             string `json:"accent_color"`
	IsKleinunternehmer      bool   `json:"is_kleinunternehmer"`
	DefaultPaymentTermsDays int32  `json:"default_payment_terms_days"`
	DefaultQuoteValidityDays int32 `json:"default_quote_validity_days"`
	Basiszinssatz           string `json:"basiszinssatz"`
}

func (b *BizRoutes) HandleUpdateCompanySettings(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	var req updateCompanySettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateCompanySettings(r.Context(), &bizv1.UpdateCompanySettingsRequest{
		TenantId: tenantID,
		Settings: &bizv1.CompanySettings{
			TenantId:                 tenantID,
			Name:                     req.Name,
			Street:                   req.Street,
			Plz:                      req.PLZ,
			City:                     req.City,
			Country:                  req.Country,
			Steuernummer:             req.Steuernummer,
			UstIdNr:                  req.UstIdNr,
			Handelsregister:          req.Handelsregister,
			BankName:                 req.BankName,
			Iban:                     req.IBAN,
			Bic:                      req.BIC,
			LogoUrl:                  req.LogoURL,
			AccentColor:              req.AccentColor,
			IsKleinunternehmer:       req.IsKleinunternehmer,
			DefaultPaymentTermsDays:  req.DefaultPaymentTermsDays,
			DefaultQuoteValidityDays: req.DefaultQuoteValidityDays,
			Basiszinssatz:            req.Basiszinssatz,
		},
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Settings)
}

// ============================================================================
// Quote Handlers
// ============================================================================

type createQuoteRequest struct {
	Customer   *bizv1.CustomerSnapshot `json:"customer"`
	LineItems  []*bizv1.LineItem       `json:"line_items"`
	TaxMode    string                  `json:"tax_mode"`
	ValidUntil string                  `json:"valid_until"`
	Notes      string                  `json:"notes"`
	DealID     string                  `json:"deal_id"`
}

func (b *BizRoutes) HandleCreateQuote(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	userID := middleware.GetUserID(r.Context())

	var req createQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	tenantID := getTenantID(r)
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

	tenantID := getTenantID(r)
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
	TaxMode    string                  `json:"tax_mode"`
	ValidUntil string                  `json:"valid_until"`
	Notes      string                  `json:"notes"`
}

func (b *BizRoutes) HandleUpdateQuote(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	id := chi.URLParam(r, "id")

	var req updateQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	tenantID := getTenantID(r)
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

	tenantID := getTenantID(r)
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

	tenantID := getTenantID(r)
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

	tenantID := getTenantID(r)
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
	InvoiceDate  string `json:"invoice_date"`
	DueDate      string `json:"due_date"`
	DeliveryDate string `json:"delivery_date"`
	PaymentTerms string `json:"payment_terms"`
}

func (b *BizRoutes) HandleConvertQuoteToInvoice(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	id := chi.URLParam(r, "id")

	var req convertQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	tenantID := getTenantID(r)
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
// Invoice Handlers
// ============================================================================

type createInvoiceRequest struct {
	Customer      *bizv1.CustomerSnapshot `json:"customer"`
	LineItems     []*bizv1.LineItem       `json:"line_items"`
	TaxMode       string                  `json:"tax_mode"`
	InvoiceDate   string                  `json:"invoice_date"`
	DeliveryDate  string                  `json:"delivery_date"`
	DueDate       string                  `json:"due_date"`
	PaymentTerms  string                  `json:"payment_terms"`
	Notes         string                  `json:"notes"`
	SourceQuoteID string                  `json:"source_quote_id"`
}

func (b *BizRoutes) HandleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	userID := middleware.GetUserID(r.Context())

	var req createInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateInvoice(r.Context(), &bizv1.CreateInvoiceRequest{
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
	})
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

	tenantID := getTenantID(r)
	page, pageSize := parsePagination(r, 1, 50)

	req := &bizv1.ListInvoicesRequest{
		TenantId: tenantID,
		Status:   invoiceStatusToProto(r.URL.Query().Get("status")),
		Page:     int32(page),
		PerPage:  int32(pageSize),
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

	tenantID := getTenantID(r)
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
	TaxMode      string                  `json:"tax_mode"`
	InvoiceDate  string                  `json:"invoice_date"`
	DeliveryDate string                  `json:"delivery_date"`
	DueDate      string                  `json:"due_date"`
	PaymentTerms string                  `json:"payment_terms"`
	Notes        string                  `json:"notes"`
}

func (b *BizRoutes) HandleUpdateInvoice(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	id := chi.URLParam(r, "id")

	var req updateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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

	tenantID := getTenantID(r)
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

	tenantID := getTenantID(r)
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

	tenantID := getTenantID(r)
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

	tenantID := getTenantID(r)
	id := chi.URLParam(r, "id")

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

// ============================================================================
// Credit Note Handlers
// ============================================================================

type createCreditNoteRequest struct {
	OriginalInvoiceID string                  `json:"original_invoice_id"`
	Customer          *bizv1.CustomerSnapshot `json:"customer"`
	LineItems         []*bizv1.LineItem       `json:"line_items"`
	TaxMode           string                  `json:"tax_mode"`
	Reason            string                  `json:"reason"`
}

func (b *BizRoutes) HandleCreateCreditNote(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	userID := middleware.GetUserID(r.Context())

	var req createCreditNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateCreditNote(r.Context(), &bizv1.CreateCreditNoteRequest{
		TenantId:          tenantID,
		OriginalInvoiceId: req.OriginalInvoiceID,
		Customer:          req.Customer,
		LineItems:         req.LineItems,
		TaxMode:           taxModeToProto(req.TaxMode),
		Reason:            req.Reason,
		CreatedBy:         userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.CreditNote)
}

func (b *BizRoutes) HandleListCreditNotes(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListCreditNotes(r.Context(), &bizv1.ListCreditNotesRequest{
		TenantId:  tenantID,
		InvoiceId: r.URL.Query().Get("invoice_id"),
		Page:      int32(page),
		PerPage:   int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"credit_notes": resp.CreditNotes,
		"total":        resp.Total,
	})
}

func (b *BizRoutes) HandleGetCreditNote(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	id := chi.URLParam(r, "id")

	resp, err := client.GetCreditNote(r.Context(), &bizv1.GetCreditNoteRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.CreditNote)
}

func (b *BizRoutes) HandleSendCreditNote(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	id := chi.URLParam(r, "id")

	resp, err := client.SendCreditNote(r.Context(), &bizv1.SendCreditNoteRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.CreditNote)
}

func (b *BizRoutes) HandleGenerateCreditNotePDF(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	id := chi.URLParam(r, "id")

	resp, err := client.GenerateCreditNotePDF(r.Context(), &bizv1.GenerateCreditNotePDFRequest{
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
// Payment Handlers
// ============================================================================

type recordPaymentRequest struct {
	Amount      string `json:"amount"`
	PaymentDate string `json:"payment_date"`
	Method      string `json:"method"`
	Reference   string `json:"reference"`
	Notes       string `json:"notes"`
}

func (b *BizRoutes) HandleRecordPayment(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	userID := middleware.GetUserID(r.Context())
	invoiceID := chi.URLParam(r, "id")

	var req recordPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.RecordPayment(r.Context(), &bizv1.RecordPaymentRequest{
		TenantId:    tenantID,
		InvoiceId:   invoiceID,
		Amount:      req.Amount,
		PaymentDate: req.PaymentDate,
		Method:      paymentMethodToProto(req.Method),
		Reference:   req.Reference,
		Notes:       req.Notes,
		CreatedBy:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.Payment)
}

func (b *BizRoutes) HandleListPayments(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	invoiceID := chi.URLParam(r, "id")
	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListPayments(r.Context(), &bizv1.ListPaymentsRequest{
		TenantId:  tenantID,
		InvoiceId: invoiceID,
		Page:      int32(page),
		PerPage:   int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"payments": resp.Payments,
		"total":    resp.Total,
	})
}

func (b *BizRoutes) HandleDeletePayment(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	id := chi.URLParam(r, "id")

	_, err = client.DeletePayment(r.Context(), &bizv1.DeletePaymentRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// ============================================================================
// Dunning Handlers
// ============================================================================

func (b *BizRoutes) HandleListDunnings(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListDunnings(r.Context(), &bizv1.ListDunningsRequest{
		TenantId:  tenantID,
		InvoiceId: r.URL.Query().Get("invoice_id"),
		Status:    dunningStatusToProto(r.URL.Query().Get("status")),
		Page:      int32(page),
		PerPage:   int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"dunnings": resp.Dunnings,
		"total":    resp.Total,
	})
}

type createDunningRequest struct {
	InvoiceID string `json:"invoice_id"`
	Level     int32  `json:"level"`
	Fee       string `json:"fee"`
	Interest  string `json:"interest"`
}

func (b *BizRoutes) HandleCreateDunning(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	userID := middleware.GetUserID(r.Context())

	var req createDunningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateDunning(r.Context(), &bizv1.CreateDunningRequest{
		TenantId:  tenantID,
		InvoiceId: req.InvoiceID,
		Level:     req.Level,
		Fee:       req.Fee,
		Interest:  req.Interest,
		CreatedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.Dunning)
}

func (b *BizRoutes) HandleSendDunning(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	id := chi.URLParam(r, "id")

	resp, err := client.SendDunning(r.Context(), &bizv1.SendDunningRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Dunning)
}

func (b *BizRoutes) HandleEscalateDunning(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	userID := middleware.GetUserID(r.Context())

	// The escalate endpoint uses the dunning ID from the URL, but the proto
	// uses invoice_id. We need to get the invoice_id from the request body.
	var req struct {
		InvoiceID string `json:"invoice_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.EscalateDunning(r.Context(), &bizv1.EscalateDunningRequest{
		InvoiceId: req.InvoiceID,
		TenantId:  tenantID,
		CreatedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Dunning)
}

func (b *BizRoutes) HandleGenerateDunningPDF(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	id := chi.URLParam(r, "id")

	resp, err := client.GenerateDunningPDF(r.Context(), &bizv1.GenerateDunningPDFRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	respondPDF(w, resp.PdfData, resp.Filename)
}

func (b *BizRoutes) HandleGetDunningConfig(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	resp, err := client.GetDunningConfig(r.Context(), &bizv1.GetDunningConfigRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Config)
}

type updateDunningConfigRequest struct {
	Level1DaysAfterDue    int32  `json:"level1_days_after_due"`
	Level2DaysAfterLevel1 int32  `json:"level2_days_after_level1"`
	Level3DaysAfterLevel2 int32  `json:"level3_days_after_level2"`
	Level1Fee             string `json:"level1_fee"`
	Level2Fee             string `json:"level2_fee"`
	Level3Fee             string `json:"level3_fee"`
}

func (b *BizRoutes) HandleUpdateDunningConfig(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	var req updateDunningConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateDunningConfig(r.Context(), &bizv1.UpdateDunningConfigRequest{
		TenantId: tenantID,
		Config: &bizv1.DunningConfig{
			TenantId:              tenantID,
			Level1DaysAfterDue:    req.Level1DaysAfterDue,
			Level2DaysAfterLevel1: req.Level2DaysAfterLevel1,
			Level3DaysAfterLevel2: req.Level3DaysAfterLevel2,
			Level1Fee:             req.Level1Fee,
			Level2Fee:             req.Level2Fee,
			Level3Fee:             req.Level3Fee,
		},
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Config)
}

// ============================================================================
// Dashboard Handler
// ============================================================================

func (b *BizRoutes) HandleGetFinanceDashboard(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	resp, err := client.GetFinanceDashboard(r.Context(), &bizv1.GetFinanceDashboardRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Dashboard)
}

// ============================================================================
// DATEV Export Handler
// ============================================================================

type exportDATEVRequest struct {
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	FiscalYearStart string `json:"fiscal_year_start"`
}

func (b *BizRoutes) HandleExportDATEV(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	var req exportDATEVRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.ExportDATEV(r.Context(), &bizv1.ExportDATEVRequest{
		TenantId:        tenantID,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		FiscalYearStart: req.FiscalYearStart,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", resp.Filename))
	w.WriteHeader(http.StatusOK)
	w.Write(resp.CsvData)
}

// ============================================================================
// Deal-to-Quote Handler
// ============================================================================

// HandleCreateQuoteFromDeal creates a new draft quote pre-populated with
// customer data from a CRM deal. It fetches the deal, its linked contact
// and company, then creates a quote via the biz service with deal_id set
// (which triggers deal value auto-sync).
func (b *BizRoutes) HandleCreateQuoteFromDeal(w http.ResponseWriter, r *http.Request) {
	bizClient, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	crmClient, err := b.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	tenantID := getTenantID(r)
	userID := middleware.GetUserID(r.Context())
	dealID := chi.URLParam(r, "dealId")

	// 1. Fetch the CRM deal
	dealResp, err := crmClient.GetDeal(r.Context(), &crmv1.GetDealRequest{Id: dealID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	deal := dealResp.GetDeal()

	// 2. Build customer snapshot from deal's contact and company info
	customerName := ""
	customerAddress := ""
	customerEmail := ""

	// Prefer company name if available, fall back to contact name (B2B norm in DACH)
	if deal.GetCompanyName() != "" {
		customerName = deal.GetCompanyName()
	} else if deal.GetContactName() != "" {
		customerName = deal.GetContactName()
	}

	// Try to get more details from the contact if available
	if deal.GetContactId() != "" {
		contactResp, contactErr := crmClient.GetContact(r.Context(), &crmv1.GetContactRequest{Id: deal.GetContactId()})
		if contactErr == nil && contactResp.GetContact() != nil {
			contact := contactResp.GetContact()
			if customerEmail == "" {
				customerEmail = contact.GetEmail()
			}
			if customerName == "" {
				customerName = strings.TrimSpace(contact.GetFirstName() + " " + contact.GetLastName())
			}
		}
	}

	// Try to get address from company if available
	if deal.GetCompanyId() != "" {
		companyResp, companyErr := crmClient.GetCompany(r.Context(), &crmv1.GetCompanyRequest{Id: deal.GetCompanyId()})
		if companyErr == nil && companyResp.GetCompany() != nil {
			company := companyResp.GetCompany()
			if customerAddress == "" {
				// Build address from company fields
				parts := []string{}
				if company.GetAddress() != "" {
					parts = append(parts, company.GetAddress())
				}
				if company.GetCity() != "" {
					parts = append(parts, company.GetCity())
				}
				if company.GetCountry() != "" {
					parts = append(parts, company.GetCountry())
				}
				customerAddress = strings.Join(parts, ", ")
			}
			if customerName == "" {
				customerName = company.GetName()
			}
		}
	}

	// 3. Create quote via biz service with deal_id and pre-populated customer
	quoteResp, err := bizClient.CreateQuote(r.Context(), &bizv1.CreateQuoteRequest{
		TenantId: tenantID,
		Customer: &bizv1.CustomerSnapshot{
			Name:    customerName,
			Address: customerAddress,
			Email:   customerEmail,
		},
		TaxMode:   bizv1.TaxMode_TAX_MODE_STANDARD,
		DealId:    dealID,
		CreatedBy: userID,
		// LineItems left empty -- user fills in the quote form
		// ValidUntil left empty -- uses company default
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, quoteResp.GetQuote())
}

// ============================================================================
// Proto Enum Helpers
// ============================================================================

func taxModeToProto(mode string) bizv1.TaxMode {
	switch mode {
	case "standard":
		return bizv1.TaxMode_TAX_MODE_STANDARD
	case "reverse_charge":
		return bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE
	case "kleinunternehmer":
		return bizv1.TaxMode_TAX_MODE_KLEINUNTERNEHMER
	default:
		return bizv1.TaxMode_TAX_MODE_UNSPECIFIED
	}
}

func quoteStatusToProto(status string) bizv1.QuoteStatus {
	switch status {
	case "draft":
		return bizv1.QuoteStatus_QUOTE_DRAFT
	case "sent":
		return bizv1.QuoteStatus_QUOTE_SENT
	case "accepted":
		return bizv1.QuoteStatus_QUOTE_ACCEPTED
	case "rejected":
		return bizv1.QuoteStatus_QUOTE_REJECTED
	case "expired":
		return bizv1.QuoteStatus_QUOTE_EXPIRED
	default:
		return bizv1.QuoteStatus_QUOTE_STATUS_UNSPECIFIED
	}
}

func invoiceStatusToProto(status string) bizv1.InvoiceStatus {
	switch status {
	case "draft":
		return bizv1.InvoiceStatus_INVOICE_DRAFT
	case "sent":
		return bizv1.InvoiceStatus_INVOICE_SENT
	case "paid":
		return bizv1.InvoiceStatus_INVOICE_PAID
	case "overdue":
		return bizv1.InvoiceStatus_INVOICE_OVERDUE
	case "cancelled":
		return bizv1.InvoiceStatus_INVOICE_CANCELLED
	default:
		return bizv1.InvoiceStatus_INVOICE_STATUS_UNSPECIFIED
	}
}

func dunningStatusToProto(status string) bizv1.DunningStatus {
	switch status {
	case "draft":
		return bizv1.DunningStatus_DUNNING_DRAFT
	case "sent":
		return bizv1.DunningStatus_DUNNING_SENT
	case "paid":
		return bizv1.DunningStatus_DUNNING_PAID
	default:
		return bizv1.DunningStatus_DUNNING_STATUS_UNSPECIFIED
	}
}

func paymentMethodToProto(method string) bizv1.PaymentMethod {
	switch method {
	case "bank_transfer":
		return bizv1.PaymentMethod_PAYMENT_METHOD_BANK_TRANSFER
	case "cash":
		return bizv1.PaymentMethod_PAYMENT_METHOD_CASH
	case "credit_card":
		return bizv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD
	case "other":
		return bizv1.PaymentMethod_PAYMENT_METHOD_OTHER
	default:
		return bizv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

// respondPDF writes PDF binary data as an HTTP response with proper headers.
func respondPDF(w http.ResponseWriter, pdfData []byte, filename string) {
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfData)))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfData)
}
