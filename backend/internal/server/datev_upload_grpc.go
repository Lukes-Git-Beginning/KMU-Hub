package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/datev"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// datevDateLayout is the wire format for the period fields of the upload RPCs.
const datevDateLayout = "2006-01-02"

// DatevUploadGRPCServer implements the DatevUploadService gRPC service.
type DatevUploadGRPCServer struct {
	bizv1.UnimplementedDatevUploadServiceServer
	uploadService *datev.UploadService
}

// NewDatevUploadGRPCServer creates a new DatevUploadGRPCServer.
func NewDatevUploadGRPCServer(uploadService *datev.UploadService) *DatevUploadGRPCServer {
	return &DatevUploadGRPCServer{uploadService: uploadService}
}

// GetDatevAuthURL returns the DATEV OAuth authorization URL.
func (s *DatevUploadGRPCServer) GetDatevAuthURL(_ context.Context, req *bizv1.GetDatevAuthURLRequest) (*bizv1.GetDatevAuthURLResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	authBaseURL := "https://login.datev.de/openid/authorize"
	url := s.uploadService.GetAuthorizationURL(tenantID, req.GetRedirectUrl(), authBaseURL)
	if url == "" {
		return nil, status.Error(codes.FailedPrecondition, "DATEV OAuth not configured")
	}

	return &bizv1.GetDatevAuthURLResponse{
		AuthorizationUrl: url,
	}, nil
}

// HandleDatevOAuthCallback exchanges the authorization code for tokens.
func (s *DatevUploadGRPCServer) HandleDatevOAuthCallback(ctx context.Context, req *bizv1.HandleDatevOAuthCallbackRequest) (*bizv1.HandleDatevOAuthCallbackResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	if req.GetCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	if err := s.uploadService.HandleOAuthCallback(ctx, tenantID, req.GetCode(), req.GetRedirectUrl()); err != nil {
		slog.Error("datev oauth callback failed", "tenant_id", tenantID, "error", err)
		// ExchangeCode wraps Vault/HTTP internals (e.g. "failed to store refresh
		// token: %w"); a masked status keeps that text out of the redirect the
		// gateway builds from this error.
		return nil, status.Error(codes.Internal, "DATEV OAuth callback failed")
	}

	return &bizv1.HandleDatevOAuthCallbackResponse{Success: true}, nil
}

// DisconnectDatev revokes tokens and deactivates the integration.
func (s *DatevUploadGRPCServer) DisconnectDatev(ctx context.Context, req *bizv1.DisconnectDatevRequest) (*bizv1.DisconnectDatevResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	if err := s.uploadService.Disconnect(ctx, tenantID); err != nil {
		slog.Error("datev disconnect failed", "tenant_id", tenantID, "error", err)
		return nil, status.Error(codes.Internal, "failed to disconnect")
	}

	return &bizv1.DisconnectDatevResponse{Success: true}, nil
}

// GetDatevConnectionStatus returns the current DATEV connection state.
func (s *DatevUploadGRPCServer) GetDatevConnectionStatus(ctx context.Context, req *bizv1.GetDatevConnectionStatusRequest) (*bizv1.GetDatevConnectionStatusResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	connected, err := s.uploadService.GetConnectionStatus(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get connection status")
	}

	return &bizv1.GetDatevConnectionStatusResponse{
		Connected: connected,
	}, nil
}

// UploadDatevBuchungsstapel renders the tenant's booking batch for the requested
// period and transfers it to DATEV.
//
// The period is mandatory here, unlike in the transport layer where the fields
// are optional: an absent range used to mean "everything the tenant ever
// booked", which is not a request anyone makes on purpose. Success requires that
// the batch actually reached DATEV — the predecessor of this RPC uploaded a
// document-less CSV and wrote a "completed" log for it.
func (s *DatevUploadGRPCServer) UploadDatevBuchungsstapel(ctx context.Context, req *bizv1.UploadDatevBuchungsstapelRequest) (*bizv1.UploadDatevBuchungsstapelResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	startDate, err := time.Parse(datevDateLayout, req.GetStartDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid start_date (expected YYYY-MM-DD)")
	}
	endDate, err := time.Parse(datevDateLayout, req.GetEndDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid end_date (expected YYYY-MM-DD)")
	}

	fiscalYearStart := startDate
	if req.GetFiscalYearStart() != "" {
		fys, fysErr := time.Parse(datevDateLayout, req.GetFiscalYearStart())
		if fysErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid fiscal_year_start (expected YYYY-MM-DD)")
		}
		fiscalYearStart = fys
	}

	batch, err := s.uploadService.UploadBuchungsstapel(ctx, tenantID, startDate, endDate, fiscalYearStart)
	if err != nil {
		slog.Error("datev buchungsstapel upload failed", "tenant_id", tenantID, "error", err)
		return nil, mapDatevUploadError(err)
	}

	return &bizv1.UploadDatevBuchungsstapelResponse{
		Success:       true,
		DocumentCount: int32(batch.DocumentCount),
		FileSize:      int32(len(batch.CSV)),
	}, nil
}

// UploadDatevBeleg renders the invoice PDF and transfers it as a Belegbild.
// It reports success only after the transfer returned without error; the
// predecessor logged the request and claimed success without fetching a PDF.
func (s *DatevUploadGRPCServer) UploadDatevBeleg(ctx context.Context, req *bizv1.UploadDatevBelegRequest) (*bizv1.UploadDatevBelegResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	invoiceID, err := uuid.Parse(req.GetInvoiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
	}

	if err := s.uploadService.UploadInvoiceBeleg(ctx, tenantID, invoiceID); err != nil {
		slog.Error("datev belegbild upload failed", "tenant_id", tenantID, "invoice_id", invoiceID, "error", err)
		return nil, mapDatevUploadError(err)
	}

	return &bizv1.UploadDatevBelegResponse{Success: true}, nil
}

// mapDatevUploadError translates the upload service's sentinels into gRPC codes.
// Everything the tenant can fix — a missing OAuth connection, an unset client or
// advisor number, an empty period — is FailedPrecondition and carries its reason
// to the client, because "upload failed" alone leaves an admin without a next
// step. There is deliberately no success=false path: the field would be omitted
// from the JSON entirely by the proto marshaler.
func mapDatevUploadError(err error) error {
	switch {
	case errors.Is(err, datev.ErrInvoiceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, datev.ErrInvalidPeriod):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, datev.ErrNotConnected),
		errors.Is(err, datev.ErrNoAPIConfig),
		errors.Is(err, datev.ErrNoUploadConfig),
		errors.Is(err, datev.ErrAdvisorNumbersMissing),
		errors.Is(err, datev.ErrCompanySettingsIncomplete),
		errors.Is(err, datev.ErrNothingToUpload):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, datev.ErrBuilderNotConfigured):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, "DATEV upload failed")
	}
}

// GetDatevUploadConfig returns the current DATEV upload configuration.
func (s *DatevUploadGRPCServer) GetDatevUploadConfig(ctx context.Context, req *bizv1.GetDatevUploadConfigRequest) (*bizv1.GetDatevUploadConfigResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	cfg, err := s.uploadService.GetUploadConfig(ctx)
	if err != nil {
		return nil, status.Error(codes.NotFound, "upload config not found")
	}

	return &bizv1.GetDatevUploadConfigResponse{
		ClientNumber:      cfg.ClientNumber,
		AutoUploadEnabled: cfg.AutoUploadEnabled,
		UploadAfterExport: cfg.UploadAfterExport,
	}, nil
}

// UpdateDatevUploadConfig updates the DATEV upload configuration.
func (s *DatevUploadGRPCServer) UpdateDatevUploadConfig(ctx context.Context, req *bizv1.UpdateDatevUploadConfigRequest) (*bizv1.UpdateDatevUploadConfigResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	cfg := &models.DatevUploadConfig{
		ClientNumber:      req.GetClientNumber(),
		AutoUploadEnabled: req.GetAutoUploadEnabled(),
		UploadAfterExport: req.GetUploadAfterExport(),
	}

	if err := s.uploadService.UpdateUploadConfig(ctx, cfg); err != nil {
		slog.Error("datev update upload config failed", "error", err)
		return nil, status.Error(codes.Internal, "failed to update config")
	}

	return &bizv1.UpdateDatevUploadConfigResponse{Success: true}, nil
}

// ListDatevUploadLogs returns recent DATEV upload log entries.
func (s *DatevUploadGRPCServer) ListDatevUploadLogs(ctx context.Context, req *bizv1.ListDatevUploadLogsRequest) (*bizv1.ListDatevUploadLogsResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}

	logs, err := s.uploadService.ListUploadLogs(ctx, limit)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list upload logs")
	}

	entries := make([]*bizv1.DatevUploadLogEntry, 0, len(logs))
	for _, l := range logs {
		entries = append(entries, datevUploadLogToProto(l))
	}

	return &bizv1.ListDatevUploadLogsResponse{Entries: entries}, nil
}

// datevUploadLogToProto converts a DatevUploadLog to the gRPC entry.
func datevUploadLogToProto(l models.DatevUploadLog) *bizv1.DatevUploadLogEntry {
	entry := &bizv1.DatevUploadLogEntry{
		Id:            l.ID.String(),
		UploadType:    l.UploadType,
		Status:        l.Status,
		FileSize:      int32(l.FileSize),
		DocumentCount: int32(l.DocumentCount),
		StartedAt:     timestamppb.New(l.StartedAt),
	}

	if l.ErrorMessage != nil {
		entry.ErrorMessage = *l.ErrorMessage
	}
	if l.CompletedAt != nil {
		entry.CompletedAt = timestamppb.New(*l.CompletedAt)
	}

	return entry
}
