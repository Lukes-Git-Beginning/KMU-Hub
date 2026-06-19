package gateway

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
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

// getTenantID extracts and validates the tenant UUID from the JWT claims in the request context.
// Returns ErrMissingTenantID (from middleware) when the tid claim is absent or not a valid UUID.
// Callers must respond with 401 Unauthorized on error.
func getTenantID(r *http.Request) (string, error) {
	id, err := middleware.GetTenantID(r.Context())
	if err != nil {
		return "", err
	}
	return id.String(), nil
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
		// E-Rechnung Eingang: PDF (ZUGFeRD) oder XML (XRechnung/CII) importieren
		r.With(middleware.RequirePermission("finance", "write")).Post("/import", b.HandleImportInvoice)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListInvoices)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}", b.HandleGetInvoice)
		r.With(middleware.RequirePermission("finance", "write")).Put("/{id}", b.HandleUpdateInvoice)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/send", b.HandleSendInvoice)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/pay", b.HandleMarkInvoicePaid)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/cancel", b.HandleCancelInvoice)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}/pdf", b.HandleGenerateInvoicePDF)
		r.With(middleware.RequirePermission("finance", "admin")).Post("/{id}/lock", b.HandleLockInvoice)
		// Payments nested under invoices
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/payments", b.HandleRecordPayment)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}/payments", b.HandleListPayments)
		// GoBD number validation
		r.With(middleware.RequirePermission("finance", "read")).Get("/validate-number", b.HandleValidateInvoiceNumber)
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
		// GoBD dunning gaps (Sprint 2 / Wave 1.B)
		r.With(middleware.RequirePermission("finance", "admin")).Put("/{id}/status", b.HandleUpdateDunningStatus)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/notice", b.HandleSendDunningNotice)
	})

	// Dashboard
	r.Route("/api/v1/finance/dashboard", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleGetFinanceDashboard)
	})

	// DATEV + GoBD Export
	r.Route("/api/v1/finance/export", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Post("/datev", b.HandleExportDATEV)
		r.With(middleware.RequirePermission("finance", "admin")).Post("/gobd", b.HandleGenerateGoBDExport)
	})

	// GoBD Journal summary
	r.Route("/api/v1/finance/journal", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/summary", b.HandleGetJournalSummary)
	})

	// Payment Stats
	r.Route("/api/v1/finance/stats", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/payments", b.HandleGetPaymentStats)
	})

	// Deal-to-Quote conversion
	r.Route("/api/v1/finance/deals/{dealId}/quote", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "write")).Post("/", b.HandleCreateQuoteFromDeal)
	})

	// GoBD Belegarchiv (§147 AO — revisionssichere Beleg-Ablage)
	r.Route("/api/v1/finance/gobd-archive", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("gobd-archive", "write")).Post("/", b.HandleArchiveDocument)
		r.With(middleware.RequirePermission("gobd-archive", "write")).Post("/from-invoice/{invoiceId}", b.HandleArchiveInvoiceDocument)
		r.With(middleware.RequirePermission("gobd-archive", "read")).Get("/", b.HandleListGobdDocuments)
		r.With(middleware.RequirePermission("gobd-archive", "read")).Get("/{id}", b.HandleGetGobdDocument)
		r.With(middleware.RequirePermission("gobd-archive", "read")).Get("/{id}/download", b.HandleDownloadGobdDocument)
		r.With(middleware.RequirePermission("gobd-archive", "write")).Post("/{id}/annotations", b.HandleAddDocumentAnnotation)
	})

	// Incoming Invoices (E-Rechnung Eingang) — Liste + Detail + Status-Übergang
	r.Route("/api/v1/finance/incoming-invoices", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListIncomingInvoices)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}", b.HandleGetIncomingInvoice)
		r.With(middleware.RequirePermission("finance", "write")).Patch("/{id}/status", b.HandleUpdateIncomingInvoiceStatus)
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

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
	Name                     string `json:"name"`
	Street                   string `json:"street"`
	PLZ                      string `json:"plz"                validate:"omitempty,plz_dach"`
	City                     string `json:"city"`
	Country                  string `json:"country"`
	Steuernummer             string `json:"steuernummer"        validate:"omitempty,steuernr"`
	UstIdNr                  string `json:"ust_id_nr"           validate:"omitempty,ustid_dach"`
	Handelsregister          string `json:"handelsregister"`
	BankName                 string `json:"bank_name"`
	IBAN                     string `json:"iban"               validate:"omitempty,iban"`
	BIC                      string `json:"bic"                validate:"omitempty,bic"`
	LogoURL                  string `json:"logo_url"           validate:"omitempty,url"`
	AccentColor              string `json:"accent_color"`
	IsKleinunternehmer       bool   `json:"is_kleinunternehmer"`
	DefaultPaymentTermsDays  int32  `json:"default_payment_terms_days"  validate:"omitempty,gte=0"`
	DefaultQuoteValidityDays int32  `json:"default_quote_validity_days" validate:"omitempty,gte=0"`
	Basiszinssatz            string `json:"basiszinssatz"`
}

func (b *BizRoutes) HandleUpdateCompanySettings(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[updateCompanySettingsRequest](w, r)
	if !ok {
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
	_, _ = w.Write(pdfData)
}
