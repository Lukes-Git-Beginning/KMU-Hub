package trigger

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

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
	resolver     DueResolver

	// timeBasedTypes is the set of TriggerDefinition.Type values with
	// TimeBased=true, computed once from registry at construction. The
	// registry only registers built-ins at startup (see
	// TriggerRegistry.registerBuiltins), so this does not go stale.
	timeBasedTypes []string
}

// NewTimeTriggerPoller creates a new time-based trigger poller. resolver
// decides which entities make a trigger due; without one the poller has no
// way to distinguish "the invoice went overdue" from "five minutes elapsed"
// and would fire every active time-based automation on every tick.
func NewTimeTriggerPoller(
	repo workflow.Repository,
	engine EngineExecutor,
	resolver DueResolver,
	registry *TriggerRegistry,
) *TimeTriggerPoller {
	return &TimeTriggerPoller{
		interval:       5 * time.Minute,
		workflowRepo:   repo,
		engine:         engine,
		resolver:       resolver,
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

		p.fireDueEntities(ctx, auto, now)
	}
}

// fireDueEntities resolves which entities make auto's trigger due and executes
// auto once per entity that has not fired for that entity before.
//
// Resolving is what makes the trigger time-BASED rather than time-DRIVEN: the
// tick only decides when to look, the resolver decides whether there is
// anything to fire for. An empty result means nothing is due and nothing runs.
func (p *TimeTriggerPoller) fireDueEntities(ctx context.Context, auto *models.Automation, now time.Time) {
	entities, err := p.resolver.Resolve(ctx, auto.TriggerType, auto.TenantID, now)
	if err != nil {
		// Includes ErrUnknownTimeTrigger: a trigger flagged TimeBased in the
		// registry but never taught to the resolver is a wiring bug, and a
		// loud one beats an automation that quietly never runs.
		slog.Error("failed to resolve due entities for time-based trigger",
			"automation_id", auto.ID,
			"trigger_type", auto.TriggerType,
			"error", err,
		)
		return
	}
	if len(entities) == 0 {
		return
	}
	if len(entities) >= maxDueEntitiesPerAutomation {
		slog.Warn("due entities hit the per-automation cap, remainder deferred to the next tick",
			"automation_id", auto.ID,
			"trigger_type", auto.TriggerType,
			"cap", maxDueEntitiesPerAutomation,
		)
	}

	for _, entity := range entities {
		fired, fireErr := p.workflowRepo.ClaimTimeTriggerFire(ctx, auto.TenantID, auto.ID, entity.Key, now)
		if fireErr != nil {
			slog.Error("failed to claim time-based trigger fire",
				"automation_id", auto.ID,
				"entity_key", entity.Key,
				"error", fireErr,
			)
			continue
		}
		if !fired {
			// Already fired for this entity -- by an earlier tick, or by
			// another instance racing this one.
			continue
		}

		payload, marshalErr := json.Marshal(entity.Payload)
		if marshalErr != nil {
			slog.Error("failed to marshal due entity payload",
				"automation_id", auto.ID,
				"entity_key", entity.Key,
				"error", marshalErr,
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
			Type:       auto.TriggerType,
			TenantID:   auto.TenantID,
			ModuleID:   "scheduler",
			ResourceID: entity.ResourceID,
			Payload:    payload,
			Timestamp:  now,
		}

		// Execute in a goroutine with timeout
		go func(a *models.Automation, e models.EventPayload) {
			execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if execErr := p.engine.Execute(execCtx, *a, e); execErr != nil {
				slog.Error("time-based trigger execution failed",
					"automation_id", a.ID,
					"trigger_type", a.TriggerType,
					"resource_id", e.ResourceID,
					"error", execErr,
				)
			}
		}(auto, evt)
	}
}
