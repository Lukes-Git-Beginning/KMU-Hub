package einkauf

import "errors"

var (
	ErrSupplierNotFound    = errors.New("einkauf supplier not found")
	ErrPONotFound          = errors.New("einkauf purchase order not found")
	ErrPOLineNotFound      = errors.New("einkauf po line not found")
	ErrPONumberTaken       = errors.New("einkauf po number already exists for this tenant")
	ErrInvalidInput        = errors.New("einkauf invalid input")
	ErrPONotDraft          = errors.New("einkauf purchase order must be in draft status for this operation")
	ErrPONotSubmittable    = errors.New("einkauf purchase order has no lines and cannot be submitted")
	ErrPONotReceivable     = errors.New("einkauf purchase order cannot receive goods in its current status")
	ErrPONotCancellable    = errors.New("einkauf purchase order cannot be cancelled in its current status")
	ErrInvalidQuantity     = errors.New("einkauf invalid quantity: must be >= 0")
	ErrExceedsOrderedQty   = errors.New("einkauf received quantity exceeds ordered quantity")

	// Extended: Catalog
	ErrCatalogItemNotFound = errors.New("einkauf catalog item not found")

	// Extended: Supplier Ratings
	ErrSupplierRatingNotFound = errors.New("einkauf supplier rating not found")
	ErrDuplicateRating        = errors.New("einkauf supplier rating already exists for this category")

	// Extended: Framework Contracts
	ErrContractNotFound     = errors.New("einkauf framework contract not found")
	ErrContractItemNotFound = errors.New("einkauf framework contract item not found")
	ErrContractNrTaken      = errors.New("einkauf contract number already exists for this tenant")
	ErrContractCallNotFound = errors.New("einkauf framework contract call not found")
	// ErrContractNotActive rejects a call-off against a draft or expired
	// framework contract.
	ErrContractNotActive = errors.New("einkauf framework contract must be active to be called off")
	// ErrContractBudgetExceeded rejects a call-off above the contract's
	// remaining value. The wrapping message names that remaining value.
	ErrContractBudgetExceeded = errors.New("einkauf contract call-off exceeds the contract's remaining value")
	// ErrContractCurrencyMismatch rejects a call-off whose currency differs
	// from the contract's. No conversion is attempted: a USD call-off would
	// otherwise consume EUR of the contract volume one-to-one.
	ErrContractCurrencyMismatch = errors.New("einkauf contract call-off currency differs from the contract's currency")
	// ErrPOContractCurrencyMismatch rejects binding a purchase order to a
	// framework contract in another currency. Unlike the call-off variant it
	// is a malformed request, not a state conflict: the caller picked a
	// contract that can never match this PO.
	ErrPOContractCurrencyMismatch = errors.New("einkauf purchase order currency differs from the framework contract's currency")
)
