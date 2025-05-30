package wasi

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/database"
	"github.com/tomyedwab/yesterday/database/events"
)

//go:wasmimport env register_event_handler
func register_event_handler(eventType string, handlerId uint32)

//go:wasmimport env report_event_error
func report_event_error(error string)

var EVENT_HANDLERS map[int]database.GenericEventHandler
var NEXT_EVENT_HANDLER_ID int

func InitEvents() {
	EVENT_HANDLERS = make(map[int]database.GenericEventHandler)
	NEXT_EVENT_HANDLER_ID = 1
}

func RegisterEventHandler[T events.Event](eventType string, handler database.EventHandler[T]) {
	register_event_handler(eventType, uint32(NEXT_EVENT_HANDLER_ID))
	EVENT_HANDLERS[NEXT_EVENT_HANDLER_ID] = func(tx *sqlx.Tx, eventJson []byte) (bool, error) {
		var event T
		if err := json.Unmarshal(eventJson, &event); err != nil {
			return false, fmt.Errorf("failed to unmarshal event of type %s: %w", eventType, err)
		}
		return handler(tx, event)
	}
	NEXT_EVENT_HANDLER_ID++
}

//go:wasmexport handle_event
func handle_event(eventHandle uint32, txHandle uint32, handlerId uint32) int32 {
	handler := EVENT_HANDLERS[int(handlerId)]
	if handler == nil {
		report_event_error("Handler not found for ID: " + fmt.Sprintf("%d", handlerId))
		return -1
	}

	db, err := sqlx.Connect("sqlproxy", string(GetBytes(txHandle)))
	if err != nil {
		report_event_error("Error connecting to database: " + err.Error())
		return -1
	}
	defer db.Close()

	tx, err := db.Beginx()
	if err != nil {
		report_event_error("Error beginning transaction: " + err.Error())
		return -1
	}

	updated, err := handler(tx, GetBytes(eventHandle))
	if err != nil {
		report_event_error("Error handling event: " + err.Error())
		return -1
	}
	if !updated {
		return 0
	}
	return 1
}
