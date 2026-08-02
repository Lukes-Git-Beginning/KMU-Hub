package bookmark

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/chat/message"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type bookmarkKey struct {
	tenant, user, message uuid.UUID
}

type mockRepo struct {
	bookmarks   []bookmarkKey // insertion order, newest last
	addCalls    int
	removeCalls int
}

func (m *mockRepo) Exists(_ context.Context, tenantID, userID, messageID uuid.UUID) (bool, error) {
	key := bookmarkKey{tenantID, userID, messageID}
	return slices.Contains(m.bookmarks, key), nil
}

func (m *mockRepo) Add(_ context.Context, tenantID, userID, messageID uuid.UUID) error {
	m.addCalls++
	key := bookmarkKey{tenantID, userID, messageID}
	if slices.Contains(m.bookmarks, key) {
		return nil // idempotent, matches ON CONFLICT DO NOTHING
	}
	m.bookmarks = append(m.bookmarks, key)
	return nil
}

func (m *mockRepo) Remove(_ context.Context, tenantID, userID, messageID uuid.UUID) error {
	m.removeCalls++
	key := bookmarkKey{tenantID, userID, messageID}
	for i, b := range m.bookmarks {
		if b == key {
			m.bookmarks = append(m.bookmarks[:i], m.bookmarks[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepo) ListMessageIDs(_ context.Context, tenantID, userID uuid.UUID) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0)
	// Reverse insertion order: most recently bookmarked first, like ORDER BY created_at DESC.
	for i := len(m.bookmarks) - 1; i >= 0; i-- {
		b := m.bookmarks[i]
		if b.tenant == tenantID && b.user == userID {
			out = append(out, b.message)
		}
	}
	return out, nil
}

type mockMessages struct {
	messages map[uuid.UUID]*models.MessageWithSender
	// notMember marks message ids that exist but the caller is not a member
	// of their channel -- distinct from "does not exist at all".
	notMember map[uuid.UUID]bool
	getCalls  int
}

func (m *mockMessages) GetByID(_ context.Context, id, tenantID, _ uuid.UUID) (*models.MessageWithSender, error) {
	m.getCalls++
	if m.notMember[id] {
		return nil, message.ErrNotChannelMember
	}
	msg, ok := m.messages[id]
	if !ok || msg.TenantID != tenantID {
		return nil, message.ErrMessageNotFound
	}
	return msg, nil
}

func withID(id, tenant uuid.UUID) *models.MessageWithSender {
	m := &models.MessageWithSender{}
	m.ID = id
	m.TenantID = tenant
	return m
}

// ---------------------------------------------------------------------------
// Toggle
// ---------------------------------------------------------------------------

func TestToggle_AddsThenRemoves(t *testing.T) {
	t.Parallel()
	tenant, user, msgID := uuid.New(), uuid.New(), uuid.New()
	messages := &mockMessages{messages: map[uuid.UUID]*models.MessageWithSender{msgID: withID(msgID, tenant)}}
	repo := &mockRepo{}
	svc := NewService(repo, messages)

	bookmarked, err := svc.Toggle(context.Background(), tenant, user, msgID)
	if err != nil {
		t.Fatalf("first Toggle: %v", err)
	}
	if !bookmarked {
		t.Fatalf("bookmarked = false, want true after first toggle")
	}
	if repo.addCalls != 1 {
		t.Errorf("addCalls = %d, want 1", repo.addCalls)
	}

	bookmarked, err = svc.Toggle(context.Background(), tenant, user, msgID)
	if err != nil {
		t.Fatalf("second Toggle: %v", err)
	}
	if bookmarked {
		t.Fatalf("bookmarked = true, want false after second toggle")
	}
	if repo.removeCalls != 1 {
		t.Errorf("removeCalls = %d, want 1", repo.removeCalls)
	}
}

func TestToggle_UnknownMessagePropagatesNotFound(t *testing.T) {
	t.Parallel()
	repo := &mockRepo{}
	messages := &mockMessages{messages: map[uuid.UUID]*models.MessageWithSender{}}
	svc := NewService(repo, messages)

	if _, err := svc.Toggle(context.Background(), uuid.New(), uuid.New(), uuid.New()); !errors.Is(err, message.ErrMessageNotFound) {
		t.Fatalf("err = %v, want %v", err, message.ErrMessageNotFound)
	}
	if repo.addCalls != 0 {
		t.Errorf("addCalls = %d, want 0 (write must not happen before visibility check passes)", repo.addCalls)
	}
}

func TestToggle_NotChannelMemberPropagates(t *testing.T) {
	t.Parallel()
	msgID := uuid.New()
	repo := &mockRepo{}
	messages := &mockMessages{notMember: map[uuid.UUID]bool{msgID: true}}
	svc := NewService(repo, messages)

	if _, err := svc.Toggle(context.Background(), uuid.New(), uuid.New(), msgID); !errors.Is(err, message.ErrNotChannelMember) {
		t.Fatalf("err = %v, want %v", err, message.ErrNotChannelMember)
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList_ReturnsNewestBookmarkFirst(t *testing.T) {
	t.Parallel()
	tenant, user := uuid.New(), uuid.New()
	first, second := uuid.New(), uuid.New()
	messages := &mockMessages{messages: map[uuid.UUID]*models.MessageWithSender{
		first:  withID(first, tenant),
		second: withID(second, tenant),
	}}
	repo := &mockRepo{}
	svc := NewService(repo, messages)

	if _, err := svc.Toggle(context.Background(), tenant, user, first); err != nil {
		t.Fatalf("Toggle first: %v", err)
	}
	if _, err := svc.Toggle(context.Background(), tenant, user, second); err != nil {
		t.Fatalf("Toggle second: %v", err)
	}

	got, err := svc.List(context.Background(), tenant, user)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 || got[0].ID != second || got[1].ID != first {
		t.Fatalf("List = %v, want [second, first]", got)
	}
}

func TestList_SkipsInaccessibleMessages(t *testing.T) {
	t.Parallel()
	tenant, user := uuid.New(), uuid.New()
	visible, revoked := uuid.New(), uuid.New()
	messages := &mockMessages{
		messages:  map[uuid.UUID]*models.MessageWithSender{visible: withID(visible, tenant)},
		notMember: map[uuid.UUID]bool{},
	}
	repo := &mockRepo{}
	svc := NewService(repo, messages)

	if _, err := svc.Toggle(context.Background(), tenant, user, visible); err != nil {
		t.Fatalf("Toggle visible: %v", err)
	}
	// Simulate a message that existed (and was bookmarked) but whose channel
	// the caller has since left: mark it notMember only after bookmarking.
	messages.messages[revoked] = withID(revoked, tenant)
	if _, err := svc.Toggle(context.Background(), tenant, user, revoked); err != nil {
		t.Fatalf("Toggle revoked: %v", err)
	}
	messages.notMember[revoked] = true

	got, err := svc.List(context.Background(), tenant, user)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].ID != visible {
		t.Fatalf("List = %v, want only [visible] (revoked-access bookmark must be skipped, not erred)", got)
	}
}

func TestList_EmptyIsEmptyNotNil(t *testing.T) {
	t.Parallel()
	svc := NewService(&mockRepo{}, &mockMessages{messages: map[uuid.UUID]*models.MessageWithSender{}})

	got, err := svc.List(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got == nil {
		t.Fatal("List returned nil, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("List = %v, want empty", got)
	}
}
