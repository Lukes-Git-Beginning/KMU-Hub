package delivery

import (
	"context"
	"log/slog"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/preference"
)

// DeliveryCallback is called when a notification should be delivered.
// This is the integration point for the gateway's WebSocket delivery
// and desktop push delivery.
type DeliveryCallback func(ctx context.Context, notif *models.Notification, decision *preference.DeliveryDecision)

// Dispatcher routes notifications to delivery channels based on
// the preference evaluation decision.
type Dispatcher struct {
	callbacks []DeliveryCallback
}

// NewDispatcher creates a new delivery dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

// OnDeliver registers a callback that will be called for each notification delivery.
func (d *Dispatcher) OnDeliver(cb DeliveryCallback) {
	d.callbacks = append(d.callbacks, cb)
}

// Dispatch sends a notification through all registered delivery channels.
func (d *Dispatcher) Dispatch(ctx context.Context, notif *models.Notification, decision *preference.DeliveryDecision) {
	if !decision.Deliver {
		return
	}

	slog.Debug("dispatching notification",
		"notification_id", notif.ID,
		"user_id", notif.UserID,
		"event_type", notif.EventTypeKey,
		"in_app", decision.InApp,
		"desktop_push", decision.DesktopPush,
	)

	for _, cb := range d.callbacks {
		cb(ctx, notif, decision)
	}
}

// CallbackCount returns the number of registered callbacks (for testing).
func (d *Dispatcher) CallbackCount() int {
	return len(d.callbacks)
}
