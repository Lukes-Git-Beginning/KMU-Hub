package search

import "errors"

var (
	ErrQueryTooShort = errors.New("search query must be at least 2 characters")
	ErrQueryTooLong  = errors.New("search query must not exceed 200 characters")
)
