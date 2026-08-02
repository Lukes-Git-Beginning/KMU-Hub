package gateway

// Expenses (Ausgaben) — Migration 000257.

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// expenseWire is the exact shape the desktop client reads (the Expense type in
// stores/finance.ts, consumed through financeExpenseApi in finance-client.ts).
//
// It is hand-mapped instead of protojson-marshalled because the frontend
// contract differs from the repo-wide protojson options in two places, and the
// frontend is not being changed for this endpoint: receiptName is camelCase
// where UseProtoNames would emit receipt_name, and amount is read as a JSON
// number where the proto carries the decimal as a string.
//
// receipt is derived from receiptName rather than stored, so the boolean the
// list renders and the filename the detail shows can never disagree.
type expenseWire struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"`
	Category    string  `json:"category"`
	Supplier    string  `json:"supplier"`
	Project     *string `json:"project,omitempty"`
	Account     *string `json:"account,omitempty"`
	Receipt     bool    `json:"receipt"`
	ReceiptName *string `json:"receiptName,omitempty"`
	Status      string  `json:"status"`
}

func toExpenseWire(p *bizv1.Expense) expenseWire {
	// The decimal string is authoritative; an unparsable one would be a bug in
	// the service, and 0 is the honest rendering of "this did not survive the
	// wire" rather than a made-up figure.
	amount, _ := strconv.ParseFloat(p.GetAmount(), 64)
	w := expenseWire{
		ID:          p.GetId(),
		Description: p.GetDescription(),
		Amount:      amount,
		Date:        p.GetExpenseDate(),
		Category:    p.GetCategory(),
		Supplier:    p.GetSupplier(),
		Project:     p.Project,
		Account:     p.Account,
		Status:      p.GetStatus(),
	}
	if name := p.GetReceiptName(); name != "" {
		w.Receipt = true
		w.ReceiptName = &name
	}
	return w
}

// createExpenseRequest mirrors CreateExpenseRequest in finance-client.ts.
// Amount arrives as a JSON number and is carried on as a decimal string, so no
// float ever reaches the database.
type createExpenseRequest struct {
	Description string  `json:"description" validate:"required,max=500"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Date        string  `json:"date" validate:"required,datetime=2006-01-02"`
	Category    string  `json:"category" validate:"max=120"`
	Supplier    string  `json:"supplier" validate:"max=255"`
	Project     string  `json:"project" validate:"max=255"`
	Account     string  `json:"account" validate:"max=20"`
	ReceiptName string  `json:"receiptName" validate:"max=255"`
}

// updateExpenseRequest mirrors UpdateExpenseRequest, which the frontend types as
// Partial<CreateExpenseRequest>. Every field is a pointer so an absent one stays
// untouched instead of being cleared -- the Kontierung dialog sends account
// alone, and a non-pointer struct would blank the description with it.
type updateExpenseRequest struct {
	Description *string  `json:"description" validate:"omitempty,max=500"`
	Amount      *float64 `json:"amount" validate:"omitempty,gt=0"`
	Date        *string  `json:"date" validate:"omitempty,datetime=2006-01-02"`
	Category    *string  `json:"category" validate:"omitempty,max=120"`
	Supplier    *string  `json:"supplier" validate:"omitempty,max=255"`
	Project     *string  `json:"project" validate:"omitempty,max=255"`
	Account     *string  `json:"account" validate:"omitempty,max=20"`
	ReceiptName *string  `json:"receiptName" validate:"omitempty,max=255"`
}

type attachReceiptRequest struct {
	ReceiptName string `json:"receiptName" validate:"required,max=255"`
}

// HandleListExpenses returns the tenant's expenses.
// GET /api/v1/finance/expenses?status=&page=&per_page=
func (b *BizRoutes) HandleListExpenses(w http.ResponseWriter, r *http.Request) {
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

	req := &bizv1.ListExpensesRequest{
		TenantId: tenantID,
		Status:   r.URL.Query().Get("status"),
		Page:     int32(page),
		PerPage:  int32(pageSize),
	}
	// A grant narrowed to "own" shrinks the list to what the caller submitted.
	// Assigned unconditionally: at any wider scope this is nil and the whole
	// tenant is listed.
	//
	// finance:expense:read is not in capability-catalog.ts yet, so today this
	// always resolves to "all". It is wired here rather than left out so the
	// narrowing takes effect the moment the key is catalogued and seeded --
	// a scope that silently fails to apply is the direction that leaks rows.
	submittedBy, ok := ownerFilterForScope(w, r, "finance:expense", "read")
	if !ok {
		return
	}
	req.SubmittedBy = submittedBy

	resp, err := client.ListExpenses(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// Built with make(..., 0, n) so an empty tenant serializes as [] rather than
	// null -- the frontend maps over the array without a guard.
	expenses := make([]expenseWire, 0, len(resp.GetExpenses()))
	for _, e := range resp.GetExpenses() {
		expenses = append(expenses, toExpenseWire(e))
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"expenses": expenses,
		"total":    resp.GetTotal(),
	})
}

// HandleCreateExpense records a new expense, always as pending.
// POST /api/v1/finance/expenses
func (b *BizRoutes) HandleCreateExpense(w http.ResponseWriter, r *http.Request) {
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
	req, ok := decodeAndValidate[createExpenseRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateExpense(r.Context(), &bizv1.CreateExpenseRequest{
		TenantId:    tenantID,
		UserId:      middleware.GetUserID(r.Context()),
		Description: req.Description,
		Amount:      strconv.FormatFloat(req.Amount, 'f', 2, 64),
		Date:        req.Date,
		Category:    req.Category,
		Supplier:    req.Supplier,
		Project:     req.Project,
		Account:     req.Account,
		ReceiptName: req.ReceiptName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"expense": toExpenseWire(resp.GetExpense())})
}

// HandleUpdateExpense edits an expense. Amount and date close once a decision
// was made; the SKR03/04 account and the receipt filename stay open.
// PUT /api/v1/finance/expenses/{id}
func (b *BizRoutes) HandleUpdateExpense(w http.ResponseWriter, r *http.Request) {
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
	req, ok := decodeAndValidate[updateExpenseRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &bizv1.UpdateExpenseRequest{
		TenantId:    tenantID,
		Id:          chi.URLParam(r, "id"),
		Description: req.Description,
		Date:        req.Date,
		Category:    req.Category,
		Supplier:    req.Supplier,
		Project:     req.Project,
		Account:     req.Account,
		ReceiptName: req.ReceiptName,
	}
	if req.Amount != nil {
		amount := strconv.FormatFloat(*req.Amount, 'f', 2, 64)
		grpcReq.Amount = &amount
	}

	resp, err := client.UpdateExpense(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"expense": toExpenseWire(resp.GetExpense())})
}

// HandleApproveExpense accepts a pending expense.
// POST /api/v1/finance/expenses/{id}/approve
func (b *BizRoutes) HandleApproveExpense(w http.ResponseWriter, r *http.Request) {
	b.decideExpense(w, r, "approved")
}

// HandleRejectExpense refuses a pending expense.
// POST /api/v1/finance/expenses/{id}/reject
func (b *BizRoutes) HandleRejectExpense(w http.ResponseWriter, r *http.Request) {
	b.decideExpense(w, r, "rejected")
}

// decideExpense is the shared body of approve and reject. The decision itself,
// including the refusal to decide one's own expense, is made in the service.
func (b *BizRoutes) decideExpense(w http.ResponseWriter, r *http.Request, status string) {
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

	resp, err := client.DecideExpense(r.Context(), &bizv1.DecideExpenseRequest{
		TenantId: tenantID,
		Id:       chi.URLParam(r, "id"),
		UserId:   middleware.GetUserID(r.Context()),
		Status:   status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"expense": toExpenseWire(resp.GetExpense())})
}

// HandleAttachExpenseReceipt records the filename of a handed-in document.
// POST /api/v1/finance/expenses/{id}/receipt
func (b *BizRoutes) HandleAttachExpenseReceipt(w http.ResponseWriter, r *http.Request) {
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
	req, ok := decodeAndValidate[attachReceiptRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.AttachExpenseReceipt(r.Context(), &bizv1.AttachExpenseReceiptRequest{
		TenantId:    tenantID,
		Id:          chi.URLParam(r, "id"),
		ReceiptName: req.ReceiptName,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"expense": toExpenseWire(resp.GetExpense())})
}

// HandleDeleteExpense removes a pending expense. A decided one is refused with
// 409 -- it is the record of a decision, not a draft.
// DELETE /api/v1/finance/expenses/{id}
func (b *BizRoutes) HandleDeleteExpense(w http.ResponseWriter, r *http.Request) {
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

	if _, err := client.DeleteExpense(r.Context(), &bizv1.DeleteExpenseRequest{
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
