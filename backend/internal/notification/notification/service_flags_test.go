package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

type flagAction struct {
	name  string
	call  func(*Service, context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*models.Notification, error)
	state func(*models.Notification) bool
	want  bool
}

func flagActions() []flagAction {
	return []flagAction{
		{"pin", (*Service).PinNotification, func(n *models.Notification) bool { return n.IsPinned }, true},
		{"unpin", (*Service).UnpinNotification, func(n *models.Notification) bool { return n.IsPinned }, false},
		{"dismiss", (*Service).DismissNotification, func(n *models.Notification) bool { return n.IsDismissed }, true},
	}
}

func TestPinUnpinDismiss_UpdateOwnNotification(t *testing.T) {
	for _, a := range flagActions() {
		t.Run(a.name, func(t *testing.T) {
			svc, repo, _ := setupTestService(t)
			tenantID, userID, id := uuid.New(), uuid.New(), uuid.New()
			// Start from the opposite state so the call has to flip it.
			repo.notifications = append(repo.notifications, &models.Notification{
				ID: id, TenantID: tenantID, UserID: userID, IsPinned: !a.want && a.name == "unpin",
			})

			got, err := a.call(svc, context.Background(), tenantID, id, userID)
			require.NoError(t, err)
			assert.Equal(t, a.want, a.state(got), "returned notification")
			assert.Equal(t, a.want, a.state(repo.notifications[0]), "persisted notification")
		})
	}
}

func TestPinUnpinDismiss_RejectForeignUserWithoutWriting(t *testing.T) {
	for _, a := range flagActions() {
		t.Run(a.name, func(t *testing.T) {
			svc, repo, _ := setupTestService(t)
			tenantID, id := uuid.New(), uuid.New()
			repo.notifications = append(repo.notifications, &models.Notification{
				ID: id, TenantID: tenantID, UserID: uuid.New(), IsPinned: a.name == "unpin",
			})
			before := *repo.notifications[0]

			_, err := a.call(svc, context.Background(), tenantID, id, uuid.New())
			assert.ErrorIs(t, err, ErrUnauthorized)
			assert.Equal(t, before, *repo.notifications[0], "a rejected call must not touch the row")
		})
	}
}

func TestPinUnpinDismiss_NotFound(t *testing.T) {
	for _, a := range flagActions() {
		t.Run(a.name, func(t *testing.T) {
			svc, _, _ := setupTestService(t)
			_, err := a.call(svc, context.Background(), uuid.New(), uuid.New(), uuid.New())
			assert.ErrorIs(t, err, ErrNotificationNotFound)
		})
	}
}

// failingFlagRepo lets GetByID succeed and fails the write, so the error is
// proven to come back to the caller instead of being swallowed after the lookup.
type failingFlagRepo struct {
	*mockNotifRepo
	err error
}

func (f *failingFlagRepo) SetPinned(context.Context, uuid.UUID, uuid.UUID, bool) error {
	return f.err
}

func (f *failingFlagRepo) SetDismissed(context.Context, uuid.UUID, uuid.UUID, bool) error {
	return f.err
}

func TestPinUnpinDismiss_PropagateWriteError(t *testing.T) {
	boom := errors.New("db down")
	for _, a := range flagActions() {
		t.Run(a.name, func(t *testing.T) {
			svc, repo, _ := setupTestService(t)
			tenantID, userID, id := uuid.New(), uuid.New(), uuid.New()
			repo.notifications = append(repo.notifications, &models.Notification{ID: id, TenantID: tenantID, UserID: userID})
			svc.repo = &failingFlagRepo{mockNotifRepo: repo, err: boom}

			_, err := a.call(svc, context.Background(), tenantID, id, userID)
			assert.ErrorIs(t, err, boom)
		})
	}
}

func TestNormalizePriority(t *testing.T) {
	assert.Equal(t, models.PriorityUrgent, normalizePriority("high"), "client-facing high maps to the DB value")
	assert.Equal(t, models.PriorityNormal, normalizePriority(models.PriorityNormal))
	assert.Equal(t, "", normalizePriority(""))
}

func TestProcessEvent_HighPriorityIsStoredAsUrgent(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	target := uuid.New()

	err := svc.ProcessEvent(context.Background(), models.EventPayload{
		Type: "chat.mention", Priority: "high", ModuleID: "chat",
		ActorID: uuid.NewString(), TargetUserIDs: []string{target.String()},
		Title: "Ping", Timestamp: time.Now(),
	})
	require.NoError(t, err)
	require.Len(t, repo.notifications, 1)
	assert.Equal(t, models.PriorityUrgent, repo.notifications[0].Priority)
}

// deliveryErrRepo fails only the pg_notify push; the notification itself must
// still be stored and the event marked processed.
type deliveryErrRepo struct {
	*mockNotifRepo
	payloads []string
	err      error
}

func (d *deliveryErrRepo) NotifyDelivery(_ context.Context, payload string) error {
	d.payloads = append(d.payloads, payload)
	return d.err
}

func TestProcessEvent_DeliveryPushFailureDoesNotLoseTheNotification(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	wrapped := &deliveryErrRepo{mockNotifRepo: repo, err: errors.New("pg_notify failed")}
	svc.repo = wrapped
	target := uuid.New()

	err := svc.ProcessEvent(context.Background(), models.EventPayload{
		Type: "chat.mention", Priority: models.PriorityNormal, ModuleID: "chat",
		ActorID: uuid.NewString(), TargetUserIDs: []string{target.String()},
		Title: "Ping", Timestamp: time.Now(),
	})
	require.NoError(t, err)
	assert.Len(t, repo.notifications, 1, "notification is stored despite the failed push")
	assert.Len(t, wrapped.payloads, 1, "the push was attempted once")
	assert.Equal(t, 1, repo.markProcessedCount)
}

func TestNotifyDelivery_PayloadCarriesRoutingFields(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	wrapped := &deliveryErrRepo{mockNotifRepo: repo}
	svc.repo = wrapped
	n := &models.Notification{
		ID: uuid.New(), UserID: uuid.New(), Title: "T", EventTypeKey: "chat.mention",
		ModuleID: "chat", Priority: models.PriorityUrgent,
	}

	svc.notifyDelivery(context.Background(), n)

	require.Len(t, wrapped.payloads, 1)
	assert.JSONEq(t, `{"notification_id":"`+n.ID.String()+`","user_id":"`+n.UserID.String()+
		`","title":"T","event_type_key":"chat.mention","module_id":"chat","priority":"urgent"}`, wrapped.payloads[0])
}
