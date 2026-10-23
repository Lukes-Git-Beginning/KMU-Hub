package adapter

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdapterRegistry_GetUnknownChannel(t *testing.T) {
	r := NewAdapterRegistry()

	a, ok := r.Get("email")
	assert.False(t, ok)
	assert.Nil(t, a)
	assert.Empty(t, r.All())
}

func TestAdapterRegistry_RegisterIndexesByChannel(t *testing.T) {
	r := NewAdapterRegistry()
	email := NewEmailAdapter(nil)
	r.Register(email)
	r.Register(NewChatAdapter(nil))
	r.Register(NewNotificationAdapter(nil))
	r.Register(NewGuestAdapter(nil))

	got, ok := r.Get("email")
	assert.True(t, ok)
	assert.Same(t, email, got)

	channels := make([]string, 0, 4)
	for _, a := range r.All() {
		channels = append(channels, a.Channel())
	}
	sort.Strings(channels)
	assert.Equal(t, []string{"chat", "email", "guest", "notification"}, channels)
}

func TestAdapterRegistry_RegisterReplacesSameChannel(t *testing.T) {
	r := NewAdapterRegistry()
	first := NewEmailAdapter(nil)
	second := NewEmailAdapter(nil)
	r.Register(first)
	r.Register(second)

	got, ok := r.Get("email")
	assert.True(t, ok)
	assert.Same(t, second, got)
	assert.Len(t, r.All(), 1)
}
