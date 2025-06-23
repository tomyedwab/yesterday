build:
	mkdir -p dist/github.com/tomyedwab/yesterday/apps/login/db
	cp -R nexushub/krunclient/rootfs/* dist/github.com/tomyedwab/yesterday/apps/login/
	go build -o dist/github.com/tomyedwab/yesterday/apps/login/bin/app apps/login/main.go
	cp -R apps/login/web dist/github.com/tomyedwab/yesterday/apps/login/static
	mkdir -p dist/github.com/tomyedwab/yesterday/apps/admin/db
	cp -R nexushub/krunclient/rootfs/* dist/github.com/tomyedwab/yesterday/apps/admin/
	go build -o dist/github.com/tomyedwab/yesterday/apps/admin/bin/app apps/admin/main.go
	mkdir -p dist/github.com/tomyedwab/yesterday/apps/example/db
	cp -R nexushub/krunclient/rootfs/* dist/github.com/tomyedwab/yesterday/apps/example/
	go build -o dist/github.com/tomyedwab/yesterday/apps/example/bin/app apps/example/main.go
	mkdir -p dist/github.com/tomyedwab/yesterday/nexushub
	go build -o dist/github.com/tomyedwab/yesterday/nexushub/bin/app nexushub/cmd/serve/main.go
	gcc -o dist/github.com/tomyedwab/yesterday/nexushub/bin/krunclient nexushub/krunclient/main.c -l krun

serve: build
	if command -v hl >/dev/null 2>&1; then \
		./dist/github.com/tomyedwab/yesterday/nexushub/bin/app | hl -F -h component -h pid; \
	else \
		./dist/github.com/tomyedwab/yesterday/nexushub/bin/app; \
	fi

deploy:
	aws s3 sync ./www s3://login-tomyedwab-com/
	aws cloudfront create-invalidation --distribution-id E311IV8B19C9AA --paths "/*"
