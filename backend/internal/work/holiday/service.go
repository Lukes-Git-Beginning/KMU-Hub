package holiday

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kmuhub/kmuhub/internal/models"
)

// HolidayFetcher abstracts the Nager.Date API client for testability
type HolidayFetcher interface {
	FetchHolidays(ctx context.Context, year int, countryCode string) ([]NagerHoliday, error)
}

// Service handles holiday business logic
type Service struct {
	repo        Repository
	nagerClient HolidayFetcher
}

// NewService creates a new holiday service.
// nagerClient can be nil for environments without internet access.
func NewService(repo Repository, nagerClient HolidayFetcher) *Service {
	return &Service{
		repo:        repo,
		nagerClient: nagerClient,
	}
}

// DACHCountries returns the supported DACH country codes
var DACHCountries = []string{"DE", "AT", "CH"}

// SeedHolidays fetches holidays from Nager.Date API and stores them
func (s *Service) SeedHolidays(ctx context.Context, year int, countryCode string) error {
	if err := validateInput(year, countryCode); err != nil {
		return err
	}

	if s.nagerClient == nil {
		return ErrNagerAPIUnavailable
	}

	nagerHolidays, err := s.nagerClient.FetchHolidays(ctx, year, countryCode)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSeedFailed, err)
	}

	holidays, err := MapToModels(nagerHolidays)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSeedFailed, err)
	}

	if upsertErr := s.repo.BulkUpsert(ctx, holidays); upsertErr != nil {
		return fmt.Errorf("%w: %v", ErrSeedFailed, upsertErr)
	}

	slog.Info("holidays seeded",
		"year", year,
		"country", countryCode,
		"count", len(holidays),
	)

	return nil
}

// SeedAllDACH seeds holidays for DE, AT, and CH for a given year
func (s *Service) SeedAllDACH(ctx context.Context, year int) error {
	if year < 2000 || year > 2100 {
		return ErrInvalidYear
	}

	for _, cc := range DACHCountries {
		if err := s.SeedHolidays(ctx, year, cc); err != nil {
			return fmt.Errorf("seed %s: %w", cc, err)
		}
	}

	return nil
}

// EnsureHolidaysLoaded checks if data exists for a year/country, seeds if missing
func (s *Service) EnsureHolidaysLoaded(ctx context.Context, year int, countryCode string) error {
	if err := validateInput(year, countryCode); err != nil {
		return err
	}

	exists, err := s.repo.HasDataForYear(ctx, year, countryCode)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return s.SeedHolidays(ctx, year, countryCode)
}

// ListHolidays returns holidays for a date range, automatically loading missing data.
// Handles year boundaries (e.g., Dec 2025 - Jan 2026 spans two years).
func (s *Service) ListHolidays(ctx context.Context, start, end time.Time, countryCode string, subdivisionCode *string) ([]models.PublicHoliday, error) {
	if !validCountryCodes[countryCode] {
		return nil, ErrInvalidCountryCode
	}

	// Ensure data is loaded for all years in the range
	startYear := start.Year()
	endYear := end.Year()

	for year := startYear; year <= endYear; year++ {
		if err := s.EnsureHolidaysLoaded(ctx, year, countryCode); err != nil {
			// Log warning but continue -- data may still be available from previous seeds
			slog.Warn("failed to ensure holidays loaded",
				"year", year,
				"country", countryCode,
				"error", err,
			)
		}
	}

	return s.repo.ListByDateRange(ctx, start, end, countryCode, subdivisionCode)
}

func validateInput(year int, countryCode string) error {
	if year < 2000 || year > 2100 {
		return ErrInvalidYear
	}
	if !validCountryCodes[countryCode] {
		return ErrInvalidCountryCode
	}
	return nil
}
