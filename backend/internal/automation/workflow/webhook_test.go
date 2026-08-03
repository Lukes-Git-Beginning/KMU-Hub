package workflow

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return webhookSignaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

// waitExecuted blocks until fakeExecutor.Execute has run once (TriggerWebhook
// dispatches it asynchronously) or fails the test after 2s.
func waitExecuted(t *testing.T, executor *fakeExecutor) {
	t.Helper()
	select {
	case <-executor.executed:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for webhook-triggered execution")
	}
}

// ============================================================================
// ensureWebhookSecret / Create+Update integration
// ============================================================================

func TestCreate_WebhookReceived_GeneratesSecret(t *testing.T) {
	var captured *models.Automation
	repo := &mockRepo{
		createFn: func(_ context.Context, a *models.Automation) error {
			captured = a
			return nil
		},
	}
	svc := newTestService(repo, nil, nil)

	auto := &models.Automation{
		Name:        "Incoming Webhook",
		TriggerType: webhookReceivedTriggerType,
		Actions:     mustJSON([]models.ActionConfig{}),
	}

	err := svc.Create(context.Background(), auto)
	require.NoError(t, err)

	var cfg struct {
		Secret string `json:"secret"`
	}
	require.NoError(t, json.Unmarshal(captured.TriggerConfig, &cfg))
	assert.Len(t, cfg.Secret, webhookSecretBytes*2, "secret should be hex-encoded (2 chars per byte)")
}

func TestUpdate_WebhookReceived_PreservesExistingSecret(t *testing.T) {
	var captured *models.Automation
	repo := &mockRepo{
		updateFn: func(_ context.Context, a *models.Automation) error {
			captured = a
			return nil
		},
	}
	svc := newTestService(repo, nil, nil)

	auto := &models.Automation{
		ID:            uuid.New(),
		Name:          "Incoming Webhook",
		TriggerType:   webhookReceivedTriggerType,
		TriggerConfig: mustJSON(map[string]string{"secret": "fixed-secret-value"}),
		Actions:       mustJSON([]models.ActionConfig{}),
	}

	err := svc.Update(context.Background(), auto)
	require.NoError(t, err)

	secret, err := webhookSecret(captured)
	require.NoError(t, err)
	assert.Equal(t, "fixed-secret-value", secret, "an existing secret must not be regenerated")
}

// ============================================================================
// TriggerWebhook
// ============================================================================

func webhookAutomation(tenantID, ownerID uuid.UUID, secret string) *models.Automation {
	return &models.Automation{
		ID:            uuid.New(),
		TenantID:      tenantID,
		OwnerID:       ownerID,
		TriggerType:   webhookReceivedTriggerType,
		TriggerConfig: mustJSON(map[string]string{"secret": secret}),
		Actions:       mustJSON([]models.ActionConfig{}),
		IsActive:      true,
	}
}

func TestTriggerWebhook_Success(t *testing.T) {
	tenantID, ownerID := uuid.New(), uuid.New()
	auto := webhookAutomation(tenantID, ownerID, "s3cr3t")
	body := []byte(`{"hello":"world"}`)

	repo := &mockRepo{
		getByIDUnscopedFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			require.Equal(t, auto.ID, id)
			return auto, nil
		},
	}
	executor := newFakeExecutor()
	svc := newTestServiceWithWebhookDeps(repo, nil, nil, nil, executor)

	result, err := svc.TriggerWebhook(context.Background(), auto.ID, body, sign(body, "s3cr3t"), "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Duplicate)

	waitExecuted(t, executor)
	assert.Equal(t, 1, executor.callCount())
	assert.Equal(t, auto.ID, executor.lastAuto.ID)
	assert.Equal(t, webhookReceivedTriggerType, executor.lastEvt.Type)
	assert.JSONEq(t, `{"hello":"world"}`, string(executor.lastEvt.Payload))
}

func TestTriggerWebhook_WrongSignature(t *testing.T) {
	auto := webhookAutomation(uuid.New(), uuid.New(), "s3cr3t")
	repo := &mockRepo{
		getByIDUnscopedFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return auto, nil
		},
	}
	executor := newFakeExecutor()
	svc := newTestServiceWithWebhookDeps(repo, nil, nil, nil, executor)

	_, err := svc.TriggerWebhook(context.Background(), auto.ID, []byte(`{}`), "sha256=deadbeef", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookSignatureInvalid)
	assert.Equal(t, 0, executor.callCount())
}

func TestTriggerWebhook_MissingSignatureHeader(t *testing.T) {
	auto := webhookAutomation(uuid.New(), uuid.New(), "s3cr3t")
	repo := &mockRepo{
		getByIDUnscopedFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return auto, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	_, err := svc.TriggerWebhook(context.Background(), auto.ID, []byte(`{}`), "", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookSignatureInvalid)
}

func TestTriggerWebhook_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDUnscopedFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return nil, ErrAutomationNotFound
		},
	}
	svc := newTestService(repo, nil, nil)

	_, err := svc.TriggerWebhook(context.Background(), uuid.New(), []byte(`{}`), "sha256=deadbeef", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookNotFound)
}

// TestTriggerWebhook_WrongTriggerType verifies that hitting the webhook URL of
// an automation whose trigger type is NOT "webhook.received" fails with the
// same error as "not found" -- no enumeration signal about which case applied.
func TestTriggerWebhook_WrongTriggerType(t *testing.T) {
	auto := webhookAutomation(uuid.New(), uuid.New(), "s3cr3t")
	auto.TriggerType = "crm.deal.stage_changed"
	repo := &mockRepo{
		getByIDUnscopedFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return auto, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	body := []byte(`{}`)
	_, err := svc.TriggerWebhook(context.Background(), auto.ID, body, sign(body, "s3cr3t"), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookNotFound)
}

func TestTriggerWebhook_Inactive(t *testing.T) {
	auto := webhookAutomation(uuid.New(), uuid.New(), "s3cr3t")
	auto.IsActive = false
	repo := &mockRepo{
		getByIDUnscopedFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return auto, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	body := []byte(`{}`)
	_, err := svc.TriggerWebhook(context.Background(), auto.ID, body, sign(body, "s3cr3t"), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAutomationInactive)
}

func TestTriggerWebhook_PayloadTooLarge(t *testing.T) {
	repo := &mockRepo{
		getByIDUnscopedFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			t.Fatal("repo lookup must not run before the size cap is enforced")
			return nil, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	oversized := make([]byte, maxWebhookBodyBytes+1)
	_, err := svc.TriggerWebhook(context.Background(), uuid.New(), oversized, "sha256=x", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookPayloadTooLarge)
}

func TestTriggerWebhook_DuplicateViaIdempotencyKey(t *testing.T) {
	auto := webhookAutomation(uuid.New(), uuid.New(), "s3cr3t")
	repo := &mockRepo{
		getByIDUnscopedFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return auto, nil
		},
	}
	executor := newFakeExecutor()
	svc := newTestServiceWithWebhookDeps(repo, nil, nil, nil, executor)

	body := []byte(`{"n":1}`)
	sig := sign(body, "s3cr3t")

	result1, err := svc.TriggerWebhook(context.Background(), auto.ID, body, sig, "delivery-42")
	require.NoError(t, err)
	assert.False(t, result1.Duplicate)
	waitExecuted(t, executor)

	result2, err := svc.TriggerWebhook(context.Background(), auto.ID, body, sig, "delivery-42")
	require.NoError(t, err)
	assert.True(t, result2.Duplicate, "resend of the same delivery key must not run a second execution")

	assert.Equal(t, 1, executor.callCount())
}

func TestTriggerWebhook_DuplicateViaBodyHashFallback(t *testing.T) {
	auto := webhookAutomation(uuid.New(), uuid.New(), "s3cr3t")
	repo := &mockRepo{
		getByIDUnscopedFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return auto, nil
		},
	}
	executor := newFakeExecutor()
	svc := newTestServiceWithWebhookDeps(repo, nil, nil, nil, executor)

	body := []byte(`{"n":1}`)
	sig := sign(body, "s3cr3t")

	// No Idempotency-Key at all -- an exact resend must still dedupe via the body hash.
	_, err := svc.TriggerWebhook(context.Background(), auto.ID, body, sig, "")
	require.NoError(t, err)
	waitExecuted(t, executor)

	result2, err := svc.TriggerWebhook(context.Background(), auto.ID, body, sig, "")
	require.NoError(t, err)
	assert.True(t, result2.Duplicate)
	assert.Equal(t, 1, executor.callCount())

	// A genuinely different body must NOT be treated as a duplicate.
	body2 := []byte(`{"n":2}`)
	_, err = svc.TriggerWebhook(context.Background(), auto.ID, body2, sign(body2, "s3cr3t"), "")
	require.NoError(t, err)
	waitExecuted(t, executor)
	assert.Equal(t, 2, executor.callCount())
}

func TestTriggerWebhook_IdempotencyConflict(t *testing.T) {
	auto := webhookAutomation(uuid.New(), uuid.New(), "s3cr3t")
	repo := &mockRepo{
		getByIDUnscopedFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return auto, nil
		},
	}
	executor := newFakeExecutor()
	svc := newTestServiceWithWebhookDeps(repo, nil, nil, nil, executor)

	body1 := []byte(`{"n":1}`)
	_, err := svc.TriggerWebhook(context.Background(), auto.ID, body1, sign(body1, "s3cr3t"), "same-key")
	require.NoError(t, err)
	waitExecuted(t, executor)

	// Same idempotency key, different body -- sender bug (key reuse). Reject.
	body2 := []byte(`{"n":2}`)
	_, err = svc.TriggerWebhook(context.Background(), auto.ID, body2, sign(body2, "s3cr3t"), "same-key")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookIdempotencyConflict)
	assert.Equal(t, 1, executor.callCount())
}

// ============================================================================
// Signature / payload helpers
// ============================================================================

func TestWebhookSignatureValid(t *testing.T) {
	body := []byte(`{"a":1}`)
	valid := sign(body, "secret")

	assert.True(t, webhookSignatureValid(body, valid, "secret"))
	assert.False(t, webhookSignatureValid(body, valid, "wrong-secret"))
	assert.False(t, webhookSignatureValid([]byte(`{"a":2}`), valid, "secret"), "tampered body must not verify")
	assert.False(t, webhookSignatureValid(body, "deadbeef", "secret"), "missing sha256= prefix must be rejected")
	assert.False(t, webhookSignatureValid(body, "sha256=not-hex", "secret"))
	assert.False(t, webhookSignatureValid(body, "", "secret"))
}

func TestJSONPayload_ValidJSONPassesThrough(t *testing.T) {
	raw := []byte(`{"a":1}`)
	assert.Equal(t, json.RawMessage(raw), jsonPayload(raw))
}

func TestJSONPayload_NonJSONIsWrapped(t *testing.T) {
	raw := []byte("not json at all")
	wrapped := jsonPayload(raw)
	assert.True(t, json.Valid(wrapped), "wrapped payload must always be valid JSON")

	var s string
	require.NoError(t, json.Unmarshal(wrapped, &s))
	assert.Equal(t, "not json at all", s)
}
