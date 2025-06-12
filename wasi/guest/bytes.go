package guest

import (
	"unsafe"
)

func GetBytesFromPtr(ptr uint32, size uint32) []byte {
	if size == 0 {
		return nil
	}
	// Cast the pointer to a slice using unsafe operations
	slice := (*[1 << 30]byte)(unsafe.Pointer(uintptr(ptr)))[:size:size]
	// Create a new array and copy the bytes
	result := make([]byte, size)
	copy(result, slice)
	// TODO(tom) STOPSHIP free ptr
	return result
}
