package sdk

// Host function imports for WASM plugins.
// These are wrappers around the host functions provided by the kmuhub module.
// Plugins compiled to WASM import these functions.

//go:wasmimport kmuhub kv_get
func _kvGet(keyPtr, keyLen uint32) uint64

//go:wasmimport kmuhub kv_set
func _kvSet(keyPtr, keyLen, valPtr, valLen uint32) uint32

//go:wasmimport kmuhub kv_delete
func _kvDelete(keyPtr, keyLen uint32) uint32

//go:wasmimport kmuhub log_info
func _logInfo(msgPtr, msgLen uint32)

//go:wasmimport kmuhub log_warn
func _logWarn(msgPtr, msgLen uint32)

//go:wasmimport kmuhub log_error
func _logError(msgPtr, msgLen uint32)

//go:wasmimport kmuhub config_get
func _configGet(keyPtr, keyLen uint32) uint64

// KVGet retrieves a value from the plugin's KV store
func KVGet(key string) ([]byte, bool) {
	keyBytes := []byte(key)
	result := _kvGet(bytesToPtr(keyBytes), uint32(len(keyBytes)))
	if result == 0 {
		return nil, false
	}
	ptr := uint32(result >> 32)
	length := uint32(result & 0xFFFFFFFF)
	return ptrToBytes(ptr, length), true
}

// KVSet stores a value in the plugin's KV store
func KVSet(key string, value []byte) error {
	keyBytes := []byte(key)
	result := _kvSet(bytesToPtr(keyBytes), uint32(len(keyBytes)), bytesToPtr(value), uint32(len(value)))
	if result != 0 {
		return ErrHostCallFailed
	}
	return nil
}

// KVDelete removes a value from the plugin's KV store
func KVDelete(key string) error {
	keyBytes := []byte(key)
	result := _kvDelete(bytesToPtr(keyBytes), uint32(len(keyBytes)))
	if result != 0 {
		return ErrHostCallFailed
	}
	return nil
}

// LogInfo logs an info message via the host
func LogInfo(msg string) {
	msgBytes := []byte(msg)
	_logInfo(bytesToPtr(msgBytes), uint32(len(msgBytes)))
}

// LogWarn logs a warning message via the host
func LogWarn(msg string) {
	msgBytes := []byte(msg)
	_logWarn(bytesToPtr(msgBytes), uint32(len(msgBytes)))
}

// LogError logs an error message via the host
func LogError(msg string) {
	msgBytes := []byte(msg)
	_logError(bytesToPtr(msgBytes), uint32(len(msgBytes)))
}

// ConfigGet retrieves a configuration value for the plugin
func ConfigGet(key string) ([]byte, bool) {
	keyBytes := []byte(key)
	result := _configGet(bytesToPtr(keyBytes), uint32(len(keyBytes)))
	if result == 0 {
		return nil, false
	}
	ptr := uint32(result >> 32)
	length := uint32(result & 0xFFFFFFFF)
	return ptrToBytes(ptr, length), true
}

// ErrHostCallFailed is returned when a host function call fails
var ErrHostCallFailed = hostError("host call failed")

type hostError string

func (e hostError) Error() string { return string(e) }

// bytesToPtr converts a byte slice to a pointer (for WASM ABI)
func bytesToPtr(b []byte) uint32 {
	if len(b) == 0 {
		return 0
	}
	return uint32(uintptr(unsafePointer(&b[0])))
}

// ptrToBytes reads bytes from WASM memory at the given pointer
func ptrToBytes(ptr, length uint32) []byte {
	if length == 0 || ptr == 0 {
		return nil
	}
	buf := make([]byte, length)
	src := unsafeSlice(ptr, length)
	copy(buf, src)
	return buf
}
