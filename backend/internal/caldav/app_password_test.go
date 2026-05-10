package caldav

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// mockAppPasswordRepo captures the AppSpecificPassword passed to Create.
type mockAppPasswordRepo struct {
	created []*models.AppSpecificPassword
}

func (m *mockAppPasswordRepo) Create(_ context.Context, pw *models.AppSpecificPassword) error {
	m.created = append(m.created, pw)
	return nil
}

func (m *mockAppPasswordRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]*models.AppSpecificPassword, error) {
	return nil, nil
}

func (m *mockAppPasswordRepo) Revoke(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (m *mockAppPasswordRepo) FindActiveByUser(_ context.Context, _ uuid.UUID) ([]*models.AppSpecificPassword, error) {
	return nil, nil
}

func (m *mockAppPasswordRepo) UpdateLastUsed(_ context.Context, _ uuid.UUID) error {
	return nil
}

// Compile-time interface check.
var _ AppPasswordRepository = (*mockAppPasswordRepo)(nil)

// TestAppPasswordService_Create_TenantID verifies that AppPasswordService.Create
// propagates tenantID into the AppSpecificPassword entity that is persisted.
// This is the wiring-gap closure for app_specific_passwords.tenant_id NOT NULL
// (Sprint 4 Welle 1b Stream A).
func TestAppPasswordService_Create_TenantID(t *testing.T) {
	repo := &mockAppPasswordRepo{}
	svc := NewAppPasswordService(repo, nil) // pool nil: Create only uses repo

	userID := uuid.New()
	tenantID := uuid.New()

	plaintext, pw, err := svc.Create(context.Background(), userID, tenantID, "test label")
	require.NoError(t, err)
	assert.NotEmpty(t, plaintext, "plaintext password must not be empty")
	assert.Equal(t, tenantID, pw.TenantID,
		"AppSpecificPassword.TenantID must equal the provided tenantID (not uuid.Nil)")
	assert.Equal(t, userID, pw.UserID)

	require.Len(t, repo.created, 1)
	assert.Equal(t, tenantID, repo.created[0].TenantID,
		"persisted AppSpecificPassword must carry tenant_id (not uuid.Nil)")
}

// TestPushSubscription_TenantID verifies that the PushSubscription struct
// carries TenantID (wiring-gap closure for caldav_push_subscriptions.tenant_id
// NOT NULL — Sprint 4 Welle 1b Stream A).
// Note: Subscribe() requires a live DB; this test validates struct initialization only.
func TestPushSubscription_TenantID(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	sub := PushSubscription{
		ID:             uuid.New(),
		TenantID:       tenantID,
		UserID:         userID,
		CollectionType: "calendar",
		CollectionID:   uuid.New(),
		PushURL:        "https://example.com/push",
		PushTopic:      "test-topic",
	}

	assert.Equal(t, tenantID, sub.TenantID,
		"PushSubscription.TenantID must be set correctly")
	assert.Equal(t, userID, sub.UserID)
}
