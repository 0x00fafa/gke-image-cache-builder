# Tool configuration
TOOL_NAME ?= gke-image-cache-builder
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "2.0.0")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags for static binary
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT) -w -s"
BUILD_FLAGS = -a -installsuffix cgo

# Directories
BIN_DIR = bin
DIST_DIR = dist

.PHONY: build build-static build-all clean install test test-binary lint help

# Default target - build static binary
all: build-static

# Build static binary (recommended for distribution)
build-static:
	@echo "Building static $(TOOL_NAME) v$(VERSION)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(TOOL_NAME) ./cmd

# Build regular binary
build:
	@echo "Building $(TOOL_NAME) v$(VERSION)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(TOOL_NAME) ./cmd

# Build with multiple names for different preferences (all static)
build-all:
	@echo "Building all static variants..."
	@mkdir -p $(BIN_DIR)
	# Main name
	CGO_ENABLED=0 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/gke-image-cache-builder ./cmd
	# Short variants
	CGO_ENABLED=0 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/gkeimg ./cmd
	CGO_ENABLED=0 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/imgcache ./cmd

# Install to system
install:
	@echo "Installing $(TOOL_NAME)..."
	CGO_ENABLED=0 go install $(BUILD_FLAGS) $(LDFLAGS) ./cmd

# Create release binaries (all static)
release:
	@echo "Creating static release binaries..."
	@mkdir -p $(DIST_DIR)
	# Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(TOOL_NAME)-linux-amd64 ./cmd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(TOOL_NAME)-linux-arm64 ./cmd
	# macOS
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(TOOL_NAME)-darwin-amd64 ./cmd
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(TOOL_NAME)-darwin-arm64 ./cmd
	# Windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(TOOL_NAME)-windows-amd64.exe ./cmd

# Test the binary works independently
test-binary: build-static
	@echo "Testing binary independence..."
	@mkdir -p /tmp/gke-test
	@cp $(BIN_DIR)/$(TOOL_NAME) /tmp/gke-test/
	@cd /tmp/gke-test && ./$(TOOL_NAME) --version
	@rm -rf /tmp/gke-test
	@echo "✅ Binary works independently!"

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
	@echo "  build-static - Build static binary (default, recommended)"
	@echo "  build        - Build regular binary"
	@echo "  build-all    - Build all name variants (static)"
	@echo "  install      - Install to system"
	@echo "  release      - Create release binaries for all platforms"
	@echo "  test-binary  - Test binary independence"
	@echo "  test         - Run tests"
	@echo "  lint         - Run linter"
	@echo "  clean        - Clean build artifacts"
	@echo "  help         - Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  TOOL_NAME  - Tool name (default: $(TOOL_NAME))"
	@echo "  VERSION    - Version (default: $(VERSION))"
	@echo ""
	@echo "Static binary features:"
	@echo "  ✅ No external dependencies"
	@echo "  ✅ Embedded setup scripts"
	@echo "  ✅ Portable across systems"
	@echo "  ✅ Optimized size"
