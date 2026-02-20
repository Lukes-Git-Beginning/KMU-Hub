package teams

import (
	"encoding/json"
	"fmt"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// ModuleIcons maps module IDs to emoji icons for card display.
var ModuleIcons = map[string]string{
	"crm":      "\U0001F4BC", // Briefcase
	"work":     "\U0001F4CB", // Clipboard
	"hr":       "\U0001F465", // People
	"biz":      "\U0001F9FE", // Receipt
	"email":    "\U00002709", // Envelope
	"chat":     "\U0001F4AC", // Speech Balloon
	"calendar": "\U0001F4C5", // Calendar
	"document": "\U0001F4C4", // Document
	"system":   "\U0001F514", // Bell
}

// AdaptiveCard represents an Adaptive Card JSON structure.
type AdaptiveCard struct {
	Schema  string          `json:"$schema"`
	Type    string          `json:"type"`
	Version string          `json:"version"`
	Body    []CardElement   `json:"body"`
	Actions []CardAction    `json:"actions,omitempty"`
	Meta    json.RawMessage `json:"metadata,omitempty"`
}

// CardElement represents an element in the card body.
type CardElement struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	Wrap     bool          `json:"wrap,omitempty"`
	Weight   string        `json:"weight,omitempty"`
	Size     string        `json:"size,omitempty"`
	IsSubtle bool          `json:"isSubtle,omitempty"`
	Style    string        `json:"style,omitempty"`
	Items    []CardElement `json:"items,omitempty"`
}

// CardAction represents an Action.Execute button on the card.
type CardAction struct {
	Type  string                 `json:"type"`
	Title string                 `json:"title"`
	Verb  string                 `json:"verb,omitempty"`
	Data  map[string]interface{} `json:"data,omitempty"`
	Style string                 `json:"style,omitempty"`
}

// BuildCard creates an Adaptive Card for a KMU Hub notification.
func BuildCard(notif *models.Notification, actions []integration.ActionType) AdaptiveCard {
	icon := ModuleIcons[notif.ModuleID]
	if icon == "" {
		icon = "\U0001F514" // Default: bell
	}

	titleText := fmt.Sprintf("%s %s", icon, notif.Title)

	// Build body elements
	bodyElements := []CardElement{
		{
			Type:   "TextBlock",
			Text:   titleText,
			Wrap:   true,
			Weight: "bolder",
			Size:   "medium",
		},
	}

	if notif.Body != nil && *notif.Body != "" {
		bodyElements = append(bodyElements, CardElement{
			Type: "TextBlock",
			Text: *notif.Body,
			Wrap: true,
			Size: "default",
		})
	}

	// Wrap in accent container
	container := CardElement{
		Type:  "Container",
		Style: "accent",
		Items: bodyElements,
	}

	// Build action buttons
	var cardActions []CardAction
	for _, actionType := range actions {
		label, style := actionButtonConfig(actionType)
		cardActions = append(cardActions, CardAction{
			Type:  "Action.Execute",
			Title: label,
			Verb:  string(actionType),
			Data: map[string]interface{}{
				"action_type":     string(actionType),
				"notification_id": notif.ID.String(),
			},
			Style: style,
		})
	}

	return AdaptiveCard{
		Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
		Type:    "AdaptiveCard",
		Version: "1.4",
		Body:    []CardElement{container},
		Actions: cardActions,
	}
}

// BuildUpdatedCard creates an Adaptive Card with a resolved action banner.
// Action buttons are removed; the card shows who performed what action.
func BuildUpdatedCard(notif *models.Notification, actionTaken, actorName string) AdaptiveCard {
	icon := ModuleIcons[notif.ModuleID]
	if icon == "" {
		icon = "\U0001F514"
	}

	titleText := fmt.Sprintf("%s %s", icon, notif.Title)

	bodyElements := []CardElement{
		{
			Type:   "TextBlock",
			Text:   titleText,
			Wrap:   true,
			Weight: "bolder",
			Size:   "medium",
		},
	}

	if notif.Body != nil && *notif.Body != "" {
		bodyElements = append(bodyElements, CardElement{
			Type: "TextBlock",
			Text: *notif.Body,
			Wrap: true,
			Size: "default",
		})
	}

	// Resolved action banner
	bannerText := resolvedBannerText(actionTaken, actorName)
	bodyElements = append(bodyElements, CardElement{
		Type:     "TextBlock",
		Text:     bannerText,
		Wrap:     false,
		Size:     "small",
		IsSubtle: true,
	})

	return AdaptiveCard{
		Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
		Type:    "AdaptiveCard",
		Version: "1.4",
		Body:    bodyElements,
		// No action buttons on resolved card
	}
}

// actionButtonConfig returns the label and style for an action button.
func actionButtonConfig(actionType integration.ActionType) (string, string) {
	switch actionType {
	case integration.ActionAcknowledge:
		return "\u2714 Gelesen", "default"
	case integration.ActionReply:
		return "\U0001F4AC Antworten", "default"
	case integration.ActionApprove:
		return "\u2705 Genehmigen", "positive"
	case integration.ActionReject:
		return "\u274C Ablehnen", "destructive"
	default:
		return string(actionType), "default"
	}
}

// resolvedBannerText generates the resolved action text.
func resolvedBannerText(actionTaken, actorName string) string {
	switch actionTaken {
	case string(integration.ActionAcknowledge):
		return fmt.Sprintf("\u2714 Gelesen von %s", actorName)
	case string(integration.ActionApprove):
		return fmt.Sprintf("\u2705 Genehmigt von %s", actorName)
	case string(integration.ActionReject):
		return fmt.Sprintf("\u274C Abgelehnt von %s", actorName)
	case string(integration.ActionReply):
		return fmt.Sprintf("\U0001F4AC Beantwortet von %s", actorName)
	default:
		return fmt.Sprintf("Verarbeitet von %s", actorName)
	}
}
