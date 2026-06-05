package datev

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/circuitbreaker"
)

const (
	datevMaxRetries     = 3
	datevInitialBackoff = 1 * time.Second
)

// ErrDatevCircuitOpen is returned when the Uploader's circuit breaker is open
// and the request is shed without reaching DATEV. Callers can use
// errors.Is(err, ErrDatevCircuitOpen) or errors.Is(err,
// circuitbreaker.ErrCircuitOpen) — both work.
var ErrDatevCircuitOpen = circuitbreaker.ErrCircuitOpen

// Uploader sends documents to the DATEV REST API.
type Uploader struct {
	oauthManager *OAuthManager
	httpClient   *http.Client
	baseURL      string
	breaker      *circuitbreaker.CircuitBreaker
}

// NewUploader creates an Uploader with a shared circuit breaker.
//
// The circuit breaker trips after 5 consecutive post-retry failures and stays
// open for 30 s before allowing a probe through. Retry policy: up to 3
// attempts, 1 s / 2 s / 4 s backoff — applied only to 5xx and network errors.
// 4xx responses are not retried (they indicate a client-side problem that
// repeated attempts cannot fix).
func NewUploader(oauthManager *OAuthManager, baseURL string) *Uploader {
	return &Uploader{
		oauthManager: oauthManager,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
		baseURL:      baseURL,
		breaker:      circuitbreaker.New("datev"),
	}
}

// UploadBuchungsstapel uploads a DATEV CSV Buchungsstapel to the API.
//
// Transient 5xx and network errors are retried up to datevMaxRetries times
// with exponential back-off. The overall outcome (after retries) is counted by
// the embedded circuit breaker.
func (u *Uploader) UploadBuchungsstapel(ctx context.Context, tenantID uuid.UUID, clientNumber string, csvData []byte) error {
	uploadURL := fmt.Sprintf("%s/platform/v2/clients/%s/accounting-documents", u.baseURL, clientNumber)

	err := u.breaker.Execute(func() error {
		return u.doWithRetry(ctx, tenantID, http.MethodPut, uploadURL, "text/csv; charset=utf-8", csvData, func(status int, body []byte) {
			slog.Error("datev upload failed",
				"status", status,
				"body", string(body),
				"tenant_id", tenantID,
				"client_number", clientNumber,
			)
		})
	})

	if err != nil {
		if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
			slog.Warn("datev upload shed by circuit breaker",
				"tenant_id", tenantID,
				"client_number", clientNumber,
			)
		}
		return err
	}

	slog.Info("datev buchungsstapel uploaded",
		"tenant_id", tenantID,
		"client_number", clientNumber,
		"size_bytes", len(csvData),
	)
	return nil
}

// ListClients retrieves the list of DATEV clients (Mandanten).
//
// The request is protected by the same circuit breaker and retried on transient
// errors.
func (u *Uploader) ListClients(ctx context.Context, tenantID uuid.UUID) ([]byte, error) {
	clientsURL := fmt.Sprintf("%s/platform/v2/clients", u.baseURL)

	var result []byte

	err := u.breaker.Execute(func() error {
		token, err := u.oauthManager.GetAccessToken(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("datev list clients: get token: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientsURL, nil)
		if err != nil {
			return fmt.Errorf("datev list clients: build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := u.httpClient.Do(req)
		if err != nil {
			// Network error — counts as a retryable failure for the breaker.
			return fmt.Errorf("datev list clients: request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("datev list clients: read response: %w", err)
		}

		if resp.StatusCode >= 500 {
			// 5xx — retryable; breaker will count this as a failure.
			return fmt.Errorf("datev list clients: API returned status %d", resp.StatusCode)
		}

		if resp.StatusCode >= 400 {
			// 4xx — not retried; return directly (breaker still counts it as a
			// failure because the fn returns an error).
			return fmt.Errorf("datev list clients: API returned status %d", resp.StatusCode)
		}

		result = body
		return nil
	})

	if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		slog.Warn("datev list clients shed by circuit breaker", "tenant_id", tenantID)
	}
	return result, err
}

// doWithRetry executes an authenticated HTTP request with retry for transient
// errors (5xx, network). 4xx responses are returned immediately without retry.
//
// logFailure is called with the status code and body on 4xx/5xx responses so
// callers can add context-specific log fields.
func (u *Uploader) doWithRetry(
	ctx context.Context,
	tenantID uuid.UUID,
	method, url, contentType string,
	bodyData []byte,
	logFailure func(status int, body []byte),
) error {
	var lastErr error

	for attempt := 0; attempt <= datevMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := datevInitialBackoff * time.Duration(1<<(attempt-1)) // 1s, 2s, 4s
			slog.Info("datev upload retry",
				"attempt", attempt,
				"backoff_ms", backoff.Milliseconds(),
				"url", url,
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		token, err := u.oauthManager.GetAccessToken(ctx, tenantID)
		if err != nil {
			// Token errors are not retried (likely a persistent auth problem).
			return fmt.Errorf("datev upload: get token: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyData))
		if err != nil {
			return fmt.Errorf("datev upload: build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Accept", "application/json")

		resp, err := u.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("datev upload: request failed: %w", err)
			continue // network error → retry
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("datev upload: read response: %w", err)
			continue
		}

		slog.Info("datev api call",
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"tenant_id", tenantID,
		)

		if resp.StatusCode >= 500 {
			if logFailure != nil {
				logFailure(resp.StatusCode, respBody)
			}
			lastErr = fmt.Errorf("datev upload: API returned status %d: %s", resp.StatusCode, string(respBody))
			continue // 5xx → retry
		}

		if resp.StatusCode >= 400 {
			if logFailure != nil {
				logFailure(resp.StatusCode, respBody)
			}
			// 4xx: client-side error — do NOT retry, return immediately.
			return fmt.Errorf("datev upload: API returned status %d: %s", resp.StatusCode, string(respBody))
		}

		// 2xx success.
		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("datev upload: request failed after %d retries", datevMaxRetries)
}
