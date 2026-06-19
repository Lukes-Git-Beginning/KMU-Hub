package gateway

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// Credit Note Handlers
// ============================================================================

type createCreditNoteRequest struct {
	OriginalInvoiceID string                  `json:"original_invoice_id" validate:"omitempty,uuid"`
	Customer          *bizv1.CustomerSnapshot `json:"customer"            validate:"required"`
	LineItems         []*bizv1.LineItem       `json:"line_items"          validate:"required,min=1"`
	TaxMode           string                  `json:"tax_mode"            validate:"required,oneof=standard reverse_charge kleinunternehmer"`
	Reason            string                  `json:"reason"              validate:"required"`
}

func (b *BizRoutes) HandleCreateCreditNote(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[createCreditNoteRequest](w, r)
	if !ok {
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
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

	response.JSON(w, http.StatusOK, map[string]any{
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
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
	Amount      string `json:"amount"       validate:"required,decimal_gt0"`
	PaymentDate string `json:"payment_date" validate:"required,datetime=2006-01-02"`
	Method      string `json:"method"       validate:"required,oneof=bank_transfer cash credit_card other"`
	Reference   string `json:"reference"`
	Notes       string `json:"notes"`
}

func (b *BizRoutes) HandleRecordPayment(w http.ResponseWriter, r *http.Request) {
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
	invoiceID := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[recordPaymentRequest](w, r)
	if !ok {
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
		// Forward the client Idempotency-Key (already required by the idempotency
		// middleware in HardMode) to the biz service for DB-level dedup (F5).
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
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

	response.JSON(w, http.StatusOK, map[string]any{
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
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

	response.JSON(w, http.StatusOK, map[string]any{
		"dunnings": resp.Dunnings,
		"total":    resp.Total,
	})
}

type createDunningRequest struct {
	InvoiceID string `json:"invoice_id" validate:"required,uuid"`
	Level     int32  `json:"level"      validate:"required,gte=1,lte=3"`
	Fee       string `json:"fee"`
	Interest  string `json:"interest"`
}

func (b *BizRoutes) HandleCreateDunning(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[createDunningRequest](w, r)
	if !ok {
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	// The escalate endpoint uses the dunning ID from the URL, but the proto
	// uses invoice_id. We need to get the invoice_id from the request body.
	type escalateDunningRequest struct {
		InvoiceID string `json:"invoice_id" validate:"required,uuid"`
	}
	req, ok := decodeAndValidate[escalateDunningRequest](w, r)
	if !ok {
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

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
	Level1DaysAfterDue    int32  `json:"level1_days_after_due"    validate:"gte=0"`
	Level2DaysAfterLevel1 int32  `json:"level2_days_after_level1" validate:"gte=0"`
	Level3DaysAfterLevel2 int32  `json:"level3_days_after_level2" validate:"gte=0"`
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[updateDunningConfigRequest](w, r)
	if !ok {
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

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

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
	StartDate       string `json:"start_date"        validate:"required,datetime=2006-01-02"`
	EndDate         string `json:"end_date"          validate:"required,datetime=2006-01-02"`
	FiscalYearStart string `json:"fiscal_year_start" validate:"omitempty,datetime=2006-01-02"`
}

func (b *BizRoutes) HandleExportDATEV(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[exportDATEVRequest](w, r)
	if !ok {
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
	_, _ = w.Write(resp.CsvData)
}

// ============================================================================
// GoBD Journal & Compliance Handlers (Sprint 2 / Wave 1.B)
// ============================================================================

// HandleGetJournalSummary returns the gap-detection journal summary for a fiscal year.
// GET /api/v1/finance/journal/summary?year=2026
func (b *BizRoutes) HandleGetJournalSummary(w http.ResponseWriter, r *http.Request) {
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
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		response.Error(w, http.StatusBadRequest, "year query parameter is required")
		return
	}
	var year int32
	if _, scanErr := fmt.Sscanf(yearStr, "%d", &year); scanErr != nil || year < 2000 || year > 2100 {
		response.Error(w, http.StatusBadRequest, "year must be a 4-digit number between 2000 and 2100")
		return
	}

	resp, err := client.GetJournalSummary(r.Context(), &bizv1.GetJournalSummaryRequest{
		TenantId: tenantID,
		Year:     year,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleValidateInvoiceNumber checks format validity and uniqueness of an invoice number.
// GET /api/v1/finance/invoices/validate-number?number=RE-2026-0042
func (b *BizRoutes) HandleValidateInvoiceNumber(w http.ResponseWriter, r *http.Request) {
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
	number := r.URL.Query().Get("number")
	if number == "" {
		response.Error(w, http.StatusBadRequest, "number query parameter is required")
		return
	}

	resp, err := client.ValidateInvoiceNumber(r.Context(), &bizv1.ValidateInvoiceNumberRequest{
		TenantId:      tenantID,
		InvoiceNumber: number,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleLockInvoice administratively locks an invoice (GoBD: immutability after lock).
// POST /api/v1/finance/invoices/{id}/lock
func (b *BizRoutes) HandleLockInvoice(w http.ResponseWriter, r *http.Request) {
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
	userID := middleware.GetUserID(r.Context())

	resp, err := client.LockInvoice(r.Context(), &bizv1.LockInvoiceRequest{
		Id:       id,
		TenantId: tenantID,
		LockedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleGetPaymentStats returns aggregated payment statistics for a date range.
// GET /api/v1/finance/stats/payments?from=2026-01-01&to=2026-12-31
func (b *BizRoutes) HandleGetPaymentStats(w http.ResponseWriter, r *http.Request) {
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
	fromDate := r.URL.Query().Get("from")
	toDate := r.URL.Query().Get("to")
	if fromDate == "" || toDate == "" {
		response.Error(w, http.StatusBadRequest, "from and to query parameters are required (YYYY-MM-DD)")
		return
	}

	resp, err := client.GetPaymentStats(r.Context(), &bizv1.GetPaymentStatsRequest{
		TenantId: tenantID,
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleUpdateDunningStatus directly sets a dunning record's status (admin override).
// PUT /api/v1/finance/dunning/{id}/status
func (b *BizRoutes) HandleUpdateDunningStatus(w http.ResponseWriter, r *http.Request) {
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

	type updateDunningStatusRequest struct {
		Status string `json:"status" validate:"required,oneof=draft sent paid"`
	}
	req, ok := decodeAndValidate[updateDunningStatusRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateDunningStatus(r.Context(), &bizv1.UpdateDunningStatusRequest{
		Id:       id,
		TenantId: tenantID,
		Status:   req.Status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.GetDunning())
}

// HandleSendDunningNotice marks a dunning as sent.
// POST /api/v1/finance/dunning/{id}/notice
func (b *BizRoutes) HandleSendDunningNotice(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.SendDunningNotice(r.Context(), &bizv1.SendDunningNoticeRequest{
		Id:       id,
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"dunning":      resp.GetDunning(),
		"email_queued": resp.GetEmailQueued(),
	})
}

// HandleGenerateGoBDExport generates and downloads a GoBD-compliant CSV export.
// POST /api/v1/finance/export/gobd
func (b *BizRoutes) HandleGenerateGoBDExport(w http.ResponseWriter, r *http.Request) {
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

	type generateGoBDExportRequest struct {
		FromDate string `json:"from_date" validate:"required,datetime=2006-01-02"`
		ToDate   string `json:"to_date"   validate:"required,datetime=2006-01-02"`
	}
	req, ok := decodeAndValidate[generateGoBDExportRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.GenerateGoBDExport(r.Context(), &bizv1.GenerateGoBDExportRequest{
		TenantId: tenantID,
		FromDate: req.FromDate,
		ToDate:   req.ToDate,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	filename := resp.GetFilename()
	if filename == "" {
		filename = "gobd-export.csv"
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetCsvData())
}
