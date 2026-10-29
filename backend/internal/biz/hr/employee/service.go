// Package employee provides business logic for employee profile and document management.
// It enforces role-based field restrictions for profile updates (self-service for
// contact/address, admin/HR/manager for contract fields) and category-based document
// visibility (hr_only, manager, employee).
package employee

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
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
	// IsMinor drives the JArbSchG limits in the shift compliance check, so it
	// is an HR field — an employee must not be able to clear it on themselves.
	IsMinor *bool

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
	IsMinor               bool
}

// UploadDocumentInput contains the data needed to link an HR document.
type UploadDocumentInput struct {
	EmployeeID uuid.UUID
	CategoryID uuid.UUID
	// CategoryKey is the alternative to CategoryID for callers that only know
	// the slug (the Personalakte UI). Exactly one of the two must be set.
	CategoryKey string
	FileID      *uuid.UUID
	UploadedBy  uuid.UUID
	Notes       string
	Title       string
	FileName    string
	FileSize    string
	ExpiresAt   *time.Time
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

	// Zero values from optional request fields would override the column
	// defaults and violate chk_hr_work_days / chk_hr_contract_type, so
	// mirror the schema defaults here.
	if input.ContractType == "" {
		input.ContractType = "full_time"
	}
	if input.WorkDaysPerWeek == 0 {
		input.WorkDaysPerWeek = 5
	}
	if input.AnnualLeaveDays == 0 {
		input.AnnualLeaveDays = 20
	}
	if input.AddressCountry == "" {
		input.AddressCountry = "DE"
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
		IsMinor:               input.IsMinor,
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

	if input.AddressCountry != nil && len(*input.AddressCountry) != 2 {
		return nil, ErrInvalidCountry
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
	if input.IsMinor != nil {
		profile.IsMinor = *input.IsMinor
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

// SalaryDocumentCategoryKey is the hr_document_categories.key of salary
// statements (migration 000310). A manager reading a report's file does not
// see it without team:salary:view beyond "own".
const SalaryDocumentCategoryKey = "gehaltsabrechnung"

// managerRole is the role RLS policy hr_document_access grants the manager
// AND employee tier of every file in the tenant.
const managerRole = "manager"

// withoutManagerRole drops the manager role from the roles the database layer
// stamps into app.user_roles. hr_document_access knows no reporting line: with
// the role on the session a manager reads the manager and employee tier of
// every colleague. Every document access therefore runs without it, except
// ListReportingLineDocuments once it has checked that the employee reports to
// the caller. A manager on any other path sees exactly what a member sees.
func withoutManagerRole(ctx context.Context) context.Context {
	roles := middleware.GetUserRoles(ctx)
	kept := make([]string, 0, len(roles))
	for _, r := range roles {
		if r != managerRole {
			kept = append(kept, r)
		}
	}
	if len(kept) == len(roles) {
		return ctx
	}
	return context.WithValue(ctx, middleware.UserRolesKey, kept)
}

// ListEmployeeDocuments retrieves one employee's documents. Which of them the
// caller may see is decided by RLS policy hr_document_access from the roles on
// the session, so this carries no role parameter a caller could influence --
// exactly like ListPersonnelDocuments below. The manager role does not count
// here; a manager reads a report's file through ListReportingLineDocuments.
func (s *Service) ListEmployeeDocuments(ctx context.Context, employeeID uuid.UUID) ([]*models.EmployeeDocument, error) {
	return s.docRepo.ListByEmployee(withoutManagerRole(ctx), employeeID)
}

// ListReportingLineDocuments reads the file of managerID themselves or of one
// of their direct reports (hr_employee_profiles.manager_user_id). Anyone else,
// including an employee without a profile, is ErrInsufficientDocumentAccess,
// so the answer does not reveal who exists.
//
// The caller's own file is read like ListEmployeeDocuments. A report's file is
// read with the manager role on the session, so RLS adds the manager tier;
// hr_only stays hidden. hideSalary drops the gehaltsabrechnung category for a
// caller who may not see the team's salary statements.
//
// lean: post-filter on the category key, safe because ListByEmployee returns
// one unpaginated file; push it into the query once that list pages.
func (s *Service) ListReportingLineDocuments(ctx context.Context, tenantID, employeeID, managerID uuid.UUID, hideSalary bool) ([]*models.EmployeeDocument, error) {
	if employeeID == managerID {
		return s.ListEmployeeDocuments(ctx, employeeID)
	}
	emp, err := s.employeeRepo.GetByUserID(ctx, employeeID)
	if errors.Is(err, ErrEmployeeNotFound) {
		return nil, ErrInsufficientDocumentAccess
	}
	if err != nil {
		return nil, err
	}
	if emp.TenantID != tenantID || emp.ManagerUserID == nil || *emp.ManagerUserID != managerID {
		return nil, ErrInsufficientDocumentAccess
	}

	docs, err := s.docRepo.ListByEmployee(ctx, employeeID)
	if err != nil || !hideSalary {
		return docs, err
	}
	kept := make([]*models.EmployeeDocument, 0, len(docs))
	for _, d := range docs {
		if d.CategoryKey != SalaryDocumentCategoryKey {
			kept = append(kept, d)
		}
	}
	return kept, nil
}

// ListPersonnelDocuments retrieves every personnel document of the tenant the
// caller is allowed to see. The visibility tiers are enforced by RLS policy
// hr_document_access, which reads the caller's roles from the app.user_roles
// GUC — so this deliberately carries no role parameter that a caller could
// influence, unlike ListEmployeeDocuments. Without the manager role (see
// withoutManagerRole) a manager gets their own employee-tier documents here.
func (s *Service) ListPersonnelDocuments(ctx context.Context, tenantID uuid.UUID) ([]*models.EmployeeDocument, error) {
	return s.docRepo.ListByTenant(withoutManagerRole(ctx), tenantID)
}

// GetEmployeeDocument reads a single personnel document, tenant-scoped and
// subject to the same RLS visibility tiers as the list.
func (s *Service) GetEmployeeDocument(ctx context.Context, tenantID, id uuid.UUID) (*models.EmployeeDocument, error) {
	return s.docRepo.GetByID(withoutManagerRole(ctx), tenantID, id)
}

// UploadEmployeeDocument creates an HR document record linking a file to an employee.
// The actual file upload happens via the document service; this creates the HR metadata link.
func (s *Service) UploadEmployeeDocument(ctx context.Context, tenantID uuid.UUID, input UploadDocumentInput) (*models.EmployeeDocument, error) {
	// Validate the category exists for this tenant — category_id/key come from
	// the client, so a foreign tenant's id must not resolve here.
	category, catErr := s.resolveDocumentCategory(ctx, tenantID, input)
	if catErr != nil {
		return nil, catErr
	}

	if input.EmployeeID == uuid.Nil {
		return nil, ErrEmployeeRequired
	}

	now := time.Now()
	doc := &models.EmployeeDocument{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EmployeeID: input.EmployeeID,
		CategoryID: category.ID,
		FileID:     input.FileID,
		UploadedBy: input.UploadedBy,
		Notes:      input.Notes,
		Title:      input.Title,
		FileName:   input.FileName,
		FileSize:   input.FileSize,
		ExpiresAt:  input.ExpiresAt,
		CreatedAt:  now,
		// Denormalized so the caller gets the same shape the list returns
		// without a second round trip.
		CategoryName: category.Name,
		CategoryKey:  category.Key,
		Visibility:   string(category.Visibility),
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

// resolveDocumentCategory accepts either the category id or its slug and
// returns the row, so the tenant check happens exactly once regardless of
// which the caller supplied.
func (s *Service) resolveDocumentCategory(ctx context.Context, tenantID uuid.UUID, input UploadDocumentInput) (*models.HRDocumentCategory, error) {
	if input.CategoryID != uuid.Nil {
		return s.docCatRepo.GetByID(ctx, tenantID, input.CategoryID)
	}
	if input.CategoryKey != "" {
		return s.docCatRepo.GetByKey(ctx, tenantID, input.CategoryKey)
	}
	return nil, ErrDocumentCategoryNotFound
}

// DeleteEmployeeDocument deletes an HR document link.
// hr_document_access has no DELETE clause of its own, so its USING tiers
// apply: with the manager role a manager could delete colleagues' documents.
func (s *Service) DeleteEmployeeDocument(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.docRepo.Delete(withoutManagerRole(ctx), tenantID, id)
}

// ListDocumentCategories retrieves document categories for a tenant, filtered
// to the visibility tiers callerScope (the caller's team:documents:view
// permission scope: "own", "team", or "all") permits — the same tiers RLS
// migration 000127 enforces on the documents themselves. A caller who cannot
// see hr_only documents must not learn hr_only categories exist either.
func (s *Service) ListDocumentCategories(ctx context.Context, tenantID uuid.UUID, callerScope string) ([]*models.HRDocumentCategory, error) {
	categories, err := s.docCatRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return filterCategoriesByScope(categories, callerScope), nil
}

// filterCategoriesByScope mirrors the visibility tiers of RLS policy
// hr_document_access: scope "all" (admin/hr_admin) sees every category, "team"
// (manager) sees manager+employee, and "own" (or any other/empty value) sees
// employee-visibility only. This one filters in Go because categories are a
// catalogue read, not a document read -- the policy does not cover them, and
// the scope here comes from the route guard rather than from the caller.
func filterCategoriesByScope(categories []*models.HRDocumentCategory, callerScope string) []*models.HRDocumentCategory {
	if callerScope == auth.ScopeAll {
		return categories
	}

	allowed := map[models.HRDocVisibility]bool{models.HRDocVisibilityEmployee: true}
	if callerScope == auth.ScopeTeam {
		allowed[models.HRDocVisibilityManager] = true
	}

	filtered := make([]*models.HRDocumentCategory, 0, len(categories))
	for _, c := range categories {
		if allowed[c.Visibility] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// ============================================================================
// Offboarding
// ============================================================================

// OffboardInput is the request side of an exit, straight from the offboard
// dialog. Backfill is recorded for the audit trail only.
type OffboardInput struct {
	LastWorkDay     *time.Time
	ExitDate        time.Time
	ExitType        string
	Reason          string
	Backfill        bool
	SuccessorUserID *uuid.UUID
}

// OffboardEmployee ends an employment: the personnel record goes inactive, the
// account loses its login and its roles, its seat comes back, and the direct
// reports move to the successor. The cascade itself is one transaction in the
// repository — everything here decides whether it may run at all.
func (s *Service) OffboardEmployee(
	ctx context.Context,
	tenantID, employeeID, actorUserID uuid.UUID,
	in OffboardInput,
) (*models.EmployeeProfile, error) {
	if !models.ValidExitType(in.ExitType) {
		return nil, ErrInvalidExitType
	}
	if in.LastWorkDay != nil && in.ExitDate.Before(*in.LastWorkDay) {
		return nil, ErrExitBeforeLastWorkDay
	}

	profile, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if profile.Status == models.EmployeeStatusInactive {
		return nil, ErrAlreadyOffboarded
	}

	// Nobody offboards themselves. The session stays valid until it expires, so
	// the mistake would only surface at the next login — the same reasoning
	// behind auth.ErrSelfDeactivation.
	if profile.UserID == actorUserID {
		return nil, ErrSelfOffboard
	}

	// A tenant that loses its last role administrator cannot give the right
	// back to itself.
	remaining, err := s.employeeRepo.CountOtherActiveRoleAdmins(ctx, profile.UserID)
	if err != nil {
		return nil, err
	}
	if remaining == 0 {
		return nil, ErrLastRoleAdmin
	}

	// Orphaned reports are the point of this guard: approvals would otherwise
	// hang off a locked account.
	reports, err := s.employeeRepo.CountDirectReports(ctx, tenantID, profile.UserID)
	if err != nil {
		return nil, err
	}
	successor, err := s.resolveSuccessor(ctx, profile, reports, in.SuccessorUserID)
	if err != nil {
		return nil, err
	}

	updated, err := s.employeeRepo.Offboard(ctx, OffboardWrite{
		TenantID:            tenantID,
		EmployeeID:          employeeID,
		UserID:              profile.UserID,
		LastWorkDay:         in.LastWorkDay,
		ExitDate:            in.ExitDate,
		ExitType:            in.ExitType,
		ExitReason:          in.Reason,
		SuccessorUserID:     successor,
		LeaverManagerUserID: profile.ManagerUserID,
	})
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "employee offboarded",
		"tenant_id", tenantID, "employee_id", employeeID, "user_id", profile.UserID,
		"actor_id", actorUserID, "exit_type", in.ExitType, "backfill", in.Backfill,
		"reports_reassigned", reports,
	)
	return updated, nil
}

// ReactivateEmployee brings an offboarded employee back: active personnel
// record, unlocked login, base role member. The cascade is one transaction in
// the repository, which also refuses when no seat is left.
func (s *Service) ReactivateEmployee(ctx context.Context, tenantID, employeeID, actorUserID uuid.UUID) (*models.EmployeeProfile, error) {
	profile, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if profile.Status != models.EmployeeStatusInactive {
		return nil, ErrNotOffboarded
	}

	updated, err := s.employeeRepo.Reactivate(ctx, tenantID, employeeID, profile.UserID)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "employee reactivated",
		"tenant_id", tenantID, "employee_id", employeeID, "user_id", profile.UserID,
		"actor_id", actorUserID,
	)
	return updated, nil
}

// resolveSuccessor decides whether a successor is needed and whether the named
// one may take over. Returns nil when the leaver has no reports — naming a
// successor without reports is harmless and simply has no effect.
func (s *Service) resolveSuccessor(
	ctx context.Context,
	profile *models.EmployeeProfile,
	reports int,
	successorUserID *uuid.UUID,
) (*uuid.UUID, error) {
	if reports == 0 {
		return nil, nil
	}
	if successorUserID == nil {
		return nil, ErrSuccessorRequired
	}
	if *successorUserID == profile.UserID {
		return nil, ErrInvalidSuccessor
	}

	// The successor has to be a real, active employee of this tenant: RLS scopes
	// the lookup, and an inactive one would inherit a team they can no longer
	// log in to manage.
	successor, err := s.employeeRepo.GetByUserID(ctx, *successorUserID)
	if err != nil {
		if errors.Is(err, ErrEmployeeNotFound) {
			return nil, ErrInvalidSuccessor
		}
		return nil, err
	}
	if successor.TenantID != profile.TenantID || successor.Status != models.EmployeeStatusActive {
		return nil, ErrInvalidSuccessor
	}
	return successorUserID, nil
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
		input.StartDate != nil ||
		input.IsMinor != nil
}
