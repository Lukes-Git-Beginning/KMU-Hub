package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/bexio"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// BexioGRPCServer implements the BexioIntegrationService gRPC service.
type BexioGRPCServer struct {
	bizv1.UnimplementedBexioIntegrationServiceServer
	bexioService *bexio.Service
}

// NewBexioGRPCServer creates a new BexioGRPCServer.
func NewBexioGRPCServer(bexioService *bexio.Service) *BexioGRPCServer {
	return &BexioGRPCServer{bexioService: bexioService}
}

// GetBexioAuthURL returns the Bexio OAuth authorization URL.
func (s *BexioGRPCServer) GetBexioAuthURL(_ context.Context, req *bizv1.GetBexioAuthURLRequest) (*bizv1.GetBexioAuthURLResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	url := s.bexioService.GetAuthorizationURL(req.GetTenantId())
	return &bizv1.GetBexioAuthURLResponse{
		AuthorizationUrl: url,
	}, nil
}

// HandleBexioOAuthCallback exchanges the authorization code for tokens.
func (s *BexioGRPCServer) HandleBexioOAuthCallback(ctx context.Context, req *bizv1.HandleBexioOAuthCallbackRequest) (*bizv1.HandleBexioOAuthCallbackResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	if req.GetCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	if err := s.bexioService.HandleOAuthCallback(ctx, tenantID, req.GetCode()); err != nil {
		slog.Error("bexio oauth callback failed", "tenant_id", tenantID, "error", err)
		return nil, mapBexioError(err)
	}

	return &bizv1.HandleBexioOAuthCallbackResponse{Success: true}, nil
}

// DisconnectBexio revokes tokens and stops the integration.
func (s *BexioGRPCServer) DisconnectBexio(ctx context.Context, req *bizv1.DisconnectBexioRequest) (*bizv1.DisconnectBexioResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	if err := s.bexioService.Disconnect(ctx, tenantID); err != nil {
		return nil, mapBexioError(err)
	}

	return &bizv1.DisconnectBexioResponse{Success: true}, nil
}

// GetBexioConnectionStatus returns the current connection state.
func (s *BexioGRPCServer) GetBexioConnectionStatus(ctx context.Context, req *bizv1.GetBexioConnectionStatusRequest) (*bizv1.GetBexioConnectionStatusResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	connStatus, err := s.bexioService.GetConnectionStatus(ctx)
	if err != nil {
		return nil, mapBexioError(err)
	}

	resp := &bizv1.GetBexioConnectionStatusResponse{
		Connected: connStatus.Connected,
		OrgName:   connStatus.OrgName,
	}
	if connStatus.ConnectedAt != nil {
		resp.ConnectedAt = timestamppb.New(*connStatus.ConnectedAt)
	}

	return resp, nil
}

// TriggerBexioSync triggers a manual sync.
func (s *BexioGRPCServer) TriggerBexioSync(ctx context.Context, req *bizv1.TriggerBexioSyncRequest) (*bizv1.TriggerBexioSyncResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	syncID := uuid.New().String()

	switch req.GetSyncType() {
	case "contacts":
		if _, err := s.bexioService.SyncContacts(ctx, tenantID); err != nil {
			return nil, mapBexioError(err)
		}
	case "payments":
		if _, err := s.bexioService.PollPayments(ctx, tenantID); err != nil {
			return nil, mapBexioError(err)
		}
	case "invoices":
		if _, err := s.bexioService.PullInvoices(ctx, tenantID); err != nil {
			return nil, mapBexioError(err)
		}
	default:
		if err := s.bexioService.TriggerSync(ctx, tenantID); err != nil {
			return nil, mapBexioError(err)
		}
	}

	return &bizv1.TriggerBexioSyncResponse{
		SyncId: syncID,
		Status: "started",
	}, nil
}

// GetBexioSyncStatus returns the sync status overview.
func (s *BexioGRPCServer) GetBexioSyncStatus(ctx context.Context, req *bizv1.GetBexioSyncStatusRequest) (*bizv1.GetBexioSyncStatusResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	syncStatus, err := s.bexioService.GetSyncStatus(ctx)
	if err != nil {
		return nil, mapBexioError(err)
	}

	return syncStatusToProto(syncStatus), nil
}

// UpdateBexioSyncConfig persists updated sync flags and polling intervals.
func (s *BexioGRPCServer) UpdateBexioSyncConfig(ctx context.Context, req *bizv1.UpdateBexioSyncConfigRequest) (*bizv1.UpdateBexioSyncConfigResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	update := &models.BexioSyncConfig{
		ContactSyncEnabled:     req.GetContactSyncEnabled(),
		ContactSyncIntervalMin: int(req.GetContactSyncIntervalMinutes()),
		InvoicePushEnabled:     req.GetInvoicePushEnabled(),
		QuotePushEnabled:       req.GetQuotePushEnabled(),
		PaymentPollEnabled:     req.GetPaymentPollEnabled(),
		PaymentPollIntervalMin: int(req.GetPaymentPollIntervalMinutes()),
		InvoicePullEnabled:     req.GetInvoicePullEnabled(),
		InvoicePullIntervalMin: int(req.GetInvoicePullIntervalMinutes()),
	}

	if err := s.bexioService.UpdateSyncConfig(ctx, update); err != nil {
		return nil, mapBexioError(err)
	}

	return &bizv1.UpdateBexioSyncConfigResponse{Success: true}, nil
}

// ListBexioSyncLogs returns recent sync log entries.
func (s *BexioGRPCServer) ListBexioSyncLogs(ctx context.Context, req *bizv1.ListBexioSyncLogsRequest) (*bizv1.ListBexioSyncLogsResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}

	logs, err := s.bexioService.GetSyncLogs(ctx, limit)
	if err != nil {
		return nil, mapBexioError(err)
	}

	entries := make([]*bizv1.BexioSyncLogEntry, 0, len(logs))
	for _, l := range logs {
		entries = append(entries, syncLogToProto(l))
	}

	return &bizv1.ListBexioSyncLogsResponse{Entries: entries}, nil
}

// GetBexioFieldMappings returns field mappings for an entity type.
func (s *BexioGRPCServer) GetBexioFieldMappings(ctx context.Context, req *bizv1.GetBexioFieldMappingsRequest) (*bizv1.GetBexioFieldMappingsResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}
	if req.GetEntityType() == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_type is required")
	}

	fm, err := s.bexioService.GetFieldMappings(ctx, req.GetEntityType())
	if err != nil {
		return nil, mapBexioError(err)
	}

	if fm == nil {
		return &bizv1.GetBexioFieldMappingsResponse{}, nil
	}

	mappings := make([]*bizv1.BexioFieldMappingEntry, 0, len(fm.Mappings))
	for _, m := range fm.Mappings {
		mappings = append(mappings, &bizv1.BexioFieldMappingEntry{
			KmuhubField: m.KmuhubField,
			BexioField:  m.BexioField,
			Direction:   m.Direction,
			Required:    m.Required,
		})
	}

	return &bizv1.GetBexioFieldMappingsResponse{Mappings: mappings}, nil
}

// UpdateBexioFieldMappings validates and saves field mappings.
func (s *BexioGRPCServer) UpdateBexioFieldMappings(ctx context.Context, req *bizv1.UpdateBexioFieldMappingsRequest) (*bizv1.UpdateBexioFieldMappingsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	if req.GetEntityType() == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_type is required")
	}

	mappings := make([]models.BexioFieldMappingEntry, 0, len(req.GetMappings()))
	for _, m := range req.GetMappings() {
		mappings = append(mappings, models.BexioFieldMappingEntry{
			KmuhubField: m.GetKmuhubField(),
			BexioField:  m.GetBexioField(),
			Direction:   m.GetDirection(),
			Required:    m.GetRequired(),
		})
	}

	if err := s.bexioService.UpdateFieldMappings(ctx, tenantID, req.GetEntityType(), mappings); err != nil {
		return nil, mapBexioError(err)
	}

	return &bizv1.UpdateBexioFieldMappingsResponse{Success: true}, nil
}

// PushInvoiceToBexio pushes an invoice to Bexio.
func (s *BexioGRPCServer) PushInvoiceToBexio(ctx context.Context, req *bizv1.PushInvoiceToBexioRequest) (*bizv1.PushInvoiceToBexioResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	invoiceID, err := uuid.Parse(req.GetInvoiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
	}

	if err := s.bexioService.PushInvoice(ctx, tenantID, invoiceID); err != nil {
		return nil, mapBexioError(err)
	}

	return &bizv1.PushInvoiceToBexioResponse{Success: true}, nil
}

// PushQuoteToBexio pushes a quote to Bexio.
func (s *BexioGRPCServer) PushQuoteToBexio(ctx context.Context, req *bizv1.PushQuoteToBexioRequest) (*bizv1.PushQuoteToBexioResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	quoteID, err := uuid.Parse(req.GetQuoteId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid quote_id")
	}

	if err := s.bexioService.PushQuote(ctx, tenantID, quoteID); err != nil {
		return nil, mapBexioError(err)
	}

	return &bizv1.PushQuoteToBexioResponse{Success: true}, nil
}

// mapBexioError converts Bexio service errors to gRPC status codes.
func mapBexioError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, bexio.ErrBexioUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, bexio.ErrBexioRateLimited):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, bexio.ErrBexioNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, bexio.ErrConfigNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, bexio.ErrMappingNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, bexio.ErrSyncAlreadyRunning):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, bexio.ErrSyncConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, bexio.ErrInvalidFieldMapping):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, bexio.ErrBexioServerError):
		return status.Error(codes.Unavailable, err.Error())
	default:
		slog.Error("unhandled bexio service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

// syncStatusToProto converts a BexioSyncStatus to the gRPC response.
func syncStatusToProto(s *models.BexioSyncStatus) *bizv1.GetBexioSyncStatusResponse {
	resp := &bizv1.GetBexioSyncStatusResponse{
		ContactSyncEnabled:  s.ContactSyncEnabled,
		InvoicePushEnabled:  s.InvoicePushEnabled,
		QuotePushEnabled:    s.QuotePushEnabled,
		PaymentPollEnabled:  s.PaymentPollEnabled,
		InvoicePullEnabled:  s.InvoicePullEnabled,
		TotalContactsMapped: int32(s.TotalContactsMapped),
		TotalInvoicesMapped: int32(s.TotalInvoicesMapped),
		TotalQuotesMapped:   int32(s.TotalQuotesMapped),
	}

	if s.LastContactSyncAt != nil {
		resp.LastContactSyncAt = timestamppb.New(*s.LastContactSyncAt)
	}
	if s.LastPaymentPollAt != nil {
		resp.LastPaymentPollAt = timestamppb.New(*s.LastPaymentPollAt)
	}
	if s.LastInvoicePullAt != nil {
		resp.LastInvoicePullAt = timestamppb.New(*s.LastInvoicePullAt)
	}
	if s.LastSyncError != nil {
		resp.LastSyncError = *s.LastSyncError
	}
	if s.LastSyncErrorAt != nil {
		resp.LastSyncErrorAt = timestamppb.New(*s.LastSyncErrorAt)
	}

	return resp
}

// syncLogToProto converts a BexioSyncLog to the gRPC entry.
func syncLogToProto(l models.BexioSyncLog) *bizv1.BexioSyncLogEntry {
	entry := &bizv1.BexioSyncLogEntry{
		Id:             l.ID.String(),
		SyncType:       l.SyncType,
		Status:         l.Status,
		ItemsProcessed: int32(l.ItemsProcessed),
		ItemsCreated:   int32(l.ItemsCreated),
		ItemsUpdated:   int32(l.ItemsUpdated),
		ItemsFailed:    int32(l.ItemsFailed),
		StartedAt:      timestamppb.New(l.StartedAt),
	}

	if l.ErrorMessage != nil {
		entry.ErrorMessage = *l.ErrorMessage
	}
	if l.CompletedAt != nil {
		entry.CompletedAt = timestamppb.New(*l.CompletedAt)
	}

	return entry
}

