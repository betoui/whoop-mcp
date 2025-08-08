# Whoop MCP Server Makefile

.PHONY: build test run clean install lint fmt vet deps help

# Default target
all: build

# Build the application
build:
	@echo "Building Whoop MCP Server..."
	go build -o bin/whoop-mcp-server .

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run the server (requires WHOOP_API_KEY environment variable)
run: build
	@echo "Starting Whoop MCP Server..."
	@if [ -z "$(WHOOP_API_KEY)" ]; then \
		echo "Error: WHOOP_API_KEY environment variable is required"; \
		echo "Set it with: export WHOOP_API_KEY=your_api_key"; \
		exit 1; \
	fi
	./bin/whoop-mcp-server

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golint ./...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install the binary to GOPATH/bin
install: build
	@echo "Installing to GOPATH/bin..."
	go install .

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	@if ! command -v golint >/dev/null 2>&1; then \
		echo "Installing golint..."; \
		go install golang.org/x/lint/golint@latest; \
	fi
	@if [ ! -f .env ]; then \
		echo "Creating .env file from example..."; \
		cp env.example .env; \
		echo "Please edit .env and add your WHOOP_API_KEY"; \
	fi

# Check all code quality
check: fmt vet lint test

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/whoop-mcp-server-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o bin/whoop-mcp-server-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o bin/whoop-mcp-server-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o bin/whoop-mcp-server-windows-amd64.exe .

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t whoop-mcp-server .

# Run with Docker
docker-run: docker-build
	@if [ -z "$(WHOOP_API_KEY)" ]; then \
		echo "Error: WHOOP_API_KEY environment variable is required"; \
		exit 1; \
	fi
	docker run -e WHOOP_API_KEY=$(WHOOP_API_KEY) whoop-mcp-server

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  deps           - Install dependencies"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  run            - Build and run the server"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  vet            - Vet code"
	@echo "  clean          - Clean build artifacts"
	@echo "  install        - Install binary to GOPATH/bin"
	@echo "  dev-setup      - Set up development environment"
	@echo "  check          - Run all code quality checks"
	@echo "  build-all      - Build for multiple platforms"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run with Docker"
	@echo "  help           - Show this help message" 