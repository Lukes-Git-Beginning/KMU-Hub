package models

import (
	"regexp"
	"time"

	"github.com/google/uuid"
)

// Tag represents a tag that can be assigned to CRM entities
type Tag struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Color      string     `json:"color"`       // Hex color code (e.g., "#ff5733")
	EntityType EntityType `json:"entity_type"` // contact, company, deal, activity
	CreatedAt  time.Time  `json:"created_at"`
}

// Default tag colors
var DefaultTagColors = []string{
	"#ef4444", // red
	"#f97316", // orange
	"#eab308", // yellow
	"#22c55e", // green
	"#14b8a6", // teal
	"#3b82f6", // blue
	"#8b5cf6", // violet
	"#ec4899", // pink
	"#6b7280", // gray
}

var hexColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// IsValidHexColor checks if a string is a valid hex color code
func IsValidHexColor(color string) bool {
	return hexColorRegex.MatchString(color)
}
