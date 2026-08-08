package fuhrpark

import (
	"time"

	"github.com/google/uuid"
)

// ServiceStatus represents the lifecycle state of a vehicle service entry.
type ServiceStatus string

const (
	ServiceStatusScheduled  ServiceStatus = "scheduled"
	ServiceStatusInProgress ServiceStatus = "in_progress"
	ServiceStatusCompleted  ServiceStatus = "completed"
	ServiceStatusCancelled  ServiceStatus = "cancelled"
)

// DamageStatus represents the lifecycle state of a damage report.
type DamageStatus string

const (
	DamageStatusReported DamageStatus = "reported"
	DamageStatusInRepair DamageStatus = "in_repair"
	DamageStatusResolved DamageStatus = "resolved"
)

// VehicleStatus represents the operational state of a vehicle.
type VehicleStatus string

const (
	VehicleStatusActive         VehicleStatus = "active"
	VehicleStatusInService      VehicleStatus = "in_service"
	VehicleStatusInactive       VehicleStatus = "inactive"
	VehicleStatusDecommissioned VehicleStatus = "decommissioned"
)

// Vehicle represents a fleet vehicle.
type Vehicle struct {
	ID                 uuid.UUID     `json:"id"`
	TenantID           uuid.UUID     `json:"tenant_id"`
	LicensePlate       string        `json:"license_plate"`
	Make               string        `json:"make"`
	Model              string        `json:"model"`
	Year               int           `json:"year"`
	VIN                *string       `json:"vin,omitempty"`
	Color              *string       `json:"color,omitempty"`
	FuelType           string        `json:"fuel_type"`
	Status             VehicleStatus `json:"status"`
	MileageKm          int64         `json:"mileage_km"`
	TuevDueDate        *time.Time    `json:"tuev_due_date,omitempty"`
	TuevReminderSentAt *time.Time    `json:"tuev_reminder_sent_at,omitempty"`
	AssignedDriverID   *uuid.UUID    `json:"assigned_driver_id,omitempty"`
	Notes              *string       `json:"notes,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	DeletedAt          *time.Time    `json:"deleted_at,omitempty"`
}

// VehicleService records a maintenance or inspection event for a vehicle.
type VehicleService struct {
	ID          uuid.UUID     `json:"id"`
	TenantID    uuid.UUID     `json:"tenant_id"`
	VehicleID   uuid.UUID     `json:"vehicle_id"`
	ServiceType string        `json:"service_type"`
	Description *string       `json:"description,omitempty"`
	ScheduledAt time.Time     `json:"scheduled_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	CostCents   *int64        `json:"cost_cents,omitempty"`
	Workshop    *string       `json:"workshop,omitempty"`
	MileageKm   *int64        `json:"mileage_km,omitempty"`
	Status      ServiceStatus `json:"status"`
	Notes       *string       `json:"notes,omitempty"`
	CreatedBy   *uuid.UUID    `json:"created_by,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// VehicleDamage records a reported damage incident for a vehicle.
type VehicleDamage struct {
	ID          uuid.UUID    `json:"id"`
	TenantID    uuid.UUID    `json:"tenant_id"`
	VehicleID   uuid.UUID    `json:"vehicle_id"`
	Description string       `json:"description"`
	Severity    string       `json:"severity"`
	Status      DamageStatus `json:"status"`
	ReportedBy  *uuid.UUID   `json:"reported_by,omitempty"`
	ResolvedBy  *uuid.UUID   `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time   `json:"resolved_at,omitempty"`
	PhotoKeys   []string     `json:"photo_keys"`
	CostCents   *int64       `json:"cost_cents,omitempty"`
	Notes       *string      `json:"notes,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// HistoryEntry is a unified timeline entry for a vehicle (service or damage).
type HistoryEntry struct {
	Kind       string    `json:"kind"` // "service" | "damage"
	ID         uuid.UUID `json:"id"`
	Summary    string    `json:"summary"`
	OccurredAt time.Time `json:"occurred_at"`
}

// FuelLog represents a fuel/charging record for a vehicle.
type FuelLog struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	VehicleID uuid.UUID `json:"vehicle_id"`
	Date      time.Time `json:"date"`
	Liters    float64   `json:"liters"`
	CostCents int64     `json:"cost_cents"`
	MileageKm int64     `json:"mileage_km"`
	FuelType  string    `json:"fuel_type"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TripLog represents a single trip/logbook entry.
type TripLog struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	VehicleID     uuid.UUID `json:"vehicle_id"`
	Date          time.Time `json:"date"`
	StartLocation string    `json:"start_location"`
	EndLocation   string    `json:"end_location"`
	Purpose       string    `json:"purpose"`
	StartKm       int64     `json:"start_km"`
	EndKm         int64     `json:"end_km"`
	Km            int64     `json:"km"` // computed column
	IsPrivate     bool      `json:"is_private"`
	DriverName    string    `json:"driver_name"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// VehicleDocument represents a document attached to a vehicle.
type VehicleDocument struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	VehicleID  uuid.UUID  `json:"vehicle_id"`
	DocType    string     `json:"doc_type"`
	Name       string     `json:"name"`
	ObjectKey  string     `json:"object_key"`
	UploadDate time.Time  `json:"upload_date"`
	ExpiryDate *time.Time `json:"expiry_date,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// DriverLicense records a single driving-licence compliance check for a
// driver (Fuehrerscheinkontrolle). Rows are append-only history; the most
// recent row per driver_id is the current status.
type DriverLicense struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	DriverID         uuid.UUID  `json:"driver_id"`
	LicenseClasses   []string   `json:"license_classes"`
	ExpiryDate       *time.Time `json:"expiry_date,omitempty"`
	CheckedAt        time.Time  `json:"checked_at"`
	NextCheckDueDate time.Time  `json:"next_check_due_date"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// GpsPosition represents a single GPS fix for a vehicle.
type GpsPosition struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	VehicleID  uuid.UUID `json:"vehicle_id"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	SpeedKmh   *float64  `json:"speed_kmh,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// VehicleRouteAggregation is the result of aggregating GPS positions per vehicle per day.
type VehicleRouteAggregation struct {
	VehicleID   uuid.UUID
	VehicleName string
	Date        time.Time
	DailyKm     float64
	Status      string
	Positions   []GpsPosition
}

// ListFuelLogsParams holds filtering and pagination for fuel log queries.
type ListFuelLogsParams struct {
	TenantID  uuid.UUID
	VehicleID uuid.UUID // zero value = no filter
	Page      int32
	PageSize  int32
}

// ListTripLogsParams holds filtering and pagination for trip log queries.
type ListTripLogsParams struct {
	TenantID  uuid.UUID
	VehicleID uuid.UUID // zero value = no filter
	Page      int32
	PageSize  int32
}

// ListVehicleDocumentsParams holds filtering and pagination for vehicle document queries.
type ListVehicleDocumentsParams struct {
	TenantID  uuid.UUID
	VehicleID uuid.UUID
	Page      int32
	PageSize  int32
}

// ListDriverLicensesParams holds filtering and pagination for driver license queries.
type ListDriverLicensesParams struct {
	TenantID uuid.UUID
	DriverID uuid.UUID // zero value = no filter
	Page     int32
	PageSize int32
}

// GetVehicleRoutesParams holds filtering for vehicle route aggregation.
type GetVehicleRoutesParams struct {
	TenantID  uuid.UUID
	DateFrom  time.Time
	DateTo    time.Time
	VehicleID uuid.UUID // zero value = no filter
}

// GetGpsPositionsParams holds filtering for GPS position queries.
type GetGpsPositionsParams struct {
	TenantID  uuid.UUID
	VehicleID uuid.UUID
	From      time.Time
	To        time.Time
	Limit     int32
}

// BookingStatus represents the lifecycle state of a vehicle booking.
// Vocabulary matches machine_bookings so a reservation means the same thing
// across modules.
type BookingStatus string

const (
	BookingStatusBooked    BookingStatus = "booked"
	BookingStatusInUse     BookingStatus = "in_use"
	BookingStatusCompleted BookingStatus = "completed"
	BookingStatusCancelled BookingStatus = "cancelled"
)

// ActiveBookingStatuses are the states that block the vehicle for the booked
// interval. Cancelled and completed bookings free it again.
var ActiveBookingStatuses = []BookingStatus{BookingStatusBooked, BookingStatusInUse}

// VehicleBooking reserves a pool vehicle for a user over a time interval.
type VehicleBooking struct {
	ID        uuid.UUID     `json:"id"`
	TenantID  uuid.UUID     `json:"tenant_id"`
	VehicleID uuid.UUID     `json:"vehicle_id"`
	UserID    uuid.UUID     `json:"user_id"`
	StartsAt  time.Time     `json:"starts_at"`
	EndsAt    time.Time     `json:"ends_at"`
	Purpose   string        `json:"purpose"`
	Status    BookingStatus `json:"status"`
	CreatedBy *uuid.UUID    `json:"created_by"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// ListVehicleBookingsParams holds filtering and pagination for booking queries.
type ListVehicleBookingsParams struct {
	TenantID  uuid.UUID
	VehicleID uuid.UUID      // zero value = no filter
	UserID    uuid.UUID      // zero value = no filter
	Status    *BookingStatus // nil = no filter
	From      *time.Time     // nil = no lower bound on ends_at
	To        *time.Time     // nil = no upper bound on starts_at
	Page      int32
	PageSize  int32
}
