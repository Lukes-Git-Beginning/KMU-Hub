package meeting

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Repository ---

var testTenantID = uuid.New()

type mockRepo struct {
	meetings     map[uuid.UUID]*Meeting
	attendees    map[uuid.UUID][]MeetingAttendeeWithUser
	notes        map[uuid.UUID][]MeetingNotes // keyed by meeting ID
	actionItems  map[uuid.UUID][]MeetingActionItem
	chatMessages map[uuid.UUID][]MeetingChatMessage // keyed by meeting ID
	cohosts      map[uuid.UUID][]MeetingCoHost      // keyed by meeting ID
	breakoutRooms              map[uuid.UUID]*BreakoutRoom
	breakoutAssignments        map[string]*BreakoutAssignment
	breakoutRoomsClosed        bool
	breakoutAssignmentsCleared bool
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		meetings:     make(map[uuid.UUID]*Meeting),
		attendees:    make(map[uuid.UUID][]MeetingAttendeeWithUser),
		notes:        make(map[uuid.UUID][]MeetingNotes),
		actionItems:  make(map[uuid.UUID][]MeetingActionItem),
		chatMessages: make(map[uuid.UUID][]MeetingChatMessage),
		cohosts:      make(map[uuid.UUID][]MeetingCoHost),
	}
}

func (m *mockRepo) CreateMeeting(_ context.Context, mtg *Meeting) error {
	m.meetings[mtg.ID] = mtg
	return nil
}

func (m *mockRepo) GetMeeting(_ context.Context, id, tenantID uuid.UUID) (*Meeting, error) {
	mtg, ok := m.meetings[id]
	if !ok {
		return nil, ErrNotFound
	}
	// Tenant isolation: deny cross-tenant access
	if mtg.TenantID != uuid.Nil && mtg.TenantID != tenantID {
		return nil, ErrNotFound
	}
	// Return a copy to prevent mutations leaking
	cp := *mtg
	return &cp, nil
}

func (m *mockRepo) UpdateMeeting(_ context.Context, mtg *Meeting) error {
	if _, ok := m.meetings[mtg.ID]; !ok {
		return ErrNotFound
	}
	m.meetings[mtg.ID] = mtg
	return nil
}

func (m *mockRepo) UpdateAISummary(_ context.Context, tenantID, meetingID uuid.UUID, summary string, at time.Time) error {
	mtg, ok := m.meetings[meetingID]
	if !ok {
		return ErrNotFound
	}
	if mtg.TenantID != uuid.Nil && mtg.TenantID != tenantID {
		return ErrNotFound
	}
	mtg.AISummary = &summary
	mtg.AISummaryAt = &at
	return nil
}

func (m *mockRepo) DeleteMeeting(_ context.Context, id, tenantID uuid.UUID) error {
	mtg, ok := m.meetings[id]
	if !ok {
		return ErrNotFound
	}
	if mtg.TenantID != uuid.Nil && mtg.TenantID != tenantID {
		return ErrNotFound
	}
	delete(m.meetings, id)
	return nil
}

func (m *mockRepo) ListMeetings(_ context.Context, filter MeetingFilter) ([]Meeting, error) {
	var result []Meeting
	for _, mtg := range m.meetings {
		// Always filter by tenant when TenantID is set in filter
		if filter.TenantID != uuid.Nil && mtg.TenantID != filter.TenantID {
			continue
		}
		if filter.Status != nil && mtg.Status != *filter.Status {
			continue
		}
		if filter.OrganizerID != nil && mtg.OrganizerID != *filter.OrganizerID {
			continue
		}
		result = append(result, *mtg)
	}
	return result, nil
}

func (m *mockRepo) AddAttendee(_ context.Context, meetingID, userID uuid.UUID) error {
	m.attendees[meetingID] = append(m.attendees[meetingID], MeetingAttendeeWithUser{MeetingAttendee: MeetingAttendee{
		MeetingID:  meetingID,
		UserID:     userID,
		RSVPStatus: MeetingRSVPPending,
	}})
	return nil
}

func (m *mockRepo) RemoveAttendee(_ context.Context, meetingID, userID uuid.UUID) error {
	atts := m.attendees[meetingID]
	for i, a := range atts {
		if a.UserID == userID {
			m.attendees[meetingID] = append(atts[:i], atts[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepo) UpdateAttendeeRSVP(_ context.Context, meetingID, userID uuid.UUID, rsvp string) error {
	atts := m.attendees[meetingID]
	for i, a := range atts {
		if a.UserID == userID {
			atts[i].RSVPStatus = rsvp
			return nil
		}
	}
	return ErrNotFound
}

func (m *mockRepo) GetAttendees(_ context.Context, meetingID uuid.UUID) ([]MeetingAttendeeWithUser, error) {
	return m.attendees[meetingID], nil
}

func (m *mockRepo) SaveNotes(_ context.Context, notes *MeetingNotes) error {
	existing := m.notes[notes.MeetingID]
	// Check for upsert: same meeting_id + author_id + is_private
	for i, n := range existing {
		if n.AuthorID == notes.AuthorID && n.IsPrivate == notes.IsPrivate {
			existing[i] = *notes
			m.notes[notes.MeetingID] = existing
			return nil
		}
	}
	m.notes[notes.MeetingID] = append(existing, *notes)
	return nil
}

func (m *mockRepo) GetNotes(_ context.Context, meetingID, authorID uuid.UUID) (*MeetingNotes, error) {
	for _, n := range m.notes[meetingID] {
		if n.AuthorID == authorID {
			return &n, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) GetAllNotes(_ context.Context, meetingID uuid.UUID) ([]MeetingNotes, error) {
	var result []MeetingNotes
	for _, n := range m.notes[meetingID] {
		if !n.IsPrivate {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *mockRepo) GetPreviousMeetingNotes(_ context.Context, recurringMeetingID uuid.UUID, beforeDate time.Time) (*MeetingNotes, error) {
	// Find meetings with matching recurring_meeting_id before the date
	for _, mtg := range m.meetings {
		if mtg.RecurringMeetingID != nil && *mtg.RecurringMeetingID == recurringMeetingID &&
			mtg.Status == MeetingStatusCompleted && mtg.ScheduledStart.Before(beforeDate) {
			// Return first public note for this meeting
			for _, n := range m.notes[mtg.ID] {
				if !n.IsPrivate {
					return &n, nil
				}
			}
		}
	}
	return nil, ErrNoPreviousNotes
}

func (m *mockRepo) CreateActionItem(_ context.Context, item *MeetingActionItem) error {
	m.actionItems[item.MeetingID] = append(m.actionItems[item.MeetingID], *item)
	return nil
}

func (m *mockRepo) GetActionItemByID(_ context.Context, id, _ uuid.UUID) (*MeetingActionItem, error) {
	for _, items := range m.actionItems {
		for _, item := range items {
			if item.ID == id {
				cp := item
				return &cp, nil
			}
		}
	}
	return nil, ErrActionItemNotFound
}

func (m *mockRepo) UpdateActionItem(_ context.Context, item *MeetingActionItem, _ uuid.UUID) error {
	items := m.actionItems[item.MeetingID]
	for i, existing := range items {
		if existing.ID == item.ID {
			items[i] = *item
			return nil
		}
	}
	return ErrActionItemNotFound
}

func (m *mockRepo) DeleteActionItem(_ context.Context, id, _ uuid.UUID) error {
	for meetingID, items := range m.actionItems {
		for i, item := range items {
			if item.ID == id {
				m.actionItems[meetingID] = append(items[:i], items[i+1:]...)
				return nil
			}
		}
	}
	return ErrActionItemNotFound
}

func (m *mockRepo) ListActionItems(_ context.Context, meetingID, _ uuid.UUID) ([]MeetingActionItem, error) {
	return m.actionItems[meetingID], nil
}

func (m *mockRepo) UpdateActionItemTaskID(_ context.Context, itemID, taskID uuid.UUID) error {
	for meetingID, items := range m.actionItems {
		for i, item := range items {
			if item.ID == itemID {
				items[i].TaskID = &taskID
				m.actionItems[meetingID] = items
				return nil
			}
		}
	}
	return ErrActionItemNotFound
}

func (m *mockRepo) GetMeetingByRoomName(_ context.Context, roomName string) (*Meeting, error) {
	for _, mtg := range m.meetings {
		if mtg.RoomName != nil && *mtg.RoomName == roomName {
			cp := *mtg
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) ListStaleMeetings(_ context.Context, cutoff time.Time) ([]Meeting, error) {
	var result []Meeting
	for _, mtg := range m.meetings {
		if mtg.Status == MeetingStatusInProgress && mtg.ScheduledEnd.Before(cutoff) {
			result = append(result, *mtg)
		}
	}
	return result, nil
}

func (m *mockRepo) SaveChatMessage(_ context.Context, msg *MeetingChatMessage) error {
	m.chatMessages[msg.MeetingID] = append(m.chatMessages[msg.MeetingID], *msg)
	return nil
}

func (m *mockRepo) ListChatMessages(_ context.Context, meetingID, _ uuid.UUID, limit int) ([]MeetingChatMessage, error) {
	msgs := m.chatMessages[meetingID]
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[:limit]
	}
	return msgs, nil
}

func (m *mockRepo) AddCoHost(_ context.Context, tenantID, meetingID, userID, grantedBy uuid.UUID) error {
	// idempotent
	for _, ch := range m.cohosts[meetingID] {
		if ch.UserID == userID {
			return nil
		}
	}
	m.cohosts[meetingID] = append(m.cohosts[meetingID], MeetingCoHost{
		ID:        uuid.New(),
		TenantID:  tenantID,
		MeetingID: meetingID,
		UserID:    userID,
		GrantedBy: grantedBy,
		CreatedAt: time.Now().UTC(),
	})
	return nil
}

func (m *mockRepo) RemoveCoHost(_ context.Context, _, meetingID, userID uuid.UUID) error {
	cohosts := m.cohosts[meetingID]
	for i, ch := range cohosts {
		if ch.UserID == userID {
			m.cohosts[meetingID] = append(cohosts[:i], cohosts[i+1:]...)
			return nil
		}
	}
	return nil // idempotent
}

func (m *mockRepo) IsCoHost(_ context.Context, _, meetingID, userID uuid.UUID) (bool, error) {
	for _, ch := range m.cohosts[meetingID] {
		if ch.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepo) ListCoHosts(_ context.Context, _, meetingID uuid.UUID) ([]MeetingCoHost, error) {
	return m.cohosts[meetingID], nil
}

func (m *mockRepo) SetLocked(_ context.Context, _, meetingID uuid.UUID, locked bool) error {
	mtg, ok := m.meetings[meetingID]
	if !ok {
		return ErrNotFound
	}
	mtg.Locked = locked
	return nil
}

// mockRoomManager implements RoomManager for tests.
type mockRoomManager struct {
	deletedRooms    []string
	participantMap  map[string][]string // room -> participants
	deleteRoomError error
	mutedIdentities []string
	kickedIdentities []string
}

func newMockRoomManager() *mockRoomManager {
	return &mockRoomManager{participantMap: make(map[string][]string)}
}

func (m *mockRoomManager) DeleteRoom(_ context.Context, name string) error {
	if m.deleteRoomError != nil {
		return m.deleteRoomError
	}
	m.deletedRooms = append(m.deletedRooms, name)
	return nil
}

func (m *mockRoomManager) ListParticipants(_ context.Context, roomName string) ([]string, error) {
	return m.participantMap[roomName], nil
}

func (m *mockRoomManager) MuteParticipant(_ context.Context, _, identity string) error {
	m.mutedIdentities = append(m.mutedIdentities, identity)
	return nil
}

func (m *mockRoomManager) RemoveParticipant(_ context.Context, _, identity string) error {
	m.kickedIdentities = append(m.kickedIdentities, identity)
	return nil
}

// --- Helper Functions ---

func newTestService() (*Service, *mockRepo) {
	repo := newMockRepo()
	svc := NewService(repo)
	return svc, repo
}

func newTestServiceWithRoomMgr() (*Service, *mockRepo, *mockRoomManager) {
	repo := newMockRepo()
	mgr := newMockRoomManager()
	svc := NewServiceWithRoomManager(repo, mgr)
	return svc, repo, mgr
}

func validCreateInput() CreateMeetingInput {
	return CreateMeetingInput{
		TenantID:       testTenantID,
		Title:          "Weekly Standup",
		OrganizerID:    uuid.New(),
		ScheduledStart: time.Now().Add(1 * time.Hour),
		ScheduledEnd:   time.Now().Add(2 * time.Hour),
		AttendeeIDs:    []uuid.UUID{uuid.New(), uuid.New()},
	}
}

// --- Tests ---

func TestCreateMeeting_Success(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, m.ID)
	assert.Equal(t, "Weekly Standup", m.Title)
	assert.Equal(t, MeetingStatusScheduled, m.Status)
	assert.Equal(t, testTenantID, m.TenantID)

	// Organizer should be added as attendee with accepted status
	attendees := repo.attendees[m.ID]
	assert.GreaterOrEqual(t, len(attendees), 1)
	foundOrganizer := false
	for _, a := range attendees {
		if a.UserID == input.OrganizerID {
			assert.Equal(t, MeetingRSVPAccepted, a.RSVPStatus)
			foundOrganizer = true
		}
	}
	assert.True(t, foundOrganizer, "organizer should be among attendees")
}

func TestCreateMeeting_EmptyTitle(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()
	input.Title = ""

	_, err := svc.CreateMeeting(ctx, input)
	assert.ErrorIs(t, err, ErrTitleRequired)
}

func TestCreateMeeting_TitleTooLong(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()
	input.Title = string(make([]byte, 501))

	_, err := svc.CreateMeeting(ctx, input)
	assert.ErrorIs(t, err, ErrTitleTooLong)
}

func TestCreateMeeting_InvalidTimeRange(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()
	input.ScheduledEnd = input.ScheduledStart.Add(-1 * time.Hour)

	_, err := svc.CreateMeeting(ctx, input)
	assert.ErrorIs(t, err, ErrInvalidTimeRange)
}

func TestCreateMeeting_NoAttendees(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()
	input.AttendeeIDs = nil

	_, err := svc.CreateMeeting(ctx, input)
	assert.ErrorIs(t, err, ErrNoAttendeesProvided)
}

func TestUpdateMeeting_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	newTitle := "Updated Standup"
	updated, err := svc.UpdateMeeting(ctx, m.ID, testTenantID, UpdateMeetingInput{
		Title: &newTitle,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Standup", updated.Title)
}

func TestUpdateMeeting_NotScheduled(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	// Start the meeting
	_, err = svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)

	newTitle := "Updated"
	_, err = svc.UpdateMeeting(ctx, m.ID, testTenantID, UpdateMeetingInput{Title: &newTitle})
	assert.ErrorIs(t, err, ErrNotScheduled)
}

func TestDeleteMeeting_Success(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	err = svc.DeleteMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Empty(t, repo.meetings)
}

func TestDeleteMeeting_InProgress(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)

	err = svc.DeleteMeeting(ctx, m.ID, testTenantID)
	assert.ErrorIs(t, err, ErrCannotDelete)
}

func TestStartMeeting_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	started, err := svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusInProgress, started.Status)
	assert.NotNil(t, started.ActualStart)
	assert.NotNil(t, started.RoomName)
}

func TestStartMeeting_AlreadyInProgress(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)

	_, err = svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	assert.ErrorIs(t, err, ErrNotScheduled)
}

func TestStartMeeting_NotOrganizer(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	nonOrganizer := uuid.New()
	_, err = svc.StartMeeting(ctx, m.ID, testTenantID, nonOrganizer)
	assert.ErrorIs(t, err, ErrNotOrganizer)
}

func TestStartMeeting_OrganizerCanStart(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	started, err := svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusInProgress, started.Status)
	assert.NotNil(t, started.ActualStart)
	assert.NotNil(t, started.RoomName)
}

// joinTestInput builds a meeting with an explicit organizer + single attendee so
// JoinMeeting authorization paths can be asserted against known identities.
func joinTestInput(organizerID, attendeeID uuid.UUID) CreateMeetingInput {
	return CreateMeetingInput{
		TenantID:       testTenantID,
		Title:          "Sync",
		OrganizerID:    organizerID,
		ScheduledStart: time.Now().Add(1 * time.Hour),
		ScheduledEnd:   time.Now().Add(2 * time.Hour),
		AttendeeIDs:    []uuid.UUID{attendeeID},
	}
}

func TestJoinMeeting_OrganizerStartsScheduled(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	organizerID := uuid.New()

	m, err := svc.CreateMeeting(ctx, joinTestInput(organizerID, uuid.New()))
	require.NoError(t, err)

	joined, err := svc.JoinMeeting(ctx, m.ID, testTenantID, organizerID)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusInProgress, joined.Status)
	assert.NotNil(t, joined.RoomName)
}

func TestJoinMeeting_AttendeeBeforeStartRejected(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	attendeeID := uuid.New()

	m, err := svc.CreateMeeting(ctx, joinTestInput(uuid.New(), attendeeID))
	require.NoError(t, err)

	_, err = svc.JoinMeeting(ctx, m.ID, testTenantID, attendeeID)
	assert.ErrorIs(t, err, ErrNotStarted)
}

func TestJoinMeeting_AttendeeAfterStart(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	organizerID := uuid.New()
	attendeeID := uuid.New()

	m, err := svc.CreateMeeting(ctx, joinTestInput(organizerID, attendeeID))
	require.NoError(t, err)

	// Organizer brings it live via join.
	_, err = svc.JoinMeeting(ctx, m.ID, testTenantID, organizerID)
	require.NoError(t, err)

	// Invited attendee joins the in-progress room.
	joined, err := svc.JoinMeeting(ctx, m.ID, testTenantID, attendeeID)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusInProgress, joined.Status)
	assert.NotNil(t, joined.RoomName)
}

func TestJoinMeeting_NonAttendeeRejected(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	organizerID := uuid.New()

	m, err := svc.CreateMeeting(ctx, joinTestInput(organizerID, uuid.New()))
	require.NoError(t, err)
	_, err = svc.JoinMeeting(ctx, m.ID, testTenantID, organizerID)
	require.NoError(t, err)

	_, err = svc.JoinMeeting(ctx, m.ID, testTenantID, uuid.New())
	assert.ErrorIs(t, err, ErrNotAttendee)
}

func TestJoinMeeting_Idempotent(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	organizerID := uuid.New()

	m, err := svc.CreateMeeting(ctx, joinTestInput(organizerID, uuid.New()))
	require.NoError(t, err)

	first, err := svc.JoinMeeting(ctx, m.ID, testTenantID, organizerID)
	require.NoError(t, err)
	second, err := svc.JoinMeeting(ctx, m.ID, testTenantID, organizerID)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusInProgress, second.Status)
	require.NotNil(t, first.RoomName)
	require.NotNil(t, second.RoomName)
	assert.Equal(t, *first.RoomName, *second.RoomName)
}

func TestJoinMeeting_CompletedRejected(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	organizerID := uuid.New()

	m, err := svc.CreateMeeting(ctx, joinTestInput(organizerID, uuid.New()))
	require.NoError(t, err)
	_, err = svc.JoinMeeting(ctx, m.ID, testTenantID, organizerID)
	require.NoError(t, err)
	_, err = svc.EndMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)

	_, err = svc.JoinMeeting(ctx, m.ID, testTenantID, organizerID)
	assert.ErrorIs(t, err, ErrNotInProgress)
}

func TestEndMeeting_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)

	summary, err := svc.EndMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, m.ID, summary.MeetingID)
	assert.GreaterOrEqual(t, summary.DurationMinutes, 0)
}

func TestEndMeeting_NotInProgress(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.EndMeeting(ctx, m.ID, testTenantID)
	assert.ErrorIs(t, err, ErrNotInProgress)
}

func TestLifecycle_ScheduledToInProgressToCompleted(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	// Create
	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusScheduled, m.Status)

	// Start
	started, err := svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusInProgress, started.Status)
	assert.NotNil(t, started.ActualStart)

	// End
	summary, err := svc.EndMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, m.ID, summary.MeetingID)

	// Verify final state
	final := repo.meetings[m.ID]
	assert.Equal(t, MeetingStatusCompleted, final.Status)
	assert.NotNil(t, final.ActualEnd)
}

func TestCancelMeeting_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	cancelled, err := svc.CancelMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusCancelled, cancelled.Status)
}

func TestCancelMeeting_NotScheduled(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)

	_, err = svc.CancelMeeting(ctx, m.ID, testTenantID)
	assert.ErrorIs(t, err, ErrNotScheduled)
}

func TestSaveNotes_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)

	authorID := uuid.New()
	notes, err := svc.SaveNotes(ctx, m.ID, authorID, testTenantID, "Important decisions here", false)
	require.NoError(t, err)
	assert.Equal(t, "Important decisions here", notes.Content)
	assert.False(t, notes.IsPrivate)
}

func TestSaveNotes_NotInProgress(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.SaveNotes(ctx, m.ID, uuid.New(), testTenantID, "Notes", false)
	assert.ErrorIs(t, err, ErrNotInProgress)
}

func TestSaveNotes_EmptyContent(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)

	_, err = svc.SaveNotes(ctx, m.ID, uuid.New(), testTenantID, "   ", false)
	assert.ErrorIs(t, err, ErrNotesContentRequired)
}

func TestGetPreviousMeetingNotes_Success(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	recurringID := uuid.New()

	// Create and complete a previous meeting
	prevInput := validCreateInput()
	prevInput.RecurringMeetingID = &recurringID
	prevInput.ScheduledStart = time.Now().Add(-48 * time.Hour)
	prevInput.ScheduledEnd = time.Now().Add(-47 * time.Hour)

	prevMeeting, err := svc.CreateMeeting(ctx, prevInput)
	require.NoError(t, err)

	// Manually set to completed with notes (simulating lifecycle)
	repo.meetings[prevMeeting.ID].Status = MeetingStatusCompleted
	repo.notes[prevMeeting.ID] = []MeetingNotes{{
		ID:        uuid.New(),
		MeetingID: prevMeeting.ID,
		AuthorID:  uuid.New(),
		Content:   "Previous meeting notes",
		IsPrivate: false,
	}}

	// Create current recurring meeting
	currentInput := validCreateInput()
	currentInput.RecurringMeetingID = &recurringID
	currentInput.ScheduledStart = time.Now().Add(1 * time.Hour)
	currentInput.ScheduledEnd = time.Now().Add(2 * time.Hour)

	currentMeeting, err := svc.CreateMeeting(ctx, currentInput)
	require.NoError(t, err)

	// Get previous notes
	notes, err := svc.GetPreviousMeetingNotes(ctx, currentMeeting.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, "Previous meeting notes", notes.Content)
}

func TestGetPreviousMeetingNotes_NotRecurring(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.GetPreviousMeetingNotes(ctx, m.ID, testTenantID)
	assert.ErrorIs(t, err, ErrNotRecurring)
}

func TestCreateActionItem_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	assignee := uuid.New()
	item, err := svc.CreateActionItem(ctx, CreateActionItemInput{
		MeetingID:   m.ID,
		TenantID:    testTenantID,
		Description: "Follow up with client",
		AssigneeID:  &assignee,
		SortOrder:   1,
	})
	require.NoError(t, err)
	assert.Equal(t, "Follow up with client", item.Description)
	assert.False(t, item.IsCompleted)
}

func TestCreateActionItem_EmptyDescription(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.CreateActionItem(ctx, CreateActionItemInput{
		MeetingID:   m.ID,
		TenantID:    testTenantID,
		Description: "",
	})
	assert.ErrorIs(t, err, ErrActionDescRequired)
}

func TestCreateActionItem_MeetingNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.CreateActionItem(ctx, CreateActionItemInput{
		MeetingID:   uuid.New(),
		TenantID:    testTenantID,
		Description: "Something",
	})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListActionItems_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.CreateActionItem(ctx, CreateActionItemInput{
		MeetingID:   m.ID,
		TenantID:    testTenantID,
		Description: "Item 1",
		SortOrder:   0,
	})
	require.NoError(t, err)

	_, err = svc.CreateActionItem(ctx, CreateActionItemInput{
		MeetingID:   m.ID,
		TenantID:    testTenantID,
		Description: "Item 2",
		SortOrder:   1,
	})
	require.NoError(t, err)

	items, err := svc.ListActionItems(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestConvertActionItemsToTasks(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	// Create 3 action items, link one to a task
	item1, err := svc.CreateActionItem(ctx, CreateActionItemInput{
		MeetingID:   m.ID,
		TenantID:    testTenantID,
		Description: "Todo 1",
	})
	require.NoError(t, err)

	_, err = svc.CreateActionItem(ctx, CreateActionItemInput{
		MeetingID:   m.ID,
		TenantID:    testTenantID,
		Description: "Todo 2",
	})
	require.NoError(t, err)

	// Link item1 to a task
	taskID := uuid.New()
	items := repo.actionItems[m.ID]
	for i := range items {
		if items[i].ID == item1.ID {
			items[i].TaskID = &taskID
		}
	}

	// Convert should return only unconverted
	unconverted, err := svc.ConvertActionItemsToTasks(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Len(t, unconverted, 1)
	assert.Equal(t, "Todo 2", unconverted[0].Description)
}

func TestDeleteActionItem_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	item, err := svc.CreateActionItem(ctx, CreateActionItemInput{
		MeetingID:   m.ID,
		TenantID:    testTenantID,
		Description: "To delete",
	})
	require.NoError(t, err)

	err = svc.DeleteActionItem(ctx, item.ID, testTenantID)
	require.NoError(t, err)

	items, err := svc.ListActionItems(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestDeleteActionItem_NotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	err := svc.DeleteActionItem(ctx, uuid.New(), testTenantID)
	assert.ErrorIs(t, err, ErrActionItemNotFound)
}

func TestUpdateAttendeeRSVP_InvalidStatus(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	err = svc.UpdateAttendeeRSVP(ctx, m.ID, input.OrganizerID, "invalid_status")
	assert.ErrorIs(t, err, ErrInvalidRSVP)
}

func TestEndMeeting_WithNotesAndActionItems(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.StartMeeting(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)

	// Add notes
	_, err = svc.SaveNotes(ctx, m.ID, input.OrganizerID, testTenantID, "Meeting notes content", false)
	require.NoError(t, err)

	// Add action items
	_, err = svc.CreateActionItem(ctx, CreateActionItemInput{
		MeetingID:   m.ID,
		TenantID:    testTenantID,
		Description: "Follow up action",
	})
	require.NoError(t, err)

	// End meeting - summary should include notes and action items
	summary, err := svc.EndMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.NotNil(t, summary.NotesSummary)
	assert.Contains(t, *summary.NotesSummary, "Meeting notes content")
	assert.Len(t, summary.ActionItems, 1)
}

func TestGetMeeting_WithAttendees(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	result, err := svc.GetMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, m.ID, result.ID)
	assert.GreaterOrEqual(t, len(result.Attendees), 1)
}

func TestGetMeeting_NotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.GetMeeting(ctx, uuid.New(), testTenantID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteCancelledMeeting_Success(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.CancelMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)

	err = svc.DeleteMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Empty(t, repo.meetings)
}

func TestLinkActionItemToTask(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	input := validCreateInput()

	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	item, err := svc.CreateActionItem(ctx, CreateActionItemInput{
		MeetingID:   m.ID,
		TenantID:    testTenantID,
		Description: "Linkable item",
	})
	require.NoError(t, err)

	taskID := uuid.New()
	err = svc.LinkActionItemToTask(ctx, item.ID, taskID)
	require.NoError(t, err)

	// Verify the link
	items := repo.actionItems[m.ID]
	for _, i := range items {
		if i.ID == item.ID {
			assert.NotNil(t, i.TaskID)
			assert.Equal(t, taskID, *i.TaskID)
		}
	}
}

// TestMeetingRepo_TenantIsolation verifies that meetings created for TenantA
// are not visible when querying as TenantB.
func TestMeetingRepo_TenantIsolation(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create meeting as TenantA
	inputA := CreateMeetingInput{
		TenantID:       tenantA,
		Title:          "TenantA Standup",
		OrganizerID:    uuid.New(),
		ScheduledStart: time.Now().Add(1 * time.Hour),
		ScheduledEnd:   time.Now().Add(2 * time.Hour),
		AttendeeIDs:    []uuid.UUID{uuid.New()},
	}
	mtgA, err := svc.CreateMeeting(ctx, inputA)
	require.NoError(t, err)

	// Query as TenantB → should not find meeting
	_, err = svc.GetMeeting(ctx, mtgA.ID, tenantB)
	assert.ErrorIs(t, err, ErrNotFound, "TenantB should not see TenantA's meeting")

	// Query as TenantA → should find it
	found, err := svc.GetMeeting(ctx, mtgA.ID, tenantA)
	require.NoError(t, err)
	assert.Equal(t, mtgA.ID, found.ID)

	// ListMeetings with TenantB filter → empty
	listB, err := svc.ListMeetings(ctx, MeetingFilter{TenantID: tenantB})
	require.NoError(t, err)
	assert.Empty(t, listB, "ListMeetings with TenantB should return no results")

	// ListMeetings with TenantA filter → finds the meeting
	listA, err := svc.ListMeetings(ctx, MeetingFilter{TenantID: tenantA})
	require.NoError(t, err)
	assert.Len(t, listA, 1)
	assert.Equal(t, mtgA.ID, listA[0].ID)
}

// ============================================================================
// Wave 0 — auto-close tests
// ============================================================================

func TestEndMeeting_CallsDeleteRoom(t *testing.T) {
	svc, repo, mgr := newTestServiceWithRoomMgr()
	ctx := context.Background()

	roomName := "meeting-abc123"
	m := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    uuid.New(),
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-2 * time.Hour),
		ScheduledEnd:   time.Now().Add(-1 * time.Hour),
		RoomName:       &roomName,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMeeting(ctx, m))
	repo.attendees[m.ID] = []MeetingAttendeeWithUser{{MeetingAttendee: MeetingAttendee{MeetingID: m.ID, UserID: m.OrganizerID}}}

	_, err := svc.EndMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)

	assert.Contains(t, mgr.deletedRooms, roomName, "DeleteRoom should be called with the meeting's room name")
}

func TestEndMeeting_NoRoomName_DoesNotPanic(t *testing.T) {
	svc, repo, _ := newTestServiceWithRoomMgr()
	ctx := context.Background()

	m := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    uuid.New(),
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-2 * time.Hour),
		ScheduledEnd:   time.Now().Add(-1 * time.Hour),
		RoomName:       nil, // no room
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMeeting(ctx, m))
	repo.attendees[m.ID] = []MeetingAttendeeWithUser{{MeetingAttendee: MeetingAttendee{MeetingID: m.ID, UserID: m.OrganizerID}}}

	_, err := svc.EndMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err) // must not panic when RoomName is nil
}

func TestCompleteMeetingByRoom_InProgress(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	roomName := "meeting-xyz"
	m := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    uuid.New(),
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-2 * time.Hour),
		ScheduledEnd:   time.Now().Add(-1 * time.Hour),
		RoomName:       &roomName,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMeeting(ctx, m))

	err := svc.CompleteMeetingByRoom(ctx, roomName)
	require.NoError(t, err)

	updated := repo.meetings[m.ID]
	assert.Equal(t, MeetingStatusCompleted, updated.Status)
	assert.NotNil(t, updated.ActualEnd)
}

func TestCompleteMeetingByRoom_AlreadyCompleted_Idempotent(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	roomName := "meeting-done"
	now := time.Now()
	m := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    uuid.New(),
		Status:         MeetingStatusCompleted,
		ScheduledStart: time.Now().Add(-2 * time.Hour),
		ScheduledEnd:   time.Now().Add(-1 * time.Hour),
		ActualEnd:      &now,
		RoomName:       &roomName,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMeeting(ctx, m))

	// Must return nil (idempotent) and must NOT overwrite ActualEnd
	err := svc.CompleteMeetingByRoom(ctx, roomName)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusCompleted, repo.meetings[m.ID].Status)
}

func TestCompleteMeetingByRoom_AlreadyCancelled_Idempotent(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	roomName := "meeting-cancelled"
	m := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    uuid.New(),
		Status:         MeetingStatusCancelled,
		ScheduledStart: time.Now().Add(-2 * time.Hour),
		ScheduledEnd:   time.Now().Add(-1 * time.Hour),
		RoomName:       &roomName,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMeeting(ctx, m))

	err := svc.CompleteMeetingByRoom(ctx, roomName)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusCancelled, repo.meetings[m.ID].Status)
}

func TestCompleteMeetingByRoom_NotFound_Ignored(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	// Non-existent room → should not error (webhook might fire for old/test rooms)
	err := svc.CompleteMeetingByRoom(ctx, "meeting-nonexistent")
	require.NoError(t, err)
}

func TestGetMeetingByRoomName_Found(t *testing.T) {
	_, repo := newTestService()
	ctx := context.Background()

	roomName := "meeting-findme"
	m := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    uuid.New(),
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-1 * time.Hour),
		ScheduledEnd:   time.Now().Add(1 * time.Hour),
		RoomName:       &roomName,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMeeting(ctx, m))

	found, err := repo.GetMeetingByRoomName(ctx, roomName)
	require.NoError(t, err)
	assert.Equal(t, m.ID, found.ID)
}

func TestGetMeetingByRoomName_NotFound(t *testing.T) {
	_, repo := newTestService()
	ctx := context.Background()

	_, err := repo.GetMeetingByRoomName(ctx, "meeting-ghost")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListStaleMeetings_BelowCutoff(t *testing.T) {
	_, repo := newTestService()
	ctx := context.Background()

	// Meeting ended 30 min ago → stale if grace is 15 min
	roomName := "meeting-stale"
	stale := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Stale",
		OrganizerID:    uuid.New(),
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-2 * time.Hour),
		ScheduledEnd:   time.Now().Add(-30 * time.Minute),
		RoomName:       &roomName,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	// Meeting ending in future → not stale
	roomName2 := "meeting-active"
	active := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Active",
		OrganizerID:    uuid.New(),
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-1 * time.Hour),
		ScheduledEnd:   time.Now().Add(1 * time.Hour),
		RoomName:       &roomName2,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	require.NoError(t, repo.CreateMeeting(ctx, stale))
	require.NoError(t, repo.CreateMeeting(ctx, active))

	// cutoff = now - 15min
	cutoff := time.Now().Add(-15 * time.Minute)
	results, err := repo.ListStaleMeetings(ctx, cutoff)
	require.NoError(t, err)

	ids := make([]uuid.UUID, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	assert.Contains(t, ids, stale.ID, "stale meeting should be in results")
	assert.NotContains(t, ids, active.ID, "active meeting should not be in results")
}

func TestListStaleMeetings_OnlyInProgress(t *testing.T) {
	_, repo := newTestService()
	ctx := context.Background()

	roomName := "meeting-completed-old"
	completed := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Done",
		OrganizerID:    uuid.New(),
		Status:         MeetingStatusCompleted, // already done
		ScheduledStart: time.Now().Add(-3 * time.Hour),
		ScheduledEnd:   time.Now().Add(-2 * time.Hour),
		RoomName:       &roomName,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMeeting(ctx, completed))

	cutoff := time.Now().Add(-15 * time.Minute)
	results, err := repo.ListStaleMeetings(ctx, cutoff)
	require.NoError(t, err)
	assert.Empty(t, results, "completed meetings should not appear in stale list")
}

// ============================================================================
// Meeting Chat Tests
// ============================================================================

func TestSaveChatMessage_InProgress(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	m := setupInProgressMeeting(t, svc, ctx)

	senderID := uuid.New()
	msg, err := svc.SaveChatMessage(ctx, m.ID, senderID, testTenantID, "Alice", "Hello!")
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, msg.ID)
	assert.Equal(t, m.ID, msg.MeetingID)
	assert.Equal(t, senderID, msg.SenderID)
	assert.Equal(t, "Alice", msg.SenderName)
	assert.Equal(t, "Hello!", msg.Message)
	assert.Equal(t, testTenantID, msg.TenantID)
	assert.False(t, msg.CreatedAt.IsZero())
}

func TestSaveChatMessage_EmptyMessage(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	m := setupInProgressMeeting(t, svc, ctx)

	_, err := svc.SaveChatMessage(ctx, m.ID, uuid.New(), testTenantID, "Alice", "   ")
	require.ErrorIs(t, err, ErrChatMessageRequired)
}

func TestSaveChatMessage_NotInProgress(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	orgID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	input := CreateMeetingInput{
		TenantID:       testTenantID,
		Title:          "Scheduled Meeting",
		OrganizerID:    orgID,
		ScheduledStart: start,
		ScheduledEnd:   end,
		AttendeeIDs:    []uuid.UUID{orgID},
	}
	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.SaveChatMessage(ctx, m.ID, uuid.New(), testTenantID, "Bob", "Hi")
	require.ErrorIs(t, err, ErrNotInProgress)
}

func TestListChatMessages_RoundTrip(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	m := setupInProgressMeeting(t, svc, ctx)

	// Empty history
	msgs, err := svc.ListChatMessages(ctx, m.ID, testTenantID, 0)
	require.NoError(t, err)
	assert.Empty(t, msgs)

	// Save two messages
	_, err = svc.SaveChatMessage(ctx, m.ID, uuid.New(), testTenantID, "Alice", "First")
	require.NoError(t, err)
	_, err = svc.SaveChatMessage(ctx, m.ID, uuid.New(), testTenantID, "Bob", "Second")
	require.NoError(t, err)

	msgs, err = svc.ListChatMessages(ctx, m.ID, testTenantID, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, "First", msgs[0].Message)
	assert.Equal(t, "Second", msgs[1].Message)
}

func TestListChatMessages_TenantScoping(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	m := setupInProgressMeeting(t, svc, ctx)

	_, err := svc.SaveChatMessage(ctx, m.ID, uuid.New(), testTenantID, "Alice", "Secret")
	require.NoError(t, err)

	// Different tenant should get ErrNotFound (meeting not found in that tenant).
	otherTenant := uuid.New()
	_, err = svc.ListChatMessages(ctx, m.ID, otherTenant, 0)
	require.ErrorIs(t, err, ErrNotFound, "cross-tenant list must be rejected")
}

func TestListChatMessages_Limit(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	m := setupInProgressMeeting(t, svc, ctx)

	for i := 0; i < 5; i++ {
		_, err := svc.SaveChatMessage(ctx, m.ID, uuid.New(), testTenantID, "User", "msg")
		require.NoError(t, err)
	}

	msgs, err := svc.ListChatMessages(ctx, m.ID, testTenantID, 3)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)
}

func TestSaveChatMessage_Completed(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	m := setupInProgressMeeting(t, svc, ctx)
	_, err := svc.EndMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)

	// Completed meetings still allow chat (post-call history write).
	_, err = svc.SaveChatMessage(ctx, m.ID, uuid.New(), testTenantID, "Alice", "Late message")
	require.NoError(t, err)
}

// setupInProgressMeeting is a helper that creates and starts a meeting.
func setupInProgressMeeting(t *testing.T, svc *Service, ctx context.Context) *Meeting {
	t.Helper()
	orgID := uuid.New()
	start := time.Now().Add(-time.Hour) // in the past so it can be started
	end := start.Add(2 * time.Hour)
	input := CreateMeetingInput{
		TenantID:       testTenantID,
		Title:          "Test Meeting",
		OrganizerID:    orgID,
		ScheduledStart: start,
		ScheduledEnd:   end,
		AttendeeIDs:    []uuid.UUID{orgID},
	}
	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	started, err := svc.StartMeeting(ctx, m.ID, testTenantID, orgID)
	require.NoError(t, err)
	return started
}

// =============================================================================
// Wave 3 Host-Controls Tests
// =============================================================================

// helpers for host-controls tests

func newInProgressMeetingWithRoomMgr(t *testing.T) (*Service, *mockRepo, *mockRoomManager, *Meeting, uuid.UUID) {
	t.Helper()
	repo := newMockRepo()
	mgr := newMockRoomManager()
	svc := NewServiceWithRoomManager(repo, mgr)
	ctx := context.Background()

	orgID := uuid.New()
	input := CreateMeetingInput{
		TenantID:       testTenantID,
		Title:          "Host Controls Meeting",
		OrganizerID:    orgID,
		ScheduledStart: time.Now().Add(1 * time.Hour),
		ScheduledEnd:   time.Now().Add(2 * time.Hour),
		AttendeeIDs:    []uuid.UUID{orgID},
	}
	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)
	m, err = svc.StartMeeting(ctx, m.ID, testTenantID, orgID)
	require.NoError(t, err)
	// Seed the room participant list for mute-all tests
	mgr.participantMap[*m.RoomName] = []string{orgID.String()}
	return svc, repo, mgr, m, orgID
}

// --- PromoteCoHost / DemoteCoHost ---

func TestPromoteCoHost_OrganizerSuccess(t *testing.T) {
	svc, _, _, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	targetID := uuid.New()

	err := svc.PromoteCoHost(ctx, m.ID, testTenantID, orgID, targetID)
	require.NoError(t, err)

	cohosts, err := svc.ListCoHosts(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	require.Len(t, cohosts, 1)
	assert.Equal(t, targetID, cohosts[0].UserID)
}

func TestPromoteCoHost_NonOrganizerDenied(t *testing.T) {
	svc, _, _, m, _ := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	stranger := uuid.New()
	target := uuid.New()

	err := svc.PromoteCoHost(ctx, m.ID, testTenantID, stranger, target)
	assert.ErrorIs(t, err, ErrNotOrganizer)
}

func TestPromoteCoHost_Idempotent(t *testing.T) {
	svc, _, _, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	targetID := uuid.New()

	require.NoError(t, svc.PromoteCoHost(ctx, m.ID, testTenantID, orgID, targetID))
	// Second call must not error
	require.NoError(t, svc.PromoteCoHost(ctx, m.ID, testTenantID, orgID, targetID))

	cohosts, err := svc.ListCoHosts(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Len(t, cohosts, 1) // still only one entry
}

func TestDemoteCoHost_OrganizerSuccess(t *testing.T) {
	svc, _, _, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	targetID := uuid.New()

	require.NoError(t, svc.PromoteCoHost(ctx, m.ID, testTenantID, orgID, targetID))
	require.NoError(t, svc.DemoteCoHost(ctx, m.ID, testTenantID, orgID, targetID))

	cohosts, err := svc.ListCoHosts(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Empty(t, cohosts)
}

func TestDemoteCoHost_NonOrganizerDenied(t *testing.T) {
	svc, _, _, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	targetID := uuid.New()
	stranger := uuid.New()

	require.NoError(t, svc.PromoteCoHost(ctx, m.ID, testTenantID, orgID, targetID))
	err := svc.DemoteCoHost(ctx, m.ID, testTenantID, stranger, targetID)
	assert.ErrorIs(t, err, ErrNotOrganizer)
}

func TestDemoteCoHost_Idempotent(t *testing.T) {
	svc, _, _, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	targetID := uuid.New()

	// Not a co-host — demotion is a no-op
	err := svc.DemoteCoHost(ctx, m.ID, testTenantID, orgID, targetID)
	assert.NoError(t, err)
}

// --- MuteMeetingParticipant ---

func TestMuteMeetingParticipant_HostSuccess(t *testing.T) {
	svc, _, mgr, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	targetID := uuid.New()
	mgr.participantMap[*m.RoomName] = append(mgr.participantMap[*m.RoomName], targetID.String())

	err := svc.MuteMeetingParticipant(ctx, m.ID, testTenantID, orgID, targetID)
	require.NoError(t, err)
	assert.Contains(t, mgr.mutedIdentities, targetID.String())
}

func TestMuteMeetingParticipant_CoHostSuccess(t *testing.T) {
	svc, _, mgr, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	coHostID := uuid.New()
	targetID := uuid.New()

	require.NoError(t, svc.PromoteCoHost(ctx, m.ID, testTenantID, orgID, coHostID))
	mgr.participantMap[*m.RoomName] = append(mgr.participantMap[*m.RoomName], targetID.String())

	err := svc.MuteMeetingParticipant(ctx, m.ID, testTenantID, coHostID, targetID)
	require.NoError(t, err)
	assert.Contains(t, mgr.mutedIdentities, targetID.String())
}

func TestMuteMeetingParticipant_NonHostDenied(t *testing.T) {
	svc, _, _, m, _ := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	stranger := uuid.New()
	targetID := uuid.New()

	err := svc.MuteMeetingParticipant(ctx, m.ID, testTenantID, stranger, targetID)
	assert.ErrorIs(t, err, ErrNotHost)
}

// --- RemoveMeetingParticipant (kick) ---

func TestRemoveMeetingParticipant_HostSuccess(t *testing.T) {
	svc, _, mgr, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	targetID := uuid.New()

	err := svc.RemoveMeetingParticipant(ctx, m.ID, testTenantID, orgID, targetID)
	require.NoError(t, err)
	assert.Contains(t, mgr.kickedIdentities, targetID.String())
}

func TestRemoveMeetingParticipant_NonHostDenied(t *testing.T) {
	svc, _, _, m, _ := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	stranger := uuid.New()
	targetID := uuid.New()

	err := svc.RemoveMeetingParticipant(ctx, m.ID, testTenantID, stranger, targetID)
	assert.ErrorIs(t, err, ErrNotHost)
}

// --- SetMeetingLock ---

func TestSetMeetingLock_HostSuccess(t *testing.T) {
	svc, repo, _, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()

	err := svc.SetMeetingLock(ctx, m.ID, testTenantID, orgID, true)
	require.NoError(t, err)
	assert.True(t, repo.meetings[m.ID].Locked)

	err = svc.SetMeetingLock(ctx, m.ID, testTenantID, orgID, false)
	require.NoError(t, err)
	assert.False(t, repo.meetings[m.ID].Locked)
}

func TestSetMeetingLock_NonHostDenied(t *testing.T) {
	svc, _, _, m, _ := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	stranger := uuid.New()

	err := svc.SetMeetingLock(ctx, m.ID, testTenantID, stranger, true)
	assert.ErrorIs(t, err, ErrNotHost)
}

func TestSetMeetingLock_CoHostCanLock(t *testing.T) {
	svc, repo, _, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()
	coHostID := uuid.New()

	require.NoError(t, svc.PromoteCoHost(ctx, m.ID, testTenantID, orgID, coHostID))
	err := svc.SetMeetingLock(ctx, m.ID, testTenantID, coHostID, true)
	require.NoError(t, err)
	assert.True(t, repo.meetings[m.ID].Locked)
}

// --- Lock blocks non-host join ---

func TestJoinMeeting_LockedBlocksNonHost(t *testing.T) {
	svc, repo, _, m, orgID := newInProgressMeetingWithRoomMgr(t)
	ctx := context.Background()

	// Lock the meeting
	require.NoError(t, svc.SetMeetingLock(ctx, m.ID, testTenantID, orgID, true))

	// Add an attendee
	attendeeID := uuid.New()
	require.NoError(t, svc.AddAttendee(ctx, m.ID, attendeeID, testTenantID))

	// Attendee should be blocked
	_, err := svc.JoinMeeting(ctx, m.ID, testTenantID, attendeeID)
	assert.ErrorIs(t, err, ErrMeetingLocked)

	// Organizer can still join
	_, err = svc.JoinMeeting(ctx, m.ID, testTenantID, orgID)
	assert.NoError(t, err)

	// Co-host can still join
	coHostID := uuid.New()
	require.NoError(t, repo.AddAttendee(ctx, m.ID, coHostID))
	require.NoError(t, svc.PromoteCoHost(ctx, m.ID, testTenantID, orgID, coHostID))
	_, err = svc.JoinMeeting(ctx, m.ID, testTenantID, coHostID)
	assert.NoError(t, err)
}

// --- AI Summary (Wave 7C) ---

// mockSummarizer is a stub LLMSummarizer for the GenerateAISummary tests.
type mockSummarizer struct {
	out   string
	err   error
	calls int
}

func (m *mockSummarizer) Summarize(_ context.Context, _ string) (string, error) {
	m.calls++
	if m.err != nil {
		return "", m.err
	}
	return m.out, nil
}

func TestGenerateAISummary_NoLLMConfigured(t *testing.T) {
	svc, _ := newTestService() // no LLM attached
	ctx := context.Background()
	input := validCreateInput()
	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	_, err = svc.GenerateAISummary(ctx, m.ID, testTenantID, input.OrganizerID)
	assert.ErrorIs(t, err, ErrLLMUnavailable)
}

func TestGenerateAISummary_NotHost(t *testing.T) {
	svc, _ := newTestService()
	svc.WithLLM(&mockSummarizer{out: "summary"})
	ctx := context.Background()
	input := validCreateInput()
	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	// A stranger (neither organizer nor co-host) is rejected before any LLM call.
	_, err = svc.GenerateAISummary(ctx, m.ID, testTenantID, uuid.New())
	assert.ErrorIs(t, err, ErrNotHost)
}

func TestGenerateAISummary_NoNotes(t *testing.T) {
	svc, _ := newTestService()
	llm := &mockSummarizer{out: "summary"}
	svc.WithLLM(llm)
	ctx := context.Background()
	input := validCreateInput()
	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	// No public notes seeded → fail before reaching the LLM.
	_, err = svc.GenerateAISummary(ctx, m.ID, testTenantID, input.OrganizerID)
	assert.ErrorIs(t, err, ErrNoNotesToSummarize)
	assert.Equal(t, 0, llm.calls, "LLM must not be called when there are no notes")
}

func TestGenerateAISummary_Success(t *testing.T) {
	svc, repo := newTestService()
	llm := &mockSummarizer{out: "Kurzfassung der Notizen."}
	svc.WithLLM(llm)
	ctx := context.Background()
	input := validCreateInput()
	m, err := svc.CreateMeeting(ctx, input)
	require.NoError(t, err)

	// Seed one public note (private notes are excluded by GetAllNotes).
	repo.notes[m.ID] = []MeetingNotes{
		{ID: uuid.New(), MeetingID: m.ID, AuthorID: input.OrganizerID, Content: "Wichtige Entscheidungen", IsPrivate: false},
		{ID: uuid.New(), MeetingID: m.ID, AuthorID: uuid.New(), Content: "geheim", IsPrivate: true},
	}

	updated, err := svc.GenerateAISummary(ctx, m.ID, testTenantID, input.OrganizerID)
	require.NoError(t, err)
	require.NotNil(t, updated.AISummary)
	assert.Equal(t, "Kurzfassung der Notizen.", *updated.AISummary)
	require.NotNil(t, updated.AISummaryAt)
	assert.Equal(t, 1, llm.calls)

	// Persisted on the stored meeting row.
	stored := repo.meetings[m.ID]
	require.NotNil(t, stored.AISummary)
	assert.Equal(t, "Kurzfassung der Notizen.", *stored.AISummary)
	require.NotNil(t, stored.AISummaryAt)
}

// --- Breakout Room Mock Methods ---

func (m *mockRepo) CreateBreakoutRoom(_ context.Context, r *BreakoutRoom) error {
	if m.breakoutRooms == nil {
		m.breakoutRooms = make(map[uuid.UUID]*BreakoutRoom)
	}
	cp := *r
	m.breakoutRooms[r.ID] = &cp
	return nil
}

func (m *mockRepo) ListBreakoutRooms(_ context.Context, meetingID, _ uuid.UUID) ([]*BreakoutRoom, error) {
	var result []*BreakoutRoom
	for _, r := range m.breakoutRooms {
		if r.MeetingID == meetingID {
			cp := *r
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *mockRepo) GetBreakoutRoom(_ context.Context, id, _ uuid.UUID) (*BreakoutRoom, error) {
	r, ok := m.breakoutRooms[id]
	if !ok {
		return nil, ErrBreakoutRoomNotFound
	}
	cp := *r
	return &cp, nil
}

func (m *mockRepo) CloseAllBreakoutRooms(_ context.Context, meetingID, _ uuid.UUID) ([]string, error) {
	m.breakoutRoomsClosed = true
	var names []string
	for _, r := range m.breakoutRooms {
		if r.MeetingID == meetingID && r.Status == "open" {
			r.Status = "closed"
			names = append(names, r.RoomName)
		}
	}
	return names, nil
}

func (m *mockRepo) UpsertBreakoutAssignment(_ context.Context, meetingID, breakoutRoomID, userID, assignedBy uuid.UUID) error {
	if m.breakoutAssignments == nil {
		m.breakoutAssignments = make(map[string]*BreakoutAssignment)
	}
	key := meetingID.String() + ":" + userID.String()
	m.breakoutAssignments[key] = &BreakoutAssignment{
		MeetingID:      meetingID,
		BreakoutRoomID: breakoutRoomID,
		UserID:         userID,
	}
	_ = assignedBy
	return nil
}

func (m *mockRepo) GetBreakoutAssignmentForUser(_ context.Context, meetingID, userID, tenantID uuid.UUID) (*BreakoutRoom, error) {
	if m.breakoutAssignments == nil {
		return nil, nil
	}
	key := meetingID.String() + ":" + userID.String()
	asgn, ok := m.breakoutAssignments[key]
	if !ok {
		return nil, nil
	}
	_ = tenantID
	return m.GetBreakoutRoom(context.Background(), asgn.BreakoutRoomID, uuid.Nil)
}

func (m *mockRepo) ListBreakoutAssignments(_ context.Context, meetingID, _ uuid.UUID) ([]BreakoutAssignment, error) {
	var result []BreakoutAssignment
	for _, a := range m.breakoutAssignments {
		if a.MeetingID == meetingID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockRepo) ClearBreakoutAssignment(_ context.Context, meetingID, userID, _ uuid.UUID) error {
	if m.breakoutAssignments == nil {
		return nil
	}
	key := meetingID.String() + ":" + userID.String()
	delete(m.breakoutAssignments, key)
	return nil
}

func (m *mockRepo) ClearAllBreakoutAssignments(_ context.Context, meetingID, _ uuid.UUID) error {
	m.breakoutAssignmentsCleared = true
	for key, a := range m.breakoutAssignments {
		if a.MeetingID == meetingID {
			delete(m.breakoutAssignments, key)
		}
	}
	return nil
}

// --- Breakout Room Tests ---

func TestCreateBreakoutRooms_HostOnly(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	hostID := uuid.New()
	mtg := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    hostID,
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-1 * time.Hour),
		ScheduledEnd:   time.Now().Add(1 * time.Hour),
	}
	require.NoError(t, repo.CreateMeeting(ctx, mtg))

	nonHost := uuid.New()
	_, err := svc.CreateBreakoutRooms(ctx, mtg.ID, testTenantID, nonHost, 3, nil)
	assert.ErrorIs(t, err, ErrBreakoutNotAuthorized)
}

func TestCreateBreakoutRooms_CountBounds(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	hostID := uuid.New()
	mtg := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    hostID,
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-1 * time.Hour),
		ScheduledEnd:   time.Now().Add(1 * time.Hour),
	}
	require.NoError(t, repo.CreateMeeting(ctx, mtg))

	_, err := svc.CreateBreakoutRooms(ctx, mtg.ID, testTenantID, hostID, 0, nil)
	assert.ErrorIs(t, err, ErrBreakoutCountInvalid, "count=0 should be invalid")

	_, err = svc.CreateBreakoutRooms(ctx, mtg.ID, testTenantID, hostID, 21, nil)
	assert.ErrorIs(t, err, ErrBreakoutCountInvalid, "count=21 should be invalid")
}

func TestAssignBreakoutParticipant_NonHost(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	hostID := uuid.New()
	roomID := uuid.New()
	mtg := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    hostID,
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-1 * time.Hour),
		ScheduledEnd:   time.Now().Add(1 * time.Hour),
	}
	require.NoError(t, repo.CreateMeeting(ctx, mtg))

	nonHost := uuid.New()
	targetUser := uuid.New()
	err := svc.AssignBreakoutParticipant(ctx, mtg.ID, testTenantID, nonHost, targetUser, &roomID)
	assert.ErrorIs(t, err, ErrBreakoutNotAuthorized)
}

func TestJoinBreakoutRoom_NoAssignment(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	userID := uuid.New()
	mtg := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    uuid.New(),
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-1 * time.Hour),
		ScheduledEnd:   time.Now().Add(1 * time.Hour),
	}
	require.NoError(t, repo.CreateMeeting(ctx, mtg))

	_, err := svc.JoinBreakoutRoom(ctx, mtg.ID, testTenantID, userID)
	assert.ErrorIs(t, err, ErrNoBreakoutAssignment)
}

func TestEndMeeting_CallsBreakoutCleanup(t *testing.T) {
	svc, repo, _ := newTestServiceWithRoomMgr()
	ctx := context.Background()

	hostID := uuid.New()
	roomName := "meeting-test-room"
	mtg := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Test",
		OrganizerID:    hostID,
		Status:         MeetingStatusInProgress,
		ScheduledStart: time.Now().Add(-2 * time.Hour),
		ScheduledEnd:   time.Now().Add(-1 * time.Hour),
		RoomName:       &roomName,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.CreateMeeting(ctx, mtg))
	repo.attendees[mtg.ID] = []MeetingAttendeeWithUser{{MeetingAttendee: MeetingAttendee{MeetingID: mtg.ID, UserID: hostID}}}

	_, err := svc.EndMeeting(ctx, mtg.ID, testTenantID)
	require.NoError(t, err)

	assert.True(t, repo.breakoutRoomsClosed, "CloseAllBreakoutRooms should have been called")
	assert.True(t, repo.breakoutAssignmentsCleared, "ClearAllBreakoutAssignments should have been called")
}
