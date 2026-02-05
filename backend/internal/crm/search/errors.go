package search

import "errors"

var (
	ErrQueryTooShort = errors.New("search query too short (minimum 2 characters)")
	ErrInvalidEntity = errors.New("invalid entity type")
	ErrQueryTooLong  = errors.New("search query too long (maximum 200 characters)")
)
