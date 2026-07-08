.PHONY: build run test migrate-up migrate-down lint

# Default DATABASE_URL for local development
DATABASE_URL ?= postgres://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

migrate-up:
	migrate -path internal/db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path internal/db/migrations -database "$(DATABASE_URL)" down

lint:
	golangci-lint run ./...
