package caldav

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	calendarv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// webdavStatusCode extracts the HTTP status code webdav.NewHTTPError encoded
// into err.Error(), formatted as "<code> <status text>[: <cause>]" by the
// underlying (unexported) go-webdav/internal.HTTPError - we cannot import
// that internal package from outside the go-webdav module tree, so parsing
// the documented Error() format is the only way to assert on it.
func webdavStatusCode(t *testing.T, err error) int {
	t.Helper()
	require.Error(t, err)
	code, convErr := strconv.Atoi(strings.SplitN(err.Error(), " ", 2)[0])
	require.NoError(t, convErr, "expected webdav HTTPError format, got %q", err.Error())
	return code
}

func TestCalendarIDFromPath_Valid(t *testing.T) {
	id := uuid.New()
	path := "/caldav/principals/u1/calendars/" + id.String() + "/"

	got, err := calendarIDFromPath(path)

	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestCalendarIDFromPath_NoCalendarsSegment(t *testing.T) {
	_, err := calendarIDFromPath("/caldav/principals/u1/")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "calendar ID not found in path")
}

func TestCalendarIDFromPath_CalendarsIsLastSegment(t *testing.T) {
	_, err := calendarIDFromPath("/caldav/principals/u1/calendars")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "calendar ID not found in path")
}

func TestCalendarIDFromPath_InvalidUUID(t *testing.T) {
	_, err := calendarIDFromPath("/caldav/principals/u1/calendars/not-a-uuid/")

	require.Error(t, err)
}

func TestEventUIDFromPath_Valid(t *testing.T) {
	got, err := eventUIDFromPath("/caldav/principals/u1/calendars/cal1/event-123.ics")

	require.NoError(t, err)
	assert.Equal(t, "event-123", got)
}

func TestEventUIDFromPath_EmptyPath(t *testing.T) {
	_, err := eventUIDFromPath("///")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "event UID not found in path")
}

func TestProtoEventToModel_FullFields(t *testing.T) {
	id := uuid.New()
	calID := uuid.New()
	createdBy := uuid.New()
	resourceID := uuid.New()
	categoryID := uuid.New()
	start := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	recurrenceEnd := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	description := "Quarterly planning"
	location := "Room 42"
	rrule := "FREQ=WEEKLY"
	roomName := "room-abc"

	evt := &calendarv1.CalendarEventProto{
		Id:              id.String(),
		CalendarId:      calID.String(),
		Title:           "Kickoff",
		Description:     &description,
		Location:        &location,
		ResourceId:      strPtr(resourceID.String()),
		StartTime:       timestamppb.New(start),
		EndTime:         timestamppb.New(end),
		IsAllDay:        true,
		Timezone:        "Europe/Berlin",
		Rrule:           &rrule,
		RecurrenceEnd:   timestamppb.New(recurrenceEnd),
		HasVideoCall:    true,
		LivekitRoomName: &roomName,
		CategoryId:      strPtr(categoryID.String()),
		CreatedBy:       createdBy.String(),
		CreatedAt:       timestamppb.New(created),
		UpdatedAt:       timestamppb.New(updated),
	}

	model := protoEventToModel(evt)

	require.NotNil(t, model)
	assert.Equal(t, id, model.ID)
	assert.Equal(t, calID, model.CalendarID)
	assert.Equal(t, "Kickoff", model.Title)
	require.NotNil(t, model.Description)
	assert.Equal(t, description, *model.Description)
	require.NotNil(t, model.Location)
	assert.Equal(t, location, *model.Location)
	require.NotNil(t, model.ResourceID)
	assert.Equal(t, resourceID, *model.ResourceID)
	assert.True(t, start.Equal(model.StartTime))
	assert.True(t, end.Equal(model.EndTime))
	assert.True(t, model.IsAllDay)
	assert.Equal(t, "Europe/Berlin", model.Timezone)
	require.NotNil(t, model.RRule)
	assert.Equal(t, rrule, *model.RRule)
	require.NotNil(t, model.RecurrenceEnd)
	assert.True(t, recurrenceEnd.Equal(*model.RecurrenceEnd))
	assert.True(t, model.HasVideoCall)
	require.NotNil(t, model.LiveKitRoomName)
	assert.Equal(t, roomName, *model.LiveKitRoomName)
	require.NotNil(t, model.CategoryID)
	assert.Equal(t, categoryID, *model.CategoryID)
	assert.Equal(t, createdBy, model.CreatedBy)
	assert.True(t, created.Equal(model.CreatedAt))
	assert.True(t, updated.Equal(model.UpdatedAt))
}

func TestProtoEventToModel_MinimalFields_NoOptionalsSet(t *testing.T) {
	id := uuid.New()
	calID := uuid.New()
	createdBy := uuid.New()

	evt := &calendarv1.CalendarEventProto{
		Id:         id.String(),
		CalendarId: calID.String(),
		Title:      "Minimal",
		StartTime:  timestamppb.New(time.Now()),
		EndTime:    timestamppb.New(time.Now()),
		CreatedBy:  createdBy.String(),
		CreatedAt:  timestamppb.New(time.Now()),
		UpdatedAt:  timestamppb.New(time.Now()),
	}

	model := protoEventToModel(evt)

	require.NotNil(t, model)
	assert.Nil(t, model.Description)
	assert.Nil(t, model.Location)
	assert.Nil(t, model.ResourceID)
	assert.Nil(t, model.RRule)
	assert.Nil(t, model.RecurrenceEnd)
	assert.Nil(t, model.LiveKitRoomName)
	assert.Nil(t, model.CategoryID)
}

func TestProtoAttendeesToModels_NilResponse(t *testing.T) {
	got := protoAttendeesToModels(nil)

	assert.Nil(t, got)
}

func TestProtoAttendeesToModels_EmptyList(t *testing.T) {
	got := protoAttendeesToModels(&calendarv1.ListEventAttendeesResponse{})

	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestProtoAttendeesToModels_FullFields(t *testing.T) {
	eventID := uuid.New()
	userID := uuid.New()
	responded := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	created := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)

	resp := &calendarv1.ListEventAttendeesResponse{
		Attendees: []*calendarv1.EventAttendeeProto{
			{
				EventId:     eventID.String(),
				UserId:      userID.String(),
				RsvpStatus:  "accepted",
				RespondedAt: timestamppb.New(responded),
				CreatedAt:   timestamppb.New(created),
				FirstName:   "Anna",
				LastName:    "Mueller",
			},
			{
				EventId:    eventID.String(),
				UserId:     uuid.New().String(),
				RsvpStatus: "pending",
				CreatedAt:  timestamppb.New(created),
			},
		},
	}

	got := protoAttendeesToModels(resp)

	require.Len(t, got, 2)
	assert.Equal(t, eventID, got[0].EventID)
	assert.Equal(t, userID, got[0].UserID)
	assert.Equal(t, "accepted", got[0].RSVPStatus)
	require.NotNil(t, got[0].RespondedAt)
	assert.True(t, responded.Equal(*got[0].RespondedAt))
	assert.Equal(t, "Anna", got[0].FirstName)
	assert.Equal(t, "Mueller", got[0].LastName)

	assert.Nil(t, got[1].RespondedAt)
}

func TestGrpcToWebDAVError_Nil(t *testing.T) {
	assert.Nil(t, grpcToWebDAVError(nil))
}

func TestGrpcToWebDAVError_NonGRPCStatusError_ReturnedUnchanged(t *testing.T) {
	plain := errors.New("boom")

	got := grpcToWebDAVError(plain)

	assert.Same(t, plain, got)
}

func TestGrpcToWebDAVError_CodeMapping(t *testing.T) {
	cases := []struct {
		name string
		code codes.Code
		want int
	}{
		{"not_found", codes.NotFound, 404},
		{"permission_denied", codes.PermissionDenied, 403},
		{"unauthenticated", codes.Unauthenticated, 401},
		{"invalid_argument", codes.InvalidArgument, 400},
		{"unavailable", codes.Unavailable, 503},
		{"internal_falls_to_default", codes.Internal, 500},
		{"unknown_falls_to_default", codes.Unknown, 500},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := status.Error(tc.code, "boom")

			got := grpcToWebDAVError(err)

			assert.Equal(t, tc.want, webdavStatusCode(t, got))
		})
	}
}
