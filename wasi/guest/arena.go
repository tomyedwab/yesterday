package guest

import (
	"unsafe"
)

var ARENA_PAGES map[uint32][]byte
var NEXT_ARENA_HANDLE uint32

func InitArena() {
	ARENA_PAGES = make(map[uint32][]byte)
	NEXT_ARENA_HANDLE = 1
}

//go:wasmexport alloc_page
func AllocPage() uint32 {
	bytes := make([]byte, 4096, 4096)
	handle := NEXT_ARENA_HANDLE
	NEXT_ARENA_HANDLE++
	ARENA_PAGES[handle] = bytes
	return uint32(uintptr(unsafe.Pointer(&bytes[0])))
}
