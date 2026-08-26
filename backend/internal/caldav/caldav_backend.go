package caldav

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/gateway"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
	calendarv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// Compile-time interface compliance check.
var _ caldav.Backend = (*CalDAVBackend)(nil)

// caldavUserKey is the context key for the authenticated CalDAV user.
type caldavUserKey struct{}

// UserFromCtx extracts the authenticated user ID from the context.
// The user ID is injected by the Basic Auth middleware.
func UserFromCtx(ctx context.Context) uuid.UUID {
	if v, ok := ctx.Value(caldavUserKey{}).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

// CtxWithUser returns a context with the authenticated user ID set.
func CtxWithUser(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, caldavUserKey{}, userID)
}

// resolveTenantID looks up the tenant owning userID. CalDAV/CardDAV requests
// authenticate via Basic Auth (app-specific password) and never carry a JWT,
// so the request context has no tenant set — this is the one legitimate
// lookup that must cross tenants to find out which one the user belongs to.
func resolveTenantID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (uuid.UUID, error) {
	var tenantID uuid.UUID
	err := pool.QueryRow(sysctx.With(ctx),
		`SELECT tenant_id FROM users WHERE id = $1`,
		userID,
	).Scan(&tenantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve tenant for caldav user: %w", err)
	}
	return tenantID, nil
}

// CalDAVBackend implements the caldav.Backend interface from go-webdav,
// backed by KMU Hub's calendar gRPC service.
type CalDAVBackend struct {
	registry    *gateway.ServiceRegistry
	syncService *SyncTokenService
	pool        *pgxpool.Pool
}

// NewCalDAVBackend creates a new CalDAV backend adapter.
func NewCalDAVBackend(registry *gateway.ServiceRegistry, syncService *SyncTokenService, pool *pgxpool.Pool) *CalDAVBackend {
	return &CalDAVBackend{
		registry:    registry,
		syncService: syncService,
		pool:        pool,
	}
}

// calendarClient returns a CalendarServiceClient from the service registry.
func (b *CalDAVBackend) calendarClient() (calendarv1.CalendarServiceClient, error) {
	conn, err := b.registry.GetConnection("work")
	if err != nil {
		return nil, fmt.Errorf("get calendar gRPC connection: %w", err)
	}
	return calendarv1.NewCalendarServiceClient(conn), nil
}

// CurrentUserPrincipal returns the principal URL for the authenticated user.
func (b *CalDAVBackend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	userID := UserFromCtx(ctx)
	if userID == uuid.Nil {
		return "", webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	return fmt.Sprintf("/caldav/principals/%s/", userID.String()), nil
}

// CalendarHomeSetPath returns the calendar home set path for the authenticated user.
func (b *CalDAVBackend) CalendarHomeSetPath(ctx context.Context) (string, error) {
	userID := UserFromCtx(ctx)
	if userID == uuid.Nil {
		return "", webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	return fmt.Sprintf("/caldav/principals/%s/calendars/", userID.String()), nil
}

// CreateCalendar creates a new calendar. Not currently supported via CalDAV --
// calendars are managed through the web UI.
func (b *CalDAVBackend) CreateCalendar(ctx context.Context, calendar *caldav.Calendar) error {
	return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("calendar creation via CalDAV is not supported"))
}

// ListCalendars returns all calendars the authenticated user has access to.
func (b *CalDAVBackend) ListCalendars(ctx context.Context) ([]caldav.Calendar, error) {
	userID := UserFromCtx(ctx)
	client, err := b.calendarClient()
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	resp, err := client.ListCalendars(ctx, &calendarv1.ListCalendarsRequest{
		UserId:        userID.String(),
		IncludeHidden: false,
	})
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	calendars := make([]caldav.Calendar, 0, len(resp.Calendars))
	for _, cwm := range resp.Calendars {
		cal := cwm.Calendar
		calID := cal.Id
		path := fmt.Sprintf("/caldav/principals/%s/calendars/%s/", userID.String(), calID)

		desc := ""
		if cal.Description != nil {
			desc = *cal.Description
		}

		calendars = append(calendars, caldav.Calendar{
			Path:                  path,
			Name:                  cal.Name,
			Description:           desc,
			SupportedComponentSet: []string{"VEVENT"},
		})
	}

	return calendars, nil
}

// GetCalendar returns a specific calendar by path.
func (b *CalDAVBackend) GetCalendar(ctx context.Context, path string) (*caldav.Calendar, error) {
	userID := UserFromCtx(ctx)
	calID, err := calendarIDFromPath(path)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	client, err := b.calendarClient()
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	resp, err := client.GetCalendar(ctx, &calendarv1.GetCalendarRequest{
		Id: calID.String(),
	})
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	cal := resp.Calendar
	desc := ""
	if cal.Description != nil {
		desc = *cal.Description
	}

	return &caldav.Calendar{
		Path:                  fmt.Sprintf("/caldav/principals/%s/calendars/%s/", userID.String(), cal.Id),
		Name:                  cal.Name,
		Description:           desc,
		SupportedComponentSet: []string{"VEVENT"},
	}, nil
}

// GetCalendarObject returns a single calendar object (event) by path.
func (b *CalDAVBackend) GetCalendarObject(ctx context.Context, path string, req *caldav.CalendarCompRequest) (*caldav.CalendarObject, error) {
	userID := UserFromCtx(ctx)
	eventUID, err := eventUIDFromPath(path)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	client, err := b.calendarClient()
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	// Get the event via gRPC
	resp, err := client.GetEvent(ctx, &calendarv1.GetEventRequest{
		Id:     eventUID,
		UserId: userID.String(),
	})
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	evt := resp.Event.Event

	// Get attendees
	attendeeResp, err := client.ListEventAttendees(ctx, &calendarv1.ListEventAttendeesRequest{
		EventId: evt.Id,
	})
	if err != nil {
		slog.Warn("failed to list event attendees for CalDAV",
			"event_id", evt.Id,
			"error", err,
		)
	}

	// Get exceptions from database directly (no gRPC for this)
	exceptions, err := b.listEventExceptions(ctx, uuid.MustParse(evt.Id))
	if err != nil {
		slog.Warn("failed to list event exceptions for CalDAV",
			"event_id", evt.Id,
			"error", err,
		)
	}

	// Convert to models
	modelEvent := protoEventToModel(evt)
	modelAttendees := protoAttendeesToModels(attendeeResp)

	// Convert to iCal
	icalCal, err := EventToICal(modelEvent, exceptions, modelAttendees)
	if err != nil {
		return nil, fmt.Errorf("convert event to iCal: %w", err)
	}

	calID, _ := calendarIDFromPath(path)
	return &caldav.CalendarObject{
		Path:    fmt.Sprintf("/caldav/principals/%s/calendars/%s/%s.ics", userID.String(), calID.String(), evt.Id),
		ModTime: evt.UpdatedAt.AsTime(),
		ETag:    GenerateETag(uuid.MustParse(evt.Id), evt.UpdatedAt.AsTime()),
		Data:    icalCal,
	}, nil
}

// ListCalendarObjects returns all calendar objects in a calendar.
func (b *CalDAVBackend) ListCalendarObjects(ctx context.Context, path string, req *caldav.CalendarCompRequest) ([]caldav.CalendarObject, error) {
	userID := UserFromCtx(ctx)
	calID, err := calendarIDFromPath(path)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	client, err := b.calendarClient()
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	// Use a wide range: 2 years back, 2 years forward
	now := time.Now()
	start := now.AddDate(-2, 0, 0)
	end := now.AddDate(2, 0, 0)

	resp, err := client.ListEventsInRange(ctx, &calendarv1.ListEventsInRangeRequest{
		CalendarIds: []string{calID.String()},
		Start:       timestamppb.New(start),
		End:         timestamppb.New(end),
		UserId:      userID.String(),
	})
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	return b.expandedEventsToObjects(ctx, userID, calID, resp.Events)
}

// QueryCalendarObjects returns calendar objects matching a query with time-range filter.
func (b *CalDAVBackend) QueryCalendarObjects(ctx context.Context, path string, query *caldav.CalendarQuery) ([]caldav.CalendarObject, error) {
	userID := UserFromCtx(ctx)
	calID, err := calendarIDFromPath(path)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	client, err := b.calendarClient()
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	// Extract time range from query filter
	start, end := queryTimeRange(query)

	resp, err := client.ListEventsInRange(ctx, &calendarv1.ListEventsInRangeRequest{
		CalendarIds: []string{calID.String()},
		Start:       timestamppb.New(start),
		End:         timestamppb.New(end),
		UserId:      userID.String(),
	})
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	return b.expandedEventsToObjects(ctx, userID, calID, resp.Events)
}

// queryTimeRange extracts the time range to search from a CalDAV REPORT
// query. A nil query (or a query with no time-range filter set) falls back
// to a wide 2-year-back/2-year-forward window. A top-level VCALENDAR filter
// range applies first; a nested VEVENT filter range, if present, overrides
// it -- most real-world clients (Apple Calendar, Thunderbird/Lightning) send
// the range on the VEVENT filter, not the VCALENDAR one.
func queryTimeRange(query *caldav.CalendarQuery) (start, end time.Time) {
	start = time.Now().AddDate(-2, 0, 0)
	end = time.Now().AddDate(2, 0, 0)
	if query == nil {
		return start, end
	}

	if !query.CompFilter.Start.IsZero() {
		start = query.CompFilter.Start
	}
	if !query.CompFilter.End.IsZero() {
		end = query.CompFilter.End
	}
	for _, cf := range query.CompFilter.Comps {
		if cf.Name == ical.CompEvent {
			if !cf.Start.IsZero() {
				start = cf.Start
			}
			if !cf.End.IsZero() {
				end = cf.End
			}
		}
	}
	return start, end
}

// PutCalendarObject creates or updates a calendar object.
func (b *CalDAVBackend) PutCalendarObject(ctx context.Context, path string, cal *ical.Calendar, opts *caldav.PutCalendarObjectOptions) (*caldav.CalendarObject, error) {
	userID := UserFromCtx(ctx)
	calID, err := calendarIDFromPath(path)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	// Check write permissions
	if err := b.checkCalendarWritePermission(ctx, userID, calID); err != nil {
		return nil, err
	}

	eventUID, err := eventUIDFromPath(path)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	// Parse incoming iCalendar data
	input, exceptions, err := ICalToEventInput(cal)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	client, err := b.calendarClient()
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	// Try to get existing event to decide create vs update
	var eventID string
	existingResp, err := client.GetEvent(ctx, &calendarv1.GetEventRequest{
		Id:     eventUID,
		UserId: userID.String(),
	})
	if err == nil && existingResp.Event != nil {
		// Update existing event
		eventID = existingResp.Event.Event.Id
		updateReq := &calendarv1.UpdateEventRequest{
			Id:        eventID,
			Title:     &input.Title,
			UpdatedBy: userID.String(),
		}
		if input.Description != nil {
			updateReq.Description = input.Description
		}
		if input.Location != nil {
			updateReq.Location = input.Location
		}
		updateReq.StartTime = timestamppb.New(input.StartTime)
		updateReq.EndTime = timestamppb.New(input.EndTime)
		isAllDay := input.IsAllDay
		updateReq.IsAllDay = &isAllDay
		if input.Timezone != "" {
			updateReq.Timezone = &input.Timezone
		}
		if input.RRule != nil {
			updateReq.Rrule = input.RRule
		}

		_, err = client.UpdateEvent(ctx, updateReq)
		if err != nil {
			return nil, grpcToWebDAVError(err)
		}

		// Handle exceptions via UpdateRecurringEvent
		for _, exc := range exceptions {
			if exc.IsCancelled {
				// Delete this occurrence
				_, updateErr := client.UpdateRecurringEvent(ctx, &calendarv1.UpdateRecurringEventRequest{
					EventId:      eventID,
					Scope:        calendarv1.RecurringEventScope_THIS_EVENT,
					OriginalDate: exc.OriginalDate.Format("2006-01-02"),
					UpdatedBy:    userID.String(),
				})
				if updateErr != nil {
					slog.Warn("failed to cancel recurring occurrence via CalDAV",
						"event_id", eventID,
						"original_date", exc.OriginalDate,
						"error", updateErr,
					)
				}
			} else if exc.Title != nil || exc.StartTime != nil || exc.EndTime != nil {
				req := &calendarv1.UpdateRecurringEventRequest{
					EventId:      eventID,
					Scope:        calendarv1.RecurringEventScope_THIS_EVENT,
					OriginalDate: exc.OriginalDate.Format("2006-01-02"),
					UpdatedBy:    userID.String(),
				}
				if exc.Title != nil {
					req.Title = exc.Title
				}
				if exc.StartTime != nil {
					req.StartTime = timestamppb.New(*exc.StartTime)
				}
				if exc.EndTime != nil {
					req.EndTime = timestamppb.New(*exc.EndTime)
				}
				_, updateErr := client.UpdateRecurringEvent(ctx, req)
				if updateErr != nil {
					slog.Warn("failed to update recurring occurrence via CalDAV",
						"event_id", eventID,
						"original_date", exc.OriginalDate,
						"error", updateErr,
					)
				}
			}
		}
	} else {
		// Create new event
		createReq := &calendarv1.CreateEventRequest{
			CalendarId: calID.String(),
			Title:      input.Title,
			StartTime:  timestamppb.New(input.StartTime),
			EndTime:    timestamppb.New(input.EndTime),
			IsAllDay:   input.IsAllDay,
			CreatedBy:  userID.String(),
		}
		if input.Description != nil {
			createReq.Description = input.Description
		}
		if input.Location != nil {
			createReq.Location = input.Location
		}
		if input.Timezone != "" {
			createReq.Timezone = &input.Timezone
		}
		if input.RRule != nil {
			createReq.Rrule = input.RRule
		}

		createResp, createErr := client.CreateEvent(ctx, createReq)
		if createErr != nil {
			return nil, grpcToWebDAVError(createErr)
		}
		eventID = createResp.Event.Id
	}

	// Increment sync token
	if tenantID, tErr := resolveTenantID(ctx, b.pool, userID); tErr != nil {
		slog.Warn("failed to resolve tenant for sync token after CalDAV PUT",
			"calendar_id", calID,
			"error", tErr,
		)
	} else if syncErr := b.syncService.IncrementAndLog(ctx, tenantID, "calendar", calID, path, "modified"); syncErr != nil {
		slog.Warn("failed to increment sync token after CalDAV PUT",
			"calendar_id", calID,
			"error", syncErr,
		)
	}

	// Return the updated/created object
	now := time.Now()
	return &caldav.CalendarObject{
		Path:    fmt.Sprintf("/caldav/principals/%s/calendars/%s/%s.ics", userID.String(), calID.String(), eventID),
		ModTime: now,
		ETag:    GenerateETag(uuid.MustParse(eventID), now),
		Data:    cal,
	}, nil
}

// DeleteCalendarObject deletes a calendar object (event).
func (b *CalDAVBackend) DeleteCalendarObject(ctx context.Context, path string) error {
	userID := UserFromCtx(ctx)
	calID, err := calendarIDFromPath(path)
	if err != nil {
		return webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	// Check write permissions
	if err := b.checkCalendarWritePermission(ctx, userID, calID); err != nil {
		return err
	}

	eventUID, err := eventUIDFromPath(path)
	if err != nil {
		return webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	client, err := b.calendarClient()
	if err != nil {
		return grpcToWebDAVError(err)
	}

	_, err = client.DeleteEvent(ctx, &calendarv1.DeleteEventRequest{
		Id: eventUID,
	})
	if err != nil {
		return grpcToWebDAVError(err)
	}

	// Increment sync token
	if tenantID, tErr := resolveTenantID(ctx, b.pool, userID); tErr != nil {
		slog.Warn("failed to resolve tenant for sync token after CalDAV DELETE",
			"calendar_id", calID,
			"error", tErr,
		)
	} else if syncErr := b.syncService.IncrementAndLog(ctx, tenantID, "calendar", calID, path, "deleted"); syncErr != nil {
		slog.Warn("failed to increment sync token after CalDAV DELETE",
			"calendar_id", calID,
			"error", syncErr,
		)
	}

	return nil
}

// expandedEventsToObjects converts a list of ExpandedEventProto to CalendarObjects.
// Deduplicates by event ID (expanded events may include recurring instances).
func (b *CalDAVBackend) expandedEventsToObjects(ctx context.Context, userID, calID uuid.UUID, events []*calendarv1.ExpandedEventProto) ([]caldav.CalendarObject, error) {
	client, err := b.calendarClient()
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	// Deduplicate by master event ID
	seen := make(map[string]bool)
	var objects []caldav.CalendarObject

	for _, expanded := range events {
		evt := expanded.Event
		masterID := evt.Id
		if expanded.OriginalEventId != nil {
			masterID = *expanded.OriginalEventId
		}

		if seen[masterID] {
			continue
		}
		seen[masterID] = true

		// Get attendees for this event
		var modelAttendees []models.EventAttendee
		attendeeResp, attendeeErr := client.ListEventAttendees(ctx, &calendarv1.ListEventAttendeesRequest{
			EventId: masterID,
		})
		if attendeeErr == nil {
			modelAttendees = protoAttendeesToModels(attendeeResp)
		}

		// Get exceptions from database
		exceptions, _ := b.listEventExceptions(ctx, uuid.MustParse(masterID))

		// If this is a recurring instance, get the master event
		eventProto := evt
		if expanded.OriginalEventId != nil {
			masterResp, masterErr := client.GetEvent(ctx, &calendarv1.GetEventRequest{
				Id:     masterID,
				UserId: userID.String(),
			})
			if masterErr == nil && masterResp.Event != nil {
				eventProto = masterResp.Event.Event
			}
		}

		modelEvent := protoEventToModel(eventProto)

		icalCal, convertErr := EventToICal(modelEvent, exceptions, modelAttendees)
		if convertErr != nil {
			slog.Warn("failed to convert event to iCal",
				"event_id", masterID,
				"error", convertErr,
			)
			continue
		}

		objects = append(objects, caldav.CalendarObject{
			Path:    fmt.Sprintf("/caldav/principals/%s/calendars/%s/%s.ics", userID.String(), calID.String(), masterID),
			ModTime: eventProto.UpdatedAt.AsTime(),
			ETag:    GenerateETag(uuid.MustParse(masterID), eventProto.UpdatedAt.AsTime()),
			Data:    icalCal,
		})
	}

	if objects == nil {
		objects = []caldav.CalendarObject{}
	}
	return objects, nil
}

// checkCalendarWritePermission checks if the user has write access to a calendar.
// Personal calendars are writable by the owner. Team/shared calendars require
// admin or manager role (edit or admin permission).
func (b *CalDAVBackend) checkCalendarWritePermission(ctx context.Context, userID, calendarID uuid.UUID) error {
	// Check the calendar_members table for permission level
	var permission string
	var ownerID string
	err := b.pool.QueryRow(ctx,
		`SELECT c.owner_id, COALESCE(cm.permission, '')
		 FROM calendars c
		 LEFT JOIN calendar_members cm ON cm.calendar_id = c.id AND cm.user_id = $2
		 WHERE c.id = $1`,
		calendarID, userID,
	).Scan(&ownerID, &permission)
	if err != nil {
		return webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar not found"))
	}

	// Owner always has write access
	if ownerID == userID.String() {
		return nil
	}

	// Members need edit or admin permission
	if permission == "edit" || permission == "admin" {
		return nil
	}

	return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("insufficient permissions to write to this calendar"))
}

// listEventExceptions queries event exceptions directly from the database.
func (b *CalDAVBackend) listEventExceptions(ctx context.Context, eventID uuid.UUID) ([]models.EventException, error) {
	rows, err := b.pool.Query(ctx,
		`SELECT id, event_id, original_date, is_cancelled, title, description, location,
		        start_time, end_time, resource_id, created_at, updated_at
		 FROM event_exceptions
		 WHERE event_id = $1
		 ORDER BY original_date`,
		eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("query event exceptions: %w", err)
	}
	defer rows.Close()

	var exceptions []models.EventException
	for rows.Next() {
		var exc models.EventException
		if err := rows.Scan(
			&exc.ID, &exc.EventID, &exc.OriginalDate, &exc.IsCancelled,
			&exc.Title, &exc.Description, &exc.Location,
			&exc.StartTime, &exc.EndTime, &exc.ResourceID,
			&exc.CreatedAt, &exc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event exception: %w", err)
		}
		exceptions = append(exceptions, exc)
	}
	if exceptions == nil {
		exceptions = []models.EventException{}
	}
	return exceptions, nil
}

// Path parsing helpers

// calendarIDFromPath extracts the calendar UUID from a CalDAV path.
// Expected path format: /caldav/principals/{userID}/calendars/{calendarID}/...
func calendarIDFromPath(path string) (uuid.UUID, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	// Find "calendars" segment and take the next one
	for i, part := range parts {
		if part == "calendars" && i+1 < len(parts) {
			return uuid.Parse(parts[i+1])
		}
	}
	return uuid.Nil, fmt.Errorf("calendar ID not found in path: %s", path)
}

// eventUIDFromPath extracts the event UID from a CalDAV path.
// Expected path format: .../calendars/{calendarID}/{eventUID}.ics
func eventUIDFromPath(path string) (string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("event UID not found in path: %s", path)
	}
	last := parts[len(parts)-1]
	// Strip .ics extension
	uid := strings.TrimSuffix(last, ".ics")
	if uid == "" || uid == last && strings.Contains(last, "/") {
		return "", fmt.Errorf("event UID not found in path: %s", path)
	}
	return uid, nil
}

// Proto to model conversion helpers

// protoEventToModel converts a CalendarEventProto to a CalendarEvent model.
func protoEventToModel(evt *calendarv1.CalendarEventProto) *models.CalendarEvent {
	event := &models.CalendarEvent{
		ID:         uuid.MustParse(evt.Id),
		CalendarID: uuid.MustParse(evt.CalendarId),
		Title:      evt.Title,
		StartTime:  evt.StartTime.AsTime(),
		EndTime:    evt.EndTime.AsTime(),
		IsAllDay:   evt.IsAllDay,
		Timezone:   evt.Timezone,
		HasVideoCall: evt.HasVideoCall,
		CreatedBy:  uuid.MustParse(evt.CreatedBy),
		CreatedAt:  evt.CreatedAt.AsTime(),
		UpdatedAt:  evt.UpdatedAt.AsTime(),
	}

	if evt.Description != nil {
		event.Description = evt.Description
	}
	if evt.Location != nil {
		event.Location = evt.Location
	}
	if evt.Rrule != nil {
		event.RRule = evt.Rrule
	}
	if evt.RecurrenceEnd != nil {
		t := evt.RecurrenceEnd.AsTime()
		event.RecurrenceEnd = &t
	}
	if evt.ResourceId != nil {
		rid := uuid.MustParse(*evt.ResourceId)
		event.ResourceID = &rid
	}
	if evt.CategoryId != nil {
		cid := uuid.MustParse(*evt.CategoryId)
		event.CategoryID = &cid
	}
	if evt.LivekitRoomName != nil {
		event.LiveKitRoomName = evt.LivekitRoomName
	}

	return event
}

// protoAttendeesToModels converts ListEventAttendeesResponse to model attendees.
func protoAttendeesToModels(resp *calendarv1.ListEventAttendeesResponse) []models.EventAttendee {
	if resp == nil {
		return nil
	}
	attendees := make([]models.EventAttendee, 0, len(resp.Attendees))
	for _, att := range resp.Attendees {
		attendee := models.EventAttendee{
			EventID:    uuid.MustParse(att.EventId),
			UserID:     uuid.MustParse(att.UserId),
			RSVPStatus: att.RsvpStatus,
			CreatedAt:  att.CreatedAt.AsTime(),
			FirstName:  att.FirstName,
			LastName:   att.LastName,
		}
		if att.RespondedAt != nil {
			t := att.RespondedAt.AsTime()
			attendee.RespondedAt = &t
		}
		attendees = append(attendees, attendee)
	}
	return attendees
}

// grpcToWebDAVError converts gRPC errors to appropriate WebDAV HTTP errors.
func grpcToWebDAVError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.NotFound:
		return webdav.NewHTTPError(http.StatusNotFound, err)
	case codes.PermissionDenied:
		return webdav.NewHTTPError(http.StatusForbidden, err)
	case codes.Unauthenticated:
		return webdav.NewHTTPError(http.StatusUnauthorized, err)
	case codes.InvalidArgument:
		return webdav.NewHTTPError(http.StatusBadRequest, err)
	case codes.Unavailable:
		return webdav.NewHTTPError(http.StatusServiceUnavailable, err)
	default:
		return webdav.NewHTTPError(http.StatusInternalServerError, err)
	}
}
