package resource

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles resource business logic
type Service struct {
	repo Repository
}

// NewService creates a new resource service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains data needed to create a resource
type CreateInput struct {
	TenantID     uuid.UUID
	Name         string
	ResourceType string
	Capacity     *int
	Floor        *string
	Location     *string
	Description  *string
	Tags         []string
	CreatedBy    uuid.UUID
	IsAdmin      bool
}

// Create creates a new bookable resource
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.Resource, error) {
	if !input.IsAdmin {
		return nil, ErrNotAuthorized
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrResourceNameRequired
	}

	if !models.ValidResourceTypes[input.ResourceType] {
		return nil, ErrInvalidResourceType
	}

	now := time.Now()
	resource := &models.Resource{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		Name:         name,
		ResourceType: input.ResourceType,
		Capacity:     input.Capacity,
		Floor:        input.Floor,
		Location:     input.Location,
		Description:  input.Description,
		IsActive:     true,
		CreatedBy:    input.CreatedBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, resource); err != nil {
		return nil, err
	}

	if len(input.Tags) > 0 {
		if err := s.repo.SetTags(ctx, resource.ID, input.Tags); err != nil {
			return nil, err
		}
		resource.Tags = input.Tags
	}

	slog.Info("resource created",
		"resource_id", resource.ID,
		"name", resource.Name,
		"type", resource.ResourceType,
		"tenant_id", input.TenantID,
	)

	return resource, nil
}

// Get retrieves a resource by ID
func (s *Service) Get(ctx context.Context, id, tenantID uuid.UUID) (*models.Resource, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// List lists resources with optional filters. Defaults to active-only.
func (s *Service) List(ctx context.Context, filters ResourceFilters) ([]models.Resource, error) {
	if filters.IsActive == nil {
		active := true
		filters.IsActive = &active
	}
	return s.repo.List(ctx, filters)
}

// UpdateInput contains fields that can be changed on a resource
type UpdateInput struct {
	Name         *string
	ResourceType *string
	Capacity     *int
	Floor        *string
	Location     *string
	Description  *string
}

// Update modifies an existing resource. Only admin/managers can update.
func (s *Service) Update(ctx context.Context, resourceID, tenantID uuid.UUID, isAdmin bool, input UpdateInput) (*models.Resource, error) {
	if !isAdmin {
		return nil, ErrNotAuthorized
	}

	resource, err := s.repo.GetByID(ctx, resourceID, tenantID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrResourceNameRequired
		}
		resource.Name = name
	}

	if input.ResourceType != nil {
		if !models.ValidResourceTypes[*input.ResourceType] {
			return nil, ErrInvalidResourceType
		}
		resource.ResourceType = *input.ResourceType
	}

	if input.Capacity != nil {
		resource.Capacity = input.Capacity
	}
	if input.Floor != nil {
		resource.Floor = input.Floor
	}
	if input.Location != nil {
		resource.Location = input.Location
	}
	if input.Description != nil {
		resource.Description = input.Description
	}

	resource.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, resource); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("resource updated",
		"resource_id", resource.ID,
		"name", resource.Name,
	)

	return resource, nil
}

// SetTags replaces all tags on a resource. Only admin/managers can set tags.
func (s *Service) SetTags(ctx context.Context, resourceID, tenantID uuid.UUID, isAdmin bool, tags []string) error {
	if !isAdmin {
		return ErrNotAuthorized
	}

	// Verify resource exists
	if _, err := s.repo.GetByID(ctx, resourceID, tenantID); err != nil {
		return err
	}

	return s.repo.SetTags(ctx, resourceID, tags)
}

// Delete soft-deletes a resource (sets is_active = false). Only admin/managers.
func (s *Service) Delete(ctx context.Context, resourceID, tenantID uuid.UUID, isAdmin bool) error {
	if !isAdmin {
		return ErrNotAuthorized
	}

	if err := s.repo.Delete(ctx, resourceID, tenantID); err != nil {
		return err
	}

	slog.Info("resource soft-deleted", "resource_id", resourceID)
	return nil
}

// Book creates a booking for a resource. If a double-booking is detected (via
// the DB EXCLUDE GIST constraint), it fetches alternative resources of the
// same type and returns a BookingConflictError with suggestions.
func (s *Service) Book(ctx context.Context, actorID, resourceID, eventID, tenantID uuid.UUID, start, end time.Time) (*models.ResourceBooking, error) {
	if !end.After(start) {
		return nil, ErrInvalidBookingTimeRange
	}

	resource, err := s.repo.GetByID(ctx, resourceID, tenantID)
	if err != nil {
		return nil, err
	}

	if !resource.IsActive {
		return nil, ErrResourceInactive
	}

	now := time.Now()
	booking := &models.ResourceBooking{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ResourceID: resourceID,
		EventID:    eventID,
		BookedBy:   actorID,
		StartTime:  start,
		EndTime:    end,
		CreatedAt:  now,
	}

	createErr := s.repo.CreateBooking(ctx, booking)
	if createErr != nil {
		if errors.Is(createErr, ErrBookingConflict) {
			// Fetch the existing conflicting bookings
			conflicting, _ := s.repo.ListBookings(ctx, resourceID, start, end)
			var conflictStart, conflictEnd time.Time
			if len(conflicting) > 0 {
				conflictStart = conflicting[0].StartTime
				conflictEnd = conflicting[0].EndTime
			} else {
				conflictStart = start
				conflictEnd = end
			}

			// Find alternative resources of the same type
			alternatives, _ := s.repo.FindAlternatives(ctx, resourceID, start, end, resource.ResourceType, tenantID)

			return nil, &BookingConflictError{
				ResourceName:  resource.Name,
				ConflictStart: conflictStart,
				ConflictEnd:   conflictEnd,
				Alternatives:  alternatives,
			}
		}
		return nil, createErr
	}

	booking.ResourceName = resource.Name

	slog.Info("resource booked",
		"booking_id", booking.ID,
		"resource_id", resourceID,
		"event_id", eventID,
		"start", start,
		"end", end,
	)

	return booking, nil
}

// CancelBooking cancels a booking. Only the booking owner can cancel.
func (s *Service) CancelBooking(ctx context.Context, bookingID, actorID, tenantID uuid.UUID) error {
	booking, err := s.repo.GetBooking(ctx, bookingID, tenantID)
	if err != nil {
		return err
	}

	if booking.CancelledAt != nil {
		return ErrBookingAlreadyCancelled
	}

	if booking.BookedBy != actorID {
		return ErrNotBookingOwner
	}

	if cancelErr := s.repo.CancelBooking(ctx, bookingID, tenantID); cancelErr != nil {
		return cancelErr
	}

	slog.Info("booking cancelled",
		"booking_id", bookingID,
		"actor_id", actorID,
	)

	return nil
}

// ListAvailability lists all active bookings for a resource on a given date
func (s *Service) ListAvailability(ctx context.Context, resourceID, tenantID uuid.UUID, date time.Time) ([]models.ResourceBooking, error) {
	// Verify resource exists
	if _, err := s.repo.GetByID(ctx, resourceID, tenantID); err != nil {
		return nil, err
	}

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	return s.repo.ListBookings(ctx, resourceID, dayStart, dayEnd)
}

// FindAvailable finds resources available for a given time range with optional filters
func (s *Service) FindAvailable(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filters ResourceFilters) ([]models.Resource, error) {
	if !end.After(start) {
		return nil, ErrInvalidBookingTimeRange
	}
	filters.TenantID = tenantID
	return s.repo.FindAvailableResources(ctx, start, end, filters)
}
