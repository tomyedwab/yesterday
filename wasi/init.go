package wasi

func Init() {
	InitSQLProxy()
	InitBytes()
	InitRequests()
	InitEvents()

	RegisterHandler("/api/status", func(params RequestParams) (string, error) {
		return "ok", nil
	})
}
