# Makefile for WebSeed2Torznab

.PHONY: build run clean test docker-build docker-run help

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  help         - Show this help message"

# Build the application
build:
	go mod tidy
	go build -o webseed2torznab main.go

# Run the application
run: build
	./webseed2torznab

# Clean build artifacts
clean:
	rm -f webseed2torznab
	go clean

# Run tests (if any)
test:
	go test -v ./...

# Build Docker image
docker-build:
	docker build -t webseed2torznab .

# Run with Docker Compose
docker-run:
	docker-compose up --build

# Run with Docker Compose in background
docker-run-detached:
	docker-compose up --build -d

# Stop Docker Compose
docker-stop:
	docker-compose down

# View Docker logs
docker-logs:
	docker-compose logs -f webseed2torznab
