SHELL := /bin/bash

PHONY:

clean: PHONY
	rm -rf build
	rm -rf dist/install

build: clean
	mkdir -p build/pkg
	# Build login app
	mkdir -p build/login/bin
	go build -o build/login/bin/app apps/login/main.go
	cp -R apps/login/web build/login/static
	(cd build/login && zip -r ../pkg/github_com__tomyedwab__yesterday__apps__login.zip .)
	# Build admin app
	mkdir -p build/admin/bin
	mkdir -p build/admin/static
	go build -o build/admin/bin/app apps/admin/main.go
	(cd apps/admin/web && npm run build && cp -R dist/* ../../../build/admin/static/)
	(cd build/admin && zip -r ../pkg/github_com__tomyedwab__yesterday__apps__admin.zip .)
	# Build example app
	mkdir -p build/example/bin
	go build -o build/example/bin/app apps/example/main.go
	(cd build/example && zip -r ../pkg/github_com__tomyedwab__yesterday__apps__example.zip .)
	# Build libkrun
	mkdir -p build/libkrun/bin build/libkrun/lib
	cp -R nexushub/krunclient/rootfs/* build/libkrun/
	gcc -o build/libkrun/bin/krunclient nexushub/krunclient/main.c -l krun
	cp /lib/x86_64-linux-gnu/libkrun.so.1 build/libkrun/lib/
	cp /lib/x86_64-linux-gnu/libkrunfw.so.4 build/libkrun/lib/
	(cd build/libkrun && zip -r ../pkg/github_com__tomyedwab__yesterday__libkrun.zip .)
	# Build nexushub executable
	mkdir -p build/nexusdebug
	(cd nexusdebug && go build -o ../build/nexusdebug/nexusdebug cmd/main.go)
	mkdir -p build/nexushub
	go build -o build/nexushub/nexushub nexushub/cmd/serve/main.go

install: PHONY
	mkdir -p /usr/local/etc/nexushub/{certs,install,packages} /usr/local/bin
	openssl req -x509 -newkey rsa:4096 \
	    -keyout /usr/local/etc/nexushub/certs/server.key \
		-out /usr/local/etc/nexushub/certs/server.crt \
		-days 365 -nodes \
		-subj "/C=US/ST=California/L=San Francisco/O=Yesterday/OU=Development/CN=yesterday.localhost" \
		-addext "subjectAltName = DNS:*.yesterday.localhost"
	cp build/pkg/* /usr/local/etc/nexushub/packages/
	install -m 755 build/nexusdebug/nexusdebug /usr/local/bin/
	install -m 755 build/nexushub/nexushub /usr/local/bin/

serve: PHONY
	mkdir -p dist/install
	if command -v hl >/dev/null 2>&1; then \
		PKG_DIR=$(PWD)/build/pkg \
		CERTS_DIR=$(PWD)/dist/certs \
		INSTALL_DIR=$(PWD)/dist/install \
		./build/nexushub/nexushub | hl -F -h component -h pid; \
	else \
    	PKG_DIR=$(PWD)/build/pkg \
    	CERTS_DIR=$(PWD)/dist/certs \
    	INSTALL_DIR=$(PWD)/dist/install \
		./build/nexushub/nexushub; \
	fi

deploy:
	aws s3 sync ./www s3://login-tomyedwab-com/
	aws cloudfront create-invalidation --distribution-id E311IV8B19C9AA --paths "/*"
