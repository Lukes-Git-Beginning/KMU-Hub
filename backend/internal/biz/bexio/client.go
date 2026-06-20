package bexio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/circuitbreaker"
)

const (
	maxRetries     = 3
	initialBackoff = 1 * time.Second
)

// Client is the HTTP client for the Bexio REST API v2.0.
type Client struct {
	httpClient   *http.Client
	tokenManager *TokenManager
	rateLimiter  *RateLimiter
	breaker      *circuitbreaker.CircuitBreaker
	baseURL      string
}

// NewClient creates a new Bexio API client.
//
// The embedded circuit breaker trips after 5 consecutive post-retry failures
// and stays open for 30 s before allowing a probe request through.
func NewClient(config ClientConfig, vault VaultService) *Client {
	tm := NewTokenManager(vault, config.ClientID, config.ClientSecret, config.TokenURL)

	return &Client{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		tokenManager: tm,
		rateLimiter:  NewRateLimiter(),
		breaker:      circuitbreaker.New("bexio"),
		baseURL:      strings.TrimRight(config.BaseURL, "/"),
	}
}

// TokenManager returns the underlying token manager for OAuth flows.
func (c *Client) TokenManager() *TokenManager {
	return c.tokenManager
}

// do executes an authenticated request against the Bexio API with rate
// limiting, automatic token refresh on 401, exponential backoff on 5xx/429,
// and circuit-breaker protection.
//
// The circuit breaker wraps the entire retry loop: it counts one outcome (pass
// or fail) per call to do(), not per individual retry attempt. This means
// transient hiccups that resolve within the retry window are invisible to the
// breaker, which only trips when the downstream is consistently unreachable
// after all retries have been exhausted.
func (c *Client) do(ctx context.Context, tenantID uuid.UUID, method, path string, body, result any) error {
	breakerErr := c.breaker.Execute(func() error {
		return c.doWithRetry(ctx, tenantID, method, path, body, result)
	})

	// Surface ErrCircuitOpen directly so callers can distinguish a shed request
	// from a genuine API error.
	if errors.Is(breakerErr, circuitbreaker.ErrCircuitOpen) {
		slog.Warn("bexio request shed by circuit breaker",
			"method", method,
			"path", path,
			"tenant_id", tenantID,
		)
		return breakerErr
	}
	return breakerErr
}

// doWithRetry contains the actual retry loop. It is called by do() inside the
// circuit breaker's Execute closure.
func (c *Client) doWithRetry(ctx context.Context, tenantID uuid.UUID, method, path string, body, result any) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := initialBackoff * time.Duration(1<<(attempt-1))
			slog.Info("bexio request retry",
				"attempt", attempt,
				"backoff_ms", backoff.Milliseconds(),
				"path", path,
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Wait for rate limiter.
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("bexio: rate limiter wait cancelled: %w", err)
		}

		// Get access token.
		token, err := c.tokenManager.GetAccessToken(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("bexio: failed to get access token: %w", err)
		}

		// Build request.
		reqURL := c.baseURL + "/" + strings.TrimLeft(path, "/")
		var bodyReader io.Reader
		if body != nil {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("bexio: failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return fmt.Errorf("bexio: failed to build request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		} else if method == http.MethodGet {
			// Bexio quirk: GET requests need Content-Length: 0.
			req.Header.Set("Content-Length", "0")
		}

		// Execute.
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("bexio: request failed: %w", err)
			continue
		}

		// Update rate limiter from response headers.
		c.rateLimiter.UpdateFromHeaders(resp.Header)

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("bexio: failed to read response: %w", err)
			continue
		}

		slog.Info("bexio api call",
			"method", method,
			"path", path,
			"status", resp.StatusCode,
			"tenant_id", tenantID,
		)

		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			lastErr = ErrBexioRateLimited
			continue

		case resp.StatusCode == http.StatusUnauthorized:
			// Try refreshing the token once.
			if attempt == 0 {
				if _, refreshErr := c.tokenManager.RefreshAccessToken(ctx, tenantID); refreshErr != nil {
					return ErrBexioUnauthorized
				}
				continue
			}
			return ErrBexioUnauthorized

		case resp.StatusCode == http.StatusNotFound:
			return ErrBexioNotFound

		case resp.StatusCode >= 500:
			lastErr = fmt.Errorf("%w: status %d", ErrBexioServerError, resp.StatusCode)
			continue

		case resp.StatusCode >= 400:
			var apiErr BexioErrorResponse
			if jsonErr := json.Unmarshal(respBody, &apiErr); jsonErr == nil && apiErr.Message != "" {
				return fmt.Errorf("bexio: API error %d: %s", apiErr.ErrorCode, apiErr.Message)
			}
			return fmt.Errorf("bexio: request failed with status %d: %s", resp.StatusCode, string(respBody))

		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			if result != nil && len(respBody) > 0 {
				if err := json.Unmarshal(respBody, result); err != nil {
					return fmt.Errorf("bexio: failed to decode response: %w", err)
				}
			}
			return nil
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("bexio: request failed after %d retries", maxRetries)
}

// BexioCompanyProfile holds the response from GET /2.0/company_profile.
type BexioCompanyProfile struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// GetCompanyProfile fetches the authenticated tenant's company name from Bexio.
func (c *Client) GetCompanyProfile(ctx context.Context, tenantID uuid.UUID) (*BexioCompanyProfile, error) {
	// Bexio returns an array from GET /company_profile.
	var profiles []BexioCompanyProfile
	if err := c.do(ctx, tenantID, http.MethodGet, "company_profile", nil, &profiles); err != nil {
		return nil, fmt.Errorf("bexio: get company profile: %w", err)
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("bexio: company_profile returned empty list")
	}
	return &profiles[0], nil
}

// doList executes a paginated list request against the Bexio API.
func (c *Client) doList(ctx context.Context, tenantID uuid.UUID, path string, params url.Values, result any) error {
	if params == nil {
		params = url.Values{}
	}
	if params.Get("limit") == "" {
		params.Set("limit", "500")
	}
	if params.Get("offset") == "" {
		params.Set("offset", "0")
	}

	fullPath := path
	if encoded := params.Encode(); encoded != "" {
		fullPath = path + "?" + encoded
	}

	return c.do(ctx, tenantID, http.MethodGet, fullPath, nil, result)
}
