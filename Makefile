build:
	mkdir -p dist/0001-0001 dist/0001-0002 dist/bin
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o dist/0001-0001/app.wasm apps/login/main.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o dist/0001-0002/app.wasm apps/admin/main.go
	go build -o dist/bin/servicehost cmd/servicehost/main.go
	go build -o dist/bin/nexushub cmd/nexushub/main.go

serve: build
	./dist/bin/nexushub

deploy:
	aws s3 sync ./www s3://login-tomyedwab-com/
	aws cloudfront create-invalidation --distribution-id E311IV8B19C9AA --paths "/*"