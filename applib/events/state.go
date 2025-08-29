package events

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type EventState struct {
	// The current application event ID
	CurrentEventId int

	// A set of subscribers who want to know when the event ID has changed
	mutex       sync.RWMutex
	subscribers []chan int
}

func NewEventState(db *sqlx.DB) (*EventState, error) {
	// Create users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS event_state (
		    id INTEGER PRIMARY KEY,
			current_event_id INTEGER
		)`)
	if err != nil {
		return nil, fmt.Errorf("failed to create event state table: %w", err)
	}

	// Create event state with event ID set to zero
	_, err = db.Exec(`
		INSERT INTO event_state (id, current_event_id)
		SELECT 0, 0
		ON CONFLICT DO NOTHING
		`)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event state: %w", err)
	}

	var initialEventId int
	err = db.Get(&initialEventId, `SELECT current_event_id FROM event_state WHERE id = 0`)
	if err != nil {
		return nil, fmt.Errorf("failed to get event state: %w", err)
	}

	log.Printf("Initialized event state with ID %d\n", initialEventId)

	return &EventState{
		CurrentEventId: initialEventId,
		subscribers:    make([]chan int, 0),
	}, nil
}

func (state *EventState) Subscribe() chan int {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	ch := make(chan int)
	state.subscribers = append(state.subscribers, ch)
	return ch
}

func (state *EventState) Unsubscribe(ch chan int) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	for i, subscriber := range state.subscribers {
		if subscriber == ch {
			state.subscribers = append(state.subscribers[:i], state.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
	close(ch)
}

func (state *EventState) SetCurrentEventId(eventId int, tx *sqlx.Tx) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	_, err := tx.Exec(`UPDATE event_state SET current_event_id = $1 WHERE id = 0`, eventId)
	if err != nil {
		log.Printf("Failed to update event state: %v", err)
		return
	}

	fmt.Printf("Updating event state to ID %d\n", eventId)
	state.CurrentEventId = eventId
	for _, subscriber := range state.subscribers {
		subscriber <- eventId
	}
}

func (state *EventState) PollForEventId(eventId int) bool {
	if state.CurrentEventId >= eventId {
		return true
	}
	ch := state.Subscribe()
	defer state.Unsubscribe(ch)

	timeoutCh := make(chan bool)
	go func() {
		time.Sleep(50 * time.Second)
		timeoutCh <- true
		close(timeoutCh)
	}()

	for {
		select {
		case newId := <-ch:
			if newId >= eventId {
				return true
			}
		case <-timeoutCh:
			return false
		}
	}
}
