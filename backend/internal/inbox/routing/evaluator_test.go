package routing

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

func newTestMessage() *models.InboxMessage {
	senderEmail := "alice@firma.de"
	senderID := uuid.New()
	crmID := uuid.New()
	teamID := uuid.New()

	return &models.InboxMessage{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		Channel:      "email",
		SourceID:     "thread-123",
		SenderName:   "Alice Mueller",
		SenderID:     &senderID,
		SenderEmail:  &senderEmail,
		Subject:      "Dringende Anfrage zum Projekt",
		Preview:      "Hallo, ich habe eine dringende Anfrage zum Projekt Alpha.",
		Tags:         []string{"support", "vip"},
		CRMContactID: &crmID,
		TeamInboxID:  &teamID,
		ReceivedAt:   time.Now(),
	}
}

func TestEvaluateLeaf_Equals(t *testing.T) {
	msg := newTestMessage()

	tests := []struct {
		name     string
		field    string
		value    string
		expected bool
	}{
		{"exact match", "channel", "email", true},
		{"case insensitive", "channel", "EMAIL", true},
		{"mismatch", "channel", "chat", false},
		{"sender name exact", "sender_name", "Alice Mueller", true},
		{"sender name case", "sender_name", "alice mueller", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Condition{Field: tt.field, Operator: "equals", Value: tt.value}
			if got := c.Evaluate(msg); got != tt.expected {
				t.Errorf("Evaluate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEvaluateLeaf_NotEquals(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "channel", Operator: "not_equals", Value: "chat"}
	if !c.Evaluate(msg) {
		t.Error("expected true for not_equals with different value")
	}

	c = &Condition{Field: "channel", Operator: "not_equals", Value: "email"}
	if c.Evaluate(msg) {
		t.Error("expected false for not_equals with same value")
	}
}

func TestEvaluateLeaf_Contains(t *testing.T) {
	msg := newTestMessage()

	tests := []struct {
		name     string
		field    string
		value    string
		expected bool
	}{
		{"substring match", "subject", "Anfrage", true},
		{"case insensitive", "subject", "dringende", true},
		{"preview contains", "preview", "Projekt Alpha", true},
		{"no match", "subject", "Rechnung", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Condition{Field: tt.field, Operator: "contains", Value: tt.value}
			if got := c.Evaluate(msg); got != tt.expected {
				t.Errorf("Evaluate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEvaluateLeaf_NotContains(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "subject", Operator: "not_contains", Value: "Rechnung"}
	if !c.Evaluate(msg) {
		t.Error("expected true for not_contains with absent substring")
	}

	c = &Condition{Field: "subject", Operator: "not_contains", Value: "Anfrage"}
	if c.Evaluate(msg) {
		t.Error("expected false for not_contains with present substring")
	}
}

func TestEvaluateLeaf_StartsWith(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "subject", Operator: "starts_with", Value: "Dringende"}
	if !c.Evaluate(msg) {
		t.Error("expected true for starts_with with matching prefix")
	}

	c = &Condition{Field: "subject", Operator: "starts_with", Value: "dringende"}
	if !c.Evaluate(msg) {
		t.Error("expected true for case insensitive starts_with")
	}

	c = &Condition{Field: "subject", Operator: "starts_with", Value: "Anfrage"}
	if c.Evaluate(msg) {
		t.Error("expected false for starts_with with non-matching prefix")
	}
}

func TestEvaluateLeaf_EndsWith(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "subject", Operator: "ends_with", Value: "Projekt"}
	if !c.Evaluate(msg) {
		t.Error("expected true for ends_with with matching suffix")
	}

	c = &Condition{Field: "subject", Operator: "ends_with", Value: "projekt"}
	if !c.Evaluate(msg) {
		t.Error("expected true for case insensitive ends_with")
	}

	c = &Condition{Field: "subject", Operator: "ends_with", Value: "Anfrage"}
	if c.Evaluate(msg) {
		t.Error("expected false for ends_with with non-matching suffix")
	}
}

func TestEvaluateLeaf_In(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "channel", Operator: "in", Value: "email,chat,notification"}
	if !c.Evaluate(msg) {
		t.Error("expected true for in with value in list")
	}

	c = &Condition{Field: "channel", Operator: "in", Value: "chat,notification"}
	if c.Evaluate(msg) {
		t.Error("expected false for in with value not in list")
	}

	// With spaces
	c = &Condition{Field: "channel", Operator: "in", Value: "email , chat , notification"}
	if !c.Evaluate(msg) {
		t.Error("expected true for in with value in list (spaces)")
	}
}

func TestEvaluateLeaf_NotIn(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "channel", Operator: "not_in", Value: "chat,notification"}
	if !c.Evaluate(msg) {
		t.Error("expected true for not_in with value not in list")
	}

	c = &Condition{Field: "channel", Operator: "not_in", Value: "email,chat"}
	if c.Evaluate(msg) {
		t.Error("expected false for not_in with value in list")
	}
}

func TestEvaluateLeaf_Exists(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "sender_email", Operator: "exists"}
	if !c.Evaluate(msg) {
		t.Error("expected true for exists with non-empty field")
	}

	// Message without sender_email
	msgNoEmail := newTestMessage()
	msgNoEmail.SenderEmail = nil
	c = &Condition{Field: "sender_email", Operator: "exists"}
	if c.Evaluate(msgNoEmail) {
		t.Error("expected false for exists with nil pointer field")
	}
}

func TestEvaluateLeaf_NotExists(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "sender_email", Operator: "not_exists"}
	if c.Evaluate(msg) {
		t.Error("expected false for not_exists with non-empty field")
	}

	// Message without team_inbox
	msgNoTeam := newTestMessage()
	msgNoTeam.TeamInboxID = nil
	c = &Condition{Field: "team_inbox", Operator: "not_exists"}
	if !c.Evaluate(msgNoTeam) {
		t.Error("expected true for not_exists with nil team inbox")
	}
}

func TestEvaluateAnd_AllTrue(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{
		And: []Condition{
			{Field: "channel", Operator: "equals", Value: "email"},
			{Field: "sender_email", Operator: "contains", Value: "@firma.de"},
		},
	}
	if !c.Evaluate(msg) {
		t.Error("expected true when all AND conditions are true")
	}
}

func TestEvaluateAnd_OneFalse(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{
		And: []Condition{
			{Field: "channel", Operator: "equals", Value: "email"},
			{Field: "channel", Operator: "equals", Value: "chat"}, // false
		},
	}
	if c.Evaluate(msg) {
		t.Error("expected false when one AND condition is false")
	}
}

func TestEvaluateAnd_Empty(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{And: []Condition{}}
	if !c.Evaluate(msg) {
		t.Error("expected true for empty AND (vacuous truth)")
	}
}

func TestEvaluateOr_OneTrue(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{
		Or: []Condition{
			{Field: "channel", Operator: "equals", Value: "chat"},  // false
			{Field: "channel", Operator: "equals", Value: "email"}, // true
		},
	}
	if !c.Evaluate(msg) {
		t.Error("expected true when one OR condition is true")
	}
}

func TestEvaluateOr_AllFalse(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{
		Or: []Condition{
			{Field: "channel", Operator: "equals", Value: "chat"},
			{Field: "channel", Operator: "equals", Value: "notification"},
		},
	}
	if c.Evaluate(msg) {
		t.Error("expected false when all OR conditions are false")
	}
}

func TestEvaluateOr_Empty(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Or: []Condition{}}
	if c.Evaluate(msg) {
		t.Error("expected false for empty OR")
	}
}

func TestEvaluateNested_AndContainingOr(t *testing.T) {
	msg := newTestMessage()

	// (channel = email AND (sender contains @firma.de OR subject contains Dringend))
	c := &Condition{
		And: []Condition{
			{Field: "channel", Operator: "equals", Value: "email"},
			{
				Or: []Condition{
					{Field: "sender_email", Operator: "contains", Value: "@firma.de"},
					{Field: "subject", Operator: "contains", Value: "Dringend"},
				},
			},
		},
	}
	if !c.Evaluate(msg) {
		t.Error("expected true for nested AND(OR) with matching conditions")
	}
}

func TestEvaluateNested_OrContainingAnd(t *testing.T) {
	msg := newTestMessage()

	// (channel = chat AND subject = "test") OR (channel = email AND sender contains firma)
	c := &Condition{
		Or: []Condition{
			{
				And: []Condition{
					{Field: "channel", Operator: "equals", Value: "chat"},
					{Field: "subject", Operator: "equals", Value: "test"},
				},
			},
			{
				And: []Condition{
					{Field: "channel", Operator: "equals", Value: "email"},
					{Field: "sender_email", Operator: "contains", Value: "firma"},
				},
			},
		},
	}
	if !c.Evaluate(msg) {
		t.Error("expected true for nested OR(AND) with one matching branch")
	}
}

func TestEvaluateLeaf_CRMLinked(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "crm_linked", Operator: "equals", Value: "true"}
	if !c.Evaluate(msg) {
		t.Error("expected true for crm_linked when CRMContactID is set")
	}

	// Without CRM link
	msgNoCRM := newTestMessage()
	msgNoCRM.CRMContactID = nil
	c = &Condition{Field: "crm_linked", Operator: "equals", Value: "true"}
	if c.Evaluate(msgNoCRM) {
		t.Error("expected false for crm_linked when CRMContactID is nil")
	}

	c = &Condition{Field: "crm_linked", Operator: "equals", Value: "false"}
	if !c.Evaluate(msgNoCRM) {
		t.Error("expected true for crm_linked=false when CRMContactID is nil")
	}
}

func TestEvaluateLeaf_Tags(t *testing.T) {
	msg := newTestMessage()
	msg.Tags = []string{"support", "vip"}

	c := &Condition{Field: "tags", Operator: "contains", Value: "support"}
	if !c.Evaluate(msg) {
		t.Error("expected true for tags containing 'support'")
	}

	c = &Condition{Field: "tags", Operator: "contains", Value: "billing"}
	if c.Evaluate(msg) {
		t.Error("expected false for tags not containing 'billing'")
	}
}

func TestEvaluateLeaf_UnknownField(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "nonexistent_field", Operator: "equals", Value: "anything"}
	if c.Evaluate(msg) {
		t.Error("expected false for unknown field name")
	}
}

func TestEvaluateLeaf_UnknownOperator(t *testing.T) {
	msg := newTestMessage()

	c := &Condition{Field: "channel", Operator: "regex_match", Value: "email.*"}
	if c.Evaluate(msg) {
		t.Error("expected false for unknown operator")
	}
}

func TestParseCondition(t *testing.T) {
	msg := newTestMessage()

	// Test that ParseCondition correctly converts models.Condition to routing.Condition
	mc := models.Condition{
		And: []models.Condition{
			{Field: "channel", Operator: "equals", Value: "email"},
			{
				Or: []models.Condition{
					{Field: "subject", Operator: "contains", Value: "Dringend"},
					{Field: "sender_email", Operator: "ends_with", Value: "@firma.de"},
				},
			},
		},
	}

	c := ParseCondition(mc)
	if !c.Evaluate(msg) {
		t.Error("expected ParseCondition to produce working Condition from models.Condition")
	}
}

func TestEvaluateLeaf_SenderEmailNil(t *testing.T) {
	msg := newTestMessage()
	msg.SenderEmail = nil

	c := &Condition{Field: "sender_email", Operator: "equals", Value: "test@example.com"}
	if c.Evaluate(msg) {
		t.Error("expected false when sender_email is nil")
	}

	c = &Condition{Field: "sender_email", Operator: "not_exists"}
	if !c.Evaluate(msg) {
		t.Error("expected true for not_exists when sender_email is nil")
	}
}

func TestEvaluateLeaf_TeamInboxField(t *testing.T) {
	msg := newTestMessage()
	teamID := uuid.New()
	msg.TeamInboxID = &teamID

	c := &Condition{Field: "team_inbox", Operator: "equals", Value: teamID.String()}
	if !c.Evaluate(msg) {
		t.Error("expected true when team_inbox matches UUID")
	}

	c = &Condition{Field: "team_inbox", Operator: "exists"}
	if !c.Evaluate(msg) {
		t.Error("expected true for team_inbox exists when set")
	}

	msg.TeamInboxID = nil
	c = &Condition{Field: "team_inbox", Operator: "exists"}
	if c.Evaluate(msg) {
		t.Error("expected false for team_inbox exists when nil")
	}
}

func TestEvaluateLeaf_BodyAlias(t *testing.T) {
	msg := newTestMessage()

	// "body" is an alias for "preview"
	c := &Condition{Field: "body", Operator: "contains", Value: "Projekt Alpha"}
	if !c.Evaluate(msg) {
		t.Error("expected true for body alias matching preview content")
	}
}
