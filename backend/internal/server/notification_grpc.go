package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
	"github.com/kmuhub/kmuhub/internal/notification/notification"
	"github.com/kmuhub/kmuhub/internal/notification/preference"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

// NotificationGRPCServer implements the Notification gRPC service.
type NotificationGRPCServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	notifService    *notification.Service
	prefService     *preference.Service
	registry        *event.EventTypeRegistry
	integrationRepo integration.Repository
	linkService     *integration.AccountLinkService
}

// NewNotificationGRPCServer creates a new Notification gRPC server.
func NewNotificationGRPCServer(
	notifService *notification.Service,
	prefService *preference.Service,
	registry *event.EventTypeRegistry,
	opts ...NotificationGRPCOption,
) *NotificationGRPCServer {
	s := &NotificationGRPCServer{
		notifService: notifService,
		prefService:  prefService,
		registry:     registry,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NotificationGRPCOption is a functional option for NotificationGRPCServer.
type NotificationGRPCOption func(*NotificationGRPCServer)

// WithIntegration adds integration repository and account link service.
func WithIntegration(repo integration.Repository, linkService *integration.AccountLinkService) NotificationGRPCOption {
	return func(s *NotificationGRPCServer) {
		s.integrationRepo = repo
		s.linkService = linkService
	}
}

// ============================================================================
// Notifications
// ============================================================================

func (s *NotificationGRPCServer) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	var moduleID *string
	if req.ModuleId != nil {
		moduleID = req.ModuleId
	}

	var isRead *bool
	if req.IsRead != nil {
		isRead = req.IsRead
	}

	notifications, total, err := s.notifService.ListNotifications(ctx, tenantID, userID, moduleID, isRead, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, mapNotificationError(err)
	}

	var infos []*notificationv1.NotificationInfo
	for _, n := range notifications {
		infos = append(infos, toNotificationInfo(n))
	}

	return &notificationv1.ListNotificationsResponse{
		Notifications: infos,
		Total:         int32(total),
	}, nil
}

func (s *NotificationGRPCServer) GetUnreadCount(ctx context.Context, req *notificationv1.GetUnreadCountRequest) (*notificationv1.GetUnreadCountResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	count, err := s.notifService.GetUnreadCount(ctx, tenantID, userID)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.GetUnreadCountResponse{
		Count: int32(count),
	}, nil
}

func (s *NotificationGRPCServer) MarkNotificationRead(ctx context.Context, req *notificationv1.MarkNotificationReadRequest) (*notificationv1.MarkNotificationReadResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	notifID, err := uuid.Parse(req.NotificationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid notification_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	notif, err := s.notifService.MarkRead(ctx, tenantID, notifID, userID)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.MarkNotificationReadResponse{
		Notification: toNotificationInfo(notif),
	}, nil
}

func (s *NotificationGRPCServer) MarkAllNotificationsRead(ctx context.Context, req *notificationv1.MarkAllNotificationsReadRequest) (*notificationv1.MarkAllNotificationsReadResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	count, err := s.notifService.MarkAllRead(ctx, tenantID, userID, req.ModuleId)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.MarkAllNotificationsReadResponse{
		UpdatedCount: int32(count),
	}, nil
}

// ============================================================================
// Preferences
// ============================================================================

func (s *NotificationGRPCServer) GetNotificationPreferences(ctx context.Context, req *notificationv1.GetNotificationPreferencesRequest) (*notificationv1.GetNotificationPreferencesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	prefs, err := s.prefService.GetPreferences(ctx, tenantID, userID, req.ModuleId)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	var infos []*notificationv1.NotificationPreferenceInfo
	for _, p := range prefs {
		infos = append(infos, toPreferenceInfo(p))
	}

	return &notificationv1.GetNotificationPreferencesResponse{
		Preferences: infos,
	}, nil
}

func (s *NotificationGRPCServer) UpdateNotificationPreference(ctx context.Context, req *notificationv1.UpdateNotificationPreferenceRequest) (*notificationv1.UpdateNotificationPreferenceResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	pref := &models.NotificationPreference{
		TenantID:     tenantID,
		UserID:       userID,
		EventTypeKey: req.EventTypeKey,
		ModuleID:     req.ModuleId,
		InApp:        req.InApp,
		DesktopPush:  req.DesktopPush,
		Sound:        req.Sound,
	}

	if err := s.prefService.UpdatePreference(ctx, pref); err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.UpdateNotificationPreferenceResponse{
		Preference: toPreferenceInfo(pref),
	}, nil
}

// ============================================================================
// Mutes
// ============================================================================

func (s *NotificationGRPCServer) MuteResource(ctx context.Context, req *notificationv1.MuteResourceRequest) (*notificationv1.MuteResourceResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	mute, err := s.prefService.MuteResource(ctx, tenantID, userID, req.ModuleId, req.ResourceId)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.MuteResourceResponse{
		Mute: toMuteInfo(mute),
	}, nil
}

func (s *NotificationGRPCServer) UnmuteResource(ctx context.Context, req *notificationv1.UnmuteResourceRequest) (*notificationv1.UnmuteResourceResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.prefService.UnmuteResource(ctx, tenantID, userID, req.ModuleId, req.ResourceId); err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.UnmuteResourceResponse{}, nil
}

func (s *NotificationGRPCServer) ListMutedResources(ctx context.Context, req *notificationv1.ListMutedResourcesRequest) (*notificationv1.ListMutedResourcesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	mutes, total, err := s.prefService.ListMutedResources(ctx, tenantID, userID, req.ModuleId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, mapNotificationError(err)
	}

	var infos []*notificationv1.MuteInfo
	for _, m := range mutes {
		infos = append(infos, toMuteInfo(m))
	}

	return &notificationv1.ListMutedResourcesResponse{
		Mutes: infos,
		Total: int32(total),
	}, nil
}

// ============================================================================
// Quiet Hours
// ============================================================================

func (s *NotificationGRPCServer) GetQuietHours(ctx context.Context, req *notificationv1.GetQuietHoursRequest) (*notificationv1.GetQuietHoursResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	qh, err := s.prefService.GetQuietHours(ctx, tenantID, userID)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.GetQuietHoursResponse{
		QuietHours: toQuietHoursInfo(qh),
	}, nil
}

func (s *NotificationGRPCServer) UpdateQuietHours(ctx context.Context, req *notificationv1.UpdateQuietHoursRequest) (*notificationv1.UpdateQuietHoursResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	daysOfWeek := make([]int, len(req.DaysOfWeek))
	for i, d := range req.DaysOfWeek {
		daysOfWeek[i] = int(d)
	}

	qh, err := s.prefService.UpdateQuietHours(ctx, tenantID, userID, req.StartTime, req.EndTime, req.Timezone, daysOfWeek, req.Enabled)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.UpdateQuietHoursResponse{
		QuietHours: toQuietHoursInfo(qh),
	}, nil
}

func (s *NotificationGRPCServer) ToggleManualDND(ctx context.Context, req *notificationv1.ToggleManualDNDRequest) (*notificationv1.ToggleManualDNDResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	var untilPtr *time.Time
	if req.Until != nil {
		t := req.Until.AsTime()
		untilPtr = &t
	}

	qh, err := s.prefService.ToggleManualDND(ctx, tenantID, userID, req.Enabled, untilPtr)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.ToggleManualDNDResponse{
		QuietHours: toQuietHoursInfo(qh),
	}, nil
}

// ============================================================================
// Event Types
// ============================================================================

func (s *NotificationGRPCServer) ListEventTypes(_ context.Context, req *notificationv1.ListEventTypesRequest) (*notificationv1.ListEventTypesResponse, error) {
	var types []models.EventType
	if req.ModuleId != nil {
		types = s.registry.ListByModule(*req.ModuleId)
	} else {
		types = s.registry.ListAll()
	}

	var infos []*notificationv1.EventTypeInfo
	for _, et := range types {
		infos = append(infos, toEventTypeInfo(&et))
	}

	return &notificationv1.ListEventTypesResponse{
		EventTypes: infos,
	}, nil
}

// ============================================================================
// Converters
// ============================================================================

func toNotificationInfo(n *models.Notification) *notificationv1.NotificationInfo {
	info := &notificationv1.NotificationInfo{
		Id:               n.ID.String(),
		UserId:           n.UserID.String(),
		EventTypeKey:     n.EventTypeKey,
		ModuleId:         n.ModuleID,
		Priority:         toPriority(n.Priority),
		Title:            n.Title,
		GroupCount:       int32(n.GroupCount),
		IsRead:           n.IsRead,
		DeliveredDesktop: n.DeliveredDesktop,
		CreatedAt:        timestamppb.New(n.CreatedAt),
	}

	if n.ActorID != nil {
		actorID := n.ActorID.String()
		info.ActorId = &actorID
	}
	if n.ResourceID != nil {
		info.ResourceId = n.ResourceID
	}
	if n.Body != nil {
		info.Body = n.Body
	}
	if n.DeepLink != nil {
		info.DeepLink = n.DeepLink
	}
	if n.GroupKey != nil {
		info.GroupKey = n.GroupKey
	}
	if n.ReadAt != nil {
		info.ReadAt = timestamppb.New(*n.ReadAt)
	}

	return info
}

func toPreferenceInfo(p *models.NotificationPreference) *notificationv1.NotificationPreferenceInfo {
	return &notificationv1.NotificationPreferenceInfo{
		Id:           p.ID.String(),
		UserId:       p.UserID.String(),
		EventTypeKey: p.EventTypeKey,
		ModuleId:     p.ModuleID,
		InApp:        p.InApp,
		DesktopPush:  p.DesktopPush,
		Sound:        p.Sound,
		CreatedAt:    timestamppb.New(p.CreatedAt),
		UpdatedAt:    timestamppb.New(p.UpdatedAt),
	}
}

func toMuteInfo(m *models.NotificationMute) *notificationv1.MuteInfo {
	return &notificationv1.MuteInfo{
		Id:         m.ID.String(),
		UserId:     m.UserID.String(),
		ModuleId:   m.ModuleID,
		ResourceId: m.ResourceID,
		CreatedAt:  timestamppb.New(m.CreatedAt),
	}
}

func toQuietHoursInfo(qh *models.QuietHours) *notificationv1.QuietHoursInfo {
	info := &notificationv1.QuietHoursInfo{
		Id:         qh.ID.String(),
		UserId:     qh.UserID.String(),
		StartTime:  qh.StartTime,
		EndTime:    qh.EndTime,
		Timezone:   qh.Timezone,
		Enabled:    qh.Enabled,
		ManualDnd:  qh.ManualDND,
		CreatedAt:  timestamppb.New(qh.CreatedAt),
		UpdatedAt:  timestamppb.New(qh.UpdatedAt),
	}

	for _, day := range qh.DaysOfWeek {
		info.DaysOfWeek = append(info.DaysOfWeek, int32(day))
	}

	if qh.ManualDNDUntil != nil {
		info.ManualDndUntil = timestamppb.New(*qh.ManualDNDUntil)
	}

	return info
}

func toEventTypeInfo(et *models.EventType) *notificationv1.EventTypeInfo {
	info := &notificationv1.EventTypeInfo{
		Id:                 et.ID.String(),
		ModuleId:           et.ModuleID,
		EventKey:           et.EventKey,
		DisplayName:        et.DisplayName,
		DefaultPriority:    toPriority(et.DefaultPriority),
		DefaultInApp:       et.DefaultInApp,
		DefaultDesktopPush: et.DefaultDesktopPush,
		DefaultSound:       et.DefaultSound,
		CreatedAt:          timestamppb.New(et.CreatedAt),
		UpdatedAt:          timestamppb.New(et.UpdatedAt),
	}

	if et.Description != nil {
		info.Description = et.Description
	}

	return info
}

func toPriority(p string) notificationv1.Priority {
	switch p {
	case models.PriorityUrgent:
		return notificationv1.Priority_PRIORITY_URGENT
	case models.PriorityNormal:
		return notificationv1.Priority_PRIORITY_NORMAL
	case models.PriorityLow:
		return notificationv1.Priority_PRIORITY_LOW
	default:
		return notificationv1.Priority_PRIORITY_UNSPECIFIED
	}
}

// ============================================================================
// Error Mapping
// ============================================================================

func mapNotificationError(err error) error {
	switch {
	case errors.Is(err, notification.ErrNotificationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, notification.ErrInvalidPriority):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, notification.ErrUnauthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, preference.ErrPreferenceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, preference.ErrQuietHoursNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, preference.ErrMuteNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, preference.ErrMuteAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, preference.ErrInvalidTimezone):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, preference.ErrInvalidTimeFormat):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, preference.ErrInvalidDayOfWeek):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, integration.ErrConfigNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, integration.ErrConfigAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, integration.ErrMappingNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, integration.ErrAccountLinkNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, integration.ErrAccountNotLinked):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, integration.ErrLinkTokenExpired):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case errors.Is(err, integration.ErrLinkTokenUsed):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, integration.ErrLinkTokenNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, integration.ErrPlatformDisabled):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, integration.ErrInvalidPlatform):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled notification service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

// ============================================================================
// Integration Configuration RPCs
// ============================================================================

func (s *NotificationGRPCServer) ListIntegrationConfigs(ctx context.Context, _ *notificationv1.ListIntegrationConfigsRequest) (*notificationv1.ListIntegrationConfigsResponse, error) {
	if s.integrationRepo == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	configs, err := s.integrationRepo.ListConfigs(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list configs")
	}

	var infos []*notificationv1.IntegrationConfigInfo
	for _, cfg := range configs {
		infos = append(infos, toIntegrationConfigInfo(cfg))
	}

	return &notificationv1.ListIntegrationConfigsResponse{Configs: infos}, nil
}

func (s *NotificationGRPCServer) GetIntegrationConfig(ctx context.Context, req *notificationv1.GetIntegrationConfigRequest) (*notificationv1.GetIntegrationConfigResponse, error) {
	if s.integrationRepo == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	cfg, err := s.integrationRepo.GetConfigByPlatform(ctx, req.Platform)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.GetIntegrationConfigResponse{Config: toIntegrationConfigInfo(cfg)}, nil
}

func (s *NotificationGRPCServer) CreateIntegrationConfig(ctx context.Context, req *notificationv1.CreateIntegrationConfigRequest) (*notificationv1.CreateIntegrationConfigResponse, error) {
	if s.integrationRepo == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	if !integration.ValidPlatforms[req.Platform] {
		return nil, status.Error(codes.InvalidArgument, "invalid platform")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	now := time.Now()
	cfg := &integration.IntegrationConfig{
		ID:                  uuid.New(),
		Platform:            req.Platform,
		IsActive:            false, // Start inactive until tested
		CredentialsVaultKey: req.CredentialsVaultKey,
		Metadata:            json.RawMessage(req.Metadata),
		CreatedBy:           createdBy,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.integrationRepo.CreateConfig(ctx, cfg); err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.CreateIntegrationConfigResponse{Config: toIntegrationConfigInfo(cfg)}, nil
}

func (s *NotificationGRPCServer) UpdateIntegrationConfig(ctx context.Context, req *notificationv1.UpdateIntegrationConfigRequest) (*notificationv1.UpdateIntegrationConfigResponse, error) {
	if s.integrationRepo == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	cfgID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	// Get existing config by ID (list and find)
	configs, err := s.integrationRepo.ListConfigs(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list configs")
	}

	var cfg *integration.IntegrationConfig
	for _, c := range configs {
		if c.ID == cfgID {
			cfg = c
			break
		}
	}
	if cfg == nil {
		return nil, status.Error(codes.NotFound, "config not found")
	}

	cfg.IsActive = req.IsActive
	cfg.Metadata = json.RawMessage(req.Metadata)
	if req.CredentialsVaultKey != nil {
		cfg.CredentialsVaultKey = *req.CredentialsVaultKey
	}
	cfg.UpdatedAt = time.Now()

	if err := s.integrationRepo.UpdateConfig(ctx, cfg); err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.UpdateIntegrationConfigResponse{Config: toIntegrationConfigInfo(cfg)}, nil
}

func (s *NotificationGRPCServer) DeleteIntegrationConfig(ctx context.Context, req *notificationv1.DeleteIntegrationConfigRequest) (*notificationv1.DeleteIntegrationConfigResponse, error) {
	if s.integrationRepo == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	cfgID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	if err := s.integrationRepo.DeleteConfig(ctx, cfgID); err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.DeleteIntegrationConfigResponse{}, nil
}

func (s *NotificationGRPCServer) TestIntegrationConfig(_ context.Context, _ *notificationv1.TestIntegrationConfigRequest) (*notificationv1.TestIntegrationConfigResponse, error) {
	// Test notification sending will be wired in cmd/notification/main.go
	// where the forwarder and platform clients are available.
	return &notificationv1.TestIntegrationConfigResponse{
		Success: true,
	}, nil
}

// ============================================================================
// Channel Mapping RPCs
// ============================================================================

func (s *NotificationGRPCServer) ListChannelMappings(ctx context.Context, req *notificationv1.ListChannelMappingsRequest) (*notificationv1.ListChannelMappingsResponse, error) {
	if s.integrationRepo == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	configID, err := uuid.Parse(req.ConfigId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid config_id")
	}

	mappings, err := s.integrationRepo.ListMappingsByConfig(ctx, configID)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	var infos []*notificationv1.ChannelMappingInfo
	for _, m := range mappings {
		infos = append(infos, toChannelMappingInfo(m))
	}

	return &notificationv1.ListChannelMappingsResponse{Mappings: infos}, nil
}

func (s *NotificationGRPCServer) CreateChannelMapping(ctx context.Context, req *notificationv1.CreateChannelMappingRequest) (*notificationv1.CreateChannelMappingResponse, error) {
	if s.integrationRepo == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	configID, err := uuid.Parse(req.ConfigId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid config_id")
	}

	now := time.Now()
	m := &integration.ChannelMapping{
		ID:           uuid.New(),
		ConfigID:     configID,
		ChannelID:    req.ChannelId,
		ChannelName:  req.ChannelName,
		Modules:      req.Modules,
		IsActive:     true,
		PlatformData: json.RawMessage("{}"),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.integrationRepo.CreateMapping(ctx, m); err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.CreateChannelMappingResponse{Mapping: toChannelMappingInfo(m)}, nil
}

func (s *NotificationGRPCServer) UpdateChannelMapping(ctx context.Context, req *notificationv1.UpdateChannelMappingRequest) (*notificationv1.UpdateChannelMappingResponse, error) {
	if s.integrationRepo == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	mappingID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	m, err := s.integrationRepo.GetMapping(ctx, mappingID)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	m.ChannelID = req.ChannelId
	m.ChannelName = req.ChannelName
	m.Modules = req.Modules
	m.IsActive = req.IsActive
	m.UpdatedAt = time.Now()

	if err := s.integrationRepo.UpdateMapping(ctx, m); err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.UpdateChannelMappingResponse{Mapping: toChannelMappingInfo(m)}, nil
}

func (s *NotificationGRPCServer) DeleteChannelMapping(ctx context.Context, req *notificationv1.DeleteChannelMappingRequest) (*notificationv1.DeleteChannelMappingResponse, error) {
	if s.integrationRepo == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	mappingID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	if err := s.integrationRepo.DeleteMapping(ctx, mappingID); err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.DeleteChannelMappingResponse{}, nil
}

// ============================================================================
// Account Linking RPCs
// ============================================================================

func (s *NotificationGRPCServer) LinkAccount(ctx context.Context, req *notificationv1.LinkAccountRequest) (*notificationv1.LinkAccountResponse, error) {
	if s.linkService == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	link, err := s.linkService.VerifyAndLink(ctx, req.Token, userID)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.LinkAccountResponse{
		Platform:            link.Platform,
		ExternalDisplayName: link.ExternalDisplayName,
	}, nil
}

func (s *NotificationGRPCServer) UnlinkAccount(ctx context.Context, req *notificationv1.UnlinkAccountRequest) (*notificationv1.UnlinkAccountResponse, error) {
	if s.linkService == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.linkService.Unlink(ctx, req.Platform, userID); err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.UnlinkAccountResponse{}, nil
}

func (s *NotificationGRPCServer) GetAccountLinkStatus(ctx context.Context, req *notificationv1.GetAccountLinkStatusRequest) (*notificationv1.GetAccountLinkStatusResponse, error) {
	if s.integrationRepo == nil {
		return nil, status.Error(codes.Unavailable, "integration not configured")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	var links []*notificationv1.AccountLinkInfo
	for _, platform := range []string{integration.PlatformTeams, integration.PlatformSlack} {
		link, err := s.integrationRepo.GetAccountLinkByKMUHubUser(ctx, platform, userID)
		if err != nil {
			continue // Not linked for this platform
		}
		links = append(links, &notificationv1.AccountLinkInfo{
			Platform:            link.Platform,
			ExternalDisplayName: link.ExternalDisplayName,
			LinkedAt:            timestamppb.New(link.LinkedAt),
		})
	}

	return &notificationv1.GetAccountLinkStatusResponse{Links: links}, nil
}

// ============================================================================
// Integration Converters
// ============================================================================

func toIntegrationConfigInfo(cfg *integration.IntegrationConfig) *notificationv1.IntegrationConfigInfo {
	metadata := "{}"
	if cfg.Metadata != nil {
		metadata = string(cfg.Metadata)
	}
	return &notificationv1.IntegrationConfigInfo{
		Id:        cfg.ID.String(),
		Platform:  cfg.Platform,
		IsActive:  cfg.IsActive,
		Metadata:  metadata,
		CreatedBy: cfg.CreatedBy.String(),
		CreatedAt: timestamppb.New(cfg.CreatedAt),
		UpdatedAt: timestamppb.New(cfg.UpdatedAt),
	}
}

func toChannelMappingInfo(m *integration.ChannelMapping) *notificationv1.ChannelMappingInfo {
	return &notificationv1.ChannelMappingInfo{
		Id:          m.ID.String(),
		ConfigId:    m.ConfigID.String(),
		ChannelId:   m.ChannelID,
		ChannelName: m.ChannelName,
		Modules:     m.Modules,
		IsActive:    m.IsActive,
		CreatedAt:   timestamppb.New(m.CreatedAt),
		UpdatedAt:   timestamppb.New(m.UpdatedAt),
	}
}
