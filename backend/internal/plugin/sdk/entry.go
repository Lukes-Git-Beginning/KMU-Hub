package sdk

import "unsafe"

// Plugin entry point boilerplate for WASM plugins.
// Plugins implement the HandleHook function and register it.

// unsafePointer returns an unsafe pointer to the given value
func unsafePointer(p *byte) unsafe.Pointer {
	return unsafe.Pointer(p)
}

// unsafeSlice creates a byte slice from a pointer and length in WASM memory
func unsafeSlice(ptr, length uint32) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

// Malloc exports a memory allocation function for the host to use.
// This is needed so the host can write data into the WASM module's memory.
//
//export malloc
func Malloc(size uint32) uint32 {
	buf := make([]byte, size)
	return uint32(uintptr(unsafe.Pointer(&buf[0])))
}

// Free is a no-op since Go's garbage collector handles deallocation.
//
//export free
func Free(_ uint32) {}
