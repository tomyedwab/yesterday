package database

// Package database provides a connection to the SQLite database and manages the
// shared database schema such as the event log and versions table. This package
// is a generic utility; application-specific event types & state tables are
// delegated to a separate package and injected in at startup.
