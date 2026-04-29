package message

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock Repository ---

type mockRepo struct {
	messages      map[uuid.UUID]*models.EmailMessage
	byMessageID   map[string]*models.EmailMessage
	byFolderUID   map[string]*models.EmailMessage // "folderID:uid"
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		messages:    make(map[uuid.UUID]*models.EmailMessage),
		byMessageID: make(map[string]*models.EmailMessage),
		byFolderUID: make(map[string]*models.EmailMessage),
	}
}

func (r *mockRepo) Create(_ context.Context, msg *models.EmailMessage) error {
	copy := *msg
	r.messages[msg.ID] = &copy
	if msg.MessageID != "" {
		r.byMessageID[msg.MessageID] = &copy
	}
	return nil
}

func (r *mockRepo) GetByID(_ context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailMessage, error) {
	m, ok := r.messages[id]
	if !ok {
		return nil, ErrMessageNotFound
	}
	// Enforce tenant isolation when a real tenantID is supplied.
	if tenantID != uuid.Nil && m.TenantID != uuid.Nil && m.TenantID != tenantID {
		return nil, ErrMessageNotFound
	}
	copy := *m
	return &copy, nil
}

func (r *mockRepo) GetByFolderUID(_ context.Context, folderID uuid.UUID, uid uint32) (*models.EmailMessage, error) {
	return nil, ErrMessageNotFound
}

func (r *mockRepo) ListByFolder(_ context.Context, _ uuid.UUID, _ ListOpts) ([]*models.EmailMessage, int, error) {
	return nil, 0, nil
}

func (r *mockRepo) ListByThread(_ context.Context, threadID uuid.UUID) ([]*models.EmailMessage, error) {
	var result []*models.EmailMessage
	for _, m := range r.messages {
		if m.ThreadID != nil && *m.ThreadID == threadID {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *mockRepo) Search(_ context.Context, _ uuid.UUID, _ string, _ ListOpts) ([]*models.EmailMessage, int, error) {
	return nil, 0, nil
}

func (r *mockRepo) UpdateFlags(_ context.Context, id uuid.UUID, isRead, isStarred *bool) error {
	m, ok := r.messages[id]
	if !ok {
		return ErrMessageNotFound
	}
	if isRead != nil {
		m.IsRead = *isRead
	}
	if isStarred != nil {
		m.IsStarred = *isStarred
	}
	return nil
}

func (r *mockRepo) UpdateThreadID(_ context.Context, id uuid.UUID, threadID uuid.UUID) error {
	m, ok := r.messages[id]
	if !ok {
		return ErrMessageNotFound
	}
	m.ThreadID = &threadID
	return nil
}

func (r *mockRepo) MoveToFolder(_ context.Context, id, folderID uuid.UUID) error {
	m, ok := r.messages[id]
	if !ok {
		return ErrMessageNotFound
	}
	m.FolderID = folderID
	return nil
}

func (r *mockRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.messages, id)
	return nil
}

func (r *mockRepo) DeleteByFolder(_ context.Context, folderID uuid.UUID) error {
	for id, m := range r.messages {
		if m.FolderID == folderID {
			delete(r.messages, id)
		}
	}
	return nil
}

func (r *mockRepo) GetHighestUID(_ context.Context, _ uuid.UUID) (uint32, error) {
	return 0, nil
}

func (r *mockRepo) GetByMessageIDHeader(_ context.Context, msgID string) (*models.EmailMessage, error) {
	m, ok := r.byMessageID[msgID]
	if !ok {
		return nil, ErrMessageNotFound
	}
	return m, nil
}

func (r *mockRepo) FindThreadByReferences(_ context.Context, accountID uuid.UUID, references []string, inReplyTo string) (uuid.UUID, error) {
	// Try InReplyTo first
	if inReplyTo != "" {
		if m, ok := r.byMessageID[inReplyTo]; ok && m.ThreadID != nil {
			return *m.ThreadID, nil
		}
	}
	// Try references (last first)
	for i := len(references) - 1; i >= 0; i-- {
		if m, ok := r.byMessageID[references[i]]; ok && m.ThreadID != nil {
			return *m.ThreadID, nil
		}
	}
	return uuid.Nil, ErrThreadNotFound
}

func (r *mockRepo) CountUnreadByFolder(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *mockRepo) FindBySubjectAndParticipants(_ context.Context, _ uuid.UUID, _ string, _ string, _, _ interface{}) (*models.EmailMessage, error) {
	return nil, ErrMessageNotFound
}

// --- Mock Folder Repository ---

type mockFolderRepo struct{}

func (r *mockFolderRepo) Create(_ context.Context, _ *models.EmailFolder) error              { return nil }
func (r *mockFolderRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.EmailFolder, error) {
	return nil, ErrFolderNotFound
}
func (r *mockFolderRepo) GetByIMAPName(_ context.Context, _ uuid.UUID, _ string) (*models.EmailFolder, error) {
	return nil, ErrFolderNotFound
}
func (r *mockFolderRepo) ListByAccount(_ context.Context, _ uuid.UUID) ([]*models.EmailFolder, error) {
	return nil, nil
}
func (r *mockFolderRepo) UpdateCounts(_ context.Context, _ uuid.UUID, _, _ int) error { return nil }
func (r *mockFolderRepo) UpdateUIDValidity(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}
func (r *mockFolderRepo) DeleteMessagesByFolder(_ context.Context, _ uuid.UUID) error { return nil }

// --- Tests ---

func newTestMessage(accountID uuid.UUID) *models.EmailMessage {
	return &models.EmailMessage{
		ID:        uuid.New(),
		AccountID: accountID,
		FolderID:  uuid.New(),
		UID:       1,
		MessageID: "<msg-001@example.com>",
		Subject:   "Test Subject",
		FromEmail: "alice@example.com",
		FromName:  "Alice",
		Date:      time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func TestAssignThread_NewConversation(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockFolderRepo{})
	accountID := uuid.New()

	msg := newTestMessage(accountID)
	err := svc.Create(context.Background(), msg)
	require.NoError(t, err)

	stored := repo.messages[msg.ID]
	require.NotNil(t, stored.ThreadID)
	assert.NotEqual(t, uuid.Nil, *stored.ThreadID)
}

func TestAssignThread_ReplyChain(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockFolderRepo{})
	accountID := uuid.New()

	// Create original message
	original := newTestMessage(accountID)
	original.MessageID = "<original@example.com>"
	err := svc.Create(context.Background(), original)
	require.NoError(t, err)

	originalThread := *repo.messages[original.ID].ThreadID

	// Create reply with InReplyTo
	reply := newTestMessage(accountID)
	reply.MessageID = "<reply@example.com>"
	reply.InReplyTo = "<original@example.com>"
	err = svc.Create(context.Background(), reply)
	require.NoError(t, err)

	replyThread := *repo.messages[reply.ID].ThreadID
	assert.Equal(t, originalThread, replyThread)
}

func TestAssignThread_ReferencesChain(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockFolderRepo{})
	accountID := uuid.New()

	// Create original message
	original := newTestMessage(accountID)
	original.MessageID = "<original@example.com>"
	err := svc.Create(context.Background(), original)
	require.NoError(t, err)

	originalThread := *repo.messages[original.ID].ThreadID

	// Create reply using References (no InReplyTo)
	reply := newTestMessage(accountID)
	reply.MessageID = "<reply@example.com>"
	reply.References = []string{"<original@example.com>"}
	err = svc.Create(context.Background(), reply)
	require.NoError(t, err)

	replyThread := *repo.messages[reply.ID].ThreadID
	assert.Equal(t, originalThread, replyThread)
}

func TestAssignThread_DeepChain(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockFolderRepo{})
	accountID := uuid.New()

	// Create chain: A -> B -> C
	msgA := newTestMessage(accountID)
	msgA.MessageID = "<A@example.com>"
	require.NoError(t, svc.Create(context.Background(), msgA))
	threadA := *repo.messages[msgA.ID].ThreadID

	msgB := newTestMessage(accountID)
	msgB.MessageID = "<B@example.com>"
	msgB.InReplyTo = "<A@example.com>"
	msgB.References = []string{"<A@example.com>"}
	require.NoError(t, svc.Create(context.Background(), msgB))
	assert.Equal(t, threadA, *repo.messages[msgB.ID].ThreadID)

	msgC := newTestMessage(accountID)
	msgC.MessageID = "<C@example.com>"
	msgC.InReplyTo = "<B@example.com>"
	msgC.References = []string{"<A@example.com>", "<B@example.com>"}
	require.NoError(t, svc.Create(context.Background(), msgC))
	assert.Equal(t, threadA, *repo.messages[msgC.ID].ThreadID)
}

func TestNormalizeSubject(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Re: Hello", "Hello"},
		{"Fwd: Hello", "Hello"},
		{"AW: Hello", "Hello"},
		{"WG: Hello", "Hello"},
		{"RE: FW: Hello", "Hello"},
		{"Re: Re: Re: Hello", "Hello"},
		{"AW: WG: AW: Vertrag", "Vertrag"},
		{"Hello", "Hello"},
		{"", ""},
		{"Re:", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeSubject(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_MarkReadUnread(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockFolderRepo{})

	msg := newTestMessage(uuid.New())
	msg.IsRead = false
	require.NoError(t, svc.Create(context.Background(), msg))

	// Mark read
	require.NoError(t, svc.MarkRead(context.Background(), msg.ID))
	assert.True(t, repo.messages[msg.ID].IsRead)

	// Mark unread
	require.NoError(t, svc.MarkUnread(context.Background(), msg.ID))
	assert.False(t, repo.messages[msg.ID].IsRead)
}

func TestService_ToggleStar(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockFolderRepo{})

	msg := newTestMessage(uuid.New())
	msg.IsStarred = false
	require.NoError(t, svc.Create(context.Background(), msg))

	// Toggle on
	require.NoError(t, svc.ToggleStar(context.Background(), msg.ID, uuid.Nil))
	assert.True(t, repo.messages[msg.ID].IsStarred)

	// Toggle off
	require.NoError(t, svc.ToggleStar(context.Background(), msg.ID, uuid.Nil))
	assert.False(t, repo.messages[msg.ID].IsStarred)
}

func TestService_Delete(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockFolderRepo{})

	msg := newTestMessage(uuid.New())
	require.NoError(t, svc.Create(context.Background(), msg))

	require.NoError(t, svc.Delete(context.Background(), msg.ID))
	_, err := svc.GetByID(context.Background(), msg.ID, uuid.Nil)
	assert.ErrorIs(t, err, ErrMessageNotFound)
}

func TestReconstructThreads(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockFolderRepo{})

	accountID := uuid.New()
	msgs := []*models.EmailMessage{
		{ID: uuid.New(), AccountID: accountID, MessageID: "<1@ex>", Date: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New(), AccountID: accountID, MessageID: "<2@ex>", InReplyTo: "<1@ex>", References: []string{"<1@ex>"}, Date: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New(), AccountID: accountID, MessageID: "<3@ex>", InReplyTo: "<2@ex>", References: []string{"<1@ex>", "<2@ex>"}, Date: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, m := range msgs {
		_ = repo.Create(context.Background(), m)
	}

	err := svc.ReconstructThreads(context.Background(), msgs)
	require.NoError(t, err)

	// All should share same thread
	thread1 := *msgs[0].ThreadID
	assert.Equal(t, thread1, *msgs[1].ThreadID)
	assert.Equal(t, thread1, *msgs[2].ThreadID)
}

// ============================================================================
// Cross-tenant isolation tests (email_messages)
// ============================================================================

// TestGetByID_CrossTenant verifies that an email message owned by tenant A
// cannot be fetched with tenant B's ID.
func TestGetByID_CrossTenant(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockFolderRepo{})

	tenantA := uuid.New()
	tenantB := uuid.New()

	msg := newTestMessage(uuid.New())
	msg.TenantID = tenantA
	require.NoError(t, repo.Create(context.Background(), msg))

	result, err := svc.GetByID(context.Background(), msg.ID, tenantB)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrMessageNotFound)
}

// TestToggleStar_CrossTenant verifies that starring a message belonging to
// tenant A is rejected when the request carries tenant B's ID.
func TestToggleStar_CrossTenant(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockFolderRepo{})

	tenantA := uuid.New()
	tenantB := uuid.New()

	msg := newTestMessage(uuid.New())
	msg.TenantID = tenantA
	msg.IsStarred = false
	require.NoError(t, repo.Create(context.Background(), msg))

	err := svc.ToggleStar(context.Background(), msg.ID, tenantB)

	assert.ErrorIs(t, err, ErrMessageNotFound)
	// The star state must remain unchanged.
	assert.False(t, repo.messages[msg.ID].IsStarred)
}
