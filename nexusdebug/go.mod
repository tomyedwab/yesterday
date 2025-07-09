module github.com/tomyedwab/yesterday/nexusdebug

go 1.24.4

require (
	github.com/tomyedwab/yesterday/clients/go v0.0.0-00010101000000-000000000000
	golang.org/x/term v0.28.0
)

require golang.org/x/sys v0.29.0 // indirect

replace github.com/tomyedwab/yesterday/clients/go => ../clients/go
