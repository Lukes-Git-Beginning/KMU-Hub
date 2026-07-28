package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

const testSigningSecret = "test-signing-secret"

// signedRequest builds a Slack request with a signature over the exact bytes.
func signedRequest(t *testing.T, kind, body string) integration.WebhookRequest {
	t.Helper()

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(testSigningSecret))
	mac.Write([]byte("v0:" + timestamp + ":" + body))

	return integration.WebhookRequest{
		Kind: kind,
		Body: []byte(body),
		Headers: map[string]string{
			"Content-Type":              "application/x-www-form-urlencoded",
			"X-Slack-Request-Timestamp": timestamp,
			"X-Slack-Signature":         "v0=" + hex.EncodeToString(mac.Sum(nil)),
		},
	}
}

// ============================================================================
// Test doubles
// ============================================================================

type stubResolver struct {
	tenantID uuid.UUID
	err      error
	calls    int
}

func (s *stubResolver) ResolveTenant(_ context.Context, _, _ string) (uuid.UUID, error) {
	s.calls++
	if s.err != nil {
		return uuid.Nil, s.err
	}
	return s.tenantID, nil
}

type spyAcknowledger struct {
	calls    int
	tenantID uuid.UUID
	notifID  uuid.UUID
	userID   uuid.UUID
}

func (s *spyAcknowledger) MarkRead(_ context.Context, tenantID, notifID, userID uuid.UUID) (*models.Notification, error) {
	s.calls++
	s.tenantID, s.notifID, s.userID = tenantID, notifID, userID
	return &models.Notification{}, nil
}

// stubRepo answers GetAccountLink and records the tenant it saw; every other
// method panics, so a handler that reaches for data it should not need fails
// loudly instead of silently passing.
type stubRepo struct {
	integration.Repository
	link       *integration.AccountLink
	seenTenant uuid.UUID
	seenTenErr error
}

func (s *stubRepo) GetAccountLink(ctx context.Context, _, _ string) (*integration.AccountLink, error) {
	s.seenTenant, s.seenTenErr = middleware.GetTenantID(ctx)
	if s.link == nil {
		return nil, integration.ErrAccountLinkNotFound
	}
	return s.link, nil
}

// ============================================================================
// Signature verification
// ============================================================================

// A forged signature must be refused before anything is parsed or read. The
// handler holds a nil repository and a nil resolver here: any data access would
// panic, so a clean 401 is the proof that the check comes first.
func TestSlackWebhook_RejectsForgedSignature(t *testing.T) {
	h := NewWebhookHandler(nil, testSigningSecret, nil, nil, nil, nil)

	req := signedRequest(t, integration.WebhookKindSlackCommand, "team_id=T1&user_id=U1&text=link")
	req.Headers["X-Slack-Signature"] = "v0=deadbeef"

	resp := h.Process(context.Background(), req)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("forged signature: got status %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

// Slack signs the raw request body. The fields below are in Slack's own order,
// not sorted -- the pre-tunnel handler verified against url.Values.Encode(),
// which sorts by key, so this exact payload used to fail verification and every
// slash command answered 401. Verification must now pass and the request must
// reach the tenant resolution.
func TestSlackWebhook_VerifiesUnsortedRawBody(t *testing.T) {
	resolver := &stubResolver{err: integration.ErrTenantUnresolved}
	h := NewWebhookHandler(nil, testSigningSecret, nil, nil, resolver, nil)

	body := "token=xyz&team_id=T1&user_id=U1&command=%2Fkmuhub&text=link&channel_id=C1"
	resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackCommand, body))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("raw-body signature: got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if resolver.calls != 1 {
		t.Fatalf("tenant resolution: got %d calls, want 1 (request never got past verification)", resolver.calls)
	}
}

// ============================================================================
// Tenant resolution
// ============================================================================

// An unmapped workspace is refused, not guessed. The nil repository proves no
// data access happens on that path.
func TestSlackWebhook_UnresolvedWorkspaceTouchesNoData(t *testing.T) {
	resolver := &stubResolver{err: integration.ErrTenantUnresolved}
	h := NewWebhookHandler(nil, testSigningSecret, nil, nil, resolver, nil)

	resp := h.Process(context.Background(),
		signedRequest(t, integration.WebhookKindSlackCommand, "team_id=T-unknown&user_id=U1&text=unlink"))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unresolved workspace: got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		t.Fatalf("unmarshal ephemeral response: %v", err)
	}
	if payload.Text == "" {
		t.Fatal("unresolved workspace: expected an explanatory ephemeral message")
	}
}

// The resolved tenant has to reach the repository, otherwise RLS filters every
// row away and the webhook silently finds nothing.
func TestSlackWebhook_ResolvedTenantReachesRepository(t *testing.T) {
	tenantID := uuid.New()
	resolver := &stubResolver{tenantID: tenantID}
	repo := &stubRepo{}
	h := NewWebhookHandler(nil, testSigningSecret, repo, nil, resolver, nil)

	h.Process(context.Background(),
		signedRequest(t, integration.WebhookKindSlackCommand, "team_id=T1&user_id=U1&text=unlink"))

	if repo.seenTenErr != nil {
		t.Fatalf("repository saw no tenant: %v", repo.seenTenErr)
	}
	if repo.seenTenant != tenantID {
		t.Fatalf("repository tenant: got %s, want %s", repo.seenTenant, tenantID)
	}
}

// ============================================================================
// Honest actions
// ============================================================================

func interactionBody(t *testing.T, actionID string) string {
	t.Helper()

	callback := map[string]any{
		"type": "block_actions",
		"user": map[string]any{"id": "U1"},
		"team": map[string]any{"id": "T1"},
		"actions": []any{
			map[string]any{
				"action_id": actionID,
				// block_id is what slack-go uses to tell a block action from an
				// attachment action; a fixture without it silently decodes into
				// the wrong list and no action is ever seen.
				"block_id": "kmuhub_actions",
				"value":    uuid.New().String(),
				"type":     "button",
			},
		},
	}
	encoded, err := json.Marshal(callback)
	if err != nil {
		t.Fatalf("marshal callback: %v", err)
	}
	return "payload=" + urlEncode(string(encoded))
}

func urlEncode(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '.', c == '~':
			b = append(b, c)
		default:
			b = append(b, '%', "0123456789ABCDEF"[c>>4], "0123456789ABCDEF"[c&0x0f])
		}
	}
	return string(b)
}

// Acknowledge is the one card action Cosmi actually executes.
func TestSlackWebhook_AcknowledgeMarksNotificationRead(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	resolver := &stubResolver{tenantID: tenantID}
	repo := &stubRepo{link: &integration.AccountLink{ID: uuid.New(), KMUHubUserID: userID}}
	acks := &spyAcknowledger{}
	h := NewWebhookHandler(nil, testSigningSecret, repo, nil, resolver, acks)

	h.Process(context.Background(),
		signedRequest(t, integration.WebhookKindSlackInteraction, interactionBody(t, "kmuhub_acknowledge")))

	if acks.calls != 1 {
		t.Fatalf("acknowledge: got %d MarkRead calls, want 1", acks.calls)
	}
	if acks.tenantID != tenantID || acks.userID != userID {
		t.Fatalf("acknowledge: marked read as tenant %s / user %s, want %s / %s",
			acks.tenantID, acks.userID, tenantID, userID)
	}
}

// Approve and reject have no counterpart in Cosmi. They must not run anything
// and must not be answered as if they had -- the card stays untouched and the
// user gets an honest ephemeral reply.
func TestSlackWebhook_UnsupportedActionRunsNothing(t *testing.T) {
	for _, actionID := range []string{"kmuhub_approve", "kmuhub_reject", "kmuhub_reply"} {
		t.Run(actionID, func(t *testing.T) {
			resolver := &stubResolver{tenantID: uuid.New()}
			repo := &stubRepo{link: &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New()}}
			acks := &spyAcknowledger{}
			h := NewWebhookHandler(nil, testSigningSecret, repo, nil, resolver, acks)

			resp := h.Process(context.Background(),
				signedRequest(t, integration.WebhookKindSlackInteraction, interactionBody(t, actionID)))

			if acks.calls != 0 {
				t.Fatalf("%s: got %d MarkRead calls, want 0", actionID, acks.calls)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%s: got status %d, want %d", actionID, resp.StatusCode, http.StatusOK)
			}
		})
	}
}
