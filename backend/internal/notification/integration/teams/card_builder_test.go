package teams

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

func TestBuildCardTitleUsesTheModuleIcon(t *testing.T) {
	n := &models.Notification{ID: uuid.New(), ModuleID: "calendar", Title: "Meeting"}

	card := BuildCard(n, nil)

	title := card.Body[0].Items[0]
	require.Equal(t, ModuleIcons["calendar"]+" Meeting", title.Text)
	require.Equal(t, "bolder", title.Weight)
}

func TestBuildCardFallsBackToTheBellForUnknownModules(t *testing.T) {
	n := &models.Notification{ID: uuid.New(), ModuleID: "no-such-module", Title: "Hi"}

	card := BuildCard(n, nil)

	require.Equal(t, ModuleIcons["system"]+" Hi", card.Body[0].Items[0].Text)
}

func TestBuildCardBodyBlockOnlyWhenBodyIsNonEmpty(t *testing.T) {
	empty := ""
	full := "Details"

	none := BuildCard(&models.Notification{ID: uuid.New(), Title: "t"}, nil)
	blank := BuildCard(&models.Notification{ID: uuid.New(), Title: "t", Body: &empty}, nil)
	with := BuildCard(&models.Notification{ID: uuid.New(), Title: "t", Body: &full}, nil)

	require.Len(t, none.Body[0].Items, 1)
	require.Len(t, blank.Body[0].Items, 1)
	require.Len(t, with.Body[0].Items, 2)
	require.Equal(t, "Details", with.Body[0].Items[1].Text)
}

func TestBuildCardEnvelope(t *testing.T) {
	card := BuildCard(&models.Notification{ID: uuid.New(), Title: "t"}, nil)

	require.Equal(t, "AdaptiveCard", card.Type)
	require.Equal(t, "1.4", card.Version)
	require.Equal(t, "http://adaptivecards.io/schemas/adaptive-card.json", card.Schema)
	require.Len(t, card.Body, 1)
	require.Equal(t, "Container", card.Body[0].Type)
	require.Equal(t, "accent", card.Body[0].Style)
	require.Empty(t, card.Actions)
}

// The button payload is what the webhook reads back on click: the verb, the
// action_type and the notification id must all round-trip through the JSON.
func TestBuildCardActionsCarryTheClickPayload(t *testing.T) {
	n := &models.Notification{ID: uuid.New(), Title: "t"}

	card := BuildCard(n, []integration.ActionType{
		integration.ActionAcknowledge, integration.ActionApprove, integration.ActionReject, integration.ActionReply,
	})

	raw, err := json.Marshal(card)
	require.NoError(t, err)
	var back AdaptiveCard
	require.NoError(t, json.Unmarshal(raw, &back))

	require.Len(t, back.Actions, 4)
	for i, want := range []string{"acknowledge", "approve", "reject", "reply"} {
		a := back.Actions[i]
		require.Equal(t, "Action.Execute", a.Type)
		require.Equal(t, want, a.Verb)
		require.Equal(t, want, a.Data["action_type"])
		require.Equal(t, n.ID.String(), a.Data["notification_id"])
	}
}

func TestActionButtonConfig(t *testing.T) {
	tests := []struct {
		action    integration.ActionType
		wantLabel string
		wantStyle string
	}{
		{integration.ActionAcknowledge, "✔ Gelesen", "default"},
		{integration.ActionReply, "\U0001F4AC Antworten", "default"},
		{integration.ActionApprove, "✅ Genehmigen", "positive"},
		{integration.ActionReject, "❌ Ablehnen", "destructive"},
		{integration.ActionType("custom"), "custom", "default"},
	}
	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			label, style := actionButtonConfig(tt.action)
			require.Equal(t, tt.wantLabel, label)
			require.Equal(t, tt.wantStyle, style)
		})
	}
}

func TestBuildUpdatedCardDropsButtonsAndAppendsTheBanner(t *testing.T) {
	body := "Details"
	n := &models.Notification{ID: uuid.New(), ModuleID: "hr", Title: "Leave request", Body: &body}

	card := BuildUpdatedCard(n, string(integration.ActionApprove), "Anna")

	require.Empty(t, card.Actions)
	require.Len(t, card.Body, 3, "title, body and banner")
	require.Equal(t, ModuleIcons["hr"]+" Leave request", card.Body[0].Text)
	require.Equal(t, "Details", card.Body[1].Text)
	banner := card.Body[2]
	require.Equal(t, "✅ Genehmigt von Anna", banner.Text)
	require.True(t, banner.IsSubtle)
	require.Equal(t, "small", banner.Size)
}

func TestBuildUpdatedCardWithoutBodyAndUnknownModule(t *testing.T) {
	n := &models.Notification{ID: uuid.New(), ModuleID: "no-such-module", Title: "Hi"}

	card := BuildUpdatedCard(n, string(integration.ActionAcknowledge), "Max")

	require.Len(t, card.Body, 2, "title and banner")
	require.Equal(t, ModuleIcons["system"]+" Hi", card.Body[0].Text)
	require.Equal(t, "✔ Gelesen von Max", card.Body[1].Text)
}

func TestResolvedBannerText(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{"acknowledge", "✔ Gelesen von Eva"},
		{"approve", "✅ Genehmigt von Eva"},
		{"reject", "❌ Abgelehnt von Eva"},
		{"reply", "\U0001F4AC Beantwortet von Eva"},
		{"something-else", "Verarbeitet von Eva"},
	}
	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			require.Equal(t, tt.want, resolvedBannerText(tt.action, "Eva"))
		})
	}
}
