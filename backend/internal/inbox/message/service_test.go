package message

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
	"github.com/kmuhub/kmuhub/internal/models"
)

// mockRepository implements Repository for testing.
type mockRepository struct {
	mu       sync.RWMutex
	messages map[uuid.UUID]*models.InboxMessage
	// sourceIndex maps "userID|channel|sourceID" -> message for dedup
	sourceIndex map[string]*models.InboxMessage

	// Error injection
	createErr          error
	getByIDErr         error
	listErr            error
	updateErr          error
	markReadErr        error
	markUnreadErr      error
	toggleStarErr      error
	setStatusErr       error
	addTagErr          error
	removeTagErr       error
	archiveErr         error
	unarchiveErr       error
	snoozeErr          error
	unsnoozeExpiredErr error
	assignErr          error
	unreadCountsErr    error
	bulkMarkReadErr    error
	bulkArchiveErr     error
	getBySourceIDErr   error
	notifyErr          error

	// Control assign behavior
	assignReturns bool
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		messages:      make(map[uuid.UUID]*models.InboxMessage),
		sourceIndex:   make(map[string]*models.InboxMessage),
		assignReturns: true,
	}
}

func sourceKey(userID uuid.UUID, channel, sourceID string) string {
	return userID.String() + "|" + channel + "|" + sourceID
}

func (m *mockRepository) Create(_ context.Context, msg *models.InboxMessage) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[msg.ID] = msg
	m.sourceIndex[sourceKey(msg.UserID, msg.Channel, msg.SourceID)] = msg
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.InboxMessage, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	msg, ok := m.messages[id]
	if !ok {
		return nil, ErrMessageNotFound
	}
	// Enforce tenant isolation when a real tenantID is supplied.
	if tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID {
		return nil, ErrMessageNotFound
	}
	return msg, nil
}

func (m *mockRepository) List(_ context.Context, _ ListFilter) ([]*models.InboxMessage, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.InboxMessage
	for _, msg := range m.messages {
		result = append(result, msg)
	}
	return result, len(result), nil
}

func (m *mockRepository) Update(_ context.Context, msg *models.InboxMessage) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[msg.ID] = msg
	return nil
}

func (m *mockRepository) MarkRead(_ context.Context, tenantID, id uuid.UUID) error {
	if m.markReadErr != nil {
		return m.markReadErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok || (tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID) {
		return ErrMessageNotFound
	}
	msg.IsRead = true
	return nil
}

func (m *mockRepository) MarkUnread(_ context.Context, tenantID, id uuid.UUID) error {
	if m.markUnreadErr != nil {
		return m.markUnreadErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok || (tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID) {
		return ErrMessageNotFound
	}
	msg.IsRead = false
	return nil
}

func (m *mockRepository) ToggleStar(_ context.Context, tenantID, id uuid.UUID) error {
	if m.toggleStarErr != nil {
		return m.toggleStarErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok || (tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID) {
		return ErrMessageNotFound
	}
	msg.IsStarred = !msg.IsStarred
	return nil
}

func (m *mockRepository) SetStatus(_ context.Context, tenantID, id uuid.UUID, status string) error {
	if m.setStatusErr != nil {
		return m.setStatusErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok || (tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID) {
		return ErrMessageNotFound
	}
	msg.Status = status
	return nil
}

func (m *mockRepository) AddTag(_ context.Context, tenantID, id uuid.UUID, tag string) error {
	if m.addTagErr != nil {
		return m.addTagErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok || (tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID) {
		return ErrMessageNotFound
	}
	if slices.Contains(msg.Tags, tag) {
		return nil
	}
	msg.Tags = append(msg.Tags, tag)
	return nil
}

func (m *mockRepository) RemoveTag(_ context.Context, tenantID, id uuid.UUID, tag string) error {
	if m.removeTagErr != nil {
		return m.removeTagErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok || (tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID) {
		return ErrMessageNotFound
	}
	filtered := make([]string, 0, len(msg.Tags))
	for _, t := range msg.Tags {
		if t != tag {
			filtered = append(filtered, t)
		}
	}
	msg.Tags = filtered
	return nil
}

func (m *mockRepository) Archive(_ context.Context, tenantID, id uuid.UUID) error {
	if m.archiveErr != nil {
		return m.archiveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok || (tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID) {
		return ErrMessageNotFound
	}
	msg.IsArchived = true
	return nil
}

func (m *mockRepository) Unarchive(_ context.Context, tenantID, id uuid.UUID) error {
	if m.unarchiveErr != nil {
		return m.unarchiveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok || (tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID) {
		return ErrMessageNotFound
	}
	msg.IsArchived = false
	return nil
}

func (m *mockRepository) Snooze(_ context.Context, tenantID, id uuid.UUID, until time.Time) error {
	if m.snoozeErr != nil {
		return m.snoozeErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok || (tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID) {
		return ErrMessageNotFound
	}
	msg.SnoozedUntil = &until
	return nil
}

func (m *mockRepository) UnsnoozeExpired(_ context.Context) (int, error) {
	if m.unsnoozeExpiredErr != nil {
		return 0, m.unsnoozeExpiredErr
	}
	return 0, nil
}

func (m *mockRepository) AssignMessage(_ context.Context, tenantID, id uuid.UUID, assigneeID uuid.UUID) (bool, error) {
	if m.assignErr != nil {
		return false, m.assignErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[id]
	if !ok || (tenantID != uuid.Nil && msg.TenantID != uuid.Nil && msg.TenantID != tenantID) {
		return false, ErrMessageNotFound
	}
	if !m.assignReturns {
		return false, nil
	}
	msg.AssignedTo = &assigneeID
	return true, nil
}

func (m *mockRepository) GetUnreadCounts(_ context.Context, _ uuid.UUID) ([]models.UnreadCount, error) {
	if m.unreadCountsErr != nil {
		return nil, m.unreadCountsErr
	}
	return []models.UnreadCount{
		{Channel: "email", Count: 5},
		{Channel: "chat", Count: 3},
	}, nil
}

func (m *mockRepository) BulkMarkRead(_ context.Context, tenantID uuid.UUID, ids []uuid.UUID) error {
	if m.bulkMarkReadErr != nil {
		return m.bulkMarkReadErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		if msg, ok := m.messages[id]; ok && (tenantID == uuid.Nil || msg.TenantID == uuid.Nil || msg.TenantID == tenantID) {
			msg.IsRead = true
		}
	}
	return nil
}

func (m *mockRepository) BulkArchive(_ context.Context, tenantID uuid.UUID, ids []uuid.UUID) error {
	if m.bulkArchiveErr != nil {
		return m.bulkArchiveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		if msg, ok := m.messages[id]; ok && (tenantID == uuid.Nil || msg.TenantID == uuid.Nil || msg.TenantID == tenantID) {
			msg.IsArchived = true
		}
	}
	return nil
}

func (m *mockRepository) GetBySourceID(_ context.Context, userID uuid.UUID, channel, srcID string) (*models.InboxMessage, error) {
	if m.getBySourceIDErr != nil {
		return nil, m.getBySourceIDErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	msg, ok := m.sourceIndex[sourceKey(userID, channel, srcID)]
	if !ok {
		return nil, nil
	}
	return msg, nil
}

func (m *mockRepository) NotifyDelivery(_ context.Context, _ string) error {
	if m.notifyErr != nil {
		return m.notifyErr
	}
	return nil
}

// helper to seed a message into the mock repo
func seedMessage(repo *mockRepository, msg *models.InboxMessage) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.messages[msg.ID] = msg
	repo.sourceIndex[sourceKey(msg.UserID, msg.Channel, msg.SourceID)] = msg
}

func newTestMessage() *models.InboxMessage {
	return &models.InboxMessage{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Channel:    "email",
		SourceID:   "src-123",
		SenderName: "Alice",
		Subject:    "Hello",
		Preview:    "Hello world",
		Tags:       []string{},
		ReceivedAt: time.Now(),
	}
}

func TestCreate_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	err := svc.Create(context.Background(), msg)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, msg.ID)

	stored, err := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	require.NoError(t, err)
	assert.Equal(t, msg.Subject, stored.Subject)
}

func TestCreate_DuplicateDetected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	existing := newTestMessage()
	existing.ReceivedAt = time.Now().Add(-1 * time.Hour)
	seedMessage(repo, existing)

	dup := &models.InboxMessage{
		ID:         uuid.New(),
		UserID:     existing.UserID,
		Channel:    existing.Channel,
		SourceID:   existing.SourceID,
		Subject:    "Updated Subject",
		Preview:    "Updated preview",
		Tags:       []string{},
		ReceivedAt: time.Now(), // newer
	}

	err := svc.Create(context.Background(), dup)

	require.ErrorIs(t, err, ErrDuplicateMessage)
	// Existing message should have updated subject since dup is newer
	stored, _ := repo.GetByID(context.Background(), existing.ID, uuid.Nil)
	assert.Equal(t, "Updated Subject", stored.Subject)
}

func TestGetByID_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	seedMessage(repo, msg)

	result, err := svc.GetByID(context.Background(), msg.ID, uuid.Nil)

	require.NoError(t, err)
	assert.Equal(t, msg.ID, result.ID)
	assert.Equal(t, msg.Subject, result.Subject)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.Nil)

	require.ErrorIs(t, err, ErrMessageNotFound)
}

func TestList_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	seedMessage(repo, newTestMessage())
	seedMessage(repo, newTestMessage())

	msgs, count, err := svc.List(context.Background(), ListFilter{
		UserID:   uuid.New(),
		PageSize: 50,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Len(t, msgs, 2)
}

func TestMarkRead_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	msg.IsRead = false
	seedMessage(repo, msg)

	err := svc.MarkRead(context.Background(), msg.ID, uuid.Nil)

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.True(t, stored.IsRead)
}

func TestMarkUnread_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	msg.IsRead = true
	seedMessage(repo, msg)

	err := svc.MarkUnread(context.Background(), msg.ID, uuid.Nil)

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.False(t, stored.IsRead)
}

func TestToggleStar_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	msg.IsStarred = false
	seedMessage(repo, msg)

	err := svc.ToggleStar(context.Background(), msg.ID, uuid.Nil)

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.True(t, stored.IsStarred)
}

func TestSetStatus_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	msg.Status = "open"
	seedMessage(repo, msg)

	err := svc.SetStatus(context.Background(), msg.ID, uuid.Nil, "resolved")

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.Equal(t, "resolved", stored.Status)
}

func TestSetStatus_InvalidValue(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	msg.Status = "open"
	seedMessage(repo, msg)

	err := svc.SetStatus(context.Background(), msg.ID, uuid.Nil, "archived")

	require.ErrorIs(t, err, ErrInvalidStatus)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.Equal(t, "open", stored.Status, "invalid status must not be persisted")
}

func TestSetStatus_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	err := svc.SetStatus(context.Background(), uuid.New(), uuid.Nil, "closed")

	require.ErrorIs(t, err, ErrMessageNotFound)
}

func TestAddTag_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	seedMessage(repo, msg)

	err := svc.AddTag(context.Background(), msg.ID, uuid.Nil, "  Demo  ")

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.Equal(t, []string{"Demo"}, stored.Tags, "tag must be trimmed")
}

func TestAddTag_Idempotent(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	msg.Tags = []string{"Demo"}
	seedMessage(repo, msg)

	err := svc.AddTag(context.Background(), msg.ID, uuid.Nil, "Demo")

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.Equal(t, []string{"Demo"}, stored.Tags, "adding an existing tag must not duplicate it")
}

func TestAddTag_EmptyRejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	seedMessage(repo, msg)

	err := svc.AddTag(context.Background(), msg.ID, uuid.Nil, "   ")

	require.ErrorIs(t, err, ErrInvalidTag)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.Empty(t, stored.Tags, "an empty tag must not be persisted")
}

func TestRemoveTag_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	msg.Tags = []string{"Demo", "Lead"}
	seedMessage(repo, msg)

	err := svc.RemoveTag(context.Background(), msg.ID, uuid.Nil, "Demo")

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.Equal(t, []string{"Lead"}, stored.Tags)
}

func TestRemoveTag_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	err := svc.RemoveTag(context.Background(), uuid.New(), uuid.Nil, "Demo")

	require.ErrorIs(t, err, ErrMessageNotFound)
}

// mockForwardAdapter is a minimal ChannelAdapter stub for exercising
// Service.Reply and Service.Forward.
type mockForwardAdapter struct {
	channel   string
	replyFn   func(ctx context.Context, sourceID string, userID uuid.UUID, body string) error
	forwardFn func(ctx context.Context, sourceID string, userID uuid.UUID, to string, note string) error
}

func (a *mockForwardAdapter) Channel() string { return a.channel }
func (a *mockForwardAdapter) FetchNewMessages(_ context.Context, _ uuid.UUID, _ time.Time) ([]models.InboxMessage, error) {
	return nil, nil
}
func (a *mockForwardAdapter) HandleReply(ctx context.Context, sourceID string, userID uuid.UUID, body string) error {
	if a.replyFn == nil {
		return nil
	}
	return a.replyFn(ctx, sourceID, userID, body)
}
func (a *mockForwardAdapter) HandleForward(ctx context.Context, sourceID string, userID uuid.UUID, to string, note string) error {
	return a.forwardFn(ctx, sourceID, userID, to, note)
}
func (a *mockForwardAdapter) MarkReadOnSource(_ context.Context, _ string, _ uuid.UUID) error {
	return nil
}

func TestForward_Success(t *testing.T) {
	repo := newMockRepository()
	registry := adapter.NewAdapterRegistry()

	var gotSourceID, gotTo, gotNote string
	registry.Register(&mockForwardAdapter{
		channel: "email",
		forwardFn: func(_ context.Context, sourceID string, _ uuid.UUID, to string, note string) error {
			gotSourceID, gotTo, gotNote = sourceID, to, note
			return nil
		},
	})
	svc := NewService(repo, registry)

	msg := newTestMessage()
	seedMessage(repo, msg)

	err := svc.Forward(context.Background(), msg.ID, uuid.Nil, uuid.New(), "colleague@example.com", "FYI")

	require.NoError(t, err)
	// The adapter must receive the message's origin-system id (SourceID), not
	// the inbox message's own id -- the email/chat services have no record
	// under the inbox message's id.
	assert.Equal(t, msg.SourceID, gotSourceID)
	assert.Equal(t, "colleague@example.com", gotTo)
	assert.Equal(t, "FYI", gotNote)
}

func TestReply_Success(t *testing.T) {
	repo := newMockRepository()
	registry := adapter.NewAdapterRegistry()

	var gotSourceID, gotBody string
	registry.Register(&mockForwardAdapter{
		channel: "email",
		replyFn: func(_ context.Context, sourceID string, _ uuid.UUID, body string) error {
			gotSourceID, gotBody = sourceID, body
			return nil
		},
	})
	svc := NewService(repo, registry)

	msg := newTestMessage()
	seedMessage(repo, msg)

	err := svc.Reply(context.Background(), msg.ID, uuid.Nil, uuid.New(), "Danke!")

	require.NoError(t, err)
	assert.Equal(t, msg.SourceID, gotSourceID)
	assert.Equal(t, "Danke!", gotBody)
}

func TestReply_NoAdapter(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	seedMessage(repo, msg)

	err := svc.Reply(context.Background(), msg.ID, uuid.Nil, uuid.New(), "Danke!")

	require.ErrorIs(t, err, ErrAdapterNotFound)
}

func TestForward_NoAdapter(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	seedMessage(repo, msg)

	err := svc.Forward(context.Background(), msg.ID, uuid.Nil, uuid.New(), "colleague@example.com", "")

	require.ErrorIs(t, err, ErrAdapterNotFound)
}

func TestForward_NotSupportedByChannel(t *testing.T) {
	repo := newMockRepository()
	registry := adapter.NewAdapterRegistry()
	registry.Register(&mockForwardAdapter{
		channel: "email",
		forwardFn: func(context.Context, string, uuid.UUID, string, string) error {
			return adapter.ErrForwardNotSupported
		},
	})
	svc := NewService(repo, registry)

	msg := newTestMessage()
	seedMessage(repo, msg)

	err := svc.Forward(context.Background(), msg.ID, uuid.Nil, uuid.New(), "colleague@example.com", "")

	require.ErrorIs(t, err, ErrForwardNotSupported)
}

func TestArchive_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	seedMessage(repo, msg)

	err := svc.Archive(context.Background(), msg.ID, uuid.Nil)

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.True(t, stored.IsArchived)
}

func TestSnooze_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	seedMessage(repo, msg)

	until := time.Now().Add(1 * time.Hour)
	err := svc.Snooze(context.Background(), msg.ID, uuid.Nil, until)

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	require.NotNil(t, stored.SnoozedUntil)
	assert.WithinDuration(t, until, *stored.SnoozedUntil, time.Second)
}

func TestSnooze_PastTime(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	seedMessage(repo, msg)

	pastTime := time.Now().Add(-1 * time.Hour)
	err := svc.Snooze(context.Background(), msg.ID, uuid.Nil, pastTime)

	require.ErrorIs(t, err, ErrInvalidSnoozeTime)
}

func TestGetUnreadCounts_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	userID := uuid.New()
	counts, err := svc.GetUnreadCounts(context.Background(), userID)

	require.NoError(t, err)
	assert.Len(t, counts, 2)
	assert.Equal(t, "email", counts[0].Channel)
	assert.Equal(t, 5, counts[0].Count)
}

func TestAssignMessage_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	seedMessage(repo, msg)

	assigneeID := uuid.New()
	err := svc.AssignMessage(context.Background(), msg.ID, uuid.Nil, assigneeID)

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), msg.ID, uuid.Nil)
	require.NotNil(t, stored.AssignedTo)
	assert.Equal(t, assigneeID, *stored.AssignedTo)
}

func TestAssignMessage_AlreadyAssigned(t *testing.T) {
	repo := newMockRepository()
	repo.assignReturns = false
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg := newTestMessage()
	existingAssignee := uuid.New()
	msg.AssignedTo = &existingAssignee
	seedMessage(repo, msg)

	err := svc.AssignMessage(context.Background(), msg.ID, uuid.Nil, uuid.New())

	require.ErrorIs(t, err, ErrAlreadyAssigned)
}

func TestBulkMarkRead_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg1 := newTestMessage()
	msg2 := newTestMessage()
	seedMessage(repo, msg1)
	seedMessage(repo, msg2)

	err := svc.BulkMarkRead(context.Background(), []uuid.UUID{msg1.ID, msg2.ID}, uuid.Nil)

	require.NoError(t, err)
	s1, _ := repo.GetByID(context.Background(), msg1.ID, uuid.Nil)
	s2, _ := repo.GetByID(context.Background(), msg2.ID, uuid.Nil)
	assert.True(t, s1.IsRead)
	assert.True(t, s2.IsRead)
}

func TestBulkArchive_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	msg1 := newTestMessage()
	msg2 := newTestMessage()
	seedMessage(repo, msg1)
	seedMessage(repo, msg2)

	err := svc.BulkArchive(context.Background(), []uuid.UUID{msg1.ID, msg2.ID}, uuid.Nil)

	require.NoError(t, err)
	s1, _ := repo.GetByID(context.Background(), msg1.ID, uuid.Nil)
	s2, _ := repo.GetByID(context.Background(), msg2.ID, uuid.Nil)
	assert.True(t, s1.IsArchived)
	assert.True(t, s2.IsArchived)
}

// ============================================================================
// Cross-tenant isolation tests (inbox_messages)
// ============================================================================

// TestGetByID_CrossTenant verifies that a message owned by tenant A cannot be
// fetched using tenant B's ID.
func TestGetByID_CrossTenant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	tenantA := uuid.New()
	tenantB := uuid.New()

	msg := newTestMessage()
	msg.TenantID = tenantA
	seedMessage(repo, msg)

	result, err := svc.GetByID(context.Background(), msg.ID, tenantB)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrMessageNotFound)
}

// TestMarkRead_CrossTenant verifies that marking a message as read fails when
// the request carries a different tenant's ID.
func TestMarkRead_CrossTenant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, adapter.NewAdapterRegistry())

	tenantA := uuid.New()
	tenantB := uuid.New()

	msg := newTestMessage()
	msg.TenantID = tenantA
	msg.IsRead = false
	seedMessage(repo, msg)

	err := svc.MarkRead(context.Background(), msg.ID, tenantB)

	assert.ErrorIs(t, err, ErrMessageNotFound)
	// Read flag must remain unchanged.
	assert.False(t, repo.messages[msg.ID].IsRead)
}
