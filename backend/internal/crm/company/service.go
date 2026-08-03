package company

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles company business logic
type Service struct {
	repo Repository
}

// NewService creates a new company service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a company
type CreateInput struct {
	Name          string
	Domain        *string
	Industry      *string
	EmployeeCount *int
	Address       *string
	City          *string
	Country       *string
	Notes         *string
	TagIDs        []uuid.UUID
	CustomFields  map[uuid.UUID]any // field_id -> value
	CreatedBy     uuid.UUID
	TenantID      uuid.UUID
}

// Create creates a new company
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.CompanyWithRelations, error) {
	tenantID := input.TenantID
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}

	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrNameRequired
	}

	// Validate tags are for companies
	for _, tagID := range input.TagIDs {
		exists, err := s.repo.TagExists(ctx, tagID, models.EntityTypeCompany)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrTagNotFound
		}
	}

	company := &models.Company{
		ID:            uuid.New(),
		Name:          name,
		Domain:        trimStringPtr(input.Domain),
		Industry:      trimStringPtr(input.Industry),
		EmployeeCount: input.EmployeeCount,
		Address:       trimStringPtr(input.Address),
		City:          trimStringPtr(input.City),
		Country:       trimStringPtr(input.Country),
		Notes:         input.Notes,
		TenantID:      tenantID,
		CreatedBy:     input.CreatedBy,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(ctx, company); err != nil {
		return nil, err
	}

	// Add tags
	if len(input.TagIDs) > 0 {
		if err := s.repo.AddTags(ctx, company.ID, input.TagIDs); err != nil {
			return nil, err
		}
	}

	// Set custom field values
	if len(input.CustomFields) > 0 {
		if err := s.repo.SetCustomFieldValues(ctx, company.ID, input.CustomFields); err != nil {
			return nil, err
		}
	}

	slog.Info("company created",
		"company_id", company.ID,
		"name", company.Name,
	)

	return s.getWithRelations(ctx, company, tenantID)
}

// FindOrCreateByName finds a non-merged company by case-insensitive name match within a
// tenant, or creates a bare company (name only) if none exists. Used by contact import,
// where a company name string is the only signal available and re-running the same import
// must not create duplicate companies.
func (s *Service) FindOrCreateByName(ctx context.Context, tenantID uuid.UUID, name string, createdBy uuid.UUID) (*models.Company, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNameRequired
	}

	existing, err := s.repo.GetByName(ctx, tenantID, name)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrCompanyNotFound) {
		return nil, err
	}

	company := &models.Company{
		ID:        uuid.New(),
		Name:      name,
		TenantID:  tenantID,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, company); err != nil {
		return nil, err
	}

	slog.Info("company created via find-or-create", "company_id", company.ID, "name", company.Name)

	return company, nil
}

// GetNamesByIDs returns company names keyed by ID, scoped to a tenant.
func (s *Service) GetNamesByIDs(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) (map[uuid.UUID]string, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	return s.repo.GetNamesByIDs(ctx, ids, tenantID)
}

// GetByID retrieves a company by ID scoped to a tenant
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.CompanyWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	company, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, ErrCompanyNotFound
	}
	return s.getWithRelations(ctx, company, tenantID)
}

// ListInput contains filtering options for listing companies
type ListInput struct {
	Search   string
	Industry *string
	TagIDs   []uuid.UUID
	Page     int
	PageSize int
	SortBy   string
	SortDesc bool
	TenantID uuid.UUID
}

// List retrieves companies with optional filtering
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.CompanyWithRelations, int, error) {
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
		TenantID: tenantID,
		Search:   input.Search,
		Industry: input.Industry,
		TagIDs:   input.TagIDs,
		SortBy:   input.SortBy,
		SortDesc: input.SortDesc,
	}

	companies, total, err := s.repo.List(ctx, filter, offset, input.PageSize)
	if err != nil {
		return nil, 0, err
	}

	var results []*models.CompanyWithRelations
	for _, c := range companies {
		withRel, relErr := s.getWithRelations(ctx, c, tenantID)
		if relErr != nil {
			return nil, 0, relErr
		}
		results = append(results, withRel)
	}

	return results, total, nil
}

// UpdateInput contains the data that can be updated on a company
type UpdateInput struct {
	Name          *string
	Domain        *string
	Industry      *string
	EmployeeCount *int
	Address       *string
	City          *string
	Country       *string
	Notes         *string
	CustomFields  map[uuid.UUID]any
}

// Update updates an existing company scoped to a tenant
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput, tenantID uuid.UUID) (*models.CompanyWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	company, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, ErrCompanyNotFound
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrNameRequired
		}
		company.Name = name
	}

	if input.Domain != nil {
		company.Domain = trimStringPtr(input.Domain)
	}
	if input.Industry != nil {
		company.Industry = trimStringPtr(input.Industry)
	}
	if input.EmployeeCount != nil {
		company.EmployeeCount = input.EmployeeCount
	}
	if input.Address != nil {
		company.Address = trimStringPtr(input.Address)
	}
	if input.City != nil {
		company.City = trimStringPtr(input.City)
	}
	if input.Country != nil {
		company.Country = trimStringPtr(input.Country)
	}
	if input.Notes != nil {
		company.Notes = input.Notes
	}

	company.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, company, tenantID); updateErr != nil {
		return nil, updateErr
	}

	// Update custom fields if provided
	if len(input.CustomFields) > 0 {
		if cfErr := s.repo.SetCustomFieldValues(ctx, company.ID, input.CustomFields); cfErr != nil {
			return nil, cfErr
		}
	}

	slog.Info("company updated", "company_id", company.ID)

	return s.getWithRelations(ctx, company, tenantID)
}

// Delete removes a company scoped to a tenant
func (s *Service) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenant
	}
	company, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return ErrCompanyNotFound
	}

	// Check if company has contacts
	hasContacts, err := s.repo.HasContacts(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if hasContacts {
		return ErrCompanyInUse
	}

	if deleteErr := s.repo.Delete(ctx, id, tenantID); deleteErr != nil {
		return deleteErr
	}

	slog.Info("company deleted",
		"company_id", company.ID,
		"name", company.Name,
	)

	return nil
}

// AddTags adds tags to a company scoped to a tenant
func (s *Service) AddTags(ctx context.Context, companyID uuid.UUID, tagIDs []uuid.UUID, tenantID uuid.UUID) (*models.CompanyWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	company, err := s.repo.GetByID(ctx, companyID, tenantID)
	if err != nil {
		return nil, ErrCompanyNotFound
	}

	// Validate tags
	for _, tagID := range tagIDs {
		exists, tagErr := s.repo.TagExists(ctx, tagID, models.EntityTypeCompany)
		if tagErr != nil {
			return nil, tagErr
		}
		if !exists {
			return nil, ErrTagNotFound
		}
	}

	if addErr := s.repo.AddTags(ctx, companyID, tagIDs); addErr != nil {
		return nil, addErr
	}

	return s.getWithRelations(ctx, company, tenantID)
}

// RemoveTags removes tags from a company scoped to a tenant
func (s *Service) RemoveTags(ctx context.Context, companyID uuid.UUID, tagIDs []uuid.UUID, tenantID uuid.UUID) (*models.CompanyWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	company, err := s.repo.GetByID(ctx, companyID, tenantID)
	if err != nil {
		return nil, ErrCompanyNotFound
	}

	if removeErr := s.repo.RemoveTags(ctx, companyID, tagIDs); removeErr != nil {
		return nil, removeErr
	}

	return s.getWithRelations(ctx, company, tenantID)
}

// getWithRelations loads all relations for a company
func (s *Service) getWithRelations(ctx context.Context, company *models.Company, tenantID uuid.UUID) (*models.CompanyWithRelations, error) {
	result := &models.CompanyWithRelations{
		Company: *company,
	}

	// Load contact count
	count, _ := s.repo.GetContactCount(ctx, company.ID, tenantID)
	result.ContactCount = count

	// Load tags
	tags, _ := s.repo.GetTags(ctx, company.ID)
	result.Tags = tags

	// Load custom field values
	values, _ := s.repo.GetCustomFieldValues(ctx, company.ID)
	if len(values) > 0 {
		result.CustomFields = make(map[string]any)
		for _, v := range values {
			result.CustomFields[v.FieldName] = v.Value
		}
	}

	return result, nil
}

// DuplicateCandidate represents a potential duplicate company with similarity scoring.
type DuplicateCandidate struct {
	Company    *models.CompanyWithRelations `json:"company"`
	Similarity float64                      `json:"similarity"`
	MatchType  string                       `json:"match_type"` // "domain_exact", "name_fuzzy"
}

// FindDuplicates returns potential duplicate companies for the given company ID scoped to a tenant.
func (s *Service) FindDuplicates(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) ([]*DuplicateCandidate, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	if _, err := s.repo.GetByID(ctx, companyID, tenantID); err != nil {
		return nil, ErrCompanyNotFound
	}

	candidates, err := s.repo.FindDuplicateCandidates(ctx, companyID, tenantID)
	if err != nil {
		return nil, err
	}

	for _, c := range candidates {
		withRel, relErr := s.getWithRelations(ctx, &c.Company.Company, tenantID)
		if relErr == nil {
			c.Company = withRel
		}
	}

	return candidates, nil
}

// MergeCompanies merges a duplicate company into a primary company scoped to a tenant.
func (s *Service) MergeCompanies(ctx context.Context, primaryID, duplicateID uuid.UUID, tenantID uuid.UUID) (*models.CompanyWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	if primaryID == duplicateID {
		return nil, ErrCannotMergeSelf
	}

	primary, err := s.repo.GetByID(ctx, primaryID, tenantID)
	if err != nil {
		return nil, ErrCompanyNotFound
	}
	if primary.MergedIntoID != nil {
		return nil, ErrAlreadyMerged
	}

	dup, err := s.repo.GetByID(ctx, duplicateID, tenantID)
	if err != nil {
		return nil, ErrCompanyNotFound
	}
	if dup.MergedIntoID != nil {
		return nil, ErrAlreadyMerged
	}

	if err := s.repo.MergeInto(ctx, primaryID, duplicateID, tenantID); err != nil {
		return nil, err
	}

	slog.Info("companies merged",
		"primary_id", primaryID,
		"duplicate_id", duplicateID,
	)

	return s.getWithRelations(ctx, primary, tenantID)
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
