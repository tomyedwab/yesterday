# Go Yesterday client

The Go client is a package that provides utilities for connecting any Go
application (CLI, server, etc.) to Yesterday's API.

## Features

- Authentication and username/password login
- Asynchronous polling for updated event number
- Generic data provider that given a URI, parameters, and return type, wraps the
  API endpoint and auto-refreshes the data when the event number changes
- Generic event publishing utility that queues up events to send and continues
  trying to publish them until they succeed, using exponential backoff