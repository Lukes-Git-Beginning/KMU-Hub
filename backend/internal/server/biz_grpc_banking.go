package server

// Bank statement import and reconciliation (Zahlungsabgleich) — Migration 000247.
//
// Thin handlers: parse, call, respond. Every parsing rule, the idempotency of an
// import and the matching heuristics live in banking.Service.

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/biz/banking"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

const bankDateLayout = "2006-01-02"

func (s *BizGRPCServer) requireBanking() error {
	if s.bankingSvc == nil {
		return status.Error(codes.Unimplemented, "bank statement import not configured")
	}
	return nil
}

// bankingError maps the package's sentinel errors onto gRPC codes. A malformed
// upload is the client's fault and must surface as a 4xx, never as a 500 that
// looks like this system broke.
func bankingError(err error) error {
	switch {
	case errors.Is(err, banking.ErrEmptyFile),
		errors.Is(err, banking.ErrUnknownFormat),
		errors.Is(err, banking.ErrMalformed),
		errors.Is(err, banking.ErrFileTooLarge),
		errors.Is(err, banking.ErrTooManyEntries):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, banking.ErrStatementNotFound),
		errors.Is(err, banking.ErrTransactionNotFound),
		errors.Is(err, banking.ErrInvoiceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, banking.ErrAlreadyReconciled),
		errors.Is(err, banking.ErrNothingToReject),
		errors.Is(err, banking.ErrNotACredit):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "failed to process bank statement")
	}
}

func (s *BizGRPCServer) ImportBankStatement(ctx context.Context, req *bizv1.ImportBankStatementRequest) (*bizv1.ImportBankStatementResponse, error) {
	if err := s.requireBanking(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	userID, err := parseOptionalUUID(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	res, err := s.bankingSvc.Import(ctx, banking.ImportInput{
		TenantID: tenantID,
		Filename: req.GetFilename(),
		Content:  req.GetContent(),
		UserID:   userID,
	})
	if err != nil {
		return nil, bankingError(err)
	}

	return &bizv1.ImportBankStatementResponse{
		Statement:       bankStatementToProto(res.Statement),
		Transactions:    bankTransactionsToProto(res.Transactions),
		AlreadyImported: res.AlreadyImported,
	}, nil
}

func (s *BizGRPCServer) GetBankStatement(ctx context.Context, req *bizv1.GetBankStatementRequest) (*bizv1.GetBankStatementResponse, error) {
	if err := s.requireBanking(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	stmt, txs, err := s.bankingSvc.GetStatement(ctx, tenantID, id)
	if err != nil {
		return nil, bankingError(err)
	}
	return &bizv1.GetBankStatementResponse{
		Statement:    bankStatementToProto(stmt),
		Transactions: bankTransactionsToProto(txs),
	}, nil
}

func (s *BizGRPCServer) ListBankStatements(ctx context.Context, req *bizv1.ListBankStatementsRequest) (*bizv1.ListBankStatementsResponse, error) {
	if err := s.requireBanking(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	perPage := int(req.GetPerPage())
	offset := (max(int(req.GetPage()), 1) - 1) * max(perPage, 1)
	stmts, total, err := s.bankingSvc.ListStatements(ctx, tenantID, perPage, offset)
	if err != nil {
		return nil, bankingError(err)
	}

	out := make([]*bizv1.BankStatement, 0, len(stmts))
	for _, stmt := range stmts {
		out = append(out, bankStatementToProto(stmt))
	}
	return &bizv1.ListBankStatementsResponse{Statements: out, Total: int32(total)}, nil
}

func (s *BizGRPCServer) ListBankTransactions(ctx context.Context, req *bizv1.ListBankTransactionsRequest) (*bizv1.ListBankTransactionsResponse, error) {
	if err := s.requireBanking(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	filter := models.BankTransactionFilter{
		MatchStatus:        req.GetMatchStatus(),
		ExcludeMatchStatus: req.GetExcludeMatchStatus(),
	}
	if raw := req.GetStatementId(); raw != "" {
		statementID, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid statement_id")
		}
		filter.StatementID = &statementID
	}
	filter.Limit = int(req.GetPerPage())
	filter.Offset = (max(int(req.GetPage()), 1) - 1) * max(filter.Limit, 1)

	txs, total, err := s.bankingSvc.ListTransactions(ctx, tenantID, filter)
	if err != nil {
		return nil, bankingError(err)
	}
	return &bizv1.ListBankTransactionsResponse{
		Transactions: bankTransactionsToProto(txs),
		Total:        int32(total),
	}, nil
}

func (s *BizGRPCServer) ReconcileBankTransaction(ctx context.Context, req *bizv1.ReconcileBankTransactionRequest) (*bizv1.ReconcileBankTransactionResponse, error) {
	if err := s.requireBanking(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	// Empty means "take the suggested invoice" — the common case.
	invoiceID, err := parseOptionalUUID(req.GetInvoiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
	}
	userID, err := parseOptionalUUID(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	tx, err := s.bankingSvc.Reconcile(ctx, banking.ReconcileInput{
		TenantID:      tenantID,
		TransactionID: id,
		InvoiceID:     invoiceID,
		InvoiceNumber: req.GetInvoiceNumber(),
		UserID:        userID,
	})
	if err != nil {
		return nil, bankingError(err)
	}
	return &bizv1.ReconcileBankTransactionResponse{Transaction: bankTransactionToProto(tx)}, nil
}

// RejectBankTransactionMatch discards a suggestion. Distinct from
// IgnoreBankTransaction: the entry stays in the queue as unmatched.
func (s *BizGRPCServer) RejectBankTransactionMatch(ctx context.Context, req *bizv1.RejectBankTransactionMatchRequest) (*bizv1.RejectBankTransactionMatchResponse, error) {
	if err := s.requireBanking(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	userID, err := parseOptionalUUID(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	tx, err := s.bankingSvc.RejectMatch(ctx, tenantID, id, userID)
	if err != nil {
		return nil, bankingError(err)
	}
	return &bizv1.RejectBankTransactionMatchResponse{Transaction: bankTransactionToProto(tx)}, nil
}

func (s *BizGRPCServer) IgnoreBankTransaction(ctx context.Context, req *bizv1.IgnoreBankTransactionRequest) (*bizv1.IgnoreBankTransactionResponse, error) {
	if err := s.requireBanking(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	userID, err := parseOptionalUUID(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	tx, err := s.bankingSvc.Ignore(ctx, tenantID, id, userID)
	if err != nil {
		return nil, bankingError(err)
	}
	return &bizv1.IgnoreBankTransactionResponse{Transaction: bankTransactionToProto(tx)}, nil
}

// parseOptionalUUID reads an id that may legitimately be absent: an import run
// by an automation has no user, and a reconcile without an explicit invoice
// takes the suggestion.
func parseOptionalUUID(raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(raw)
}

func bankStatementToProto(s *models.BankStatement) *bizv1.BankStatement {
	if s == nil {
		return nil
	}
	out := &bizv1.BankStatement{
		Id:               s.ID.String(),
		Format:           s.Format,
		Filename:         s.Filename,
		ContentHash:      s.ContentHash,
		AccountIban:      s.AccountIBAN,
		StatementRef:     s.StatementRef,
		Currency:         s.Currency,
		TransactionCount: int32(s.TransactionCount),
		CreatedAt:        s.CreatedAt.Format(time.RFC3339),
	}
	if s.OpeningBalance != nil {
		v := s.OpeningBalance.String()
		out.OpeningBalance = &v
	}
	if s.ClosingBalance != nil {
		v := s.ClosingBalance.String()
		out.ClosingBalance = &v
	}
	if s.StatementDate != nil {
		v := s.StatementDate.Format(bankDateLayout)
		out.StatementDate = &v
	}
	if s.ImportedBy != nil {
		v := s.ImportedBy.String()
		out.ImportedBy = &v
	}
	return out
}

func bankTransactionToProto(t *models.BankTransaction) *bizv1.BankTransaction {
	if t == nil {
		return nil
	}
	out := &bizv1.BankTransaction{
		Id:               t.ID.String(),
		StatementId:      t.StatementID.String(),
		EntryRef:         t.EntryRef,
		EndToEndId:       t.EndToEndID,
		ValueDate:        t.ValueDate.Format(bankDateLayout),
		Amount:           t.Amount.String(),
		Currency:         t.Currency,
		CounterpartyName: t.CounterpartyName,
		CounterpartyIban: t.CounterpartyIBAN,
		RemittanceInfo:   t.RemittanceInfo,
		MatchStatus:      t.MatchStatus,
		MatchReason:      t.MatchReason,
		CreatedAt:        t.CreatedAt.Format(time.RFC3339),
	}
	if t.BookingDate != nil {
		v := t.BookingDate.Format(bankDateLayout)
		out.BookingDate = &v
	}
	if t.MatchedInvoiceID != nil {
		v := t.MatchedInvoiceID.String()
		out.MatchedInvoiceId = &v
	}
	if t.MatchedInvoiceNumber != "" {
		v := t.MatchedInvoiceNumber
		out.MatchedInvoiceNumber = &v
	}
	if t.PaymentID != nil {
		v := t.PaymentID.String()
		out.PaymentId = &v
	}
	if t.ReconciledAt != nil {
		v := t.ReconciledAt.Format(time.RFC3339)
		out.ReconciledAt = &v
	}
	if t.ReconciledBy != nil {
		v := t.ReconciledBy.String()
		out.ReconciledBy = &v
	}
	return out
}

// bankTransactionsToProto maps a list. The result is never nil so an empty
// statement serialises as [] rather than null.
func bankTransactionsToProto(txs []*models.BankTransaction) []*bizv1.BankTransaction {
	out := make([]*bizv1.BankTransaction, 0, len(txs))
	for _, t := range txs {
		out = append(out, bankTransactionToProto(t))
	}
	return out
}
