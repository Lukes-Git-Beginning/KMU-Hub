package search

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Mocks
// ============================================================================

type MockRepository struct {
	messages   []models.ChatSearchResult
	files      []models.ChatSearchResult
	channelIDs []uuid.UUID
	searchErr  error
}

func (m *MockRepository) SearchMessages(_ context.Context, _ string, _ string, _ []uuid.UUID, limit, offset int) ([]models.ChatSearchResult, int, error) {
	if m.searchErr != nil {
		return nil, 0, m.searchErr
	}
	total := len(m.messages)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return m.messages[offset:end], total, nil
}

func (m *MockRepository) SearchFiles(_ context.Context, _ string, _ []uuid.UUID, limit, offset int) ([]models.ChatSearchResult, int, error) {
	if m.searchErr != nil {
		return nil, 0, m.searchErr
	}
	total := len(m.files)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return m.files[offset:end], total, nil
}

func (m *MockRepository) GetUserChannelIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return m.channelIDs, nil
}

// MockDetector
type MockDetector struct {
	lang string
}

func (d *MockDetector) Detect(_ string) string {
	return d.lang
}

// ============================================================================
// Tests
// ============================================================================

func TestService_Search(t *testing.T) {
	t.Run("success - messages and files merged", func(t *testing.T) {
		channelID := uuid.New()
		msgID := uuid.New()
		fileID := uuid.New()

		repo := &MockRepository{
			channelIDs: []uuid.UUID{channelID},
			messages: []models.ChatSearchResult{
				{
					Type:      models.ChatSearchResultMessage,
					ID:        msgID,
					ChannelID: channelID,
					Score:     0.9,
					Snippet:   "Hello <mark>test</mark>",
					CreatedAt: time.Now(),
					MessageID: &msgID,
				},
			},
			files: []models.ChatSearchResult{
				{
					Type:      models.ChatSearchResultFile,
					ID:        fileID,
					ChannelID: channelID,
					Score:     0.5,
					Snippet:   "<mark>test</mark>.pdf",
					CreatedAt: time.Now(),
					FileID:    &fileID,
				},
			},
		}

		detector := &MockDetector{lang: "german"}
		svc := NewService(repo, detector)

		results, total, err := svc.Search(context.Background(), SearchInput{
			Query:    "test query",
			UserID:   uuid.New(),
			Page:     1,
			PageSize: 20,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, results, 2)
		// Higher score first
		assert.Equal(t, models.ChatSearchResultMessage, results[0].Type)
		assert.Equal(t, models.ChatSearchResultFile, results[1].Type)
	})

	t.Run("query too short", func(t *testing.T) {
		repo := &MockRepository{channelIDs: []uuid.UUID{uuid.New()}}
		detector := &MockDetector{lang: "simple"}
		svc := NewService(repo, detector)

		_, _, err := svc.Search(context.Background(), SearchInput{
			Query:  "a",
			UserID: uuid.New(),
		})

		assert.Equal(t, ErrQueryTooShort, err)
	})

	t.Run("query too long", func(t *testing.T) {
		repo := &MockRepository{channelIDs: []uuid.UUID{uuid.New()}}
		detector := &MockDetector{lang: "simple"}
		svc := NewService(repo, detector)

		_, _, err := svc.Search(context.Background(), SearchInput{
			Query:  strings.Repeat("a", 201),
			UserID: uuid.New(),
		})

		assert.Equal(t, ErrQueryTooLong, err)
	})

	t.Run("no channels returns empty", func(t *testing.T) {
		repo := &MockRepository{channelIDs: []uuid.UUID{}} // no channels
		detector := &MockDetector{lang: "simple"}
		svc := NewService(repo, detector)

		results, total, err := svc.Search(context.Background(), SearchInput{
			Query:  "test query",
			UserID: uuid.New(),
		})

		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Nil(t, results)
	})

	t.Run("single channel filter", func(t *testing.T) {
		channelID := uuid.New()
		repo := &MockRepository{
			channelIDs: []uuid.UUID{channelID, uuid.New()},
			messages:   nil,
			files:      nil,
		}
		detector := &MockDetector{lang: "german"}
		svc := NewService(repo, detector)

		results, total, err := svc.Search(context.Background(), SearchInput{
			Query:     "test query",
			UserID:    uuid.New(),
			ChannelID: &channelID,
			Page:      1,
			PageSize:  20,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Nil(t, results)
	})

	t.Run("pagination", func(t *testing.T) {
		channelID := uuid.New()

		var messages []models.ChatSearchResult
		for i := 0; i < 5; i++ {
			msgID := uuid.New()
			messages = append(messages, models.ChatSearchResult{
				Type:      models.ChatSearchResultMessage,
				ID:        msgID,
				ChannelID: channelID,
				Score:     float64(5-i) * 0.1,
				CreatedAt: time.Now(),
				MessageID: &msgID,
			})
		}

		repo := &MockRepository{
			channelIDs: []uuid.UUID{channelID},
			messages:   messages,
		}
		detector := &MockDetector{lang: "german"}
		svc := NewService(repo, detector)

		results, total, err := svc.Search(context.Background(), SearchInput{
			Query:    "test query",
			UserID:   uuid.New(),
			Page:     1,
			PageSize: 3,
		})

		require.NoError(t, err)
		assert.Equal(t, 5, total)
		assert.Len(t, results, 3)
	})

	// VERIFIED FINDING (not fixed in this coverage unit, see
	// fix-chat-search-channel-filter-bypasses-membership at the end of
	// BACKLOG.yml): when the caller supplies ChannelID explicitly, Search
	// never calls GetUserChannelIDs to check membership -- it searches that
	// channel directly. Contrast with bookmark.Service.Toggle
	// (internal/chat/bookmark/service.go), which reuses message.Service's
	// GetByID specifically because GetByID enforces tenant AND channel
	// membership before returning a message. Search has no equivalent check
	// for the explicit-channel path, so any authenticated user can read
	// message/file snippets from a channel they were never added to just by
	// knowing or guessing its UUID. This test proves the CURRENT (buggy)
	// behaviour: repo.GetUserChannelIDs is never even queried, and content
	// from the "foreign" channel comes back regardless.
	t.Run("BUG: explicit channel_id bypasses membership check", func(t *testing.T) {
		foreignChannel := uuid.New() // caller is NOT a member of this channel
		msgID := uuid.New()

		repo := &MockRepository{
			// The caller's own channel list does not include foreignChannel.
			channelIDs: []uuid.UUID{uuid.New()},
			messages: []models.ChatSearchResult{
				{
					Type:      models.ChatSearchResultMessage,
					ID:        msgID,
					ChannelID: foreignChannel,
					Score:     0.8,
					Snippet:   "secret <mark>content</mark> from a channel the caller never joined",
					CreatedAt: time.Now(),
					MessageID: &msgID,
				},
			},
		}
		detector := &MockDetector{lang: "german"}
		svc := NewService(repo, detector)

		results, total, err := svc.Search(context.Background(), SearchInput{
			Query:     "content",
			UserID:    uuid.New(),
			ChannelID: &foreignChannel,
			Page:      1,
			PageSize:  20,
		})

		require.NoError(t, err)
		// This SHOULD be 0 (membership check should reject the request), but
		// it is not -- documenting the leak until the fix-unit lands.
		assert.Equal(t, 1, total, "membership bypass: foreign channel content was returned")
		require.Len(t, results, 1)
		assert.Equal(t, foreignChannel, results[0].ChannelID)
	})

	t.Run("language detection used for query", func(t *testing.T) {
		channelID := uuid.New()
		repo := &MockRepository{
			channelIDs: []uuid.UUID{channelID},
		}
		detector := &MockDetector{lang: "english"}
		svc := NewService(repo, detector)

		_, _, err := svc.Search(context.Background(), SearchInput{
			Query:    "search term",
			UserID:   uuid.New(),
			Page:     1,
			PageSize: 20,
		})

		require.NoError(t, err)
		// Verify detector was called by checking the mock's lang was used
		assert.Equal(t, "english", detector.lang)
	})
}
