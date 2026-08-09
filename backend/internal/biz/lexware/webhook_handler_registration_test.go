package lexware

// Registration/unregistration paths for WebhookHandler, exercised against a
// real HTTP server (httptest) the same way bexio's contact_sync_happy_test.go
// does — RegisterWebhooks and UnregisterWebhooks reach cs.client.do, so a
// mocked Repository alone cannot drive them.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClientLW creates a *Client pointing at the given httptest.Server,
// pre-loaded with a fixed API key so Client.do never has to hit the vault.
func newTestClientLW(srv *httptest.Server) *Client {
	cfg := ClientConfig{BaseURL: srv.URL + "/"}
	vault := &mockVaultService{
		getSecretFn: func(context.Context, string) (string, error) {
			return "test-api-key", nil
		},
	}
	return NewClient(cfg, vault)
}

func TestWebhookHandler_RegisterWebhooks_PersistsEverySubscription(t *testing.T) {
	tenantID := uuid.New()
	configID := uuid.New()

	var registeredEventTypes []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sub WebhookSubscription
		require.NoError(t, json.NewDecoder(r.Body).Decode(&sub))
		registeredEventTypes = append(registeredEventTypes, sub.EventType)
		sub.ID = "sub-" + sub.EventType
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(sub))
	}))
	defer srv.Close()

	var persisted []string
	repo := &mockRepository{}
	// upsertWebhookSubscriptionFn is not a mockRepository field — track via
	// UpsertWebhookSubscription's real signature by wrapping the mock.
	wh := NewWebhookHandler(newTestClientLW(srv), &trackingWebhookRepo{mockRepository: repo, onUpsert: func(subscriptionID, eventType string) {
		persisted = append(persisted, eventType)
		_ = subscriptionID
	}}, &mockEmitter{}, nil)

	err := wh.RegisterWebhooks(context.Background(), configID, tenantID, "https://app.zentria.tech/webhooks/lexware")
	require.NoError(t, err)

	// All four event types the handler subscribes to must both hit the API
	// and be persisted — losing either half would silently break either the
	// live subscription or KMU Hub's own bookkeeping of it.
	wantEvents := []string{"contact.created", "contact.changed", "invoice.status.changed", "quotation.status.changed"}
	assert.ElementsMatch(t, wantEvents, registeredEventTypes)
	assert.ElementsMatch(t, wantEvents, persisted)
}

func TestWebhookHandler_RegisterWebhooks_APIFailureSkipsPersistForThatEvent(t *testing.T) {
	// 401 returns immediately without the client's retry/backoff loop (unlike
	// 5xx, which would make this test spend several real seconds sleeping) —
	// still enough to prove a failed registration call never reaches the
	// persistence step for that event.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	var persistCount int
	repo := &trackingWebhookRepo{mockRepository: &mockRepository{}, onUpsert: func(string, string) { persistCount++ }}
	wh := NewWebhookHandler(newTestClientLW(srv), repo, &mockEmitter{}, nil)

	err := wh.RegisterWebhooks(context.Background(), uuid.New(), uuid.New(), "https://cb.example.com")
	require.NoError(t, err) // RegisterWebhooks never returns an error itself — failures are logged per event.
	assert.Equal(t, 0, persistCount)
}

func TestWebhookHandler_UnregisterWebhooks_DeletesEverySubscription(t *testing.T) {
	configID := uuid.New()

	var deletedIDs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		deletedIDs = append(deletedIDs, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	subs := []WebhookSubscriptionRecord{
		{ID: uuid.New(), ConfigID: configID, SubscriptionID: "sub-1", EventType: "contact.created"},
		{ID: uuid.New(), ConfigID: configID, SubscriptionID: "sub-2", EventType: "contact.changed"},
	}
	var deletedRecordIDs []uuid.UUID
	tracking := &trackingWebhookRepo{
		mockRepository: &mockRepository{},
		onList: func() []WebhookSubscriptionRecord {
			return subs
		},
		onDelete: func(id uuid.UUID) {
			deletedRecordIDs = append(deletedRecordIDs, id)
		},
	}

	wh := NewWebhookHandler(newTestClientLW(srv), tracking, &mockEmitter{}, nil)

	err := wh.UnregisterWebhooks(context.Background(), configID, uuid.New())
	require.NoError(t, err)

	require.Len(t, deletedRecordIDs, 2)
	assert.Contains(t, deletedIDs, "/v1/event-subscriptions/sub-1")
	assert.Contains(t, deletedIDs, "/v1/event-subscriptions/sub-2")
}

func TestWebhookHandler_UnregisterWebhooks_NoSubscriptionsIsANoop(t *testing.T) {
	// mockRepository.ListWebhookSubscriptions defaults to (nil, nil) — the
	// zero-subscriptions path must return cleanly without calling the client.
	wh := NewWebhookHandler(nil, &mockRepository{}, &mockEmitter{}, nil)

	err := wh.UnregisterWebhooks(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
}

// trackingWebhookRepo wraps mockRepository to observe UpsertWebhookSubscription
// / ListWebhookSubscriptions / DeleteWebhookSubscription calls without
// widening mockRepository's shared struct (used by many other tests in this
// package) with fields only these tests need.
type trackingWebhookRepo struct {
	*mockRepository
	onUpsert func(subscriptionID, eventType string)
	onList   func() []WebhookSubscriptionRecord
	onDelete func(id uuid.UUID)
}

func (r *trackingWebhookRepo) UpsertWebhookSubscription(_ context.Context, _ uuid.UUID, subscriptionID, eventType, _ string) error {
	if r.onUpsert != nil {
		r.onUpsert(subscriptionID, eventType)
	}
	return nil
}

func (r *trackingWebhookRepo) ListWebhookSubscriptions(context.Context, uuid.UUID) ([]WebhookSubscriptionRecord, error) {
	if r.onList != nil {
		return r.onList(), nil
	}
	return nil, nil
}

func (r *trackingWebhookRepo) DeleteWebhookSubscription(_ context.Context, id uuid.UUID) error {
	if r.onDelete != nil {
		r.onDelete(id)
	}
	return nil
}
