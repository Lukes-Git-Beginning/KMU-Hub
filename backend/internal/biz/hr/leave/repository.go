package leave

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// LeaveRequestRepository defines the interface for leave request persistence.
type LeaveRequestRepository interface {
	Create(ctx context.Context, req *models.LeaveRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.LeaveRequest, error)
	List(ctx context.Context, filter LeaveRequestFilter) ([]*models.LeaveRequest, int, error)
	Update(ctx context.Context, req *models.LeaveRequest) error
	FindOverlaps(ctx context.Context, employeeID uuid.UUID, startDate, endDate time.Time, excludeID *uuid.UUID) ([]*models.LeaveRequest, error)
}

// LeaveBalanceRepository defines the interface for leave balance persistence.
type LeaveBalanceRepository interface {
	GetByEmployeeYear(ctx context.Context, tenantID, employeeID uuid.UUID, year int) (*models.HRLeaveBalance, error)
	Upsert(ctx context.Context, balance *models.HRLeaveBalance) error
}

// LeaveTypeRepository defines the interface for leave type persistence.
type LeaveTypeRepository interface {
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.LeaveType, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.LeaveType, error)
	GetByKey(ctx context.Context, tenantID uuid.UUID, key string) (*models.LeaveType, error)
}

// HRSettingsRepository defines the interface for HR company settings persistence.
type HRSettingsRepository interface {
	GetByTenant(ctx context.Context, tenantID uuid.UUID) (*models.HRCompanySettings, error)
	Upsert(ctx context.Context, settings *models.HRCompanySettings) error
}

// LeaveRequestFilter contains filtering options for listing leave requests.
type LeaveRequestFilter struct {
	TenantID      uuid.UUID
	EmployeeID    *uuid.UUID
	Status        *string
	StartDateFrom *time.Time
	StartDateTo   *time.Time
	Page          int
	PerPage       int
}
