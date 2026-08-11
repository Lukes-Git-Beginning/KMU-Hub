package fuhrpark

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// FuelLog / TripLog / VehicleDocument / DriverLicense / GPS — validation and
// happy-path coverage for the service layer. These methods are currently only
// exercised at the repository layer (tenant_write_test.go); the validation
// and delegation logic in service.go itself was uncovered.
// ============================================================================

func TestService_CreateFuelLog_InvalidInput(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()
	vehicleID := uuid.New()

	cases := map[string]FuelLog{
		"zero vehicle id": {TenantID: tenantID, VehicleID: uuid.UUID{}, Liters: 10, CostCents: 100, MileageKm: 5},
		"non-positive liters": {TenantID: tenantID, VehicleID: vehicleID, Liters: 0, CostCents: 100, MileageKm: 5},
		"negative cost": {TenantID: tenantID, VehicleID: vehicleID, Liters: 10, CostCents: -1, MileageKm: 5},
		"negative mileage": {TenantID: tenantID, VehicleID: vehicleID, Liters: 10, CostCents: 100, MileageKm: -1},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.CreateFuelLog(context.Background(), input)
			assert.ErrorIs(t, err, ErrInvalidInput)
		})
	}
}

func TestService_CreateFuelLog_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	got, err := svc.CreateFuelLog(context.Background(), FuelLog{
		VehicleID: uuid.New(), Liters: 42.5, CostCents: 6500, MileageKm: 1000,
	})
	require.NoError(t, err)
	assert.Equal(t, 42.5, got.Liters)
}

func TestService_UpdateFuelLog_InvalidInput(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateFuelLog(context.Background(), FuelLog{ID: uuid.UUID{}})
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_ListFuelLogs_Passthrough(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	logs, total, err := svc.ListFuelLogs(context.Background(), ListFuelLogsParams{TenantID: uuid.New()})
	require.NoError(t, err)
	assert.Nil(t, logs)
	assert.Equal(t, 0, total)
}

func TestService_CreateTripLog_InvalidInput(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	vehicleID := uuid.New()

	cases := map[string]TripLog{
		"zero vehicle id": {VehicleID: uuid.UUID{}, StartLocation: "A", EndLocation: "B", StartKm: 0, EndKm: 10},
		"empty start location": {VehicleID: vehicleID, StartLocation: "", EndLocation: "B", StartKm: 0, EndKm: 10},
		"empty end location": {VehicleID: vehicleID, StartLocation: "A", EndLocation: "", StartKm: 0, EndKm: 10},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.CreateTripLog(context.Background(), input)
			assert.ErrorIs(t, err, ErrInvalidInput)
		})
	}
}

func TestService_CreateTripLog_EndKmBeforeStartKm(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateTripLog(context.Background(), TripLog{
		VehicleID: uuid.New(), StartLocation: "A", EndLocation: "B", StartKm: 100, EndKm: 50,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateTripLog_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	got, err := svc.CreateTripLog(context.Background(), TripLog{
		VehicleID: uuid.New(), StartLocation: "A", EndLocation: "B", StartKm: 100, EndKm: 150,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(150), got.EndKm)
}

func TestService_UpdateTripLog_InvalidInput(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateTripLog(context.Background(), TripLog{ID: uuid.UUID{}})
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_ListTripLogs_Passthrough(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	logs, total, err := svc.ListTripLogs(context.Background(), ListTripLogsParams{TenantID: uuid.New()})
	require.NoError(t, err)
	assert.Nil(t, logs)
	assert.Equal(t, 0, total)
}

func TestService_CreateVehicleDocument_InvalidInput(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	vehicleID := uuid.New()

	cases := map[string]VehicleDocument{
		"zero vehicle id": {VehicleID: uuid.UUID{}, ObjectKey: "key", Name: "doc.pdf"},
		"empty object key": {VehicleID: vehicleID, ObjectKey: "", Name: "doc.pdf"},
		"empty name": {VehicleID: vehicleID, ObjectKey: "key", Name: ""},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.CreateVehicleDocument(context.Background(), input)
			assert.ErrorIs(t, err, ErrInvalidInput)
		})
	}
}

func TestService_CreateVehicleDocument_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	got, err := svc.CreateVehicleDocument(context.Background(), VehicleDocument{
		VehicleID: uuid.New(), ObjectKey: "objects/doc.pdf", Name: "doc.pdf",
	})
	require.NoError(t, err)
	assert.Equal(t, "doc.pdf", got.Name)
}

func TestService_ListVehicleDocuments_Passthrough(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	docs, total, err := svc.ListVehicleDocuments(context.Background(), ListVehicleDocumentsParams{TenantID: uuid.New()})
	require.NoError(t, err)
	assert.Nil(t, docs)
	assert.Equal(t, 0, total)
}

func TestService_CreateDriverLicense_InvalidInput(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	cases := map[string]DriverLicense{
		"zero driver id":         {DriverID: uuid.UUID{}, NextCheckDueDate: time.Now().AddDate(1, 0, 0)},
		"zero next check due date": {DriverID: uuid.New(), NextCheckDueDate: time.Time{}},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.CreateDriverLicense(context.Background(), input)
			assert.ErrorIs(t, err, ErrInvalidInput)
		})
	}
}

func TestService_CreateDriverLicense_DefaultsCheckedAt(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	got, err := svc.CreateDriverLicense(context.Background(), DriverLicense{
		DriverID: uuid.New(), NextCheckDueDate: time.Now().AddDate(1, 0, 0),
	})
	require.NoError(t, err)
	assert.False(t, got.CheckedAt.IsZero())
}

func TestService_UpdateDriverLicense_InvalidInput(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	cases := map[string]DriverLicense{
		"zero id":                   {ID: uuid.UUID{}, NextCheckDueDate: time.Now().AddDate(1, 0, 0)},
		"zero next check due date":  {ID: uuid.New(), NextCheckDueDate: time.Time{}},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.UpdateDriverLicense(context.Background(), input)
			assert.ErrorIs(t, err, ErrInvalidInput)
		})
	}
}

func TestService_UpdateDriverLicense_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	got, err := svc.UpdateDriverLicense(context.Background(), DriverLicense{
		ID: uuid.New(), NextCheckDueDate: time.Now().AddDate(1, 0, 0),
	})
	require.NoError(t, err)
	assert.False(t, got.NextCheckDueDate.IsZero())
}

func TestService_ListDriverLicenses_Passthrough(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	lics, total, err := svc.ListDriverLicenses(context.Background(), ListDriverLicensesParams{TenantID: uuid.New()})
	require.NoError(t, err)
	assert.Nil(t, lics)
	assert.Equal(t, 0, total)
}

func TestService_IngestGpsPositions_InvalidInput(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	vehicleID := uuid.New()

	_, err := svc.IngestGpsPositions(context.Background(), uuid.New(), uuid.UUID{}, []GpsPosition{{Lat: 1, Lng: 2}})
	assert.ErrorIs(t, err, ErrInvalidInput)

	_, err = svc.IngestGpsPositions(context.Background(), uuid.New(), vehicleID, nil)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_IngestGpsPositions_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	n, err := svc.IngestGpsPositions(context.Background(), uuid.New(), uuid.New(), []GpsPosition{{Lat: 1, Lng: 2}, {Lat: 3, Lng: 4}})
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

// ============================================================================
// TuevWorker + EventEmitter
// ============================================================================

// fakeEventEmitter records every payload it is asked to emit and optionally
// fails, so tests can assert both the payload shape and that a failed emit is
// non-fatal to the scan.
type fakeEventEmitter struct {
	mu       sync.Mutex
	payloads []models.EventPayload
	err      error
}

func (f *fakeEventEmitter) EmitTuevEvent(_ context.Context, payload models.EventPayload) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payloads = append(f.payloads, payload)
	return f.err
}

func (f *fakeEventEmitter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.payloads)
}

func TestBuildTuevEventPayload_PriorityByWindow(t *testing.T) {
	worker := NewTuevWorker(newMockRepository(), nil, nil)
	due := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	v := &Vehicle{
		ID: uuid.New(), TenantID: uuid.New(), Make: "VW", Model: "Golf",
		LicensePlate: "AB-CD-123", TuevDueDate: &due,
	}

	urgent := worker.buildTuevEventPayload(v, "1_day")
	assert.Equal(t, models.PriorityUrgent, urgent.Priority)

	normal := worker.buildTuevEventPayload(v, "7_days")
	assert.Equal(t, models.PriorityNormal, normal.Priority)

	assert.Equal(t, v.TenantID, urgent.TenantID)
	assert.Equal(t, v.ID.String(), urgent.ResourceID)
	assert.Contains(t, urgent.Title, v.LicensePlate)
}

func TestTuevWorker_WithEventEmitter_EmitsOnDueVehicle(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	due := now.AddDate(0, 0, 1)
	vid := uuid.New()
	repo.vehicles[vid] = &Vehicle{ID: vid, TenantID: tenantID, LicensePlate: "EMIT-1", TuevDueDate: &due}

	emitter := &fakeEventEmitter{}
	worker := NewTuevWorker(repo, nil, nil).WithEventEmitter(emitter)

	require.NoError(t, worker.ProcessTuevReminders(context.Background(), now))

	assert.Equal(t, 1, emitter.count())
	assert.Equal(t, models.PriorityUrgent, emitter.payloads[0].Priority)
	assert.NotNil(t, repo.vehicles[vid].TuevReminderSentAt)
}

func TestTuevWorker_WithEventEmitter_ErrorIsNonFatal(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	due := now.AddDate(0, 0, 7)
	vid := uuid.New()
	repo.vehicles[vid] = &Vehicle{ID: vid, TenantID: tenantID, LicensePlate: "EMIT-ERR", TuevDueDate: &due}

	emitter := &fakeEventEmitter{err: errors.New("pg_notify unavailable")}
	worker := NewTuevWorker(repo, nil, nil).WithEventEmitter(emitter)

	err := worker.ProcessTuevReminders(context.Background(), now)
	require.NoError(t, err, "a failed emit must not abort the scan")

	assert.Equal(t, 1, emitter.count())
	assert.NotNil(t, repo.vehicles[vid].TuevReminderSentAt, "reminder must still be marked sent despite the emit failure")
}
