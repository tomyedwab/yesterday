package guest

import "github.com/tomyedwab/yesterday/wasi/types"

func Init() {
	InitSQLProxy()
	InitBytes()
	InitRequests()
	InitEvents()

	RegisterHandler("/api/status", func(params types.RequestParams) types.Response {
		return RespondSuccess("ok")
	})
}
