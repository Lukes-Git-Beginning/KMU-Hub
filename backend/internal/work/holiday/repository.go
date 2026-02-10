package holiday

import (
	"context"
	"time"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for holiday persistence
type Repository interface {
	// BulkUpsert inserts or updates holidays using ON CONFLICT
	BulkUpsert(ctx context.Context, holidays []models.PublicHoliday) error

	// ListByDateRange returns holidays within a date range for a country
	// If subdivisionCode is provided, returns global holidays + subdivision-specific ones
	ListByDateRange(ctx context.Context, start, end time.Time, countryCode string, subdivisionCode *string) ([]models.PublicHoliday, error)

	// HasDataForYear checks if holiday data exists for a year/country combination
	HasDataForYear(ctx context.Context, year int, countryCode string) (bool, error)

	// DeleteByYear removes all holidays for a year/country combination
	DeleteByYear(ctx context.Context, year int, countryCode string) error
}
