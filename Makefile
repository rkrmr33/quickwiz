.PHONY: help build run test clean docker-build docker-run docker-stop install

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -o quickwiz ./cmd/server

run: build ## Build and run the application
	./quickwiz

dev: ## Run the application in development mode
	go run cmd/server/main.go

test: ## Run all tests
	go test ./...

test-coverage: ## Run tests with coverage
	go test -cover ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

clean: ## Clean build artifacts
	rm -f quickwiz
	rm -f cmd/server/server
	go clean

docker-build: ## Build Docker image
	docker build -t quickwiz:latest .

docker-run: ## Run application in Docker
	docker run -p 8080:8080 quickwiz:latest

docker-compose-up: ## Start application with docker-compose
	docker-compose up --build

docker-compose-down: ## Stop docker-compose services
	docker-compose down

docker-stop: ## Stop all running quickwiz containers
	docker stop $$(docker ps -q --filter ancestor=quickwiz:latest)

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: fmt vet ## Run linters

all: clean lint test build ## Clean, lint, test, and build
