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
	mkdir -p build/libkrun/bin
	cp -R nexushub/krunclient/rootfs/* build/libkrun/
	gcc -o build/libkrun/bin/krunclient nexushub/krunclient/main.c -l krun
	(cd build/libkrun && zip -r ../pkg/github_com__tomyedwab__yesterday__libkrun.zip .)
	# Build nexushub executable
	mkdir -p build/nexushub/bin
	go build -o build/nexushub/bin/nexushub nexushub/cmd/serve/main.go

serve:
	mkdir -p dist/install
	if command -v hl >/dev/null 2>&1; then \
		PKG_DIR=$(PWD)/build/pkg INSTALL_DIR=$(PWD)/dist/install ./build/nexushub/bin/nexushub | hl -F -h component -h pid; \
	else \
		PKG_DIR=$(PWD)/build/pkg INSTALL_DIR=$(PWD)/dist/install ./build/nexushub/bin/nexushub; \
	fi

deploy:
	aws s3 sync ./www s3://login-tomyedwab-com/
	aws cloudfront create-invalidation --distribution-id E311IV8B19C9AA --paths "/*"
