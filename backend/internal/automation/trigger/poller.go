package trigger

import (
	"context"
	"log/slog"
	"time"

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
	engine       EngineExecutor
	pool         *pgxpool.Pool

	// timeBasedTypes is the set of TriggerDefinition.Type values with
	// TimeBased=true, computed once from registry at construction. The
	// registry only registers built-ins at startup (see
	// TriggerRegistry.registerBuiltins), so this does not go stale.
	timeBasedTypes []string
}

// NewTimeTriggerPoller creates a new time-based trigger poller.
func NewTimeTriggerPoller(
	repo workflow.Repository,
	engine EngineExecutor,
	pool *pgxpool.Pool,
	registry *TriggerRegistry,
) *TimeTriggerPoller {
	return &TimeTriggerPoller{
		interval:       5 * time.Minute,
		workflowRepo:   repo,
		engine:         engine,
		pool:           pool,
		timeBasedTypes: timeBasedTriggerTypes(registry),
	}
}

// timeBasedTriggerTypes collects the Type of every registered
// TriggerDefinition with TimeBased=true.
func timeBasedTriggerTypes(registry *TriggerRegistry) []string {
	var types []string
	for _, def := range registry.All() {
		if def.TimeBased {
			types = append(types, def.Type)
		}
	}
	return types
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
	if len(p.timeBasedTypes) == 0 {
		return
	}

	automations, err := p.workflowRepo.ListActiveTimeBased(ctx, p.timeBasedTypes)
	if err != nil {
		slog.Error("failed to list time-based automations", "error", err)
		return
	}

	if len(automations) == 0 {
		return
	}

	slog.Debug("checking time-based triggers", "count", len(automations))

	now := time.Now()
	for _, auto := range automations {
		// Atomic claim: last_polled_at only advances if it still matches the
		// value just read above. Two TimeTriggerPoller instances (or two
		// overlapping ticks) racing on the same automation will have exactly
		// one Exec affect a row -- the other observes 0 rows and skips,
		// exactly mirroring berichte/scheduler.ClaimSchedule. Replaces the
		// former history-query dedup (workflowRepo.ListExecutions lookback),
		// which was a check-then-act race between separate instances.
		claimed, claimErr := p.workflowRepo.ClaimTimeTrigger(ctx, auto.ID, auto.LastPolledAt, now)
		if claimErr != nil {
			slog.Error("failed to claim time-based trigger",
				"automation_id", auto.ID,
				"error", claimErr,
			)
			continue
		}
		if !claimed {
			slog.Debug("time-based trigger claim lost to another tick/instance",
				"automation_id", auto.ID,
			)
			continue
		}

		// Create synthetic event payload for the time-based trigger.
		//
		// ModuleID must NOT be "automation" (event.ModuleAutomation): engine.Execute's
		// loop-prevention check drops any event carrying that exact value before
		// evaluating conditions or running actions (see engine.go), so a synthetic
		// event tagged that way is silently discarded here -- neither
		// biz.invoice.overdue nor calendar.event.upcoming ever actually executed.
		// "scheduler" identifies the poller as the origin without colliding with
		// the sentinel.
		evt := models.EventPayload{
			Type:      auto.TriggerType,
			ModuleID:  "scheduler",
			Timestamp: now,
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
