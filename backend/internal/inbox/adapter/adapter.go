package adapter

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ErrForwardNotSupported is returned by HandleForward when a channel has no
// concept of forwarding to an arbitrary recipient (chat, guest, notification).
var ErrForwardNotSupported = errors.New("forward not supported for this channel")

// ChannelAdapter normalizes messages from a specific source (email, chat, notification)
// into the unified InboxMessage format.
type ChannelAdapter interface {
	// Channel returns the channel identifier ("email", "chat", "notification").
	Channel() string

	// FetchNewMessages retrieves messages received since the given time for the user.
	// Returns normalized InboxMessage entries ready for persistence.
	FetchNewMessages(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.InboxMessage, error)

	// HandleReply routes a reply back through the originating channel. sourceID
	// is the message's identifier in the *originating* system (InboxMessage.SourceID),
	// not the inbox message's own id -- e.g. the email message id or chat channel id.
	HandleReply(ctx context.Context, sourceID string, userID uuid.UUID, body string) error

	// HandleForward routes a message to a new recipient through the originating
	// channel. sourceID is the message's identifier in the originating system
	// (see HandleReply). Returns ErrForwardNotSupported if the channel has no
	// concept of forwarding to an arbitrary recipient.
	HandleForward(ctx context.Context, sourceID string, userID uuid.UUID, to string, note string) error

	// MarkReadOnSource syncs read status back to the originating system.
	MarkReadOnSource(ctx context.Context, sourceID string, userID uuid.UUID) error
}

// AdapterRegistry holds all registered channel adapters for lookup by channel name.
type AdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]ChannelAdapter
}

// NewAdapterRegistry creates a new empty adapter registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]ChannelAdapter),
	}
}

// Register adds a channel adapter to the registry.
// If an adapter for the same channel already exists, it is replaced.
func (r *AdapterRegistry) Register(adapter ChannelAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.Channel()] = adapter
}

// Get retrieves a channel adapter by its channel name.
func (r *AdapterRegistry) Get(channel string) (ChannelAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[channel]
	return a, ok
}

// All returns all registered channel adapters.
func (r *AdapterRegistry) All() []ChannelAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ChannelAdapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		result = append(result, a)
	}
	return result
}
