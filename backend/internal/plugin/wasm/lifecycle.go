package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// HookRequest represents a request sent to a WASM plugin hook
type HookRequest struct {
	HookType   string          `json:"hook_type"`
	Module     string          `json:"module"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	EntityData json.RawMessage `json:"entity_data"`
	Settings   json.RawMessage `json:"settings"`
}

// HookResponse represents a response from a WASM plugin hook
type HookResponse struct {
	Modified     bool            `json:"modified"`
	ModifiedData json.RawMessage `json:"modified_data,omitempty"`
	Error        string          `json:"error,omitempty"`
}

// PluginExecutor manages the lifecycle of WASM plugin executions
type PluginExecutor struct {
	runtime *Runtime
	hostAPI *HostAPI
}

// NewPluginExecutor creates a new plugin executor
func NewPluginExecutor(runtime *Runtime, hostAPI *HostAPI) *PluginExecutor {
	return &PluginExecutor{
		runtime: runtime,
		hostAPI: hostAPI,
	}
}

// LoadPlugin compiles a plugin's WASM binary and caches it
func (e *PluginExecutor) LoadPlugin(ctx context.Context, manifest *models.PluginManifest, wasmBinary []byte) error {
	if len(wasmBinary) == 0 {
		return fmt.Errorf("empty WASM binary for plugin %s", manifest.Slug)
	}

	_, err := e.runtime.CompileModule(ctx, manifest.Slug, wasmBinary)
	if err != nil {
		return fmt.Errorf("compile plugin %s: %w", manifest.Slug, err)
	}

	slog.Info("plugin loaded",
		"plugin", manifest.Slug,
		"version", manifest.Version,
	)

	return nil
}

// ExecuteHook runs a plugin's hook handler within a sandbox
func (e *PluginExecutor) ExecuteHook(ctx context.Context, manifest *models.PluginManifest, installation *models.PluginInstallation, request *HookRequest) (*HookResponse, *models.PluginExecutionLog, error) {
	start := time.Now()
	logEntry := &models.PluginExecutionLog{
		InstallationID: installation.ID,
		HookType:       request.HookType,
		Module:         request.Module,
		EntityType:     request.EntityType,
		Status:         models.ExecutionStatusSuccess,
	}
	if request.EntityID != "" {
		id, _ := uuid.Parse(request.EntityID)
		logEntry.EntityID = &id
	}

	// Get compiled module
	e.runtime.mu.RLock()
	compiled, ok := e.runtime.cache[manifest.Slug]
	e.runtime.mu.RUnlock()

	if !ok {
		logEntry.Status = models.ExecutionStatusError
		errMsg := "plugin not loaded"
		logEntry.ErrorMessage = &errMsg
		logEntry.DurationMs = int(time.Since(start).Milliseconds())
		return nil, logEntry, fmt.Errorf("plugin %s not loaded", manifest.Slug)
	}

	// Register host functions for this execution
	if err := e.hostAPI.RegisterHostFunctions(ctx, e.runtime, installation.ID); err != nil {
		logEntry.Status = models.ExecutionStatusError
		errMsg := err.Error()
		logEntry.ErrorMessage = &errMsg
		logEntry.DurationMs = int(time.Since(start).Milliseconds())
		return nil, logEntry, fmt.Errorf("register host functions: %w", err)
	}

	// Create sandbox and execute
	sandbox := NewSandbox(e.runtime, compiled, DefaultSandboxConfig())
	results, err := sandbox.Execute(ctx, "handle_hook")
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logEntry.Status = models.ExecutionStatusTimeout
		} else {
			logEntry.Status = models.ExecutionStatusError
		}
		errMsg := err.Error()
		logEntry.ErrorMessage = &errMsg
		logEntry.DurationMs = int(time.Since(start).Milliseconds())
		return nil, logEntry, err
	}

	// Parse response from WASM return value
	response := &HookResponse{Modified: false}
	if len(results) > 0 && results[0] != 0 {
		response.Modified = true
	}

	logEntry.DurationMs = int(time.Since(start).Milliseconds())

	slog.Info("plugin hook executed",
		"plugin", manifest.Slug,
		"hook", request.HookType,
		"duration_ms", logEntry.DurationMs,
		"modified", response.Modified,
	)

	return response, logEntry, nil
}

// UnloadPlugin removes a plugin from the compilation cache
func (e *PluginExecutor) UnloadPlugin(slug string) {
	e.runtime.InvalidateCache(slug)
	slog.Info("plugin unloaded", "plugin", slug)
}

// Close shuts down the executor
func (e *PluginExecutor) Close(ctx context.Context) error {
	return e.runtime.Close(ctx)
}
