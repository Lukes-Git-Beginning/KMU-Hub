package trigger

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/models"
)

// EngineExecutor is the interface the poller uses to execute automations.
// This avoids circular dependency with the engine package.
type EngineExecutor interface {
	Execute(ctx context.Context, auto models.Automation, evt models.EventPayload) error
}

// TimeTriggerPoller periodically checks for time-based triggers
// (e.g., overdue invoices, upcoming calendar events) and creates
// synthetic events to trigger matching automations.
type TimeTriggerPoller struct {
	interval     time.Duration
	workflowRepo workflow.Repository
	execRepo     workflow.ExecutionRepository
	engine       EngineExecutor
	pool         *pgxpool.Pool
}

// NewTimeTriggerPoller creates a new time-based trigger poller.
func NewTimeTriggerPoller(
	repo workflow.Repository,
	execRepo workflow.ExecutionRepository,
	engine EngineExecutor,
	pool *pgxpool.Pool,
) *TimeTriggerPoller {
	return &TimeTriggerPoller{
		interval:     5 * time.Minute,
		workflowRepo: repo,
		execRepo:     execRepo,
		engine:       engine,
		pool:         pool,
	}
}

// Start begins the polling loop. It blocks until the context is cancelled.
func (p *TimeTriggerPoller) Start(ctx context.Context) {
	ctx = database.WithSystemContext(ctx)
	slog.Info("time trigger poller started", "interval", p.interval)

	// Run immediately on start
	p.checkTimeTriggers(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("time trigger poller stopped")
			return
		case <-ticker.C:
			p.checkTimeTriggers(ctx)
		}
	}
}

// checkTimeTriggers queries active time-based automations and creates
// synthetic events for each that matches.
func (p *TimeTriggerPoller) checkTimeTriggers(ctx context.Context) {
	automations, err := p.workflowRepo.ListActiveTimeBased(ctx)
	if err != nil {
		slog.Error("failed to list time-based automations", "error", err)
		return
	}

	if len(automations) == 0 {
		return
	}

	slog.Debug("checking time-based triggers", "count", len(automations))

	for _, auto := range automations {
		// Dedup: skip if this automation was already executed in the last polling interval
		// for the same resource (check via automation_executions table)
		if p.wasRecentlyExecuted(ctx, auto.ID) {
			continue
		}

		// Create synthetic event payload for the time-based trigger
		evt := models.EventPayload{
			Type:      auto.TriggerType,
			ModuleID:  "automation",
			Timestamp: time.Now(),
		}

		// Execute in a goroutine with timeout
		go func(a *models.Automation, e models.EventPayload) {
			execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if execErr := p.engine.Execute(execCtx, *a, e); execErr != nil {
				slog.Error("time-based trigger execution failed",
					"automation_id", a.ID,
					"trigger_type", a.TriggerType,
					"error", execErr,
				)
			}
		}(auto, evt)
	}
}

// wasRecentlyExecuted checks if the automation was executed within the last polling interval.
func (p *TimeTriggerPoller) wasRecentlyExecuted(ctx context.Context, automationID uuid.UUID) bool {
	since := time.Now().Add(-p.interval)
	filter := workflow.ExecutionFilter{
		AutomationID: &automationID,
		StartedAfter: &since,
		Limit:        1,
	}
	execs, _, err := p.execRepo.ListExecutions(ctx, filter)
	if err != nil {
		slog.Warn("failed to check recent executions for dedup",
			"automation_id", automationID,
			"error", err,
		)
		return false
	}
	return len(execs) > 0
}
