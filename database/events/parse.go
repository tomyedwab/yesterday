package events

import (
	"encoding/json"
)

func ParseEvent(body []byte, mapper MapEventType) (Event, error) {
	// Decode JSON
	var event json.RawMessage
	err := json.Unmarshal(body, &event)
	if err != nil {
		return nil, err
	}

	var generic GenericEvent
	err = json.Unmarshal(event, &generic)
	if err != nil {
		return nil, err
	}

	return mapper(&event, &generic)
}
