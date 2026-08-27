package server

// Tests for the six GoBD Belegarchiv gRPC handlers (§147 AO).
//
// The archive is the one place in the finance service where the law, not the
// product, defines correctness: a Beleg must stay retrievable, unchanged and
// tenant-bound for ten years. The handlers below therefore get the same
// treatment as an auth boundary — every read path is asserted to fail closed
// for a foreign tenant, and AddDocumentAnnotation is asserted to leave the
// archived document itself byte-identical.
//
// gobdArchiveSvc is a concrete *gobdarchive.Service, not an interface, so the
// seam sits one layer lower: a stub Repository plus a stub FileStore behind a
// real gobdarchive.Service. That keeps the service's own invariants (SHA-256
// via TeeReader, retention = 31.12 of year+10, event append) in the test path
// instead of mocking them away.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/biz/gobdarchive"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/pdf"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// Test doubles
// ============================================================================

// stubGobdRepo mirrors the tenant behaviour of the Postgres repository: a
// document that belongs to another tenant is indistinguishable from one that
// does not exist (that is what RLS produces on the real repository).
type stubGobdRepo struct {
	docs   map[uuid.UUID]*models.GobdDocument
	events map[uuid.UUID][]*models.GobdDocumentEvent

	createErr     error
	getErr        error
	listErr       error
	appendErr     error
	listEventsErr error

	listResult []*models.GobdDocument
	listTotal  int
	lastFilter gobdarchive.ListFilter

	appendedEvents []*models.GobdDocumentEvent
}

func newStubGobdRepo() *stubGobdRepo {
	return &stubGobdRepo{
		docs:   make(map[uuid.UUID]*models.GobdDocument),
		events: make(map[uuid.UUID][]*models.GobdDocumentEvent),
	}
}

func (r *stubGobdRepo) Create(_ context.Context, doc *models.GobdDocument) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.docs[doc.ID] = doc
	return nil
}

func (r *stubGobdRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.GobdDocument, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	doc, ok := r.docs[id]
	if !ok || doc.TenantID != tenantID {
		return nil, gobdarchive.ErrDocumentNotFound
	}
	return doc, nil
}

func (r *stubGobdRepo) List(_ context.Context, filter gobdarchive.ListFilter) ([]*models.GobdDocument, int, error) {
	r.lastFilter = filter
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	if r.listResult != nil {
		return r.listResult, r.listTotal, nil
	}
	result := make([]*models.GobdDocument, 0)
	for _, doc := range r.docs {
		if doc.TenantID == filter.TenantID {
			result = append(result, doc)
		}
	}
	return result, len(result), nil
}

func (r *stubGobdRepo) AppendEvent(_ context.Context, event *models.GobdDocumentEvent) error {
	if r.appendErr != nil {
		return r.appendErr
	}
	r.appendedEvents = append(r.appendedEvents, event)
	r.events[event.DocumentID] = append(r.events[event.DocumentID], event)
	return nil
}

func (r *stubGobdRepo) ListEvents(_ context.Context, tenantID, documentID uuid.UUID) ([]*models.GobdDocumentEvent, error) {
	if r.listEventsErr != nil {
		return nil, r.listEventsErr
	}
	result := make([]*models.GobdDocumentEvent, 0)
	for _, ev := range r.events[documentID] {
		if ev.TenantID == tenantID {
			result = append(result, ev)
		}
	}
	return result, nil
}

// eventsOfType counts appended events by type — used to prove that a failed
// read never leaves an audit trace on a foreign tenant's document.
func (r *stubGobdRepo) eventsOfType(eventType string) int {
	n := 0
	for _, ev := range r.appendedEvents {
		if ev.EventType == eventType {
			n++
		}
	}
	return n
}

// stubGobdStore is a chatfile.FileStore that keeps uploads in memory.
type stubGobdStore struct {
	uploads      map[string][]byte
	uploadErr    error
	presignErr   error
	presignedURL string
	presignCalls int
}

func newStubGobdStore() *stubGobdStore {
	return &stubGobdStore{
		uploads:      make(map[string][]byte),
		presignedURL: "https://minio.example/gobd/presigned",
	}
}

func (s *stubGobdStore) Upload(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	if s.uploadErr != nil {
		return s.uploadErr
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	s.uploads[key] = data
	return nil
}

func (s *stubGobdStore) Download(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := s.uploads[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *stubGobdStore) Delete(_ context.Context, key string) error {
	delete(s.uploads, key)
	return nil
}

func (s *stubGobdStore) GetPresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	s.presignCalls++
	if s.presignErr != nil {
		return "", s.presignErr
	}
	return s.presignedURL, nil
}

func (s *stubGobdStore) GetPresignedUploadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", errors.New("not used")
}

// newGobdTestServer wires a BizGRPCServer with nothing but the archive service,
// so a handler that reaches for another dependency fails loudly instead of
// silently passing.
func newGobdTestServer(repo *stubGobdRepo, store *stubGobdStore) *BizGRPCServer {
	return &BizGRPCServer{gobdArchiveSvc: gobdarchive.NewService(repo, store, nil)}
}

// newGobdInvoiceTestServer additionally wires the invoice service and a real
// PDF generator, as ArchiveInvoiceDocument renders the invoice before archiving.
func newGobdInvoiceTestServer(repo *stubGobdRepo, store *stubGobdStore, invRepo *stubInvoiceRepo) *BizGRPCServer {
	invSvc := invoice.NewService(invRepo, nil, &stubCompanySettingsRepo{}, nil, fakeTxBeginner{})
	return &BizGRPCServer{
		invoiceService: invSvc,
		pdfGenerator:   pdf.NewGenerator(pdfReadyCompanySettings()),
		gobdArchiveSvc: gobdarchive.NewService(repo, store, invSvc),
	}
}

// pdfReadyCompanySettings satisfies ValidateCompanySettingsForPDF (§14 UStG
// Pflichtangaben) so the generator produces bytes instead of a validation error.
func pdfReadyCompanySettings() models.CompanySettings {
	return models.CompanySettings{
		Name:         "Zentria UG",
		Street:       "Mainzer Strasse 47",
		PLZ:          "55124",
		City:         "Mainz",
		Steuernummer: "26/123/45678",
	}
}

func archivedDoc(tenantID uuid.UUID) *models.GobdDocument {
	now := time.Date(2026, time.March, 15, 10, 0, 0, 0, time.UTC)
	return &models.GobdDocument{
		ID:               uuid.New(),
		TenantID:         tenantID,
		DocType:          models.GobdDocTypeReceipt,
		StorageKey:       "gobd/" + tenantID.String() + "/beleg.pdf",
		SHA256:           "deadbeef",
		OriginalFilename: "beleg.pdf",
		MimeType:         "application/pdf",
		FileSizeBytes:    1234,
		ArchivedAt:       now,
		ArchivedBy:       uuid.New(),
		RetentionUntil:   time.Date(2036, time.December, 31, 0, 0, 0, 0, time.UTC),
	}
}

// ============================================================================
// ArchiveDocument
// ============================================================================

func TestArchiveDocument(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	validReq := func() *bizv1.ArchiveDocumentRequest {
		return &bizv1.ArchiveDocumentRequest{
			TenantId:         tenantID.String(),
			DocType:          models.GobdDocTypeReceipt,
			OriginalFilename: "beleg.pdf",
			MimeType:         "application/pdf",
			Content:          []byte("%PDF-1.7 receipt bytes"),
			ArchivedBy:       userID.String(),
		}
	}

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		req := validReq()
		req.TenantId = "not-a-uuid"
		_, err := srv.ArchiveDocument(context.Background(), req)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid archived_by", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		req := validReq()
		req.ArchivedBy = "not-a-uuid"
		_, err := srv.ArchiveDocument(context.Background(), req)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("empty content is rejected before any upload", func(t *testing.T) {
		store := newStubGobdStore()
		srv := newGobdTestServer(newStubGobdRepo(), store)
		req := validReq()
		req.Content = nil
		_, err := srv.ArchiveDocument(context.Background(), req)
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Empty(t, store.uploads, "no bytes may reach storage for an empty archive request")
	})

	t.Run("invalid source_invoice_id", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		req := validReq()
		req.SourceInvoiceId = "not-a-uuid"
		_, err := srv.ArchiveDocument(context.Background(), req)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("unknown doc_type maps to InvalidArgument", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		req := validReq()
		req.DocType = "sonstiges"
		_, err := srv.ArchiveDocument(context.Background(), req)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("success stores the document under the caller tenant with a matching hash", func(t *testing.T) {
		repo := newStubGobdRepo()
		store := newStubGobdStore()
		srv := newGobdTestServer(repo, store)

		req := validReq()
		resp, err := srv.ArchiveDocument(context.Background(), req)
		requireGRPCOK(t, err)
		require.NotNil(t, resp.GetDocument())

		doc := resp.GetDocument()
		assert.Equal(t, tenantID.String(), doc.GetTenantId())
		assert.Equal(t, models.GobdDocTypeReceipt, doc.GetDocType())
		assert.Equal(t, "beleg.pdf", doc.GetOriginalFilename())
		assert.Equal(t, "application/pdf", doc.GetMimeType())
		assert.Equal(t, int64(len(req.GetContent())), doc.GetFileSizeBytes())
		assert.Equal(t, userID.String(), doc.GetArchivedBy())
		assert.Empty(t, doc.GetSourceInvoiceId(), "a document without an invoice must report an empty id, not a zero uuid")

		// The hash must be the hash of the archived bytes — this is the whole
		// point of the SHA-256 column (§147 AO Unveraenderbarkeit).
		sum := sha256.Sum256(req.GetContent())
		assert.Equal(t, hex.EncodeToString(sum[:]), doc.GetSha256())

		// Retention: 31.12 of archival year + 10 (§147 Abs. 3 AO).
		archivedYear := doc.GetArchivedAt().AsTime().UTC().Year()
		assert.Equal(t, fmt.Sprintf("%d-12-31", archivedYear+10), doc.GetRetentionUntil())

		// Bytes really landed in storage under the returned key.
		assert.Equal(t, req.GetContent(), store.uploads[doc.GetStorageKey()])

		// The archival is on the audit trail.
		assert.Equal(t, 1, repo.eventsOfType(models.GobdEventTypeArchived))
	})

	t.Run("source_invoice_id is carried through", func(t *testing.T) {
		invoiceID := uuid.New()
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		req := validReq()
		req.DocType = models.GobdDocTypeInvoice
		req.SourceInvoiceId = invoiceID.String()

		resp, err := srv.ArchiveDocument(context.Background(), req)
		requireGRPCOK(t, err)
		assert.Equal(t, invoiceID.String(), resp.GetDocument().GetSourceInvoiceId())
	})

	t.Run("storage failure maps to Internal", func(t *testing.T) {
		store := newStubGobdStore()
		store.uploadErr = errors.New("minio unreachable")
		srv := newGobdTestServer(newStubGobdRepo(), store)
		_, err := srv.ArchiveDocument(context.Background(), validReq())
		requireGRPCCode(t, err, codes.Internal)
	})

	t.Run("persist failure maps to Internal", func(t *testing.T) {
		repo := newStubGobdRepo()
		repo.createErr = errors.New("connection reset by peer")
		srv := newGobdTestServer(repo, newStubGobdStore())
		_, err := srv.ArchiveDocument(context.Background(), validReq())
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// ArchiveInvoiceDocument
// ============================================================================

func TestArchiveInvoiceDocument(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	lockedInvoice := func(tid uuid.UUID) *models.Invoice {
		inv := draftInvoice(tid)
		inv.InvoiceNumber = "RE-2026-0042"
		inv.Status = models.InvoiceStatusSent
		lockedAt := time.Date(2026, time.February, 1, 12, 0, 0, 0, time.UTC)
		inv.LockedAt = &lockedAt
		return inv
	}

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newGobdInvoiceTestServer(newStubGobdRepo(), newStubGobdStore(), newStubInvoiceRepo())
		_, err := srv.ArchiveInvoiceDocument(context.Background(), &bizv1.ArchiveInvoiceDocumentRequest{
			TenantId: "not-a-uuid", InvoiceId: uuid.NewString(), ArchivedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid invoice_id", func(t *testing.T) {
		srv := newGobdInvoiceTestServer(newStubGobdRepo(), newStubGobdStore(), newStubInvoiceRepo())
		_, err := srv.ArchiveInvoiceDocument(context.Background(), &bizv1.ArchiveInvoiceDocumentRequest{
			TenantId: tenantID.String(), InvoiceId: "not-a-uuid", ArchivedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid archived_by", func(t *testing.T) {
		srv := newGobdInvoiceTestServer(newStubGobdRepo(), newStubGobdStore(), newStubInvoiceRepo())
		_, err := srv.ArchiveInvoiceDocument(context.Background(), &bizv1.ArchiveInvoiceDocumentRequest{
			TenantId: tenantID.String(), InvoiceId: uuid.NewString(), ArchivedBy: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("foreign tenant invoice is not archived", func(t *testing.T) {
		invRepo := newStubInvoiceRepo()
		inv := lockedInvoice(uuid.New()) // belongs to another tenant
		invRepo.invoices[inv.ID] = inv

		repo := newStubGobdRepo()
		store := newStubGobdStore()
		srv := newGobdInvoiceTestServer(repo, store, invRepo)

		_, err := srv.ArchiveInvoiceDocument(context.Background(), &bizv1.ArchiveInvoiceDocumentRequest{
			TenantId: tenantID.String(), InvoiceId: inv.ID.String(), ArchivedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
		assert.Empty(t, repo.docs, "a foreign invoice must never enter this tenant's archive")
		assert.Empty(t, store.uploads)
	})

	t.Run("unlocked invoice maps to FailedPrecondition", func(t *testing.T) {
		invRepo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID) // LockedAt == nil
		inv.InvoiceNumber = "RE-2026-0043"
		invRepo.invoices[inv.ID] = inv

		repo := newStubGobdRepo()
		srv := newGobdInvoiceTestServer(repo, newStubGobdStore(), invRepo)

		_, err := srv.ArchiveInvoiceDocument(context.Background(), &bizv1.ArchiveInvoiceDocumentRequest{
			TenantId: tenantID.String(), InvoiceId: inv.ID.String(), ArchivedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
		assert.Empty(t, repo.docs, "GoBD: only finalized documents are archived")
	})

	t.Run("unrenderable invoice maps to Internal and archives nothing", func(t *testing.T) {
		invRepo := newStubInvoiceRepo()
		inv := lockedInvoice(tenantID)
		invRepo.invoices[inv.ID] = inv

		repo := newStubGobdRepo()
		store := newStubGobdStore()
		srv := newGobdInvoiceTestServer(repo, store, invRepo)
		// Incomplete company settings fail ValidateCompanySettingsForPDF (§14 UStG),
		// so the renderer errors before anything reaches the archive.
		srv.pdfGenerator = pdf.NewGenerator(models.CompanySettings{})

		_, err := srv.ArchiveInvoiceDocument(context.Background(), &bizv1.ArchiveInvoiceDocumentRequest{
			TenantId: tenantID.String(), InvoiceId: inv.ID.String(), ArchivedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.Internal)
		assert.Empty(t, repo.docs, "a document that could not be rendered must not be recorded as archived")
		assert.Empty(t, store.uploads)
	})

	t.Run("locked invoice is archived as an invoice document", func(t *testing.T) {
		invRepo := newStubInvoiceRepo()
		inv := lockedInvoice(tenantID)
		invRepo.invoices[inv.ID] = inv

		repo := newStubGobdRepo()
		store := newStubGobdStore()
		srv := newGobdInvoiceTestServer(repo, store, invRepo)

		resp, err := srv.ArchiveInvoiceDocument(context.Background(), &bizv1.ArchiveInvoiceDocumentRequest{
			TenantId: tenantID.String(), InvoiceId: inv.ID.String(), ArchivedBy: userID.String(),
		})
		requireGRPCOK(t, err)
		doc := resp.GetDocument()
		require.NotNil(t, doc)
		assert.Equal(t, models.GobdDocTypeInvoice, doc.GetDocType())
		assert.Equal(t, inv.ID.String(), doc.GetSourceInvoiceId())
		assert.Equal(t, "invoice-RE-2026-0042.pdf", doc.GetOriginalFilename())
		assert.Equal(t, "application/pdf", doc.GetMimeType())
		assert.Equal(t, tenantID.String(), doc.GetTenantId())
		assert.NotEmpty(t, store.uploads[doc.GetStorageKey()], "the rendered PDF must be in storage")
	})
}

// ============================================================================
// GetGobdDocument
// ============================================================================

func TestGetGobdDocument(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.GetGobdDocument(context.Background(), &bizv1.GetGobdDocumentRequest{
			TenantId: "not-a-uuid", Id: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid id", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.GetGobdDocument(context.Background(), &bizv1.GetGobdDocumentRequest{
			TenantId: tenantID.String(), Id: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("unknown document maps to NotFound", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.GetGobdDocument(context.Background(), &bizv1.GetGobdDocumentRequest{
			TenantId: tenantID.String(), Id: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("document of a foreign tenant is not disclosed", func(t *testing.T) {
		repo := newStubGobdRepo()
		foreign := archivedDoc(uuid.New())
		repo.docs[foreign.ID] = foreign

		srv := newGobdTestServer(repo, newStubGobdStore())
		resp, err := srv.GetGobdDocument(context.Background(), &bizv1.GetGobdDocumentRequest{
			TenantId: tenantID.String(), Id: foreign.ID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("success returns document plus audit trail", func(t *testing.T) {
		repo := newStubGobdRepo()
		doc := archivedDoc(tenantID)
		repo.docs[doc.ID] = doc
		repo.events[doc.ID] = []*models.GobdDocumentEvent{
			{
				ID: uuid.New(), TenantID: tenantID, DocumentID: doc.ID,
				EventType: models.GobdEventTypeArchived, CreatedBy: doc.ArchivedBy,
				CreatedAt: doc.ArchivedAt,
			},
			{
				ID: uuid.New(), TenantID: tenantID, DocumentID: doc.ID,
				EventType: models.GobdEventTypeAnnotation, CreatedBy: uuid.New(),
				CreatedAt: doc.ArchivedAt.Add(time.Hour), Note: "Beleg nachgereicht",
				Metadata: []byte(`{"source":"import"}`),
			},
		}

		srv := newGobdTestServer(repo, newStubGobdStore())
		resp, err := srv.GetGobdDocument(context.Background(), &bizv1.GetGobdDocumentRequest{
			TenantId: tenantID.String(), Id: doc.ID.String(),
		})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.GetDocument())
		assert.Equal(t, doc.ID.String(), resp.GetDocument().GetId())
		assert.Equal(t, doc.SHA256, resp.GetDocument().GetSha256())
		assert.Equal(t, "2036-12-31", resp.GetDocument().GetRetentionUntil())

		require.Len(t, resp.GetEvents(), 2)
		assert.Equal(t, models.GobdEventTypeArchived, resp.GetEvents()[0].GetEventType())
		// Missing metadata must serialize as an empty JSON object, never as "".
		assert.Equal(t, "{}", resp.GetEvents()[0].GetMetadataJson())
		assert.Equal(t, `{"source":"import"}`, resp.GetEvents()[1].GetMetadataJson())
		assert.Equal(t, "Beleg nachgereicht", resp.GetEvents()[1].GetNote())
	})

	t.Run("audit trail failure maps to Internal", func(t *testing.T) {
		repo := newStubGobdRepo()
		doc := archivedDoc(tenantID)
		repo.docs[doc.ID] = doc
		repo.listEventsErr = errors.New("connection reset by peer")

		srv := newGobdTestServer(repo, newStubGobdStore())
		_, err := srv.GetGobdDocument(context.Background(), &bizv1.GetGobdDocumentRequest{
			TenantId: tenantID.String(), Id: doc.ID.String(),
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// ListGobdDocuments
// ============================================================================

func TestListGobdDocuments(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.ListGobdDocuments(context.Background(), &bizv1.ListGobdDocumentsRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid source_invoice_id", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.ListGobdDocuments(context.Background(), &bizv1.ListGobdDocumentsRequest{
			TenantId: tenantID.String(), SourceInvoiceId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid date_from", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.ListGobdDocuments(context.Background(), &bizv1.ListGobdDocumentsRequest{
			TenantId: tenantID.String(), DateFrom: "15.03.2026",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid date_to", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.ListGobdDocuments(context.Background(), &bizv1.ListGobdDocumentsRequest{
			TenantId: tenantID.String(), DateTo: "2026-13-01",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("filters reach the repository unchanged", func(t *testing.T) {
		repo := newStubGobdRepo()
		invoiceID := uuid.New()
		srv := newGobdTestServer(repo, newStubGobdStore())

		_, err := srv.ListGobdDocuments(context.Background(), &bizv1.ListGobdDocumentsRequest{
			TenantId:        tenantID.String(),
			DocType:         models.GobdDocTypeInvoice,
			SourceInvoiceId: invoiceID.String(),
			DateFrom:        "2026-01-01",
			DateTo:          "2026-12-31",
			Page:            3,
			PerPage:         25,
		})
		requireGRPCOK(t, err)

		assert.Equal(t, tenantID, repo.lastFilter.TenantID)
		assert.Equal(t, models.GobdDocTypeInvoice, repo.lastFilter.DocType)
		require.NotNil(t, repo.lastFilter.SourceInvoiceID)
		assert.Equal(t, invoiceID, *repo.lastFilter.SourceInvoiceID)
		require.NotNil(t, repo.lastFilter.DateFrom)
		assert.Equal(t, "2026-01-01", repo.lastFilter.DateFrom.Format("2006-01-02"))
		require.NotNil(t, repo.lastFilter.DateTo)
		assert.Equal(t, "2026-12-31", repo.lastFilter.DateTo.Format("2006-01-02"))
		assert.Equal(t, 3, repo.lastFilter.Page)
		assert.Equal(t, 25, repo.lastFilter.PerPage)
	})

	t.Run("only own-tenant documents are listed", func(t *testing.T) {
		repo := newStubGobdRepo()
		own := archivedDoc(tenantID)
		foreign := archivedDoc(uuid.New())
		repo.docs[own.ID] = own
		repo.docs[foreign.ID] = foreign

		srv := newGobdTestServer(repo, newStubGobdStore())
		resp, err := srv.ListGobdDocuments(context.Background(), &bizv1.ListGobdDocumentsRequest{
			TenantId: tenantID.String(),
		})
		requireGRPCOK(t, err)
		require.Len(t, resp.GetDocuments(), 1)
		assert.Equal(t, own.ID.String(), resp.GetDocuments()[0].GetId())
		assert.Equal(t, int32(1), resp.GetTotal())
	})

	t.Run("empty result defaults page and per_page", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		resp, err := srv.ListGobdDocuments(context.Background(), &bizv1.ListGobdDocumentsRequest{
			TenantId: tenantID.String(),
		})
		requireGRPCOK(t, err)
		assert.NotNil(t, resp.GetDocuments(), "an empty list must serialize as [], never as null")
		assert.Empty(t, resp.GetDocuments())
		assert.Equal(t, int32(0), resp.GetTotal())
		assert.Equal(t, int32(1), resp.GetPage())
		assert.Equal(t, int32(50), resp.GetPerPage())
	})

	t.Run("repository failure maps to Internal", func(t *testing.T) {
		repo := newStubGobdRepo()
		repo.listErr = errors.New("connection reset by peer")
		srv := newGobdTestServer(repo, newStubGobdStore())
		_, err := srv.ListGobdDocuments(context.Background(), &bizv1.ListGobdDocumentsRequest{
			TenantId: tenantID.String(),
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// DownloadGobdDocument
// ============================================================================

func TestDownloadGobdDocument(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.DownloadGobdDocument(context.Background(), &bizv1.DownloadGobdDocumentRequest{
			TenantId: "not-a-uuid", Id: uuid.NewString(), RequestedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid id", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.DownloadGobdDocument(context.Background(), &bizv1.DownloadGobdDocumentRequest{
			TenantId: tenantID.String(), Id: "not-a-uuid", RequestedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid requested_by", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.DownloadGobdDocument(context.Background(), &bizv1.DownloadGobdDocumentRequest{
			TenantId: tenantID.String(), Id: uuid.NewString(), RequestedBy: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no presigned url is minted for a foreign tenant's document", func(t *testing.T) {
		repo := newStubGobdRepo()
		foreign := archivedDoc(uuid.New())
		repo.docs[foreign.ID] = foreign
		store := newStubGobdStore()

		srv := newGobdTestServer(repo, store)
		_, err := srv.DownloadGobdDocument(context.Background(), &bizv1.DownloadGobdDocumentRequest{
			TenantId: tenantID.String(), Id: foreign.ID.String(), RequestedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
		assert.Zero(t, store.presignCalls, "a foreign document must not be presigned — the URL needs no auth")
		assert.Zero(t, repo.eventsOfType(models.GobdEventTypeAccess),
			"a rejected read must not write an access event on a foreign document")
	})

	t.Run("unknown document maps to NotFound", func(t *testing.T) {
		store := newStubGobdStore()
		srv := newGobdTestServer(newStubGobdRepo(), store)
		_, err := srv.DownloadGobdDocument(context.Background(), &bizv1.DownloadGobdDocumentRequest{
			TenantId: tenantID.String(), Id: uuid.NewString(), RequestedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
		assert.Zero(t, store.presignCalls)
	})

	t.Run("success returns url plus filename and records an access event", func(t *testing.T) {
		repo := newStubGobdRepo()
		doc := archivedDoc(tenantID)
		repo.docs[doc.ID] = doc
		store := newStubGobdStore()

		srv := newGobdTestServer(repo, store)
		resp, err := srv.DownloadGobdDocument(context.Background(), &bizv1.DownloadGobdDocumentRequest{
			TenantId: tenantID.String(), Id: doc.ID.String(), RequestedBy: userID.String(),
		})
		requireGRPCOK(t, err)
		assert.Equal(t, store.presignedURL, resp.GetPresignedUrl())
		assert.Equal(t, doc.OriginalFilename, resp.GetOriginalFilename())
		assert.Equal(t, doc.MimeType, resp.GetMimeType())

		require.Equal(t, 1, repo.eventsOfType(models.GobdEventTypeAccess))
		ev := repo.appendedEvents[0]
		assert.Equal(t, tenantID, ev.TenantID)
		assert.Equal(t, doc.ID, ev.DocumentID)
		assert.Equal(t, userID, ev.CreatedBy)
	})

	t.Run("presign failure maps to Internal", func(t *testing.T) {
		repo := newStubGobdRepo()
		doc := archivedDoc(tenantID)
		repo.docs[doc.ID] = doc
		store := newStubGobdStore()
		store.presignErr = errors.New("minio unreachable")

		srv := newGobdTestServer(repo, store)
		_, err := srv.DownloadGobdDocument(context.Background(), &bizv1.DownloadGobdDocumentRequest{
			TenantId: tenantID.String(), Id: doc.ID.String(), RequestedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// AddDocumentAnnotation
// ============================================================================

func TestAddDocumentAnnotation(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.AddDocumentAnnotation(context.Background(), &bizv1.AddDocumentAnnotationRequest{
			TenantId: "not-a-uuid", DocumentId: uuid.NewString(), AddedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid document_id", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.AddDocumentAnnotation(context.Background(), &bizv1.AddDocumentAnnotationRequest{
			TenantId: tenantID.String(), DocumentId: "not-a-uuid", AddedBy: userID.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid added_by", func(t *testing.T) {
		srv := newGobdTestServer(newStubGobdRepo(), newStubGobdStore())
		_, err := srv.AddDocumentAnnotation(context.Background(), &bizv1.AddDocumentAnnotationRequest{
			TenantId: tenantID.String(), DocumentId: uuid.NewString(), AddedBy: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("foreign tenant document cannot be annotated", func(t *testing.T) {
		repo := newStubGobdRepo()
		foreign := archivedDoc(uuid.New())
		repo.docs[foreign.ID] = foreign

		srv := newGobdTestServer(repo, newStubGobdStore())
		_, err := srv.AddDocumentAnnotation(context.Background(), &bizv1.AddDocumentAnnotationRequest{
			TenantId: tenantID.String(), DocumentId: foreign.ID.String(),
			AddedBy: userID.String(), Note: "fremder Beleg",
		})
		requireGRPCCode(t, err, codes.NotFound)
		assert.Empty(t, repo.appendedEvents, "no audit entry may be written on a foreign tenant's document")
	})

	t.Run("annotation lands on the audit trail and leaves the document untouched", func(t *testing.T) {
		repo := newStubGobdRepo()
		doc := archivedDoc(tenantID)
		repo.docs[doc.ID] = doc
		before := *doc // value copy taken before the call

		srv := newGobdTestServer(repo, newStubGobdStore())
		resp, err := srv.AddDocumentAnnotation(context.Background(), &bizv1.AddDocumentAnnotationRequest{
			TenantId: tenantID.String(), DocumentId: doc.ID.String(),
			AddedBy: userID.String(), Note: "Zahlungseingang manuell geprueft",
		})
		requireGRPCOK(t, err)
		assert.True(t, resp.GetOk())

		require.Len(t, repo.appendedEvents, 1)
		ev := repo.appendedEvents[0]
		assert.Equal(t, models.GobdEventTypeAnnotation, ev.EventType)
		assert.Equal(t, "Zahlungseingang manuell geprueft", ev.Note)
		assert.Equal(t, userID, ev.CreatedBy)
		assert.Equal(t, tenantID, ev.TenantID)
		assert.Equal(t, doc.ID, ev.DocumentID)
		assert.JSONEq(t, "{}", string(ev.Metadata))

		// §147 AO: the archived Beleg itself is immutable. The annotation goes
		// into gobd_document_events; not one field of gobd_documents may move.
		assert.Equal(t, before, *repo.docs[doc.ID],
			"annotating a document must not modify the archived document record")
	})

	t.Run("audit append failure maps to Internal", func(t *testing.T) {
		repo := newStubGobdRepo()
		doc := archivedDoc(tenantID)
		repo.docs[doc.ID] = doc
		repo.appendErr = errors.New("connection reset by peer")

		srv := newGobdTestServer(repo, newStubGobdStore())
		_, err := srv.AddDocumentAnnotation(context.Background(), &bizv1.AddDocumentAnnotationRequest{
			TenantId: tenantID.String(), DocumentId: doc.ID.String(),
			AddedBy: userID.String(), Note: "note",
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// mapGobdArchiveError
// ============================================================================

func TestMapGobdArchiveError(t *testing.T) {
	assert.NoError(t, mapGobdArchiveError(nil))

	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"not found", gobdarchive.ErrDocumentNotFound, codes.NotFound},
		{"immutable", gobdarchive.ErrDocumentImmutable, codes.FailedPrecondition},
		{"invalid doc type", gobdarchive.ErrInvalidDocumentType, codes.InvalidArgument},
		{"file too large", gobdarchive.ErrFileTooLarge, codes.InvalidArgument},
		{"empty file", gobdarchive.ErrEmptyFile, codes.InvalidArgument},
		{"invoice not locked", gobdarchive.ErrInvoiceNotLocked, codes.FailedPrecondition},
		{"opaque error", errors.New("connection reset by peer"), codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapGobdArchiveError(tc.err), tc.code)
		})
	}

	// Wrapped sentinels must keep their code — the service wraps with %w.
	requireGRPCCode(t, mapGobdArchiveError(
		errors.Join(errors.New("persist gobd document"), gobdarchive.ErrDocumentNotFound)), codes.NotFound)
}

// ============================================================================
// toProtoGobdDocument / toProtoGobdEventList
// ============================================================================

func TestToProtoGobdDocumentNil(t *testing.T) {
	assert.Nil(t, toProtoGobdDocument(nil))
	assert.NotNil(t, toProtoGobdEventList(nil), "an empty event list must serialize as [], never as null")
	assert.Empty(t, toProtoGobdEventList(nil))
}
