package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// BookingService handles booking-page and public-booking business logic.
type BookingService struct {
	repo BookingRepository
}

// NewBookingService creates a new BookingService.
func NewBookingService(repo BookingRepository) *BookingService {
	return &BookingService{repo: repo}
}

// ============================================================================
// Admin: Booking Page CRUD
// ============================================================================

// CreateBookingPageInput holds the required fields for creating a booking page.
type CreateBookingPageInput struct {
	CalendarID          uuid.UUID
	Slug                string
	CompanyName         string
	LogoURL             *string
	Services            []models.BookingService
	AvailabilityRulesJSON string // raw JSON for the rules blob
}

// CreateBookingPage validates and persists a new booking page.
func (s *BookingService) CreateBookingPage(ctx context.Context, tenantID uuid.UUID, input CreateBookingPageInput) (*models.BookingPage, error) {
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if slug == "" {
		return nil, ErrBookingPageSlugRequired
	}

	companyName := strings.TrimSpace(input.CompanyName)
	if companyName == "" {
		return nil, ErrBookingPageCompanyNameRequired
	}

	var rules models.AvailabilityRules
	if err := json.Unmarshal([]byte(input.AvailabilityRulesJSON), &rules); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBookingPageInvalidRules, err.Error())
	}

	now := time.Now()
	page := &models.BookingPage{
		ID:                uuid.New(),
		TenantID:          tenantID,
		CalendarID:        input.CalendarID,
		Slug:              slug,
		CompanyName:       companyName,
		LogoURL:           input.LogoURL,
		Services:          input.Services,
		AvailabilityRules: rules,
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreateBookingPage(ctx, page); err != nil {
		return nil, err
	}

	slog.Info("booking page created", "page_id", page.ID, "slug", page.Slug, "tenant_id", tenantID)
	return page, nil
}

// GetBookingPage retrieves a booking page by ID (tenant-scoped).
func (s *BookingService) GetBookingPage(ctx context.Context, id, tenantID uuid.UUID) (*models.BookingPage, error) {
	return s.repo.GetBookingPageByID(ctx, id, tenantID)
}

// ListBookingPages lists all booking pages for a tenant.
func (s *BookingService) ListBookingPages(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]models.BookingPage, error) {
	return s.repo.ListBookingPages(ctx, tenantID, includeInactive)
}

// UpdateBookingPageInput holds updateable fields.
type UpdateBookingPageInput struct {
	CompanyName           *string
	LogoURL               *string
	Services              []models.BookingService // nil = no change
	AvailabilityRulesJSON *string
	Active                *bool
}

// UpdateBookingPage applies partial updates to a booking page.
func (s *BookingService) UpdateBookingPage(ctx context.Context, id, tenantID uuid.UUID, input UpdateBookingPageInput) (*models.BookingPage, error) {
	page, err := s.repo.GetBookingPageByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if input.CompanyName != nil {
		cn := strings.TrimSpace(*input.CompanyName)
		if cn == "" {
			return nil, ErrBookingPageCompanyNameRequired
		}
		page.CompanyName = cn
	}

	if input.LogoURL != nil {
		page.LogoURL = input.LogoURL
	}

	if input.Services != nil {
		page.Services = input.Services
	}

	if input.AvailabilityRulesJSON != nil {
		var rules models.AvailabilityRules
		if err := json.Unmarshal([]byte(*input.AvailabilityRulesJSON), &rules); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrBookingPageInvalidRules, err.Error())
		}
		page.AvailabilityRules = rules
	}

	if input.Active != nil {
		page.Active = *input.Active
	}

	page.UpdatedAt = time.Now()

	if err := s.repo.UpdateBookingPage(ctx, page); err != nil {
		return nil, err
	}

	slog.Info("booking page updated", "page_id", page.ID, "tenant_id", tenantID)
	return page, nil
}

// DeleteBookingPage removes a booking page (tenant-scoped).
func (s *BookingService) DeleteBookingPage(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.DeleteBookingPage(ctx, id, tenantID); err != nil {
		return err
	}
	slog.Info("booking page deleted", "page_id", id, "tenant_id", tenantID)
	return nil
}

// ============================================================================
// Public: Booking Page + Availability
// ============================================================================

// GetPublicBookingPage retrieves a booking page by slug (public, no auth).
// Returns only the public fields. Returns ErrBookingPageNotFound if inactive.
func (s *BookingService) GetPublicBookingPage(ctx context.Context, slug string) (*models.BookingPage, error) {
	return s.repo.GetBookingPageBySlug(ctx, slug)
}

// GetAvailabilityInput holds query parameters for availability calculation.
type GetAvailabilityInput struct {
	Slug      string
	DateFrom  time.Time
	DateTo    time.Time
	ServiceID string // optional; overrides slot duration from service definition
}

// DayAvailability holds free slots for a single date.
type DayAvailability struct {
	Date  time.Time
	Slots []string // "HH:MM" start times
}

// GetAvailability returns the list of free slot start times per day.
// Algorithm:
//  1. Load the booking page (active only).
//  2. For each day in [from, to]:
//     a. Check if weekday is in availability_rules.weekdays.
//     b. Generate all candidate slots from slots_per_weekday, applying slot_duration_min / buffer_min.
//     c. Remove slots before lead_time_hours from now.
//     d. Remove slots that overlap with existing calendar events.
//     e. Remove slots that are already booked (public_bookings).
func (s *BookingService) GetAvailability(ctx context.Context, input GetAvailabilityInput) ([]DayAvailability, error) {
	page, err := s.repo.GetBookingPageBySlug(ctx, input.Slug)
	if err != nil {
		return nil, err
	}

	rules := page.AvailabilityRules
	slotDur := rules.SlotDurationMin
	if slotDur <= 0 {
		slotDur = 30 // safe default
	}

	// Honour per-service duration override if a matching service exists.
	if input.ServiceID != "" {
		for _, svc := range page.Services {
			if svc.ID == input.ServiceID && svc.DurationMin > 0 {
				slotDur = svc.DurationMin
				break
			}
		}
	}

	bufMin := rules.BufferMin
	leadCutoff := time.Now().Add(time.Duration(rules.LeadTimeHours) * time.Hour)

	// Build a set of weekdays that are open.
	openWeekdays := make(map[time.Weekday]bool, len(rules.Weekdays))
	for _, wd := range rules.Weekdays {
		openWeekdays[time.Weekday(wd)] = true
	}

	// Load existing calendar events in the whole range at once.
	rangeStart := input.DateFrom
	rangeEnd := input.DateTo.Add(24 * time.Hour) // exclusive upper bound
	existingEvents, err := s.repo.GetCalendarEventsInRange(ctx, page.CalendarID, rangeStart, rangeEnd)
	if err != nil {
		return nil, err
	}

	// Load existing public bookings in the whole range at once.
	existingBookings, err := s.repo.GetBookedSlotsForPage(ctx, page.ID, rangeStart, rangeEnd)
	if err != nil {
		return nil, err
	}

	// Build a set of (date, time_slot) pairs that are taken.
	bookedSet := make(map[string]bool, len(existingBookings))
	for _, b := range existingBookings {
		bookedSet[b.Date.Format("2006-01-02")+"|"+b.TimeSlot] = true
	}

	var result []DayAvailability

	for d := input.DateFrom; !d.After(input.DateTo); d = d.AddDate(0, 0, 1) {
		// Skip closed weekdays.
		if !openWeekdays[d.Weekday()] {
			continue
		}

		dayStr := d.Format("2006-01-02")
		var freeSlots []string

		for _, window := range rules.SlotsPerWeekday {
			windowStart, err := parseHHMM(window.Start)
			if err != nil {
				continue
			}
			windowEnd, err := parseHHMM(window.End)
			if err != nil {
				continue
			}

			slotStart := windowStart
			for {
				slotEnd := slotStart + time.Duration(slotDur)*time.Minute
				// Slot must fit within window.
				if slotEnd > windowEnd {
					break
				}

				slotTime := combineDateAndOffset(d, slotStart)
				slotEndTime := combineDateAndOffset(d, slotEnd)
				slotKey := dayStr + "|" + formatHHMM(slotStart)

				// Check lead time: slot must be far enough in the future.
				if slotTime.Before(leadCutoff) {
					slotStart += time.Duration(slotDur+bufMin) * time.Minute
					continue
				}

				// Check breaks.
				if isInBreak(slotStart, slotEnd, rules.Breaks) {
					slotStart += time.Duration(slotDur+bufMin) * time.Minute
					continue
				}

				// Check calendar event overlap.
				if overlapsCalendarEvents(slotTime, slotEndTime, existingEvents) {
					slotStart += time.Duration(slotDur+bufMin) * time.Minute
					continue
				}

				// Check already booked.
				if bookedSet[slotKey] {
					slotStart += time.Duration(slotDur+bufMin) * time.Minute
					continue
				}

				freeSlots = append(freeSlots, formatHHMM(slotStart))
				slotStart += time.Duration(slotDur+bufMin) * time.Minute
			}
		}

		if len(freeSlots) > 0 {
			result = append(result, DayAvailability{Date: d, Slots: freeSlots})
		}
	}

	return result, nil
}

// ============================================================================
// Public: Create Booking
// ============================================================================

// CreatePublicBookingInput holds all fields from the public form submission.
type CreatePublicBookingInput struct {
	Slug          string
	ServiceID     string
	Date          time.Time
	TimeSlot      string
	CustomerName  string
	CustomerEmail string
	CustomerPhone *string
	Notes         *string
}

// CreatePublicBookingResult holds the created booking and a confirmation flag.
type CreatePublicBookingResult struct {
	Booking *models.PublicBooking
}

// CreatePublicBooking validates, checks slot collision, and persists a public booking.
// It derives tenant_id from the booking page (the public caller has no tenant context).
func (s *BookingService) CreatePublicBooking(
	ctx context.Context,
	emailClient PublicBookingEmailClient,
	calClient PublicBookingCalClient,
	input CreatePublicBookingInput,
) (*CreatePublicBookingResult, error) {
	// Load the page (active check included).
	page, err := s.repo.GetBookingPageBySlug(ctx, input.Slug)
	if err != nil {
		return nil, err
	}

	// Validate service exists on this page.
	var matchedService *models.BookingService
	for i := range page.Services {
		if page.Services[i].ID == input.ServiceID {
			matchedService = &page.Services[i]
			break
		}
	}
	if matchedService == nil {
		return nil, ErrBookingServiceNotFound
	}

	rules := page.AvailabilityRules
	slotDur := matchedService.DurationMin
	if slotDur <= 0 {
		slotDur = rules.SlotDurationMin
	}
	if slotDur <= 0 {
		slotDur = 30
	}

	// Lead-time guard: slot cannot be in the past or too soon.
	slotTime := combineDateTimeSlot(input.Date, input.TimeSlot)
	leadCutoff := time.Now().Add(time.Duration(rules.LeadTimeHours) * time.Hour)
	if slotTime.Before(leadCutoff) {
		return nil, ErrBookingSlotInPast
	}

	// Slot-end time for overlap check.
	slotEndTime := slotTime.Add(time.Duration(slotDur) * time.Minute)

	// Re-check for calendar event collision (defence-in-depth against race).
	dayStart := input.Date.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)
	existingEvents, err := s.repo.GetCalendarEventsInRange(ctx, page.CalendarID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	if overlapsCalendarEvents(slotTime, slotEndTime, existingEvents) {
		return nil, ErrBookingSlotUnavailable
	}

	booking := &models.PublicBooking{
		ID:            uuid.New(),
		TenantID:      page.TenantID,
		BookingPageID: page.ID,
		ServiceID:     input.ServiceID,
		CustomerName:  strings.TrimSpace(input.CustomerName),
		CustomerEmail: strings.ToLower(strings.TrimSpace(input.CustomerEmail)),
		CustomerPhone: input.CustomerPhone,
		Notes:         input.Notes,
		Date:          input.Date.Truncate(24 * time.Hour),
		TimeSlot:      input.TimeSlot,
		Status:        models.PublicBookingStatusConfirmed,
		CreatedAt:     time.Now(),
	}

	// Persist — the UNIQUE constraint (booking_page_id, customer_email, date, time_slot)
	// serves as the DB-level idempotency backstop for double-submit.
	if err := s.repo.CreatePublicBooking(ctx, booking); err != nil {
		return nil, err
	}

	slog.Info("public booking created",
		"booking_id", booking.ID,
		"page_id", page.ID,
		"service_id", input.ServiceID,
		"date", input.Date.Format("2006-01-02"),
		"slot", input.TimeSlot,
	)

	// Best-effort: create calendar event.
	if calClient != nil {
		eventTitle := fmt.Sprintf("%s — %s", matchedService.Name, booking.CustomerName)
		eventID, calErr := calClient.CreateEvent(ctx, PublicBookingCalEventInput{
			TenantID:    page.TenantID,
			CalendarID:  page.CalendarID.String(),
			Title:       eventTitle,
			Description: fmt.Sprintf("Öffentliche Buchung: %s (%s)", booking.CustomerName, booking.CustomerEmail),
			StartTime:   slotTime,
			EndTime:     slotEndTime,
		})
		if calErr != nil {
			slog.Warn("failed to create calendar event for public booking",
				"booking_id", booking.ID, "error", calErr)
		} else if eventID != "" {
			evID, parseErr := uuid.Parse(eventID)
			if parseErr == nil {
				booking.CalendarEventID = &evID
				// Fire-and-forget update — don't fail the booking on this.
				go func() {
					_, _ = s.repo.GetBookedSlotsForPage(context.Background(), page.ID, dayStart, dayEnd)
				}()
			}
		}
	}

	// Best-effort: send confirmation email.
	if emailClient != nil {
		emailErr := emailClient.SendConfirmation(ctx, PublicBookingConfirmationEmail{
			ToEmail:     booking.CustomerEmail,
			ToName:      booking.CustomerName,
			ServiceName: matchedService.Name,
			CompanyName: page.CompanyName,
			Date:        input.Date.Format("02.01.2006"),
			TimeSlot:    input.TimeSlot,
		})
		if emailErr != nil {
			slog.Warn("failed to send booking confirmation email",
				"booking_id", booking.ID, "error", emailErr)
		}
	}

	return &CreatePublicBookingResult{Booking: booking}, nil
}

// ============================================================================
// External client interfaces (injected, allows mocking in tests)
// ============================================================================

// PublicBookingEmailClient sends the confirmation email after a successful booking.
type PublicBookingEmailClient interface {
	SendConfirmation(ctx context.Context, email PublicBookingConfirmationEmail) error
}

// PublicBookingConfirmationEmail holds the data for the confirmation email template.
type PublicBookingConfirmationEmail struct {
	ToEmail     string
	ToName      string
	ServiceName string
	CompanyName string
	Date        string
	TimeSlot    string
}

// PublicBookingCalClient creates a calendar event after a booking is confirmed.
type PublicBookingCalClient interface {
	CreateEvent(ctx context.Context, input PublicBookingCalEventInput) (eventID string, err error)
}

// PublicBookingCalEventInput holds the fields for the calendar event creation.
type PublicBookingCalEventInput struct {
	TenantID    uuid.UUID
	CalendarID  string
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
}

// ============================================================================
// Internal helpers
// ============================================================================

// parseHHMM converts "HH:MM" to a time.Duration offset from midnight.
func parseHHMM(hhmm string) (time.Duration, error) {
	var h, m int
	if _, err := fmt.Sscanf(hhmm, "%d:%d", &h, &m); err != nil {
		return 0, fmt.Errorf("invalid HH:MM format %q: %w", hhmm, err)
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute, nil
}

// formatHHMM converts a duration offset from midnight to "HH:MM".
func formatHHMM(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

// combineDateAndOffset creates a UTC time by combining the date's Y/M/D with an offset.
func combineDateAndOffset(date time.Time, offset time.Duration) time.Time {
	loc := date.Location()
	if loc == nil {
		loc = time.UTC
	}
	y, mo, dy := date.Date()
	midnight := time.Date(y, mo, dy, 0, 0, 0, 0, loc)
	return midnight.Add(offset)
}

// combineDateTimeSlot combines a date and "HH:MM" string into a time.Time.
func combineDateTimeSlot(date time.Time, slot string) time.Time {
	dur, err := parseHHMM(slot)
	if err != nil {
		return date
	}
	return combineDateAndOffset(date, dur)
}

// isInBreak returns true if [slotStart, slotEnd) overlaps with any break window.
func isInBreak(slotStart, slotEnd time.Duration, breaks []models.AvailabilityBreak) bool {
	for _, br := range breaks {
		brStart, err := parseHHMM(br.Start)
		if err != nil {
			continue
		}
		brEnd, err := parseHHMM(br.End)
		if err != nil {
			continue
		}
		if slotStart < brEnd && slotEnd > brStart {
			return true
		}
	}
	return false
}

// overlapsCalendarEvents returns true if [slotStart, slotEnd) overlaps any event.
func overlapsCalendarEvents(slotStart, slotEnd time.Time, events []eventSlot) bool {
	for _, ev := range events {
		if slotStart.Before(ev.EndTime) && slotEnd.After(ev.StartTime) {
			return true
		}
	}
	return false
}
