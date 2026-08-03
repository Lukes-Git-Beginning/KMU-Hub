package fuhrpark

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service handles fuhrpark business logic.
type Service struct {
	repo Repository
}

// NewService creates a new fuhrpark service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ============================================================================
// Input types
// ============================================================================

// CreateVehicleInput holds data to create a vehicle.
type CreateVehicleInput struct {
	TenantID         uuid.UUID
	LicensePlate     string
	Make             string
	Model            string
	Year             int
	VIN              *string
	Color            *string
	FuelType         string
	MileageKm        int64
	TuevDueDate      *time.Time
	AssignedDriverID *uuid.UUID
	Notes            *string
}

// UpdateVehicleInput holds mutable fields for a vehicle.
type UpdateVehicleInput struct {
	TenantID         uuid.UUID
	VehicleID        uuid.UUID
	LicensePlate     *string
	Make             *string
	Model            *string
	Year             *int
	VIN              *string
	Color            *string
	FuelType         *string
	Status           *VehicleStatus
	MileageKm        *int64
	TuevDueDate      *time.Time
	ClearTuevDueDate bool // explicit nil set
	AssignedDriverID *uuid.UUID
	Notes            *string
}

// ListVehiclesInput holds filtering and pagination.
type ListVehiclesInput struct {
	TenantID uuid.UUID
	Search   string
	Status   *VehicleStatus
	FuelType *string
	Page     int
	PageSize int
}

// ScheduleServiceInput holds data to schedule a vehicle service.
type ScheduleServiceInput struct {
	TenantID    uuid.UUID
	VehicleID   uuid.UUID
	ServiceType string
	Description *string
	ScheduledAt time.Time
	CostCents   *int64
	Workshop    *string
	MileageKm   *int64
	Notes       *string
	CreatedBy   *uuid.UUID
}

// UpdateServiceInput holds mutable fields for a service entry.
type UpdateServiceInput struct {
	TenantID    uuid.UUID
	ServiceID   uuid.UUID
	ServiceType *string
	Description *string
	ScheduledAt *time.Time
	CostCents   *int64
	Workshop    *string
	MileageKm   *int64
	Status      *ServiceStatus
	Notes       *string
}

// CompleteServiceInput marks a service entry as completed.
type CompleteServiceInput struct {
	TenantID  uuid.UUID
	ServiceID uuid.UUID
	CostCents *int64
	MileageKm *int64
	Notes     *string
}

// ListServicesInput holds filtering and pagination for services.
type ListServicesInput struct {
	TenantID  uuid.UUID
	VehicleID *uuid.UUID
	Status    *ServiceStatus
	Page      int
	PageSize  int
}

// ReportDamageInput holds data to report a damage.
type ReportDamageInput struct {
	TenantID    uuid.UUID
	VehicleID   uuid.UUID
	Description string
	Severity    string
	ReportedBy  *uuid.UUID
	PhotoKeys   []string
	CostCents   *int64
	Notes       *string
}

// UpdateDamageInput holds mutable fields for a damage record.
type UpdateDamageInput struct {
	TenantID    uuid.UUID
	DamageID    uuid.UUID
	Description *string
	Severity    *string
	Status      *DamageStatus
	PhotoKeys   []string
	CostCents   *int64
	Notes       *string
}

// ResolveDamageInput marks a damage as resolved.
type ResolveDamageInput struct {
	TenantID   uuid.UUID
	DamageID   uuid.UUID
	ResolvedBy *uuid.UUID
	CostCents  *int64
	Notes      *string
}

// ListDamagesInput holds filtering and pagination for damages.
type ListDamagesInput struct {
	TenantID  uuid.UUID
	VehicleID *uuid.UUID
	Status    *DamageStatus
	Page      int
	PageSize  int
}

// GetVehicleHistoryInput holds pagination for vehicle history.
type GetVehicleHistoryInput struct {
	TenantID  uuid.UUID
	VehicleID uuid.UUID
	Page      int
	PageSize  int
}

// ============================================================================
// Vehicles
// ============================================================================

// CreateVehicle creates a new fleet vehicle.
func (s *Service) CreateVehicle(ctx context.Context, input CreateVehicleInput) (*Vehicle, error) {
	plate := strings.TrimSpace(input.LicensePlate)
	if plate == "" {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(input.Make) == "" || strings.TrimSpace(input.Model) == "" {
		return nil, ErrInvalidInput
	}
	if input.Year < 1900 || input.Year > time.Now().Year()+2 {
		return nil, ErrInvalidInput
	}

	exists, err := s.repo.PlateExists(ctx, input.TenantID, plate, nil)
	if err != nil {
		return nil, fmt.Errorf("check license plate: %w", err)
	}
	if exists {
		return nil, ErrPlateTaken
	}

	fuelType := input.FuelType
	if fuelType == "" {
		fuelType = "petrol"
	}

	now := time.Now()
	v := &Vehicle{
		ID:               uuid.New(),
		TenantID:         input.TenantID,
		LicensePlate:     plate,
		Make:             strings.TrimSpace(input.Make),
		Model:            strings.TrimSpace(input.Model),
		Year:             input.Year,
		VIN:              input.VIN,
		Color:            input.Color,
		FuelType:         fuelType,
		Status:           VehicleStatusActive,
		MileageKm:        input.MileageKm,
		TuevDueDate:      input.TuevDueDate,
		AssignedDriverID: input.AssignedDriverID,
		Notes:            input.Notes,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if createErr := s.repo.CreateVehicle(ctx, v); createErr != nil {
		return nil, fmt.Errorf("create vehicle: %w", createErr)
	}

	slog.Info("vehicle created",
		"vehicle_id", v.ID,
		"tenant_id", v.TenantID,
		"license_plate", v.LicensePlate,
	)
	return v, nil
}

// UpdateVehicle updates mutable fields on a vehicle.
func (s *Service) UpdateVehicle(ctx context.Context, input UpdateVehicleInput) (*Vehicle, error) {
	v, err := s.repo.GetVehicle(ctx, input.TenantID, input.VehicleID)
	if err != nil {
		return nil, err
	}

	if input.LicensePlate != nil {
		plate := strings.TrimSpace(*input.LicensePlate)
		if plate == "" {
			return nil, ErrInvalidInput
		}
		if plate != v.LicensePlate {
			exists, plateErr := s.repo.PlateExists(ctx, input.TenantID, plate, &v.ID)
			if plateErr != nil {
				return nil, fmt.Errorf("check license plate: %w", plateErr)
			}
			if exists {
				return nil, ErrPlateTaken
			}
			v.LicensePlate = plate
		}
	}
	if input.Make != nil {
		v.Make = strings.TrimSpace(*input.Make)
	}
	if input.Model != nil {
		v.Model = strings.TrimSpace(*input.Model)
	}
	if input.Year != nil {
		v.Year = *input.Year
	}
	if input.VIN != nil {
		v.VIN = input.VIN
	}
	if input.Color != nil {
		v.Color = input.Color
	}
	if input.FuelType != nil {
		v.FuelType = *input.FuelType
	}
	if input.Status != nil {
		v.Status = *input.Status
	}
	if input.MileageKm != nil {
		// Guard: odometer cannot go backwards (Bug #12 — mileage decrement prevention)
		if *input.MileageKm < v.MileageKm {
			return nil, ErrInvalidInput
		}
		v.MileageKm = *input.MileageKm
	}
	if input.ClearTuevDueDate {
		v.TuevDueDate = nil
	} else if input.TuevDueDate != nil {
		v.TuevDueDate = input.TuevDueDate
	}
	if input.AssignedDriverID != nil {
		v.AssignedDriverID = input.AssignedDriverID
	}
	if input.Notes != nil {
		v.Notes = input.Notes
	}

	v.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateVehicle(ctx, v); updateErr != nil {
		return nil, fmt.Errorf("update vehicle: %w", updateErr)
	}

	slog.Info("vehicle updated", "vehicle_id", v.ID, "tenant_id", v.TenantID)
	return v, nil
}

// DeleteVehicle soft-deletes a vehicle.
func (s *Service) DeleteVehicle(ctx context.Context, tenantID, vehicleID uuid.UUID) error {
	if _, err := s.repo.GetVehicle(ctx, tenantID, vehicleID); err != nil {
		return err
	}
	if delErr := s.repo.SoftDeleteVehicle(ctx, tenantID, vehicleID); delErr != nil {
		return fmt.Errorf("soft delete vehicle: %w", delErr)
	}
	slog.Info("vehicle deleted", "vehicle_id", vehicleID, "tenant_id", tenantID)
	return nil
}

// GetVehicle retrieves a vehicle by ID.
func (s *Service) GetVehicle(ctx context.Context, tenantID, vehicleID uuid.UUID) (*Vehicle, error) {
	return s.repo.GetVehicle(ctx, tenantID, vehicleID)
}

// ListVehicles retrieves vehicles with optional filtering and pagination.
func (s *Service) ListVehicles(ctx context.Context, input ListVehiclesInput) ([]*Vehicle, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	filter := ListVehiclesFilter{
		Search:   input.Search,
		Status:   input.Status,
		FuelType: input.FuelType,
	}
	return s.repo.ListVehicles(ctx, input.TenantID, filter, offset, input.PageSize)
}

// CheckTuevDue lists vehicles with tuev_due_date within the next daysAhead days.
func (s *Service) CheckTuevDue(ctx context.Context, tenantID uuid.UUID, daysAhead int) ([]*Vehicle, error) {
	if daysAhead <= 0 {
		daysAhead = 30
	}
	now := time.Now()
	to := now.AddDate(0, 0, daysAhead)
	all, _, err := s.repo.ListVehicles(ctx, tenantID, ListVehiclesFilter{}, 0, 1000)
	if err != nil {
		return nil, err
	}
	var due []*Vehicle
	for _, v := range all {
		if v.TuevDueDate != nil && !v.TuevDueDate.After(to) && !v.TuevDueDate.Before(now) {
			due = append(due, v)
		}
	}
	return due, nil
}

// ============================================================================
// Services
// ============================================================================

// ScheduleService creates a new vehicle service entry.
func (s *Service) ScheduleService(ctx context.Context, input ScheduleServiceInput) (*VehicleService, error) {
	if strings.TrimSpace(input.ServiceType) == "" {
		return nil, ErrInvalidInput
	}
	// Verify vehicle belongs to tenant
	if _, err := s.repo.GetVehicle(ctx, input.TenantID, input.VehicleID); err != nil {
		return nil, err
	}

	now := time.Now()
	svc := &VehicleService{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		VehicleID:   input.VehicleID,
		ServiceType: strings.TrimSpace(input.ServiceType),
		Description: input.Description,
		ScheduledAt: input.ScheduledAt,
		CostCents:   input.CostCents,
		Workshop:    input.Workshop,
		MileageKm:   input.MileageKm,
		Status:      ServiceStatusScheduled,
		Notes:       input.Notes,
		CreatedBy:   input.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if createErr := s.repo.CreateService(ctx, svc); createErr != nil {
		return nil, fmt.Errorf("create service: %w", createErr)
	}

	slog.Info("vehicle service scheduled",
		"service_id", svc.ID,
		"vehicle_id", svc.VehicleID,
		"service_type", svc.ServiceType,
	)
	return svc, nil
}

// UpdateService updates mutable fields on a service entry.
func (s *Service) UpdateService(ctx context.Context, input UpdateServiceInput) (*VehicleService, error) {
	svc, err := s.repo.GetService(ctx, input.TenantID, input.ServiceID)
	if err != nil {
		return nil, err
	}

	if input.ServiceType != nil {
		svc.ServiceType = strings.TrimSpace(*input.ServiceType)
	}
	if input.Description != nil {
		svc.Description = input.Description
	}
	if input.ScheduledAt != nil {
		svc.ScheduledAt = *input.ScheduledAt
	}
	if input.CostCents != nil {
		svc.CostCents = input.CostCents
	}
	if input.Workshop != nil {
		svc.Workshop = input.Workshop
	}
	if input.MileageKm != nil {
		svc.MileageKm = input.MileageKm
	}
	if input.Status != nil {
		svc.Status = *input.Status
	}
	if input.Notes != nil {
		svc.Notes = input.Notes
	}
	svc.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateService(ctx, svc); updateErr != nil {
		return nil, fmt.Errorf("update service: %w", updateErr)
	}
	return svc, nil
}

// DeleteService removes a service entry permanently.
func (s *Service) DeleteService(ctx context.Context, tenantID, serviceID uuid.UUID) error {
	if _, err := s.repo.GetService(ctx, tenantID, serviceID); err != nil {
		return err
	}
	return s.repo.DeleteService(ctx, tenantID, serviceID)
}

// CompleteService marks a service entry as completed.
// Pre-check guard: only "scheduled" or "in_progress" services may be completed.
func (s *Service) CompleteService(ctx context.Context, input CompleteServiceInput) (*VehicleService, error) {
	svc, err := s.repo.GetService(ctx, input.TenantID, input.ServiceID)
	if err != nil {
		return nil, err
	}

	// Pre-check: only scheduled/in_progress may be completed (Welle-1-guard pattern)
	if svc.Status != ServiceStatusScheduled && svc.Status != ServiceStatusInProgress {
		return nil, ErrInvalidTransition
	}

	now := time.Now()
	svc.Status = ServiceStatusCompleted
	svc.CompletedAt = &now
	if input.CostCents != nil {
		svc.CostCents = input.CostCents
	}
	if input.MileageKm != nil {
		svc.MileageKm = input.MileageKm
	}
	if input.Notes != nil {
		svc.Notes = input.Notes
	}
	svc.UpdatedAt = now

	if updateErr := s.repo.UpdateService(ctx, svc); updateErr != nil {
		return nil, fmt.Errorf("complete service: %w", updateErr)
	}

	slog.Info("vehicle service completed",
		"service_id", svc.ID,
		"vehicle_id", svc.VehicleID,
	)
	return svc, nil
}

// ListServices retrieves services with optional filtering and pagination.
func (s *Service) ListServices(ctx context.Context, input ListServicesInput) ([]*VehicleService, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	filter := ListServicesFilter{
		VehicleID: input.VehicleID,
		Status:    input.Status,
	}
	return s.repo.ListServices(ctx, input.TenantID, filter, offset, input.PageSize)
}

// ListUpcomingServices returns scheduled services within daysAhead days.
// Only services with scheduled_at <= now+daysAhead are returned, so the caller
// sees genuinely upcoming work rather than the entire scheduled backlog.
func (s *Service) ListUpcomingServices(ctx context.Context, tenantID uuid.UUID, daysAhead, page, pageSize int) ([]*VehicleService, int, error) {
	if daysAhead <= 0 {
		daysAhead = 30
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	status := ServiceStatusScheduled
	cutoff := time.Now().AddDate(0, 0, daysAhead)
	offset := (page - 1) * pageSize
	return s.repo.ListServices(ctx, tenantID, ListServicesFilter{
		Status:          &status,
		ScheduledBefore: &cutoff,
	}, offset, pageSize)
}

// ============================================================================
// Damages
// ============================================================================

// ReportDamage creates a new damage report for a vehicle.
func (s *Service) ReportDamage(ctx context.Context, input ReportDamageInput) (*VehicleDamage, error) {
	if strings.TrimSpace(input.Description) == "" {
		return nil, ErrInvalidInput
	}
	// Verify vehicle belongs to tenant
	if _, err := s.repo.GetVehicle(ctx, input.TenantID, input.VehicleID); err != nil {
		return nil, err
	}

	severity := input.Severity
	if severity == "" {
		severity = "minor"
	}

	photoKeys := input.PhotoKeys
	if photoKeys == nil {
		photoKeys = []string{}
	}

	now := time.Now()
	d := &VehicleDamage{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		VehicleID:   input.VehicleID,
		Description: strings.TrimSpace(input.Description),
		Severity:    severity,
		Status:      DamageStatusReported,
		ReportedBy:  input.ReportedBy,
		PhotoKeys:   photoKeys,
		CostCents:   input.CostCents,
		Notes:       input.Notes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if createErr := s.repo.CreateDamage(ctx, d); createErr != nil {
		return nil, fmt.Errorf("create damage: %w", createErr)
	}

	slog.Info("vehicle damage reported",
		"damage_id", d.ID,
		"vehicle_id", d.VehicleID,
		"severity", d.Severity,
	)
	return d, nil
}

// UpdateDamage updates mutable fields on a damage record.
func (s *Service) UpdateDamage(ctx context.Context, input UpdateDamageInput) (*VehicleDamage, error) {
	d, err := s.repo.GetDamage(ctx, input.TenantID, input.DamageID)
	if err != nil {
		return nil, err
	}

	if input.Description != nil {
		d.Description = strings.TrimSpace(*input.Description)
	}
	if input.Severity != nil {
		d.Severity = *input.Severity
	}
	if input.Status != nil {
		d.Status = *input.Status
	}
	if input.PhotoKeys != nil {
		d.PhotoKeys = input.PhotoKeys
	}
	if input.CostCents != nil {
		d.CostCents = input.CostCents
	}
	if input.Notes != nil {
		d.Notes = input.Notes
	}
	d.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateDamage(ctx, d); updateErr != nil {
		return nil, fmt.Errorf("update damage: %w", updateErr)
	}
	return d, nil
}

// ResolveDamage marks a damage as resolved.
// Pre-check guard: only "reported" or "in_repair" may be resolved (no double-resolve).
func (s *Service) ResolveDamage(ctx context.Context, input ResolveDamageInput) (*VehicleDamage, error) {
	d, err := s.repo.GetDamage(ctx, input.TenantID, input.DamageID)
	if err != nil {
		return nil, err
	}

	// Pre-check: only reported/in_repair may be resolved (Welle-1-guard pattern)
	if d.Status != DamageStatusReported && d.Status != DamageStatusInRepair {
		return nil, ErrInvalidTransition
	}

	now := time.Now()
	d.Status = DamageStatusResolved
	d.ResolvedAt = &now
	d.ResolvedBy = input.ResolvedBy
	if input.CostCents != nil {
		d.CostCents = input.CostCents
	}
	if input.Notes != nil {
		d.Notes = input.Notes
	}
	d.UpdatedAt = now

	if updateErr := s.repo.UpdateDamage(ctx, d); updateErr != nil {
		return nil, fmt.Errorf("resolve damage: %w", updateErr)
	}

	slog.Info("vehicle damage resolved",
		"damage_id", d.ID,
		"vehicle_id", d.VehicleID,
	)
	return d, nil
}

// ListDamages retrieves damage records with optional filtering and pagination.
func (s *Service) ListDamages(ctx context.Context, input ListDamagesInput) ([]*VehicleDamage, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	filter := ListDamagesFilter{
		VehicleID: input.VehicleID,
		Status:    input.Status,
	}
	return s.repo.ListDamages(ctx, input.TenantID, filter, offset, input.PageSize)
}

// ============================================================================
// History
// ============================================================================

// ============================================================================
// Fuel Logs
// ============================================================================

// ListFuelLogs retrieves fuel log entries with optional filtering and pagination.
func (s *Service) ListFuelLogs(ctx context.Context, params ListFuelLogsParams) ([]FuelLog, int, error) {
	return s.repo.ListFuelLogs(ctx, params)
}

// CreateFuelLog creates a new fuel/charging log entry.
func (s *Service) CreateFuelLog(ctx context.Context, req FuelLog) (FuelLog, error) {
	if req.VehicleID == (uuid.UUID{}) || req.Liters <= 0 || req.CostCents < 0 || req.MileageKm < 0 {
		return FuelLog{}, ErrInvalidInput
	}
	return s.repo.CreateFuelLog(ctx, req)
}

// UpdateFuelLog updates an existing fuel log entry.
func (s *Service) UpdateFuelLog(ctx context.Context, req FuelLog) (FuelLog, error) {
	if req.ID == (uuid.UUID{}) {
		return FuelLog{}, ErrInvalidInput
	}
	return s.repo.UpdateFuelLog(ctx, req)
}

// DeleteFuelLog deletes a fuel log entry.
func (s *Service) DeleteFuelLog(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteFuelLog(ctx, tenantID, id)
}

// ============================================================================
// Trip Logs
// ============================================================================

// ListTripLogs retrieves trip log entries with optional filtering and pagination.
func (s *Service) ListTripLogs(ctx context.Context, params ListTripLogsParams) ([]TripLog, int, error) {
	return s.repo.ListTripLogs(ctx, params)
}

// CreateTripLog creates a new trip/logbook entry.
func (s *Service) CreateTripLog(ctx context.Context, req TripLog) (TripLog, error) {
	if req.VehicleID == (uuid.UUID{}) || strings.TrimSpace(req.StartLocation) == "" || strings.TrimSpace(req.EndLocation) == "" {
		return TripLog{}, ErrInvalidInput
	}
	if req.EndKm < req.StartKm {
		return TripLog{}, fmt.Errorf("%w: end_km must be >= start_km", ErrInvalidInput)
	}
	return s.repo.CreateTripLog(ctx, req)
}

// UpdateTripLog updates an existing trip log entry.
func (s *Service) UpdateTripLog(ctx context.Context, req TripLog) (TripLog, error) {
	if req.ID == (uuid.UUID{}) {
		return TripLog{}, ErrInvalidInput
	}
	return s.repo.UpdateTripLog(ctx, req)
}

// DeleteTripLog deletes a trip log entry.
func (s *Service) DeleteTripLog(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteTripLog(ctx, tenantID, id)
}

// ============================================================================
// Vehicle Documents
// ============================================================================

// ListVehicleDocuments retrieves documents attached to a vehicle.
func (s *Service) ListVehicleDocuments(ctx context.Context, params ListVehicleDocumentsParams) ([]VehicleDocument, int, error) {
	return s.repo.ListVehicleDocuments(ctx, params)
}

// CreateVehicleDocument records a new vehicle document (after upload via presign).
func (s *Service) CreateVehicleDocument(ctx context.Context, doc VehicleDocument) (VehicleDocument, error) {
	if doc.VehicleID == (uuid.UUID{}) || strings.TrimSpace(doc.ObjectKey) == "" || strings.TrimSpace(doc.Name) == "" {
		return VehicleDocument{}, ErrInvalidInput
	}
	return s.repo.CreateVehicleDocument(ctx, doc)
}

// DeleteVehicleDocument deletes a vehicle document record.
func (s *Service) DeleteVehicleDocument(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteVehicleDocument(ctx, tenantID, id)
}

// ============================================================================
// Driver Licenses
// ============================================================================

// ListDriverLicenses retrieves driver license checks, optionally filtered by driver.
func (s *Service) ListDriverLicenses(ctx context.Context, params ListDriverLicensesParams) ([]DriverLicense, int, error) {
	return s.repo.ListDriverLicenses(ctx, params)
}

// CreateDriverLicense records a new driver license compliance check.
func (s *Service) CreateDriverLicense(ctx context.Context, lic DriverLicense) (DriverLicense, error) {
	if lic.DriverID == (uuid.UUID{}) || lic.NextCheckDueDate.IsZero() {
		return DriverLicense{}, ErrInvalidInput
	}
	if lic.CheckedAt.IsZero() {
		lic.CheckedAt = time.Now()
	}
	return s.repo.CreateDriverLicense(ctx, lic)
}

// UpdateDriverLicense corrects an existing driver license check entry.
func (s *Service) UpdateDriverLicense(ctx context.Context, lic DriverLicense) (DriverLicense, error) {
	if lic.ID == (uuid.UUID{}) || lic.NextCheckDueDate.IsZero() {
		return DriverLicense{}, ErrInvalidInput
	}
	return s.repo.UpdateDriverLicense(ctx, lic)
}

// DeleteDriverLicense deletes a driver license check entry.
func (s *Service) DeleteDriverLicense(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteDriverLicense(ctx, tenantID, id)
}

// ============================================================================
// GPS
// ============================================================================

// IngestGpsPositions ingests a batch of GPS positions for a vehicle.
func (s *Service) IngestGpsPositions(ctx context.Context, tenantID, vehicleID uuid.UUID, positions []GpsPosition) (int, error) {
	if vehicleID == (uuid.UUID{}) || len(positions) == 0 {
		return 0, ErrInvalidInput
	}
	return s.repo.IngestGpsPositions(ctx, tenantID, vehicleID, positions)
}

// GetVehicleRoutes returns aggregated route data per vehicle per day.
func (s *Service) GetVehicleRoutes(ctx context.Context, params GetVehicleRoutesParams) ([]VehicleRouteAggregation, error) {
	return s.repo.GetVehicleRoutes(ctx, params)
}

// GetGpsPositions returns raw GPS positions for a vehicle in a time range.
func (s *Service) GetGpsPositions(ctx context.Context, params GetGpsPositionsParams) ([]GpsPosition, error) {
	return s.repo.GetGpsPositions(ctx, params)
}

// GetVehicleHistory returns a unified timeline of services and damages.
func (s *Service) GetVehicleHistory(ctx context.Context, input GetVehicleHistoryInput) ([]*HistoryEntry, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	// Verify vehicle belongs to tenant
	if _, err := s.repo.GetVehicle(ctx, input.TenantID, input.VehicleID); err != nil {
		return nil, 0, err
	}
	offset := (input.Page - 1) * input.PageSize
	return s.repo.GetVehicleHistory(ctx, input.TenantID, input.VehicleID, offset, input.PageSize)
}
