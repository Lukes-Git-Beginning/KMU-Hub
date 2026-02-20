package slack

import (
	"fmt"
	"time"

	slackapi "github.com/slack-go/slack"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// ModuleIcons maps module IDs to emoji identifiers for Slack formatting.
var ModuleIcons = map[string]string{
	"crm":      ":briefcase:",
	"work":     ":clipboard:",
	"hr":       ":busts_in_silhouette:",
	"biz":      ":receipt:",
	"email":    ":envelope:",
	"chat":     ":speech_balloon:",
	"calendar": ":calendar:",
	"document": ":page_facing_up:",
	"system":   ":bell:",
}

// ModuleNames maps module IDs to human-readable German names.
var ModuleNames = map[string]string{
	"crm":      "CRM",
	"work":     "Aufgaben",
	"hr":       "Personal",
	"biz":      "Finanzen",
	"email":    "E-Mail",
	"chat":     "Chat",
	"calendar": "Kalender",
	"document": "Dokumente",
	"system":   "System",
}

// BuildBlocks creates Slack Block Kit blocks for a KMU Hub notification.
func BuildBlocks(notif *models.Notification, actions []integration.ActionType) []slackapi.Block {
	icon := ModuleIcons[notif.ModuleID]
	if icon == "" {
		icon = ":bell:"
	}

	// Header section with module icon + bold title
	headerText := fmt.Sprintf("%s *%s*", icon, notif.Title)
	blocks := []slackapi.Block{
		slackapi.NewSectionBlock(
			slackapi.NewTextBlockObject("mrkdwn", headerText, false, false),
			nil,
			nil,
		),
	}

	// Body section (if present)
	if notif.Body != nil && *notif.Body != "" {
		blocks = append(blocks, slackapi.NewSectionBlock(
			slackapi.NewTextBlockObject("mrkdwn", *notif.Body, false, false),
			nil,
			nil,
		))
	}

	// Context block with module name + timestamp
	moduleName := ModuleNames[notif.ModuleID]
	if moduleName == "" {
		moduleName = notif.ModuleID
	}
	contextText := fmt.Sprintf("%s | %s", moduleName, notif.CreatedAt.Format(time.RFC822))
	blocks = append(blocks, slackapi.NewContextBlock(
		"",
		slackapi.NewTextBlockObject("mrkdwn", contextText, false, false),
	))

	// Actions block with buttons
	if len(actions) > 0 {
		var btnElements []slackapi.BlockElement
		for _, actionType := range actions {
			btn := buildActionButton(actionType, notif.ID.String())
			btnElements = append(btnElements, btn)
		}

		actionBlockID := fmt.Sprintf("notif_%s", notif.ID.String())
		blocks = append(blocks, slackapi.NewActionBlock(actionBlockID, btnElements...))
	}

	return blocks
}

// BuildUpdatedBlocks creates Slack Block Kit blocks with a resolved action banner.
// Action buttons are removed; replaced with a context block showing who acted.
func BuildUpdatedBlocks(notif *models.Notification, actionTaken, actorName string) []slackapi.Block {
	icon := ModuleIcons[notif.ModuleID]
	if icon == "" {
		icon = ":bell:"
	}

	headerText := fmt.Sprintf("%s *%s*", icon, notif.Title)
	blocks := []slackapi.Block{
		slackapi.NewSectionBlock(
			slackapi.NewTextBlockObject("mrkdwn", headerText, false, false),
			nil,
			nil,
		),
	}

	if notif.Body != nil && *notif.Body != "" {
		blocks = append(blocks, slackapi.NewSectionBlock(
			slackapi.NewTextBlockObject("mrkdwn", *notif.Body, false, false),
			nil,
			nil,
		))
	}

	// Resolved action banner instead of action buttons
	bannerText := resolvedBannerText(actionTaken, actorName)
	blocks = append(blocks, slackapi.NewContextBlock(
		"",
		slackapi.NewTextBlockObject("mrkdwn", bannerText, false, false),
	))

	return blocks
}

// buildActionButton creates a Slack button element for an action type.
func buildActionButton(actionType integration.ActionType, notificationID string) slackapi.BlockElement {
	label, style := actionButtonConfig(actionType)
	actionID := fmt.Sprintf("kmuhub_%s", string(actionType))

	btnText := slackapi.NewTextBlockObject("plain_text", label, true, false)
	btn := slackapi.NewButtonBlockElement(actionID, notificationID, btnText)

	switch style {
	case "primary":
		btn.Style = slackapi.StylePrimary
	case "danger":
		btn.Style = slackapi.StyleDanger
	}

	return btn
}

// actionButtonConfig returns the label and style for an action button.
func actionButtonConfig(actionType integration.ActionType) (string, string) {
	switch actionType {
	case integration.ActionAcknowledge:
		return "Gelesen", "primary"
	case integration.ActionReply:
		return "Antworten", ""
	case integration.ActionApprove:
		return "Genehmigen", "primary"
	case integration.ActionReject:
		return "Ablehnen", "danger"
	default:
		return string(actionType), ""
	}
}

// resolvedBannerText generates the resolved action text for Slack.
func resolvedBannerText(actionTaken, actorName string) string {
	switch actionTaken {
	case string(integration.ActionAcknowledge):
		return fmt.Sprintf(":white_check_mark: Gelesen von %s", actorName)
	case string(integration.ActionApprove):
		return fmt.Sprintf(":white_check_mark: Genehmigt von %s", actorName)
	case string(integration.ActionReject):
		return fmt.Sprintf(":x: Abgelehnt von %s", actorName)
	case string(integration.ActionReply):
		return fmt.Sprintf(":speech_balloon: Beantwortet von %s", actorName)
	default:
		return fmt.Sprintf("Verarbeitet von %s", actorName)
	}
}
