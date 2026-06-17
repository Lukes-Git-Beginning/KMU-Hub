package bexio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// mockQuoteReader is a minimal QuoteReader mock.
type mockQuoteReader struct {
	getByIDFn func(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error)
}

func (m *mockQuoteReader) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return nil, errors.New("not implemented")
}

// testQuote returns a minimal valid Quote for push tests.
func testQuote(quoteID, tenantID uuid.UUID) *models.Quote {
	return &models.Quote{
		ID:            quoteID,
		TenantID:      tenantID,
		QuoteNumber:   "AN-2024-001",
		Status:        "draft",
		CustomerEmail: "customer@example.com",
		GrossTotal:    decimal.NewFromFloat(99.00),
		CreatedAt:     time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Now().UTC(),
	}
}

// buildQuotePusher builds a QuotePusher with an httptest server.
func buildQuotePusher(
	srv *httptest.Server,
	tenantID uuid.UUID,
	repo Repository,
	quotes QuoteReader,
	contacts ContactService,
) *QuotePusher {
	client := newTestClient(srv)
	seedToken(client, tenantID)
	return &QuotePusher{
		client:      client,
		repo:        repo,
		fieldMapper: NewFieldMapper(),
		quotes:      quotes,
		contacts:    contacts,
	}
}

// TestPushQuote_HappyPath_Create verifies that a new quote is created in Bexio
// and the mapping is persisted when no prior mapping exists.
func TestPushQuote_HappyPath_Create(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	quoteID := uuid.New()
	contactBexioID := 21
	createdBexioQuoteID := 400

	createdBexioQuote := BexioQuote{ID: createdBexioQuoteID}
	rawCreated, _ := json.Marshal(createdBexioQuote)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/kb_offer" {
			w.Write(rawCreated)
			return
		}
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	q := testQuote(quoteID, tenantID)
	quotes := &mockQuoteReader{
		getByIDFn: func(_ context.Context, _, id uuid.UUID) (*models.Quote, error) {
			assert.Equal(t, quoteID, id)
			return q, nil
		},
	}
	contacts := &mockContactService{
		getByEmailFn: func(_ context.Context, _ string) (*ContactResult, error) {
			return &ContactResult{ID: uuid.New()}, nil
		},
	}
	repo := &fullSyncRepo{
		getEntityMappingFn: func(_ context.Context, _ uuid.UUID, entityType string, _ uuid.UUID) (*models.BexioEntityMapping, error) {
			if entityType == "contact" {
				return &models.BexioEntityMapping{BexioID: contactBexioID}, nil
			}
			return nil, ErrMappingNotFound
		},
	}

	pusher := buildQuotePusher(srv, tenantID, repo, quotes, contacts)

	err := pusher.PushQuote(context.Background(), configID, tenantID, quoteID)

	require.NoError(t, err)
	assert.Equal(t, 1, repo.upsertEntityMappingCalls, "new quote mapping must be upserted exactly once")
}

// TestPushQuote_ContactResolveFails verifies that contact resolution errors
// are returned before any Bexio API call.
func TestPushQuote_ContactResolveFails(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	quoteID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Bexio API call: %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer srv.Close()

	q := testQuote(quoteID, tenantID)
	quotes := &mockQuoteReader{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Quote, error) {
			return q, nil
		},
	}
	contacts := &mockContactService{
		getByEmailFn: func(_ context.Context, _ string) (*ContactResult, error) {
			return nil, errors.New("crm lookup failed")
		},
	}
	repo := &fullSyncRepo{}

	pusher := buildQuotePusher(srv, tenantID, repo, quotes, contacts)

	err := pusher.PushQuote(context.Background(), configID, tenantID, quoteID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve contact")
	assert.Equal(t, 0, repo.upsertEntityMappingCalls)
}

// TestPushQuote_BexioAPIFails verifies that a persistent Bexio API error is
// returned and no mapping is created.
func TestPushQuote_BexioAPIFails(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	quoteID := uuid.New()
	contactBexioID := 22

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	q := testQuote(quoteID, tenantID)
	quotes := &mockQuoteReader{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Quote, error) {
			return q, nil
		},
	}
	contacts := &mockContactService{
		getByEmailFn: func(_ context.Context, _ string) (*ContactResult, error) {
			return &ContactResult{ID: uuid.New()}, nil
		},
	}
	repo := &fullSyncRepo{
		getEntityMappingFn: func(_ context.Context, _ uuid.UUID, entityType string, _ uuid.UUID) (*models.BexioEntityMapping, error) {
			if entityType == "contact" {
				return &models.BexioEntityMapping{BexioID: contactBexioID}, nil
			}
			return nil, ErrMappingNotFound
		},
	}

	pusher := buildQuotePusher(srv, tenantID, repo, quotes, contacts)

	err := pusher.PushQuote(context.Background(), configID, tenantID, quoteID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create in bexio")
	assert.Equal(t, 0, repo.upsertEntityMappingCalls)
}

// TestPushQuote_Idempotent_ExistingMapping verifies that re-pushing a quote
// with an existing mapping triggers the update path and does not create a
// duplicate mapping.
func TestPushQuote_Idempotent_ExistingMapping(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	quoteID := uuid.New()
	contactBexioID := 23
	existingBexioQuoteID := 401

	updatedBexioQuote := BexioQuote{ID: existingBexioQuoteID}
	rawUpdated, _ := json.Marshal(updatedBexioQuote)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(rawUpdated)
	}))
	defer srv.Close()

	q := testQuote(quoteID, tenantID)
	quotes := &mockQuoteReader{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Quote, error) {
			return q, nil
		},
	}
	contacts := &mockContactService{
		getByEmailFn: func(_ context.Context, _ string) (*ContactResult, error) {
			return &ContactResult{ID: uuid.New()}, nil
		},
	}

	existingMapping := &models.BexioEntityMapping{
		ID:         uuid.New(),
		ConfigID:   configID,
		EntityType: "quote",
		KmuhubID:   quoteID,
		BexioID:    existingBexioQuoteID,
	}

	repo := &fullSyncRepo{
		getEntityMappingFn: func(_ context.Context, _ uuid.UUID, entityType string, _ uuid.UUID) (*models.BexioEntityMapping, error) {
			if entityType == "contact" {
				return &models.BexioEntityMapping{BexioID: contactBexioID}, nil
			}
			return existingMapping, nil
		},
	}

	pusher := buildQuotePusher(srv, tenantID, repo, quotes, contacts)

	// Push 1
	require.NoError(t, pusher.PushQuote(context.Background(), configID, tenantID, quoteID))
	// Push 2 — idempotent
	require.NoError(t, pusher.PushQuote(context.Background(), configID, tenantID, quoteID))

	assert.Equal(t, 2, repo.upsertEntityMappingCalls,
		"each push with existing mapping must upsert once (update path)")
}

// TestPushQuote_QuoteReaderError verifies that a failure to load the quote
// is returned immediately without reaching the Bexio API.
func TestPushQuote_QuoteReaderError(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	quoteID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Bexio API call: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	quotes := &mockQuoteReader{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Quote, error) {
			return nil, errors.New("quote not found")
		},
	}
	repo := &fullSyncRepo{}

	pusher := buildQuotePusher(srv, tenantID, repo, quotes, nil)

	err := pusher.PushQuote(context.Background(), configID, tenantID, quoteID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "load quote")
}
