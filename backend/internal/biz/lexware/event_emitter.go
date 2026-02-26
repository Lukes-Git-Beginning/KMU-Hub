package lexware

import "context"

type EventPayload struct {
	Type     string         `json:"type"`
	TenantID string         `json:"tenant_id"`
	Data     map[string]any `json:"data,omitempty"`
}

const (
	EventSyncStarted     = "lexware.sync.started"
	EventSyncCompleted   = "lexware.sync.completed"
	EventSyncFailed      = "lexware.sync.failed"
	EventConnected       = "lexware.connected"
	EventDisconnected    = "lexware.disconnected"
	EventWebhookReceived = "lexware.webhook.received"
)

type EventEmitter interface {
	Emit(ctx context.Context, payload EventPayload) error
}

type noopEmitter struct{}

func (noopEmitter) Emit(_ context.Context, _ EventPayload) error { return nil }
