package server

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/notification/notification"
	"github.com/kmuhub/kmuhub/internal/notification/preference"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

// NotificationGRPCServer implements the Notification gRPC service.
type NotificationGRPCServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	notifService *notification.Service
	prefService  *preference.Service
	registry     *event.EventTypeRegistry
}

// NewNotificationGRPCServer creates a new Notification gRPC server.
func NewNotificationGRPCServer(
	notifService *notification.Service,
	prefService *preference.Service,
	registry *event.EventTypeRegistry,
) *NotificationGRPCServer {
	return &NotificationGRPCServer{
		notifService: notifService,
		prefService:  prefService,
		registry:     registry,
	}
}

// ============================================================================
// Notifications
// ============================================================================

func (s *NotificationGRPCServer) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
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

	notifications, total, err := s.notifService.ListNotifications(ctx, userID, moduleID, isRead, int(req.Page), int(req.PageSize))
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
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	count, err := s.notifService.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.GetUnreadCountResponse{
		Count: int32(count),
	}, nil
}

func (s *NotificationGRPCServer) MarkNotificationRead(ctx context.Context, req *notificationv1.MarkNotificationReadRequest) (*notificationv1.MarkNotificationReadResponse, error) {
	notifID, err := uuid.Parse(req.NotificationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid notification_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	notif, err := s.notifService.MarkRead(ctx, notifID, userID)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.MarkNotificationReadResponse{
		Notification: toNotificationInfo(notif),
	}, nil
}

func (s *NotificationGRPCServer) MarkAllNotificationsRead(ctx context.Context, req *notificationv1.MarkAllNotificationsReadRequest) (*notificationv1.MarkAllNotificationsReadResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	count, err := s.notifService.MarkAllRead(ctx, userID, req.ModuleId)
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
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	prefs, err := s.prefService.GetPreferences(ctx, userID, req.ModuleId)
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
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	pref := &models.NotificationPreference{
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
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	mute, err := s.prefService.MuteResource(ctx, userID, req.ModuleId, req.ResourceId)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.MuteResourceResponse{
		Mute: toMuteInfo(mute),
	}, nil
}

func (s *NotificationGRPCServer) UnmuteResource(ctx context.Context, req *notificationv1.UnmuteResourceRequest) (*notificationv1.UnmuteResourceResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.prefService.UnmuteResource(ctx, userID, req.ModuleId, req.ResourceId); err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.UnmuteResourceResponse{}, nil
}

func (s *NotificationGRPCServer) ListMutedResources(ctx context.Context, req *notificationv1.ListMutedResourcesRequest) (*notificationv1.ListMutedResourcesResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	mutes, total, err := s.prefService.ListMutedResources(ctx, userID, req.ModuleId, int(req.Page), int(req.PageSize))
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
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	qh, err := s.prefService.GetQuietHours(ctx, userID)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.GetQuietHoursResponse{
		QuietHours: toQuietHoursInfo(qh),
	}, nil
}

func (s *NotificationGRPCServer) UpdateQuietHours(ctx context.Context, req *notificationv1.UpdateQuietHoursRequest) (*notificationv1.UpdateQuietHoursResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	daysOfWeek := make([]int, len(req.DaysOfWeek))
	for i, d := range req.DaysOfWeek {
		daysOfWeek[i] = int(d)
	}

	qh, err := s.prefService.UpdateQuietHours(ctx, userID, req.StartTime, req.EndTime, req.Timezone, daysOfWeek, req.Enabled)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	return &notificationv1.UpdateQuietHoursResponse{
		QuietHours: toQuietHoursInfo(qh),
	}, nil
}

func (s *NotificationGRPCServer) ToggleManualDND(ctx context.Context, req *notificationv1.ToggleManualDNDRequest) (*notificationv1.ToggleManualDNDResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	var untilPtr *time.Time
	if req.Until != nil {
		t := req.Until.AsTime()
		untilPtr = &t
	}

	qh, err := s.prefService.ToggleManualDND(ctx, userID, req.Enabled, untilPtr)
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
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
