package server

// Expenses (Ausgaben) — Migration 000257.
//
// Thin handlers: parse, call, respond. The approval transition, the four-eyes
// rule and every validation live in expense.Service.

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/expense"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

const expenseDateLayout = "2006-01-02"

func (s *BizGRPCServer) requireExpense() error {
	if s.expenseSvc == nil {
		return status.Error(codes.Unimplemented, "expenses not configured")
	}
	return nil
}

// expenseError maps the package's sentinel errors onto gRPC codes. A refused
// self-approval or a decision on an already decided expense is the caller's
// mistake and must not surface as a 500.
func expenseError(err error) error {
	switch {
	case errors.Is(err, expense.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, expense.ErrInvalidAmount),
		errors.Is(err, expense.ErrInvalidDate),
		errors.Is(err, expense.ErrDescriptionRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, expense.ErrNotPending),
		errors.Is(err, expense.ErrDecided),
		errors.Is(err, expense.ErrSelfApproval):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, expense.ErrMissingActor):
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		return status.Error(codes.Internal, "failed to process expense")
	}
}

func (s *BizGRPCServer) CreateExpense(ctx context.Context, req *bizv1.CreateExpenseRequest) (*bizv1.CreateExpenseResponse, error) {
	if err := s.requireExpense(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	// An unparsable user_id is refused rather than silently becoming uuid.Nil:
	// a nil submitter would make every later approval pass the four-eyes check.
	actorID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	created, err := s.expenseSvc.Create(ctx, tenantID, actorID, expense.CreateInput{
		Description: req.GetDescription(),
		Amount:      req.GetAmount(),
		Date:        req.GetDate(),
		Category:    req.GetCategory(),
		Supplier:    req.GetSupplier(),
		Project:     req.GetProject(),
		Account:     req.GetAccount(),
		ReceiptName: req.GetReceiptName(),
	})
	if err != nil {
		return nil, expenseError(err)
	}
	return &bizv1.CreateExpenseResponse{Expense: expenseToProto(created)}, nil
}

func (s *BizGRPCServer) ListExpenses(ctx context.Context, req *bizv1.ListExpensesRequest) (*bizv1.ListExpensesResponse, error) {
	if err := s.requireExpense(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	in := expense.ListInput{
		Status:   req.GetStatus(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPerPage()),
	}
	if raw := req.GetSubmittedBy(); raw != "" {
		submittedBy, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid submitted_by")
		}
		in.SubmittedBy = &submittedBy
	}

	expenses, total, err := s.expenseSvc.List(ctx, tenantID, in)
	if err != nil {
		return nil, expenseError(err)
	}
	return &bizv1.ListExpensesResponse{
		Expenses: expensesToProto(expenses),
		Total:    int32(total),
	}, nil
}

func (s *BizGRPCServer) UpdateExpense(ctx context.Context, req *bizv1.UpdateExpenseRequest) (*bizv1.UpdateExpenseResponse, error) {
	if err := s.requireExpense(); err != nil {
		return nil, err
	}
	tenantID, id, err := parseExpenseIDs(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	updated, err := s.expenseSvc.Update(ctx, tenantID, id, expense.UpdateInput{
		Description: req.Description,
		Amount:      req.Amount,
		Date:        req.Date,
		Category:    req.Category,
		Supplier:    req.Supplier,
		Project:     req.Project,
		Account:     req.Account,
		ReceiptName: req.ReceiptName,
	})
	if err != nil {
		return nil, expenseError(err)
	}
	return &bizv1.UpdateExpenseResponse{Expense: expenseToProto(updated)}, nil
}

func (s *BizGRPCServer) DeleteExpense(ctx context.Context, req *bizv1.DeleteExpenseRequest) (*bizv1.DeleteExpenseResponse, error) {
	if err := s.requireExpense(); err != nil {
		return nil, err
	}
	tenantID, id, err := parseExpenseIDs(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.expenseSvc.Delete(ctx, tenantID, id); err != nil {
		return nil, expenseError(err)
	}
	return &bizv1.DeleteExpenseResponse{}, nil
}

func (s *BizGRPCServer) DecideExpense(ctx context.Context, req *bizv1.DecideExpenseRequest) (*bizv1.DecideExpenseResponse, error) {
	if err := s.requireExpense(); err != nil {
		return nil, err
	}
	tenantID, id, err := parseExpenseIDs(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}
	actorID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	var decided *models.Expense
	switch req.GetStatus() {
	case models.ExpenseStatusApproved:
		decided, err = s.expenseSvc.Approve(ctx, tenantID, id, actorID)
	case models.ExpenseStatusRejected:
		decided, err = s.expenseSvc.Reject(ctx, tenantID, id, actorID)
	default:
		return nil, status.Error(codes.InvalidArgument, `status must be "approved" or "rejected"`)
	}
	if err != nil {
		return nil, expenseError(err)
	}
	return &bizv1.DecideExpenseResponse{Expense: expenseToProto(decided)}, nil
}

func (s *BizGRPCServer) AttachExpenseReceipt(ctx context.Context, req *bizv1.AttachExpenseReceiptRequest) (*bizv1.AttachExpenseReceiptResponse, error) {
	if err := s.requireExpense(); err != nil {
		return nil, err
	}
	tenantID, id, err := parseExpenseIDs(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	updated, err := s.expenseSvc.AttachReceipt(ctx, tenantID, id, req.GetReceiptName())
	if err != nil {
		return nil, expenseError(err)
	}
	return &bizv1.AttachExpenseReceiptResponse{Expense: expenseToProto(updated)}, nil
}

func parseExpenseIDs(rawTenant, rawID string) (uuid.UUID, uuid.UUID, error) {
	tenantID, err := uuid.Parse(rawTenant)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	return tenantID, id, nil
}

func expenseToProto(e *models.Expense) *bizv1.Expense {
	if e == nil {
		return nil
	}
	p := &bizv1.Expense{
		Id:          e.ID.String(),
		TenantId:    e.TenantID.String(),
		Description: e.Description,
		Amount:      e.Amount.StringFixed(2),
		ExpenseDate: e.ExpenseDate.Format(expenseDateLayout),
		Category:    e.Category,
		Supplier:    e.Supplier,
		Status:      e.Status,
		Project:     e.Project,
		Account:     e.Account,
		ReceiptName: e.ReceiptName,
		CreatedAt:   timestamppb.New(e.CreatedAt),
		UpdatedAt:   timestamppb.New(e.UpdatedAt),
	}
	if e.SubmittedBy != nil {
		submittedBy := e.SubmittedBy.String()
		p.SubmittedBy = &submittedBy
	}
	if e.DecidedBy != nil {
		decidedBy := e.DecidedBy.String()
		p.DecidedBy = &decidedBy
	}
	if e.DecidedAt != nil {
		p.DecidedAt = timestamppb.New(*e.DecidedAt)
	}
	return p
}

func expensesToProto(expenses []*models.Expense) []*bizv1.Expense {
	out := make([]*bizv1.Expense, 0, len(expenses))
	for _, e := range expenses {
		out = append(out, expenseToProto(e))
	}
	return out
}
