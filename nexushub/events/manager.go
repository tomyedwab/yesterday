package events

import "github.com/jmoiron/sqlx"

type EventManager struct {
	DB            *sqlx.DB
	LatestEventId int
}

func CreateEventManager(db *sqlx.DB) (*EventManager, error) {
	err := EventDBInit(db)
	if err != nil {
		return nil, err
	}
	return &EventManager{
		DB:            db,
		LatestEventId: 0,
	}, nil
}
