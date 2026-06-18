package inventar

import "errors"

var (
	ErrItemNotFound             = errors.New("inventory item not found")
	ErrMovementNotFound         = errors.New("inventory movement not found")
	ErrWarningNotFound          = errors.New("inventory warning not found")
	ErrInvalidInput             = errors.New("inventory input is invalid")
	ErrSKUTaken                 = errors.New("inventory item SKU already taken")
	ErrInsufficientStock        = errors.New("inventory item has insufficient stock for this operation")
	ErrLocationNotFound         = errors.New("inventory location not found")
	ErrInventurSessionNotFound  = errors.New("inventur session not found")
	ErrInventurCountNotFound    = errors.New("inventur count not found")
	ErrInventurAlreadyCompleted = errors.New("inventur session is already completed")
)
