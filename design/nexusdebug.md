# Nexus Debug application

In order to debug go servers running inside the Nexushub application, a CLI tool
called `nexusdebug` will be created. The tool will be run in an application
directory and do the following:

1. Log into the admin app using normal username/password login through the Nexushub API.
2. Create a new debug application using the Nexushub API.
3. Build the application server package using a provided command, defaulting to `make build`.
4. Upload the package to Nexushub using a chunked file upload API.
5. Install the package using the Nexushub API and start the server process.
6. Tail the server logs through the Nexushub API and display server status.
7. Listen for user input. If the user presses the R key, re-build the server and repeat the process.
8. If the user presses the Q key, stop the server process and exit.

The tool will be written in Go and live in a new `nexusdebug` directory. It will
use the Go client in `clients/go` for authentication and API access.

The relevant Nexushub APIs are:

- `POST /public/login` (handled by the Admin app)
- `POST /debug/application` (handled by Nexushub internally)
- `POST /debug/application/{id}/upload` (handled by Nexushub internally)
- `POST /debug/application/{id}/install` (handled by Nexushub internally)
- `GET /debug/application/{id}/status` (handled by Nexushub internally)
- `GET /debug/application/{id}/logs` (handled by Nexushub internally)

The CLI tool takes as parameters:
- The admin app URL to log into
- The name of the application (used to generate an AppID, DisplayName, and HostName)
- The build command to run (defaulting to `make build`)
- The package filename to upload (defaulting to `dist/package.zip`)
- An optional static service URL to proxy frontend requests to