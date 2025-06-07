package guest

import "github.com/tomyedwab/yesterday/wasi/types"

//go:wasmimport env init_module
func init_module(version string)

func Init(version string) {
	init_module(version)
	InitSQLProxy()
	InitBytes()
	InitRequests()
	InitEvents()

	RegisterHandler("/api/status", func(params types.RequestParams) types.Response {
		return RespondSuccess("ok")
	})
}
