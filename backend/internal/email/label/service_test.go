package label

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockRepo struct {
	labels   []*models.EmailLabel
	messages map[uuid.UUID][]uuid.UUID // message id -> label ids
	assigns  int
}

func newMockRepo() *mockRepo {
	return &mockRepo{messages: map[uuid.UUID][]uuid.UUID{}}
}

func (m *mockRepo) Create(_ context.Context, l *models.EmailLabel) error {
	for _, existing := range m.labels {
		if existing.TenantID == l.TenantID && existing.Name == l.Name {
			return ErrDuplicateName
		}
	}
	m.labels = append(m.labels, l)
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.EmailLabel, error) {
	for _, l := range m.labels {
		if l.ID == id && l.TenantID == tenantID {
			return l, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) List(_ context.Context, tenantID uuid.UUID) ([]*models.EmailLabel, error) {
	out := make([]*models.EmailLabel, 0)
	for _, l := range m.labels {
		if l.TenantID == tenantID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *mockRepo) Update(_ context.Context, l *models.EmailLabel) error {
	for i, existing := range m.labels {
		if existing.ID == l.ID && existing.TenantID == l.TenantID {
			m.labels[i] = l
			return nil
		}
	}
	return ErrNotFound
}

func (m *mockRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	for i, l := range m.labels {
		if l.ID == id && l.TenantID == tenantID {
			m.labels = append(m.labels[:i], m.labels[i+1:]...)
			for msgID, ids := range m.messages {
				m.messages[msgID] = removeUUID(ids, id)
			}
			return nil
		}
	}
	return ErrNotFound
}

func (m *mockRepo) LabelIDsBelongToTenant(_ context.Context, tenantID uuid.UUID, ids []uuid.UUID) (bool, error) {
	for _, id := range ids {
		found := false
		for _, l := range m.labels {
			if l.ID == id && l.TenantID == tenantID {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}

func (m *mockRepo) AssignToMessage(_ context.Context, _, messageID uuid.UUID, labelIDs []uuid.UUID) error {
	m.assigns++
	if _, ok := m.messages[messageID]; !ok {
		return ErrMessageNotFound
	}
	m.messages[messageID] = labelIDs
	return nil
}

func removeUUID(ids []uuid.UUID, target uuid.UUID) []uuid.UUID {
	out := ids[:0]
	for _, id := range ids {
		if id != target {
			out = append(out, id)
		}
	}
	return out
}

type mockMessages struct {
	messages map[uuid.UUID]*models.EmailMessage
}

func (m *mockMessages) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.EmailMessage, error) {
	msg, ok := m.messages[id]
	if !ok || msg.TenantID != tenantID {
		return nil, errors.New("message not found")
	}
	return msg, nil
}

// ---------------------------------------------------------------------------
// Create / Update / Delete
// ---------------------------------------------------------------------------

func TestCreate_Valid(t *testing.T) {
	t.Parallel()
	svc := NewService(newMockRepo(), &mockMessages{})
	tenant := uuid.New()

	l, err := svc.Create(context.Background(), tenant, "Wichtig", "#EF4444")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if l.TenantID != tenant || l.Name != "Wichtig" || l.Color != "#EF4444" {
		t.Errorf("label = %+v, unexpected", l)
	}
}

func TestCreate_DefaultsColorWhenBlank(t *testing.T) {
	t.Parallel()
	svc := NewService(newMockRepo(), &mockMessages{})

	l, err := svc.Create(context.Background(), uuid.New(), "Wichtig", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if l.Color != defaultColor {
		t.Errorf("color = %q, want default %q", l.Color, defaultColor)
	}
}

func TestCreate_Rejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, label, color string
		want               error
	}{
		{"blank name", "   ", "#EF4444", ErrNameRequired},
		{"bad color", "Wichtig", "red", ErrColorInvalid},
		{"short hex", "Wichtig", "#FFF", ErrColorInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := NewService(newMockRepo(), &mockMessages{})
			if _, err := svc.Create(context.Background(), uuid.New(), tc.label, tc.color); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCreate_DuplicateNameWithinTenant(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo, &mockMessages{})
	tenant := uuid.New()

	if _, err := svc.Create(context.Background(), tenant, "Wichtig", "#EF4444"); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if _, err := svc.Create(context.Background(), tenant, "Wichtig", "#3B82F6"); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("err = %v, want %v", err, ErrDuplicateName)
	}
	// Same name, different tenant must not collide.
	if _, err := svc.Create(context.Background(), uuid.New(), "Wichtig", "#3B82F6"); err != nil {
		t.Fatalf("other tenant rejected: %v", err)
	}
}

func TestUpdate_PartialKeepsUntouchedFields(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo, &mockMessages{})
	tenant := uuid.New()

	created, err := svc.Create(context.Background(), tenant, "Wichtig", "#EF4444")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newName := "Sehr wichtig"
	updated, err := svc.Update(context.Background(), created.ID, tenant, &newName, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("name = %q, want %q", updated.Name, newName)
	}
	if updated.Color != created.Color {
		t.Errorf("color changed: %q, want %q", updated.Color, created.Color)
	}
}

func TestUpdate_RejectsInvalidColorOnMerge(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo, &mockMessages{})
	tenant := uuid.New()

	created, err := svc.Create(context.Background(), tenant, "Wichtig", "#EF4444")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	badColor := "not-a-color"
	if _, err := svc.Update(context.Background(), created.ID, tenant, nil, &badColor); !errors.Is(err, ErrColorInvalid) {
		t.Fatalf("err = %v, want %v", err, ErrColorInvalid)
	}
}

func TestUpdate_ForeignTenantIsNotFound(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo, &mockMessages{})
	tenant := uuid.New()

	created, err := svc.Create(context.Background(), tenant, "Wichtig", "#EF4444")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newName := "x"
	if _, err := svc.Update(context.Background(), created.ID, uuid.New(), &newName, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrNotFound)
	}
	if err := svc.Delete(context.Background(), created.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete err = %v, want %v", err, ErrNotFound)
	}
}

// ---------------------------------------------------------------------------
// AssignToMessage
// ---------------------------------------------------------------------------

func TestAssignToMessage_ReplacesSetAndReturnsFreshMessage(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	tenant := uuid.New()
	messageID := uuid.New()
	repo.messages[messageID] = nil

	seed := NewService(repo, &mockMessages{})
	created, err := seed.Create(context.Background(), tenant, "Wichtig", "#EF4444")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	afterAssign := &models.EmailMessage{ID: messageID, TenantID: tenant, LabelIDs: []uuid.UUID{created.ID}}
	messages := &mockMessages{messages: map[uuid.UUID]*models.EmailMessage{messageID: afterAssign}}
	svc := NewService(repo, messages)

	msg, err := svc.AssignToMessage(context.Background(), tenant, messageID, []uuid.UUID{created.ID})
	if err != nil {
		t.Fatalf("AssignToMessage: %v", err)
	}
	if msg.ID != messageID {
		t.Errorf("message id = %s, want %s", msg.ID, messageID)
	}
	if repo.assigns != 1 {
		t.Errorf("assigns = %d, want 1", repo.assigns)
	}
}

func TestAssignToMessage_RejectsForeignLabelID(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	tenant := uuid.New()
	messageID := uuid.New()
	repo.messages[messageID] = nil
	svc := NewService(repo, &mockMessages{messages: map[uuid.UUID]*models.EmailMessage{}})

	foreignLabel := uuid.New()
	if _, err := svc.AssignToMessage(context.Background(), tenant, messageID, []uuid.UUID{foreignLabel}); !errors.Is(err, ErrTargetInvalid) {
		t.Fatalf("err = %v, want %v", err, ErrTargetInvalid)
	}
	if repo.assigns != 0 {
		t.Errorf("assigns = %d, want 0 (write must not happen before validation passes)", repo.assigns)
	}
}

func TestAssignToMessage_ForeignMessageIsNotFound(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo, &mockMessages{messages: map[uuid.UUID]*models.EmailMessage{}})

	if _, err := svc.AssignToMessage(context.Background(), uuid.New(), uuid.New(), nil); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrMessageNotFound)
	}
}

func TestAssignToMessage_EmptySetIsValid(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	tenant := uuid.New()
	messageID := uuid.New()
	repo.messages[messageID] = []uuid.UUID{uuid.New()}

	cleared := &models.EmailMessage{ID: messageID, TenantID: tenant, LabelIDs: []uuid.UUID{}}
	svc := NewService(repo, &mockMessages{messages: map[uuid.UUID]*models.EmailMessage{messageID: cleared}})

	msg, err := svc.AssignToMessage(context.Background(), tenant, messageID, nil)
	if err != nil {
		t.Fatalf("AssignToMessage: %v", err)
	}
	if len(msg.LabelIDs) != 0 {
		t.Errorf("label_ids = %v, want empty", msg.LabelIDs)
	}
}
