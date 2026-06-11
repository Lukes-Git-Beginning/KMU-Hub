package customfield_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kmuhub/kmuhub/internal/work/customfield"
)

// ============================================================================
// Mock repository
// ============================================================================

type mockCFRepo struct {
	defs map[string]*customfield.Definition // id → definition
}

func newMockCFRepo() *mockCFRepo {
	return &mockCFRepo{defs: make(map[string]*customfield.Definition)}
}

func (m *mockCFRepo) Create(_ context.Context, tenantID, name, fieldType string, options []string, position int) (*customfield.Definition, error) {
	for _, d := range m.defs {
		if d.TenantID == tenantID && d.Name == name {
			return nil, customfield.ErrDuplicateName
		}
	}
	d := &customfield.Definition{
		ID:        tenantID + ":" + name,
		TenantID:  tenantID,
		Name:      name,
		FieldType: fieldType,
		Options:   options,
		Position:  position,
	}
	m.defs[d.ID] = d
	return d, nil
}

func (m *mockCFRepo) GetByID(_ context.Context, tenantID, id string) (*customfield.Definition, error) {
	d, ok := m.defs[id]
	if !ok || d.TenantID != tenantID {
		return nil, customfield.ErrNotFound
	}
	return d, nil
}

func (m *mockCFRepo) List(_ context.Context, tenantID string) ([]*customfield.Definition, error) {
	var result []*customfield.Definition
	for _, d := range m.defs {
		if d.TenantID == tenantID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockCFRepo) Update(_ context.Context, tenantID, id, name, fieldType string, options []string, position int) (*customfield.Definition, error) {
	d, ok := m.defs[id]
	if !ok || d.TenantID != tenantID {
		return nil, customfield.ErrNotFound
	}
	d.Name = name
	d.FieldType = fieldType
	d.Options = options
	d.Position = position
	return d, nil
}

func (m *mockCFRepo) Delete(_ context.Context, tenantID, id string) error {
	d, ok := m.defs[id]
	if !ok || d.TenantID != tenantID {
		return customfield.ErrNotFound
	}
	delete(m.defs, id)
	return nil
}

// ============================================================================
// Tests
// ============================================================================

func TestCustomFieldService_Create(t *testing.T) {
	ctx := context.Background()
	svc := customfield.NewService(newMockCFRepo())

	t.Run("creates definition successfully", func(t *testing.T) {
		d, err := svc.Create(ctx, "tenant-1", "Priority", "select", []string{"High", "Low"}, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.Name != "Priority" || d.FieldType != "select" {
			t.Errorf("unexpected definition: %+v", d)
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		_, err := svc.Create(ctx, "tenant-1", "", "text", nil, 0)
		if !errors.Is(err, customfield.ErrNameRequired) {
			t.Errorf("expected ErrNameRequired, got %v", err)
		}
	})

	t.Run("rejects invalid field type", func(t *testing.T) {
		_, err := svc.Create(ctx, "tenant-1", "Bad", "invalid_type", nil, 0)
		if !errors.Is(err, customfield.ErrInvalidFieldType) {
			t.Errorf("expected ErrInvalidFieldType, got %v", err)
		}
	})

	t.Run("accepts all valid field types", func(t *testing.T) {
		validTypes := []string{"text", "number", "date", "boolean", "select", "multi_select", "url", "email", "phone"}
		for i, ft := range validTypes {
			name := ft + "_field"
			_, err := svc.Create(ctx, "tenant-valid", name, ft, nil, i)
			if err != nil {
				t.Errorf("fieldType %q should be valid, got error: %v", ft, err)
			}
		}
	})
}

func TestCustomFieldService_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	repo := newMockCFRepo()
	svc := customfield.NewService(repo)

	// Seed definitions in two tenants
	d1, _ := svc.Create(ctx, "tenant-a", "Notes", "text", nil, 0)
	_, _ = svc.Create(ctx, "tenant-b", "Score", "number", nil, 0)

	t.Run("list returns only own tenant definitions", func(t *testing.T) {
		defs, err := svc.List(ctx, "tenant-a")
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		for _, d := range defs {
			if d.TenantID != "tenant-a" {
				t.Errorf("tenant isolation broken: got definition from tenant %q", d.TenantID)
			}
		}
	})

	t.Run("cannot get definition from other tenant", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "tenant-b", d1.ID)
		if !errors.Is(err, customfield.ErrNotFound) {
			t.Errorf("expected ErrNotFound for cross-tenant get, got %v", err)
		}
	})

	t.Run("cannot delete definition from other tenant", func(t *testing.T) {
		err := svc.Delete(ctx, "tenant-b", d1.ID)
		if !errors.Is(err, customfield.ErrNotFound) {
			t.Errorf("expected ErrNotFound for cross-tenant delete, got %v", err)
		}
	})
}

func TestCustomFieldService_DuplicateName(t *testing.T) {
	ctx := context.Background()
	svc := customfield.NewService(newMockCFRepo())

	_, err := svc.Create(ctx, "tenant-1", "Effort", "number", nil, 0)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = svc.Create(ctx, "tenant-1", "Effort", "text", nil, 1)
	if !errors.Is(err, customfield.ErrDuplicateName) {
		t.Errorf("expected ErrDuplicateName, got %v", err)
	}
}

func TestCustomFieldService_SameNameDifferentTenants(t *testing.T) {
	ctx := context.Background()
	svc := customfield.NewService(newMockCFRepo())

	_, err := svc.Create(ctx, "tenant-1", "Effort", "number", nil, 0)
	if err != nil {
		t.Fatalf("tenant-1 create: %v", err)
	}
	_, err = svc.Create(ctx, "tenant-2", "Effort", "number", nil, 0)
	if err != nil {
		t.Errorf("same name in different tenant should be allowed, got: %v", err)
	}
}
