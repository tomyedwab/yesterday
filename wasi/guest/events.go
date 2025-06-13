package guest

import (
	"encoding/json"
	"fmt"

	"github.com/tomyedwab/yesterday/database/events"
)

//go:wasmimport env register_event_handler
func register_event_handler(eventType string, handlerId uint32)

//go:wasmimport env report_event_error
func report_event_error(error string)

type EventHandler[T events.Event] func(tx *Tx, event T) (bool, error)
type GenericEventHandler func(tx *Tx, eventJson []byte) (bool, error)

var EVENT_HANDLERS map[int]GenericEventHandler
var NEXT_EVENT_HANDLER_ID int

func InitEvents() {
	EVENT_HANDLERS = make(map[int]GenericEventHandler)
	NEXT_EVENT_HANDLER_ID = 1
}

func RegisterEventHandler[T events.Event](eventType string, handler EventHandler[T]) {
	register_event_handler(eventType, uint32(NEXT_EVENT_HANDLER_ID))
	EVENT_HANDLERS[NEXT_EVENT_HANDLER_ID] = func(tx *Tx, eventJson []byte) (bool, error) {
		var event T
		if err := json.Unmarshal(eventJson, &event); err != nil {
			return false, fmt.Errorf("failed to unmarshal event of type %s: %w", eventType, err)
		}
		return handler(tx, event)
	}
	NEXT_EVENT_HANDLER_ID++
}

//go:wasmexport handle_event
func handle_event(eventPtr, eventSize, txPtr, txSize, handlerId uint32) int32 {
	handler := EVENT_HANDLERS[int(handlerId)]
	if handler == nil {
		report_event_error("Handler not found for ID: " + fmt.Sprintf("%d", handlerId))
		return -1
	}

	db := NewDB()

	tx := db.UseTx(string(GetBytesFromPtr(txPtr, txSize)))
	updated, err := handler(tx, GetBytesFromPtr(eventPtr, eventSize))
	if err != nil {
		report_event_error("Error handling event: " + err.Error())
		return -1
	}
	if !updated {
		return 0
	}
	return 1
}
