package formulare

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/security/safehttp"
)

// backoffDurations defines exponential retry delays indexed by attempt number (1-based).
// Attempt 1 → 30s, 2 → 2min, 3 → 10min, 4 → 30min, 5+ → 2h
var backoffDurations = []time.Duration{
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
}

func backoffFor(attemptCount int) time.Duration {
	idx := min(max(attemptCount-1, 0), len(backoffDurations)-1)
	return backoffDurations[idx]
}

// ============================================================================
// WebhookWorker
// ============================================================================

// WebhookWorker polls for pending webhook deliveries and dispatches them with
// HMAC signing and exponential back-off. It is stateless modulo repo calls
// and is horizontally scalable thanks to FOR UPDATE SKIP LOCKED in the repo.
type WebhookWorker struct {
	repo         Repository
	httpClient   *http.Client
	logger       *slog.Logger
	pollInterval time.Duration
	batchSize    int
}

// NewWebhookWorker creates a new WebhookWorker with default configuration.
func NewWebhookWorker(repo Repository, logger *slog.Logger) *WebhookWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookWorker{
		repo: repo,
		// The URL comes from the tenant, so the same guard the automation
		// http.request action uses applies here: without it a form webhook
		// pointed at 169.254.169.254 or at an internal service turns this
		// worker into a request forwarder inside our own network.
		httpClient:   safehttp.New(safehttp.WithTimeout(10 * time.Second)),
		logger:       logger,
		pollInterval: 5 * time.Second,
		batchSize:    10,
	}
}

// Run blocks until ctx is done, polling and dispatching deliveries on each tick.
func (w *WebhookWorker) Run(ctx context.Context) error {
	ctx = database.WithSystemContext(ctx)
	w.logger.Info("formulare webhook worker starting",
		"poll_interval", w.pollInterval,
		"batch_size", w.batchSize,
	)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("formulare webhook worker stopped")
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				w.logger.Error("formulare webhook worker batch error", "error", err)
				// Continue — don't abort on transient errors.
			}
		}
	}
}

// processBatch claims up to batchSize pending deliveries and dispatches each one.
func (w *WebhookWorker) processBatch(ctx context.Context) error {
	deliveries, err := w.repo.ClaimPendingDeliveries(ctx, w.batchSize)
	if err != nil {
		return fmt.Errorf("claim pending deliveries: %w", err)
	}

	for _, d := range deliveries {
		w.deliver(ctx, d)
	}
	return nil
}

// deliver performs a single HTTP POST with HMAC signature and updates delivery status.
func (w *WebhookWorker) deliver(ctx context.Context, delivery *WebhookDelivery) {
	webhook, err := w.repo.GetWebhook(ctx, delivery.WebhookID, delivery.TenantID)
	if err != nil {
		w.logger.Error("formulare webhook worker: failed to load webhook",
			"webhook_id", delivery.WebhookID,
			"delivery_id", delivery.ID,
			"error", err,
		)
		w.markFailed(ctx, delivery, 0, fmt.Sprintf("load webhook: %v", err))
		return
	}

	// Skip delivery if webhook is inactive — mark dead immediately.
	if !webhook.Active {
		w.logger.Info("formulare webhook worker: webhook inactive, marking dead",
			"webhook_id", webhook.ID,
			"delivery_id", delivery.ID,
		)
		w.markResult(ctx, delivery, WebhookDeliveryStatusDead, delivery.AttemptCount+1, nil, "webhook is inactive", 0)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, nil)
	if err != nil {
		w.markFailed(ctx, delivery, 0, fmt.Sprintf("build request: %v", err))
		return
	}

	req.Body = http.NoBody
	if len(delivery.Payload) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(delivery.Payload))
		req.ContentLength = int64(len(delivery.Payload))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cosmi-Event", "submission.created")

	// Add HMAC signature if webhook has a secret.
	if webhook.Secret != nil && *webhook.Secret != "" {
		mac := hmac.New(sha256.New, []byte(*webhook.Secret))
		mac.Write(delivery.Payload)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Cosmi-Signature", "sha256="+sig)
	}

	resp, err := w.httpClient.Do(req)
	newAttemptCount := delivery.AttemptCount + 1

	if err != nil {
		w.logger.Warn("formulare webhook worker: HTTP request failed",
			"delivery_id", delivery.ID,
			"webhook_url", webhook.URL,
			"attempt", newAttemptCount,
			"error", err,
		)
		w.markFailed(ctx, delivery, 0, err.Error())
		return
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success
		now := time.Now().UTC()
		w.logger.Info("formulare webhook worker: delivery succeeded",
			"delivery_id", delivery.ID,
			"webhook_url", webhook.URL,
			"status_code", resp.StatusCode,
			"attempt", newAttemptCount,
		)
		markErr := w.repo.MarkDeliveryResult(ctx, delivery.ID, WebhookDeliveryStatusDelivered, newAttemptCount, &now, "", resp.StatusCode)
		if markErr != nil {
			w.logger.Error("formulare webhook worker: mark delivered failed", "error", markErr)
		}
		return
	}

	// Non-2xx response
	w.logger.Warn("formulare webhook worker: delivery failed with non-2xx",
		"delivery_id", delivery.ID,
		"webhook_url", webhook.URL,
		"status_code", resp.StatusCode,
		"attempt", newAttemptCount,
	)
	w.markFailed(ctx, delivery, resp.StatusCode, fmt.Sprintf("non-2xx response: %d", resp.StatusCode))
}

// markFailed applies exponential back-off or transitions to dead after max attempts.
func (w *WebhookWorker) markFailed(ctx context.Context, delivery *WebhookDelivery, lastCode int, lastErr string) {
	newAttemptCount := delivery.AttemptCount + 1
	maxAttempts := delivery.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	var finalStatus WebhookDeliveryStatus
	var nextAttemptAt *time.Time

	if newAttemptCount >= maxAttempts {
		finalStatus = WebhookDeliveryStatusDead
	} else {
		finalStatus = WebhookDeliveryStatusFailed
		t := time.Now().UTC().Add(backoffFor(newAttemptCount))
		nextAttemptAt = &t
	}

	w.markResult(ctx, delivery, finalStatus, newAttemptCount, nextAttemptAt, lastErr, lastCode)
}

func (w *WebhookWorker) markResult(ctx context.Context, delivery *WebhookDelivery, status WebhookDeliveryStatus, attemptCount int, nextAttemptAt *time.Time, lastErr string, lastCode int) {
	if err := w.repo.MarkDeliveryResult(ctx, delivery.ID, status, attemptCount, nextAttemptAt, lastErr, lastCode); err != nil {
		if !errors.Is(err, context.Canceled) {
			w.logger.Error("formulare webhook worker: mark result failed",
				"delivery_id", delivery.ID,
				"error", err,
			)
		}
	}
}

