package bexio

import "context"

// EventPayload represents an event emitted by the Bexio service.
type EventPayload struct {
	Type     string         `json:"type"`
	TenantID string         `json:"tenant_id"`
	Data     map[string]any `json:"data,omitempty"`
}

// Event types emitted by the Bexio integration.
const (
	EventSyncStarted    = "bexio.sync.started"
	EventSyncCompleted  = "bexio.sync.completed"
	EventSyncFailed     = "bexio.sync.failed"
	EventConnected      = "bexio.connected"
	EventDisconnected   = "bexio.disconnected"
)

// EventEmitter emits Bexio integration events.
type EventEmitter interface {
	Emit(ctx context.Context, payload EventPayload) error
}

// noopEmitter is a no-op implementation for when no emitter is configured.
type noopEmitter struct{}

func (noopEmitter) Emit(_ context.Context, _ EventPayload) error { return nil }
