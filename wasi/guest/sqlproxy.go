package guest

import (
	"fmt"
	"unsafe"

	sqlproxy "github.com/tomyedwab/yesterday/sqlproxy/driver"
)

//go:wasmimport env sqlite_host_handler
func sqlite_host_handler(requestPayload string, destPtr uint32) int32

func GetPtrAddress(destPtr *uint32) uint32 {
	return uint32(uintptr(unsafe.Pointer(destPtr)))
}

func InitSQLProxy() {
	sqlproxy.SetHostHandler(func(payload []byte) ([]byte, error) {
		var destPtr uint32
		destSize := sqlite_host_handler(string(payload), GetPtrAddress(&destPtr))
		if destSize < 0 {
			ret := GetBytesFromPtr(destPtr, uint32(-destSize))
			return nil, fmt.Errorf("sqlite_host_handler returned error: %s", string(ret))
		}
		return GetBytesFromPtr(destPtr, uint32(destSize)), nil
	})

}
