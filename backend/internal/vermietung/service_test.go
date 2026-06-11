package vermietung

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// MockRepository
// ============================================================================

type mockRepository struct {
	objects     map[uuid.UUID]*RentalObject
	rentals     map[uuid.UUID]*Rental
	inspections map[uuid.UUID]*RentalInspection

	createObjectErr    error
	updateObjectErr    error
	getObjectErr       error
	createRentalErr    error
	updateRentalErr    error
	hasOverlapResult   bool
	hasOverlapErr      error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		objects:     make(map[uuid.UUID]*RentalObject),
		rentals:     make(map[uuid.UUID]*Rental),
		inspections: make(map[uuid.UUID]*RentalInspection),
	}
}

// Objects

func (m *mockRepository) CreateObject(ctx context.Context, obj *RentalObject) error {
	if m.createObjectErr != nil {
		return m.createObjectErr
	}
	m.objects[obj.ID] = obj
	return nil
}

func (m *mockRepository) UpdateObject(ctx context.Context, obj *RentalObject) error {
	if m.updateObjectErr != nil {
		return m.updateObjectErr
	}
	m.objects[obj.ID] = obj
	return nil
}

func (m *mockRepository) SoftDeleteObject(ctx context.Context, tenantID, objectID uuid.UUID) error {
	obj, ok := m.objects[objectID]
	if !ok || obj.TenantID != tenantID {
		return ErrObjectNotFound
	}
	now := time.Now()
	obj.DeletedAt = &now
	return nil
}

func (m *mockRepository) GetObject(ctx context.Context, tenantID, objectID uuid.UUID) (*RentalObject, error) {
	if m.getObjectErr != nil {
		return nil, m.getObjectErr
	}
	obj, ok := m.objects[objectID]
	if !ok || obj.TenantID != tenantID || obj.DeletedAt != nil {
		return nil, ErrObjectNotFound
	}
	return obj, nil
}

func (m *mockRepository) ListObjects(ctx context.Context, tenantID uuid.UUID, filter ListObjectsFilter, offset, limit int) ([]*RentalObject, int, error) {
	var result []*RentalObject
	for _, obj := range m.objects {
		if obj.TenantID != tenantID || obj.DeletedAt != nil {
			continue
		}
		if filter.ActiveOnly && !obj.Active {
			continue
		}
		result = append(result, obj)
	}
	total := len(result)
	if offset >= total {
		return []*RentalObject{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

// Rentals

func (m *mockRepository) CreateRental(ctx context.Context, rental *Rental) error {
	if m.createRentalErr != nil {
		return m.createRentalErr
	}
	m.rentals[rental.ID] = rental
	return nil
}

func (m *mockRepository) UpdateRental(ctx context.Context, rental *Rental) error {
	if m.updateRentalErr != nil {
		return m.updateRentalErr
	}
	_, ok := m.rentals[rental.ID]
	if !ok {
		return ErrRentalNotFound
	}
	m.rentals[rental.ID] = rental
	return nil
}

func (m *mockRepository) DeleteRental(ctx context.Context, tenantID, rentalID uuid.UUID) error {
	r, ok := m.rentals[rentalID]
	if !ok || r.TenantID != tenantID {
		return ErrRentalNotFound
	}
	delete(m.rentals, rentalID)
	return nil
}

func (m *mockRepository) GetRental(ctx context.Context, tenantID, rentalID uuid.UUID) (*Rental, error) {
	r, ok := m.rentals[rentalID]
	if !ok || r.TenantID != tenantID {
		return nil, ErrRentalNotFound
	}
	return r, nil
}

func (m *mockRepository) ListRentals(ctx context.Context, tenantID uuid.UUID, filter ListRentalsFilter, offset, limit int) ([]*Rental, int, error) {
	var result []*Rental
	for _, r := range m.rentals {
		if r.TenantID != tenantID {
			continue
		}
		if filter.ObjectID != nil && r.ObjectID != *filter.ObjectID {
			continue
		}
		if filter.Status != nil && r.Status != *filter.Status {
			continue
		}
		if filter.From != nil && r.EndDate.Before(*filter.From) {
			continue
		}
		if filter.To != nil && r.StartDate.After(*filter.To) {
			continue
		}
		result = append(result, r)
	}
	total := len(result)
	if offset >= total {
		return []*Rental{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

// HasOverlap uses in-memory date-range check (mirrors the GIST operator logic).
func (m *mockRepository) HasOverlap(ctx context.Context, tenantID, objectID uuid.UUID, start, end time.Time, excludeRentalID *uuid.UUID) (bool, error) {
	if m.hasOverlapErr != nil {
		return false, m.hasOverlapErr
	}
	if m.hasOverlapResult {
		// allow override from test
		return true, nil
	}
	for _, r := range m.rentals {
		if r.TenantID != tenantID || r.ObjectID != objectID {
			continue
		}
		if r.Status == RentalStatusCancelled {
			continue
		}
		if excludeRentalID != nil && r.ID == *excludeRentalID {
			continue
		}
		// tstzrange overlap: [r.Start, r.End) && [start, end)
		// Two ranges overlap iff start < r.End && end > r.Start
		if start.Before(r.EndDate) && end.After(r.StartDate) {
			return true, nil
		}
	}
	return false, nil
}

// Inspections

func (m *mockRepository) CreateInspection(ctx context.Context, ins *RentalInspection) error {
	m.inspections[ins.ID] = ins
	return nil
}

func (m *mockRepository) UpdateInspection(ctx context.Context, ins *RentalInspection) error {
	_, ok := m.inspections[ins.ID]
	if !ok {
		return ErrInspectionNotFound
	}
	m.inspections[ins.ID] = ins
	return nil
}

func (m *mockRepository) GetInspection(ctx context.Context, tenantID, inspectionID uuid.UUID) (*RentalInspection, error) {
	ins, ok := m.inspections[inspectionID]
	if !ok || ins.TenantID != tenantID {
		return nil, ErrInspectionNotFound
	}
	return ins, nil
}

func (m *mockRepository) ListInspections(ctx context.Context, tenantID, rentalID uuid.UUID, offset, limit int) ([]*RentalInspection, int, error) {
	var result []*RentalInspection
	for _, ins := range m.inspections {
		if ins.TenantID == tenantID && ins.RentalID == rentalID {
			result = append(result, ins)
		}
	}
	total := len(result)
	if offset >= total {
		return []*RentalInspection{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) GetInspectionByKind(ctx context.Context, tenantID, rentalID uuid.UUID, kind InspectionKind) (*RentalInspection, error) {
	for _, ins := range m.inspections {
		if ins.TenantID == tenantID && ins.RentalID == rentalID && ins.Kind == kind {
			return ins, nil
		}
	}
	return nil, ErrInspectionNotFound
}

func (m *mockRepository) SaveSignature(ctx context.Context, tenantID, rentalID, signatureData, signedBy string) (*Rental, error) {
	rentalUUID, err := uuid.Parse(rentalID)
	if err != nil {
		return nil, ErrRentalNotFound
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, ErrRentalNotFound
	}
	r, ok := m.rentals[rentalUUID]
	if !ok || r.TenantID != tenantUUID {
		return nil, ErrRentalNotFound
	}
	now := time.Now()
	r.SignatureData = &signatureData
	r.SignedAt = &now
	r.SignedBy = &signedBy
	r.UpdatedAt = now
	m.rentals[rentalUUID] = r
	return r, nil
}

// compile-time interface check
var _ Repository = (*mockRepository)(nil)

// ============================================================================
// Helpers
// ============================================================================

func addObject(repo *mockRepository, tenantID uuid.UUID, name string) *RentalObject {
	obj := &RentalObject{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		Category:  "Geraet",
		DailyRate: 50.00,
		Deposit:   100.00,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.objects[obj.ID] = obj
	return obj
}

func addRental(repo *mockRepository, tenantID, objectID uuid.UUID, start, end time.Time, status RentalStatus) *Rental {
	r := &Rental{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ObjectID:   objectID,
		RenterName: "Max Mustermann",
		StartDate:  start,
		EndDate:    end,
		Status:     status,
		TotalPrice: 100.00,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	repo.rentals[r.ID] = r
	return r
}

func may(year int, day int) time.Time {
	return time.Date(year, time.May, day, 0, 0, 0, 0, time.UTC)
}

// ============================================================================
// CreateObject Tests
// ============================================================================

func TestService_CreateObject_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj, err := svc.CreateObject(context.Background(), CreateObjectInput{
		TenantID:  tenantID,
		Name:      "Betonmischer BM-100",
		Category:  "Baugeraete",
		DailyRate: 75.00,
		Deposit:   200.00,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, obj.ID)
	assert.Equal(t, "Betonmischer BM-100", obj.Name)
	assert.Equal(t, tenantID, obj.TenantID)
	assert.True(t, obj.Active)
}

func TestService_CreateObject_EmptyName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateObject(context.Background(), CreateObjectInput{
		TenantID:  uuid.New(),
		Name:      "  ",
		DailyRate: 10.00,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateObject_NegativeRate(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateObject(context.Background(), CreateObjectInput{
		TenantID:  uuid.New(),
		Name:      "Objekt",
		DailyRate: -5.00,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// GetObject / Cross-Tenant Tests
// ============================================================================

func TestService_GetObject_CrossTenant_ReturnsNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()
	obj := addObject(repo, tenantA, "Gerüst")

	// Tenant B tries to read Tenant A's object
	_, err := svc.GetObject(context.Background(), tenantB, obj.ID)

	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestService_GetObject_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Hebebühne")

	result, err := svc.GetObject(context.Background(), tenantID, obj.ID)

	require.NoError(t, err)
	assert.Equal(t, obj.ID, result.ID)
}

// ============================================================================
// DeleteObject Tests
// ============================================================================

func TestService_DeleteObject_SoftDelete(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Kran")

	err := svc.DeleteObject(context.Background(), tenantID, obj.ID)

	require.NoError(t, err)
	assert.NotNil(t, repo.objects[obj.ID].DeletedAt)
}

func TestService_DeleteObject_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteObject(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrObjectNotFound)
}

// ============================================================================
// CreateRental — Date-Range Overlap Tests (CRITICAL PATH)
// ============================================================================

// Existing booking: 1.5.–10.5.  New booking: 5.5.–8.5. → CONFLICT
func TestService_CreateRental_Overlap_Inside_Conflict(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")

	// Existing rental: May 1 – May 10
	addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 10), RentalStatusReserved)

	// New rental: May 5 – May 8 → inside → conflict
	_, err := svc.CreateRental(context.Background(), CreateRentalInput{
		TenantID:   tenantID,
		ObjectID:   obj.ID,
		RenterName: "Karl Schmidt",
		StartDate:  may(2026, 5),
		EndDate:    may(2026, 8),
	})

	assert.ErrorIs(t, err, ErrRentalConflict)
}

// Existing booking: 1.5.–10.5.  New booking: 11.5.–15.5. → OK (adjacent, no overlap)
func TestService_CreateRental_Adjacent_NoConflict(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")

	// Existing rental: May 1 – May 10 (end exclusive: ends at midnight May 10)
	addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 10), RentalStatusReserved)

	// New rental: May 11 – May 15 → no overlap
	rental, err := svc.CreateRental(context.Background(), CreateRentalInput{
		TenantID:   tenantID,
		ObjectID:   obj.ID,
		RenterName: "Anna Müller",
		StartDate:  may(2026, 11),
		EndDate:    may(2026, 15),
	})

	require.NoError(t, err)
	assert.NotNil(t, rental)
}

// Existing booking: 1.5.–10.5.  New booking: 10.5.–12.5.
// End_date is exclusive, so May 10 (start of new) == May 10 (end of existing) → touching, not overlapping.
func TestService_CreateRental_TouchingEndDate_NoConflict(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")

	// Existing rental: May 1 – May 10 (end_date = May 10 00:00)
	addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 10), RentalStatusReserved)

	// New rental starts at May 10 00:00 — same as end_date of existing
	// With tstzrange [start, end): existing=[May1,May10), new=[May10,May12)
	// They touch but do NOT overlap.
	rental, err := svc.CreateRental(context.Background(), CreateRentalInput{
		TenantID:   tenantID,
		ObjectID:   obj.ID,
		RenterName: "Bernd Bauer",
		StartDate:  may(2026, 10),
		EndDate:    may(2026, 12),
	})

	require.NoError(t, err)
	assert.NotNil(t, rental)
}

// Cancelled rental does not block availability
func TestService_CreateRental_CancelledDoesNotBlock(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")

	addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 10), RentalStatusCancelled)

	rental, err := svc.CreateRental(context.Background(), CreateRentalInput{
		TenantID:   tenantID,
		ObjectID:   obj.ID,
		RenterName: "Lisa Huber",
		StartDate:  may(2026, 5),
		EndDate:    may(2026, 8),
	})

	require.NoError(t, err)
	assert.NotNil(t, rental)
}

func TestService_CreateRental_EmptyRenterName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")

	_, err := svc.CreateRental(context.Background(), CreateRentalInput{
		TenantID:   tenantID,
		ObjectID:   obj.ID,
		RenterName: "",
		StartDate:  may(2026, 1),
		EndDate:    may(2026, 5),
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateRental_InvalidDateRange(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")

	_, err := svc.CreateRental(context.Background(), CreateRentalInput{
		TenantID:   tenantID,
		ObjectID:   obj.ID,
		RenterName: "Test",
		StartDate:  may(2026, 10),
		EndDate:    may(2026, 5), // end before start
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateRental_ObjectNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateRental(context.Background(), CreateRentalInput{
		TenantID:   uuid.New(),
		ObjectID:   uuid.New(), // does not exist
		RenterName: "Test",
		StartDate:  may(2026, 1),
		EndDate:    may(2026, 5),
	})

	assert.ErrorIs(t, err, ErrObjectNotFound)
}

// ============================================================================
// State-Machine Tests
// ============================================================================

func TestService_StartRental_FromReserved_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusReserved)

	result, err := svc.StartRental(context.Background(), tenantID, rental.ID)

	require.NoError(t, err)
	assert.Equal(t, RentalStatusActive, result.Status)
}

func TestService_StartRental_FromActive_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	_, err := svc.StartRental(context.Background(), tenantID, rental.ID)

	assert.ErrorIs(t, err, ErrInvalidStateTransition)
}

// EndRental on reserved → ErrInvalidStateTransition
func TestService_EndRental_FromReserved_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusReserved)

	_, err := svc.EndRental(context.Background(), tenantID, rental.ID)

	assert.ErrorIs(t, err, ErrInvalidStateTransition)
}

func TestService_EndRental_FromActive_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	result, err := svc.EndRental(context.Background(), tenantID, rental.ID)

	require.NoError(t, err)
	assert.Equal(t, RentalStatusCompleted, result.Status)
}

// ============================================================================
// GetRental / Cross-Tenant Tests
// ============================================================================

func TestService_GetRental_CrossTenant_ReturnsNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()
	obj := addObject(repo, tenantA, "Gerüst")
	rental := addRental(repo, tenantA, obj.ID, may(2026, 1), may(2026, 5), RentalStatusReserved)

	// Tenant B tries to read Tenant A's rental
	_, err := svc.GetRental(context.Background(), tenantB, rental.ID)

	assert.ErrorIs(t, err, ErrRentalNotFound)
}

// ============================================================================
// Inspection Tests
// ============================================================================

func TestService_CreateInspection_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	ins, err := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID:  tenantID,
		RentalID:  rental.ID,
		Kind:      InspectionKindHandover,
		Notes:     "Fahrzeug in gutem Zustand",
		PhotoURLs: []string{"https://example.com/photo1.jpg"},
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, ins.ID)
	assert.Equal(t, InspectionKindHandover, ins.Kind)
	assert.Len(t, ins.PhotoURLs, 1)
}

func TestService_CreateInspection_InvalidKind(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	_, err := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID: tenantID,
		RentalID: rental.ID,
		Kind:     "invalid",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_UploadInspectionPhoto_AppendsURL(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	ins, _ := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID: tenantID,
		RentalID: rental.ID,
		Kind:     InspectionKindHandover,
	})

	updated, err := svc.UploadInspectionPhoto(context.Background(), tenantID, ins.ID, "https://example.com/new-photo.jpg")

	require.NoError(t, err)
	assert.Len(t, updated.PhotoURLs, 1)
	assert.Equal(t, "https://example.com/new-photo.jpg", updated.PhotoURLs[0])
}

// ============================================================================
// ListRentals / Pagination Tests
// ============================================================================

func TestService_ListRentals_DefaultPagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")

	for i := range 60 {
		start := may(2026, 1).Add(time.Duration(i*24) * time.Hour * 100)
		addRental(repo, tenantID, obj.ID, start, start.AddDate(0, 0, 1), RentalStatusReserved)
	}

	rentals, total, err := svc.ListRentals(context.Background(), ListRentalsInput{
		TenantID: tenantID,
	})

	require.NoError(t, err)
	assert.Equal(t, 60, total)
	assert.Len(t, rentals, 50)
}

// ============================================================================
// CheckAvailability Tests
// ============================================================================

func TestService_CheckAvailability_Available(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")

	result, err := svc.CheckAvailability(context.Background(), CheckAvailabilityInput{
		TenantID:  tenantID,
		ObjectID:  obj.ID,
		StartDate: may(2026, 1),
		EndDate:   may(2026, 5),
	})

	require.NoError(t, err)
	assert.True(t, result.Available)
}

func TestService_CheckAvailability_Conflict(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 10), RentalStatusReserved)

	result, err := svc.CheckAvailability(context.Background(), CheckAvailabilityInput{
		TenantID:  tenantID,
		ObjectID:  obj.ID,
		StartDate: may(2026, 5),
		EndDate:   may(2026, 8),
	})

	require.NoError(t, err)
	assert.False(t, result.Available)
}

func TestService_CheckAvailability_InvalidRange(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CheckAvailability(context.Background(), CheckAvailabilityInput{
		TenantID:  uuid.New(),
		ObjectID:  uuid.New(),
		StartDate: may(2026, 10),
		EndDate:   may(2026, 5),
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// Calendar Tests
// ============================================================================

// ============================================================================
// UpdateObject / DeleteObject / ListObjects Tests
// ============================================================================

func TestService_UpdateObject_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Bagger")

	newName := "Bagger XL"
	newRate := 120.00
	updated, err := svc.UpdateObject(context.Background(), UpdateObjectInput{
		TenantID:  tenantID,
		ObjectID:  obj.ID,
		Name:      &newName,
		DailyRate: &newRate,
	})

	require.NoError(t, err)
	assert.Equal(t, "Bagger XL", updated.Name)
	assert.InDelta(t, 120.00, updated.DailyRate, 0.001)
}

func TestService_UpdateObject_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateObject(context.Background(), UpdateObjectInput{
		TenantID: uuid.New(),
		ObjectID: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestService_ListObjects_ActiveOnly(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addObject(repo, tenantID, "Aktiv")
	inactive := addObject(repo, tenantID, "Inaktiv")
	inactive.Active = false
	repo.objects[inactive.ID] = inactive

	objs, total, err := svc.ListObjects(context.Background(), ListObjectsInput{
		TenantID:   tenantID,
		ActiveOnly: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, objs, 1)
	assert.Equal(t, "Aktiv", objs[0].Name)
}

// ============================================================================
// UpdateRental Tests
// ============================================================================

func TestService_UpdateRental_RenterName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusReserved)

	newName := "Heidi Maier"
	updated, err := svc.UpdateRental(context.Background(), UpdateRentalInput{
		TenantID:   tenantID,
		RentalID:   rental.ID,
		RenterName: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "Heidi Maier", updated.RenterName)
}

func TestService_DeleteRental_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteRental(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrRentalNotFound)
}

func TestService_DeleteRental_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusReserved)

	err := svc.DeleteRental(context.Background(), tenantID, rental.ID)

	require.NoError(t, err)
	_, ok := repo.rentals[rental.ID]
	assert.False(t, ok, "rental should be removed")
}

// ============================================================================
// ListInspections Tests
// ============================================================================

func TestService_ListInspections_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	// Add 3 inspections
	for range 3 {
		repo.inspections[uuid.New()] = &RentalInspection{
			ID:        uuid.New(),
			TenantID:  tenantID,
			RentalID:  rental.ID,
			Kind:      InspectionKindHandover,
			PhotoURLs: []string{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	inspections, total, err := svc.ListInspections(context.Background(), ListInspectionsInput{
		TenantID: tenantID,
		RentalID: rental.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, inspections, 3)
}

func TestService_GetInspection_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.GetInspection(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrInspectionNotFound)
}

func TestService_GetRentalCalendar_GroupsByObject(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	objA := addObject(repo, tenantID, "Fahrzeug A")
	objB := addObject(repo, tenantID, "Fahrzeug B")

	// 2 rentals for objA and 1 for objB, all in May 2026
	addRental(repo, tenantID, objA.ID, may(2026, 1), may(2026, 3), RentalStatusReserved)
	addRental(repo, tenantID, objA.ID, may(2026, 10), may(2026, 15), RentalStatusActive)
	addRental(repo, tenantID, objB.ID, may(2026, 5), may(2026, 8), RentalStatusReserved)

	entries, err := svc.GetRentalCalendar(context.Background(), tenantID, 2026, 5)

	require.NoError(t, err)
	assert.Len(t, entries, 2)

	countByObj := make(map[uuid.UUID]int)
	for _, e := range entries {
		countByObj[e.Object.ID] = len(e.Rentals)
	}
	assert.Equal(t, 2, countByObj[objA.ID])
	assert.Equal(t, 1, countByObj[objB.ID])
}
