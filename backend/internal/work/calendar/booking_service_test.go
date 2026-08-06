package calendar

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Mock implementations
// ============================================================================

type mockBookingRepository struct {
	pages          map[uuid.UUID]*models.BookingPage
	pagesBySlug    map[string]*models.BookingPage
	bookings       map[uuid.UUID]*models.PublicBooking
	calendarEvents []eventSlot

	createPageErr     error
	getPageErr        error
	createBookErr     error
	dupBookErr        bool // simulate UNIQUE constraint violation
	updateCalEventErr error
}

func newMockBookingRepository() *mockBookingRepository {
	return &mockBookingRepository{
		pages:       make(map[uuid.UUID]*models.BookingPage),
		pagesBySlug: make(map[string]*models.BookingPage),
		bookings:    make(map[uuid.UUID]*models.PublicBooking),
	}
}

func (m *mockBookingRepository) CreateBookingPage(ctx context.Context, page *models.BookingPage) error {
	if m.createPageErr != nil {
		return m.createPageErr
	}
	cp := *page
	m.pages[page.ID] = &cp
	m.pagesBySlug[page.Slug] = &cp
	return nil
}

func (m *mockBookingRepository) GetBookingPageByID(ctx context.Context, id, tenantID uuid.UUID) (*models.BookingPage, error) {
	if m.getPageErr != nil {
		return nil, m.getPageErr
	}
	p, ok := m.pages[id]
	if !ok || p.TenantID != tenantID {
		return nil, ErrBookingPageNotFound
	}
	return p, nil
}

func (m *mockBookingRepository) GetBookingPageBySlug(ctx context.Context, slug string) (*models.BookingPage, error) {
	if m.getPageErr != nil {
		return nil, m.getPageErr
	}
	p, ok := m.pagesBySlug[slug]
	if !ok || !p.Active {
		return nil, ErrBookingPageNotFound
	}
	return p, nil
}

func (m *mockBookingRepository) UpdateBookingPage(ctx context.Context, page *models.BookingPage) error {
	if _, ok := m.pages[page.ID]; !ok {
		return ErrBookingPageNotFound
	}
	cp := *page
	m.pages[page.ID] = &cp
	m.pagesBySlug[page.Slug] = &cp
	return nil
}

func (m *mockBookingRepository) DeleteBookingPage(ctx context.Context, id, tenantID uuid.UUID) error {
	if _, ok := m.pages[id]; !ok {
		return ErrBookingPageNotFound
	}
	delete(m.pagesBySlug, m.pages[id].Slug)
	delete(m.pages, id)
	return nil
}

func (m *mockBookingRepository) ListBookingPages(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]models.BookingPage, error) {
	var result []models.BookingPage
	for _, p := range m.pages {
		if p.TenantID != tenantID {
			continue
		}
		if !includeInactive && !p.Active {
			continue
		}
		result = append(result, *p)
	}
	return result, nil
}

func (m *mockBookingRepository) CreatePublicBooking(ctx context.Context, b *models.PublicBooking) error {
	if m.createBookErr != nil {
		return m.createBookErr
	}
	if m.dupBookErr {
		// Simulate UNIQUE constraint error (Postgres error code 23505)
		return errors.New("duplicate key value violates unique constraint")
	}
	cp := *b
	m.bookings[b.ID] = &cp
	return nil
}

func (m *mockBookingRepository) UpdatePublicBookingCalendarEventID(ctx context.Context, bookingID, tenantID, eventID uuid.UUID) error {
	if m.updateCalEventErr != nil {
		return m.updateCalEventErr
	}
	b, ok := m.bookings[bookingID]
	if !ok || b.TenantID != tenantID {
		return nil
	}
	b.CalendarEventID = &eventID
	return nil
}

func (m *mockBookingRepository) GetBookedSlotsForPage(ctx context.Context, bookingPageID uuid.UUID, from, to time.Time) ([]models.PublicBooking, error) {
	var result []models.PublicBooking
	for _, b := range m.bookings {
		if b.BookingPageID != bookingPageID {
			continue
		}
		if b.Status == models.PublicBookingStatusCancelled {
			continue
		}
		if !b.Date.Before(to) || b.Date.Before(from) {
			continue
		}
		result = append(result, *b)
	}
	return result, nil
}

func (m *mockBookingRepository) GetCalendarEventsInRange(ctx context.Context, calendarID uuid.UUID, from, to time.Time) ([]eventSlot, error) {
	return m.calendarEvents, nil
}

// ============================================================================
// Helper: build a standard booking page
// ============================================================================

func defaultRulesJSON(t *testing.T) string {
	rules := models.AvailabilityRules{
		// Mon–Fri
		Weekdays: []int{1, 2, 3, 4, 5},
		SlotsPerWeekday: []models.AvailabilitySlot{
			{Start: "09:00", End: "17:00"},
		},
		SlotDurationMin: 60,
		BufferMin:       0,
		LeadTimeHours:   0, // no lead time in tests (easier to hit today's slots)
	}
	b, err := json.Marshal(rules)
	require.NoError(t, err)
	return string(b)
}

func seedBookingPage(t *testing.T, svc *BookingService, tenantID uuid.UUID) *models.BookingPage {
	t.Helper()
	page, err := svc.CreateBookingPage(context.Background(), tenantID, CreateBookingPageInput{
		CalendarID:  uuid.New(),
		Slug:        "test-firma",
		CompanyName: "Test Firma GmbH",
		Services: []models.BookingService{
			{ID: "svc-beratung", Name: "Beratung", DurationMin: 60, Price: "120.00"},
		},
		AvailabilityRulesJSON: defaultRulesJSON(t),
	})
	require.NoError(t, err)
	return page
}

// ============================================================================
// Tests: Create Booking Page
// ============================================================================

func TestBookingService_CreateBookingPage_Success(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)
	tenantID := uuid.New()

	page := seedBookingPage(t, svc, tenantID)

	assert.NotEqual(t, uuid.Nil, page.ID)
	assert.Equal(t, "test-firma", page.Slug)
	assert.Equal(t, "Test Firma GmbH", page.CompanyName)
	assert.Equal(t, tenantID, page.TenantID)
	assert.True(t, page.Active)
	assert.Len(t, page.Services, 1)
}

func TestBookingService_CreateBookingPage_SlugRequired(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)

	_, err := svc.CreateBookingPage(context.Background(), uuid.New(), CreateBookingPageInput{
		CalendarID:            uuid.New(),
		Slug:                  "",
		CompanyName:           "Firma",
		AvailabilityRulesJSON: defaultRulesJSON(t),
	})
	assert.ErrorIs(t, err, ErrBookingPageSlugRequired)
}

func TestBookingService_CreateBookingPage_InvalidRules(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)

	_, err := svc.CreateBookingPage(context.Background(), uuid.New(), CreateBookingPageInput{
		CalendarID:            uuid.New(),
		Slug:                  "valid-slug",
		CompanyName:           "Firma",
		AvailabilityRulesJSON: "not-json",
	})
	assert.ErrorIs(t, err, ErrBookingPageInvalidRules)
}

// ============================================================================
// Tests: Availability Calculation
// ============================================================================

// nextWeekdayDate returns a time.Time at local midnight for the next occurrence of
// the given weekday, always at least 1 day in the future.
func nextWeekdayDate(wd time.Weekday) time.Time {
	now := time.Now()
	y, m, d := now.Date()
	loc := now.Location()
	today := time.Date(y, m, d, 0, 0, 0, 0, loc)
	days := (int(wd) - int(today.Weekday()) + 7) % 7
	if days == 0 {
		days = 7
	}
	return today.AddDate(0, 0, days)
}

func TestBookingService_GetAvailability_OpenDay(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	seedBookingPage(t, svc, tenantID)

	monday := nextWeekdayDate(time.Monday)

	days, err := svc.GetAvailability(context.Background(), GetAvailabilityInput{
		Slug:     "test-firma",
		DateFrom: monday,
		DateTo:   monday,
	})
	require.NoError(t, err)
	require.Len(t, days, 1)
	assert.Equal(t, monday.Format("2006-01-02"), days[0].Date.Format("2006-01-02"))
	assert.NotEmpty(t, days[0].Slots)
	// 09:00–17:00 with 60-min slots = 8 slots
	assert.Len(t, days[0].Slots, 8)
}

func TestBookingService_GetAvailability_ClosedDay(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	seedBookingPage(t, svc, tenantID)

	saturday := nextWeekdayDate(time.Saturday)

	days, err := svc.GetAvailability(context.Background(), GetAvailabilityInput{
		Slug:     "test-firma",
		DateFrom: saturday,
		DateTo:   saturday,
	})
	require.NoError(t, err)
	assert.Len(t, days, 0, "Saturday should be closed")
}

func TestBookingService_GetAvailability_SlotBlockedByCalendarEvent(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	page := seedBookingPage(t, svc, tenantID)

	monday := nextWeekdayDate(time.Monday)
	y, mo, dy := monday.Date()
	loc := monday.Location()

	repo.calendarEvents = []eventSlot{
		{
			StartTime: time.Date(y, mo, dy, 9, 0, 0, 0, loc),
			EndTime:   time.Date(y, mo, dy, 10, 0, 0, 0, loc),
		},
	}
	_ = page

	days, err := svc.GetAvailability(context.Background(), GetAvailabilityInput{
		Slug:     "test-firma",
		DateFrom: monday,
		DateTo:   monday,
	})
	require.NoError(t, err)
	require.Len(t, days, 1)
	// 09:00 should be blocked, 10:00–17:00 = 7 slots
	for _, slot := range days[0].Slots {
		assert.NotEqual(t, "09:00", slot, "09:00 slot must be blocked by calendar event")
	}
	assert.Len(t, days[0].Slots, 7)
}

func TestBookingService_GetAvailability_SlotBlockedByPublicBooking(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	page := seedBookingPage(t, svc, tenantID)

	monday := nextWeekdayDate(time.Monday)

	// Pre-seed a booking for 10:00 on that Monday
	repo.bookings[uuid.New()] = &models.PublicBooking{
		ID:            uuid.New(),
		TenantID:      tenantID,
		BookingPageID: page.ID,
		ServiceID:     "svc-beratung",
		CustomerName:  "Max Mustermann",
		CustomerEmail: "max@example.com",
		Date:          monday,
		TimeSlot:      "10:00",
		Status:        models.PublicBookingStatusConfirmed,
	}

	days, err := svc.GetAvailability(context.Background(), GetAvailabilityInput{
		Slug:     "test-firma",
		DateFrom: monday,
		DateTo:   monday,
	})
	require.NoError(t, err)
	require.Len(t, days, 1)
	for _, slot := range days[0].Slots {
		assert.NotEqual(t, "10:00", slot, "10:00 slot must be blocked by existing booking")
	}
}

// ============================================================================
// Tests: Create Public Booking
// ============================================================================

func TestBookingService_CreatePublicBooking_Success(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	seedBookingPage(t, svc, tenantID)

	monday := nextWeekdayDate(time.Monday)

	result, err := svc.CreatePublicBooking(context.Background(), nil, nil, CreatePublicBookingInput{
		Slug:          "test-firma",
		ServiceID:     "svc-beratung",
		Date:          monday,
		TimeSlot:      "09:00",
		CustomerName:  "Erika Musterfrau",
		CustomerEmail: "erika@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, models.PublicBookingStatusConfirmed, result.Booking.Status)
	assert.Equal(t, "erika@example.com", result.Booking.CustomerEmail)
}

func TestBookingService_CreatePublicBooking_UnknownService(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	seedBookingPage(t, svc, tenantID)

	monday := nextWeekdayDate(time.Monday)

	_, err := svc.CreatePublicBooking(context.Background(), nil, nil, CreatePublicBookingInput{
		Slug:          "test-firma",
		ServiceID:     "nonexistent-service",
		Date:          monday,
		TimeSlot:      "09:00",
		CustomerName:  "Test",
		CustomerEmail: "test@example.com",
	})
	assert.ErrorIs(t, err, ErrBookingServiceNotFound)
}

func TestBookingService_CreatePublicBooking_SlotInPast(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	seedBookingPage(t, svc, tenantID)

	// Use a date from last year
	pastDate := time.Now().AddDate(-1, 0, 0).Truncate(24 * time.Hour)

	_, err := svc.CreatePublicBooking(context.Background(), nil, nil, CreatePublicBookingInput{
		Slug:          "test-firma",
		ServiceID:     "svc-beratung",
		Date:          pastDate,
		TimeSlot:      "09:00",
		CustomerName:  "Test",
		CustomerEmail: "test@example.com",
	})
	assert.ErrorIs(t, err, ErrBookingSlotInPast)
}

func TestBookingService_CreatePublicBooking_SlotBlockedByCalendarEvent(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	seedBookingPage(t, svc, tenantID)

	monday := nextWeekdayDate(time.Monday)
	y, mo, dy := monday.Date()
	loc := monday.Location()

	// Block 09:00–10:00 with a calendar event
	repo.calendarEvents = []eventSlot{
		{
			StartTime: time.Date(y, mo, dy, 9, 0, 0, 0, loc),
			EndTime:   time.Date(y, mo, dy, 10, 0, 0, 0, loc),
		},
	}

	_, err := svc.CreatePublicBooking(context.Background(), nil, nil, CreatePublicBookingInput{
		Slug:          "test-firma",
		ServiceID:     "svc-beratung",
		Date:          monday,
		TimeSlot:      "09:00",
		CustomerName:  "Test",
		CustomerEmail: "test@example.com",
	})
	assert.ErrorIs(t, err, ErrBookingSlotUnavailable)
}

func TestBookingService_CreatePublicBooking_TenantIsolation(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create page for tenant A
	pageA, err := svc.CreateBookingPage(context.Background(), tenantA, CreateBookingPageInput{
		CalendarID:            uuid.New(),
		Slug:                  "firma-a",
		CompanyName:           "Firma A GmbH",
		Services:              []models.BookingService{{ID: "svc-a", Name: "Service A", DurationMin: 30, Price: "0.00"}},
		AvailabilityRulesJSON: defaultRulesJSON(t),
	})
	require.NoError(t, err)

	// Tenant B must not be able to access tenant A's page by ID
	_, err = svc.GetBookingPage(context.Background(), pageA.ID, tenantB)
	assert.ErrorIs(t, err, ErrBookingPageNotFound)
}

func TestBookingService_CreatePublicBooking_Idempotency_UniqueConstraint(t *testing.T) {
	repo := newMockBookingRepository()
	repo.dupBookErr = true
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	seedBookingPage(t, svc, tenantID)

	monday := nextWeekdayDate(time.Monday)

	_, err := svc.CreatePublicBooking(context.Background(), nil, nil, CreatePublicBookingInput{
		Slug:          "test-firma",
		ServiceID:     "svc-beratung",
		Date:          monday,
		TimeSlot:      "09:00",
		CustomerName:  "Test",
		CustomerEmail: "test@example.com",
	})
	// The underlying error is propagated (DB unique constraint); caller can handle as 409.
	assert.Error(t, err)
}

// mockCalClient creates a calendar event with a fixed ID, simulating a real
// calendar integration for the write-back test below.
type mockCalClient struct {
	eventID string
	err     error
}

func (m *mockCalClient) CreateEvent(ctx context.Context, input PublicBookingCalEventInput) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.eventID, nil
}

func TestBookingService_CreatePublicBooking_PersistsCalendarEventID(t *testing.T) {
	repo := newMockBookingRepository()
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	seedBookingPage(t, svc, tenantID)

	monday := nextWeekdayDate(time.Monday)
	eventID := uuid.New()
	calClient := &mockCalClient{eventID: eventID.String()}

	result, err := svc.CreatePublicBooking(context.Background(), nil, calClient, CreatePublicBookingInput{
		Slug:          "test-firma",
		ServiceID:     "svc-beratung",
		Date:          monday,
		TimeSlot:      "09:00",
		CustomerName:  "Erika Musterfrau",
		CustomerEmail: "erika@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, result.Booking.CalendarEventID)
	assert.Equal(t, eventID, *result.Booking.CalendarEventID)

	// The write-back must have reached the repository, not just the in-memory
	// struct returned to the caller — that's the bug this test guards against.
	stored, ok := repo.bookings[result.Booking.ID]
	require.True(t, ok)
	require.NotNil(t, stored.CalendarEventID)
	assert.Equal(t, eventID, *stored.CalendarEventID)
}

func TestBookingService_CreatePublicBooking_CalendarEventWriteBackFailureDoesNotFailBooking(t *testing.T) {
	repo := newMockBookingRepository()
	repo.updateCalEventErr = errors.New("connection reset")
	svc := NewBookingService(repo)
	tenantID := uuid.New()
	seedBookingPage(t, svc, tenantID)

	monday := nextWeekdayDate(time.Monday)
	calClient := &mockCalClient{eventID: uuid.New().String()}

	result, err := svc.CreatePublicBooking(context.Background(), nil, calClient, CreatePublicBookingInput{
		Slug:          "test-firma",
		ServiceID:     "svc-beratung",
		Date:          monday,
		TimeSlot:      "09:00",
		CustomerName:  "Test",
		CustomerEmail: "test@example.com",
	})
	require.NoError(t, err, "a failed calendar event write-back must not fail the booking")
	require.NotNil(t, result)
}
