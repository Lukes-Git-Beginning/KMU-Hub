package meeting

// Service-level coverage for the host controls, breakout rooms, attendee and
// action-item paths the lifecycle tests in service_test.go leave open
// (BACKLOG unit cov-work-meeting). Uses the mockRepo from service_test.go; the
// SQL behind these calls is covered by postgres_repository_db_test.go.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// flakyRepo wraps mockRepo and fails selected calls so the service's error
// wrapping and best-effort branches are exercised.
type flakyRepo struct {
	*mockRepo
	updateMeetingErr  error
	isCoHostErr       error
	listCoHostsErr    error
	addCoHostErr      error
	removeCoHostErr   error
	setLockedErr      error
	createBreakoutErr error
	closeBreakoutErr  error
	clearAllErr       error
	getAssignmentErr  error
}

func (f *flakyRepo) UpdateMeeting(ctx context.Context, m *Meeting) error {
	if f.updateMeetingErr != nil {
		return f.updateMeetingErr
	}
	return f.mockRepo.UpdateMeeting(ctx, m)
}

func (f *flakyRepo) IsCoHost(ctx context.Context, t, m, u uuid.UUID) (bool, error) {
	if f.isCoHostErr != nil {
		return false, f.isCoHostErr
	}
	return f.mockRepo.IsCoHost(ctx, t, m, u)
}

func (f *flakyRepo) ListCoHosts(ctx context.Context, t, m uuid.UUID) ([]MeetingCoHost, error) {
	if f.listCoHostsErr != nil {
		return nil, f.listCoHostsErr
	}
	return f.mockRepo.ListCoHosts(ctx, t, m)
}

func (f *flakyRepo) AddCoHost(ctx context.Context, t, m, u, by uuid.UUID) error {
	if f.addCoHostErr != nil {
		return f.addCoHostErr
	}
	return f.mockRepo.AddCoHost(ctx, t, m, u, by)
}

func (f *flakyRepo) RemoveCoHost(ctx context.Context, t, m, u uuid.UUID) error {
	if f.removeCoHostErr != nil {
		return f.removeCoHostErr
	}
	return f.mockRepo.RemoveCoHost(ctx, t, m, u)
}

func (f *flakyRepo) SetLocked(ctx context.Context, t, m uuid.UUID, locked bool) error {
	if f.setLockedErr != nil {
		return f.setLockedErr
	}
	return f.mockRepo.SetLocked(ctx, t, m, locked)
}

func (f *flakyRepo) CreateBreakoutRoom(ctx context.Context, r *BreakoutRoom) error {
	if f.createBreakoutErr != nil {
		return f.createBreakoutErr
	}
	return f.mockRepo.CreateBreakoutRoom(ctx, r)
}

func (f *flakyRepo) CloseAllBreakoutRooms(ctx context.Context, m, t uuid.UUID) ([]string, error) {
	if f.closeBreakoutErr != nil {
		return nil, f.closeBreakoutErr
	}
	return f.mockRepo.CloseAllBreakoutRooms(ctx, m, t)
}

func (f *flakyRepo) ClearAllBreakoutAssignments(ctx context.Context, m, t uuid.UUID) error {
	if f.clearAllErr != nil {
		return f.clearAllErr
	}
	return f.mockRepo.ClearAllBreakoutAssignments(ctx, m, t)
}

func (f *flakyRepo) GetBreakoutAssignmentForUser(ctx context.Context, m, u, t uuid.UUID) (*BreakoutRoom, error) {
	if f.getAssignmentErr != nil {
		return nil, f.getAssignmentErr
	}
	return f.mockRepo.GetBreakoutAssignmentForUser(ctx, m, u, t)
}

// scriptedRooms is a RoomManager that records every call and can fail per call.
type scriptedRooms struct {
	participants []string
	listErr      error
	muteErr      error
	removeErr    error
	muted        []string
	removed      []string
	deleteErr    error
	deleted      []string
}

func (s *scriptedRooms) DeleteRoom(_ context.Context, name string) error {
	s.deleted = append(s.deleted, name)
	return s.deleteErr
}

func (s *scriptedRooms) ListParticipants(context.Context, string) ([]string, error) {
	return s.participants, s.listErr
}

func (s *scriptedRooms) MuteParticipant(_ context.Context, _, identity string) error {
	s.muted = append(s.muted, identity)
	return s.muteErr
}

func (s *scriptedRooms) RemoveParticipant(_ context.Context, _, identity string) error {
	s.removed = append(s.removed, identity)
	return s.removeErr
}

func seedMeeting(t *testing.T, repo *mockRepo, organizer uuid.UUID, status string, room string) *Meeting {
	t.Helper()
	m := &Meeting{
		ID:             uuid.New(),
		TenantID:       testTenantID,
		Title:          "Host Controls",
		OrganizerID:    organizer,
		Status:         status,
		ScheduledStart: time.Now().Add(-time.Hour),
		ScheduledEnd:   time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if room != "" {
		m.RoomName = &room
	}
	require.NoError(t, repo.CreateMeeting(context.Background(), m))
	return m
}

func ptr[T any](v T) *T { return &v }

// --- UpdateMeeting ---

func TestUpdateMeeting_Validation(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestService()
	m := seedMeeting(t, repo, uuid.New(), MeetingStatusScheduled, "")
	m.ScheduledStart = time.Now().Add(time.Hour)
	m.ScheduledEnd = time.Now().Add(2 * time.Hour)

	_, err := svc.UpdateMeeting(ctx, m.ID, testTenantID, UpdateMeetingInput{Title: ptr("   ")})
	assert.ErrorIs(t, err, ErrTitleRequired)

	_, err = svc.UpdateMeeting(ctx, m.ID, testTenantID, UpdateMeetingInput{Title: ptr(strings.Repeat("x", 501))})
	assert.ErrorIs(t, err, ErrTitleTooLong)

	// Only the end moves: it must still be after the stored start.
	_, err = svc.UpdateMeeting(ctx, m.ID, testTenantID, UpdateMeetingInput{ScheduledEnd: ptr(m.ScheduledStart.Add(-time.Minute))})
	assert.ErrorIs(t, err, ErrInvalidTimeRange)

	// Only the start moves past the stored end.
	_, err = svc.UpdateMeeting(ctx, m.ID, testTenantID, UpdateMeetingInput{ScheduledStart: ptr(m.ScheduledEnd.Add(time.Minute))})
	assert.ErrorIs(t, err, ErrInvalidTimeRange)

	_, err = svc.UpdateMeeting(ctx, uuid.New(), testTenantID, UpdateMeetingInput{})
	assert.ErrorIs(t, err, ErrNotFound)

	// A rejected update must not have touched the stored row.
	stored, err := repo.GetMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, "Host Controls", stored.Title)
}

func TestUpdateMeeting_AppliesEveryField(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestService()
	m := seedMeeting(t, repo, uuid.New(), MeetingStatusScheduled, "")

	start := time.Now().Add(3 * time.Hour).UTC()
	end := start.Add(time.Hour)
	contact, deal := uuid.New(), uuid.New()

	got, err := svc.UpdateMeeting(ctx, m.ID, testTenantID, UpdateMeetingInput{
		Title:          ptr("  Renamed  "),
		Description:    ptr("desc"),
		Agenda:         ptr("agenda"),
		ScheduledStart: &start,
		ScheduledEnd:   &end,
		ContactID:      &contact,
		DealID:         &deal,
	})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.Title)

	stored, err := repo.GetMeeting(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", stored.Title)
	assert.Equal(t, "desc", *stored.Description)
	assert.Equal(t, "agenda", *stored.Agenda)
	assert.True(t, stored.ScheduledStart.Equal(start))
	assert.True(t, stored.ScheduledEnd.Equal(end))
	assert.Equal(t, contact, *stored.ContactID)
	assert.Equal(t, deal, *stored.DealID)
}

func TestUpdateMeeting_RepoErrorSurfaces(t *testing.T) {
	boom := errors.New("write failed")
	repo := &flakyRepo{mockRepo: newMockRepo(), updateMeetingErr: boom}
	svc := NewService(repo)
	m := seedMeeting(t, repo.mockRepo, uuid.New(), MeetingStatusScheduled, "")

	_, err := svc.UpdateMeeting(context.Background(), m.ID, testTenantID, UpdateMeetingInput{Title: ptr("New")})
	assert.ErrorIs(t, err, boom)
}

func TestDeleteMeeting_UnknownMeeting(t *testing.T) {
	svc, _ := newTestService()
	assert.ErrorIs(t, svc.DeleteMeeting(context.Background(), uuid.New(), testTenantID), ErrNotFound)
}

func TestDeleteMeeting_CompletedKeepsRow(t *testing.T) {
	svc, repo := newTestService()
	m := seedMeeting(t, repo, uuid.New(), MeetingStatusCompleted, "")

	assert.ErrorIs(t, svc.DeleteMeeting(context.Background(), m.ID, testTenantID), ErrCannotDelete)
	_, err := repo.GetMeeting(context.Background(), m.ID, testTenantID)
	assert.NoError(t, err, "a completed meeting must survive a rejected delete")
}

// --- EndMeeting / ListStaleMeetings ---

func TestEndMeeting_UnknownMeeting(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.EndMeeting(context.Background(), uuid.New(), testTenantID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestEndMeeting_UpdateFailureKeepsRoomAlive(t *testing.T) {
	boom := errors.New("write failed")
	repo := &flakyRepo{mockRepo: newMockRepo(), updateMeetingErr: boom}
	rooms := &scriptedRooms{}
	svc := NewServiceWithRoomManager(repo, rooms)
	m := seedMeeting(t, repo.mockRepo, uuid.New(), MeetingStatusInProgress, "room-a")

	_, err := svc.EndMeeting(context.Background(), m.ID, testTenantID)
	require.ErrorIs(t, err, boom)
	assert.Empty(t, rooms.deleted, "the LiveKit room must not be deleted while the meeting stays in_progress")
	assert.False(t, repo.breakoutRoomsClosed)
}

func TestEndMeeting_RoomDeleteAndBreakoutCleanupFailuresAreNonFatal(t *testing.T) {
	repo := &flakyRepo{
		mockRepo:         newMockRepo(),
		closeBreakoutErr: errors.New("close failed"),
		clearAllErr:      errors.New("clear failed"),
	}
	rooms := &scriptedRooms{deleteErr: errors.New("livekit down")}
	svc := NewServiceWithRoomManager(repo, rooms)
	m := seedMeeting(t, repo.mockRepo, uuid.New(), MeetingStatusInProgress, "room-b")

	summary, err := svc.EndMeeting(context.Background(), m.ID, testTenantID)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, []string{"room-b"}, rooms.deleted)

	stored, err := repo.GetMeeting(context.Background(), m.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, MeetingStatusCompleted, stored.Status)
	assert.NotNil(t, stored.ActualEnd)
}

func TestService_ListStaleMeetings(t *testing.T) {
	svc, repo := newTestService()
	stale := seedMeeting(t, repo, uuid.New(), MeetingStatusInProgress, "")
	stale.ScheduledEnd = time.Now().Add(-3 * time.Hour)
	seedMeeting(t, repo, uuid.New(), MeetingStatusScheduled, "").ScheduledEnd = time.Now().Add(-3 * time.Hour)

	got, err := svc.ListStaleMeetings(context.Background(), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, stale.ID, got[0].ID)
}

// --- Action items ---

func TestUpdateActionItem_AppliesFieldsAndGuardsDescription(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestService()
	m := seedMeeting(t, repo, uuid.New(), MeetingStatusInProgress, "")
	item, err := svc.CreateActionItem(ctx, CreateActionItemInput{MeetingID: m.ID, TenantID: testTenantID, Description: "Original"})
	require.NoError(t, err)

	assignee := uuid.New()
	got, err := svc.UpdateActionItem(ctx, item.ID, testTenantID, UpdateActionItemInput{
		Description: ptr("  Rewritten  "),
		AssigneeID:  &assignee,
		IsCompleted: ptr(true),
		SortOrder:   ptr(7),
	})
	require.NoError(t, err)
	assert.Equal(t, "Rewritten", got.Description)

	stored, err := repo.GetActionItemByID(ctx, item.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, "Rewritten", stored.Description)
	assert.Equal(t, assignee, *stored.AssigneeID)
	assert.True(t, stored.IsCompleted)
	assert.Equal(t, 7, stored.SortOrder)

	// A blank description is refused and leaves the stored text alone.
	_, err = svc.UpdateActionItem(ctx, item.ID, testTenantID, UpdateActionItemInput{Description: ptr("   ")})
	assert.ErrorIs(t, err, ErrActionDescRequired)
	stored, err = repo.GetActionItemByID(ctx, item.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, "Rewritten", stored.Description)

	_, err = svc.UpdateActionItem(ctx, uuid.New(), testTenantID, UpdateActionItemInput{IsCompleted: ptr(true)})
	assert.ErrorIs(t, err, ErrActionItemNotFound)
}

func TestGetMeetingIDForActionItem(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestService()
	m := seedMeeting(t, repo, uuid.New(), MeetingStatusInProgress, "")
	item, err := svc.CreateActionItem(ctx, CreateActionItemInput{MeetingID: m.ID, TenantID: testTenantID, Description: "Follow up"})
	require.NoError(t, err)

	id, err := svc.GetMeetingIDForActionItem(ctx, item.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, m.ID, id)

	id, err = svc.GetMeetingIDForActionItem(ctx, uuid.New(), testTenantID)
	assert.ErrorIs(t, err, ErrActionItemNotFound)
	assert.Equal(t, uuid.Nil, id)
}

func TestListActionItems_UnknownMeetingVsAllMeetings(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestService()
	m := seedMeeting(t, repo, uuid.New(), MeetingStatusInProgress, "")
	_, err := svc.CreateActionItem(ctx, CreateActionItemInput{MeetingID: m.ID, TenantID: testTenantID, Description: "One"})
	require.NoError(t, err)

	_, err = svc.ListActionItems(ctx, uuid.New(), testTenantID)
	assert.ErrorIs(t, err, ErrNotFound, "a specific but unknown meeting must be a 404, not an empty list")

	// uuid.Nil means "all meetings of the tenant": no existence check, straight to the repo.
	_, err = svc.ListActionItems(ctx, uuid.Nil, testTenantID)
	assert.NoError(t, err)

	items, err := svc.ListActionItems(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Len(t, items, 1)
}

func TestConvertActionItemsToTasks_UnknownMeeting(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.ConvertActionItemsToTasks(context.Background(), uuid.New(), testTenantID)
	assert.ErrorIs(t, err, ErrNotFound)
}

// --- Attendees ---

func TestAttendees_ServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestService()
	m := seedMeeting(t, repo, uuid.New(), MeetingStatusScheduled, "")
	guest := uuid.New()

	require.NoError(t, svc.AddAttendee(ctx, m.ID, guest, testTenantID))
	require.Len(t, repo.attendees[m.ID], 1)
	assert.Equal(t, MeetingRSVPPending, repo.attendees[m.ID][0].RSVPStatus)

	require.NoError(t, svc.UpdateAttendeeRSVP(ctx, m.ID, guest, MeetingRSVPAccepted))
	assert.Equal(t, MeetingRSVPAccepted, repo.attendees[m.ID][0].RSVPStatus)

	assert.ErrorIs(t, svc.UpdateAttendeeRSVP(ctx, m.ID, uuid.New(), MeetingRSVPAccepted), ErrNotFound,
		"RSVP for a user who is not invited must not silently succeed")

	require.NoError(t, svc.RemoveAttendee(ctx, m.ID, guest))
	assert.Empty(t, repo.attendees[m.ID])
}

func TestAddAttendee_UnknownOrForeignTenantMeeting(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestService()
	m := seedMeeting(t, repo, uuid.New(), MeetingStatusScheduled, "")

	assert.ErrorIs(t, svc.AddAttendee(ctx, uuid.New(), uuid.New(), testTenantID), ErrNotFound)
	assert.ErrorIs(t, svc.AddAttendee(ctx, m.ID, uuid.New(), uuid.New()), ErrNotFound,
		"a meeting of another tenant must look like it does not exist")
	assert.Empty(t, repo.attendees[m.ID])
}

// --- IsHostOrCoHost / co-host management ---

func TestIsHostOrCoHost(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestService()
	organizer, coHost, stranger := uuid.New(), uuid.New(), uuid.New()
	m := seedMeeting(t, repo, organizer, MeetingStatusInProgress, "")
	require.NoError(t, repo.AddCoHost(ctx, testTenantID, m.ID, coHost, organizer))

	for name, tc := range map[string]struct {
		user uuid.UUID
		want bool
	}{"organizer": {organizer, true}, "co-host": {coHost, true}, "stranger": {stranger, false}} {
		t.Run(name, func(t *testing.T) {
			got, err := svc.IsHostOrCoHost(ctx, m.ID, testTenantID, tc.user)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	_, err := svc.IsHostOrCoHost(ctx, uuid.New(), testTenantID, organizer)
	assert.ErrorIs(t, err, ErrNotFound)

	got, err := svc.IsHostOrCoHost(ctx, m.ID, uuid.New(), organizer)
	assert.ErrorIs(t, err, ErrNotFound, "another tenant must not learn who hosts the meeting")
	assert.False(t, got)

	boom := errors.New("db down")
	flaky := &flakyRepo{mockRepo: repo, isCoHostErr: boom}
	_, err = NewService(flaky).IsHostOrCoHost(ctx, m.ID, testTenantID, stranger)
	assert.ErrorIs(t, err, boom)
}

func TestCoHostManagement_ErrorPaths(t *testing.T) {
	ctx := context.Background()
	organizer, target := uuid.New(), uuid.New()

	svc, repo := newTestService()
	m := seedMeeting(t, repo, organizer, MeetingStatusInProgress, "")

	assert.ErrorIs(t, svc.PromoteCoHost(ctx, uuid.New(), testTenantID, organizer, target), ErrNotFound)
	assert.ErrorIs(t, svc.DemoteCoHost(ctx, uuid.New(), testTenantID, organizer, target), ErrNotFound)
	_, err := svc.ListCoHosts(ctx, uuid.New(), testTenantID)
	assert.ErrorIs(t, err, ErrNotFound)

	boom := errors.New("db down")
	flaky := &flakyRepo{mockRepo: repo, addCoHostErr: boom, removeCoHostErr: boom}
	fsvc := NewService(flaky)
	err = fsvc.PromoteCoHost(ctx, m.ID, testTenantID, organizer, target)
	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "promote co-host")
	err = fsvc.DemoteCoHost(ctx, m.ID, testTenantID, organizer, target)
	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "demote co-host")

	require.NoError(t, svc.PromoteCoHost(ctx, m.ID, testTenantID, organizer, target))
	hosts, err := svc.ListCoHosts(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, target, hosts[0].UserID)
}

// --- Mute / kick / lock ---

func TestMuteAll_SkipsOrganizerAndCoHosts(t *testing.T) {
	ctx := context.Background()
	organizer, coHost, guestA, guestB := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	rooms := &scriptedRooms{participants: []string{organizer.String(), coHost.String(), guestA.String(), guestB.String()}}
	repo := newMockRepo()
	svc := NewServiceWithRoomManager(repo, rooms)
	m := seedMeeting(t, repo, organizer, MeetingStatusInProgress, "room-mute")
	require.NoError(t, repo.AddCoHost(ctx, testTenantID, m.ID, coHost, organizer))

	require.NoError(t, svc.MuteAllMeetingParticipants(ctx, m.ID, testTenantID, coHost), "a co-host may mute everyone")
	assert.ElementsMatch(t, []string{guestA.String(), guestB.String()}, rooms.muted)
}

func TestMuteAll_Guards(t *testing.T) {
	ctx := context.Background()
	organizer, guest := uuid.New(), uuid.New()
	rooms := &scriptedRooms{participants: []string{guest.String()}}
	repo := newMockRepo()
	svc := NewServiceWithRoomManager(repo, rooms)

	scheduled := seedMeeting(t, repo, organizer, MeetingStatusScheduled, "room-1")
	live := seedMeeting(t, repo, organizer, MeetingStatusInProgress, "room-2")

	assert.ErrorIs(t, svc.MuteAllMeetingParticipants(ctx, uuid.New(), testTenantID, organizer), ErrNotFound)
	assert.ErrorIs(t, svc.MuteAllMeetingParticipants(ctx, scheduled.ID, testTenantID, organizer), ErrNotInProgress)
	assert.ErrorIs(t, svc.MuteAllMeetingParticipants(ctx, live.ID, testTenantID, guest), ErrNotHost)
	assert.Empty(t, rooms.muted, "no guard failure may reach LiveKit")
}

func TestMuteAll_NoRoomManagerOrNoRoomIsNoop(t *testing.T) {
	ctx := context.Background()
	organizer := uuid.New()

	svc, repo := newTestService() // no room manager
	m := seedMeeting(t, repo, organizer, MeetingStatusInProgress, "room-x")
	assert.NoError(t, svc.MuteAllMeetingParticipants(ctx, m.ID, testTenantID, organizer))

	rooms := &scriptedRooms{}
	repo2 := newMockRepo()
	svc2 := NewServiceWithRoomManager(repo2, rooms)
	noRoom := seedMeeting(t, repo2, organizer, MeetingStatusInProgress, "")
	assert.NoError(t, svc2.MuteAllMeetingParticipants(ctx, noRoom.ID, testTenantID, organizer))
	assert.Empty(t, rooms.muted)
}

func TestMuteAll_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	organizer, guestA, guestB := uuid.New(), uuid.New(), uuid.New()

	// Listing the room fails: the error surfaces.
	rooms := &scriptedRooms{listErr: errors.New("livekit down")}
	repo := newMockRepo()
	m := seedMeeting(t, repo, organizer, MeetingStatusInProgress, "room-e")
	err := NewServiceWithRoomManager(repo, rooms).MuteAllMeetingParticipants(ctx, m.ID, testTenantID, organizer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list participants")

	// Listing co-hosts fails: nobody may be muted blindly.
	rooms = &scriptedRooms{participants: []string{guestA.String()}}
	flaky := &flakyRepo{mockRepo: repo, listCoHostsErr: errors.New("db down")}
	err = NewServiceWithRoomManager(flaky, rooms).MuteAllMeetingParticipants(ctx, m.ID, testTenantID, organizer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list co-hosts")
	assert.Empty(t, rooms.muted)

	// One mute fails: the others are still muted.
	rooms = &scriptedRooms{participants: []string{guestA.String(), guestB.String()}, muteErr: errors.New("gone")}
	require.NoError(t, NewServiceWithRoomManager(repo, rooms).MuteAllMeetingParticipants(ctx, m.ID, testTenantID, organizer))
	assert.Len(t, rooms.muted, 2, "a failing participant must not stop the sweep")
}

func TestMuteParticipant_Guards(t *testing.T) {
	ctx := context.Background()
	organizer, guest := uuid.New(), uuid.New()
	rooms := &scriptedRooms{}
	repo := newMockRepo()
	svc := NewServiceWithRoomManager(repo, rooms)
	scheduled := seedMeeting(t, repo, organizer, MeetingStatusScheduled, "r1")
	live := seedMeeting(t, repo, organizer, MeetingStatusInProgress, "r2")

	assert.ErrorIs(t, svc.MuteMeetingParticipant(ctx, uuid.New(), testTenantID, organizer, guest), ErrNotFound)
	assert.ErrorIs(t, svc.MuteMeetingParticipant(ctx, scheduled.ID, testTenantID, organizer, guest), ErrNotInProgress)

	boom := errors.New("db down")
	flaky := &flakyRepo{mockRepo: repo, isCoHostErr: boom}
	assert.ErrorIs(t, NewServiceWithRoomManager(flaky, rooms).MuteMeetingParticipant(ctx, live.ID, testTenantID, guest, organizer), boom)

	rooms.muteErr = errors.New("livekit down")
	err := svc.MuteMeetingParticipant(ctx, live.ID, testTenantID, organizer, guest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mute participant")

	// Without a room manager (or before the room exists) a mute is a logged no-op, not an error.
	bare, bareRepo := newTestService()
	m := seedMeeting(t, bareRepo, organizer, MeetingStatusInProgress, "")
	assert.NoError(t, bare.MuteMeetingParticipant(ctx, m.ID, testTenantID, organizer, guest))
}

func TestRemoveMeetingParticipant_Guards(t *testing.T) {
	ctx := context.Background()
	organizer, guest := uuid.New(), uuid.New()
	rooms := &scriptedRooms{}
	repo := newMockRepo()
	svc := NewServiceWithRoomManager(repo, rooms)
	scheduled := seedMeeting(t, repo, organizer, MeetingStatusScheduled, "r1")
	live := seedMeeting(t, repo, organizer, MeetingStatusInProgress, "r2")

	assert.ErrorIs(t, svc.RemoveMeetingParticipant(ctx, uuid.New(), testTenantID, organizer, guest), ErrNotFound)
	assert.ErrorIs(t, svc.RemoveMeetingParticipant(ctx, scheduled.ID, testTenantID, organizer, guest), ErrNotInProgress)

	boom := errors.New("db down")
	flaky := &flakyRepo{mockRepo: repo, isCoHostErr: boom}
	assert.ErrorIs(t, NewServiceWithRoomManager(flaky, rooms).RemoveMeetingParticipant(ctx, live.ID, testTenantID, guest, organizer), boom)
	assert.Empty(t, rooms.removed)

	rooms.removeErr = errors.New("livekit down")
	err := svc.RemoveMeetingParticipant(ctx, live.ID, testTenantID, organizer, guest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove participant")

	bare, bareRepo := newTestService()
	m := seedMeeting(t, bareRepo, organizer, MeetingStatusInProgress, "")
	assert.NoError(t, bare.RemoveMeetingParticipant(ctx, m.ID, testTenantID, organizer, guest))
}

func TestSetMeetingLock_ErrorPaths(t *testing.T) {
	ctx := context.Background()
	organizer, guest := uuid.New(), uuid.New()
	svc, repo := newTestService()
	m := seedMeeting(t, repo, organizer, MeetingStatusScheduled, "")

	assert.ErrorIs(t, svc.SetMeetingLock(ctx, uuid.New(), testTenantID, organizer, true), ErrNotFound)

	boom := errors.New("db down")
	assert.ErrorIs(t, NewService(&flakyRepo{mockRepo: repo, isCoHostErr: boom}).SetMeetingLock(ctx, m.ID, testTenantID, guest, true), boom)

	err := NewService(&flakyRepo{mockRepo: repo, setLockedErr: boom}).SetMeetingLock(ctx, m.ID, testTenantID, organizer, true)
	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "set meeting lock")
	assert.False(t, repo.meetings[m.ID].Locked, "a failed write must not leave the meeting locked")
}

// --- Breakout rooms ---

func TestCreateBreakoutRooms_LabelsAndPersistence(t *testing.T) {
	ctx := context.Background()
	host, coHost := uuid.New(), uuid.New()
	svc, repo := newTestService()
	m := seedMeeting(t, repo, host, MeetingStatusInProgress, "")
	require.NoError(t, repo.AddCoHost(ctx, testTenantID, m.ID, coHost, host))

	rooms, err := svc.CreateBreakoutRooms(ctx, m.ID, testTenantID, coHost, 3, []string{"Vertrieb", ""})
	require.NoError(t, err)
	require.Len(t, rooms, 3)
	assert.Equal(t, []string{"Vertrieb", "Raum 2", "Raum 3"}, []string{rooms[0].Label, rooms[1].Label, rooms[2].Label},
		"custom labels win, blanks and missing labels fall back to 'Raum n'")

	seen := map[string]bool{}
	for i, r := range rooms {
		assert.Equal(t, i, r.SortIndex)
		assert.Equal(t, "open", r.Status)
		assert.Equal(t, coHost, r.CreatedBy)
		assert.Equal(t, m.ID, r.MeetingID)
		assert.True(t, strings.HasPrefix(r.RoomName, "meeting-"+m.ID.String()+"-breakout-"))
		assert.False(t, seen[r.RoomName], "room names must be unique")
		seen[r.RoomName] = true
	}

	listed, err := svc.ListBreakoutRooms(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	assert.Len(t, listed, 3)
}

func TestBreakoutRooms_ErrorPaths(t *testing.T) {
	ctx := context.Background()
	host, stranger := uuid.New(), uuid.New()
	repo := newMockRepo()
	svc := NewService(repo)
	m := seedMeeting(t, repo, host, MeetingStatusInProgress, "")
	boom := errors.New("db down")

	_, err := svc.CreateBreakoutRooms(ctx, uuid.New(), testTenantID, host, 2, nil)
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = NewService(&flakyRepo{mockRepo: repo, isCoHostErr: boom}).CreateBreakoutRooms(ctx, m.ID, testTenantID, stranger, 2, nil)
	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "check host")

	_, err = NewService(&flakyRepo{mockRepo: repo, createBreakoutErr: boom}).CreateBreakoutRooms(ctx, m.ID, testTenantID, host, 2, nil)
	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "create breakout room 1")

	_, err = svc.ListBreakoutRooms(ctx, uuid.New(), testTenantID)
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = svc.GetBreakoutAssignment(ctx, uuid.New(), testTenantID, host)
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = svc.JoinBreakoutRoom(ctx, uuid.New(), testTenantID, host)
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = NewService(&flakyRepo{mockRepo: repo, getAssignmentErr: boom}).GetBreakoutAssignment(ctx, m.ID, testTenantID, host)
	require.ErrorIs(t, err, boom)
	_, err = NewService(&flakyRepo{mockRepo: repo, getAssignmentErr: boom}).JoinBreakoutRoom(ctx, m.ID, testTenantID, host)
	require.ErrorIs(t, err, boom)
}

func TestBreakoutAssignment_AssignJoinReturnClose(t *testing.T) {
	ctx := context.Background()
	host, coHost, guest, other := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	svc, repo := newTestService()
	m := seedMeeting(t, repo, host, MeetingStatusInProgress, "")
	require.NoError(t, repo.AddCoHost(ctx, testTenantID, m.ID, coHost, host))

	rooms, err := svc.CreateBreakoutRooms(ctx, m.ID, testTenantID, host, 2, nil)
	require.NoError(t, err)

	// Nobody is assigned yet.
	none, err := svc.GetBreakoutAssignment(ctx, m.ID, testTenantID, guest)
	require.NoError(t, err)
	assert.Nil(t, none)

	// A co-host assigns; the guest then resolves to exactly that room.
	require.NoError(t, svc.AssignBreakoutParticipant(ctx, m.ID, testTenantID, coHost, guest, &rooms[1].ID))
	got, err := svc.JoinBreakoutRoom(ctx, m.ID, testTenantID, guest)
	require.NoError(t, err)
	assert.Equal(t, rooms[1].ID, got.ID)
	assert.Equal(t, rooms[1].RoomName, got.RoomName)

	// Re-assigning moves the guest rather than duplicating the row.
	require.NoError(t, svc.AssignBreakoutParticipant(ctx, m.ID, testTenantID, host, guest, &rooms[0].ID))
	got, err = svc.GetBreakoutAssignment(ctx, m.ID, testTenantID, guest)
	require.NoError(t, err)
	assert.Equal(t, rooms[0].ID, got.ID)

	// A nil room unassigns.
	require.NoError(t, svc.AssignBreakoutParticipant(ctx, m.ID, testTenantID, host, guest, nil))
	_, err = svc.JoinBreakoutRoom(ctx, m.ID, testTenantID, guest)
	assert.ErrorIs(t, err, ErrNoBreakoutAssignment)

	assert.ErrorIs(t, svc.AssignBreakoutParticipant(ctx, uuid.New(), testTenantID, host, guest, nil), ErrNotFound)

	// ReturnToMainRoom: own assignment needs no host rights...
	require.NoError(t, svc.AssignBreakoutParticipant(ctx, m.ID, testTenantID, host, guest, &rooms[0].ID))
	require.NoError(t, svc.AssignBreakoutParticipant(ctx, m.ID, testTenantID, host, other, &rooms[0].ID))
	require.NoError(t, svc.ReturnToMainRoom(ctx, m.ID, testTenantID, guest, nil))
	_, err = svc.JoinBreakoutRoom(ctx, m.ID, testTenantID, guest)
	assert.ErrorIs(t, err, ErrNoBreakoutAssignment)

	// ...pulling someone else back does.
	assert.ErrorIs(t, svc.ReturnToMainRoom(ctx, m.ID, testTenantID, guest, &other), ErrBreakoutNotAuthorized)
	still, err := svc.GetBreakoutAssignment(ctx, m.ID, testTenantID, other)
	require.NoError(t, err)
	require.NotNil(t, still, "a rejected force-return must leave the assignment in place")
	require.NoError(t, svc.ReturnToMainRoom(ctx, m.ID, testTenantID, coHost, &other))
	gone, err := svc.GetBreakoutAssignment(ctx, m.ID, testTenantID, other)
	require.NoError(t, err)
	assert.Nil(t, gone)

	assert.ErrorIs(t, svc.ReturnToMainRoom(ctx, uuid.New(), testTenantID, guest, nil), ErrNotFound)

	// Closing is host-only, closes every room and drops the assignments.
	require.NoError(t, svc.AssignBreakoutParticipant(ctx, m.ID, testTenantID, host, other, &rooms[1].ID))
	assert.ErrorIs(t, svc.CloseBreakoutRooms(ctx, m.ID, testTenantID, guest), ErrBreakoutNotAuthorized)
	require.NoError(t, svc.CloseBreakoutRooms(ctx, m.ID, testTenantID, coHost))
	listed, err := svc.ListBreakoutRooms(ctx, m.ID, testTenantID)
	require.NoError(t, err)
	for _, r := range listed {
		assert.Equal(t, "closed", r.Status)
	}
	cleared, err := svc.GetBreakoutAssignment(ctx, m.ID, testTenantID, other)
	require.NoError(t, err)
	assert.Nil(t, cleared)
}

func TestCloseBreakoutRooms_ErrorPaths(t *testing.T) {
	ctx := context.Background()
	host, stranger := uuid.New(), uuid.New()
	repo := newMockRepo()
	m := seedMeeting(t, repo, host, MeetingStatusInProgress, "")
	boom := errors.New("db down")

	assert.ErrorIs(t, NewService(repo).CloseBreakoutRooms(ctx, uuid.New(), testTenantID, host), ErrNotFound)

	err := NewService(&flakyRepo{mockRepo: repo, isCoHostErr: boom}).CloseBreakoutRooms(ctx, m.ID, testTenantID, stranger)
	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "check host")

	err = NewService(&flakyRepo{mockRepo: repo, closeBreakoutErr: boom}).CloseBreakoutRooms(ctx, m.ID, testTenantID, host)
	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "close breakout rooms")

	// Clearing the assignments is best-effort: the rooms are closed, the call still succeeds.
	assert.NoError(t, NewService(&flakyRepo{mockRepo: repo, clearAllErr: boom}).CloseBreakoutRooms(ctx, m.ID, testTenantID, host))

	// Host-check failures also guard assign and return.
	flaky := NewService(&flakyRepo{mockRepo: repo, isCoHostErr: boom})
	roomID := uuid.New()
	require.ErrorIs(t, flaky.AssignBreakoutParticipant(ctx, m.ID, testTenantID, stranger, uuid.New(), &roomID), boom)
	target := uuid.New()
	require.ErrorIs(t, flaky.ReturnToMainRoom(ctx, m.ID, testTenantID, stranger, &target), boom)
}
