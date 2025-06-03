package guest

import (
	"fmt"

	sqlproxy "github.com/tomyedwab/yesterday/sqlproxy/driver"
)

//go:wasmimport env sqlite_host_handler
func sqlite_host_handler(requestPayload string) (responseHandle uint64)

func InitSQLProxy() {
	sqlproxy.SetHostHandler(func(payload []byte) ([]byte, error) {
		responseHandle := sqlite_host_handler(string(payload))
		ret := GetBytes(uint32(responseHandle))
		FreeBytes(uint32(responseHandle))
		if responseHandle>>32 != 0 {
			return nil, fmt.Errorf("sqlite_host_handler returned error: %s", string(ret))
		}
		return ret, nil
	})

}
