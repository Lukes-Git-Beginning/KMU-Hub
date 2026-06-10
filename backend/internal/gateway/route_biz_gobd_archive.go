package gateway

// GoBD Belegarchiv HTTP handlers (§147 AO).
// Route registration: /api/v1/finance/gobd-archive
// Pattern: multipart for ArchiveDocument, JSON for read/annotate operations.

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// HandleArchiveDocument handles POST /api/v1/finance/gobd-archive
// Multipart form with field "file" (the document) and form fields:
//   - doc_type: required
//   - source_invoice_id: optional
func (b *BizRoutes) HandleArchiveDocument(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	userID := middleware.GetUserID(r.Context())

	// Parse multipart (max 55 MiB: 50 MiB file + 5 MiB form fields)
	if err := r.ParseMultipartForm(55 << 20); err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	f, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing field 'file'", http.StatusBadRequest)
		return
	}
	defer f.Close()

	fileBytes, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusBadRequest)
		return
	}

	docType := r.FormValue("doc_type")
	if docType == "" {
		http.Error(w, "missing field 'doc_type'", http.StatusBadRequest)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Allow up to 60 MiB send size for file bytes — default gRPC limit is 4 MiB.
	resp, err := client.ArchiveDocument(r.Context(), &bizv1.ArchiveDocumentRequest{
		TenantId:         tenantID,
		DocType:          docType,
		SourceInvoiceId:  r.FormValue("source_invoice_id"),
		OriginalFilename: header.Filename,
		MimeType:         mimeType,
		Content:          fileBytes,
		ArchivedBy:       userID,
	}, grpc.MaxCallSendMsgSize(60<<20))
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.Document)
}

// HandleArchiveInvoiceDocument handles POST /api/v1/finance/gobd-archive/from-invoice/{invoiceId}
func (b *BizRoutes) HandleArchiveInvoiceDocument(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	userID := middleware.GetUserID(r.Context())
	invoiceID := chi.URLParam(r, "invoiceId")

	resp, err := client.ArchiveInvoiceDocument(r.Context(), &bizv1.ArchiveInvoiceDocumentRequest{
		TenantId:   tenantID,
		InvoiceId:  invoiceID,
		ArchivedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.Document)
}

// HandleListGobdDocuments handles GET /api/v1/finance/gobd-archive
func (b *BizRoutes) HandleListGobdDocuments(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	resp, err := client.ListGobdDocuments(r.Context(), &bizv1.ListGobdDocumentsRequest{
		TenantId:        tenantID,
		DocType:         q.Get("doc_type"),
		SourceInvoiceId: q.Get("source_invoice_id"),
		DateFrom:        q.Get("date_from"),
		DateTo:          q.Get("date_to"),
		Page:            parsePageParam(q.Get("page")),
		PerPage:         parsePageParam(q.Get("per_page")),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleGetGobdDocument handles GET /api/v1/finance/gobd-archive/{id}
func (b *BizRoutes) HandleGetGobdDocument(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}

	resp, err := client.GetGobdDocument(r.Context(), &bizv1.GetGobdDocumentRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleDownloadGobdDocument handles GET /api/v1/finance/gobd-archive/{id}/download
func (b *BizRoutes) HandleDownloadGobdDocument(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	userID := middleware.GetUserID(r.Context())

	resp, err := client.DownloadGobdDocument(r.Context(), &bizv1.DownloadGobdDocumentRequest{
		TenantId:    tenantID,
		Id:          chi.URLParam(r, "id"),
		RequestedBy: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleAddDocumentAnnotation handles POST /api/v1/finance/gobd-archive/{id}/annotations
type addAnnotationRequest struct {
	Note string `json:"note" validate:"required,min=1,max=2000"`
}

func (b *BizRoutes) HandleAddDocumentAnnotation(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[addAnnotationRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.AddDocumentAnnotation(r.Context(), &bizv1.AddDocumentAnnotationRequest{
		TenantId:   tenantID,
		DocumentId: chi.URLParam(r, "id"),
		AddedBy:    userID,
		Note:       req.Note,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

// parsePageParam converts a query string to int32, defaulting to 0 (server applies its default).
func parsePageParam(s string) int32 {
	if s == "" {
		return 0
	}
	var n int32
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int32(c-'0')
	}
	return n
}
