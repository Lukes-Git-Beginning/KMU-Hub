package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/inbox/message"
	"github.com/kmuhub/kmuhub/internal/models"
)

// EmailSender is a local interface for sending auto-reply emails.
type EmailSender interface {
	Send(ctx context.Context, to string, subject string, body string) error
}

// tenantRuleCache holds rules and refresh time for a single tenant.
type tenantRuleCache struct {
	rules       []*models.RoutingRule
	lastRefresh time.Time
}

// Service handles routing rule business logic including evaluation,
// action execution, and rule caching.
type Service struct {
	repo        Repository
	messageRepo message.Repository
	emailSender EmailSender

	// Rule cache with 60s TTL, keyed by tenantID
	cacheMu  sync.RWMutex
	cache    map[uuid.UUID]*tenantRuleCache
	cacheTTL time.Duration
}

// NewService creates a new routing service.
func NewService(repo Repository, messageRepo message.Repository, emailSender EmailSender) *Service {
	return &Service{
		repo:        repo,
		messageRepo: messageRepo,
		emailSender: emailSender,
		cache:       make(map[uuid.UUID]*tenantRuleCache),
		cacheTTL:    60 * time.Second,
	}
}

// Create creates a new routing rule.
func (s *Service) Create(ctx context.Context, rule *models.RoutingRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}

	if err := validateConditions(rule.Conditions); err != nil {
		return ErrInvalidConditions
	}

	if err := validateActions(rule.Actions); err != nil {
		return err
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return err
	}

	// Invalidate cache for this tenant
	s.invalidateCache(rule.TenantID)
	return nil
}

// Update updates an existing routing rule.
func (s *Service) Update(ctx context.Context, rule *models.RoutingRule) error {
	if err := validateConditions(rule.Conditions); err != nil {
		return ErrInvalidConditions
	}

	if err := validateActions(rule.Actions); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, rule); err != nil {
		return err
	}

	s.invalidateCache(rule.TenantID)
	return nil
}

// Delete deletes a routing rule within a tenant.
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	s.invalidateCache(tenantID)
	return nil
}

// GetByID retrieves a routing rule by ID within a tenant.
func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.RoutingRule, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// ListAll returns all routing rules within a tenant.
func (s *Service) ListAll(ctx context.Context, tenantID uuid.UUID) ([]*models.RoutingRule, error) {
	return s.repo.ListAll(ctx, tenantID)
}

// EvaluateAndApply loads active rules (cached), evaluates them in priority order,
// and on first match executes all actions. First-match-wins semantics.
func (s *Service) EvaluateAndApply(ctx context.Context, msg *models.InboxMessage) error {
	rules, err := s.getCachedRules(ctx, msg.TenantID, &msg.Channel)
	if err != nil {
		return fmt.Errorf("load routing rules: %w", err)
	}

	for _, rule := range rules {
		// Parse conditions from JSON
		var mc models.Condition
		if err := json.Unmarshal(rule.Conditions, &mc); err != nil {
			slog.Warn("routing: failed to parse rule conditions",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"error", err,
			)
			continue
		}

		condition := ParseCondition(mc)
		if condition.Evaluate(msg) {
			slog.Info("routing rule matched",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"message_id", msg.ID,
			)

			// Parse and execute actions
			var actions []models.Action
			if err := json.Unmarshal(rule.Actions, &actions); err != nil {
				slog.Error("routing: failed to parse rule actions",
					"rule_id", rule.ID,
					"error", err,
				)
				return fmt.Errorf("parse actions for rule %s: %w", rule.ID, err)
			}

			if err := s.executeActions(ctx, msg, actions); err != nil {
				return fmt.Errorf("execute actions for rule %s: %w", rule.ID, err)
			}

			// First-match-wins: stop after first matching rule
			return nil
		}
	}

	return nil
}

// TestRule tests whether conditions match a message without executing actions.
// Used for the TestRoutingRule RPC.
func (s *Service) TestRule(_ context.Context, conditions models.Condition, msg *models.InboxMessage) bool {
	c := ParseCondition(conditions)
	return c.Evaluate(msg)
}

// executeActions applies all actions from a matched rule to the message.
func (s *Service) executeActions(ctx context.Context, msg *models.InboxMessage, actions []models.Action) error {
	for _, action := range actions {
		switch action.Type {
		case "route_to_team":
			if err := s.actionRouteToTeam(ctx, msg, action.Config); err != nil {
				slog.Error("routing action failed: route_to_team",
					"message_id", msg.ID,
					"error", err,
				)
				return err
			}

		case "assign_to":
			if err := s.actionAssignTo(ctx, msg, action.Config); err != nil {
				slog.Error("routing action failed: assign_to",
					"message_id", msg.ID,
					"error", err,
				)
				return err
			}

		case "add_tags":
			if err := s.actionAddTags(ctx, msg, action.Config); err != nil {
				slog.Error("routing action failed: add_tags",
					"message_id", msg.ID,
					"error", err,
				)
				return err
			}

		case "auto_reply":
			if err := s.actionAutoReply(ctx, msg, action.Config); err != nil {
				slog.Warn("routing action warning: auto_reply",
					"message_id", msg.ID,
					"error", err,
				)
				// Auto-reply failure is non-fatal
			}

		default:
			slog.Warn("routing: unknown action type",
				"action_type", action.Type,
				"message_id", msg.ID,
			)
		}
	}
	return nil
}

// actionRouteToTeam sets the team_inbox_id on the message.
func (s *Service) actionRouteToTeam(ctx context.Context, msg *models.InboxMessage, config json.RawMessage) error {
	var cfg struct {
		TeamInboxID string `json:"team_inbox_id"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse route_to_team config: %w", err)
	}

	teamID, err := uuid.Parse(cfg.TeamInboxID)
	if err != nil {
		return fmt.Errorf("invalid team_inbox_id: %w", err)
	}

	msg.TeamInboxID = &teamID
	return s.messageRepo.Update(ctx, msg)
}

// actionAssignTo sets the assigned_to on the message.
func (s *Service) actionAssignTo(ctx context.Context, msg *models.InboxMessage, config json.RawMessage) error {
	var cfg struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse assign_to config: %w", err)
	}

	userID, err := uuid.Parse(cfg.UserID)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}

	msg.AssignedTo = &userID
	return s.messageRepo.Update(ctx, msg)
}

// actionAddTags appends tags to the message.
func (s *Service) actionAddTags(ctx context.Context, msg *models.InboxMessage, config json.RawMessage) error {
	var cfg struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse add_tags config: %w", err)
	}

	// Append only new tags (dedup)
	existing := make(map[string]bool)
	for _, t := range msg.Tags {
		existing[t] = true
	}
	for _, t := range cfg.Tags {
		if !existing[t] {
			msg.Tags = append(msg.Tags, t)
			existing[t] = true
		}
	}

	return s.messageRepo.Update(ctx, msg)
}

// actionAutoReply sends an automatic reply email. Only works for email channel messages.
func (s *Service) actionAutoReply(ctx context.Context, msg *models.InboxMessage, config json.RawMessage) error {
	if msg.Channel != "email" {
		slog.Warn("auto-reply skipped: not an email channel message",
			"channel", msg.Channel,
			"message_id", msg.ID,
		)
		return nil
	}

	if s.emailSender == nil {
		slog.Debug("auto-reply skipped: email sender not configured")
		return nil
	}

	var cfg struct {
		Body    string `json:"body"`
		Subject string `json:"subject"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse auto_reply config: %w", err)
	}

	if msg.SenderEmail == nil || *msg.SenderEmail == "" {
		slog.Warn("auto-reply skipped: no sender email on message",
			"message_id", msg.ID,
		)
		return nil
	}

	subject := cfg.Subject
	if subject == "" {
		subject = "Re: " + msg.Subject
	}

	return s.emailSender.Send(ctx, *msg.SenderEmail, subject, cfg.Body)
}

// getCachedRules returns rules from cache if fresh, otherwise refreshes from DB.
func (s *Service) getCachedRules(ctx context.Context, tenantID uuid.UUID, channel *string) ([]*models.RoutingRule, error) {
	s.cacheMu.RLock()
	if tc, ok := s.cache[tenantID]; ok && tc.rules != nil && time.Since(tc.lastRefresh) < s.cacheTTL {
		rules := s.filterByChannel(tc.rules, channel)
		s.cacheMu.RUnlock()
		return rules, nil
	}
	s.cacheMu.RUnlock()

	return s.refreshCache(ctx, tenantID, channel)
}

// refreshCache reloads rules from the database for a given tenant.
func (s *Service) refreshCache(ctx context.Context, tenantID uuid.UUID, channel *string) ([]*models.RoutingRule, error) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// Double-check after acquiring write lock
	if tc, ok := s.cache[tenantID]; ok && tc.rules != nil && time.Since(tc.lastRefresh) < s.cacheTTL {
		return s.filterByChannel(tc.rules, channel), nil
	}

	// Load all active rules (not filtered by channel) for cache
	rules, err := s.repo.ListActive(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}

	s.cache[tenantID] = &tenantRuleCache{
		rules:       rules,
		lastRefresh: time.Now(),
	}

	return s.filterByChannel(rules, channel), nil
}

// filterByChannel returns rules that apply to the given channel
// (rules with nil channel apply to all channels).
func (s *Service) filterByChannel(rules []*models.RoutingRule, channel *string) []*models.RoutingRule {
	if channel == nil {
		return rules
	}

	var filtered []*models.RoutingRule
	for _, r := range rules {
		if r.Channel == nil || *r.Channel == *channel {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// invalidateCache forces a cache refresh on next access for a given tenant.
func (s *Service) invalidateCache(tenantID uuid.UUID) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.cache, tenantID)
}

// validateConditions checks that conditions JSON is valid.
func validateConditions(data json.RawMessage) error {
	if len(data) == 0 {
		return ErrInvalidConditions
	}
	var c models.Condition
	return json.Unmarshal(data, &c)
}

// validateActions checks that actions JSON contains valid action types.
func validateActions(data json.RawMessage) error {
	if len(data) == 0 {
		return ErrInvalidAction
	}

	var actions []models.Action
	if err := json.Unmarshal(data, &actions); err != nil {
		return ErrInvalidAction
	}

	validTypes := map[string]bool{
		"route_to_team": true,
		"assign_to":     true,
		"add_tags":      true,
		"auto_reply":    true,
	}

	for _, a := range actions {
		if !validTypes[a.Type] {
			return fmt.Errorf("%w: %s", ErrInvalidAction, a.Type)
		}
	}

	return nil
}
