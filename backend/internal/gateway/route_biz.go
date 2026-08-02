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
	// Additive RBAC guards (RequirePermissionAny keeps the legacy coarse token
	// valid while granting the capability-catalog.ts fine keys, per the FE gate
	// calls that reach each route). Company settings, invoice import/lock/
	// validate-number, dunning status/notice, payment delete, open-items, bank
	// statements/transactions, incoming-invoices, journal/stats and gobd-archive
	// have zero FE caller today (grepped across desktop/src — several are
	// pre-wired for the still-`todo` Block D FE-only finance units) and stay on
	// the legacy-only guard; see JOURNAL.md for the full inventory.
	invoiceRead := middleware.RequirePermissionAny([2]string{"finance", "read"}, [2]string{"finance:invoice", "read"})
	invoiceCreate := middleware.RequirePermissionAny([2]string{"finance", "write"}, [2]string{"finance:invoice", "create"})
	invoiceEdit := middleware.RequirePermissionAny([2]string{"finance", "write"}, [2]string{"finance:invoice", "edit"})
	invoiceDelete := middleware.RequirePermissionAny([2]string{"finance", "write"}, [2]string{"finance:invoice", "delete"})
	invoiceSend := middleware.RequirePermissionAny([2]string{"finance", "write"}, [2]string{"finance:invoice", "send"})
	// Credit-note create is reached both for a fresh credit note (invoice:create,
	// FinanzenPage.tsx canCreateInvoice) and for a storno of an issued invoice
	// (invoice:delete, canDeleteInvoice) — both must open the route.
	creditNoteCreate := middleware.RequirePermissionAny([2]string{"finance", "write"}, [2]string{"finance:invoice", "create"}, [2]string{"finance:invoice", "delete"})

	quoteRead := middleware.RequirePermissionAny([2]string{"finance", "read"}, [2]string{"finance:quote", "read"})
	// Quote edit/accept/reject are gated by quote:create in QuoteDetailPanel.tsx
	// ("Edit -> quote:create", "Accept/Reject -> quote:create") — there is no
	// separate quote:edit catalogue key.
	quoteWrite := middleware.RequirePermissionAny([2]string{"finance", "write"}, [2]string{"finance:quote", "create"})
	quoteSend := middleware.RequirePermissionAny([2]string{"finance", "write"}, [2]string{"finance:quote", "send"})
	// Convert-to-invoice creates an invoice (QuoteDetailPanel.tsx "Convert -> invoice:create").
	quoteConvert := middleware.RequirePermissionAny([2]string{"finance", "write"}, [2]string{"finance:invoice", "create"})

	dunningRun := middleware.RequirePermissionAny([2]string{"finance", "read"}, [2]string{"finance:dunning", "run"})
	dunningRunWrite := middleware.RequirePermissionAny([2]string{"finance", "write"}, [2]string{"finance:dunning", "run"})
	dunningSettings := middleware.RequirePermissionAny([2]string{"finance", "read"}, [2]string{"finance:settings", "manage"})
	dunningSettingsWrite := middleware.RequirePermissionAny([2]string{"finance", "admin"}, [2]string{"finance:settings", "manage"})

	exportRun := middleware.RequirePermissionAny([2]string{"finance", "read"}, [2]string{"finance:export", "run"})

	// Company Settings
	r.Route("/api/v1/finance/settings", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleGetCompanySettings)
		r.With(middleware.RequirePermission("finance", "admin")).Put("/", b.HandleUpdateCompanySettings)
	})

	// Quotes
	r.Route("/api/v1/finance/quotes", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(quoteWrite).Post("/", b.HandleCreateQuote)
		r.With(quoteRead).Get("/", b.HandleListQuotes)
		r.With(quoteRead).Get("/{id}", b.HandleGetQuote)
		r.With(quoteWrite).Put("/{id}", b.HandleUpdateQuote)
		r.With(middleware.RequirePermission("finance", "delete")).Delete("/{id}", b.HandleDeleteQuote)
		r.With(quoteSend).Post("/{id}/send", b.HandleSendQuote)
		r.With(quoteWrite).Post("/{id}/accept", b.HandleAcceptQuote)
		r.With(quoteWrite).Post("/{id}/reject", b.HandleRejectQuote)
		r.With(quoteConvert).Post("/{id}/convert", b.HandleConvertQuoteToInvoice)
		r.With(quoteRead).Get("/{id}/pdf", b.HandleGenerateQuotePDF)
	})

	// Invoices
	r.Route("/api/v1/finance/invoices", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(invoiceCreate).Post("/", b.HandleCreateInvoice)
		// E-Rechnung Eingang: PDF (ZUGFeRD) oder XML (XRechnung/CII) importieren
		r.With(middleware.RequirePermission("finance", "write")).Post("/import", b.HandleImportInvoice)
		r.With(invoiceRead).Get("/", b.HandleListInvoices)
		r.With(invoiceRead).Get("/{id}", b.HandleGetInvoice)
		r.With(invoiceEdit).Put("/{id}", b.HandleUpdateInvoice)
		r.With(invoiceSend).Post("/{id}/send", b.HandleSendInvoice)
		r.With(invoiceEdit).Post("/{id}/mark-paid", b.HandleMarkInvoicePaid)
		r.With(invoiceDelete).Post("/{id}/cancel", b.HandleCancelInvoice)
		r.With(invoiceRead).Get("/{id}/pdf", b.HandleGenerateInvoicePDF)
		// E-Rechnung Ausgang: XRechnung UBL-XML oder ZUGFeRD-PDF exportieren
		r.With(middleware.RequirePermission("finance", "read")).Post("/{id}/erechnung", b.HandleGenerateEInvoice)
		r.With(middleware.RequirePermission("finance", "admin")).Post("/{id}/lock", b.HandleLockInvoice)
		// Payments nested under invoices
		r.With(invoiceEdit).Post("/{id}/payments", b.HandleRecordPayment)
		r.With(invoiceRead).Get("/{id}/payments", b.HandleListPayments)
		// GoBD number validation
		r.With(middleware.RequirePermission("finance", "read")).Get("/validate-number", b.HandleValidateInvoiceNumber)
	})

	// Recurring invoices (Abo-Rechnungen)
	r.Route("/api/v1/finance/recurring", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(invoiceRead).Get("/", b.HandleListRecurringInvoices)
		r.With(invoiceCreate).Post("/", b.HandleCreateRecurringInvoice)
		r.With(invoiceRead).Get("/{id}", b.HandleGetRecurringInvoice)
		r.With(invoiceEdit).Put("/{id}", b.HandleUpdateRecurringInvoice)
		r.With(invoiceDelete).Delete("/{id}", b.HandleDeleteRecurringInvoice)
		r.With(invoiceEdit).Post("/{id}/pause", b.HandlePauseRecurringInvoice)
		r.With(invoiceEdit).Post("/{id}/resume", b.HandleResumeRecurringInvoice)
		r.With(invoiceCreate).Post("/{id}/generate", b.HandleGenerateRecurringInvoice)
	})

	// Credit Notes
	r.Route("/api/v1/finance/credit-notes", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(creditNoteCreate).Post("/", b.HandleCreateCreditNote)
		r.With(invoiceRead).Get("/", b.HandleListCreditNotes)
		r.With(invoiceRead).Get("/{id}", b.HandleGetCreditNote)
		r.With(invoiceSend).Post("/{id}/send", b.HandleSendCreditNote)
		r.With(invoiceRead).Get("/{id}/pdf", b.HandleGenerateCreditNotePDF)
	})

	// Payments (top-level for delete by payment ID)
	r.Route("/api/v1/finance/payments", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "delete")).Delete("/{id}", b.HandleDeletePayment)
	})

	// Dunning
	r.Route("/api/v1/finance/dunning", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(dunningRun).Get("/", b.HandleListDunnings)
		r.With(dunningRunWrite).Post("/detect", b.HandleCreateDunning)
		r.With(dunningRunWrite).Post("/{id}/send", b.HandleSendDunning)
		r.With(dunningRunWrite).Post("/{id}/escalate", b.HandleEscalateDunning)
		r.With(dunningRun).Get("/{id}/pdf", b.HandleGenerateDunningPDF)
		r.With(dunningSettings).Get("/config", b.HandleGetDunningConfig)
		r.With(dunningSettingsWrite).Put("/config", b.HandleUpdateDunningConfig)
		// GoBD dunning gaps (Sprint 2 / Wave 1.B)
		r.With(middleware.RequirePermission("finance", "admin")).Put("/{id}/status", b.HandleUpdateDunningStatus)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/notice", b.HandleSendDunningNotice)
	})

	// Open items (Offene Posten) — receivables aging, the read side of dunning
	r.Route("/api/v1/finance/open-items", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListOpenItems)
	})

	// Document chains (Belegkette): quote -> invoice -> payment/dunning/credit note
	r.Route("/api/v1/finance/document-chains", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListDocumentChains)
	})

	// Time entries (Stunden -> Rechnung): billable time entries for invoicing.
	r.Route("/api/v1/finance/time-entries", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListTimeEntries)
	})

	// Bank statements (Zahlungsabgleich): CAMT.053 / MT940 import
	r.Route("/api/v1/finance/bank-statements", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListBankStatements)
		r.With(middleware.RequirePermission("finance", "write")).Post("/import", b.HandleImportBankStatement)
		r.With(middleware.RequirePermission("finance", "read")).Get("/{id}", b.HandleGetBankStatement)
	})

	// Bank transactions: the reconciliation queue of the imports above.
	//
	// match/reject-match are what the desktop client calls; reconcile was the
	// same operation under an earlier name and is gone rather than aliased --
	// two names for one booking is how the two drift apart. ignore stays: it is
	// a different decision (not a customer payment at all), not a synonym.
	r.Route("/api/v1/finance/bank-transactions", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListBankTransactions)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/match", b.HandleMatchBankTransaction)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/reject-match", b.HandleRejectBankTransactionMatch)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/ignore", b.HandleIgnoreBankTransaction)
	})

	// Bank accounts (Bankkonten) — Migration 000258.
	//
	// Legacy-only guards, same reason as the expenses below: capability-catalog.ts
	// has no finance:bank-account:* key, so there is no fine key to add
	// additively yet. connect sits on finance:write because it changes stored
	// state, not on read.
	r.Route("/api/v1/finance/bank-accounts", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListBankAccounts)
		r.With(middleware.RequirePermission("finance", "write")).Post("/", b.HandleCreateBankAccount)
		r.With(middleware.RequirePermission("finance", "write")).Patch("/{id}", b.HandleUpdateBankAccount)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/connect", b.HandleConnectBankAccount)
		r.With(middleware.RequirePermission("finance", "delete")).Delete("/{id}", b.HandleDeleteBankAccount)
	})

	// Expenses (Ausgaben) — Migration 000257.
	//
	// Legacy-only guards: capability-catalog.ts has no finance:expense:* key, so
	// there is no fine key to add additively yet. Approve/reject sit on
	// finance:write like every other state change here; the control that makes
	// the approval mean something is the server-side refusal to decide one's own
	// expense (expense.ErrSelfApproval), not the coarseness of this guard. A
	// dedicated finance:expense:approve key belongs in the catalogue -- see
	// JOURNAL.md.
	r.Route("/api/v1/finance/expenses", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListExpenses)
		r.With(middleware.RequirePermission("finance", "write")).Post("/", b.HandleCreateExpense)
		r.With(middleware.RequirePermission("finance", "write")).Put("/{id}", b.HandleUpdateExpense)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/approve", b.HandleApproveExpense)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/reject", b.HandleRejectExpense)
		r.With(middleware.RequirePermission("finance", "write")).Post("/{id}/receipt", b.HandleAttachExpenseReceipt)
		r.With(middleware.RequirePermission("finance", "delete")).Delete("/{id}", b.HandleDeleteExpense)
	})

	// Transactions (Transaktionen): consolidated payment ledger over payments +
	// approved expenses. Legacy-only guards, same reason as expenses above:
	// capability-catalog.ts has no finance:transaction:* key.
	r.Route("/api/v1/finance/transactions", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleListTransactions)
		r.With(middleware.RequirePermission("finance", "delete")).Delete("/{id}", b.HandleDeleteTransaction)
	})

	// Dashboard
	r.Route("/api/v1/finance/dashboard", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("finance", "read")).Get("/", b.HandleGetFinanceDashboard)
	})

	// DATEV + GoBD Export
	r.Route("/api/v1/finance/export", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(exportRun).Post("/datev", b.HandleExportDATEV)
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
		r.With(quoteWrite).Post("/", b.HandleCreateQuoteFromDeal)
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

	response.Proto(w, http.StatusOK, resp.Settings)
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
	DatevBeraterNr           string `json:"datev_berater_nr"`
	DatevMandantNr           string `json:"datev_mandant_nr"`
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
			DatevBeraterNr:           req.DatevBeraterNr,
			DatevMandantNr:           req.DatevMandantNr,
		},
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Settings)
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
	respondFile(w, pdfData, filename, "application/pdf")
}

// respondFile writes binary data as an HTTP response with the given content
// type and a Content-Disposition attachment header.
func respondFile(w http.ResponseWriter, data []byte, filename, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
