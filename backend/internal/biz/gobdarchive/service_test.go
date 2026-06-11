package gobdarchive

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// ============================================================================
// Mock Repository
// ============================================================================

type mockRepository struct {
	docs   map[uuid.UUID]*models.GobdDocument
	events []*models.GobdDocumentEvent

	createErr      error
	getErr         error
	appendEventErr error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		docs: make(map[uuid.UUID]*models.GobdDocument),
	}
}

func (m *mockRepository) Create(_ context.Context, doc *models.GobdDocument) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.docs[doc.ID] = doc
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.GobdDocument, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	doc, ok := m.docs[id]
	if !ok || doc.TenantID != tenantID {
		return nil, ErrDocumentNotFound
	}
	return doc, nil
}

func (m *mockRepository) List(_ context.Context, _ ListFilter) ([]*models.GobdDocument, int, error) {
	var docs []*models.GobdDocument
	for _, d := range m.docs {
		docs = append(docs, d)
	}
	return docs, len(docs), nil
}

func (m *mockRepository) AppendEvent(_ context.Context, event *models.GobdDocumentEvent) error {
	if m.appendEventErr != nil {
		return m.appendEventErr
	}
	m.events = append(m.events, event)
	return nil
}

func (m *mockRepository) ListEvents(_ context.Context, _, documentID uuid.UUID) ([]*models.GobdDocumentEvent, error) {
	var result []*models.GobdDocumentEvent
	for _, ev := range m.events {
		if ev.DocumentID == documentID {
			result = append(result, ev)
		}
	}
	return result, nil
}

// ============================================================================
// Mock FileStore
// ============================================================================

type mockFileStore struct {
	uploadErr error
	uploaded  map[string][]byte
}

func newMockFileStore() *mockFileStore {
	return &mockFileStore{uploaded: make(map[string][]byte)}
}

func (m *mockFileStore) Upload(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(reader)
	m.uploaded[key] = buf.Bytes()
	return nil
}

func (m *mockFileStore) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockFileStore) Delete(_ context.Context, _ string) error { return nil }

func (m *mockFileStore) GetPresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "https://presigned.example.com/test", nil
}

func (m *mockFileStore) GetPresignedUploadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "https://presigned.example.com/upload/test", nil
}

// ============================================================================
// Helpers
// ============================================================================

func newTestService() (*Service, *mockRepository, *mockFileStore) {
	repo := newMockRepository()
	store := newMockFileStore()
	svc := NewService(repo, store, nil)
	return svc, repo, store
}

// ============================================================================
// computeRetentionUntil tests
// ============================================================================

func TestComputeRetentionUntil_Normal(t *testing.T) {
	t.Parallel()
	// 2026-03-15 → retention until 2034-12-31
	archivedAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	got := computeRetentionUntil(archivedAt)
	assert.Equal(t, time.Date(2034, 12, 31, 0, 0, 0, 0, time.UTC), got)
}

func TestComputeRetentionUntil_Dec31(t *testing.T) {
	t.Parallel()
	// 2026-12-31 → retention still 2034-12-31 (same year)
	archivedAt := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	got := computeRetentionUntil(archivedAt)
	assert.Equal(t, time.Date(2034, 12, 31, 0, 0, 0, 0, time.UTC), got)
}

func TestComputeRetentionUntil_Jan1(t *testing.T) {
	t.Parallel()
	// 2027-01-01 → retention until 2035-12-31
	archivedAt := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	got := computeRetentionUntil(archivedAt)
	assert.Equal(t, time.Date(2035, 12, 31, 0, 0, 0, 0, time.UTC), got)
}

// ============================================================================
// ArchiveDocument — SHA-256 correctness
// ============================================================================

func TestArchiveDocument_SHA256Correctness(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newTestService()

	content := []byte("Hello, GoBD archive!")
	expectedHash := sha256.Sum256(content)
	expectedHex := hex.EncodeToString(expectedHash[:])

	doc, err := svc.ArchiveDocument(context.Background(), ArchiveInput{
		TenantID:         uuid.New(),
		DocType:          models.GobdDocTypeInvoice,
		OriginalFilename: "test.pdf",
		MimeType:         "application/pdf",
		Content:          bytes.NewReader(content),
		Size:             int64(len(content)),
		ArchivedBy:       uuid.New(),
	})
	require.NoError(t, err)
	assert.Equal(t, expectedHex, doc.SHA256)

	// Verify document is stored in repo
	require.Len(t, repo.docs, 1)
}

// ============================================================================
// ArchiveDocument — EmptyFile rejection
// ============================================================================

func TestArchiveDocument_EmptyFileRejected(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestService()

	_, err := svc.ArchiveDocument(context.Background(), ArchiveInput{
		TenantID:         uuid.New(),
		DocType:          models.GobdDocTypeReceipt,
		OriginalFilename: "empty.pdf",
		MimeType:         "application/pdf",
		Content:          bytes.NewReader(nil),
		Size:             0,
		ArchivedBy:       uuid.New(),
	})
	assert.ErrorIs(t, err, ErrEmptyFile)
}

// ============================================================================
// ArchiveDocument — FileTooLarge rejection
// ============================================================================

func TestArchiveDocument_FileTooLargeRejected(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestService()

	_, err := svc.ArchiveDocument(context.Background(), ArchiveInput{
		TenantID:         uuid.New(),
		DocType:          models.GobdDocTypeReceipt,
		OriginalFilename: "large.pdf",
		MimeType:         "application/pdf",
		Content:          strings.NewReader("x"),
		Size:             maxArchiveBytes + 1,
		ArchivedBy:       uuid.New(),
	})
	assert.ErrorIs(t, err, ErrFileTooLarge)
}

// ============================================================================
// ArchiveDocument — InvalidDocType rejection
// ============================================================================

func TestArchiveDocument_InvalidDocType(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestService()

	_, err := svc.ArchiveDocument(context.Background(), ArchiveInput{
		TenantID:         uuid.New(),
		DocType:          "not_a_real_type",
		OriginalFilename: "file.pdf",
		MimeType:         "application/pdf",
		Content:          strings.NewReader("data"),
		Size:             4,
		ArchivedBy:       uuid.New(),
	})
	assert.ErrorIs(t, err, ErrInvalidDocumentType)
}

// ============================================================================
// AddAnnotation — not-found document
// ============================================================================

func TestAddAnnotation_DocumentNotFound(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestService()

	err := svc.AddAnnotation(context.Background(), uuid.New(), uuid.New(), uuid.New(), "some note")
	assert.ErrorIs(t, err, ErrDocumentNotFound)
}

// ============================================================================
// AddAnnotation — success appends event
// ============================================================================

func TestAddAnnotation_Success(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newTestService()

	tenantID := uuid.New()
	docID := uuid.New()
	// Manually insert a document into the mock repo.
	repo.docs[docID] = &models.GobdDocument{
		ID:       docID,
		TenantID: tenantID,
	}

	err := svc.AddAnnotation(context.Background(), tenantID, docID, uuid.New(), "Compliance note")
	require.NoError(t, err)
	require.Len(t, repo.events, 1)
	assert.Equal(t, models.GobdEventTypeAnnotation, repo.events[0].EventType)
	assert.Equal(t, "Compliance note", repo.events[0].Note)
}

// ============================================================================
// ArchiveInvoiceDocument — invoice not locked → ErrInvoiceNotLocked
// ============================================================================

type mockInvoiceReader struct {
	inv *models.Invoice
	err error
}

func (m *mockInvoiceReader) GetByID(_ context.Context, _, _ uuid.UUID) (*models.Invoice, error) {
	return m.inv, m.err
}

func TestArchiveInvoiceDocument_InvoiceNotLocked(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	store := newMockFileStore()

	invReader := &mockInvoiceReader{
		inv: &models.Invoice{
			ID:            uuid.New(),
			InvoiceNumber: "RE-2026-0001",
			LockedAt:      nil, // not locked
		},
	}
	svc := NewService(repo, store, invReader)

	_, err := svc.ArchiveInvoiceDocument(context.Background(), uuid.New(), uuid.New(), uuid.New(), []byte("pdf"))
	assert.ErrorIs(t, err, ErrInvoiceNotLocked)
}
