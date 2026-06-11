// Package employee provides business logic for employee profile and document management.
// It enforces role-based field restrictions for profile updates (self-service for
// contact/address, admin/HR/manager for contract fields) and category-based document
// visibility (hr_only, manager, employee).
package employee

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles employee profile and document business logic.
type Service struct {
	employeeRepo EmployeeRepository
	docCatRepo   DocumentCategoryRepository
	docRepo      EmployeeDocumentRepository
}

// NewService creates a new employee service.
func NewService(
	employeeRepo EmployeeRepository,
	docCatRepo DocumentCategoryRepository,
	docRepo EmployeeDocumentRepository,
) *Service {
	return &Service{
		employeeRepo: employeeRepo,
		docCatRepo:   docCatRepo,
		docRepo:      docRepo,
	}
}

// UpdateEmployeeInput contains all fields that can be updated on an employee profile.
// Pointer fields mean "nil = no change".
type UpdateEmployeeInput struct {
	// HR/admin/manager fields
	Department      *string
	PositionTitle   *string
	ContractType    *models.HRContractType
	WorkDaysPerWeek *int
	AnnualLeaveDays *int
	ManagerUserID   *uuid.UUID
	StartDate       *time.Time

	// Self-service fields (employee can update)
	EmergencyContactName  *string
	EmergencyContactPhone *string
	AddressStreet         *string
	AddressCity           *string
	AddressPostalCode     *string
	AddressCountry        *string
}

// SelfProfileInput contains only the fields an employee can update themselves.
type SelfProfileInput struct {
	EmergencyContactName  *string
	EmergencyContactPhone *string
	AddressStreet         *string
	AddressCity           *string
	AddressPostalCode     *string
	AddressCountry        *string
}

// CreateEmployeeInput contains the data required to create a new employee profile.
// It assumes the auth user already exists; only the HR profile row is created here.
type CreateEmployeeInput struct {
	TenantID              uuid.UUID
	UserID                uuid.UUID
	Department            string
	PositionTitle         string
	ContractType          models.HRContractType
	WorkDaysPerWeek       int
	AnnualLeaveDays       int
	ManagerUserID         *uuid.UUID
	StartDate             time.Time
	EmergencyContactName  string
	EmergencyContactPhone string
	AddressStreet         string
	AddressCity           string
	AddressPostalCode     string
	AddressCountry        string
}

// UploadDocumentInput contains the data needed to link an HR document.
type UploadDocumentInput struct {
	EmployeeID uuid.UUID
	CategoryID uuid.UUID
	FileID     uuid.UUID
	UploadedBy uuid.UUID
	Notes      string
}

// ============================================================================
// Employee Profile Operations
// ============================================================================

// ListEmployees retrieves employee profiles with optional filtering.
func (s *Service) ListEmployees(ctx context.Context, filter EmployeeFilter) ([]*models.EmployeeProfile, int, error) {
	return s.employeeRepo.List(ctx, filter)
}

// GetEmployee retrieves an employee profile by ID.
func (s *Service) GetEmployee(ctx context.Context, id uuid.UUID) (*models.EmployeeProfile, error) {
	return s.employeeRepo.GetByID(ctx, id)
}

// GetByUserID retrieves an employee profile by user ID.
func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmployeeProfile, error) {
	return s.employeeRepo.GetByUserID(ctx, userID)
}

// CreateEmployee creates a new employee profile for an existing auth user.
// The caller is responsible for ensuring the user exists in the auth service.
// Returns ErrProfileAlreadyExists if an employee profile already exists for that user in this tenant.
func (s *Service) CreateEmployee(ctx context.Context, input CreateEmployeeInput) (*models.EmployeeProfile, error) {
	// Check for duplicate
	existing, err := s.employeeRepo.GetByUserID(ctx, input.UserID)
	if err == nil && existing != nil {
		return nil, ErrProfileAlreadyExists
	}

	now := time.Now()
	profile := &models.EmployeeProfile{
		ID:                    uuid.New(),
		TenantID:              input.TenantID,
		UserID:                input.UserID,
		Department:            input.Department,
		PositionTitle:         input.PositionTitle,
		ContractType:          input.ContractType,
		WorkDaysPerWeek:       input.WorkDaysPerWeek,
		AnnualLeaveDays:       input.AnnualLeaveDays,
		ManagerUserID:         input.ManagerUserID,
		StartDate:             input.StartDate,
		EmergencyContactName:  input.EmergencyContactName,
		EmergencyContactPhone: input.EmergencyContactPhone,
		AddressStreet:         input.AddressStreet,
		AddressCity:           input.AddressCity,
		AddressPostalCode:     input.AddressPostalCode,
		AddressCountry:        input.AddressCountry,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if createErr := s.employeeRepo.Create(ctx, profile); createErr != nil {
		return nil, fmt.Errorf("create employee profile: %w", createErr)
	}

	slog.Info("employee profile created",
		"employee_id", profile.ID,
		"user_id", profile.UserID,
		"tenant_id", profile.TenantID,
	)
	return profile, nil
}

// UpdateEmployee updates an employee profile with role-based field restrictions.
// callerRole must be one of: "admin", "hr", "manager", "employee".
//
// If callerRole is "employee": only self-service fields (emergency contact, address) are allowed.
// If callerRole is "admin", "hr", or "manager": all fields are allowed.
func (s *Service) UpdateEmployee(ctx context.Context, id uuid.UUID, input UpdateEmployeeInput, callerRole string) (*models.EmployeeProfile, error) {
	profile, err := s.employeeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check role-based field restrictions
	if callerRole == "employee" {
		if hasRestrictedFields(input) {
			return nil, ErrUnauthorizedFieldUpdate
		}
	}

	// Apply updates
	if input.Department != nil {
		profile.Department = *input.Department
	}
	if input.PositionTitle != nil {
		profile.PositionTitle = *input.PositionTitle
	}
	if input.ContractType != nil {
		profile.ContractType = *input.ContractType
	}
	if input.WorkDaysPerWeek != nil {
		profile.WorkDaysPerWeek = *input.WorkDaysPerWeek
	}
	if input.AnnualLeaveDays != nil {
		profile.AnnualLeaveDays = *input.AnnualLeaveDays
	}
	if input.ManagerUserID != nil {
		profile.ManagerUserID = input.ManagerUserID
	}
	if input.StartDate != nil {
		profile.StartDate = *input.StartDate
	}
	if input.EmergencyContactName != nil {
		profile.EmergencyContactName = *input.EmergencyContactName
	}
	if input.EmergencyContactPhone != nil {
		profile.EmergencyContactPhone = *input.EmergencyContactPhone
	}
	if input.AddressStreet != nil {
		profile.AddressStreet = *input.AddressStreet
	}
	if input.AddressCity != nil {
		profile.AddressCity = *input.AddressCity
	}
	if input.AddressPostalCode != nil {
		profile.AddressPostalCode = *input.AddressPostalCode
	}
	if input.AddressCountry != nil {
		profile.AddressCountry = *input.AddressCountry
	}

	profile.UpdatedAt = time.Now()

	if updateErr := s.employeeRepo.Update(ctx, profile); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("employee profile updated",
		"employee_id", profile.ID,
		"user_id", profile.UserID,
		"caller_role", callerRole,
	)

	return profile, nil
}

// UpdateSelfProfile is a convenience method for employee self-service profile updates.
// Looks up the profile by userID, then calls UpdateEmployee with role="employee".
func (s *Service) UpdateSelfProfile(ctx context.Context, userID uuid.UUID, input SelfProfileInput) (*models.EmployeeProfile, error) {
	profile, err := s.employeeRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	updateInput := UpdateEmployeeInput{
		EmergencyContactName:  input.EmergencyContactName,
		EmergencyContactPhone: input.EmergencyContactPhone,
		AddressStreet:         input.AddressStreet,
		AddressCity:           input.AddressCity,
		AddressPostalCode:     input.AddressPostalCode,
		AddressCountry:        input.AddressCountry,
	}

	return s.UpdateEmployee(ctx, profile.ID, updateInput, "employee")
}

// ============================================================================
// Document Operations
// ============================================================================

// ListEmployeeDocuments retrieves documents for an employee filtered by caller role visibility.
func (s *Service) ListEmployeeDocuments(ctx context.Context, employeeID, callerUserID uuid.UUID, callerRole string) ([]*models.EmployeeDocument, error) {
	return s.docRepo.ListByEmployee(ctx, employeeID, callerRole)
}

// UploadEmployeeDocument creates an HR document record linking a file to an employee.
// The actual file upload happens via the document service; this creates the HR metadata link.
func (s *Service) UploadEmployeeDocument(ctx context.Context, tenantID uuid.UUID, input UploadDocumentInput) (*models.EmployeeDocument, error) {
	// Validate category exists
	_, catErr := s.docCatRepo.GetByID(ctx, input.CategoryID)
	if catErr != nil {
		return nil, catErr
	}

	now := time.Now()
	doc := &models.EmployeeDocument{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EmployeeID: input.EmployeeID,
		CategoryID: input.CategoryID,
		FileID:     input.FileID,
		UploadedBy: input.UploadedBy,
		Notes:      input.Notes,
		CreatedAt:  now,
	}

	if createErr := s.docRepo.Create(ctx, doc); createErr != nil {
		return nil, createErr
	}

	slog.Info("employee document uploaded",
		"document_id", doc.ID,
		"employee_id", doc.EmployeeID,
		"category_id", doc.CategoryID,
		"file_id", doc.FileID,
	)

	return doc, nil
}

// DeleteEmployeeDocument deletes an HR document link.
func (s *Service) DeleteEmployeeDocument(ctx context.Context, id uuid.UUID) error {
	return s.docRepo.Delete(ctx, id)
}

// ListDocumentCategories retrieves all document categories for a tenant.
func (s *Service) ListDocumentCategories(ctx context.Context, tenantID uuid.UUID) ([]*models.HRDocumentCategory, error) {
	return s.docCatRepo.ListByTenant(ctx, tenantID)
}

// ============================================================================
// Internal helpers
// ============================================================================

// hasRestrictedFields checks if any HR-only fields are set in the input.
func hasRestrictedFields(input UpdateEmployeeInput) bool {
	return input.Department != nil ||
		input.PositionTitle != nil ||
		input.ContractType != nil ||
		input.WorkDaysPerWeek != nil ||
		input.AnnualLeaveDays != nil ||
		input.ManagerUserID != nil ||
		input.StartDate != nil
}
