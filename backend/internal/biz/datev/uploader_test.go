package datev

// Covers Uploader against an httptest server standing in for the DATEV REST
// API: retry/no-retry classification (5xx retries, 4xx does not), the
// circuit breaker integration, and ListClients (which — unlike
// UploadBuchungsstapel — is not retried). Token acquisition is short-circuited
// via a pre-populated OAuthManager cache so these tests exercise Uploader's
// own logic, not oauth.go's.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/circuitbreaker"
)

func cachedOAuthManager(tenantID uuid.UUID, token string) *OAuthManager {
	return &OAuthManager{
		cache: map[uuid.UUID]*tokenCacheEntry{
			tenantID: {accessToken: token, expiresAt: time.Now().Add(time.Hour)},
		},
	}
}

func newTestUploader(baseURL string, om *OAuthManager) *Uploader {
	return &Uploader{
		oauthManager: om,
		httpClient:   &http.Client{Timeout: 5 * time.Second},
		baseURL:      baseURL,
		breaker:      circuitbreaker.New("test-datev"),
	}
}

// flakyTransport fails the first N RoundTrips with a network error, then
// delegates to the real transport — used to exercise the network-error retry
// branch without an actually dropped connection.
type flakyTransport struct {
	failCount int
	calls     int
}

func (f *flakyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	if f.calls <= f.failCount {
		return nil, errors.New("simulated network error")
	}
	return http.DefaultTransport.RoundTrip(req)
}

func TestNewUploader_WiresFields(t *testing.T) {
	om := &OAuthManager{}
	u := NewUploader(om, "https://api.datev.de")

	if u.baseURL != "https://api.datev.de" {
		t.Errorf("baseURL = %q", u.baseURL)
	}
	if u.oauthManager != om {
		t.Error("oauthManager not wired")
	}
	if u.breaker == nil {
		t.Error("breaker not initialized")
	}
	if u.httpClient == nil || u.httpClient.Timeout != 60*time.Second {
		t.Errorf("httpClient timeout = %v, want 60s", u.httpClient.Timeout)
	}
}

func TestUploadBuchungsstapel_SucceedsOnFirstAttempt(t *testing.T) {
	tenantID := uuid.New()
	var gotMethod, gotPath, gotAuth, gotContentType string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u := newTestUploader(server.URL, cachedOAuthManager(tenantID, "test-token"))

	if err := u.UploadBuchungsstapel(context.Background(), tenantID, "12345", []byte("csv-content")); err != nil {
		t.Fatalf("UploadBuchungsstapel: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/platform/v2/clients/12345/accounting-documents" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotContentType != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if string(gotBody) != "csv-content" {
		t.Errorf("body = %q", gotBody)
	}
}

func TestUploadBuchungsstapel_DoesNotRetry4xx(t *testing.T) {
	tenantID := uuid.New()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer server.Close()

	u := newTestUploader(server.URL, cachedOAuthManager(tenantID, "test-token"))

	if err := u.UploadBuchungsstapel(context.Background(), tenantID, "12345", []byte("csv")); err == nil {
		t.Fatal("expected an error on 400")
	}
	if calls != 1 {
		t.Errorf("server called %d times, want 1 (4xx must not retry)", calls)
	}
}

func TestUploadBuchungsstapel_Retries5xxThenSucceeds(t *testing.T) {
	tenantID := uuid.New()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u := newTestUploader(server.URL, cachedOAuthManager(tenantID, "test-token"))

	if err := u.UploadBuchungsstapel(context.Background(), tenantID, "12345", []byte("csv")); err != nil {
		t.Fatalf("UploadBuchungsstapel: %v", err)
	}
	if calls != 2 {
		t.Errorf("server called %d times, want 2 (one retry after 503)", calls)
	}
}

func TestUploadBuchungsstapel_RetriesNetworkErrorThenSucceeds(t *testing.T) {
	tenantID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &flakyTransport{failCount: 1}
	u := newTestUploader(server.URL, cachedOAuthManager(tenantID, "test-token"))
	u.httpClient = &http.Client{Timeout: 5 * time.Second, Transport: transport}

	if err := u.UploadBuchungsstapel(context.Background(), tenantID, "12345", []byte("csv")); err != nil {
		t.Fatalf("UploadBuchungsstapel: %v", err)
	}
	if transport.calls != 2 {
		t.Errorf("transport called %d times, want 2 (one retry after a network error)", transport.calls)
	}
}

func TestUploadBuchungsstapel_TokenErrorIsNotRetried(t *testing.T) {
	tenantID := uuid.New()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	om := NewOAuthManager(&vaultStub{getErr: errors.New("vault unavailable")}, "cid", "csecret", "http://unused.invalid")
	u := newTestUploader(server.URL, om)

	if err := u.UploadBuchungsstapel(context.Background(), tenantID, "12345", []byte("csv")); err == nil {
		t.Fatal("expected an error when the token cannot be obtained")
	}
	if calls != 0 {
		t.Errorf("server contacted %d times, want 0 (token errors are not retried)", calls)
	}
}

func TestUploadBuchungsstapel_CircuitBreakerOpensAfterConsecutiveFailures(t *testing.T) {
	tenantID := uuid.New()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest) // 4xx: fails fast, no retry wait.
	}))
	defer server.Close()

	u := newTestUploader(server.URL, cachedOAuthManager(tenantID, "test-token"))
	u.breaker = circuitbreaker.New("test-datev", circuitbreaker.WithFailureThreshold(2))

	for i := range 2 {
		if err := u.UploadBuchungsstapel(context.Background(), tenantID, "12345", []byte("csv")); err == nil {
			t.Fatalf("call %d: expected an error", i)
		}
	}
	if calls != 2 {
		t.Fatalf("server called %d times before the breaker should trip, want 2", calls)
	}

	err := u.UploadBuchungsstapel(context.Background(), tenantID, "12345", []byte("csv"))
	if !errors.Is(err, ErrDatevCircuitOpen) {
		t.Fatalf("err = %v, want ErrDatevCircuitOpen", err)
	}
	if calls != 2 {
		t.Errorf("server called %d times after the breaker tripped, want still 2 (shed, not forwarded)", calls)
	}
}

func TestListClients_Success(t *testing.T) {
	tenantID := uuid.New()
	var gotAuth, gotAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":1}]`))
	}))
	defer server.Close()

	u := newTestUploader(server.URL, cachedOAuthManager(tenantID, "test-token"))

	body, err := u.ListClients(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("ListClients: %v", err)
	}
	if string(body) != `[{"id":1}]` {
		t.Errorf("body = %q", body)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q", gotAccept)
	}
}

func TestListClients_ReturnsErrorWithoutRetryOn5xx(t *testing.T) {
	tenantID := uuid.New()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	u := newTestUploader(server.URL, cachedOAuthManager(tenantID, "test-token"))

	if _, err := u.ListClients(context.Background(), tenantID); err == nil {
		t.Fatal("expected an error on 500")
	}
	if calls != 1 {
		t.Errorf("server called %d times, want 1 (ListClients does not retry)", calls)
	}
}

func TestListClients_ReturnsErrorOn4xx(t *testing.T) {
	tenantID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	u := newTestUploader(server.URL, cachedOAuthManager(tenantID, "test-token"))

	if _, err := u.ListClients(context.Background(), tenantID); err == nil {
		t.Fatal("expected an error on 404")
	}
}

func TestListClients_TokenErrorPropagates(t *testing.T) {
	tenantID := uuid.New()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	om := NewOAuthManager(&vaultStub{getErr: errors.New("vault unavailable")}, "cid", "csecret", "http://unused.invalid")
	u := newTestUploader(server.URL, om)

	if _, err := u.ListClients(context.Background(), tenantID); err == nil {
		t.Fatal("expected an error when the token cannot be obtained")
	}
	if calls != 0 {
		t.Errorf("server contacted %d times, want 0", calls)
	}
}
