package formulare

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Helpers
// ============================================================================

// newWorkerWithHTTPClient creates a WebhookWorker with the given http.Client.
func newWorkerWithHTTPClient(repo Repository, client *http.Client) *WebhookWorker {
	w := NewWebhookWorker(repo, nil)
	w.httpClient = client
	return w
}

// newDelivery builds a minimal WebhookDelivery with given attempt count.
func newDelivery(webhookID, tenantID uuid.UUID, attemptCount, maxAttempts int) *WebhookDelivery {
	return &WebhookDelivery{
		ID:            uuid.New(),
		WebhookID:     webhookID,
		SubmissionID:  uuid.New(),
		TenantID:      tenantID,
		Payload:       []byte(`{"submission_id":"test"}`),
		Status:        WebhookDeliveryStatusPending,
		AttemptCount:  attemptCount,
		MaxAttempts:   maxAttempts,
		NextAttemptAt: time.Time{},
		CreatedAt:     time.Now().UTC(),
	}
}

// newWebhook builds a minimal FormWebhook.
func newWebhook(id, tenantID, formSchemaID uuid.UUID, urlStr string, active bool, secret *string) *FormWebhook {
	return &FormWebhook{
		ID:           id,
		TenantID:     tenantID,
		FormSchemaID: formSchemaID,
		URL:          urlStr,
		Active:       active,
		Secret:       secret,
		Events:       []string{"submission.created"},
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
}

// ============================================================================
// TestWorker_SuccessfulDelivery_MarksDelivered
// ============================================================================

func TestWorker_SuccessfulDelivery_MarksDelivered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := newStubRepo()
	tenantID := uuid.New()
	webhookID := uuid.New()
	schemaID := uuid.New()

	wh := newWebhook(webhookID, tenantID, schemaID, server.URL, true, nil)
	repo.webhooks[webhookID] = wh

	del := newDelivery(webhookID, tenantID, 0, 5)
	repo.deliveries[del.ID] = del

	worker := newWorkerWithHTTPClient(repo, server.Client())
	worker.deliver(context.Background(), del)

	repo.mu.Lock()
	updated := repo.deliveries[del.ID]
	repo.mu.Unlock()

	if updated.Status != WebhookDeliveryStatusDelivered {
		t.Errorf("expected delivered, got %s", updated.Status)
	}
	if updated.AttemptCount != 1 {
		t.Errorf("expected attempt_count=1, got %d", updated.AttemptCount)
	}
}

// ============================================================================
// TestWorker_ServerError_IncrementsAttempt
// ============================================================================

func TestWorker_ServerError_IncrementsAttempt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	repo := newStubRepo()
	tenantID := uuid.New()
	webhookID := uuid.New()
	schemaID := uuid.New()

	wh := newWebhook(webhookID, tenantID, schemaID, server.URL, true, nil)
	repo.webhooks[webhookID] = wh

	del := newDelivery(webhookID, tenantID, 0, 5)
	repo.deliveries[del.ID] = del

	worker := newWorkerWithHTTPClient(repo, server.Client())
	worker.deliver(context.Background(), del)

	repo.mu.Lock()
	updated := repo.deliveries[del.ID]
	repo.mu.Unlock()

	if updated.Status != WebhookDeliveryStatusFailed {
		t.Errorf("expected failed, got %s", updated.Status)
	}
	if updated.AttemptCount != 1 {
		t.Errorf("expected attempt_count=1, got %d", updated.AttemptCount)
	}
	if updated.LastError == nil {
		t.Error("expected last_error to be set")
	}
}

// ============================================================================
// TestWorker_MaxAttemptsReached_MarksDead
// ============================================================================

func TestWorker_MaxAttemptsReached_MarksDead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	repo := newStubRepo()
	tenantID := uuid.New()
	webhookID := uuid.New()
	schemaID := uuid.New()

	wh := newWebhook(webhookID, tenantID, schemaID, server.URL, true, nil)
	repo.webhooks[webhookID] = wh

	// AttemptCount=4, MaxAttempts=5 → next attempt = 5 = max → dead
	del := newDelivery(webhookID, tenantID, 4, 5)
	repo.deliveries[del.ID] = del

	worker := newWorkerWithHTTPClient(repo, server.Client())
	worker.deliver(context.Background(), del)

	repo.mu.Lock()
	updated := repo.deliveries[del.ID]
	repo.mu.Unlock()

	if updated.Status != WebhookDeliveryStatusDead {
		t.Errorf("expected dead, got %s", updated.Status)
	}
	if updated.AttemptCount != 5 {
		t.Errorf("expected attempt_count=5, got %d", updated.AttemptCount)
	}
}

// ============================================================================
// TestWorker_HMACSignatureCorrect
// ============================================================================

func TestWorker_HMACSignatureCorrect(t *testing.T) {
	secret := "webhook-secret-key"
	payload := []byte(`{"submission_id":"test-123"}`)

	var receivedSig string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Cosmi-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := newStubRepo()
	tenantID := uuid.New()
	webhookID := uuid.New()
	schemaID := uuid.New()

	wh := newWebhook(webhookID, tenantID, schemaID, server.URL, true, &secret)
	repo.webhooks[webhookID] = wh

	del := &WebhookDelivery{
		ID:           uuid.New(),
		WebhookID:    webhookID,
		SubmissionID: uuid.New(),
		TenantID:     tenantID,
		Payload:      payload,
		Status:       WebhookDeliveryStatusPending,
		AttemptCount: 0,
		MaxAttempts:  5,
		CreatedAt:    time.Now().UTC(),
	}
	repo.deliveries[del.ID] = del

	worker := newWorkerWithHTTPClient(repo, server.Client())
	worker.deliver(context.Background(), del)

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if receivedSig != expectedSig {
		t.Errorf("HMAC mismatch: got %q, want %q", receivedSig, expectedSig)
	}
}

// ============================================================================
// TestWorker_NoSecret_NoSignatureHeader
// ============================================================================

func TestWorker_NoSecret_NoSignatureHeader(t *testing.T) {
	var receivedSig string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Cosmi-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := newStubRepo()
	tenantID := uuid.New()
	webhookID := uuid.New()
	schemaID := uuid.New()

	wh := newWebhook(webhookID, tenantID, schemaID, server.URL, true, nil)
	repo.webhooks[webhookID] = wh

	del := newDelivery(webhookID, tenantID, 0, 5)
	repo.deliveries[del.ID] = del

	worker := newWorkerWithHTTPClient(repo, server.Client())
	worker.deliver(context.Background(), del)

	if receivedSig != "" {
		t.Errorf("expected no X-Cosmi-Signature header, got %q", receivedSig)
	}
}

// ============================================================================
// TestWorker_NetworkError_IncrementsAttempt
// ============================================================================

func TestWorker_NetworkError_IncrementsAttempt(t *testing.T) {
	// Create and immediately close a server so connections fail
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // Close immediately to force connection error

	repo := newStubRepo()
	tenantID := uuid.New()
	webhookID := uuid.New()
	schemaID := uuid.New()

	wh := newWebhook(webhookID, tenantID, schemaID, serverURL, true, nil)
	repo.webhooks[webhookID] = wh

	del := newDelivery(webhookID, tenantID, 0, 5)
	repo.deliveries[del.ID] = del

	worker := NewWebhookWorker(repo, nil)
	worker.deliver(context.Background(), del)

	repo.mu.Lock()
	updated := repo.deliveries[del.ID]
	repo.mu.Unlock()

	if updated.Status != WebhookDeliveryStatusFailed {
		t.Errorf("expected failed on network error, got %s", updated.Status)
	}
	if updated.AttemptCount != 1 {
		t.Errorf("expected attempt_count=1, got %d", updated.AttemptCount)
	}
}

// ============================================================================
// TestWorker_BackoffExponential
// ============================================================================

func TestWorker_BackoffExponential(t *testing.T) {
	// Verify backoffFor returns the expected durations
	cases := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 30 * time.Second},
		{2, 2 * time.Minute},
		{3, 10 * time.Minute},
		{4, 30 * time.Minute},
		{5, 2 * time.Hour}, // clamped to last
	}
	for _, tc := range cases {
		got := backoffFor(tc.attempt)
		if got != tc.expected {
			t.Errorf("backoffFor(%d) = %v, want %v", tc.attempt, got, tc.expected)
		}
	}
}

// ============================================================================
// TestWorker_WebhookDeactivated_MarksDead
// ============================================================================

func TestWorker_WebhookDeactivated_MarksDead(t *testing.T) {
	repo := newStubRepo()
	tenantID := uuid.New()
	webhookID := uuid.New()
	schemaID := uuid.New()

	// Inactive webhook
	wh := newWebhook(webhookID, tenantID, schemaID, "https://example.com/wh", false, nil)
	repo.webhooks[webhookID] = wh

	del := newDelivery(webhookID, tenantID, 0, 5)
	repo.deliveries[del.ID] = del

	worker := NewWebhookWorker(repo, nil)
	worker.deliver(context.Background(), del)

	repo.mu.Lock()
	updated := repo.deliveries[del.ID]
	repo.mu.Unlock()

	if updated.Status != WebhookDeliveryStatusDead {
		t.Errorf("expected dead for inactive webhook, got %s", updated.Status)
	}
}

// ============================================================================
// TestWorker_ContextCancel_ExitsCleanly
// ============================================================================

func TestWorker_ContextCancel_ExitsCleanly(t *testing.T) {
	repo := newStubRepo()
	worker := NewWebhookWorker(repo, nil)
	worker.pollInterval = 10 * time.Second // long poll so we don't tick before cancel

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
	}()

	// Cancel after a short delay
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() returned non-nil error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not exit after context cancel within 2s")
	}
}

// ============================================================================
// TestWorker_ProcessBatch_RespectsBatchSize
// ============================================================================

func TestWorker_ProcessBatch_RespectsBatchSize(t *testing.T) {
	// Server that responds 200 to all requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := newStubRepo()
	tenantID := uuid.New()
	webhookID := uuid.New()
	schemaID := uuid.New()

	wh := newWebhook(webhookID, tenantID, schemaID, server.URL, true, nil)
	repo.webhooks[webhookID] = wh

	// Add 15 pending deliveries
	for i := 0; i < 15; i++ {
		del := newDelivery(webhookID, tenantID, 0, 5)
		repo.mu.Lock()
		repo.deliveries[del.ID] = del
		repo.mu.Unlock()
	}

	worker := newWorkerWithHTTPClient(repo, server.Client())
	worker.batchSize = 10

	// Run one batch
	if err := worker.processBatch(context.Background()); err != nil {
		t.Fatalf("processBatch: %v", err)
	}

	// Count delivered (processed) vs still pending
	repo.mu.Lock()
	delivered := 0
	pending := 0
	for _, d := range repo.deliveries {
		if d.Status == WebhookDeliveryStatusDelivered {
			delivered++
		} else if d.Status == WebhookDeliveryStatusPending {
			pending++
		}
	}
	repo.mu.Unlock()

	if delivered != 10 {
		t.Errorf("expected 10 delivered, got %d", delivered)
	}
	if pending != 5 {
		t.Errorf("expected 5 still pending, got %d", pending)
	}
}

// ============================================================================
// TestWorker_EmptySecret_NoSignatureHeader (edge case: empty string secret)
// ============================================================================

func TestWorker_EmptySecret_NoSignatureHeader(t *testing.T) {
	var receivedSig string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Cosmi-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := newStubRepo()
	tenantID := uuid.New()
	webhookID := uuid.New()
	schemaID := uuid.New()

	emptySecret := ""
	wh := newWebhook(webhookID, tenantID, schemaID, server.URL, true, &emptySecret)
	repo.webhooks[webhookID] = wh

	del := newDelivery(webhookID, tenantID, 0, 5)
	repo.deliveries[del.ID] = del

	worker := newWorkerWithHTTPClient(repo, server.Client())
	worker.deliver(context.Background(), del)

	if receivedSig != "" {
		t.Errorf("expected no X-Cosmi-Signature for empty secret, got %q", receivedSig)
	}
}

// ============================================================================
// TestWorker_EventHeader
// ============================================================================

func TestWorker_EventHeader(t *testing.T) {
	var receivedEvent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedEvent = r.Header.Get("X-Cosmi-Event")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := newStubRepo()
	tenantID := uuid.New()
	webhookID := uuid.New()
	schemaID := uuid.New()

	wh := newWebhook(webhookID, tenantID, schemaID, server.URL, true, nil)
	repo.webhooks[webhookID] = wh

	del := newDelivery(webhookID, tenantID, 0, 5)
	repo.deliveries[del.ID] = del

	worker := newWorkerWithHTTPClient(repo, server.Client())
	worker.deliver(context.Background(), del)

	if !strings.HasPrefix(receivedEvent, "submission") {
		t.Errorf("expected X-Cosmi-Event to start with 'submission', got %q", receivedEvent)
	}
}
