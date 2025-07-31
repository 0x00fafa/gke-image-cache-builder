# Tool configuration
TOOL_NAME ?= gke-image-cache-builder
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "2.0.0")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# Directories
BIN_DIR = bin
DIST_DIR = dist

.PHONY: build build-all clean install test lint help

# Default target
all: build

# Build the main executable
build:
	@echo "Building $(TOOL_NAME) v$(VERSION)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(TOOL_NAME) ./cmd

# Build with multiple names for different preferences
build-all:
	@echo "Building all variants..."
	@mkdir -p $(BIN_DIR)
	# Main name
	go build $(LDFLAGS) -o $(BIN_DIR)/gke-image-cache-builder ./cmd
	# Short variants
	go build $(LDFLAGS) -o $(BIN_DIR)/gkeimg ./cmd
	go build $(LDFLAGS) -o $(BIN_DIR)/imgcache ./cmd

# Install to system
install:
	@echo "Installing $(TOOL_NAME)..."
	go install $(LDFLAGS) ./cmd

# Create release binaries
release:
	@echo "Creating release binaries..."
	@mkdir -p $(DIST_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(TOOL_NAME)-linux-amd64 ./cmd
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(TOOL_NAME)-linux-arm64 ./cmd
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(TOOL_NAME)-darwin-amd64 ./cmd
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(TOOL_NAME)-darwin-arm64 ./cmd
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(TOOL_NAME)-windows-amd64.exe ./cmd

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BIN_DIR) $(DIST_DIR)

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the main executable"
	@echo "  build-all  - Build all name variants"
	@echo "  install    - Install to system"
	@echo "  release    - Create release binaries"
	@echo "  test       - Run tests"
	@echo "  lint       - Run linter"
	@echo "  clean      - Clean build artifacts"
	@echo "  help       - Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  TOOL_NAME  - Tool name (default: $(TOOL_NAME))"
	@echo "  VERSION    - Version (default: $(VERSION))"
