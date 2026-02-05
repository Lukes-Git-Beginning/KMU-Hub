package contact

import (
	"context"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles contact business logic
type Service struct {
	repo Repository
}

// NewService creates a new contact service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a contact
type CreateInput struct {
	FirstName    string
	LastName     string
	Email        *string
	Phone        *string
	CompanyID    *uuid.UUID
	Position     *string
	Notes        *string
	TagIDs       []uuid.UUID
	CustomFields map[uuid.UUID]any // field_id -> value
	CreatedBy    uuid.UUID
}

// Create creates a new contact
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.ContactWithRelations, error) {
	// Validate first name
	firstName := strings.TrimSpace(input.FirstName)
	if firstName == "" {
		return nil, ErrFirstNameRequired
	}

	// Validate last name
	lastName := strings.TrimSpace(input.LastName)
	if lastName == "" {
		return nil, ErrLastNameRequired
	}

	// Validate and check email uniqueness if provided
	var email *string
	if input.Email != nil && *input.Email != "" {
		trimmed := strings.TrimSpace(*input.Email)
		if _, err := mail.ParseAddress(trimmed); err != nil {
			return nil, ErrInvalidEmail
		}
		// Check uniqueness
		existing, _ := s.repo.GetByEmail(ctx, trimmed)
		if existing != nil {
			return nil, ErrEmailExists
		}
		email = &trimmed
	}

	// Validate company exists if provided
	if input.CompanyID != nil {
		exists, err := s.repo.CompanyExists(ctx, *input.CompanyID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrCompanyNotFound
		}
	}

	// Validate tags are for contacts
	for _, tagID := range input.TagIDs {
		exists, err := s.repo.TagExists(ctx, tagID, models.EntityTypeContact)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrTagNotFound
		}
	}

	contact := &models.Contact{
		ID:        uuid.New(),
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Phone:     trimStringPtr(input.Phone),
		CompanyID: input.CompanyID,
		Position:  trimStringPtr(input.Position),
		Notes:     input.Notes,
		CreatedBy: input.CreatedBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, contact); err != nil {
		return nil, err
	}

	// Add tags
	if len(input.TagIDs) > 0 {
		if err := s.repo.AddTags(ctx, contact.ID, input.TagIDs); err != nil {
			return nil, err
		}
	}

	// Set custom field values
	if len(input.CustomFields) > 0 {
		if err := s.repo.SetCustomFieldValues(ctx, contact.ID, input.CustomFields); err != nil {
			return nil, err
		}
	}

	slog.Info("contact created",
		"contact_id", contact.ID,
		"email", contact.Email,
	)

	return s.getWithRelations(ctx, contact)
}

// GetByID retrieves a contact by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.ContactWithRelations, error) {
	contact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrContactNotFound
	}
	return s.getWithRelations(ctx, contact)
}

// ListInput contains filtering options for listing contacts
type ListInput struct {
	CompanyID *uuid.UUID
	TagIDs    []uuid.UUID
	Search    string
	Page      int
	PageSize  int
	SortBy    string
	SortDesc  bool
}

// List retrieves contacts with optional filtering
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.ContactWithRelations, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := (input.Page - 1) * input.PageSize
	filter := ListFilter{
		CompanyID: input.CompanyID,
		TagIDs:    input.TagIDs,
		Search:    input.Search,
		SortBy:    input.SortBy,
		SortDesc:  input.SortDesc,
	}

	contacts, total, err := s.repo.List(ctx, filter, offset, input.PageSize)
	if err != nil {
		return nil, 0, err
	}

	var results []*models.ContactWithRelations
	for _, c := range contacts {
		withRel, relErr := s.getWithRelations(ctx, c)
		if relErr != nil {
			return nil, 0, relErr
		}
		results = append(results, withRel)
	}

	return results, total, nil
}

// UpdateInput contains the data that can be updated on a contact
type UpdateInput struct {
	FirstName    *string
	LastName     *string
	Email        *string
	Phone        *string
	CompanyID    *uuid.UUID // use uuid.Nil to clear
	Position     *string
	Notes        *string
	CustomFields map[uuid.UUID]any
}

// Update updates an existing contact
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*models.ContactWithRelations, error) {
	contact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrContactNotFound
	}

	if input.FirstName != nil {
		firstName := strings.TrimSpace(*input.FirstName)
		if firstName == "" {
			return nil, ErrFirstNameRequired
		}
		contact.FirstName = firstName
	}

	if input.LastName != nil {
		lastName := strings.TrimSpace(*input.LastName)
		if lastName == "" {
			return nil, ErrLastNameRequired
		}
		contact.LastName = lastName
	}

	if input.Email != nil {
		if *input.Email == "" {
			contact.Email = nil
		} else {
			trimmed := strings.TrimSpace(*input.Email)
			if _, parseErr := mail.ParseAddress(trimmed); parseErr != nil {
				return nil, ErrInvalidEmail
			}
			// Check uniqueness (excluding self)
			existing, _ := s.repo.GetByEmail(ctx, trimmed)
			if existing != nil && existing.ID != contact.ID {
				return nil, ErrEmailExists
			}
			contact.Email = &trimmed
		}
	}

	if input.Phone != nil {
		contact.Phone = trimStringPtr(input.Phone)
	}

	if input.CompanyID != nil {
		if *input.CompanyID == uuid.Nil {
			contact.CompanyID = nil
		} else {
			exists, companyErr := s.repo.CompanyExists(ctx, *input.CompanyID)
			if companyErr != nil {
				return nil, companyErr
			}
			if !exists {
				return nil, ErrCompanyNotFound
			}
			contact.CompanyID = input.CompanyID
		}
	}

	if input.Position != nil {
		contact.Position = trimStringPtr(input.Position)
	}

	if input.Notes != nil {
		contact.Notes = input.Notes
	}

	contact.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, contact); updateErr != nil {
		return nil, updateErr
	}

	// Update custom fields if provided
	if len(input.CustomFields) > 0 {
		if cfErr := s.repo.SetCustomFieldValues(ctx, contact.ID, input.CustomFields); cfErr != nil {
			return nil, cfErr
		}
	}

	slog.Info("contact updated", "contact_id", contact.ID)

	return s.getWithRelations(ctx, contact)
}

// Delete removes a contact
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	contact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrContactNotFound
	}

	// Check if contact is used by deals/activities (for future use)
	inUse, err := s.repo.IsInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return ErrContactInUse
	}

	if deleteErr := s.repo.Delete(ctx, id); deleteErr != nil {
		return deleteErr
	}

	slog.Info("contact deleted",
		"contact_id", contact.ID,
		"email", contact.Email,
	)

	return nil
}

// AddTags adds tags to a contact
func (s *Service) AddTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) (*models.ContactWithRelations, error) {
	contact, err := s.repo.GetByID(ctx, contactID)
	if err != nil {
		return nil, ErrContactNotFound
	}

	// Validate tags
	for _, tagID := range tagIDs {
		exists, tagErr := s.repo.TagExists(ctx, tagID, models.EntityTypeContact)
		if tagErr != nil {
			return nil, tagErr
		}
		if !exists {
			return nil, ErrTagNotFound
		}
	}

	if addErr := s.repo.AddTags(ctx, contactID, tagIDs); addErr != nil {
		return nil, addErr
	}

	return s.getWithRelations(ctx, contact)
}

// RemoveTags removes tags from a contact
func (s *Service) RemoveTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) (*models.ContactWithRelations, error) {
	contact, err := s.repo.GetByID(ctx, contactID)
	if err != nil {
		return nil, ErrContactNotFound
	}

	if removeErr := s.repo.RemoveTags(ctx, contactID, tagIDs); removeErr != nil {
		return nil, removeErr
	}

	return s.getWithRelations(ctx, contact)
}

// getWithRelations loads all relations for a contact
func (s *Service) getWithRelations(ctx context.Context, contact *models.Contact) (*models.ContactWithRelations, error) {
	result := &models.ContactWithRelations{
		Contact: *contact,
	}

	// Load company name
	if contact.CompanyID != nil {
		name, _ := s.repo.GetCompanyName(ctx, *contact.CompanyID)
		if name != "" {
			result.CompanyName = &name
		}
	}

	// Load tags
	tags, _ := s.repo.GetTags(ctx, contact.ID)
	result.Tags = tags

	// Load custom field values
	values, _ := s.repo.GetCustomFieldValues(ctx, contact.ID)
	if len(values) > 0 {
		result.CustomFields = make(map[string]any)
		for _, v := range values {
			result.CustomFields[v.FieldName] = v.Value
		}
	}

	return result, nil
}

// Helper to trim and nil empty strings
func trimStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
