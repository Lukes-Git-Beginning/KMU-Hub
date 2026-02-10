package models

import (
	"time"

	"github.com/google/uuid"
)

// PublicHoliday represents a public holiday for a DACH country
type PublicHoliday struct {
	ID               uuid.UUID `json:"id"`
	Date             time.Time `json:"date"`
	Name             string    `json:"name"`
	LocalName        string    `json:"local_name"`
	CountryCode      string    `json:"country_code"`
	IsGlobal         bool      `json:"is_global"`
	SubdivisionCodes []string  `json:"subdivision_codes,omitempty"`
	HolidayType      string    `json:"holiday_type"`
	Year             int       `json:"year"`
	CreatedAt        time.Time `json:"created_at"`
}
