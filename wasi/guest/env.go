package guest

//go:wasmimport env get_env
func get_env(string) uint32

func GetEnv(key string) string {
	handle := get_env(key)
	byte := GetBytes(handle)
	FreeBytes(handle)
	return string(byte)
}
