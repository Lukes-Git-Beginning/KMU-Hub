package lexware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// hmacSignaturePrefix is the prefix used in the X-Signature header.
const hmacSignaturePrefix = "hmac-sha256="

type WebhookSubscription struct {
	ID          string `json:"id,omitempty"`
	EventType   string `json:"eventType"`
	CallbackURL string `json:"callbackUrl"`
}

type WebhookHandler struct {
	client        *Client
	repo          Repository
	emitter       EventEmitter
	contactSyncer *ContactSyncer
}

// ValidateHMAC checks whether the given signature (value of the X-Signature header,
// including the "hmac-sha256=" prefix) matches an HMAC-SHA256 digest of body using
// secret as the key. Comparison is done in constant time to prevent timing attacks.
//
// Returns false when the signature is malformed, has the wrong prefix, or does not match.
func ValidateHMAC(body []byte, signature, secret string) bool {
	sig := strings.TrimPrefix(signature, hmacSignaturePrefix)
	if sig == signature {
		// Prefix was not present — reject.
		return false
	}
	expectedBytes, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), expectedBytes)
}

// ComputeHMAC computes an HMAC-SHA256 signature for body using secret.
// Returned string includes the "hmac-sha256=" prefix and is suitable for use as
// the value of the X-Signature header.
// This helper is used in tests and by the webhook registration flow.
func ComputeHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmacSignaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

func NewWebhookHandler(client *Client, repo Repository, emitter EventEmitter, contactSyncer *ContactSyncer) *WebhookHandler {
	return &WebhookHandler{
		client:        client,
		repo:          repo,
		emitter:       emitter,
		contactSyncer: contactSyncer,
	}
}

// SetEventEmitter swaps the emitter after construction, so Service.SetEventEmitter
// reaches the webhook path too.
func (wh *WebhookHandler) SetEventEmitter(emitter EventEmitter) {
	wh.emitter = emitter
}

// RegisterWebhooks registers webhook subscriptions for all relevant events.
// System operation invoked from OAuth-setup or sync-init flows; wrap in sysctx
// so the UpsertWebhookSubscription INSERTs into lexware_webhook_subscriptions
// (RLS-enabled in mig 122) pass WITH CHECK regardless of caller context.
func (wh *WebhookHandler) RegisterWebhooks(ctx context.Context, configID, tenantID uuid.UUID, callbackURL string) error {
	ctx = sysctx.With(ctx)

	eventTypes := []string{
		"contact.created",
		"contact.changed",
		"invoice.status.changed",
		"quotation.status.changed",
	}

	for _, eventType := range eventTypes {
		sub := &WebhookSubscription{
			EventType:   eventType,
			CallbackURL: callbackURL,
		}

		var result WebhookSubscription
		if err := wh.client.do(ctx, tenantID, http.MethodPost, "v1/event-subscriptions", sub, &result); err != nil {
			slog.Error("lexware: failed to register webhook",
				"event_type", eventType,
				"error", err,
			)
			continue
		}

		if err := wh.repo.UpsertWebhookSubscription(ctx, tenantID, configID, result.ID, eventType, callbackURL); err != nil {
			slog.Error("lexware: failed to persist webhook subscription",
				"event_type", eventType,
				"error", err,
			)
		}

		slog.Info("lexware webhook registered",
			"event_type", eventType,
			"subscription_id", result.ID,
		)
	}

	return nil
}

// UnregisterWebhooks removes all webhook subscriptions for a config.
// System operation; wrap in sysctx so the DeleteWebhookSubscription writes
// pass RLS regardless of caller context.
func (wh *WebhookHandler) UnregisterWebhooks(ctx context.Context, configID, tenantID uuid.UUID) error {
	ctx = sysctx.With(ctx)

	subs, err := wh.repo.ListWebhookSubscriptions(ctx, configID)
	if err != nil {
		return fmt.Errorf("lexware: list webhook subscriptions: %w", err)
	}

	for _, sub := range subs {
		if err := wh.client.do(ctx, tenantID, http.MethodDelete, fmt.Sprintf("v1/event-subscriptions/%s", sub.SubscriptionID), nil, nil); err != nil {
			slog.Error("lexware: failed to unregister webhook",
				"subscription_id", sub.SubscriptionID,
				"error", err,
			)
		}

		if err := wh.repo.DeleteWebhookSubscription(ctx, sub.ID); err != nil {
			slog.Error("lexware: failed to delete webhook subscription record",
				"id", sub.ID,
				"error", err,
			)
		}
	}

	return nil
}

// HandleEvent processes an incoming webhook event.
// Pre-JWT path (HMAC-validated webhook, no user JWT): wrap in sysctx so any
// downstream emitter writes that touch RLS-enabled tables pass WITH CHECK.
//
// lean: today wh.emitter is always noopEmitter{} in production -- NewService wires
// it as the default and cmd/biz/main.go never calls SetEventEmitter on the Lexware
// service -- so the Emit call below and the sysctx wrap it justifies have no live
// effect yet. No automation trigger in the registry listens for "lexware.*" either.
// Upgrade note lives with the follow-up decision in BACKLOG-NEXT.yml
// (wire-biz-event-emitters-for-finance-triggers).
//
// Contact events pull the changed record and apply it to the CRM immediately —
// that is the whole point of subscribing to them, and it keeps the CRM current
// between the scheduler's polling intervals.
//
// lean: document status events (invoice.status.changed, quotation.status.changed)
// are acknowledged and logged but not applied — unlike Bexio, the Lexware
// service has no InvoiceStatusUpdater wired, so there is nothing to write the
// new status to. Wire one and dispatch here as soon as paid-status feedback
// from Lexware is required (Bexio's PaymentPoller is the template).
func (wh *WebhookHandler) HandleEvent(ctx context.Context, configID, tenantID uuid.UUID, event LexwareWebhookEvent) error {
	ctx = sysctx.With(ctx)

	slog.Info("lexware webhook event received",
		"event_type", event.EventType,
		"resource_id", event.ResourceID,
		"tenant_id", tenantID,
	)

	_ = wh.emitter.Emit(ctx, EventPayload{
		Type:     EventWebhookReceived,
		TenantID: tenantID.String(),
		Data:     map[string]any{"event_type": event.EventType, "resource_id": event.ResourceID},
	})

	switch event.EventType {
	case "contact.created", "contact.changed":
		if wh.contactSyncer == nil {
			slog.Warn("lexware webhook: contact syncer not wired, event dropped",
				"event_type", event.EventType, "resource_id", event.ResourceID)
			return nil
		}
		result, err := wh.contactSyncer.SyncContactByLexwareID(ctx, configID, tenantID, event.ResourceID)
		if err != nil {
			return fmt.Errorf("lexware webhook %s: %w", event.EventType, err)
		}
		slog.Info("lexware webhook: contact applied",
			"event_type", event.EventType,
			"resource_id", event.ResourceID,
			"created", result.ItemsCreated,
			"updated", result.ItemsUpdated,
		)
		return nil

	case "invoice.status.changed", "quotation.status.changed":
		slog.Info("lexware webhook: document status event acknowledged but not applied (no status updater wired)",
			"event_type", event.EventType, "resource_id", event.ResourceID)
		return nil

	default:
		slog.Info("lexware webhook: unhandled event type", "event_type", event.EventType)
		return nil
	}
}
