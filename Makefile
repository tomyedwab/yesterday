build:
	mkdir -p dist/0001-0001
	mkdir -p dist/bin
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o dist/0001-0001/app.wasm users/cmd/serve/main.go
	go build -o dist/bin/servicehost cmd/servicehost/main.go

serve: build
	./dist/bin/servicehost -wasm dist/0001-0001/app.wasm -dbPath users.db -port 8080

deploy:
	aws s3 sync ./www s3://login-tomyedwab-com/
	aws cloudfront create-invalidation --distribution-id E311IV8B19C9AA --paths "/*"