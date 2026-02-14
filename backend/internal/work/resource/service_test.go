package resource

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock Repository ---

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

func (m *mockRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Resource, error) {
	r, ok := m.resources[id]
	if !ok {
		return nil, ErrResourceNotFound
	}
	r.Tags = m.resourceTags[id]
	return r, nil
}

func (m *mockRepo) List(_ context.Context, filters ResourceFilters) ([]models.Resource, error) {
	var result []models.Resource
	for _, r := range m.resources {
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

func (m *mockRepo) Delete(_ context.Context, id uuid.UUID) error {
	r, ok := m.resources[id]
	if !ok {
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

func (m *mockRepo) CancelBooking(_ context.Context, bookingID uuid.UUID) error {
	b, ok := m.bookings[bookingID]
	if !ok {
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

func (m *mockRepo) GetBooking(_ context.Context, bookingID uuid.UUID) (*models.ResourceBooking, error) {
	b, ok := m.bookings[bookingID]
	if !ok {
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

func (m *mockRepo) FindAlternatives(_ context.Context, _ uuid.UUID, _, _ time.Time, _ string) ([]AlternativeResource, error) {
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

	got, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("id = %v, want %v", got.ID, created.ID)
	}
}

func TestGet_NotFound(t *testing.T) {
	svc, _ := setupService()

	_, err := svc.Get(context.Background(), uuid.New())
	if err != ErrResourceNotFound {
		t.Errorf("err = %v, want ErrResourceNotFound", err)
	}
}

func TestList_DefaultActiveOnly(t *testing.T) {
	svc, repo := setupService()

	// Create two resources, deactivate one
	r1 := createRoom(t, svc)
	r2, _ := svc.Create(context.Background(), CreateInput{
		Name:         "Raum B",
		ResourceType: models.ResourceTypeRoom,
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})
	repo.resources[r2.ID].IsActive = false

	list, err := svc.List(context.Background(), ResourceFilters{})
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
		Name:         "Beamer",
		ResourceType: models.ResourceTypeEquipment,
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})

	roomType := models.ResourceTypeRoom
	list, err := svc.List(context.Background(), ResourceFilters{Type: &roomType})
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
		Name:         "Kleine Kammer",
		ResourceType: models.ResourceTypeRoom,
		Capacity:     ptrInt(3),
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})

	minCap := 5
	list, err := svc.List(context.Background(), ResourceFilters{MinCapacity: &minCap})
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
		Name:         "Erdgeschoss Raum",
		ResourceType: models.ResourceTypeRoom,
		Floor:        ptrStr("EG"),
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})

	floor := "2. OG"
	list, err := svc.List(context.Background(), ResourceFilters{Floor: &floor})
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
		Name:         "Raum ohne Ausstattung",
		ResourceType: models.ResourceTypeRoom,
		CreatedBy:    uuid.New(),
		IsAdmin:      true,
	})

	list, err := svc.List(context.Background(), ResourceFilters{Tags: []string{"beamer"}})
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
	updated, err := svc.Update(context.Background(), created.ID, true, UpdateInput{
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
	updated, err := svc.Update(context.Background(), created.ID, true, UpdateInput{
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
	_, err := svc.Update(context.Background(), created.ID, true, UpdateInput{
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
	_, err := svc.Update(context.Background(), created.ID, true, UpdateInput{
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
	_, err := svc.Update(context.Background(), created.ID, false, UpdateInput{
		Name: &newName,
	})
	if err != ErrNotAuthorized {
		t.Errorf("err = %v, want ErrNotAuthorized", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	svc, _ := setupService()

	newName := "Test"
	_, err := svc.Update(context.Background(), uuid.New(), true, UpdateInput{
		Name: &newName,
	})
	if err != ErrResourceNotFound {
		t.Errorf("err = %v, want ErrResourceNotFound", err)
	}
}

func TestDelete_SoftDelete(t *testing.T) {
	svc, repo := setupService()
	created := createRoom(t, svc)

	err := svc.Delete(context.Background(), created.ID, true)
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

	err := svc.Delete(context.Background(), created.ID, false)
	if err != ErrNotAuthorized {
		t.Errorf("err = %v, want ErrNotAuthorized", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc, _ := setupService()

	err := svc.Delete(context.Background(), uuid.New(), true)
	if err != ErrResourceNotFound {
		t.Errorf("err = %v, want ErrResourceNotFound", err)
	}
}

func TestSetTags(t *testing.T) {
	svc, repo := setupService()
	created := createRoom(t, svc)

	// Replace tags
	err := svc.SetTags(context.Background(), created.ID, true, []string{"tv", "telefon"})
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

	err := svc.SetTags(context.Background(), created.ID, false, []string{"new"})
	if err != ErrNotAuthorized {
		t.Errorf("err = %v, want ErrNotAuthorized", err)
	}
}

func TestSetTags_ResourceNotFound(t *testing.T) {
	svc, _ := setupService()

	err := svc.SetTags(context.Background(), uuid.New(), true, []string{"new"})
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

	booking, err := svc.Book(context.Background(), actorID, created.ID, eventID, start, end)
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

	_, err := svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), start, end)
	if err != ErrInvalidBookingTimeRange {
		t.Errorf("err = %v, want ErrInvalidBookingTimeRange", err)
	}
}

func TestBook_EqualStartEnd(t *testing.T) {
	svc, _ := setupService()
	created := createRoom(t, svc)

	now := time.Now().Add(time.Hour)
	_, err := svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), now, now)
	if err != ErrInvalidBookingTimeRange {
		t.Errorf("err = %v, want ErrInvalidBookingTimeRange", err)
	}
}

func TestBook_ResourceNotFound(t *testing.T) {
	svc, _ := setupService()

	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	_, err := svc.Book(context.Background(), uuid.New(), uuid.New(), uuid.New(), start, end)
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
	_, err := svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), start, end)
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
	_, err := svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), start, end)
	if err != nil {
		t.Fatalf("first booking failed: %v", err)
	}

	// Second booking conflicts (simulate EXCLUDE constraint)
	repo.conflictOnNext = true
	_, err = svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), start, end)
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
	_, _ = svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), start, end)

	repo.conflictOnNext = true
	_, err := svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), start, end)

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

	booking, _ := svc.Book(context.Background(), actorID, created.ID, uuid.New(), start, end)

	err := svc.CancelBooking(context.Background(), booking.ID, actorID)
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

	booking, _ := svc.Book(context.Background(), actorID, created.ID, uuid.New(), start, end)

	otherUser := uuid.New()
	err := svc.CancelBooking(context.Background(), booking.ID, otherUser)
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

	booking, _ := svc.Book(context.Background(), actorID, created.ID, uuid.New(), start, end)
	_ = svc.CancelBooking(context.Background(), booking.ID, actorID)

	err := svc.CancelBooking(context.Background(), booking.ID, actorID)
	if err != ErrBookingAlreadyCancelled {
		t.Errorf("err = %v, want ErrBookingAlreadyCancelled", err)
	}
}

func TestCancelBooking_NotFound(t *testing.T) {
	svc, _ := setupService()

	err := svc.CancelBooking(context.Background(), uuid.New(), uuid.New())
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

	booking, _ := svc.Book(context.Background(), actorID, created.ID, uuid.New(), start, end)
	_ = svc.CancelBooking(context.Background(), booking.ID, actorID)

	// Re-book same slot should succeed (no conflict since cancelled)
	newBooking, err := svc.Book(context.Background(), actorID, created.ID, uuid.New(), start, end)
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
	_, _ = svc.Book(context.Background(), uuid.New(), created.ID, uuid.New(), start, end)

	bookings, err := svc.ListAvailability(context.Background(), created.ID, baseDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bookings) != 1 {
		t.Errorf("bookings count = %d, want 1", len(bookings))
	}
}

func TestListAvailability_ResourceNotFound(t *testing.T) {
	svc, _ := setupService()

	_, err := svc.ListAvailability(context.Background(), uuid.New(), time.Now())
	if err != ErrResourceNotFound {
		t.Errorf("err = %v, want ErrResourceNotFound", err)
	}
}

func TestFindAvailable(t *testing.T) {
	svc, _ := setupService()
	createRoom(t, svc)

	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	available, err := svc.FindAvailable(context.Background(), start, end, ResourceFilters{})
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

	_, err := svc.FindAvailable(context.Background(), start, end, ResourceFilters{})
	if err != ErrInvalidBookingTimeRange {
		t.Errorf("err = %v, want ErrInvalidBookingTimeRange", err)
	}
}
