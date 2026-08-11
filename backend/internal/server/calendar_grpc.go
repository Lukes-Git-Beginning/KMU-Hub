package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
	"github.com/kmuhub/kmuhub/internal/work/calendar"
	"github.com/kmuhub/kmuhub/internal/work/event"
	"github.com/kmuhub/kmuhub/internal/work/holiday"
	"github.com/kmuhub/kmuhub/internal/work/livekit"
	"github.com/kmuhub/kmuhub/internal/work/resource"
	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// CalendarGRPCServer implements the Calendar gRPC service
type CalendarGRPCServer struct {
	calv1.UnimplementedCalendarServiceServer
	calendarService *calendar.Service
	eventService    *event.Service
	resourceService *resource.Service
	holidayService  *holiday.Service
	livekitService  *livekit.Service
	bookingService  *calendar.BookingService
	emailClient     emailv1.EmailServiceClient // nil if email service unavailable
}

// NewCalendarGRPCServer creates a new Calendar gRPC server
func NewCalendarGRPCServer(
	calendarSvc *calendar.Service,
	eventSvc *event.Service,
	resourceSvc *resource.Service,
	holidaySvc *holiday.Service,
	livekitSvc *livekit.Service,
) *CalendarGRPCServer {
	return &CalendarGRPCServer{
		calendarService: calendarSvc,
		eventService:    eventSvc,
		resourceService: resourceSvc,
		holidayService:  holidaySvc,
		livekitService:  livekitSvc,
	}
}

// SetBookingService injects the booking service (call after construction).
func (s *CalendarGRPCServer) SetBookingService(svc *calendar.BookingService) {
	s.bookingService = svc
}

// SetEmailClient injects the email gRPC client for booking confirmations.
func (s *CalendarGRPCServer) SetEmailClient(c emailv1.EmailServiceClient) {
	s.emailClient = c
}

// ============================================================================
// Calendars
// ============================================================================

func (s *CalendarGRPCServer) CreateCalendar(ctx context.Context, req *calv1.CreateCalendarRequest) (*calv1.CreateCalendarResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
	}

	input := calendar.CreateInput{
		Name:         req.Name,
		CalendarType: req.CalendarType,
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.Color != nil {
		input.Color = *req.Color
	}
	if req.Timezone != nil {
		input.Timezone = *req.Timezone
	}

	cal, err := s.calendarService.Create(ctx, ownerID, tenantID, input)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.CreateCalendarResponse{
		Calendar: calendarToProto(cal),
	}, nil
}

func (s *CalendarGRPCServer) GetCalendar(ctx context.Context, req *calv1.GetCalendarRequest) (*calv1.GetCalendarResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar id")
	}

	cal, err := s.calendarService.Get(ctx, id, tenantID)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.GetCalendarResponse{
		Calendar: calendarToProto(cal),
	}, nil
}

func (s *CalendarGRPCServer) ListCalendars(ctx context.Context, req *calv1.ListCalendarsRequest) (*calv1.ListCalendarsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	calendars, err := s.calendarService.ListByUser(ctx, userID, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list calendars")
	}

	protos := make([]*calv1.CalendarWithMemberInfoProto, 0, len(calendars))
	for _, c := range calendars {
		protos = append(protos, calendarWithMemberInfoToProto(&c))
	}

	return &calv1.ListCalendarsResponse{
		Calendars: protos,
	}, nil
}

func (s *CalendarGRPCServer) UpdateCalendar(ctx context.Context, req *calv1.UpdateCalendarRequest) (*calv1.UpdateCalendarResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar id")
	}

	// Gateway handles auth; use uuid.Nil as actorID
	input := calendar.UpdateInput{}
	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.Color != nil {
		input.Color = req.Color
	}
	if req.Timezone != nil {
		input.Timezone = req.Timezone
	}

	cal, err := s.calendarService.Update(ctx, id, uuid.Nil, tenantID, input)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.UpdateCalendarResponse{
		Calendar: calendarToProto(cal),
	}, nil
}

func (s *CalendarGRPCServer) DeleteCalendar(ctx context.Context, req *calv1.DeleteCalendarRequest) (*calv1.DeleteCalendarResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar id")
	}

	// Gateway handles auth; use uuid.Nil as actorID
	if err := s.calendarService.Delete(ctx, id, uuid.Nil, tenantID); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.DeleteCalendarResponse{}, nil
}

// ============================================================================
// Calendar Members
// ============================================================================

func (s *CalendarGRPCServer) AddCalendarMember(ctx context.Context, req *calv1.AddCalendarMemberRequest) (*calv1.AddCalendarMemberResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	calendarID, err := uuid.Parse(req.CalendarId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar_id")
	}

	targetUserID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Gateway handles auth; use uuid.Nil as actorID
	if err := s.calendarService.AddMember(ctx, calendarID, uuid.Nil, targetUserID, tenantID, req.Permission); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.AddCalendarMemberResponse{}, nil
}

func (s *CalendarGRPCServer) RemoveCalendarMember(ctx context.Context, req *calv1.RemoveCalendarMemberRequest) (*calv1.RemoveCalendarMemberResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	calendarID, err := uuid.Parse(req.CalendarId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar_id")
	}

	targetUserID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Gateway handles auth; use uuid.Nil as actorID
	if err := s.calendarService.RemoveMember(ctx, calendarID, uuid.Nil, targetUserID, tenantID); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.RemoveCalendarMemberResponse{}, nil
}

func (s *CalendarGRPCServer) ListCalendarMembers(ctx context.Context, req *calv1.ListCalendarMembersRequest) (*calv1.ListCalendarMembersResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	calendarID, err := uuid.Parse(req.CalendarId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar_id")
	}

	members, err := s.calendarService.ListMembers(ctx, calendarID, tenantID)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	protos := make([]*calv1.CalendarMemberProto, 0, len(members))
	for _, m := range members {
		protos = append(protos, calendarMemberToProto(&m))
	}

	return &calv1.ListCalendarMembersResponse{
		Members: protos,
	}, nil
}

func (s *CalendarGRPCServer) UpdateCalendarMemberPermission(ctx context.Context, req *calv1.UpdateCalendarMemberPermissionRequest) (*calv1.UpdateCalendarMemberPermissionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	calendarID, err := uuid.Parse(req.CalendarId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar_id")
	}

	targetUserID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Gateway handles auth; use uuid.Nil as actorID
	if err := s.calendarService.UpdateMemberPermission(ctx, calendarID, uuid.Nil, targetUserID, tenantID, req.Permission); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.UpdateCalendarMemberPermissionResponse{}, nil
}

// ============================================================================
// Calendar Discovery / Subscription
// ============================================================================

func (s *CalendarGRPCServer) ListBrowsableCalendars(ctx context.Context, req *calv1.ListBrowsableCalendarsRequest) (*calv1.ListBrowsableCalendarsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	calendars, err := s.calendarService.ListBrowsable(ctx, userID, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list browsable calendars")
	}

	protos := make([]*calv1.CalendarProto, 0, len(calendars))
	for _, c := range calendars {
		protos = append(protos, calendarToProto(&c))
	}

	return &calv1.ListBrowsableCalendarsResponse{
		Calendars: protos,
	}, nil
}

func (s *CalendarGRPCServer) SubscribeToCalendar(ctx context.Context, req *calv1.SubscribeToCalendarRequest) (*calv1.SubscribeToCalendarResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	calendarID, err := uuid.Parse(req.CalendarId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.calendarService.Subscribe(ctx, calendarID, userID, tenantID); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.SubscribeToCalendarResponse{}, nil
}

func (s *CalendarGRPCServer) UnsubscribeFromCalendar(ctx context.Context, req *calv1.UnsubscribeFromCalendarRequest) (*calv1.UnsubscribeFromCalendarResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	calendarID, err := uuid.Parse(req.CalendarId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.calendarService.Unsubscribe(ctx, calendarID, userID, tenantID); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.UnsubscribeFromCalendarResponse{}, nil
}

// ============================================================================
// Events
// ============================================================================

func (s *CalendarGRPCServer) CreateEvent(ctx context.Context, req *calv1.CreateEventRequest) (*calv1.CreateEventResponse, error) {
	calendarID, err := uuid.Parse(req.CalendarId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar_id")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	input := event.CreateInput{
		CalendarID:   calendarID,
		Title:        req.Title,
		IsAllDay:     req.IsAllDay,
		HasVideoCall: req.HasVideoCall,
	}

	if req.Description != nil {
		input.Description = req.Description
	}
	if req.Location != nil {
		input.Location = req.Location
	}
	if req.ResourceId != nil {
		resID, parseErr := uuid.Parse(*req.ResourceId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid resource_id")
		}
		input.ResourceID = &resID
	}
	if req.StartTime != nil {
		input.StartTime = req.StartTime.AsTime()
	}
	if req.EndTime != nil {
		input.EndTime = req.EndTime.AsTime()
	}
	if req.Timezone != nil {
		input.Timezone = *req.Timezone
	}
	if req.Rrule != nil {
		input.RRule = req.Rrule
	}
	if req.CategoryId != nil {
		catID, parseErr := uuid.Parse(*req.CategoryId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid category_id")
		}
		input.CategoryID = &catID
	}

	// Parse attendee IDs
	for _, aidStr := range req.AttendeeUserIds {
		aid, parseErr := uuid.Parse(aidStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid attendee user id")
		}
		input.AttendeeIDs = append(input.AttendeeIDs, aid)
	}

	evt, err := s.eventService.Create(ctx, createdBy, input)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.CreateEventResponse{
		Event: eventToProto(evt),
	}, nil
}

func (s *CalendarGRPCServer) GetEvent(ctx context.Context, req *calv1.GetEventRequest) (*calv1.GetEventResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event id")
	}

	evt, err := s.eventService.Get(ctx, id, tenantID)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.GetEventResponse{
		Event: &calv1.ExpandedEventProto{
			Event: eventToProto(evt),
		},
	}, nil
}

func (s *CalendarGRPCServer) ListEventsInRange(ctx context.Context, req *calv1.ListEventsInRangeRequest) (*calv1.ListEventsInRangeResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	var calendarIDs []uuid.UUID
	for _, idStr := range req.CalendarIds {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid calendar_id in list")
		}
		calendarIDs = append(calendarIDs, id)
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	start := req.Start.AsTime()
	end := req.End.AsTime()

	events, err := s.eventService.ListEventsInRange(ctx, calendarIDs, start, end, userID, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list events")
	}

	protos := make([]*calv1.ExpandedEventProto, 0, len(events))
	for _, e := range events {
		protos = append(protos, expandedEventToProto(&e))
	}

	return &calv1.ListEventsInRangeResponse{
		Events: protos,
	}, nil
}

func (s *CalendarGRPCServer) UpdateEvent(ctx context.Context, req *calv1.UpdateEventRequest) (*calv1.UpdateEventResponse, error) {
	eventID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event id")
	}

	actorID, err := uuid.Parse(req.UpdatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid updated_by")
	}

	input := event.UpdateInput{}
	if req.Title != nil {
		input.Title = req.Title
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.Location != nil {
		input.Location = req.Location
	}
	if req.ResourceId != nil {
		resID, parseErr := uuid.Parse(*req.ResourceId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid resource_id")
		}
		input.ResourceID = &resID
	}
	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		input.StartTime = &t
	}
	if req.EndTime != nil {
		t := req.EndTime.AsTime()
		input.EndTime = &t
	}
	if req.IsAllDay != nil {
		input.IsAllDay = req.IsAllDay
	}
	if req.CategoryId != nil {
		catID, parseErr := uuid.Parse(*req.CategoryId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid category_id")
		}
		input.CategoryID = &catID
	}
	if req.HasVideoCall != nil {
		input.HasVideoCall = req.HasVideoCall
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	evt, err := s.eventService.UpdateEvent(ctx, eventID, actorID, tenantID, input)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.UpdateEventResponse{
		Event: eventToProto(evt),
	}, nil
}

func (s *CalendarGRPCServer) DeleteEvent(ctx context.Context, req *calv1.DeleteEventRequest) (*calv1.DeleteEventResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	eventID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event id")
	}

	// Gateway handles auth; use uuid.Nil as actorID
	if err := s.eventService.DeleteEvent(ctx, eventID, uuid.Nil, tenantID); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.DeleteEventResponse{}, nil
}

func (s *CalendarGRPCServer) UpdateRecurringEvent(ctx context.Context, req *calv1.UpdateRecurringEventRequest) (*calv1.UpdateRecurringEventResponse, error) {
	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event_id")
	}

	actorID, err := uuid.Parse(req.UpdatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid updated_by")
	}

	// Parse original date from string (YYYY-MM-DD)
	originalDate, err := time.Parse("2006-01-02", req.OriginalDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid original_date format, expected YYYY-MM-DD")
	}

	scope := recurringEventScopeToString(req.Scope)

	// Build changes
	changes := event.UpdateInput{}
	if req.Title != nil {
		changes.Title = req.Title
	}
	if req.Description != nil {
		changes.Description = req.Description
	}
	if req.Location != nil {
		changes.Location = req.Location
	}
	if req.ResourceId != nil {
		resID, parseErr := uuid.Parse(*req.ResourceId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid resource_id")
		}
		changes.ResourceID = &resID
	}
	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		changes.StartTime = &t
	}
	if req.EndTime != nil {
		t := req.EndTime.AsTime()
		changes.EndTime = &t
	}
	if req.IsAllDay != nil {
		changes.IsAllDay = req.IsAllDay
	}
	if req.HasVideoCall != nil {
		changes.HasVideoCall = req.HasVideoCall
	}
	if req.CategoryId != nil {
		catID, parseErr := uuid.Parse(*req.CategoryId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid category_id")
		}
		changes.CategoryID = &catID
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	evt, err := s.eventService.UpdateRecurringEvent(ctx, eventID, actorID, tenantID, scope, originalDate, changes)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.UpdateRecurringEventResponse{
		Event: eventToProto(evt),
	}, nil
}

func (s *CalendarGRPCServer) RSVPToEvent(ctx context.Context, req *calv1.RSVPToEventRequest) (*calv1.RSVPToEventResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.eventService.RSVPToEvent(ctx, eventID, userID, tenantID, req.RsvpStatus); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.RSVPToEventResponse{}, nil
}

func (s *CalendarGRPCServer) ListEventAttendees(ctx context.Context, req *calv1.ListEventAttendeesRequest) (*calv1.ListEventAttendeesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event_id")
	}

	attendees, err := s.eventService.ListAttendees(ctx, eventID, tenantID)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	protos := make([]*calv1.EventAttendeeProto, 0, len(attendees))
	for _, a := range attendees {
		protos = append(protos, attendeeToProto(&a))
	}

	return &calv1.ListEventAttendeesResponse{
		Attendees: protos,
	}, nil
}

// ============================================================================
// Event Categories
// ============================================================================

func (s *CalendarGRPCServer) CreateEventCategory(ctx context.Context, req *calv1.CreateEventCategoryRequest) (*calv1.CreateEventCategoryResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	cat, err := s.calendarService.CreateCategory(ctx, userID, tenantID, req.Name, req.Color)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.CreateEventCategoryResponse{
		Category: categoryToProto(cat),
	}, nil
}

func (s *CalendarGRPCServer) ListEventCategories(ctx context.Context, req *calv1.ListEventCategoriesRequest) (*calv1.ListEventCategoriesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	categories, err := s.calendarService.ListCategories(ctx, userID, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list categories")
	}

	protos := make([]*calv1.EventCategoryProto, 0, len(categories))
	for _, c := range categories {
		protos = append(protos, categoryToProto(&c))
	}

	return &calv1.ListEventCategoriesResponse{
		Categories: protos,
	}, nil
}

func (s *CalendarGRPCServer) DeleteEventCategory(ctx context.Context, req *calv1.DeleteEventCategoryRequest) (*calv1.DeleteEventCategoryResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	categoryID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid category id")
	}

	// Gateway handles auth; use uuid.Nil as userID
	if err := s.calendarService.DeleteCategory(ctx, categoryID, uuid.Nil, tenantID); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.DeleteEventCategoryResponse{}, nil
}

// ============================================================================
// Reminders
// ============================================================================

func (s *CalendarGRPCServer) SetEventReminders(ctx context.Context, req *calv1.SetEventRemindersRequest) (*calv1.SetEventRemindersResponse, error) {
	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event_id")
	}

	// Convert int32 to int slice
	var minutesBefore []int
	for _, m := range req.MinutesBefore {
		minutesBefore = append(minutesBefore, int(m))
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	// Gateway handles auth; use uuid.Nil as actorID
	if err := s.eventService.SetReminders(ctx, eventID, uuid.Nil, tenantID, minutesBefore); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.SetEventRemindersResponse{}, nil
}

func (s *CalendarGRPCServer) ListEventReminders(ctx context.Context, req *calv1.ListEventRemindersRequest) (*calv1.ListEventRemindersResponse, error) {
	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event_id")
	}

	reminders, err := s.eventService.ListReminders(ctx, eventID)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	protos := make([]*calv1.EventReminderProto, 0, len(reminders))
	for _, r := range reminders {
		protos = append(protos, reminderToProto(&r))
	}

	return &calv1.ListEventRemindersResponse{
		Reminders: protos,
	}, nil
}

// ============================================================================
// Resources
// ============================================================================

func (s *CalendarGRPCServer) CreateResource(ctx context.Context, req *calv1.CreateResourceRequest) (*calv1.CreateResourceResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	input := resource.CreateInput{
		TenantID:     tenantID,
		Name:         req.Name,
		ResourceType: req.ResourceType,
		Tags:         req.Tags,
		CreatedBy:    createdBy,
		IsAdmin:      true, // Gateway handles auth
	}
	if req.Capacity != nil {
		c := int(*req.Capacity)
		input.Capacity = &c
	}
	if req.Floor != nil {
		input.Floor = req.Floor
	}
	if req.Location != nil {
		input.Location = req.Location
	}
	if req.Description != nil {
		input.Description = req.Description
	}

	res, err := s.resourceService.Create(ctx, input)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.CreateResourceResponse{
		Resource: resourceToProto(res),
	}, nil
}

func (s *CalendarGRPCServer) GetResource(ctx context.Context, req *calv1.GetResourceRequest) (*calv1.GetResourceResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid resource id")
	}

	res, err := s.resourceService.Get(ctx, id, tenantID)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.GetResourceResponse{
		Resource: resourceToProto(res),
	}, nil
}

func (s *CalendarGRPCServer) ListResources(ctx context.Context, req *calv1.ListResourcesRequest) (*calv1.ListResourcesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	filters := resource.ResourceFilters{
		TenantID: tenantID,
		Tags:     req.Tags,
	}
	if req.ResourceType != nil {
		filters.Type = req.ResourceType
	}
	if req.MinCapacity != nil {
		mc := int(*req.MinCapacity)
		filters.MinCapacity = &mc
	}
	if req.Floor != nil {
		filters.Floor = req.Floor
	}
	if req.IncludeInactive {
		// Don't set IsActive filter, so all resources are returned
	} else {
		active := true
		filters.IsActive = &active
	}

	resources, err := s.resourceService.List(ctx, filters)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list resources")
	}

	protos := make([]*calv1.ResourceProto, 0, len(resources))
	for _, r := range resources {
		protos = append(protos, resourceToProto(&r))
	}

	return &calv1.ListResourcesResponse{
		Resources: protos,
	}, nil
}

func (s *CalendarGRPCServer) UpdateResource(ctx context.Context, req *calv1.UpdateResourceRequest) (*calv1.UpdateResourceResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid resource id")
	}

	input := resource.UpdateInput{}
	if req.Name != nil {
		input.Name = req.Name
	}
	if req.ResourceType != nil {
		input.ResourceType = req.ResourceType
	}
	if req.Capacity != nil {
		c := int(*req.Capacity)
		input.Capacity = &c
	}
	if req.Floor != nil {
		input.Floor = req.Floor
	}
	if req.Location != nil {
		input.Location = req.Location
	}
	if req.Description != nil {
		input.Description = req.Description
	}

	res, err := s.resourceService.Update(ctx, id, tenantID, true, input) // isAdmin=true, gateway handles auth
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.UpdateResourceResponse{
		Resource: resourceToProto(res),
	}, nil
}

func (s *CalendarGRPCServer) DeleteResource(ctx context.Context, req *calv1.DeleteResourceRequest) (*calv1.DeleteResourceResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid resource id")
	}

	if err := s.resourceService.Delete(ctx, id, tenantID, true); err != nil { // isAdmin=true, gateway handles auth
		return nil, mapCalendarError(err)
	}

	return &calv1.DeleteResourceResponse{}, nil
}

// ============================================================================
// Resource Availability & Booking
// ============================================================================

func (s *CalendarGRPCServer) ListResourceAvailability(ctx context.Context, req *calv1.ListResourceAvailabilityRequest) (*calv1.ListResourceAvailabilityResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	resourceID, err := uuid.Parse(req.ResourceId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid resource_id")
	}

	date := req.Start.AsTime()

	bookings, err := s.resourceService.ListAvailability(ctx, resourceID, tenantID, date)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	protos := make([]*calv1.ResourceBookingProto, 0, len(bookings))
	for _, b := range bookings {
		protos = append(protos, bookingToProto(&b))
	}

	return &calv1.ListResourceAvailabilityResponse{
		Bookings: protos,
	}, nil
}

func (s *CalendarGRPCServer) BookResource(ctx context.Context, req *calv1.BookResourceRequest) (*calv1.BookResourceResponse, error) {
	resourceID, err := uuid.Parse(req.ResourceId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid resource_id")
	}

	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event_id")
	}

	actorID, err := uuid.Parse(req.BookedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid booked_by")
	}

	start := req.StartTime.AsTime()
	end := req.EndTime.AsTime()

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	booking, err := s.resourceService.Book(ctx, actorID, resourceID, eventID, tenantID, start, end)
	if err != nil {
		// Check for BookingConflictError and include alternatives in error details
		var conflictErr *resource.BookingConflictError
		if errors.As(err, &conflictErr) {
			var msgBuilder strings.Builder
			msgBuilder.WriteString(conflictErr.Error())
			if len(conflictErr.Alternatives) > 0 {
				msgBuilder.WriteString(" | alternatives: ")
				for i, alt := range conflictErr.Alternatives {
					if i > 0 {
						msgBuilder.WriteString(", ")
					}
					fmt.Fprintf(&msgBuilder, "%s (%s)", alt.ResourceName, alt.ResourceID.String())
				}
			}
			return nil, status.Error(codes.AlreadyExists, msgBuilder.String())
		}
		return nil, mapCalendarError(err)
	}

	return &calv1.BookResourceResponse{
		Booking: bookingToProto(booking),
	}, nil
}

func (s *CalendarGRPCServer) CancelBooking(ctx context.Context, req *calv1.CancelBookingRequest) (*calv1.CancelBookingResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	bookingID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid booking id")
	}

	actorID, err := uuid.Parse(middleware.GetUserID(ctx))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid user_id in token")
	}

	if err := s.resourceService.CancelBooking(ctx, bookingID, actorID, tenantID); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.CancelBookingResponse{}, nil
}

func (s *CalendarGRPCServer) ListResourceBookings(ctx context.Context, req *calv1.ListResourceBookingsRequest) (*calv1.ListResourceBookingsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	if req.ResourceId == nil {
		return nil, status.Error(codes.InvalidArgument, "resource_id is required")
	}

	resourceID, err := uuid.Parse(*req.ResourceId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid resource_id")
	}

	// Use start date for ListAvailability
	date := req.Start.AsTime()

	bookings, err := s.resourceService.ListAvailability(ctx, resourceID, tenantID, date)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	protos := make([]*calv1.ResourceBookingProto, 0, len(bookings))
	for _, b := range bookings {
		protos = append(protos, bookingToProto(&b))
	}

	return &calv1.ListResourceBookingsResponse{
		Bookings: protos,
	}, nil
}

// ============================================================================
// Holidays
// ============================================================================

func (s *CalendarGRPCServer) ListHolidays(ctx context.Context, req *calv1.ListHolidaysRequest) (*calv1.ListHolidaysResponse, error) {
	start := req.Start.AsTime()
	end := req.End.AsTime()

	var subdivisionCode *string
	if req.SubdivisionCode != nil {
		subdivisionCode = req.SubdivisionCode
	}

	holidays, err := s.holidayService.ListHolidays(ctx, start, end, req.CountryCode, subdivisionCode)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	protos := make([]*calv1.PublicHolidayProto, 0, len(holidays))
	for _, h := range holidays {
		protos = append(protos, holidayToProto(&h))
	}

	return &calv1.ListHolidaysResponse{
		Holidays: protos,
	}, nil
}

func (s *CalendarGRPCServer) SeedHolidays(ctx context.Context, req *calv1.SeedHolidaysRequest) (*calv1.SeedHolidaysResponse, error) {
	if err := s.holidayService.SeedHolidays(ctx, int(req.Year), req.CountryCode); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.SeedHolidaysResponse{}, nil
}

// ============================================================================
// Preferences
// ============================================================================

func (s *CalendarGRPCServer) GetCalendarPreferences(ctx context.Context, req *calv1.GetCalendarPreferencesRequest) (*calv1.GetCalendarPreferencesResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	prefs, err := s.calendarService.GetPreferences(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get preferences")
	}

	return &calv1.GetCalendarPreferencesResponse{
		Preferences: calPreferencesToProto(prefs),
	}, nil
}

func (s *CalendarGRPCServer) UpdateCalendarPreferences(ctx context.Context, req *calv1.UpdateCalendarPreferencesRequest) (*calv1.UpdateCalendarPreferencesResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Get existing or use defaults
	prefs, _ := s.calendarService.GetPreferences(ctx, userID)
	if prefs == nil {
		prefs = &models.UserCalendarPreferences{
			UserID:                       userID,
			DefaultView:                  "week",
			WeekDays:                     5,
			DefaultReminderMinutes:       15,
			DefaultAllDayReminderMinutes: 1440,
			ShowTaskDeadlines:            true,
			UpdatedAt:                    time.Now(),
		}
	}

	if req.DefaultView != nil {
		prefs.DefaultView = *req.DefaultView
	}
	if req.WeekDays != nil {
		prefs.WeekDays = int(*req.WeekDays)
	}
	if req.DefaultReminderMinutes != nil {
		prefs.DefaultReminderMinutes = int(*req.DefaultReminderMinutes)
	}
	if req.DefaultAlldayReminderMinutes != nil {
		prefs.DefaultAllDayReminderMinutes = int(*req.DefaultAlldayReminderMinutes)
	}
	if req.SubdivisionCode != nil {
		prefs.SubdivisionCode = req.SubdivisionCode
	}
	if req.ShowTaskDeadlines != nil {
		prefs.ShowTaskDeadlines = *req.ShowTaskDeadlines
	}

	if err := s.calendarService.UpsertPreferences(ctx, prefs); err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.UpdateCalendarPreferencesResponse{
		Preferences: calPreferencesToProto(prefs),
	}, nil
}

// ============================================================================
// Task Deadlines
// ============================================================================

func (s *CalendarGRPCServer) ListTaskDeadlinesInRange(ctx context.Context, req *calv1.ListTaskDeadlinesInRangeRequest) (*calv1.ListTaskDeadlinesInRangeResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	start := req.Start.AsTime()
	end := req.End.AsTime()

	deadlines, err := s.eventService.ListTaskDeadlinesInRange(ctx, userID, start, end)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list task deadlines")
	}

	protos := make([]*calv1.TaskDeadlineStubProto, 0, len(deadlines))
	for _, d := range deadlines {
		protos = append(protos, taskDeadlineToProto(&d))
	}

	return &calv1.ListTaskDeadlinesInRangeResponse{
		Deadlines: protos,
	}, nil
}

// ============================================================================
// LiveKit
// ============================================================================

func (s *CalendarGRPCServer) GenerateJoinToken(ctx context.Context, req *calv1.GenerateJoinTokenRequest) (*calv1.GenerateJoinTokenResponse, error) {
	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event_id")
	}

	roomName := s.livekitService.GenerateRoomName(eventID)

	token, err := s.livekitService.GenerateJoinToken(roomName, req.UserId, req.UserName)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	meetingLink := s.livekitService.GenerateMeetingLink(roomName)

	return &calv1.GenerateJoinTokenResponse{
		Token:    token,
		RoomName: roomName,
		WsUrl:    meetingLink,
	}, nil
}

// ============================================================================
// Proto Converters
// ============================================================================

func calendarToProto(c *models.Calendar) *calv1.CalendarProto {
	proto := &calv1.CalendarProto{
		Id:           c.ID.String(),
		Name:         c.Name,
		CalendarType: c.CalendarType,
		Color:        c.Color,
		OwnerId:      c.OwnerID.String(),
		IsDefault:    c.IsDefault,
		Timezone:     c.Timezone,
		CreatedAt:    timestamppb.New(c.CreatedAt),
		UpdatedAt:    timestamppb.New(c.UpdatedAt),
	}
	if c.Description != nil {
		proto.Description = c.Description
	}
	return proto
}

func calendarWithMemberInfoToProto(c *models.CalendarWithMemberInfo) *calv1.CalendarWithMemberInfoProto {
	return &calv1.CalendarWithMemberInfoProto{
		Calendar:      calendarToProto(&c.Calendar),
		Permission:    c.Permission,
		ColorOverride: c.ColorOverride,
		IsVisible:     c.IsVisible,
	}
}

func calendarMemberToProto(m *models.CalendarMember) *calv1.CalendarMemberProto {
	proto := &calv1.CalendarMemberProto{
		CalendarId: m.CalendarID.String(),
		UserId:     m.UserID.String(),
		Permission: m.Permission,
		IsVisible:  m.IsVisible,
		CreatedAt:  timestamppb.New(m.CreatedAt),
		FirstName:  m.FirstName,
		LastName:   m.LastName,
	}
	if m.ColorOverride != nil {
		proto.ColorOverride = m.ColorOverride
	}
	return proto
}

func eventToProto(e *models.CalendarEvent) *calv1.CalendarEventProto {
	proto := &calv1.CalendarEventProto{
		Id:           e.ID.String(),
		CalendarId:   e.CalendarID.String(),
		Title:        e.Title,
		StartTime:    timestamppb.New(e.StartTime),
		EndTime:      timestamppb.New(e.EndTime),
		IsAllDay:     e.IsAllDay,
		Timezone:     e.Timezone,
		HasVideoCall: e.HasVideoCall,
		CreatedBy:    e.CreatedBy.String(),
		CreatedAt:    timestamppb.New(e.CreatedAt),
		UpdatedAt:    timestamppb.New(e.UpdatedAt),
	}
	if e.Description != nil {
		proto.Description = e.Description
	}
	if e.Location != nil {
		proto.Location = e.Location
	}
	if e.ResourceID != nil {
		s := e.ResourceID.String()
		proto.ResourceId = &s
	}
	if e.RRule != nil {
		proto.Rrule = e.RRule
	}
	if e.RecurrenceEnd != nil {
		proto.RecurrenceEnd = timestamppb.New(*e.RecurrenceEnd)
	}
	if e.LiveKitRoomName != nil {
		proto.LivekitRoomName = e.LiveKitRoomName
	}
	if e.CategoryID != nil {
		s := e.CategoryID.String()
		proto.CategoryId = &s
	}
	return proto
}

func expandedEventToProto(e *models.ExpandedEvent) *calv1.ExpandedEventProto {
	proto := &calv1.ExpandedEventProto{
		Event:         eventToProto(&e.CalendarEvent),
		CalendarName:  e.CalendarName,
		CalendarColor: e.CalendarColor,
		DisplayColor:  e.DisplayColor,
		AttendeeCount: int32(e.AttendeeCount),
	}
	if e.OriginalEventID != nil {
		s := e.OriginalEventID.String()
		proto.OriginalEventId = &s
	}
	if e.OriginalDate != nil {
		proto.OriginalDate = timestamppb.New(*e.OriginalDate)
	}
	if e.CategoryName != nil {
		proto.CategoryName = e.CategoryName
	}
	if e.CategoryColor != nil {
		proto.CategoryColor = e.CategoryColor
	}
	if e.ResourceName != nil {
		proto.ResourceName = e.ResourceName
	}
	if e.MyRSVP != nil {
		proto.MyRsvp = e.MyRSVP
	}
	return proto
}

func attendeeToProto(a *models.EventAttendee) *calv1.EventAttendeeProto {
	proto := &calv1.EventAttendeeProto{
		EventId:    a.EventID.String(),
		UserId:     a.UserID.String(),
		RsvpStatus: a.RSVPStatus,
		CreatedAt:  timestamppb.New(a.CreatedAt),
		FirstName:  a.FirstName,
		LastName:   a.LastName,
	}
	if a.RespondedAt != nil {
		proto.RespondedAt = timestamppb.New(*a.RespondedAt)
	}
	return proto
}

func reminderToProto(r *models.EventReminder) *calv1.EventReminderProto {
	return &calv1.EventReminderProto{
		Id:            r.ID.String(),
		EventId:       r.EventID.String(),
		MinutesBefore: int32(r.MinutesBefore),
		CreatedAt:     timestamppb.New(r.CreatedAt),
	}
}

func categoryToProto(c *models.EventCategory) *calv1.EventCategoryProto {
	return &calv1.EventCategoryProto{
		Id:        c.ID.String(),
		UserId:    c.UserID.String(),
		Name:      c.Name,
		Color:     c.Color,
		SortOrder: int32(c.SortOrder),
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
}

func resourceToProto(r *models.Resource) *calv1.ResourceProto {
	proto := &calv1.ResourceProto{
		Id:           r.ID.String(),
		Name:         r.Name,
		ResourceType: r.ResourceType,
		IsActive:     r.IsActive,
		CreatedBy:    r.CreatedBy.String(),
		CreatedAt:    timestamppb.New(r.CreatedAt),
		UpdatedAt:    timestamppb.New(r.UpdatedAt),
		Tags:         r.Tags,
	}
	if r.Capacity != nil {
		c := int32(*r.Capacity)
		proto.Capacity = &c
	}
	if r.Floor != nil {
		proto.Floor = r.Floor
	}
	if r.Location != nil {
		proto.Location = r.Location
	}
	if r.Description != nil {
		proto.Description = r.Description
	}
	return proto
}

func bookingToProto(b *models.ResourceBooking) *calv1.ResourceBookingProto {
	proto := &calv1.ResourceBookingProto{
		Id:           b.ID.String(),
		ResourceId:   b.ResourceID.String(),
		EventId:      b.EventID.String(),
		BookedBy:     b.BookedBy.String(),
		StartTime:    timestamppb.New(b.StartTime),
		EndTime:      timestamppb.New(b.EndTime),
		CreatedAt:    timestamppb.New(b.CreatedAt),
		ResourceName: b.ResourceName,
		EventTitle:   b.EventTitle,
	}
	if b.CancelledAt != nil {
		proto.CancelledAt = timestamppb.New(*b.CancelledAt)
	}
	return proto
}

func holidayToProto(h *models.PublicHoliday) *calv1.PublicHolidayProto {
	return &calv1.PublicHolidayProto{
		Id:               h.ID.String(),
		Date:             h.Date.Format("2006-01-02"),
		Name:             h.Name,
		LocalName:        h.LocalName,
		CountryCode:      h.CountryCode,
		IsGlobal:         h.IsGlobal,
		SubdivisionCodes: h.SubdivisionCodes,
		HolidayType:      h.HolidayType,
		Year:             int32(h.Year),
	}
}

func calPreferencesToProto(p *models.UserCalendarPreferences) *calv1.CalendarPreferencesProto {
	proto := &calv1.CalendarPreferencesProto{
		UserId:                       p.UserID.String(),
		DefaultView:                  p.DefaultView,
		WeekDays:                     int32(p.WeekDays),
		DefaultReminderMinutes:       int32(p.DefaultReminderMinutes),
		DefaultAlldayReminderMinutes: int32(p.DefaultAllDayReminderMinutes),
		ShowTaskDeadlines:            p.ShowTaskDeadlines,
	}
	if p.SubdivisionCode != nil {
		proto.SubdivisionCode = p.SubdivisionCode
	}
	return proto
}

func taskDeadlineToProto(t *models.TaskDeadlineStub) *calv1.TaskDeadlineStubProto {
	proto := &calv1.TaskDeadlineStubProto{
		TaskId:   t.TaskID.String(),
		Title:    t.Title,
		DueDate:  t.DueDate.Format("2006-01-02"),
		Priority: t.Priority,
	}
	if t.ProjectKey != nil {
		proto.ProjectKey = t.ProjectKey
	}
	return proto
}

// ============================================================================
// Helpers
// ============================================================================

func recurringEventScopeToString(scope calv1.RecurringEventScope) string {
	switch scope {
	case calv1.RecurringEventScope_THIS_EVENT:
		return "this"
	case calv1.RecurringEventScope_THIS_AND_FUTURE:
		return "this_and_future"
	case calv1.RecurringEventScope_ALL_EVENTS:
		return "all"
	default:
		return "this"
	}
}

// mapCalendarError maps calendar/event/resource/holiday/livekit service errors to gRPC status errors
func mapCalendarError(err error) error {
	// Calendar errors
	switch {
	case errors.Is(err, calendar.ErrCalendarNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, calendar.ErrCalendarNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, calendar.ErrInvalidCalendarType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, calendar.ErrMemberAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, calendar.ErrMemberNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, calendar.ErrNotCalendarOwner):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, calendar.ErrNotCalendarAdmin):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, calendar.ErrInsufficientPermission):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, calendar.ErrCannotDeleteDefaultCalendar):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, calendar.ErrCannotDeletePersonalCalendar):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, calendar.ErrCategoryNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, calendar.ErrCategoryNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, calendar.ErrDuplicateCategoryName):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, calendar.ErrSubscriptionExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, calendar.ErrSubscriptionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, calendar.ErrInvalidPermission):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, calendar.ErrInvalidColor):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, calendar.ErrCannotSubscribePersonal):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, calendar.ErrOwnerCannotBeAdded):
		return status.Error(codes.InvalidArgument, err.Error())
	// Event errors
	case errors.Is(err, event.ErrEventNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, event.ErrEventTitleRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, event.ErrInvalidTimeRange):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, event.ErrInvalidRRule):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, event.ErrInvalidRSVPStatus):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, event.ErrAlreadyAttendee):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, event.ErrNotAttendee):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, event.ErrNotEventCreator):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, event.ErrInvalidRecurringEditScope):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, event.ErrEventNotRecurring):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, event.ErrReminderLimitExceeded):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, event.ErrInvalidReminderMinutes):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, event.ErrInsufficientPermission):
		return status.Error(codes.PermissionDenied, err.Error())
	// Resource errors
	case errors.Is(err, resource.ErrResourceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, resource.ErrResourceNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, resource.ErrInvalidResourceType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, resource.ErrResourceInactive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, resource.ErrBookingNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, resource.ErrBookingConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, resource.ErrBookingAlreadyCancelled):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, resource.ErrNotBookingOwner):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, resource.ErrInvalidBookingTimeRange):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, resource.ErrNotAuthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	// Holiday errors
	case errors.Is(err, holiday.ErrInvalidCountryCode):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, holiday.ErrInvalidYear):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, holiday.ErrNagerAPIUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	// LiveKit errors
	case errors.Is(err, livekit.ErrLiveKitNotConfigured):
		return status.Error(codes.Unavailable, err.Error())
	// Booking page errors
	case errors.Is(err, calendar.ErrBookingPageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, calendar.ErrBookingPageSlugRequired),
		errors.Is(err, calendar.ErrBookingPageCompanyNameRequired),
		errors.Is(err, calendar.ErrBookingPageInvalidRules):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, calendar.ErrBookingServiceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, calendar.ErrBookingSlotUnavailable),
		errors.Is(err, calendar.ErrBookingSlotInPast):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		slog.Error("unhandled calendar service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

// ============================================================================
// Booking Page Admin RPCs
// ============================================================================

func (s *CalendarGRPCServer) CreateBookingPage(ctx context.Context, req *calv1.CreateBookingPageRequest) (*calv1.CreateBookingPageResponse, error) {
	if s.bookingService == nil {
		return nil, status.Error(codes.Unimplemented, "booking service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	calID, err := uuid.Parse(req.GetCalendarId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid calendar_id")
	}

	input := calendar.CreateBookingPageInput{
		CalendarID:            calID,
		Slug:                  req.GetSlug(),
		CompanyName:           req.GetCompanyName(),
		AvailabilityRulesJSON: req.GetAvailabilityRulesJson(),
	}
	if v := req.GetLogoUrl(); v != "" {
		input.LogoURL = &v
	}
	for _, svc := range req.GetServices() {
		bs := protoToBookingService(svc)
		input.Services = append(input.Services, bs)
	}

	page, err := s.bookingService.CreateBookingPage(ctx, tenantID, input)
	if err != nil {
		return nil, mapCalendarError(err)
	}
	return &calv1.CreateBookingPageResponse{Page: bookingPageToProto(page)}, nil
}

func (s *CalendarGRPCServer) GetBookingPage(ctx context.Context, req *calv1.GetBookingPageRequest) (*calv1.GetBookingPageResponse, error) {
	if s.bookingService == nil {
		return nil, status.Error(codes.Unimplemented, "booking service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	page, err := s.bookingService.GetBookingPage(ctx, id, tenantID)
	if err != nil {
		return nil, mapCalendarError(err)
	}
	return &calv1.GetBookingPageResponse{Page: bookingPageToProto(page)}, nil
}

func (s *CalendarGRPCServer) ListBookingPages(ctx context.Context, req *calv1.ListBookingPagesRequest) (*calv1.ListBookingPagesResponse, error) {
	if s.bookingService == nil {
		return nil, status.Error(codes.Unimplemented, "booking service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	pages, err := s.bookingService.ListBookingPages(ctx, tenantID, req.GetIncludeInactive())
	if err != nil {
		return nil, mapCalendarError(err)
	}
	protos := make([]*calv1.BookingPageProto, 0, len(pages))
	for _, p := range pages {
		pp := p
		protos = append(protos, bookingPageToProto(&pp))
	}
	return &calv1.ListBookingPagesResponse{Pages: protos}, nil
}

func (s *CalendarGRPCServer) UpdateBookingPage(ctx context.Context, req *calv1.UpdateBookingPageRequest) (*calv1.UpdateBookingPageResponse, error) {
	if s.bookingService == nil {
		return nil, status.Error(codes.Unimplemented, "booking service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	input := calendar.UpdateBookingPageInput{}
	if v := req.GetCompanyName(); v != "" {
		input.CompanyName = &v
	}
	if v := req.GetLogoUrl(); v != "" {
		input.LogoURL = &v
	}
	if len(req.GetServices()) > 0 {
		svcs := make([]models.BookingService, 0, len(req.GetServices()))
		for _, s := range req.GetServices() {
			svcs = append(svcs, protoToBookingService(s))
		}
		input.Services = svcs
	}
	if v := req.GetAvailabilityRulesJson(); v != "" {
		input.AvailabilityRulesJSON = &v
	}
	if req.Active != nil {
		a := req.GetActive()
		input.Active = &a
	}

	page, err := s.bookingService.UpdateBookingPage(ctx, id, tenantID, input)
	if err != nil {
		return nil, mapCalendarError(err)
	}
	return &calv1.UpdateBookingPageResponse{Page: bookingPageToProto(page)}, nil
}

func (s *CalendarGRPCServer) DeleteBookingPage(ctx context.Context, req *calv1.DeleteBookingPageRequest) (*calv1.DeleteBookingPageResponse, error) {
	if s.bookingService == nil {
		return nil, status.Error(codes.Unimplemented, "booking service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	if err := s.bookingService.DeleteBookingPage(ctx, id, tenantID); err != nil {
		return nil, mapCalendarError(err)
	}
	return &calv1.DeleteBookingPageResponse{}, nil
}

// ============================================================================
// Booking Pages Public RPCs
// ============================================================================

func (s *CalendarGRPCServer) GetPublicBookingPage(ctx context.Context, req *calv1.GetPublicBookingPageRequest) (*calv1.GetPublicBookingPageResponse, error) {
	if s.bookingService == nil {
		return nil, status.Error(codes.Unimplemented, "booking service not configured")
	}
	// Public route: no JWT, no tenant metadata — RLS requires system context.
	ctx = sysctx.With(ctx)
	page, err := s.bookingService.GetPublicBookingPage(ctx, req.GetSlug())
	if err != nil {
		return nil, mapCalendarError(err)
	}

	svcs := make([]*calv1.BookingServiceProto, 0, len(page.Services))
	for _, svc := range page.Services {
		s := svc
		svcs = append(svcs, bookingServiceToProto(&s))
	}
	pub := &calv1.PublicBookingPageProto{
		Slug:        page.Slug,
		CompanyName: page.CompanyName,
		LogoUrl:     page.LogoURL,
		Services:    svcs,
	}
	return &calv1.GetPublicBookingPageResponse{Page: pub}, nil
}

func (s *CalendarGRPCServer) GetAvailability(ctx context.Context, req *calv1.GetAvailabilityRequest) (*calv1.GetAvailabilityResponse, error) {
	if s.bookingService == nil {
		return nil, status.Error(codes.Unimplemented, "booking service not configured")
	}
	// Public route: no JWT, no tenant metadata — RLS requires system context.
	ctx = sysctx.With(ctx)
	from, err := parseBookingDate(req.GetDateFrom())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid date_from: "+err.Error())
	}
	to, err := parseBookingDate(req.GetDateTo())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid date_to: "+err.Error())
	}

	days, err := s.bookingService.GetAvailability(ctx, calendar.GetAvailabilityInput{
		Slug:      req.GetSlug(),
		DateFrom:  from,
		DateTo:    to,
		ServiceID: req.GetServiceId(),
	})
	if err != nil {
		return nil, mapCalendarError(err)
	}

	dayProtos := make([]*calv1.AvailabilityDayProto, 0, len(days))
	for _, d := range days {
		dayProtos = append(dayProtos, &calv1.AvailabilityDayProto{
			Date:  d.Date.Format("2006-01-02"),
			Slots: d.Slots,
		})
	}
	return &calv1.GetAvailabilityResponse{Days: dayProtos}, nil
}

func (s *CalendarGRPCServer) CreatePublicBooking(ctx context.Context, req *calv1.CreatePublicBookingRequest) (*calv1.CreatePublicBookingResponse, error) {
	if s.bookingService == nil {
		return nil, status.Error(codes.Unimplemented, "booking service not configured")
	}
	// Public route: no JWT, no tenant metadata — RLS requires system context.
	ctx = sysctx.With(ctx)

	date, err := parseBookingDate(req.GetDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid date: "+err.Error())
	}

	cust := req.GetCustomer()
	if cust == nil {
		return nil, status.Error(codes.InvalidArgument, "customer is required")
	}

	input := calendar.CreatePublicBookingInput{
		Slug:          req.GetSlug(),
		ServiceID:     req.GetServiceId(),
		Date:          date,
		TimeSlot:      req.GetTimeSlot(),
		CustomerName:  cust.GetName(),
		CustomerEmail: cust.GetEmail(),
	}
	if v := cust.GetPhone(); v != "" {
		input.CustomerPhone = &v
	}
	if v := cust.GetNotes(); v != "" {
		input.Notes = &v
	}

	// Wire email client if available.
	var emailCli calendar.PublicBookingEmailClient
	if s.emailClient != nil {
		emailCli = &grpcEmailClientAdapter{client: s.emailClient}
	}

	// Wire calendar client so a confirmed public booking lands in the internal
	// calendar of the page owner (the published booking page is the authorization).
	var calCli calendar.PublicBookingCalClient
	if s.eventService != nil && s.calendarService != nil {
		calCli = &grpcCalClientAdapter{events: s.eventService, calendars: s.calendarService}
	}

	result, err := s.bookingService.CreatePublicBooking(ctx, emailCli, calCli, input)
	if err != nil {
		return nil, mapCalendarError(err)
	}

	return &calv1.CreatePublicBookingResponse{Booking: publicBookingToProto(result.Booking)}, nil
}

// ============================================================================
// Booking proto <-> model converters
// ============================================================================

func bookingPageToProto(p *models.BookingPage) *calv1.BookingPageProto {
	if p == nil {
		return nil
	}
	rulesJSON, _ := json.Marshal(p.AvailabilityRules)
	svcs := make([]*calv1.BookingServiceProto, 0, len(p.Services))
	for _, s := range p.Services {
		sc := s
		svcs = append(svcs, bookingServiceToProto(&sc))
	}
	return &calv1.BookingPageProto{
		Id:                  p.ID.String(),
		TenantId:            p.TenantID.String(),
		CalendarId:          p.CalendarID.String(),
		Slug:                p.Slug,
		CompanyName:         p.CompanyName,
		LogoUrl:             p.LogoURL,
		Services:            svcs,
		AvailabilityRulesJson: string(rulesJSON),
		Active:              p.Active,
		CreatedAt:           timestamppb.New(p.CreatedAt),
		UpdatedAt:           timestamppb.New(p.UpdatedAt),
	}
}

func bookingServiceToProto(s *models.BookingService) *calv1.BookingServiceProto {
	return &calv1.BookingServiceProto{
		Id:          s.ID,
		Name:        s.Name,
		DurationMin: int32(s.DurationMin),
		Price:       s.Price,
		Description: s.Description,
	}
}

func protoToBookingService(p *calv1.BookingServiceProto) models.BookingService {
	bs := models.BookingService{
		ID:          p.GetId(),
		Name:        p.GetName(),
		DurationMin: int(p.GetDurationMin()),
		Price:       p.GetPrice(),
	}
	if v := p.GetDescription(); v != "" {
		bs.Description = &v
	}
	return bs
}

func publicBookingToProto(b *models.PublicBooking) *calv1.PublicBookingProto {
	if b == nil {
		return nil
	}
	pb := &calv1.PublicBookingProto{
		Id:            b.ID.String(),
		BookingPageId: b.BookingPageID.String(),
		ServiceId:     b.ServiceID,
		CustomerName:  b.CustomerName,
		CustomerEmail: b.CustomerEmail,
		Date:          b.Date.Format("2006-01-02"),
		TimeSlot:      b.TimeSlot,
		Status:        b.Status,
		CreatedAt:     timestamppb.New(b.CreatedAt),
	}
	if b.CustomerPhone != nil {
		pb.CustomerPhone = b.CustomerPhone
	}
	if b.Notes != nil {
		pb.Notes = b.Notes
	}
	return pb
}

// parseBookingDate parses a YYYY-MM-DD string into a time.Time.
func parseBookingDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// grpcEmailClientAdapter wraps the email gRPC client as a PublicBookingEmailClient.
type grpcEmailClientAdapter struct {
	client emailv1.EmailServiceClient
}

func (a *grpcEmailClientAdapter) SendConfirmation(ctx context.Context, email calendar.PublicBookingConfirmationEmail) error {
	body := fmt.Sprintf(
		"Ihre Buchung wurde bestätigt.\n\nService: %s\nDatum: %s\nUhrzeit: %s\nUnternehmen: %s\n",
		email.ServiceName, email.Date, email.TimeSlot, email.CompanyName,
	)
	_, err := a.client.SendEmail(ctx, &emailv1.SendEmailRequest{
		AccountId: "system",
		To: []*emailv1.EmailAddress{
			{Email: email.ToEmail, Name: email.ToName},
		},
		Subject:  fmt.Sprintf("Buchungsbestätigung — %s", email.ServiceName),
		BodyText: body,
		BodyHtml: "<pre>" + body + "</pre>",
	})
	return err
}

// grpcCalClientAdapter wraps the event + calendar services so a confirmed public
// booking creates an internal calendar event. The page owner is used as the event
// actor — the published booking page itself is the authorization, and the public
// caller has no JWT/actor of its own.
type grpcCalClientAdapter struct {
	events    *event.Service
	calendars *calendar.Service
}

func (a *grpcCalClientAdapter) CreateEvent(ctx context.Context, in calendar.PublicBookingCalEventInput) (string, error) {
	calID, err := uuid.Parse(in.CalendarID)
	if err != nil {
		return "", fmt.Errorf("invalid calendar id: %w", err)
	}
	cal, err := a.calendars.Get(ctx, calID, in.TenantID)
	if err != nil {
		return "", err
	}
	desc := in.Description
	evt, err := a.events.Create(ctx, cal.OwnerID, event.CreateInput{
		TenantID:    in.TenantID,
		CalendarID:  calID,
		Title:       in.Title,
		Description: &desc,
		StartTime:   in.StartTime,
		EndTime:     in.EndTime,
	})
	if err != nil {
		return "", err
	}
	return evt.ID.String(), nil
}
