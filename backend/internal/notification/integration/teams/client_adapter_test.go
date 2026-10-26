package teams

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/infracloudio/msbotbuilder-go/core"
	"github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// fakeAdapter replaces the Bot Framework adapter at its interface. ProactiveMessage
// runs the real handler the client passes in, so the activity the client would
// have sent is observable; the network hop is the only thing missing.
type fakeAdapter struct {
	core.Adapter

	parsed    schema.Activity
	parseErr  error
	sendErr   error
	sentRef   schema.ConversationReference
	sent      schema.Activity
	sendCalls int
}

func (f *fakeAdapter) ParseRequest(_ context.Context, _ *http.Request) (schema.Activity, error) {
	return f.parsed, f.parseErr
}

func (f *fakeAdapter) ProactiveMessage(_ context.Context, ref schema.ConversationReference, h activity.Handler) error {
	f.sendCalls++
	f.sentRef = ref
	if f.sendErr != nil {
		return f.sendErr
	}
	act, err := h.OnMessage(&activity.TurnContext{Activity: schema.Activity{Type: schema.Message}})
	f.sent = act
	return err
}

func newFakeClient(f *fakeAdapter) *Client {
	return &Client{adapter: f, appID: "app-id", appPassword: "app-secret"}
}

func testMapping(t *testing.T) *integration.ChannelMapping {
	t.Helper()
	ref, err := json.Marshal(schema.ConversationReference{
		ServiceURL:   "https://smba.example/emea/",
		Conversation: schema.ConversationAccount{ID: "conv-1"},
	})
	require.NoError(t, err)
	return &integration.ChannelMapping{ChannelID: "chan-1", PlatformData: ref}
}

func testNotification(body string) *models.Notification {
	n := &models.Notification{ID: uuid.New(), ModuleID: "crm", Title: "Deal won"}
	if body != "" {
		n.Body = &body
	}
	return n
}

// sentCard decodes the Adaptive Card from the only attachment the client sent.
func sentCard(t *testing.T, f *fakeAdapter) AdaptiveCard {
	t.Helper()
	require.Len(t, f.sent.Attachments, 1)
	require.Equal(t, "application/vnd.microsoft.card.adaptive", f.sent.Attachments[0].ContentType)
	raw, err := json.Marshal(f.sent.Attachments[0].Content)
	require.NoError(t, err)
	var card AdaptiveCard
	require.NoError(t, json.Unmarshal(raw, &card))
	return card
}

// ============================================================================
// NewClient
// ============================================================================

func TestNewClientBuildsTheLoginEndpointAndKeepsCredentials(t *testing.T) {
	c, err := NewClient("app-id", "app-secret")
	require.NoError(t, err)
	require.NotNil(t, c.adapter)
	require.Equal(t, "app-id", c.appID)
	require.Equal(t, "app-secret", c.appPassword)
	require.True(t, strings.HasPrefix(c.tokenURL, "https://"), "token endpoint must be https, got %q", c.tokenURL)
	require.NotNil(t, c.httpClient)
	require.NotZero(t, c.httpClient.Timeout, "the probe must not hang on an unreachable endpoint")
}

// ============================================================================
// PostNotification / UpdateCard
// ============================================================================

func TestPostNotificationSendsTheCardToTheStoredConversation(t *testing.T) {
	f := &fakeAdapter{}
	c := newFakeClient(f)
	notif := testNotification("Contract signed")

	res, err := c.PostNotification(context.Background(), testMapping(t), notif,
		[]integration.ActionType{integration.ActionAcknowledge})

	require.NoError(t, err)
	require.True(t, res.Success)
	require.Equal(t, "conv-1", f.sentRef.Conversation.ID)
	require.Equal(t, "https://smba.example/emea/", f.sentRef.ServiceURL)

	card := sentCard(t, f)
	require.Len(t, card.Actions, 1)
	require.Equal(t, "Action.Execute", card.Actions[0].Type)
	require.Equal(t, notif.ID.String(), card.Actions[0].Data["notification_id"])
	require.Equal(t, "acknowledge", card.Actions[0].Data["action_type"])
}

func TestPostNotificationRejectsBrokenPlatformData(t *testing.T) {
	f := &fakeAdapter{}
	c := newFakeClient(f)
	mapping := &integration.ChannelMapping{ChannelID: "chan-1", PlatformData: json.RawMessage(`{not json`)}

	_, err := c.PostNotification(context.Background(), mapping, testNotification(""), nil)

	require.ErrorContains(t, err, "parse conversation reference")
	require.Zero(t, f.sendCalls, "nothing may be sent without a usable conversation")
}

func TestPostNotificationWrapsSendFailure(t *testing.T) {
	boom := errors.New("bot framework down")
	f := &fakeAdapter{sendErr: boom}
	c := newFakeClient(f)

	res, err := c.PostNotification(context.Background(), testMapping(t), testNotification(""), nil)

	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "send proactive message")
	require.Nil(t, res, "a failed delivery must not report success")
}

func TestUpdateCardSendsTheResolvedCardWithoutButtons(t *testing.T) {
	f := &fakeAdapter{}
	c := newFakeClient(f)

	err := c.UpdateCard(context.Background(), testMapping(t), "act-1", testNotification("Body"), "approve", "Max")

	require.NoError(t, err)
	card := sentCard(t, f)
	require.Empty(t, card.Actions, "a resolved card must not offer the buttons again")
	require.Contains(t, card.Body[len(card.Body)-1].Text, "Max")
}

func TestUpdateCardRejectsBrokenPlatformData(t *testing.T) {
	f := &fakeAdapter{}
	c := newFakeClient(f)
	mapping := &integration.ChannelMapping{PlatformData: json.RawMessage(`[`)}

	err := c.UpdateCard(context.Background(), mapping, "act-1", testNotification(""), "approve", "Max")

	require.ErrorContains(t, err, "parse conversation reference")
	require.Zero(t, f.sendCalls)
}

func TestUpdateCardWrapsSendFailure(t *testing.T) {
	boom := errors.New("bot framework down")
	c := newFakeClient(&fakeAdapter{sendErr: boom})

	err := c.UpdateCard(context.Background(), testMapping(t), "act-1", testNotification(""), "approve", "Max")

	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "update card")
}

// ============================================================================
// HandleIncomingActivity
// ============================================================================

func TestHandleIncomingActivityParsesTheButtonPayload(t *testing.T) {
	notifID := uuid.NewString()
	f := &fakeAdapter{parsed: schema.Activity{
		Type:  schema.Invoke,
		From:  schema.ChannelAccount{ID: "user-9"},
		Value: map[string]any{"action_type": "reply", "notification_id": notifID, "reply_text": "ok"},
	}}
	c := newFakeClient(f)

	got, err := c.HandleIncomingActivity(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil))

	require.NoError(t, err)
	require.Equal(t, "user-9", got.ExternalUserID)
	require.Equal(t, "reply", got.ActionType)
	require.Equal(t, notifID, got.NotificationID)
	require.Equal(t, "ok", got.ReplyText)
}

// A plain message carries no Value; it must parse with an empty action, not fail.
func TestHandleIncomingActivityWithoutValueHasNoAction(t *testing.T) {
	f := &fakeAdapter{parsed: schema.Activity{Type: schema.Message, From: schema.ChannelAccount{ID: "user-9"}}}
	c := newFakeClient(f)

	got, err := c.HandleIncomingActivity(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil))

	require.NoError(t, err)
	require.Empty(t, got.ActionType)
	require.Empty(t, got.NotificationID)
}

func TestHandleIncomingActivityPropagatesVerificationFailure(t *testing.T) {
	f := &fakeAdapter{parseErr: errors.New("bad jwt")}
	c := newFakeClient(f)

	got, err := c.HandleIncomingActivity(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil))

	require.Nil(t, got)
	require.ErrorContains(t, err, "parse teams request")
}

// ============================================================================
// WebhookHandler.Process — dispatch behind the adapter's JWT check
// ============================================================================

func TestProcessRefusesWhenTheActivityDoesNotVerify(t *testing.T) {
	resolver := &stubResolver{tenantID: uuid.New()}
	h := NewWebhookHandler(newFakeClient(&fakeAdapter{parseErr: errors.New("bad jwt")}), &stubRepo{}, nil, resolver, nil)

	resp := h.Process(context.Background(), integration.WebhookRequest{Kind: integration.WebhookKindTeamsActivity})

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Zero(t, resolver.calls, "nothing may be resolved before the signature check passes")
}

func TestProcessDispatchesInvokeToTheActionHandler(t *testing.T) {
	notifID := uuid.New()
	ack := &spyAcknowledger{}
	f := &fakeAdapter{parsed: schema.Activity{
		Type:        schema.Invoke,
		From:        schema.ChannelAccount{ID: "user-1"},
		ChannelData: map[string]any{"tenant": map[string]any{"id": "org-1"}},
		Value:       map[string]any{"action_type": "acknowledge", "notification_id": notifID.String()},
	}}
	repo := &stubRepo{link: &integration.AccountLink{KMUHubUserID: uuid.New()}}
	h := NewWebhookHandler(newFakeClient(f), repo, nil, &stubResolver{tenantID: uuid.New()}, ack)

	resp := h.Process(context.Background(), integration.WebhookRequest{
		Kind:    integration.WebhookKindTeamsActivity,
		Body:    []byte(`{}`),
		Headers: map[string]string{"Authorization": "Bearer x"},
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "Als gelesen markiert.", decodeInvokeText(t, resp))
	require.Equal(t, 1, ack.calls)
	require.Equal(t, notifID, ack.notifID)
}

func TestProcessDispatchesMessageToTheCommandHandler(t *testing.T) {
	f := &fakeAdapter{parsed: schema.Activity{Type: schema.Message, Text: "hello"}}
	h := NewWebhookHandler(newFakeClient(f), &stubRepo{}, nil, &stubResolver{}, nil)

	resp := h.Process(context.Background(), integration.WebhookRequest{Kind: integration.WebhookKindTeamsActivity})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, decodeMessageText(t, resp), "Verfügbare Befehle")
}

// Activity types the bot does not act on (typing, conversationUpdate, ...) are
// acknowledged with 200 and an empty body so Teams does not retry them.
func TestProcessAcknowledgesOtherActivityTypesWithAnEmptyBody(t *testing.T) {
	f := &fakeAdapter{parsed: schema.Activity{Type: schema.ConversationUpdate}}
	h := NewWebhookHandler(newFakeClient(f), &stubRepo{}, nil, &stubResolver{}, nil)

	resp := h.Process(context.Background(), integration.WebhookRequest{Kind: integration.WebhookKindTeamsActivity})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Empty(t, resp.Body)
}
