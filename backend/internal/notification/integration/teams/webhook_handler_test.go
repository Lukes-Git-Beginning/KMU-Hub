package teams

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// ============================================================================
// Test doubles
// ============================================================================

type stubResolver struct {
	tenantID uuid.UUID
	err      error
	calls    int
}

func (s *stubResolver) ResolveTenant(_ context.Context, _, workspaceID string) (uuid.UUID, error) {
	s.calls++
	if s.err != nil {
		return uuid.Nil, s.err
	}
	return s.tenantID, nil
}

type spyAcknowledger struct {
	calls    int
	err      error
	tenantID uuid.UUID
	notifID  uuid.UUID
	userID   uuid.UUID
}

func (s *spyAcknowledger) MarkRead(_ context.Context, tenantID, notifID, userID uuid.UUID) (*models.Notification, error) {
	s.calls++
	s.tenantID, s.notifID, s.userID = tenantID, notifID, userID
	if s.err != nil {
		return nil, s.err
	}
	return &models.Notification{}, nil
}

// stubRepo answers GetAccountLink and records the tenant it saw on the
// context; every other Repository method panics, so a handler that reaches
// for data it should not need fails loudly instead of silently passing.
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

// linkTokenRepo backs integration.NewAccountLinkService for the /kmuhub link
// flow. failCreate lets a test force GenerateLinkToken to fail past the
// cleanup step, without needing a real database.
type linkTokenRepo struct {
	integration.Repository
	failCreate bool
}

func (r *linkTokenRepo) CleanupExpiredTokens(_ context.Context) (int, error) {
	return 0, nil
}

func (r *linkTokenRepo) CreateLinkToken(_ context.Context, _ *integration.LinkToken) error {
	if r.failCreate {
		return context.DeadlineExceeded
	}
	return nil
}

func incomingWithTenant(teamsTenantID string) *IncomingAction {
	act := schema.Activity{
		Type: schema.Message,
		From: schema.ChannelAccount{ID: "user-1"},
	}
	if teamsTenantID != "" {
		act.ChannelData = map[string]any{
			"tenant": map[string]any{"id": teamsTenantID},
		}
	}
	return &IncomingAction{Activity: act, ExternalUserID: "user-1"}
}

func decodeInvokeText(t *testing.T, resp integration.WebhookResponse) string {
	t.Helper()
	var payload struct {
		Value string `json:"value"`
	}
	require.NoError(t, json.Unmarshal(resp.Body, &payload))
	return payload.Value
}

func decodeMessageText(t *testing.T, resp integration.WebhookResponse) string {
	t.Helper()
	var payload struct {
		Text string `json:"text"`
	}
	require.NoError(t, json.Unmarshal(resp.Body, &payload))
	return payload.Text
}

// ============================================================================
// Process — the one path that is safe to exercise without a real Bot
// Framework adapter, since h.client == nil short-circuits before ParseRequest.
// ============================================================================

func TestProcess_NilClientReturnsServiceUnavailable(t *testing.T) {
	h := NewWebhookHandler(nil, nil, nil, nil, nil)

	resp := h.Process(context.Background(), integration.WebhookRequest{Kind: integration.WebhookKindTeamsActivity})

	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// ============================================================================
// tenantContext / activityTenantID
// ============================================================================

func TestTenantContext_ResolvesFromChannelDataTenantID(t *testing.T) {
	tenantID := uuid.New()
	resolver := &stubResolver{tenantID: tenantID}
	h := NewWebhookHandler(nil, nil, nil, resolver, nil)

	incoming := incomingWithTenant("aad-tenant-1")
	ctx, gotTenant, err := h.tenantContext(context.Background(), incoming)

	require.NoError(t, err)
	require.Equal(t, tenantID, gotTenant)
	ctxTenant, ctxErr := middleware.GetTenantID(ctx)
	require.NoError(t, ctxErr)
	require.Equal(t, tenantID, ctxTenant)
	require.Equal(t, 1, resolver.calls)
}

// A Teams activity with no channelData.tenant.id at all -- e.g. a locally
// crafted test payload or a channel type that never sets it -- must resolve
// against an empty workspace id and be refused, not panic on a failed
// type assertion.
func TestTenantContext_MissingChannelDataResolvesEmptyWorkspace(t *testing.T) {
	resolver := &stubResolver{err: integration.ErrTenantUnresolved}
	h := NewWebhookHandler(nil, nil, nil, resolver, nil)

	_, _, err := h.tenantContext(context.Background(), incomingWithTenant(""))

	require.ErrorIs(t, err, integration.ErrTenantUnresolved)
}

// ============================================================================
// handleInvoke — Action.Execute (adaptive card button) activities.
// ============================================================================

func TestHandleInvoke_UnresolvedTenantTouchesNoRepository(t *testing.T) {
	resolver := &stubResolver{err: integration.ErrTenantUnresolved}
	h := NewWebhookHandler(nil, nil, nil, resolver, nil)

	incoming := incomingWithTenant("unmapped-org")
	incoming.ActionType = string(integration.ActionAcknowledge)

	resp := h.handleInvoke(context.Background(), incoming)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, decodeInvokeText(t, resp), "keinem Cosmi-Mandanten zugeordnet")
}

func TestHandleInvoke_UnlinkedUserAsksToLink(t *testing.T) {
	resolver := &stubResolver{tenantID: uuid.New()}
	repo := &stubRepo{}
	h := NewWebhookHandler(nil, repo, nil, resolver, nil)

	incoming := incomingWithTenant("org-1")
	incoming.ActionType = string(integration.ActionAcknowledge)

	resp := h.handleInvoke(context.Background(), incoming)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, decodeInvokeText(t, resp), "/kmuhub link")
	require.NoError(t, repo.seenTenErr, "GetAccountLink must run under the resolved tenant")
}

// Approve/reject/reply have no counterpart in Cosmi today and must not be
// answered as if they had run -- the acknowledger must see zero calls.
func TestHandleInvoke_UnsupportedActionRunsNothing(t *testing.T) {
	for _, action := range []string{"approve", "reject", "reply"} {
		t.Run(action, func(t *testing.T) {
			resolver := &stubResolver{tenantID: uuid.New()}
			repo := &stubRepo{link: &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New()}}
			acks := &spyAcknowledger{}
			h := NewWebhookHandler(nil, repo, nil, resolver, acks)

			incoming := incomingWithTenant("org-1")
			incoming.ActionType = action

			resp := h.handleInvoke(context.Background(), incoming)

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Contains(t, decodeInvokeText(t, resp), "noch nicht ausgefuehrt")
			require.Equal(t, 0, acks.calls)
		})
	}
}

func TestHandleInvoke_AcknowledgeMarksNotificationRead(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	resolver := &stubResolver{tenantID: tenantID}
	repo := &stubRepo{link: &integration.AccountLink{ID: uuid.New(), KMUHubUserID: userID}}
	acks := &spyAcknowledger{}
	h := NewWebhookHandler(nil, repo, nil, resolver, acks)

	incoming := incomingWithTenant("org-1")
	incoming.ActionType = string(integration.ActionAcknowledge)
	incoming.NotificationID = uuid.New().String()

	resp := h.handleInvoke(context.Background(), incoming)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, decodeInvokeText(t, resp), "Als gelesen markiert")
	require.Equal(t, 1, acks.calls)
	require.Equal(t, tenantID, acks.tenantID)
	require.Equal(t, userID, acks.userID)
}

// An invalid (non-UUID) notification id from a malformed or forged Action.Value
// payload must produce the honest failure message, not propagate a parse panic.
func TestHandleInvoke_AcknowledgeInvalidNotificationID(t *testing.T) {
	resolver := &stubResolver{tenantID: uuid.New()}
	repo := &stubRepo{link: &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New()}}
	acks := &spyAcknowledger{}
	h := NewWebhookHandler(nil, repo, nil, resolver, acks)

	incoming := incomingWithTenant("org-1")
	incoming.ActionType = string(integration.ActionAcknowledge)
	incoming.NotificationID = "not-a-uuid"

	resp := h.handleInvoke(context.Background(), incoming)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, decodeInvokeText(t, resp), "konnte nicht als gelesen markiert werden")
	require.Equal(t, 0, acks.calls, "uuid.Parse must fail before MarkRead is ever called")
}

func TestHandleInvoke_AcknowledgeServiceErrorSurfacesFailureMessage(t *testing.T) {
	resolver := &stubResolver{tenantID: uuid.New()}
	repo := &stubRepo{link: &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New()}}
	acks := &spyAcknowledger{err: context.DeadlineExceeded}
	h := NewWebhookHandler(nil, repo, nil, resolver, acks)

	incoming := incomingWithTenant("org-1")
	incoming.ActionType = string(integration.ActionAcknowledge)
	incoming.NotificationID = uuid.New().String()

	resp := h.handleInvoke(context.Background(), incoming)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, decodeInvokeText(t, resp), "konnte nicht als gelesen markiert werden")
}

// The notification service is optional wiring (h.notifications can be nil in
// a partially configured deployment); acknowledge must fail cleanly instead
// of dereferencing a nil interface.
func TestAcknowledge_NilNotificationsServiceReturnsWiringError(t *testing.T) {
	h := NewWebhookHandler(nil, nil, nil, nil, nil)

	err := h.acknowledge(context.Background(), uuid.New(), &integration.AccountLink{KMUHubUserID: uuid.New()}, uuid.New().String())

	require.ErrorContains(t, err, "not wired")
}

// ============================================================================
// handleMessage — text commands, in particular /kmuhub link.
// ============================================================================

func TestHandleMessage_UnknownCommandListsAvailableCommands(t *testing.T) {
	h := NewWebhookHandler(nil, nil, nil, nil, nil)

	incoming := incomingWithTenant("org-1")
	incoming.Activity.Text = "hello there"

	resp := h.handleMessage(context.Background(), incoming)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, decodeMessageText(t, resp), "Verfuegbare Befehle")
}

func TestHandleMessage_LinkCommand_UnresolvedTenant(t *testing.T) {
	resolver := &stubResolver{err: integration.ErrTenantUnresolved}
	h := NewWebhookHandler(nil, nil, nil, resolver, nil)

	incoming := incomingWithTenant("unmapped-org")
	incoming.Activity.Text = "/kmuhub link"

	resp := h.handleMessage(context.Background(), incoming)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, decodeMessageText(t, resp), "keinem Cosmi-Mandanten zugeordnet")
}

func TestHandleMessage_LinkCommand_TokenGenerationFailureIsHonest(t *testing.T) {
	resolver := &stubResolver{tenantID: uuid.New()}
	linkService := integration.NewAccountLinkService(&linkTokenRepo{failCreate: true})
	h := NewWebhookHandler(nil, nil, linkService, resolver, nil)

	incoming := incomingWithTenant("org-1")
	incoming.Activity.Text = "link"

	resp := h.handleMessage(context.Background(), incoming)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, decodeMessageText(t, resp), "Fehler beim Erstellen des Verknuepfungstokens")
}

func TestHandleMessage_LinkCommand_SuccessReturnsToken(t *testing.T) {
	resolver := &stubResolver{tenantID: uuid.New()}
	linkService := integration.NewAccountLinkService(&linkTokenRepo{})
	h := NewWebhookHandler(nil, nil, linkService, resolver, nil)

	incoming := incomingWithTenant("org-1")
	incoming.Activity.Text = "/kmuhub link"

	resp := h.handleMessage(context.Background(), incoming)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	text := decodeMessageText(t, resp)
	require.Contains(t, text, "Verknuepfungstoken")
	require.Contains(t, text, "Einstellungen > Integrationen > Konto verknuepfen")
}
