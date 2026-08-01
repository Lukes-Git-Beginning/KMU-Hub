package rule

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/automation/condition"
	"github.com/kmuhub/kmuhub/internal/models"
)

// applyScanLimit caps how many messages one ApplyAll run inspects. Running the
// rule set over an unbounded mailbox is a request the user can fire from a
// button, so it needs a ceiling rather than a promise of eventual completion.
// Newest messages are scanned first, which is where rules matter.
//
// lean: single fixed ceiling, newest-first, run synchronously. Move to a
// background job with a cursor when a tenant's mailbox regularly exceeds this
// limit or the request starts hitting the gateway timeout.
const applyScanLimit = 2000

// Service holds the business logic for email rules: validation, persistence and
// the bulk application of the rule set to stored messages.
type Service struct {
	repo Repository
	eval *condition.Evaluator
}

// NewService creates a new email rule service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, eval: condition.NewEvaluator()}
}

// RuleInput carries the mutable fields of a rule. Nil pointers mean "leave as
// is" on update; Create requires all of them.
type RuleInput struct {
	Name         *string
	Field        *string
	Op           *string
	Value        *string
	ActionType   *string
	ActionTarget *string
}

// List returns all rules of a tenant in application order (oldest first).
func (s *Service) List(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailRule, error) {
	return s.repo.List(ctx, tenantID)
}

// GetByID returns one rule of a tenant.
func (s *Service) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailRule, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// Create validates and stores a new rule.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, in RuleInput) (*models.EmailRule, error) {
	now := time.Now().UTC()
	r := &models.EmailRule{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Op:        models.EmailRuleOpContains,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.applyInput(ctx, r, in); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("failed to create email rule: %w", err)
	}

	slog.Info("email rule created",
		"rule_id", r.ID, "tenant_id", tenantID,
		"field", r.Field, "action_type", r.ActionType)
	return r, nil
}

// Update applies a partial change to an existing rule.
func (s *Service) Update(ctx context.Context, id, tenantID uuid.UUID, in RuleInput) (*models.EmailRule, error) {
	r, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.applyInput(ctx, r, in); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, r); err != nil {
		return nil, err
	}

	slog.Info("email rule updated", "rule_id", id, "tenant_id", tenantID)
	return r, nil
}

// Delete removes a rule of the tenant.
func (s *Service) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		return err
	}
	slog.Info("email rule deleted", "rule_id", id, "tenant_id", tenantID)
	return nil
}

// applyInput writes the set fields of in onto r and validates the result. The
// validation always runs against the merged rule, not the patch, so a partial
// update cannot produce a combination that a create would have rejected (e.g.
// switching action_type to "move" without also moving the target).
func (s *Service) applyInput(ctx context.Context, r *models.EmailRule, in RuleInput) error {
	if in.Name != nil {
		r.Name = strings.TrimSpace(*in.Name)
	}
	if in.Field != nil {
		r.Field = strings.TrimSpace(*in.Field)
	}
	if in.Op != nil && strings.TrimSpace(*in.Op) != "" {
		r.Op = strings.TrimSpace(*in.Op)
	}
	if in.Value != nil {
		r.Value = strings.TrimSpace(*in.Value)
	}
	if in.ActionType != nil {
		r.ActionType = strings.TrimSpace(*in.ActionType)
	}
	if in.ActionTarget != nil {
		target, err := uuid.Parse(strings.TrimSpace(*in.ActionTarget))
		if err != nil {
			return ErrTargetRequired
		}
		r.ActionTarget = target
	}

	if r.Name == "" {
		return ErrNameRequired
	}
	if r.Field != models.EmailRuleFieldFrom && r.Field != models.EmailRuleFieldSubject {
		return ErrInvalidField
	}
	if r.Op != models.EmailRuleOpContains {
		return ErrInvalidOperator
	}
	if r.Value == "" {
		return ErrValueRequired
	}
	if r.ActionType != models.EmailRuleActionLabel && r.ActionType != models.EmailRuleActionMove {
		return ErrInvalidActionType
	}
	if r.ActionTarget == uuid.Nil {
		return ErrTargetRequired
	}

	// A move target has to be a folder of this tenant -- without the check a
	// rule could push messages into a foreign mailbox, which RLS on
	// email_messages would not catch (the row keeps its own tenant_id).
	//
	// lean: label targets stay unvalidated because email_labels does not exist
	// yet. Add the same existence check once the labels unit lands.
	if r.ActionType == models.EmailRuleActionMove {
		ok, err := s.repo.FolderBelongsToTenant(ctx, r.ActionTarget, r.TenantID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrTargetNotFound
		}
	}
	return nil
}

// ApplyResult reports what one ApplyAll run did.
type ApplyResult struct {
	// Affected counts messages that changed, not rule matches: a message hit by
	// three rules that all restate its current state counts as zero.
	Affected int
	// Scanned is how many messages were inspected, so a caller can distinguish
	// "nothing matched" from "the scan limit cut the run short".
	Scanned int
}

// ApplyAll runs every rule of the tenant over its stored messages and persists
// the resulting label and folder assignment.
//
// Rules are applied in creation order and accumulate: a later move overrides an
// earlier one, labels add up. That mirrors the frontend's demo behaviour so the
// switch from the mock to this service does not change what users see.
func (s *Service) ApplyAll(ctx context.Context, tenantID uuid.UUID) (ApplyResult, error) {
	rules, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return ApplyResult{}, err
	}
	if len(rules) == 0 {
		return ApplyResult{}, nil
	}

	candidates, err := s.repo.ListApplyCandidates(ctx, tenantID, applyScanLimit)
	if err != nil {
		return ApplyResult{}, err
	}

	result := ApplyResult{Scanned: len(candidates)}
	for _, c := range candidates {
		folderID, labelIDs, changed, err := s.evaluateMessage(ctx, rules, c)
		if err != nil {
			return result, err
		}
		if !changed {
			continue
		}
		if err := s.repo.ApplyToMessage(ctx, tenantID, c.ID, folderID, labelIDs); err != nil {
			return result, fmt.Errorf("failed to apply rules to message %s: %w", c.ID, err)
		}
		result.Affected++
	}

	slog.Info("email rules applied",
		"tenant_id", tenantID, "rules", len(rules),
		"scanned", result.Scanned, "affected", result.Affected)
	return result, nil
}

// evaluateMessage returns the folder and label set a message should end up with
// after all rules ran, plus whether that differs from its current state.
func (s *Service) evaluateMessage(ctx context.Context, rules []*models.EmailRule, c *models.EmailRuleCandidate) (uuid.UUID, []uuid.UUID, bool, error) {
	env := matchEnv(c)
	folderID := c.FolderID
	labelIDs := slices.Clone(c.LabelIDs)
	changed := false

	for _, r := range rules {
		match, err := s.eval.Evaluate(ctx, models.ConditionConfig{
			Mode: models.ConditionModeSimple,
			Simple: &models.Condition{
				Field:    r.Field,
				Operator: r.Op,
				Value:    r.Value,
			},
		}, env)
		if err != nil {
			return uuid.Nil, nil, false, fmt.Errorf("failed to evaluate rule %s: %w", r.ID, err)
		}
		if !match {
			continue
		}

		switch r.ActionType {
		case models.EmailRuleActionLabel:
			if !slices.Contains(labelIDs, r.ActionTarget) {
				labelIDs = append(labelIDs, r.ActionTarget)
				changed = true
			}
		case models.EmailRuleActionMove:
			if folderID != r.ActionTarget {
				folderID = r.ActionTarget
				changed = true
			}
		}
	}
	return folderID, labelIDs, changed, nil
}

// matchEnv builds the evaluation environment. "from" folds display name and
// address into one haystack so a rule on "from" matches either -- the frontend
// offers a single "Absender" field, not two.
func matchEnv(c *models.EmailRuleCandidate) map[string]any {
	return map[string]any{
		models.EmailRuleFieldFrom:    strings.TrimSpace(c.FromName + " " + c.FromEmail),
		models.EmailRuleFieldSubject: c.Subject,
	}
}
