package gateway

// Bank accounts (Bankkonten) — Migration 000258.

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/dachfmt"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// bankAccountWire is the exact shape the desktop client reads (the BankAccount
// type in types/finance-types.ts, consumed through financeBankingApi).
//
// Hand-mapped rather than protojson-marshalled for the same two reasons as the
// expense endpoints: the repo-wide marshaller emits snake_case where this
// contract is camelCase, and it carries the decimal as a string where the
// client reads balance as a JSON number.
//
// LastSync is a non-omitted pointer: the frontend types it as string | null and
// renders "never synced" from the null, so dropping the key would leave that
// branch reading undefined instead.
type bankAccountWire struct {
	ID        string  `json:"id"`
	BankName  string  `json:"bankName"`
	IBAN      string  `json:"iban"`
	BIC       string  `json:"bic"`
	Balance   float64 `json:"balance"`
	Currency  string  `json:"currency"`
	Connected bool    `json:"connected"`
	LastSync  *string `json:"lastSync"`
}

func toBankAccountWire(p *bizv1.BankAccount) bankAccountWire {
	// The decimal string is authoritative; an unparsable one would be a bug in
	// the service, and 0 is the honest rendering of "this did not survive the
	// wire" rather than a made-up figure.
	balance, _ := strconv.ParseFloat(p.GetBalance(), 64)
	w := bankAccountWire{
		ID: p.GetId(),
		// Stored canonically, grouped for display -- the client prints this
		// straight onto a card next to the bank name.
		IBAN:      dachfmt.FormatIBAN(p.GetIban()),
		BankName:  p.GetBankName(),
		BIC:       p.GetBic(),
		Balance:   balance,
		Currency:  p.GetCurrency(),
		Connected: p.GetConnected(),
	}
	if sync := p.GetLastSync(); sync != "" {
		w.LastSync = &sync
	}
	return w
}

// createBankAccountRequest is the body of POST /finance/bank-accounts. The IBAN
// is validated here as well as in the service: rejecting a typo at the edge
// gives the caller a field-level error instead of a bare 400 from the RPC.
type createBankAccountRequest struct {
	BankName string `json:"bankName" validate:"required,max=160"`
	IBAN     string `json:"iban"     validate:"required,iban"`
	BIC      string `json:"bic"      validate:"omitempty,bic"`
	Currency string `json:"currency" validate:"omitempty,len=3,alpha"`
}

// updateBankAccountRequest is a partial edit: every field is a pointer so an
// absent one stays untouched rather than being cleared.
type updateBankAccountRequest struct {
	BankName  *string `json:"bankName" validate:"omitempty,max=160"`
	IBAN      *string `json:"iban"     validate:"omitempty,iban"`
	BIC       *string `json:"bic"      validate:"omitempty,bic"`
	Currency  *string `json:"currency" validate:"omitempty,len=3,alpha"`
	Connected *bool   `json:"connected"`
}

// HandleListBankAccounts returns the tenant's bank accounts.
// GET /api/v1/finance/bank-accounts
func (b *BizRoutes) HandleListBankAccounts(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.ListBankAccounts(r.Context(), &bizv1.ListBankAccountsRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// Built with make(..., 0, n) so a tenant without accounts serializes as []
	// rather than null -- the frontend maps over the array without a guard.
	accounts := make([]bankAccountWire, 0, len(resp.GetAccounts()))
	for _, a := range resp.GetAccounts() {
		accounts = append(accounts, toBankAccountWire(a))
	}
	response.JSON(w, http.StatusOK, map[string]any{"accounts": accounts})
}

// HandleCreateBankAccount records a new bank account.
// POST /api/v1/finance/bank-accounts
func (b *BizRoutes) HandleCreateBankAccount(w http.ResponseWriter, r *http.Request) {
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
	req, ok := decodeAndValidate[createBankAccountRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateBankAccount(r.Context(), &bizv1.CreateBankAccountRequest{
		TenantId: tenantID,
		BankName: req.BankName,
		Iban:     req.IBAN,
		Bic:      req.BIC,
		Currency: req.Currency,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"account": toBankAccountWire(resp.GetAccount())})
}

// HandleUpdateBankAccount edits an account's master data.
// PATCH /api/v1/finance/bank-accounts/{id}
func (b *BizRoutes) HandleUpdateBankAccount(w http.ResponseWriter, r *http.Request) {
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
	req, ok := decodeAndValidate[updateBankAccountRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateBankAccount(r.Context(), &bizv1.UpdateBankAccountRequest{
		TenantId:  tenantID,
		Id:        chi.URLParam(r, "id"),
		BankName:  req.BankName,
		Iban:      req.IBAN,
		Bic:       req.BIC,
		Currency:  req.Currency,
		Connected: req.Connected,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"account": toBankAccountWire(resp.GetAccount())})
}

// HandleConnectBankAccount marks an account as linked. The PSD2 handshake is
// simulated in the desktop client; this only records the outcome.
// POST /api/v1/finance/bank-accounts/{id}/connect
func (b *BizRoutes) HandleConnectBankAccount(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.ConnectBankAccount(r.Context(), &bizv1.ConnectBankAccountRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"account": toBankAccountWire(resp.GetAccount())})
}

// HandleDeleteBankAccount removes an account. Statements already imported for
// its IBAN stay: they record what a bank reported.
// DELETE /api/v1/finance/bank-accounts/{id}
func (b *BizRoutes) HandleDeleteBankAccount(w http.ResponseWriter, r *http.Request) {
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

	if _, err := client.DeleteBankAccount(r.Context(), &bizv1.DeleteBankAccountRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
	}); err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, json.RawMessage(`{}`))
}
