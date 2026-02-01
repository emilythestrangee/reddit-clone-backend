# Simple Makefile for Reddit Clone Go project

# Load environment variables from .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: all build run deps docker-run docker-down docker-db docker-db-down test itest clean watch db-create db-drop db-reset docker-logs help

# Default target
all: build test

## build: Build the application binary
build:
	@echo "Building..."
	@go build -o main cmd/api/main.go

## run: Run the application locally (requires DB to be running)
run:
	@echo "Starting server..."
	@go run cmd/api/main.go

## deps: Download and tidy dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

## docker-run: Start both PostgreSQL and backend in Docker
docker-run:
	@echo "Starting all services with Docker Compose..."
	@docker compose up --build -d
	@echo "Services started! Backend: http://localhost:8080"

## docker-db: Start ONLY PostgreSQL in Docker (run backend locally)
docker-db:
	@echo "Starting PostgreSQL container..."
	@docker compose up postgres -d
	@echo "PostgreSQL started on localhost:5432"
	@echo "Run 'make run' to start the backend locally"

## docker-down: Stop all Docker services
docker-down:
	@echo "Stopping Docker services..."
	@docker compose down

## docker-db-down: Stop only PostgreSQL
docker-db-down:
	@echo "Stopping PostgreSQL..."
	@docker compose stop postgres

## docker-logs: View Docker logs
docker-logs:
	@docker compose logs -f

## docker-clean: Stop and remove all containers and volumes
docker-clean:
	@echo "Cleaning up Docker resources..."
	@docker compose down -v
	@echo "All containers and volumes removed"

## test: Run all tests
test:
	@echo "Running tests..."
	@go test ./... -v

## itest: Run integration tests
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

## watch: Run with hot reload (Air)
watch:
	@powershell -ExecutionPolicy Bypass -Command "if (Get-Command air -ErrorAction SilentlyContinue) { \
		air; \
		Write-Output 'Watching...'; \
	} else { \
		Write-Output 'Installing air...'; \
		go install github.com/air-verse/air@latest; \
		air; \
		Write-Output 'Watching...'; \
	}"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -f main main.exe
	@rm -rf bin/

## help: Show this help message
help:
	@echo "Available targets:"
	@echo ""
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'