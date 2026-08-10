package hook

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin"
	"github.com/kmuhub/kmuhub/internal/plugin/config"
	"github.com/kmuhub/kmuhub/internal/plugin/wasm"
)

// Dispatcher finds active plugins and executes their hooks
type Dispatcher struct {
	service  *plugin.Service
	executor *wasm.PluginExecutor
}

// NewDispatcher creates a new hook dispatcher
func NewDispatcher(service *plugin.Service, executor *wasm.PluginExecutor) *Dispatcher {
	return &Dispatcher{
		service:  service,
		executor: executor,
	}
}

// HookResult represents the result of executing hooks
type HookResult struct {
	Modified     bool
	ModifiedData map[string]any
	Errors       []string
}

// ExecuteBeforeHooks executes before_* hooks sequentially (pipe pattern)
// Returns modified data if any plugin modified it
func (d *Dispatcher) ExecuteBeforeHooks(ctx context.Context, tenantID uuid.UUID, hookType, module, entityType, entityID string, data map[string]any) *HookResult {
	result := &HookResult{
		ModifiedData: data,
	}

	// Run config-based validation first
	validationErrors := d.service.ValidateEntity(ctx, tenantID, entityType, data)
	if len(validationErrors) > 0 {
		for _, ve := range validationErrors {
			result.Errors = append(result.Errors, ve.Message)
		}
		return result
	}

	// Find active installations for this hook
	installations, err := d.service.ListActiveInstallationsForHook(ctx, tenantID, hookType, module, entityType)
	if err != nil {
		slog.Warn("failed to list hook installations",
			"hook_type", hookType,
			"error", err,
		)
		return result
	}

	// Execute each plugin sequentially (pipe pattern: output feeds into next input)
	currentData := data
	for _, inst := range installations {
		manifest, manifestErr := d.service.GetManifestForInstallation(ctx, inst.ID)
		if manifestErr != nil {
			continue
		}

		if manifest.PluginType == models.PluginTypeWASM && d.executor != nil {
			hookReq := &wasm.HookRequest{
				HookType:   hookType,
				Module:     module,
				EntityType: entityType,
				EntityID:   entityID,
			}
			entityJSON, _ := json.Marshal(currentData)
			hookReq.EntityData = entityJSON
			hookReq.Settings = inst.Settings

			resp, logEntry, execErr := d.executor.ExecuteHook(ctx, manifest, inst, hookReq)
			if logEntry != nil {
				if logErr := d.service.LogExecution(ctx, logEntry); logErr != nil {
					slog.Warn("plugin execution log write failed", "plugin", manifest.Slug, "hook", hookType, "error", logErr)
				}
			}
			if execErr != nil {
				slog.Warn("plugin hook execution failed",
					"plugin", manifest.Slug,
					"hook", hookType,
					"error", execErr,
				)
				continue
			}
			if resp != nil && resp.Modified && resp.ModifiedData != nil {
				var modified map[string]any
				if unmarshalErr := json.Unmarshal(resp.ModifiedData, &modified); unmarshalErr == nil {
					currentData = modified
					result.Modified = true
				}
			}
		}
	}

	result.ModifiedData = currentData
	return result
}

// ExecuteAfterHooks executes after_* hooks in parallel (fire-and-forget)
func (d *Dispatcher) ExecuteAfterHooks(ctx context.Context, tenantID uuid.UUID, hookType, module, entityType, entityID string, data map[string]any) {
	installations, err := d.service.ListActiveInstallationsForHook(ctx, tenantID, hookType, module, entityType)
	if err != nil || len(installations) == 0 {
		return
	}

	// Evaluate workflow rules
	triggered := d.service.EvaluateWorkflows(ctx, tenantID, hookType+"_"+entityType, data)
	for _, action := range triggered {
		slog.Info("workflow rule triggered",
			"rule", action.RuleName,
			"action_type", action.Action.Type,
		)
		d.executeWorkflowAction(ctx, tenantID, action)
	}

	// Execute WASM plugin hooks in parallel
	var wg sync.WaitGroup
	for _, inst := range installations {
		inst := inst
		wg.Add(1)
		go func() {
			defer wg.Done()

			manifest, manifestErr := d.service.GetManifestForInstallation(ctx, inst.ID)
			if manifestErr != nil {
				return
			}

			if manifest.PluginType == models.PluginTypeWASM && d.executor != nil {
				hookReq := &wasm.HookRequest{
					HookType:   hookType,
					Module:     module,
					EntityType: entityType,
					EntityID:   entityID,
				}
				entityJSON, _ := json.Marshal(data)
				hookReq.EntityData = entityJSON
				hookReq.Settings = inst.Settings

				_, logEntry, execErr := d.executor.ExecuteHook(ctx, manifest, inst, hookReq)
				if logEntry != nil {
					if logErr := d.service.LogExecution(ctx, logEntry); logErr != nil {
						slog.Warn("plugin execution log write failed", "plugin", manifest.Slug, "hook", hookType, "error", logErr)
					}
				}
				if execErr != nil {
					slog.Warn("after hook execution failed",
						"plugin", manifest.Slug,
						"hook", hookType,
						"error", execErr,
					)
				}
			}
		}()
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		slog.Warn("after hooks timed out",
			"hook_type", hookType,
			"module", module,
			"entity_type", entityType,
		)
	}
}

func (d *Dispatcher) executeWorkflowAction(_ context.Context, _ uuid.UUID, action config.TriggeredAction) {
	switch action.Action.Type {
	case "create_task":
		slog.Info("workflow action: create_task",
			"rule", action.RuleName,
			"config", action.Action.Config,
		)
	case "send_notification":
		slog.Info("workflow action: send_notification",
			"rule", action.RuleName,
			"config", action.Action.Config,
		)
	case "set_field":
		slog.Info("workflow action: set_field",
			"rule", action.RuleName,
			"config", action.Action.Config,
		)
	case "log":
		slog.Info("workflow action: log",
			"rule", action.RuleName,
			"config", action.Action.Config,
		)
	default:
		slog.Warn("unknown workflow action type",
			"type", action.Action.Type,
			"rule", action.RuleName,
		)
	}
}
