package server

// Bank accounts (Bankkonten) — Migration 000258.
//
// Thin handlers: parse, call, respond. IBAN normalisation, the mod-97 check and
// the connect transition live in banking.Service.
//
// No handler here logs an IBAN, a BIC or a bank name: account numbers are
// personal financial data and the id identifies the row well enough.

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/banking"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// bankAccountError maps the package's sentinel errors onto gRPC codes. A typo'd
// IBAN and a duplicate account are the caller's mistake, not a server fault.
func bankAccountError(err error) error {
	switch {
	case errors.Is(err, banking.ErrAccountNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, banking.ErrInvalidIBAN),
		errors.Is(err, banking.ErrInvalidAccount):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, banking.ErrAccountExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "failed to process bank account")
	}
}

func (s *BizGRPCServer) ListBankAccounts(ctx context.Context, req *bizv1.ListBankAccountsRequest) (*bizv1.ListBankAccountsResponse, error) {
	if err := s.requireBanking(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	accounts, err := s.bankingSvc.ListAccounts(ctx, tenantID)
	if err != nil {
		return nil, bankAccountError(err)
	}
	out := make([]*bizv1.BankAccount, 0, len(accounts))
	for _, acc := range accounts {
		out = append(out, bankAccountToProto(acc))
	}
	return &bizv1.ListBankAccountsResponse{Accounts: out}, nil
}

func (s *BizGRPCServer) CreateBankAccount(ctx context.Context, req *bizv1.CreateBankAccountRequest) (*bizv1.CreateBankAccountResponse, error) {
	if err := s.requireBanking(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	created, err := s.bankingSvc.CreateAccount(ctx, banking.CreateAccountInput{
		TenantID: tenantID,
		BankName: req.GetBankName(),
		IBAN:     req.GetIban(),
		BIC:      req.GetBic(),
		Currency: req.GetCurrency(),
	})
	if err != nil {
		return nil, bankAccountError(err)
	}
	return &bizv1.CreateBankAccountResponse{Account: bankAccountToProto(created)}, nil
}

func (s *BizGRPCServer) UpdateBankAccount(ctx context.Context, req *bizv1.UpdateBankAccountRequest) (*bizv1.UpdateBankAccountResponse, error) {
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

	updated, err := s.bankingSvc.UpdateAccount(ctx, banking.UpdateAccountInput{
		TenantID:  tenantID,
		ID:        id,
		BankName:  req.BankName,
		IBAN:      req.Iban,
		BIC:       req.Bic,
		Currency:  req.Currency,
		Connected: req.Connected,
	})
	if err != nil {
		return nil, bankAccountError(err)
	}
	return &bizv1.UpdateBankAccountResponse{Account: bankAccountToProto(updated)}, nil
}

func (s *BizGRPCServer) DeleteBankAccount(ctx context.Context, req *bizv1.DeleteBankAccountRequest) (*bizv1.DeleteBankAccountResponse, error) {
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

	if err := s.bankingSvc.DeleteAccount(ctx, tenantID, id); err != nil {
		return nil, bankAccountError(err)
	}
	return &bizv1.DeleteBankAccountResponse{}, nil
}

func (s *BizGRPCServer) ConnectBankAccount(ctx context.Context, req *bizv1.ConnectBankAccountRequest) (*bizv1.ConnectBankAccountResponse, error) {
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

	connected, err := s.bankingSvc.ConnectAccount(ctx, tenantID, id)
	if err != nil {
		return nil, bankAccountError(err)
	}
	return &bizv1.ConnectBankAccountResponse{Account: bankAccountToProto(connected)}, nil
}

// bankAccountToProto maps the model onto the wire message. The balance travels
// as a fixed two-decimal string so no float rounding happens between the
// database and the client.
func bankAccountToProto(acc *models.BankAccount) *bizv1.BankAccount {
	if acc == nil {
		return nil
	}
	out := &bizv1.BankAccount{
		Id:        acc.ID.String(),
		TenantId:  acc.TenantID.String(),
		BankName:  acc.BankName,
		Iban:      acc.IBAN,
		Bic:       acc.BIC,
		Currency:  acc.Currency,
		Connected: acc.Connected,
		CreatedAt: timestamppb.New(acc.CreatedAt),
		UpdatedAt: timestamppb.New(acc.UpdatedAt),
		Balance:   acc.Balance.StringFixed(2),
	}
	if acc.ConnectedAt != nil {
		out.ConnectedAt = timestamppb.New(*acc.ConnectedAt)
	}
	if acc.LastSync != nil {
		lastSync := acc.LastSync.Format(bankDateLayout)
		out.LastSync = &lastSync
	}
	return out
}
