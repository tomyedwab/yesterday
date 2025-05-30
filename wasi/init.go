package wasi

func Init() {
	InitSQLProxy()
	InitBytes()
	InitRequests()

	RegisterHandler("/api/status", func(params RequestParams) (string, error) {
		return "ok", nil
	})
}
