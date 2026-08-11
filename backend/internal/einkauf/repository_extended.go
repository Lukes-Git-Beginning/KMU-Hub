package einkauf

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryExtended extends the base Repository with catalog, supplier ratings,
// and framework contract persistence.
type RepositoryExtended interface {
	Repository

	// -------------------------------------------------------------------------
	// Catalog Items
	// -------------------------------------------------------------------------

	CreateCatalogItem(ctx context.Context, item *CatalogItem) error
	UpdateCatalogItem(ctx context.Context, item *CatalogItem) error
	DeleteCatalogItem(ctx context.Context, tenantID, itemID uuid.UUID) error
	GetCatalogItem(ctx context.Context, tenantID, itemID uuid.UUID) (*CatalogItem, error)
	ListCatalogItems(ctx context.Context, tenantID uuid.UUID, filter ListCatalogItemsFilter, offset, limit int) ([]*CatalogItem, int, error)

	// -------------------------------------------------------------------------
	// Supplier Ratings
	// -------------------------------------------------------------------------

	CreateSupplierRating(ctx context.Context, rating *SupplierRating) error
	DeleteSupplierRating(ctx context.Context, tenantID, ratingID uuid.UUID) error
	GetSupplierRating(ctx context.Context, tenantID, ratingID uuid.UUID) (*SupplierRating, error)
	ListSupplierRatings(ctx context.Context, tenantID, supplierID uuid.UUID) ([]*SupplierRating, error)

	// -------------------------------------------------------------------------
	// Framework Contracts
	// -------------------------------------------------------------------------

	CreateFrameworkContract(ctx context.Context, fc *FrameworkContract) error
	UpdateFrameworkContract(ctx context.Context, fc *FrameworkContract) error
	DeleteFrameworkContract(ctx context.Context, tenantID, contractID uuid.UUID) error
	GetFrameworkContract(ctx context.Context, tenantID, contractID uuid.UUID) (*FrameworkContract, error)
	GetFrameworkContractWithItems(ctx context.Context, tenantID, contractID uuid.UUID) (*FrameworkContract, error)
	ListFrameworkContracts(ctx context.Context, tenantID uuid.UUID, filter ListContractsFilter, offset, limit int) ([]*FrameworkContract, int, error)
	ContractNrExists(ctx context.Context, tenantID uuid.UUID, contractNr string, excludeID *uuid.UUID) (bool, error)

	// -------------------------------------------------------------------------
	// Framework Contract Items
	// -------------------------------------------------------------------------

	CreateContractItem(ctx context.Context, item *FrameworkContractItem) error
	UpdateContractItem(ctx context.Context, item *FrameworkContractItem) error
	DeleteContractItem(ctx context.Context, tenantID, itemID uuid.UUID) error
	ListContractItems(ctx context.Context, tenantID, contractID uuid.UUID) ([]*FrameworkContractItem, error)

	// -------------------------------------------------------------------------
	// Framework Contract Calls
	// -------------------------------------------------------------------------

	// CreateContractCall persists a call-off, enforces the contract's status
	// and remaining value, and recomputes used_value — all in one transaction.
	// It is the only writer of used_value, so the column cannot drift away
	// from the sum of the calls it caches. Rejects with ErrContractNotFound,
	// ErrContractNotActive or ErrContractBudgetExceeded without writing.
	CreateContractCall(ctx context.Context, call *FrameworkContractCall) error
	ListContractCalls(ctx context.Context, tenantID, contractID uuid.UUID) ([]*FrameworkContractCall, error)
}

// ListCatalogItemsFilter contains filtering options for catalog items.
type ListCatalogItemsFilter struct {
	SupplierID *uuid.UUID
	Category   string
	Search     string
	Available  *bool
}

// ListContractsFilter contains filtering options for framework contracts.
type ListContractsFilter struct {
	SupplierID *uuid.UUID
	Status     *ContractStatus
}
