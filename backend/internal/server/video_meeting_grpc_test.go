package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/work/meeting"
	"github.com/kmuhub/kmuhub/internal/work/presence"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// ---------------------------------------------------------------------------
// meeting.Repository stateful mock — covers the meetings/participants/
// breakout-rooms half of video_grpc.go (b-cov-server-video-meetings).
// ---------------------------------------------------------------------------

type meetingMockRepo struct {
	meetings             map[uuid.UUID]*meeting.Meeting
	roomIndex            map[string]uuid.UUID
	attendees            map[uuid.UUID][]meeting.MeetingAttendeeWithUser
	userNames            map[uuid.UUID][2]string
	notes                map[uuid.UUID][]meeting.MeetingNotes
	prevNotesByRecurring map[uuid.UUID]*meeting.MeetingNotes
	actionItems          map[uuid.UUID]*meeting.MeetingActionItem
	chatMessages         map[uuid.UUID][]meeting.MeetingChatMessage
	coHosts              map[uuid.UUID][]meeting.MeetingCoHost
	breakoutRooms        map[uuid.UUID]*meeting.BreakoutRoom
	breakoutByMeeting    map[uuid.UUID][]*meeting.BreakoutRoom
	breakoutAssignments  map[string]meeting.BreakoutAssignment
}

func newMeetingMockRepo() *meetingMockRepo {
	return &meetingMockRepo{
		meetings:             make(map[uuid.UUID]*meeting.Meeting),
		roomIndex:            make(map[string]uuid.UUID),
		attendees:            make(map[uuid.UUID][]meeting.MeetingAttendeeWithUser),
		userNames:            make(map[uuid.UUID][2]string),
		notes:                make(map[uuid.UUID][]meeting.MeetingNotes),
		prevNotesByRecurring: make(map[uuid.UUID]*meeting.MeetingNotes),
		actionItems:          make(map[uuid.UUID]*meeting.MeetingActionItem),
		chatMessages:         make(map[uuid.UUID][]meeting.MeetingChatMessage),
		coHosts:              make(map[uuid.UUID][]meeting.MeetingCoHost),
		breakoutRooms:        make(map[uuid.UUID]*meeting.BreakoutRoom),
		breakoutByMeeting:    make(map[uuid.UUID][]*meeting.BreakoutRoom),
		breakoutAssignments:  make(map[string]meeting.BreakoutAssignment),
	}
}

func breakoutAssignmentKey(meetingID, userID uuid.UUID) string {
	return meetingID.String() + "|" + userID.String()
}

// --- Meeting CRUD ---

func (r *meetingMockRepo) CreateMeeting(_ context.Context, m *meeting.Meeting) error {
	cp := *m
	r.meetings[m.ID] = &cp
	return nil
}

func (r *meetingMockRepo) GetMeeting(_ context.Context, id, tenantID uuid.UUID) (*meeting.Meeting, error) {
	m, ok := r.meetings[id]
	if !ok || m.TenantID != tenantID {
		return nil, meeting.ErrNotFound
	}
	cp := *m
	return &cp, nil
}

func (r *meetingMockRepo) UpdateMeeting(_ context.Context, m *meeting.Meeting) error {
	if _, ok := r.meetings[m.ID]; !ok {
		return meeting.ErrNotFound
	}
	cp := *m
	r.meetings[m.ID] = &cp
	if m.RoomName != nil {
		r.roomIndex[*m.RoomName] = m.ID
	}
	return nil
}

func (r *meetingMockRepo) UpdateAISummary(_ context.Context, tenantID, meetingID uuid.UUID, summary string, at time.Time) error {
	m, ok := r.meetings[meetingID]
	if !ok || m.TenantID != tenantID {
		return meeting.ErrNotFound
	}
	m.AISummary = &summary
	m.AISummaryAt = &at
	return nil
}

func (r *meetingMockRepo) DeleteMeeting(_ context.Context, id, tenantID uuid.UUID) error {
	m, ok := r.meetings[id]
	if !ok || m.TenantID != tenantID {
		return meeting.ErrNotFound
	}
	delete(r.meetings, id)
	return nil
}

func (r *meetingMockRepo) ListMeetings(_ context.Context, filter meeting.MeetingFilter) ([]meeting.Meeting, error) {
	var out []meeting.Meeting
	for _, m := range r.meetings {
		if m.TenantID != filter.TenantID {
			continue
		}
		if filter.AttendeeID != nil {
			found := false
			for _, a := range r.attendees[m.ID] {
				if a.UserID == *filter.AttendeeID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if filter.Status != nil && m.Status != *filter.Status {
			continue
		}
		out = append(out, *m)
	}
	return out, nil
}

func (r *meetingMockRepo) GetMeetingByRoomName(_ context.Context, roomName string) (*meeting.Meeting, error) {
	id, ok := r.roomIndex[roomName]
	if !ok {
		return nil, meeting.ErrNotFound
	}
	cp := *r.meetings[id]
	return &cp, nil
}

func (r *meetingMockRepo) ListStaleMeetings(_ context.Context, _ time.Time) ([]meeting.Meeting, error) {
	return nil, nil
}

// --- Attendees ---

func (r *meetingMockRepo) AddAttendee(_ context.Context, meetingID, userID uuid.UUID) error {
	for _, a := range r.attendees[meetingID] {
		if a.UserID == userID {
			return nil
		}
	}
	first, last := "", ""
	if n, ok := r.userNames[userID]; ok {
		first, last = n[0], n[1]
	}
	r.attendees[meetingID] = append(r.attendees[meetingID], meeting.MeetingAttendeeWithUser{
		MeetingAttendee: meeting.MeetingAttendee{MeetingID: meetingID, UserID: userID, RSVPStatus: meeting.MeetingRSVPPending},
		FirstName:       first,
		LastName:        last,
	})
	return nil
}

func (r *meetingMockRepo) RemoveAttendee(_ context.Context, meetingID, userID uuid.UUID) error {
	list := r.attendees[meetingID]
	for i, a := range list {
		if a.UserID == userID {
			r.attendees[meetingID] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *meetingMockRepo) UpdateAttendeeRSVP(_ context.Context, meetingID, userID uuid.UUID, rsvp string) error {
	list := r.attendees[meetingID]
	for i, a := range list {
		if a.UserID == userID {
			list[i].RSVPStatus = rsvp
			return nil
		}
	}
	return meeting.ErrNotFound
}

func (r *meetingMockRepo) GetAttendees(_ context.Context, meetingID uuid.UUID) ([]meeting.MeetingAttendeeWithUser, error) {
	return r.attendees[meetingID], nil
}

// --- Notes ---

func (r *meetingMockRepo) SaveNotes(_ context.Context, notes *meeting.MeetingNotes) error {
	r.notes[notes.MeetingID] = append(r.notes[notes.MeetingID], *notes)
	return nil
}

func (r *meetingMockRepo) GetNotes(_ context.Context, meetingID, authorID uuid.UUID) (*meeting.MeetingNotes, error) {
	for i := len(r.notes[meetingID]) - 1; i >= 0; i-- {
		if r.notes[meetingID][i].AuthorID == authorID {
			n := r.notes[meetingID][i]
			return &n, nil
		}
	}
	return nil, meeting.ErrNotFound
}

func (r *meetingMockRepo) GetAllNotes(_ context.Context, meetingID uuid.UUID) ([]meeting.MeetingNotes, error) {
	return r.notes[meetingID], nil
}

func (r *meetingMockRepo) GetPreviousMeetingNotes(_ context.Context, recurringMeetingID uuid.UUID, _ time.Time) (*meeting.MeetingNotes, error) {
	notes, ok := r.prevNotesByRecurring[recurringMeetingID]
	if !ok {
		return nil, meeting.ErrNoPreviousNotes
	}
	return notes, nil
}

// --- Action Items ---

func (r *meetingMockRepo) CreateActionItem(_ context.Context, item *meeting.MeetingActionItem) error {
	cp := *item
	r.actionItems[item.ID] = &cp
	return nil
}

func (r *meetingMockRepo) GetActionItemByID(_ context.Context, id, tenantID uuid.UUID) (*meeting.MeetingActionItem, error) {
	item, ok := r.actionItems[id]
	if !ok || item.TenantID != tenantID {
		return nil, meeting.ErrActionItemNotFound
	}
	cp := *item
	return &cp, nil
}

func (r *meetingMockRepo) UpdateActionItem(_ context.Context, item *meeting.MeetingActionItem, tenantID uuid.UUID) error {
	existing, ok := r.actionItems[item.ID]
	if !ok || existing.TenantID != tenantID {
		return meeting.ErrActionItemNotFound
	}
	cp := *item
	r.actionItems[item.ID] = &cp
	return nil
}

func (r *meetingMockRepo) DeleteActionItem(_ context.Context, id, tenantID uuid.UUID) error {
	existing, ok := r.actionItems[id]
	if !ok || existing.TenantID != tenantID {
		return meeting.ErrActionItemNotFound
	}
	delete(r.actionItems, id)
	return nil
}

func (r *meetingMockRepo) ListActionItems(_ context.Context, meetingID, tenantID uuid.UUID) ([]meeting.MeetingActionItem, error) {
	var out []meeting.MeetingActionItem
	for _, item := range r.actionItems {
		if item.TenantID != tenantID {
			continue
		}
		if meetingID != uuid.Nil && item.MeetingID != meetingID {
			continue
		}
		out = append(out, *item)
	}
	return out, nil
}

func (r *meetingMockRepo) UpdateActionItemTaskID(_ context.Context, itemID, taskID uuid.UUID) error {
	item, ok := r.actionItems[itemID]
	if !ok {
		return meeting.ErrActionItemNotFound
	}
	item.TaskID = &taskID
	return nil
}

// --- Chat ---

func (r *meetingMockRepo) SaveChatMessage(_ context.Context, msg *meeting.MeetingChatMessage) error {
	r.chatMessages[msg.MeetingID] = append(r.chatMessages[msg.MeetingID], *msg)
	return nil
}

func (r *meetingMockRepo) ListChatMessages(_ context.Context, meetingID, _ uuid.UUID, limit int) ([]meeting.MeetingChatMessage, error) {
	msgs := r.chatMessages[meetingID]
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[:limit]
	}
	return msgs, nil
}

// --- Co-hosts ---

func (r *meetingMockRepo) AddCoHost(_ context.Context, tenantID, meetingID, userID, grantedBy uuid.UUID) error {
	for _, ch := range r.coHosts[meetingID] {
		if ch.UserID == userID {
			return nil
		}
	}
	r.coHosts[meetingID] = append(r.coHosts[meetingID], meeting.MeetingCoHost{
		ID: uuid.New(), TenantID: tenantID, MeetingID: meetingID, UserID: userID, GrantedBy: grantedBy, CreatedAt: time.Now().UTC(),
	})
	return nil
}

func (r *meetingMockRepo) RemoveCoHost(_ context.Context, _, meetingID, userID uuid.UUID) error {
	list := r.coHosts[meetingID]
	for i, ch := range list {
		if ch.UserID == userID {
			r.coHosts[meetingID] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *meetingMockRepo) IsCoHost(_ context.Context, _, meetingID, userID uuid.UUID) (bool, error) {
	for _, ch := range r.coHosts[meetingID] {
		if ch.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

func (r *meetingMockRepo) ListCoHosts(_ context.Context, _, meetingID uuid.UUID) ([]meeting.MeetingCoHost, error) {
	return r.coHosts[meetingID], nil
}

// --- Lock ---

func (r *meetingMockRepo) SetLocked(_ context.Context, tenantID, meetingID uuid.UUID, locked bool) error {
	m, ok := r.meetings[meetingID]
	if !ok || m.TenantID != tenantID {
		return meeting.ErrNotFound
	}
	m.Locked = locked
	return nil
}

// --- Breakout rooms ---

func (r *meetingMockRepo) CreateBreakoutRoom(_ context.Context, room *meeting.BreakoutRoom) error {
	r.breakoutRooms[room.ID] = room
	r.breakoutByMeeting[room.MeetingID] = append(r.breakoutByMeeting[room.MeetingID], room)
	return nil
}

func (r *meetingMockRepo) ListBreakoutRooms(_ context.Context, meetingID, _ uuid.UUID) ([]*meeting.BreakoutRoom, error) {
	return r.breakoutByMeeting[meetingID], nil
}

func (r *meetingMockRepo) GetBreakoutRoom(_ context.Context, id, _ uuid.UUID) (*meeting.BreakoutRoom, error) {
	room, ok := r.breakoutRooms[id]
	if !ok {
		return nil, meeting.ErrBreakoutRoomNotFound
	}
	return room, nil
}

func (r *meetingMockRepo) CloseAllBreakoutRooms(_ context.Context, meetingID, _ uuid.UUID) ([]string, error) {
	var closed []string
	for _, room := range r.breakoutByMeeting[meetingID] {
		if room.Status == "open" {
			room.Status = "closed"
			closed = append(closed, room.RoomName)
		}
	}
	return closed, nil
}

func (r *meetingMockRepo) UpsertBreakoutAssignment(_ context.Context, meetingID, breakoutRoomID, userID, _ uuid.UUID) error {
	r.breakoutAssignments[breakoutAssignmentKey(meetingID, userID)] = meeting.BreakoutAssignment{
		MeetingID: meetingID, BreakoutRoomID: breakoutRoomID, UserID: userID,
	}
	return nil
}

func (r *meetingMockRepo) GetBreakoutAssignmentForUser(_ context.Context, meetingID, userID, _ uuid.UUID) (*meeting.BreakoutRoom, error) {
	assignment, ok := r.breakoutAssignments[breakoutAssignmentKey(meetingID, userID)]
	if !ok {
		return nil, nil
	}
	room, ok := r.breakoutRooms[assignment.BreakoutRoomID]
	if !ok {
		return nil, nil
	}
	return room, nil
}

func (r *meetingMockRepo) ListBreakoutAssignments(_ context.Context, meetingID, _ uuid.UUID) ([]meeting.BreakoutAssignment, error) {
	var out []meeting.BreakoutAssignment
	for _, a := range r.breakoutAssignments {
		if a.MeetingID == meetingID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *meetingMockRepo) ClearBreakoutAssignment(_ context.Context, meetingID, userID, _ uuid.UUID) error {
	delete(r.breakoutAssignments, breakoutAssignmentKey(meetingID, userID))
	return nil
}

func (r *meetingMockRepo) ClearAllBreakoutAssignments(_ context.Context, meetingID, _ uuid.UUID) error {
	for k, a := range r.breakoutAssignments {
		if a.MeetingID == meetingID {
			delete(r.breakoutAssignments, k)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// presence.Store / presence.ConfigRepository stateful mocks
// ---------------------------------------------------------------------------

type presenceMockStore struct {
	data map[string]*presence.UserPresence
}

func newPresenceMockStore() *presenceMockStore {
	return &presenceMockStore{data: make(map[string]*presence.UserPresence)}
}

func (s *presenceMockStore) SetPresence(_ context.Context, userID string, st presence.PresenceStatus, manual bool) error {
	uid, _ := uuid.Parse(userID)
	now := time.Now().UTC()
	s.data[userID] = &presence.UserPresence{UserID: uid, Status: st, ManualOverride: manual, LastActivity: &now}
	return nil
}

func (s *presenceMockStore) GetPresence(_ context.Context, userID string) (*presence.UserPresence, error) {
	p, ok := s.data[userID]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (s *presenceMockStore) GetBulkPresence(_ context.Context, userIDs []string) (map[string]*presence.UserPresence, error) {
	out := make(map[string]*presence.UserPresence)
	for _, id := range userIDs {
		if p, ok := s.data[id]; ok {
			cp := *p
			out[id] = &cp
		}
	}
	return out, nil
}

func (s *presenceMockStore) RemovePresence(_ context.Context, userID string) error {
	delete(s.data, userID)
	return nil
}

func (s *presenceMockStore) UpdateLastActivity(_ context.Context, userID string) error {
	if p, ok := s.data[userID]; ok {
		now := time.Now().UTC()
		p.LastActivity = &now
	}
	return nil
}

type presenceMockConfigRepo struct {
	configs map[uuid.UUID]*presence.PresenceConfig
}

func newPresenceMockConfigRepo() *presenceMockConfigRepo {
	return &presenceMockConfigRepo{configs: make(map[uuid.UUID]*presence.PresenceConfig)}
}

func (c *presenceMockConfigRepo) GetConfig(_ context.Context, tenantID uuid.UUID) (*presence.PresenceConfig, error) {
	cfg, ok := c.configs[tenantID]
	if !ok {
		return nil, presence.ErrConfigNotFound
	}
	return cfg, nil
}

func (c *presenceMockConfigRepo) UpdateConfig(_ context.Context, tenantID uuid.UUID, awayTimeoutSeconds int, updatedBy uuid.UUID) error {
	c.configs[tenantID] = &presence.PresenceConfig{TenantID: tenantID, AwayTimeoutSeconds: awayTimeoutSeconds, UpdatedAt: time.Now().UTC(), UpdatedBy: &updatedBy}
	return nil
}

// ---------------------------------------------------------------------------
// Test server construction
// ---------------------------------------------------------------------------

func newTestVideoMeetingServer() (*VideoGRPCServer, *meetingMockRepo, *presenceMockStore, *presenceMockConfigRepo) {
	repo := newMeetingMockRepo()
	store := newPresenceMockStore()
	configRepo := newPresenceMockConfigRepo()
	meetingSvc := meeting.NewService(repo)
	presenceSvc := presence.NewService(store, configRepo)
	srv := NewVideoGRPCServer(nil, meetingSvc, nil, presenceSvc, nil, nil, nil, "wss://video.example.com")
	return srv, repo, store, configRepo
}

func meetingCtxTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

func meetingCtxAuth(tenantID, userID uuid.UUID) context.Context {
	ctx := meetingCtxTenant(tenantID)
	return context.WithValue(ctx, middleware.UserIDKey, userID.String())
}

// seedMeeting creates a scheduled meeting with the given organizer as sole
// attendee and returns it (as stored, with attendees resolved).
func seedMeeting(t *testing.T, srv *VideoGRPCServer, _ *meetingMockRepo, tenantID, organizerID uuid.UUID) *videov1.Meeting {
	t.Helper()
	resp, err := srv.CreateMeeting(meetingCtxTenant(tenantID), &videov1.CreateMeetingRequest{
		Title:           "Sprint Planning",
		OrganizerId:     organizerID.String(),
		ScheduledStart:  timestamppb.New(time.Now().Add(time.Hour)),
		ScheduledEnd:    timestamppb.New(time.Now().Add(2 * time.Hour)),
		AttendeeUserIds: []string{organizerID.String()},
	})
	require.NoError(t, err)
	return resp
}

// ---------------------------------------------------------------------------
// Error mapping — mapMeetingError / mapPresenceError
// ---------------------------------------------------------------------------

func TestMapMeetingError_AllSentinels(t *testing.T) {
	cases := []struct {
		err  error
		code codes.Code
	}{
		{meeting.ErrNotFound, codes.NotFound},
		{meeting.ErrTitleRequired, codes.InvalidArgument},
		{meeting.ErrTitleTooLong, codes.InvalidArgument},
		{meeting.ErrInvalidTimeRange, codes.InvalidArgument},
		{meeting.ErrNotOrganizer, codes.PermissionDenied},
		{meeting.ErrNotAttendee, codes.PermissionDenied},
		{meeting.ErrNotStarted, codes.FailedPrecondition},
		{meeting.ErrNotScheduled, codes.FailedPrecondition},
		{meeting.ErrNotInProgress, codes.FailedPrecondition},
		{meeting.ErrCannotDelete, codes.FailedPrecondition},
		{meeting.ErrNotesContentRequired, codes.InvalidArgument},
		{meeting.ErrActionDescRequired, codes.InvalidArgument},
		{meeting.ErrChatMessageRequired, codes.InvalidArgument},
		{meeting.ErrActionItemNotFound, codes.NotFound},
		{meeting.ErrNotRecurring, codes.FailedPrecondition},
		{meeting.ErrNoPreviousNotes, codes.NotFound},
		{meeting.ErrInvalidRecurrence, codes.FailedPrecondition},
		{meeting.ErrSeriesUnavailable, codes.Unavailable},
		{meeting.ErrNoAttendeesProvided, codes.InvalidArgument},
		{meeting.ErrInvalidRSVP, codes.InvalidArgument},
		{meeting.ErrNotHost, codes.PermissionDenied},
		{meeting.ErrMeetingLocked, codes.PermissionDenied},
		{meeting.ErrCoHostNotFound, codes.NotFound},
		{meeting.ErrLLMUnavailable, codes.Unavailable},
		{meeting.ErrNoNotesToSummarize, codes.FailedPrecondition},
		{meeting.ErrBreakoutRoomNotFound, codes.NotFound},
		{meeting.ErrNoBreakoutAssignment, codes.FailedPrecondition},
		{meeting.ErrBreakoutCountInvalid, codes.InvalidArgument},
		{meeting.ErrBreakoutNotAuthorized, codes.PermissionDenied},
		// Declared sentinels the switch does not (yet) special-case — current
		// behavior is Internal. Neither is returned by any service method
		// today (grep confirms zero call sites), so this documents dead
		// mapping rather than a live gRPC-code bug.
		{meeting.ErrStartInPast, codes.Internal},
		{meeting.ErrInvalidStatus, codes.Internal},
		// Unrelated error not part of the meeting error family.
		{context.DeadlineExceeded, codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			mapped := mapMeetingError(tc.err)
			requireGRPCCode(t, mapped, tc.code)
		})
	}
}

func TestMapPresenceError_AllSentinels(t *testing.T) {
	cases := []struct {
		err  error
		code codes.Code
	}{
		{presence.ErrInvalidManualStatus, codes.InvalidArgument},
		{presence.ErrInvalidAwayTimeout, codes.InvalidArgument},
		{presence.ErrMissingTenant, codes.Unauthenticated},
		{context.DeadlineExceeded, codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			mapped := mapPresenceError(tc.err)
			requireGRPCCode(t, mapped, tc.code)
		})
	}
}

// ---------------------------------------------------------------------------
// Enum converters — both directions
// ---------------------------------------------------------------------------

func TestMeetingStatusConversion_RoundTrip(t *testing.T) {
	cases := []struct {
		str   string
		proto videov1.MeetingStatus
	}{
		{meeting.MeetingStatusScheduled, videov1.MeetingStatus_MEETING_STATUS_SCHEDULED},
		{meeting.MeetingStatusInProgress, videov1.MeetingStatus_MEETING_STATUS_IN_PROGRESS},
		{meeting.MeetingStatusCompleted, videov1.MeetingStatus_MEETING_STATUS_COMPLETED},
		{meeting.MeetingStatusCancelled, videov1.MeetingStatus_MEETING_STATUS_CANCELLED},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, stringToProtoMeetingStatus(tc.str))
		require.Equal(t, tc.str, protoMeetingStatusToString(tc.proto))
	}
	// Unknown string -> UNSPECIFIED.
	require.Equal(t, videov1.MeetingStatus_MEETING_STATUS_UNSPECIFIED, stringToProtoMeetingStatus("bogus"))
	// Unknown/unspecified proto enum falls back to "scheduled" rather than
	// an empty string — documents the existing default, not a defect: a
	// list filter that forgets to set status_filter should not silently
	// filter on an empty status string.
	require.Equal(t, meeting.MeetingStatusScheduled, protoMeetingStatusToString(videov1.MeetingStatus_MEETING_STATUS_UNSPECIFIED))
}

func TestRsvpStatusConversion(t *testing.T) {
	cases := []struct {
		str   string
		proto videov1.RsvpStatus
	}{
		{meeting.MeetingRSVPPending, videov1.RsvpStatus_RSVP_PENDING},
		{meeting.MeetingRSVPAccepted, videov1.RsvpStatus_RSVP_ACCEPTED},
		{meeting.MeetingRSVPDeclined, videov1.RsvpStatus_RSVP_DECLINED},
		{meeting.MeetingRSVPTentative, videov1.RsvpStatus_RSVP_TENTATIVE},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, stringToProtoRsvpStatus(tc.str))
	}
	require.Equal(t, videov1.RsvpStatus_RSVP_UNSPECIFIED, stringToProtoRsvpStatus("bogus"))
}

func TestPresenceLevelConversion_RoundTrip(t *testing.T) {
	cases := []struct {
		str   string
		proto videov1.PresenceLevel
	}{
		{presence.PresenceOnline, videov1.PresenceLevel_PRESENCE_ONLINE},
		{presence.PresenceAway, videov1.PresenceLevel_PRESENCE_AWAY},
		{presence.PresenceDND, videov1.PresenceLevel_PRESENCE_DND},
		{presence.PresenceInCall, videov1.PresenceLevel_PRESENCE_IN_CALL},
		{presence.PresenceOffline, videov1.PresenceLevel_PRESENCE_OFFLINE},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, stringToProtoPresenceLevel(tc.str))
		require.Equal(t, tc.str, protoPresenceLevelToString(tc.proto))
	}
	require.Equal(t, videov1.PresenceLevel_PRESENCE_UNSPECIFIED, stringToProtoPresenceLevel("bogus"))
	// Unknown/unspecified proto enum falls back to offline.
	require.Equal(t, presence.PresenceOffline, protoPresenceLevelToString(videov1.PresenceLevel_PRESENCE_UNSPECIFIED))
}

func TestDerefString(t *testing.T) {
	require.Equal(t, "", derefString(nil))
	s := "hello"
	require.Equal(t, "hello", derefString(&s))
}

func TestAttendeeDisplayName(t *testing.T) {
	uid := uuid.New()
	attendees := []meeting.MeetingAttendeeWithUser{
		{MeetingAttendee: meeting.MeetingAttendee{UserID: uid}, FirstName: "Anna", LastName: "Muster"},
		{MeetingAttendee: meeting.MeetingAttendee{UserID: uuid.New()}, FirstName: "Bob", LastName: ""},
	}
	require.Equal(t, "Anna Muster", attendeeDisplayName(attendees, uid.String()))
	require.Equal(t, "Bob", attendeeDisplayName(attendees, attendees[1].UserID.String()))
	require.Equal(t, "", attendeeDisplayName(attendees, uuid.New().String()))
}

// ---------------------------------------------------------------------------
// toProto converters — nil-safety and empty-list wire shape
// ---------------------------------------------------------------------------

func TestMeetingToProto_OptionalFields(t *testing.T) {
	organizerID := uuid.New()
	now := time.Now().UTC()
	bare := &meeting.Meeting{
		ID:             uuid.New(),
		Title:          "Bare",
		OrganizerID:    organizerID,
		Status:         meeting.MeetingStatusScheduled,
		ScheduledStart: now,
		ScheduledEnd:   now.Add(time.Hour),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	proto := meetingToProto(bare)
	require.Nil(t, proto.Description)
	require.Nil(t, proto.Agenda)
	require.Nil(t, proto.ActualStart)
	require.Nil(t, proto.RoomName)
	require.Nil(t, proto.CalendarEventId)
	require.False(t, proto.Locked)

	desc := "desc"
	roomName := "meeting-x"
	calID := uuid.New()
	full := *bare
	full.Description = &desc
	full.RoomName = &roomName
	full.CalendarEventID = &calID
	full.Locked = true
	full.ActualStart = &now
	protoFull := meetingToProto(&full)
	require.Equal(t, desc, *protoFull.Description)
	require.Equal(t, roomName, *protoFull.RoomName)
	require.Equal(t, calID.String(), *protoFull.CalendarEventId)
	require.True(t, protoFull.Locked)
	require.NotNil(t, protoFull.ActualStart)
}

func TestMeetingWithAttendeesToProto(t *testing.T) {
	m := meeting.Meeting{ID: uuid.New(), OrganizerID: uuid.New(), Status: meeting.MeetingStatusScheduled}
	mwa := &meeting.MeetingWithAttendees{Meeting: m}
	proto := meetingWithAttendeesToProto(mwa)
	require.Empty(t, proto.Attendees)

	mwa.Attendees = []meeting.MeetingAttendeeWithUser{
		{MeetingAttendee: meeting.MeetingAttendee{MeetingID: m.ID, UserID: uuid.New(), RSVPStatus: meeting.MeetingRSVPAccepted}, FirstName: "A", LastName: "B"},
	}
	proto = meetingWithAttendeesToProto(mwa)
	require.Len(t, proto.Attendees, 1)
	require.Equal(t, videov1.RsvpStatus_RSVP_ACCEPTED, proto.Attendees[0].RsvpStatus)
}

func TestActionItemToProto(t *testing.T) {
	item := &meeting.MeetingActionItem{ID: uuid.New(), MeetingID: uuid.New(), Description: "do it"}
	proto := actionItemToProto(item)
	require.Nil(t, proto.AssigneeId)
	require.Nil(t, proto.TaskId)

	aid := uuid.New()
	tid := uuid.New()
	item.AssigneeID = &aid
	item.TaskID = &tid
	proto = actionItemToProto(item)
	require.Equal(t, aid.String(), *proto.AssigneeId)
	require.Equal(t, tid.String(), *proto.TaskId)

	withAssignee := &meeting.MeetingActionItemWithAssignee{MeetingActionItem: *item}
	first := "Anna"
	withAssignee.AssigneeFirstName = &first
	protoWA := actionItemWithAssigneeToProto(withAssignee)
	require.Equal(t, "Anna", *protoWA.AssigneeFirstName)
}

func TestMeetingChatMessageToProto(t *testing.T) {
	msg := &meeting.MeetingChatMessage{ID: uuid.New(), MeetingID: uuid.New(), SenderID: uuid.New(), SenderName: "Anna", Message: "hi"}
	proto := meetingChatMessageToProto(msg)
	require.Equal(t, "hi", proto.Message)
	require.Equal(t, "Anna", proto.SenderName)
}

func TestMeetingCoHostToProto(t *testing.T) {
	ch := &meeting.MeetingCoHost{ID: uuid.New(), MeetingID: uuid.New(), UserID: uuid.New(), GrantedBy: uuid.New()}
	proto := meetingCoHostToProto(ch)
	require.Equal(t, ch.UserID.String(), proto.UserId)
}

func TestBreakoutRoomToProto(t *testing.T) {
	r := &meeting.BreakoutRoom{ID: uuid.New(), MeetingID: uuid.New(), RoomName: "r1", Label: "Raum 1", Status: "open", CreatedBy: uuid.New()}
	proto := breakoutRoomToProto(r)
	require.Nil(t, proto.ClosedAt)

	now := time.Now().UTC()
	r.ClosedAt = &now
	proto = breakoutRoomToProto(r)
	require.NotNil(t, proto.ClosedAt)
}

func TestPresenceToProto(t *testing.T) {
	p := &presence.UserPresence{UserID: uuid.New(), Status: presence.PresenceOnline, ManualOverride: true}
	proto := presenceToProto(p)
	require.Equal(t, videov1.PresenceLevel_PRESENCE_ONLINE, proto.Status)
	require.True(t, proto.ManualOverride)
	require.Nil(t, proto.LastActivity)

	now := time.Now().UTC()
	p.LastActivity = &now
	proto = presenceToProto(p)
	require.NotNil(t, proto.LastActivity)
}

// ---------------------------------------------------------------------------
// Validation paths — one invalid-input case per handler in scope.
// ---------------------------------------------------------------------------

func TestVideoMeetingHandlers_Validation(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	tenantID := uuid.New()
	userID := uuid.New()
	noTenant := context.Background()
	authed := meetingCtxAuth(tenantID, userID)

	cases := []struct {
		name string
		code codes.Code
		call func() error
	}{
		{"CreateMeeting/missing_tenant", codes.Unauthenticated, func() error {
			_, err := srv.CreateMeeting(noTenant, &videov1.CreateMeetingRequest{OrganizerId: userID.String()})
			return err
		}},
		{"CreateMeeting/invalid_organizer_id", codes.InvalidArgument, func() error {
			_, err := srv.CreateMeeting(authed, &videov1.CreateMeetingRequest{OrganizerId: "not-a-uuid"})
			return err
		}},
		{"CreateMeeting/invalid_attendee_id", codes.InvalidArgument, func() error {
			_, err := srv.CreateMeeting(authed, &videov1.CreateMeetingRequest{OrganizerId: userID.String(), AttendeeUserIds: []string{"nope"}})
			return err
		}},
		{"GetMeeting/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.GetMeeting(authed, &videov1.GetMeetingRequest{MeetingId: "nope"})
			return err
		}},
		{"GetMeeting/missing_tenant", codes.Unauthenticated, func() error {
			_, err := srv.GetMeeting(noTenant, &videov1.GetMeetingRequest{MeetingId: uuid.New().String()})
			return err
		}},
		{"UpdateMeeting/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.UpdateMeeting(authed, &videov1.UpdateMeetingRequest{MeetingId: "nope"})
			return err
		}},
		{"DeleteMeeting/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.DeleteMeeting(authed, &videov1.DeleteMeetingRequest{MeetingId: "nope"})
			return err
		}},
		{"ListMeetings/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.ListMeetings(authed, &videov1.ListMeetingsRequest{UserId: "nope"})
			return err
		}},
		{"StartMeeting/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.StartMeeting(authed, &videov1.StartMeetingRequest{MeetingId: "nope", UserId: userID.String()})
			return err
		}},
		{"StartMeeting/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.StartMeeting(authed, &videov1.StartMeetingRequest{MeetingId: uuid.New().String(), UserId: "nope"})
			return err
		}},
		{"JoinMeeting/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.JoinMeeting(authed, &videov1.JoinMeetingRequest{MeetingId: "nope", UserId: userID.String()})
			return err
		}},
		{"EndMeeting/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.EndMeeting(authed, &videov1.EndMeetingRequest{MeetingId: "nope"})
			return err
		}},
		{"GenerateMeetingSummary/missing_tenant", codes.Unauthenticated, func() error {
			_, err := srv.GenerateMeetingSummary(noTenant, &videov1.GenerateMeetingSummaryRequest{MeetingId: uuid.New().String()})
			return err
		}},
		{"GenerateMeetingSummary/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.GenerateMeetingSummary(authed, &videov1.GenerateMeetingSummaryRequest{MeetingId: "nope"})
			return err
		}},
		{"SaveMeetingNotes/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.SaveMeetingNotes(authed, &videov1.SaveMeetingNotesRequest{MeetingId: "nope", AuthorId: userID.String(), Content: "x"})
			return err
		}},
		{"SaveMeetingNotes/invalid_author_id", codes.InvalidArgument, func() error {
			_, err := srv.SaveMeetingNotes(authed, &videov1.SaveMeetingNotesRequest{MeetingId: uuid.New().String(), AuthorId: "nope", Content: "x"})
			return err
		}},
		{"GetPreviousMeetingNotes/invalid_current_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.GetPreviousMeetingNotes(authed, &videov1.GetPreviousMeetingNotesRequest{CurrentMeetingId: "nope"})
			return err
		}},
		{"ListMeetingOccurrences/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.ListMeetingOccurrences(authed, &videov1.ListMeetingOccurrencesRequest{MeetingId: "nope"})
			return err
		}},
		{"ListMeetingOccurrences/missing_start_end", codes.InvalidArgument, func() error {
			_, err := srv.ListMeetingOccurrences(authed, &videov1.ListMeetingOccurrencesRequest{MeetingId: uuid.New().String()})
			return err
		}},
		{"CreateActionItem/missing_tenant", codes.Unauthenticated, func() error {
			_, err := srv.CreateActionItem(noTenant, &videov1.CreateActionItemRequest{MeetingId: uuid.New().String(), Description: "x"})
			return err
		}},
		{"CreateActionItem/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.CreateActionItem(authed, &videov1.CreateActionItemRequest{MeetingId: "nope", Description: "x"})
			return err
		}},
		{"UpdateActionItem/invalid_action_item_id", codes.InvalidArgument, func() error {
			_, err := srv.UpdateActionItem(authed, &videov1.UpdateActionItemRequest{ActionItemId: "nope"})
			return err
		}},
		{"DeleteActionItem/invalid_action_item_id", codes.InvalidArgument, func() error {
			_, err := srv.DeleteActionItem(authed, &videov1.DeleteActionItemRequest{ActionItemId: "nope"})
			return err
		}},
		{"ListActionItems/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.ListActionItems(authed, &videov1.ListActionItemsRequest{MeetingId: "nope"})
			return err
		}},
		{"ConvertActionItemsToTasks/missing_tenant", codes.Unauthenticated, func() error {
			_, err := srv.ConvertActionItemsToTasks(noTenant, &videov1.ConvertActionItemsToTasksRequest{UserId: userID.String(), ProjectId: uuid.New().String()})
			return err
		}},
		{"ConvertActionItemsToTasks/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.ConvertActionItemsToTasks(authed, &videov1.ConvertActionItemsToTasksRequest{UserId: "nope", ProjectId: uuid.New().String()})
			return err
		}},
		{"ConvertActionItemsToTasks/invalid_project_id", codes.InvalidArgument, func() error {
			_, err := srv.ConvertActionItemsToTasks(authed, &videov1.ConvertActionItemsToTasksRequest{UserId: userID.String(), ProjectId: "nope"})
			return err
		}},
		{"UpdatePresenceConfig/missing_tenant", codes.Unauthenticated, func() error {
			_, err := srv.UpdatePresenceConfig(noTenant, &videov1.UpdatePresenceConfigRequest{AwayTimeoutSeconds: 300})
			return err
		}},
		{"UpdatePresenceConfig/missing_user_id", codes.Unauthenticated, func() error {
			_, err := srv.UpdatePresenceConfig(meetingCtxTenant(tenantID), &videov1.UpdatePresenceConfigRequest{AwayTimeoutSeconds: 300})
			return err
		}},
		{"GetPresenceConfig/missing_tenant", codes.Unauthenticated, func() error {
			_, err := srv.GetPresenceConfig(noTenant, &videov1.GetPresenceConfigRequest{})
			return err
		}},
		{"SaveMeetingChatMessage/missing_tenant", codes.Unauthenticated, func() error {
			_, err := srv.SaveMeetingChatMessage(noTenant, &videov1.SaveMeetingChatMessageRequest{MeetingId: uuid.New().String(), Message: "x"})
			return err
		}},
		{"SaveMeetingChatMessage/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.SaveMeetingChatMessage(authed, &videov1.SaveMeetingChatMessageRequest{MeetingId: "nope", Message: "x"})
			return err
		}},
		{"ListMeetingChatMessages/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.ListMeetingChatMessages(authed, &videov1.ListMeetingChatMessagesRequest{MeetingId: "nope"})
			return err
		}},
		{"PromoteCoHost/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.PromoteCoHost(authed, &videov1.PromoteCoHostRequest{MeetingId: "nope", TargetUserId: uuid.New().String()})
			return err
		}},
		{"PromoteCoHost/invalid_target_user_id", codes.InvalidArgument, func() error {
			_, err := srv.PromoteCoHost(authed, &videov1.PromoteCoHostRequest{MeetingId: uuid.New().String(), TargetUserId: "nope"})
			return err
		}},
		{"DemoteCoHost/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.DemoteCoHost(authed, &videov1.DemoteCoHostRequest{MeetingId: "nope", TargetUserId: uuid.New().String()})
			return err
		}},
		{"ListCoHosts/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.ListCoHosts(authed, &videov1.ListCoHostsRequest{MeetingId: "nope"})
			return err
		}},
		{"MuteMeetingParticipant/invalid_target_user_id", codes.InvalidArgument, func() error {
			_, err := srv.MuteMeetingParticipant(authed, &videov1.MuteMeetingParticipantRequest{MeetingId: uuid.New().String(), TargetUserId: "nope"})
			return err
		}},
		{"MuteAllMeetingParticipants/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.MuteAllMeetingParticipants(authed, &videov1.MuteAllMeetingParticipantsRequest{MeetingId: "nope"})
			return err
		}},
		{"RemoveMeetingParticipant/invalid_target_user_id", codes.InvalidArgument, func() error {
			_, err := srv.RemoveMeetingParticipant(authed, &videov1.RemoveMeetingParticipantRequest{MeetingId: uuid.New().String(), TargetUserId: "nope"})
			return err
		}},
		{"SetMeetingLock/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.SetMeetingLock(authed, &videov1.SetMeetingLockRequest{MeetingId: "nope"})
			return err
		}},
		{"CompleteMeetingByRoom/missing_room_name", codes.InvalidArgument, func() error {
			_, err := srv.CompleteMeetingByRoom(context.Background(), &videov1.CompleteMeetingByRoomRequest{})
			return err
		}},
		{"CreateBreakoutRooms/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.CreateBreakoutRooms(authed, &videov1.CreateBreakoutRoomsRequest{MeetingId: "nope"})
			return err
		}},
		{"ListBreakoutRooms/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.ListBreakoutRooms(authed, &videov1.ListBreakoutRoomsRequest{MeetingId: "nope"})
			return err
		}},
		{"AssignBreakoutParticipant/invalid_target_user_id", codes.InvalidArgument, func() error {
			_, err := srv.AssignBreakoutParticipant(authed, &videov1.AssignBreakoutParticipantRequest{MeetingId: uuid.New().String(), TargetUserId: "nope"})
			return err
		}},
		{"AssignBreakoutParticipant/invalid_breakout_room_id", codes.InvalidArgument, func() error {
			bad := "nope"
			_, err := srv.AssignBreakoutParticipant(authed, &videov1.AssignBreakoutParticipantRequest{MeetingId: uuid.New().String(), TargetUserId: uuid.New().String(), BreakoutRoomId: &bad})
			return err
		}},
		{"JoinBreakoutRoom/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.JoinBreakoutRoom(authed, &videov1.JoinBreakoutRoomRequest{MeetingId: "nope"})
			return err
		}},
		{"GetBreakoutAssignment/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.GetBreakoutAssignment(authed, &videov1.GetBreakoutAssignmentRequest{MeetingId: "nope"})
			return err
		}},
		{"ReturnToMainRoom/invalid_target_user_id", codes.InvalidArgument, func() error {
			_, err := srv.ReturnToMainRoom(authed, &videov1.ReturnToMainRoomRequest{MeetingId: uuid.New().String(), TargetUserId: "nope"})
			return err
		}},
		{"CloseBreakoutRooms/invalid_meeting_id", codes.InvalidArgument, func() error {
			_, err := srv.CloseBreakoutRooms(authed, &videov1.CloseBreakoutRoomsRequest{MeetingId: "nope"})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, tc.call(), tc.code)
		})
	}
}

// ---------------------------------------------------------------------------
// Empty-list wire shape — handler responses must be [] not null.
// ---------------------------------------------------------------------------

func TestListMeetings_EmptyResult_NotNull(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	tenantID := uuid.New()
	resp, err := srv.ListMeetings(meetingCtxTenant(tenantID), &videov1.ListMeetingsRequest{UserId: uuid.New().String(), Page: 1, PageSize: 10})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.Meetings)
	require.Empty(t, resp.Meetings)
}

func TestListActionItems_EmptyResult_NotNull(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	resp, err := srv.ListActionItems(meetingCtxTenant(tenantID), &videov1.ListActionItemsRequest{MeetingId: m.Id})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.ActionItems)
	require.Empty(t, resp.ActionItems)
}

func TestListCoHosts_EmptyResult_NotNull(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	resp, err := srv.ListCoHosts(meetingCtxTenant(tenantID), &videov1.ListCoHostsRequest{MeetingId: m.Id})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.CoHosts)
	require.Empty(t, resp.CoHosts)
}

func TestGetBulkPresence_EmptyResult_NotNull(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	resp, err := srv.GetBulkPresence(context.Background(), &videov1.GetBulkPresenceRequest{UserIds: nil})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.Statuses)
	require.Empty(t, resp.Statuses)
}

func TestConvertActionItemsToTasks_EmptyResult_NotNull(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	tenantID, userID := uuid.New(), uuid.New()
	resp, err := srv.ConvertActionItemsToTasks(meetingCtxTenant(tenantID), &videov1.ConvertActionItemsToTasksRequest{
		UserId: userID.String(), ProjectId: uuid.New().String(), ActionItemIds: nil,
	})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.ActionItems)
	require.Empty(t, resp.ActionItems)
}

// ---------------------------------------------------------------------------
// Deeper flows — meeting lifecycle, action items, breakout rooms, co-hosts.
// ---------------------------------------------------------------------------

func TestCreateMeeting_OrganizerIsAttendee(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	require.Len(t, m.Attendees, 1)
	require.Equal(t, organizerID.String(), m.Attendees[0].UserId)
	require.Equal(t, videov1.RsvpStatus_RSVP_ACCEPTED, m.Attendees[0].RsvpStatus)
}

func TestCreateMeeting_NoAttendees_InvalidTimeRange(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	_, err := srv.CreateMeeting(meetingCtxTenant(tenantID), &videov1.CreateMeetingRequest{
		Title:          "x",
		OrganizerId:    organizerID.String(),
		ScheduledStart: timestamppb.New(time.Now().Add(2 * time.Hour)),
		ScheduledEnd:   timestamppb.New(time.Now().Add(time.Hour)), // end before start
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestStartMeeting_NonOrganizer_Denied(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	_, err := srv.StartMeeting(meetingCtxTenant(tenantID), &videov1.StartMeetingRequest{MeetingId: m.Id, UserId: uuid.New().String()})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestStartMeeting_Organizer_SetsRoomName(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	resp, err := srv.StartMeeting(meetingCtxTenant(tenantID), &videov1.StartMeetingRequest{MeetingId: m.Id, UserId: organizerID.String()})
	requireGRPCOK(t, err)
	require.NotEmpty(t, resp.RoomName)
	require.Equal(t, videov1.MeetingStatus_MEETING_STATUS_IN_PROGRESS, resp.Meeting.Status)
	// No tokenGen configured -> empty token, no ice servers.
	require.Empty(t, resp.Token)
}

func TestJoinMeeting_LockedMeeting_NonAttendee_Denied(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	_, err := srv.StartMeeting(meetingCtxTenant(tenantID), &videov1.StartMeetingRequest{MeetingId: m.Id, UserId: organizerID.String()})
	requireGRPCOK(t, err)

	mid, _ := uuid.Parse(m.Id)
	require.NoError(t, repo.SetLocked(context.Background(), tenantID, mid, true))

	stranger := uuid.New()
	_, err = srv.JoinMeeting(meetingCtxTenant(tenantID), &videov1.JoinMeetingRequest{MeetingId: m.Id, UserId: stranger.String()})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestJoinMeeting_CoHost_BypassesLock(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	_, err := srv.StartMeeting(meetingCtxTenant(tenantID), &videov1.StartMeetingRequest{MeetingId: m.Id, UserId: organizerID.String()})
	requireGRPCOK(t, err)
	mid, _ := uuid.Parse(m.Id)
	require.NoError(t, repo.SetLocked(context.Background(), tenantID, mid, true))

	coHost := uuid.New()
	_, err = srv.PromoteCoHost(meetingCtxAuth(tenantID, organizerID), &videov1.PromoteCoHostRequest{MeetingId: m.Id, TargetUserId: coHost.String()})
	requireGRPCOK(t, err)

	_, err = srv.JoinMeeting(meetingCtxTenant(tenantID), &videov1.JoinMeetingRequest{MeetingId: m.Id, UserId: coHost.String()})
	requireGRPCOK(t, err)
}

func TestPromoteCoHost_NonOrganizer_Denied(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	_, err := srv.PromoteCoHost(meetingCtxAuth(tenantID, uuid.New()), &videov1.PromoteCoHostRequest{MeetingId: m.Id, TargetUserId: uuid.New().String()})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestSetMeetingLock_NonHost_Denied(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	_, err := srv.SetMeetingLock(meetingCtxAuth(tenantID, uuid.New()), &videov1.SetMeetingLockRequest{MeetingId: m.Id, Locked: true})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestMuteMeetingParticipant_NotInProgress_FailsPrecondition(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID) // still scheduled, never started
	_, err := srv.MuteMeetingParticipant(meetingCtxAuth(tenantID, organizerID), &videov1.MuteMeetingParticipantRequest{MeetingId: m.Id, TargetUserId: uuid.New().String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestCompleteMeetingByRoom_UnknownRoom_Idempotent(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	_, err := srv.CompleteMeetingByRoom(context.Background(), &videov1.CompleteMeetingByRoomRequest{RoomName: "meeting-does-not-exist"})
	requireGRPCOK(t, err)
}

func TestCompleteMeetingByRoom_TransitionsInProgressToCompleted(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	start, err := srv.StartMeeting(meetingCtxTenant(tenantID), &videov1.StartMeetingRequest{MeetingId: m.Id, UserId: organizerID.String()})
	requireGRPCOK(t, err)

	_, err = srv.CompleteMeetingByRoom(context.Background(), &videov1.CompleteMeetingByRoomRequest{RoomName: start.RoomName})
	requireGRPCOK(t, err)

	got, err := srv.GetMeeting(meetingCtxTenant(tenantID), &videov1.GetMeetingRequest{MeetingId: m.Id})
	requireGRPCOK(t, err)
	require.Equal(t, videov1.MeetingStatus_MEETING_STATUS_COMPLETED, got.Status)
}

// TestUpdateActionItem_PartialUpdate_PreservesOtherFields is a regression test
// for a real bug found while writing this coverage: Service.UpdateActionItem
// used to build its base item from a bare {ID: id} placeholder instead of
// loading the stored item, so any field not present in the request payload
// (description, assignee, sort order) was silently wiped to its zero value
// on every partial update. Fixed to load via repo.GetActionItemByID first.
func TestUpdateActionItem_PartialUpdate_PreservesOtherFields(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)

	created, err := srv.CreateActionItem(meetingCtxTenant(tenantID), &videov1.CreateActionItemRequest{
		MeetingId:   m.Id,
		Description: "original description",
	})
	requireGRPCOK(t, err)
	require.Equal(t, "original description", created.Description)

	completed := true
	updated, err := srv.UpdateActionItem(meetingCtxTenant(tenantID), &videov1.UpdateActionItemRequest{
		ActionItemId: created.Id,
		IsCompleted:  &completed,
	})
	requireGRPCOK(t, err)
	require.True(t, updated.IsCompleted)
	require.Equal(t, "original description", updated.Description, "description must survive an update that only touches is_completed")
}

func TestCreateActionItem_TenantScoped(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)

	created, err := srv.CreateActionItem(meetingCtxTenant(tenantID), &videov1.CreateActionItemRequest{
		MeetingId:   m.Id,
		Description: "task",
	})
	requireGRPCOK(t, err)

	id, _ := uuid.Parse(created.Id)
	stored, ok := repo.actionItems[id]
	require.True(t, ok)
	require.Equal(t, tenantID, stored.TenantID, "action item must be stamped with the caller's tenant, not uuid.Nil")

	// A different tenant must not see the meeting (and therefore not be able
	// to create an action item against it).
	_, err = srv.CreateActionItem(meetingCtxTenant(uuid.New()), &videov1.CreateActionItemRequest{
		MeetingId:   m.Id,
		Description: "cross-tenant",
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestBreakoutRoomLifecycle(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	hostCtx := meetingCtxAuth(tenantID, organizerID)

	created, err := srv.CreateBreakoutRooms(hostCtx, &videov1.CreateBreakoutRoomsRequest{MeetingId: m.Id, Count: 2})
	requireGRPCOK(t, err)
	require.Len(t, created.Rooms, 2)

	participant := uuid.New()
	roomID := created.Rooms[0].Id
	_, err = srv.AssignBreakoutParticipant(hostCtx, &videov1.AssignBreakoutParticipantRequest{
		MeetingId: m.Id, TargetUserId: participant.String(), BreakoutRoomId: &roomID,
	})
	requireGRPCOK(t, err)

	assignment, err := srv.GetBreakoutAssignment(meetingCtxAuth(tenantID, participant), &videov1.GetBreakoutAssignmentRequest{MeetingId: m.Id})
	requireGRPCOK(t, err)
	require.NotNil(t, assignment.Room)
	require.Equal(t, roomID, assignment.Room.Id)

	_, err = srv.ReturnToMainRoom(meetingCtxAuth(tenantID, participant), &videov1.ReturnToMainRoomRequest{MeetingId: m.Id})
	requireGRPCOK(t, err)

	afterReturn, err := srv.GetBreakoutAssignment(meetingCtxAuth(tenantID, participant), &videov1.GetBreakoutAssignmentRequest{MeetingId: m.Id})
	requireGRPCOK(t, err)
	require.Nil(t, afterReturn.Room)

	_, err = srv.CloseBreakoutRooms(hostCtx, &videov1.CloseBreakoutRoomsRequest{MeetingId: m.Id})
	requireGRPCOK(t, err)
}

func TestJoinBreakoutRoom_NoAssignment_FailsPrecondition(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	_, err := srv.JoinBreakoutRoom(meetingCtxAuth(tenantID, uuid.New()), &videov1.JoinBreakoutRoomRequest{MeetingId: m.Id})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestCreateBreakoutRooms_InvalidCount(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	_, err := srv.CreateBreakoutRooms(meetingCtxAuth(tenantID, organizerID), &videov1.CreateBreakoutRoomsRequest{MeetingId: m.Id, Count: 0})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestPresence_SetAndGet_ManualOverride(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	userID := uuid.New()
	_, err := srv.SetPresenceStatus(context.Background(), &videov1.SetPresenceStatusRequest{UserId: userID.String(), Status: videov1.PresenceLevel_PRESENCE_DND})
	requireGRPCOK(t, err)

	got, err := srv.GetPresence(context.Background(), &videov1.GetPresenceRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	require.Equal(t, videov1.PresenceLevel_PRESENCE_DND, got.Status)
	require.True(t, got.ManualOverride)
}

func TestPresence_GetPresence_UnknownUser_Offline(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	got, err := srv.GetPresence(context.Background(), &videov1.GetPresenceRequest{UserId: uuid.New().String()})
	requireGRPCOK(t, err)
	require.Equal(t, videov1.PresenceLevel_PRESENCE_OFFLINE, got.Status)
}

func TestPresenceConfig_DefaultsBeforeFirstWrite(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	tenantID := uuid.New()
	cfg, err := srv.GetPresenceConfig(meetingCtxTenant(tenantID), &videov1.GetPresenceConfigRequest{})
	requireGRPCOK(t, err)
	require.EqualValues(t, presence.DefaultAwayTimeoutSeconds, cfg.AwayTimeoutSeconds)

	_, err = srv.UpdatePresenceConfig(meetingCtxAuth(tenantID, uuid.New()), &videov1.UpdatePresenceConfigRequest{AwayTimeoutSeconds: 900})
	requireGRPCOK(t, err)

	updated, err := srv.GetPresenceConfig(meetingCtxTenant(tenantID), &videov1.GetPresenceConfigRequest{})
	requireGRPCOK(t, err)
	require.EqualValues(t, 900, updated.AwayTimeoutSeconds)
}

func TestSaveAndListMeetingChatMessages(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	_, err := srv.StartMeeting(meetingCtxTenant(tenantID), &videov1.StartMeetingRequest{MeetingId: m.Id, UserId: organizerID.String()})
	requireGRPCOK(t, err)

	_, err = srv.SaveMeetingChatMessage(meetingCtxAuth(tenantID, organizerID), &videov1.SaveMeetingChatMessageRequest{MeetingId: m.Id, SenderName: "Anna", Message: "hi all"})
	requireGRPCOK(t, err)

	list, err := srv.ListMeetingChatMessages(meetingCtxTenant(tenantID), &videov1.ListMeetingChatMessagesRequest{MeetingId: m.Id})
	requireGRPCOK(t, err)
	require.Len(t, list.Messages, 1)
	require.Equal(t, "hi all", list.Messages[0].Message)
}

func TestSaveMeetingChatMessage_EmptyMessage_InvalidArgument(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID, organizerID := uuid.New(), uuid.New()
	m := seedMeeting(t, srv, repo, tenantID, organizerID)
	_, err := srv.StartMeeting(meetingCtxTenant(tenantID), &videov1.StartMeetingRequest{MeetingId: m.Id, UserId: organizerID.String()})
	requireGRPCOK(t, err)

	_, err = srv.SaveMeetingChatMessage(meetingCtxAuth(tenantID, organizerID), &videov1.SaveMeetingChatMessageRequest{MeetingId: m.Id, Message: "  "})
	requireGRPCCode(t, err, codes.InvalidArgument)
}
