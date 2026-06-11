package search

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock Repository ---

type MockRepository struct {
	results    []SearchResult
	total      int
	failSearch bool
}

func (m *MockRepository) Search(_ context.Context, _ string, _ SearchFilter) ([]SearchResult, int, error) {
	if m.failSearch {
		return nil, 0, errors.New("repo search error")
	}
	return m.results, m.total, nil
}

// --- Mock FileStore ---

type MockFileStore struct {
	content      string
	failDownload bool
}

func (m *MockFileStore) Upload(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return nil
}

func (m *MockFileStore) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	if m.failDownload {
		return nil, errors.New("store download error")
	}
	return io.NopCloser(strings.NewReader(m.content)), nil
}

func (m *MockFileStore) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *MockFileStore) GetPresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}

func (m *MockFileStore) GetPresignedUploadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}

// --- Mock FileRepo ---

type MockFileRepo struct {
	indexed   map[uuid.UUID]string
	failUpdate bool
}

func NewMockFileRepo() *MockFileRepo {
	return &MockFileRepo{indexed: make(map[uuid.UUID]string)}
}

func (m *MockFileRepo) UpdateSearchContent(_ context.Context, id uuid.UUID, text string) error {
	if m.failUpdate {
		return errors.New("repo update search content error")
	}
	m.indexed[id] = text
	return nil
}

// --- Helper ---

func newTestService() (*Service, *MockRepository, *MockFileStore) {
	repo := &MockRepository{}
	store := &MockFileStore{content: "extracted text content"}
	extractor := NewExtractor(DefaultMaxIndexBytes)
	svc := NewService(repo, extractor, store)
	return svc, repo, store
}

func sampleSearchResults() ([]SearchResult, int) {
	results := []SearchResult{
		{
			File: models.DocumentFile{
				ID:       uuid.New(),
				Filename: "report.pdf",
				MimeType: "application/pdf",
				FileSize: 1024,
			},
			Rank:    0.95,
			Snippet: "...matching text...",
		},
		{
			File: models.DocumentFile{
				ID:       uuid.New(),
				Filename: "notes.txt",
				MimeType: "text/plain",
				FileSize: 512,
			},
			Rank:    0.72,
			Snippet: "...another match...",
		},
	}
	return results, len(results)
}

// --- Tests ---

func TestSearch_Success(t *testing.T) {
	svc, repo, _ := newTestService()
	results, total := sampleSearchResults()
	repo.results = results
	repo.total = total

	got, count, err := svc.Search(context.Background(), "report", SearchFilter{Limit: 10})

	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Len(t, got, 2)
	assert.Equal(t, "report.pdf", got[0].File.Filename)
}

func TestSearch_EmptyQuery(t *testing.T) {
	svc, _, _ := newTestService()

	results, count, err := svc.Search(context.Background(), "", SearchFilter{})

	assert.Nil(t, results)
	assert.Equal(t, 0, count)
	assert.ErrorIs(t, err, ErrSearchQueryRequired)
}

func TestSearch_WhitespaceQuery(t *testing.T) {
	svc, _, _ := newTestService()

	results, count, err := svc.Search(context.Background(), "   ", SearchFilter{})

	assert.Nil(t, results)
	assert.Equal(t, 0, count)
	assert.ErrorIs(t, err, ErrSearchQueryRequired)
}

func TestSearch_Empty(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.results = []SearchResult{}
	repo.total = 0

	results, count, err := svc.Search(context.Background(), "nonexistent", SearchFilter{})

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, results)
}

func TestSearch_RepoError(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.failSearch = true

	results, count, err := svc.Search(context.Background(), "query", SearchFilter{})

	assert.Nil(t, results)
	assert.Equal(t, 0, count)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repo search error")
}

func TestExtractAndIndex_Success(t *testing.T) {
	svc, _, store := newTestService()
	store.content = "This is the extracted document text."
	fileRepo := NewMockFileRepo()
	fileID := uuid.New()

	err := svc.ExtractAndIndex(context.Background(), fileID, "documents/test.txt", "text/plain", fileRepo)

	require.NoError(t, err)
	assert.Equal(t, "This is the extracted document text.", fileRepo.indexed[fileID])
}

func TestExtractAndIndex_UnsupportedFormat(t *testing.T) {
	svc, _, _ := newTestService()
	fileRepo := NewMockFileRepo()
	fileID := uuid.New()

	// image/png is not extractable — should return nil (skip, not error)
	err := svc.ExtractAndIndex(context.Background(), fileID, "documents/photo.png", "image/png", fileRepo)

	require.NoError(t, err)
	// Nothing should be indexed
	_, indexed := fileRepo.indexed[fileID]
	assert.False(t, indexed)
}

func TestExtractAndIndex_DownloadError(t *testing.T) {
	svc, _, store := newTestService()
	store.failDownload = true
	fileRepo := NewMockFileRepo()
	fileID := uuid.New()

	err := svc.ExtractAndIndex(context.Background(), fileID, "documents/test.txt", "text/plain", fileRepo)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store download error")
}
