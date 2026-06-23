// Package llm provides a minimal, provider-agnostic client for chat-completion
// LLMs that speak the OpenAI /v1/chat/completions wire format. The default
// target is a self-hosted Ollama instance (EU data sovereignty); OpenAI and any
// OpenAI-compatible gateway work unchanged by pointing BaseURL/Model/APIKey at
// them. The client is intentionally tiny — it exposes only what the meeting
// AI-summary feature (Wave 7C) needs.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	maxRetries     = 2
	initialBackoff = 500 * time.Millisecond
	defaultTimeout = 120 * time.Second
)

// summarySystemPrompt instructs the model to produce a concise, structured
// summary in the language of the notes. Kept in German since the product locale
// is de-DE; the "antworte in der Sprache der Notizen" clause keeps it correct
// for non-German notes too.
const summarySystemPrompt = "Du bist ein Assistent, der Meeting-Notizen knapp und strukturiert zusammenfasst. " +
	"Fasse die folgenden Notizen in wenigen Sätzen zusammen: Kernpunkte, getroffene Entscheidungen und offene Aufgaben. " +
	"Antworte in der Sprache der Notizen, ohne einleitende Floskeln."

// Config configures a Client. Only BaseURL is required for the client to be
// usable; callers gate construction on a non-empty BaseURL (feature flag).
type Config struct {
	Provider       string
	BaseURL        string
	Model          string
	APIKey         string
	TimeoutSeconds int
}

// Client is an OpenAI-compatible chat-completion client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	model      string
	apiKey     string
}

// NewClient builds a Client from cfg. The caller is responsible for only
// constructing one when cfg.BaseURL is set, so the meeting service can keep a
// nil interface (feature off) rather than a typed-nil that would panic.
func NewClient(cfg Config) *Client {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		model:      cfg.Model,
		apiKey:     cfg.APIKey,
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Summarize sends the given notes to the model and returns a concise summary.
// Returns ErrEmptyInput for blank input and ErrNoCompletion when the model
// yields no usable text.
func (c *Client) Summarize(ctx context.Context, notes string) (string, error) {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return "", ErrEmptyInput
	}

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: summarySystemPrompt},
			{Role: "user", Content: notes},
		},
		Stream: false,
	}

	var resp chatResponse
	if err := c.do(ctx, "/v1/chat/completions", reqBody, &resp); err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", ErrNoCompletion
	}
	summary := strings.TrimSpace(resp.Choices[0].Message.Content)
	if summary == "" {
		return "", ErrNoCompletion
	}
	return summary, nil
}

// do POSTs body to baseURL+path with bounded retries on transient (5xx / 429)
// failures and exponential backoff. 4xx responses fail immediately.
func (c *Client) do(ctx context.Context, path string, body, result any) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("llm: marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := initialBackoff * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(bodyBytes))
		if reqErr != nil {
			return fmt.Errorf("llm: build request: %w", reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, doErr := c.httpClient.Do(req)
		if doErr != nil {
			lastErr = fmt.Errorf("llm: request failed: %w", doErr)
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("llm: read response: %w", readErr)
			continue
		}

		slog.Info("llm api call", "path", path, "status", resp.StatusCode, "model", c.model)

		switch {
		case resp.StatusCode == http.StatusTooManyRequests, resp.StatusCode >= 500:
			lastErr = fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
			continue
		case resp.StatusCode >= 400:
			return fmt.Errorf("llm: request failed with status %d: %s", resp.StatusCode, string(respBody))
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			if result != nil && len(respBody) > 0 {
				if err := json.Unmarshal(respBody, result); err != nil {
					return fmt.Errorf("llm: decode response: %w", err)
				}
			}
			return nil
		default:
			return fmt.Errorf("llm: unexpected status %d", resp.StatusCode)
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("llm: request failed after %d retries", maxRetries)
}
