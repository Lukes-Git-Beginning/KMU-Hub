package inventar

import "errors"

var (
	ErrItemNotFound     = errors.New("inventory item not found")
	ErrMovementNotFound = errors.New("inventory movement not found")
	ErrWarningNotFound  = errors.New("inventory warning not found")
	ErrInvalidInput     = errors.New("inventory input is invalid")
	ErrSKUTaken         = errors.New("inventory item SKU already taken")
)
