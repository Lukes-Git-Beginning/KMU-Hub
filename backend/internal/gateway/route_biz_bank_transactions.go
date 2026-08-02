package gateway

// Bank transactions: the reconciliation queue of an imported statement
// (Migration 000247), in the shape the desktop client reads.

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// bankMatchStatuses are the states a caller may ask for by name. "ignored" is
// listed although the desktop client has no label for it: the value exists in
// the database since 000247, and refusing to name it would make those rows
// unreachable over HTTP.
var bankMatchStatuses = map[string]bool{
	"unmatched": true,
	"suggested": true,
	"matched":   true,
	"ignored":   true,
}

// bankTransactionWire is the exact shape the desktop client reads (the
// BankTransaction type in types/finance-types.ts, consumed through
// financeBankingApi).
//
// Hand-mapped rather than protojson-marshalled, same two reasons as the bank
// accounts next door: the repo-wide marshaller emits snake_case where this
// contract is camelCase, and it carries the decimal as a string where the
// client reads amount as a JSON number.
//
// The signed amount is passed through unchanged and Type restates its sign --
// the client branches on type and formats amount, and deriving one from the
// other on the client would put that rule in two places.
type bankTransactionWire struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Counterpart string  `json:"counterpart"`
	MatchStatus string  `json:"matchStatus"`
	// MatchedInvoice is the invoice *number*, joined on read -- that is what the
	// client prints. Omitted rather than null: the frontend types it optional.
	MatchedInvoice string `json:"matchedInvoice,omitempty"`
}

func toBankTransactionWire(p *bizv1.BankTransaction) bankTransactionWire {
	// The decimal string is authoritative; an unparsable one would be a bug in
	// the service, and 0 is the honest rendering of "this did not survive the
	// wire" rather than a made-up figure.
	amount, _ := strconv.ParseFloat(p.GetAmount(), 64)
	txType := "credit"
	if amount < 0 {
		txType = "debit"
	}
	// A bank fee often carries no remittance information at all. The entry
	// reference is what the statement then identifies the line by, and an empty
	// row in the queue would be unidentifiable.
	description := p.GetRemittanceInfo()
	if description == "" {
		description = p.GetEntryRef()
	}
	return bankTransactionWire{
		ID:             p.GetId(),
		Date:           p.GetValueDate(),
		Description:    description,
		Amount:         amount,
		Type:           txType,
		Counterpart:    p.GetCounterpartyName(),
		MatchStatus:    p.GetMatchStatus(),
		MatchedInvoice: p.GetMatchedInvoiceNumber(),
	}
}

// HandleListBankTransactions lists imported transactions across statements.
// GET /api/v1/finance/bank-transactions?status=&statement_id=&page=&per_page=
//
// Without an explicit status the list is the reconciliation queue and leaves
// out what was set aside; status=ignored asks for exactly those.
func (b *BizRoutes) HandleListBankTransactions(w http.ResponseWriter, r *http.Request) {
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

	req := &bizv1.ListBankTransactionsRequest{
		TenantId: tenantID,
		Page:     int32(page),
		PerPage:  int32(pageSize),
	}
	switch status := r.URL.Query().Get("status"); status {
	case "", "all":
		req.ExcludeMatchStatus = []string{"ignored"}
	default:
		if !bankMatchStatuses[status] {
			response.Error(w, http.StatusBadRequest, "unknown status")
			return
		}
		req.MatchStatus = status
	}
	if statementID := r.URL.Query().Get("statement_id"); statementID != "" {
		req.StatementId = &statementID
	}

	resp, err := client.ListBankTransactions(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// Built with make(..., 0, n) so a tenant without transactions serializes as
	// [] rather than null -- the frontend maps over the array without a guard.
	transactions := make([]bankTransactionWire, 0, len(resp.GetTransactions()))
	for _, t := range resp.GetTransactions() {
		transactions = append(transactions, toBankTransactionWire(t))
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"transactions": transactions,
		"total":        resp.GetTotal(),
	})
}

// matchBankTransactionRequest is the body of a confirmation. Both fields absent
// confirms the suggestion the import attached, which is the common case.
type matchBankTransactionRequest struct {
	InvoiceID     string `json:"invoice_id"     validate:"omitempty,uuid4"`
	InvoiceNumber string `json:"invoice_number" validate:"omitempty,max=64"`
}

// HandleMatchBankTransaction books a bank credit as a payment against an
// invoice.
// POST /api/v1/finance/bank-transactions/{id}/match
func (b *BizRoutes) HandleMatchBankTransaction(w http.ResponseWriter, r *http.Request) {
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
	req, ok := decodeAndValidate[matchBankTransactionRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ReconcileBankTransaction(r.Context(), &bizv1.ReconcileBankTransactionRequest{
		TenantId:      tenantID,
		Id:            chi.URLParam(r, "id"),
		InvoiceId:     req.InvoiceID,
		InvoiceNumber: req.InvoiceNumber,
		UserId:        middleware.GetUserID(r.Context()),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"transaction": toBankTransactionWire(resp.GetTransaction()),
	})
}

// HandleRejectBankTransactionMatch discards a suggestion without booking it.
// POST /api/v1/finance/bank-transactions/{id}/reject-match
//
// Distinct from ignore below: the entry stays in the queue as unmatched and
// still needs a decision.
func (b *BizRoutes) HandleRejectBankTransactionMatch(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.RejectBankTransactionMatch(r.Context(), &bizv1.RejectBankTransactionMatchRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
		UserId:   middleware.GetUserID(r.Context()),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"transaction": toBankTransactionWire(resp.GetTransaction()),
	})
}

// HandleIgnoreBankTransaction takes an entry out of the reconciliation queue.
// POST /api/v1/finance/bank-transactions/{id}/ignore
func (b *BizRoutes) HandleIgnoreBankTransaction(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.IgnoreBankTransaction(r.Context(), &bizv1.IgnoreBankTransactionRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
		UserId:   middleware.GetUserID(r.Context()),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"transaction": toBankTransactionWire(resp.GetTransaction()),
	})
}
