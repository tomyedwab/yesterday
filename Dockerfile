FROM golang:1.23-bookworm AS builder

COPY go.mod go.sum /src/

WORKDIR /src
RUN go mod download

COPY database /src/database
COPY users /src/users
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
  go build -o users-server users/cmd/serve/main.go

FROM debian:12.9

COPY --from=builder /src/users-server /usr/local/bin/users-server
WORKDIR /data

CMD ["/usr/local/bin/users-server"]