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
	meetings    map[uuid.UUID]*Meeting
	attendees   map[uuid.UUID][]MeetingAttendee
	notes       map[uuid.UUID][]MeetingNotes // keyed by meeting ID
	actionItems map[uuid.UUID][]MeetingActionItem
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		meetings:    make(map[uuid.UUID]*Meeting),
		attendees:   make(map[uuid.UUID][]MeetingAttendee),
		notes:       make(map[uuid.UUID][]MeetingNotes),
		actionItems: make(map[uuid.UUID][]MeetingActionItem),
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
	m.attendees[meetingID] = append(m.attendees[meetingID], MeetingAttendee{
		MeetingID:  meetingID,
		UserID:     userID,
		RSVPStatus: MeetingRSVPPending,
	})
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

func (m *mockRepo) GetAttendees(_ context.Context, meetingID uuid.UUID) ([]MeetingAttendee, error) {
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

// --- Helper Functions ---

func newTestService() (*Service, *mockRepo) {
	repo := newMockRepo()
	svc := NewService(repo)
	return svc, repo
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
