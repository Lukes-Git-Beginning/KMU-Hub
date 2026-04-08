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
		ID:         uuid.New(),
		FirstName:  firstName,
		LastName:   lastName,
		Email:      email,
		Phone:      trimStringPtr(input.Phone),
		CompanyID:  input.CompanyID,
		Position:   trimStringPtr(input.Position),
		Notes:      input.Notes,
		Visibility: "shared",
		CreatedBy:  input.CreatedBy,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
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
	CompanyID        *uuid.UUID
	TagIDs           []uuid.UUID
	Search           string
	Page             int
	PageSize         int
	SortBy           string
	SortDesc         bool
	VisibilityFilter string // "", "shared", "personal"
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

	results, enrichErr := s.enrichWithRelationsBatch(ctx, contacts)
	if enrichErr != nil {
		return nil, 0, enrichErr
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

// enrichWithRelationsBatch loads all relations for a slice of contacts in batch.
// Uses 3 queries total instead of 3*N.
func (s *Service) enrichWithRelationsBatch(ctx context.Context, contacts []*models.Contact) ([]*models.ContactWithRelations, error) {
	if len(contacts) == 0 {
		return nil, nil
	}

	// Collect IDs
	contactIDs := make([]uuid.UUID, len(contacts))
	companyIDSet := make(map[uuid.UUID]struct{})
	for i, c := range contacts {
		contactIDs[i] = c.ID
		if c.CompanyID != nil {
			companyIDSet[*c.CompanyID] = struct{}{}
		}
	}

	companyIDs := make([]uuid.UUID, 0, len(companyIDSet))
	for id := range companyIDSet {
		companyIDs = append(companyIDs, id)
	}

	// Batch fetch all relations
	companyNames, err := s.repo.GetCompanyNames(ctx, companyIDs)
	if err != nil {
		return nil, err
	}

	tagsByContact, err := s.repo.GetTagsBatch(ctx, contactIDs)
	if err != nil {
		return nil, err
	}

	cfByContact, err := s.repo.GetCustomFieldValuesBatch(ctx, contactIDs)
	if err != nil {
		return nil, err
	}

	// Assemble results
	results := make([]*models.ContactWithRelations, len(contacts))
	for i, c := range contacts {
		result := &models.ContactWithRelations{
			Contact: *c,
		}

		if c.CompanyID != nil {
			if name, ok := companyNames[*c.CompanyID]; ok && name != "" {
				result.CompanyName = &name
			}
		}

		result.Tags = tagsByContact[c.ID]

		if values := cfByContact[c.ID]; len(values) > 0 {
			result.CustomFields = make(map[string]any)
			for _, v := range values {
				result.CustomFields[v.FieldName] = v.Value
			}
		}

		results[i] = result
	}

	return results, nil
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

// DuplicateCandidate represents a potential duplicate contact with similarity scoring.
type DuplicateCandidate struct {
	Contact    *models.ContactWithRelations `json:"contact"`
	Similarity float64                      `json:"similarity"`
	MatchType  string                       `json:"match_type"` // "email_exact", "name_fuzzy", "phone_exact"
}

// FindDuplicates returns potential duplicate contacts for the given contact ID.
// Matches by: exact email (similarity 1.0), trigram name (>= 0.7), exact phone (0.9).
func (s *Service) FindDuplicates(ctx context.Context, contactID uuid.UUID) ([]*DuplicateCandidate, error) {
	// Verify contact exists
	if _, err := s.repo.GetByID(ctx, contactID); err != nil {
		return nil, ErrContactNotFound
	}

	candidates, err := s.repo.FindDuplicateCandidates(ctx, contactID)
	if err != nil {
		return nil, err
	}

	// Enrich with relations
	for _, c := range candidates {
		withRel, relErr := s.getWithRelations(ctx, &c.Contact.Contact)
		if relErr == nil {
			c.Contact = withRel
		}
	}

	return candidates, nil
}

// MergeContacts merges a duplicate contact into a primary contact.
// All activities, deals, tags, and custom fields are reassigned to the primary.
// The duplicate is soft-deleted by setting merged_into_id.
func (s *Service) MergeContacts(ctx context.Context, primaryID, duplicateID uuid.UUID) (*models.ContactWithRelations, error) {
	if primaryID == duplicateID {
		return nil, ErrCannotMergeSelf
	}

	// Verify both exist
	primary, err := s.repo.GetByID(ctx, primaryID)
	if err != nil {
		return nil, ErrContactNotFound
	}
	if primary.MergedIntoID != nil {
		return nil, ErrAlreadyMerged
	}

	dup, err := s.repo.GetByID(ctx, duplicateID)
	if err != nil {
		return nil, ErrContactNotFound
	}
	if dup.MergedIntoID != nil {
		return nil, ErrAlreadyMerged
	}

	if err := s.repo.MergeInto(ctx, primaryID, duplicateID); err != nil {
		return nil, err
	}

	slog.Info("contacts merged",
		"primary_id", primaryID,
		"duplicate_id", duplicateID,
	)

	return s.getWithRelations(ctx, primary)
}

// GetByEmail retrieves a contact by email address (used by import merge logic).
func (s *Service) GetByEmail(ctx context.Context, email string) (*models.Contact, error) {
	return s.repo.GetByEmail(ctx, email)
}

// CreateForImport creates a contact without the usual validation (used by bulk import).
func (s *Service) CreateForImport(ctx context.Context, contact *models.Contact) error {
	if contact.Visibility == "" {
		contact.Visibility = "shared"
	}
	return s.repo.Create(ctx, contact)
}

// UpdateForImport updates a contact during import merge.
func (s *Service) UpdateForImport(ctx context.Context, contact *models.Contact) error {
	return s.repo.Update(ctx, contact)
}

// ListByIDs retrieves contacts by a list of IDs (used by export).
func (s *Service) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Contact, error) {
	return s.repo.ListByIDs(ctx, ids)
}

// ListVisible retrieves all contacts visible to the given user (used by export all).
func (s *Service) ListVisible(ctx context.Context, userID uuid.UUID, isAdmin bool) ([]*models.Contact, error) {
	return s.repo.ListAll(ctx, userID, isAdmin)
}

// ListWithVisibility retrieves contacts with visibility filtering.
func (s *Service) ListWithVisibility(ctx context.Context, userID uuid.UUID, isAdmin bool, input ListInput) ([]*models.ContactWithRelations, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := (input.Page - 1) * input.PageSize
	filter := ListFilter{
		CompanyID:        input.CompanyID,
		TagIDs:           input.TagIDs,
		Search:           input.Search,
		SortBy:           input.SortBy,
		SortDesc:         input.SortDesc,
		VisibilityFilter: input.VisibilityFilter,
	}

	contacts, total, err := s.repo.ListWithVisibility(ctx, userID, isAdmin, filter, offset, input.PageSize)
	if err != nil {
		return nil, 0, err
	}

	results, enrichErr := s.enrichWithRelationsBatch(ctx, contacts)
	if enrichErr != nil {
		return nil, 0, enrichErr
	}

	return results, total, nil
}

// UpdateVisibility updates the visibility and owner of a contact.
func (s *Service) UpdateVisibility(ctx context.Context, contactID uuid.UUID, visibility string, ownerID *uuid.UUID) error {
	return s.repo.UpdateVisibility(ctx, contactID, visibility, ownerID)
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
