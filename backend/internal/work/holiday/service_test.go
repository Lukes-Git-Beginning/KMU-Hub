package holiday

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock Nager Client ---

type mockNagerClient struct {
	holidays map[string][]NagerHoliday // key: "year-country"
	failNext bool
}

func newMockNagerClient() *mockNagerClient {
	return &mockNagerClient{
		holidays: make(map[string][]NagerHoliday),
	}
}

func (m *mockNagerClient) FetchHolidays(_ context.Context, year int, countryCode string) ([]NagerHoliday, error) {
	if m.failNext {
		m.failNext = false
		return nil, fmt.Errorf("API unavailable")
	}
	key := fmt.Sprintf("%d-%s", year, countryCode)
	h, ok := m.holidays[key]
	if !ok {
		return []NagerHoliday{}, nil
	}
	return h, nil
}

// --- Mock Repository ---

type mockRepo struct {
	holidays      map[string][]models.PublicHoliday // key: "year-country"
	upsertCalls   int
	deletedYears  []string
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		holidays: make(map[string][]models.PublicHoliday),
	}
}

func (m *mockRepo) BulkUpsert(_ context.Context, holidays []models.PublicHoliday) error {
	m.upsertCalls++
	for _, h := range holidays {
		key := fmt.Sprintf("%d-%s", h.Year, h.CountryCode)
		m.holidays[key] = append(m.holidays[key], h)
	}
	return nil
}

func (m *mockRepo) ListByDateRange(_ context.Context, start, end time.Time, countryCode string, subdivisionCode *string) ([]models.PublicHoliday, error) {
	var result []models.PublicHoliday
	for _, holidays := range m.holidays {
		for _, h := range holidays {
			if h.CountryCode != countryCode {
				continue
			}
			if h.Date.Before(start) || h.Date.After(end) {
				continue
			}
			if subdivisionCode != nil {
				if !h.IsGlobal {
					found := false
					for _, sc := range h.SubdivisionCodes {
						if sc == *subdivisionCode {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}
			}
			result = append(result, h)
		}
	}
	return result, nil
}

func (m *mockRepo) HasDataForYear(_ context.Context, year int, countryCode string) (bool, error) {
	key := fmt.Sprintf("%d-%s", year, countryCode)
	_, ok := m.holidays[key]
	return ok, nil
}

func (m *mockRepo) DeleteByYear(_ context.Context, year int, countryCode string) error {
	key := fmt.Sprintf("%d-%s", year, countryCode)
	delete(m.holidays, key)
	m.deletedYears = append(m.deletedYears, key)
	return nil
}

// --- Helpers ---

func sampleDEHolidays() []NagerHoliday {
	return []NagerHoliday{
		{
			Date:        "2026-01-01",
			LocalName:   "Neujahr",
			Name:        "New Year's Day",
			CountryCode: "DE",
			Fixed:       true,
			Global:      true,
			Counties:    nil,
			Types:       []string{"Public"},
		},
		{
			Date:        "2026-10-03",
			LocalName:   "Tag der Deutschen Einheit",
			Name:        "German Unity Day",
			CountryCode: "DE",
			Fixed:       true,
			Global:      true,
			Counties:    nil,
			Types:       []string{"Public"},
		},
		{
			Date:        "2026-10-31",
			LocalName:   "Reformationstag",
			Name:        "Reformation Day",
			CountryCode: "DE",
			Fixed:       true,
			Global:      false,
			Counties:    []string{"DE-BB", "DE-HB", "DE-HH", "DE-MV", "DE-NI", "DE-SN", "DE-ST", "DE-SH", "DE-TH"},
			Types:       []string{"Public"},
		},
	}
}

func ptrStr(s string) *string { return &s }

// --- Tests ---

func TestSeedHolidays(t *testing.T) {
	repo := newMockRepo()
	client := newMockNagerClient()
	client.holidays["2026-DE"] = sampleDEHolidays()
	svc := NewService(repo, client)

	err := svc.SeedHolidays(context.Background(), 2026, "DE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	holidays := repo.holidays["2026-DE"]
	if len(holidays) != 3 {
		t.Fatalf("holiday count = %d, want 3", len(holidays))
	}

	// Verify mapping
	neujahr := holidays[0]
	if neujahr.Name != "New Year's Day" {
		t.Errorf("name = %q, want %q", neujahr.Name, "New Year's Day")
	}
	if neujahr.LocalName != "Neujahr" {
		t.Errorf("local_name = %q, want %q", neujahr.LocalName, "Neujahr")
	}
	if !neujahr.IsGlobal {
		t.Error("Neujahr should be global")
	}
	if neujahr.HolidayType != "public" {
		t.Errorf("holiday_type = %q, want %q", neujahr.HolidayType, "public")
	}
	if neujahr.Year != 2026 {
		t.Errorf("year = %d, want 2026", neujahr.Year)
	}

	// Verify Reformationstag has subdivision codes
	reform := holidays[2]
	if reform.IsGlobal {
		t.Error("Reformationstag should not be global")
	}
	if len(reform.SubdivisionCodes) != 9 {
		t.Errorf("subdivision codes = %d, want 9", len(reform.SubdivisionCodes))
	}

	if repo.upsertCalls != 1 {
		t.Errorf("upsert calls = %d, want 1", repo.upsertCalls)
	}
}

func TestSeedHolidays_InvalidCountry(t *testing.T) {
	svc := NewService(newMockRepo(), newMockNagerClient())

	err := svc.SeedHolidays(context.Background(), 2026, "US")
	if err != ErrInvalidCountryCode {
		t.Errorf("err = %v, want ErrInvalidCountryCode", err)
	}
}

func TestSeedHolidays_InvalidYear(t *testing.T) {
	svc := NewService(newMockRepo(), newMockNagerClient())

	err := svc.SeedHolidays(context.Background(), 1999, "DE")
	if err != ErrInvalidYear {
		t.Errorf("err = %v, want ErrInvalidYear", err)
	}
}

func TestSeedHolidays_NilClient(t *testing.T) {
	svc := NewService(newMockRepo(), nil)

	err := svc.SeedHolidays(context.Background(), 2026, "DE")
	if err != ErrNagerAPIUnavailable {
		t.Errorf("err = %v, want ErrNagerAPIUnavailable", err)
	}
}

func TestSeedHolidays_APIError(t *testing.T) {
	client := newMockNagerClient()
	client.failNext = true
	svc := NewService(newMockRepo(), client)

	err := svc.SeedHolidays(context.Background(), 2026, "DE")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSeedAllDACH(t *testing.T) {
	repo := newMockRepo()
	client := newMockNagerClient()
	client.holidays["2026-DE"] = sampleDEHolidays()
	client.holidays["2026-AT"] = []NagerHoliday{
		{Date: "2026-01-01", LocalName: "Neujahr", Name: "New Year's Day", CountryCode: "AT", Global: true, Types: []string{"Public"}},
	}
	client.holidays["2026-CH"] = []NagerHoliday{
		{Date: "2026-01-01", LocalName: "Neujahr", Name: "New Year's Day", CountryCode: "CH", Global: true, Types: []string{"Public"}},
	}

	svc := NewService(repo, client)

	err := svc.SeedAllDACH(context.Background(), 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all 3 countries seeded
	if _, ok := repo.holidays["2026-DE"]; !ok {
		t.Error("DE holidays not seeded")
	}
	if _, ok := repo.holidays["2026-AT"]; !ok {
		t.Error("AT holidays not seeded")
	}
	if _, ok := repo.holidays["2026-CH"]; !ok {
		t.Error("CH holidays not seeded")
	}
	if repo.upsertCalls != 3 {
		t.Errorf("upsert calls = %d, want 3", repo.upsertCalls)
	}
}

func TestSeedAllDACH_InvalidYear(t *testing.T) {
	svc := NewService(newMockRepo(), newMockNagerClient())

	err := svc.SeedAllDACH(context.Background(), 1800)
	if err != ErrInvalidYear {
		t.Errorf("err = %v, want ErrInvalidYear", err)
	}
}

func TestEnsureHolidaysLoaded_FirstCall_Seeds(t *testing.T) {
	repo := newMockRepo()
	client := newMockNagerClient()
	client.holidays["2026-DE"] = sampleDEHolidays()
	svc := NewService(repo, client)

	err := svc.EnsureHolidaysLoaded(context.Background(), 2026, "DE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.upsertCalls != 1 {
		t.Errorf("upsert calls = %d, want 1 (should seed on first call)", repo.upsertCalls)
	}
}

func TestEnsureHolidaysLoaded_SecondCall_UsesCache(t *testing.T) {
	repo := newMockRepo()
	client := newMockNagerClient()
	client.holidays["2026-DE"] = sampleDEHolidays()
	svc := NewService(repo, client)

	// First call seeds
	_ = svc.EnsureHolidaysLoaded(context.Background(), 2026, "DE")

	// Second call should NOT fetch again
	err := svc.EnsureHolidaysLoaded(context.Background(), 2026, "DE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.upsertCalls != 1 {
		t.Errorf("upsert calls = %d, want 1 (should use cache on second call)", repo.upsertCalls)
	}
}

func TestListHolidays_Basic(t *testing.T) {
	repo := newMockRepo()
	client := newMockNagerClient()
	client.holidays["2026-DE"] = sampleDEHolidays()
	svc := NewService(repo, client)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	holidays, err := svc.ListHolidays(context.Background(), start, end, "DE", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 3 {
		t.Errorf("holiday count = %d, want 3", len(holidays))
	}
}

func TestListHolidays_SubdivisionFilter(t *testing.T) {
	repo := newMockRepo()
	client := newMockNagerClient()
	client.holidays["2026-DE"] = sampleDEHolidays()
	svc := NewService(repo, client)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	// Filter by Bayern (BY) -- Reformationstag is NOT in BY
	subdivision := "DE-BY"
	holidays, err := svc.ListHolidays(context.Background(), start, end, "DE", &subdivision)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should get 2 global holidays, but NOT Reformationstag (BY not in list)
	if len(holidays) != 2 {
		t.Errorf("holiday count = %d, want 2 (global only, Reformationstag excluded for BY)", len(holidays))
	}
}

func TestListHolidays_SubdivisionIncluded(t *testing.T) {
	repo := newMockRepo()
	client := newMockNagerClient()
	client.holidays["2026-DE"] = sampleDEHolidays()
	svc := NewService(repo, client)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	// Filter by Sachsen (SN) -- Reformationstag IS in SN
	subdivision := "DE-SN"
	holidays, err := svc.ListHolidays(context.Background(), start, end, "DE", &subdivision)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should get all 3 (2 global + Reformationstag for SN)
	if len(holidays) != 3 {
		t.Errorf("holiday count = %d, want 3 (2 global + Reformationstag for SN)", len(holidays))
	}
}

func TestListHolidays_YearBoundary(t *testing.T) {
	repo := newMockRepo()
	client := newMockNagerClient()
	client.holidays["2025-DE"] = []NagerHoliday{
		{Date: "2025-12-25", LocalName: "Weihnachten", Name: "Christmas Day", CountryCode: "DE", Global: true, Types: []string{"Public"}},
	}
	client.holidays["2026-DE"] = []NagerHoliday{
		{Date: "2026-01-01", LocalName: "Neujahr", Name: "New Year's Day", CountryCode: "DE", Global: true, Types: []string{"Public"}},
	}
	svc := NewService(repo, client)

	start := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	holidays, err := svc.ListHolidays(context.Background(), start, end, "DE", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 2 {
		t.Errorf("holiday count = %d, want 2 (Dec 2025 + Jan 2026)", len(holidays))
	}

	// Both years should have been seeded
	if repo.upsertCalls != 2 {
		t.Errorf("upsert calls = %d, want 2 (one per year)", repo.upsertCalls)
	}
}

func TestListHolidays_InvalidCountry(t *testing.T) {
	svc := NewService(newMockRepo(), newMockNagerClient())

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	_, err := svc.ListHolidays(context.Background(), start, end, "FR", nil)
	if err != ErrInvalidCountryCode {
		t.Errorf("err = %v, want ErrInvalidCountryCode", err)
	}
}

func TestListHolidays_APIUnavailable_StillQueries(t *testing.T) {
	repo := newMockRepo()
	// Pre-populate some data
	repo.holidays["2026-DE"] = []models.PublicHoliday{
		{
			Name:        "New Year's Day",
			LocalName:   "Neujahr",
			CountryCode: "DE",
			Date:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			IsGlobal:    true,
			Year:        2026,
		},
	}

	// nil client = no API
	svc := NewService(repo, nil)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	// Should still return pre-populated data even if API is unavailable
	holidays, err := svc.ListHolidays(context.Background(), start, end, "DE", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 1 {
		t.Errorf("holiday count = %d, want 1", len(holidays))
	}
}

func TestMapToModels(t *testing.T) {
	nagerHolidays := []NagerHoliday{
		{
			Date:        "2026-05-01",
			LocalName:   "Tag der Arbeit",
			Name:        "Labour Day",
			CountryCode: "DE",
			Global:      true,
			Types:       []string{"Public"},
		},
		{
			Date:        "2026-11-01",
			LocalName:   "Allerheiligen",
			Name:        "All Saints' Day",
			CountryCode: "DE",
			Global:      false,
			Counties:    []string{"DE-BW", "DE-BY", "DE-NW", "DE-RP", "DE-SL"},
			Types:       []string{"Public"},
		},
	}

	models, err := MapToModels(nagerHolidays)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("model count = %d, want 2", len(models))
	}

	// Verify Labour Day
	labour := models[0]
	if labour.Name != "Labour Day" {
		t.Errorf("name = %q, want %q", labour.Name, "Labour Day")
	}
	if labour.Date.Month() != time.May || labour.Date.Day() != 1 {
		t.Errorf("date = %v, want May 1", labour.Date)
	}
	if !labour.IsGlobal {
		t.Error("Labour Day should be global")
	}
	if labour.HolidayType != "public" {
		t.Errorf("type = %q, want %q", labour.HolidayType, "public")
	}

	// Verify All Saints
	saints := models[1]
	if saints.IsGlobal {
		t.Error("All Saints should not be global")
	}
	if len(saints.SubdivisionCodes) != 5 {
		t.Errorf("subdivision codes = %d, want 5", len(saints.SubdivisionCodes))
	}
}

func TestMapToModels_InvalidDate(t *testing.T) {
	nagerHolidays := []NagerHoliday{
		{Date: "not-a-date", Name: "Bad", CountryCode: "DE"},
	}

	_, err := MapToModels(nagerHolidays)
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestMapToModels_NoTypes(t *testing.T) {
	nagerHolidays := []NagerHoliday{
		{Date: "2026-01-01", Name: "Test", CountryCode: "DE", Types: nil},
	}

	models, err := MapToModels(nagerHolidays)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if models[0].HolidayType != "public" {
		t.Errorf("type = %q, want %q (default)", models[0].HolidayType, "public")
	}
}
