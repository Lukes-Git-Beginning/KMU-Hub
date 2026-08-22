package server

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/biz/banking"
	"github.com/kmuhub/kmuhub/internal/biz/bexio"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ---------------------------------------------------------------------------
// bankingError — every sentinel individually against its gRPC code
// ---------------------------------------------------------------------------

func TestBankingError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"empty_file", banking.ErrEmptyFile, codes.InvalidArgument},
		{"unknown_format", banking.ErrUnknownFormat, codes.InvalidArgument},
		{"malformed", banking.ErrMalformed, codes.InvalidArgument},
		{"file_too_large", banking.ErrFileTooLarge, codes.InvalidArgument},
		{"too_many_entries", banking.ErrTooManyEntries, codes.InvalidArgument},
		{"statement_not_found", banking.ErrStatementNotFound, codes.NotFound},
		{"transaction_not_found", banking.ErrTransactionNotFound, codes.NotFound},
		{"invoice_not_found", banking.ErrInvoiceNotFound, codes.NotFound},
		{"already_reconciled", banking.ErrAlreadyReconciled, codes.FailedPrecondition},
		{"nothing_to_reject", banking.ErrNothingToReject, codes.FailedPrecondition},
		{"not_a_credit", banking.ErrNotACredit, codes.FailedPrecondition},
		{"unmapped_default", banking.ErrAccountNotFound, codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, bankingError(tc.err), tc.code)
		})
	}
}

// ---------------------------------------------------------------------------
// mapBexioError — every sentinel individually against its gRPC code
// ---------------------------------------------------------------------------

func TestMapBexioError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"unauthorized", bexio.ErrBexioUnauthorized, codes.Unauthenticated},
		{"rate_limited", bexio.ErrBexioRateLimited, codes.ResourceExhausted},
		{"not_found", bexio.ErrBexioNotFound, codes.NotFound},
		{"config_not_found", bexio.ErrConfigNotFound, codes.NotFound},
		{"mapping_not_found", bexio.ErrMappingNotFound, codes.NotFound},
		{"sync_already_running", bexio.ErrSyncAlreadyRunning, codes.AlreadyExists},
		{"sync_conflict", bexio.ErrSyncConflict, codes.Aborted},
		{"invalid_field_mapping", bexio.ErrInvalidFieldMapping, codes.InvalidArgument},
		{"server_error", bexio.ErrBexioServerError, codes.Unavailable},
		{"unmapped_default", errors.New("bexio: unmapped test sentinel"), codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapBexioError(tc.err), tc.code)
		})
	}

	t.Run("nil_is_nil", func(t *testing.T) {
		require.NoError(t, mapBexioError(nil))
	})
}

// TestMapBexioError_UnmappedErrorMasksMessage extends the table above (which
// only asserts on codes.Code) with the actual leak fix from
// fix-gateway-bexio-error-message-leakage: HandleBexioOAuthCallback,
// PushInvoiceToBexio and PushQuoteToBexio used to return
// Success:false + ErrorMessage: err.Error() directly to the client, bypassing
// this function entirely. Now all three funnel through mapBexioError like
// every other Bexio RPC, so an unmapped error (a wrapped DB/Vault/network
// error, never a bexio.Err* sentinel) must come out with the fixed "internal
// error" message — never the original text, which the gateway would go on to
// embed unescaped in a redirect Location (route_bexio.go) or a JSON body.
func TestMapBexioError_UnmappedErrorMasksMessage(t *testing.T) {
	raw := errors.New("bexio push invoice: get field mappings: dial tcp 10.0.0.5:5432: connect: \nSet-Cookie: evil=1&x=y")

	err := mapBexioError(raw)
	requireGRPCCode(t, err, codes.Internal)

	st, _ := status.FromError(err)
	require.Equal(t, "internal error", st.Message())
}

// ---------------------------------------------------------------------------
// Banking RPCs — service-not-configured guard + validation paths.
//
// bankingSvc is *banking.Service, a concrete type wired via a Repository
// interface. Every case below is rejected before the handler ever reaches the
// repository, so banking.NewService(nil, nil, nil, nil) is a safe stand-in —
// no fake repository needed for this lean, validation-focused pass. Deep
// happy-path coverage per method is left for a dedicated unit (see BACKLOG).
// ---------------------------------------------------------------------------

func newTestBankingServer() *BizGRPCServer {
	return &BizGRPCServer{bankingSvc: banking.NewService(nil, nil, nil, nil)}
}

func TestImportBankStatement_Validation(t *testing.T) {
	ctx := context.Background()
	validTenant := uuid.New().String()

	t.Run("service_not_configured", func(t *testing.T) {
		s := &BizGRPCServer{}
		_, err := s.ImportBankStatement(ctx, &bizv1.ImportBankStatementRequest{TenantId: validTenant})
		requireGRPCCode(t, err, codes.Unimplemented)
	})
	t.Run("invalid_tenant_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.ImportBankStatement(ctx, &bizv1.ImportBankStatementRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_user_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.ImportBankStatement(ctx, &bizv1.ImportBankStatementRequest{
			TenantId: validTenant,
			UserId:   "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestGetBankStatement_Validation(t *testing.T) {
	ctx := context.Background()
	validTenant := uuid.New().String()

	t.Run("service_not_configured", func(t *testing.T) {
		s := &BizGRPCServer{}
		_, err := s.GetBankStatement(ctx, &bizv1.GetBankStatementRequest{TenantId: validTenant, Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unimplemented)
	})
	t.Run("invalid_tenant_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.GetBankStatement(ctx, &bizv1.GetBankStatementRequest{TenantId: "not-a-uuid", Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.GetBankStatement(ctx, &bizv1.GetBankStatementRequest{TenantId: validTenant, Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestListBankStatements_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("service_not_configured", func(t *testing.T) {
		s := &BizGRPCServer{}
		_, err := s.ListBankStatements(ctx, &bizv1.ListBankStatementsRequest{TenantId: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unimplemented)
	})
	t.Run("invalid_tenant_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.ListBankStatements(ctx, &bizv1.ListBankStatementsRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestListBankTransactions_Validation(t *testing.T) {
	ctx := context.Background()
	validTenant := uuid.New().String()

	t.Run("service_not_configured", func(t *testing.T) {
		s := &BizGRPCServer{}
		_, err := s.ListBankTransactions(ctx, &bizv1.ListBankTransactionsRequest{TenantId: validTenant})
		requireGRPCCode(t, err, codes.Unimplemented)
	})
	t.Run("invalid_tenant_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.ListBankTransactions(ctx, &bizv1.ListBankTransactionsRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_statement_id", func(t *testing.T) {
		s := newTestBankingServer()
		bad := "not-a-uuid"
		_, err := s.ListBankTransactions(ctx, &bizv1.ListBankTransactionsRequest{TenantId: validTenant, StatementId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestReconcileBankTransaction_Validation(t *testing.T) {
	ctx := context.Background()
	validTenant := uuid.New().String()
	validID := uuid.New().String()

	t.Run("service_not_configured", func(t *testing.T) {
		s := &BizGRPCServer{}
		_, err := s.ReconcileBankTransaction(ctx, &bizv1.ReconcileBankTransactionRequest{TenantId: validTenant, Id: validID})
		requireGRPCCode(t, err, codes.Unimplemented)
	})
	t.Run("invalid_tenant_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.ReconcileBankTransaction(ctx, &bizv1.ReconcileBankTransactionRequest{TenantId: "not-a-uuid", Id: validID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.ReconcileBankTransaction(ctx, &bizv1.ReconcileBankTransactionRequest{TenantId: validTenant, Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_invoice_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.ReconcileBankTransaction(ctx, &bizv1.ReconcileBankTransactionRequest{
			TenantId:  validTenant,
			Id:        validID,
			InvoiceId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_user_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.ReconcileBankTransaction(ctx, &bizv1.ReconcileBankTransactionRequest{
			TenantId: validTenant,
			Id:       validID,
			UserId:   "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestRejectBankTransactionMatch_Validation(t *testing.T) {
	ctx := context.Background()
	validTenant := uuid.New().String()
	validID := uuid.New().String()

	t.Run("service_not_configured", func(t *testing.T) {
		s := &BizGRPCServer{}
		_, err := s.RejectBankTransactionMatch(ctx, &bizv1.RejectBankTransactionMatchRequest{TenantId: validTenant, Id: validID})
		requireGRPCCode(t, err, codes.Unimplemented)
	})
	t.Run("invalid_tenant_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.RejectBankTransactionMatch(ctx, &bizv1.RejectBankTransactionMatchRequest{TenantId: "not-a-uuid", Id: validID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.RejectBankTransactionMatch(ctx, &bizv1.RejectBankTransactionMatchRequest{TenantId: validTenant, Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_user_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.RejectBankTransactionMatch(ctx, &bizv1.RejectBankTransactionMatchRequest{
			TenantId: validTenant,
			Id:       validID,
			UserId:   "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestIgnoreBankTransaction_Validation(t *testing.T) {
	ctx := context.Background()
	validTenant := uuid.New().String()
	validID := uuid.New().String()

	t.Run("service_not_configured", func(t *testing.T) {
		s := &BizGRPCServer{}
		_, err := s.IgnoreBankTransaction(ctx, &bizv1.IgnoreBankTransactionRequest{TenantId: validTenant, Id: validID})
		requireGRPCCode(t, err, codes.Unimplemented)
	})
	t.Run("invalid_tenant_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.IgnoreBankTransaction(ctx, &bizv1.IgnoreBankTransactionRequest{TenantId: "not-a-uuid", Id: validID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.IgnoreBankTransaction(ctx, &bizv1.IgnoreBankTransactionRequest{TenantId: validTenant, Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_user_id", func(t *testing.T) {
		s := newTestBankingServer()
		_, err := s.IgnoreBankTransaction(ctx, &bizv1.IgnoreBankTransactionRequest{
			TenantId: validTenant,
			Id:       validID,
			UserId:   "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestParseOptionalUUID(t *testing.T) {
	t.Run("empty_is_nil", func(t *testing.T) {
		id, err := parseOptionalUUID("")
		require.NoError(t, err)
		require.Equal(t, uuid.Nil, id)
	})
	t.Run("invalid_errors", func(t *testing.T) {
		_, err := parseOptionalUUID("not-a-uuid")
		require.Error(t, err)
	})
	t.Run("valid_roundtrips", func(t *testing.T) {
		want := uuid.New()
		got, err := parseOptionalUUID(want.String())
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

// ---------------------------------------------------------------------------
// Bexio RPCs — validation paths.
//
// bexioService is *bexio.Service, wired through ten constructor parameters
// (client, repo, config repo, vault, ...). Every case below is rejected
// before the handler dereferences bexioService, so a zero-value
// BexioGRPCServer{} (nil service) exercises the validation path without
// standing up the full dependency graph — deep happy-path coverage per
// method is left for a dedicated unit (see BACKLOG).
// ---------------------------------------------------------------------------

func TestGetBexioAuthURL_Validation(t *testing.T) {
	s := &BexioGRPCServer{}
	_, err := s.GetBexioAuthURL(context.Background(), &bizv1.GetBexioAuthURLRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestHandleBexioOAuthCallback_Validation(t *testing.T) {
	ctx := context.Background()
	s := &BexioGRPCServer{}

	t.Run("invalid_tenant_id", func(t *testing.T) {
		_, err := s.HandleBexioOAuthCallback(ctx, &bizv1.HandleBexioOAuthCallbackRequest{TenantId: "not-a-uuid", Code: "abc"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("missing_code", func(t *testing.T) {
		_, err := s.HandleBexioOAuthCallback(ctx, &bizv1.HandleBexioOAuthCallbackRequest{TenantId: uuid.New().String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestDisconnectBexio_Validation(t *testing.T) {
	s := &BexioGRPCServer{}
	_, err := s.DisconnectBexio(context.Background(), &bizv1.DisconnectBexioRequest{TenantId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetBexioConnectionStatus_Validation(t *testing.T) {
	s := &BexioGRPCServer{}
	_, err := s.GetBexioConnectionStatus(context.Background(), &bizv1.GetBexioConnectionStatusRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestTriggerBexioSync_Validation(t *testing.T) {
	s := &BexioGRPCServer{}
	_, err := s.TriggerBexioSync(context.Background(), &bizv1.TriggerBexioSyncRequest{TenantId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetBexioSyncStatus_Validation(t *testing.T) {
	s := &BexioGRPCServer{}
	_, err := s.GetBexioSyncStatus(context.Background(), &bizv1.GetBexioSyncStatusRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateBexioSyncConfig_Validation(t *testing.T) {
	s := &BexioGRPCServer{}
	_, err := s.UpdateBexioSyncConfig(context.Background(), &bizv1.UpdateBexioSyncConfigRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListBexioSyncLogs_Validation(t *testing.T) {
	s := &BexioGRPCServer{}
	_, err := s.ListBexioSyncLogs(context.Background(), &bizv1.ListBexioSyncLogsRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetBexioFieldMappings_Validation(t *testing.T) {
	ctx := context.Background()
	s := &BexioGRPCServer{}

	t.Run("missing_tenant", func(t *testing.T) {
		_, err := s.GetBexioFieldMappings(ctx, &bizv1.GetBexioFieldMappingsRequest{EntityType: "contact"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("missing_entity_type", func(t *testing.T) {
		_, err := s.GetBexioFieldMappings(ctx, &bizv1.GetBexioFieldMappingsRequest{TenantId: uuid.New().String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestUpdateBexioFieldMappings_Validation(t *testing.T) {
	ctx := context.Background()
	s := &BexioGRPCServer{}

	t.Run("invalid_tenant_id", func(t *testing.T) {
		_, err := s.UpdateBexioFieldMappings(ctx, &bizv1.UpdateBexioFieldMappingsRequest{TenantId: "not-a-uuid", EntityType: "contact"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("missing_entity_type", func(t *testing.T) {
		_, err := s.UpdateBexioFieldMappings(ctx, &bizv1.UpdateBexioFieldMappingsRequest{TenantId: uuid.New().String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestPushInvoiceToBexio_Validation(t *testing.T) {
	ctx := context.Background()
	s := &BexioGRPCServer{}
	validTenant := uuid.New().String()

	t.Run("invalid_tenant_id", func(t *testing.T) {
		_, err := s.PushInvoiceToBexio(ctx, &bizv1.PushInvoiceToBexioRequest{TenantId: "not-a-uuid", InvoiceId: uuid.New().String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_invoice_id", func(t *testing.T) {
		_, err := s.PushInvoiceToBexio(ctx, &bizv1.PushInvoiceToBexioRequest{TenantId: validTenant, InvoiceId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestPushQuoteToBexio_Validation(t *testing.T) {
	ctx := context.Background()
	s := &BexioGRPCServer{}
	validTenant := uuid.New().String()

	t.Run("invalid_tenant_id", func(t *testing.T) {
		_, err := s.PushQuoteToBexio(ctx, &bizv1.PushQuoteToBexioRequest{TenantId: "not-a-uuid", QuoteId: uuid.New().String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("invalid_quote_id", func(t *testing.T) {
		_, err := s.PushQuoteToBexio(ctx, &bizv1.PushQuoteToBexioRequest{TenantId: validTenant, QuoteId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}
