package models

import (
	"time"

	"github.com/google/uuid"
)

// EventCategory represents a user-defined color category for calendar events
type EventCategory struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}
