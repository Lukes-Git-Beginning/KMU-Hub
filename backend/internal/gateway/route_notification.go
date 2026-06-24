package gateway

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

// NotificationRoutes handles HTTP routes for the notification backend service.
type NotificationRoutes struct {
	registry *ServiceRegistry
}

// NewNotificationRoutes creates a new NotificationRoutes with the given service registry.
func NewNotificationRoutes(registry *ServiceRegistry) *NotificationRoutes {
	return &NotificationRoutes{registry: registry}
}

// ServiceName returns the backend service name.
func (n *NotificationRoutes) ServiceName() string { return "notification" }

// getNotificationClient lazily obtains a gRPC client for the notification service.
func (n *NotificationRoutes) getNotificationClient() (notificationv1.NotificationServiceClient, error) {
	conn, err := n.registry.GetConnection("notification")
	if err != nil {
		return nil, err
	}
	return notificationv1.NewNotificationServiceClient(conn), nil
}

// RegisterRoutes registers all notification HTTP routes.
func (n *NotificationRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/notifications", func(r chi.Router) {
		r.Use(authMiddleware)

		// Notification CRUD -- specific paths before parameterized paths
		r.With(middleware.RequirePermission("notifications", "write")).Post("/read-all", n.HandleMarkAllRead)
		r.With(middleware.RequirePermission("notifications", "read")).Get("/unread-count", n.HandleGetUnreadCount)
		r.With(middleware.RequirePermission("notifications", "read")).Get("/", n.HandleListNotifications)
		r.With(middleware.RequirePermission("notifications", "write")).Post("/{id}/read", n.HandleMarkRead)
		r.With(middleware.RequirePermission("notifications", "write")).Post("/{id}/pin", n.HandlePin)
		r.With(middleware.RequirePermission("notifications", "write")).Delete("/{id}/pin", n.HandleUnpin)
		r.With(middleware.RequirePermission("notifications", "write")).Post("/{id}/dismiss", n.HandleDismiss)

		// Preferences
		r.With(middleware.RequirePermission("notifications", "read")).Get("/preferences", n.HandleGetPreferences)
		r.With(middleware.RequirePermission("notifications", "write")).Put("/preferences", n.HandleUpdatePreference)

		// Event Types
		r.With(middleware.RequirePermission("notifications", "read")).Get("/event-types", n.HandleListEventTypes)

		// Mutes
		r.With(middleware.RequirePermission("notifications", "write")).Post("/mutes", n.HandleMuteResource)
		r.With(middleware.RequirePermission("notifications", "write")).Delete("/mutes", n.HandleUnmuteResource)
		r.With(middleware.RequirePermission("notifications", "read")).Get("/mutes", n.HandleListMutedResources)

		// Quiet Hours
		r.With(middleware.RequirePermission("notifications", "read")).Get("/quiet-hours", n.HandleGetQuietHours)
		r.With(middleware.RequirePermission("notifications", "write")).Put("/quiet-hours", n.HandleUpdateQuietHours)

		// Do Not Disturb -- GET reads the current manual-DND status, POST enables it,
		// DELETE disables it. All three return the flat DNDStatus shape {is_active, expires_at}.
		r.With(middleware.RequirePermission("notifications", "read")).Get("/dnd", n.HandleGetDND)
		r.With(middleware.RequirePermission("notifications", "write")).Post("/dnd", n.HandleToggleDND)
		r.With(middleware.RequirePermission("notifications", "write")).Delete("/dnd", n.HandleDisableDND)
	})
}

// ============================================================================
// Notification Handlers
// ============================================================================

func (n *NotificationRoutes) HandleListNotifications(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	page, pageSize := parsePagination(r, 1, 20)
	moduleID := r.URL.Query().Get("module_id")
	isReadStr := r.URL.Query().Get("is_read")

	grpcReq := &notificationv1.ListNotificationsRequest{
		UserId:   userID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}

	if moduleID != "" {
		grpcReq.ModuleId = &moduleID
	}
	if isReadStr != "" {
		isRead := isReadStr == "true"
		grpcReq.IsRead = &isRead
	}

	resp, err := client.ListNotifications(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (n *NotificationRoutes) HandleGetUnreadCount(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetUnreadCount(r.Context(), &notificationv1.GetUnreadCountRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (n *NotificationRoutes) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	notificationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.MarkNotificationRead(r.Context(), &notificationv1.MarkNotificationReadRequest{
		NotificationId: notificationID,
		UserId:         userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (n *NotificationRoutes) HandlePin(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	notificationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.PinNotification(r.Context(), &notificationv1.PinNotificationRequest{
		NotificationId: notificationID,
		UserId:         userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (n *NotificationRoutes) HandleUnpin(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	notificationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.UnpinNotification(r.Context(), &notificationv1.UnpinNotificationRequest{
		NotificationId: notificationID,
		UserId:         userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (n *NotificationRoutes) HandleDismiss(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	notificationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.DismissNotification(r.Context(), &notificationv1.DismissNotificationRequest{
		NotificationId: notificationID,
		UserId:         userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (n *NotificationRoutes) HandleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	type markAllReadBody struct {
		ModuleID *string `json:"module_id,omitempty"`
	}

	var body markAllReadBody
	// Body is optional for mark-all-read
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	grpcReq := &notificationv1.MarkAllNotificationsReadRequest{
		UserId: userID,
	}
	if body.ModuleID != nil {
		grpcReq.ModuleId = body.ModuleID
	}

	resp, err := client.MarkAllNotificationsRead(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Preference Handlers
// ============================================================================

func (n *NotificationRoutes) HandleGetPreferences(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	moduleID := r.URL.Query().Get("module_id")

	grpcReq := &notificationv1.GetNotificationPreferencesRequest{
		UserId: userID,
	}
	if moduleID != "" {
		grpcReq.ModuleId = &moduleID
	}

	resp, err := client.GetNotificationPreferences(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updatePreferenceRequest struct {
	EventTypeKey *string `json:"event_type_key,omitempty"`
	ModuleID     *string `json:"module_id,omitempty"`
	InApp        bool    `json:"in_app"`
	DesktopPush  bool    `json:"desktop_push"`
	Sound        string  `json:"sound"`
}

func (n *NotificationRoutes) HandleUpdatePreference(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[updatePreferenceRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &notificationv1.UpdateNotificationPreferenceRequest{
		UserId:      userID,
		InApp:       req.InApp,
		DesktopPush: req.DesktopPush,
		Sound:       req.Sound,
	}
	if req.EventTypeKey != nil {
		grpcReq.EventTypeKey = req.EventTypeKey
	}
	if req.ModuleID != nil {
		grpcReq.ModuleId = req.ModuleID
	}

	resp, err := client.UpdateNotificationPreference(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Event Types Handler
// ============================================================================

func (n *NotificationRoutes) HandleListEventTypes(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	moduleID := r.URL.Query().Get("module_id")

	grpcReq := &notificationv1.ListEventTypesRequest{}
	if moduleID != "" {
		grpcReq.ModuleId = &moduleID
	}

	resp, err := client.ListEventTypes(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Mute Handlers
// ============================================================================

type muteResourceRequest struct {
	ModuleID   string `json:"module_id"   validate:"required"`
	ResourceID string `json:"resource_id" validate:"required"`
}

func (n *NotificationRoutes) HandleMuteResource(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[muteResourceRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.MuteResource(r.Context(), &notificationv1.MuteResourceRequest{
		UserId:     userID,
		ModuleId:   req.ModuleID,
		ResourceId: req.ResourceID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

type unmuteResourceRequest struct {
	ModuleID   string `json:"module_id"   validate:"required"`
	ResourceID string `json:"resource_id" validate:"required"`
}

func (n *NotificationRoutes) HandleUnmuteResource(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[unmuteResourceRequest](w, r)
	if !ok {
		return
	}

	_, err = client.UnmuteResource(r.Context(), &notificationv1.UnmuteResourceRequest{
		UserId:     userID,
		ModuleId:   req.ModuleID,
		ResourceId: req.ResourceID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "resource unmuted"})
}

func (n *NotificationRoutes) HandleListMutedResources(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	moduleID := r.URL.Query().Get("module_id")
	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &notificationv1.ListMutedResourcesRequest{
		UserId:   userID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if moduleID != "" {
		grpcReq.ModuleId = &moduleID
	}

	resp, err := client.ListMutedResources(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Quiet Hours Handlers
// ============================================================================

func (n *NotificationRoutes) HandleGetQuietHours(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetQuietHours(r.Context(), &notificationv1.GetQuietHoursRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateQuietHoursRequest struct {
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Timezone   string `json:"timezone"`
	DaysOfWeek []int  `json:"days_of_week"`
	Enabled    bool   `json:"enabled"`
}

func (n *NotificationRoutes) HandleUpdateQuietHours(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[updateQuietHoursRequest](w, r)
	if !ok {
		return
	}

	daysOfWeek := make([]int32, len(req.DaysOfWeek))
	for i, d := range req.DaysOfWeek {
		daysOfWeek[i] = int32(d)
	}

	resp, err := client.UpdateQuietHours(r.Context(), &notificationv1.UpdateQuietHoursRequest{
		UserId:     userID,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Timezone:   req.Timezone,
		DaysOfWeek: daysOfWeek,
		Enabled:    req.Enabled,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type toggleDNDRequest struct {
	Enabled bool   `json:"enabled"`
	Until   string `json:"until,omitempty"`
}

func (n *NotificationRoutes) HandleToggleDND(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[toggleDNDRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &notificationv1.ToggleManualDNDRequest{
		UserId:  userID,
		Enabled: req.Enabled,
	}

	if req.Until != "" {
		t, parseErr := time.Parse(time.RFC3339, req.Until)
		if parseErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid 'until' timestamp, expected RFC3339 format")
			return
		}
		grpcReq.Until = timestamppb.New(t)
	}

	resp, err := client.ToggleManualDND(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dndStatusFromQuietHours(resp.GetQuietHours()))
}

// dndStatusFromQuietHours maps the QuietHours backing record to the flat DNDStatus
// shape the desktop client expects: {is_active, expires_at?}.
func dndStatusFromQuietHours(qh *notificationv1.QuietHoursInfo) map[string]any {
	out := map[string]any{"is_active": false}
	if qh == nil {
		return out
	}
	out["is_active"] = qh.GetManualDnd()
	if until := qh.GetManualDndUntil(); until != nil {
		out["expires_at"] = until.AsTime().Format(time.RFC3339)
	}
	return out
}

// HandleGetDND returns the current manual Do-Not-Disturb status, derived from the
// user's quiet-hours record. Returns a disabled default when no record exists.
func (n *NotificationRoutes) HandleGetDND(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetQuietHours(r.Context(), &notificationv1.GetQuietHoursRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dndStatusFromQuietHours(resp.GetQuietHours()))
}

// HandleDisableDND turns off manual Do-Not-Disturb mode.
func (n *NotificationRoutes) HandleDisableDND(w http.ResponseWriter, r *http.Request) {
	client, err := n.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, n.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.ToggleManualDND(r.Context(), &notificationv1.ToggleManualDNDRequest{
		UserId:  userID,
		Enabled: false,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dndStatusFromQuietHours(resp.GetQuietHours()))
}
