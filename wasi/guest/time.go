package guest

// Time utilities needed because the WASI environment does not seem to provide a
// correct time source.

import (
	"time"
)

//go:wasmimport env get_time
func get_time() uint64

func GetTime() time.Time {
	return time.Unix(int64(get_time()), 0)
}

func TimeSince(t time.Time) time.Duration {
	return GetTime().Sub(t)
}
