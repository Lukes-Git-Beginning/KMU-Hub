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
	// 2026-03-15 → retention until 2036-12-31 (§147 AO: 10 years)
	archivedAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	got := computeRetentionUntil(archivedAt)
	assert.Equal(t, time.Date(2036, 12, 31, 0, 0, 0, 0, time.UTC), got)
}

func TestComputeRetentionUntil_Dec31(t *testing.T) {
	t.Parallel()
	// 2026-12-31 → retention still 2036-12-31 (same year)
	archivedAt := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	got := computeRetentionUntil(archivedAt)
	assert.Equal(t, time.Date(2036, 12, 31, 0, 0, 0, 0, time.UTC), got)
}

func TestComputeRetentionUntil_Jan1(t *testing.T) {
	t.Parallel()
	// 2027-01-01 → retention until 2037-12-31
	archivedAt := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	got := computeRetentionUntil(archivedAt)
	assert.Equal(t, time.Date(2037, 12, 31, 0, 0, 0, 0, time.UTC), got)
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

// ============================================================================
// ArchiveInvoiceDocument — no invoice reader configured
// ============================================================================

func TestArchiveInvoiceDocument_NoInvoiceReaderConfigured(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestService() // NewService(repo, store, nil)

	_, err := svc.ArchiveInvoiceDocument(context.Background(), uuid.New(), uuid.New(), uuid.New(), []byte("pdf"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invoice reader not configured")
}

// ============================================================================
// ArchiveInvoiceDocument — invoice lookup failure propagates
// ============================================================================

func TestArchiveInvoiceDocument_InvoiceLookupErrorPropagates(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	store := newMockFileStore()
	lookupErr := errors.New("invoice service unavailable")
	invReader := &mockInvoiceReader{err: lookupErr}
	svc := NewService(repo, store, invReader)

	_, err := svc.ArchiveInvoiceDocument(context.Background(), uuid.New(), uuid.New(), uuid.New(), []byte("pdf"))
	assert.ErrorIs(t, err, lookupErr)
	assert.Empty(t, repo.docs, "no document must be persisted when the invoice lookup fails")
}

// ============================================================================
// ArchiveInvoiceDocument — filename derivation
// ============================================================================

func TestArchiveInvoiceDocument_FilenameUsesInvoiceNumberWhenPresent(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	store := newMockFileStore()
	invID := uuid.New()
	locked := time.Now().UTC()
	invReader := &mockInvoiceReader{
		inv: &models.Invoice{ID: invID, InvoiceNumber: "RE-2026-0042", LockedAt: &locked},
	}
	svc := NewService(repo, store, invReader)

	doc, err := svc.ArchiveInvoiceDocument(context.Background(), uuid.New(), invID, uuid.New(), []byte("pdf-bytes"))
	require.NoError(t, err)
	assert.Equal(t, "invoice-RE-2026-0042.pdf", doc.OriginalFilename)
	assert.Equal(t, models.GobdDocTypeInvoice, doc.DocType)
	require.NotNil(t, doc.SourceInvoiceID)
	assert.Equal(t, invID, *doc.SourceInvoiceID)
}

func TestArchiveInvoiceDocument_FilenameFallsBackToIDWhenNumberBlank(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	store := newMockFileStore()
	invID := uuid.New()
	locked := time.Now().UTC()
	invReader := &mockInvoiceReader{
		inv: &models.Invoice{ID: invID, InvoiceNumber: "", LockedAt: &locked},
	}
	svc := NewService(repo, store, invReader)

	doc, err := svc.ArchiveInvoiceDocument(context.Background(), uuid.New(), invID, uuid.New(), []byte("pdf-bytes"))
	require.NoError(t, err)
	assert.Equal(t, "invoice-"+invID.String()+".pdf", doc.OriginalFilename)
}

// ============================================================================
// ArchiveDocument — upload failure propagates, nothing is persisted
// ============================================================================

func TestArchiveDocument_UploadFailurePropagatesAndSkipsCreate(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	store := newMockFileStore()
	store.uploadErr = errors.New("minio unreachable")
	svc := NewService(repo, store, nil)

	_, err := svc.ArchiveDocument(context.Background(), ArchiveInput{
		TenantID:         uuid.New(),
		DocType:          models.GobdDocTypeOther,
		OriginalFilename: "file.pdf",
		MimeType:         "application/pdf",
		Content:          strings.NewReader("data"),
		Size:             4,
		ArchivedBy:       uuid.New(),
	})
	require.Error(t, err)
	assert.Empty(t, repo.docs, "a failed upload must never produce a persisted document record")
}

// ============================================================================
// ArchiveDocument — repository Create failure propagates
// ============================================================================

func TestArchiveDocument_RepositoryCreateFailurePropagates(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	repo.createErr = errors.New("db unavailable")
	store := newMockFileStore()
	svc := NewService(repo, store, nil)

	_, err := svc.ArchiveDocument(context.Background(), ArchiveInput{
		TenantID:         uuid.New(),
		DocType:          models.GobdDocTypeOther,
		OriginalFilename: "file.pdf",
		MimeType:         "application/pdf",
		Content:          strings.NewReader("data"),
		Size:             4,
		ArchivedBy:       uuid.New(),
	})
	require.Error(t, err)
}

// ============================================================================
// ArchiveDocument — the recorded checksum reflects exactly the bytes that
// were streamed to storage, so any later divergence between stored bytes and
// SHA256 (e.g. storage-layer corruption or tampering) becomes detectable by
// recomputing the hash from storage and comparing it to the immutable
// doc.SHA256 on record.
// ============================================================================

func TestArchiveDocument_ChecksumDetectsStorageCorruption(t *testing.T) {
	t.Parallel()
	svc, repo, store := newTestService()

	content := []byte("original, unmodified invoice content")
	doc, err := svc.ArchiveDocument(context.Background(), ArchiveInput{
		TenantID:         uuid.New(),
		DocType:          models.GobdDocTypeReceipt,
		OriginalFilename: "receipt.pdf",
		MimeType:         "application/pdf",
		Content:          bytes.NewReader(content),
		Size:             int64(len(content)),
		ArchivedBy:       uuid.New(),
	})
	require.NoError(t, err)

	// The checksum recorded at archival time matches exactly what was streamed to storage.
	storedHash := sha256.Sum256(store.uploaded[doc.StorageKey])
	assert.Equal(t, doc.SHA256, hex.EncodeToString(storedHash[:]))

	// Simulate the stored object being tampered with after archival (e.g. a
	// compromised storage backend). The archive record's checksum is
	// immutable — recomputing the hash from the (now corrupted) stored bytes
	// no longer matches, which is precisely how tampering becomes visible.
	store.uploaded[doc.StorageKey] = []byte("tampered content, different from the original")
	corruptedHash := sha256.Sum256(store.uploaded[doc.StorageKey])
	assert.NotEqual(t, doc.SHA256, hex.EncodeToString(corruptedHash[:]))
	assert.Equal(t, doc.SHA256, repo.docs[doc.ID].SHA256, "the persisted record's checksum must never change")
}

// ============================================================================
// GetByID — passthrough with tenant scoping delegated to the repository
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newTestService()
	tenantID, docID := uuid.New(), uuid.New()
	repo.docs[docID] = &models.GobdDocument{ID: docID, TenantID: tenantID}

	doc, err := svc.GetByID(context.Background(), tenantID, docID)
	require.NoError(t, err)
	assert.Equal(t, docID, doc.ID)
}

func TestService_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestService()

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrDocumentNotFound)
}

// ============================================================================
// List — passthrough
// ============================================================================

func TestService_List_Passthrough(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newTestService()
	tenantID := uuid.New()
	repo.docs[uuid.New()] = &models.GobdDocument{ID: uuid.New(), TenantID: tenantID}
	repo.docs[uuid.New()] = &models.GobdDocument{ID: uuid.New(), TenantID: tenantID}

	docs, total, err := svc.List(context.Background(), ListFilter{TenantID: tenantID})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, docs, 2)
}

// ============================================================================
// GetDownloadURL
// ============================================================================

func TestService_GetDownloadURL_Success_AppendsAccessEvent(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newTestService()
	tenantID, docID := uuid.New(), uuid.New()
	repo.docs[docID] = &models.GobdDocument{ID: docID, TenantID: tenantID, StorageKey: "gobd/x/y/z.pdf"}

	url, err := svc.GetDownloadURL(context.Background(), tenantID, docID, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "https://presigned.example.com/test", url)

	require.Len(t, repo.events, 1)
	assert.Equal(t, models.GobdEventTypeAccess, repo.events[0].EventType)
	assert.Equal(t, docID, repo.events[0].DocumentID)
}

func TestService_GetDownloadURL_DocumentNotFound(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestService()

	_, err := svc.GetDownloadURL(context.Background(), uuid.New(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrDocumentNotFound)
}

type erroringFileStore struct {
	*mockFileStore
	presignErr error
}

func (m *erroringFileStore) GetPresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", m.presignErr
}

func TestService_GetDownloadURL_PresignErrorPropagates(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	tenantID, docID := uuid.New(), uuid.New()
	repo.docs[docID] = &models.GobdDocument{ID: docID, TenantID: tenantID}
	store := &erroringFileStore{mockFileStore: newMockFileStore(), presignErr: errors.New("s3 down")}
	svc := NewService(repo, store, nil)

	_, err := svc.GetDownloadURL(context.Background(), tenantID, docID, uuid.New())
	require.Error(t, err)
	assert.Empty(t, repo.events, "no access event must be recorded when the presign call itself fails")
}

func TestService_GetDownloadURL_AccessEventAppendFailureIsNonFatal(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	tenantID, docID := uuid.New(), uuid.New()
	repo.docs[docID] = &models.GobdDocument{ID: docID, TenantID: tenantID}
	repo.appendEventErr = errors.New("events table unavailable")
	store := newMockFileStore()
	svc := NewService(repo, store, nil)

	url, err := svc.GetDownloadURL(context.Background(), tenantID, docID, uuid.New())
	require.NoError(t, err, "a failed access-log append must not fail the download itself")
	assert.NotEmpty(t, url)
}

// ============================================================================
// AddAnnotation — repository append failure propagates
// ============================================================================

func TestAddAnnotation_AppendEventFailurePropagates(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	tenantID, docID := uuid.New(), uuid.New()
	repo.docs[docID] = &models.GobdDocument{ID: docID, TenantID: tenantID}
	repo.appendEventErr = errors.New("events table unavailable")
	store := newMockFileStore()
	svc := NewService(repo, store, nil)

	err := svc.AddAnnotation(context.Background(), tenantID, docID, uuid.New(), "note")
	require.Error(t, err)
}

// ============================================================================
// ListEvents
// ============================================================================

func TestService_ListEvents_DocumentNotFound(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestService()

	_, err := svc.ListEvents(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrDocumentNotFound)
}

func TestService_ListEvents_Success(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newTestService()
	tenantID, docID := uuid.New(), uuid.New()
	repo.docs[docID] = &models.GobdDocument{ID: docID, TenantID: tenantID}
	repo.events = append(repo.events,
		&models.GobdDocumentEvent{ID: uuid.New(), DocumentID: docID, EventType: models.GobdEventTypeArchived},
		&models.GobdDocumentEvent{ID: uuid.New(), DocumentID: uuid.New(), EventType: models.GobdEventTypeArchived}, // other document
	)

	events, err := svc.ListEvents(context.Background(), tenantID, docID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, docID, events[0].DocumentID)
}
