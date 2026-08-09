package server

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/fuhrpark"
	"github.com/kmuhub/kmuhub/internal/middleware"
	fuhrparkv1 "github.com/kmuhub/kmuhub/proto/fuhrpark/v1"
)

// ============================================================================
// Stub repository (server-package copy — fuhrpark's own test doubles live in
// that package's _test.go files and aren't importable here). Every method
// returns r.err when set; otherwise it echoes back plausible defaults so the
// proto mappers in fuhrpark_grpc.go never dereference a nil pointer. Selected
// GPS/booking calls capture their arguments to prove tenant/actor scoping
// comes from the authenticated context, not the request body.
// ============================================================================

type stubFuhrparkRepo struct {
	mu  sync.Mutex
	err error

	lastIngestTenantID   uuid.UUID
	lastRoutesParams     fuhrpark.GetVehicleRoutesParams
	lastGpsParams        fuhrpark.GetGpsPositionsParams
	lastBookingCreatedBy *uuid.UUID

	gpsPositions []fuhrpark.GpsPosition
	routes       []fuhrpark.VehicleRouteAggregation
}

func newStubFuhrparkRepo() *stubFuhrparkRepo { return &stubFuhrparkRepo{} }

// --- Vehicles ---

func (r *stubFuhrparkRepo) CreateVehicle(_ context.Context, v *fuhrpark.Vehicle) error {
	if r.err != nil {
		return r.err
	}
	v.ID = uuid.New()
	v.Status = fuhrpark.VehicleStatusActive
	return nil
}

func (r *stubFuhrparkRepo) UpdateVehicle(_ context.Context, _ *fuhrpark.Vehicle) error {
	return r.err
}

func (r *stubFuhrparkRepo) SoftDeleteVehicle(_ context.Context, _, _ uuid.UUID) error {
	return r.err
}

func (r *stubFuhrparkRepo) GetVehicle(_ context.Context, tenantID, vehicleID uuid.UUID) (*fuhrpark.Vehicle, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &fuhrpark.Vehicle{
		ID: vehicleID, TenantID: tenantID, LicensePlate: "AB-123",
		Make: "Test", Model: "X", FuelType: "petrol", Status: fuhrpark.VehicleStatusActive,
	}, nil
}

func (r *stubFuhrparkRepo) ListVehicles(_ context.Context, _ uuid.UUID, _ fuhrpark.ListVehiclesFilter, _, _ int) ([]*fuhrpark.Vehicle, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

func (r *stubFuhrparkRepo) PlateExists(_ context.Context, _ uuid.UUID, _ string, _ *uuid.UUID) (bool, error) {
	return false, r.err
}

// --- Services ---

func (r *stubFuhrparkRepo) CreateService(_ context.Context, s *fuhrpark.VehicleService) error {
	if r.err != nil {
		return r.err
	}
	s.ID = uuid.New()
	s.Status = fuhrpark.ServiceStatusScheduled
	return nil
}

func (r *stubFuhrparkRepo) UpdateService(_ context.Context, _ *fuhrpark.VehicleService) error {
	return r.err
}

func (r *stubFuhrparkRepo) DeleteService(_ context.Context, _, _ uuid.UUID) error { return r.err }

func (r *stubFuhrparkRepo) GetService(_ context.Context, tenantID, serviceID uuid.UUID) (*fuhrpark.VehicleService, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &fuhrpark.VehicleService{
		ID: serviceID, TenantID: tenantID, ServiceType: "inspection",
		ScheduledAt: time.Now(), Status: fuhrpark.ServiceStatusScheduled,
	}, nil
}

func (r *stubFuhrparkRepo) ListServices(_ context.Context, _ uuid.UUID, _ fuhrpark.ListServicesFilter, _, _ int) ([]*fuhrpark.VehicleService, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

// --- Damages ---

func (r *stubFuhrparkRepo) CreateDamage(_ context.Context, d *fuhrpark.VehicleDamage) error {
	if r.err != nil {
		return r.err
	}
	d.ID = uuid.New()
	d.Status = fuhrpark.DamageStatusReported
	return nil
}

func (r *stubFuhrparkRepo) UpdateDamage(_ context.Context, _ *fuhrpark.VehicleDamage) error {
	return r.err
}

func (r *stubFuhrparkRepo) GetDamage(_ context.Context, tenantID, damageID uuid.UUID) (*fuhrpark.VehicleDamage, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &fuhrpark.VehicleDamage{
		ID: damageID, TenantID: tenantID, Description: "dent",
		Severity: "minor", Status: fuhrpark.DamageStatusReported,
	}, nil
}

func (r *stubFuhrparkRepo) ListDamages(_ context.Context, _ uuid.UUID, _ fuhrpark.ListDamagesFilter, _, _ int) ([]*fuhrpark.VehicleDamage, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

// --- History ---

func (r *stubFuhrparkRepo) GetVehicleHistory(_ context.Context, _, _ uuid.UUID, _, _ int) ([]*fuhrpark.HistoryEntry, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

// --- Fuel logs ---

func (r *stubFuhrparkRepo) ListFuelLogs(_ context.Context, _ fuhrpark.ListFuelLogsParams) ([]fuhrpark.FuelLog, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

func (r *stubFuhrparkRepo) CreateFuelLog(_ context.Context, log fuhrpark.FuelLog) (fuhrpark.FuelLog, error) {
	if r.err != nil {
		return fuhrpark.FuelLog{}, r.err
	}
	log.ID = uuid.New()
	return log, nil
}

func (r *stubFuhrparkRepo) UpdateFuelLog(_ context.Context, log fuhrpark.FuelLog) (fuhrpark.FuelLog, error) {
	if r.err != nil {
		return fuhrpark.FuelLog{}, r.err
	}
	return log, nil
}

func (r *stubFuhrparkRepo) DeleteFuelLog(_ context.Context, _, _ uuid.UUID) error { return r.err }

// --- Trip logs ---

func (r *stubFuhrparkRepo) ListTripLogs(_ context.Context, _ fuhrpark.ListTripLogsParams) ([]fuhrpark.TripLog, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

func (r *stubFuhrparkRepo) CreateTripLog(_ context.Context, log fuhrpark.TripLog) (fuhrpark.TripLog, error) {
	if r.err != nil {
		return fuhrpark.TripLog{}, r.err
	}
	log.ID = uuid.New()
	return log, nil
}

func (r *stubFuhrparkRepo) UpdateTripLog(_ context.Context, log fuhrpark.TripLog) (fuhrpark.TripLog, error) {
	if r.err != nil {
		return fuhrpark.TripLog{}, r.err
	}
	return log, nil
}

func (r *stubFuhrparkRepo) DeleteTripLog(_ context.Context, _, _ uuid.UUID) error { return r.err }

func (r *stubFuhrparkRepo) ListTripLogsForExport(_ context.Context, _ fuhrpark.ExportTripLogsParams) ([]fuhrpark.TripLog, error) {
	return nil, r.err
}

// --- Vehicle documents ---

func (r *stubFuhrparkRepo) ListVehicleDocuments(_ context.Context, _ fuhrpark.ListVehicleDocumentsParams) ([]fuhrpark.VehicleDocument, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

func (r *stubFuhrparkRepo) CreateVehicleDocument(_ context.Context, doc fuhrpark.VehicleDocument) (fuhrpark.VehicleDocument, error) {
	if r.err != nil {
		return fuhrpark.VehicleDocument{}, r.err
	}
	doc.ID = uuid.New()
	return doc, nil
}

func (r *stubFuhrparkRepo) DeleteVehicleDocument(_ context.Context, _, _ uuid.UUID) error {
	return r.err
}

// --- Driver licenses ---

func (r *stubFuhrparkRepo) ListDriverLicenses(_ context.Context, _ fuhrpark.ListDriverLicensesParams) ([]fuhrpark.DriverLicense, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

func (r *stubFuhrparkRepo) CreateDriverLicense(_ context.Context, lic fuhrpark.DriverLicense) (fuhrpark.DriverLicense, error) {
	if r.err != nil {
		return fuhrpark.DriverLicense{}, r.err
	}
	lic.ID = uuid.New()
	return lic, nil
}

func (r *stubFuhrparkRepo) UpdateDriverLicense(_ context.Context, lic fuhrpark.DriverLicense) (fuhrpark.DriverLicense, error) {
	if r.err != nil {
		return fuhrpark.DriverLicense{}, r.err
	}
	return lic, nil
}

func (r *stubFuhrparkRepo) DeleteDriverLicense(_ context.Context, _, _ uuid.UUID) error {
	return r.err
}

// --- Bookings ---

func (r *stubFuhrparkRepo) CreateBookingWithLock(_ context.Context, b *fuhrpark.VehicleBooking) (*uuid.UUID, error) {
	r.mu.Lock()
	r.lastBookingCreatedBy = b.CreatedBy
	r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	b.ID = uuid.New()
	b.Status = fuhrpark.BookingStatusBooked
	return nil, nil
}

func (r *stubFuhrparkRepo) UpdateBooking(_ context.Context, _ *fuhrpark.VehicleBooking) error {
	return r.err
}

func (r *stubFuhrparkRepo) DeleteBooking(_ context.Context, _, _ uuid.UUID) error { return r.err }

func (r *stubFuhrparkRepo) GetBooking(_ context.Context, tenantID, bookingID uuid.UUID) (*fuhrpark.VehicleBooking, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &fuhrpark.VehicleBooking{
		ID: bookingID, TenantID: tenantID, VehicleID: uuid.New(), UserID: uuid.New(),
		StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour), Status: fuhrpark.BookingStatusBooked,
	}, nil
}

func (r *stubFuhrparkRepo) ListBookings(_ context.Context, _ fuhrpark.ListVehicleBookingsParams) ([]*fuhrpark.VehicleBooking, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

func (r *stubFuhrparkRepo) FindConflictingBooking(_ context.Context, _, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*uuid.UUID, error) {
	return nil, r.err
}

// --- GPS ---

func (r *stubFuhrparkRepo) IngestGpsPositions(_ context.Context, tenantID, _ uuid.UUID, positions []fuhrpark.GpsPosition) (int, error) {
	r.mu.Lock()
	r.lastIngestTenantID = tenantID
	r.mu.Unlock()
	if r.err != nil {
		return 0, r.err
	}
	return len(positions), nil
}

func (r *stubFuhrparkRepo) GetVehicleRoutes(_ context.Context, params fuhrpark.GetVehicleRoutesParams) ([]fuhrpark.VehicleRouteAggregation, error) {
	r.mu.Lock()
	r.lastRoutesParams = params
	r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	return r.routes, nil
}

func (r *stubFuhrparkRepo) GetGpsPositions(_ context.Context, params fuhrpark.GetGpsPositionsParams) ([]fuhrpark.GpsPosition, error) {
	r.mu.Lock()
	r.lastGpsParams = params
	r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	return r.gpsPositions, nil
}

// --- TUEV cron (worker-only; not reachable from any gRPC handler under test) ---

func (r *stubFuhrparkRepo) FindVehiclesDueTuev(_ context.Context, _, _ time.Time) ([]*fuhrpark.Vehicle, error) {
	return nil, nil
}

func (r *stubFuhrparkRepo) MarkTuevReminderSent(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// ============================================================================
// Test helpers
// ============================================================================

func newFuhrparkTestServer(repo fuhrpark.Repository) *FuhrparkGRPCServer {
	return NewFuhrparkGRPCServer(fuhrpark.NewService(repo))
}

func fuhrparkCtxWithTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

func fuhrparkCtxWithTenantAndUser(tenantID, userID uuid.UUID) context.Context {
	ctx := context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
	return context.WithValue(ctx, middleware.UserIDKey, userID.String())
}

// ============================================================================
// Error mapping — table test
// ============================================================================

func TestMapFuhrparkError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"vehicle not found", fuhrpark.ErrVehicleNotFound, codes.NotFound},
		{"service not found", fuhrpark.ErrServiceNotFound, codes.NotFound},
		{"damage not found", fuhrpark.ErrDamageNotFound, codes.NotFound},
		{"fuel log not found", fuhrpark.ErrFuelLogNotFound, codes.NotFound},
		{"trip log not found", fuhrpark.ErrTripLogNotFound, codes.NotFound},
		{"document not found", fuhrpark.ErrDocumentNotFound, codes.NotFound},
		{"driver license not found", fuhrpark.ErrDriverLicenseNotFound, codes.NotFound},
		{"driver not found", fuhrpark.ErrDriverNotFound, codes.NotFound},
		{"booking not found", fuhrpark.ErrBookingNotFound, codes.NotFound},
		{"plate taken", fuhrpark.ErrPlateTaken, codes.AlreadyExists},
		{"booking conflict", fuhrpark.ErrBookingConflict, codes.AlreadyExists},
		{"invalid input", fuhrpark.ErrInvalidInput, codes.InvalidArgument},
		{"invalid transition", fuhrpark.ErrInvalidTransition, codes.FailedPrecondition},
		{"unmapped error falls back to internal", errors.New("boom"), codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapFuhrparkError(tc.err), tc.want)
		})
	}
}

// ============================================================================
// Vehicle RPCs
// ============================================================================

func TestFuhrparkVehicleHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	vehicleID := uuid.New()

	t.Run("CreateVehicle invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateVehicle(ctx, &fuhrparkv1.CreateVehicleRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateVehicle invalid tuev_due_date", func(t *testing.T) {
		bad := "not-a-date"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateVehicle(ctx, &fuhrparkv1.CreateVehicleRequest{TenantId: tenantID.String(), TuevDueDate: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateVehicle invalid assigned_driver_id", func(t *testing.T) {
		bad := "not-a-uuid"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateVehicle(ctx, &fuhrparkv1.CreateVehicleRequest{TenantId: tenantID.String(), AssignedDriverId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateVehicle happy path echoes tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.CreateVehicle(ctx, &fuhrparkv1.CreateVehicleRequest{
			TenantId: tenantID.String(), LicensePlate: "AB-1", Make: "VW", Model: "Golf", Year: 2020, FuelType: "petrol",
		})
		requireGRPCOK(t, err)
		require.Equal(t, tenantID.String(), resp.Vehicle.TenantId)
	})

	t.Run("CreateVehicle repo error maps through", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		repo.err = fuhrpark.ErrPlateTaken
		s := newFuhrparkTestServer(repo)
		_, err := s.CreateVehicle(ctx, &fuhrparkv1.CreateVehicleRequest{
			TenantId: tenantID.String(), LicensePlate: "AB-1", Make: "VW", Model: "Golf", Year: 2020,
		})
		requireGRPCCode(t, err, codes.AlreadyExists)
	})

	t.Run("UpdateVehicle invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateVehicle(ctx, &fuhrparkv1.UpdateVehicleRequest{TenantId: "bad", VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateVehicle invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateVehicle(ctx, &fuhrparkv1.UpdateVehicleRequest{TenantId: tenantID.String(), VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateVehicle invalid tuev_due_date", func(t *testing.T) {
		bad := "nope"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateVehicle(ctx, &fuhrparkv1.UpdateVehicleRequest{TenantId: tenantID.String(), VehicleId: vehicleID.String(), TuevDueDate: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateVehicle invalid assigned_driver_id", func(t *testing.T) {
		bad := "nope"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateVehicle(ctx, &fuhrparkv1.UpdateVehicleRequest{TenantId: tenantID.String(), VehicleId: vehicleID.String(), AssignedDriverId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteVehicle invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteVehicle(ctx, &fuhrparkv1.DeleteVehicleRequest{TenantId: "bad", VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteVehicle invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteVehicle(ctx, &fuhrparkv1.DeleteVehicleRequest{TenantId: tenantID.String(), VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteVehicle not found maps through", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		repo.err = fuhrpark.ErrVehicleNotFound
		s := newFuhrparkTestServer(repo)
		_, err := s.DeleteVehicle(ctx, &fuhrparkv1.DeleteVehicleRequest{TenantId: tenantID.String(), VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("GetVehicle invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.GetVehicle(ctx, &fuhrparkv1.GetVehicleRequest{TenantId: "bad", VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetVehicle invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.GetVehicle(ctx, &fuhrparkv1.GetVehicleRequest{TenantId: tenantID.String(), VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetVehicle happy path echoes ids", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.GetVehicle(ctx, &fuhrparkv1.GetVehicleRequest{TenantId: tenantID.String(), VehicleId: vehicleID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, vehicleID.String(), resp.Vehicle.Id)
	})

	t.Run("ListVehicles invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListVehicles(ctx, &fuhrparkv1.ListVehiclesRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListVehicles empty result wraps as empty slice not null", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.ListVehicles(ctx, &fuhrparkv1.ListVehiclesRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Vehicles)
		require.Empty(t, resp.Vehicles)
	})
}

// ============================================================================
// Service RPCs
// ============================================================================

func TestFuhrparkServiceHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	vehicleID := uuid.New()
	serviceID := uuid.New()

	t.Run("ScheduleService invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ScheduleService(ctx, &fuhrparkv1.ScheduleServiceRequest{TenantId: "bad", VehicleId: vehicleID.String(), ScheduledAt: "2026-01-01"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ScheduleService invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ScheduleService(ctx, &fuhrparkv1.ScheduleServiceRequest{TenantId: tenantID.String(), VehicleId: "bad", ScheduledAt: "2026-01-01"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ScheduleService invalid scheduled_at", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ScheduleService(ctx, &fuhrparkv1.ScheduleServiceRequest{TenantId: tenantID.String(), VehicleId: vehicleID.String(), ScheduledAt: "not-a-date"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ScheduleService invalid created_by", func(t *testing.T) {
		bad := "nope"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ScheduleService(ctx, &fuhrparkv1.ScheduleServiceRequest{
			TenantId: tenantID.String(), VehicleId: vehicleID.String(), ScheduledAt: "2026-01-01", CreatedBy: &bad,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateService invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateService(ctx, &fuhrparkv1.UpdateServiceRequest{TenantId: "bad", ServiceId: serviceID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateService invalid service_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateService(ctx, &fuhrparkv1.UpdateServiceRequest{TenantId: tenantID.String(), ServiceId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateService invalid scheduled_at", func(t *testing.T) {
		bad := "nope"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateService(ctx, &fuhrparkv1.UpdateServiceRequest{TenantId: tenantID.String(), ServiceId: serviceID.String(), ScheduledAt: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteService invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteService(ctx, &fuhrparkv1.DeleteServiceRequest{TenantId: "bad", ServiceId: serviceID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteService invalid service_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteService(ctx, &fuhrparkv1.DeleteServiceRequest{TenantId: tenantID.String(), ServiceId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CompleteService invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CompleteService(ctx, &fuhrparkv1.CompleteServiceRequest{TenantId: "bad", ServiceId: serviceID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CompleteService invalid service_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CompleteService(ctx, &fuhrparkv1.CompleteServiceRequest{TenantId: tenantID.String(), ServiceId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CompleteService not found maps through", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		repo.err = fuhrpark.ErrServiceNotFound
		s := newFuhrparkTestServer(repo)
		_, err := s.CompleteService(ctx, &fuhrparkv1.CompleteServiceRequest{TenantId: tenantID.String(), ServiceId: serviceID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("ListServices invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListServices(ctx, &fuhrparkv1.ListServicesRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListServices invalid vehicle_id filter", func(t *testing.T) {
		bad := "nope"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListServices(ctx, &fuhrparkv1.ListServicesRequest{TenantId: tenantID.String(), VehicleId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListServices empty result wraps as empty slice not null", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.ListServices(ctx, &fuhrparkv1.ListServicesRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Services)
		require.Empty(t, resp.Services)
	})

	t.Run("ScheduleService happy path maps the created entry to proto", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.ScheduleService(ctx, &fuhrparkv1.ScheduleServiceRequest{
			TenantId: tenantID.String(), VehicleId: vehicleID.String(), ServiceType: "inspection", ScheduledAt: "2026-01-01",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "inspection", resp.Service.ServiceType)
		require.Equal(t, string(fuhrpark.ServiceStatusScheduled), resp.Service.Status)
	})
}

// ============================================================================
// Damage RPCs
// ============================================================================

func TestFuhrparkDamageHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	vehicleID := uuid.New()
	damageID := uuid.New()

	t.Run("ReportDamage invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ReportDamage(ctx, &fuhrparkv1.ReportDamageRequest{TenantId: "bad", VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ReportDamage invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ReportDamage(ctx, &fuhrparkv1.ReportDamageRequest{TenantId: tenantID.String(), VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ReportDamage invalid reported_by", func(t *testing.T) {
		bad := "nope"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ReportDamage(ctx, &fuhrparkv1.ReportDamageRequest{
			TenantId: tenantID.String(), VehicleId: vehicleID.String(), ReportedBy: &bad,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateDamage invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateDamage(ctx, &fuhrparkv1.UpdateDamageRequest{TenantId: "bad", DamageId: damageID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateDamage invalid damage_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateDamage(ctx, &fuhrparkv1.UpdateDamageRequest{TenantId: tenantID.String(), DamageId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ResolveDamage invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ResolveDamage(ctx, &fuhrparkv1.ResolveDamageRequest{TenantId: "bad", DamageId: damageID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ResolveDamage invalid damage_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ResolveDamage(ctx, &fuhrparkv1.ResolveDamageRequest{TenantId: tenantID.String(), DamageId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ResolveDamage invalid resolved_by", func(t *testing.T) {
		bad := "nope"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ResolveDamage(ctx, &fuhrparkv1.ResolveDamageRequest{
			TenantId: tenantID.String(), DamageId: damageID.String(), ResolvedBy: &bad,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListDamages invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListDamages(ctx, &fuhrparkv1.ListDamagesRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListDamages invalid vehicle_id filter", func(t *testing.T) {
		bad := "nope"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListDamages(ctx, &fuhrparkv1.ListDamagesRequest{TenantId: tenantID.String(), VehicleId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListDamages empty result wraps as empty slice not null", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.ListDamages(ctx, &fuhrparkv1.ListDamagesRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Damages)
		require.Empty(t, resp.Damages)
	})

	t.Run("ReportDamage happy path maps the created entry to proto", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.ReportDamage(ctx, &fuhrparkv1.ReportDamageRequest{
			TenantId: tenantID.String(), VehicleId: vehicleID.String(), Description: "scratch", Severity: "minor",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "scratch", resp.Damage.Description)
		require.Equal(t, string(fuhrpark.DamageStatusReported), resp.Damage.Status)
	})
}

// ============================================================================
// History & Report RPCs
// ============================================================================

func TestFuhrparkHistoryAndReportHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	vehicleID := uuid.New()

	t.Run("GetVehicleHistory invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.GetVehicleHistory(ctx, &fuhrparkv1.GetVehicleHistoryRequest{TenantId: "bad", VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetVehicleHistory invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.GetVehicleHistory(ctx, &fuhrparkv1.GetVehicleHistoryRequest{TenantId: tenantID.String(), VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CheckTuevDue invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CheckTuevDue(ctx, &fuhrparkv1.CheckTuevDueRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CheckTuevDue zero days_ahead defaults instead of erroring", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.CheckTuevDue(ctx, &fuhrparkv1.CheckTuevDueRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Vehicles)
	})

	t.Run("ListUpcomingServices invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListUpcomingServices(ctx, &fuhrparkv1.ListUpcomingServicesRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ExportVehicleReport invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ExportVehicleReport(ctx, &fuhrparkv1.ExportVehicleReportRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ExportVehicleReport happy path returns csv with header", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.ExportVehicleReport(ctx, &fuhrparkv1.ExportVehicleReportRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, "text/csv; charset=utf-8", resp.ContentType)
		require.Contains(t, string(resp.Payload), "License Plate")
	})
}

// ============================================================================
// Fuel Log RPCs (tenant comes from the authenticated context, never the body)
// ============================================================================

func TestFuhrparkFuelLogHandlers(t *testing.T) {
	tenantID := uuid.New()
	vehicleID := uuid.New()
	logID := uuid.New()

	t.Run("ListFuelLogs missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListFuelLogs(context.Background(), &fuhrparkv1.ListFuelLogsRequest{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("ListFuelLogs invalid vehicle_id filter", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListFuelLogs(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.ListFuelLogsRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateFuelLog missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateFuelLog(context.Background(), &fuhrparkv1.CreateFuelLogRequest{VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("CreateFuelLog invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateFuelLog(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateFuelLogRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateFuelLog missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateFuelLog(context.Background(), &fuhrparkv1.UpdateFuelLogRequest{Id: logID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("UpdateFuelLog invalid id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateFuelLog(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.UpdateFuelLogRequest{Id: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteFuelLog missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteFuelLog(context.Background(), &fuhrparkv1.DeleteFuelLogRequest{Id: logID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("DeleteFuelLog invalid id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteFuelLog(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.DeleteFuelLogRequest{Id: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteFuelLog not found maps through", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		repo.err = fuhrpark.ErrFuelLogNotFound
		s := newFuhrparkTestServer(repo)
		_, err := s.DeleteFuelLog(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.DeleteFuelLogRequest{Id: logID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("CreateFuelLog happy path maps the created entry to proto", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.CreateFuelLog(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateFuelLogRequest{
			VehicleId: vehicleID.String(), Date: "2026-01-01", Liters: 40, FuelType: "petrol",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "petrol", resp.FuelLog.FuelType)
		require.Equal(t, 40.0, resp.FuelLog.Liters)
	})
}

// ============================================================================
// Trip Log RPCs
// ============================================================================

func TestFuhrparkTripLogHandlers(t *testing.T) {
	tenantID := uuid.New()
	vehicleID := uuid.New()
	logID := uuid.New()

	t.Run("ListTripLogs missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListTripLogs(context.Background(), &fuhrparkv1.ListTripLogsRequest{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("ListTripLogs invalid vehicle_id filter", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListTripLogs(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.ListTripLogsRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateTripLog missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateTripLog(context.Background(), &fuhrparkv1.CreateTripLogRequest{VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("CreateTripLog invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateTripLog(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateTripLogRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateTripLog missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateTripLog(context.Background(), &fuhrparkv1.UpdateTripLogRequest{Id: logID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("UpdateTripLog invalid id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateTripLog(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.UpdateTripLogRequest{Id: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteTripLog missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteTripLog(context.Background(), &fuhrparkv1.DeleteTripLogRequest{Id: logID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("DeleteTripLog invalid id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteTripLog(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.DeleteTripLogRequest{Id: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ExportTripLogs invalid tenant", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ExportTripLogs(context.Background(), &fuhrparkv1.ExportTripLogsRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ExportTripLogs invalid vehicle_id filter", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ExportTripLogs(context.Background(), &fuhrparkv1.ExportTripLogsRequest{TenantId: tenantID.String(), VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ExportTripLogs invalid from date", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ExportTripLogs(context.Background(), &fuhrparkv1.ExportTripLogsRequest{TenantId: tenantID.String(), From: "not-a-date"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ExportTripLogs invalid to date", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ExportTripLogs(context.Background(), &fuhrparkv1.ExportTripLogsRequest{TenantId: tenantID.String(), To: "not-a-date"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateTripLog happy path maps the created entry to proto", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.CreateTripLog(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateTripLogRequest{
			VehicleId: vehicleID.String(), Date: "2026-01-01", StartLocation: "A", EndLocation: "B", Purpose: "business",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "A", resp.TripLog.StartLocation)
		require.Equal(t, "business", resp.TripLog.Purpose)
	})
}

// ============================================================================
// Vehicle Booking RPCs
// ============================================================================

func TestFuhrparkBookingHandlers(t *testing.T) {
	tenantID := uuid.New()
	vehicleID := uuid.New()
	userID := uuid.New()
	bookingID := uuid.New()

	t.Run("ListVehicleBookings missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListVehicleBookings(context.Background(), &fuhrparkv1.ListVehicleBookingsRequest{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("ListVehicleBookings invalid vehicle_id filter", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListVehicleBookings(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.ListVehicleBookingsRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListVehicleBookings invalid user_id filter", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListVehicleBookings(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.ListVehicleBookingsRequest{UserId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListVehicleBookings invalid from window", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListVehicleBookings(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.ListVehicleBookingsRequest{From: "not-rfc3339"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListVehicleBookings invalid to window", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListVehicleBookings(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.ListVehicleBookingsRequest{To: "not-rfc3339"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateVehicleBooking missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateVehicleBooking(context.Background(), &fuhrparkv1.CreateVehicleBookingRequest{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("CreateVehicleBooking invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateVehicleBooking(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateVehicleBookingRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateVehicleBooking invalid user_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateVehicleBooking(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateVehicleBookingRequest{
			VehicleId: vehicleID.String(), UserId: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateVehicleBooking invalid starts_at", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateVehicleBooking(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateVehicleBookingRequest{
			VehicleId: vehicleID.String(), UserId: userID.String(), StartsAt: "not-rfc3339",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateVehicleBooking invalid ends_at", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateVehicleBooking(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateVehicleBookingRequest{
			VehicleId: vehicleID.String(), UserId: userID.String(),
			StartsAt: time.Now().Format(time.RFC3339), EndsAt: "not-rfc3339",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateVehicleBooking takes the creator from the authenticated context, not the body", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		s := newFuhrparkTestServer(repo)
		callerID := uuid.New()
		start := time.Now().Add(time.Hour)
		end := start.Add(time.Hour)
		_, err := s.CreateVehicleBooking(fuhrparkCtxWithTenantAndUser(tenantID, callerID), &fuhrparkv1.CreateVehicleBookingRequest{
			VehicleId: vehicleID.String(), UserId: userID.String(),
			StartsAt: start.Format(time.RFC3339), EndsAt: end.Format(time.RFC3339),
		})
		requireGRPCOK(t, err)
		require.NotNil(t, repo.lastBookingCreatedBy)
		require.Equal(t, callerID, *repo.lastBookingCreatedBy)
	})

	t.Run("UpdateVehicleBooking missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateVehicleBooking(context.Background(), &fuhrparkv1.UpdateVehicleBookingRequest{Id: bookingID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("UpdateVehicleBooking invalid id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateVehicleBooking(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.UpdateVehicleBookingRequest{Id: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateVehicleBooking invalid starts_at", func(t *testing.T) {
		bad := "not-rfc3339"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateVehicleBooking(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.UpdateVehicleBookingRequest{Id: bookingID.String(), StartsAt: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateVehicleBooking invalid ends_at", func(t *testing.T) {
		bad := "not-rfc3339"
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateVehicleBooking(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.UpdateVehicleBookingRequest{Id: bookingID.String(), EndsAt: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteVehicleBooking missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteVehicleBooking(context.Background(), &fuhrparkv1.DeleteVehicleBookingRequest{Id: bookingID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("DeleteVehicleBooking invalid id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteVehicleBooking(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.DeleteVehicleBookingRequest{Id: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteVehicleBooking conflict maps to already_exists", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		repo.err = fuhrpark.ErrBookingConflict
		s := newFuhrparkTestServer(repo)
		_, err := s.DeleteVehicleBooking(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.DeleteVehicleBookingRequest{Id: bookingID.String()})
		requireGRPCCode(t, err, codes.AlreadyExists)
	})
}

// ============================================================================
// Vehicle Document RPCs
// ============================================================================

func TestFuhrparkVehicleDocumentHandlers(t *testing.T) {
	tenantID := uuid.New()
	vehicleID := uuid.New()
	docID := uuid.New()

	t.Run("ListVehicleDocuments missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListVehicleDocuments(context.Background(), &fuhrparkv1.ListVehicleDocumentsRequest{VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("ListVehicleDocuments invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListVehicleDocuments(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.ListVehicleDocumentsRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateVehicleDocument missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateVehicleDocument(context.Background(), &fuhrparkv1.CreateVehicleDocumentRequest{VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("CreateVehicleDocument invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateVehicleDocument(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateVehicleDocumentRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteVehicleDocument missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteVehicleDocument(context.Background(), &fuhrparkv1.DeleteVehicleDocumentRequest{Id: docID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("DeleteVehicleDocument invalid id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteVehicleDocument(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.DeleteVehicleDocumentRequest{Id: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateVehicleDocument happy path maps the created entry to proto", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.CreateVehicleDocument(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateVehicleDocumentRequest{
			VehicleId: vehicleID.String(), DocType: "registration", Name: "Fahrzeugschein", ObjectKey: "docs/1.pdf",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "Fahrzeugschein", resp.Document.Name)
		require.Equal(t, "docs/1.pdf", resp.Document.ObjectKey)
	})
}

// ============================================================================
// Driver License RPCs
// ============================================================================

func TestFuhrparkDriverLicenseHandlers(t *testing.T) {
	tenantID := uuid.New()
	driverID := uuid.New()
	licenseID := uuid.New()

	t.Run("ListDriverLicenses missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListDriverLicenses(context.Background(), &fuhrparkv1.ListDriverLicensesRequest{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("ListDriverLicenses invalid driver_id filter", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.ListDriverLicenses(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.ListDriverLicensesRequest{DriverId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateDriverLicense missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateDriverLicense(context.Background(), &fuhrparkv1.CreateDriverLicenseRequest{DriverId: driverID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("CreateDriverLicense invalid driver_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateDriverLicense(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateDriverLicenseRequest{
			DriverId: "bad", NextCheckDueDate: "2026-01-01",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateDriverLicense invalid checked_at", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateDriverLicense(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateDriverLicenseRequest{
			DriverId: driverID.String(), NextCheckDueDate: "2026-01-01", CheckedAt: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateDriverLicense invalid next_check_due_date", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.CreateDriverLicense(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateDriverLicenseRequest{
			DriverId: driverID.String(), NextCheckDueDate: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateDriverLicense missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateDriverLicense(context.Background(), &fuhrparkv1.UpdateDriverLicenseRequest{Id: licenseID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("UpdateDriverLicense invalid id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateDriverLicense(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.UpdateDriverLicenseRequest{
			Id: "bad", NextCheckDueDate: "2026-01-01",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateDriverLicense invalid next_check_due_date", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateDriverLicense(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.UpdateDriverLicenseRequest{
			Id: licenseID.String(), NextCheckDueDate: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateDriverLicense invalid checked_at", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateDriverLicense(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.UpdateDriverLicenseRequest{
			Id: licenseID.String(), NextCheckDueDate: "2026-01-01", CheckedAt: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteDriverLicense missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteDriverLicense(context.Background(), &fuhrparkv1.DeleteDriverLicenseRequest{Id: licenseID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("DeleteDriverLicense invalid id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteDriverLicense(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.DeleteDriverLicenseRequest{Id: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateDriverLicense happy path maps the created entry to proto", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.CreateDriverLicense(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.CreateDriverLicenseRequest{
			DriverId: driverID.String(), LicenseClasses: []string{"B"}, NextCheckDueDate: "2027-01-01",
		})
		requireGRPCOK(t, err)
		require.Equal(t, []string{"B"}, resp.License.LicenseClasses)
	})
}

// ============================================================================
// GPS RPCs — GPS positions are personal movement data, so the tenant-scoping
// case matters as much as the validation cases: every read/write must use
// the tenant stamped by auth middleware, never a client-suppliable value.
// ============================================================================

func TestFuhrparkGpsHandlers(t *testing.T) {
	tenantID := uuid.New()
	vehicleID := uuid.New()

	t.Run("IngestGpsPositions missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.IngestGpsPositions(context.Background(), &fuhrparkv1.IngestGpsPositionsRequest{VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("IngestGpsPositions invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.IngestGpsPositions(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.IngestGpsPositionsRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("IngestGpsPositions scopes the write to the context tenant, not a client-suppliable value", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		s := newFuhrparkTestServer(repo)
		resp, err := s.IngestGpsPositions(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.IngestGpsPositionsRequest{
			VehicleId: vehicleID.String(),
			Positions: []*fuhrparkv1.GpsPosition{{Lat: 47.0, Lng: 8.0, RecordedAt: time.Now().Format(time.RFC3339)}},
		})
		requireGRPCOK(t, err)
		require.Equal(t, int32(1), resp.Ingested)
		require.Equal(t, tenantID, repo.lastIngestTenantID)
	})

	t.Run("GetVehicleRoutes missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.GetVehicleRoutes(context.Background(), &fuhrparkv1.GetVehicleRoutesRequest{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("GetVehicleRoutes invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.GetVehicleRoutes(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.GetVehicleRoutesRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetVehicleRoutes defaults to a 7-day window and the context tenant when unset", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		s := newFuhrparkTestServer(repo)
		_, err := s.GetVehicleRoutes(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.GetVehicleRoutesRequest{})
		requireGRPCOK(t, err)
		require.Equal(t, tenantID, repo.lastRoutesParams.TenantID)
		require.False(t, repo.lastRoutesParams.DateFrom.IsZero())
		require.False(t, repo.lastRoutesParams.DateTo.IsZero())
		require.WithinDuration(t, repo.lastRoutesParams.DateTo.AddDate(0, 0, -7), repo.lastRoutesParams.DateFrom, time.Second)
	})

	t.Run("GetGpsPositions missing tenant is unauthenticated", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.GetGpsPositions(context.Background(), &fuhrparkv1.GetGpsPositionsRequest{VehicleId: vehicleID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("GetGpsPositions invalid vehicle_id", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.GetGpsPositions(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.GetGpsPositionsRequest{VehicleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetGpsPositions defaults to a 24h window and scopes the read to the context tenant", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		s := newFuhrparkTestServer(repo)
		_, err := s.GetGpsPositions(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.GetGpsPositionsRequest{VehicleId: vehicleID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, tenantID, repo.lastGpsParams.TenantID)
		require.Equal(t, vehicleID, repo.lastGpsParams.VehicleID)
		require.WithinDuration(t, repo.lastGpsParams.To.Add(-24*time.Hour), repo.lastGpsParams.From, time.Second)
	})

	t.Run("GetGpsPositions empty result wraps as empty slice not null", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		resp, err := s.GetGpsPositions(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.GetGpsPositionsRequest{VehicleId: vehicleID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Positions)
		require.Empty(t, resp.Positions)
	})

	t.Run("GetGpsPositions happy path maps positions to proto", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		speed := 42.0
		repo.gpsPositions = []fuhrpark.GpsPosition{{ID: uuid.New(), VehicleID: vehicleID, Lat: 47.1, Lng: 8.5, SpeedKmh: &speed, RecordedAt: time.Now()}}
		s := newFuhrparkTestServer(repo)
		resp, err := s.GetGpsPositions(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.GetGpsPositionsRequest{VehicleId: vehicleID.String()})
		requireGRPCOK(t, err)
		require.Len(t, resp.Positions, 1)
		require.Equal(t, 47.1, resp.Positions[0].Lat)
		require.Equal(t, 42.0, resp.Positions[0].SpeedKmh)
	})

	t.Run("GetVehicleRoutes happy path maps routes and their positions to proto", func(t *testing.T) {
		repo := newStubFuhrparkRepo()
		repo.routes = []fuhrpark.VehicleRouteAggregation{{
			VehicleID: vehicleID, VehicleName: "AB-123", Date: time.Now(), DailyKm: 12.5, Status: "active",
			Positions: []fuhrpark.GpsPosition{{ID: uuid.New(), VehicleID: vehicleID, Lat: 47.1, Lng: 8.5, RecordedAt: time.Now()}},
		}}
		s := newFuhrparkTestServer(repo)
		resp, err := s.GetVehicleRoutes(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.GetVehicleRoutesRequest{})
		requireGRPCOK(t, err)
		require.Len(t, resp.Routes, 1)
		require.Equal(t, "AB-123", resp.Routes[0].VehicleName)
		require.Len(t, resp.Routes[0].Positions, 1)
	})
}
