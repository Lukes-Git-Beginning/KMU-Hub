package contact

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// mockContactRepository implements ContactRepository for testing.
type mockContactRepository struct {
	lastContactID  uuid.UUID
	lastVisibility string
	lastOwnerID    *uuid.UUID
	callCount      int
}

func (m *mockContactRepository) UpdateVisibility(_ context.Context, contactID uuid.UUID, visibility string, ownerID *uuid.UUID, _ uuid.UUID) error {
	m.lastContactID = contactID
	m.lastVisibility = visibility
	m.lastOwnerID = ownerID
	m.callCount++
	return nil
}

func TestSetVisibility_Shared(t *testing.T) {
	repo := &mockContactRepository{}
	svc := NewVisibilityService(repo, nil)

	contactID := uuid.New()
	userID := uuid.New()
	tenantID := uuid.New()

	err := svc.SetVisibility(context.Background(), contactID, VisibilityShared, userID, tenantID)
	if err != nil {
		t.Fatalf("SetVisibility failed: %v", err)
	}

	if repo.lastVisibility != VisibilityShared {
		t.Errorf("expected shared, got %s", repo.lastVisibility)
	}
	if repo.lastOwnerID != nil {
		t.Error("expected nil ownerID for shared visibility")
	}
}

func TestSetVisibility_Personal(t *testing.T) {
	repo := &mockContactRepository{}
	svc := NewVisibilityService(repo, nil)

	contactID := uuid.New()
	userID := uuid.New()
	tenantID := uuid.New()

	err := svc.SetVisibility(context.Background(), contactID, VisibilityPersonal, userID, tenantID)
	if err != nil {
		t.Fatalf("SetVisibility failed: %v", err)
	}

	if repo.lastVisibility != VisibilityPersonal {
		t.Errorf("expected personal, got %s", repo.lastVisibility)
	}
	if repo.lastOwnerID == nil || *repo.lastOwnerID != userID {
		t.Error("expected ownerID to match userID for personal visibility")
	}
}

func TestSetVisibility_Invalid(t *testing.T) {
	repo := &mockContactRepository{}
	svc := NewVisibilityService(repo, nil)

	err := svc.SetVisibility(context.Background(), uuid.New(), "invalid", uuid.New(), uuid.New())
	if err != ErrInvalidVisibility {
		t.Errorf("expected ErrInvalidVisibility, got %v", err)
	}

	if repo.callCount != 0 {
		t.Error("expected no repo call for invalid visibility")
	}
}

func TestAdminOverride_Shared(t *testing.T) {
	repo := &mockContactRepository{}
	svc := NewVisibilityService(repo, nil)

	contactID := uuid.New()
	tenantID := uuid.New()

	err := svc.AdminOverride(context.Background(), contactID, VisibilityShared, tenantID)
	if err != nil {
		t.Fatalf("AdminOverride failed: %v", err)
	}

	if repo.lastVisibility != VisibilityShared {
		t.Errorf("expected shared, got %s", repo.lastVisibility)
	}
	if repo.lastOwnerID != nil {
		t.Error("admin override to shared should clear owner_id")
	}
}

func TestAdminOverride_Invalid(t *testing.T) {
	repo := &mockContactRepository{}
	svc := NewVisibilityService(repo, nil)

	err := svc.AdminOverride(context.Background(), uuid.New(), "bogus", uuid.New())
	if err != ErrInvalidVisibility {
		t.Errorf("expected ErrInvalidVisibility, got %v", err)
	}
}

func TestSetOwner(t *testing.T) {
	repo := &mockContactRepository{}
	svc := NewVisibilityService(repo, nil)

	contactID := uuid.New()
	ownerID := uuid.New()
	tenantID := uuid.New()

	err := svc.SetOwner(context.Background(), contactID, ownerID, tenantID)
	if err != nil {
		t.Fatalf("SetOwner failed: %v", err)
	}

	if repo.lastVisibility != VisibilityPersonal {
		t.Errorf("expected personal, got %s", repo.lastVisibility)
	}
	if repo.lastOwnerID == nil || *repo.lastOwnerID != ownerID {
		t.Errorf("expected ownerID %s, got %v", ownerID, repo.lastOwnerID)
	}
}

func TestIsValidVisibility(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"shared", true},
		{"personal", true},
		{"", false},
		{"public", false},
		{"private", false},
	}

	for _, tt := range tests {
		if got := isValidVisibility(tt.input); got != tt.valid {
			t.Errorf("isValidVisibility(%q) = %v, want %v", tt.input, got, tt.valid)
		}
	}
}
