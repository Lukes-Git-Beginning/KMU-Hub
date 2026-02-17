package search

import "errors"

var (
	// ErrSearchQueryRequired is returned when a search query is empty.
	ErrSearchQueryRequired = errors.New("search query is required")

	// ErrExtractionFailed is returned when text extraction from a file fails.
	ErrExtractionFailed = errors.New("text extraction failed")

	// ErrUnsupportedFormat is returned when a file's MIME type is not supported for text extraction.
	ErrUnsupportedFormat = errors.New("unsupported file format for text extraction")
)
