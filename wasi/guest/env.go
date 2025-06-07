package guest

//go:wasmimport env get_host
func get_host() uint32

func GetHost() string {
	handle := get_host()
	byte := GetBytes(handle)
	FreeBytes(handle)
	return string(byte)
}
