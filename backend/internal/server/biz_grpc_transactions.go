package server

// Finance transactions (Transaktionen) — consolidated payment ledger.
//
// Thin handlers: the union query lives in invoiceService.ListTransactions
// (internal/biz/invoice/postgres_transactions.go). Deleting an entry means
// something different depending on which table it names, so this file routes
// by the "pay-"/"exp-" id prefix that ListTransactions assigns: a payment
// delete goes through paymentService.Delete (which also reverts the
// invoice's paid status if needed), an expense delete goes through
// expenseSvc.Delete (which refuses an already-decided expense — every
// expense-derived transaction here is approved, so that refusal is the
// expected, honest outcome, not a bug).

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

const (
	financeTransactionDateLayout = "2006-01-02"
	financeTransactionPayPrefix  = "pay-"
	financeTransactionExpPrefix  = "exp-"
)

func (s *BizGRPCServer) ListFinanceTransactions(ctx context.Context, req *bizv1.ListFinanceTransactionsRequest) (*bizv1.ListFinanceTransactionsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	transactions, err := s.invoiceService.ListTransactions(ctx, tenantID)
	if err != nil {
		return nil, mapBizError(err)
	}

	out := make([]*bizv1.FinanceTransaction, 0, len(transactions))
	for _, t := range transactions {
		out = append(out, toProtoFinanceTransaction(t))
	}
	return &bizv1.ListFinanceTransactionsResponse{Transactions: out}, nil
}

func (s *BizGRPCServer) DeleteFinanceTransaction(ctx context.Context, req *bizv1.DeleteFinanceTransactionRequest) (*bizv1.DeleteFinanceTransactionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	switch {
	case strings.HasPrefix(req.GetId(), financeTransactionPayPrefix):
		id, parseErr := uuid.Parse(strings.TrimPrefix(req.GetId(), financeTransactionPayPrefix))
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid id")
		}
		if delErr := s.paymentService.Delete(ctx, tenantID, id); delErr != nil {
			return nil, mapBizError(delErr)
		}
	case strings.HasPrefix(req.GetId(), financeTransactionExpPrefix):
		if reqErr := s.requireExpense(); reqErr != nil {
			return nil, reqErr
		}
		id, parseErr := uuid.Parse(strings.TrimPrefix(req.GetId(), financeTransactionExpPrefix))
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid id")
		}
		if delErr := s.expenseSvc.Delete(ctx, tenantID, id); delErr != nil {
			return nil, expenseError(delErr)
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	return &bizv1.DeleteFinanceTransactionResponse{}, nil
}

func toProtoFinanceTransaction(t *models.FinanceTransaction) *bizv1.FinanceTransaction {
	p := &bizv1.FinanceTransaction{
		Id:          t.ID,
		Type:        t.Type,
		Description: t.Description,
		Amount:      t.Amount.StringFixed(2),
		Date:        t.Date.Format(financeTransactionDateLayout),
		Category:    t.Category,
		Status:      t.Status,
	}
	if t.Reference != "" {
		reference := t.Reference
		p.Reference = &reference
	}
	if t.InvoiceID != nil {
		invoiceID := t.InvoiceID.String()
		p.InvoiceId = &invoiceID
	}
	return p
}
