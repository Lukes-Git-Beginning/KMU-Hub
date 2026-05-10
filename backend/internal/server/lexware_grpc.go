package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/lexware"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// LexwareGRPCServer implements the LexwareIntegrationService gRPC service.
type LexwareGRPCServer struct {
	bizv1.UnimplementedLexwareIntegrationServiceServer
	lexwareService *lexware.Service
}

// NewLexwareGRPCServer creates a new LexwareGRPCServer.
func NewLexwareGRPCServer(lexwareService *lexware.Service) *LexwareGRPCServer {
	return &LexwareGRPCServer{lexwareService: lexwareService}
}

// ConnectLexware stores the API key and activates the integration.
func (s *LexwareGRPCServer) ConnectLexware(ctx context.Context, req *bizv1.ConnectLexwareRequest) (*bizv1.ConnectLexwareResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	if req.GetApiKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "api_key is required")
	}

	if err := s.lexwareService.Connect(ctx, tenantID, req.GetApiKey()); err != nil {
		slog.Error("lexware connect failed", "tenant_id", tenantID, "error", err)
		return &bizv1.ConnectLexwareResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &bizv1.ConnectLexwareResponse{Success: true}, nil
}

// DisconnectLexware deactivates the integration and stops the scheduler.
func (s *LexwareGRPCServer) DisconnectLexware(ctx context.Context, req *bizv1.DisconnectLexwareRequest) (*bizv1.DisconnectLexwareResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	if err := s.lexwareService.Disconnect(ctx, tenantID); err != nil {
		return nil, mapLexwareError(err)
	}

	return &bizv1.DisconnectLexwareResponse{Success: true}, nil
}

// GetLexwareConnectionStatus returns the current connection state.
func (s *LexwareGRPCServer) GetLexwareConnectionStatus(ctx context.Context, req *bizv1.GetLexwareConnectionStatusRequest) (*bizv1.GetLexwareConnectionStatusResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	connStatus, err := s.lexwareService.GetConnectionStatus(ctx)
	if err != nil {
		return nil, mapLexwareError(err)
	}

	resp := &bizv1.GetLexwareConnectionStatusResponse{
		Connected: connStatus.Connected,
	}
	if connStatus.ConnectedAt != nil {
		resp.ConnectedAt = timestamppb.New(*connStatus.ConnectedAt)
	}

	return resp, nil
}

// TestLexwareConnection verifies the stored API key is valid.
func (s *LexwareGRPCServer) TestLexwareConnection(ctx context.Context, req *bizv1.TestLexwareConnectionRequest) (*bizv1.TestLexwareConnectionResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	if err := s.lexwareService.TestConnection(ctx); err != nil {
		return &bizv1.TestLexwareConnectionResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &bizv1.TestLexwareConnectionResponse{Success: true}, nil
}

// TriggerLexwareSync triggers a manual sync.
func (s *LexwareGRPCServer) TriggerLexwareSync(ctx context.Context, req *bizv1.TriggerLexwareSyncRequest) (*bizv1.TriggerLexwareSyncResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	syncID := uuid.New().String()

	switch req.GetSyncType() {
	case "contacts":
		if _, err := s.lexwareService.SyncContacts(ctx, tenantID); err != nil {
			return nil, mapLexwareError(err)
		}
	default:
		if err := s.lexwareService.TriggerSync(ctx, tenantID); err != nil {
			return nil, mapLexwareError(err)
		}
	}

	return &bizv1.TriggerLexwareSyncResponse{
		SyncId: syncID,
		Status: "started",
	}, nil
}

// GetLexwareSyncStatus returns the sync status overview.
func (s *LexwareGRPCServer) GetLexwareSyncStatus(ctx context.Context, req *bizv1.GetLexwareSyncStatusRequest) (*bizv1.GetLexwareSyncStatusResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	syncStatus, err := s.lexwareService.GetSyncStatus(ctx)
	if err != nil {
		return nil, mapLexwareError(err)
	}

	return lexwareSyncStatusToProto(syncStatus), nil
}

// ListLexwareSyncLogs returns recent sync log entries.
func (s *LexwareGRPCServer) ListLexwareSyncLogs(ctx context.Context, req *bizv1.ListLexwareSyncLogsRequest) (*bizv1.ListLexwareSyncLogsResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}

	logs, err := s.lexwareService.GetSyncLogs(ctx, limit)
	if err != nil {
		return nil, mapLexwareError(err)
	}

	entries := make([]*bizv1.LexwareSyncLogEntry, 0, len(logs))
	for _, l := range logs {
		entries = append(entries, lexwareSyncLogToProto(l))
	}

	return &bizv1.ListLexwareSyncLogsResponse{Entries: entries}, nil
}

// GetLexwareFieldMappings returns field mappings for an entity type.
func (s *LexwareGRPCServer) GetLexwareFieldMappings(ctx context.Context, req *bizv1.GetLexwareFieldMappingsRequest) (*bizv1.GetLexwareFieldMappingsResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}
	if req.GetEntityType() == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_type is required")
	}

	fm, err := s.lexwareService.GetFieldMappings(ctx, req.GetEntityType())
	if err != nil {
		return nil, mapLexwareError(err)
	}

	if fm == nil {
		return &bizv1.GetLexwareFieldMappingsResponse{}, nil
	}

	mappings := make([]*bizv1.LexwareFieldMappingEntry, 0, len(fm.Mappings))
	for _, m := range fm.Mappings {
		mappings = append(mappings, &bizv1.LexwareFieldMappingEntry{
			KmuhubField:  m.KmuhubField,
			LexwareField: m.LexwareField,
			Direction:    m.Direction,
			Required:     m.Required,
		})
	}

	return &bizv1.GetLexwareFieldMappingsResponse{Mappings: mappings}, nil
}

// UpdateLexwareFieldMappings validates and saves field mappings.
func (s *LexwareGRPCServer) UpdateLexwareFieldMappings(ctx context.Context, req *bizv1.UpdateLexwareFieldMappingsRequest) (*bizv1.UpdateLexwareFieldMappingsResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}
	if req.GetEntityType() == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_type is required")
	}

	mappings := make([]models.LexwareFieldMappingEntry, 0, len(req.GetMappings()))
	for _, m := range req.GetMappings() {
		mappings = append(mappings, models.LexwareFieldMappingEntry{
			KmuhubField:  m.GetKmuhubField(),
			LexwareField: m.GetLexwareField(),
			Direction:    m.GetDirection(),
			Required:     m.GetRequired(),
		})
	}

	if err := s.lexwareService.UpdateFieldMappings(ctx, req.GetEntityType(), mappings); err != nil {
		return nil, mapLexwareError(err)
	}

	return &bizv1.UpdateLexwareFieldMappingsResponse{Success: true}, nil
}

// PushInvoiceToLexware pushes an invoice to Lexware.
func (s *LexwareGRPCServer) PushInvoiceToLexware(ctx context.Context, req *bizv1.PushInvoiceToLexwareRequest) (*bizv1.PushInvoiceToLexwareResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	invoiceID, err := uuid.Parse(req.GetInvoiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
	}

	if err := s.lexwareService.PushInvoice(ctx, tenantID, invoiceID); err != nil {
		return &bizv1.PushInvoiceToLexwareResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &bizv1.PushInvoiceToLexwareResponse{Success: true}, nil
}

// PushQuoteToLexware pushes a quote to Lexware.
func (s *LexwareGRPCServer) PushQuoteToLexware(ctx context.Context, req *bizv1.PushQuoteToLexwareRequest) (*bizv1.PushQuoteToLexwareResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	quoteID, err := uuid.Parse(req.GetQuoteId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid quote_id")
	}

	if err := s.lexwareService.PushQuote(ctx, tenantID, quoteID); err != nil {
		return &bizv1.PushQuoteToLexwareResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &bizv1.PushQuoteToLexwareResponse{Success: true}, nil
}

// HandleLexwareWebhookEvent processes an incoming Lexware webhook event.
func (s *LexwareGRPCServer) HandleLexwareWebhookEvent(ctx context.Context, req *bizv1.HandleLexwareWebhookEventRequest) (*bizv1.HandleLexwareWebhookEventResponse, error) {
	if req.GetEventType() == "" {
		return nil, status.Error(codes.InvalidArgument, "event_type is required")
	}

	payload := map[string]any{
		"resource_id":     req.GetResourceId(),
		"organization_id": req.GetOrganizationId(),
	}

	if err := s.lexwareService.HandleWebhookEvent(ctx, req.GetEventType(), payload); err != nil {
		slog.Error("lexware webhook processing failed",
			"event_type", req.GetEventType(),
			"error", err,
		)
		return &bizv1.HandleLexwareWebhookEventResponse{Success: false}, nil
	}

	return &bizv1.HandleLexwareWebhookEventResponse{Success: true}, nil
}

// mapLexwareError converts Lexware service errors to gRPC status codes.
func mapLexwareError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, lexware.ErrLexwareUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, lexware.ErrLexwareRateLimited):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, lexware.ErrLexwareNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, lexware.ErrConfigNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, lexware.ErrMappingNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, lexware.ErrSyncAlreadyRunning):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, lexware.ErrSyncConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, lexware.ErrLexwareVersionConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, lexware.ErrInvalidFieldMapping):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, lexware.ErrLexwareServerError):
		return status.Error(codes.Unavailable, err.Error())
	default:
		slog.Error("unhandled lexware service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

// lexwareSyncStatusToProto converts a LexwareSyncStatus to the gRPC response.
func lexwareSyncStatusToProto(s *models.LexwareSyncStatus) *bizv1.GetLexwareSyncStatusResponse {
	resp := &bizv1.GetLexwareSyncStatusResponse{
		ContactSyncEnabled:  s.ContactSyncEnabled,
		InvoicePushEnabled:  s.InvoicePushEnabled,
		QuotePushEnabled:    s.QuotePushEnabled,
		WebhookEnabled:      s.WebhookEnabled,
		TotalContactsMapped: int32(s.TotalContactsMapped),
		TotalInvoicesMapped: int32(s.TotalInvoicesMapped),
		TotalQuotesMapped:   int32(s.TotalQuotesMapped),
	}

	if s.LastContactSyncAt != nil {
		resp.LastContactSyncAt = timestamppb.New(*s.LastContactSyncAt)
	}
	if s.LastSyncError != nil {
		resp.LastSyncError = *s.LastSyncError
	}
	if s.LastSyncErrorAt != nil {
		resp.LastSyncErrorAt = timestamppb.New(*s.LastSyncErrorAt)
	}

	return resp
}

// lexwareSyncLogToProto converts a LexwareSyncLog to the gRPC entry.
func lexwareSyncLogToProto(l models.LexwareSyncLog) *bizv1.LexwareSyncLogEntry {
	entry := &bizv1.LexwareSyncLogEntry{
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
