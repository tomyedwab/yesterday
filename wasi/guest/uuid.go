package guest

//go:wasmimport env create_uuid
func create_uuid(destPtr *uint32) uint32

func CreateUUID() string {
	var destPtr uint32
	size := create_uuid(&destPtr)
	return string(GetBytesFromPtr(destPtr, size))
}
