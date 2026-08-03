package workflow

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

const (
	// webhookReceivedTriggerType is the TriggerDefinition.Type registered in
	// trigger.TriggerRegistry for inbound webhooks.
	webhookReceivedTriggerType = "webhook.received"

	// webhookSecretBytes is the size of a generated webhook secret (256 bit).
	webhookSecretBytes = 32

	// maxWebhookBodyBytes caps the payload accepted by TriggerWebhook.
	// Defense-in-depth: the gateway handler already enforces this via
	// http.MaxBytesReader before the raw bytes ever reach the service.
	maxWebhookBodyBytes = 256 * 1024

	// webhookSignaturePrefix matches the convention already used for outbound
	// signatures in internal/formulare/worker.go (X-Cosmi-Signature: sha256=...).
	webhookSignaturePrefix = "sha256="

	// webhookExecutionTimeout bounds the background execution triggered by an
	// inbound webhook, mirroring trigger.TimeTriggerPoller's execution timeout.
	webhookExecutionTimeout = 30 * time.Second
)

// Executor runs a single automation for a given event. Defined locally
// (rather than imported from the engine package) to avoid an import cycle:
// engine depends on workflow for repository types, so workflow cannot import
// engine back. *engine.WorkflowEngine satisfies this interface as-is (same
// pattern as trigger.EngineExecutor).
type Executor interface {
	Execute(ctx context.Context, auto models.Automation, evt models.EventPayload) error
}

// WebhookTriggerResult communicates whether a webhook delivery produced a
// fresh execution or was recognised as a repeat of an already-processed one.
type WebhookTriggerResult struct {
	Duplicate bool
}

// ensureWebhookSecret makes sure an automation of trigger type
// "webhook.received" carries a secret in its trigger_config. Called from
// Create/Update so a webhook automation can never exist without one --
// otherwise TriggerWebhook would have nothing to verify a signature against.
func ensureWebhookSecret(auto *models.Automation) error {
	if auto.TriggerType != webhookReceivedTriggerType {
		return nil
	}

	cfg := map[string]any{}
	if len(auto.TriggerConfig) > 0 {
		if err := json.Unmarshal(auto.TriggerConfig, &cfg); err != nil {
			return fmt.Errorf("%w: invalid trigger_config: %v", ErrInvalidCondition, err)
		}
	}

	if secret, ok := cfg["secret"].(string); !ok || secret == "" {
		raw := make([]byte, webhookSecretBytes)
		if _, err := rand.Read(raw); err != nil {
			return fmt.Errorf("generate webhook secret: %w", err)
		}
		cfg["secret"] = hex.EncodeToString(raw)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal trigger_config: %w", err)
	}
	auto.TriggerConfig = data
	return nil
}

// webhookSecret extracts the configured secret from an automation's
// trigger_config. Returns an error if missing -- this should not happen for
// an automation created/updated through the service (ensureWebhookSecret
// guarantees it), but a defensive check here means a row written by any
// other path fails safely instead of comparing against an empty secret.
func webhookSecret(auto *models.Automation) (string, error) {
	var cfg struct {
		Secret string `json:"secret"`
	}
	if len(auto.TriggerConfig) > 0 {
		if err := json.Unmarshal(auto.TriggerConfig, &cfg); err != nil {
			return "", fmt.Errorf("invalid trigger_config: %w", err)
		}
	}
	if cfg.Secret == "" {
		return "", fmt.Errorf("webhook automation has no configured secret")
	}
	return cfg.Secret, nil
}

// webhookSignatureValid reports whether header (the value of the
// X-Cosmi-Signature request header, "sha256=<hex>") is a valid HMAC-SHA256
// signature of rawBody under secret. Comparison is constant-time.
func webhookSignatureValid(rawBody []byte, header, secret string) bool {
	sig := strings.TrimPrefix(header, webhookSignaturePrefix)
	if sig == header || sig == "" {
		// Prefix absent or empty signature -- reject.
		return false
	}
	expected, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	return hmac.Equal(mac.Sum(nil), expected)
}

// jsonPayload returns rawBody unchanged if it is valid JSON, otherwise wraps
// it as a JSON string. models.EventPayload.Payload is a json.RawMessage that
// gets embedded verbatim by encoding/json when execution logs are persisted;
// an invalid-JSON webhook body reaching that path as-is would fail the whole
// marshal, not just drop the field.
func jsonPayload(rawBody []byte) json.RawMessage {
	if json.Valid(rawBody) {
		return json.RawMessage(rawBody)
	}
	wrapped, err := json.Marshal(string(rawBody))
	if err != nil {
		return json.RawMessage(`null`)
	}
	return wrapped
}

// TriggerWebhook validates and executes an inbound webhook delivery for the
// automation identified by automationID.
//
// The tenant is not known up front -- it is read off the automation row
// itself, which is why the lookup runs under sysctx (RLS bypass) and why the
// trigger-type/active checks below are mandatory rather than defensive: they
// are the only thing standing between this endpoint and cross-tenant probing
// of arbitrary automation IDs.
//
// Idempotency reuses the shared internal/idempotency store (same one behind
// the authenticated Idempotency-Key middleware) rather than a bespoke
// dedup table. Precedence: an explicit idempotencyKey (delivery/event ID
// header from the sender) if present, otherwise the SHA-256 of the body --
// an exact resend of the same delivery (the common webhook retry pattern)
// dedupes even when the sender sets no such header.
//
// lean: internal/idempotency.Reserve does not fully distinguish a fresh key
// from a concurrent in-flight one under the real Postgres backend (both
// currently return (nil, nil) if completed_at hasn't been set yet) -- a very
// narrow window where two near-simultaneous deliveries of the same key could
// both execute. Pre-existing gap in shared infra used across the codebase,
// not specific to this trigger; out of scope here. Upgrade when
// internal/idempotency's Reserve is hardened for that race.
func (s *Service) TriggerWebhook(ctx context.Context, automationID uuid.UUID, rawBody []byte, signatureHeader, idempotencyKey string) (*WebhookTriggerResult, error) {
	if len(rawBody) > maxWebhookBodyBytes {
		return nil, ErrWebhookPayloadTooLarge
	}

	sysCtx := sysctx.With(ctx)

	auto, err := s.repo.GetByIDUnscoped(sysCtx, automationID)
	if err != nil {
		if errors.Is(err, ErrAutomationNotFound) {
			return nil, ErrWebhookNotFound
		}
		return nil, err
	}
	if auto.TriggerType != webhookReceivedTriggerType {
		return nil, ErrWebhookNotFound
	}
	if !auto.IsActive {
		return nil, ErrAutomationInactive
	}

	secret, err := webhookSecret(auto)
	if err != nil {
		slog.Error("webhook automation has no usable secret", "automation_id", auto.ID, "error", err)
		return nil, ErrWebhookNotFound
	}
	if !webhookSignatureValid(rawBody, signatureHeader, secret) {
		return nil, ErrWebhookSignatureInvalid
	}

	sum := sha256.Sum256(rawBody)
	bodyHash := hex.EncodeToString(sum[:])
	dedupeKey := idempotencyKey
	if dedupeKey == "" {
		dedupeKey = bodyHash
	}

	rec, err := s.idempotencyRepo.Reserve(sysCtx, dedupeKey, auto.TenantID, auto.OwnerID,
		http.MethodPost, "automation-webhook:"+auto.ID.String(), bodyHash)
	switch {
	case errors.Is(err, idempotency.ErrConflict):
		return nil, ErrWebhookIdempotencyConflict
	case errors.Is(err, idempotency.ErrInFlight):
		return &WebhookTriggerResult{Duplicate: true}, nil
	case err != nil:
		return nil, err
	}
	if rec != nil {
		// Already completed -- this exact delivery already ran.
		return &WebhookTriggerResult{Duplicate: true}, nil
	}

	evt := models.EventPayload{
		Type:       auto.TriggerType,
		TenantID:   auto.TenantID,
		ModuleID:   event.ModuleIntegration,
		ResourceID: auto.ID.String(),
		Payload:    jsonPayload(rawBody),
		Timestamp:  time.Now(),
	}

	// Execute asynchronously (mirrors trigger.TimeTriggerPoller): the sender
	// gets a fast ack, the action chain runs in the background under its own
	// timeout so a slow/hanging action can't hold the HTTP connection open.
	execCtx, cancel := context.WithTimeout(context.WithoutCancel(sysCtx), webhookExecutionTimeout)
	go func() {
		defer cancel()
		if execErr := s.executor.Execute(execCtx, *auto, evt); execErr != nil {
			slog.Error("webhook-triggered automation execution failed",
				"automation_id", auto.ID,
				"error", execErr,
			)
		}
	}()

	if compErr := s.idempotencyRepo.Complete(sysCtx, auto.TenantID, dedupeKey, http.StatusAccepted, []byte(`{"accepted":true}`)); compErr != nil {
		slog.Warn("failed to complete webhook idempotency record", "automation_id", auto.ID, "error", compErr)
	}

	return &WebhookTriggerResult{Duplicate: false}, nil
}
