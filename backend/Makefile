APP_NAME := mini-store-api

.PHONY: run build test tidy fmt

run:
	go run ./cmd/api

build:
	go build -o ./bin/$(APP_NAME) ./cmd/api

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	gofmt -w ./cmd ./internal
