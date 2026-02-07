package search

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Detector detects the language of a text for search query processing
type Detector interface {
	Detect(text string) string
}

// Service handles chat search business logic
type Service struct {
	repo     Repository
	detector Detector
}

// NewService creates a new search service
func NewService(repo Repository, detector Detector) *Service {
	return &Service{
		repo:     repo,
		detector: detector,
	}
}

// SearchInput contains the parameters for a search query
type SearchInput struct {
	Query     string
	UserID    uuid.UUID
	ChannelID *uuid.UUID // Optional: filter to specific channel
	Page      int
	PageSize  int
}

// Search performs a full-text search across messages and files
func (s *Service) Search(ctx context.Context, input SearchInput) ([]models.ChatSearchResult, int, error) {
	// Validate query
	query := strings.TrimSpace(input.Query)
	if len(query) < 2 {
		return nil, 0, ErrQueryTooShort
	}
	if len(query) > 200 {
		return nil, 0, ErrQueryTooLong
	}

	// Normalize pagination
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	// Get channel IDs to search
	var channelIDs []uuid.UUID
	if input.ChannelID != nil {
		channelIDs = []uuid.UUID{*input.ChannelID}
	} else {
		var err error
		channelIDs, err = s.repo.GetUserChannelIDs(ctx, input.UserID)
		if err != nil {
			return nil, 0, err
		}
	}

	if len(channelIDs) == 0 {
		return nil, 0, nil
	}

	// Detect query language for message search
	lang := s.detector.Detect(query)

	// Search messages and files separately, then merge
	// Use a larger limit for each to ensure good merged results
	fetchLimit := input.PageSize * 2

	msgResults, msgTotal, err := s.repo.SearchMessages(ctx, query, lang, channelIDs, fetchLimit, 0)
	if err != nil {
		return nil, 0, err
	}

	fileResults, fileTotal, err := s.repo.SearchFiles(ctx, query, channelIDs, fetchLimit, 0)
	if err != nil {
		return nil, 0, err
	}

	// Merge and sort by score
	all := append(msgResults, fileResults...)
	sort.Slice(all, func(i, j int) bool {
		if all[i].Score != all[j].Score {
			return all[i].Score > all[j].Score
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	totalCount := msgTotal + fileTotal

	// Paginate merged results
	offset := (input.Page - 1) * input.PageSize
	if offset >= len(all) {
		return nil, totalCount, nil
	}
	end := offset + input.PageSize
	if end > len(all) {
		end = len(all)
	}

	return all[offset:end], totalCount, nil
}
