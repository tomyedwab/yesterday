package state

import (
	"encoding/json"
	"fmt"

	"github.com/tomyedwab/yesterday/database/events"
)

// MapUserEventType maps incoming JSON to user-specific Go event types.
func MapUserEventType(rawMessage *json.RawMessage, generic *events.GenericEvent) (events.Event, error) {
	switch generic.GetType() {
	case "users:ADD_USER":
		var specificEvent UserAddedEvent
		if err := json.Unmarshal(*rawMessage, &specificEvent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserAddedEvent: %w", err)
		}
		specificEvent.GenericEvent = *generic // Copy generic fields
		return &specificEvent, nil

	case "users:CHANGE_PASSWORD":
		var specificEvent UserChangedPasswordEvent
		if err := json.Unmarshal(*rawMessage, &specificEvent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserChangedPasswordEvent: %w", err)
		}
		specificEvent.GenericEvent = *generic // Copy generic fields
		return &specificEvent, nil

	case "users:USER_PROFILE_UPDATED":
		var specificEvent UserProfileUpdatedEvent
		if err := json.Unmarshal(*rawMessage, &specificEvent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserProfileUpdatedEvent: %w", err)
		}
		specificEvent.GenericEvent = *generic // Copy generic fields
		return &specificEvent, nil

	case "users:ADD_APPLICATION":
		var specificEvent ApplicationAddedEvent
		if err := json.Unmarshal(*rawMessage, &specificEvent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ApplicationAddedEvent: %w", err)
		}
		specificEvent.GenericEvent = *generic // Copy generic fields
		return &specificEvent, nil

	case "users:DELETE_APPLICATION":
		var specificEvent ApplicationDeletedEvent
		if err := json.Unmarshal(*rawMessage, &specificEvent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ApplicationDeletedEvent: %w", err)
		}
		specificEvent.GenericEvent = *generic // Copy generic fields
		return &specificEvent, nil

	case "users:UPDATE_APPLICATION_HOSTNAME":
		var specificEvent ApplicationHostNameUpdatedEvent
		if err := json.Unmarshal(*rawMessage, &specificEvent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ApplicationHostNameUpdatedEvent: %w", err)
		}
		specificEvent.GenericEvent = *generic // Copy generic fields
		return &specificEvent, nil

	default:
		// If the type isn't user-specific, return the generic event
		// This allows other handlers (if any) to process non-user events.
		fmt.Printf("Unknown or non-user event type encountered by user mapper: %s", generic.GetType())
		return generic, nil
	}
}
