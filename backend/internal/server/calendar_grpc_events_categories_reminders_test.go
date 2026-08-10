package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/calendar"
	"github.com/kmuhub/kmuhub/internal/work/event"
	"github.com/kmuhub/kmuhub/internal/work/livekit"
	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// ============================================================================
// stubEventRepo — implements event.Repository
// ============================================================================

type stubEventRepo struct {
	events     map[uuid.UUID]*models.CalendarEvent
	attendees  map[uuid.UUID]map[uuid.UUID]*models.EventAttendee
	exceptions map[uuid.UUID][]models.EventException
	reminders  map[uuid.UUID][]models.EventReminder
	deadlines  []models.TaskDeadlineStub

	createErr                   error
	getByIDErr                  error
	updateErr                   error
	deleteErr                   error
	listInRangeErr               error
	listRecurringOverlappingErr  error
	addAttendeeErr               error
	removeAttendeeErr            error
	updateRSVPErr                error
	listAttendeesErr             error
	createExceptionErr           error
	listExceptionsErr            error
	setRemindersErr              error
	listRemindersErr             error
	listTaskDeadlinesInRangeErr  error
}

func newStubEventRepo() *stubEventRepo {
	return &stubEventRepo{
		events:     make(map[uuid.UUID]*models.CalendarEvent),
		attendees:  make(map[uuid.UUID]map[uuid.UUID]*models.EventAttendee),
		exceptions: make(map[uuid.UUID][]models.EventException),
		reminders:  make(map[uuid.UUID][]models.EventReminder),
	}
}

func (r *stubEventRepo) Create(_ context.Context, e *models.CalendarEvent) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.events[e.ID] = e
	return nil
}

func (r *stubEventRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.CalendarEvent, error) {
	if r.getByIDErr != nil {
		return nil, r.getByIDErr
	}
	e, ok := r.events[id]
	if !ok || e.TenantID != tenantID {
		return nil, event.ErrEventNotFound
	}
	return e, nil
}

func (r *stubEventRepo) Update(_ context.Context, e *models.CalendarEvent) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.events[e.ID] = e
	return nil
}

func (r *stubEventRepo) Delete(_ context.Context, id, _ uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.events, id)
	return nil
}

func (r *stubEventRepo) ListInRange(_ context.Context, _ []uuid.UUID, _, _ time.Time, _, _ uuid.UUID) ([]models.ExpandedEvent, error) {
	if r.listInRangeErr != nil {
		return nil, r.listInRangeErr
	}
	var out []models.ExpandedEvent
	for _, e := range r.events {
		out = append(out, models.ExpandedEvent{CalendarEvent: *e})
	}
	return out, nil
}

func (r *stubEventRepo) ListRecurringOverlapping(_ context.Context, _ []uuid.UUID, _, _ time.Time, _ uuid.UUID) ([]models.CalendarEvent, error) {
	return nil, r.listRecurringOverlappingErr
}

func (r *stubEventRepo) AddAttendee(_ context.Context, a *models.EventAttendee) error {
	if r.addAttendeeErr != nil {
		return r.addAttendeeErr
	}
	if _, ok := r.attendees[a.EventID]; !ok {
		r.attendees[a.EventID] = make(map[uuid.UUID]*models.EventAttendee)
	}
	r.attendees[a.EventID][a.UserID] = a
	return nil
}

func (r *stubEventRepo) RemoveAttendee(_ context.Context, eventID, userID uuid.UUID) error {
	if r.removeAttendeeErr != nil {
		return r.removeAttendeeErr
	}
	delete(r.attendees[eventID], userID)
	return nil
}

func (r *stubEventRepo) UpdateRSVP(_ context.Context, eventID, userID uuid.UUID, status string) error {
	if r.updateRSVPErr != nil {
		return r.updateRSVPErr
	}
	if a, ok := r.attendees[eventID][userID]; ok {
		a.RSVPStatus = status
	}
	return nil
}

func (r *stubEventRepo) ListAttendees(_ context.Context, eventID uuid.UUID) ([]models.EventAttendee, error) {
	if r.listAttendeesErr != nil {
		return nil, r.listAttendeesErr
	}
	var out []models.EventAttendee
	for _, a := range r.attendees[eventID] {
		out = append(out, *a)
	}
	return out, nil
}

func (r *stubEventRepo) ListAttendeeEventIDs(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]uuid.UUID, error) {
	return nil, nil
}

func (r *stubEventRepo) CreateException(_ context.Context, ex *models.EventException) error {
	if r.createExceptionErr != nil {
		return r.createExceptionErr
	}
	r.exceptions[ex.EventID] = append(r.exceptions[ex.EventID], *ex)
	return nil
}

func (r *stubEventRepo) ListExceptions(_ context.Context, eventID uuid.UUID, _, _ time.Time) ([]models.EventException, error) {
	if r.listExceptionsErr != nil {
		return nil, r.listExceptionsErr
	}
	return r.exceptions[eventID], nil
}

func (r *stubEventRepo) DeleteExceptionsAfterDate(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (r *stubEventRepo) SetReminders(_ context.Context, eventID uuid.UUID, minutesBefore []int) error {
	if r.setRemindersErr != nil {
		return r.setRemindersErr
	}
	var reminders []models.EventReminder
	for _, m := range minutesBefore {
		reminders = append(reminders, models.EventReminder{ID: uuid.New(), EventID: eventID, MinutesBefore: m})
	}
	r.reminders[eventID] = reminders
	return nil
}

func (r *stubEventRepo) ListReminders(_ context.Context, eventID uuid.UUID) ([]models.EventReminder, error) {
	if r.listRemindersErr != nil {
		return nil, r.listRemindersErr
	}
	return r.reminders[eventID], nil
}

func (r *stubEventRepo) ListUpcomingReminders(_ context.Context, _, _ time.Time) ([]event.ReminderWithEvent, error) {
	return nil, nil
}

func (r *stubEventRepo) ListTaskDeadlinesInRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]models.TaskDeadlineStub, error) {
	if r.listTaskDeadlinesInRangeErr != nil {
		return nil, r.listTaskDeadlinesInRangeErr
	}
	return r.deadlines, nil
}

// ============================================================================
// Test server construction
// ============================================================================

func newTestCalendarServerWithEvents(calRepo calendar.Repository, eventRepo event.Repository, lk *livekit.Service) *CalendarGRPCServer {
	return NewCalendarGRPCServer(calendar.NewService(calRepo), event.NewService(eventRepo, calRepo), nil, nil, lk)
}

func disabledLiveKit() *livekit.Service {
	return livekit.NewService("", "", "")
}

func enabledLiveKit() *livekit.Service {
	return livekit.NewService("test-api-key", "test-api-secret", "wss://livekit.example.com")
}

// ============================================================================
// CreateEvent
//
// FUND (nicht Teil dieser Unit, nur dokumentiert): CreateEvent
// (calendar_grpc.go:391-459) setzt an keiner Stelle input.TenantID -- anders
// als GetEvent/ListEventsInRange/UpdateEvent/DeleteEvent/... ruft es
// middleware.GetTenantID(ctx) nicht einmal auf. event.CreateInput.TenantID
// bleibt damit immer die Nullwert-UUID, und event.Service.Create (service.go:77)
// laedt den Kalender ueber s.calendarRepo.GetByID(ctx, input.CalendarID,
// input.TenantID) -- also mit tenant_id = 00000000-...-0000 statt dem echten
// Tenant. Der Postgres-Repository-Query filtert explizit "WHERE id=$1 AND
// tenant_id=$2" (calendar/postgres_repository.go:41), daher kann diese Abfrage
// fuer einen echten (Nicht-Null-)Tenant NIE einen Treffer liefern. Das
// TestCreateEvent/"calendar not found (missing tenant scoping)" unten belegt
// das direkt: derselbe Tenant im ctx und auf dem Kalender-Datensatz fuehrt
// trotzdem zu NotFound. In Produktion heisst das: jeder CreateEvent-Aufruf
// ueber den Gateway (route_calendar.go:649, HandleCreateEvent reicht r.Context()
// durch) scheitert an "calendar not found", solange kein Kalender zufaellig
// die Tenant-ID 00000000-0000-0000-0000-000000000000 traegt.
// ============================================================================

func TestCreateEvent(t *testing.T) {
	t.Run("invalid calendar_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.CreateEvent(context.Background(), &calv1.CreateEventRequest{
			CalendarId: "not-a-uuid",
			CreatedBy:  uuid.New().String(),
			Title:      "Standup",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid created_by", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.CreateEvent(context.Background(), &calv1.CreateEventRequest{
			CalendarId: uuid.New().String(),
			CreatedBy:  "not-a-uuid",
			Title:      "Standup",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid resource_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		resID := "not-a-uuid"
		_, err := srv.CreateEvent(context.Background(), &calv1.CreateEventRequest{
			CalendarId: uuid.New().String(),
			CreatedBy:  uuid.New().String(),
			Title:      "Standup",
			ResourceId: &resID,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("calendar not found (missing tenant scoping)", func(t *testing.T) {
		calRepo := newStubCalendarRepo()
		tenantID := uuid.New()
		actorID := uuid.New()
		calID := uuid.New()
		calRepo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: actorID, CalendarType: models.CalendarTypeShared}
		srv := newTestCalendarServerWithEvents(calRepo, newStubEventRepo(), disabledLiveKit())

		// Same tenant on the calendar and (irrelevantly) in ctx -- CreateEvent
		// never reads it, so the lookup still runs with the zero UUID and fails.
		_, err := srv.CreateEvent(ctxWithTenant(tenantID), &calv1.CreateEventRequest{
			CalendarId: calID.String(),
			CreatedBy:  actorID.String(),
			Title:      "Standup",
			IsAllDay:   true,
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("happy path (documents current zero-tenant handler behavior)", func(t *testing.T) {
		calRepo := newStubCalendarRepo()
		actorID := uuid.New()
		calID := uuid.New()
		attendeeID := uuid.New()
		// TenantID intentionally uuid.Nil: matches the zero value CreateEvent
		// actually passes today (see finding above). A calendar in a real
		// tenant would make this same call fail with NotFound instead.
		calRepo.calendars[calID] = &models.Calendar{ID: calID, TenantID: uuid.Nil, OwnerID: actorID, CalendarType: models.CalendarTypeShared, Timezone: "Europe/Berlin"}
		eventRepo := newStubEventRepo()
		srv := newTestCalendarServerWithEvents(calRepo, eventRepo, disabledLiveKit())

		desc := "Weekly sync"
		resp, err := srv.CreateEvent(context.Background(), &calv1.CreateEventRequest{
			CalendarId:      calID.String(),
			CreatedBy:       actorID.String(),
			Title:           "Standup",
			Description:     &desc,
			IsAllDay:        true,
			HasVideoCall:    true,
			AttendeeUserIds: []string{attendeeID.String()},
		})
		require.NoError(t, err)
		assert.Equal(t, "Standup", resp.Event.Title)
		assert.True(t, resp.Event.HasVideoCall)
		assert.NotNil(t, eventRepo.attendees[uuid.MustParse(resp.Event.Id)][actorID])
		assert.NotNil(t, eventRepo.attendees[uuid.MustParse(resp.Event.Id)][attendeeID])
	})
}

// ============================================================================
// GetEvent
// ============================================================================

func TestGetEvent(t *testing.T) {
	tenantID := uuid.New()
	eventID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.events[eventID] = &models.CalendarEvent{ID: eventID, TenantID: tenantID, Title: "Standup"}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		resp, err := srv.GetEvent(ctxWithTenant(tenantID), &calv1.GetEventRequest{Id: eventID.String()})
		require.NoError(t, err)
		assert.Equal(t, "Standup", resp.Event.Event.Title)
	})

	t.Run("invalid id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.GetEvent(ctxWithTenant(tenantID), &calv1.GetEventRequest{Id: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.GetEvent(ctxWithTenant(tenantID), &calv1.GetEventRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// ListEventsInRange
// ============================================================================

func TestListEventsInRange(t *testing.T) {
	tenantID := uuid.New()
	calID := uuid.New()
	userID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.events[uuid.New()] = &models.CalendarEvent{ID: uuid.New(), TenantID: tenantID, CalendarID: calID, Title: "Standup"}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		resp, err := srv.ListEventsInRange(ctxWithTenant(tenantID), &calv1.ListEventsInRangeRequest{
			CalendarIds: []string{calID.String()},
			UserId:      userID.String(),
		})
		require.NoError(t, err)
		assert.Len(t, resp.Events, 1)
	})

	t.Run("invalid calendar_id in list", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.ListEventsInRange(ctxWithTenant(tenantID), &calv1.ListEventsInRangeRequest{
			CalendarIds: []string{"nope"},
			UserId:      userID.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid user_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.ListEventsInRange(ctxWithTenant(tenantID), &calv1.ListEventsInRangeRequest{
			CalendarIds: []string{calID.String()},
			UserId:      "nope",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("repo error maps to Internal", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.listInRangeErr = assert.AnError
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		_, err := srv.ListEventsInRange(ctxWithTenant(tenantID), &calv1.ListEventsInRangeRequest{
			CalendarIds: []string{calID.String()},
			UserId:      userID.String(),
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// UpdateEvent
// ============================================================================

func TestUpdateEvent(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	eventID := uuid.New()

	t.Run("happy path (creator updates)", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.events[eventID] = &models.CalendarEvent{ID: eventID, TenantID: tenantID, CreatedBy: actorID, Title: "Old"}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())
		newTitle := "New"

		resp, err := srv.UpdateEvent(ctxWithTenant(tenantID), &calv1.UpdateEventRequest{
			Id:        eventID.String(),
			UpdatedBy: actorID.String(),
			Title:     &newTitle,
		})
		require.NoError(t, err)
		assert.Equal(t, "New", resp.Event.Title)
	})

	t.Run("invalid id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.UpdateEvent(ctxWithTenant(tenantID), &calv1.UpdateEventRequest{Id: "nope", UpdatedBy: actorID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.UpdateEvent(ctxWithTenant(tenantID), &calv1.UpdateEventRequest{Id: uuid.New().String(), UpdatedBy: actorID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// DeleteEvent
//
// FUND (nicht Teil dieser Unit, nur dokumentiert): DeleteEvent
// (calendar_grpc.go:598-599) ruft s.eventService.DeleteEvent mit einer fest
// verdrahteten uuid.Nil als actorID auf ("Gateway handles auth"-Kommentar).
// event.Service.DeleteEvent (service.go:398-400) lehnt aber JEDEN Aufruf ab,
// dessen evt.CreatedBy != actorID ist -- und CreatedBy ist bei einem echten
// Event nie die Nullwert-UUID. middleware.GetUserID(ctx) liefert den echten
// Actor bereits serverseitig (middleware/auth.go:66, vom gRPC-Tenant-
// Interceptor gesetzt, middleware/grpc_tenant.go:83) und wird hier schlicht
// nicht gelesen. TestDeleteEvent/"real creator denied (uuid.Nil actor bug)"
// belegt das direkt. Derselbe Fehlklassen-Fund gilt fuer SetEventReminders
// (:820-821) und DeleteEventCategory (:791-792) unten, sowie fuer
// UpdateCalendar (:180), DeleteCalendar (:202), AddCalendarMember (:230),
// RemoveCalendarMember (:254) und UpdateCalendarMemberPermission (:304) --
// deren Tests in calendar_grpc_errormap_calendars_members_test.go (Iteration 17)
// setzen durchweg OwnerID: uuid.Nil auf den Test-Kalendern, um den Happy Path
// ueberhaupt gruen zu bekommen; das ist ein Workaround um genau diesen Bug,
// kein Beleg, dass die RPCs fuer echte Owner funktionieren.
// ============================================================================

func TestDeleteEvent(t *testing.T) {
	tenantID := uuid.New()
	eventID := uuid.New()

	t.Run("real creator denied (uuid.Nil actor bug)", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.events[eventID] = &models.CalendarEvent{ID: eventID, TenantID: tenantID, CreatedBy: uuid.New()}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		_, err := srv.DeleteEvent(ctxWithTenant(tenantID), &calv1.DeleteEventRequest{Id: eventID.String()})
		requireGRPCCode(t, err, codes.PermissionDenied)
		_, stillExists := eventRepo.events[eventID]
		assert.True(t, stillExists)
	})

	t.Run("invalid id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.DeleteEvent(ctxWithTenant(tenantID), &calv1.DeleteEventRequest{Id: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path (documents current uuid.Nil-creator-only behavior)", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		// CreatedBy intentionally uuid.Nil to match the actorID DeleteEvent
		// actually passes today (see finding above).
		eventRepo.events[eventID] = &models.CalendarEvent{ID: eventID, TenantID: tenantID, CreatedBy: uuid.Nil}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		_, err := srv.DeleteEvent(ctxWithTenant(tenantID), &calv1.DeleteEventRequest{Id: eventID.String()})
		require.NoError(t, err)
		_, stillExists := eventRepo.events[eventID]
		assert.False(t, stillExists)
	})
}

// ============================================================================
// UpdateRecurringEvent
// ============================================================================

func TestUpdateRecurringEvent(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	eventID := uuid.New()
	rrule := "FREQ=WEEKLY;BYDAY=MO"

	t.Run("happy path (THIS_EVENT scope)", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		start := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
		eventRepo.events[eventID] = &models.CalendarEvent{
			ID: eventID, TenantID: tenantID, CreatedBy: actorID, Title: "Standup",
			RRule: &rrule, StartTime: start, EndTime: start.Add(time.Hour),
		}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())
		newTitle := "Standup (moved)"

		resp, err := srv.UpdateRecurringEvent(ctxWithTenant(tenantID), &calv1.UpdateRecurringEventRequest{
			EventId:      eventID.String(),
			UpdatedBy:    actorID.String(),
			OriginalDate: "2026-01-12",
			Scope:        calv1.RecurringEventScope_THIS_EVENT,
			Title:        &newTitle,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Event)
		assert.Len(t, eventRepo.exceptions[eventID], 1)
	})

	t.Run("event not recurring (scope error path)", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.events[eventID] = &models.CalendarEvent{ID: eventID, TenantID: tenantID, CreatedBy: actorID, Title: "One-off"}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		_, err := srv.UpdateRecurringEvent(ctxWithTenant(tenantID), &calv1.UpdateRecurringEventRequest{
			EventId:      eventID.String(),
			UpdatedBy:    actorID.String(),
			OriginalDate: "2026-01-12",
			Scope:        calv1.RecurringEventScope_ALL_EVENTS,
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("invalid original_date format", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.UpdateRecurringEvent(ctxWithTenant(tenantID), &calv1.UpdateRecurringEventRequest{
			EventId:      eventID.String(),
			UpdatedBy:    actorID.String(),
			OriginalDate: "12.01.2026",
			Scope:        calv1.RecurringEventScope_THIS_EVENT,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid event_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.UpdateRecurringEvent(ctxWithTenant(tenantID), &calv1.UpdateRecurringEventRequest{
			EventId:      "nope",
			UpdatedBy:    actorID.String(),
			OriginalDate: "2026-01-12",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// ============================================================================
// RSVPToEvent
// ============================================================================

func TestRSVPToEvent(t *testing.T) {
	tenantID := uuid.New()
	eventID := uuid.New()
	userID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.events[eventID] = &models.CalendarEvent{ID: eventID, TenantID: tenantID, CreatedBy: uuid.New()}
		eventRepo.attendees[eventID] = map[uuid.UUID]*models.EventAttendee{
			userID: {EventID: eventID, UserID: userID, RSVPStatus: models.RSVPPending},
		}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		_, err := srv.RSVPToEvent(ctxWithTenant(tenantID), &calv1.RSVPToEventRequest{
			EventId:    eventID.String(),
			UserId:     userID.String(),
			RsvpStatus: models.RSVPAccepted,
		})
		require.NoError(t, err)
		assert.Equal(t, models.RSVPAccepted, eventRepo.attendees[eventID][userID].RSVPStatus)
	})

	t.Run("invalid rsvp_status", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.events[eventID] = &models.CalendarEvent{ID: eventID, TenantID: tenantID}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		_, err := srv.RSVPToEvent(ctxWithTenant(tenantID), &calv1.RSVPToEventRequest{
			EventId:    eventID.String(),
			UserId:     userID.String(),
			RsvpStatus: "maybe-later",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid event_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.RSVPToEvent(ctxWithTenant(tenantID), &calv1.RSVPToEventRequest{
			EventId:    "nope",
			UserId:     userID.String(),
			RsvpStatus: models.RSVPAccepted,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// ============================================================================
// ListEventAttendees
// ============================================================================

func TestListEventAttendees(t *testing.T) {
	tenantID := uuid.New()
	eventID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.events[eventID] = &models.CalendarEvent{ID: eventID, TenantID: tenantID}
		userID := uuid.New()
		eventRepo.attendees[eventID] = map[uuid.UUID]*models.EventAttendee{
			userID: {EventID: eventID, UserID: userID, RSVPStatus: models.RSVPAccepted},
		}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		resp, err := srv.ListEventAttendees(ctxWithTenant(tenantID), &calv1.ListEventAttendeesRequest{EventId: eventID.String()})
		require.NoError(t, err)
		assert.Len(t, resp.Attendees, 1)
	})

	t.Run("invalid event_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.ListEventAttendees(ctxWithTenant(tenantID), &calv1.ListEventAttendeesRequest{EventId: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("event not found", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.ListEventAttendees(ctxWithTenant(tenantID), &calv1.ListEventAttendeesRequest{EventId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// CreateEventCategory
// ============================================================================

func TestCreateEventCategory(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		calRepo := newStubCalendarRepo()
		srv := newTestCalendarServerWithEvents(calRepo, newStubEventRepo(), disabledLiveKit())

		resp, err := srv.CreateEventCategory(ctxWithTenant(tenantID), &calv1.CreateEventCategoryRequest{
			UserId: userID.String(),
			Name:   "Client Work",
			Color:  "#FF0000",
		})
		require.NoError(t, err)
		assert.Equal(t, "Client Work", resp.Category.Name)
		assert.Len(t, calRepo.categories, 1)
	})

	t.Run("invalid user_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.CreateEventCategory(ctxWithTenant(tenantID), &calv1.CreateEventCategoryRequest{
			UserId: "nope",
			Name:   "Client Work",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("name required", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.CreateEventCategory(ctxWithTenant(tenantID), &calv1.CreateEventCategoryRequest{
			UserId: userID.String(),
			Name:   "   ",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// ============================================================================
// ListEventCategories
// ============================================================================

func TestListEventCategories(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		calRepo := newStubCalendarRepo()
		catID := uuid.New()
		calRepo.categories[catID] = &models.EventCategory{ID: catID, TenantID: tenantID, UserID: userID, Name: "Client Work"}
		srv := newTestCalendarServerWithEvents(calRepo, newStubEventRepo(), disabledLiveKit())

		resp, err := srv.ListEventCategories(ctxWithTenant(tenantID), &calv1.ListEventCategoriesRequest{UserId: userID.String()})
		require.NoError(t, err)
		assert.Len(t, resp.Categories, 1)
	})

	t.Run("invalid user_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.ListEventCategories(ctxWithTenant(tenantID), &calv1.ListEventCategoriesRequest{UserId: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// ============================================================================
// DeleteEventCategory
//
// Siehe Fund oberhalb TestDeleteEvent -- derselbe uuid.Nil-Actor-Bug, hier
// gegen die DELETE ... WHERE user_id=$2-Semantik der echten Postgres-
// Implementierung nachgebildet (postgres_repository.go:337-349).
// ============================================================================

func TestDeleteEventCategory(t *testing.T) {
	tenantID := uuid.New()
	catID := uuid.New()

	t.Run("real owner denied (uuid.Nil actor bug)", func(t *testing.T) {
		calRepo := newStubCalendarRepo()
		calRepo.categories[catID] = &models.EventCategory{ID: catID, TenantID: tenantID, UserID: uuid.New(), Name: "Client Work"}
		srv := newTestCalendarServerWithEvents(calRepo, newStubEventRepo(), disabledLiveKit())

		_, err := srv.DeleteEventCategory(ctxWithTenant(tenantID), &calv1.DeleteEventCategoryRequest{Id: catID.String()})
		requireGRPCCode(t, err, codes.NotFound)
		_, stillExists := calRepo.categories[catID]
		assert.True(t, stillExists)
	})

	t.Run("invalid id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.DeleteEventCategory(ctxWithTenant(tenantID), &calv1.DeleteEventCategoryRequest{Id: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path (documents current uuid.Nil-owner-only behavior)", func(t *testing.T) {
		calRepo := newStubCalendarRepo()
		calRepo.categories[catID] = &models.EventCategory{ID: catID, TenantID: tenantID, UserID: uuid.Nil, Name: "Client Work"}
		srv := newTestCalendarServerWithEvents(calRepo, newStubEventRepo(), disabledLiveKit())

		_, err := srv.DeleteEventCategory(ctxWithTenant(tenantID), &calv1.DeleteEventCategoryRequest{Id: catID.String()})
		require.NoError(t, err)
		_, stillExists := calRepo.categories[catID]
		assert.False(t, stillExists)
	})
}

// ============================================================================
// SetEventReminders
//
// Siehe Fund oberhalb TestDeleteEvent -- derselbe uuid.Nil-Actor-Bug
// (calendar_grpc.go:820-821), hier gegen die Limit-/Minuten-Validierung
// abgegrenzt, die in event.Service.SetReminders VOR dem Ownership-Check
// laeuft (service.go:487-495) und deshalb unabhaengig vom Bug testbar ist.
// ============================================================================

func TestSetEventReminders(t *testing.T) {
	tenantID := uuid.New()
	eventID := uuid.New()

	t.Run("real creator denied (uuid.Nil actor bug)", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.events[eventID] = &models.CalendarEvent{ID: eventID, TenantID: tenantID, CreatedBy: uuid.New(), CalendarID: uuid.New()}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		_, err := srv.SetEventReminders(ctxWithTenant(tenantID), &calv1.SetEventRemindersRequest{
			EventId:       eventID.String(),
			MinutesBefore: []int32{15},
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})

	t.Run("reminder limit exceeded", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.SetEventReminders(ctxWithTenant(tenantID), &calv1.SetEventRemindersRequest{
			EventId:       eventID.String(),
			MinutesBefore: []int32{5, 10, 15, 30},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid reminder minutes", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.SetEventReminders(ctxWithTenant(tenantID), &calv1.SetEventRemindersRequest{
			EventId:       eventID.String(),
			MinutesBefore: []int32{0},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path (documents current uuid.Nil-creator-only behavior)", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.events[eventID] = &models.CalendarEvent{ID: eventID, TenantID: tenantID, CreatedBy: uuid.Nil, CalendarID: uuid.New()}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		_, err := srv.SetEventReminders(ctxWithTenant(tenantID), &calv1.SetEventRemindersRequest{
			EventId:       eventID.String(),
			MinutesBefore: []int32{15, 30},
		})
		require.NoError(t, err)
		assert.Len(t, eventRepo.reminders[eventID], 2)
	})
}

// ============================================================================
// ListEventReminders
// ============================================================================

func TestListEventReminders(t *testing.T) {
	eventID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.reminders[eventID] = []models.EventReminder{{ID: uuid.New(), EventID: eventID, MinutesBefore: 15}}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		resp, err := srv.ListEventReminders(context.Background(), &calv1.ListEventRemindersRequest{EventId: eventID.String()})
		require.NoError(t, err)
		assert.Len(t, resp.Reminders, 1)
	})

	t.Run("invalid event_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.ListEventReminders(context.Background(), &calv1.ListEventRemindersRequest{EventId: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("repo error maps to Internal", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.listRemindersErr = assert.AnError
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		_, err := srv.ListEventReminders(context.Background(), &calv1.ListEventRemindersRequest{EventId: eventID.String()})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// ListTaskDeadlinesInRange
// ============================================================================

func TestListTaskDeadlinesInRange(t *testing.T) {
	userID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.deadlines = []models.TaskDeadlineStub{{TaskID: uuid.New(), Title: "Ship it", DueDate: time.Now(), Priority: "high"}}
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		resp, err := srv.ListTaskDeadlinesInRange(context.Background(), &calv1.ListTaskDeadlinesInRangeRequest{UserId: userID.String()})
		require.NoError(t, err)
		assert.Len(t, resp.Deadlines, 1)
	})

	t.Run("invalid user_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.ListTaskDeadlinesInRange(context.Background(), &calv1.ListTaskDeadlinesInRangeRequest{UserId: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("repo error maps to Internal", func(t *testing.T) {
		eventRepo := newStubEventRepo()
		eventRepo.listTaskDeadlinesInRangeErr = assert.AnError
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), eventRepo, disabledLiveKit())

		_, err := srv.ListTaskDeadlinesInRange(context.Background(), &calv1.ListTaskDeadlinesInRangeRequest{UserId: userID.String()})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// GenerateJoinToken
// ============================================================================

func TestGenerateJoinToken(t *testing.T) {
	eventID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), enabledLiveKit())
		resp, err := srv.GenerateJoinToken(context.Background(), &calv1.GenerateJoinTokenRequest{
			EventId:  eventID.String(),
			UserId:   uuid.New().String(),
			UserName: "Jane Doe",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		assert.NotEmpty(t, resp.RoomName)
		assert.NotEmpty(t, resp.WsUrl)
	})

	t.Run("livekit not configured", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), disabledLiveKit())
		_, err := srv.GenerateJoinToken(context.Background(), &calv1.GenerateJoinTokenRequest{
			EventId:  eventID.String(),
			UserId:   uuid.New().String(),
			UserName: "Jane Doe",
		})
		requireGRPCCode(t, err, codes.Unavailable)
	})

	t.Run("invalid event_id", func(t *testing.T) {
		srv := newTestCalendarServerWithEvents(newStubCalendarRepo(), newStubEventRepo(), enabledLiveKit())
		_, err := srv.GenerateJoinToken(context.Background(), &calv1.GenerateJoinTokenRequest{
			EventId:  "nope",
			UserId:   uuid.New().String(),
			UserName: "Jane Doe",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}
