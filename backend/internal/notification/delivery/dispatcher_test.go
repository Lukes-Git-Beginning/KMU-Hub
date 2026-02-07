package delivery

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/preference"
)

func TestDispatcherCallsCallbacks(t *testing.T) {
	d := NewDispatcher()

	var called atomic.Int32
	d.OnDeliver(func(_ context.Context, _ *models.Notification, _ *preference.DeliveryDecision) {
		called.Add(1)
	})
	d.OnDeliver(func(_ context.Context, _ *models.Notification, _ *preference.DeliveryDecision) {
		called.Add(1)
	})

	notif := &models.Notification{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		EventTypeKey: "chat.mention",
		Title:        "Test",
	}
	decision := &preference.DeliveryDecision{
		Deliver:     true,
		InApp:       true,
		DesktopPush: true,
		Sound:       "default",
	}

	d.Dispatch(context.Background(), notif, decision)
	assert.Equal(t, int32(2), called.Load())
}

func TestDispatcherSkipsWhenNotDeliver(t *testing.T) {
	d := NewDispatcher()

	var called atomic.Bool
	d.OnDeliver(func(_ context.Context, _ *models.Notification, _ *preference.DeliveryDecision) {
		called.Store(true)
	})

	notif := &models.Notification{ID: uuid.New()}
	decision := &preference.DeliveryDecision{Deliver: false}

	d.Dispatch(context.Background(), notif, decision)
	assert.False(t, called.Load())
}

func TestDispatcherCallbackCount(t *testing.T) {
	d := NewDispatcher()
	assert.Equal(t, 0, d.CallbackCount())

	d.OnDeliver(func(_ context.Context, _ *models.Notification, _ *preference.DeliveryDecision) {})
	assert.Equal(t, 1, d.CallbackCount())

	d.OnDeliver(func(_ context.Context, _ *models.Notification, _ *preference.DeliveryDecision) {})
	assert.Equal(t, 2, d.CallbackCount())
}

func TestDispatcherReceivesCorrectData(t *testing.T) {
	d := NewDispatcher()

	var receivedNotif *models.Notification
	var receivedDecision *preference.DeliveryDecision

	d.OnDeliver(func(_ context.Context, n *models.Notification, dec *preference.DeliveryDecision) {
		receivedNotif = n
		receivedDecision = dec
	})

	notif := &models.Notification{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		EventTypeKey: "crm.deal.assigned",
		Title:        "Deal assigned",
	}
	decision := &preference.DeliveryDecision{
		Deliver:     true,
		InApp:       true,
		DesktopPush: false,
		Sound:       "subtle",
		Reason:      "user preference",
	}

	d.Dispatch(context.Background(), notif, decision)

	assert.Equal(t, notif.ID, receivedNotif.ID)
	assert.Equal(t, "crm.deal.assigned", receivedNotif.EventTypeKey)
	assert.False(t, receivedDecision.DesktopPush)
	assert.Equal(t, "subtle", receivedDecision.Sound)
}

func TestDispatcherNoCallbacks(t *testing.T) {
	d := NewDispatcher()

	notif := &models.Notification{ID: uuid.New()}
	decision := &preference.DeliveryDecision{Deliver: true}

	// Should not panic
	d.Dispatch(context.Background(), notif, decision)
}
