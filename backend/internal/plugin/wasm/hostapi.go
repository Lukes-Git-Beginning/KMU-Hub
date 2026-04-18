//go:build !no_wasm
// +build !no_wasm

package wasm

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/tetratelabs/wazero/api"

	"github.com/kmuhub/kmuhub/internal/plugin/repository"
)

// HostAPI provides host functions that WASM plugins can call
type HostAPI struct {
	kvStore     *repository.KVStoreRepository
	permissions *repository.PermissionRepository
	rateLimiter *RateLimiter
}

// NewHostAPI creates a new host API
func NewHostAPI(kvStore *repository.KVStoreRepository, permissions *repository.PermissionRepository, rateLimiter *RateLimiter) *HostAPI {
	return &HostAPI{
		kvStore:     kvStore,
		permissions: permissions,
		rateLimiter: rateLimiter,
	}
}

// RegisterHostFunctions registers all host functions with the WASM runtime
func (h *HostAPI) RegisterHostFunctions(ctx context.Context, runtime *Runtime, installationID uuid.UUID) error {
	builder := runtime.Engine().NewHostModuleBuilder("kmuhub")

	// KV Store functions
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, keyLen uint32) uint64 {
			return h.hostKVGet(ctx, m, installationID, keyPtr, keyLen)
		}).
		Export("kv_get")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, keyLen, valPtr, valLen uint32) uint32 {
			return h.hostKVSet(ctx, m, installationID, keyPtr, keyLen, valPtr, valLen)
		}).
		Export("kv_set")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, keyLen uint32) uint32 {
			return h.hostKVDelete(ctx, m, installationID, keyPtr, keyLen)
		}).
		Export("kv_delete")

	// Logging functions
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, msgPtr, msgLen uint32) {
			h.hostLogInfo(m, installationID, msgPtr, msgLen)
		}).
		Export("log_info")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, msgPtr, msgLen uint32) {
			h.hostLogWarn(m, installationID, msgPtr, msgLen)
		}).
		Export("log_warn")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, msgPtr, msgLen uint32) {
			h.hostLogError(m, installationID, msgPtr, msgLen)
		}).
		Export("log_error")

	// Config function
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, keyLen uint32) uint64 {
			return h.hostConfigGet(ctx, m, installationID, keyPtr, keyLen)
		}).
		Export("config_get")

	_, err := builder.Instantiate(ctx)
	return err
}

// ============================================================================
// KV Store Host Functions
// ============================================================================

func (h *HostAPI) hostKVGet(ctx context.Context, m api.Module, installationID uuid.UUID, keyPtr, keyLen uint32) uint64 {
	if !h.rateLimiter.Allow(installationID.String()) {
		return 0
	}

	key := readString(m, keyPtr, keyLen)
	if key == "" {
		return 0
	}

	value, found, err := h.kvStore.Get(ctx, installationID, key)
	if err != nil || !found {
		return 0
	}

	// Write value to module memory (ptr << 32 | len)
	return writeToMemory(m, value)
}

func (h *HostAPI) hostKVSet(ctx context.Context, m api.Module, installationID uuid.UUID, keyPtr, keyLen, valPtr, valLen uint32) uint32 {
	if !h.rateLimiter.Allow(installationID.String()) {
		return 1
	}

	key := readString(m, keyPtr, keyLen)
	val := readBytes(m, valPtr, valLen)
	if key == "" || val == nil {
		return 1
	}

	if err := h.kvStore.Set(ctx, installationID, key, json.RawMessage(val)); err != nil {
		return 1
	}
	return 0
}

func (h *HostAPI) hostKVDelete(ctx context.Context, m api.Module, installationID uuid.UUID, keyPtr, keyLen uint32) uint32 {
	if !h.rateLimiter.Allow(installationID.String()) {
		return 1
	}

	key := readString(m, keyPtr, keyLen)
	if key == "" {
		return 1
	}

	if err := h.kvStore.Delete(ctx, installationID, key); err != nil {
		return 1
	}
	return 0
}

// ============================================================================
// Logging Host Functions
// ============================================================================

func (h *HostAPI) hostLogInfo(m api.Module, installationID uuid.UUID, msgPtr, msgLen uint32) {
	msg := readString(m, msgPtr, msgLen)
	slog.Info("plugin log",
		"installation_id", installationID,
		"message", msg,
	)
}

func (h *HostAPI) hostLogWarn(m api.Module, installationID uuid.UUID, msgPtr, msgLen uint32) {
	msg := readString(m, msgPtr, msgLen)
	slog.Warn("plugin log",
		"installation_id", installationID,
		"message", msg,
	)
}

func (h *HostAPI) hostLogError(m api.Module, installationID uuid.UUID, msgPtr, msgLen uint32) {
	msg := readString(m, msgPtr, msgLen)
	slog.Error("plugin log",
		"installation_id", installationID,
		"message", msg,
	)
}

// ============================================================================
// Config Host Functions
// ============================================================================

func (h *HostAPI) hostConfigGet(ctx context.Context, m api.Module, installationID uuid.UUID, keyPtr, keyLen uint32) uint64 {
	if !h.rateLimiter.Allow(installationID.String()) {
		return 0
	}

	key := readString(m, keyPtr, keyLen)
	if key == "" {
		return 0
	}

	// Plugin settings are stored in the KV store under a special prefix
	value, found, err := h.kvStore.Get(ctx, installationID, "_config:"+key)
	if err != nil || !found {
		return 0
	}

	return writeToMemory(m, value)
}

// ============================================================================
// Memory Helpers
// ============================================================================

func readString(m api.Module, ptr, length uint32) string {
	if length == 0 {
		return ""
	}
	mem := m.Memory()
	if mem == nil {
		return ""
	}
	data, ok := mem.Read(ptr, length)
	if !ok {
		return ""
	}
	return string(data)
}

func readBytes(m api.Module, ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	mem := m.Memory()
	if mem == nil {
		return nil
	}
	data, ok := mem.Read(ptr, length)
	if !ok {
		return nil
	}
	result := make([]byte, len(data))
	copy(result, data)
	return result
}

func writeToMemory(m api.Module, data []byte) uint64 {
	mem := m.Memory()
	if mem == nil {
		return 0
	}

	// Use the module's malloc if available, otherwise write at a known offset
	mallocFn := m.ExportedFunction("malloc")
	if mallocFn == nil {
		return 0
	}

	results, err := mallocFn.Call(context.Background(), uint64(len(data)))
	if err != nil || len(results) == 0 {
		return 0
	}

	ptr := uint32(results[0])
	if !mem.Write(ptr, data) {
		return 0
	}

	// Return ptr and length packed as uint64 (ptr << 32 | len)
	return (uint64(ptr) << 32) | uint64(len(data))
}
