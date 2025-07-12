PHONY:

build: PHONY
	mkdir -p build/bin dist
	go build -o build/bin/app main.go
	(cd build && zip -r ../dist/package.zip .)

dev: PHONY
	nexusdebug -admin-url=https://admin.yesterday.localhost:8443 -app-name=example
