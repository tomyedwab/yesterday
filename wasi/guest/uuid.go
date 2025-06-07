package guest

//go:wasmimport env create_uuid
func create_uuid() uint32

func CreateUUID() string {
	handle := create_uuid()
	ret := string(GetBytes(handle))
	FreeBytes(handle)
	return ret
}
