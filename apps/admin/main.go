package main

import (
	"github.com/tomyedwab/yesterday/apps/admin/state"
	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/wasi"
)

//go:wasmexport init
func init() {
	wasi.Init()
	wasi.RegisterEventHandler(events.DBInitEventType, state.HandleInitEvent)
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
