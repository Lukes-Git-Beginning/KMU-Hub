package preference

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Evaluate Pipeline Tests
// ============================================================================

func TestEvaluateUrgentBypassesAllFilters(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	event := models.EventPayload{
		Type:       "system.alert",
		Priority:   models.PriorityUrgent,
		ModuleID:   "system",
		ResourceID: "resource-1",
	}

	// Even with muted resource
	repo.mutes = []models.NotificationMute{
		{UserID: userID, ModuleID: "system", ResourceID: "resource-1"},
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.True(t, decision.Deliver)
	assert.True(t, decision.InApp)
	assert.True(t, decision.DesktopPush)
	assert.Equal(t, "urgent", decision.Sound)
	assert.Contains(t, decision.Reason, "urgent")
}

func TestEvaluateResourceMuted(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	event := models.EventPayload{
		Type:       "chat.channel.message",
		Priority:   models.PriorityNormal,
		ModuleID:   "chat",
		ResourceID: "channel-abc",
	}

	repo.mutes = []models.NotificationMute{
		{UserID: userID, ModuleID: "chat", ResourceID: "channel-abc"},
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.False(t, decision.Deliver)
	assert.Equal(t, "resource muted", decision.Reason)
}

func TestEvaluateEventTypePreference(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	eventKey := "chat.mention"
	event := models.EventPayload{
		Type:     eventKey,
		Priority: models.PriorityNormal,
		ModuleID: "chat",
	}

	repo.prefs = []models.NotificationPreference{
		{
			UserID:       userID,
			EventTypeKey: &eventKey,
			InApp:        true,
			DesktopPush:  false,
			Email:        false,
			SMS:          true,
			Sound:        "subtle",
		},
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.True(t, decision.Deliver)
	assert.True(t, decision.InApp)
	assert.False(t, decision.DesktopPush)
	assert.False(t, decision.Email)
	assert.True(t, decision.SMS)
	assert.Equal(t, "subtle", decision.Sound)
}

func TestEvaluateModuleDefault(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	chatModule := "chat"
	event := models.EventPayload{
		Type:     "chat.channel.message",
		Priority: models.PriorityNormal,
		ModuleID: "chat",
	}

	// No event-type preference, but module default exists
	repo.prefs = []models.NotificationPreference{
		{
			UserID:      userID,
			ModuleID:    &chatModule,
			InApp:       true,
			DesktopPush: true,
			Sound:       "module-sound",
		},
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.True(t, decision.Deliver)
	assert.True(t, decision.InApp)
	assert.True(t, decision.DesktopPush)
	assert.Equal(t, "module-sound", decision.Sound)
}

func TestEvaluateSystemDefaultWhenNoPrefs(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	event := models.EventPayload{
		Type:     "chat.mention",
		Priority: models.PriorityNormal,
		ModuleID: "chat",
	}

	// No preferences at all
	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.True(t, decision.Deliver)
	assert.True(t, decision.InApp)
	assert.True(t, decision.DesktopPush)
	assert.True(t, decision.Email)
	assert.False(t, decision.SMS)
	assert.Equal(t, "default", decision.Sound)
	assert.Equal(t, "system default", decision.Reason)
}

func TestEvaluateLowPriorityNeverDesktopPush(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	event := models.EventPayload{
		Type:     "chat.channel.message",
		Priority: models.PriorityLow,
		ModuleID: "chat",
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.True(t, decision.Deliver)
	assert.True(t, decision.InApp)
	assert.False(t, decision.DesktopPush)
	assert.Empty(t, decision.Sound)
	assert.Contains(t, decision.Reason, "low priority")
}

func TestEvaluateLowPriorityIgnoresDesktopPushPreference(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	eventKey := "chat.channel.message"
	event := models.EventPayload{
		Type:     eventKey,
		Priority: models.PriorityLow,
		ModuleID: "chat",
	}

	// User explicitly enables desktop push, but low priority overrides
	repo.prefs = []models.NotificationPreference{
		{
			UserID:       userID,
			EventTypeKey: &eventKey,
			InApp:        true,
			DesktopPush:  true,
			Sound:        "default",
		},
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.False(t, decision.DesktopPush)
}

func TestEvaluateAllChannelsDisabled(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	eventKey := "chat.channel.message"
	event := models.EventPayload{
		Type:     eventKey,
		Priority: models.PriorityNormal,
		ModuleID: "chat",
	}

	// User disables both in_app and desktop_push
	repo.prefs = []models.NotificationPreference{
		{
			UserID:       userID,
			EventTypeKey: &eventKey,
			InApp:        false,
			DesktopPush:  false,
			Sound:        "",
		},
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.False(t, decision.Deliver)
	assert.Contains(t, decision.Reason, "all delivery channels disabled")
}

func TestEvaluateEmailOnlyChannelStillDelivers(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	eventKey := "chat.channel.message"
	event := models.EventPayload{
		Type:     eventKey,
		Priority: models.PriorityNormal,
		ModuleID: "chat",
	}

	// User turned off in_app and desktop_push but kept email on -- the
	// notification must still be marked deliverable so a future email
	// callback is not silently suppressed by this decision.
	repo.prefs = []models.NotificationPreference{
		{
			UserID:       userID,
			EventTypeKey: &eventKey,
			InApp:        false,
			DesktopPush:  false,
			Email:        true,
			SMS:          false,
			Sound:        "",
		},
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.True(t, decision.Deliver)
	assert.True(t, decision.Email)
}

func TestEvaluateAllChannelsIncludingEmailSMSDisabled(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	eventKey := "chat.channel.message"
	event := models.EventPayload{
		Type:     eventKey,
		Priority: models.PriorityNormal,
		ModuleID: "chat",
	}

	// Only email/sms explicitly set true would previously have been ignored --
	// disabling every channel including the new ones must still suppress delivery.
	repo.prefs = []models.NotificationPreference{
		{
			UserID:       userID,
			EventTypeKey: &eventKey,
			InApp:        false,
			DesktopPush:  false,
			Email:        false,
			SMS:          false,
			Sound:        "",
		},
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.False(t, decision.Deliver)
	assert.Contains(t, decision.Reason, "all delivery channels disabled")
}

func TestEvaluateQuietHoursActiveNormalPriority(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	event := models.EventPayload{
		Type:     "chat.mention",
		Priority: models.PriorityNormal,
		ModuleID: "chat",
	}

	// Set quiet hours to cover the current time using manual DND
	repo.quietHours = &models.QuietHours{
		UserID:     userID,
		ManualDND:  true,
		DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7},
		Enabled:    false,
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.True(t, decision.Deliver)
	assert.True(t, decision.InApp)
	assert.False(t, decision.DesktopPush)
	assert.Empty(t, decision.Sound)
	assert.Contains(t, decision.Reason, "quiet hours")
}

func TestEvaluateUrgentIgnoresQuietHours(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	event := models.EventPayload{
		Type:     "system.alert",
		Priority: models.PriorityUrgent,
		ModuleID: "system",
	}

	repo.quietHours = &models.QuietHours{
		UserID:    userID,
		ManualDND: true,
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.True(t, decision.Deliver)
	assert.True(t, decision.DesktopPush)
	assert.Equal(t, "urgent", decision.Sound)
}

func TestEvaluateNoResourceIDSkipsMuteCheck(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	event := models.EventPayload{
		Type:       "system.alert",
		Priority:   models.PriorityNormal,
		ModuleID:   "system",
		ResourceID: "", // No resource ID
	}

	decision, err := svc.Evaluate(context.Background(), uuid.New(), userID, event)
	require.NoError(t, err)
	assert.True(t, decision.Deliver)
}

// ============================================================================
// isInQuietHours Tests
// ============================================================================

func TestIsInQuietHoursNoConfig(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	result, err := svc.isInQuietHours(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.False(t, result)
}

func TestIsInQuietHoursManualDNDIndefinite(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	repo.quietHours = &models.QuietHours{
		UserID:    userID,
		ManualDND: true,
	}

	result, err := svc.isInQuietHours(context.Background(), uuid.New(), userID)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestIsInQuietHoursManualDNDTimedActive(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	future := time.Now().Add(1 * time.Hour)
	repo.quietHours = &models.QuietHours{
		UserID:         userID,
		ManualDND:      true,
		ManualDNDUntil: &future,
	}

	result, err := svc.isInQuietHours(context.Background(), uuid.New(), userID)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestIsInQuietHoursManualDNDTimedExpired(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	past := time.Now().Add(-1 * time.Hour)
	repo.quietHours = &models.QuietHours{
		UserID:         userID,
		ManualDND:      true,
		ManualDNDUntil: &past,
	}

	result, err := svc.isInQuietHours(context.Background(), uuid.New(), userID)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestIsInQuietHoursDisabled(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	repo.quietHours = &models.QuietHours{
		UserID:     userID,
		Enabled:    false,
		ManualDND:  false,
		StartTime:  "00:00",
		EndTime:    "23:59",
		Timezone:   "Europe/Berlin",
		DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7},
	}

	result, err := svc.isInQuietHours(context.Background(), uuid.New(), userID)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestIsInQuietHoursOvernightWindow(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	// Overnight window: 22:00 - 06:00
	// This test checks the parsing logic works with overnight windows
	repo.quietHours = &models.QuietHours{
		UserID:     userID,
		Enabled:    true,
		StartTime:  "22:00",
		EndTime:    "06:00",
		Timezone:   "UTC",
		DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7},
	}

	// This test primarily validates that the overnight parsing logic
	// doesn't error - the actual result depends on current time
	_, err := svc.isInQuietHours(context.Background(), uuid.New(), userID)
	require.NoError(t, err)
}

func TestIsInQuietHoursInvalidTimezone(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	repo.quietHours = &models.QuietHours{
		UserID:     userID,
		Enabled:    true,
		StartTime:  "18:00",
		EndTime:    "08:00",
		Timezone:   "Invalid/Timezone",
		DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7},
	}

	_, err := svc.isInQuietHours(context.Background(), uuid.New(), userID)
	assert.ErrorIs(t, err, ErrInvalidTimezone)
}

func TestIsInQuietHoursInvalidTimeFormat(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	userID := uuid.New()
	repo.quietHours = &models.QuietHours{
		UserID:     userID,
		Enabled:    true,
		StartTime:  "bad-time",
		EndTime:    "08:00",
		Timezone:   "UTC",
		DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7},
	}

	_, err := svc.isInQuietHours(context.Background(), uuid.New(), userID)
	assert.ErrorIs(t, err, ErrInvalidTimeFormat)
}

// ============================================================================
// CRUD Tests
// ============================================================================

func TestUpdatePreference(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	pref := &models.NotificationPreference{
		UserID:      uuid.New(),
		InApp:       true,
		DesktopPush: false,
		Sound:       "subtle",
	}

	err := svc.UpdatePreference(context.Background(), pref)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, pref.ID)
	assert.False(t, pref.UpdatedAt.IsZero())
}

func TestMuteResource(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	mute, err := svc.MuteResource(context.Background(), tenantID, userID, "chat", "channel-abc")
	require.NoError(t, err)
	assert.Equal(t, userID, mute.UserID)
	assert.Equal(t, "chat", mute.ModuleID)
	assert.Equal(t, "channel-abc", mute.ResourceID)
}

func TestMuteResourceAlreadyMuted(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	repo.mutes = []models.NotificationMute{
		{UserID: userID, ModuleID: "chat", ResourceID: "channel-abc"},
	}

	_, err := svc.MuteResource(context.Background(), tenantID, userID, "chat", "channel-abc")
	assert.ErrorIs(t, err, ErrMuteAlreadyExists)
}

func TestUnmuteResource(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	muteID := uuid.New()
	repo.mutes = []models.NotificationMute{
		{ID: muteID, UserID: userID, ModuleID: "chat", ResourceID: "channel-abc"},
	}

	err := svc.UnmuteResource(context.Background(), tenantID, userID, muteID)
	require.NoError(t, err)
}

func TestUnmuteResourceNotFound(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	err := svc.UnmuteResource(context.Background(), uuid.New(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrMuteNotFound)
}

func TestUnmuteResourceWrongUserRejected(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	tenantID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()
	muteID := uuid.New()
	repo.mutes = []models.NotificationMute{
		{ID: muteID, UserID: ownerID, ModuleID: "chat", ResourceID: "channel-abc"},
	}

	err := svc.UnmuteResource(context.Background(), tenantID, otherUserID, muteID)
	assert.ErrorIs(t, err, ErrMuteNotFound)
	assert.Len(t, repo.mutes, 1, "mute of another user must not be deletable")
}

func TestListMutedResources(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	repo.mutes = []models.NotificationMute{
		{UserID: userID, ModuleID: "chat", ResourceID: "channel-1"},
		{UserID: userID, ModuleID: "chat", ResourceID: "channel-2"},
	}

	mutes, total, err := svc.ListMutedResources(context.Background(), tenantID, userID, nil, 1, 10)
	require.NoError(t, err)
	assert.Len(t, mutes, 2)
	assert.Equal(t, 2, total)
}

func TestUpdateQuietHours(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	qh, err := svc.UpdateQuietHours(context.Background(), tenantID, userID, "18:00", "08:00", "Europe/Berlin", []int{1, 2, 3, 4, 5}, true)
	require.NoError(t, err)
	assert.Equal(t, "18:00", qh.StartTime)
	assert.Equal(t, "08:00", qh.EndTime)
	assert.Equal(t, "Europe/Berlin", qh.Timezone)
	assert.True(t, qh.Enabled)
}

func TestUpdateQuietHoursInvalidTimezone(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	_, err := svc.UpdateQuietHours(context.Background(), uuid.New(), uuid.New(), "18:00", "08:00", "Bad/Zone", []int{1}, true)
	assert.ErrorIs(t, err, ErrInvalidTimezone)
}

func TestUpdateQuietHoursInvalidTimeFormat(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	_, err := svc.UpdateQuietHours(context.Background(), uuid.New(), uuid.New(), "bad", "08:00", "UTC", []int{1}, true)
	assert.ErrorIs(t, err, ErrInvalidTimeFormat)
}

func TestUpdateQuietHoursInvalidDayOfWeek(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	_, err := svc.UpdateQuietHours(context.Background(), uuid.New(), uuid.New(), "18:00", "08:00", "UTC", []int{0}, true)
	assert.ErrorIs(t, err, ErrInvalidDayOfWeek)

	_, err = svc.UpdateQuietHours(context.Background(), uuid.New(), uuid.New(), "18:00", "08:00", "UTC", []int{8}, true)
	assert.ErrorIs(t, err, ErrInvalidDayOfWeek)
}

func TestToggleManualDND(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()

	// Enable DND
	qh, err := svc.ToggleManualDND(context.Background(), tenantID, userID, true, nil)
	require.NoError(t, err)
	assert.True(t, qh.ManualDND)
	assert.Nil(t, qh.ManualDNDUntil)

	// Disable DND
	repo.quietHours = qh
	qh, err = svc.ToggleManualDND(context.Background(), tenantID, userID, false, nil)
	require.NoError(t, err)
	assert.False(t, qh.ManualDND)
}

func TestToggleManualDNDWithExpiry(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	until := time.Now().Add(2 * time.Hour)

	qh, err := svc.ToggleManualDND(context.Background(), tenantID, userID, true, &until)
	require.NoError(t, err)
	assert.True(t, qh.ManualDND)
	assert.NotNil(t, qh.ManualDNDUntil)
}

func TestGetQuietHoursNotFound(t *testing.T) {
	repo := newMockPrefRepo()
	svc := NewService(repo)

	_, err := svc.GetQuietHours(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrQuietHoursNotFound)
}

// ============================================================================
// Mock Repository
// ============================================================================

type mockPrefRepo struct {
	prefs      []models.NotificationPreference
	mutes      []models.NotificationMute
	quietHours *models.QuietHours
}

func newMockPrefRepo() *mockPrefRepo {
	return &mockPrefRepo{}
}

func (m *mockPrefRepo) GetEventTypePreference(_ context.Context, _ uuid.UUID, userID uuid.UUID, eventTypeKey string) (*models.NotificationPreference, error) {
	for _, p := range m.prefs {
		if p.UserID == userID && p.EventTypeKey != nil && *p.EventTypeKey == eventTypeKey {
			return &p, nil
		}
	}
	return nil, ErrPreferenceNotFound
}

func (m *mockPrefRepo) GetModuleDefault(_ context.Context, _ uuid.UUID, userID uuid.UUID, moduleID string) (*models.NotificationPreference, error) {
	for _, p := range m.prefs {
		if p.UserID == userID && p.EventTypeKey == nil && p.ModuleID != nil && *p.ModuleID == moduleID {
			return &p, nil
		}
	}
	return nil, ErrPreferenceNotFound
}

func (m *mockPrefRepo) ListPreferences(_ context.Context, _ uuid.UUID, userID uuid.UUID, moduleID *string) ([]*models.NotificationPreference, error) {
	var result []*models.NotificationPreference
	for i := range m.prefs {
		if m.prefs[i].UserID == userID {
			if moduleID != nil && m.prefs[i].ModuleID != nil && *m.prefs[i].ModuleID != *moduleID {
				continue
			}
			result = append(result, &m.prefs[i])
		}
	}
	return result, nil
}

func (m *mockPrefRepo) UpsertPreference(_ context.Context, pref *models.NotificationPreference) error {
	m.prefs = append(m.prefs, *pref)
	return nil
}

func (m *mockPrefRepo) IsResourceMuted(_ context.Context, _ uuid.UUID, userID uuid.UUID, moduleID, resourceID string) (bool, error) {
	for _, mute := range m.mutes {
		if mute.UserID == userID && mute.ModuleID == moduleID && mute.ResourceID == resourceID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockPrefRepo) CreateMute(_ context.Context, mute *models.NotificationMute) error {
	m.mutes = append(m.mutes, *mute)
	return nil
}

func (m *mockPrefRepo) DeleteMute(_ context.Context, _ uuid.UUID, userID uuid.UUID, muteID uuid.UUID) error {
	for i, mute := range m.mutes {
		if mute.UserID == userID && mute.ID == muteID {
			m.mutes = append(m.mutes[:i], m.mutes[i+1:]...)
			return nil
		}
	}
	return ErrMuteNotFound
}

func (m *mockPrefRepo) ListMutes(_ context.Context, _ uuid.UUID, userID uuid.UUID, moduleID *string, offset, limit int) ([]*models.NotificationMute, int, error) {
	var filtered []*models.NotificationMute
	for i := range m.mutes {
		if m.mutes[i].UserID == userID {
			if moduleID != nil && m.mutes[i].ModuleID != *moduleID {
				continue
			}
			filtered = append(filtered, &m.mutes[i])
		}
	}
	total := len(filtered)
	if offset >= len(filtered) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

func (m *mockPrefRepo) GetQuietHours(_ context.Context, _ uuid.UUID, userID uuid.UUID) (*models.QuietHours, error) {
	if m.quietHours != nil && m.quietHours.UserID == userID {
		return m.quietHours, nil
	}
	return nil, ErrQuietHoursNotFound
}

func (m *mockPrefRepo) UpsertQuietHours(_ context.Context, qh *models.QuietHours) error {
	m.quietHours = qh
	return nil
}
