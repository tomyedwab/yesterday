package guest

//go:wasmimport env get_env
func get_env(key string, destPtr *uint32) uint32

func GetEnv(key string) string {
	var destPtr uint32
	size := get_env(key, &destPtr)
	return string(GetBytesFromPtr(destPtr, size))
}
