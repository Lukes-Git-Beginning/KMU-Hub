package dialer

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestValidateTransition exercises the full agent state-machine transition table.
// Each row names the (current, new) pair and whether the transition is allowed.
func TestValidateTransition(t *testing.T) {
	tests := []struct {
		current string
		next    string
		allowed bool
	}{
		// From available
		{AgentStatusAvailable, AgentStatusOnCall, true},
		{AgentStatusAvailable, AgentStatusBreak, true},
		{AgentStatusAvailable, AgentStatusOffline, true},
		{AgentStatusAvailable, AgentStatusWrapUp, false},
		{AgentStatusAvailable, AgentStatusAvailable, false},

		// From on_call
		{AgentStatusOnCall, AgentStatusWrapUp, true},
		{AgentStatusOnCall, AgentStatusAvailable, true},
		{AgentStatusOnCall, AgentStatusBreak, false},
		{AgentStatusOnCall, AgentStatusOffline, false},

		// From wrap_up
		{AgentStatusWrapUp, AgentStatusAvailable, true},
		{AgentStatusWrapUp, AgentStatusBreak, true},
		{AgentStatusWrapUp, AgentStatusOnCall, false},
		{AgentStatusWrapUp, AgentStatusOffline, false},

		// From break
		{AgentStatusBreak, AgentStatusAvailable, true},
		{AgentStatusBreak, AgentStatusOffline, true},
		{AgentStatusBreak, AgentStatusOnCall, false},
		{AgentStatusBreak, AgentStatusWrapUp, false},

		// From offline
		{AgentStatusOffline, AgentStatusAvailable, true},
		{AgentStatusOffline, AgentStatusOnCall, false},
		{AgentStatusOffline, AgentStatusBreak, false},
		{AgentStatusOffline, AgentStatusWrapUp, false},

		// Unknown current status
		{"unknown", AgentStatusAvailable, false},
		{"", AgentStatusAvailable, false},
	}

	for _, tt := range tests {
		name := tt.current + "→" + tt.next
		t.Run(name, func(t *testing.T) {
			got := ValidateTransition(tt.current, tt.next)
			if got != tt.allowed {
				t.Errorf("ValidateTransition(%q, %q) = %v, want %v", tt.current, tt.next, got, tt.allowed)
			}
		})
	}
}

// TestParseAgentStatusData_Valid ensures the happy path decodes all fields.
func TestParseAgentStatusData_Valid(t *testing.T) {
	userID := uuid.New()
	campaignID := uuid.New()
	since := time.Now().UTC().Truncate(time.Second)

	data := agentStatusData{
		Status:     AgentStatusAvailable,
		CampaignID: campaignID.String(),
		Since:      since,
		TenantID:   uuid.New().String(),
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	entry, err := parseAgentStatusData(userID, string(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.UserID != userID {
		t.Errorf("UserID: got %s, want %s", entry.UserID, userID)
	}
	if entry.Status != AgentStatusAvailable {
		t.Errorf("Status: got %s, want %s", entry.Status, AgentStatusAvailable)
	}
	if entry.CampaignID == nil || *entry.CampaignID != campaignID {
		t.Errorf("CampaignID: got %v, want %s", entry.CampaignID, campaignID)
	}
}

// TestParseAgentStatusData_NoCampaign ensures a missing campaign_id field
// results in a nil CampaignID pointer (no error).
func TestParseAgentStatusData_NoCampaign(t *testing.T) {
	userID := uuid.New()
	data := agentStatusData{
		Status: AgentStatusOffline,
		Since:  time.Now().UTC(),
	}
	raw, _ := json.Marshal(data)

	entry, err := parseAgentStatusData(userID, string(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.CampaignID != nil {
		t.Errorf("expected nil CampaignID, got %v", entry.CampaignID)
	}
}

// TestParseAgentStatusData_InvalidJSON verifies that malformed input returns an error.
func TestParseAgentStatusData_InvalidJSON(t *testing.T) {
	_, err := parseAgentStatusData(uuid.New(), "not-json{{{")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// TestParseAgentStatusData_InvalidCampaignUUID verifies that a malformed
// campaign_id UUID in otherwise valid JSON returns an error.
func TestParseAgentStatusData_InvalidCampaignUUID(t *testing.T) {
	raw := `{"status":"available","campaign_id":"not-a-uuid","since":"2024-01-01T00:00:00Z","tenant_id":""}`
	_, err := parseAgentStatusData(uuid.New(), raw)
	if err == nil {
		t.Error("expected error for invalid campaign_id UUID, got nil")
	}
}
