package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BookingService represents a single bookable service within a BookingPage.
type BookingService struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	DurationMin int     `json:"duration_min"`
	Price       string  `json:"price"` // decimal string e.g. "120.00"
	Description *string `json:"description,omitempty"`
}

// AvailabilitySlot represents a start/end time window within a weekday.
type AvailabilitySlot struct {
	Start string `json:"start"` // "HH:MM"
	End   string `json:"end"`   // "HH:MM"
}

// AvailabilityBreak is a time window during which no bookings are allowed (e.g. lunch).
type AvailabilityBreak struct {
	Start string `json:"start"` // "HH:MM"
	End   string `json:"end"`   // "HH:MM"
}

// AvailabilityRules defines when a booking page accepts appointments.
// JSON-stored in booking_pages.availability_rules.
type AvailabilityRules struct {
	// Weekdays on which the page is open: 0=Sunday … 6=Saturday (Go time.Weekday values).
	Weekdays       []int              `json:"weekdays"`
	SlotsPerWeekday []AvailabilitySlot `json:"slots_per_weekday"`
	SlotDurationMin int               `json:"slot_duration_min"` // default slot length in minutes
	BufferMin       int               `json:"buffer_min"`        // gap between consecutive bookings
	LeadTimeHours   int               `json:"lead_time_hours"`   // minimum advance notice
	Breaks          []AvailabilityBreak `json:"breaks,omitempty"` // daily break windows
}

// BookingPage is the database representation of a public booking page.
type BookingPage struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	CalendarID        uuid.UUID       `json:"calendar_id"`
	Slug              string          `json:"slug"`
	CompanyName       string          `json:"company_name"`
	LogoURL           *string         `json:"logo_url,omitempty"`
	Services          []BookingService `json:"services"`
	AvailabilityRules AvailabilityRules `json:"availability_rules"`
	Active            bool            `json:"active"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// AvailabilityRulesJSON marshals/unmarshals the rules to/from JSONB.
func (r *AvailabilityRules) Scan(data interface{}) error {
	var b []byte
	switch v := data.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		*r = AvailabilityRules{}
		return nil
	}
	return json.Unmarshal(b, r)
}

// PublicBookingStatus enumerates the allowed status values.
const (
	PublicBookingStatusPending   = "pending"
	PublicBookingStatusConfirmed = "confirmed"
	PublicBookingStatusCancelled = "cancelled"
	PublicBookingStatusCompleted = "completed"
)

// PublicBooking is the database representation of a customer booking.
type PublicBooking struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	BookingPageID   uuid.UUID  `json:"booking_page_id"`
	ServiceID       string     `json:"service_id"`
	CustomerName    string     `json:"customer_name"`
	CustomerEmail   string     `json:"customer_email"`
	CustomerPhone   *string    `json:"customer_phone,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	Date            time.Time  `json:"date"`    // date only, no time component
	TimeSlot        string     `json:"time_slot"` // "HH:MM"
	StaffUserID     *uuid.UUID `json:"staff_user_id,omitempty"`
	Status          string     `json:"status"`
	CalendarEventID *uuid.UUID `json:"calendar_event_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}
