package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	slackapi "github.com/slack-go/slack"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// ============================================================================
// Fake Slack Web API
// ============================================================================

type slackCall struct {
	Method string
	Auth   string
	Form   url.Values
}

// fakeSlack is an httptest stand-in for slack.com/api. The real slack-go client
// runs against it, so request encoding and response decoding are the library's
// own; only the network edge is replaced.
type fakeSlack struct {
	srv     *httptest.Server
	mu      sync.Mutex
	calls   []slackCall
	replies map[string]string
}

func newFakeSlack(t *testing.T) *fakeSlack {
	t.Helper()

	f := &fakeSlack{replies: map[string]string{}}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		method := strings.TrimPrefix(r.URL.Path, "/")

		f.mu.Lock()
		f.calls = append(f.calls, slackCall{Method: method, Auth: r.Header.Get("Authorization"), Form: r.PostForm})
		reply, ok := f.replies[method]
		f.mu.Unlock()

		if !ok {
			reply = `{"ok":true,"channel":"C1","ts":"1700000000.000100"}`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// client builds the package Client around the fake. NewClient hard-wires
// slack.com, so the struct is assembled directly with the same fields.
func (f *fakeSlack) client(token string) *Client {
	return &Client{
		api:      slackapi.New(token, slackapi.OptionAPIURL(f.srv.URL+"/")),
		botToken: token,
	}
}

func (f *fakeSlack) fail(method, code string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.replies[method] = `{"ok":false,"error":"` + code + `"}`
}

func (f *fakeSlack) only(t *testing.T, method string) slackCall {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	var found []slackCall
	for _, c := range f.calls {
		if c.Method == method {
			found = append(found, c)
		}
	}
	require.Len(t, found, 1, "calls to %s", method)
	return found[0]
}

func (f *fakeSlack) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// ============================================================================
// Client
// ============================================================================

func TestNewClient_KeepsBotToken(t *testing.T) {
	c := NewClient("xoxb-secret")

	require.NotNil(t, c.api)
	require.Equal(t, "xoxb-secret", c.botToken)
}

func TestClient_ProbeConnection(t *testing.T) {
	t.Run("reports user and workspace from auth.test", func(t *testing.T) {
		f := newFakeSlack(t)
		f.replies["auth.test"] = `{"ok":true,"user":"cosmi-bot","team":"Zentria"}`

		res, err := f.client("xoxb-1").ProbeConnection(context.Background())

		require.NoError(t, err)
		require.Equal(t, "authenticated as cosmi-bot in workspace Zentria", res.Detail)
		call := f.only(t, "auth.test")
		require.True(t, call.Auth == "Bearer xoxb-1" || call.Form.Get("token") == "xoxb-1", "the bot token must reach Slack")
	})

	t.Run("wraps a rejected token", func(t *testing.T) {
		f := newFakeSlack(t)
		f.fail("auth.test", "invalid_auth")

		res, err := f.client("xoxb-bad").ProbeConnection(context.Background())

		require.Nil(t, res)
		require.ErrorContains(t, err, "slack auth.test")
		require.ErrorContains(t, err, "invalid_auth")
	})
}

func TestClient_PostNotification(t *testing.T) {
	mapping := &integration.ChannelMapping{ChannelID: "C42"}
	body := "Rechnung 2026-17 ist überfällig"

	t.Run("sends blocks, fallback text and module colour", func(t *testing.T) {
		f := newFakeSlack(t)
		notif := testNotification("crm", &body)

		res, err := f.client("xoxb-1").PostNotification(context.Background(), mapping, notif,
			[]integration.ActionType{integration.ActionAcknowledge})

		require.NoError(t, err)
		require.True(t, res.Success)
		require.Equal(t, "1700000000.000100", res.PlatformMessageID)

		form := f.only(t, "chat.postMessage").Form
		require.Equal(t, "C42", form.Get("channel"))
		require.Equal(t, notif.Title+": "+body, form.Get("text"))

		var blocks []map[string]any
		require.NoError(t, json.Unmarshal([]byte(form.Get("blocks")), &blocks))
		require.NotEmpty(t, blocks)
		require.Contains(t, form.Get("blocks"), "kmuhub_acknowledge")
		require.Contains(t, form.Get("blocks"), notif.ID.String())

		require.Contains(t, form.Get("attachments"), integration.ModuleColors["crm"])
	})

	t.Run("falls back to the title and a neutral colour", func(t *testing.T) {
		f := newFakeSlack(t)
		notif := testNotification("unknown-module", nil)

		_, err := f.client("xoxb-1").PostNotification(context.Background(), mapping, notif, nil)

		require.NoError(t, err)
		form := f.only(t, "chat.postMessage").Form
		require.Equal(t, notif.Title, form.Get("text"))
		require.Contains(t, form.Get("attachments"), "#6b7280")
	})

	t.Run("wraps a Slack error and reports no delivery", func(t *testing.T) {
		f := newFakeSlack(t)
		f.fail("chat.postMessage", "channel_not_found")

		res, err := f.client("xoxb-1").PostNotification(context.Background(), mapping,
			testNotification("crm", nil), nil)

		require.Nil(t, res)
		require.ErrorContains(t, err, "post slack message")
		require.ErrorContains(t, err, "channel_not_found")
	})
}

func TestClient_UpdateMessage(t *testing.T) {
	blocks := []slackapi.Block{
		slackapi.NewSectionBlock(slackapi.NewTextBlockObject("mrkdwn", "erledigt", false, false), nil, nil),
	}

	t.Run("targets the original message", func(t *testing.T) {
		f := newFakeSlack(t)

		err := f.client("xoxb-1").UpdateMessage(context.Background(), "C42", "1699.0001", blocks)

		require.NoError(t, err)
		form := f.only(t, "chat.update").Form
		require.Equal(t, "C42", form.Get("channel"))
		require.Equal(t, "1699.0001", form.Get("ts"))
		require.Contains(t, form.Get("blocks"), "erledigt")
	})

	t.Run("wraps a Slack error", func(t *testing.T) {
		f := newFakeSlack(t)
		f.fail("chat.update", "message_not_found")

		err := f.client("xoxb-1").UpdateMessage(context.Background(), "C42", "1699.0001", blocks)

		require.ErrorContains(t, err, "update slack message")
		require.ErrorContains(t, err, "message_not_found")
	})
}

func TestClient_PostEphemeral(t *testing.T) {
	t.Run("addresses one user in the channel", func(t *testing.T) {
		f := newFakeSlack(t)

		err := f.client("xoxb-1").PostEphemeral(context.Background(), "C42", "U7", "nur für Sie")

		require.NoError(t, err)
		form := f.only(t, "chat.postEphemeral").Form
		require.Equal(t, "C42", form.Get("channel"))
		require.Equal(t, "U7", form.Get("user"))
		require.Equal(t, "nur für Sie", form.Get("text"))
	})

	t.Run("wraps a Slack error", func(t *testing.T) {
		f := newFakeSlack(t)
		f.fail("chat.postEphemeral", "user_not_in_channel")

		err := f.client("xoxb-1").PostEphemeral(context.Background(), "C42", "U7", "x")

		require.ErrorContains(t, err, "post ephemeral message")
		require.ErrorContains(t, err, "user_not_in_channel")
	})
}

// ============================================================================
// Webhook handler against the fake API
// ============================================================================

// flowRepo implements the account-link and token calls the handler uses.
type flowRepo struct {
	integration.Repository
	link      *integration.AccountLink
	getErr    error
	deleted   []uuid.UUID
	deleteErr error
	tokens    []*integration.LinkToken
	tokenErr  error
}

func (r *flowRepo) GetAccountLink(context.Context, string, string) (*integration.AccountLink, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.link == nil {
		return nil, integration.ErrAccountLinkNotFound
	}
	return r.link, nil
}

func (r *flowRepo) DeleteAccountLink(_ context.Context, id uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *flowRepo) CleanupExpiredTokens(context.Context) (int, error) { return 0, nil }

func (r *flowRepo) CreateLinkToken(_ context.Context, tok *integration.LinkToken) error {
	if r.tokenErr != nil {
		return r.tokenErr
	}
	r.tokens = append(r.tokens, tok)
	return nil
}

type failingAcknowledger struct{ calls int }

func (a *failingAcknowledger) MarkRead(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*models.Notification, error) {
	a.calls++
	return nil, errors.New("boom")
}

// blockActionBody builds a complete block_actions callback including the channel
// and the original message, which the card rewrite needs.
func blockActionBody(t *testing.T, teamID, channelID, messageTs, actionID, value string) string {
	t.Helper()

	callback := map[string]any{
		"type":       "block_actions",
		"user":       map[string]any{"id": "U1"},
		"team":       map[string]any{"id": teamID},
		"channel":    map[string]any{"id": channelID},
		"message_ts": messageTs,
		"message":    map[string]any{"text": "Rechnung fällig"},
		"actions": []any{
			map[string]any{"action_id": actionID, "block_id": "kmuhub_actions", "value": value, "type": "button"},
		},
	}
	encoded, err := json.Marshal(callback)
	require.NoError(t, err)
	return "payload=" + urlEncode(string(encoded))
}

func ephemeralText(t *testing.T, resp integration.WebhookResponse) string {
	t.Helper()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var payload struct {
		Type string `json:"response_type"`
		Text string `json:"text"`
	}
	require.NoError(t, json.Unmarshal(resp.Body, &payload))
	require.Equal(t, "ephemeral", payload.Type)
	return payload.Text
}

func TestSlackWebhook_AcknowledgeRewritesTheCard(t *testing.T) {
	f := newFakeSlack(t)
	notifID := uuid.New()
	link := &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New(), ExternalDisplayName: "Anna"}
	acks := &spyAcknowledger{}
	h := NewWebhookHandler(f.client("xoxb-1"), testSigningSecret, &flowRepo{link: link}, nil,
		&stubResolver{tenantID: uuid.New()}, acks)

	body := blockActionBody(t, "T1", "C42", "1699.0001", "kmuhub_acknowledge", notifID.String())
	resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, notifID, acks.notifID)
	require.Equal(t, link.KMUHubUserID, acks.userID)

	form := f.only(t, "chat.update").Form
	require.Equal(t, "C42", form.Get("channel"))
	require.Equal(t, "1699.0001", form.Get("ts"))
	require.Contains(t, form.Get("blocks"), "Rechnung fällig")
	require.Contains(t, form.Get("blocks"), "Anna")
	require.NotContains(t, form.Get("blocks"), "kmuhub_acknowledge", "buttons must be gone from the resolved card")
}

func TestSlackWebhook_AcknowledgeUsesFallbackNameWithoutDisplayName(t *testing.T) {
	f := newFakeSlack(t)
	link := &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New()}
	h := NewWebhookHandler(f.client("xoxb-1"), testSigningSecret, &flowRepo{link: link}, nil,
		&stubResolver{tenantID: uuid.New()}, &spyAcknowledger{})

	body := blockActionBody(t, "T1", "C42", "1699.0001", "kmuhub_acknowledge", uuid.NewString())
	h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

	require.Contains(t, f.only(t, "chat.update").Form.Get("blocks"), "Benutzer")
}

func TestSlackWebhook_AcknowledgeSkipsRewriteWithoutMessageContext(t *testing.T) {
	// Either half missing means there is no message to rewrite.
	for _, tc := range []struct{ name, channel, ts string }{
		{"no channel and no ts", "", ""},
		{"channel without ts", "C42", ""},
		{"ts without channel", "", "1699.0001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeSlack(t)
			link := &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New()}
			acks := &spyAcknowledger{}
			h := NewWebhookHandler(f.client("xoxb-1"), testSigningSecret, &flowRepo{link: link}, nil,
				&stubResolver{tenantID: uuid.New()}, acks)

			body := blockActionBody(t, "T1", tc.channel, tc.ts, "kmuhub_acknowledge", uuid.NewString())
			resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, 1, acks.calls)
			require.Zero(t, f.count())
		})
	}
}

// The tenant resolved from the workspace has to be on the context the link
// lookup runs with, otherwise RLS hides every link and no click ever works.
func TestSlackWebhook_InteractionCarriesResolvedTenantToLinkLookup(t *testing.T) {
	tenantID := uuid.New()
	repo := &stubRepo{link: &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New()}}
	h := NewWebhookHandler(nil, testSigningSecret, repo, nil, &stubResolver{tenantID: tenantID}, &spyAcknowledger{})

	body := blockActionBody(t, "T1", "C42", "1699.0001", "kmuhub_acknowledge", uuid.NewString())
	h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

	require.NoError(t, repo.seenTenErr)
	require.Equal(t, tenantID, repo.seenTenant)
}

func TestSlackWebhook_AcknowledgeSurvivesCardRewriteFailure(t *testing.T) {
	f := newFakeSlack(t)
	f.fail("chat.update", "message_not_found")
	acks := &spyAcknowledger{}
	h := NewWebhookHandler(f.client("xoxb-1"), testSigningSecret,
		&flowRepo{link: &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New()}}, nil,
		&stubResolver{tenantID: uuid.New()}, acks)

	body := blockActionBody(t, "T1", "C42", "1699.0001", "kmuhub_acknowledge", uuid.NewString())
	resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 1, acks.calls, "the notification is read even if the card cannot be rewritten")
}

func TestSlackWebhook_AcknowledgeFailureTellsTheUserAndKeepsTheCard(t *testing.T) {
	tests := []struct {
		name  string
		value func() string
		acks  integration.NotificationAcknowledger
	}{
		{"service error", uuid.NewString, &failingAcknowledger{}},
		{"notification id is not a uuid", func() string { return "not-a-uuid" }, &spyAcknowledger{}},
		{"notification service not wired", uuid.NewString, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeSlack(t)
			h := NewWebhookHandler(f.client("xoxb-1"), testSigningSecret,
				&flowRepo{link: &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New()}}, nil,
				&stubResolver{tenantID: uuid.New()}, tc.acks)

			body := blockActionBody(t, "T1", "C42", "1699.0001", "kmuhub_acknowledge", tc.value())
			resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

			require.Equal(t, http.StatusOK, resp.StatusCode)
			eph := f.only(t, "chat.postEphemeral").Form
			require.Equal(t, "C42", eph.Get("channel"))
			require.Equal(t, "U1", eph.Get("user"))
			require.Contains(t, eph.Get("text"), "nicht als gelesen markiert")
			require.Zero(t, countCalls(f, "chat.update"), "a failed action must not rewrite the card")
		})
	}
}

func countCalls(f *fakeSlack, method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if c.Method == method {
			n++
		}
	}
	return n
}

func TestSlackWebhook_InteractionRepliesPrivatelyWhenItCannotAct(t *testing.T) {
	t.Run("workspace not mapped to a tenant", func(t *testing.T) {
		f := newFakeSlack(t)
		h := NewWebhookHandler(f.client("xoxb-1"), testSigningSecret, &flowRepo{}, nil,
			&stubResolver{err: integration.ErrTenantUnresolved}, &spyAcknowledger{})

		body := blockActionBody(t, "T-unknown", "C42", "1699.0001", "kmuhub_acknowledge", uuid.NewString())
		resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, f.only(t, "chat.postEphemeral").Form.Get("text"), "keinem Cosmi-Mandanten zugeordnet")
	})

	t.Run("user has not linked an account", func(t *testing.T) {
		f := newFakeSlack(t)
		acks := &spyAcknowledger{}
		h := NewWebhookHandler(f.client("xoxb-1"), testSigningSecret, &flowRepo{}, nil,
			&stubResolver{tenantID: uuid.New()}, acks)

		body := blockActionBody(t, "T1", "C42", "1699.0001", "kmuhub_acknowledge", uuid.NewString())
		resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, f.only(t, "chat.postEphemeral").Form.Get("text"), "/kmuhub link")
		require.Zero(t, acks.calls)
	})

	t.Run("action Cosmi does not execute", func(t *testing.T) {
		f := newFakeSlack(t)
		acks := &spyAcknowledger{}
		h := NewWebhookHandler(f.client("xoxb-1"), testSigningSecret,
			&flowRepo{link: &integration.AccountLink{ID: uuid.New(), KMUHubUserID: uuid.New()}}, nil,
			&stubResolver{tenantID: uuid.New()}, acks)

		body := blockActionBody(t, "T1", "C42", "1699.0001", "kmuhub_approve", uuid.NewString())
		resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, f.only(t, "chat.postEphemeral").Form.Get("text"), "noch nicht ausgeführt")
		require.Zero(t, acks.calls)
		require.Zero(t, countCalls(f, "chat.update"))
	})

	t.Run("no channel to answer in", func(t *testing.T) {
		f := newFakeSlack(t)
		h := NewWebhookHandler(f.client("xoxb-1"), testSigningSecret, &flowRepo{}, nil,
			&stubResolver{tenantID: uuid.New()}, &spyAcknowledger{})

		body := blockActionBody(t, "T1", "", "", "kmuhub_acknowledge", uuid.NewString())
		resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Zero(t, f.count())
	})

	t.Run("no Slack client wired", func(t *testing.T) {
		h := NewWebhookHandler(nil, testSigningSecret, &flowRepo{}, nil,
			&stubResolver{tenantID: uuid.New()}, &spyAcknowledger{})

		body := blockActionBody(t, "T1", "C42", "1699.0001", "kmuhub_acknowledge", uuid.NewString())
		resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("ephemeral delivery fails", func(t *testing.T) {
		f := newFakeSlack(t)
		f.fail("chat.postEphemeral", "channel_not_found")
		h := NewWebhookHandler(f.client("xoxb-1"), testSigningSecret, &flowRepo{}, nil,
			&stubResolver{tenantID: uuid.New()}, &spyAcknowledger{})

		body := blockActionBody(t, "T1", "C42", "1699.0001", "kmuhub_acknowledge", uuid.NewString())
		resp := h.Process(context.Background(), signedRequest(t, integration.WebhookKindSlackInteraction, body))

		require.Equal(t, http.StatusOK, resp.StatusCode, "Slack must still get its 200 within three seconds")
	})
}

func TestSlackWebhook_RefusesBadRequests(t *testing.T) {
	h := NewWebhookHandler(nil, testSigningSecret, &flowRepo{}, nil,
		&stubResolver{tenantID: uuid.New()}, &spyAcknowledger{})

	tests := []struct {
		name string
		kind string
		body string
		want int
	}{
		{"interaction without payload", integration.WebhookKindSlackInteraction, "team_id=T1", http.StatusBadRequest},
		{"interaction with broken json", integration.WebhookKindSlackInteraction, "payload=%7Bnope", http.StatusBadRequest},
		{"interaction of another type", integration.WebhookKindSlackInteraction,
			"payload=" + urlEncode(`{"type":"view_submission"}`), http.StatusOK},
		{"block_actions without actions", integration.WebhookKindSlackInteraction,
			"payload=" + urlEncode(`{"type":"block_actions"}`), http.StatusOK},
		{"unknown webhook kind", "slack_other", "a=b", http.StatusBadRequest},
		{"undecodable form body", integration.WebhookKindSlackCommand, "a=%zz", http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := h.Process(context.Background(), signedRequest(t, tc.kind, tc.body))
			require.Equal(t, tc.want, resp.StatusCode)
		})
	}
}

// A handler without a signing secret must not fall back to an empty HMAC key:
// anyone could then sign a request with that key.
func TestSlackWebhook_RefusesEverythingWithoutSigningSecret(t *testing.T) {
	h := NewWebhookHandler(nil, "", &flowRepo{}, nil, &stubResolver{tenantID: uuid.New()}, nil)

	body := "team_id=T1&user_id=U1&text=link"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, nil)
	mac.Write([]byte("v0:" + timestamp + ":" + body))
	resp := h.Process(context.Background(), integration.WebhookRequest{
		Kind: integration.WebhookKindSlackCommand,
		Body: []byte(body),
		Headers: map[string]string{
			"X-Slack-Request-Timestamp": timestamp,
			"X-Slack-Signature":         "v0=" + hex.EncodeToString(mac.Sum(nil)),
		},
	})

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestSlackWebhook_RefusesRequestWithoutSignatureHeaders(t *testing.T) {
	h := NewWebhookHandler(nil, testSigningSecret, &flowRepo{}, nil, &stubResolver{tenantID: uuid.New()}, nil)

	resp := h.Process(context.Background(), integration.WebhookRequest{
		Kind: integration.WebhookKindSlackCommand,
		Body: []byte("team_id=T1&user_id=U1&text=link"),
	})

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ============================================================================
// Slash commands
// ============================================================================

func TestSlackSlash_LinkIssuesAHashedFiveMinuteToken(t *testing.T) {
	repo := &flowRepo{}
	h := NewWebhookHandler(nil, testSigningSecret, repo, integration.NewAccountLinkService(repo),
		&stubResolver{tenantID: uuid.New()}, nil)

	resp := h.Process(context.Background(),
		signedRequest(t, integration.WebhookKindSlackCommand, "team_id=T1&user_id=U9&text=link"))

	text := ephemeralText(t, resp)
	require.Len(t, repo.tokens, 1)
	stored := repo.tokens[0]
	require.Equal(t, integration.PlatformSlack, stored.Platform)
	require.Equal(t, "U9", stored.ExternalUserID)
	require.False(t, stored.Used)

	// The reply carries the raw token; the repository only ever sees its hash.
	require.NotContains(t, text, stored.TokenHash)
	start := strings.Index(text, "`")
	end := strings.Index(text[start+1:], "`")
	raw := text[start+1 : start+1+end]
	require.Len(t, raw, 64)
	require.NotEqual(t, raw, stored.TokenHash)
	require.Contains(t, text, "5 Minuten")
}

func TestSlackSlash_LinkReportsStorageFailureWithoutLeakingAToken(t *testing.T) {
	repo := &flowRepo{tokenErr: errors.New("db down")}
	h := NewWebhookHandler(nil, testSigningSecret, repo, integration.NewAccountLinkService(repo),
		&stubResolver{tenantID: uuid.New()}, nil)

	resp := h.Process(context.Background(),
		signedRequest(t, integration.WebhookKindSlackCommand, "team_id=T1&user_id=U9&text=link"))

	text := ephemeralText(t, resp)
	require.Contains(t, text, "Fehler beim Erstellen des Verknüpfungstokens")
	require.NotContains(t, text, "`")
}

func TestSlackSlash_LinkForUnmappedWorkspaceIssuesNothing(t *testing.T) {
	repo := &flowRepo{}
	h := NewWebhookHandler(nil, testSigningSecret, repo, integration.NewAccountLinkService(repo),
		&stubResolver{err: integration.ErrTenantUnresolved}, nil)

	resp := h.Process(context.Background(),
		signedRequest(t, integration.WebhookKindSlackCommand, "team_id=T-unknown&user_id=U9&text=link"))

	require.Contains(t, ephemeralText(t, resp), "keinem Cosmi-Mandanten zugeordnet")
	require.Empty(t, repo.tokens)
}

func TestSlackSlash_Unlink(t *testing.T) {
	t.Run("removes the link", func(t *testing.T) {
		linkID := uuid.New()
		repo := &flowRepo{link: &integration.AccountLink{ID: linkID}}
		h := NewWebhookHandler(nil, testSigningSecret, repo, nil, &stubResolver{tenantID: uuid.New()}, nil)

		resp := h.Process(context.Background(),
			signedRequest(t, integration.WebhookKindSlackCommand, "team_id=T1&user_id=U9&text=unlink"))

		require.Contains(t, ephemeralText(t, resp), "erfolgreich aufgehoben")
		require.Equal(t, []uuid.UUID{linkID}, repo.deleted)
	})

	t.Run("reports a failed delete", func(t *testing.T) {
		repo := &flowRepo{link: &integration.AccountLink{ID: uuid.New()}, deleteErr: errors.New("db down")}
		h := NewWebhookHandler(nil, testSigningSecret, repo, nil, &stubResolver{tenantID: uuid.New()}, nil)

		resp := h.Process(context.Background(),
			signedRequest(t, integration.WebhookKindSlackCommand, "team_id=T1&user_id=U9&text=unlink"))

		require.Contains(t, ephemeralText(t, resp), "Fehler beim Aufheben")
		require.Empty(t, repo.deleted)
	})

	t.Run("text is trimmed before matching", func(t *testing.T) {
		repo := &flowRepo{link: &integration.AccountLink{ID: uuid.New()}}
		h := NewWebhookHandler(nil, testSigningSecret, repo, nil, &stubResolver{tenantID: uuid.New()}, nil)

		resp := h.Process(context.Background(),
			signedRequest(t, integration.WebhookKindSlackCommand, "team_id=T1&user_id=U9&text=+unlink+"))

		require.Contains(t, ephemeralText(t, resp), "erfolgreich aufgehoben")
	})

	t.Run("unmapped workspace deletes nothing", func(t *testing.T) {
		repo := &flowRepo{link: &integration.AccountLink{ID: uuid.New()}}
		h := NewWebhookHandler(nil, testSigningSecret, repo, nil,
			&stubResolver{err: integration.ErrTenantUnresolved}, nil)

		resp := h.Process(context.Background(),
			signedRequest(t, integration.WebhookKindSlackCommand, "team_id=T-unknown&user_id=U9&text=unlink"))

		require.Contains(t, ephemeralText(t, resp), "keinem Cosmi-Mandanten zugeordnet")
		require.Empty(t, repo.deleted)
	})
}

// ============================================================================
// Helpers
// ============================================================================

func TestExtractActionType(t *testing.T) {
	require.Equal(t, "acknowledge", extractActionType("kmuhub_acknowledge"))
	require.Equal(t, "kmuhub_", extractActionType("kmuhub_"), "a bare prefix is returned as is")
	require.Equal(t, "other", extractActionType("other"))
	require.Equal(t, "", extractActionType(""))
}

func TestBuildResolvedBlocks_ShowsOriginalTextAndActor(t *testing.T) {
	blocks := marshalBlocks(t, buildResolvedBlocks("Rechnung fällig", "acknowledge", "Anna"))

	require.Len(t, blocks, 2)
	require.Equal(t, "section", blocks[0]["type"])
	require.Equal(t, "Rechnung fällig", blocks[0]["text"].(map[string]any)["text"])
	require.Equal(t, "context", blocks[1]["type"])
	require.Contains(t, blocks[1]["elements"].([]any)[0].(map[string]any)["text"], "Anna")
}

func TestBuildUpdatedBlocks_ReplacesButtonsWithBanner(t *testing.T) {
	body := "Details"
	notif := testNotification("crm", &body)

	blocks := marshalBlocks(t, BuildUpdatedBlocks(notif, "acknowledge", "Anna"))

	require.Len(t, blocks, 3)
	require.Equal(t, "context", blocks[2]["type"])
	require.Contains(t, blocks[2]["elements"].([]any)[0].(map[string]any)["text"], "Anna")
	raw, err := json.Marshal(blocks)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "kmuhub_acknowledge")

	empty := ""
	require.Len(t, BuildUpdatedBlocks(testNotification("", &empty), "acknowledge", "Anna"), 2,
		"an empty body adds no section")
}
