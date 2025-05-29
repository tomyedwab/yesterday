package wasi

import "unsafe"

var BYTE_HANDLES map[uint32][]byte
var NEXT_BYTE_HANDLE uint32

func InitBytes() {
	BYTE_HANDLES = make(map[uint32][]byte)
	NEXT_BYTE_HANDLE = 1
}

//go:wasmexport alloc_bytes
func AllocBytes(size uint32) uint64 {
	bytes := make([]byte, size)
	handle := NEXT_BYTE_HANDLE
	BYTE_HANDLES[handle] = bytes
	NEXT_BYTE_HANDLE++
	return uint64(uint32(handle))<<32 | uint64(uintptr(unsafe.Pointer(&bytes[0])))
}

//go:wasmexport free_bytes
func FreeBytes(handle uint32) {
	delete(BYTE_HANDLES, handle)
}

func GetBytes(handle uint32) []byte {
	return BYTE_HANDLES[handle]
}
