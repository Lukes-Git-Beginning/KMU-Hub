package label_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kmuhub/kmuhub/internal/work/label"
)

// ============================================================================
// Mock repository
// ============================================================================

type mockLabelRepo struct {
	labels     map[string]*label.Label // id → label
	taskLabels map[string][]string     // taskID → []labelID
	shouldFail bool
}

func newMockRepo() *mockLabelRepo {
	return &mockLabelRepo{
		labels:     make(map[string]*label.Label),
		taskLabels: make(map[string][]string),
	}
}

func (m *mockLabelRepo) Create(_ context.Context, tenantID, name, color string) (*label.Label, error) {
	if m.shouldFail {
		return nil, errors.New("db error")
	}
	// Check for duplicate within tenant
	for _, l := range m.labels {
		if l.TenantID == tenantID && l.Name == name {
			return nil, label.ErrDuplicateName
		}
	}
	l := &label.Label{
		ID:       tenantID + ":" + name, // deterministic test ID
		TenantID: tenantID,
		Name:     name,
		Color:    color,
	}
	m.labels[l.ID] = l
	return l, nil
}

func (m *mockLabelRepo) GetByID(_ context.Context, tenantID, id string) (*label.Label, error) {
	l, ok := m.labels[id]
	if !ok || l.TenantID != tenantID {
		return nil, label.ErrNotFound
	}
	return l, nil
}

func (m *mockLabelRepo) List(_ context.Context, tenantID string) ([]*label.Label, error) {
	var result []*label.Label
	for _, l := range m.labels {
		if l.TenantID == tenantID {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *mockLabelRepo) Update(_ context.Context, tenantID, id, name, color string) (*label.Label, error) {
	l, ok := m.labels[id]
	if !ok || l.TenantID != tenantID {
		return nil, label.ErrNotFound
	}
	l.Name = name
	l.Color = color
	return l, nil
}

func (m *mockLabelRepo) Delete(_ context.Context, tenantID, id string) error {
	l, ok := m.labels[id]
	if !ok || l.TenantID != tenantID {
		return label.ErrNotFound
	}
	delete(m.labels, id)
	return nil
}

func (m *mockLabelRepo) GetLabelsByTaskIDs(_ context.Context, tenantID string, taskIDs []string) (map[string][]*label.Label, error) {
	result := make(map[string][]*label.Label)
	for _, taskID := range taskIDs {
		ids := m.taskLabels[taskID]
		for _, lid := range ids {
			if l, ok := m.labels[lid]; ok && l.TenantID == tenantID {
				result[taskID] = append(result[taskID], l)
			}
		}
	}
	return result, nil
}

func (m *mockLabelRepo) SetTaskLabels(_ context.Context, tenantID, taskID string, labelIDs []string) error {
	// Verify all label IDs belong to tenant
	for _, lid := range labelIDs {
		l, ok := m.labels[lid]
		if !ok || l.TenantID != tenantID {
			return label.ErrNotFound
		}
	}
	m.taskLabels[taskID] = labelIDs
	return nil
}

func (m *mockLabelRepo) GetTaskLabelIDs(_ context.Context, tenantID, taskID string) ([]string, error) {
	ids := m.taskLabels[taskID]
	// Filter to tenant-owned labels only
	var result []string
	for _, lid := range ids {
		if l, ok := m.labels[lid]; ok && l.TenantID == tenantID {
			result = append(result, lid)
		}
	}
	return result, nil
}

// ============================================================================
// Tests
// ============================================================================

func TestLabelService_Create(t *testing.T) {
	ctx := context.Background()
	svc := label.NewService(newMockRepo())

	t.Run("creates label successfully", func(t *testing.T) {
		l, err := svc.Create(ctx, "tenant-1", "Bug", "#ef4444")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if l.Name != "Bug" || l.Color != "#ef4444" {
			t.Errorf("unexpected label: %+v", l)
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		_, err := svc.Create(ctx, "tenant-1", "", "#ef4444")
		if !errors.Is(err, label.ErrNameRequired) {
			t.Errorf("expected ErrNameRequired, got %v", err)
		}
	})

	t.Run("rejects invalid color", func(t *testing.T) {
		_, err := svc.Create(ctx, "tenant-1", "Valid", "not-a-color")
		if !errors.Is(err, label.ErrColorInvalid) {
			t.Errorf("expected ErrColorInvalid, got %v", err)
		}
	})
}

func TestLabelService_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	svc := label.NewService(repo)

	// Create labels in two different tenants
	l1, err := svc.Create(ctx, "tenant-a", "Feature", "#3b82f6")
	if err != nil {
		t.Fatalf("create tenant-a label: %v", err)
	}
	_, err = svc.Create(ctx, "tenant-b", "Bug", "#ef4444")
	if err != nil {
		t.Fatalf("create tenant-b label: %v", err)
	}

	t.Run("list returns only own tenant labels", func(t *testing.T) {
		labels, err := svc.List(ctx, "tenant-a")
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		for _, l := range labels {
			if l.TenantID != "tenant-a" {
				t.Errorf("tenant isolation broken: got label from tenant %q", l.TenantID)
			}
		}
	})

	t.Run("cannot get label from other tenant", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "tenant-b", l1.ID)
		if !errors.Is(err, label.ErrNotFound) {
			t.Errorf("expected ErrNotFound for cross-tenant get, got %v", err)
		}
	})

	t.Run("cannot delete label from other tenant", func(t *testing.T) {
		err := svc.Delete(ctx, "tenant-b", l1.ID)
		if !errors.Is(err, label.ErrNotFound) {
			t.Errorf("expected ErrNotFound for cross-tenant delete, got %v", err)
		}
	})
}

func TestLabelService_DuplicateName(t *testing.T) {
	ctx := context.Background()
	svc := label.NewService(newMockRepo())

	_, err := svc.Create(ctx, "tenant-1", "Sprint", "#6b7280")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = svc.Create(ctx, "tenant-1", "Sprint", "#6b7280")
	if !errors.Is(err, label.ErrDuplicateName) {
		t.Errorf("expected ErrDuplicateName, got %v", err)
	}
}

func TestLabelService_SameNameDifferentTenants(t *testing.T) {
	ctx := context.Background()
	svc := label.NewService(newMockRepo())

	_, err := svc.Create(ctx, "tenant-1", "Sprint", "#6b7280")
	if err != nil {
		t.Fatalf("tenant-1 create: %v", err)
	}
	// Same name in different tenant is allowed
	_, err = svc.Create(ctx, "tenant-2", "Sprint", "#6b7280")
	if err != nil {
		t.Errorf("same name in different tenant should be allowed, got: %v", err)
	}
}

func TestLabelService_SetTaskLabels(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	svc := label.NewService(repo)

	l, _ := svc.Create(ctx, "tenant-1", "Urgent", "#f97316")

	t.Run("sets and retrieves task labels", func(t *testing.T) {
		err := svc.SetTaskLabels(ctx, "tenant-1", "task-abc", []string{l.ID})
		if err != nil {
			t.Fatalf("SetTaskLabels: %v", err)
		}
		ids, err := svc.GetTaskLabelIDs(ctx, "tenant-1", "task-abc")
		if err != nil {
			t.Fatalf("GetTaskLabelIDs: %v", err)
		}
		if len(ids) != 1 || ids[0] != l.ID {
			t.Errorf("expected [%q], got %v", l.ID, ids)
		}
	})

	t.Run("cannot assign label from other tenant", func(t *testing.T) {
		err := svc.SetTaskLabels(ctx, "tenant-2", "task-xyz", []string{l.ID})
		if !errors.Is(err, label.ErrNotFound) {
			t.Errorf("expected ErrNotFound for cross-tenant label assignment, got %v", err)
		}
	})
}
