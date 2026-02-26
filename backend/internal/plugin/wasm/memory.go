package wasm

import (
	"encoding/binary"
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

// MemoryManager provides safe memory access patterns for WASM modules
type MemoryManager struct {
	module api.Module
}

// NewMemoryManager creates a new memory manager for a WASM module
func NewMemoryManager(module api.Module) *MemoryManager {
	return &MemoryManager{module: module}
}

// ReadString reads a string from WASM memory at the given pointer and length
func (mm *MemoryManager) ReadString(ptr, length uint32) (string, error) {
	mem := mm.module.Memory()
	if mem == nil {
		return "", fmt.Errorf("module has no memory")
	}
	data, ok := mem.Read(ptr, length)
	if !ok {
		return "", fmt.Errorf("failed to read %d bytes at offset %d", length, ptr)
	}
	return string(data), nil
}

// WriteString writes a string to WASM memory, returning the pointer and length
func (mm *MemoryManager) WriteString(s string) (uint32, uint32, error) {
	data := []byte(s)
	ptr, err := mm.Allocate(uint32(len(data)))
	if err != nil {
		return 0, 0, err
	}
	mem := mm.module.Memory()
	if !mem.Write(ptr, data) {
		return 0, 0, fmt.Errorf("failed to write %d bytes at offset %d", len(data), ptr)
	}
	return ptr, uint32(len(data)), nil
}

// ReadUint32 reads a uint32 from WASM memory
func (mm *MemoryManager) ReadUint32(ptr uint32) (uint32, error) {
	mem := mm.module.Memory()
	if mem == nil {
		return 0, fmt.Errorf("module has no memory")
	}
	data, ok := mem.Read(ptr, 4)
	if !ok {
		return 0, fmt.Errorf("failed to read uint32 at offset %d", ptr)
	}
	return binary.LittleEndian.Uint32(data), nil
}

// WriteUint32 writes a uint32 to WASM memory
func (mm *MemoryManager) WriteUint32(ptr, value uint32) error {
	mem := mm.module.Memory()
	if mem == nil {
		return fmt.Errorf("module has no memory")
	}
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, value)
	if !mem.Write(ptr, data) {
		return fmt.Errorf("failed to write uint32 at offset %d", ptr)
	}
	return nil
}

// Allocate allocates memory in the WASM module using its exported malloc function
func (mm *MemoryManager) Allocate(size uint32) (uint32, error) {
	mallocFn := mm.module.ExportedFunction("malloc")
	if mallocFn == nil {
		return 0, fmt.Errorf("module does not export malloc")
	}
	results, err := mallocFn.Call(nil, uint64(size))
	if err != nil {
		return 0, fmt.Errorf("malloc(%d) failed: %w", size, err)
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("malloc returned no results")
	}
	return uint32(results[0]), nil
}

// Free frees memory in the WASM module using its exported free function
func (mm *MemoryManager) Free(ptr uint32) {
	freeFn := mm.module.ExportedFunction("free")
	if freeFn == nil {
		return
	}
	_, _ = freeFn.Call(nil, uint64(ptr))
}
