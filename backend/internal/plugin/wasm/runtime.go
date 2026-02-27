package wasm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Runtime manages the wazero WASM runtime and module compilation cache
type Runtime struct {
	engine wazero.Runtime
	cache  map[string]wazero.CompiledModule
	mu     sync.RWMutex
}

// NewRuntime creates a new WASM runtime with default configuration
func NewRuntime(ctx context.Context) (*Runtime, error) {
	cfg := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(1024). // 64MB max (1024 pages * 64KB)
		WithCloseOnContextDone(true)

	engine := wazero.NewRuntimeWithConfig(ctx, cfg)

	// Instantiate WASI for basic I/O
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, engine); err != nil {
		_ = engine.Close(ctx)
		return nil, fmt.Errorf("instantiate wasi: %w", err)
	}

	return &Runtime{
		engine: engine,
		cache:  make(map[string]wazero.CompiledModule),
	}, nil
}

// CompileModule compiles and caches a WASM binary
func (r *Runtime) CompileModule(ctx context.Context, name string, wasmBytes []byte) (wazero.CompiledModule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check cache first
	if cached, ok := r.cache[name]; ok {
		return cached, nil
	}

	compiled, err := r.engine.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("compile module %s: %w", name, err)
	}

	r.cache[name] = compiled

	slog.Info("wasm module compiled",
		"name", name,
		"size_bytes", len(wasmBytes),
	)

	return compiled, nil
}

// InvalidateCache removes a compiled module from the cache
func (r *Runtime) InvalidateCache(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cache, name)
}

// Engine returns the underlying wazero runtime for host module registration
func (r *Runtime) Engine() wazero.Runtime {
	return r.engine
}

// InstantiateModule creates a new instance of a compiled module with the given config
func (r *Runtime) InstantiateModule(ctx context.Context, compiled wazero.CompiledModule, modCfg wazero.ModuleConfig) (api.Module, error) {
	return r.engine.InstantiateModule(ctx, compiled, modCfg)
}

// Close shuts down the runtime and frees resources
func (r *Runtime) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = nil
	return r.engine.Close(ctx)
}
