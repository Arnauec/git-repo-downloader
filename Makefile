# Git Repository Downloader Makefile

.PHONY: build test clean install help

# Default target
help:
	@echo "Git Repository Downloader"
	@echo "Available targets:"
	@echo "  build    - Build the application"
	@echo "  test     - Run tests"
	@echo "  clean    - Clean build artifacts"
	@echo "  install  - Install the application to GOPATH/bin"
	@echo "  run      - Show usage example"
	@echo "  deps     - Download dependencies"

# Build the application
build:
	@echo "Building git-repo-downloader..."
	go build -o git-repo-downloader .

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod tidy
	go mod download

# Test the application
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f git-repo-downloader
	rm -rf test-downloads/

# Install to GOPATH/bin
install:
	@echo "Installing git-repo-downloader..."
	go install .

# Show usage example
run:
	@echo "Running git-repo-downloader (help)..."
	./git-repo-downloader || true
	@echo "\nTo test with a real organization:"
	@echo "  ./git-repo-downloader -platform=github -org=kubernetes"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run || echo "golangci-lint not installed, skipping..."

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	GOOS=darwin GOARCH=amd64 go build -o build/git-repo-downloader-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o build/git-repo-downloader-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build -o build/git-repo-downloader-linux-amd64 .
	GOOS=windows GOARCH=amd64 go build -o build/git-repo-downloader-windows-amd64.exe . 