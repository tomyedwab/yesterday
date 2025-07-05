package events

// Package events provides a mechanism for handling events in a distributed
// system. Events are received as JSON messages and stored in the database as an
// append-only log, while at the same time being parsed and interpreted to
// update other tables transactionally which provide a view of the current
// application state.
