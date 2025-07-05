package events

import (
	"fmt"
	"sync"
	"time"
)

type EventState struct {
	// The current global event ID
	CurrentEventId int

	// A set of subscribers who want to know when the event ID has changed
	mutex       sync.RWMutex
	subscribers []chan int
}

func NewEventState(initialEventId int) *EventState {
	fmt.Printf("Initializing event state with ID %d\n", initialEventId)
	return &EventState{
		CurrentEventId: initialEventId,
		subscribers:    make([]chan int, 0),
	}
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

func (state *EventState) SetCurrentEventId(eventId int) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

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
