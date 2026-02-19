package routing

import (
	"fmt"
	"strings"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Evaluate checks whether the condition tree matches the given inbox message.
// - If c.And is set: ALL sub-conditions must be true (short-circuit on first false).
//   Empty AND is vacuous truth (returns true).
// - If c.Or is set: ANY sub-condition must be true (short-circuit on first true).
//   Empty OR returns false.
// - If neither And nor Or: evaluate as leaf condition.
func (c *Condition) Evaluate(msg *models.InboxMessage) bool {
	if c.And != nil {
		for i := range c.And {
			if !c.And[i].Evaluate(msg) {
				return false
			}
		}
		return true
	}

	if c.Or != nil {
		for i := range c.Or {
			if c.Or[i].Evaluate(msg) {
				return true
			}
		}
		return false
	}

	return evaluateLeaf(c, msg)
}

// Condition mirrors models.Condition but is local to the routing package
// to enable methods. It can be constructed from models.Condition via ParseCondition.
type Condition struct {
	And      []Condition `json:"and,omitempty"`
	Or       []Condition `json:"or,omitempty"`
	Field    string      `json:"field,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Value    interface{} `json:"value,omitempty"`
}

// ParseCondition converts a models.Condition to a routing.Condition.
func ParseCondition(mc models.Condition) Condition {
	c := Condition{
		Field:    mc.Field,
		Operator: mc.Operator,
		Value:    mc.Value,
	}

	if len(mc.And) > 0 {
		c.And = make([]Condition, len(mc.And))
		for i, sub := range mc.And {
			c.And[i] = ParseCondition(sub)
		}
	}

	if len(mc.Or) > 0 {
		c.Or = make([]Condition, len(mc.Or))
		for i, sub := range mc.Or {
			c.Or[i] = ParseCondition(sub)
		}
	}

	return c
}

// evaluateLeaf evaluates a single leaf condition against a message.
func evaluateLeaf(c *Condition, msg *models.InboxMessage) bool {
	fieldValue := extractField(c.Field, msg)
	condValue := fmt.Sprintf("%v", c.Value)

	switch c.Operator {
	case "equals":
		return strings.EqualFold(fieldValue, condValue)

	case "not_equals":
		return !strings.EqualFold(fieldValue, condValue)

	case "contains":
		return strings.Contains(strings.ToLower(fieldValue), strings.ToLower(condValue))

	case "not_contains":
		return !strings.Contains(strings.ToLower(fieldValue), strings.ToLower(condValue))

	case "starts_with":
		return strings.HasPrefix(strings.ToLower(fieldValue), strings.ToLower(condValue))

	case "ends_with":
		return strings.HasSuffix(strings.ToLower(fieldValue), strings.ToLower(condValue))

	case "in":
		values := strings.Split(condValue, ",")
		lowerField := strings.ToLower(fieldValue)
		for _, v := range values {
			if strings.EqualFold(strings.TrimSpace(v), lowerField) {
				return true
			}
		}
		return false

	case "not_in":
		values := strings.Split(condValue, ",")
		lowerField := strings.ToLower(fieldValue)
		for _, v := range values {
			if strings.EqualFold(strings.TrimSpace(v), lowerField) {
				return false
			}
		}
		return true

	case "exists":
		return fieldValue != ""

	case "not_exists":
		return fieldValue == ""

	default:
		// Unknown operator evaluates to false
		return false
	}
}

// extractField retrieves the value of a named field from an InboxMessage.
func extractField(field string, msg *models.InboxMessage) string {
	switch field {
	case "channel":
		return msg.Channel
	case "sender", "sender_name":
		return msg.SenderName
	case "sender_email":
		if msg.SenderEmail != nil {
			return *msg.SenderEmail
		}
		return ""
	case "subject":
		return msg.Subject
	case "preview", "body":
		return msg.Preview
	case "tags":
		return strings.Join(msg.Tags, ",")
	case "crm_linked":
		if msg.CRMContactID != nil {
			return "true"
		}
		return "false"
	case "team_inbox":
		if msg.TeamInboxID != nil {
			return msg.TeamInboxID.String()
		}
		return ""
	default:
		return ""
	}
}
