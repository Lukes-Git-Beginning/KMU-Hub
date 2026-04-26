package einkauf

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListSuppliersFilter contains filtering options for listing suppliers.
type ListSuppliersFilter struct {
	Search     string
	ActiveOnly bool // excludes soft-deleted suppliers
}

// ListPOsFilter contains filtering options for listing purchase orders.
type ListPOsFilter struct {
	SupplierID *uuid.UUID
	Status     *POStatus
	DateFrom   *time.Time
	DateTo     *time.Time
}

// Repository defines the interface for einkauf persistence.
type Repository interface {
	// Suppliers
	CreateSupplier(ctx context.Context, s *Supplier) error
	UpdateSupplier(ctx context.Context, s *Supplier) error
	DeleteSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) error // soft delete
	GetSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (*Supplier, error)
	ListSuppliers(ctx context.Context, tenantID uuid.UUID, filter ListSuppliersFilter, offset, limit int) ([]*Supplier, int, error)

	// Purchase Orders
	CreatePO(ctx context.Context, po *PurchaseOrder) error
	UpdatePO(ctx context.Context, po *PurchaseOrder) error
	DeletePO(ctx context.Context, tenantID, poID uuid.UUID) error // only draft
	GetPO(ctx context.Context, tenantID, poID uuid.UUID) (*PurchaseOrder, error)
	GetPOWithLines(ctx context.Context, tenantID, poID uuid.UUID) (*PurchaseOrder, error)
	ListPOs(ctx context.Context, tenantID uuid.UUID, filter ListPOsFilter, offset, limit int) ([]*PurchaseOrder, int, error)
	UpdatePOStatus(ctx context.Context, tenantID, poID uuid.UUID, status POStatus) error
	PONumberExists(ctx context.Context, tenantID uuid.UUID, poNumber string, excludeID *uuid.UUID) (bool, error)

	// PO Lines
	CreatePOLine(ctx context.Context, line *POLine) error
	UpdatePOLine(ctx context.Context, line *POLine) error
	DeletePOLine(ctx context.Context, tenantID, lineID uuid.UUID) error
	GetPOLine(ctx context.Context, tenantID, lineID uuid.UUID) (*POLine, error)
	ListPOLines(ctx context.Context, tenantID, poID uuid.UUID) ([]*POLine, error)
	UpdatePOLineReceivedQuantity(ctx context.Context, tenantID, lineID uuid.UUID, receivedQty string) error
	CountPOLines(ctx context.Context, tenantID, poID uuid.UUID) (int, error)
}
