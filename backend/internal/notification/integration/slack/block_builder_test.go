package slack

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	slackapi "github.com/slack-go/slack"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

func testNotification(moduleID string, body *string) *models.Notification {
	return &models.Notification{
		ID:        uuid.New(),
		ModuleID:  moduleID,
		Title:     "Neue Rechnung faellig",
		Body:      body,
		CreatedAt: time.Date(2026, 8, 11, 9, 30, 0, 0, time.UTC),
	}
}

// marshalBlocks round-trips through the Slack block-kit JSON so the assertions
// below inspect the shape actually sent over the wire, not internal Go fields.
func marshalBlocks(t *testing.T, blocks []slackapi.Block) []map[string]any {
	t.Helper()
	raw, err := json.Marshal(blocks)
	require.NoError(t, err)
	var decoded []map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	return decoded
}

// ============================================================================
// BuildBlocks
// ============================================================================

func TestBuildBlocks_KnownModuleUsesIconAndName(t *testing.T) {
	body := "Rechnung 2026-001 ist ueberfaellig."
	notif := testNotification("biz", &body)

	blocks := BuildBlocks(notif, nil)
	decoded := marshalBlocks(t, blocks)

	// header, body, context -- no actions block when actions is nil.
	require.Len(t, decoded, 3)
	header := decoded[0]["text"].(map[string]any)["text"].(string)
	require.Contains(t, header, ":receipt:", "biz module must use its configured icon")
	require.Contains(t, header, "Neue Rechnung faellig")

	bodyBlock := decoded[1]["text"].(map[string]any)["text"].(string)
	require.Equal(t, body, bodyBlock)

	context := decoded[2]["elements"].([]any)[0].(map[string]any)["text"].(string)
	require.Contains(t, context, "Finanzen", "biz must render its German module name")
}

// An unknown module ID must not panic on the missing map lookups -- it falls
// back to the generic bell icon and the raw module ID as the display name.
func TestBuildBlocks_UnknownModuleFallsBackToDefaults(t *testing.T) {
	notif := testNotification("does-not-exist", nil)

	blocks := BuildBlocks(notif, nil)
	decoded := marshalBlocks(t, blocks)

	header := decoded[0]["text"].(map[string]any)["text"].(string)
	require.Contains(t, header, ":bell:")

	// No body block was appended because Body is nil.
	require.Len(t, decoded, 2, "nil body must not produce an empty section block")

	context := decoded[1]["elements"].([]any)[0].(map[string]any)["text"].(string)
	require.Contains(t, context, "does-not-exist", "unknown module falls back to the raw module ID")
}

// An empty (non-nil) body pointer must be treated the same as a nil body --
// Slack rejects section blocks with empty text.
func TestBuildBlocks_EmptyBodyStringOmitsBodyBlock(t *testing.T) {
	empty := ""
	notif := testNotification("crm", &empty)

	blocks := BuildBlocks(notif, nil)
	decoded := marshalBlocks(t, blocks)

	require.Len(t, decoded, 2, "empty body string must not produce a body block")
}

func TestBuildBlocks_ActionsAppendActionBlockWithButtons(t *testing.T) {
	notif := testNotification("work", nil)
	actions := []integration.ActionType{integration.ActionAcknowledge, integration.ActionApprove}

	blocks := BuildBlocks(notif, actions)
	decoded := marshalBlocks(t, blocks)

	require.Len(t, decoded, 3, "header + context + one action block")
	actionBlock := decoded[2]
	require.Equal(t, "actions", actionBlock["type"])
	elements := actionBlock["elements"].([]any)
	require.Len(t, elements, 2)

	blockID := actionBlock["block_id"].(string)
	require.Contains(t, blockID, notif.ID.String(), "block_id must carry the notification id for correlation")
}

// ============================================================================
// BuildUpdatedBlocks
// ============================================================================

func TestBuildUpdatedBlocks_ReplacesActionsWithResolvedBanner(t *testing.T) {
	notif := testNotification("hr", nil)

	blocks := BuildUpdatedBlocks(notif, string(integration.ActionApprove), "Erika Muster")
	decoded := marshalBlocks(t, blocks)

	require.Len(t, decoded, 2, "header + resolved banner, never an actions block")
	banner := decoded[1]["elements"].([]any)[0].(map[string]any)["text"].(string)
	require.Contains(t, banner, "Genehmigt von Erika Muster")
}

// ============================================================================
// actionButtonConfig / buildActionButton
// ============================================================================

func TestActionButtonConfig_KnownAndUnknownActions(t *testing.T) {
	cases := []struct {
		action    integration.ActionType
		wantLabel string
		wantStyle string
	}{
		{integration.ActionAcknowledge, "Gelesen", "primary"},
		{integration.ActionReply, "Antworten", ""},
		{integration.ActionApprove, "Genehmigen", "primary"},
		{integration.ActionReject, "Ablehnen", "danger"},
		{integration.ActionType("unknown"), "unknown", ""},
	}
	for _, tc := range cases {
		label, style := actionButtonConfig(tc.action)
		require.Equal(t, tc.wantLabel, label, "action %s", tc.action)
		require.Equal(t, tc.wantStyle, style, "action %s", tc.action)
	}
}

func TestBuildActionButton_StylesMapToSlackConstants(t *testing.T) {
	btn := buildActionButton(integration.ActionReject, "notif-1")
	buttonEl, ok := btn.(*slackapi.ButtonBlockElement)
	require.True(t, ok)
	require.Equal(t, slackapi.StyleDanger, buttonEl.Style)
	require.Equal(t, "notif-1", buttonEl.Value)

	// Reply has no configured style -- must stay the Slack zero value, not
	// accidentally inherit primary/danger from a previous case.
	replyBtn := buildActionButton(integration.ActionReply, "notif-2")
	replyEl := replyBtn.(*slackapi.ButtonBlockElement)
	require.Equal(t, slackapi.Style(""), replyEl.Style)
}

// ============================================================================
// resolvedBannerText
// ============================================================================

func TestResolvedBannerText_AllActionsAndUnknownFallback(t *testing.T) {
	cases := map[string]string{
		string(integration.ActionAcknowledge): "Gelesen von Max",
		string(integration.ActionApprove):     "Genehmigt von Max",
		string(integration.ActionReject):      "Abgelehnt von Max",
		string(integration.ActionReply):       "Beantwortet von Max",
		"some_future_action":                  "Verarbeitet von Max",
	}
	for actionTaken, want := range cases {
		got := resolvedBannerText(actionTaken, "Max")
		require.Contains(t, got, want, "actionTaken=%s", actionTaken)
	}
}
