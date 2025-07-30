# Build variables
BINARY_NAME=flake
BUILD_DIR=build

# Go build flags
LDFLAGS=-ldflags="-s -w"

.PHONY: all build install clean test uninstall

# Default target
all: build

# Build the binary
build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Install the binary using go install
install:
	go install $(LDFLAGS) .

# Uninstall the binary
uninstall:
	@echo "To uninstall, remove $(BINARY_NAME) from your GOPATH/bin or GOBIN directory"

# Clean build artifacts
clean:
	@rm -rf $(BUILD_DIR)
	@echo "Build artifacts cleaned"

# Run tests
test:
	go test -v ./...