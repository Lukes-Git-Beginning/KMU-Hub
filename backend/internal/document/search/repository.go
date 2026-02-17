package search

import (
	"context"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for document search operations.
type Repository interface {
	// Search performs full-text search across document files using PostgreSQL tsvector.
	// Returns matching results with ranking and snippet, plus total count.
	Search(ctx context.Context, query string, filter SearchFilter) ([]SearchResult, int, error)
}

// SearchFilter constrains search results.
type SearchFilter struct {
	FolderID *uuid.UUID  // scope to a specific folder
	OwnerID  *uuid.UUID  // scope to a specific file owner
	TagIDs   []uuid.UUID // require any of these tags
	Limit    int
	Offset   int
}

// SearchResult contains a matching file with its relevance ranking and snippet.
type SearchResult struct {
	File    models.DocumentFile
	Rank    float64
	Snippet string
}

// FileRepo is a minimal interface for updating file search content.
// Defined here to avoid circular import with the file package.
// The gRPC server bridges this to the file package's repository.
type FileRepo interface {
	// UpdateSearchContent sets the content_text field for a document file,
	// triggering the search_vector update trigger.
	UpdateSearchContent(ctx context.Context, id uuid.UUID, text string) error
}
