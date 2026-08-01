package gateway

// Finance transactions (Transaktionen): consolidated payment ledger — a
// tenant-wide overview, not filtered by any single document. See
// financeTransactionApi in finance-client.ts.

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// financeTransactionWire is the exact shape the desktop client reads (the
// Transaction type in stores/finance.ts, consumed through financeTransactionApi
// in finance-client.ts). Hand-mapped for the same reason as expenses next
// door: amount is read as a JSON number where the proto carries it as a
// decimal string.
type financeTransactionWire struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"`
	Category    string  `json:"category"`
	Status      string  `json:"status"`
	Reference   *string `json:"reference,omitempty"`
	InvoiceID   *string `json:"invoiceId,omitempty"`
}

func toFinanceTransactionWire(t *bizv1.FinanceTransaction) financeTransactionWire {
	amount, _ := strconv.ParseFloat(t.GetAmount(), 64)
	return financeTransactionWire{
		ID:          t.GetId(),
		Type:        t.GetType(),
		Description: t.GetDescription(),
		Amount:      amount,
		Date:        t.GetDate(),
		Category:    t.GetCategory(),
		Status:      t.GetStatus(),
		Reference:   t.Reference,
		InvoiceID:   t.InvoiceId,
	}
}

// HandleListTransactions serves the tenant's consolidated payment ledger.
// GET /api/v1/finance/transactions
func (b *BizRoutes) HandleListTransactions(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.ListFinanceTransactions(r.Context(), &bizv1.ListFinanceTransactionsRequest{TenantId: tenantID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	transactions := make([]financeTransactionWire, 0, len(resp.GetTransactions()))
	for _, t := range resp.GetTransactions() {
		transactions = append(transactions, toFinanceTransactionWire(t))
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"transactions": transactions,
		"total":        len(transactions),
	})
}

// HandleDeleteTransaction removes a ledger entry. A payment delete reverts
// the invoice's paid status if needed; an approved expense is refused with
// 409 -- it is the record of a decision, not a draft.
// DELETE /api/v1/finance/transactions/{id}
func (b *BizRoutes) HandleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
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

	if _, err := client.DeleteFinanceTransaction(r.Context(), &bizv1.DeleteFinanceTransactionRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
	}); err != nil {
		respondGRPCError(w, err)
		return
	}
	// The frontend types the response as Record<string, never> and reads nothing
	// from it, so an empty object is the whole contract.
	response.JSON(w, http.StatusOK, json.RawMessage(`{}`))
}
