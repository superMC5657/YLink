# NanoCloud API Makefile
.PHONY: run worker migrate swagger lint test build up down

run:
	go run ./cmd/server

worker:
	go run ./cmd/worker

migrate:
	go run github.com/golang-migrate/migrate/v4/cmd/migrate -path migrations -database "$$DB_URL" up

migrate-down:
	go run github.com/golang-migrate/migrate/v4/cmd/migrate -path migrations -database "$$DB_URL" down 1

swagger:
	swag init -g cmd/server/main.go -o docs

lint:
	golangci-lint run ./...

test:
	go test ./...

build:
	go build -ldflags="-s -w" -o bin/server ./cmd/server
	go build -ldflags="-s -w" -o bin/worker ./cmd/worker

up:
	docker compose up -d

down:
	docker compose down
