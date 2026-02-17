package search

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
)

// Service handles document search business logic and coordinates
// text extraction for async post-upload indexing.
type Service struct {
	repo      Repository
	extractor *Extractor
	fileStore chatfile.FileStore
}

// NewService creates a new search Service.
func NewService(repo Repository, extractor *Extractor, fileStore chatfile.FileStore) *Service {
	return &Service{
		repo:      repo,
		extractor: extractor,
		fileStore: fileStore,
	}
}

// Search performs full-text search across document files.
// Returns ranked results with snippets and total count.
func (s *Service) Search(ctx context.Context, query string, filter SearchFilter) ([]SearchResult, int, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, 0, ErrSearchQueryRequired
	}

	results, total, err := s.repo.Search(ctx, query, filter)
	if err != nil {
		return nil, 0, err
	}

	slog.Info("document search executed",
		"query", query,
		"results", len(results),
		"total", total,
	)

	return results, total, nil
}

// ExtractAndIndex downloads a file from storage, extracts searchable text,
// and updates the file's content_text field. This method is designed to be
// called ASYNCHRONOUSLY after file upload (via goroutine from the file service
// or gRPC handler). It decouples text extraction from the upload response.
//
// Steps:
//  1. Check if the MIME type is extractable
//  2. Download file from MinIO via fileStore.Download
//  3. Extract text via extractor.Extract
//  4. Update file's content_text via fileRepo.UpdateSearchContent
func (s *Service) ExtractAndIndex(ctx context.Context, fileID uuid.UUID, storageKey, mimeType string, fileRepo FileRepo) error {
	// Step 1: check extractability
	if !s.extractor.CanExtract(mimeType) {
		slog.Info("skipping text extraction for unsupported MIME type",
			"file_id", fileID,
			"mime_type", mimeType,
		)
		return nil
	}

	slog.Info("starting text extraction",
		"file_id", fileID,
		"mime_type", mimeType,
		"storage_key", storageKey,
	)

	// Step 2: download from storage
	reader, err := s.fileStore.Download(ctx, storageKey)
	if err != nil {
		slog.Warn("failed to download file for text extraction",
			"file_id", fileID,
			"storage_key", storageKey,
			"error", err,
		)
		return err
	}
	defer reader.Close()

	// Step 3: extract text
	text, err := s.extractor.Extract(ctx, reader, mimeType)
	if err != nil {
		slog.Warn("text extraction returned error",
			"file_id", fileID,
			"mime_type", mimeType,
			"error", err,
		)
		// Don't fail the indexing on extraction error; just skip content
		return nil
	}

	if text == "" {
		slog.Info("no text extracted from file",
			"file_id", fileID,
			"mime_type", mimeType,
		)
		return nil
	}

	// Step 4: update search content
	if err := fileRepo.UpdateSearchContent(ctx, fileID, text); err != nil {
		slog.Warn("failed to update search content for file",
			"file_id", fileID,
			"error", err,
		)
		return err
	}

	slog.Info("text extraction and indexing complete",
		"file_id", fileID,
		"text_length", len(text),
	)

	return nil
}
