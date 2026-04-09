package dialer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	agentStatusKeyPrefix    = "dialer:agent:status:"
	campaignAgentsKeyPrefix = "dialer:campaign:agents:"
	agentStatusTTL          = 8 * time.Hour // one work day
)

// ErrInvalidTransition is returned when a status transition is not permitted
// by the dialer agent state machine.
var ErrInvalidTransition = fmt.Errorf("invalid agent status transition")

// validTransitions defines the allowed state machine edges.
var validTransitions = map[string][]string{
	AgentStatusAvailable: {AgentStatusOnCall, AgentStatusBreak, AgentStatusOffline},
	AgentStatusOnCall:    {AgentStatusWrapUp, AgentStatusAvailable},
	AgentStatusWrapUp:    {AgentStatusAvailable, AgentStatusBreak},
	AgentStatusBreak:     {AgentStatusAvailable, AgentStatusOffline},
	AgentStatusOffline:   {AgentStatusAvailable},
}

// agentStatusData is the JSON payload stored in Redis.
type agentStatusData struct {
	Status     string    `json:"status"`
	CampaignID string    `json:"campaign_id,omitempty"`
	Since      time.Time `json:"since"`
	TenantID   string    `json:"tenant_id"`
}

// AgentStatusStore manages dialer agent status in Redis.
type AgentStatusStore struct {
	client *redis.Client
}

// NewAgentStatusStore creates a new Redis-backed agent status store.
func NewAgentStatusStore(client *redis.Client) *AgentStatusStore {
	return &AgentStatusStore{client: client}
}

func agentStatusKey(userID uuid.UUID) string {
	return agentStatusKeyPrefix + userID.String()
}

func campaignAgentsKey(campaignID uuid.UUID) string {
	return campaignAgentsKeyPrefix + campaignID.String()
}

// ValidateTransition reports whether transitioning from currentStatus to
// newStatus is permitted by the dialer state machine.
func ValidateTransition(currentStatus, newStatus string) bool {
	allowed, ok := validTransitions[currentStatus]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == newStatus {
			return true
		}
	}
	return false
}

// SetStatus updates the agent's status in Redis, enforcing state-machine rules
// and maintaining the per-campaign agent sets.
func (s *AgentStatusStore) SetStatus(
	ctx context.Context,
	userID uuid.UUID,
	status string,
	campaignID *uuid.UUID,
	tenantID uuid.UUID,
) error {
	// Read current state to validate the transition.
	current, err := s.GetStatus(ctx, userID)
	if err != nil {
		return fmt.Errorf("get current agent status: %w", err)
	}

	// An absent key means the agent is effectively offline.
	currentStatus := AgentStatusOffline
	var prevCampaignID *uuid.UUID
	if current != nil {
		currentStatus = current.Status
		prevCampaignID = current.CampaignID
	}

	if !ValidateTransition(currentStatus, status) {
		return fmt.Errorf("%w: %s → %s", ErrInvalidTransition, currentStatus, status)
	}

	// Build the new payload.
	data := agentStatusData{
		Status:   status,
		Since:    time.Now().UTC(),
		TenantID: tenantID.String(),
	}
	if campaignID != nil {
		data.CampaignID = campaignID.String()
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal agent status: %w", err)
	}

	pipe := s.client.Pipeline()

	// Write the agent status key.
	pipe.Set(ctx, agentStatusKey(userID), jsonData, agentStatusTTL)

	// Maintain campaign agent sets.
	if campaignID != nil {
		pipe.SAdd(ctx, campaignAgentsKey(*campaignID), userID.String())
	}
	// Remove from previous campaign set if the agent switched campaigns.
	if prevCampaignID != nil && (campaignID == nil || *prevCampaignID != *campaignID) {
		pipe.SRem(ctx, campaignAgentsKey(*prevCampaignID), userID.String())
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set agent status pipeline: %w", err)
	}

	return nil
}

// GetStatus retrieves the current status of an agent. Returns nil if the agent
// has no active status record (treated as offline).
func (s *AgentStatusStore) GetStatus(ctx context.Context, userID uuid.UUID) (*AgentStatusEntry, error) {
	val, err := s.client.Get(ctx, agentStatusKey(userID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent status: %w", err)
	}

	return parseAgentStatusData(userID, val)
}

// GetCampaignAgents returns all agents currently associated with a campaign.
func (s *AgentStatusStore) GetCampaignAgents(ctx context.Context, campaignID uuid.UUID) ([]AgentStatusEntry, error) {
	members, err := s.client.SMembers(ctx, campaignAgentsKey(campaignID)).Result()
	if err != nil {
		return nil, fmt.Errorf("smembers campaign agents: %w", err)
	}
	if len(members) == 0 {
		return nil, nil
	}

	return s.fetchAgentEntries(ctx, members)
}

// GetAvailableAgents returns IDs of agents in a campaign whose status is
// "available".
func (s *AgentStatusStore) GetAvailableAgents(ctx context.Context, campaignID uuid.UUID) ([]uuid.UUID, error) {
	entries, err := s.GetCampaignAgents(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	var available []uuid.UUID
	for _, e := range entries {
		if e.Status == AgentStatusAvailable {
			available = append(available, e.UserID)
		}
	}
	return available, nil
}

// RemoveFromCampaign removes an agent from a campaign's agent set.
func (s *AgentStatusStore) RemoveFromCampaign(ctx context.Context, userID, campaignID uuid.UUID) error {
	if err := s.client.SRem(ctx, campaignAgentsKey(campaignID), userID.String()).Err(); err != nil {
		return fmt.Errorf("srem campaign agent: %w", err)
	}
	return nil
}

// fetchAgentEntries uses a pipeline to GET all agent status keys in one round-trip.
func (s *AgentStatusStore) fetchAgentEntries(ctx context.Context, userIDStrs []string) ([]AgentStatusEntry, error) {
	pipe := s.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd, len(userIDStrs))

	for _, idStr := range userIDStrs {
		uid, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		cmds[idStr] = pipe.Get(ctx, agentStatusKey(uid))
	}

	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		// Individual misses are handled per-command below; only a complete
		// pipeline failure is fatal.
		allFailed := true
		for _, cmd := range cmds {
			if cmd.Err() == nil || cmd.Err() == redis.Nil {
				allFailed = false
				break
			}
		}
		if allFailed {
			return nil, fmt.Errorf("pipeline get agent statuses: %w", err)
		}
	}

	entries := make([]AgentStatusEntry, 0, len(cmds))
	for idStr, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil {
			continue // Key expired or missing — agent is offline, skip.
		}
		uid, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		entry, err := parseAgentStatusData(uid, val)
		if err != nil {
			continue // Corrupted data — skip rather than propagate.
		}
		entries = append(entries, *entry)
	}
	return entries, nil
}

// parseAgentStatusData deserialises a raw Redis value into an AgentStatusEntry.
func parseAgentStatusData(userID uuid.UUID, raw string) (*AgentStatusEntry, error) {
	var data agentStatusData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, fmt.Errorf("unmarshal agent status: %w", err)
	}

	entry := &AgentStatusEntry{
		UserID: userID,
		Status: data.Status,
		Since:  data.Since,
	}
	if data.CampaignID != "" {
		cid, err := uuid.Parse(data.CampaignID)
		if err != nil {
			return nil, fmt.Errorf("parse campaign id: %w", err)
		}
		entry.CampaignID = &cid
	}
	return entry, nil
}
