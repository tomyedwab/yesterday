package events

import (
	"encoding/json"
	"time"
)

type Event interface {
	GetId() int
	GetType() string
	SetId(id int)
}

type EventHandler interface {
	HandleEvent(event Event) error
}

type GenericEvent struct {
	// The event ID
	Id int `json:"id"`
	// The event type
	Type string `json:"type"`
	// The event timestamp
	Timestamp time.Time `json:"timestamp"`
}

func (e GenericEvent) GetId() int {
	return e.Id
}

func (e *GenericEvent) SetId(id int) {
	e.Id = id
}

func (e *GenericEvent) GetType() string {
	return e.Type
}

func (e *GenericEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

type DBInitEvent struct {
	GenericEvent
}

// A function which takes a JSON message and returns a type-specific Event
// instance, or the original generic event.
type MapEventType func(message *json.RawMessage, generic *GenericEvent) (Event, error)
