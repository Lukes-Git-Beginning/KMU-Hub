package einkauf

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NewServiceExtended creates a Service that supports catalog, supplier ratings,
// and framework contracts in addition to the base purchasing operations.
// The provided repo must implement RepositoryExtended.
func NewServiceExtended(repo RepositoryExtended) *Service {
	return &Service{repo: repo, repoExt: repo}
}

// ============================================================================
// Input types — Catalog
// ============================================================================

// CreateCatalogItemInput holds data for creating a catalog item.
type CreateCatalogItemInput struct {
	TenantID    uuid.UUID
	SupplierID  uuid.UUID
	Name        string
	SKU         string
	Category    string
	Price       string
	Currency    string
	Unit        string
	Available   bool
	MinOrderQty string
}

// UpdateCatalogItemInput holds data for updating a catalog item.
type UpdateCatalogItemInput struct {
	TenantID    uuid.UUID
	ItemID      uuid.UUID
	Name        *string
	SKU         *string
	Category    *string
	Price       *string
	Currency    *string
	Unit        *string
	Available   *bool
	MinOrderQty *string
}

// ListCatalogItemsInput holds list/filter options for catalog items.
type ListCatalogItemsInput struct {
	TenantID   uuid.UUID
	SupplierID *uuid.UUID
	Category   string
	Search     string
	Available  *bool
	Page       int
	PageSize   int
}

// ============================================================================
// Input types — Supplier Ratings
// ============================================================================

// CreateSupplierRatingInput holds data for creating a supplier rating.
type CreateSupplierRatingInput struct {
	TenantID   uuid.UUID
	SupplierID uuid.UUID
	Category   RatingCategory
	Rating     int32
	Comment    string
	RatedBy    *uuid.UUID
}

// ============================================================================
// Input types — Framework Contracts
// ============================================================================

// CreateFrameworkContractInput holds data for creating a framework contract.
type CreateFrameworkContractInput struct {
	TenantID   uuid.UUID
	SupplierID uuid.UUID
	Title      string
	ContractNr string
	StartDate  *time.Time
	EndDate    *time.Time
	TotalValue string
	Currency   string
	Status     ContractStatus
}

// UpdateFrameworkContractInput holds data for updating a framework contract.
type UpdateFrameworkContractInput struct {
	TenantID   uuid.UUID
	ContractID uuid.UUID
	SupplierID *uuid.UUID
	Title      *string
	ContractNr *string
	StartDate  *time.Time // send zero-value pointer to clear
	EndDate    *time.Time
	TotalValue *string
	Currency   *string
	Status     *ContractStatus
}

// ListFrameworkContractsInput holds list/filter options for framework contracts.
type ListFrameworkContractsInput struct {
	TenantID   uuid.UUID
	SupplierID *uuid.UUID
	Status     *ContractStatus
	Page       int
	PageSize   int
}

// CreateContractItemInput holds data for adding an item to a framework contract.
type CreateContractItemInput struct {
	TenantID   uuid.UUID
	ContractID uuid.UUID
	Name       string
	UnitPrice  string
	Unit       string
	AgreedQty  string
}

// UpdateContractItemInput holds data for updating a framework contract item.
type UpdateContractItemInput struct {
	TenantID  uuid.UUID
	ItemID    uuid.UUID
	Name      *string
	UnitPrice *string
	Unit      *string
	AgreedQty *string
}

// CreateContractCallInput holds data for recording a contract call-off.
type CreateContractCallInput struct {
	TenantID   uuid.UUID
	ContractID uuid.UUID
	POID       *uuid.UUID
	Amount     string
	Currency   string
	Notes      string
}

// ============================================================================
// Catalog methods
// ============================================================================

// CreateCatalogItem creates a new catalog item for a supplier.
func (s *Service) CreateCatalogItem(ctx context.Context, input CreateCatalogItemInput) (*CatalogItem, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	price := strings.TrimSpace(input.Price)
	if price == "" {
		return nil, ErrInvalidInput
	}
	if v, err := strconv.ParseFloat(price, 64); err != nil || v < 0 {
		return nil, fmt.Errorf("catalog item price: %w", ErrInvalidInput)
	}

	minOrderQty := strings.TrimSpace(input.MinOrderQty)
	if minOrderQty == "" {
		minOrderQty = "1"
	}

	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "EUR"
	}

	now := time.Now()
	item := &CatalogItem{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		SupplierID:  input.SupplierID,
		Name:        name,
		SKU:         strings.TrimSpace(input.SKU),
		Category:    strings.TrimSpace(input.Category),
		Price:       price,
		Currency:    currency,
		Unit:        strings.TrimSpace(input.Unit),
		Available:   input.Available,
		MinOrderQty: minOrderQty,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repoExt.CreateCatalogItem(ctx, item); err != nil {
		return nil, fmt.Errorf("create catalog item: %w", err)
	}

	slog.Info("einkauf catalog item created", "item_id", item.ID, "tenant_id", item.TenantID, "name", item.Name)
	return item, nil
}

// UpdateCatalogItem updates an existing catalog item.
func (s *Service) UpdateCatalogItem(ctx context.Context, input UpdateCatalogItemInput) (*CatalogItem, error) {
	item, err := s.repoExt.GetCatalogItem(ctx, input.TenantID, input.ItemID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrInvalidInput
		}
		item.Name = name
	}
	if input.SKU != nil {
		item.SKU = strings.TrimSpace(*input.SKU)
	}
	if input.Category != nil {
		item.Category = strings.TrimSpace(*input.Category)
	}
	if input.Price != nil {
		price := strings.TrimSpace(*input.Price)
		if v, err := strconv.ParseFloat(price, 64); err != nil || v < 0 {
			return nil, fmt.Errorf("catalog item price: %w", ErrInvalidInput)
		}
		item.Price = price
	}
	if input.Currency != nil {
		item.Currency = strings.TrimSpace(*input.Currency)
	}
	if input.Unit != nil {
		item.Unit = strings.TrimSpace(*input.Unit)
	}
	if input.Available != nil {
		item.Available = *input.Available
	}
	if input.MinOrderQty != nil {
		item.MinOrderQty = strings.TrimSpace(*input.MinOrderQty)
	}

	item.UpdatedAt = time.Now()

	if updateErr := s.repoExt.UpdateCatalogItem(ctx, item); updateErr != nil {
		return nil, fmt.Errorf("update catalog item: %w", updateErr)
	}

	slog.Info("einkauf catalog item updated", "item_id", item.ID, "tenant_id", item.TenantID)
	return item, nil
}

// DeleteCatalogItem removes a catalog item.
func (s *Service) DeleteCatalogItem(ctx context.Context, tenantID, itemID uuid.UUID) error {
	if _, err := s.repoExt.GetCatalogItem(ctx, tenantID, itemID); err != nil {
		return err
	}
	if err := s.repoExt.DeleteCatalogItem(ctx, tenantID, itemID); err != nil {
		return fmt.Errorf("delete catalog item: %w", err)
	}
	slog.Info("einkauf catalog item deleted", "item_id", itemID, "tenant_id", tenantID)
	return nil
}

// GetCatalogItem retrieves a catalog item by ID.
func (s *Service) GetCatalogItem(ctx context.Context, tenantID, itemID uuid.UUID) (*CatalogItem, error) {
	return s.repoExt.GetCatalogItem(ctx, tenantID, itemID)
}

// ListCatalogItems retrieves catalog items with optional filtering and pagination.
func (s *Service) ListCatalogItems(ctx context.Context, input ListCatalogItemsInput) ([]*CatalogItem, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}
	offset := (input.Page - 1) * input.PageSize

	return s.repoExt.ListCatalogItems(ctx, input.TenantID, ListCatalogItemsFilter{
		SupplierID: input.SupplierID,
		Category:   input.Category,
		Search:     input.Search,
		Available:  input.Available,
	}, offset, input.PageSize)
}

// ============================================================================
// Supplier Rating methods
// ============================================================================

// CreateSupplierRating adds a rating for a supplier in a given category.
// Returns ErrDuplicateRating if a rating already exists for tenant+supplier+category.
func (s *Service) CreateSupplierRating(ctx context.Context, input CreateSupplierRatingInput) (*SupplierRating, error) {
	// Validate category
	switch input.Category {
	case RatingCategoryQuality, RatingCategoryDelivery, RatingCategoryPrice:
		// valid
	default:
		return nil, fmt.Errorf("supplier rating category %q: %w", input.Category, ErrInvalidInput)
	}

	// Validate rating range 1-5
	if input.Rating < 1 || input.Rating > 5 {
		return nil, fmt.Errorf("supplier rating value %d out of range 1-5: %w", input.Rating, ErrInvalidInput)
	}

	now := time.Now()
	rating := &SupplierRating{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		SupplierID: input.SupplierID,
		Category:   input.Category,
		Rating:     input.Rating,
		Comment:    strings.TrimSpace(input.Comment),
		RatedBy:    input.RatedBy,
		RatedAt:    now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repoExt.CreateSupplierRating(ctx, rating); err != nil {
		return nil, fmt.Errorf("create supplier rating: %w", err)
	}

	slog.Info("einkauf supplier rating created",
		"rating_id", rating.ID,
		"supplier_id", rating.SupplierID,
		"category", rating.Category,
		"tenant_id", rating.TenantID,
	)
	return rating, nil
}

// DeleteSupplierRating removes a supplier rating.
func (s *Service) DeleteSupplierRating(ctx context.Context, tenantID, ratingID uuid.UUID) error {
	if _, err := s.repoExt.GetSupplierRating(ctx, tenantID, ratingID); err != nil {
		return err
	}
	if err := s.repoExt.DeleteSupplierRating(ctx, tenantID, ratingID); err != nil {
		return fmt.Errorf("delete supplier rating: %w", err)
	}
	slog.Info("einkauf supplier rating deleted", "rating_id", ratingID, "tenant_id", tenantID)
	return nil
}

// ListSupplierRatings returns all ratings for a supplier.
func (s *Service) ListSupplierRatings(ctx context.Context, tenantID, supplierID uuid.UUID) ([]*SupplierRating, error) {
	return s.repoExt.ListSupplierRatings(ctx, tenantID, supplierID)
}

// ============================================================================
// Framework Contract methods
// ============================================================================

// CreateFrameworkContract creates a new framework contract.
func (s *Service) CreateFrameworkContract(ctx context.Context, input CreateFrameworkContractInput) (*FrameworkContract, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrInvalidInput
	}
	contractNr := strings.TrimSpace(input.ContractNr)

	if contractNr != "" {
		exists, err := s.repoExt.ContractNrExists(ctx, input.TenantID, contractNr, nil)
		if err != nil {
			return nil, fmt.Errorf("check contract nr: %w", err)
		}
		if exists {
			return nil, ErrContractNrTaken
		}
	}

	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "EUR"
	}

	totalValue := strings.TrimSpace(input.TotalValue)
	if totalValue == "" {
		totalValue = "0"
	}

	statusVal := input.Status
	if statusVal == "" {
		statusVal = ContractStatusDraft
	}

	now := time.Now()
	fc := &FrameworkContract{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		SupplierID: input.SupplierID,
		Title:      title,
		ContractNr: contractNr,
		StartDate:  input.StartDate,
		EndDate:    input.EndDate,
		TotalValue: totalValue,
		UsedValue:  "0",
		Currency:   currency,
		Status:     statusVal,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repoExt.CreateFrameworkContract(ctx, fc); err != nil {
		return nil, fmt.Errorf("create framework contract: %w", err)
	}

	slog.Info("einkauf framework contract created",
		"contract_id", fc.ID,
		"tenant_id", fc.TenantID,
		"contract_nr", fc.ContractNr,
	)
	return fc, nil
}

// UpdateFrameworkContract updates an existing framework contract.
func (s *Service) UpdateFrameworkContract(ctx context.Context, input UpdateFrameworkContractInput) (*FrameworkContract, error) {
	fc, err := s.repoExt.GetFrameworkContract(ctx, input.TenantID, input.ContractID)
	if err != nil {
		return nil, err
	}

	if input.SupplierID != nil {
		fc.SupplierID = *input.SupplierID
	}
	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return nil, ErrInvalidInput
		}
		fc.Title = title
	}
	if input.ContractNr != nil {
		newNr := strings.TrimSpace(*input.ContractNr)
		if newNr != fc.ContractNr {
			exists, checkErr := s.repoExt.ContractNrExists(ctx, input.TenantID, newNr, &fc.ID)
			if checkErr != nil {
				return nil, fmt.Errorf("check contract nr: %w", checkErr)
			}
			if exists {
				return nil, ErrContractNrTaken
			}
			fc.ContractNr = newNr
		}
	}
	if input.StartDate != nil {
		if input.StartDate.IsZero() {
			fc.StartDate = nil
		} else {
			fc.StartDate = input.StartDate
		}
	}
	if input.EndDate != nil {
		if input.EndDate.IsZero() {
			fc.EndDate = nil
		} else {
			fc.EndDate = input.EndDate
		}
	}
	if input.TotalValue != nil {
		fc.TotalValue = strings.TrimSpace(*input.TotalValue)
	}
	if input.Currency != nil {
		fc.Currency = strings.TrimSpace(*input.Currency)
	}
	if input.Status != nil {
		fc.Status = *input.Status
	}

	fc.UpdatedAt = time.Now()

	if updateErr := s.repoExt.UpdateFrameworkContract(ctx, fc); updateErr != nil {
		return nil, fmt.Errorf("update framework contract: %w", updateErr)
	}

	slog.Info("einkauf framework contract updated", "contract_id", fc.ID, "tenant_id", fc.TenantID)
	return fc, nil
}

// DeleteFrameworkContract removes a framework contract.
func (s *Service) DeleteFrameworkContract(ctx context.Context, tenantID, contractID uuid.UUID) error {
	if _, err := s.repoExt.GetFrameworkContract(ctx, tenantID, contractID); err != nil {
		return err
	}
	if err := s.repoExt.DeleteFrameworkContract(ctx, tenantID, contractID); err != nil {
		return fmt.Errorf("delete framework contract: %w", err)
	}
	slog.Info("einkauf framework contract deleted", "contract_id", contractID, "tenant_id", tenantID)
	return nil
}

// GetFrameworkContract retrieves a framework contract with all its items populated.
func (s *Service) GetFrameworkContract(ctx context.Context, tenantID, contractID uuid.UUID) (*FrameworkContract, error) {
	return s.repoExt.GetFrameworkContractWithItems(ctx, tenantID, contractID)
}

// ListFrameworkContracts retrieves framework contracts with optional filtering and pagination.
func (s *Service) ListFrameworkContracts(ctx context.Context, input ListFrameworkContractsInput) ([]*FrameworkContract, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}
	offset := (input.Page - 1) * input.PageSize

	return s.repoExt.ListFrameworkContracts(ctx, input.TenantID, ListContractsFilter{
		SupplierID: input.SupplierID,
		Status:     input.Status,
	}, offset, input.PageSize)
}

// ============================================================================
// Framework Contract Item methods
// ============================================================================

// CreateContractItem adds a line item to an existing framework contract.
func (s *Service) CreateContractItem(ctx context.Context, input CreateContractItemInput) (*FrameworkContractItem, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}

	// Verify contract exists
	if _, err := s.repoExt.GetFrameworkContract(ctx, input.TenantID, input.ContractID); err != nil {
		return nil, err
	}

	now := time.Now()
	item := &FrameworkContractItem{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		ContractID: input.ContractID,
		Name:       name,
		UnitPrice:  strings.TrimSpace(input.UnitPrice),
		Unit:       strings.TrimSpace(input.Unit),
		AgreedQty:  strings.TrimSpace(input.AgreedQty),
		CalledQty:  "0",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if item.UnitPrice == "" {
		item.UnitPrice = "0"
	}
	if item.AgreedQty == "" {
		item.AgreedQty = "0"
	}

	if err := s.repoExt.CreateContractItem(ctx, item); err != nil {
		return nil, fmt.Errorf("create contract item: %w", err)
	}

	slog.Info("einkauf contract item created",
		"item_id", item.ID,
		"contract_id", item.ContractID,
		"tenant_id", item.TenantID,
	)
	return item, nil
}

// UpdateContractItem updates an existing framework contract item.
func (s *Service) UpdateContractItem(ctx context.Context, input UpdateContractItemInput) (*FrameworkContractItem, error) {
	item, err := getContractItemByID(ctx, s.repoExt, input.TenantID, input.ItemID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrInvalidInput
		}
		item.Name = name
	}
	if input.UnitPrice != nil {
		item.UnitPrice = strings.TrimSpace(*input.UnitPrice)
	}
	if input.Unit != nil {
		item.Unit = strings.TrimSpace(*input.Unit)
	}
	if input.AgreedQty != nil {
		item.AgreedQty = strings.TrimSpace(*input.AgreedQty)
	}

	item.UpdatedAt = time.Now()

	if updateErr := s.repoExt.UpdateContractItem(ctx, item); updateErr != nil {
		return nil, fmt.Errorf("update contract item: %w", updateErr)
	}

	slog.Info("einkauf contract item updated", "item_id", item.ID, "tenant_id", input.TenantID)
	return item, nil
}

// DeleteContractItem removes a framework contract item.
func (s *Service) DeleteContractItem(ctx context.Context, tenantID, itemID uuid.UUID) error {
	if err := s.repoExt.DeleteContractItem(ctx, tenantID, itemID); err != nil {
		return fmt.Errorf("delete contract item: %w", err)
	}
	slog.Info("einkauf contract item deleted", "item_id", itemID, "tenant_id", tenantID)
	return nil
}

// ============================================================================
// Framework Contract Call methods
// ============================================================================

// CreateContractCall records a call-off against a framework contract and
// recalculates the contract's used_value.
func (s *Service) CreateContractCall(ctx context.Context, input CreateContractCallInput) (*FrameworkContractCall, error) {
	amount := strings.TrimSpace(input.Amount)
	if amount == "" {
		amount = "0"
	}
	if v, err := strconv.ParseFloat(amount, 64); err != nil || v < 0 {
		return nil, fmt.Errorf("contract call amount: %w", ErrInvalidInput)
	}

	// Verify contract exists
	if _, err := s.repoExt.GetFrameworkContract(ctx, input.TenantID, input.ContractID); err != nil {
		return nil, err
	}

	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "EUR"
	}

	now := time.Now()
	call := &FrameworkContractCall{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		ContractID: input.ContractID,
		POID:       input.POID,
		Amount:     amount,
		Currency:   currency,
		CalledAt:   now,
		Notes:      strings.TrimSpace(input.Notes),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repoExt.CreateContractCall(ctx, call); err != nil {
		return nil, fmt.Errorf("create contract call: %w", err)
	}

	// Recompute used_value on the contract
	if err := s.repoExt.UpdateContractUsedValue(ctx, input.TenantID, input.ContractID); err != nil {
		slog.Error("einkauf failed to update contract used_value after call",
			"contract_id", input.ContractID,
			"call_id", call.ID,
			"error", err,
		)
		// Non-fatal: the call is persisted; the aggregate will self-correct on next call.
	}

	slog.Info("einkauf contract call created",
		"call_id", call.ID,
		"contract_id", call.ContractID,
		"amount", call.Amount,
		"tenant_id", call.TenantID,
	)
	return call, nil
}

// ListContractCalls returns all call-offs for a framework contract.
func (s *Service) ListContractCalls(ctx context.Context, tenantID, contractID uuid.UUID) ([]*FrameworkContractCall, error) {
	return s.repoExt.ListContractCalls(ctx, tenantID, contractID)
}

// ============================================================================
// Internal helpers
// ============================================================================

// contractItemQuerier is implemented by *PostgresRepository to allow direct
// item lookups without exposing GetContractItem on the public interface.
type contractItemQuerier interface {
	QueryRowContractItem(ctx context.Context, tenantID, itemID uuid.UUID) (*FrameworkContractItem, error)
}

// getContractItemByID fetches a framework contract item by item ID + tenant ID.
// It uses a type assertion to QueryRowContractItem, which is implemented by
// *PostgresRepository. Returns ErrContractItemNotFound if the item is not found
// or the repository does not support direct item queries.
func getContractItemByID(ctx context.Context, repo RepositoryExtended, tenantID, itemID uuid.UUID) (*FrameworkContractItem, error) {
	if dq, ok := repo.(contractItemQuerier); ok {
		return dq.QueryRowContractItem(ctx, tenantID, itemID)
	}
	return nil, ErrContractItemNotFound
}
