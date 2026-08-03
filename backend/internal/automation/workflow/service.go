package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/automation/condition"
	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/models"
)

// TriggerCatalog provides read-only access to trigger definitions.
// Implemented by trigger.TriggerRegistry.
type TriggerCatalog interface {
	Get(triggerType string) (any, bool)
	All() []any
}

// ActionCatalog provides read-only access to action executors and definitions.
// Implemented by action.ActionRegistry.
type ActionCatalog interface {
	Get(actionType string) (any, bool)
	AllDefinitions() []any
}

// triggerRegistryAdapter adapts *trigger.TriggerRegistry to TriggerCatalog.
type triggerRegistryAdapter struct {
	getFn func(string) (any, bool)
	allFn func() []any
}

func (a *triggerRegistryAdapter) Get(triggerType string) (any, bool) { return a.getFn(triggerType) }
func (a *triggerRegistryAdapter) All() []any                        { return a.allFn() }

// actionRegistryAdapter adapts *action.ActionRegistry to ActionCatalog.
type actionRegistryAdapter struct {
	getFn    func(string) (any, bool)
	allDefFn func() []any
}

func (a *actionRegistryAdapter) Get(actionType string) (any, bool) { return a.getFn(actionType) }
func (a *actionRegistryAdapter) AllDefinitions() []any             { return a.allDefFn() }

// Service provides business logic for automation CRUD, execution logs,
// catalog queries, template management, and testing/dry-run operations.
type Service struct {
	repo            Repository
	execRepo        ExecutionRepository
	templateRepo    TemplateRepository
	triggerCatalog  TriggerCatalog
	actionCatalog   ActionCatalog
	condEvaluator   *condition.Evaluator
	idempotencyRepo idempotency.Repository
	executor        Executor
}

// NewService creates a new workflow service.
// triggerGet/triggerAll and actionGet/actionAllDefs are function references
// from the trigger and action registries to avoid import cycles.
// idempotencyRepo and executor are used exclusively by TriggerWebhook (the
// inbound webhook-trigger path); every other method ignores them.
func NewService(
	repo Repository,
	execRepo ExecutionRepository,
	templateRepo TemplateRepository,
	triggerGet func(string) (any, bool),
	triggerAll func() []any,
	actionGet func(string) (any, bool),
	actionAllDefs func() []any,
	condEval *condition.Evaluator,
	idempotencyRepo idempotency.Repository,
	executor Executor,
) *Service {
	return &Service{
		repo:         repo,
		execRepo:     execRepo,
		templateRepo: templateRepo,
		triggerCatalog: &triggerRegistryAdapter{
			getFn: triggerGet,
			allFn: triggerAll,
		},
		actionCatalog: &actionRegistryAdapter{
			getFn:    actionGet,
			allDefFn: actionAllDefs,
		},
		condEvaluator:   condEval,
		idempotencyRepo: idempotencyRepo,
		executor:        executor,
	}
}

// ============================================================================
// CRUD Operations
// ============================================================================

// Create creates a new automation after validating its configuration.
func (s *Service) Create(ctx context.Context, auto *models.Automation) error {
	if err := s.validateAutomation(auto); err != nil {
		return err
	}

	auto.ID = uuid.New()
	auto.CreatedAt = time.Now()
	auto.UpdatedAt = time.Now()

	if auto.MaxSteps <= 0 {
		auto.MaxSteps = 10
	}

	// trigger_config/conditions/actions are NOT NULL in the schema (default
	// '{}'/'{}'/'[]'); a caller that omits them (e.g. a trigger with no extra
	// config, or no conditions) leaves the Go field nil, which the repo would
	// otherwise INSERT as an explicit NULL and violate the constraint.
	if auto.TriggerConfig == nil {
		auto.TriggerConfig = json.RawMessage(`{}`)
	}
	if auto.Conditions == nil {
		auto.Conditions = json.RawMessage(`{}`)
	}
	if auto.Actions == nil {
		auto.Actions = json.RawMessage(`[]`)
	}

	if err := ensureWebhookSecret(auto); err != nil {
		return err
	}

	return s.repo.Create(ctx, auto)
}

// Update updates an existing automation after validating its configuration.
func (s *Service) Update(ctx context.Context, auto *models.Automation) error {
	if err := s.validateAutomation(auto); err != nil {
		return err
	}

	if err := ensureWebhookSecret(auto); err != nil {
		return err
	}

	auto.UpdatedAt = time.Now()
	return s.repo.Update(ctx, auto)
}

// Delete removes an automation by ID, scoped to the tenant.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	return s.repo.Delete(ctx, id, tenantID)
}

// GetByID retrieves an automation by ID, scoped to the tenant.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Automation, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// List retrieves automations matching the given filter.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*models.Automation, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return s.repo.List(ctx, filter)
}

// ============================================================================
// Enable/Disable
// ============================================================================

// SetActive enables or disables an automation, scoped to the tenant.
func (s *Service) SetActive(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, active bool) error {
	return s.repo.SetActive(ctx, id, tenantID, active)
}

// ============================================================================
// Execution Logs
// ============================================================================

// ListExecutions retrieves execution logs matching the given filter.
func (s *Service) ListExecutions(ctx context.Context, filter ExecutionFilter) ([]*models.AutomationExecution, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return s.execRepo.ListExecutions(ctx, filter)
}

// GetExecution retrieves a single execution log by ID, scoped to the tenant.
func (s *Service) GetExecution(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.AutomationExecution, error) {
	return s.execRepo.GetExecution(ctx, id, tenantID)
}

// ============================================================================
// Catalog
// ============================================================================

// ListTriggers returns all registered trigger definitions (as opaque values).
func (s *Service) ListTriggers() []any {
	return s.triggerCatalog.All()
}

// ListActions returns all registered action definitions (as opaque values).
func (s *Service) ListActions() []any {
	return s.actionCatalog.AllDefinitions()
}

// ============================================================================
// Templates
// ============================================================================

// ListTemplates returns all templates, optionally filtered by category.
func (s *Service) ListTemplates(ctx context.Context, category *string) ([]*models.AutomationTemplate, error) {
	return s.templateRepo.ListTemplates(ctx, category)
}

// GetTemplate retrieves a template by ID.
func (s *Service) GetTemplate(ctx context.Context, id string) (*models.AutomationTemplate, error) {
	return s.templateRepo.GetTemplate(ctx, id)
}

// CreateFromTemplate creates a new automation from a template for a given tenant.
func (s *Service) CreateFromTemplate(ctx context.Context, templateID string, tenantID uuid.UUID, ownerID uuid.UUID, name string) (*models.Automation, error) {
	tmpl, err := s.templateRepo.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if tmpl == nil {
		return nil, ErrTemplateNotFound
	}

	auto := &models.Automation{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          name,
		Description:   tmpl.Description,
		Scope:         models.AutomationScopePersonal,
		OwnerID:       ownerID,
		TriggerType:   tmpl.TriggerType,
		TriggerConfig: tmpl.TriggerConfig,
		Conditions:    tmpl.Conditions,
		Actions:       tmpl.Actions,
		IsActive:      false, // Inactive by default, user enables after review
		MaxSteps:      10,
		TemplateID:    &tmpl.ID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(ctx, auto); err != nil {
		return nil, err
	}

	return auto, nil
}

// ============================================================================
// Stats
// ============================================================================

// AutomationStats holds aggregate statistics.
type AutomationStats struct {
	TotalAutomations  int     `json:"total_automations"`
	ActiveAutomations int     `json:"active_automations"`
	TotalExecutions   int     `json:"total_executions"`
	SuccessfulExecs   int     `json:"successful_executions"`
	FailedExecs       int     `json:"failed_executions"`
	SkippedExecs      int     `json:"skipped_executions"`
	SuccessRate       float64 `json:"success_rate"`
}

// GetStats returns aggregate automation statistics for a tenant.
func (s *Service) GetStats(ctx context.Context, tenantID uuid.UUID) (*AutomationStats, error) {
	_, totalCount, err := s.repo.List(ctx, ListFilter{TenantID: tenantID, Limit: 1})
	if err != nil {
		return nil, err
	}

	isActive := true
	_, activeCount, err := s.repo.List(ctx, ListFilter{TenantID: tenantID, IsActive: &isActive, Limit: 1})
	if err != nil {
		return nil, err
	}

	stats := &AutomationStats{
		TotalAutomations:  totalCount,
		ActiveAutomations: activeCount,
	}

	statusCompleted := models.ExecutionStatusCompleted
	_, completedCount, err := s.execRepo.ListExecutions(ctx, ExecutionFilter{
		TenantID: tenantID,
		Status:   &statusCompleted,
		Limit:    1,
	})
	if err != nil {
		slog.Warn("failed to count completed executions", "error", err)
	} else {
		stats.SuccessfulExecs = completedCount
	}

	statusFailed := models.ExecutionStatusFailed
	_, failedCount, err := s.execRepo.ListExecutions(ctx, ExecutionFilter{
		TenantID: tenantID,
		Status:   &statusFailed,
		Limit:    1,
	})
	if err != nil {
		slog.Warn("failed to count failed executions", "error", err)
	} else {
		stats.FailedExecs = failedCount
	}

	statusSkipped := models.ExecutionStatusSkipped
	_, skippedCount, err := s.execRepo.ListExecutions(ctx, ExecutionFilter{
		TenantID: tenantID,
		Status:   &statusSkipped,
		Limit:    1,
	})
	if err != nil {
		slog.Warn("failed to count skipped executions", "error", err)
	} else {
		stats.SkippedExecs = skippedCount
	}

	stats.TotalExecutions = stats.SuccessfulExecs + stats.FailedExecs + stats.SkippedExecs

	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessfulExecs) / float64(stats.TotalExecutions) * 100
	}

	return stats, nil
}

// ============================================================================
// Test & Dry-Run
// ============================================================================

// TestCondition evaluates a condition against sample data without executing actions.
func (s *Service) TestCondition(ctx context.Context, conditionJSON json.RawMessage, sampleEnv map[string]any) (bool, error) {
	var config models.ConditionConfig
	if err := json.Unmarshal(conditionJSON, &config); err != nil {
		return false, fmt.Errorf("invalid condition config: %w", err)
	}

	return s.condEvaluator.Evaluate(ctx, config, sampleEnv)
}

// DryRunResult represents the result of a dry-run execution.
type DryRunResult struct {
	ConditionResult bool         `json:"condition_result"`
	Steps           []DryRunStep `json:"steps"`
}

// DryRunStep represents a simulated action step.
type DryRunStep struct {
	ActionType string `json:"action_type"`
	ActionName string `json:"action_name"`
	WouldRun   bool   `json:"would_run"`
	Error      string `json:"error,omitempty"`
}

// DryRun simulates an automation execution without calling real action gRPC methods.
// tenantID is required for isolation.
func (s *Service) DryRun(ctx context.Context, automationID uuid.UUID, tenantID uuid.UUID, sampleEnv map[string]any) (*DryRunResult, error) {
	auto, err := s.repo.GetByID(ctx, automationID, tenantID)
	if err != nil {
		return nil, err
	}
	if auto == nil {
		return nil, ErrAutomationNotFound
	}

	result := &DryRunResult{}

	// Evaluate conditions
	var condConfig models.ConditionConfig
	if len(auto.Conditions) > 0 {
		if err := json.Unmarshal(auto.Conditions, &condConfig); err != nil {
			return nil, fmt.Errorf("invalid conditions: %w", err)
		}
	}

	condResult, err := s.condEvaluator.Evaluate(ctx, condConfig, sampleEnv)
	if err != nil {
		return nil, fmt.Errorf("condition evaluation: %w", err)
	}
	result.ConditionResult = condResult

	// Simulate actions
	var actions []models.ActionConfig
	if auto.Actions != nil {
		if err := json.Unmarshal(auto.Actions, &actions); err != nil {
			return nil, fmt.Errorf("invalid actions: %w", err)
		}
	}

	for _, actionCfg := range actions {
		step := DryRunStep{
			ActionType: actionCfg.Type,
			WouldRun:   condResult,
		}

		if _, ok := s.actionCatalog.Get(actionCfg.Type); !ok {
			step.Error = "action type not registered"
			step.WouldRun = false
		}

		result.Steps = append(result.Steps, step)
	}

	return result, nil
}

// ============================================================================
// Validation
// ============================================================================

// validateAutomation checks that the trigger type and action types are registered.
func (s *Service) validateAutomation(auto *models.Automation) error {
	// Validate trigger type
	if _, ok := s.triggerCatalog.Get(auto.TriggerType); !ok {
		return fmt.Errorf("%w: unknown trigger type %q", ErrInvalidCondition, auto.TriggerType)
	}

	// Validate actions
	if auto.Actions != nil {
		var actions []models.ActionConfig
		if err := json.Unmarshal(auto.Actions, &actions); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAction, err)
		}

		for _, a := range actions {
			if _, ok := s.actionCatalog.Get(a.Type); !ok {
				return fmt.Errorf("%w: unknown action type %q", ErrInvalidAction, a.Type)
			}
		}
	}

	// Validate conditions if present
	if len(auto.Conditions) > 0 {
		var condConfig models.ConditionConfig
		if err := json.Unmarshal(auto.Conditions, &condConfig); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidCondition, err)
		}

		if condConfig.Mode == models.ConditionModeExpression && condConfig.Expression != "" {
			if err := s.condEvaluator.ValidateExpression(condConfig.Expression, map[string]any{}); err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidCondition, err)
			}
		}
	}

	return nil
}
