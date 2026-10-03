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

# Uses a dedicated PostgreSQL database; tests create and remove isolated schemas.
.PHONY: test-integration
test-integration:
	@test -n "$(TEST_DATABASE_DSN)" || (echo 'Set TEST_DATABASE_DSN to a dedicated PostgreSQL test database'; exit 1)
	go test -race -count=1 ./...
