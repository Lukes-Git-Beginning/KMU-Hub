package server

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/datev"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

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
		return &bizv1.HandleDatevOAuthCallbackResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
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

// UploadDatevBuchungsstapel is not implemented yet.
//
// It used to call UploadService.ExportAndUpload with empty invoice and credit
// note slices and report Success=true. Against a connected DATEV account that
// does not fail — it exports a document-less CSV, uploads it, and writes a
// "completed" upload log, so the client is told the accounting data reached the
// tax advisor when nothing did. A wrong success on a bookkeeping transfer is
// worse than no endpoint, so the RPC refuses instead. Implementing it means
// loading the tenant's invoices and credit notes for the requested date range
// (plus the Berater/Mandant numbers from company_settings, which only
// BizGRPCServer.ExportDATEV fills today) — see the backend loop backlog.
func (s *DatevUploadGRPCServer) UploadDatevBuchungsstapel(_ context.Context, req *bizv1.UploadDatevBuchungsstapelRequest) (*bizv1.UploadDatevBuchungsstapelResponse, error) {
	if _, err := uuid.Parse(req.GetTenantId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	return nil, status.Error(codes.Unimplemented,
		"DATEV Buchungsstapel upload is not implemented yet — use the GoBD DATEV export")
}

// UploadDatevBeleg is not implemented yet.
//
// It used to log the request and return Success=true without ever retrieving or
// uploading a PDF, which tells the client the Belegbild was transferred when it
// was not. Implementing it means rendering the invoice PDF and passing it to
// UploadService.UploadBeleg — see the backend loop backlog.
func (s *DatevUploadGRPCServer) UploadDatevBeleg(_ context.Context, req *bizv1.UploadDatevBelegRequest) (*bizv1.UploadDatevBelegResponse, error) {
	if _, err := uuid.Parse(req.GetTenantId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	if req.GetInvoiceId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invoice_id is required")
	}
	return nil, status.Error(codes.Unimplemented,
		"DATEV Belegbild upload is not implemented yet")
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
