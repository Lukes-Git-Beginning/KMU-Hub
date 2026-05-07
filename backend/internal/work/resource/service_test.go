package resource

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock Repository ---

var testTenantID = uuid.New()

type mockRepo struct {
	resources      map[uuid.UUID]*models.Resource
	resourceTags   map[uuid.UUID][]string
	bookings       map[uuid.UUID]*models.ResourceBooking
	conflictOnNext bool // simulate EXCLUDE constraint
	alternatives   []AlternativeResource
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		resources:    make(map[uuid.UUID]*models.Resource),
		resourceTags: make(map[uuid.UUID][]string),
		bookings:     make(map[uuid.UUID]*models.ResourceBooking),
	}
}

func (m *mockRepo) Create(_ context.Context, resource *models.Resource) error {
	m.resources[resource.ID] = resource
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.Resource, error) {
	r, ok := m.resources[id]
	if !ok {
		return nil, ErrResourceNotFound
	}
	// Tenant isolation: deny cross-tenant access
	if r.TenantID != uuid.Nil && r.TenantID != tenantID {
		return nil, ErrResourceNotFound
	}
	r.Tags = m.resourceTags[id]
	return r, nil
}

func (m *mockRepo) List(_ context.Context, filters ResourceFilters) ([]models.Resource, error) {
	var result []models.Resource
	for _, r := range m.resources {
		// Filter by tenant when set
		if filters.TenantID != uuid.Nil && r.TenantID != filters.TenantID {
			continue
		}
		if filters.IsActive != nil && r.IsActive != *filters.IsActive {
			continue
		}
		if filters.Type != nil && r.ResourceType != *filters.Type {
			continue
		}
		if filters.MinCapacity != nil && (r.Capacity == nil || *r.Capacity < *filters.MinCapacity) {
			continue
		}
		if filters.Floor != nil && (r.Floor == nil || *r.Floor != *filters.Floor) {
			continue
		}
		if len(filters.Tags) > 0 {
			tags := m.resourceTags[r.ID]
			found := false
			for _, ft := range filters.Tags {
				for _, t := range tags {
					if ft == t {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, *r)
	}
	return result, nil
}

func (m *mockRepo) Update(_ context.Context, resource *models.Resource) error {
	if _, ok := m.resources[resource.ID]; !ok {
		return ErrResourceNotFound
	}
	m.resources[resource.ID] = resource
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	r, ok := m.resources[id]
	if !ok {
		return ErrResourceNotFound
	}
	if r.TenantID != uuid.Nil && r.TenantID != tenantID {
		return ErrResourceNotFound
	}
	r.IsActive = false
	return nil
}

func (m *mockRepo) SetTags(_ context.Context, resourceID uuid.UUID, tags []string) error {
	m.resourceTags[resourceID] = tags
	return nil
}

func (m *mockRepo) CreateBooking(_ context.Context, booking *models.ResourceBooking) error {
	if m.conflictOnNext {
		m.conflictOnNext = false
		return ErrBookingConflict
	}
	m.bookings[booking.ID] = booking
	return nil
}

func (m *mockRepo) CancelBooking(_ context.Context, bookingID, tenantID uuid.UUID) error {
	b, ok := m.bookings[bookingID]
	if !ok {
		return ErrBookingNotFound
	}
	if b.TenantID != uuid.Nil && b.TenantID != tenantID {
		return ErrBookingNotFound
	}
	now := time.Now()
	b.CancelledAt = &now
	return nil
}

func (m *mockRepo) ListBookings(_ context.Context, resourceID uuid.UUID, start, end time.Time) ([]models.ResourceBooking, error) {
	var result []models.ResourceBooking
	for _, b := range m.bookings {
		if b.ResourceID == resourceID && b.CancelledAt == nil &&
			b.StartTime.Before(end) && b.EndTime.After(start) {
			result = append(result, *b)
		}
	}
	return result, nil
}

func (m *mockRepo) ListBookingsByEvent(_ context.Context, eventID uuid.UUID) ([]models.ResourceBooking, error) {
	var result []models.ResourceBooking
	for _, b := range m.bookings {
		if b.EventID == eventID && b.CancelledAt == nil {
			result = append(result, *b)
		}
	}
	return result, nil
}

func (m *mockRepo) GetBooking(_ context.Context, bookingID, tenantID uuid.UUID) (*models.ResourceBooking, error) {
	b, ok := m.bookings[bookingID]
	if !ok {
		return nil, ErrBookingNotFound
	}
	if b.TenantID != uuid.Nil && b.TenantID != tenantID {
		return nil, ErrBookingNotFound
	}
	return b, nil
}

func (m *mockRepo) FindAvailableResources(_ context.Context, start, end time.Time, filters ResourceFilters) ([]models.Resource, error) {
	var result []models.Resource
	for _, r := range m.resources {
		if !r.IsActive {
			continue
		}
		if filters.TenantID != uuid.Nil && r.TenantID != filters.TenantID {
			continue
		}
		if filters.Type != nil && r.ResourceType != *filters.Type {
			continue
		}
		// Check no conflicting bookings
		hasConflict := false
		for _, b := range m.bookings {
			if b.ResourceID == r.ID && b.CancelledAt == nil &&
				b.StartTime.Before(end) && b.EndTime.After(start) {
				hasConflict = true
				break
			}
		}
		if hasConflict {
			continue
		}
		result = append(result, *r)
	}
	return result, nil
}

func (m *mockRepo) FindAlternatives(_ context.Context, _ uuid.UUID, _, _ time.Time, _ string, _ uuid.UUID) ([]AlternativeResource, error) {
	return m.alternatives, nil
}

// --- Helper Functions ---

func ptrStr(s string) *string { return &s }
func ptrInt(i int) *int       { return &i }

func setupService() (*Service, *mockRepo) {
	repo := newMockRepo()
	svc := NewService(repo)
	return svc, repo
}

func createRoom(t *testing.T, svc *Service) *models.Resource {
	t.Helper()
	res, err := svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "Konferenzraum A",
		ResourceType: models.ResourceTypeRoom,
		Capacity:     ptrInt(10),
		Floor:        ptrStr("2. OG"),
		Location:     ptrStr("Gebaeude 1"),
		Description:  ptrStr("Grosser Konferenzraum"),
		Tags:         []string{"beamer", "whiteboard"},
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	return res
}

// --- Tests ---

func TestCreate_Room(t *testing.T) {
	svc, repo := setupService()

	res, err := svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "Konferenzraum A",
		ResourceType: models.ResourceTypeRoom,
		Capacity:     ptrInt(10),
		Floor:        ptrStr("2. OG"),
		Tags:         []string{"beamer", "whiteboard"},
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Name != "Konferenzraum A" {
		t.Errorf("name = %q, want %q", res.Name, "Konferenzraum A")
	}
	if res.ResourceType != models.ResourceTypeRoom {
		t.Errorf("type = %q, want %q", res.ResourceType, models.ResourceTypeRoom)
	}
	if !res.IsActive {
		t.Error("expected IsActive = true")
	}
	if len(res.Tags) != 2 {
		t.Errorf("tags count = %d, want 2", len(res.Tags))
	}
	// Verify stored
	if _, ok := repo.resources[res.ID]; !ok {
		t.Error("resource not stored in repo")
	}
	if tags := repo.resourceTags[res.ID]; len(tags) != 2 {
		t.Errorf("stored tags = %d, want 2", len(tags))
	}
}

func TestCreate_Equipment(t *testing.T) {
	svc, _ := setupService()

	res, err := svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "Beamer XR200",
		ResourceType: models.ResourceTypeEquipment,
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ResourceType != models.ResourceTypeEquipment {
		t.Errorf("type = %q, want %q", res.ResourceType, models.ResourceTypeEquipment)
	}
}

func TestCreate_Vehicle(t *testing.T) {
	svc, _ := setupService()

	res, err := svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "VW Transporter",
		ResourceType: models.ResourceTypeVehicle,
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ResourceType != models.ResourceTypeVehicle {
		t.Errorf("type = %q, want %q", res.ResourceType, models.ResourceTypeVehicle)
	}
}

func TestCreate_EmptyName(t *testing.T) {
	svc, _ := setupService()

	_, err := svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "",
		ResourceType: models.ResourceTypeRoom,
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})
	if err != ErrResourceNameRequired {
		t.Errorf("err = %v, want ErrResourceNameRequired", err)
	}
}

func TestCreate_WhitespaceName(t *testing.T) {
	svc, _ := setupService()

	_, err := svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "   ",
		ResourceType: models.ResourceTypeRoom,
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})
	if err != ErrResourceNameRequired {
		t.Errorf("err = %v, want ErrResourceNameRequired", err)
	}
}

func TestCreate_InvalidType(t *testing.T) {
	svc, _ := setupService()

	_, err := svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "Test",
		ResourceType: "spaceship",
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})
	if err != ErrInvalidResourceType {
		t.Errorf("err = %v, want ErrInvalidResourceType", err)
	}
}

func TestCreate_NotAdmin(t *testing.T) {
	svc, _ := setupService()

	_, err := svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "Room B",
		ResourceType: models.ResourceTypeRoom,
		CreatedBy:    uuid.New(),
		IsAdmin:      false,
	})
	if err != ErrNotAuthorized {
		t.Errorf("err = %v, want ErrNotAuthorized", err)
	}
}

func TestGet(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	got, err := svc.Get(context.Background(), created.ID, testTenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("id = %v, want %v", got.ID, created.ID)
	}
}

func TestGet_NotFound(t *testing.T) {
	svc, _ := setupService()

	_, err := svc.Get(context.Background(), uuid.New(), testTenantID)
	if err != ErrResourceNotFound {
		t.Errorf("err = %v, want ErrResourceNotFound", err)
	}
}

func TestList_DefaultActiveOnly(t *testing.T) {
	svc, repo := setupService()

	// Create two resources, deactivate one
	r1 := createRoom(t, svc)
	r2, _ := svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "Raum B",
		ResourceType: models.ResourceTypeRoom,
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})
	repo.resources[r2.ID].IsActive = false

	list, err := svc.List(context.Background(), ResourceFilters{TenantID: testTenantID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list count = %d, want 1", len(list))
	}
	if list[0].ID != r1.ID {
		t.Errorf("listed resource = %v, want %v", list[0].ID, r1.ID)
	}
}

func TestList_FilterByType(t *testing.T) {
	svc, _ := setupService()
	createRoom(t, svc)

	_, _ = svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "Beamer",
		ResourceType: models.ResourceTypeEquipment,
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})

	roomType := models.ResourceTypeRoom
	list, err := svc.List(context.Background(), ResourceFilters{TenantID: testTenantID, Type: &roomType})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("count = %d, want 1 (rooms only)", len(list))
	}
}

func TestList_FilterByCapacity(t *testing.T) {
	svc, _ := setupService()
	createRoom(t, svc) // capacity = 10

	_, _ = svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "Kleine Kammer",
		ResourceType: models.ResourceTypeRoom,
		Capacity:     ptrInt(3),
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})

	minCap := 5
	list, err := svc.List(context.Background(), ResourceFilters{TenantID: testTenantID, MinCapacity: &minCap})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("count = %d, want 1 (capacity >= 5)", len(list))
	}
}

func TestList_FilterByFloor(t *testing.T) {
	svc, _ := setupService()
	createRoom(t, svc) // floor = "2. OG"

	_, _ = svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "Erdgeschoss Raum",
		ResourceType: models.ResourceTypeRoom,
		Floor:        ptrStr("EG"),
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})

	floor := "2. OG"
	list, err := svc.List(context.Background(), ResourceFilters{TenantID: testTenantID, Floor: &floor})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("count = %d, want 1 (floor = 2. OG)", len(list))
	}
}

func TestList_FilterByTags(t *testing.T) {
	svc, _ := setupService()
	createRoom(t, svc) // tags: beamer, whiteboard

	_, _ = svc.Create(context.Background(), CreateInput{
		TenantID:     testTenantID,
		Name:         "Raum ohne Ausstattung",
		ResourceType: models.ResourceTypeRoom,
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})

	list, err := svc.List(context.Background(), ResourceFilters{TenantID: testTenantID, Tags: []string{"beamer"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("count = %d, want 1 (has beamer tag)", len(list))
	}
}

func TestUpdate_Capacity(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	newCap := 20
	updated, err := svc.Update(context.Background(), created.ID, testTenantID, true, UpdateInput{
		Capacity: &newCap,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Capacity == nil || *updated.Capacity != 20 {
		t.Errorf("capacity = %v, want 20", updated.Capacity)
	}
}

func TestUpdate_Name(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	newName := "Konferenzraum B"
	updated, err := svc.Update(context.Background(), created.ID, testTenantID, true, UpdateInput{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Konferenzraum B" {
		t.Errorf("name = %q, want %q", updated.Name, "Konferenzraum B")
	}
}

func TestUpdate_EmptyName(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	emptyName := "  "
	_, err := svc.Update(context.Background(), created.ID, testTenantID, true, UpdateInput{
		Name: &emptyName,
	})
	if err != ErrResourceNameRequired {
		t.Errorf("err = %v, want ErrResourceNameRequired", err)
	}
}

func TestUpdate_InvalidType(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	badType := "spaceship"
	_, err := svc.Update(context.Background(), created.ID, testTenantID, true, UpdateInput{
		ResourceType: &badType,
	})
	if err != ErrInvalidResourceType {
		t.Errorf("err = %v, want ErrInvalidResourceType", err)
	}
}

func TestUpdate_NotAdmin(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	newName := "Neue Name"
	_, err := svc.Update(context.Background(), created.ID, testTenantID, false, UpdateInput{
		Name: &newName,
	})
	if err != ErrNotAuthorized {
		t.Errorf("err = %v, want ErrNotAuthorized", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	svc, _ := setupService()

	newName := "Test"
	_, err := svc.Update(context.Background(), uuid.New(), testTenantID, true, UpdateInput{
		Name: &newName,
	})
	if err != ErrResourceNotFound {
		t.Errorf("err = %v, want ErrResourceNotFound", err)
	}
}

func TestDelete_SoftDelete(t *testing.T) {
	svc, repo := setupService()
	created := createRoom(t, svc)

	err := svc.Delete(context.Background(), created.ID, testTenantID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be inactive now
	r := repo.resources[created.ID]
	if r.IsActive {
		t.Error("expected IsActive = false after soft delete")
	}
}

func TestDelete_NotAdmin(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	err := svc.Delete(context.Background(), created.ID, testTenantID, false)
	if err != ErrNotAuthorized {
		t.Errorf("err = %v, want ErrNotAuthorized", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc, _ := setupService()

	err := svc.Delete(context.Background(), uuid.New(), testTenantID, true)
	if err != ErrResourceNotFound {
		t.Errorf("err = %v, want ErrResourceNotFound", err)
	}
}

func TestSetTags(t *testing.T) {
	svc, repo := setupService()
	created := createRoom(t, svc)

	// Replace tags
	err := svc.SetTags(context.Background(), created.ID, testTenantID, true, []string{"tv", "telefon"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tags := repo.resourceTags[created.ID]; len(tags) != 2 || tags[0] != "tv" {
		t.Errorf("tags = %v, want [tv telefon]", tags)
	}
}

func TestSetTags_NotAdmin(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	err := svc.SetTags(context.Background(), created.ID, testTenantID, false, []string{"new"})
	if err != ErrNotAuthorized {
		t.Errorf("err = %v, want ErrNotAuthorized", err)
	}
}

func TestSetTags_ResourceNotFound(t *testing.T) {
	svc, _ := setupService()

	err := svc.SetTags(context.Background(), uuid.New(), testTenantID, true, []string{"new"})
	if err != ErrResourceNotFound {
		t.Errorf("err = %v, want ErrResourceNotFound", err)
	}
}

func TestBook_Success(t *testing.T) {
	svc, repo := setupService()
	created := createRoom(t, svc)

	actorID := uuid.New()
	eventID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	booking, err := svc.Book(context.Background(), actorID, created.ID, eventID, testTenantID, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if booking.ResourceID != created.ID {
		t.Errorf("resource_id = %v, want %v", booking.ResourceID, created.ID)
	}
	if booking.EventID != eventID {
		t.Errorf("event_id = %v, want %v", booking.EventID, eventID)
	}
	if booking.BookedBy != actorID {
		t.Errorf("booked_by = %v, want %v", booking.BookedBy, actorID)
	}
	if booking.ResourceName != created.Name {
		t.Errorf("resource_name = %q, want %q", booking.ResourceName, created.Name)
	}
	// Verify stored
	if _, ok := repo.bookings[booking.ID]; !ok {
		t.Error("booking not stored in repo")
	}
}

func TestBook_InvalidTimeRange(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	start := time.Now().Add(2 * time.Hour)
	end := time.Now().Add(time.Hour) // end before start

	_, err := svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), testTenantID, start, end)
	if err != ErrInvalidBookingTimeRange {
		t.Errorf("err = %v, want ErrInvalidBookingTimeRange", err)
	}
}

func TestBook_EqualStartEnd(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	now := time.Now().Add(time.Hour)
	_, err := svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), testTenantID, now, now)
	if err != ErrInvalidBookingTimeRange {
		t.Errorf("err = %v, want ErrInvalidBookingTimeRange", err)
	}
}

func TestBook_ResourceNotFound(t *testing.T) {
	svc, _ := setupService()

	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	_, err := svc.Book(context.Background(), uuid.New(), uuid.New(), uuid.New(), testTenantID, start, end)
	if err != ErrResourceNotFound {
		t.Errorf("err = %v, want ErrResourceNotFound", err)
	}
}

func TestBook_InactiveResource(t *testing.T) {
	svc, repo := setupService()
	created := createRoom(t, svc)
	repo.resources[created.ID].IsActive = false

	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	_, err := svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), testTenantID, start, end)
	if err != ErrResourceInactive {
		t.Errorf("err = %v, want ErrResourceInactive", err)
	}
}

func TestBook_DoubleBooking_ConflictError(t *testing.T) {
	svc, repo := setupService()
	created := createRoom(t, svc)

	// Setup alternatives
	altID := uuid.New()
	altCap := 8
	altFloor := "1. OG"
	repo.alternatives = []AlternativeResource{
		{ResourceID: altID, ResourceName: "Raum B", Capacity: &altCap, Floor: &altFloor},
	}

	// First booking succeeds
	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)
	_, err := svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), testTenantID, start, end)
	if err != nil {
		t.Fatalf("first booking failed: %v", err)
	}

	// Second booking conflicts (simulate EXCLUDE constraint)
	repo.conflictOnNext = true
	_, err = svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), testTenantID, start, end)
	if err == nil {
		t.Fatal("expected error for double booking")
	}

	conflictErr, ok := err.(*BookingConflictError)
	if !ok {
		t.Fatalf("err type = %T, want *BookingConflictError", err)
	}
	if conflictErr.ResourceName != created.Name {
		t.Errorf("conflict resource = %q, want %q", conflictErr.ResourceName, created.Name)
	}
	if len(conflictErr.Alternatives) != 1 {
		t.Fatalf("alternatives count = %d, want 1", len(conflictErr.Alternatives))
	}
	if conflictErr.Alternatives[0].ResourceName != "Raum B" {
		t.Errorf("alternative name = %q, want %q", conflictErr.Alternatives[0].ResourceName, "Raum B")
	}
}

func TestBook_DoubleBooking_ErrorMessage(t *testing.T) {
	svc, repo := setupService()
	created := createRoom(t, svc)

	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)
	_, _ = svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), testTenantID, start, end)

	repo.conflictOnNext = true
	_, err := svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), testTenantID, start, end)

	conflictErr, ok := err.(*BookingConflictError)
	if !ok {
		t.Fatalf("err type = %T, want *BookingConflictError", err)
	}
	msg := conflictErr.Error()
	if msg == "" {
		t.Error("error message is empty")
	}
}

func TestCancelBooking_Success(t *testing.T) {
	svc, repo := setupService()
	created := createRoom(t, svc)

	actorID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	booking, _ := svc.Book(context.Background(), actorID, created.ID, uuid.New(), testTenantID, start, end)

	err := svc.CancelBooking(context.Background(), booking.ID, actorID, testTenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cancelled
	b := repo.bookings[booking.ID]
	if b.CancelledAt == nil {
		t.Error("expected CancelledAt to be set")
	}
}

func TestCancelBooking_NotOwner(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	actorID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	booking, _ := svc.Book(context.Background(), actorID, created.ID, uuid.New(), testTenantID, start, end)

	otherUser := uuid.New()
	err := svc.CancelBooking(context.Background(), booking.ID, otherUser, testTenantID)
	if err != ErrNotBookingOwner {
		t.Errorf("err = %v, want ErrNotBookingOwner", err)
	}
}

func TestCancelBooking_AlreadyCancelled(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	actorID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	booking, _ := svc.Book(context.Background(), actorID, created.ID, uuid.New(), testTenantID, start, end)
	_ = svc.CancelBooking(context.Background(), booking.ID, actorID, testTenantID)

	err := svc.CancelBooking(context.Background(), booking.ID, actorID, testTenantID)
	if err != ErrBookingAlreadyCancelled {
		t.Errorf("err = %v, want ErrBookingAlreadyCancelled", err)
	}
}

func TestCancelBooking_NotFound(t *testing.T) {
	svc, _ := setupService()

	err := svc.CancelBooking(context.Background(), uuid.New(), uuid.New(), testTenantID)
	if err != ErrBookingNotFound {
		t.Errorf("err = %v, want ErrBookingNotFound", err)
	}
}

func TestCancelBooking_ThenRebook(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	actorID := uuid.New()
	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	booking, _ := svc.Book(context.Background(), actorID, created.ID, uuid.New(), testTenantID, start, end)
	_ = svc.CancelBooking(context.Background(), booking.ID, actorID, testTenantID)

	// Re-book same slot should succeed (no conflict since cancelled)
	newBooking, err := svc.Book(context.Background(), actorID, created.ID, uuid.New(), testTenantID, start, end)
	if err != nil {
		t.Fatalf("re-booking after cancel failed: %v", err)
	}
	if newBooking.ID == booking.ID {
		t.Error("new booking should have different ID")
	}
}

func TestListAvailability(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	// Use a fixed date at noon to avoid midnight boundary issues
	baseDate := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	start := baseDate
	end := baseDate.Add(2 * time.Hour)
	_, _ = svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), testTenantID, start, end)

	bookings, err := svc.ListAvailability(context.Background(), created.ID, testTenantID, baseDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bookings) != 1 {
		t.Errorf("bookings count = %d, want 1", len(bookings))
	}
}

func TestListAvailability_ResourceNotFound(t *testing.T) {
	svc, _ := setupService()

	_, err := svc.ListAvailability(context.Background(), uuid.New(), testTenantID, time.Now())
	if err != ErrResourceNotFound {
		t.Errorf("err = %v, want ErrResourceNotFound", err)
	}
}

func TestFindAvailable(t *testing.T) {
	svc, _ := setupService()
	createRoom(t, svc)

	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	available, err := svc.FindAvailable(context.Background(), testTenantID, start, end, ResourceFilters{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(available) != 1 {
		t.Errorf("available count = %d, want 1", len(available))
	}
}

func TestFindAvailable_InvalidTimeRange(t *testing.T) {
	svc, _ := setupService()

	end := time.Now()
	start := end.Add(time.Hour)

	_, err := svc.FindAvailable(context.Background(), testTenantID, start, end, ResourceFilters{})
	if err != ErrInvalidBookingTimeRange {
		t.Errorf("err = %v, want ErrInvalidBookingTimeRange", err)
	}
}

// TestResourceRepo_TenantIsolation verifies that resources created for TenantA
// are not visible when querying as TenantB.
func TestResourceRepo_TenantIsolation(t *testing.T) {
	svc, _ := setupService()
	ctx := context.Background()

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create resource as TenantA
	resA, err := svc.Create(ctx, CreateInput{
		TenantID:     tenantA,
		Name:         "TenantA Konferenzraum",
		ResourceType: models.ResourceTypeRoom,
		Capacity:     ptrInt(10),
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}

	// Get as TenantB → should not find it
	_, err = svc.Get(ctx, resA.ID, tenantB)
	if err != ErrResourceNotFound {
		t.Errorf("Get as TenantB: err = %v, want ErrResourceNotFound", err)
	}

	// Get as TenantA → should find it
	found, err := svc.Get(ctx, resA.ID, tenantA)
	if err != nil {
		t.Fatalf("Get as TenantA: unexpected error: %v", err)
	}
	if found.ID != resA.ID {
		t.Errorf("found.ID = %v, want %v", found.ID, resA.ID)
	}

	// Delete as TenantB → should not delete
	err = svc.Delete(ctx, resA.ID, tenantB, true)
	if err != ErrResourceNotFound {
		t.Errorf("Delete as TenantB: err = %v, want ErrResourceNotFound", err)
	}

	// Resource should still be active (TenantB delete was blocked)
	still, err := svc.Get(ctx, resA.ID, tenantA)
	if err != nil {
		t.Fatalf("Get after failed cross-tenant delete: %v", err)
	}
	if !still.IsActive {
		t.Error("resource should still be active after blocked cross-tenant delete")
	}
}
