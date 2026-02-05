package models

import (
	"time"

	"github.com/google/uuid"
)

// SavedFilter represents a saved filter configuration
type SavedFilter struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	EntityType EntityType `json:"entity_type"`
	FilterJSON string     `json:"filter_json"`
	IsDefault  bool       `json:"is_default"`
	CreatedBy  uuid.UUID  `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
