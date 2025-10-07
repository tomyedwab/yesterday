package types

import (
	"encoding/json"
	"time"
)

type EventPublishData struct {
	// The client ID for the publish request, used for deduplication
	ClientID string `json:"clientId"`
	// The event type
	Type string `json:"type"`
	// The event timestamp
	Timestamp time.Time `json:"timestamp"`
	// The event payload
	Data json.RawMessage `json:"data"`
}
