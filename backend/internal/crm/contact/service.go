package contact

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"slices"
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
	TenantID     uuid.UUID
	// Lead is set only by CreateLead; nil means an ordinary contact, which is
	// what every existing caller creates.
	Lead *LeadIntake
	ProfileFields
}

// LeadIntake carries the lifecycle columns for a contact created through the
// lead inbox. See lead.go for the stage/status/source vocabularies.
type LeadIntake struct {
	Stage   string
	Source  *string
	Score   *int16
	Status  *string
	Company *string
}

// ProfileFields are the contact-form fields added in migration 000314. Shared
// by CreateInput and UpdateInput so the two cannot drift apart. A nil field
// means "not supplied"; an empty string clears the column.
//
// Category and Status describe the relationship, not the sales pipeline --
// LeadIntake above owns that.
type ProfileFields struct {
	Salutation     *string
	Title          *string
	Mobile         *string
	Department     *string
	AddressStreet  *string
	AddressZip     *string
	AddressCity    *string
	AddressCountry *string
	Website        *string
	LinkedIn       *string
	Xing           *string
	Category       *string
	Status         *string
}

// validate mirrors the CHECK constraints so an invalid value fails as a
// validation error rather than a database error.
func (p ProfileFields) validate() error {
	if err := checkEnum(p.Salutation, ErrInvalidSalutation, "Herr", "Frau"); err != nil {
		return err
	}
	if err := checkEnum(p.Category, ErrInvalidCategory, "employee", "customer", "partner"); err != nil {
		return err
	}
	return checkEnum(p.Status, ErrInvalidStatus, "active", "prospect", "inactive")
}

// checkEnum accepts nil and the empty string (both mean "no value") plus the
// allowed vocabulary.
func checkEnum(v *string, invalid error, allowed ...string) error {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	if slices.Contains(allowed, strings.TrimSpace(*v)) {
		return nil
	}
	return invalid
}

// applyTo writes every supplied field onto the contact. Used by Create (where
// all fields are supplied at once) and by Update (which relies on nil meaning
// "leave alone").
func (p ProfileFields) applyTo(c *models.Contact) {
	assign := func(dst **string, src *string) {
		if src != nil {
			*dst = trimStringPtr(src)
		}
	}
	assign(&c.Salutation, p.Salutation)
	assign(&c.Title, p.Title)
	assign(&c.Mobile, p.Mobile)
	assign(&c.Department, p.Department)
	assign(&c.AddressStreet, p.AddressStreet)
	assign(&c.AddressZip, p.AddressZip)
	assign(&c.AddressCity, p.AddressCity)
	assign(&c.AddressCountry, p.AddressCountry)
	assign(&c.Website, p.Website)
	assign(&c.LinkedIn, p.LinkedIn)
	assign(&c.Xing, p.Xing)
	assign(&c.Category, p.Category)
	assign(&c.Status, p.Status)
}

// Create creates a new contact
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.ContactWithRelations, error) {
	tenantID := input.TenantID
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}

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
		existing, _ := s.repo.GetByEmail(ctx, trimmed, tenantID)
		if existing != nil {
			return nil, ErrEmailExists
		}
		email = &trimmed
	}

	// Validate company exists if provided
	if input.CompanyID != nil {
		exists, err := s.repo.CompanyExists(ctx, *input.CompanyID, tenantID)
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

	if err := input.validate(); err != nil {
		return nil, err
	}

	contact := &models.Contact{
		ID:             uuid.New(),
		FirstName:      firstName,
		LastName:       lastName,
		Email:          email,
		Phone:          trimStringPtr(input.Phone),
		CompanyID:      input.CompanyID,
		Position:       trimStringPtr(input.Position),
		Notes:          input.Notes,
		Visibility:     "shared",
		TenantID:       tenantID,
		CreatedBy:      input.CreatedBy,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		LifecycleStage: LifecycleCustomer,
	}
	input.applyTo(contact)

	if input.Lead != nil {
		contact.LifecycleStage = input.Lead.Stage
		contact.LeadSource = input.Lead.Source
		contact.LeadScore = input.Lead.Score
		contact.LeadStatus = input.Lead.Status
		contact.LeadCompany = input.Lead.Company
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

	return s.getWithRelations(ctx, contact, tenantID)
}

// GetByID retrieves a contact by ID scoped to a tenant
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.ContactWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	contact, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, ErrContactNotFound
	}
	return s.getWithRelations(ctx, contact, tenantID)
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
	TenantID         uuid.UUID
}

// List retrieves contacts with optional filtering
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.ContactWithRelations, int, error) {
	tenantID := input.TenantID
	if tenantID == uuid.Nil {
		return nil, 0, ErrInvalidTenant
	}

	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := (input.Page - 1) * input.PageSize
	filter := ListFilter{
		TenantID:  tenantID,
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

	results, enrichErr := s.enrichWithRelationsBatch(ctx, contacts, tenantID)
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
	ProfileFields
}

// Update updates an existing contact scoped to a tenant
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput, tenantID uuid.UUID) (*models.ContactWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	contact, err := s.repo.GetByID(ctx, id, tenantID)
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
			existing, _ := s.repo.GetByEmail(ctx, trimmed, tenantID)
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
			exists, companyErr := s.repo.CompanyExists(ctx, *input.CompanyID, tenantID)
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

	if err := input.validate(); err != nil {
		return nil, err
	}
	input.applyTo(contact)

	contact.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, contact, tenantID); updateErr != nil {
		return nil, updateErr
	}

	// Update custom fields if provided
	if len(input.CustomFields) > 0 {
		if cfErr := s.repo.SetCustomFieldValues(ctx, contact.ID, input.CustomFields); cfErr != nil {
			return nil, cfErr
		}
	}

	slog.Info("contact updated", "contact_id", contact.ID)

	return s.getWithRelations(ctx, contact, tenantID)
}

// Delete removes a contact scoped to a tenant
func (s *Service) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenant
	}
	contact, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return ErrContactNotFound
	}

	// Contacts with call campaign history or advisory protocols cannot be hard
	// deleted (ON DELETE RESTRICT, migrations 000130/000137) -- point the
	// caller at the anonymization path (consent.Service.RequestDeletion) instead.
	inUse, reason, err := s.repo.IsInUse(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if inUse {
		return fmt.Errorf("%w: %s reference this contact; use GDPR anonymization instead of deleting", ErrContactInUse, reason)
	}

	if deleteErr := s.repo.Delete(ctx, id, tenantID); deleteErr != nil {
		return deleteErr
	}

	slog.Info("contact deleted",
		"contact_id", contact.ID,
		"email", contact.Email,
	)

	return nil
}

// PreviewDeletion reports what deleting a contact would do to every table
// that references it, without changing anything. Used to show the user the
// blast radius before they confirm a delete.
func (s *Service) PreviewDeletion(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) ([]DeletionImpactItem, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	if _, err := s.repo.GetByID(ctx, id, tenantID); err != nil {
		return nil, ErrContactNotFound
	}

	return s.repo.DeletionImpact(ctx, id, tenantID)
}

// AddTags adds tags to a contact scoped to a tenant
func (s *Service) AddTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID, tenantID uuid.UUID) (*models.ContactWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	contact, err := s.repo.GetByID(ctx, contactID, tenantID)
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

	return s.getWithRelations(ctx, contact, tenantID)
}

// RemoveTags removes tags from a contact scoped to a tenant
func (s *Service) RemoveTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID, tenantID uuid.UUID) (*models.ContactWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	contact, err := s.repo.GetByID(ctx, contactID, tenantID)
	if err != nil {
		return nil, ErrContactNotFound
	}

	if removeErr := s.repo.RemoveTags(ctx, contactID, tagIDs); removeErr != nil {
		return nil, removeErr
	}

	return s.getWithRelations(ctx, contact, tenantID)
}

// enrichWithRelationsBatch loads all relations for a slice of contacts in batch.
// Uses 3 queries total instead of 3*N.
func (s *Service) enrichWithRelationsBatch(ctx context.Context, contacts []*models.Contact, tenantID uuid.UUID) ([]*models.ContactWithRelations, error) {
	if len(contacts) == 0 {
		return []*models.ContactWithRelations{}, nil
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
	companyNames, err := s.repo.GetCompanyNames(ctx, companyIDs, tenantID)
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
func (s *Service) getWithRelations(ctx context.Context, contact *models.Contact, tenantID uuid.UUID) (*models.ContactWithRelations, error) {
	result := &models.ContactWithRelations{
		Contact: *contact,
	}

	// Load company name
	if contact.CompanyID != nil {
		name, _ := s.repo.GetCompanyName(ctx, *contact.CompanyID, tenantID)
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

// FindDuplicates returns potential duplicate contacts for the given contact ID scoped to a tenant.
// Matches by: exact email (similarity 1.0), trigram name (>= 0.7), exact phone (0.9).
func (s *Service) FindDuplicates(ctx context.Context, contactID uuid.UUID, tenantID uuid.UUID) ([]*DuplicateCandidate, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	// Verify contact exists
	if _, err := s.repo.GetByID(ctx, contactID, tenantID); err != nil {
		return nil, ErrContactNotFound
	}

	candidates, err := s.repo.FindDuplicateCandidates(ctx, contactID, tenantID)
	if err != nil {
		return nil, err
	}

	// Enrich with relations
	for _, c := range candidates {
		withRel, relErr := s.getWithRelations(ctx, &c.Contact.Contact, tenantID)
		if relErr == nil {
			c.Contact = withRel
		}
	}

	return candidates, nil
}

// MergeContacts merges a duplicate contact into a primary contact scoped to a tenant.
// All activities, deals, tags, and custom fields are reassigned to the primary.
// The duplicate is soft-deleted by setting merged_into_id.
func (s *Service) MergeContacts(ctx context.Context, primaryID, duplicateID uuid.UUID, tenantID uuid.UUID) (*models.ContactWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	if primaryID == duplicateID {
		return nil, ErrCannotMergeSelf
	}

	// Verify both exist
	primary, err := s.repo.GetByID(ctx, primaryID, tenantID)
	if err != nil {
		return nil, ErrContactNotFound
	}
	if primary.MergedIntoID != nil {
		return nil, ErrAlreadyMerged
	}

	dup, err := s.repo.GetByID(ctx, duplicateID, tenantID)
	if err != nil {
		return nil, ErrContactNotFound
	}
	if dup.MergedIntoID != nil {
		return nil, ErrAlreadyMerged
	}

	if err := s.repo.MergeInto(ctx, primaryID, duplicateID, tenantID); err != nil {
		return nil, err
	}

	slog.Info("contacts merged",
		"primary_id", primaryID,
		"duplicate_id", duplicateID,
	)

	return s.getWithRelations(ctx, primary, tenantID)
}

// GetByEmail retrieves a contact by email address scoped to a tenant (used by import merge logic).
func (s *Service) GetByEmail(ctx context.Context, email string, tenantID uuid.UUID) (*models.Contact, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	return s.repo.GetByEmail(ctx, email, tenantID)
}

// CreateForImport creates a contact without the usual validation (used by bulk import).
func (s *Service) CreateForImport(ctx context.Context, contact *models.Contact) error {
	if contact.Visibility == "" {
		contact.Visibility = "shared"
	}
	if contact.TenantID == uuid.Nil {
		return ErrInvalidTenant
	}
	return s.repo.Create(ctx, contact)
}

// UpdateForImport updates a contact during import merge.
func (s *Service) UpdateForImport(ctx context.Context, contact *models.Contact) error {
	tenantID := contact.TenantID
	if tenantID == uuid.Nil {
		return ErrInvalidTenant
	}
	return s.repo.Update(ctx, contact, tenantID)
}

// ListByIDs retrieves contacts by a list of IDs scoped to a tenant (used by export).
func (s *Service) ListByIDs(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) ([]*models.Contact, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	return s.repo.ListByIDs(ctx, ids, tenantID)
}

// ListVisible retrieves all contacts visible to the given user scoped to a tenant (used by export all).
func (s *Service) ListVisible(ctx context.Context, userID uuid.UUID, isAdmin bool, tenantID uuid.UUID) ([]*models.Contact, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	return s.repo.ListAll(ctx, userID, isAdmin, tenantID)
}

// ListWithVisibility retrieves contacts with visibility filtering.
func (s *Service) ListWithVisibility(ctx context.Context, userID uuid.UUID, isAdmin bool, input ListInput) ([]*models.ContactWithRelations, int, error) {
	tenantID := input.TenantID
	if tenantID == uuid.Nil {
		return nil, 0, ErrInvalidTenant
	}

	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := (input.Page - 1) * input.PageSize
	filter := ListFilter{
		TenantID:         tenantID,
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

	results, enrichErr := s.enrichWithRelationsBatch(ctx, contacts, tenantID)
	if enrichErr != nil {
		return nil, 0, enrichErr
	}

	return results, total, nil
}

// UpdateVisibility updates the visibility and owner of a contact scoped to a tenant.
func (s *Service) UpdateVisibility(ctx context.Context, contactID uuid.UUID, visibility string, ownerID *uuid.UUID, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenant
	}
	return s.repo.UpdateVisibility(ctx, contactID, visibility, ownerID, tenantID)
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
