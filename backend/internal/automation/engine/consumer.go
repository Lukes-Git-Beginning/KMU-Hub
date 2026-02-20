package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/kmuhub/kmuhub/internal/automation/trigger"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

const (
	// executionTimeout is the maximum time for a single automation execution
	// triggered by an event.
	executionTimeout = 30 * time.Second
)

// AutomationConsumer listens on the EventBus and dispatches matching
// automations to the WorkflowEngine for execution.
type AutomationConsumer struct {
	matcher *trigger.TriggerMatcher
	engine  *WorkflowEngine
}

// NewAutomationConsumer creates a new automation consumer.
func NewAutomationConsumer(matcher *trigger.TriggerMatcher, engine *WorkflowEngine) *AutomationConsumer {
	return &AutomationConsumer{
		matcher: matcher,
		engine:  engine,
	}
}

// HandleEvent processes an event from the EventBus.
// It finds matching automations via the TriggerMatcher and executes each
// in a separate goroutine with a timeout.
//
// This method is intended to be registered as a wildcard handler on the EventBus:
//
//	bus.RegisterHandler("*", consumer.HandleEvent)
func (ac *AutomationConsumer) HandleEvent(ctx context.Context, evt models.EventPayload) error {
	// Loop prevention: skip events from the automation module itself
	if evt.ModuleID == event.ModuleAutomation {
		return nil
	}

	matches, err := ac.matcher.FindMatching(ctx, evt.Type)
	if err != nil {
		slog.Error("failed to find matching automations",
			"event_type", evt.Type,
			"error", err,
		)
		return nil // Non-blocking: errors are logged, not propagated
	}

	if len(matches) == 0 {
		return nil
	}

	slog.Debug("matched automations for event",
		"event_type", evt.Type,
		"match_count", len(matches),
	)

	for _, match := range matches {
		// Execute each automation in a goroutine with timeout
		go func(auto models.Automation, e models.EventPayload) {
			execCtx, cancel := context.WithTimeout(context.Background(), executionTimeout)
			defer cancel()

			if execErr := ac.engine.Execute(execCtx, auto, e); execErr != nil {
				slog.Error("automation execution failed",
					"automation_id", auto.ID,
					"event_type", e.Type,
					"error", execErr,
				)
			}
		}(*match.Automation, evt)
	}

	return nil
}
