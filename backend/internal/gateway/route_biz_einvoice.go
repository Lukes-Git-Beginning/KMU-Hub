package gateway

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// Incoming Invoice (E-Rechnung Eingang) Handlers
// ============================================================================

// maxEInvoiceUploadBytes bounds the accepted e-invoice file size (10 MiB).
const maxEInvoiceUploadBytes = 10 << 20

// HandleImportInvoice accepts a multipart/form-data POST with a "file" field
// containing a PDF (ZUGFeRD) or XML (XRechnung/CII) e-invoice file.
func (b *BizRoutes) HandleImportInvoice(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(maxEInvoiceUploadBytes); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if !isEInvoiceMIME(mimeType) {
		response.Error(w, http.StatusUnsupportedMediaType, "unsupported file type: expected application/pdf, application/xml, or text/xml")
		return
	}

	content, readErr := io.ReadAll(io.LimitReader(file, maxEInvoiceUploadBytes+1))
	if readErr != nil {
		response.Error(w, http.StatusInternalServerError, "failed to read uploaded file")
		return
	}
	if len(content) == 0 {
		response.Error(w, http.StatusBadRequest, "uploaded file is empty")
		return
	}
	if len(content) > maxEInvoiceUploadBytes {
		response.Error(w, http.StatusRequestEntityTooLarge, "uploaded file exceeds 10 MiB limit")
		return
	}

	resp, err := client.ImportIncomingInvoice(r.Context(), &bizv1.ImportIncomingInvoiceRequest{
		TenantId:  tenantID,
		Filename:  header.Filename,
		MimeType:  mimeType,
		Content:   content,
		CreatedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.Invoice)
}

// HandleListIncomingInvoices lists incoming invoices with optional filters.
func (b *BizRoutes) HandleListIncomingInvoices(w http.ResponseWriter, r *http.Request) {
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
	q := r.URL.Query()

	resp, err := client.ListIncomingInvoices(r.Context(), &bizv1.ListIncomingInvoicesRequest{
		TenantId: tenantID,
		Status:   q.Get("status"),
		DateFrom: q.Get("date_from"),
		DateTo:   q.Get("date_to"),
		Page:     int32(page),
		PerPage:  int32(pageSize),
	})
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

// HandleGetIncomingInvoice retrieves a single incoming invoice by ID.
func (b *BizRoutes) HandleGetIncomingInvoice(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.GetIncomingInvoice(r.Context(), &bizv1.GetIncomingInvoiceRequest{
		TenantId: tenantID,
		Id:       id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Invoice)
}

// updateIncomingInvoiceStatusRequest is the JSON body for status update.
type updateIncomingInvoiceStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=reviewed booked rejected"`
}

// HandleUpdateIncomingInvoiceStatus transitions the status of an incoming invoice.
func (b *BizRoutes) HandleUpdateIncomingInvoiceStatus(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[updateIncomingInvoiceStatusRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateIncomingInvoiceStatus(r.Context(), &bizv1.UpdateIncomingInvoiceStatusRequest{
		TenantId:  tenantID,
		Id:        id,
		NewStatus: req.Status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Invoice)
}

// isEInvoiceMIME returns true for MIME types supported by the e-invoice import endpoint.
func isEInvoiceMIME(mimeType string) bool {
	switch mimeType {
	case "application/pdf", "application/xml", "text/xml",
		"application/octet-stream": // browsers may send this for .pdf/.xml without explicit type
		return true
	default:
		return false
	}
}
