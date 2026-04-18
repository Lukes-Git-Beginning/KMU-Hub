//go:build !no_wasm
// +build !no_wasm

package wasm

import (
	"context"
	"fmt"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// SandboxConfig defines the resource limits for WASM plugin execution
type SandboxConfig struct {
	MaxMemoryMB    int           // Maximum memory in MB (default: 64)
	MaxExecTime    time.Duration // Maximum execution time (default: 5s)
	MaxStackDepth  int           // Maximum stack depth (default: 1024)
}

// DefaultSandboxConfig returns the default sandbox configuration
func DefaultSandboxConfig() SandboxConfig {
	return SandboxConfig{
		MaxMemoryMB:   64,
		MaxExecTime:   5 * time.Second,
		MaxStackDepth: 1024,
	}
}

// Sandbox wraps a WASM module instance with resource limits
type Sandbox struct {
	config   SandboxConfig
	runtime  *Runtime
	module   api.Module
	compiled wazero.CompiledModule
}

// NewSandbox creates a sandboxed WASM execution environment
func NewSandbox(runtime *Runtime, compiled wazero.CompiledModule, cfg SandboxConfig) *Sandbox {
	return &Sandbox{
		config:   cfg,
		runtime:  runtime,
		compiled: compiled,
	}
}

// Execute runs a WASM function within the sandbox with timeout enforcement
func (s *Sandbox) Execute(ctx context.Context, funcName string, params ...uint64) ([]uint64, error) {
	// Apply timeout
	execCtx, cancel := context.WithTimeout(ctx, s.config.MaxExecTime)
	defer cancel()

	// Create a fresh module instance for this execution
	modCfg := wazero.NewModuleConfig().
		WithName("").
		WithStartFunctions()

	mod, err := s.runtime.InstantiateModule(execCtx, s.compiled, modCfg)
	if err != nil {
		return nil, fmt.Errorf("instantiate sandbox: %w", err)
	}
	defer mod.Close(execCtx)

	s.module = mod

	// Find and call the function
	fn := mod.ExportedFunction(funcName)
	if fn == nil {
		return nil, fmt.Errorf("function %q not found in WASM module", funcName)
	}

	results, err := fn.Call(execCtx, params...)
	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("execution timeout (%v)", s.config.MaxExecTime)
		}
		return nil, fmt.Errorf("execution error: %w", err)
	}

	return results, nil
}

// ReadMemory reads bytes from the WASM module's linear memory
func (s *Sandbox) ReadMemory(offset, length uint32) ([]byte, bool) {
	if s.module == nil {
		return nil, false
	}
	mem := s.module.Memory()
	if mem == nil {
		return nil, false
	}
	return mem.Read(offset, length)
}

// WriteMemory writes bytes to the WASM module's linear memory
func (s *Sandbox) WriteMemory(offset uint32, data []byte) bool {
	if s.module == nil {
		return false
	}
	mem := s.module.Memory()
	if mem == nil {
		return false
	}
	return mem.Write(offset, data)
}

// Malloc allocates memory in the WASM module (calls the module's malloc export)
func (s *Sandbox) Malloc(ctx context.Context, size uint32) (uint32, error) {
	if s.module == nil {
		return 0, fmt.Errorf("module not instantiated")
	}
	mallocFn := s.module.ExportedFunction("malloc")
	if mallocFn == nil {
		return 0, fmt.Errorf("module does not export malloc")
	}
	results, err := mallocFn.Call(ctx, uint64(size))
	if err != nil {
		return 0, fmt.Errorf("malloc failed: %w", err)
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("malloc returned no results")
	}
	return uint32(results[0]), nil
}
