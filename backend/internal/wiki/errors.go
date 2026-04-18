package wiki

import "errors"

var (
	ErrArticleNotFound  = errors.New("wiki article not found")
	ErrSlugTaken        = errors.New("wiki article slug already taken")
	ErrInvalidContent   = errors.New("wiki article content is invalid")
	ErrVersionNotFound  = errors.New("wiki version not found")
	ErrCategoryNotFound = errors.New("wiki category not found")
)
