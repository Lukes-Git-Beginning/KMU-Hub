package gateway

// Bank statement import and reconciliation (Zahlungsabgleich) — Migration 000247.

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// maxBankStatementUploadBytes bounds an accepted statement upload (10 MiB),
// matching banking.MaxStatementBytes. A month of a busy account stays far below
// it; the limit is here so an oversized upload is refused before it is buffered.
const maxBankStatementUploadBytes = 10 << 20

// HandleImportBankStatement accepts a multipart/form-data POST with a "file"
// field carrying a CAMT.053 XML or MT940 text statement.
// POST /api/v1/finance/bank-statements/import
func (b *BizRoutes) HandleImportBankStatement(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(maxBankStatementUploadBytes); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	// One byte past the limit is read on purpose, so an oversized upload is
	// detected rather than silently truncated to a valid-looking statement.
	content, readErr := io.ReadAll(io.LimitReader(file, maxBankStatementUploadBytes+1))
	if readErr != nil {
		response.Error(w, http.StatusInternalServerError, "failed to read uploaded file")
		return
	}
	if len(content) == 0 {
		response.Error(w, http.StatusBadRequest, "uploaded file is empty")
		return
	}
	if len(content) > maxBankStatementUploadBytes {
		response.Error(w, http.StatusRequestEntityTooLarge, "uploaded file exceeds 10 MiB limit")
		return
	}

	resp, err := client.ImportBankStatement(r.Context(), &bizv1.ImportBankStatementRequest{
		TenantId: tenantID,
		Filename: header.Filename,
		Content:  content,
		UserId:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	statement, err := hrMarshalProto(resp.GetStatement())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	transactions, err := hrMarshalSlice(resp.GetTransactions())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// 200 rather than 201 for a replay: the identical file was already imported
	// and this call created nothing.
	code := http.StatusCreated
	if resp.GetAlreadyImported() {
		code = http.StatusOK
	}
	response.JSON(w, code, map[string]any{
		"statement":        statement,
		"transactions":     transactions,
		"already_imported": resp.GetAlreadyImported(),
	})
}

// HandleListBankStatements lists imported statements, newest first.
// GET /api/v1/finance/bank-statements?page=&per_page=
func (b *BizRoutes) HandleListBankStatements(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.ListBankStatements(r.Context(), &bizv1.ListBankStatementsRequest{
		TenantId: tenantID,
		Page:     int32(page),
		PerPage:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	statements, err := hrMarshalSlice(resp.GetStatements())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"statements": statements,
		"total":      resp.GetTotal(),
	})
}

// HandleGetBankStatement returns one import with its transactions.
// GET /api/v1/finance/bank-statements/{id}
func (b *BizRoutes) HandleGetBankStatement(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.GetBankStatement(r.Context(), &bizv1.GetBankStatementRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	statement, err := hrMarshalProto(resp.GetStatement())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	transactions, err := hrMarshalSlice(resp.GetTransactions())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"statement":    statement,
		"transactions": transactions,
	})
}
