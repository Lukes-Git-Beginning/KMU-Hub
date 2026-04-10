package gateway

import (
	"encoding/json"
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
	_, _ = w.Write(resp.CsvData)
}
