package events

type Event interface {
	GetId() int
	GetType() string
	SetId(id int)
}

type EventHandler interface {
	HandleEvent(event Event) error
}
