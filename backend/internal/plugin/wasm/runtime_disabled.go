//go:build no_wasm
// +build no_wasm

// Package wasm provides a disabled stub for production builds compiled with -tags no_wasm.
// All types and constructors are present but inert: NewRuntime returns nil,
// and NewPluginExecutor accepts nil arguments — the hook dispatcher checks for nil
// before invoking any WASM execution path.
package wasm

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin/repository"
)

// ── Types that are normally defined in wazero-importing files ────────────────

// HookRequest represents a request sent to a WASM plugin hook (stub).
type HookRequest struct {
	HookType   string          `json:"hook_type"`
	Module     string          `json:"module"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	EntityData json.RawMessage `json:"entity_data"`
	Settings   json.RawMessage `json:"settings"`
}

// HookResponse represents a response from a WASM plugin hook (stub).
type HookResponse struct {
	Modified     bool            `json:"modified"`
	ModifiedData json.RawMessage `json:"modified_data,omitempty"`
	Error        string          `json:"error,omitempty"`
}

// Runtime is a disabled WASM runtime stub.
type Runtime struct{}

// NewRuntime returns nil — WASM support is compiled out in this build.
func NewRuntime(_ context.Context) (*Runtime, error) {
	return nil, nil
}

// Close is a no-op on the disabled runtime.
func (r *Runtime) Close(_ context.Context) error { return nil }

// HostAPI is a disabled host-API stub.
type HostAPI struct{}

// NewHostAPI returns a no-op HostAPI stub.
func NewHostAPI(_ *repository.KVStoreRepository, _ *repository.PermissionRepository, _ *RateLimiter) *HostAPI {
	return &HostAPI{}
}

// PluginExecutor is a disabled executor stub — ExecuteHook always returns nil.
type PluginExecutor struct{}

// NewPluginExecutor returns a no-op PluginExecutor stub.
func NewPluginExecutor(_ *Runtime, _ *HostAPI) *PluginExecutor {
	return &PluginExecutor{}
}

// LoadPlugin is a no-op on the disabled executor.
func (e *PluginExecutor) LoadPlugin(_ context.Context, _ *models.PluginManifest, _ []byte) error {
	return nil
}

// ExecuteHook is a no-op on the disabled executor — returns nil response.
func (e *PluginExecutor) ExecuteHook(_ context.Context, _ *models.PluginManifest, _ *models.PluginInstallation, _ *HookRequest) (*HookResponse, *models.PluginExecutionLog, error) {
	return nil, nil, nil
}

// UnloadPlugin is a no-op on the disabled executor.
func (e *PluginExecutor) UnloadPlugin(_ string) {}

// Close is a no-op on the disabled executor.
func (e *PluginExecutor) Close(_ context.Context) error { return nil }

// SandboxConfig is a no-op config type for the disabled build.
type SandboxConfig struct{}

// DefaultSandboxConfig returns an empty config.
func DefaultSandboxConfig() SandboxConfig { return SandboxConfig{} }

// MemoryManager is a no-op stub.
type MemoryManager struct{}

// ── uuid import used only to keep the type signatures consistent ─────────────
var _ = uuid.UUID{}
