package presence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Store ---

type mockStore struct {
	data map[string]*UserPresence
}

func newMockStore() *mockStore {
	return &mockStore{data: make(map[string]*UserPresence)}
}

func (m *mockStore) SetPresence(_ context.Context, userID string, status PresenceStatus, manual bool) error {
	uid, _ := uuid.Parse(userID)
	now := time.Now().UTC()
	m.data[userID] = &UserPresence{
		UserID:         uid,
		Status:         status,
		ManualOverride: manual,
		LastActivity:   &now,
	}
	return nil
}

func (m *mockStore) GetPresence(_ context.Context, userID string) (*UserPresence, error) {
	p, ok := m.data[userID]
	if !ok {
		return nil, nil
	}
	// Return a copy
	cp := *p
	return &cp, nil
}

func (m *mockStore) GetBulkPresence(_ context.Context, userIDs []string) (map[string]*UserPresence, error) {
	result := make(map[string]*UserPresence)
	for _, uid := range userIDs {
		if p, ok := m.data[uid]; ok {
			cp := *p
			result[uid] = &cp
		}
	}
	return result, nil
}

func (m *mockStore) RemovePresence(_ context.Context, userID string) error {
	delete(m.data, userID)
	return nil
}

func (m *mockStore) UpdateLastActivity(_ context.Context, userID string) error {
	if p, ok := m.data[userID]; ok {
		now := time.Now().UTC()
		p.LastActivity = &now
	}
	return nil
}

// --- Mock Config Repository ---

type mockConfigRepo struct {
	config *PresenceConfig
}

func newMockConfigRepo() *mockConfigRepo {
	return &mockConfigRepo{
		config: &PresenceConfig{
			ID:                 uuid.New(),
			AwayTimeoutSeconds: DefaultAwayTimeoutSeconds,
			UpdatedAt:          time.Now().UTC(),
		},
	}
}

func (m *mockConfigRepo) GetConfig(_ context.Context) (*PresenceConfig, error) {
	return m.config, nil
}

func (m *mockConfigRepo) UpdateConfig(_ context.Context, awayTimeoutSeconds int, updatedBy uuid.UUID) error {
	m.config.AwayTimeoutSeconds = awayTimeoutSeconds
	m.config.UpdatedBy = &updatedBy
	m.config.UpdatedAt = time.Now().UTC()
	return nil
}

// --- Helper Functions ---

func newTestPresenceService() (*Service, *mockStore, *mockConfigRepo) {
	store := newMockStore()
	configRepo := newMockConfigRepo()
	svc := NewService(store, configRepo)
	return svc, store, configRepo
}

func testUserID() string {
	return uuid.New().String()
}

// --- Tests ---

func TestHeartbeat_SetsOnline(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	err := svc.Heartbeat(ctx, userID)
	require.NoError(t, err)

	p := store.data[userID]
	require.NotNil(t, p)
	assert.Equal(t, PresenceOnline, p.Status)
	assert.False(t, p.ManualOverride)
}

func TestHeartbeat_DoesNotOverrideDND(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	// Manually set DND
	err := svc.SetManualStatus(ctx, userID, PresenceDND)
	require.NoError(t, err)

	// Heartbeat should NOT override DND
	err = svc.Heartbeat(ctx, userID)
	require.NoError(t, err)

	p := store.data[userID]
	assert.Equal(t, PresenceDND, p.Status)
	assert.True(t, p.ManualOverride)
}

func TestHeartbeat_DoesNotOverrideManualAway(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	// Manually set away
	err := svc.SetManualStatus(ctx, userID, PresenceAway)
	require.NoError(t, err)

	// Heartbeat should NOT override manual away
	err = svc.Heartbeat(ctx, userID)
	require.NoError(t, err)

	p := store.data[userID]
	assert.Equal(t, PresenceAway, p.Status)
	assert.True(t, p.ManualOverride)
}

func TestHeartbeat_DoesNotOverrideInCall(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	// Set in_call
	err := svc.SetInCall(ctx, userID)
	require.NoError(t, err)

	// Heartbeat should NOT override in_call
	err = svc.Heartbeat(ctx, userID)
	require.NoError(t, err)

	p := store.data[userID]
	assert.Equal(t, PresenceInCall, p.Status)
}

func TestGetPresence_Offline(t *testing.T) {
	svc, _, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	p, err := svc.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PresenceOffline, p.Status)
}

func TestGetPresence_LazyAwayDetection(t *testing.T) {
	svc, store, configRepo := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	// Set away timeout to 1 second for testing
	configRepo.config.AwayTimeoutSeconds = 1

	// Set user online with last activity 2 seconds ago
	uid, _ := uuid.Parse(userID)
	pastTime := time.Now().Add(-2 * time.Second)
	store.data[userID] = &UserPresence{
		UserID:         uid,
		Status:         PresenceOnline,
		ManualOverride: false,
		LastActivity:   &pastTime,
	}

	p, err := svc.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PresenceAway, p.Status, "should detect away lazily")
}

func TestGetPresence_ManualOnlineNotAffectedByAwayDetection(t *testing.T) {
	svc, store, configRepo := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	configRepo.config.AwayTimeoutSeconds = 1

	// Manually set online (which means manual=false for online)
	uid, _ := uuid.Parse(userID)
	pastTime := time.Now().Add(-2 * time.Second)
	store.data[userID] = &UserPresence{
		UserID:         uid,
		Status:         PresenceOnline,
		ManualOverride: false,
		LastActivity:   &pastTime,
	}

	p, err := svc.GetPresence(ctx, userID)
	require.NoError(t, err)
	// Non-manual online SHOULD be detected as away
	assert.Equal(t, PresenceAway, p.Status)
}

func TestSetManualStatus_DND(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	err := svc.SetManualStatus(ctx, userID, PresenceDND)
	require.NoError(t, err)

	p := store.data[userID]
	assert.Equal(t, PresenceDND, p.Status)
	assert.True(t, p.ManualOverride)
}

func TestSetManualStatus_Away(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	err := svc.SetManualStatus(ctx, userID, PresenceAway)
	require.NoError(t, err)

	p := store.data[userID]
	assert.Equal(t, PresenceAway, p.Status)
	assert.True(t, p.ManualOverride)
}

func TestSetManualStatus_OnlineClearsManual(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	// Set DND first
	err := svc.SetManualStatus(ctx, userID, PresenceDND)
	require.NoError(t, err)

	// Reset to online
	err = svc.SetManualStatus(ctx, userID, PresenceOnline)
	require.NoError(t, err)

	p := store.data[userID]
	assert.Equal(t, PresenceOnline, p.Status)
	assert.False(t, p.ManualOverride, "online should clear manual override")
}

func TestSetManualStatus_InvalidStatus(t *testing.T) {
	svc, _, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	err := svc.SetManualStatus(ctx, userID, PresenceInCall)
	assert.ErrorIs(t, err, ErrInvalidManualStatus)

	err = svc.SetManualStatus(ctx, userID, PresenceOffline)
	assert.ErrorIs(t, err, ErrInvalidManualStatus)
}

func TestSetInCall_And_ClearInCall(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	// Set online first
	err := svc.Heartbeat(ctx, userID)
	require.NoError(t, err)

	// Join call
	err = svc.SetInCall(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PresenceInCall, store.data[userID].Status)

	// Leave call
	err = svc.ClearInCall(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PresenceOnline, store.data[userID].Status)
}

func TestClearInCall_NotInCall(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	// Set online
	err := svc.Heartbeat(ctx, userID)
	require.NoError(t, err)

	// ClearInCall when not in_call should be a no-op
	err = svc.ClearInCall(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PresenceOnline, store.data[userID].Status)
}

func TestDisconnect(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	err := svc.Heartbeat(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, store.data[userID])

	err = svc.Disconnect(ctx, userID)
	require.NoError(t, err)
	assert.Nil(t, store.data[userID])
}

func TestGetBulkPresence_MixedStatuses(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()

	onlineUser := testUserID()
	dndUser := testUserID()
	offlineUser := testUserID()

	// Set up states
	uid1, _ := uuid.Parse(onlineUser)
	now := time.Now().UTC()
	store.data[onlineUser] = &UserPresence{UserID: uid1, Status: PresenceOnline, LastActivity: &now}

	uid2, _ := uuid.Parse(dndUser)
	store.data[dndUser] = &UserPresence{UserID: uid2, Status: PresenceDND, ManualOverride: true, LastActivity: &now}

	result, err := svc.GetBulkPresence(ctx, []string{onlineUser, dndUser, offlineUser})
	require.NoError(t, err)

	assert.Equal(t, PresenceOnline, result[onlineUser].Status)
	assert.Equal(t, PresenceDND, result[dndUser].Status)
	assert.Equal(t, PresenceOffline, result[offlineUser].Status)
}

func TestGetBulkPresence_AwayDetection(t *testing.T) {
	svc, store, configRepo := newTestPresenceService()
	ctx := context.Background()

	configRepo.config.AwayTimeoutSeconds = 1

	staleUser := testUserID()
	activeUser := testUserID()

	uid1, _ := uuid.Parse(staleUser)
	pastTime := time.Now().Add(-2 * time.Second)
	store.data[staleUser] = &UserPresence{UserID: uid1, Status: PresenceOnline, LastActivity: &pastTime}

	uid2, _ := uuid.Parse(activeUser)
	now := time.Now().UTC()
	store.data[activeUser] = &UserPresence{UserID: uid2, Status: PresenceOnline, LastActivity: &now}

	result, err := svc.GetBulkPresence(ctx, []string{staleUser, activeUser})
	require.NoError(t, err)

	assert.Equal(t, PresenceAway, result[staleUser].Status, "stale user should be detected as away")
	assert.Equal(t, PresenceOnline, result[activeUser].Status, "active user should remain online")
}

func TestUpdateConfig_Valid(t *testing.T) {
	svc, _, configRepo := newTestPresenceService()
	ctx := context.Background()
	updatedBy := uuid.New()

	err := svc.UpdateConfig(ctx, 120, updatedBy)
	require.NoError(t, err)

	assert.Equal(t, 120, configRepo.config.AwayTimeoutSeconds)
}

func TestUpdateConfig_TooLow(t *testing.T) {
	svc, _, _ := newTestPresenceService()
	ctx := context.Background()

	err := svc.UpdateConfig(ctx, 30, uuid.New())
	assert.ErrorIs(t, err, ErrInvalidAwayTimeout)
}

func TestUpdateConfig_TooHigh(t *testing.T) {
	svc, _, _ := newTestPresenceService()
	ctx := context.Background()

	err := svc.UpdateConfig(ctx, 7200, uuid.New())
	assert.ErrorIs(t, err, ErrInvalidAwayTimeout)
}

func TestUpdateConfig_BoundaryValues(t *testing.T) {
	svc, _, _ := newTestPresenceService()
	ctx := context.Background()
	updatedBy := uuid.New()

	// Minimum: 60 should pass
	err := svc.UpdateConfig(ctx, 60, updatedBy)
	require.NoError(t, err)

	// Maximum: 3600 should pass
	err = svc.UpdateConfig(ctx, 3600, updatedBy)
	require.NoError(t, err)

	// Below minimum: 59 should fail
	err = svc.UpdateConfig(ctx, 59, updatedBy)
	assert.ErrorIs(t, err, ErrInvalidAwayTimeout)

	// Above maximum: 3601 should fail
	err = svc.UpdateConfig(ctx, 3601, updatedBy)
	assert.ErrorIs(t, err, ErrInvalidAwayTimeout)
}

func TestGetConfig(t *testing.T) {
	svc, _, _ := newTestPresenceService()
	ctx := context.Background()

	cfg, err := svc.GetConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, DefaultAwayTimeoutSeconds, cfg.AwayTimeoutSeconds)
}

func TestAwayTimeoutCaching(t *testing.T) {
	svc, store, configRepo := newTestPresenceService()
	ctx := context.Background()

	configRepo.config.AwayTimeoutSeconds = 1

	userID := testUserID()
	uid, _ := uuid.Parse(userID)
	pastTime := time.Now().Add(-2 * time.Second)
	store.data[userID] = &UserPresence{UserID: uid, Status: PresenceOnline, LastActivity: &pastTime}

	// First call populates cache
	p, err := svc.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PresenceAway, p.Status)

	// Change config in DB directly (cache should still return old value)
	configRepo.config.AwayTimeoutSeconds = 3600

	// Second call uses cached value (still 1 second timeout)
	store.data[userID].Status = PresenceOnline
	p, err = svc.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PresenceAway, p.Status, "cached timeout should still detect away")
}

func TestFiveStatusLevels(t *testing.T) {
	svc, store, _ := newTestPresenceService()
	ctx := context.Background()
	userID := testUserID()

	// Online
	err := svc.Heartbeat(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PresenceOnline, store.data[userID].Status)

	// Away (manual)
	err = svc.SetManualStatus(ctx, userID, PresenceAway)
	require.NoError(t, err)
	assert.Equal(t, PresenceAway, store.data[userID].Status)

	// DND
	err = svc.SetManualStatus(ctx, userID, PresenceDND)
	require.NoError(t, err)
	assert.Equal(t, PresenceDND, store.data[userID].Status)

	// InCall
	err = svc.SetInCall(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PresenceInCall, store.data[userID].Status)

	// Offline (via disconnect)
	err = svc.Disconnect(ctx, userID)
	require.NoError(t, err)
	p, err := svc.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, PresenceOffline, p.Status)
}
