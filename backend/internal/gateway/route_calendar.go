package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// CalendarRoutes handles HTTP routes for the Calendar service.
// Calendar runs in the same binary as Work, so we reuse the "work" gRPC connection.
type CalendarRoutes struct {
	registry *ServiceRegistry
}

// NewCalendarRoutes creates a new CalendarRoutes with the given service registry.
func NewCalendarRoutes(registry *ServiceRegistry) *CalendarRoutes {
	return &CalendarRoutes{registry: registry}
}

// ServiceName returns the backend service name (shares "work" gRPC port).
func (cr *CalendarRoutes) ServiceName() string { return "work" }

// getCalendarClient lazily obtains a gRPC client for the Calendar service.
func (cr *CalendarRoutes) getCalendarClient() (calv1.CalendarServiceClient, error) {
	conn, err := cr.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return calv1.NewCalendarServiceClient(conn), nil
}

// RegisterRoutes registers all Calendar HTTP routes.
func (cr *CalendarRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/calendar", func(r chi.Router) {
		r.Use(authMiddleware)

		// Calendars
		r.With(middleware.RequirePermission("calendars", "read")).Get("/calendars", cr.HandleListCalendars)
		r.With(middleware.RequirePermission("calendars", "write")).Post("/calendars", cr.HandleCreateCalendar)
		r.With(middleware.RequirePermission("calendars", "read")).Get("/calendars/{id}", cr.HandleGetCalendar)
		r.With(middleware.RequirePermission("calendars", "write")).Put("/calendars/{id}", cr.HandleUpdateCalendar)
		r.With(middleware.RequirePermission("calendars", "delete")).Delete("/calendars/{id}", cr.HandleDeleteCalendar)

		// Calendar Members
		r.With(middleware.RequirePermission("calendars", "read")).Get("/calendars/{id}/members", cr.HandleListCalendarMembers)
		r.With(middleware.RequirePermission("calendars", "admin")).Post("/calendars/{id}/members", cr.HandleAddCalendarMember)
		r.With(middleware.RequirePermission("calendars", "admin")).Delete("/calendars/{id}/members/{userId}", cr.HandleRemoveCalendarMember)
		r.With(middleware.RequirePermission("calendars", "admin")).Put("/calendars/{id}/members/{userId}/permission", cr.HandleUpdateCalendarMemberPermission)

		// Calendar Discovery & Subscription
		r.With(middleware.RequirePermission("calendars", "read")).Get("/browse", cr.HandleListBrowsableCalendars)
		r.With(middleware.RequirePermission("calendars", "write")).Post("/calendars/{id}/subscribe", cr.HandleSubscribeToCalendar)
		r.With(middleware.RequirePermission("calendars", "write")).Delete("/calendars/{id}/subscribe", cr.HandleUnsubscribeFromCalendar)

		// Events
		r.With(middleware.RequirePermission("calendars", "read")).Get("/events", cr.HandleListEventsInRange)
		r.With(middleware.RequirePermission("calendars", "write")).Post("/events", cr.HandleCreateEvent)
		r.With(middleware.RequirePermission("calendars", "read")).Get("/events/{id}", cr.HandleGetEvent)
		r.With(middleware.RequirePermission("calendars", "write")).Put("/events/{id}", cr.HandleUpdateEvent)
		r.With(middleware.RequirePermission("calendars", "delete")).Delete("/events/{id}", cr.HandleDeleteEvent)
		r.With(middleware.RequirePermission("calendars", "write")).Put("/events/{id}/recurring", cr.HandleUpdateRecurringEvent)
		r.With(middleware.RequirePermission("calendars", "write")).Post("/events/{id}/rsvp", cr.HandleRSVPToEvent)
		r.With(middleware.RequirePermission("calendars", "read")).Get("/events/{id}/attendees", cr.HandleListEventAttendees)

		// Reminders
		r.With(middleware.RequirePermission("calendars", "write")).Put("/events/{id}/reminders", cr.HandleSetEventReminders)
		r.With(middleware.RequirePermission("calendars", "read")).Get("/events/{id}/reminders", cr.HandleListEventReminders)

		// Event Categories
		r.With(middleware.RequirePermission("calendars", "read")).Get("/categories", cr.HandleListEventCategories)
		r.With(middleware.RequirePermission("calendars", "write")).Post("/categories", cr.HandleCreateEventCategory)
		r.With(middleware.RequirePermission("calendars", "delete")).Delete("/categories/{id}", cr.HandleDeleteEventCategory)

		// Resources
		r.With(middleware.RequirePermission("resources", "read")).Get("/resources", cr.HandleListResources)
		r.With(middleware.RequirePermission("resources", "write")).Post("/resources", cr.HandleCreateResource)
		r.With(middleware.RequirePermission("resources", "read")).Get("/resources/{id}", cr.HandleGetResource)
		r.With(middleware.RequirePermission("resources", "write")).Put("/resources/{id}", cr.HandleUpdateResource)
		r.With(middleware.RequirePermission("resources", "delete")).Delete("/resources/{id}", cr.HandleDeleteResource)
		r.With(middleware.RequirePermission("resources", "read")).Get("/resources/{id}/availability", cr.HandleListResourceAvailability)

		// Resource Bookings
		r.With(middleware.RequirePermission("resources", "read")).Post("/bookings", cr.HandleBookResource)
		r.With(middleware.RequirePermission("resources", "read")).Delete("/bookings/{id}", cr.HandleCancelBooking)

		// Holidays
		r.With(middleware.RequirePermission("calendars", "read")).Get("/holidays", cr.HandleListHolidays)
		r.With(middleware.RequirePermission("calendars", "admin")).Post("/holidays/seed", cr.HandleSeedHolidays)

		// Preferences
		r.With(middleware.RequirePermission("calendars", "read")).Get("/preferences", cr.HandleGetCalendarPreferences)
		r.With(middleware.RequirePermission("calendars", "write")).Put("/preferences", cr.HandleUpdateCalendarPreferences)

		// Task Deadlines
		r.With(middleware.RequirePermission("calendars", "read")).Get("/task-deadlines", cr.HandleListTaskDeadlinesInRange)

		// LiveKit
		r.With(middleware.RequirePermission("calendars", "write")).Post("/events/{id}/join-token", cr.HandleGenerateJoinToken)
	})
}

// ============================================================================
// Request Types
// ============================================================================

type createCalendarRequest struct {
	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
	CalendarType string  `json:"calendar_type"`
	Color        string  `json:"color"`
	Timezone     string  `json:"timezone"`
}

type updateCalendarRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
	Timezone    *string `json:"timezone,omitempty"`
}

type addCalendarMemberRequest struct {
	UserID     string `json:"user_id"`
	Permission string `json:"permission"`
}

type updateCalendarMemberPermissionRequest struct {
	Permission string `json:"permission"`
}

type createEventRequest struct {
	CalendarID   string   `json:"calendar_id"`
	Title        string   `json:"title"`
	Description  *string  `json:"description,omitempty"`
	Location     *string  `json:"location,omitempty"`
	ResourceID   *string  `json:"resource_id,omitempty"`
	StartTime    string   `json:"start_time"`
	EndTime      string   `json:"end_time"`
	IsAllDay     bool     `json:"is_all_day"`
	Timezone     string   `json:"timezone"`
	RRule        *string  `json:"rrule,omitempty"`
	HasVideoCall bool     `json:"has_video_call"`
	CategoryID   *string  `json:"category_id,omitempty"`
	AttendeeIDs  []string `json:"attendee_ids,omitempty"`
}

type updateEventRequest struct {
	Title        *string `json:"title,omitempty"`
	Description  *string `json:"description,omitempty"`
	Location     *string `json:"location,omitempty"`
	ResourceID   *string `json:"resource_id,omitempty"`
	StartTime    *string `json:"start_time,omitempty"`
	EndTime      *string `json:"end_time,omitempty"`
	IsAllDay     *bool   `json:"is_all_day,omitempty"`
	CategoryID   *string `json:"category_id,omitempty"`
	HasVideoCall *bool   `json:"has_video_call,omitempty"`
}

type updateRecurringEventRequest struct {
	Scope        string             `json:"scope"` // "this", "this_and_future", "all"
	OriginalDate string             `json:"original_date"`
	Changes      updateEventRequest `json:"changes"`
}

type rsvpRequest struct {
	Status string `json:"status"`
}

type setRemindersRequest struct {
	MinutesBefore []int32 `json:"minutes_before"`
}

type createResourceRequest struct {
	Name         string   `json:"name"`
	ResourceType string   `json:"resource_type"`
	Capacity     *int32   `json:"capacity,omitempty"`
	Floor        *string  `json:"floor,omitempty"`
	Location     *string  `json:"location,omitempty"`
	Description  *string  `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

type updateResourceRequest struct {
	Name         *string `json:"name,omitempty"`
	ResourceType *string `json:"resource_type,omitempty"`
	Capacity     *int32  `json:"capacity,omitempty"`
	Floor        *string `json:"floor,omitempty"`
	Location     *string `json:"location,omitempty"`
	Description  *string `json:"description,omitempty"`
}

type bookResourceRequest struct {
	ResourceID string `json:"resource_id"`
	EventID    string `json:"event_id"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
}

type seedHolidaysRequest struct {
	Year        int32  `json:"year"`
	CountryCode string `json:"country_code"`
}

type createEventCategoryRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type updateCalPreferencesRequest struct {
	DefaultView                  *string `json:"default_view,omitempty"`
	WeekDays                     *int32  `json:"week_days,omitempty"`
	DefaultReminderMinutes       *int32  `json:"default_reminder_minutes,omitempty"`
	DefaultAllDayReminderMinutes *int32  `json:"default_all_day_reminder_minutes,omitempty"`
	SubdivisionCode              *string `json:"subdivision_code,omitempty"`
	ShowTaskDeadlines            *bool   `json:"show_task_deadlines,omitempty"`
}

type generateJoinTokenRequest struct {
	DisplayName string `json:"display_name"`
}

// ============================================================================
// Calendar Handlers
// ============================================================================

func (cr *CalendarRoutes) HandleCreateCalendar(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(wr, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(wr, http.StatusBadRequest, "name is required")
		return
	}

	grpcReq := &calv1.CreateCalendarRequest{
		OwnerId:      userID,
		Name:         req.Name,
		Description:  req.Description,
		CalendarType: req.CalendarType,
	}
	if req.Color != "" {
		grpcReq.Color = &req.Color
	}
	if req.Timezone != "" {
		grpcReq.Timezone = &req.Timezone
	}

	resp, err := client.CreateCalendar(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (cr *CalendarRoutes) HandleGetCalendar(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	calendarID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(calendarID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid calendar id")
		return
	}

	resp, err := client.GetCalendar(r.Context(), &calv1.GetCalendarRequest{Id: calendarID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleListCalendars(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	includeHidden := r.URL.Query().Get("include_hidden") == "true"

	resp, err := client.ListCalendars(r.Context(), &calv1.ListCalendarsRequest{
		UserId:        userID,
		IncludeHidden: includeHidden,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleUpdateCalendar(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	calendarID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(calendarID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid calendar id")
		return
	}

	var req updateCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &calv1.UpdateCalendarRequest{Id: calendarID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}
	if req.Color != nil {
		grpcReq.Color = req.Color
	}
	if req.Timezone != nil {
		grpcReq.Timezone = req.Timezone
	}

	resp, err := client.UpdateCalendar(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleDeleteCalendar(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	calendarID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(calendarID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid calendar id")
		return
	}

	_, err = client.DeleteCalendar(r.Context(), &calv1.DeleteCalendarRequest{Id: calendarID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "calendar deleted"})
}

// ============================================================================
// Calendar Member Handlers
// ============================================================================

func (cr *CalendarRoutes) HandleListCalendarMembers(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	calendarID := chi.URLParam(r, "id")

	resp, err := client.ListCalendarMembers(r.Context(), &calv1.ListCalendarMembersRequest{
		CalendarId: calendarID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleAddCalendarMember(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	calendarID := chi.URLParam(r, "id")

	var req addCalendarMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.AddCalendarMember(r.Context(), &calv1.AddCalendarMemberRequest{
		CalendarId: calendarID,
		UserId:     req.UserID,
		Permission: req.Permission,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (cr *CalendarRoutes) HandleRemoveCalendarMember(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	calendarID := chi.URLParam(r, "id")
	memberUserID := chi.URLParam(r, "userId")

	_, err = client.RemoveCalendarMember(r.Context(), &calv1.RemoveCalendarMemberRequest{
		CalendarId: calendarID,
		UserId:     memberUserID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "member removed"})
}

func (cr *CalendarRoutes) HandleUpdateCalendarMemberPermission(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	calendarID := chi.URLParam(r, "id")
	memberUserID := chi.URLParam(r, "userId")

	var req updateCalendarMemberPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateCalendarMemberPermission(r.Context(), &calv1.UpdateCalendarMemberPermissionRequest{
		CalendarId: calendarID,
		UserId:     memberUserID,
		Permission: req.Permission,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Calendar Discovery & Subscription Handlers
// ============================================================================

func (cr *CalendarRoutes) HandleListBrowsableCalendars(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	grpcReq := &calv1.ListBrowsableCalendarsRequest{
		UserId: userID,
	}
	if search := r.URL.Query().Get("search"); search != "" {
		grpcReq.Search = &search
	}

	resp, err := client.ListBrowsableCalendars(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleSubscribeToCalendar(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	calendarID := chi.URLParam(r, "id")

	resp, err := client.SubscribeToCalendar(r.Context(), &calv1.SubscribeToCalendarRequest{
		CalendarId: calendarID,
		UserId:     userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleUnsubscribeFromCalendar(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	calendarID := chi.URLParam(r, "id")

	_, err = client.UnsubscribeFromCalendar(r.Context(), &calv1.UnsubscribeFromCalendarRequest{
		CalendarId: calendarID,
		UserId:     userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "unsubscribed"})
}

// ============================================================================
// Event Handlers
// ============================================================================

func (cr *CalendarRoutes) HandleListEventsInRange(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	q := r.URL.Query()

	calendarIDs := q["calendar_ids"]
	startStr := q.Get("start")
	endStr := q.Get("end")

	if startStr == "" || endStr == "" {
		response.Error(wr, http.StatusBadRequest, "start and end query parameters are required")
		return
	}

	start, err := parseTimestamp(startStr)
	if err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid start time format")
		return
	}
	end, err := parseTimestamp(endStr)
	if err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid end time format")
		return
	}

	resp, err := client.ListEventsInRange(r.Context(), &calv1.ListEventsInRangeRequest{
		CalendarIds: calendarIDs,
		Start:       start,
		End:         end,
		UserId:      userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleCreateEvent(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(wr, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" || req.CalendarID == "" {
		response.Error(wr, http.StatusBadRequest, "title and calendar_id are required")
		return
	}

	startTime, err := parseTimestamp(req.StartTime)
	if err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid start_time format")
		return
	}
	endTime, err := parseTimestamp(req.EndTime)
	if err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid end_time format")
		return
	}

	grpcReq := &calv1.CreateEventRequest{
		CalendarId:      req.CalendarID,
		Title:           req.Title,
		Description:     req.Description,
		Location:        req.Location,
		ResourceId:      req.ResourceID,
		StartTime:       startTime,
		EndTime:         endTime,
		IsAllDay:        req.IsAllDay,
		Rrule:           req.RRule,
		HasVideoCall:    req.HasVideoCall,
		CategoryId:      req.CategoryID,
		CreatedBy:       userID,
		AttendeeUserIds: req.AttendeeIDs,
	}
	if req.Timezone != "" {
		grpcReq.Timezone = &req.Timezone
	}

	resp, err := client.CreateEvent(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (cr *CalendarRoutes) HandleGetEvent(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	eventID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(eventID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid event id")
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetEvent(r.Context(), &calv1.GetEventRequest{
		Id:     eventID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleUpdateEvent(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	eventID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(eventID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid event id")
		return
	}

	var req updateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &calv1.UpdateEventRequest{
		Id:           eventID,
		Title:        req.Title,
		Description:  req.Description,
		Location:     req.Location,
		ResourceId:   req.ResourceID,
		IsAllDay:     req.IsAllDay,
		CategoryId:   req.CategoryID,
		HasVideoCall: req.HasVideoCall,
		UpdatedBy:    userID,
	}
	if req.StartTime != nil {
		t, parseErr := parseTimestamp(*req.StartTime)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid start_time format")
			return
		}
		grpcReq.StartTime = t
	}
	if req.EndTime != nil {
		t, parseErr := parseTimestamp(*req.EndTime)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid end_time format")
			return
		}
		grpcReq.EndTime = t
	}

	resp, err := client.UpdateEvent(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleDeleteEvent(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	eventID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(eventID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid event id")
		return
	}

	_, err = client.DeleteEvent(r.Context(), &calv1.DeleteEventRequest{Id: eventID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "event deleted"})
}

func (cr *CalendarRoutes) HandleUpdateRecurringEvent(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	eventID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(eventID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid event id")
		return
	}

	var req updateRecurringEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	scopeMap := map[string]calv1.RecurringEventScope{
		"this":            calv1.RecurringEventScope_THIS_EVENT,
		"this_and_future": calv1.RecurringEventScope_THIS_AND_FUTURE,
		"all":             calv1.RecurringEventScope_ALL_EVENTS,
	}

	scope, ok := scopeMap[req.Scope]
	if !ok {
		response.Error(wr, http.StatusBadRequest, "invalid scope: must be 'this', 'this_and_future', or 'all'")
		return
	}

	grpcReq := &calv1.UpdateRecurringEventRequest{
		EventId:      eventID,
		Scope:        scope,
		OriginalDate: req.OriginalDate,
		Title:        req.Changes.Title,
		Description:  req.Changes.Description,
		Location:     req.Changes.Location,
		ResourceId:   req.Changes.ResourceID,
		IsAllDay:     req.Changes.IsAllDay,
		CategoryId:   req.Changes.CategoryID,
		HasVideoCall: req.Changes.HasVideoCall,
		UpdatedBy:    userID,
	}
	if req.Changes.StartTime != nil {
		t, parseErr := parseTimestamp(*req.Changes.StartTime)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid start_time format in changes")
			return
		}
		grpcReq.StartTime = t
	}
	if req.Changes.EndTime != nil {
		t, parseErr := parseTimestamp(*req.Changes.EndTime)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid end_time format in changes")
			return
		}
		grpcReq.EndTime = t
	}

	resp, err := client.UpdateRecurringEvent(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleRSVPToEvent(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	eventID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(eventID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid event id")
		return
	}

	var req rsvpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.RSVPToEvent(r.Context(), &calv1.RSVPToEventRequest{
		EventId:    eventID,
		UserId:     userID,
		RsvpStatus: req.Status,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleListEventAttendees(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	eventID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(eventID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid event id")
		return
	}

	resp, err := client.ListEventAttendees(r.Context(), &calv1.ListEventAttendeesRequest{
		EventId: eventID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Reminder Handlers
// ============================================================================

func (cr *CalendarRoutes) HandleSetEventReminders(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	eventID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(eventID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid event id")
		return
	}

	var req setRemindersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.SetEventReminders(r.Context(), &calv1.SetEventRemindersRequest{
		EventId:       eventID,
		MinutesBefore: req.MinutesBefore,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleListEventReminders(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	eventID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(eventID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid event id")
		return
	}

	resp, err := client.ListEventReminders(r.Context(), &calv1.ListEventRemindersRequest{
		EventId: eventID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Event Category Handlers
// ============================================================================

func (cr *CalendarRoutes) HandleListEventCategories(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.ListEventCategories(r.Context(), &calv1.ListEventCategoriesRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleCreateEventCategory(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createEventCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(wr, http.StatusBadRequest, "name is required")
		return
	}

	resp, err := client.CreateEventCategory(r.Context(), &calv1.CreateEventCategoryRequest{
		UserId: userID,
		Name:   req.Name,
		Color:  req.Color,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (cr *CalendarRoutes) HandleDeleteEventCategory(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	categoryID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(categoryID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid category id")
		return
	}

	_, err = client.DeleteEventCategory(r.Context(), &calv1.DeleteEventCategoryRequest{Id: categoryID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "category deleted"})
}

// ============================================================================
// Resource Handlers
// ============================================================================

func (cr *CalendarRoutes) HandleListResources(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	q := r.URL.Query()
	grpcReq := &calv1.ListResourcesRequest{
		IncludeInactive: q.Get("include_inactive") == "true",
	}

	if resType := q.Get("type"); resType != "" {
		grpcReq.ResourceType = &resType
	}
	if minCapStr := q.Get("min_capacity"); minCapStr != "" {
		minCap, parseErr := strconv.Atoi(minCapStr)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid min_capacity value")
			return
		}
		mc := int32(minCap)
		grpcReq.MinCapacity = &mc
	}
	if tagsStr := q.Get("tags"); tagsStr != "" {
		grpcReq.Tags = strings.Split(tagsStr, ",")
	}
	if floor := q.Get("floor"); floor != "" {
		grpcReq.Floor = &floor
	}

	resp, err := client.ListResources(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleCreateResource(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(wr, http.StatusBadRequest, "name is required")
		return
	}

	resp, err := client.CreateResource(r.Context(), &calv1.CreateResourceRequest{
		Name:         req.Name,
		ResourceType: req.ResourceType,
		Capacity:     req.Capacity,
		Floor:        req.Floor,
		Location:     req.Location,
		Description:  req.Description,
		Tags:         req.Tags,
		CreatedBy:    userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (cr *CalendarRoutes) HandleGetResource(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	resourceID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(resourceID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid resource id")
		return
	}

	resp, err := client.GetResource(r.Context(), &calv1.GetResourceRequest{Id: resourceID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleUpdateResource(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	resourceID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(resourceID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid resource id")
		return
	}

	var req updateResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &calv1.UpdateResourceRequest{Id: resourceID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.ResourceType != nil {
		grpcReq.ResourceType = req.ResourceType
	}
	if req.Capacity != nil {
		grpcReq.Capacity = req.Capacity
	}
	if req.Floor != nil {
		grpcReq.Floor = req.Floor
	}
	if req.Location != nil {
		grpcReq.Location = req.Location
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}

	resp, err := client.UpdateResource(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleDeleteResource(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	resourceID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(resourceID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid resource id")
		return
	}

	_, err = client.DeleteResource(r.Context(), &calv1.DeleteResourceRequest{Id: resourceID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "resource deleted"})
}

func (cr *CalendarRoutes) HandleListResourceAvailability(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	resourceID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(resourceID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid resource id")
		return
	}

	q := r.URL.Query()
	startStr := q.Get("start")
	endStr := q.Get("end")

	if startStr == "" || endStr == "" {
		response.Error(wr, http.StatusBadRequest, "start and end query parameters are required")
		return
	}

	start, err := parseTimestamp(startStr)
	if err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid start time format")
		return
	}
	end, err := parseTimestamp(endStr)
	if err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid end time format")
		return
	}

	resp, err := client.ListResourceAvailability(r.Context(), &calv1.ListResourceAvailabilityRequest{
		ResourceId: resourceID,
		Start:      start,
		End:        end,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Resource Booking Handlers
// ============================================================================

func (cr *CalendarRoutes) HandleBookResource(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req bookResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	startTime, err := parseTimestamp(req.StartTime)
	if err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid start_time format")
		return
	}
	endTime, err := parseTimestamp(req.EndTime)
	if err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid end_time format")
		return
	}

	resp, err := client.BookResource(r.Context(), &calv1.BookResourceRequest{
		ResourceId: req.ResourceID,
		EventId:    req.EventID,
		BookedBy:   userID,
		StartTime:  startTime,
		EndTime:    endTime,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (cr *CalendarRoutes) HandleCancelBooking(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	bookingID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(bookingID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid booking id")
		return
	}

	_, err = client.CancelBooking(r.Context(), &calv1.CancelBookingRequest{Id: bookingID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "booking cancelled"})
}

// ============================================================================
// Holiday Handlers
// ============================================================================

func (cr *CalendarRoutes) HandleListHolidays(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	q := r.URL.Query()
	countryCode := q.Get("country_code")
	if countryCode == "" {
		response.Error(wr, http.StatusBadRequest, "country_code query parameter is required")
		return
	}

	grpcReq := &calv1.ListHolidaysRequest{
		CountryCode: countryCode,
	}

	if subCode := q.Get("subdivision_code"); subCode != "" {
		grpcReq.SubdivisionCode = &subCode
	}

	if startStr := q.Get("start"); startStr != "" {
		start, parseErr := parseTimestamp(startStr)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid start time format")
			return
		}
		grpcReq.Start = start
	}
	if endStr := q.Get("end"); endStr != "" {
		end, parseErr := parseTimestamp(endStr)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid end time format")
			return
		}
		grpcReq.End = end
	}

	resp, err := client.ListHolidays(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleSeedHolidays(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	var req seedHolidaysRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CountryCode == "" || req.Year == 0 {
		response.Error(wr, http.StatusBadRequest, "country_code and year are required")
		return
	}

	resp, err := client.SeedHolidays(r.Context(), &calv1.SeedHolidaysRequest{
		CountryCode: req.CountryCode,
		Year:        req.Year,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

// ============================================================================
// Preferences Handlers
// ============================================================================

func (cr *CalendarRoutes) HandleGetCalendarPreferences(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetCalendarPreferences(r.Context(), &calv1.GetCalendarPreferencesRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (cr *CalendarRoutes) HandleUpdateCalendarPreferences(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req updateCalPreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &calv1.UpdateCalendarPreferencesRequest{
		UserId: userID,
	}
	if req.DefaultView != nil {
		grpcReq.DefaultView = req.DefaultView
	}
	if req.WeekDays != nil {
		grpcReq.WeekDays = req.WeekDays
	}
	if req.DefaultReminderMinutes != nil {
		grpcReq.DefaultReminderMinutes = req.DefaultReminderMinutes
	}
	if req.DefaultAllDayReminderMinutes != nil {
		grpcReq.DefaultAlldayReminderMinutes = req.DefaultAllDayReminderMinutes
	}
	if req.SubdivisionCode != nil {
		grpcReq.SubdivisionCode = req.SubdivisionCode
	}
	if req.ShowTaskDeadlines != nil {
		grpcReq.ShowTaskDeadlines = req.ShowTaskDeadlines
	}

	resp, err := client.UpdateCalendarPreferences(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Task Deadlines Handler
// ============================================================================

func (cr *CalendarRoutes) HandleListTaskDeadlinesInRange(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	q := r.URL.Query()

	startStr := q.Get("start")
	endStr := q.Get("end")

	if startStr == "" || endStr == "" {
		response.Error(wr, http.StatusBadRequest, "start and end query parameters are required")
		return
	}

	start, err := parseTimestamp(startStr)
	if err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid start time format")
		return
	}
	end, err := parseTimestamp(endStr)
	if err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid end time format")
		return
	}

	resp, err := client.ListTaskDeadlinesInRange(r.Context(), &calv1.ListTaskDeadlinesInRangeRequest{
		Start:  start,
		End:    end,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// LiveKit Handler
// ============================================================================

func (cr *CalendarRoutes) HandleGenerateJoinToken(wr http.ResponseWriter, r *http.Request) {
	client, err := cr.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(wr, cr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	eventID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(eventID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid event id")
		return
	}

	var req generateJoinTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.GenerateJoinToken(r.Context(), &calv1.GenerateJoinTokenRequest{
		EventId:  eventID,
		UserId:   userID,
		UserName: req.DisplayName,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}
