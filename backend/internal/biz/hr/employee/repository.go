package employee

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// EmployeeRepository defines the interface for employee profile persistence.
type EmployeeRepository interface {
	Create(ctx context.Context, profile *models.EmployeeProfile) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.EmployeeProfile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmployeeProfile, error)
	List(ctx context.Context, filter EmployeeFilter) ([]*models.EmployeeProfile, int, error)
	Update(ctx context.Context, profile *models.EmployeeProfile) error
}

// DocumentCategoryRepository defines the interface for HR document category persistence.
// Both reads carry the tenant explicitly; they resolve the tenant's own categories
// plus the system seeds (see PostgresDocCategoryRepo).
type DocumentCategoryRepository interface {
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.HRDocumentCategory, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.HRDocumentCategory, error)
}

// EmployeeDocumentRepository defines the interface for employee document persistence.
type EmployeeDocumentRepository interface {
	Create(ctx context.Context, doc *models.EmployeeDocument) error
	ListByEmployee(ctx context.Context, employeeID uuid.UUID, callerRole string) ([]*models.EmployeeDocument, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// EmployeeFilter contains filtering options for listing employees.
type EmployeeFilter struct {
	TenantID      uuid.UUID
	Department    string     // Optional
	ManagerUserID *uuid.UUID // Optional: filter by manager
	Page          int
	PerPage       int
}
