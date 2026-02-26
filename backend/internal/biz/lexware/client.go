package lexware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	maxRetries      = 3
	initialBackoff  = 1 * time.Second
	backoffMultiple = 2
)

type Client struct {
	httpClient    *http.Client
	apiKeyManager *APIKeyManager
	rateLimiter   *RateLimiter
	baseURL       string
}

func NewClient(config ClientConfig, vault VaultService) *Client {
	return &Client{
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		apiKeyManager: NewAPIKeyManager(vault),
		rateLimiter:   NewRateLimiter(),
		baseURL:       strings.TrimRight(config.BaseURL, "/"),
	}
}

func (c *Client) APIKeyManager() *APIKeyManager {
	return c.apiKeyManager
}

func (c *Client) do(ctx context.Context, tenantID uuid.UUID, method, path string, body, result any) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := initialBackoff * time.Duration(1<<(attempt-1))
			slog.Info("lexware request retry",
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

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("lexware: rate limiter wait cancelled: %w", err)
		}

		apiKey, err := c.apiKeyManager.GetAPIKey(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("lexware: failed to get API key: %w", err)
		}

		reqURL := c.baseURL + "/" + strings.TrimLeft(path, "/")
		var bodyReader io.Reader
		if body != nil {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("lexware: failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return fmt.Errorf("lexware: failed to build request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("lexware: request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("lexware: failed to read response: %w", err)
			continue
		}

		slog.Info("lexware api call",
			"method", method,
			"path", path,
			"status", resp.StatusCode,
			"tenant_id", tenantID,
		)

		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			lastErr = ErrLexwareRateLimited
			continue

		case resp.StatusCode == http.StatusUnauthorized:
			return ErrLexwareUnauthorized

		case resp.StatusCode == http.StatusNotFound:
			return ErrLexwareNotFound

		case resp.StatusCode == http.StatusConflict:
			return ErrLexwareVersionConflict

		case resp.StatusCode >= 500:
			lastErr = fmt.Errorf("%w: status %d", ErrLexwareServerError, resp.StatusCode)
			continue

		case resp.StatusCode >= 400:
			var apiErr LexwareErrorResponse
			if jsonErr := json.Unmarshal(respBody, &apiErr); jsonErr == nil && apiErr.Message != "" {
				return fmt.Errorf("lexware: API error %d: %s", resp.StatusCode, apiErr.Message)
			}
			return fmt.Errorf("lexware: request failed with status %d: %s", resp.StatusCode, string(respBody))

		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			if result != nil && len(respBody) > 0 {
				if err := json.Unmarshal(respBody, result); err != nil {
					return fmt.Errorf("lexware: failed to decode response: %w", err)
				}
			}
			return nil
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("lexware: request failed after %d retries", maxRetries)
}

// doList executes a paginated list request using Lexware page/size pagination.
func (c *Client) doList(ctx context.Context, tenantID uuid.UUID, path string, params url.Values, result any) error {
	if params == nil {
		params = url.Values{}
	}
	if params.Get("size") == "" {
		params.Set("size", "250")
	}
	if params.Get("page") == "" {
		params.Set("page", "0")
	}

	fullPath := path
	if encoded := params.Encode(); encoded != "" {
		fullPath = path + "?" + encoded
	}

	return c.do(ctx, tenantID, http.MethodGet, fullPath, nil, result)
}
