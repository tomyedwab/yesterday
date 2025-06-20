build:
	mkdir -p dist/github.com/tomyedwab/yesterday/apps/login
	go build -o dist/github.com/tomyedwab/yesterday/apps/login/app.bin apps/login/main.go
	cp -R apps/login/web dist/github.com/tomyedwab/yesterday/apps/login/static
	mkdir -p dist/github.com/tomyedwab/yesterday/apps/admin
	go build -o dist/github.com/tomyedwab/yesterday/apps/admin/app.bin apps/admin/main.go
	mkdir -p dist/github.com/tomyedwab/yesterday/apps/example
	go build -o dist/github.com/tomyedwab/yesterday/apps/example/app.bin apps/example/main.go
	mkdir -p dist/github.com/tomyedwab/yesterday/nexushub
	go build -o dist/github.com/tomyedwab/yesterday/nexushub/app.bin nexushub/cmd/serve/main.go

serve: build
	if command -v hl >/dev/null 2>&1; then \
		./dist/github.com/tomyedwab/yesterday/nexushub/app.bin | hl -F -h component -h pid; \
	else \
		./dist/github.com/tomyedwab/yesterday/nexushub/app.bin; \
	fi

deploy:
	aws s3 sync ./www s3://login-tomyedwab-com/
	aws cloudfront create-invalidation --distribution-id E311IV8B19C9AA --paths "/*"
